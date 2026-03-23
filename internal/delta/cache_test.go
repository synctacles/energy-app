package delta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestDeltaCache_GetSet(t *testing.T) {
	c := NewCache()

	ts := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)

	// Initially empty
	d, ok := c.Get(ts)
	if ok {
		t.Errorf("expected no delta, got %f", d)
	}

	// Manually set deltas (simulating a fetch)
	c.mu.Lock()
	c.deltas["2026-03-15T14"] = 0.0042
	c.deltas["2026-03-15T15"] = -0.0018
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	d, ok = c.Get(ts)
	if !ok {
		t.Fatal("expected delta to be available")
	}
	if d != 0.0042 {
		t.Errorf("expected 0.0042, got %f", d)
	}

	// Different hour
	d, ok = c.Get(ts.Add(time.Hour))
	if !ok || d != -0.0018 {
		t.Errorf("expected -0.0018, got %f (ok=%v)", d, ok)
	}

	// Non-existent hour
	_, ok = c.Get(ts.Add(10 * time.Hour))
	if ok {
		t.Error("expected no delta for missing hour")
	}
}

func TestDeltaCache_Len(t *testing.T) {
	c := NewCache()
	if c.Len() != 0 {
		t.Errorf("expected 0, got %d", c.Len())
	}

	c.mu.Lock()
	c.deltas["2026-03-15T14"] = 0.001
	c.deltas["2026-03-15T15"] = 0.002
	c.mu.Unlock()

	if c.Len() != 2 {
		t.Errorf("expected 2, got %d", c.Len())
	}
}

func TestDeltaCache_IsStale(t *testing.T) {
	c := NewCache()

	// Fresh cache: fetchedAt is zero → stale
	if !c.IsStale() {
		t.Error("empty cache should be stale")
	}

	// Recently fetched → not stale
	c.mu.Lock()
	c.fetchedAt = time.Now()
	c.mu.Unlock()
	if c.IsStale() {
		t.Error("recently fetched cache should not be stale")
	}

	// Old cache → stale
	c.mu.Lock()
	c.fetchedAt = time.Now().Add(-27 * time.Hour)
	c.mu.Unlock()
	if !c.IsStale() {
		t.Error("cache older than 26h should be stale")
	}
}

func TestDeltaCache_ConcurrentAccess(t *testing.T) {
	c := NewCache()

	// Pre-populate
	c.mu.Lock()
	for h := 0; h < 48; h++ {
		ts := time.Now().UTC().Add(time.Duration(h) * time.Hour)
		key := ts.Format("2006-01-02T15")
		c.deltas[key] = float64(h) * 0.001
	}
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	var wg sync.WaitGroup

	// 50 concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ts := time.Now().UTC().Add(time.Duration(idx%48) * time.Hour)
			c.Get(ts)
			c.Len()
			c.IsStale()
		}(i)
	}

	// 50 concurrent writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c.mu.Lock()
			key := time.Now().UTC().Add(time.Duration(idx) * time.Hour).Format("2006-01-02T15")
			c.deltas[key] = float64(idx) * 0.002
			c.fetchedAt = time.Now()
			c.mu.Unlock()
		}(i)
	}

	wg.Wait()
	// Race detector validates correctness — reaching here means no races.
}

func TestDeltaCache_DiskPersistence(t *testing.T) {
	dir := t.TempDir()

	// Create cache, populate, and save to disk
	c1 := NewCache(dir)
	c1.mu.Lock()
	c1.zone = "NL"
	c1.supplier = "zonneplan"
	c1.fetchedAt = time.Now().UTC().Truncate(time.Second) // truncate for comparison
	c1.deltas = map[string]float64{
		"2026-03-15T14": 0.0042,
		"2026-03-15T15": -0.0018,
		"2026-03-15T16": 0.0,
	}
	c1.mu.Unlock()
	c1.saveToDisk()

	// Verify file exists
	diskPath := filepath.Join(dir, "delta_cache.json")
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Fatal("disk cache file not created")
	}

	// Load into a new cache
	c2 := NewCache(dir)
	if c2.Len() != 3 {
		t.Errorf("expected 3 deltas loaded from disk, got %d", c2.Len())
	}

	ts := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)
	d, ok := c2.Get(ts)
	if !ok || d != 0.0042 {
		t.Errorf("expected 0.0042 from disk, got %f (ok=%v)", d, ok)
	}

	// Verify negative delta persisted
	d, ok = c2.Get(ts.Add(time.Hour))
	if !ok || d != -0.0018 {
		t.Errorf("expected -0.0018 from disk, got %f (ok=%v)", d, ok)
	}
}

func TestDeltaCache_DiskExpiry_72h(t *testing.T) {
	dir := t.TempDir()

	// Write a disk cache file with an old fetchedAt
	dc := diskCache{
		Zone:      "NL",
		Supplier:  "zonneplan",
		FetchedAt: time.Now().UTC().Add(-73 * time.Hour).Format(time.RFC3339),
		Deltas:    map[string]float64{"2026-03-10T14": 0.001},
	}
	data, _ := json.MarshalIndent(dc, "", "  ")
	os.WriteFile(filepath.Join(dir, "delta_cache.json"), data, 0644)

	// Load — should discard because >72h old
	c := NewCache(dir)
	if c.Len() != 0 {
		t.Errorf("expected 0 deltas (expired), got %d", c.Len())
	}
}

func TestDeltaCache_DiskCorrupted(t *testing.T) {
	dir := t.TempDir()

	// Write invalid JSON
	os.WriteFile(filepath.Join(dir, "delta_cache.json"), []byte("{invalid json}"), 0644)

	// Should not panic, just start empty
	c := NewCache(dir)
	if c.Len() != 0 {
		t.Errorf("expected 0 deltas from corrupted file, got %d", c.Len())
	}
}
