// Package engine — TaxProfileCache persists Worker-provided tax profiles to disk.
// Keyed per bidding zone so new zones work without app updates.
// The Worker is the single source of truth for tax data; this cache ensures
// offline normalization of Energy-Charts wholesale prices.
package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// WorkerTaxOverride holds a Worker-provided tax profile for a specific zone.
type WorkerTaxOverride struct {
	VATRate          float64 `json:"vat_rate"`
	EnergyTax        float64 `json:"energy_tax"`         // Current energy tax (already resolved for today)
	Surcharges       float64 `json:"surcharges"`
	NetworkTariffAvg float64 `json:"network_tariff_avg"`
	SupplierMarkup   float64 `json:"supplier_markup"`    // Calibrated EUR/kWh from Worker
	Version          string  `json:"version"`             // Worker's tax profile version string
	UpdatedAt        string  `json:"updated_at"`          // ISO 8601 timestamp
}

// TaxProfileCache manages on-disk persistence of Worker tax profiles.
// Thread-safe for concurrent reads and writes. Keyed by zone code.
type TaxProfileCache struct {
	mu       sync.RWMutex
	profiles map[string]*WorkerTaxOverride // key: zone code (e.g. "NL", "DE-LU", "NO1")
	path     string
}

// NewTaxProfileCache creates a cache backed by a JSON file in the data directory.
// Loads existing cache from disk on creation (empty on first boot).
func NewTaxProfileCache(dataPath string) *TaxProfileCache {
	c := &TaxProfileCache{
		profiles: make(map[string]*WorkerTaxOverride),
		path:     filepath.Join(dataPath, "tax_profile_cache.json"),
	}
	c.loadFromDisk()
	return c
}

// Get returns the cached tax profile for a zone, or nil if not cached or expired.
// Entries older than 90 days are treated as expired (forces Worker refresh).
func (c *TaxProfileCache) Get(zone string) *WorkerTaxOverride {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.profiles[zone]
	if !ok {
		return nil
	}
	// 90-day expiry — stale cache should not be used indefinitely
	if t, err := time.Parse(time.RFC3339, p.UpdatedAt); err == nil {
		if time.Since(t) > 90*24*time.Hour {
			return nil
		}
	}
	cp := *p
	return &cp
}

// Put updates the cache for a zone (version-based — only writes if version changed).
// Persists to disk atomically.
func (c *TaxProfileCache) Put(zone string, override *WorkerTaxOverride) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Version-based: skip if same version already cached
	if existing, ok := c.profiles[zone]; ok && existing.Version == override.Version {
		return
	}

	override.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	c.profiles[zone] = override
	c.saveToDisk()
}

// HasData returns true if the cache has at least one zone profile.
func (c *TaxProfileCache) HasData() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.profiles) > 0
}

func (c *TaxProfileCache) loadFromDisk() {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return // No cache file — first boot
	}
	var profiles map[string]*WorkerTaxOverride
	if json.Unmarshal(data, &profiles) == nil {
		c.profiles = profiles
	}
}

func (c *TaxProfileCache) saveToDisk() {
	data, err := json.MarshalIndent(c.profiles, "", "  ")
	if err != nil {
		return
	}
	tmpPath := c.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return
	}
	os.Rename(tmpPath, c.path)
}
