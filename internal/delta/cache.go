package delta

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// diskCache is the on-disk format for delta persistence.
type diskCache struct {
	Zone      string             `json:"zone"`
	Supplier  string             `json:"supplier"`
	FetchedAt string             `json:"fetched_at"`
	Deltas    map[string]float64 `json:"deltas"` // key: "2006-01-02T15"
}

// Cache holds per-hour supplier deltas fetched from the energy-data Worker.
// Thread-safe for concurrent reads and writes. Persists to disk for offline use.
type Cache struct {
	mu        sync.RWMutex
	deltas    map[string]float64 // key: "2006-01-02T15" → delta EUR/kWh
	zone      string
	supplier  string
	fetchedAt time.Time
	diskPath  string
}

// NewCache creates a new delta cache. If dataPath is non-empty, loads cached
// deltas from disk (survives restarts).
func NewCache(dataPath ...string) *Cache {
	c := &Cache{deltas: make(map[string]float64)}
	if len(dataPath) > 0 && dataPath[0] != "" {
		c.diskPath = filepath.Join(dataPath[0], "delta_cache.json")
		c.loadFromDisk()
	}
	return c
}

// Get returns the delta for a specific hour, or (0, false) if not available.
func (c *Cache) Get(t time.Time) (float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := t.UTC().Format("2006-01-02T15")
	d, ok := c.deltas[key]
	return d, ok
}

// IsStale returns true if the cache hasn't been updated in over 26 hours.
func (c *Cache) IsStale() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fetchedAt.IsZero() || time.Since(c.fetchedAt) > 26*time.Hour
}

// Len returns the number of cached delta entries.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.deltas)
}

// Fetch retrieves supplier deltas from the energy-data Worker and updates the cache.
func (c *Cache) Fetch(ctx context.Context, zone, supplier string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v1/energy/supplier-deltas?zone=%s&supplier=%s",
		energyDataBaseURL, zone, supplier)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "SynctaclesEnergy/delta")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("supplier-deltas returned %d", resp.StatusCode)
	}

	var result struct {
		Available bool `json:"available"`
		Deltas    []struct {
			TS    string  `json:"ts"`
			Delta float64 `json:"delta"`
		} `json:"deltas"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.Available || len(result.Deltas) == 0 {
		slog.Debug("delta: no deltas available from Worker", "zone", zone, "supplier", supplier)
		return nil
	}

	newDeltas := make(map[string]float64, len(result.Deltas))
	for _, d := range result.Deltas {
		t, err := time.Parse(time.RFC3339, d.TS)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05Z", d.TS)
			if err != nil {
				continue
			}
		}
		key := t.UTC().Format("2006-01-02T15")
		newDeltas[key] = d.Delta
	}

	c.mu.Lock()
	c.deltas = newDeltas
	c.zone = zone
	c.supplier = supplier
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	c.saveToDisk()

	slog.Info("delta: cache updated", "zone", zone, "supplier", supplier, "hours", len(newDeltas))
	return nil
}

// RunFetcher periodically fetches deltas from the Worker.
// Runs once at startup (after stabilization), then every hour.
func (c *Cache) RunFetcher(ctx context.Context, zone, supplier string) {
	// Wait for app stabilization
	select {
	case <-ctx.Done():
		return
	case <-time.After(2 * time.Minute):
	}

	if err := c.Fetch(ctx, zone, supplier); err != nil {
		slog.Warn("delta: initial fetch failed, using disk cache", "error", err)
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Fetch(ctx, zone, supplier); err != nil {
				slog.Warn("delta: fetch failed, using cached deltas", "error", err)
			}
		}
	}
}

// saveToDisk persists the current deltas to a JSON file.
func (c *Cache) saveToDisk() {
	if c.diskPath == "" {
		return
	}
	c.mu.RLock()
	dc := diskCache{
		Zone:      c.zone,
		Supplier:  c.supplier,
		FetchedAt: c.fetchedAt.UTC().Format(time.RFC3339),
		Deltas:    c.deltas,
	}
	c.mu.RUnlock()

	data, err := json.MarshalIndent(dc, "", "  ")
	if err != nil {
		slog.Warn("delta: failed to marshal cache", "error", err)
		return
	}
	if err := os.WriteFile(c.diskPath, data, 0644); err != nil {
		slog.Warn("delta: failed to save cache to disk", "error", err)
	}
}

// loadFromDisk restores deltas from the JSON file.
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
		slog.Warn("delta: failed to parse disk cache", "error", err)
		return
	}

	// Parse fetchedAt
	fetchedAt, err := time.Parse(time.RFC3339, dc.FetchedAt)
	if err != nil {
		return
	}

	// Don't load caches older than 72 hours — data too stale
	if time.Since(fetchedAt) > 72*time.Hour {
		slog.Info("delta: disk cache too old, discarding", "age", time.Since(fetchedAt).Round(time.Hour))
		return
	}

	c.mu.Lock()
	c.deltas = dc.Deltas
	c.zone = dc.Zone
	c.supplier = dc.Supplier
	c.fetchedAt = fetchedAt
	c.mu.Unlock()

	slog.Info("delta: loaded from disk cache", "zone", dc.Zone, "supplier", dc.Supplier,
		"hours", len(dc.Deltas), "age", time.Since(fetchedAt).Round(time.Minute))
}
