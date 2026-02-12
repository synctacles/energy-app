package hasensor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyCost_NoData(t *testing.T) {
	pt := &PowerTracker{
		hourlyUsage: make([]hourUsage, 24),
	}

	cost, kwh, ok := pt.DailyCost()
	assert.False(t, ok)
	assert.Equal(t, 0.0, cost)
	assert.Equal(t, 0.0, kwh)
}

func TestDailyCost_WithData(t *testing.T) {
	pt := &PowerTracker{
		hourlyUsage: make([]hourUsage, 24),
	}

	// Simulate 3 hours of usage
	// Hour 0: 1000W at €0.10/kWh → 1 kWh × €0.10 = €0.10
	pt.hourlyUsage[0] = hourUsage{Hour: 0, AvgWatts: 1000, PriceEUR: 0.10, Samples: 4}
	// Hour 1: 2000W at €0.20/kWh → 2 kWh × €0.20 = €0.40
	pt.hourlyUsage[1] = hourUsage{Hour: 1, AvgWatts: 2000, PriceEUR: 0.20, Samples: 4}
	// Hour 2: 500W at €0.30/kWh → 0.5 kWh × €0.30 = €0.15
	pt.hourlyUsage[2] = hourUsage{Hour: 2, AvgWatts: 500, PriceEUR: 0.30, Samples: 4}

	cost, kwh, ok := pt.DailyCost()
	require.True(t, ok)

	// Total cost: 0.10 + 0.40 + 0.15 = 0.65 EUR
	assert.InDelta(t, 0.65, cost, 0.001)
	// Total kWh: (1000 + 2000 + 500) / 1000 = 3.5 kWh
	assert.InDelta(t, 3.5, kwh, 0.001)
}

func TestDailyCost_ResetsOnNewDay(t *testing.T) {
	pt := &PowerTracker{
		hourlyUsage: make([]hourUsage, 24),
		resetDay:    15, // was day 15
	}

	// Add some data
	pt.hourlyUsage[0] = hourUsage{Hour: 0, AvgWatts: 1000, PriceEUR: 0.10, Samples: 4}

	cost, _, ok := pt.DailyCost()
	require.True(t, ok)
	assert.InDelta(t, 0.10, cost, 0.001)

	// Reset (simulated by clearing hourlyUsage as ReadPower would)
	pt.hourlyUsage = make([]hourUsage, 24)
	pt.resetDay = 16

	_, _, ok = pt.DailyCost()
	assert.False(t, ok) // No data after reset
}
