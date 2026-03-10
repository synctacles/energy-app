package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"errors"

	"github.com/synctacles/energy-app/pkg/collector"
	"github.com/synctacles/energy-app/pkg/models"
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
	lastSuccess map[string]time.Time      // source → last successful fetch time
	mu          sync.Mutex
}

// memCacheEntry holds a cached fetch result with expiry.
type memCacheEntry struct {
	result        *FetchResult
	fetchedAt     time.Time
	fromDiskCache bool // true if this entry was warmed from SQLite
}

// memCacheTTL controls how long in-memory results are reused before re-fetching.
// Day-ahead prices change once per day, so 2 hours is safe.
// This keeps Enever at ~3 calls/day (well within 250/month free tier).
const memCacheTTL = 2 * time.Hour

// estimatedCacheTTL is used for estimated/non-live data during the retry window
// (13:00-21:30 UTC). The backend retries non-ok zones hourly until 21:00 UTC;
// the shorter TTL lets the client pick up recovered data within 15 minutes.
// Outside the retry window, memCacheTTL is used (no backend retries expected).
const estimatedCacheTTL = 15 * time.Minute

// PriceCache is the interface for cached price storage.
type PriceCache interface {
	Get(zone string, date time.Time) ([]models.HourlyPrice, error)
	Put(zone string, prices []models.HourlyPrice) error
}

// SmartPriceCache extends PriceCache with provenance metadata.
// Implementations that support this enable cache warming after reboot:
// if SQLite has complete, live-quality prices, they are used directly
// instead of making unnecessary API calls.
type SmartPriceCache interface {
	PriceCache
	GetWithMeta(zone string, date time.Time) (*models.CacheEntry, error)
	PutWithTier(zone string, prices []models.HourlyPrice, tier int) error
}

// NewFallbackManager creates a fallback manager with prioritized sources.
func NewFallbackManager(sources []collector.PriceSource, cache PriceCache) *FallbackManager {
	return &FallbackManager{
		sources:     sources,
		cache:       cache,
		breaker:     newCircuitBreaker(),
		memCache:    make(map[string]*memCacheEntry),
		lastSuccess: make(map[string]time.Time),
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
// After a reboot, checks SQLite for complete live-quality data before hitting APIs.
func (f *FallbackManager) Fetch(ctx context.Context, zone string, date time.Time) (*FetchResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check in-memory cache first (prevents hammering rate-limited APIs)
	cacheKey := zone + ":" + date.Format("2006-01-02")
	if entry, ok := f.memCache[cacheKey]; ok {
		ttl := cacheTTLFor(entry.result)
		if time.Since(entry.fetchedAt) < ttl {
			slog.Debug("using in-memory cached result", "source", entry.result.Source, "age", time.Since(entry.fetchedAt).Round(time.Second))
			return entry.result, nil
		}
	}

	// Check SQLite for complete live-quality data before hitting APIs.
	// Day-ahead prices are immutable once published, so cached live data is still valid.
	if result := f.tryWarmFromDisk(zone, date, cacheKey); result != nil {
		return result, nil
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
			// Use Retry-After duration from 429 responses as circuit breaker cooldown
			var rateLimited *collector.ErrRateLimited
			if errors.As(err, &rateLimited) {
				f.breaker.recordFailureWithCooldown(src.Name(), rateLimited.RetryAfter)
			} else {
				f.breaker.recordFailure(src.Name())
			}
			continue
		}

		if len(prices) == 0 {
			continue
		}

		f.breaker.recordSuccess(src.Name())
		f.lastSuccess[src.Name()] = time.Now()

		// Cache the result (with tier metadata if supported)
		if f.cache != nil {
			if smart, ok := f.cache.(SmartPriceCache); ok {
				if err := smart.PutWithTier(zone, prices, i+1); err != nil {
					slog.Warn("cache put failed", "error", err)
				}
			} else {
				if err := f.cache.Put(zone, prices); err != nil {
					slog.Warn("cache put failed", "error", err)
				}
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

// tryWarmFromDisk checks SQLite for complete, live-quality data and warms the memCache.
// Day-ahead prices are immutable once published, so if SQLite has 24 prices for a date
// from a live source (tier 1-3), they are valid forever for that date.
// Also loads tomorrow's prices if available.
func (f *FallbackManager) tryWarmFromDisk(zone string, date time.Time, cacheKey string) *FetchResult {
	smart, ok := f.cache.(SmartPriceCache)
	if !ok {
		return nil
	}

	entry, err := smart.GetWithMeta(zone, date)
	if err != nil || entry == nil || len(entry.Prices) < 24 {
		return nil
	}
	if entry.OriginalTier < 1 || entry.OriginalTier > 3 {
		return nil // legacy (tier 0) or cache-tier data — don't trust for warming
	}
	// If the primary source changed (e.g. mode switch to Enever), skip disk cache
	// so the new source gets a chance to provide data.
	if len(f.sources) > 0 && entry.Prices[0].Source != f.sources[0].Name() {
		slog.Info("disk cache source mismatch, forcing live fetch",
			"cached_source", entry.Prices[0].Source, "primary_source", f.sources[0].Name())
		return nil
	}

	prices := entry.Prices

	// Also load tomorrow's prices if available
	tomorrow := date.Truncate(24*time.Hour).AddDate(0, 0, 1)
	tomorrowEntry, err := smart.GetWithMeta(zone, tomorrow)
	if err == nil && tomorrowEntry != nil && len(tomorrowEntry.Prices) > 0 {
		prices = append(prices, tomorrowEntry.Prices...)
	}

	result := &FetchResult{
		Prices:  prices,
		Source:  entry.Prices[0].Source,
		Tier:    entry.OriginalTier,
		Quality: entry.Prices[0].Quality,
	}
	f.memCache[cacheKey] = &memCacheEntry{
		result:        result,
		fetchedAt:     entry.FetchedAt,
		fromDiskCache: true,
	}

	slog.Info("warm from disk cache", "zone", zone, "source", result.Source,
		"tier", result.Tier, "quality", result.Quality,
		"today", len(entry.Prices), "tomorrow", len(prices)-len(entry.Prices))

	return result
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
	Active      bool   `json:"active"`                 // true if this source provided the current data
	Primary     bool   `json:"primary"`                // true if first in chain (user's configured preference)
	Tier        int    `json:"tier"`                    // position in fallback chain (1-based)
	LastFailure string `json:"last_failure,omitempty"`  // ISO 8601 timestamp, empty if healthy
	LastSuccess string `json:"last_success,omitempty"`  // ISO 8601 timestamp of last successful fetch
}

// ActiveSourceInfo describes the source of currently served data.
// This separates data provenance from API health: a source can be unhealthy
// (red dot) while its cached data is still being served (valid and [stored]).
type ActiveSourceInfo struct {
	Source        string `json:"source"`
	Quality       string `json:"quality"`
	Tier          int    `json:"tier"`
	FromMemCache  bool   `json:"from_mem_cache"`
	FromDiskCache bool   `json:"from_disk_cache"`
	FetchedAt     string `json:"fetched_at"`
}

// SourceStatus returns the health status of all configured sources.
func (f *FallbackManager) SourceStatus(activeSource string) []SourceHealth {
	f.mu.Lock()
	defer f.mu.Unlock()

	var statuses []SourceHealth
	for i, src := range f.sources {
		sh := SourceHealth{
			Name:        src.Name(),
			RequiresKey: src.RequiresKey(),
			Active:      src.Name() == activeSource,
			Primary:     i == 0,
			Tier:        i + 1,
		}
		cd := f.breaker.cooldown
		if custom, ok := f.breaker.cooldowns[src.Name()]; ok {
			cd = custom
		}
		if t, ok := f.breaker.failures[src.Name()]; ok && time.Since(t) < cd {
			sh.Healthy = false
			sh.LastFailure = t.Format(time.RFC3339)
		} else {
			sh.Healthy = true
		}
		if t, ok := f.lastSuccess[src.Name()]; ok {
			sh.LastSuccess = t.Format(time.RFC3339)
		}
		statuses = append(statuses, sh)
	}
	return statuses
}

// ActiveInfo returns metadata about the data currently being served for a zone.
// It reveals whether data comes from memory/disk cache and which source originally provided it,
// even if that source's circuit breaker is now open.
func (f *FallbackManager) ActiveInfo(zone string, date time.Time) *ActiveSourceInfo {
	f.mu.Lock()
	defer f.mu.Unlock()

	cacheKey := zone + ":" + date.Format("2006-01-02")
	entry, ok := f.memCache[cacheKey]
	if !ok || entry.result == nil {
		return nil
	}

	return &ActiveSourceInfo{
		Source:        entry.result.Source,
		Quality:       entry.result.Quality,
		Tier:          entry.result.Tier,
		FromMemCache:  !entry.fromDiskCache,
		FromDiskCache: entry.fromDiskCache,
		FetchedAt:     entry.fetchedAt.Format(time.RFC3339),
	}
}

// ClearMemCache removes all in-memory cached results, forcing the next
// Fetch call to re-query live sources or SQLite.
func (f *FallbackManager) ClearMemCache() {
	f.mu.Lock()
	f.memCache = make(map[string]*memCacheEntry)
	f.mu.Unlock()
}

// cacheTTLFor returns the appropriate cache TTL for a fetch result.
// Estimated data uses a shorter TTL during the backend retry window (13:00-21:30 UTC)
// so the client picks up recovered data quickly. Outside the window, the standard
// 2-hour TTL is used since no backend retries are expected.
func cacheTTLFor(r *FetchResult) time.Duration {
	if r == nil {
		return memCacheTTL
	}
	isEstimated := r.Source == "estimated" || r.Quality != "live"
	if !isEstimated {
		return memCacheTTL
	}
	h := time.Now().UTC().Hour()
	m := time.Now().UTC().Minute()
	inRetryWindow := (h >= 13 && h < 21) || (h == 21 && m <= 30)
	if inRetryWindow {
		return estimatedCacheTTL
	}
	return memCacheTTL
}

// FetchWholesaleForZone calls non-Enever sources to get wholesale prices
// for comparison. Returns a map of exact timestamp → wholesale EUR/kWh.
// Preserves PT15 granularity when the Worker provides quarter-hour data.
// Used by the cache view to compute breakdown + delta in Enever mode.
func (f *FallbackManager) FetchWholesaleForZone(ctx context.Context, zone string, date time.Time) map[time.Time]float64 {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check in-memory cache (keyed differently to not collide with primary fetch)
	cacheKey := "wholesale:" + zone + ":" + date.Format("2006-01-02")
	if entry, ok := f.memCache[cacheKey]; ok && time.Since(entry.fetchedAt) < memCacheTTL {
		result := make(map[time.Time]float64)
		for _, p := range entry.result.Prices {
			if p.WholesaleKWh > 0 {
				result[p.Timestamp] = p.WholesaleKWh
			}
		}
		return result
	}

	for _, src := range f.sources {
		if src.Name() == "enever" || !supportsZone(src, zone) {
			continue
		}

		prices, err := src.FetchDayAhead(ctx, zone, date)
		if err != nil {
			slog.Debug("wholesale comparison fetch failed", "source", src.Name(), "error", err)
			continue
		}

		result := make(map[time.Time]float64)
		for _, p := range prices {
			if p.WholesaleKWh > 0 {
				result[p.Timestamp] = p.WholesaleKWh
			}
		}

		// Cache for reuse
		f.memCache[cacheKey] = &memCacheEntry{
			result:    &FetchResult{Prices: prices, Source: src.Name()},
			fetchedAt: time.Now(),
		}

		return result
	}

	return nil
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
// Supports per-source cooldown durations (e.g. from 429 Retry-After headers).
type circuitBreaker struct {
	failures  map[string]time.Time    // source → last failure time
	cooldowns map[string]time.Duration // source → custom cooldown (from Retry-After)
	cooldown  time.Duration            // default cooldown
}

func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{
		failures:  make(map[string]time.Time),
		cooldowns: make(map[string]time.Duration),
		cooldown:  2 * time.Hour,
	}
}

func (cb *circuitBreaker) isOpen(source string) bool {
	t, ok := cb.failures[source]
	if !ok {
		return false
	}
	cd := cb.cooldown
	if custom, ok := cb.cooldowns[source]; ok {
		cd = custom
	}
	return time.Since(t) < cd
}

func (cb *circuitBreaker) recordFailure(source string) {
	cb.failures[source] = time.Now()
	delete(cb.cooldowns, source) // use default cooldown
}

// recordFailureWithCooldown records a failure with a custom cooldown duration.
// Used for 429 Retry-After responses to respect the server's requested wait time.
func (cb *circuitBreaker) recordFailureWithCooldown(source string, cooldown time.Duration) {
	cb.failures[source] = time.Now()
	cb.cooldowns[source] = cooldown
}

func (cb *circuitBreaker) recordSuccess(source string) {
	delete(cb.failures, source)
	delete(cb.cooldowns, source)
}
