package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
