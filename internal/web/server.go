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
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/synctacles/energy-app/internal/config"
	"github.com/synctacles/energy-app/pkg/engine"
	"github.com/synctacles/energy-app/internal/gate"
	"github.com/synctacles/energy-app/internal/ha"
	"github.com/synctacles/energy-app/pkg/models"
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
		r.Post("/calibrate", s.handleCalibrate)
		r.Get("/sensors/tariff", s.handleTariffSensors)
		r.Get("/suppliers", s.handleSuppliers)

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

		// Feedback
		r.Get("/feedback/sysinfo", s.handleFeedbackSysInfo)
		r.Post("/feedback/rating", s.handleFeedbackRating)
		r.Post("/feedback/bug", s.handleFeedbackBug)
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

	// Build hourly prices for chart
	todayPrices := make([]map[string]any, 0, len(data.TodayPrices))
	for _, p := range data.TodayPrices {
		todayPrices = append(todayPrices, map[string]any{
			"hour":  p.Timestamp.Format("15:04"),
			"price": p.PriceEUR,
		})
	}

	dashboard := map[string]any{
		"status":  "ok",
		"version": s.version,
		"zone":    data.Zone,
		"current_price": map[string]any{
			"price":       fmt.Sprintf("%.4f", data.CurrentPrice),
			"unit":        "EUR/kWh",
			"source":      data.Source,
			"leverancier": data.Leverancier,
		},
		"action": map[string]any{
			"action":    data.Action.Action,
			"reason":    data.Action.Reason,
			"deviation": data.Action.DeviationPct,
		},
		"stats": map[string]any{
			"average":        data.Stats.Average,
			"min":            data.Stats.Min,
			"max":            data.Stats.Max,
			"cheapest_hour":  data.Stats.CheapestHour,
			"expensive_hour": data.Stats.ExpensiveHour,
		},
		"prices_today": todayPrices,
		"tomorrow": map[string]any{
			"status":   data.Tomorrow.Status,
			"avg":      data.Tomorrow.AvgPrice,
			"hours":    len(data.TomorrowPrices),
		},
		"quality":    data.Quality,
		"updated_at": data.UpdatedAt.Format(time.RFC3339),
		"last_fetch": st.LastFetch,
	}

	if data.BestWindow != nil {
		dashboard["best_window"] = map[string]any{
			"start":    data.BestWindow.StartHour,
			"end":      data.BestWindow.EndHour,
			"avg":      data.BestWindow.AvgPrice,
			"duration": data.BestWindow.Duration,
		}
	}

	// Source info for chart attribution
	dashboard["source_info"] = map[string]any{
		"source":      data.Source,
		"quality":     data.Quality,
		"leverancier": data.Leverancier,
	}

	// Tax data source: "worker", "consumer", "embedded", "none"
	// "consumer" = prices already include taxes (Enever, Worker consumer prices)
	// "worker"   = live Worker tax profile applied to wholesale
	// "embedded" = fallback tax defaults (less accurate)
	// "none"     = no tax data at all (wholesale only)
	if s.normalizer != nil {
		taxSource := s.normalizer.TaxSource()
		dashboard["tax_source"] = taxSource
		if taxSource == "none" {
			dashboard["degraded"] = true
			dashboard["degraded_reason"] = "no_tax_data"
		}
	}

	writeJSON(w, dashboard)
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
		"enever_token":            s.cfg.EneverToken,
		"enever_leverancier":      s.cfg.EneverLeverancier,
		"supplier_markup":         s.cfg.SupplierMarkup,
		"supplier_id":             s.cfg.SupplierID,
		"manual_vat_rate":         s.cfg.ManualVATRate,
		"manual_energy_tax":       s.cfg.ManualEnergyTax,
		"manual_surcharges":       s.cfg.ManualSurcharges,
		"manual_network_tariff":   s.cfg.ManualNetworkTariff,
		"p1_sensor_entity":        s.cfg.P1SensorEntity,
		"fixed_rate_price":        s.cfg.FixedRatePrice,
		"debug_mode":              s.cfg.DebugMode,
		"disclaimer_accepted":     s.cfg.DisclaimerAccepted,
		"privacy_accepted":        s.cfg.PrivacyAccepted,
		"detected_power_sensor":   s.detectedPowerSensor,
		"detected_tariff_sensor":  s.detectedTariffSensor,
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
		"enever_token", "enever_leverancier", "supplier_markup", "supplier_id",
		"manual_vat_rate", "manual_energy_tax", "manual_surcharges", "manual_network_tariff",
		"p1_sensor_entity", "fixed_rate_price", "power_sensor", "debug_mode",
		"disclaimer_accepted", "privacy_accepted",
	}
	for _, key := range allowed {
		if val, ok := incoming[key]; ok {
			current[key] = val
		}
	}

	// Derive enever_enabled from pricing_mode
	if mode, ok := incoming["pricing_mode"].(string); ok {
		current["enever_enabled"] = (mode == config.ModeEnever)
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

	// Invalidate tax cache when zone changes (forces Worker re-fetch for new zone)
	if v, ok := incoming["zone"].(string); ok && v != s.cfg.BiddingZone {
		if s.taxCache != nil {
			s.taxCache.Invalidate(s.cfg.BiddingZone)
		}
	}

	// Update in-memory config for immediate effect
	if v, ok := incoming["pricing_mode"].(string); ok {
		s.cfg.PricingMode = v
	}
	if v, ok := incoming["zone"].(string); ok {
		s.cfg.BiddingZone = v
	}
	if v, ok := incoming["best_window_hours"].(float64); ok {
		s.cfg.BestWindowHours = int(v)
	}
	if v, ok := incoming["enever_token"].(string); ok {
		s.cfg.EneverToken = v
	}
	if v, ok := incoming["enever_leverancier"].(string); ok {
		s.cfg.EneverLeverancier = v
	}
	if v, ok := incoming["supplier_markup"].(float64); ok {
		s.cfg.SupplierMarkup = v
		if s.normalizer != nil {
			s.normalizer.SetSupplierMarkup(v)
		}
		// Trigger re-fetch so cached prices are re-normalized with new markup
		if s.scheduler != nil {
			s.scheduler.TriggerFetch()
		}
	}
	if v, ok := incoming["supplier_id"].(string); ok {
		s.cfg.SupplierID = v
	}
	if v, ok := incoming["manual_vat_rate"].(float64); ok {
		s.cfg.ManualVATRate = v
	}
	if v, ok := incoming["manual_energy_tax"].(float64); ok {
		s.cfg.ManualEnergyTax = v
	}
	if v, ok := incoming["manual_surcharges"].(float64); ok {
		s.cfg.ManualSurcharges = v
	}
	if v, ok := incoming["manual_network_tariff"].(float64); ok {
		s.cfg.ManualNetworkTariff = v
	}
	if v, ok := incoming["p1_sensor_entity"].(string); ok {
		s.cfg.P1SensorEntity = v
	}
	if v, ok := incoming["fixed_rate_price"].(float64); ok {
		s.cfg.FixedRatePrice = v
	}
	if v, ok := incoming["disclaimer_accepted"].(bool); ok {
		s.cfg.DisclaimerAccepted = v
	}
	if v, ok := incoming["privacy_accepted"].(bool); ok {
		s.cfg.PrivacyAccepted = v
	}

	writeJSON(w, map[string]string{"status": "saved", "message": "Settings saved. Restart app for source chain changes."})
}

func (s *Server) handleSuppliers(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = s.cfg.BiddingZone
	}
	cc, ok := s.zoneRegistry.GetCountryForZone(zone)
	if !ok || len(cc.Suppliers) == 0 {
		writeJSON(w, []any{})
		return
	}
	writeJSON(w, cc.Suppliers)
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
	tz, _ := haConfig["time_zone"].(string)
	if tz == "" {
		writeJSON(w, map[string]any{"detected": false, "reason": "no timezone in HA config"})
		return
	}
	// Match timezone to a zone
	for _, code := range s.zoneRegistry.AllZones() {
		z, ok := s.zoneRegistry.GetZone(code)
		if !ok {
			continue
		}
		if z.Timezone == tz {
			writeJSON(w, map[string]any{
				"detected": true,
				"zone":     z.Code,
				"name":     z.Name,
				"timezone": tz,
			})
			return
		}
	}
	// No exact match — try country from timezone (Europe/Berlin → DE)
	writeJSON(w, map[string]any{"detected": false, "timezone": tz, "reason": "no zone matches timezone " + tz})
}

func (s *Server) handleTaxBreakdown(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = s.cfg.BiddingZone
	}

	if s.taxCache == nil {
		writeError(w, http.StatusServiceUnavailable, "tax cache not available")
		return
	}
	override := s.taxCache.Get(zone)
	if override == nil {
		writeError(w, http.StatusNotFound, "no tax profile for zone "+zone)
		return
	}

	// Get current price for breakdown calculation
	var wholesaleKWh float64
	if data := s.sensorData.Get(); data != nil && data.CurrentPrice > 0 {
		// Reverse: consumer / (1 + VAT) - taxes = wholesale
		subtotal := data.CurrentPrice / (1 + override.VATRate)
		wholesaleKWh = subtotal - override.EnergyTax - override.Surcharges - override.SupplierMarkup
		if wholesaleKWh < 0 {
			wholesaleKWh = 0
		}
	}

	tp := models.TaxProfile{
		VATRate:          override.VATRate,
		SupplierMarkup:   override.SupplierMarkup,
		EnergyTax:        []models.EnergyTaxEntry{{From: "2000-01-01", Rate: override.EnergyTax}},
		Surcharges:       override.Surcharges,
		NetworkTariffAvg: override.NetworkTariffAvg,
	}
	breakdown := tp.CalculateBreakdown(wholesaleKWh, time.Now())

	writeJSON(w, map[string]any{
		"zone":      zone,
		"breakdown": breakdown,
		"version":   override.Version,
	})
}

func (s *Server) handleCalibrate(w http.ResponseWriter, r *http.Request) {
	if s.supervisor == nil {
		writeError(w, http.StatusServiceUnavailable, "not running inside HA app")
		return
	}

	var req struct {
		UserPrice float64 `json:"user_price"` // EUR/kWh incl VAT
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Validate input range
	if req.UserPrice < 0.01 || req.UserPrice > 2.0 {
		writeError(w, http.StatusBadRequest, "price must be between 0.01 and 2.00 EUR/kWh")
		return
	}

	// Get current Synctacles-computed price
	data := s.sensorData.Get()
	if data == nil || data.CurrentPrice <= 0 {
		writeError(w, http.StatusServiceUnavailable, "no current price available for calibration")
		return
	}

	// Calculate margin
	margin := req.UserPrice - data.CurrentPrice

	// Warn if margin is outside normal range but allow it
	warning := ""
	if margin < -0.05 || margin > 0.10 {
		warning = "Margin is outside the normal range (-0.05 to 0.10). Double-check your input."
	}

	// Save as supplier_markup via Supervisor options
	current, err := s.supervisor.GetAddonOptions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read current options")
		return
	}
	current["supplier_markup"] = margin
	if err := s.supervisor.SetAddonOptions(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save calibration")
		return
	}
	s.cfg.SupplierMarkup = margin

	writeJSON(w, map[string]any{
		"status":           "calibrated",
		"synctacles_price": data.CurrentPrice,
		"user_price":       req.UserPrice,
		"margin":           margin,
		"warning":          warning,
	})
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
		if entityID == "" {
			continue
		}
		// Look for tariff/price sensors from known integrations
		lower := strings.ToLower(entityID)
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
		// Exclude non-electricity sensors
		if strings.Contains(lower, "gas") || strings.Contains(lower, "standing") {
			continue
		}
		// Must be a sensor (not binary_sensor, etc.)
		if !strings.HasPrefix(entityID, "sensor.") {
			continue
		}

		name := entityID
		if attrs, ok := state["attributes"].(map[string]any); ok {
			if fn, ok := attrs["friendly_name"].(string); ok {
				name = fn
			}
		}
		stateVal, _ := state["state"].(string)
		unit := ""
		if attrs, ok := state["attributes"].(map[string]any); ok {
			if u, ok := attrs["unit_of_measurement"].(string); ok {
				unit = u
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
	activeSource := ""
	leverancier := ""
	quality := ""
	if data := s.sensorData.Get(); data != nil {
		activeSource = data.Source
		leverancier = data.Leverancier
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
		"sources":     statuses,
		"zone":        s.cfg.BiddingZone,
		"leverancier": leverancier,
		"quality":     quality,
		"active_info": activeInfo,
	})
}

func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
		return
	}

	// Inject default i18n (English) inline to eliminate browser/ingress cache issues.
	// The separate static/i18n/*.json fetch suffered from stale browser cache: old JSON
	// (missing new keys) was served even after app updates, causing raw keys in the UI.
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

	payload := map[string]any{
		"type":        "rating",
		"product_id":  "energy",
		"rating":      req.Rating,
		"comment":     req.Comment,
		"system_info": s.feedbackSystemInfo(),
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

	payload := map[string]any{
		"type":        "bug",
		"product_id":  "energy",
		"title":       req.Title,
		"description": req.Description,
		"system_info": s.feedbackSystemInfo(),
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

	// When entries lack wholesale data (Enever mode), fetch from Worker for comparison
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

	entries := make([]cacheEntry, 0, len(rows))
	for _, row := range rows {
		e := cacheEntry{
			Hour:         row.Timestamp,
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

		// Compute breakdown if we have tax data and a wholesale price
		if wholesale > 0 && s.taxCache != nil {
			if tp := s.taxCache.Get(zone); tp != nil {
				subtotal := wholesale + tp.SupplierMarkup + tp.EnergyTax + tp.Surcharges
				vatAmount := subtotal * tp.VATRate
				bd := models.PriceBreakdown{
					Wholesale:      wholesale,
					SupplierMarkup: tp.SupplierMarkup,
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
	if mode == "p1_meter" {
		return "P1 meter tariff"
	}
	if mode == "manual" {
		return "Manual tax config"
	}
	if isConsumer {
		switch source {
		case "synctacles":
			return "Worker (wholesale + tax)"
		case "enever":
			return "Enever (consumer price)"
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

// toFloat64 extracts a float64 from a map value (handles both float64 and json.Number).
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
