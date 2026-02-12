package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synctacles/energy-go/internal/models"
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
