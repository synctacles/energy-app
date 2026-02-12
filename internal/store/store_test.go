package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synctacles/energy-go/internal/models"
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

	// Retrieve
	got, err := cache.Get("NL", date)
	require.NoError(t, err)
	assert.Len(t, got, 3)
	assert.InDelta(t, 0.20, got[0].PriceEUR, 0.001)
	assert.InDelta(t, 0.25, got[1].PriceEUR, 0.001)
	assert.InDelta(t, 0.15, got[2].PriceEUR, 0.001)
	assert.Equal(t, "cached", got[0].Quality) // Quality becomes "cached"
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

	cache.Put("NL", []models.HourlyPrice{
		{Timestamp: date, PriceEUR: 0.20, Unit: models.UnitKWh, Source: "test", Zone: "NL"},
	})
	cache.Put("DE-LU", []models.HourlyPrice{
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

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	cache.Put("NL", []models.HourlyPrice{
		{Timestamp: date, PriceEUR: 0.20, Unit: models.UnitKWh, Source: "test", Zone: "NL"},
	})

	// Wait to ensure fetched_at is in the past (RFC3339 has second precision)
	time.Sleep(1100 * time.Millisecond)

	// Cleanup with 0 duration = delete everything older than now
	deleted, err := cache.Cleanup(0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify empty
	got, _ := cache.Get("NL", date)
	assert.Empty(t, got)
}
