// Command energy-addon runs the Synctacles Energy Home Assistant addon.
// This serves the web UI, price collectors, action engine, and sensor publisher.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/synctacles/energy-app/pkg/collector"
	"github.com/synctacles/energy-app/internal/config"
	"github.com/synctacles/energy-app/pkg/countries"
	"github.com/synctacles/energy-app/pkg/engine"
	"github.com/synctacles/energy-app/internal/ha"
	"github.com/synctacles/energy-app/internal/hasensor"
	"github.com/synctacles/energy-app/pkg/models"
	"github.com/synctacles/energy-app/internal/state"
	"github.com/synctacles/energy-app/pkg/store"
	"github.com/synctacles/energy-app/internal/telemetry"
	"github.com/synctacles/energy-app/internal/web"
)

var version = "dev"

// alertState tracks price alert deduplication (1-hour cooldown).
type alertState struct {
	mu          sync.Mutex
	lastAlertAt time.Time
}

// MaybeSendAlert sends an HA persistent notification when the price is at or below the
// threshold — but at most once per hour to avoid spamming.
func (a *alertState) MaybeSendAlert(ctx context.Context, sv *ha.SupervisorClient, price, threshold float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if time.Since(a.lastAlertAt) < time.Hour {
		return // cooldown active
	}
	a.lastAlertAt = time.Now()
	if sv != nil {
		title := "⚡ Lage energieprijs"
		msg := fmt.Sprintf("Prijs gedaald naar €%.4f/kWh (drempel: €%.4f)", price, threshold)
		if err := sv.CreateNotification(ctx, title, msg, "synctacles_price_alert"); err != nil {
			slog.Warn("price alert notification failed", "error", err)
		}
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("starting energy-app", "version", version)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.DebugMode {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	// /data is the addon's writable persistent directory.
	// /config is HA's config dir (mounted read-only via config:ro).
	dataPath := "/data"
	if v := os.Getenv("HA_DATA_PATH"); v != "" {
		dataPath = v
	}

	// Load country configs and zone registry
	registry, err := countries.LoadRegistry()
	if err != nil {
		slog.Error("failed to load country configs", "error", err)
		os.Exit(1)
	}
	slog.Info("loaded zone registry", "zones", len(registry.AllZones()))

	// Derive Enever settings from pricing mode
	if cfg.PricingMode == config.ModeEnever && cfg.EneverToken != "" && cfg.BiddingZone == "NL" {
		cfg.EneverEnabled = true
	}

	slog.Info("pricing mode", "mode", cfg.PricingMode, "zone", cfg.BiddingZone)

	// Initialize HA Supervisor client
	var supervisor *ha.SupervisorClient
	if cfg.HasSupervisor() {
		supervisor = ha.NewSupervisorClient(cfg.SupervisorToken)
	}

	// Initialize state store
	stateStore := state.NewStore(dataPath)

	// Initialize SQLite price cache (48h retention)
	var priceCache engine.PriceCache
	sqliteCache, err := store.NewSQLiteCache(dataPath)
	if err != nil {
		slog.Warn("SQLite cache disabled", "error", err)
	} else {
		priceCache = sqliteCache
		// Cleanup old entries on startup
		if deleted, err := sqliteCache.Cleanup(48 * time.Hour); err == nil && deleted > 0 {
			slog.Info("cache cleanup", "deleted", deleted)
		}
	}

	// Load persistent install UUID (shared by telemetry)
	installUUID := telemetry.LoadInstallUUID(dataPath)
	osArch := telemetry.OSArch()
	slog.Info("install identity", "uuid", installUUID, "arch", osArch)

	// Build price source chain for the configured zone
	synctaclesAPI := &collector.SynctaclesAPI{}
	sources := buildSourceChain(cfg, synctaclesAPI)
	slog.Info("source chain configured", "zone", cfg.BiddingZone, "sources", len(sources))

	// Initialize engine components
	taxCache := engine.NewTaxProfileCache(dataPath)
	normalizer := engine.NewNormalizer(taxCache, cfg.SupplierMarkup)
	normalizer.SetZoneRegistry(registry)
	normalizer.SetPricingMode(cfg.PricingMode)

	// Manual mode: build tax profile from user-defined components
	if cfg.PricingMode == config.ModeManual {
		normalizer.SetManualTaxProfile(&models.TaxProfile{
			VATRate:          cfg.ManualVATRate,
			SupplierMarkup:   cfg.SupplierMarkup,
			EnergyTax:        []models.EnergyTaxEntry{{From: "2000-01-01", Rate: cfg.ManualEnergyTax}},
			Surcharges:       cfg.ManualSurcharges,
			NetworkTariffAvg: cfg.ManualNetworkTariff,
		})
	}

	actionEngine := engine.NewActionEngine(cfg.GoThreshold, cfg.AvoidThreshold)
	fallbackMgr := engine.NewFallbackManager(sources, priceCache)

	// Initialize shared sensor data (for web dashboard)
	sensorData := web.NewSensorData()

	// Auto-detect power sensor if not explicitly configured
	var detectedPowerSensor string
	if !cfg.HasPowerSensor() && supervisor != nil {
		if detected := hasensor.DetectPowerSensor(context.Background(), supervisor); detected != "" {
			detectedPowerSensor = detected
			cfg.PowerSensorEntity = detected
			slog.Info("power sensor auto-detected", "entity", detected)
		}
	}

	// Auto-detect tariff sensor (Zonneplan, Tibber, Octopus, P1 Monitor, etc.)
	var detectedTariffSensor string
	if cfg.P1SensorEntity == "" && supervisor != nil {
		if detected := hasensor.DetectTariffSensor(context.Background(), supervisor); detected != "" {
			detectedTariffSensor = detected
			slog.Info("tariff sensor auto-detected", "entity", detected)
		}
	}
	var powerTracker *hasensor.PowerTracker
	if cfg.HasPowerSensor() && supervisor != nil {
		powerTracker = hasensor.NewPowerTracker(cfg.PowerSensorEntity, supervisor)
		slog.Info("power tracker enabled", "entity", cfg.PowerSensorEntity)
	}

	// Initialize sensor publishers
	var publishers []hasensor.Publisher
	var mqttPub *hasensor.MQTTPublisher
	if supervisor != nil {
		publishers = append(publishers, hasensor.NewRESTPublisher(supervisor))

		// Detect MQTT broker
		mqttCreds, found := hasensor.DetectMQTTBroker(context.Background(), supervisor)
		if found {
			mqttPub = hasensor.NewMQTTPublisher(mqttCreds.Host, mqttCreds.Port, mqttCreds.Username, mqttCreds.Password)
			if err := mqttPub.Connect(); err != nil {
				slog.Warn("MQTT connection failed, using REST only", "error", err)
				mqttPub = nil
			} else {
				publishers = append(publishers, mqttPub)
				slog.Info("MQTT publisher enabled (dual publishing)")
			}
		}
	}

	// Price alert state (deduplication across price updates)
	alerts := &alertState{}

	// Scheduler update callback — publishes sensors on every price update
	updateFn := func(ctx context.Context, consumerPrices []models.HourlyPrice, result *engine.FetchResult) error {
		// Split into today/tomorrow
		now := time.Now().UTC()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		tomorrow := today.Add(24 * time.Hour)

		var todayPrices, tomorrowPrices []models.HourlyPrice
		for _, p := range consumerPrices {
			if !p.Timestamp.Before(today) && p.Timestamp.Before(tomorrow) {
				todayPrices = append(todayPrices, p)
			} else if !p.Timestamp.Before(tomorrow) {
				tomorrowPrices = append(tomorrowPrices, p)
			}
		}

		// Compute sensor values
		sensorSet := hasensor.ComputeSensorSet(
			cfg.BiddingZone, todayPrices, tomorrowPrices,
			actionEngine, result, now, cfg.EneverLeverancier,
			cfg.BestWindowHours,
		)

		// External sensor mode: override consumer price with HA sensor reading
		if cfg.IsExternalSensorMode() && supervisor != nil {
			if extPrice, err := readExternalSensorPrice(ctx, supervisor, cfg.P1SensorEntity); err == nil {
				sensorSet.CurrentPrice = extPrice
			} else {
				slog.Warn("external sensor read failed, using calculated price", "entity", cfg.P1SensorEntity, "error", err)
			}
		}

		// Update shared sensor data for web dashboard
		sensorData.Update(sensorSet)

		// Price alert: notify when current price is at or below threshold
		if cfg.HasAlerts() && sensorSet.CurrentPrice > 0 && sensorSet.CurrentPrice <= cfg.AlertThreshold {
			alerts.MaybeSendAlert(ctx, supervisor, sensorSet.CurrentPrice, cfg.AlertThreshold)
		}

		// Read power sensor (if configured)
		if powerTracker != nil {
			powerTracker.ReadPower(ctx, sensorSet.CurrentPrice)
		}

		// Update Worker tax profile cache (for fallback normalization, keyed per zone)
		if result.Source == "synctacles" {
			if tp := synctaclesAPI.LastTaxProfile(); tp != nil {
				var networkCost float64
				if tp.NetworkCostKWh != nil {
					networkCost = *tp.NetworkCostKWh
				}
				taxCache.Put(cfg.BiddingZone, &engine.WorkerTaxOverride{
					VATRate:          tp.VatPct,
					EnergyTax:        tp.EnergyTaxKWh,
					Surcharges:       tp.SurchargesKWh,
					NetworkTariffAvg: networkCost,
					Version:          tp.Version,
				})
			}
		}

		// Update state store
		st := stateStore.Load()
		st.Zone = cfg.BiddingZone
		st.CurrentPrice = sensorSet.CurrentPrice
		st.Action = string(sensorSet.Action.Action)
		st.Quality = result.Quality
		st.PriceSource = result.Source
		st.LastFetch = now.Format(time.RFC3339)
		if err := stateStore.Save(st); err != nil {
			slog.Error("failed to save state", "error", err)
		}

		// Publish to all publishers
		for _, pub := range publishers {
			if err := hasensor.PublishAll(ctx, pub, sensorSet, powerTracker); err != nil {
				slog.Error("sensor publish failed", "error", err)
			}
		}

		slog.Info("sensors updated",
			"zone", cfg.BiddingZone,
			"price", sensorSet.CurrentPrice,
			"action", sensorSet.Action.Action,
			"source", result.Source,
			"quality", result.Quality,
			"today_hours", len(todayPrices),
			"tomorrow_hours", len(tomorrowPrices),
		)
		return nil
	}

	// Signal context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start telemetry sender (daily, first send after 2 min)
	telemetrySender := telemetry.NewSender(telemetry.Deps{
		DataPath: dataPath,
		Version:  version,
		Zone:     cfg.BiddingZone,
		GetCoreInfo: func(ctx context.Context) (arch, haVersion, machine string, err error) {
			if supervisor == nil {
				return "", "", "", fmt.Errorf("no supervisor")
			}
			info, err := supervisor.GetCoreInfo(ctx)
			if err != nil {
				return "", "", "", err
			}
			return info.Arch, info.Version, info.Machine, nil
		},
		GetSupervisorInfo: func(ctx context.Context) (channel string, err error) {
			if supervisor == nil {
				return "", fmt.Errorf("no supervisor")
			}
			info, err := supervisor.GetSupervisorInfo(ctx)
			if err != nil {
				return "", err
			}
			return info.Channel, nil
		},
		GetHostOS: func(ctx context.Context) (operatingSystem string, err error) {
			if supervisor == nil {
				return "", fmt.Errorf("no supervisor")
			}
			info, err := supervisor.GetHostInfo(ctx)
			if err != nil {
				return "", err
			}
			return info.OperatingSystem, nil
		},
		HasSupervisor: cfg.HasSupervisor(),
		GetLocale: func(ctx context.Context) (locale string, err error) {
			if supervisor == nil {
				return "", fmt.Errorf("no supervisor")
			}
			haConfig, err := supervisor.GetConfig(ctx)
			if err != nil {
				return "", err
			}
			if lang, ok := haConfig["language"].(string); ok {
				return lang, nil
			}
			return "", nil
		},
		GetActiveSource: func() string {
			if data := sensorData.Get(); data != nil {
				return data.Source
			}
			return ""
		},
		GetSensorCount: func() int {
			if mqttPub != nil {
				return mqttPub.SensorCount()
			}
			return 0
		},
		CheapestHoursOn: func() bool {
			return cfg.BestWindowHours > 0
		},
		GetTaxSource: func() string {
			return normalizer.TaxSource()
		},
	})
	telemetrySender.RunBackground(ctx)

	// Startup: fetch tax profile from Worker to warm cache.
	// FetchDayAhead caches the tax profile even when prices are empty (stale/missing),
	// so we call it once and check LastTaxProfile regardless of the price result.
	// If the Worker is completely unreachable, retry up to 3 times then start without
	// tax data — the fallback chain (EasyEnergy, etc.) provides its own consumer prices.
	if !taxCache.HasData() {
		slog.Info("first boot — fetching tax profile from Worker")
		const maxRetries = 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			synctaclesAPI.FetchDayAhead(ctx, cfg.BiddingZone, time.Now().UTC()) // ignore error — tax profile is cached as side effect
			if tp := synctaclesAPI.LastTaxProfile(); tp != nil {
				var networkCost float64
				if tp.NetworkCostKWh != nil {
					networkCost = *tp.NetworkCostKWh
				}
				taxCache.Put(cfg.BiddingZone, &engine.WorkerTaxOverride{
					VATRate:          tp.VatPct,
					EnergyTax:        tp.EnergyTaxKWh,
					Surcharges:       tp.SurchargesKWh,
					NetworkTariffAvg: networkCost,
					Version:          tp.Version,
				})
				slog.Info("tax profile cached from Worker", "zone", cfg.BiddingZone, "version", tp.Version)
				break
			}
			slog.Warn("Worker unreachable — no tax profile", "attempt", attempt, "max", maxRetries)
			if attempt < maxRetries {
				select {
				case <-ctx.Done():
					slog.Info("shutdown during startup retry")
					return
				case <-time.After(30 * time.Second):
				}
			}
		}
		if !taxCache.HasData() {
			slog.Warn("starting without tax profile — fallback sources will provide consumer prices")
		}
	} else {
		slog.Info("tax profile cache loaded from disk", "zone", cfg.BiddingZone)
	}

	// Start scheduler
	scheduler := engine.NewScheduler(fallbackMgr, normalizer, actionEngine, cfg.BiddingZone, updateFn)
	go scheduler.Run(ctx)

	// Detect addon slug for dynamic HA UI navigation
	var addonSlug string
	if supervisor != nil {
		addonSlug = supervisor.GetAddonSlug(ctx)
	}

	// Create web server
	srv := web.NewServer(web.Deps{
		Config:              cfg,
		StateStore:          stateStore,
		SensorData:          sensorData,
		Supervisor:          supervisor,
		Fallback:            fallbackMgr,
		Version:             version,
		DetectedPowerSensor:  detectedPowerSensor,
		DetectedTariffSensor: detectedTariffSensor,
		AddonSlug:           addonSlug,
		ZoneRegistry:        registry,
		TaxCache:            taxCache,
		Normalizer:          normalizer,
		Scheduler:           scheduler,
		SQLiteCache:         sqliteCache,
		InstallUUID:         installUUID,
	})

	addr := ":" + strconv.Itoa(cfg.IngressPort)
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", addr, "zone", cfg.BiddingZone)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	scheduler.Stop()

	// Close SQLite cache
	if sqliteCache != nil {
		sqliteCache.Close()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	slog.Info("energy-app stopped")
}

// buildSourceChain creates the prioritized source list for a bidding zone.
//
// Fallback chain (ADR_008):
//
//	Tier 0: Synctacles Worker (primary — wholesale + consumer prices for all zones)
//	Tier 1: Enever (NL only, when configured — exact supplier prices)
//	Tier 2: Energy-Charts (fallback if Worker offline)
//	Tier 3: SQLite cache (offline safety net — handled by FallbackManager)
func buildSourceChain(cfg *config.Config, synctaclesAPI *collector.SynctaclesAPI) []collector.PriceSource {
	var chain []collector.PriceSource

	// When Enever is configured, it provides exact supplier consumer prices —
	// more accurate than Worker's generic tax calculation. Put it first.
	if cfg.HasEnever() && cfg.BiddingZone == "NL" {
		chain = append(chain, &collector.Enever{
			Token:       cfg.EneverToken,
			Leverancier: cfg.EneverLeverancier,
		})
		slog.Info("Enever enabled as primary source (exact supplier prices)", "leverancier", cfg.EneverLeverancier)
	}

	// Synctacles Worker — wholesale + generic tax profile
	chain = append(chain, synctaclesAPI)
	if len(chain) == 1 {
		slog.Info("SynctaclesAPI enabled as primary source")
	} else {
		slog.Info("SynctaclesAPI enabled as fallback source")
	}

	// Energy-Charts — always present as last resort
	chain = append(chain, &collector.EnergyCharts{})

	return chain
}

// readExternalSensorPrice reads the current electricity tariff from an HA sensor entity.
// Returns the price in EUR/kWh. The sensor state must be a numeric string.
func readExternalSensorPrice(ctx context.Context, sv *ha.SupervisorClient, entityID string) (float64, error) {
	state, err := sv.GetState(ctx, entityID)
	if err != nil {
		return 0, fmt.Errorf("read external sensor: %w", err)
	}
	stateStr, ok := state["state"].(string)
	if !ok {
		return 0, fmt.Errorf("external sensor state is not a string")
	}
	price, err := strconv.ParseFloat(stateStr, 64)
	if err != nil {
		return 0, fmt.Errorf("external sensor value %q is not numeric: %w", stateStr, err)
	}
	return price, nil
}
