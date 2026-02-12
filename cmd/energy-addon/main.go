// Command energy-addon runs the Synctacles Energy Home Assistant addon.
// This serves the web UI, price collectors, action engine, and sensor publisher.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/synctacles/energy-go/internal/collector"
	"github.com/synctacles/energy-go/internal/config"
	"github.com/synctacles/energy-go/internal/countries"
	"github.com/synctacles/energy-go/internal/engine"
	"github.com/synctacles/energy-go/internal/ha"
	"github.com/synctacles/energy-go/internal/hasensor"
	"github.com/synctacles/energy-go/internal/license"
	"github.com/synctacles/energy-go/internal/models"
	"github.com/synctacles/energy-go/internal/state"
	"github.com/synctacles/energy-go/internal/store"
	"github.com/synctacles/energy-go/internal/web"
)

var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("starting energy-addon", "version", version)

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

	// Initialize license validator with 14-day free trial
	licenseValidator := license.NewValidator(cfg.LicenseKey, dataPath)
	licenseValidator.InitTrial()

	if cfg.HasLicense() {
		if err := licenseValidator.ValidateOnce(context.Background()); err != nil {
			slog.Warn("license validation failed", "error", err)
		}
	}
	slog.Info("license status", "tier", licenseValidator.Tier(), "pro", licenseValidator.IsPro(),
		"trial", licenseValidator.IsTrial(), "trial_days_left", licenseValidator.TrialDaysLeft())

	// Build price source chain for the configured zone
	sources := buildSourceChain(cfg, registry)
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
	if supervisor != nil {
		publishers = append(publishers, hasensor.NewRESTPublisher(supervisor))

		// Detect MQTT broker
		mqttHost, found := hasensor.DetectMQTTBroker(context.Background(), supervisor)
		if found {
			mqtt := hasensor.NewMQTTPublisher(mqttHost, 1883)
			if err := mqtt.Connect(); err != nil {
				slog.Warn("MQTT connection failed, using REST only", "error", err)
			} else {
				publishers = append(publishers, mqtt)
				slog.Info("MQTT publisher enabled (dual publishing)")
			}
		}
	}

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

		// Update shared sensor data for web dashboard
		sensorData.Update(sensorSet)

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
		stateStore.Save(st)

		// Publish to all publishers
		for _, pub := range publishers {
			if err := hasensor.PublishAll(ctx, pub, sensorSet, licenseValidator.IsPro(), powerTracker); err != nil {
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

	// Start scheduler
	scheduler := engine.NewScheduler(fallbackMgr, normalizer, actionEngine, cfg.BiddingZone, updateFn)
	go scheduler.Run(ctx)

	// Create web server
	srv := web.NewServer(web.Deps{
		Config:              cfg,
		StateStore:          stateStore,
		SensorData:          sensorData,
		Supervisor:          supervisor,
		Fallback:            fallbackMgr,
		License:             licenseValidator,
		Version:             version,
		DetectedPowerSensor: detectedPowerSensor,
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
	httpSrv.Shutdown(shutdownCtx)

	slog.Info("energy-addon stopped")
}

// buildSourceChain creates the prioritized source list for a bidding zone.
// If Enever is enabled and zone is NL, it's prepended as highest priority.
func buildSourceChain(cfg *config.Config, registry *models.ZoneRegistry) []collector.PriceSource {
	var chain []collector.PriceSource

	// Enever as optional highest priority (NL only, user BYO key)
	if cfg.HasEnever() && cfg.BiddingZone == "NL" {
		chain = append(chain, &collector.Enever{
			Token:       cfg.EneverToken,
			Leverancier: cfg.EneverLeverancier,
		})
		slog.Info("Enever enabled", "leverancier", cfg.EneverLeverancier)
	}

	cc, ok := registry.GetCountryForZone(cfg.BiddingZone)
	if !ok {
		slog.Warn("no country config for zone, using Energy-Charts only", "zone", cfg.BiddingZone)
		chain = append(chain, &collector.EnergyCharts{})
		return chain
	}

	sourceMap := map[string]collector.PriceSource{
		"easyenergy":       &collector.EasyEnergy{},
		"frank":            &collector.FrankEnergie{},
		"energycharts":     &collector.EnergyCharts{},
		"energidataservice": &collector.EnergiDataService{},
		"awattar":          &collector.AWATTar{},
		"omie":             &collector.OMIE{},
		"spothinta":        &collector.SpotHinta{},
	}

	for _, sp := range cc.Sources {
		if src, ok := sourceMap[sp.Name]; ok {
			chain = append(chain, src)
		}
	}

	// Always add Energy-Charts as final fallback if not already in chain
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

	return chain
}
