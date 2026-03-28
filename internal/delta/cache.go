package delta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/synctacles/energy-app/pkg/platform"
)

// diskCache is the on-disk format for price persistence.
type diskCache struct {
	Zone      string             `json:"zone"`
	Supplier  string             `json:"supplier"`
	FetchedAt string             `json:"fetched_at"`
	Prices    map[string]float64 `json:"prices"` // key: "2006-01-02T15" → consumer EUR/kWh
}

// Cache holds per-hour consumer prices fetched from the energy-data Worker.
// Thread-safe for concurrent reads and writes. Persists to disk for offline use.
type Cache struct {
	mu        sync.RWMutex
	prices    map[string]float64 // key: "2006-01-02T15" → consumer EUR/kWh
	zone      string
	supplier  string
	fetchedAt time.Time
	diskPath  string
}

// NewCache creates a new price cache. If dataPath is non-empty, loads cached
// prices from disk (survives restarts).
func NewCache(dataPath ...string) *Cache {
	c := &Cache{prices: make(map[string]float64)}
	if len(dataPath) > 0 && dataPath[0] != "" {
		c.diskPath = filepath.Join(dataPath[0], "price_cache.json")
		c.loadFromDisk()
		// Migration: try loading old delta_cache.json if price_cache.json doesn't exist
		if len(c.prices) == 0 {
			oldPath := filepath.Join(dataPath[0], "delta_cache.json")
			if _, err := os.Stat(oldPath); err == nil {
				slog.Info("price: migrating from delta_cache.json (will be ignored)")
			}
		}
	}
	return c
}

// Get returns the consumer price for a specific hour, or (0, false) if not available.
func (c *Cache) Get(t time.Time) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := t.UTC().Format("2006-01-02T15")
	p, ok := c.prices[key]
	return p, ok
}

// IsStale returns true if the cache hasn't been updated in over 26 hours.
func (c *Cache) IsStale() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fetchedAt.IsZero() || time.Since(c.fetchedAt) > 26*time.Hour
}

// Len returns the number of cached price entries.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.prices)
}

// Fetch retrieves supplier prices from the energy-data Worker and updates the cache.
func (c *Cache) Fetch(ctx context.Context, zone, supplier string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v1/energy/supplier-prices?zone=%s&supplier=%s",
		platform.EnergyDataBaseURL, zone, supplier)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "SynctaclesEnergy/prices")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("supplier-prices returned %d", resp.StatusCode)
	}

	var result struct {
		Available bool `json:"available"`
		Prices    []struct {
			TS          string  `json:"ts"`
			ConsumerKWh float64 `json:"consumer_kwh"`
		} `json:"prices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10*1024*1024)).Decode(&result); err != nil {
		return err
	}

	if !result.Available || len(result.Prices) == 0 {
		slog.Debug("price: no prices available from Worker", "zone", zone, "supplier", supplier)
		return nil
	}

	newPrices := make(map[string]float64, len(result.Prices))
	for _, p := range result.Prices {
		t, err := time.Parse(time.RFC3339, p.TS)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05Z", p.TS)
			if err != nil {
				continue
			}
		}
		key := t.UTC().Format("2006-01-02T15")
		newPrices[key] = p.ConsumerKWh
	}

	c.mu.Lock()
	c.prices = newPrices
	c.zone = zone
	c.supplier = supplier
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	c.saveToDisk()

	slog.Info("price: cache updated", "zone", zone, "supplier", supplier, "hours", len(newPrices))
	return nil
}

// RunFetcher periodically fetches prices from the Worker.
// Runs once at startup (after stabilization), then at each hour boundary
// + 30s, plus a regular 15-minute background refresh.
func (c *Cache) RunFetcher(ctx context.Context, zone, supplier string) {
	// Wait for app stabilization
	select {
	case <-ctx.Done():
		return
	case <-time.After(2 * time.Minute):
	}

	if err := c.Fetch(ctx, zone, supplier); err != nil {
		slog.Warn("price: initial fetch failed, using disk cache", "error", err)
	}

	// Hour-boundary refresh at HH:00:30
	nextHourFetch := time.Now().Truncate(time.Hour).Add(time.Hour).Add(30 * time.Second)
	hourTimer := time.NewTimer(time.Until(nextHourFetch))
	defer hourTimer.Stop()

	// Regular 15-minute background refresh
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hourTimer.C:
			if err := c.Fetch(ctx, zone, supplier); err != nil {
				slog.Warn("price: hour-boundary fetch failed", "error", err)
			}
			nextHourFetch = time.Now().Truncate(time.Hour).Add(time.Hour).Add(30 * time.Second)
			hourTimer.Reset(time.Until(nextHourFetch))
		case <-ticker.C:
			if err := c.Fetch(ctx, zone, supplier); err != nil {
				slog.Warn("price: fetch failed, using cached prices", "error", err)
			}
		}
	}
}

// saveToDisk persists the current prices to a JSON file.
func (c *Cache) saveToDisk() {
	if c.diskPath == "" {
		return
	}
	c.mu.RLock()
	dc := diskCache{
		Zone:      c.zone,
		Supplier:  c.supplier,
		FetchedAt: c.fetchedAt.UTC().Format(time.RFC3339),
		Prices:    c.prices,
	}
	c.mu.RUnlock()

	data, err := json.MarshalIndent(dc, "", "  ")
	if err != nil {
		slog.Warn("price: failed to marshal cache", "error", err)
		return
	}
	if err := os.WriteFile(c.diskPath, data, 0644); err != nil {
		slog.Warn("price: failed to save cache to disk", "error", err)
	}
}

// loadFromDisk restores prices from the JSON file.
func (c *Cache) loadFromDisk() {
	if c.diskPath == "" {
		return
	}
	data, err := os.ReadFile(c.diskPath)
	if err != nil {
		return // no cache yet — first run
	}
	var dc diskCache
	if err := json.Unmarshal(data, &dc); err != nil {
		slog.Warn("price: failed to parse disk cache", "error", err)
		return
	}

	// Parse fetchedAt
	fetchedAt, err := time.Parse(time.RFC3339, dc.FetchedAt)
	if err != nil {
		return
	}

	// Don't load caches older than 72 hours — data too stale
	if time.Since(fetchedAt) > 72*time.Hour {
		slog.Info("price: disk cache too old, discarding", "age", time.Since(fetchedAt).Round(time.Hour))
		return
	}

	c.mu.Lock()
	c.prices = dc.Prices
	c.zone = dc.Zone
	c.supplier = dc.Supplier
	c.fetchedAt = fetchedAt
	c.mu.Unlock()

	slog.Info("price: loaded from disk cache", "zone", dc.Zone, "supplier", dc.Supplier,
		"hours", len(dc.Prices), "age", time.Since(fetchedAt).Round(time.Minute))
}
