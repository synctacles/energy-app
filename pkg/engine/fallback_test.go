package engine

import (
	"context"
	"testing"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
	"github.com/synctacles/energy-app/pkg/store"
)

// ============================================================================
// FetchResult.AllowGo tests
// ============================================================================

func TestAllowGo_LiveTiers(t *testing.T) {
	for tier := 1; tier <= 3; tier++ {
		r := &FetchResult{Tier: tier, Quality: "live"}
		if !r.AllowGo() {
			t.Errorf("Tier %d live should allow GO", tier)
		}
	}
}

func TestAllowGo_CacheTier(t *testing.T) {
	r := &FetchResult{Tier: 4, Quality: "cached"}
	if r.AllowGo() {
		t.Error("Tier 4 cached should not allow GO")
	}
}

func TestAllowGo_LiveButHighTier(t *testing.T) {
	r := &FetchResult{Tier: 4, Quality: "live"}
	if r.AllowGo() {
		t.Error("Tier 4 should not allow GO even if quality is live")
	}
}

func TestAllowGo_CachedLowTier(t *testing.T) {
	r := &FetchResult{Tier: 1, Quality: "cached"}
	if r.AllowGo() {
		t.Error("cached quality should not allow GO even at Tier 1")
	}
}

// ============================================================================
// circuitBreaker tests
// ============================================================================

func TestCircuitBreaker_InitiallyClosed(t *testing.T) {
	cb := newCircuitBreaker()
	if cb.isOpen("source1") {
		t.Error("new circuit breaker should be closed for unknown source")
	}
}

func TestCircuitBreaker_OpensOnFailure(t *testing.T) {
	cb := newCircuitBreaker()
	cb.recordFailure("source1")
	if !cb.isOpen("source1") {
		t.Error("should be open immediately after failure")
	}
}

func TestCircuitBreaker_ClosesAfterCooldown(t *testing.T) {
	cb := newCircuitBreaker()
	cb.cooldown = 10 * time.Millisecond // short for test
	cb.recordFailure("source1")

	if !cb.isOpen("source1") {
		t.Error("should be open immediately after failure")
	}

	time.Sleep(15 * time.Millisecond)
	if cb.isOpen("source1") {
		t.Error("should be closed after cooldown expires")
	}
}

func TestCircuitBreaker_ClosesOnSuccess(t *testing.T) {
	cb := newCircuitBreaker()
	cb.recordFailure("source1")
	if !cb.isOpen("source1") {
		t.Fatal("should be open after failure")
	}

	cb.recordSuccess("source1")
	if cb.isOpen("source1") {
		t.Error("should be closed after success")
	}
}

func TestCircuitBreaker_IndependentSources(t *testing.T) {
	cb := newCircuitBreaker()
	cb.recordFailure("source1")

	if cb.isOpen("source2") {
		t.Error("failure of source1 should not affect source2")
	}
}

func TestCircuitBreaker_CustomCooldown(t *testing.T) {
	cb := newCircuitBreaker()
	cb.cooldown = 1 * time.Hour // default: very long

	// Custom cooldown: very short
	cb.recordFailureWithCooldown("source1", 10*time.Millisecond)
	if !cb.isOpen("source1") {
		t.Fatal("should be open after failure")
	}

	time.Sleep(15 * time.Millisecond)
	if cb.isOpen("source1") {
		t.Error("should be closed after custom cooldown expires (not default)")
	}
}

func TestCircuitBreaker_CustomCooldownClearedOnSuccess(t *testing.T) {
	cb := newCircuitBreaker()
	cb.recordFailureWithCooldown("source1", 5*time.Hour)
	cb.recordSuccess("source1")

	// After success, custom cooldown should be cleared
	cb.recordFailure("source1")
	// Should use default cooldown now, not the 5h custom one
	if _, ok := cb.cooldowns["source1"]; ok {
		t.Error("custom cooldown should be cleared after success")
	}
}

func TestCircuitBreaker_DefaultCooldownUsedAfterRegularFailure(t *testing.T) {
	cb := newCircuitBreaker()
	cb.cooldown = 10 * time.Millisecond

	// First: custom cooldown
	cb.recordFailureWithCooldown("source1", 5*time.Hour)
	// Then: regular failure should clear custom and use default
	cb.recordFailure("source1")

	if _, ok := cb.cooldowns["source1"]; ok {
		t.Error("regular failure should clear custom cooldown")
	}

	time.Sleep(15 * time.Millisecond)
	if cb.isOpen("source1") {
		t.Error("should be closed after default cooldown")
	}
}

// ============================================================================
// memCache TTL test
// ============================================================================

func TestMemCacheTTL_Constant(t *testing.T) {
	if memCacheTTL != 2*time.Hour {
		t.Errorf("memCacheTTL should be 2h, got %v", memCacheTTL)
	}
}

// ============================================================================
// Smart cache warming tests
// ============================================================================

func make24Prices(date time.Time, source, quality string) []models.HourlyPrice {
	prices := make([]models.HourlyPrice, 24)
	for i := range prices {
		prices[i] = models.HourlyPrice{
			Timestamp: date.Add(time.Duration(i) * time.Hour),
			PriceEUR:  0.20 + float64(i)*0.01,
			Unit:      models.UnitKWh,
			Source:    source,
			Quality:   quality,
			Zone:      "NL",
		}
	}
	return prices
}

func TestFetch_WarmsFromSQLite(t *testing.T) {
	// Setup: create SQLite cache with complete live data
	sqlCache, err := store.NewSQLiteCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlCache.Close()

	date := time.Now().UTC().Truncate(24 * time.Hour)
	prices := make24Prices(date, "synctacles", "live")

	// Store with tier 1 (primary live source)
	if err := sqlCache.PutWithTier("NL", prices, 1); err != nil {
		t.Fatal(err)
	}

	// Create a NEW FallbackManager (simulates reboot — empty memCache)
	// No live sources configured → would fail if SQLite warming doesn't work
	fm := NewFallbackManager(nil, sqlCache)

	// Fetch should warm from SQLite without hitting any live source
	result, err := fm.Fetch(context.Background(), "NL", date)
	if err != nil {
		t.Fatalf("Fetch should succeed from SQLite: %v", err)
	}

	if result.Source != "synctacles" {
		t.Errorf("expected source synctacles, got %s", result.Source)
	}
	if result.Tier != 1 {
		t.Errorf("expected tier 1, got %d", result.Tier)
	}
	if result.Quality != "live" {
		t.Errorf("expected quality live, got %s", result.Quality)
	}
	if len(result.Prices) != 24 {
		t.Errorf("expected 24 prices, got %d", len(result.Prices))
	}
}

func TestAllowGo_WarmedFromSQLite(t *testing.T) {
	sqlCache, err := store.NewSQLiteCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlCache.Close()

	date := time.Now().UTC().Truncate(24 * time.Hour)
	prices := make24Prices(date, "synctacles", "live")
	_ = sqlCache.PutWithTier("NL", prices, 1)

	fm := NewFallbackManager(nil, sqlCache)
	result, _ := fm.Fetch(context.Background(), "NL", date)

	if !result.AllowGo() {
		t.Error("AllowGo should return true for SQLite-warmed live data (tier 1)")
	}
}

func TestFetch_SkipsIncompleteSQLite(t *testing.T) {
	sqlCache, err := store.NewSQLiteCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlCache.Close()

	date := time.Now().UTC().Truncate(24 * time.Hour)

	// Store only 12 prices (incomplete — need 24 for warming)
	prices := make24Prices(date, "synctacles", "live")[:12]
	_ = sqlCache.PutWithTier("NL", prices, 1)

	// No live sources → will fail since SQLite warming requires 24 prices
	fm := NewFallbackManager(nil, sqlCache)
	_, err = fm.Fetch(context.Background(), "NL", date)

	// Should return the 12 prices from Tier 4 fallback (not warm)
	// because warming requires >= 24 prices
	if err != nil {
		t.Fatal("should fall through to Tier 4 cache, not error")
	}
}

func TestFetch_SkipsLegacyTierZero(t *testing.T) {
	sqlCache, err := store.NewSQLiteCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlCache.Close()

	date := time.Now().UTC().Truncate(24 * time.Hour)

	// Store 24 prices with tier 0 (legacy — no provenance info)
	prices := make24Prices(date, "unknown", "cached")
	_ = sqlCache.PutWithTier("NL", prices, 0)

	fm := NewFallbackManager(nil, sqlCache)
	result, err := fm.Fetch(context.Background(), "NL", date)

	if err != nil {
		t.Fatal("should fall through to Tier 4 cache, not error")
	}
	// Should be Tier 4 (regular cache), not warmed
	if result.Tier != 4 {
		t.Errorf("expected Tier 4 (no warming for legacy data), got Tier %d", result.Tier)
	}
}

func TestActiveInfo_ShowsDiskCache(t *testing.T) {
	sqlCache, err := store.NewSQLiteCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlCache.Close()

	date := time.Now().UTC().Truncate(24 * time.Hour)
	prices := make24Prices(date, "synctacles", "live")
	_ = sqlCache.PutWithTier("NL", prices, 1)

	fm := NewFallbackManager(nil, sqlCache)
	_, _ = fm.Fetch(context.Background(), "NL", date)

	info := fm.ActiveInfo("NL", date)
	if info == nil {
		t.Fatal("ActiveInfo should not be nil after warming from SQLite")
	}
	if !info.FromDiskCache {
		t.Error("FromDiskCache should be true for SQLite-warmed data")
	}
	if info.FromMemCache {
		t.Error("FromMemCache should be false for SQLite-warmed data")
	}
	if info.Source != "synctacles" {
		t.Errorf("expected source synctacles, got %s", info.Source)
	}
}

// ============================================================================
// UpstreamSource propagation tests
// ============================================================================

func TestUpstreamSource_SurvivedDiskCache(t *testing.T) {
	sqlCache, err := store.NewSQLiteCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlCache.Close()

	date := time.Now().UTC().Truncate(24 * time.Hour)
	prices := make24Prices(date, "synctacles", "live")

	// Store with upstream source
	if err := sqlCache.PutWithTier("NL", prices, 1, "Energy-Charts"); err != nil {
		t.Fatal(err)
	}

	// Simulate reboot: new FallbackManager, no live sources
	fm := NewFallbackManager(nil, sqlCache)
	result, err := fm.Fetch(context.Background(), "NL", date)
	if err != nil {
		t.Fatalf("Fetch should succeed from disk cache: %v", err)
	}

	if result.UpstreamSource != "Energy-Charts" {
		t.Errorf("expected upstream 'Energy-Charts', got %q", result.UpstreamSource)
	}
}

func TestActiveInfo_IncludesUpstream(t *testing.T) {
	sqlCache, err := store.NewSQLiteCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlCache.Close()

	date := time.Now().UTC().Truncate(24 * time.Hour)
	prices := make24Prices(date, "synctacles", "live")
	_ = sqlCache.PutWithTier("NL", prices, 1, "ENTSO-E")

	fm := NewFallbackManager(nil, sqlCache)
	_, _ = fm.Fetch(context.Background(), "NL", date)

	info := fm.ActiveInfo("NL", date)
	if info == nil {
		t.Fatal("ActiveInfo should not be nil")
	}
	if info.UpstreamSource != "ENTSO-E" {
		t.Errorf("expected upstream 'ENTSO-E', got %q", info.UpstreamSource)
	}
}

func TestUpstreamSource_EmptyForLocalEC(t *testing.T) {
	sqlCache, err := store.NewSQLiteCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlCache.Close()

	date := time.Now().UTC().Truncate(24 * time.Hour)
	prices := make24Prices(date, "energycharts", "live")

	// Store without upstream (local EC is the source itself)
	_ = sqlCache.PutWithTier("NL", prices, 2)

	fm := NewFallbackManager(nil, sqlCache)
	result, err := fm.Fetch(context.Background(), "NL", date)
	if err != nil {
		t.Fatalf("Fetch should succeed: %v", err)
	}

	if result.UpstreamSource != "" {
		t.Errorf("expected empty upstream for local EC, got %q", result.UpstreamSource)
	}
}

func TestUpstreamSource_PropagatedToMemCache(t *testing.T) {
	sqlCache, err := store.NewSQLiteCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlCache.Close()

	date := time.Now().UTC().Truncate(24 * time.Hour)
	prices := make24Prices(date, "synctacles", "live")
	_ = sqlCache.PutWithTier("NL", prices, 1, "Energy-Charts")

	fm := NewFallbackManager(nil, sqlCache)

	// First fetch: warms from disk
	r1, _ := fm.Fetch(context.Background(), "NL", date)
	if r1.UpstreamSource != "Energy-Charts" {
		t.Fatalf("first fetch: expected upstream 'Energy-Charts', got %q", r1.UpstreamSource)
	}

	// Second fetch: should come from mem cache with upstream preserved
	r2, _ := fm.Fetch(context.Background(), "NL", date)
	if r2.UpstreamSource != "Energy-Charts" {
		t.Errorf("second fetch (mem cache): expected upstream 'Energy-Charts', got %q", r2.UpstreamSource)
	}
}
