package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synctacles/energy-app/pkg/models"
)

func TestFindBestWindow_Basic(t *testing.T) {
	// 24 hours: hours 2-5 are cheapest
	prices := make([]float64, 24)
	for i := range prices {
		prices[i] = 0.25
	}
	prices[2] = 0.08
	prices[3] = 0.07
	prices[4] = 0.06
	prices[5] = 0.09

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	hourly := makeHourlyPrices(date, prices)

	now := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC) // Midnight — all hours are future
	window := FindBestWindow(hourly, now, 3)

	require.NotNil(t, window)
	assert.Equal(t, "02:00", window.StartHour)
	assert.Equal(t, "05:00", window.EndHour)
	assert.Equal(t, 3, window.Duration)
	assert.InDelta(t, 0.07, window.AvgPrice, 0.01)
}

func TestFindBestWindow_OnlyFutureHours(t *testing.T) {
	prices := make([]float64, 24)
	for i := range prices {
		prices[i] = 0.25
	}
	// Hours 2-4 are cheapest but in the past
	prices[2] = 0.05
	prices[3] = 0.05
	prices[4] = 0.05
	// Hours 20-22 are cheapest remaining future hours
	prices[20] = 0.10
	prices[21] = 0.10
	prices[22] = 0.10

	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	hourly := makeHourlyPrices(date, prices)

	now := time.Date(2026, 2, 11, 15, 0, 0, 0, time.UTC) // 3PM — morning is past
	window := FindBestWindow(hourly, now, 3)

	require.NotNil(t, window)
	assert.Equal(t, "20:00", window.StartHour)
}

func TestFindBestWindow_NotEnoughHours(t *testing.T) {
	prices := make([]float64, 3)
	for i := range prices {
		prices[i] = 0.20
	}
	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	hourly := makeHourlyPrices(date, prices)

	// Ask for 5-hour window but only 3 hours exist
	window := FindBestWindow(hourly, date, 5)
	assert.Nil(t, window)
}

func TestFindBestWindow_EmptyPrices(t *testing.T) {
	window := FindBestWindow(nil, time.Now(), 3)
	assert.Nil(t, window)
}

func TestFindBestWindow_PT15(t *testing.T) {
	// 8 hours of PT15 data = 32 entries, hours 2-4 cheapest
	var prices []models.HourlyPrice
	base := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	for h := 0; h < 8; h++ {
		for q := 0; q < 4; q++ {
			p := 0.25
			if h >= 2 && h <= 4 {
				p = 0.08 + float64(q)*0.01 // vary within hour
			}
			prices = append(prices, models.HourlyPrice{
				Timestamp: base.Add(time.Duration(h)*time.Hour + time.Duration(q)*15*time.Minute),
				PriceEUR:  p,
				Unit:      models.UnitKWh,
				Source:    "test",
				Quality:   "live",
				Zone:      "NL",
			})
		}
	}

	now := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	window := FindBestWindow(prices, now, 3)

	require.NotNil(t, window)
	assert.Equal(t, "02:00", window.StartHour)
	assert.Equal(t, "05:00", window.EndHour) // 3 hours later
	assert.Equal(t, 3, window.Duration)
}

func TestDetectSlotDuration_PT15(t *testing.T) {
	base := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	prices := []models.HourlyPrice{
		{Timestamp: base},
		{Timestamp: base.Add(15 * time.Minute)},
		{Timestamp: base.Add(30 * time.Minute)},
	}
	assert.Equal(t, 15*time.Minute, DetectSlotDuration(prices))
}

func TestDetectSlotDuration_Hourly(t *testing.T) {
	base := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	prices := []models.HourlyPrice{
		{Timestamp: base},
		{Timestamp: base.Add(time.Hour)},
	}
	assert.Equal(t, time.Hour, DetectSlotDuration(prices))
}

func TestDetectSlotDuration_Single(t *testing.T) {
	prices := []models.HourlyPrice{{Timestamp: time.Now()}}
	assert.Equal(t, time.Hour, DetectSlotDuration(prices))
}
