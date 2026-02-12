package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/synctacles/energy-go/internal/collector"
	"github.com/synctacles/energy-go/internal/models"
)

// FallbackManager tries multiple price sources in priority order.
// Tier 1-3: live sources (GO allowed). Tier 4: cache (GO not allowed).
// Includes an in-memory result cache to prevent excessive API calls
// (important for rate-limited sources like Enever: max 250 calls/month).
type FallbackManager struct {
	sources     []collector.PriceSource
	cache       PriceCache
	breaker     *circuitBreaker
	memCache    map[string]*memCacheEntry // zone+date → cached result
	mu          sync.Mutex
}

// memCacheEntry holds a cached fetch result with expiry.
type memCacheEntry struct {
	result    *FetchResult
	fetchedAt time.Time
}

// memCacheTTL controls how long in-memory results are reused before re-fetching.
// Day-ahead prices change once per day, so 2 hours is safe.
// This keeps Enever at ~3 calls/day (well within 250/month free tier).
const memCacheTTL = 2 * time.Hour

// PriceCache is the interface for cached price storage.
type PriceCache interface {
	Get(zone string, date time.Time) ([]models.HourlyPrice, error)
	Put(zone string, prices []models.HourlyPrice) error
}

// NewFallbackManager creates a fallback manager with prioritized sources.
func NewFallbackManager(sources []collector.PriceSource, cache PriceCache) *FallbackManager {
	return &FallbackManager{
		sources:  sources,
		cache:    cache,
		breaker:  newCircuitBreaker(),
		memCache: make(map[string]*memCacheEntry),
	}
}

// FetchResult holds prices with metadata about how they were obtained.
type FetchResult struct {
	Prices  []models.HourlyPrice
	Source  string
	Tier    int    // 1-3 = live, 4 = cache
	Quality string // "live", "cached"
}

// Fetch tries each source in order, falling back on failure.
// Uses an in-memory cache to avoid excessive API calls for rate-limited sources.
func (f *FallbackManager) Fetch(ctx context.Context, zone string, date time.Time) (*FetchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check in-memory cache first (prevents hammering rate-limited APIs)
	cacheKey := zone + ":" + date.Format("2006-01-02")
	if entry, ok := f.memCache[cacheKey]; ok && time.Since(entry.fetchedAt) < memCacheTTL {
		slog.Debug("using in-memory cached result", "source", entry.result.Source, "age", time.Since(entry.fetchedAt).Round(time.Second))
		return entry.result, nil
	}

	// Try live sources (Tier 1-3)
	for i, src := range f.sources {
		if !supportsZone(src, zone) {
			continue
		}
		if f.breaker.isOpen(src.Name()) {
			slog.Debug("circuit breaker open, skipping", "source", src.Name())
			continue
		}

		prices, err := src.FetchDayAhead(ctx, zone, date)
		if err != nil {
			slog.Warn("source failed", "source", src.Name(), "zone", zone, "error", err)
			f.breaker.recordFailure(src.Name())
			continue
		}

		if len(prices) == 0 {
			continue
		}

		f.breaker.recordSuccess(src.Name())

		// Cache the result
		if f.cache != nil {
			if err := f.cache.Put(zone, prices); err != nil {
				slog.Warn("cache put failed", "error", err)
			}
		}

		result := &FetchResult{
			Prices:  prices,
			Source:  src.Name(),
			Tier:    i + 1,
			Quality: "live",
		}

		// Store in memory cache to prevent re-fetching
		f.memCache[cacheKey] = &memCacheEntry{result: result, fetchedAt: time.Now()}

		return result, nil
	}

	// Tier 4: Try cache
	if f.cache != nil {
		prices, err := f.cache.Get(zone, date)
		if err == nil && len(prices) > 0 {
			slog.Info("using cached prices", "zone", zone, "count", len(prices))
			return &FetchResult{
				Prices:  prices,
				Source:  prices[0].Source,
				Tier:    4,
				Quality: "cached",
			}, nil
		}
	}

	return nil, fmt.Errorf("all sources failed for zone %s on %s", zone, date.Format("2006-01-02"))
}

// AllowGo returns true if the fetch quality allows GO recommendations.
// Only live data (Tier 1-3) that is < 6 hours old allows GO.
func (r *FetchResult) AllowGo() bool {
	return r.Tier <= 3 && r.Quality == "live"
}

// SourceHealth describes the health of a single price source.
type SourceHealth struct {
	Name        string `json:"name"`
	Healthy     bool   `json:"healthy"`
	RequiresKey bool   `json:"requires_key"`
	Active      bool   `json:"active"`    // true if this source was used for the last successful fetch
	LastFailure string `json:"last_failure,omitempty"` // ISO 8601 timestamp, empty if healthy
}

// SourceStatus returns the health status of all configured sources.
func (f *FallbackManager) SourceStatus(activeSource string) []SourceHealth {
	f.mu.Lock()
	defer f.mu.Unlock()

	var statuses []SourceHealth
	for _, src := range f.sources {
		sh := SourceHealth{
			Name:        src.Name(),
			RequiresKey: src.RequiresKey(),
			Active:      src.Name() == activeSource,
		}
		if t, ok := f.breaker.failures[src.Name()]; ok && time.Since(t) < f.breaker.cooldown {
			sh.Healthy = false
			sh.LastFailure = t.Format(time.RFC3339)
		} else {
			sh.Healthy = true
		}
		statuses = append(statuses, sh)
	}
	return statuses
}

func supportsZone(src collector.PriceSource, zone string) bool {
	for _, z := range src.Zones() {
		if z == zone {
			return true
		}
	}
	return false
}

// circuitBreaker prevents hammering failed sources.
type circuitBreaker struct {
	failures map[string]time.Time // source → last failure time
	cooldown time.Duration
}

func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{
		failures: make(map[string]time.Time),
		cooldown: 2 * time.Hour,
	}
}

func (cb *circuitBreaker) isOpen(source string) bool {
	t, ok := cb.failures[source]
	if !ok {
		return false
	}
	return time.Since(t) < cb.cooldown
}

func (cb *circuitBreaker) recordFailure(source string) {
	cb.failures[source] = time.Now()
}

func (cb *circuitBreaker) recordSuccess(source string) {
	delete(cb.failures, source)
}
