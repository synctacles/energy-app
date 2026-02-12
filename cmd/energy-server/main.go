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

// runCollectorLoop fetches prices for all zones every 15 minutes.
func runCollectorLoop(ctx context.Context, zones []string, fallbacks map[string]*engine.FallbackManager, normalizers map[string]*engine.Normalizer, pgStore *store.PostgresStore, cache *zoneCache) {
	// Initial fetch
	fetchAllZones(ctx, zones, fallbacks, normalizers, cache)

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fetchAllZones(ctx, zones, fallbacks, normalizers, cache)
		}
	}
}

// fetchAllZones fetches prices for all zones concurrently.
func fetchAllZones(ctx context.Context, zones []string, fallbacks map[string]*engine.FallbackManager, normalizers map[string]*engine.Normalizer, cache *zoneCache) {
	slog.Info("fetching all zones", "count", len(zones))
	start := time.Now()
	now := time.Now().UTC()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount, failCount int

	// Limit concurrency to avoid overwhelming APIs
	sem := make(chan struct{}, 5)

	for _, zone := range zones {
		fb, ok := fallbacks[zone]
		if !ok {
			continue
		}
		norm := normalizers[zone]

		wg.Add(1)
		go func(zone string, fb *engine.FallbackManager, norm *engine.Normalizer) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			// Fetch today + tomorrow
			todayResult, err := fb.Fetch(ctx, zone, now)
			if err != nil {
				slog.Warn("fetch failed", "zone", zone, "error", err)
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}

			tomorrow := now.Add(24 * time.Hour)
			tomorrowResult, _ := fb.Fetch(ctx, zone, tomorrow) // Tomorrow may not be available yet

			// Normalize to consumer prices
			allPrices := norm.ToConsumer(todayResult.Prices)
			if tomorrowResult != nil {
				allPrices = append(allPrices, norm.ToConsumer(tomorrowResult.Prices)...)
			}

			// Update in-memory cache
			cache.Set(zone, allPrices, todayResult.Source, todayResult.Quality)

			mu.Lock()
			successCount++
			mu.Unlock()

			slog.Debug("zone fetched", "zone", zone, "source", todayResult.Source, "prices", len(allPrices))
		}(zone, fb, norm)
	}

	wg.Wait()
	slog.Info("fetch cycle complete",
		"success", successCount,
		"failed", failCount,
		"duration", time.Since(start).Round(time.Millisecond),
	)
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
