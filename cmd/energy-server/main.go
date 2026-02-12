// Command energy-server is the Synctacles central price proxy.
// It polls all 30 EU bidding zones using the shared collector library,
// normalizes wholesale prices to consumer prices per country, stores
// results in PostgreSQL, and serves a REST API for addon clients.
//
// Architecture:
//   Collectors (shared with addon) → Normalizer → PostgreSQL → REST API
//   Addons poll GET /api/v1/prices?zone=NL → pre-computed JSON
//
// This is the Tier 0 source for addons. When this server is unreachable,
// addons fall back to direct polling (Energy-Charts, etc.).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/synctacles/energy-go/internal/collector"
	"github.com/synctacles/energy-go/internal/countries"
	"github.com/synctacles/energy-go/internal/engine"
	"github.com/synctacles/energy-go/internal/models"
	"github.com/synctacles/energy-go/internal/store"
)

var version = "dev"

// serverConfig holds settings loaded from environment variables.
type serverConfig struct {
	DatabaseURL string // DATABASE_URL
	ListenAddr  string // LISTEN_ADDR (default :8080)
	DebugMode   bool   // DEBUG_MODE
}

func loadServerConfig() serverConfig {
	cfg := serverConfig{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		ListenAddr:  os.Getenv("LISTEN_ADDR"),
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	cfg.DebugMode = os.Getenv("DEBUG_MODE") == "true"
	return cfg
}

// zoneCache holds pre-computed price responses per zone, refreshed by collectors.
type zoneCache struct {
	mu      sync.RWMutex
	entries map[string]*zoneCacheEntry // zone → cached response
}

type zoneCacheEntry struct {
	prices    []models.HourlyPrice
	source    string
	quality   string
	updatedAt time.Time
}

func newZoneCache() *zoneCache {
	return &zoneCache{entries: make(map[string]*zoneCacheEntry)}
}

func (c *zoneCache) Set(zone string, prices []models.HourlyPrice, source, quality string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[zone] = &zoneCacheEntry{
		prices:    prices,
		source:    source,
		quality:   quality,
		updatedAt: time.Now(),
	}
}

func (c *zoneCache) Get(zone string) *zoneCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entries[zone]
}

// completeness checks how many zones have full today/tomorrow data.
func (c *zoneCache) completeness(zones []string) (todayComplete, tomorrowComplete int) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.Add(24 * time.Hour)

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, zone := range zones {
		entry := c.entries[zone]
		if entry == nil {
			continue
		}
		th := countHoursInDay(entry.prices, today)
		mh := countHoursInDay(entry.prices, tomorrow)
		if th >= 23 {
			todayComplete++
		}
		if mh >= 12 {
			tomorrowComplete++
		}
	}
	return
}

// incompleteZones returns zones that are missing today or tomorrow data.
func (c *zoneCache) incompleteZones(zones []string) []string {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.Add(24 * time.Hour)

	c.mu.RLock()
	defer c.mu.RUnlock()

	var incomplete []string
	for _, zone := range zones {
		entry := c.entries[zone]
		if entry == nil {
			incomplete = append(incomplete, zone)
			continue
		}
		th := countHoursInDay(entry.prices, today)
		mh := countHoursInDay(entry.prices, tomorrow)
		if th < 23 || mh < 12 {
			incomplete = append(incomplete, zone)
		}
	}
	return incomplete
}

func countHoursInDay(prices []models.HourlyPrice, dayStart time.Time) int {
	dayEnd := dayStart.Add(24 * time.Hour)
	count := 0
	for _, p := range prices {
		if !p.Timestamp.Before(dayStart) && p.Timestamp.Before(dayEnd) {
			count++
		}
	}
	return count
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("starting energy-server", "version", version)

	cfg := loadServerConfig()
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	if cfg.DebugMode {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	// Load zone registry (embedded YAML configs)
	registry, err := countries.LoadRegistry()
	if err != nil {
		slog.Error("failed to load country configs", "error", err)
		os.Exit(1)
	}
	allZones := registry.AllZones()
	slog.Info("loaded zone registry", "zones", len(allZones))

	// Connect to PostgreSQL
	pgStore, err := store.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pgStore.Close()
	slog.Info("connected to PostgreSQL")

	// In-memory cache for serving API responses (updated by collectors)
	cache := newZoneCache()

	// Build per-zone fallback managers (reusing the same collector code as the addon)
	normalizers := make(map[string]*engine.Normalizer)
	fallbacks := make(map[string]*engine.FallbackManager)

	for _, zone := range allZones {
		sources := buildServerSourceChain(zone, registry)
		if len(sources) == 0 {
			continue
		}
		fallbacks[zone] = engine.NewFallbackManager(sources, pgStore)
		normalizers[zone] = engine.NewNormalizer(registry)
	}
	slog.Info("source chains configured", "zones", len(fallbacks))

	// Signal context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start collector loop (fetches all zones periodically)
	go runCollectorLoop(ctx, allZones, fallbacks, normalizers, pgStore, cache)

	// Start API server
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok", "version": version})
	})

	r.Get("/api/v1/prices", handlePrices(cache))
	r.Get("/api/v1/zones", handleZones(registry))

	httpSrv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr, "zones", len(allZones))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpSrv.Shutdown(shutdownCtx)

	// Cleanup old prices (keep 7 days)
	if deleted, err := pgStore.Cleanup(7 * 24 * time.Hour); err == nil && deleted > 0 {
		slog.Info("cleanup", "deleted_rows", deleted)
	}

	slog.Info("energy-server stopped")
}

// runCollectorLoop uses adaptive polling: fast when data is incomplete, slow when full.
//
// Intervals:
//   - Cache empty/incomplete:   2 min (fill fast after start/restart)
//   - Today full, no tomorrow:  30 min (or 15 min after 13:00 CET when ENTSO-E publishes)
//   - Today + tomorrow full:    6 hours (nothing changes, just a health check)
//   - Failed zones:             30s quick retry (once, then wait for next cycle)
func runCollectorLoop(ctx context.Context, zones []string, fallbacks map[string]*engine.FallbackManager, normalizers map[string]*engine.Normalizer, pgStore *store.PostgresStore, cache *zoneCache) {
	// Initial fetch — all zones
	failed := fetchZones(ctx, zones, fallbacks, normalizers, cache)

	// Quick retry for any zones that failed on first attempt
	if len(failed) > 0 {
		slog.Info("retrying failed zones", "count", len(failed))
		select {
		case <-ctx.Done():
			return
		case <-time.After(30 * time.Second):
			fetchZones(ctx, failed, fallbacks, normalizers, cache)
		}
	}

	for {
		interval, reason := nextInterval(cache, zones)
		slog.Info("next fetch scheduled", "interval", interval, "reason", reason)

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}

		// Determine which zones need fetching
		target := cache.incompleteZones(zones)
		if len(target) == 0 {
			// All complete — do a full refresh as health check
			target = zones
		}

		failed = fetchZones(ctx, target, fallbacks, normalizers, cache)

		// Quick retry for failed zones (once)
		if len(failed) > 0 {
			slog.Info("retrying failed zones", "count", len(failed))
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
				fetchZones(ctx, failed, fallbacks, normalizers, cache)
			}
		}
	}
}

// nextInterval determines the polling interval based on cache completeness.
func nextInterval(cache *zoneCache, zones []string) (time.Duration, string) {
	total := len(zones)
	todayOK, tomorrowOK := cache.completeness(zones)

	// Less than 80% of zones have today's data → fill fast
	if todayOK < total*80/100 {
		return 2 * time.Minute, fmt.Sprintf("today incomplete (%d/%d)", todayOK, total)
	}

	// Today mostly complete, but tomorrow is missing
	if tomorrowOK < total*80/100 {
		// After 13:00 CET (ENTSO-E publication window) → poll more aggressively
		cet := time.FixedZone("CET", 3600)
		if time.Now().In(cet).Hour() >= 13 {
			return 15 * time.Minute, fmt.Sprintf("awaiting tomorrow prices (%d/%d, post-13h CET)", tomorrowOK, total)
		}
		return 30 * time.Minute, fmt.Sprintf("awaiting tomorrow prices (%d/%d)", tomorrowOK, total)
	}

	// Everything complete → slow health check
	return 6 * time.Hour, fmt.Sprintf("all complete (today=%d/%d, tomorrow=%d/%d)", todayOK, total, tomorrowOK, total)
}

// fetchZones fetches prices for the given zones sequentially with staggered
// delays between requests to the same source. Returns zones that failed.
func fetchZones(ctx context.Context, zones []string, fallbacks map[string]*engine.FallbackManager, normalizers map[string]*engine.Normalizer, cache *zoneCache) []string {
	slog.Info("fetching zones", "count", len(zones))
	start := time.Now()
	now := time.Now().UTC()

	var failed []string
	var successCount int

	for i, zone := range zones {
		if ctx.Err() != nil {
			break
		}

		fb, ok := fallbacks[zone]
		if !ok {
			continue
		}
		norm := normalizers[zone]

		// Stagger: 200ms between zones to spread load across sources
		if i > 0 {
			time.Sleep(200 * time.Millisecond)
		}

		// Fetch today
		todayResult, err := fb.Fetch(ctx, zone, now)
		if err != nil {
			slog.Warn("fetch failed", "zone", zone, "error", err)
			failed = append(failed, zone)
			continue
		}

		// Fetch tomorrow (may not be available yet — that's OK)
		tomorrow := now.Add(24 * time.Hour)
		tomorrowResult, _ := fb.Fetch(ctx, zone, tomorrow)

		// Normalize to consumer prices
		allPrices := norm.ToConsumer(todayResult.Prices)
		if tomorrowResult != nil {
			allPrices = append(allPrices, norm.ToConsumer(tomorrowResult.Prices)...)
		}

		// Update in-memory cache
		cache.Set(zone, allPrices, todayResult.Source, todayResult.Quality)
		successCount++

		slog.Debug("zone fetched", "zone", zone, "source", todayResult.Source, "prices", len(allPrices))
	}

	slog.Info("fetch cycle complete",
		"success", successCount,
		"failed", len(failed),
		"duration", time.Since(start).Round(time.Millisecond),
	)
	return failed
}

// --- API Handlers ---

// priceResponse is the wire format served to addon clients.
type priceResponse struct {
	Zone    string       `json:"zone"`
	Source  string       `json:"source"`
	Quality string       `json:"quality"`
	Prices  []priceEntry `json:"prices"`
}

type priceEntry struct {
	Timestamp  string  `json:"timestamp"`
	PriceEUR   float64 `json:"price_eur"`
	Unit       string  `json:"unit"`
	IsConsumer bool    `json:"is_consumer"`
}

func handlePrices(cache *zoneCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zone := r.URL.Query().Get("zone")
		if zone == "" {
			writeError(w, http.StatusBadRequest, "zone parameter required")
			return
		}

		entry := cache.Get(zone)
		if entry == nil || len(entry.prices) == 0 {
			writeError(w, http.StatusNotFound, fmt.Sprintf("no prices for zone %s", zone))
			return
		}

		// Filter by date if requested
		dateStr := r.URL.Query().Get("date")
		prices := entry.prices
		if dateStr != "" {
			date, err := time.Parse("2006-01-02", dateStr)
			if err == nil {
				dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
				dayEnd := dayStart.Add(24 * time.Hour)
				var filtered []models.HourlyPrice
				for _, p := range prices {
					if !p.Timestamp.Before(dayStart) && p.Timestamp.Before(dayEnd) {
						filtered = append(filtered, p)
					}
				}
				prices = filtered
			}
		}

		resp := priceResponse{
			Zone:    zone,
			Source:  entry.source,
			Quality: entry.quality,
		}
		for _, p := range prices {
			unitStr := "EUR/kWh"
			if p.Unit == models.UnitMWh {
				unitStr = "EUR/MWh"
			}
			resp.Prices = append(resp.Prices, priceEntry{
				Timestamp:  p.Timestamp.Format(time.RFC3339),
				PriceEUR:   p.PriceEUR,
				Unit:       unitStr,
				IsConsumer: p.IsConsumer,
			})
		}

		writeJSON(w, resp)
	}
}

func handleZones(registry *models.ZoneRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"zones": registry.AllZones(),
		})
	}
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=900") // 15 min
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// buildServerSourceChain creates the source list for a zone on the server.
// Unlike the addon, the server does NOT include SynctaclesAPI (that would be recursive).
func buildServerSourceChain(zone string, registry *models.ZoneRegistry) []collector.PriceSource {
	var chain []collector.PriceSource

	cc, ok := registry.GetCountryForZone(zone)
	if !ok {
		return []collector.PriceSource{&collector.EnergyCharts{}}
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

	// Always add Energy-Charts as final fallback
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
