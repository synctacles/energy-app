package license

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_PaidTier(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth/stats", r.URL.Path)
		assert.Equal(t, "test-api-key-123", r.Header.Get("X-API-Key"))

		json.NewEncoder(w).Encode(statsResponse{
			UserID: "u1", Email: "test@example.com", Tier: "paid",
			RateLimitDaily: 100000, UsageToday: 5, RemainingToday: 99995,
		})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	v := NewValidator("test-api-key-123", tmp)
	v.baseURL = srv.URL

	assert.False(t, v.IsPro()) // Not yet validated

	err := v.ValidateOnce(context.Background())
	require.NoError(t, err)

	assert.True(t, v.IsPro())
	assert.Equal(t, "paid", v.Tier())

	// Cache file should exist
	_, err = os.Stat(filepath.Join(tmp, ".synctacles_license.json"))
	assert.NoError(t, err)
}

func TestValidator_FreeTier(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(statsResponse{
			UserID: "u2", Email: "free@example.com", Tier: "free",
			RateLimitDaily: 1000, UsageToday: 0, RemainingToday: 1000,
		})
	}))
	defer srv.Close()

	v := NewValidator("free-key", t.TempDir())
	v.baseURL = srv.URL

	err := v.ValidateOnce(context.Background())
	require.NoError(t, err)

	assert.False(t, v.IsPro())
	assert.Equal(t, "free", v.Tier())
}

func TestValidator_BetaTier(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(statsResponse{
			UserID: "u3", Email: "beta@example.com", Tier: "beta",
			RateLimitDaily: 10000, UsageToday: 0, RemainingToday: 10000,
		})
	}))
	defer srv.Close()

	v := NewValidator("beta-key", t.TempDir())
	v.baseURL = srv.URL

	err := v.ValidateOnce(context.Background())
	require.NoError(t, err)

	assert.True(t, v.IsPro())
	assert.Equal(t, "beta", v.Tier())
}

func TestValidator_NoKey(t *testing.T) {
	v := NewValidator("", t.TempDir())

	err := v.ValidateOnce(context.Background())
	require.NoError(t, err) // No error, just stays free

	assert.False(t, v.IsPro())
	assert.Equal(t, "", v.Tier())
}

func TestValidator_InvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"detail":"Invalid API key"}`))
	}))
	defer srv.Close()

	v := NewValidator("bad-key", t.TempDir())
	v.baseURL = srv.URL

	err := v.ValidateOnce(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

func TestValidator_CachePersistence(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(statsResponse{
			UserID: "u1", Email: "test@example.com", Tier: "paid",
			RateLimitDaily: 100000, UsageToday: 0, RemainingToday: 100000,
		})
	}))
	defer srv.Close()

	tmp := t.TempDir()

	// First validation
	v1 := NewValidator("key1", tmp)
	v1.baseURL = srv.URL
	err := v1.ValidateOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, calls)

	// Second validator on same cache path — should use cache
	v2 := NewValidator("key1", tmp)
	v2.baseURL = srv.URL
	err = v2.ValidateOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, calls) // No new API call
	assert.True(t, v2.IsPro())
}

func TestValidator_GracePeriodExpired(t *testing.T) {
	tmp := t.TempDir()

	// Write an expired cache file (100 days old)
	expired := cachedResult{
		Tier:        "paid",
		IsPro:       true,
		ValidatedAt: time.Now().Add(-100 * 24 * time.Hour),
		Email:       "old@example.com",
	}
	data, _ := json.Marshal(expired)
	os.WriteFile(filepath.Join(tmp, ".synctacles_license.json"), data, 0600)

	// Server is down (no mock)
	v := NewValidator("key", tmp)
	v.baseURL = "http://127.0.0.1:1" // Connection refused

	// Load from cache but grace period expired
	_ = v.ValidateOnce(context.Background())
	assert.False(t, v.IsPro()) // Grace period expired → no Pro
}

func TestValidator_OfflineGrace(t *testing.T) {
	tmp := t.TempDir()

	// Write a recent cache file (10 days old — within grace period)
	recent := cachedResult{
		Tier:        "paid",
		IsPro:       true,
		ValidatedAt: time.Now().Add(-10 * 24 * time.Hour),
		Email:       "test@example.com",
	}
	data, _ := json.Marshal(recent)
	os.WriteFile(filepath.Join(tmp, ".synctacles_license.json"), data, 0600)

	// Server is down
	v := NewValidator("key", tmp)
	v.baseURL = "http://127.0.0.1:1" // Connection refused

	err := v.ValidateOnce(context.Background())
	require.NoError(t, err) // Should succeed using cache

	assert.True(t, v.IsPro()) // Within grace period → still Pro
}
