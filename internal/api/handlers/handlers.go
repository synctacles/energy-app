package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/synctacles/energy-go/internal/api/database"
	"github.com/synctacles/energy-go/internal/api/middleware"
	"github.com/synctacles/energy-go/internal/api/models"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	db     *database.Repository
	logger *slog.Logger
}

// New creates a new Handler with the given dependencies.
func New(dbpool *pgxpool.Pool, logger *slog.Logger) *Handler {
	return &Handler{
		db:     database.New(dbpool),
		logger: logger,
	}
}

// Health handles GET /health - returns service health status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "ok",
		"version": "2.0.0",
		"service": "energy-api",
		"time":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Metrics handles GET /metrics - returns Prometheus-compatible metrics.
func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	metrics := []string{
		"# HELP energy_api_info Energy API information",
		"# TYPE energy_api_info gauge",
		`energy_api_info{version="2.0.0",service="energy-api"} 1`,
		"",
		"# HELP energy_api_up Service up status",
		"# TYPE energy_api_up gauge",
		"energy_api_up 1",
		"",
	}

	// Add database pool stats if available
	if h.db != nil {
		metrics = append(metrics,
			"# HELP energy_api_db_connections Database connection pool stats",
			"# TYPE energy_api_db_connections gauge",
			"energy_api_db_connections{state=\"idle\"} 0",
			"energy_api_db_connections{state=\"active\"} 0",
			"",
		)
	}

	for _, line := range metrics {
		w.Write([]byte(line + "\n"))
	}
}

// GetPrices handles GET /api/v1/prices - returns energy prices for a zone (Free tier).
func (h *Handler) GetPrices(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = "NL" // Default to Netherlands
	}

	// Parse date range (optional)
	startParam := r.URL.Query().Get("start")
	endParam := r.URL.Query().Get("end")

	var start, end time.Time
	now := time.Now()

	if startParam != "" {
		parsedStart, err := time.Parse(time.RFC3339, startParam)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_start_time", "Start time must be in RFC3339 format")
			return
		}
		start = parsedStart
	} else {
		// Default: start of today
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	if endParam != "" {
		parsedEnd, err := time.Parse(time.RFC3339, endParam)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid_end_time", "End time must be in RFC3339 format")
			return
		}
		end = parsedEnd
	} else {
		// Default: end of tomorrow
		tomorrow := now.AddDate(0, 0, 1)
		end = time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, tomorrow.Location())
	}

	// Fetch prices from database
	prices, err := h.db.GetPrices(r.Context(), zone, start, end)
	if err != nil {
		h.logger.Error("Failed to get prices", "error", err, "zone", zone)
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to fetch prices")
		return
	}

	// Build response
	response := models.PricesResponse{
		Zone:      zone,
		Prices:    prices,
		Count:     len(prices),
		StartTime: start,
		EndTime:   end,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetNow handles GET /api/v1/now - returns current price (Free tier, deprecated).
func (h *Handler) GetNow(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = "NL"
	}

	currentPrice, err := h.db.GetCurrentPrice(r.Context(), zone)
	if err != nil {
		h.logger.Error("Failed to get current price", "error", err, "zone", zone)
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to fetch current price")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentPrice)
}

// GetDashboard handles GET /api/v1/dashboard - returns dashboard data (Pro tier).
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = "NL"
	}

	// Get current price
	currentPrice, err := h.db.GetCurrentPrice(r.Context(), zone)
	if err != nil {
		h.logger.Error("Failed to get current price", "error", err, "zone", zone)
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to fetch current price")
		return
	}

	// Get today's statistics
	avg, min, max, err := h.db.GetTodayStats(r.Context(), zone)
	if err != nil {
		h.logger.Error("Failed to get today stats", "error", err, "zone", zone)
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to fetch statistics")
		return
	}

	// Get next hour price
	nextHour, _ := h.db.GetNextHourPrice(r.Context(), zone)

	// Get cheapest and most expensive today
	cheapest, _ := h.db.GetCheapestToday(r.Context(), zone)
	mostExpensive, _ := h.db.GetMostExpensiveToday(r.Context(), zone)

	response := models.DashboardResponse{
		Zone:          zone,
		CurrentPrice:  currentPrice.PriceEUR,
		CurrentTime:   currentPrice.Timestamp,
		TodayAvg:      avg,
		TodayMin:      min,
		TodayMax:      max,
		NextHour:      nextHour,
		Cheapest:      cheapest,
		MostExpensive: mostExpensive,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetBestWindow handles GET /api/v1/best-window - finds cheapest time window (Pro tier).
func (h *Handler) GetBestWindow(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = "NL"
	}

	windowHoursParam := r.URL.Query().Get("hours")
	windowHours := 4 // Default to 4 hours
	if windowHoursParam != "" {
		parsed, err := strconv.Atoi(windowHoursParam)
		if err != nil || parsed < 1 || parsed > 24 {
			h.writeError(w, http.StatusBadRequest, "invalid_hours", "Hours must be between 1 and 24")
			return
		}
		windowHours = parsed
	}

	// Default to today + tomorrow
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := now.AddDate(0, 0, 1)
	end := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, tomorrow.Location())

	// Calculate best window
	bestWindow, err := h.db.GetBestWindow(r.Context(), zone, windowHours, start, end)
	if err != nil {
		h.logger.Error("Failed to calculate best window", "error", err, "zone", zone)
		h.writeError(w, http.StatusInternalServerError, "calculation_error", "Failed to calculate best window")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bestWindow)
}

// GetAction handles GET /api/v1/energy-action - returns GO/WAIT/AVOID recommendation (Pro tier).
func (h *Handler) GetAction(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = "NL"
	}

	// Get current price
	currentPrice, err := h.db.GetCurrentPrice(r.Context(), zone)
	if err != nil {
		h.logger.Error("Failed to get current price", "error", err, "zone", zone)
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to fetch current price")
		return
	}

	// Get today's average
	avg, _, _, err := h.db.GetTodayStats(r.Context(), zone)
	if err != nil {
		h.logger.Error("Failed to get today stats", "error", err, "zone", zone)
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to fetch statistics")
		return
	}

	// Determine action based on price vs average
	var action models.EnergyAction
	var reason string

	priceDiff := currentPrice.PriceEUR - avg
	percentDiff := (priceDiff / avg) * 100

	if percentDiff < -10 {
		action = models.ActionGo
		reason = fmt.Sprintf("Price is %.1f%% below average - great time to use energy", -percentDiff)
	} else if percentDiff > 10 {
		action = models.ActionAvoid
		reason = fmt.Sprintf("Price is %.1f%% above average - avoid energy use", percentDiff)
	} else {
		action = models.ActionWait
		reason = "Price is near average - consider waiting for better prices"
	}

	response := models.EnergyActionResponse{
		Zone:         zone,
		Action:       action,
		CurrentPrice: currentPrice.PriceEUR,
		TodayAvg:     avg,
		Reason:       reason,
		Timestamp:    currentPrice.Timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetTomorrow handles GET /api/v1/tomorrow - returns tomorrow's prices (Pro tier).
func (h *Handler) GetTomorrow(w http.ResponseWriter, r *http.Request) {
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = "NL"
	}

	prices, err := h.db.GetTomorrowPrices(r.Context(), zone)
	if err != nil {
		h.logger.Error("Failed to get tomorrow prices", "error", err, "zone", zone)
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to fetch tomorrow's prices")
		return
	}

	if len(prices) == 0 {
		h.writeError(w, http.StatusNotFound, "no_prices", "Tomorrow's prices are not yet available")
		return
	}

	response := map[string]interface{}{
		"zone":   zone,
		"prices": prices,
		"count":  len(prices),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetBalance handles GET /api/v1/balance - returns energy balance data (Pro tier).
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement balance calculation
	// This would require additional data about energy consumption vs production

	auth := middleware.GetAuthValidation(r)
	response := map[string]interface{}{
		"message": "Balance endpoint coming soon",
		"user_id": auth.UserID,
		"tier":    auth.ProductTier,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeError writes a JSON error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := map[string]string{
		"detail":  errorCode,
		"message": message,
	}

	json.NewEncoder(w).Encode(response)
}
