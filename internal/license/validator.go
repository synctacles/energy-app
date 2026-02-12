// Package license validates Energy addon licenses against api.synctacles.com.
package license

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

const (
	defaultBaseURL   = "https://api.synctacles.com"
	validateInterval = 30 * 24 * time.Hour // Re-validate monthly
	gracePeriod      = 90 * 24 * time.Hour // Offline grace period
)

// proTiers are tiers that grant Pro features.
var proTiers = map[string]bool{
	"beta":      true,
	"paid":      true,
	"unlimited": true,
}

// statsResponse matches the GET /auth/stats response from the Energy API.
type statsResponse struct {
	UserID         string `json:"user_id"`
	Email          string `json:"email"`
	Tier           string `json:"tier"`
	RateLimitDaily int    `json:"rate_limit_daily"`
	UsageToday     int    `json:"usage_today"`
	RemainingToday int    `json:"remaining_today"`
}

// cachedResult is persisted to disk between addon restarts.
type cachedResult struct {
	Tier        string    `json:"tier"`
	IsPro       bool      `json:"is_pro"`
	ValidatedAt time.Time `json:"validated_at"`
	Email       string    `json:"email"`
}

// Validator checks license status against api.synctacles.com.
type Validator struct {
	apiKey    string
	baseURL   string
	cachePath string
	client    *http.Client

	mu     sync.RWMutex
	cached *cachedResult
}

// NewValidator creates a license validator.
// configPath is the HA config directory (e.g. /config) where the cache file is stored.
func NewValidator(apiKey, configPath string) *Validator {
	return &Validator{
		apiKey:    apiKey,
		baseURL:   defaultBaseURL,
		cachePath: filepath.Join(configPath, ".synctacles_license.json"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IsPro returns true if the current license grants Pro features.
// Safe for concurrent use.
func (v *Validator) IsPro() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.cached == nil {
		return false
	}

	// Check grace period — if last validation is too old, deny
	if time.Since(v.cached.ValidatedAt) > gracePeriod {
		slog.Warn("license grace period expired", "last_validated", v.cached.ValidatedAt)
		return false
	}

	return v.cached.IsPro
}

// Tier returns the current license tier (empty string if unknown).
func (v *Validator) Tier() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.cached == nil {
		return ""
	}
	return v.cached.Tier
}

// ValidateOnce performs a single validation check.
// Call this at startup and periodically from a goroutine.
func (v *Validator) ValidateOnce(ctx context.Context) error {
	if v.apiKey == "" {
		return nil // No key configured, stay on free tier
	}

	// Load cached result from disk
	v.loadCache()

	// Skip if recently validated
	v.mu.RLock()
	if v.cached != nil && time.Since(v.cached.ValidatedAt) < validateInterval {
		slog.Debug("license cache still valid", "tier", v.cached.Tier, "validated_at", v.cached.ValidatedAt)
		v.mu.RUnlock()
		return nil
	}
	v.mu.RUnlock()

	// Call the API
	stats, err := v.fetchStats(ctx)
	if err != nil {
		// If we have a cached result, keep using it (offline grace)
		v.mu.RLock()
		hasCached := v.cached != nil
		v.mu.RUnlock()
		if hasCached {
			slog.Warn("license validation failed, using cached result", "error", err)
			return nil
		}
		return fmt.Errorf("license validation: %w", err)
	}

	// Update cached result
	result := &cachedResult{
		Tier:        stats.Tier,
		IsPro:       proTiers[stats.Tier],
		ValidatedAt: time.Now().UTC(),
		Email:       stats.Email,
	}

	v.mu.Lock()
	v.cached = result
	v.mu.Unlock()

	v.saveCache(result)

	slog.Info("license validated", "tier", stats.Tier, "is_pro", result.IsPro, "email", stats.Email)
	return nil
}

// RunBackground starts a goroutine that re-validates monthly.
func (v *Validator) RunBackground(ctx context.Context) {
	if v.apiKey == "" {
		return
	}

	go func() {
		ticker := time.NewTicker(validateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := v.ValidateOnce(ctx); err != nil {
					slog.Error("background license validation failed", "error", err)
				}
			}
		}
	}()
}

// fetchStats calls GET /auth/stats on the Energy API.
func (v *Validator) fetchStats(ctx context.Context) (*statsResponse, error) {
	url := v.baseURL + "/auth/stats"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", v.apiKey)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid API key (401)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	var stats statsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode stats response: %w", err)
	}

	return &stats, nil
}

// loadCache reads the cached license result from disk.
func (v *Validator) loadCache() {
	data, err := os.ReadFile(v.cachePath)
	if err != nil {
		return // No cache file, that's fine
	}

	var result cachedResult
	if err := json.Unmarshal(data, &result); err != nil {
		slog.Debug("invalid license cache file", "error", err)
		return
	}

	v.mu.Lock()
	v.cached = &result
	v.mu.Unlock()
}

// saveCache writes the cached license result to disk.
func (v *Validator) saveCache(result *cachedResult) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return
	}

	// Write atomically via temp file
	tmpPath := v.cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		slog.Debug("failed to write license cache", "error", err)
		return
	}
	os.Rename(tmpPath, v.cachePath)
}
