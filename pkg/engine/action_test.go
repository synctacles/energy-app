package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synctacles/energy-app/pkg/models"
)

func makeHourlyPrices(date time.Time, pricesKWh []float64) []models.HourlyPrice {
	result := make([]models.HourlyPrice, len(pricesKWh))
	for i, p := range pricesKWh {
		result[i] = models.HourlyPrice{
			Timestamp: time.Date(date.Year(), date.Month(), date.Day(), i, 0, 0, 0, time.UTC),
			PriceEUR:  p,
			Unit:      models.UnitKWh,
			Source:    "test",
			Quality:   "live",
			Zone:      "NL",
		}
	}
	return result
}

func TestActionEngine_GO(t *testing.T) {
	// Average of these 24 prices ≈ 0.20
	prices := make([]float64, 24)
	for i := range prices {
		prices[i] = 0.20
	}
	// Make hour 3 very cheap → should trigger GO
	prices[3] = 0.10 // 50% below average

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	hourly := makeHourlyPrices(date, prices)

	engine := NewActionEngine(-15, 20)
	now := time.Date(2026, 2, 11, 3, 30, 0, 0, time.UTC)
	result := engine.Calculate(hourly, now, true)

	assert.Equal(t, models.ActionGo, result.Action)
	assert.Less(t, result.DeviationPct, -15.0)
}

func TestActionEngine_AVOID(t *testing.T) {
	prices := make([]float64, 24)
	for i := range prices {
		prices[i] = 0.20
	}
	// Make hour 18 very expensive → should trigger AVOID
	prices[18] = 0.35 // 75% above average

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	hourly := makeHourlyPrices(date, prices)

	engine := NewActionEngine(-15, 20)
	now := time.Date(2026, 2, 11, 18, 30, 0, 0, time.UTC)
	result := engine.Calculate(hourly, now, true)

	assert.Equal(t, models.ActionAvoid, result.Action)
	assert.Greater(t, result.DeviationPct, 20.0)
}

func TestActionEngine_WAIT(t *testing.T) {
	prices := make([]float64, 24)
	for i := range prices {
		prices[i] = 0.20
	}

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	hourly := makeHourlyPrices(date, prices)

	engine := NewActionEngine(-15, 20)
	now := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	result := engine.Calculate(hourly, now, true)

	assert.Equal(t, models.ActionWait, result.Action)
}

func TestActionEngine_NoGO_WhenNotAllowed(t *testing.T) {
	prices := make([]float64, 24)
	for i := range prices {
		prices[i] = 0.20
	}
	prices[3] = 0.10

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	hourly := makeHourlyPrices(date, prices)

	engine := NewActionEngine(-15, 20)
	now := time.Date(2026, 2, 11, 3, 0, 0, 0, time.UTC)
	result := engine.Calculate(hourly, now, false) // allowGo=false (cached data)

	assert.Equal(t, models.ActionWait, result.Action)
	// GO conditions are not entered when allowGo=false, so falls through to WAIT
	assert.NotEqual(t, models.ActionGo, result.Action)
}

func TestActionEngine_Best4_AlwaysGO(t *testing.T) {
	// 24 hours, hours 2-5 are cheapest (0.10), rest 0.25
	prices := make([]float64, 24)
	for i := range prices {
		prices[i] = 0.25
	}
	prices[2] = 0.10
	prices[3] = 0.11
	prices[4] = 0.12
	prices[5] = 0.13

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	hourly := makeHourlyPrices(date, prices)

	// Even with threshold that wouldn't trigger, best4 still forces GO
	engine := NewActionEngine(-50, 50) // Very strict thresholds
	now := time.Date(2026, 2, 11, 2, 0, 0, 0, time.UTC)
	result := engine.Calculate(hourly, now, true)

	assert.Equal(t, models.ActionGo, result.Action)
	assert.Contains(t, result.Reason, "cheapest 4 hours")
}

func TestActionEngine_EmptyPrices(t *testing.T) {
	engine := NewActionEngine(-15, 20)
	result := engine.Calculate(nil, time.Now(), true)
	assert.Equal(t, models.ActionWait, result.Action)
}

// --- CurrentSlotPrice tests ---

func TestCurrentSlotPrice_Hourly(t *testing.T) {
	date := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	prices := makeHourlyPrices(date, []float64{0.10, 0.15, 0.20, 0.25})

	// At 02:35, should pick the 02:00 slot (0.20)
	now := time.Date(2026, 3, 3, 2, 35, 0, 0, time.UTC)
	price, slot, found := CurrentSlotPrice(prices, now)
	assert.True(t, found)
	assert.Equal(t, 0.20, price)
	assert.Equal(t, time.Date(2026, 3, 3, 2, 0, 0, 0, time.UTC), slot)
}

func TestCurrentSlotPrice_PT15(t *testing.T) {
	// Simulate 4 PT15 entries for hour 12:00-12:45
	prices := []models.HourlyPrice{
		{Timestamp: time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC), PriceEUR: 0.18},
		{Timestamp: time.Date(2026, 3, 3, 11, 15, 0, 0, time.UTC), PriceEUR: 0.19},
		{Timestamp: time.Date(2026, 3, 3, 11, 30, 0, 0, time.UTC), PriceEUR: 0.1687},
		{Timestamp: time.Date(2026, 3, 3, 11, 45, 0, 0, time.UTC), PriceEUR: 0.15},
	}

	// At 11:35 UTC, should pick 11:30 slot (0.1687), NOT 11:00 (0.18)
	now := time.Date(2026, 3, 3, 11, 35, 0, 0, time.UTC)
	price, slot, found := CurrentSlotPrice(prices, now)
	assert.True(t, found)
	assert.Equal(t, 0.1687, price)
	assert.Equal(t, time.Date(2026, 3, 3, 11, 30, 0, 0, time.UTC), slot)
}

func TestCurrentSlotPrice_ExactMatch(t *testing.T) {
	prices := []models.HourlyPrice{
		{Timestamp: time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC), PriceEUR: 0.20},
		{Timestamp: time.Date(2026, 3, 3, 12, 15, 0, 0, time.UTC), PriceEUR: 0.22},
	}

	// Exactly at 12:15, should pick 12:15 (0.22)
	now := time.Date(2026, 3, 3, 12, 15, 0, 0, time.UTC)
	price, _, found := CurrentSlotPrice(prices, now)
	assert.True(t, found)
	assert.Equal(t, 0.22, price)
}

func TestCurrentSlotPrice_BeforeAllEntries(t *testing.T) {
	prices := []models.HourlyPrice{
		{Timestamp: time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC), PriceEUR: 0.20},
	}

	// Before any entry — should not find
	now := time.Date(2026, 3, 3, 11, 59, 0, 0, time.UTC)
	_, _, found := CurrentSlotPrice(prices, now)
	assert.False(t, found)
}

func TestCurrentSlotPrice_Empty(t *testing.T) {
	_, _, found := CurrentSlotPrice(nil, time.Now())
	assert.False(t, found)
}

func TestActionEngine_PT15_CorrectSlot(t *testing.T) {
	// PT15 prices: 4 entries per hour
	prices := []models.HourlyPrice{
		{Timestamp: time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC), PriceEUR: 0.30, Unit: models.UnitKWh, Source: "test", Quality: "live", Zone: "NL"},
		{Timestamp: time.Date(2026, 3, 3, 12, 15, 0, 0, time.UTC), PriceEUR: 0.25, Unit: models.UnitKWh, Source: "test", Quality: "live", Zone: "NL"},
		{Timestamp: time.Date(2026, 3, 3, 12, 30, 0, 0, time.UTC), PriceEUR: 0.10, Unit: models.UnitKWh, Source: "test", Quality: "live", Zone: "NL"},
		{Timestamp: time.Date(2026, 3, 3, 12, 45, 0, 0, time.UTC), PriceEUR: 0.35, Unit: models.UnitKWh, Source: "test", Quality: "live", Zone: "NL"},
	}

	engine := NewActionEngine(-15, 20)
	// At 12:35, the current slot is 12:30 (price 0.10 = cheapest → GO)
	now := time.Date(2026, 3, 3, 12, 35, 0, 0, time.UTC)
	result := engine.Calculate(prices, now, true)

	assert.Equal(t, 0.10, result.CurrentPrice)
	assert.Equal(t, models.ActionGo, result.Action)
}
