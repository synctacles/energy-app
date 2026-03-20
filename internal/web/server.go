// Package web provides the energy addon HTTP server with embedded SPA.
package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/synctacles/energy-app/internal/config"
	"github.com/synctacles/energy-app/pkg/engine"
	"github.com/synctacles/energy-app/internal/gate"
	"github.com/synctacles/energy-app/internal/ha"
	"github.com/synctacles/energy-app/pkg/kb"
	"github.com/synctacles/energy-app/pkg/models"
	"github.com/synctacles/energy-app/pkg/platform"
	"github.com/synctacles/energy-app/pkg/store"
	"github.com/synctacles/energy-app/internal/state"
)

//go:embed static/*
var staticFS embed.FS

// Server is the energy addon HTTP server.
type Server struct {
	cfg                 *config.Config
	router              chi.Router
	stateStore          *state.Store
	sensorData          *SensorData
	supervisor          *ha.SupervisorClient
	fallback            *engine.FallbackManager
	featureGate         *gate.Gate
	version             string
	detectedPowerSensor  string
	detectedTariffSensor string
	addonSlug           string
	zoneRegistry        *models.ZoneRegistry
	taxCache            *engine.TaxProfileCache
	normalizer          *engine.Normalizer
	scheduler           *engine.Scheduler
	sqliteCache         *store.SQLiteCache
	installUUID         string
	kbClient            *kb.Client
	dataPath            string
	deltaCache          deltaLenStale // ADR_010: optional delta cache for calibration indicator
}

// SetDeltaCache sets the delta cache for the calibration indicator (ADR_010).
func (s *Server) SetDeltaCache(dc deltaLenStale) {
	s.deltaCache = dc
}

// deltaLenStale is the interface for the delta cache used in the calibration indicator.
type deltaLenStale interface {
	Len() int
	IsStale() bool
	Get(t time.Time) (float64, bool)
}

// Deps holds dependencies for the web server.
type Deps struct {
	Config              *config.Config
	StateStore          *state.Store
	SensorData          *SensorData
	Supervisor          *ha.SupervisorClient
	Fallback            *engine.FallbackManager
	Gate                *gate.Gate
	Version             string
	DetectedPowerSensor  string
	DetectedTariffSensor string
	AddonSlug           string
	ZoneRegistry        *models.ZoneRegistry
	TaxCache            *engine.TaxProfileCache
	Normalizer          *engine.Normalizer
	Scheduler           *engine.Scheduler
	SQLiteCache         *store.SQLiteCache
	InstallUUID         string
	DataPath            string
	DeltaCache          deltaLenStale // ADR_010
}

// NewServer creates a new energy addon web server.
func NewServer(deps Deps) *Server {
	s := &Server{
		cfg:                 deps.Config,
		stateStore:          deps.StateStore,
		sensorData:          deps.SensorData,
		supervisor:          deps.Supervisor,
		fallback:            deps.Fallback,
		featureGate:         deps.Gate,
		version:             deps.Version,
		detectedPowerSensor:  deps.DetectedPowerSensor,
		detectedTariffSensor: deps.DetectedTariffSensor,
		addonSlug:           deps.AddonSlug,
		zoneRegistry:        deps.ZoneRegistry,
		taxCache:            deps.TaxCache,
		normalizer:          deps.Normalizer,
		scheduler:           deps.Scheduler,
		sqliteCache:         deps.SQLiteCache,
		installUUID:         deps.InstallUUID,
		kbClient:            kb.NewClient("", deps.InstallUUID),
		dataPath:            deps.DataPath,
		deltaCache:          deps.DeltaCache,
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Get("/version", s.handleVersion)
		r.Get("/status", s.handleStatus)

		// Prices
		r.Get("/prices/today", s.handlePricesToday)
		r.Get("/prices/tomorrow", s.handlePricesTomorrow)

		// Action
		r.Get("/action", s.handleAction)

		// Dashboard (bundled response)
		r.Get("/dashboard", s.handleDashboard)

		// Config & Zones
		r.Get("/config", s.handleConfig)
		r.Post("/config", s.handleConfigSave)
		r.Get("/zones", s.handleZones)
		r.Get("/zone-detect", s.handleZoneDetect)
		r.Get("/tax-breakdown", s.handleTaxBreakdown)
		r.Get("/sensors/tariff", s.handleTariffSensors)
		r.Get("/suppliers", s.handleSuppliers)
		r.Get("/country-defaults", s.handleCountryDefaults)

		// Wizard (onboarding)
		r.Get("/wizard-data", s.handleWizardData)
		r.Post("/crowdsource-submit", s.handleCrowdsourceSubmit)

		// Sources health
		r.Get("/sources", s.handleSources)

		// Debug: verify embedded i18n files
		r.Get("/debug/i18n", func(w http.ResponseWriter, r *http.Request) {
			lang := r.URL.Query().Get("lang")
			if lang == "" {
				lang = "en"
			}
			data, err := staticFS.ReadFile("static/i18n/" + lang + ".json")
			if err != nil {
				writeJSON(w, map[string]string{"error": err.Error()})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		})

		// Cache transparency
		r.Get("/cache", s.handleCacheView)
		r.Post("/cache/reset", s.handleCacheReset)

		// KB search (proxied with product=energy)
		r.Get("/kb/search", s.handleKBSearch)

		// Feedback
		r.Get("/feedback/sysinfo", s.handleFeedbackSysInfo)
		r.Post("/feedback/rating", s.handleFeedbackRating)
		r.Post("/feedback/bug", s.handleFeedbackBug)

		// Zone request (unsupported region crowdsource)
		r.Post("/zone-request", s.handleZoneRequest)

		// GDPR data deletion
		r.Post("/delete-data", s.handleDeleteData)
	})

	// Static files and SPA — no-cache headers to prevent stale i18n/JS across app updates
	staticSub, err := fs.Sub(staticFS, "static")
	if err == nil {
		fileServer := http.FileServer(http.FS(staticSub))
		r.Handle("/static/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
			http.StripPrefix("/static/", fileServer).ServeHTTP(w, r)
		}))
	}

	// SPA fallback
	r.Get("/*", s.handleSPA)

	s.router = r
	return s
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.router
}

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "version": s.version})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"version":       s.version,
		"zone":          s.cfg.BiddingZone,
		"alerts_enabled": s.cfg.HasAlerts(),
	}
	if s.addonSlug != "" {
		resp["addon_slug"] = s.addonSlug
	}

	// Detect locale from HA supervisor config
	if s.supervisor != nil {
		if cfg, err := s.supervisor.GetConfig(r.Context()); err == nil {
			if lang, ok := cfg["language"].(string); ok {
				resp["locale"] = lang
			}
		}
	}

	writeJSON(w, resp)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	st := s.stateStore.Load()
	writeJSON(w, map[string]any{
		"zone":          st.Zone,
		"current_price": st.CurrentPrice,
		"action":        st.Action,
		"quality":       st.Quality,
		"last_fetch":    st.LastFetch,
		"price_source":  st.PriceSource,
	})
}

func (s *Server) handlePricesToday(w http.ResponseWriter, r *http.Request) {
	data := s.sensorData.Get()
	if data == nil {
		writeJSON(w, map[string]any{"prices": []any{}, "status": "waiting"})
		return
	}

	prices := make([]map[string]any, 0, len(data.TodayPrices))
	for _, p := range data.TodayPrices {
		prices = append(prices, map[string]any{
			"hour":  p.Timestamp.Format("15:04"),
			"price": p.PriceEUR,
		})
	}
	writeJSON(w, map[string]any{
		"prices":  prices,
		"average": data.Stats.Average,
		"min":     data.Stats.Min,
		"max":     data.Stats.Max,
		"zone":    data.Zone,
	})
}

func (s *Server) handlePricesTomorrow(w http.ResponseWriter, r *http.Request) {
	data := s.sensorData.Get()
	if data == nil {
		writeJSON(w, map[string]any{"prices": []any{}, "status": "waiting"})
		return
	}

	prices := make([]map[string]any, 0, len(data.TomorrowPrices))
	for _, p := range data.TomorrowPrices {
		prices = append(prices, map[string]any{
			"hour":  p.Timestamp.Format("15:04"),
			"price": p.PriceEUR,
		})
	}
	writeJSON(w, map[string]any{
		"prices":   prices,
		"preview":  data.Tomorrow.Status,
		"avg":      data.Tomorrow.AvgPrice,
		"zone":     data.Zone,
	})
}

func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	data := s.sensorData.Get()
	if data == nil {
		writeJSON(w, map[string]any{"action": "WAIT", "reason": "waiting for data"})
		return
	}

	writeJSON(w, map[string]any{
		"action":        data.Action.Action,
		"reason":        data.Action.Reason,
		"deviation_pct": data.Action.DeviationPct,
		"current_price": data.Action.CurrentPrice,
		"average_price": data.Action.AveragePrice,
		"quality":       data.Action.Quality,
	})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := s.sensorData.Get()
	st := s.stateStore.Load()

	if data == nil {
		writeJSON(w, map[string]any{
			"status":  "waiting",
			"zone":    s.cfg.BiddingZone,
			"version": s.version,
		})
		return
	}

	// Load zone timezone for local time display
	loc := s.zoneLoc()

	// Build hourly prices for chart (local time labels)
	todayPrices := make([]map[string]any, 0, len(data.TodayPrices))
	for _, p := range data.TodayPrices {
		todayPrices = append(todayPrices, map[string]any{
			"hour":  p.Timestamp.In(loc).Format("15:04"),
			"price": p.PriceEUR,
		})
	}

	// Determine if zone has wholesale market data
	zoneInfo, _ := s.zoneRegistry.GetZone(data.Zone)
	hasWholesale := zoneInfo.HasWholesale()

	dashboard := map[string]any{
		"status":        "ok",
		"version":       s.version,
		"zone":          data.Zone,
		"pricing_mode":  s.cfg.PricingMode,
		"has_wholesale": hasWholesale,
		"current_price": map[string]any{
			"price":       fmt.Sprintf("%.4f", data.CurrentPrice),
			"unit":        "EUR/kWh",
			"source":      data.Source,
			},
		"action": map[string]any{
			"action":          data.Action.Action,
			"reason":          data.Action.Reason,
			"deviation":       data.Action.DeviationPct,
			"next_transition": data.Action.NextTransition,
			"next_rate":       data.Action.NextRate,
		},
		"stats": map[string]any{
			"average":        data.Stats.Average,
			"min":            data.Stats.Min,
			"max":            data.Stats.Max,
			"cheapest_hour":  utcHourToLocal(data.Stats.CheapestHour, loc),
			"expensive_hour": utcHourToLocal(data.Stats.ExpensiveHour, loc),
		},
		"prices_today": todayPrices,
		"tomorrow": map[string]any{
			"status":   data.Tomorrow.Status,
			"avg":      data.Tomorrow.AvgPrice,
			"hours":    len(data.TomorrowPrices),
		},
		"quality":    data.Quality,
		"updated_at": data.UpdatedAt.In(loc).Format(time.RFC3339),
		"last_fetch": st.LastFetch,
	}

	if data.BestWindow != nil {
		dashboard["best_window"] = map[string]any{
			"start":    utcHourToLocal(data.BestWindow.StartHour, loc),
			"end":      utcHourToLocal(data.BestWindow.EndHour, loc),
			"avg":      data.BestWindow.AvgPrice,
			"duration": data.BestWindow.Duration,
		}
	}

	// Price resolution: "PT15M" or "PT60M"
	slotDur := engine.DetectSlotDuration(data.TodayPrices)
	if slotDur == 15*time.Minute {
		dashboard["resolution"] = "PT15M"
	} else {
		dashboard["resolution"] = "PT60M"
	}

	// Source info for chart attribution
	sourceInfo := map[string]any{
		"source":      data.Source,
		"quality":     data.Quality,
	}
	dashboard["source_info"] = sourceInfo

	// Calibration supplier for chart label (e.g. "Zonneplan (NL) calibrated")
	// Set when: sensor mode (supplier from entity) or supplier selected in settings
	calibrationSupplier := s.cfg.SupplierID
	if calibrationSupplier == "" && s.cfg.P1SensorEntity != "" {
		entity := strings.ToLower(s.cfg.P1SensorEntity)
		for _, name := range []string{"zonneplan", "tibber", "octopus", "nordpool", "easyenergy", "frank"} {
			if strings.Contains(entity, name) {
				calibrationSupplier = name
				break
			}
		}
	}
	if calibrationSupplier != "" {
		dashboard["calibration_supplier"] = calibrationSupplier
	}

	// Tax data source: "worker", "consumer", "embedded", "none"
	// "consumer" = prices already include taxes (Worker consumer prices)
	// "worker"   = live Worker tax profile applied to wholesale
	// "embedded" = fallback tax defaults (less accurate)
	// "none"     = no tax data at all (wholesale only)
	if s.normalizer != nil {
		taxSource := s.normalizer.TaxSource()
		dashboard["tax_source"] = taxSource
		// Regulated tariffs are all-in consumer prices — no tax gap possible
		isRegulated := data.Source == "regulated"
		if taxSource == "none" && !isRegulated {
			dashboard["degraded"] = true
			dashboard["degraded_reason"] = "no_tax_data"
		}
	}

	// ADR_010: price accuracy level
	// "exact"      = sensor (real supplier price)
	// "calibrated" = ENTSO-E + per-hour delta (crowdsourced average)
	// "estimated"  = ENTSO-E + static tax only (no supplier markup data)
	if s.cfg.IsExternalSensorMode() {
		dashboard["price_accuracy"] = "exact"
	} else if s.deltaCache != nil && s.deltaCache.Len() > 0 && !s.deltaCache.IsStale() {
		dashboard["price_accuracy"] = "calibrated"
		dashboard["price_calibration"] = map[string]any{
			"hours": s.deltaCache.Len(),
		}
	} else {
		dashboard["price_accuracy"] = "estimated"
	}

	// Setup hints: suggest configuration improvements
	if hints := s.buildSetupHints(); len(hints) > 0 {
		dashboard["setup_hints"] = hints
	}

	writeJSON(w, dashboard)
}

// buildSetupHints returns actionable hints when the user's config can be improved.
func (s *Server) buildSetupHints() []map[string]string {
	var hints []map[string]string
	mode := s.cfg.PricingMode
	isSensorMode := mode == "external_sensor" || mode == "p1_meter" || mode == "meter_tariff"
	hasSensor := s.detectedTariffSensor != ""
	sensorConfigured := s.cfg.P1SensorEntity != ""

	// Hint: day-ahead sensor available but not configured anywhere
	// (not in sensor mode, and not saved as p1_sensor_entity for delta)
	if !isSensorMode && hasSensor && !sensorConfigured {
		hints = append(hints, map[string]string{
			"id":      "sensor_available",
			"sensor":  s.detectedTariffSensor,
			"action":  "settings",
			"section": "external_sensor",
		})
	}

	return hints
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"pricing_mode":            s.cfg.PricingMode,
		"zone":                    s.cfg.BiddingZone,
		"currency":                s.cfg.Currency,
		"go_threshold":            s.cfg.GoThreshold,
		"avoid_threshold":         s.cfg.AvoidThreshold,
		"best_window_hours":       s.cfg.BestWindowHours,
		"has_power_sensor":        s.cfg.HasPowerSensor(),
		"power_sensor":            s.cfg.PowerSensorEntity,
		"supplier_markup":         s.cfg.SupplierMarkup,
		"supplier_id":             s.cfg.SupplierID,
		"manual_vat_rate":         s.cfg.ManualVATRate,
		"manual_energy_tax":       s.cfg.ManualEnergyTax,
		"manual_surcharges":       s.cfg.ManualSurcharges,
		"manual_network_tariff":   s.cfg.ManualNetworkTariff,
		"p1_sensor_entity":        s.cfg.P1SensorEntity,
		"fixed_rate_price":        s.cfg.FixedRatePrice,
		"tou_config":              s.cfg.TOUConfigJSON,
		"debug_mode":              s.cfg.DebugMode,
		"disclaimer_accepted":     s.cfg.DisclaimerAccepted,
		"privacy_accepted":        s.cfg.PrivacyAccepted,
		"detected_power_sensor":   s.detectedPowerSensor,
		"detected_tariff_sensor":  s.detectedTariffSensor,
		"onboarding_completed":    s.cfg.OnboardingCompleted,
		"telemetry_enabled":       s.cfg.TelemetryEnabled,
		"purged":                  false,
	}

	writeJSON(w, resp)
}

func (s *Server) handleConfigSave(w http.ResponseWriter, r *http.Request) {
	if s.supervisor == nil {
		writeError(w, http.StatusServiceUnavailable, "not running inside HA app")
		return
	}

	var incoming map[string]any
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Read current options from Supervisor
	current, err := s.supervisor.GetAddonOptions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read current options")
		return
	}

	// Merge allowed fields
	allowed := []string{
		"pricing_mode", "zone", "go_threshold", "avoid_threshold", "best_window_hours",
		"supplier_markup", "supplier_id",
		"manual_vat_rate", "manual_energy_tax", "manual_surcharges", "manual_network_tariff",
		"p1_sensor_entity", "fixed_rate_price", "tou_config", "power_sensor", "debug_mode",
		"disclaimer_accepted", "privacy_accepted", "onboarding_completed", "telemetry_enabled",
	}
	for _, key := range allowed {
		if val, ok := incoming[key]; ok {
			current[key] = val
		}
	}
	// Always preserve in-memory supplier (set by dropdown persistConfig, may not be in payload)
	if _, ok := incoming["supplier_id"]; !ok && s.cfg.SupplierID != "" {
		current["supplier_id"] = s.cfg.SupplierID
		current["supplier_markup"] = s.cfg.SupplierMarkup
	}

	// Validate manual tax inputs (CC_INSTRUCTION §10 ranges)
	vatRate, _ := toFloat64(current["manual_vat_rate"])
	energyTax, _ := toFloat64(current["manual_energy_tax"])
	surcharges, _ := toFloat64(current["manual_surcharges"])
	networkTariff, _ := toFloat64(current["manual_network_tariff"])
	if err := config.ValidateTaxInputs(vatRate, energyTax, surcharges, networkTariff); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Write back to Supervisor
	if err := s.supervisor.SetAddonOptions(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save options")
		return
	}

	// Check if source chain needs rebuild BEFORE updating in-memory config
	needsRestart := false
	if v, ok := incoming["pricing_mode"].(string); ok && v != s.cfg.PricingMode {
		needsRestart = true
	}
	if v, ok := incoming["zone"].(string); ok && v != s.cfg.BiddingZone {
		needsRestart = true
	}

	// Invalidate tax cache when zone changes (forces Worker re-fetch for new zone)
	if v, ok := incoming["zone"].(string); ok && v != s.cfg.BiddingZone {
		if s.taxCache != nil {
			s.taxCache.Invalidate(s.cfg.BiddingZone)
		}
	}

	// Update in-memory config for immediate effect
	applyStringField(incoming, "pricing_mode", &s.cfg.PricingMode)
	applyStringField(incoming, "zone", &s.cfg.BiddingZone)
	applyStringField(incoming, "supplier_id", &s.cfg.SupplierID)
	applyStringField(incoming, "p1_sensor_entity", &s.cfg.P1SensorEntity)
	applyStringField(incoming, "tou_config", &s.cfg.TOUConfigJSON)

	applyFloatField(incoming, "go_threshold", &s.cfg.GoThreshold)
	applyFloatField(incoming, "avoid_threshold", &s.cfg.AvoidThreshold)
	if v, ok := incoming["best_window_hours"].(float64); ok && int(v) >= 1 && int(v) <= 8 {
		s.cfg.BestWindowHours = int(v)
	}

	applyFloatField(incoming, "manual_vat_rate", &s.cfg.ManualVATRate)
	applyFloatField(incoming, "manual_energy_tax", &s.cfg.ManualEnergyTax)
	applyFloatField(incoming, "manual_surcharges", &s.cfg.ManualSurcharges)
	applyFloatField(incoming, "manual_network_tariff", &s.cfg.ManualNetworkTariff)
	applyFloatField(incoming, "fixed_rate_price", &s.cfg.FixedRatePrice)

	applyBoolField(incoming, "onboarding_completed", &s.cfg.OnboardingCompleted)
	applyBoolField(incoming, "disclaimer_accepted", &s.cfg.DisclaimerAccepted)
	applyBoolField(incoming, "privacy_accepted", &s.cfg.PrivacyAccepted)
	applyBoolField(incoming, "telemetry_enabled", &s.cfg.TelemetryEnabled)

	if v, ok := incoming["best_window_hours"].(float64); ok {
		s.cfg.BestWindowHours = int(v)
	}
	if v, ok := incoming["supplier_markup"].(float64); ok {
		s.cfg.SupplierMarkup = v
		if s.normalizer != nil {
			s.normalizer.SetSupplierMarkup(v)
		}
		if s.scheduler != nil {
			s.scheduler.TriggerFetch()
		}
	}

	// Save consent flags to dedicated file (survives Supervisor options resets)
	_, hasD := incoming["disclaimer_accepted"]
	_, hasP := incoming["privacy_accepted"]
	_, hasO := incoming["onboarding_completed"]
	if s.dataPath != "" && (hasD || hasP || hasO) {
		if err := config.SaveConsent(s.dataPath, config.ConsentState{
			DisclaimerAccepted:  s.cfg.DisclaimerAccepted,
			PrivacyAccepted:    s.cfg.PrivacyAccepted,
			OnboardingCompleted: s.cfg.OnboardingCompleted,
		}); err != nil {
			slog.Warn("failed to save consent file", "error", err)
		}
	}

	// Save non-schema settings to backup file (protects against HA Options page wipe)
	// IMPORTANT: must happen BEFORE restart return, otherwise settings are lost on mode change
	if s.dataPath != "" {
		settingsMap := config.BuildSettingsMap(s.cfg)
		if err := config.SaveSettingsFile(config.SettingsFilePath(s.dataPath), settingsMap); err != nil {
			slog.Warn("failed to save settings backup", "error", err)
		}
	}

	if needsRestart && s.supervisor != nil {
		writeJSON(w, map[string]string{"status": "restarting", "message": "Settings saved. Restarting to apply source chain changes..."})
		// Delay restart so HTTP response is sent first
		go func() {
			time.Sleep(500 * time.Millisecond)
			ctx2, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := s.supervisor.RestartSelf(ctx2); err != nil {
				slog.Error("self-restart failed", "error", err)
			}
		}()
		return
	}

	writeJSON(w, map[string]string{"status": "saved", "message": "Settings saved."})
}

// handleCountryDefaults returns country-specific defaults for a zone.
// Used by the settings UI to pre-fill fields when the user changes zone.
func (s *Server) handleCountryDefaults(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = s.cfg.BiddingZone
	}

	cc, ok := s.zoneRegistry.GetCountryForZone(zone)
	if !ok {
		writeJSON(w, map[string]any{"error": "unknown zone"})
		return
	}

	resp := map[string]any{
		"country":  cc.Country,
		"name":     cc.Name,
		"currency": cc.Currency,
	}

	// Check if zone has ENTSO-E wholesale data
	zoneInfo, _ := s.zoneRegistry.GetZone(zone)
	hasWholesale := zoneInfo.HasWholesale()

	// Available pricing modes for this country/zone
	modes := []string{"auto", "manual", "external_sensor", "fixed", "tou"}
	// Non-wholesale zones: only allow fixed and tou (no ENTSO-E spot prices, regulated tariffs apply)
	if !hasWholesale {
		modes = []string{"fixed", "tou"}
	}
	resp["pricing_modes"] = modes
	resp["has_wholesale"] = hasWholesale

	// TOU presets: zone-level override if available, else country-level
	touPresets := s.zoneRegistry.GetTOUPresets(zone)
	if len(touPresets) > 0 {
		resp["tou_presets"] = touPresets
	} else {
		resp["tou_presets"] = []any{}
	}

	// Suppliers
	if len(cc.Suppliers) > 0 {
		resp["suppliers"] = cc.Suppliers
	} else {
		resp["suppliers"] = []any{}
	}

	// Tax defaults: zone-level override if available, else country-level
	taxDefaults := s.zoneRegistry.GetTaxDefaults(zone)
	if taxDefaults != nil {
		resp["tax_defaults"] = taxDefaults
	}

	// Regulated tariffs for non-wholesale zones
	if zoneInfo.RegulatedTariffs != nil {
		resp["regulated_tariffs"] = zoneInfo.RegulatedTariffs
	}

	writeJSON(w, resp)
}

func (s *Server) handleSuppliers(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = s.cfg.BiddingZone
	}
	// Fetch suppliers with delta data from Worker
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://energy-data.synctacles.com/api/v1/energy/supplier-deltas?zone="+zone, nil)
	if err != nil {
		writeJSON(w, []any{})
		return
	}
	req.Header.Set("User-Agent", "SynctaclesEnergy/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		// Fallback to static YAML suppliers
		cc, ok := s.zoneRegistry.GetCountryForZone(zone)
		if !ok || len(cc.Suppliers) == 0 {
			writeJSON(w, []any{})
			return
		}
		writeJSON(w, cc.Suppliers)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Suppliers []struct {
			ID       string  `json:"id"`
			AvgDelta float64 `json:"avg_delta"`
			Hours    int     `json:"hours"`
		} `json:"suppliers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Suppliers) == 0 {
		writeJSON(w, []any{})
		return
	}
	writeJSON(w, result.Suppliers)
}

func (s *Server) handleZones(w http.ResponseWriter, r *http.Request) {
	// Build zone list grouped by country from zone registry
	type zoneEntry struct {
		Code    string `json:"code"`
		Name    string `json:"name"`
		Country string `json:"country"`
	}
	type zoneGroup struct {
		Name  string      `json:"name"`
		Zones []zoneEntry `json:"zones"`
	}

	// Group zones by country
	countryZones := make(map[string][]zoneEntry)
	countryNames := make(map[string]string)
	for _, code := range s.zoneRegistry.AllZones() {
		z, ok := s.zoneRegistry.GetZone(code)
		if !ok {
			continue
		}
		cc, ok := s.zoneRegistry.GetCountry(z.Country)
		if !ok {
			continue
		}
		countryZones[z.Country] = append(countryZones[z.Country], zoneEntry{
			Code:    z.Code,
			Name:    z.Name,
			Country: z.Country,
		})
		countryNames[z.Country] = cc.Name
	}

	// Sort country codes for deterministic output
	var countryCodes []string
	for cc := range countryZones {
		countryCodes = append(countryCodes, cc)
	}
	sort.Strings(countryCodes)

	var groups []zoneGroup
	for _, cc := range countryCodes {
		groups = append(groups, zoneGroup{
			Name:  countryNames[cc],
			Zones: countryZones[cc],
		})
	}

	writeJSON(w, map[string]any{
		"groups":  groups,
		"current": s.cfg.BiddingZone,
	})
}

func (s *Server) handleZoneDetect(w http.ResponseWriter, r *http.Request) {
	if s.supervisor == nil {
		writeJSON(w, map[string]any{"detected": false, "reason": "not running inside HA"})
		return
	}
	haConfig, err := s.supervisor.GetConfig(r.Context())
	if err != nil {
		writeJSON(w, map[string]any{"detected": false, "reason": "could not read HA config"})
		return
	}

	lat, _ := haConfig["latitude"].(float64)
	lon, _ := haConfig["longitude"].(float64)
	tz, _ := haConfig["time_zone"].(string)
	haCountry, _ := haConfig["country"].(string)

	result := s.zoneRegistry.DetectZone(lat, lon, tz, haCountry)
	if result == nil {
		writeJSON(w, map[string]any{
			"detected":           false,
			"unsupported_region": true,
			"timezone":           tz,
			"ha_country":         haCountry,
			"reason":             "no supported zone found for this location",
		})
		return
	}

	resp := map[string]any{
		"detected":   true,
		"zone":       result.Zone.Code,
		"name":       result.Zone.Name,
		"country":    result.Country,
		"method":     result.Method,
		"timezone":   tz,
		"ha_country": result.HACountry,
	}
	if result.Mismatch {
		resp["mismatch"] = true
		cc, _ := s.zoneRegistry.GetCountry(result.Country)
		if cc != nil {
			resp["detected_country_name"] = cc.Name
		}
	}
	if result.Distance > 0 {
		resp["distance_km"] = int(result.Distance)
	}
	writeJSON(w, resp)
}

func (s *Server) handleTaxBreakdown(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = s.cfg.BiddingZone
	}

	// Resolve tax data: Worker cache first, then zone config fallback.
	var override *engine.WorkerTaxOverride
	if s.taxCache != nil {
		override = s.taxCache.Get(zone)
	}

	// For non-wholesale zones (fixed/TOU): build override from embedded zone tax_defaults.
	// This enables tax insight for regulated zones that never get Worker tax profiles.
	if override == nil && s.zoneRegistry != nil {
		if defaults := s.zoneRegistry.GetTaxDefaults(zone); defaults != nil {
			override = &engine.WorkerTaxOverride{
				VATRate:          defaults.VATRate,
				EnergyTax:        defaults.EnergyTax,
				Surcharges:       defaults.Surcharges,
				NetworkTariffAvg: defaults.NetworkTariffAvg,
				Version:          "embedded-" + defaults.ValidFrom,
			}
		}
	}

	if override == nil {
		writeError(w, http.StatusNotFound, "no tax profile for zone "+zone)
		return
	}

	// Get current price and wholesale for breakdown calculation
	var wholesaleKWh float64
	var markupEstimated bool
	data := s.sensorData.Get()
	mode := s.cfg.PricingMode
	isRegulated := mode == "fixed" || mode == "tou"

	// For regulated modes: reverse-calculate from the all-in consumer price.
	// Consumer price = (energy_cost + energy_tax + surcharges) × (1 + VAT)
	// So: energy_cost = consumer / (1 + VAT) - energy_tax - surcharges
	if isRegulated && data != nil && data.CurrentPrice > 0 {
		subtotal := data.CurrentPrice / (1 + override.VATRate)
		energyCost := subtotal - override.EnergyTax - override.Surcharges
		if energyCost < 0 {
			energyCost = 0
		}
		vatAmount := data.CurrentPrice - subtotal

		breakdown := models.PriceBreakdown{
			Wholesale:      energyCost, // For regulated: this is the base energy cost (no wholesale market)
			SupplierMarkup: 0,          // Included in the regulated tariff, not separable
			EnergyTax:      override.EnergyTax,
			Surcharges:     override.Surcharges,
			NetworkTariff:  override.NetworkTariffAvg,
			Subtotal:       subtotal,
			VATRate:        override.VATRate,
			VATAmount:      vatAmount,
			ConsumerTotal:  data.CurrentPrice,
		}

		resp := map[string]any{
			"zone":      zone,
			"breakdown": breakdown,
			"version":   override.Version,
			"regulated": true, // Signal to frontend: this is a regulated breakdown
		}
		writeJSON(w, resp)
		return
	}

	// Step 1: Get REAL ENTSO-E wholesale for the current slot.
	if s.fallback != nil {
		now := time.Now().UTC()
		wholesaleMap := s.fallback.FetchWholesaleForZone(r.Context(), zone, now)
		currentHour := now.Truncate(time.Hour)
		if w, ok := wholesaleMap[currentHour]; ok {
			wholesaleKWh = w
		}
	}

	// Step 2: Determine supplier markup.
	// When we have both wholesale AND consumer price: markup = derived (exact).
	// When we only have consumer price: reverse-calculate wholesale from known markup.
	var supplierMarkup float64

	if wholesaleKWh != 0 && data != nil && data.CurrentPrice > 0 {
		// Real wholesale available — derive markup as residual.
		// markup = consumer_excl_vat - wholesale - taxes
		// This ensures: wholesale is ENTSO-E (real), total = dashboard price.
		subtotal := data.CurrentPrice / (1 + override.VATRate)
		supplierMarkup = subtotal - override.EnergyTax - override.Surcharges - wholesaleKWh
		// Negative markup is valid: supplier can be cheaper than wholesale + taxes
	} else if data != nil && data.CurrentPrice > 0 {
		// No wholesale available — reverse-calculate from consumer price.
		// Use best available markup: user config > Worker crowdsource > 0.
		if s.cfg.SupplierMarkup > 0 {
			supplierMarkup = s.cfg.SupplierMarkup
		} else if override.SupplierMarkup > 0 {
			supplierMarkup = override.SupplierMarkup
		}
		subtotal := data.CurrentPrice / (1 + override.VATRate)
		wholesaleKWh = subtotal - override.EnergyTax - override.Surcharges - supplierMarkup
		if supplierMarkup == 0 {
			markupEstimated = true
		}
	}

	tp := models.TaxProfile{
		VATRate:          override.VATRate,
		SupplierMarkup:   supplierMarkup,
		EnergyTax:        []models.EnergyTaxEntry{{From: "2000-01-01", Rate: override.EnergyTax}},
		Surcharges:       override.Surcharges,
		NetworkTariffAvg: override.NetworkTariffAvg,
	}
	breakdown := tp.CalculateBreakdown(wholesaleKWh, time.Now())

	resp := map[string]any{
		"zone":      zone,
		"breakdown": breakdown,
		"version":   override.Version,
	}
	if markupEstimated {
		resp["markup_estimated"] = true
	}

	writeJSON(w, resp)
}

// readSensorPrice reads the current price from an HA sensor entity.
// Returns 0 if the sensor is unavailable or has no valid state.
func (s *Server) readSensorPrice(ctx context.Context, entityID string) float64 {
	if s.supervisor == nil || entityID == "" {
		return 0
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	state, err := s.supervisor.GetState(ctx, entityID)
	if err != nil {
		return 0
	}
	stateStr, _ := state["state"].(string)
	val, err := strconv.ParseFloat(stateStr, 64)
	if err != nil || val <= 0 {
		return 0
	}

	// Check unit — sensor might be in ct/kWh or EUR/kWh
	attrs, _ := state["attributes"].(map[string]any)
	if attrs != nil {
		unit, _ := attrs["unit_of_measurement"].(string)
		unitLower := strings.ToLower(unit)
		// If unit contains "ct" or "cent", it's already in cents — convert to EUR
		if strings.Contains(unitLower, "ct") || strings.Contains(unitLower, "cent") {
			val = val / 100
		}
	}
	return val
}

func (s *Server) handleTariffSensors(w http.ResponseWriter, r *http.Request) {
	if s.supervisor == nil {
		writeJSON(w, map[string]any{"sensors": []any{}})
		return
	}

	states, err := s.supervisor.GetAllStates(r.Context())
	if err != nil {
		writeJSON(w, map[string]any{"sensors": []any{}})
		return
	}

	type sensorEntry struct {
		EntityID string `json:"entity_id"`
		Name     string `json:"name"`
		State    string `json:"state"`
		Unit     string `json:"unit"`
	}

	var sensors []sensorEntry
	for _, state := range states {
		entityID, _ := state["entity_id"].(string)
		if entityID == "" || !strings.HasPrefix(entityID, "sensor.") {
			continue
		}
		lower := strings.ToLower(entityID)

		// Exclude our own sensors (circular) and non-electricity sensors
		if strings.Contains(lower, "synctacles") ||
			strings.Contains(lower, "gas") || strings.Contains(lower, "standing") {
			continue
		}

		// Must have a price-like unit (EUR/kWh, €/kWh, GBP/kWh, etc.)
		attrs, _ := state["attributes"].(map[string]any)
		unit := ""
		if attrs != nil {
			if u, ok := attrs["unit_of_measurement"].(string); ok {
				unit = u
			}
		}
		if !strings.Contains(strings.ToLower(unit), "/kwh") {
			continue
		}

		// Must have a numeric state (skip "low", "2", "unavailable", etc.)
		stateVal, _ := state["state"].(string)
		if _, err := strconv.ParseFloat(stateVal, 64); err != nil {
			continue
		}

		// Look for tariff/price sensors from known integrations
		isTariff := strings.Contains(lower, "tariff") ||
			strings.Contains(lower, "tarief") ||
			strings.Contains(lower, "electricity_price") ||
			strings.Contains(lower, "energy_price") ||
			strings.Contains(lower, "unit_rate") ||       // UK Glow CAD
			strings.Contains(lower, "current_rate") ||    // UK Octopus
			strings.Contains(lower, "current_price") ||   // Nord Pool
			strings.Contains(lower, "net_price") ||       // EPEX Spot
			strings.Contains(lower, "current_hour_price") || // easyEnergy
			strings.Contains(lower, "consumption_price") || // P1 Monitor
			strings.Contains(lower, "energidataservice")    // DK
		if !isTariff {
			continue
		}

		name := entityID
		if attrs != nil {
			if fn, ok := attrs["friendly_name"].(string); ok {
				name = fn
			}
		}

		sensors = append(sensors, sensorEntry{
			EntityID: entityID,
			Name:     name,
			State:    stateVal,
			Unit:     unit,
		})
	}

	writeJSON(w, map[string]any{"sensors": sensors})
}

func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	// Non-wholesale zones with fixed/TOU pricing use regulated tariffs — no wholesale sources to show
	if z, ok := s.zoneRegistry.GetZone(s.cfg.BiddingZone); ok && !z.HasWholesale() {
		if s.cfg.PricingMode == "fixed" || s.cfg.PricingMode == "tou" {
			writeJSON(w, map[string]any{
				"sources":      []any{},
				"zone":         s.cfg.BiddingZone,
				"pricing_mode": s.cfg.PricingMode,
				"regulated":    true,
			})
			return
		}
	}

	activeSource := ""
	quality := ""
	if data := s.sensorData.Get(); data != nil {
		activeSource = data.Source
		quality = data.Quality
	}
	var statuses []engine.SourceHealth
	if s.fallback != nil {
		statuses = s.fallback.SourceStatus(activeSource)
	}

	// Active source info — reveals what's actually being served, even if circuit breaker is open
	var activeInfo *engine.ActiveSourceInfo
	if s.fallback != nil {
		activeInfo = s.fallback.ActiveInfo(s.cfg.BiddingZone, time.Now().UTC())
	}

	writeJSON(w, map[string]any{
		"sources":      statuses,
		"zone":         s.cfg.BiddingZone,
		"pricing_mode": s.cfg.PricingMode,
		"quality":      quality,
		"active_info":  activeInfo,
	})
}

func (s *Server) handleKBSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, map[string]string{"error": "query parameter 'q' is required"})
		return
	}

	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	result, err := s.kbClient.Search(ctx, query, limit)
	if err != nil {
		slog.Warn("kb search failed", "error", err)
		writeJSON(w, map[string]any{"results": []any{}, "total": 0, "error": err.Error()})
		return
	}

	writeJSON(w, result)
}

func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
		return
	}

	// Inject ALL i18n bundles inline to eliminate browser/ingress cache issues.
	// Previous approach (fetch /static/i18n/<lang>.json) suffered from stale browser
	// cache: old JSON (missing new keys) caused English fallback text in NL installs.
	var allI18n []byte
	allI18n = append(allI18n, '{')
	langs := []string{"en", "nl", "de", "da", "es", "fi", "fr", "pt"}
	first := true
	for _, lang := range langs {
		langJSON, err2 := staticFS.ReadFile("static/i18n/" + lang + ".json")
		if err2 != nil {
			continue
		}
		if !first {
			allI18n = append(allI18n, ',')
		}
		allI18n = append(allI18n, '"')
		allI18n = append(allI18n, lang...)
		allI18n = append(allI18n, '"', ':')
		allI18n = append(allI18n, langJSON...)
		first = false
	}
	allI18n = append(allI18n, '}')
	data = bytes.Replace(data, []byte("var _i18nAll = null;"), []byte("var _i18nAll = "+string(allI18n)+";"), 1)
	// Also inject English as default _i18n for immediate rendering
	if enJSON, err2 := staticFS.ReadFile("static/i18n/en.json"); err2 == nil {
		data = bytes.Replace(data, []byte("var _i18n = {};"), []byte("var _i18n = "+string(enJSON)+";"), 1)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	_, _ = w.Write(data)
}

// --- Feedback Handlers ---

const feedbackBaseURL = "https://api.synctacles.com"

// feedbackSystemInfo collects system information for feedback submissions.
func (s *Server) feedbackSystemInfo() map[string]any {
	info := map[string]any{
		"addon_version": s.version,
		"zone":          s.cfg.BiddingZone,
	}

	if s.supervisor != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if core, err := s.supervisor.GetCoreInfo(ctx); err == nil {
			info["ha_version"] = core.Version
			info["ha_arch"] = core.Arch
			info["ha_machine"] = core.Machine
		}
	}

	return info
}

func (s *Server) handleFeedbackSysInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.feedbackSystemInfo())
}

func (s *Server) handleFeedbackRating(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		writeError(w, http.StatusBadRequest, "rating must be between 1 and 5")
		return
	}

	// Build message: "Rating: 4/5 — user comment here"
	msg := fmt.Sprintf("Rating: %d/5", req.Rating)
	if req.Comment != "" {
		msg += " — " + req.Comment
	}

	payload := map[string]any{
		"type":    "other",
		"product": "energy",
		"message": msg,
	}

	resp, err := s.forwardFeedback(payload)
	if err != nil {
		slog.Error("failed to forward rating", "error", err)
		writeError(w, http.StatusBadGateway, "failed to submit feedback")
		return
	}
	writeJSON(w, resp)
}

func (s *Server) handleFeedbackBug(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Description) == "" {
		writeError(w, http.StatusBadRequest, "title and description are required")
		return
	}

	sysInfo := s.feedbackSystemInfo()
	msg := req.Title + "\n\n" + req.Description + "\n\nSystem: " + fmt.Sprintf("%v", sysInfo)

	payload := map[string]any{
		"type":    "bug",
		"product": "energy",
		"message": msg,
	}

	resp, err := s.forwardFeedback(payload)
	if err != nil {
		slog.Error("failed to forward bug report", "error", err)
		writeError(w, http.StatusBadGateway, "failed to submit bug report")
		return
	}
	writeJSON(w, resp)
}

// forwardFeedback sends a feedback payload to the auth service.
func (s *Server) forwardFeedback(payload map[string]any) (map[string]any, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", feedbackBaseURL+"/api/v1/feedback", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	platform.SignRequest(req, jsonData)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("auth service returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return result, nil
}

// --- GDPR Data Deletion ---

func (s *Server) handleDeleteData(w http.ResponseWriter, r *http.Request) {
	if s.installUUID == "" {
		writeError(w, http.StatusServiceUnavailable, "install UUID not available")
		return
	}

	payload := map[string]string{"install_uuid": s.installUUID}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build request")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Delete from platform Worker (synctacles-prod D1)
	req, err := http.NewRequestWithContext(ctx, "DELETE", feedbackBaseURL+"/api/v1/install/data", bytes.NewReader(jsonData))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to contact server")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		writeError(w, http.StatusBadGateway, "server returned error: "+string(body))
		return
	}

	// Also delete from energy-data Worker (synctacles-energy-db D1) — best-effort
	go func() {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel2()
		req2, err := http.NewRequestWithContext(ctx2, "DELETE", energyDataBaseURL+"/api/v1/install/data", bytes.NewReader(jsonData))
		if err != nil {
			return
		}
		req2.Header.Set("Content-Type", "application/json")
		resp2, err := http.DefaultClient.Do(req2)
		if err != nil {
			return
		}
		resp2.Body.Close()
	}()

	// Delete local UUID file so a fresh one is generated on restart
	_ = os.Remove("/config/.synctacles_install_id")

	// Restart Care app so it picks up the UUID change (best-effort)
	if s.supervisor != nil {
		go func() {
			ctx2, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := s.supervisor.RestartAddon(ctx2, "308ee12f_synctacles_care"); err != nil {
				slog.Warn("GDPR: could not restart Care app", "error", err)
			} else {
				slog.Info("GDPR: Care app restart triggered for UUID sync")
			}
		}()
	}

	writeJSON(w, map[string]any{
		"status":         "ok",
		"install_uuid":   s.installUUID,
		"restart_needed": true,
	})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// handleCacheView returns all cached prices for the active zone with source labels.
func (s *Server) handleCacheView(w http.ResponseWriter, r *http.Request) {
	if s.sqliteCache == nil {
		writeError(w, http.StatusServiceUnavailable, "cache not available")
		return
	}

	zone := s.cfg.BiddingZone
	now := time.Now().UTC()
	rows, err := s.sqliteCache.GetAllForZone(zone, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cache read failed")
		return
	}

	mode := s.cfg.PricingMode
	isConsumerPriceMode := mode == "external_sensor" || mode == "p1_meter" || mode == "meter_tariff"
	taxSource := ""
	if s.normalizer != nil {
		taxSource = s.normalizer.TaxSource()
	}

	type cacheEntry struct {
		Hour        string             `json:"hour"`
		WholesaleKWh float64           `json:"wholesale_kwh"`
		ConsumerKWh float64            `json:"consumer_kwh"`
		Source      string             `json:"source"`
		SourceLabel string             `json:"source_label"`
		Quality     string             `json:"quality"`
		Tier        int                `json:"tier"`
		IsConsumer  bool               `json:"is_consumer"`
		FetchedAt   string             `json:"fetched_at"`
		Breakdown   *models.PriceBreakdown `json:"breakdown,omitempty"`
	}

	// When entries lack wholesale data, fetch from Worker for comparison
	var wholesaleMap map[time.Time]float64
	needsWholesale := false
	for _, row := range rows {
		if row.WholesaleKWh == 0 {
			needsWholesale = true
			break
		}
	}
	if needsWholesale && s.fallback != nil {
		wholesaleMap = s.fallback.FetchWholesaleForZone(r.Context(), zone, now)
	}

	loc := s.zoneLoc()

	entries := make([]cacheEntry, 0, len(rows))
	for _, row := range rows {
		// Convert timestamp to local time for display
		localTs := row.Timestamp
		if ts, err := time.Parse(time.RFC3339, row.Timestamp); err == nil {
			localTs = ts.In(loc).Format(time.RFC3339)
		}
		e := cacheEntry{
			Hour:         localTs,
			WholesaleKWh: row.WholesaleKWh,
			ConsumerKWh:  row.PriceEUR,
			Source:       row.Source,
			SourceLabel:  deriveSourceLabel(mode, row.Source, taxSource, row.IsConsumer),
			Quality:      row.Quality,
			Tier:         row.Tier,
			IsConsumer:   row.IsConsumer,
			FetchedAt:    row.FetchedAt,
		}

		// Fill wholesale from Worker comparison data if missing.
		// Try exact PT15 timestamp first, fall back to hourly for mixed resolution.
		wholesale := row.WholesaleKWh
		if wholesale == 0 && wholesaleMap != nil {
			ts, err := time.Parse(time.RFC3339, row.Timestamp)
			if err == nil {
				if w, ok := wholesaleMap[ts]; ok {
					wholesale = w
					e.WholesaleKWh = w
				} else if w, ok := wholesaleMap[ts.Truncate(time.Hour)]; ok {
					wholesale = w
					e.WholesaleKWh = w
				}
			}
		}

		// Compute breakdown if we have tax data and a wholesale price (including negative).
		// Markup priority: exact calc > user config > Worker calibration > 2% fallback.
		if wholesale != 0 && s.taxCache != nil {
			if tp := s.taxCache.Get(zone); tp != nil {
				var markup float64
				displayWholesale := wholesale
				if isConsumerPriceMode && row.PriceEUR > 0 {
					// Exact: decompose consumer price using known wholesale
					subtotal := row.PriceEUR / (1 + tp.VATRate)
					markup = subtotal - tp.EnergyTax - tp.Surcharges - wholesale
					if markup < 0 {
						markup = 0
					}
				} else if s.cfg.SupplierMarkup > 0 {
					markup = s.cfg.SupplierMarkup
				} else if tp.SupplierMarkup > 0 {
					markup = tp.SupplierMarkup
				}
				subtotal := displayWholesale + markup + tp.EnergyTax + tp.Surcharges
				vatAmount := subtotal * tp.VATRate
				bd := models.PriceBreakdown{
					Wholesale:      displayWholesale,
					SupplierMarkup: markup,
					EnergyTax:      tp.EnergyTax,
					Surcharges:     tp.Surcharges,
					NetworkTariff:  tp.NetworkTariffAvg,
					Subtotal:       subtotal,
					VATRate:        tp.VATRate,
					VATAmount:      vatAmount,
					ConsumerTotal:  subtotal + vatAmount,
				}
				e.Breakdown = &bd
			}
		}

		entries = append(entries, e)
	}

	// Active source info
	var activeInfo *engine.ActiveSourceInfo
	if s.fallback != nil {
		activeInfo = s.fallback.ActiveInfo(zone, now)
	}

	writeJSON(w, map[string]any{
		"zone":         zone,
		"pricing_mode": mode,
		"tax_source":   taxSource,
		"entries":      entries,
		"count":        len(entries),
		"active_info":  activeInfo,
	})
}

// handleCacheReset clears the price cache for the active zone and triggers a refetch.
func (s *Server) handleCacheReset(w http.ResponseWriter, r *http.Request) {
	zone := s.cfg.BiddingZone

	var deleted int64
	if s.sqliteCache != nil {
		var err error
		deleted, err = s.sqliteCache.ClearZone(zone)
		if err != nil {
			slog.Error("cache reset: sqlite clear failed", "error", err)
		}
	}

	if s.fallback != nil {
		s.fallback.ClearMemCache()
	}

	if s.scheduler != nil {
		s.scheduler.TriggerFetch()
	}

	slog.Info("cache reset", "zone", zone, "deleted", deleted)
	writeJSON(w, map[string]any{
		"status":  "ok",
		"deleted": deleted,
	})
}

// deriveSourceLabel returns a human-readable label describing how the price was composed.
func deriveSourceLabel(mode, source, taxSource string, isConsumer bool) string {
	if mode == "external_sensor" || mode == "p1_meter" || mode == "meter_tariff" {
		return "External sensor"
	}
	if mode == "manual" {
		return "Manual tax config"
	}
	if isConsumer {
		switch source {
		case "synctacles":
			return "Worker (wholesale + tax)"
		default:
			return source + " (consumer)"
		}
	}
	switch taxSource {
	case "worker":
		return "Wholesale + Worker tax"
	case "embedded":
		return "Wholesale + embedded tax"
	default:
		return "Wholesale only (no tax)"
	}
}

// zoneLoc returns the *time.Location for the configured bidding zone.
// Falls back to UTC if the zone is not found or the timezone is invalid.
func (s *Server) zoneLoc() *time.Location {
	if z, ok := s.zoneRegistry.GetZone(s.cfg.BiddingZone); ok {
		if loc, err := time.LoadLocation(z.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// utcHourToLocal converts a UTC "HH:MM" string to local time for display.
func utcHourToLocal(utcHour string, loc *time.Location) string {
	if loc == time.UTC {
		return utcHour
	}
	parts := strings.SplitN(utcHour, ":", 2)
	if len(parts) != 2 {
		return utcHour
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return utcHour
	}
	now := time.Now().UTC()
	ts := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, time.UTC)
	return ts.In(loc).Format("15:04")
}

// toFloat64 extracts a float64 from a map value (handles both float64 and json.Number).
// applyStringField sets *dst if key exists in incoming as a string.
func applyStringField(incoming map[string]any, key string, dst *string) {
	if v, ok := incoming[key].(string); ok {
		*dst = v
	}
}

// applyFloatField sets *dst if key exists in incoming as a float64.
func applyFloatField(incoming map[string]any, key string, dst *float64) {
	if v, ok := incoming[key].(float64); ok {
		*dst = v
	}
}

// applyBoolField sets *dst if key exists in incoming as a bool.
func applyBoolField(incoming map[string]any, key string, dst *bool) {
	if v, ok := incoming[key].(bool); ok {
		*dst = v
	}
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case json.Number:
		f, err := val.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
