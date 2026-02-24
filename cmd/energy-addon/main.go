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

	"github.com/synctacles/energy-backend/pkg/collector"
	"github.com/synctacles/energy-app/internal/config"
	"github.com/synctacles/energy-app/internal/gate"
	"github.com/synctacles/energy-app/internal/heartbeat"
	"github.com/synctacles/energy-app/internal/plan"
	"github.com/synctacles/energy-backend/pkg/countries"
	"github.com/synctacles/energy-backend/pkg/engine"
	"github.com/synctacles/energy-app/internal/ha"
	"github.com/synctacles/energy-app/internal/hasensor"
	"github.com/synctacles/energy-app/internal/license"
	"github.com/synctacles/energy-backend/pkg/models"
	"github.com/synctacles/energy-app/internal/state"
	"github.com/synctacles/energy-backend/pkg/store"
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

	// Build plan registry and resolve active plan
	planRegistry := plan.Build(registry)
	if cfg.PlanID != "" {
		if p := planRegistry.Get(cfg.PlanID); p != nil {
			cfg.ApplyPlan(p.Zone, p.EneverSupplier)
			slog.Info("plan applied", "plan", p.ID, "zone", p.Zone, "enever", p.HasEnever())
		} else {
			slog.Warn("unknown plan ID, using config defaults", "plan", cfg.PlanID)
		}
	} else {
		// Resolve plan from legacy config fields for display purposes
		cfg.PlanID = planRegistry.ResolveFromConfig(cfg.BiddingZone, cfg.EneverEnabled, cfg.EneverLeverancier)
		slog.Info("plan resolved from config", "plan", cfg.PlanID)
	}

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

	// Initialize license validator (all features free)
	licenseValidator := license.NewValidator(cfg.LicenseKey, dataPath)
	slog.Info("license status", "tier", licenseValidator.Tier(), "pro", licenseValidator.IsPro())

	// Load persistent install UUID (shared by heartbeat, telemetry, and lease)
	installUUID := telemetry.LoadInstallUUID(dataPath)
	osArch := telemetry.OSArch()
	slog.Info("install identity", "uuid", installUUID, "arch", osArch)

	// Initialize feature gate (controls what's available based on registration + heartbeat)
	featureGate := gate.New(cfg.BiddingZone, cfg.HasEnever(), cfg.LicenseKey)
	slog.Info("feature gate initialized",
		"heartbeat_enabled", cfg.HeartbeatEnabled,
		"zone", cfg.BiddingZone,
		"enever", cfg.HasEnever(),
		"registered", cfg.HasLicense(),
	)

	// Build price source chain for the configured zone
	sources, synctaclesSource := buildSourceChain(cfg, registry, installUUID)
	slog.Info("source chain configured", "zone", cfg.BiddingZone, "sources", len(sources))

	// Initialize engine components
	normalizer := engine.NewNormalizer(registry, cfg.Coefficient)
	actionEngine := engine.NewActionEngine(cfg.GoThreshold, cfg.AvoidThreshold)
	fallbackMgr := engine.NewFallbackManager(sources, priceCache)

	// Initialize shared sensor data (for web dashboard)
	sensorData := web.NewSensorData()

	// Initialize power tracker (for Live Cost, Savings, Usage Score)
	// Auto-detect power sensor if not explicitly configured
	var detectedPowerSensor string
	if !cfg.HasPowerSensor() && supervisor != nil {
		if detected := hasensor.DetectPowerSensor(context.Background(), supervisor); detected != "" {
			detectedPowerSensor = detected
			cfg.PowerSensorEntity = detected
			slog.Info("power sensor auto-detected", "entity", detected)
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
		// Update lease from Synctacles server response (for fallback authorization)
		if synctaclesSource != nil {
			if l := synctaclesSource.LastLease(); l != nil {
				featureGate.UpdateLease(l)
			}
		}

		// Feature gate: check if actions are allowed (logged for diagnostics)
		if !featureGate.CanUseActions() {
			slog.Debug("actions disabled (registration required)")
		}

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

	// Start background license re-validation (monthly)
	licenseValidator.RunBackground(ctx)

	// Start heartbeat sender (always-on when enabled, controls feature gate)
	if cfg.HeartbeatEnabled {
		hb := heartbeat.NewSender(heartbeat.Config{
			InstallUUID:  installUUID,
			Product:      "energy",
			AddonVersion: version,
			OSArch:       osArch,
			OnSuccess:    func() { featureGate.SetHeartbeatOK(true) },
			OnFailure:    func() { featureGate.SetHeartbeatOK(false) },
		})
		go hb.Run(ctx)
		slog.Info("heartbeat sender started")
	} else {
		slog.Warn("heartbeat disabled — remote features will be limited")
	}

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
	})
	telemetrySender.RunBackground(ctx)

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
		License:             licenseValidator,
		Gate:                featureGate,
		Version:             version,
		DetectedPowerSensor: detectedPowerSensor,
		AddonSlug:           addonSlug,
		PlanRegistry:        planRegistry,
	})

	addr := ":" + strconv.Itoa(cfg.IngressPort)
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      srv.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", addr, "zone", cfg.BiddingZone, "pro", licenseValidator.IsPro())
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
// Returns the source chain and a reference to the SynctaclesAPI source (for lease access).
//
// Hybrid architecture:
//
//	Tier 0: Synctacles API (central server, pre-computed consumer prices)
//	Tier 1+: Direct sources (only activated when Synctacles server is unreachable)
//
// When the Synctacles server is healthy, the addon makes exactly 1 API call.
// When it's down, the circuit breaker opens and the full fallback chain takes over,
// including Enever (NL), EasyEnergy, Energy-Charts, etc.
func buildSourceChain(cfg *config.Config, registry *models.ZoneRegistry, installUUID string) ([]collector.PriceSource, *collector.SynctaclesAPI) {
	var chain []collector.PriceSource
	var synctaclesSource *collector.SynctaclesAPI

	// When user explicitly chose an Enever supplier, use it as primary source.
	// Enever provides exact consumer prices for the user's contract.
	if cfg.HasEnever() && cfg.BiddingZone == "NL" {
		chain = append(chain, &collector.Enever{
			Token:       cfg.EneverToken,
			Leverancier: cfg.EneverLeverancier,
		})
		slog.Info("Enever enabled as primary source", "leverancier", cfg.EneverLeverancier)
	}

	// Synctacles central server (primary for non-Enever, fallback for Enever users)
	if cfg.HasSynctaclesServer() {
		synctaclesSource = &collector.SynctaclesAPI{
			BaseURL:     cfg.SynctaclesURL,
			InstallUUID: installUUID,
		}
		chain = append(chain, synctaclesSource)
		if cfg.HasEnever() {
			slog.Info("Synctacles API enabled as fallback", "url", cfg.SynctaclesURL)
		} else {
			slog.Info("Synctacles API enabled as primary source", "url", cfg.SynctaclesURL)
		}
	}

	cc, ok := registry.GetCountryForZone(cfg.BiddingZone)
	if !ok {
		slog.Warn("no country config for zone, using Energy-Charts only", "zone", cfg.BiddingZone)
		chain = append(chain, &collector.EnergyCharts{})
		return chain, synctaclesSource
	}

	sourceMap := map[string]collector.PriceSource{
		"easyenergy":        &collector.EasyEnergy{},
		"frank":             &collector.FrankEnergie{},
		"energycharts":      &collector.EnergyCharts{},
		"energidataservice": &collector.EnergiDataService{},
		"awattar":           &collector.AWATTar{},
		"omie":              &collector.OMIE{},
		"spothinta":         &collector.SpotHinta{},
	}

	for _, sp := range cc.Sources {
		if src, ok := sourceMap[sp.Name]; ok {
			chain = append(chain, src)
		}
	}

	// Always add Energy-Charts as final direct fallback if not already in chain
	hasEC := false
	for _, src := range chain {
		if src.Name() == "energycharts" {
			hasEC = true
			break
		}
	}
	if !hasEC {
		chain = append(chain, &collector.EnergyCharts{})
	}

	return chain, synctaclesSource
}
