package delta

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Cache holds per-hour supplier deltas fetched from the energy-data Worker.
// Thread-safe for concurrent reads and writes.
type Cache struct {
	mu      sync.RWMutex
	deltas  map[string]float64 // key: "2006-01-02T15" → delta EUR/kWh
	zone    string
	supplier string
	fetchedAt time.Time
}

// NewCache creates a new delta cache.
func NewCache() *Cache {
	return &Cache{deltas: make(map[string]float64)}
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
		// Parse ISO timestamp and key by hour
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
		slog.Warn("delta: initial fetch failed", "error", err)
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Fetch(ctx, zone, supplier); err != nil {
				slog.Warn("delta: fetch failed", "error", err)
			}
		}
	}
}
