package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synctacles/energy-app/pkg/models"
)

func TestSQLiteCache_PutGet(t *testing.T) {
	cache, err := NewSQLiteCache(t.TempDir())
	require.NoError(t, err)
	defer cache.Close()

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	prices := []models.HourlyPrice{
		{Timestamp: date, PriceEUR: 0.20, Unit: models.UnitKWh, Source: "test", Quality: "live", Zone: "NL"},
		{Timestamp: date.Add(time.Hour), PriceEUR: 0.25, Unit: models.UnitKWh, Source: "test", Quality: "live", Zone: "NL"},
		{Timestamp: date.Add(2 * time.Hour), PriceEUR: 0.15, Unit: models.UnitKWh, Source: "test", Quality: "live", Zone: "NL"},
	}

	err = cache.Put("NL", prices)
	require.NoError(t, err)

	// Retrieve — quality is now preserved from original data
	got, err := cache.Get("NL", date)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.InDelta(t, 0.20, got[0].PriceEUR, 0.001)
	assert.InDelta(t, 0.25, got[1].PriceEUR, 0.001)
	assert.InDelta(t, 0.15, got[2].PriceEUR, 0.001)
	assert.Equal(t, "live", got[0].Quality) // Quality preserved from source
	assert.Equal(t, "test", got[0].Source)
}

func TestSQLiteCache_Upsert(t *testing.T) {
	cache, err := NewSQLiteCache(t.TempDir())
	require.NoError(t, err)
	defer cache.Close()

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)

	// Insert
	err = cache.Put("NL", []models.HourlyPrice{
		{Timestamp: date, PriceEUR: 0.20, Unit: models.UnitKWh, Source: "old", Zone: "NL"},
	})
	require.NoError(t, err)

	// Update same timestamp
	err = cache.Put("NL", []models.HourlyPrice{
		{Timestamp: date, PriceEUR: 0.30, Unit: models.UnitKWh, Source: "new", Zone: "NL"},
	})
	require.NoError(t, err)

	got, err := cache.Get("NL", date)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.InDelta(t, 0.30, got[0].PriceEUR, 0.001)
}

func TestSQLiteCache_DifferentZones(t *testing.T) {
	cache, err := NewSQLiteCache(t.TempDir())
	require.NoError(t, err)
	defer cache.Close()

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)

	_ = cache.Put("NL", []models.HourlyPrice{
		{Timestamp: date, PriceEUR: 0.20, Unit: models.UnitKWh, Source: "test", Zone: "NL"},
	})
	_ = cache.Put("DE-LU", []models.HourlyPrice{
		{Timestamp: date, PriceEUR: 0.10, Unit: models.UnitMWh, Source: "test", Zone: "DE-LU"},
	})

	nl, _ := cache.Get("NL", date)
	de, _ := cache.Get("DE-LU", date)

	assert.Len(t, nl, 1)
	assert.Len(t, de, 1)
	assert.InDelta(t, 0.20, nl[0].PriceEUR, 0.001)
	assert.InDelta(t, 0.10, de[0].PriceEUR, 0.001)
}

func TestSQLiteCache_EmptyGet(t *testing.T) {
	cache, err := NewSQLiteCache(t.TempDir())
	require.NoError(t, err)
	defer cache.Close()

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	got, err := cache.Get("XX", date)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestSQLiteCache_Cleanup(t *testing.T) {
	cache, err := NewSQLiteCache(t.TempDir())
	require.NoError(t, err)
	defer cache.Close()

	// Insert a price with a timestamp 3 days in the past
	oldDate := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Hour)
	_ = cache.Put("NL", []models.HourlyPrice{
		{Timestamp: oldDate, PriceEUR: 0.20, Unit: models.UnitKWh, Source: "test", Zone: "NL"},
	})

	// Cleanup with 48h — should delete the 3-day-old price
	deleted, err := cache.Cleanup(48 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify empty
	got, _ := cache.Get("NL", oldDate)
	assert.Empty(t, got)
}

func TestSQLiteCache_CleanupKeepsRecentTimestamps(t *testing.T) {
	cache, err := NewSQLiteCache(t.TempDir())
	require.NoError(t, err)
	defer cache.Close()

	// Insert a price with today's timestamp
	todayHour := time.Now().UTC().Truncate(time.Hour)
	_ = cache.Put("NL", []models.HourlyPrice{
		{Timestamp: todayHour, PriceEUR: 0.20, Unit: models.UnitKWh, Source: "test", Zone: "NL"},
	})

	// Cleanup with 48h — should NOT delete today's price
	deleted, err := cache.Cleanup(48 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted)

	got, _ := cache.Get("NL", todayHour)
	assert.Len(t, got, 1)
}

func TestSQLiteCache_PutWithTier_GetWithMeta(t *testing.T) {
	cache, err := NewSQLiteCache(t.TempDir())
	require.NoError(t, err)
	defer cache.Close()

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	prices := make([]models.HourlyPrice, 24)
	for i := range prices {
		prices[i] = models.HourlyPrice{
			Timestamp: date.Add(time.Duration(i) * time.Hour),
			PriceEUR:  0.20 + float64(i)*0.01,
			Unit:      models.UnitKWh,
			Source:    "enever",
			Quality:   "live",
			Zone:      "NL",
		}
	}

	// Store with tier 1 (Enever is primary)
	err = cache.PutWithTier("NL", prices, 1)
	require.NoError(t, err)

	// Retrieve with metadata
	entry, err := cache.GetWithMeta("NL", date)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Len(t, entry.Prices, 24)
	assert.Equal(t, 1, entry.OriginalTier)
	assert.False(t, entry.FetchedAt.IsZero())
	assert.Equal(t, "live", entry.Prices[0].Quality)
	assert.Equal(t, "enever", entry.Prices[0].Source)
}

func TestSQLiteCache_GetWithMeta_Empty(t *testing.T) {
	cache, err := NewSQLiteCache(t.TempDir())
	require.NoError(t, err)
	defer cache.Close()

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	entry, err := cache.GetWithMeta("XX", date)
	require.NoError(t, err)
	assert.Nil(t, entry)
}

func TestSQLiteCache_LegacyRowsGetTierZero(t *testing.T) {
	cache, err := NewSQLiteCache(t.TempDir())
	require.NoError(t, err)
	defer cache.Close()

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)

	// Put with tier 0 (simulates legacy data without tier info)
	err = cache.Put("NL", []models.HourlyPrice{
		{Timestamp: date, PriceEUR: 0.20, Unit: models.UnitKWh, Source: "test", Quality: "live", Zone: "NL"},
	})
	require.NoError(t, err)

	entry, err := cache.GetWithMeta("NL", date)
	require.NoError(t, err)
	require.NotNil(t, entry)

	// Legacy rows have tier 0
	assert.Equal(t, 0, entry.OriginalTier)
}
