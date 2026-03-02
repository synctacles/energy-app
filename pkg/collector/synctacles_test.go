package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synctacles/energy-app/pkg/models"
)

func TestSynctaclesAPI_Metadata(t *testing.T) {
	s := &SynctaclesAPI{BaseURL: "https://energy.synctacles.com"}
	assert.Equal(t, "synctacles", s.Name())
	assert.False(t, s.RequiresKey())
	assert.Len(t, s.Zones(), 38)
	assert.Contains(t, s.Zones(), "NL")
	assert.Contains(t, s.Zones(), "DE-LU")
	assert.Contains(t, s.Zones(), "NO1")
}

func TestAggregateToHourly_PT15M(t *testing.T) {
	// Simulate 4 quarter-hourly entries for hour 10:00
	base := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)
	prices := []models.HourlyPrice{
		{Timestamp: base, PriceEUR: 0.30, WholesaleKWh: 0.08, Unit: models.UnitKWh, Source: "synctacles", Quality: "live", Zone: "NL", IsConsumer: true},
		{Timestamp: base.Add(15 * time.Minute), PriceEUR: 0.32, WholesaleKWh: 0.09, Unit: models.UnitKWh, Source: "synctacles", Quality: "live", Zone: "NL", IsConsumer: true},
		{Timestamp: base.Add(30 * time.Minute), PriceEUR: 0.34, WholesaleKWh: 0.10, Unit: models.UnitKWh, Source: "synctacles", Quality: "live", Zone: "NL", IsConsumer: true},
		{Timestamp: base.Add(45 * time.Minute), PriceEUR: 0.36, WholesaleKWh: 0.11, Unit: models.UnitKWh, Source: "synctacles", Quality: "live", Zone: "NL", IsConsumer: true},
		// Hour 11:00
		{Timestamp: base.Add(60 * time.Minute), PriceEUR: 0.40, WholesaleKWh: 0.12, Unit: models.UnitKWh, Source: "synctacles", Quality: "live", Zone: "NL", IsConsumer: true},
		{Timestamp: base.Add(75 * time.Minute), PriceEUR: 0.42, WholesaleKWh: 0.13, Unit: models.UnitKWh, Source: "synctacles", Quality: "live", Zone: "NL", IsConsumer: true},
		{Timestamp: base.Add(90 * time.Minute), PriceEUR: 0.44, WholesaleKWh: 0.14, Unit: models.UnitKWh, Source: "synctacles", Quality: "live", Zone: "NL", IsConsumer: true},
		{Timestamp: base.Add(105 * time.Minute), PriceEUR: 0.46, WholesaleKWh: 0.15, Unit: models.UnitKWh, Source: "synctacles", Quality: "live", Zone: "NL", IsConsumer: true},
	}

	result := aggregateToHourly(prices)
	require.Len(t, result, 2, "8 quarter-hourly entries should become 2 hourly")

	// Hour 10:00 — average of 0.30, 0.32, 0.34, 0.36 = 0.33
	assert.Equal(t, base, result[0].Timestamp)
	assert.InDelta(t, 0.33, result[0].PriceEUR, 0.001)
	assert.InDelta(t, 0.095, result[0].WholesaleKWh, 0.001)
	assert.True(t, result[0].IsConsumer)
	assert.Equal(t, "synctacles", result[0].Source)
	assert.Equal(t, "NL", result[0].Zone)
	assert.Equal(t, models.UnitKWh, result[0].Unit)

	// Hour 11:00 — average of 0.40, 0.42, 0.44, 0.46 = 0.43
	assert.Equal(t, base.Add(time.Hour), result[1].Timestamp)
	assert.InDelta(t, 0.43, result[1].PriceEUR, 0.001)
	assert.InDelta(t, 0.135, result[1].WholesaleKWh, 0.001)
	assert.True(t, result[1].IsConsumer)
}

func TestAggregateToHourly_PT30M(t *testing.T) {
	base := time.Date(2026, 3, 2, 14, 0, 0, 0, time.UTC)
	prices := []models.HourlyPrice{
		{Timestamp: base, PriceEUR: 50.0, Unit: models.UnitMWh, Source: "synctacles", Quality: "live", Zone: "GB", IsConsumer: false},
		{Timestamp: base.Add(30 * time.Minute), PriceEUR: 60.0, Unit: models.UnitMWh, Source: "synctacles", Quality: "live", Zone: "GB", IsConsumer: false},
	}

	result := aggregateToHourly(prices)
	require.Len(t, result, 1)
	assert.Equal(t, base, result[0].Timestamp)
	assert.InDelta(t, 55.0, result[0].PriceEUR, 0.001)
	assert.False(t, result[0].IsConsumer)
}

func TestAggregateToHourly_AlreadyHourly(t *testing.T) {
	base := time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC)
	prices := []models.HourlyPrice{
		{Timestamp: base, PriceEUR: 0.25, Unit: models.UnitKWh, Source: "synctacles", Zone: "FI"},
		{Timestamp: base.Add(time.Hour), PriceEUR: 0.28, Unit: models.UnitKWh, Source: "synctacles", Zone: "FI"},
	}

	result := aggregateToHourly(prices)
	require.Len(t, result, 2, "already hourly — should pass through unchanged")
	assert.InDelta(t, 0.25, result[0].PriceEUR, 0.001)
	assert.InDelta(t, 0.28, result[1].PriceEUR, 0.001)
}
