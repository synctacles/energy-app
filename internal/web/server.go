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
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/synctacles/energy-app/internal/config"
	"github.com/synctacles/energy-app/internal/plan"
	"github.com/synctacles/energy-backend/pkg/engine"
	"github.com/synctacles/energy-app/internal/ha"
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
	version             string
	detectedPowerSensor string
	addonSlug           string
	planRegistry        *plan.Registry
}

// Deps holds dependencies for the web server.
type Deps struct {
	Config              *config.Config
	StateStore          *state.Store
	SensorData          *SensorData
	Supervisor          *ha.SupervisorClient
	Fallback            *engine.FallbackManager
	Version             string
	DetectedPowerSensor string
	AddonSlug           string
	PlanRegistry        *plan.Registry
}

// NewServer creates a new energy addon web server.
func NewServer(deps Deps) *Server {
	s := &Server{
		cfg:                 deps.Config,
		stateStore:          deps.StateStore,
		sensorData:          deps.SensorData,
		supervisor:          deps.Supervisor,
		fallback:            deps.Fallback,
		version:             deps.Version,
		detectedPowerSensor: deps.DetectedPowerSensor,
		addonSlug:           deps.AddonSlug,
		planRegistry:        deps.PlanRegistry,
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

		// Config & Plans
		r.Get("/config", s.handleConfig)
		r.Post("/config", s.handleConfigSave)
		r.Get("/plans", s.handlePlans)

		// Sources health
		r.Get("/sources", s.handleSources)

		// Feedback
		r.Get("/feedback/sysinfo", s.handleFeedbackSysInfo)
		r.Post("/feedback/rating", s.handleFeedbackRating)
		r.Post("/feedback/bug", s.handleFeedbackBug)
	})

	// Static files and SPA
	staticSub, err := fs.Sub(staticFS, "static")
	if err == nil {
		fileServer := http.FileServer(http.FS(staticSub))
		r.Handle("/static/*", http.StripPrefix("/static/", fileServer))
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
		"version":        s.version,
		"zone":           s.cfg.BiddingZone,
		"is_pro":         true,
		"tier":           "pro",
		"gate_status":    "full",
		"can_actions":    true,
		"can_tomorrow":   true,
		"all_features":   true,
		"alerts_enabled": s.cfg.HasAlerts(),
	}
	if s.addonSlug != "" {
		resp["addon_slug"] = s.addonSlug
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

	// All features are free
	dashboard["is_pro"] = true
	dashboard["tier"] = "pro"

	writeJSON(w, dashboard)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"plan":                    s.cfg.PlanID,
		"zone":                    s.cfg.BiddingZone,
		"currency":                s.cfg.Currency,
		"go_threshold":            s.cfg.GoThreshold,
		"avoid_threshold":         s.cfg.AvoidThreshold,
		"best_window_hours":       s.cfg.BestWindowHours,
		"has_power_sensor":        s.cfg.HasPowerSensor(),
		"power_sensor":            s.cfg.PowerSensorEntity,
		"enever_enabled":          s.cfg.EneverEnabled,
		"enever_token":            s.cfg.EneverToken,
		"enever_leverancier":      s.cfg.EneverLeverancier,
		"coefficient":             s.cfg.Coefficient,
		"debug_mode":              s.cfg.DebugMode,
		"detected_power_sensor":   s.detectedPowerSensor,
		"is_pro":                  true,
		"tier":                    "pro",
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
	allowed := []string{"plan", "zone", "go_threshold", "avoid_threshold", "best_window_hours",
		"enever_enabled", "enever_token", "enever_leverancier",
		"coefficient", "power_sensor", "debug_mode"}
	for _, key := range allowed {
		if val, ok := incoming[key]; ok {
			current[key] = val
		}
	}

	// Derive zone + Enever settings from plan (so HA Supervisor has correct state on restart)
	if planID, ok := incoming["plan"].(string); ok {
		if p := s.planRegistry.Get(planID); p != nil {
			current["zone"] = p.Zone
			current["enever_enabled"] = p.HasEnever()
			if p.HasEnever() {
				current["enever_leverancier"] = p.EneverSupplier
			}
		}
	}

	// Write back to Supervisor (single call with all fields + derived)
	if err := s.supervisor.SetAddonOptions(r.Context(), current); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save options")
		return
	}

	// Update in-memory config for immediate effect
	if v, ok := incoming["best_window_hours"].(float64); ok {
		s.cfg.BestWindowHours = int(v)
	}
	if v, ok := incoming["enever_token"].(string); ok {
		s.cfg.EneverToken = v
	}
	if v, ok := incoming["plan"].(string); ok {
		s.cfg.PlanID = v
		if p := s.planRegistry.Get(v); p != nil {
			s.cfg.ApplyPlan(p.Zone, p.EneverSupplier)
		}
	}

	writeJSON(w, map[string]string{"status": "saved", "message": "Settings saved. Restart app for source chain changes."})
}

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"groups":  s.planRegistry.Grouped(),
		"current": s.cfg.PlanID,
	})
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
