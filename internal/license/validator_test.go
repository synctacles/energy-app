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

func TestValidator_AlwaysPro(t *testing.T) {
	// All features are free — IsPro() always returns true regardless of state
	v := NewValidator("", t.TempDir())
	assert.True(t, v.IsPro())
	assert.False(t, v.IsTrial())
	assert.Equal(t, 0, v.TrialDaysLeft())
	assert.Equal(t, "pro", v.Tier())
}

func TestValidator_InitTrialNoOp(t *testing.T) {
	tmp := t.TempDir()
	v := NewValidator("", tmp)
	v.InitTrial()

	// InitTrial is a no-op, no install file created
	assert.True(t, v.IsPro())
	assert.False(t, v.IsTrial())
	assert.Equal(t, "pro", v.Tier())
}

func TestValidator_ValidateOnce_NoKey(t *testing.T) {
	v := NewValidator("", t.TempDir())
	err := v.ValidateOnce(context.Background())
	require.NoError(t, err)
	assert.True(t, v.IsPro())
}

func TestValidator_ValidateOnce_WithKey(t *testing.T) {
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

	err := v.ValidateOnce(context.Background())
	require.NoError(t, err)
	assert.True(t, v.IsPro())

	// Cache file should exist
	_, err = os.Stat(filepath.Join(tmp, ".synctacles_license.json"))
	assert.NoError(t, err)
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

	// IsPro still true even with invalid key (all features free)
	assert.True(t, v.IsPro())
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

	assert.True(t, v.IsPro())
}
