package engine

import (
	"testing"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// ============================================================================
// findCheapestN edge cases
// ============================================================================

func TestFindCheapestN_ExactN(t *testing.T) {
	// 4 prices, ask for 4 cheapest
	date := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	prices := makeHourlyPrices(date, []float64{0.20, 0.10, 0.30, 0.15})

	result := findCheapestN(prices, 4)
	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}
}

func TestFindCheapestN_MoreThanAvailable(t *testing.T) {
	date := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	prices := makeHourlyPrices(date, []float64{0.20, 0.10})

	result := findCheapestN(prices, 10)
	if len(result) != 2 {
		t.Fatalf("expected 2 results (all available), got %d", len(result))
	}
}

func TestFindCheapestN_SinglePrice(t *testing.T) {
	date := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	prices := makeHourlyPrices(date, []float64{0.25})

	result := findCheapestN(prices, 4)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0] != "00:00" {
		t.Errorf("expected 00:00, got %s", result[0])
	}
}

func TestFindCheapestN_SortedByTime(t *testing.T) {
	// Cheapest 3 are at hours 5, 1, 10 (unsorted by time)
	// Result should be sorted: 01:00, 05:00, 10:00
	date := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	values := make([]float64, 24)
	for i := range values {
		values[i] = 0.30 // expensive default
	}
	values[5] = 0.05  // cheapest
	values[1] = 0.08  // 2nd cheapest
	values[10] = 0.10 // 3rd cheapest

	prices := makeHourlyPrices(date, values)
	result := findCheapestN(prices, 3)

	expected := []string{"01:00", "05:00", "10:00"}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	for i, r := range result {
		if r != expected[i] {
			t.Errorf("result[%d] = %s, want %s", i, r, expected[i])
		}
	}
}

func TestFindCheapestN_EqualPrices(t *testing.T) {
	// All same price — should still return N hours sorted by time
	date := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	values := make([]float64, 24)
	for i := range values {
		values[i] = 0.20
	}
	prices := makeHourlyPrices(date, values)
	result := findCheapestN(prices, 4)
	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}
}

// ============================================================================
// CalcStats edge cases
// ============================================================================

func TestCalcStats_SinglePrice(t *testing.T) {
	date := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	prices := makeHourlyPrices(date, []float64{0.15})

	stats := CalcStats(prices)
	if stats.Average != 0.15 {
		t.Errorf("average = %f, want 0.15", stats.Average)
	}
	if stats.Min != 0.15 || stats.Max != 0.15 {
		t.Error("min and max should equal the single price")
	}
	if stats.CheapestHour != "00:00" || stats.ExpensiveHour != "00:00" {
		t.Error("cheapest and most expensive should both be 00:00")
	}
}

func TestCalcStats_NegativePrices(t *testing.T) {
	date := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	prices := makeHourlyPrices(date, []float64{-0.05, 0.10, -0.02, 0.20})

	stats := CalcStats(prices)
	if stats.Min != -0.05 {
		t.Errorf("min = %f, want -0.05", stats.Min)
	}
	if stats.CheapestHour != "00:00" {
		t.Errorf("cheapest hour = %s, want 00:00", stats.CheapestHour)
	}
}

func TestCalcStats_Best4WithFewerHours(t *testing.T) {
	date := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	prices := makeHourlyPrices(date, []float64{0.10, 0.20})

	stats := CalcStats(prices)
	if len(stats.Best4Hours) != 2 {
		t.Errorf("best4 should have 2 entries when only 2 hours, got %d", len(stats.Best4Hours))
	}
}

func TestCalcStats_24Hours(t *testing.T) {
	date := time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC)
	values := make([]float64, 24)
	for i := range values {
		values[i] = float64(i) * 0.01 // 0.00, 0.01, ..., 0.23
	}
	prices := makeHourlyPrices(date, values)

	stats := CalcStats(prices)
	if stats.CheapestHour != "00:00" {
		t.Errorf("cheapest = %s, want 00:00", stats.CheapestHour)
	}
	if stats.ExpensiveHour != "23:00" {
		t.Errorf("most expensive = %s, want 23:00", stats.ExpensiveHour)
	}

	// Average of 0..23 * 0.01 = 0.115
	expected := (0.0 + 0.23) / 2.0
	if stats.Average < expected-0.001 || stats.Average > expected+0.001 {
		t.Errorf("average = %f, want ~%f", stats.Average, expected)
	}

	if len(stats.Best4Hours) != 4 {
		t.Fatalf("best4 should have 4 entries, got %d", len(stats.Best4Hours))
	}
	// Cheapest 4 hours: 00:00, 01:00, 02:00, 03:00
	expected4 := []string{"00:00", "01:00", "02:00", "03:00"}
	for i, h := range stats.Best4Hours {
		if h != expected4[i] {
			t.Errorf("best4[%d] = %s, want %s", i, h, expected4[i])
		}
	}
}

func TestCalcStats_EmptyNilInputs(t *testing.T) {
	stats := CalcStats(nil)
	if stats.Average != 0 || stats.Min != 0 || stats.Max != 0 {
		t.Error("nil input should return zero stats")
	}

	stats = CalcStats([]models.HourlyPrice{})
	if stats.Average != 0 {
		t.Error("empty slice should return zero stats")
	}
}
