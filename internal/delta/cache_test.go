package delta

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestPriceCache_GetSet(t *testing.T) {
	c := NewCache()

	ts := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)

	// Initially empty
	p, ok := c.Get(ts)
	if ok {
		t.Errorf("expected no price, got %f", p)
	}

	// Manually set prices (simulating a fetch)
	c.mu.Lock()
	c.prices["2026-03-15T14"] = 0.2758
	c.prices["2026-03-15T15"] = 0.3077
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	p, ok = c.Get(ts)
	if !ok {
		t.Fatal("expected price to be available")
	}
	if p != 0.2758 {
		t.Errorf("expected 0.2758, got %f", p)
	}

	// Different hour
	p, ok = c.Get(ts.Add(time.Hour))
	if !ok || p != 0.3077 {
		t.Errorf("expected 0.3077, got %f (ok=%v)", p, ok)
	}

	// Non-existent hour
	_, ok = c.Get(ts.Add(10 * time.Hour))
	if ok {
		t.Error("expected no price for missing hour")
	}
}

func TestPriceCache_Len(t *testing.T) {
	c := NewCache()
	if c.Len() != 0 {
		t.Errorf("expected 0, got %d", c.Len())
	}

	c.mu.Lock()
	c.prices["2026-03-15T14"] = 0.25
	c.prices["2026-03-15T15"] = 0.30
	c.mu.Unlock()

	if c.Len() != 2 {
		t.Errorf("expected 2, got %d", c.Len())
	}
}

func TestPriceCache_IsStale(t *testing.T) {
	c := NewCache()

	if !c.IsStale() {
		t.Error("empty cache should be stale")
	}

	c.mu.Lock()
	c.fetchedAt = time.Now()
	c.mu.Unlock()
	if c.IsStale() {
		t.Error("recently fetched cache should not be stale")
	}

	c.mu.Lock()
	c.fetchedAt = time.Now().Add(-27 * time.Hour)
	c.mu.Unlock()
	if !c.IsStale() {
		t.Error("cache older than 26h should be stale")
	}
}

func TestPriceCache_ConcurrentAccess(t *testing.T) {
	c := NewCache()

	c.mu.Lock()
	for h := 0; h < 48; h++ {
		ts := time.Now().UTC().Add(time.Duration(h) * time.Hour)
		key := ts.Format("2006-01-02T15")
		c.prices[key] = float64(h) * 0.01
	}
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	var wg sync.WaitGroup

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

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c.mu.Lock()
			key := time.Now().UTC().Add(time.Duration(idx) * time.Hour).Format("2006-01-02T15")
			c.prices[key] = float64(idx) * 0.01
			c.fetchedAt = time.Now()
			c.mu.Unlock()
		}(i)
	}

	wg.Wait()
}

func TestPriceCache_DiskPersistence(t *testing.T) {
	dir := t.TempDir()

	c1 := NewCache(dir)
	c1.mu.Lock()
	c1.zone = "NL"
	c1.supplier = "zonneplan"
	c1.fetchedAt = time.Now().UTC().Truncate(time.Second)
	c1.prices = map[string]float64{
		"2026-03-15T14": 0.2758,
		"2026-03-15T15": 0.3077,
		"2026-03-15T16": 0.1341,
	}
	c1.mu.Unlock()
	c1.saveToDisk()

	diskPath := filepath.Join(dir, "price_cache.json")
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Fatal("disk cache file not created")
	}

	c2 := NewCache(dir)
	if c2.Len() != 3 {
		t.Errorf("expected 3 prices loaded from disk, got %d", c2.Len())
	}

	ts := time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC)
	p, ok := c2.Get(ts)
	if !ok || p != 0.2758 {
		t.Errorf("expected 0.2758 from disk, got %f (ok=%v)", p, ok)
	}
}

func TestPriceCache_DiskExpiry_72h(t *testing.T) {
	dir := t.TempDir()

	dc := diskCache{
		Zone:      "NL",
		Supplier:  "zonneplan",
		FetchedAt: time.Now().UTC().Add(-73 * time.Hour).Format(time.RFC3339),
		Prices:    map[string]float64{"2026-03-10T14": 0.25},
	}
	data, _ := json.MarshalIndent(dc, "", "  ")
	os.WriteFile(filepath.Join(dir, "price_cache.json"), data, 0644)

	c := NewCache(dir)
	if c.Len() != 0 {
		t.Errorf("expected 0 prices (expired), got %d", c.Len())
	}
}

func TestPriceCache_DiskCorrupted(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "price_cache.json"), []byte("{invalid json}"), 0644)

	c := NewCache(dir)
	if c.Len() != 0 {
		t.Errorf("expected 0 prices from corrupted file, got %d", c.Len())
	}
}
