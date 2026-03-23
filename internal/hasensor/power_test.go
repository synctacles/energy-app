package hasensor

import (
	"testing"
	"time"

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

// ============================================================================
// Additional PowerTracker tests: accumulation, daily reset, savings, live cost
// ============================================================================

func TestPowerTracker_AccumulateUsage(t *testing.T) {
	pt := &PowerTracker{
		hourlyUsage: make([]hourUsage, 24),
		resetDay:    15,
	}

	// Simulate multiple readings in the same hour (running average)
	// Hour 10: readings of 1000W, 2000W, 1500W
	pt.mu.Lock()
	h := 10
	u := &pt.hourlyUsage[h]
	u.Hour = h
	u.PriceEUR = 0.20

	// First sample: 1000W
	u.Samples++
	u.AvgWatts = u.AvgWatts + (1000-u.AvgWatts)/float64(u.Samples) // 1000

	// Second sample: 2000W
	u.Samples++
	u.AvgWatts = u.AvgWatts + (2000-u.AvgWatts)/float64(u.Samples) // 1500

	// Third sample: 1500W
	u.Samples++
	u.AvgWatts = u.AvgWatts + (1500-u.AvgWatts)/float64(u.Samples) // 1500

	pt.mu.Unlock()

	// Average should be (1000+2000+1500)/3 = 1500
	assert.Equal(t, 3, pt.hourlyUsage[h].Samples)
	assert.InDelta(t, 1500.0, pt.hourlyUsage[h].AvgWatts, 0.1)
}

func TestPowerTracker_DailyReset(t *testing.T) {
	pt := &PowerTracker{
		hourlyUsage: make([]hourUsage, 24),
		resetDay:    14, // yesterday
	}

	// Pre-populate some data from "yesterday"
	pt.hourlyUsage[5] = hourUsage{Hour: 5, AvgWatts: 1200, PriceEUR: 0.15, Samples: 4}
	pt.hourlyUsage[6] = hourUsage{Hour: 6, AvgWatts: 800, PriceEUR: 0.18, Samples: 4}

	// Verify pre-existing data
	cost, _, ok := pt.DailyCost()
	require.True(t, ok)
	assert.Greater(t, cost, 0.0)

	// Simulate day change: resetDay differs from current day
	// In the real ReadPower method, this would reset hourlyUsage.
	// We simulate it directly:
	currentDay := 15
	pt.mu.Lock()
	if currentDay != pt.resetDay {
		pt.hourlyUsage = make([]hourUsage, 24)
		pt.resetDay = currentDay
	}
	pt.mu.Unlock()

	// After reset, no data
	_, _, ok = pt.DailyCost()
	assert.False(t, ok)

	// Add new data for today
	pt.mu.Lock()
	pt.hourlyUsage[8] = hourUsage{Hour: 8, AvgWatts: 500, PriceEUR: 0.12, Samples: 2}
	pt.mu.Unlock()

	cost, kwh, ok := pt.DailyCost()
	require.True(t, ok)
	assert.InDelta(t, 0.06, cost, 0.001)  // 0.5 kWh × €0.12
	assert.InDelta(t, 0.5, kwh, 0.001)    // 500W → 0.5 kWh
}

func TestPowerTracker_LiveCost(t *testing.T) {
	pt := &PowerTracker{
		hourlyUsage: make([]hourUsage, 24),
	}

	// No readings yet
	_, _, ok := pt.LiveCost(0.20)
	assert.False(t, ok)

	// Set a power reading
	pt.mu.Lock()
	pt.currentW = 3000 // 3 kW
	pt.lastRead = time.Now()
	pt.mu.Unlock()

	eurPerHour, powerW, ok := pt.LiveCost(0.25)
	require.True(t, ok)
	assert.InDelta(t, 3000.0, powerW, 0.01)
	// 3 kW × €0.25/kWh = €0.75/h
	assert.InDelta(t, 0.75, eurPerHour, 0.001)
}

func TestPowerTracker_DailySavings(t *testing.T) {
	pt := &PowerTracker{
		hourlyUsage: make([]hourUsage, 24),
	}

	avgPrice := 0.20

	// No data
	_, _, ok := pt.DailySavings(avgPrice)
	assert.False(t, ok)

	// Cheap hour: 2000W at €0.10 (below avg)
	pt.hourlyUsage[3] = hourUsage{Hour: 3, AvgWatts: 2000, PriceEUR: 0.10, Samples: 4}
	// Expensive hour: 500W at €0.35 (above avg)
	pt.hourlyUsage[18] = hourUsage{Hour: 18, AvgWatts: 500, PriceEUR: 0.35, Samples: 4}

	savings, totalKWh, ok := pt.DailySavings(avgPrice)
	require.True(t, ok)

	// actual cost = 2×0.10 + 0.5×0.35 = 0.20 + 0.175 = 0.375
	// flat cost   = 2×0.20 + 0.5×0.20 = 0.40 + 0.10  = 0.50
	// savings     = 0.50 - 0.375 = 0.125
	assert.InDelta(t, 0.125, savings, 0.001)
	assert.InDelta(t, 2.5, totalKWh, 0.001) // (2000+500)/1000
}

func TestPowerTracker_UsageScore(t *testing.T) {
	pt := &PowerTracker{
		hourlyUsage: make([]hourUsage, 24),
	}

	avgPrice := 0.20

	// No data
	_, _, _, _, ok := pt.UsageScore(avgPrice)
	assert.False(t, ok)

	// All usage during cheap hours (< 85% of avg = 0.17)
	pt.hourlyUsage[3] = hourUsage{Hour: 3, AvgWatts: 3000, PriceEUR: 0.10, Samples: 4}
	pt.hourlyUsage[4] = hourUsage{Hour: 4, AvgWatts: 2000, PriceEUR: 0.12, Samples: 4}

	score, cheapPct, _, expPct, ok := pt.UsageScore(avgPrice)
	require.True(t, ok)
	assert.Equal(t, 100.0, cheapPct) // all usage is cheap
	assert.Equal(t, 0.0, expPct)
	assert.GreaterOrEqual(t, score, 90) // high score for cheap-only usage
}
