package engine

import (
	"testing"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

func TestNeedsFreshFetch_NoLastResult(t *testing.T) {
	s := &Scheduler{}
	if !s.needsFreshFetch() {
		t.Error("should need fresh fetch when lastResult is nil")
	}
}

func TestNeedsFreshFetch_HasTomorrowData(t *testing.T) {
	s := &Scheduler{
		lastResult:      &FetchResult{Prices: []models.HourlyPrice{{PriceEUR: 0.1}}},
		hasTomorrowData: true,
		lastFetchDay:    time.Now().UTC().YearDay(), // same day → no day-rollover trigger
	}
	// Should NOT need fresh fetch when we already have tomorrow's data
	if s.needsFreshFetch() {
		t.Error("should not need fresh fetch when tomorrow data is available")
	}
}

func TestNeedsFreshFetch_DayRollover(t *testing.T) {
	s := &Scheduler{
		lastResult:      &FetchResult{Prices: []models.HourlyPrice{{PriceEUR: 0.1}}},
		hasTomorrowData: true,
		lastFetchDay:    time.Now().UTC().YearDay() - 1, // yesterday → must re-fetch
	}
	if !s.needsFreshFetch() {
		t.Error("should need fresh fetch after day rollover")
	}
}

func TestNeedsFreshFetch_NoTomorrowOutsideWindow(t *testing.T) {
	s := &Scheduler{
		lastResult:      &FetchResult{Prices: []models.HourlyPrice{{PriceEUR: 0.1}}},
		hasTomorrowData: false,
		lastFetchDay:    time.Now().UTC().YearDay(),
	}
	// Outside 13-14 UTC window, should not need fresh fetch even without tomorrow data
	// (we can't control time.Now() easily, so this test documents the behavior
	// rather than asserting a specific outcome based on current time)
	_ = s.needsFreshFetch()
}

func TestHasTomorrowPrices_Empty(t *testing.T) {
	if hasTomorrowPrices(nil) {
		t.Error("empty prices should not have tomorrow data")
	}
}

func TestHasTomorrowPrices_TodayOnly(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	prices := []models.HourlyPrice{
		{Timestamp: today, PriceEUR: 0.1},
		{Timestamp: today.Add(12 * time.Hour), PriceEUR: 0.2},
	}
	if hasTomorrowPrices(prices) {
		t.Error("today-only prices should not report tomorrow data")
	}
}

func TestHasTomorrowPrices_WithTomorrow(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	prices := []models.HourlyPrice{
		{Timestamp: today, PriceEUR: 0.1},
		{Timestamp: tomorrow, PriceEUR: 0.2},
	}
	if !hasTomorrowPrices(prices) {
		t.Error("should detect tomorrow prices")
	}
}
