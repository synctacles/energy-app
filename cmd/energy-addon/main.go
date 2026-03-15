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
	"github.com/synctacles/energy-app/internal/delta"
	"github.com/synctacles/energy-app/internal/gate"
	"github.com/synctacles/energy-app/internal/heartbeat"
	"github.com/synctacles/energy-app/internal/markup"
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
	locale      string // detected from HA config (e.g. "nl", "en", "de")
}

// alertI18n maps locale to [title, messageFormat].
var alertI18n = map[string][2]string{
	"nl": {"⚡ Lage energieprijs", "Prijs gedaald naar €%.4f/kWh (drempel: €%.4f)"},
	"en": {"⚡ Low energy price", "Price dropped to €%.4f/kWh (threshold: €%.4f)"},
	"de": {"⚡ Niedriger Energiepreis", "Preis gefallen auf €%.4f/kWh (Schwelle: €%.4f)"},
	"fr": {"⚡ Prix bas de l'énergie", "Prix tombé à €%.4f/kWh (seuil : €%.4f)"},
	"es": {"⚡ Precio bajo de energía", "Precio bajó a €%.4f/kWh (umbral: €%.4f)"},
	"da": {"⚡ Lav energipris", "Prisen faldt til €%.4f/kWh (tærskel: €%.4f)"},
	"fi": {"⚡ Matala energian hinta", "Hinta laski tasolle €%.4f/kWh (kynnys: €%.4f)"},
	"pt": {"⚡ Preço baixo de energia", "Preço caiu para €%.4f/kWh (limite: €%.4f)"},
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
		strings := alertI18n["en"] // default
		if s, ok := alertI18n[a.locale]; ok {
			strings = s
		}
		title := strings[0]
		msg := fmt.Sprintf(strings[1], price, threshold)
		if err := sv.CreateNotification(ctx, title, msg, "synctacles_price_alert"); err != nil {
			slog.Warn("price alert notification failed", "error", err)
		}
	}
}

func main() {
	// Health check for Docker HEALTHCHECK (no logging, no startup)
	if len(os.Args) > 1 && os.Args[1] == "--health" {
		port := os.Getenv("INGRESS_PORT")
		if port == "" {
			port = "8098"
		}
		resp, err := http.Get("http://localhost:" + port + "/api/version")
		if err != nil || resp.StatusCode != 200 {
			os.Exit(1)
		}
		os.Exit(0)
	}

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
	// /config is HA's shared config dir (mounted via config:rw for shared UUID).
	dataPath := "/data"
	if v := os.Getenv("HA_DATA_PATH"); v != "" {
		dataPath = v
	}
	configPath := "/config"
	if v := os.Getenv("HA_CONFIG_PATH"); v != "" {
		configPath = v
	}

	// Restore non-schema settings from backup file.
	// This protects against HA Options page saves wiping web UI settings
	// (HA Options only knows schema fields; non-schema fields are in the backup).
	config.RestoreFromSettingsFile(cfg, dataPath)

	// Restore consent flags from dedicated file (independent of Supervisor options).
	config.RestoreConsent(cfg, dataPath)

	// Save initial backup (migration: captures existing options.json values on first run)
	settingsPath := config.SettingsFilePath(dataPath)
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		_ = config.SaveSettingsFile(settingsPath, config.BuildSettingsMap(cfg))
		slog.Info("created initial settings backup")
	}

	// Load country configs and zone registry
	registry, err := countries.LoadRegistry()
	if err != nil {
		slog.Error("failed to load country configs", "error", err)
		os.Exit(1)
	}
	slog.Info("loaded zone registry", "zones", len(registry.AllZones()))

	// Auto-detect bidding zone from HA timezone when zone is empty or default.
	// Triggers on first run (empty zone) or legacy default ("NL").
	if (cfg.BiddingZone == "" || cfg.BiddingZone == "NL") && cfg.HasSupervisor() {
		sup := ha.NewSupervisorClient(cfg.SupervisorToken)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		haConfig, err := sup.GetConfig(ctx)
		cancel()
		if err == nil {
			if tz, ok := haConfig["time_zone"].(string); ok && tz != "" {
				for _, code := range registry.AllZones() {
					if z, found := registry.GetZone(code); found && z.Timezone == tz {
						if z.Code != cfg.BiddingZone {
							slog.Info("auto-detected bidding zone from HA timezone",
								"timezone", tz, "zone", z.Code, "previous", cfg.BiddingZone)
							cfg.BiddingZone = z.Code
						}
						break
					}
				}
			}
		} else {
			slog.Debug("zone auto-detect: could not read HA config", "error", err)
		}
	}

	// Fallback: if zone is still empty after auto-detect, default to NL
	if cfg.BiddingZone == "" {
		cfg.BiddingZone = "NL"
		slog.Warn("no zone configured and auto-detect failed, falling back to NL")
	}

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

	// Load persistent install UUID (shared with Care app via /config)
	installUUID := telemetry.LoadInstallUUID(configPath, dataPath)
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
				mqttPub.CleanupStaleTopics()
				publishers = append(publishers, mqttPub)
				slog.Info("MQTT publisher enabled (dual publishing)")
			}
		}
	}

	// Price alert state (deduplication across price updates)
	alerts := &alertState{}

	// Detect locale from HA for localized alert notifications
	if supervisor != nil {
		if haCfg, err := supervisor.GetConfig(context.Background()); err == nil {
			if lang, ok := haCfg["language"].(string); ok && lang != "" {
				alerts.locale = lang[:2] // normalize to 2-char code
				slog.Info("locale detected for alerts", "locale", alerts.locale)
			}
		}
	}

	// Load zone timezone for local-day filtering (energy day = midnight local time)
	var zoneLoc *time.Location
	if z, ok := registry.GetZone(cfg.BiddingZone); ok {
		if loc, err := time.LoadLocation(z.Timezone); err == nil {
			zoneLoc = loc
		}
	}
	if zoneLoc == nil {
		zoneLoc = time.UTC
	}

	// Scheduler update callback — publishes sensors on every price update
	updateFn := func(ctx context.Context, consumerPrices []models.HourlyPrice, result *engine.FetchResult) error {
		// TOU mode: replace fetched prices with synthetic schedule-based prices
		if cfg.IsTOUMode() {
			touCfg, err := engine.ParseTOUConfig(cfg.TOUConfigJSON)
			if err != nil {
				slog.Warn("invalid TOU config, falling back to fetched prices", "error", err)
			} else {
				todayTOU, tomorrowTOU := engine.GenerateTOUPrices(touCfg, zoneLoc)
				// Set zone on synthetic prices
				for i := range todayTOU {
					todayTOU[i].Zone = cfg.BiddingZone
				}
				for i := range tomorrowTOU {
					tomorrowTOU[i].Zone = cfg.BiddingZone
				}
				consumerPrices = append(todayTOU, tomorrowTOU...)
				result = &engine.FetchResult{
					Source:  "tou",
					Tier:    1,
					Quality: "live",
				}
			}
		}

		// Split into today/tomorrow using local midnight (energy day boundary)
		now := time.Now().UTC()
		localNow := now.In(zoneLoc)
		today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, zoneLoc).UTC()
		tomorrow := today.Add(24 * time.Hour)

		var todayPrices, tomorrowPrices []models.HourlyPrice
		for _, p := range consumerPrices {
			if !p.Timestamp.Before(today) && p.Timestamp.Before(tomorrow) {
				todayPrices = append(todayPrices, p)
			} else if !p.Timestamp.Before(tomorrow) {
				tomorrowPrices = append(tomorrowPrices, p)
			}
		}

		// Midnight gap mitigation: if no prices classified as "today" yet
		// (e.g. Enever data for new day not available until ~01:00), use all
		// available prices so the current slot can still be found.
		if len(todayPrices) == 0 && len(consumerPrices) > 0 {
			todayPrices = consumerPrices
			slog.Info("midnight gap: using all cached prices for current slot")
		}

		// Compute sensor values
		sensorSet := hasensor.ComputeSensorSet(
			cfg.BiddingZone, todayPrices, tomorrowPrices,
			actionEngine, result, now, cfg.EneverLeverancier,
			cfg.BestWindowHours, cfg.PricingMode,
		)

		// Sensor override: use HA sensor reading as CurrentPrice when available.
		// Works in both explicit sensor mode AND Enever+sensor (complementary) mode.
		sensorEntity := cfg.P1SensorEntity
		if sensorEntity != "" && supervisor != nil {
			if extPrice, err := readExternalSensorPrice(ctx, supervisor, sensorEntity); err == nil {
				if cfg.IsEneverMode() {
					// Enever+sensor: sensor becomes CurrentPrice, preserve Enever for delta
					sensorSet.EneverPrice = sensorSet.CurrentPrice
					sensorSet.CurrentPrice = extPrice
					sensorSet.Source = "sensor"
				} else if cfg.IsExternalSensorMode() {
					sensorSet.CurrentPrice = extPrice
				}
			} else {
				slog.Warn("sensor read failed, using calculated price", "entity", sensorEntity, "error", err)
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
		DataPath:   dataPath,
		ConfigPath: configPath,
		Version:    version,
		Zone:       cfg.BiddingZone,
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
		GetConfigSnapshot: func() map[string]any {
			return map[string]any{
				"pricing_mode":     cfg.PricingMode,
				"supplier_id":     cfg.SupplierID,
				"supplier_markup": cfg.SupplierMarkup,
				"enever_enabled":  cfg.EneverEnabled,
				"go_threshold":   cfg.GoThreshold,
				"avoid_threshold": cfg.AvoidThreshold,
				"best_window_h":  cfg.BestWindowHours,
				"alerts_enabled": cfg.AlertEnabled,
			}
		},
		GetEntityCount: func(ctx context.Context) int {
			if supervisor == nil {
				return 0
			}
			states, err := supervisor.GetAllStates(ctx)
			if err != nil {
				return 0
			}
			return len(states)
		},
		GetAddonCount: func(ctx context.Context) int {
			if supervisor == nil {
				return 0
			}
			addons, err := supervisor.ListAddons(ctx)
			if err != nil {
				return 0
			}
			return len(addons)
		},
	})
	if cfg.TelemetryEnabled {
		telemetrySender.RunBackground(ctx)
		slog.Info("telemetry enabled")
	} else {
		slog.Info("telemetry disabled by user preference")
	}

	// Heartbeat sender (always active, independent of telemetry preference)

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

	// Feature gate (controls price fetch + actions after GDPR purge)
	featureGate := gate.New(cfg.BiddingZone, false, "")

	// Start scheduler
	scheduler := engine.NewScheduler(fallbackMgr, normalizer, actionEngine, cfg.BiddingZone, updateFn, featureGate)
	zoneHasWholesale := true
	if z, ok := registry.GetZone(cfg.BiddingZone); ok {
		zoneHasWholesale = z.HasWholesale()
		scheduler.SetZoneInfo(zoneHasWholesale, cfg.PricingMode)
	}

	// Non-wholesale zones with fixed/TOU: seed dashboard with full synthetic prices
	if !zoneHasWholesale && (cfg.PricingMode == config.ModeFixed || cfg.PricingMode == config.ModeTOU) {
		now := time.Now().UTC()
		var todayPrices, tomorrowPrices []models.HourlyPrice

		if cfg.PricingMode == config.ModeTOU && cfg.TOUConfigJSON != "" {
			if touCfg, err := engine.ParseTOUConfig(cfg.TOUConfigJSON); err == nil {
				todayPrices, tomorrowPrices = engine.GenerateTOUPrices(touCfg, zoneLoc)
				for i := range todayPrices {
					todayPrices[i].Zone = cfg.BiddingZone
				}
				for i := range tomorrowPrices {
					tomorrowPrices[i].Zone = cfg.BiddingZone
				}
			}
		}
		// Fixed mode or TOU parse failed: flat 24h at the fixed rate
		if len(todayPrices) == 0 && cfg.FixedRatePrice > 0 {
			localNow := now.In(zoneLoc)
			dayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, zoneLoc)
			for h := 0; h < 24; h++ {
				ts := dayStart.Add(time.Duration(h) * time.Hour)
				todayPrices = append(todayPrices, models.HourlyPrice{
					Timestamp: ts.UTC(), PriceEUR: cfg.FixedRatePrice,
					Unit: models.UnitKWh, Source: "fixed", Quality: "live",
					Zone: cfg.BiddingZone, IsConsumer: true,
				})
			}
		}
		if len(todayPrices) > 0 {
			fetchResult := &engine.FetchResult{Source: "regulated", Tier: 1, Quality: "live"}
			ss := hasensor.ComputeSensorSet(
				cfg.BiddingZone, todayPrices, tomorrowPrices,
				actionEngine, fetchResult, now, "", cfg.BestWindowHours, cfg.PricingMode,
			)
			ss.Source = "regulated"
			ss.Quality = "static"
			sensorData.Update(ss)
			slog.Info("seeded dashboard with regulated tariff", "zone", cfg.BiddingZone,
				"mode", cfg.PricingMode, "prices", len(todayPrices), "price", ss.CurrentPrice)
		}
	}

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
		Gate:                featureGate,
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
		DataPath:            dataPath,
	})

	// Start heartbeat sender (install counting)
	var haVersion string
	if supervisor != nil {
		if info, err := supervisor.GetCoreInfo(ctx); err == nil {
			haVersion = info.Version
		}
	}
	go heartbeat.NewSender(heartbeat.Config{
		InstallUUID:  installUUID,
		Product:      "energy",
		AddonVersion: version,
		OSArch:       osArch,
		HAVersion:    haVersion,
	}).Run(ctx)
	slog.Info("heartbeat sender started", "uuid", installUUID)

	// Start markup submitter for crowdsourced EMA tracking (care-app#64 / energy-app#40)
	// Source 1: Enever auto-markup (NL only, 23 suppliers)
	if cfg.IsEneverMode() && cfg.BiddingZone == "NL" {
		go markup.NewSubmitter(markup.SubmitterConfig{
			InstallUUID: installUUID,
			Zone:        cfg.BiddingZone,
			Supplier:    cfg.EneverLeverancier,
			Source:      "enever",
			GetConsumerPrice: func() (float64, bool) {
				data := sensorData.Get()
				if data == nil || data.Source != "enever" {
					return 0, false
				}
				return data.CurrentPrice, data.CurrentPrice > 0
			},
			TaxCache: taxCache,
		}).Run(ctx)
		slog.Info("markup submitter started (enever)", "supplier", cfg.EneverLeverancier)
	}
	// Source 2: Tariff sensor auto-markup (all countries — Tibber, Octopus, etc.)
	if cfg.IsExternalSensorMode() && supervisor != nil {
		sensorSupplier := cfg.SupplierID
		if sensorSupplier == "" && detectedTariffSensor != "" {
			// Extract supplier hint from detected sensor entity ID
			sensorSupplier = hasensor.SupplierHintFromEntity(detectedTariffSensor)
		}
		if sensorSupplier != "" {
			go markup.NewSubmitter(markup.SubmitterConfig{
				InstallUUID: installUUID,
				Zone:        cfg.BiddingZone,
				Supplier:    sensorSupplier,
				Source:      "sensor",
				GetConsumerPrice: func() (float64, bool) {
					data := sensorData.Get()
					if data == nil {
						return 0, false
					}
					return data.CurrentPrice, data.CurrentPrice > 0
				},
				TaxCache: taxCache,
			}).Run(ctx)
			slog.Info("markup submitter started (sensor)", "supplier", sensorSupplier, "entity", cfg.P1SensorEntity)
		}
	}

	// Start delta submitter for per-hour supplier correction factors (ADR_010)
	if cfg.BiddingZone != "" {
		// Enever: submit deltas for all 23 NL suppliers (any mode, as long as token is available)
		if cfg.BiddingZone == "NL" && cfg.EneverToken != "" {
			go delta.NewSubmitter(delta.SubmitterConfig{
				InstallUUID: installUUID,
				Zone:        cfg.BiddingZone,
				Source:      "enever",
				TaxCache:    taxCache,
				GetWholesalePrices: func(ctx context.Context, zone string) ([]delta.WholesalePrice, error) {
					return delta.FetchWholesalePrices(ctx, zone)
				},
				GetDayAheadPrices: func(ctx context.Context, supplier string) ([]delta.HourlyConsumerPrice, error) {
					enever := &collector.Enever{Token: cfg.EneverToken, Leverancier: supplier}
					today, err := enever.FetchDayAhead(ctx, "NL", time.Now())
					if err != nil {
						return nil, err
					}
					// Also try tomorrow
					tomorrow, _ := enever.FetchDayAhead(ctx, "NL", time.Now().Add(24*time.Hour))
					all := append(today, tomorrow...)
					result := make([]delta.HourlyConsumerPrice, 0, len(all))
					for _, p := range all {
						if p.IsConsumer && p.PriceEUR > 0 {
							result = append(result, delta.HourlyConsumerPrice{
								Timestamp: p.Timestamp,
								PriceKWh:  p.PriceEUR,
							})
						}
					}
					return result, nil
				},
				Suppliers: func() []string {
					suppliers := make([]string, 0, len(collector.Leveranciers))
					for k := range collector.Leveranciers {
						suppliers = append(suppliers, k)
					}
					return suppliers
				},
			}).Run(ctx)
			slog.Info("delta submitter started (enever)", "suppliers", len(collector.Leveranciers))
		}

		// Sensor mode: submit deltas for ALL detected tariff sensors
		if supervisor != nil {
			allSensors := hasensor.DetectAllTariffSensors(ctx, supervisor)
			svDelta := supervisor
			for _, ds := range allSensors {
				entityID := ds.EntityID
				supplier := ds.Supplier
				go delta.NewSubmitter(delta.SubmitterConfig{
					InstallUUID: installUUID,
					Zone:        cfg.BiddingZone,
					Source:      "sensor",
					TaxCache:    taxCache,
					GetWholesalePrices: func(ctx context.Context, zone string) ([]delta.WholesalePrice, error) {
						return delta.FetchWholesalePrices(ctx, zone)
					},
					GetDayAheadPrices: func(ctx context.Context, _ string) ([]delta.HourlyConsumerPrice, error) {
						forecast, err := delta.ReadSensorForecast(ctx, svDelta, entityID)
						if err != nil {
							return nil, err
						}
						result := make([]delta.HourlyConsumerPrice, 0, len(forecast))
						for _, f := range forecast {
							if f.PriceKWh > 0 {
								result = append(result, delta.HourlyConsumerPrice{
									Timestamp: f.Timestamp,
									PriceKWh:  f.PriceKWh,
								})
							}
						}
						return result, nil
					},
					Suppliers: func() []string { return []string{supplier} },
				}).Run(ctx)
				slog.Info("delta submitter started (sensor)", "supplier", supplier, "entity", entityID)
			}
		}
	}

	// ADR_010: delta cache for non-sensor installs using ENTSO-E + supplier deltas
	if cfg.PricingMode == config.ModeAuto && cfg.SupplierID != "" && cfg.BiddingZone != "" {
		dc := delta.NewCache()
		go dc.RunFetcher(ctx, cfg.BiddingZone, cfg.SupplierID)
		normalizer.SetDeltaLookup(dc.Get)
		srv.SetDeltaCache(dc)
		slog.Info("delta: consumer cache enabled", "zone", cfg.BiddingZone, "supplier", cfg.SupplierID)
	}

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

	// Remove MQTT discovery topics so sensors don't become orphans
	if mqttPub != nil {
		mqttPub.RemoveAllDiscovery()
		mqttPub.Close()
	}

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
