package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptCicloDiario() *TOUConfig {
	return &TOUConfig{
		Rates: []TOURate{
			{ID: "offpeak", Name: "Vazio", Price: 0.0951},
			{ID: "peak", Name: "Fora de vazio", Price: 0.1645},
		},
		Periods: []TOUPeriod{
			{Days: []int{0, 1, 2, 3, 4, 5, 6}, Start: "08:00", End: "22:00", RateID: "peak"},
		},
		Default: "offpeak",
	}
}

func ptCicloSemanal() *TOUConfig {
	return &TOUConfig{
		Rates: []TOURate{
			{ID: "offpeak", Name: "Vazio", Price: 0.0951},
			{ID: "peak", Name: "Fora de vazio", Price: 0.1645},
		},
		Periods: []TOUPeriod{
			{Days: []int{1, 2, 3, 4, 5}, Start: "08:00", End: "22:00", RateID: "peak"},
			{Days: []int{6}, Start: "09:00", End: "14:00", RateID: "peak"},
			{Days: []int{6}, Start: "20:00", End: "22:00", RateID: "peak"},
		},
		Default: "offpeak",
	}
}

func TestTOUConfig_Validate(t *testing.T) {
	cfg := ptCicloDiario()
	assert.NoError(t, cfg.Validate())
}

func TestTOUConfig_Validate_TooFewRates(t *testing.T) {
	cfg := &TOUConfig{
		Rates:   []TOURate{{ID: "flat", Price: 0.15}},
		Default: "flat",
	}
	assert.Error(t, cfg.Validate())
}

func TestTOUConfig_Validate_MissingDefault(t *testing.T) {
	cfg := ptCicloDiario()
	cfg.Default = "nonexistent"
	assert.Error(t, cfg.Validate())
}

func TestTOUConfig_Validate_BadPeriodRate(t *testing.T) {
	cfg := ptCicloDiario()
	cfg.Periods[0].RateID = "nonexistent"
	assert.Error(t, cfg.Validate())
}

func TestTOUConfig_Validate_BadTime(t *testing.T) {
	cfg := ptCicloDiario()
	cfg.Periods[0].Start = "25:00"
	assert.Error(t, cfg.Validate())
}

func TestParseTOUConfig(t *testing.T) {
	cfg := ptCicloDiario()
	data, _ := json.Marshal(cfg)
	parsed, err := ParseTOUConfig(string(data))
	require.NoError(t, err)
	assert.Equal(t, 2, len(parsed.Rates))
	assert.Equal(t, "offpeak", parsed.Default)
}

func TestParseTOUConfig_Empty(t *testing.T) {
	_, err := ParseTOUConfig("")
	assert.Error(t, err)
}

func TestGenerateTOUPrices_CicloDiario(t *testing.T) {
	cfg := ptCicloDiario()
	// Wednesday 2026-03-11
	loc, _ := time.LoadLocation("Europe/Lisbon")

	today, tomorrow := GenerateTOUPrices(cfg, loc)

	assert.Len(t, today, 24)
	assert.Len(t, tomorrow, 24)

	// Check today: hours 0-7 off-peak, 8-21 peak, 22-23 off-peak
	for h := 0; h < 8; h++ {
		assert.InDelta(t, 0.0951, today[h].PriceEUR, 0.0001, "hour %d should be offpeak", h)
	}
	for h := 8; h < 22; h++ {
		assert.InDelta(t, 0.1645, today[h].PriceEUR, 0.0001, "hour %d should be peak", h)
	}
	for h := 22; h < 24; h++ {
		assert.InDelta(t, 0.0951, today[h].PriceEUR, 0.0001, "hour %d should be offpeak", h)
	}

	// All should be consumer prices
	for _, p := range today {
		assert.True(t, p.IsConsumer)
		assert.Equal(t, "tou", p.Source)
		assert.Equal(t, "live", p.Quality)
	}
}

func TestGenerateTOUPrices_CicloSemanal_Sunday(t *testing.T) {
	cfg := ptCicloSemanal()
	loc, _ := time.LoadLocation("Europe/Lisbon")

	// Force "now" to be a Sunday by generating from a known Sunday
	// Sunday 2026-03-15
	sundayStart := time.Date(2026, 3, 15, 0, 0, 0, 0, loc)
	prices := generateDay(cfg, sundayStart, loc)

	// Sunday (day 0) has no periods → all off-peak
	for h := 0; h < 24; h++ {
		assert.InDelta(t, 0.0951, prices[h].PriceEUR, 0.0001, "sunday hour %d should be offpeak", h)
	}
}

func TestGenerateTOUPrices_CicloSemanal_Saturday(t *testing.T) {
	cfg := ptCicloSemanal()
	loc, _ := time.LoadLocation("Europe/Lisbon")

	// Saturday 2026-03-14
	satStart := time.Date(2026, 3, 14, 0, 0, 0, 0, loc)
	prices := generateDay(cfg, satStart, loc)

	// Saturday: peak 09-14 and 20-22, rest off-peak
	for h := 0; h < 9; h++ {
		assert.InDelta(t, 0.0951, prices[h].PriceEUR, 0.0001, "sat hour %d should be offpeak", h)
	}
	for h := 9; h < 14; h++ {
		assert.InDelta(t, 0.1645, prices[h].PriceEUR, 0.0001, "sat hour %d should be peak", h)
	}
	for h := 14; h < 20; h++ {
		assert.InDelta(t, 0.0951, prices[h].PriceEUR, 0.0001, "sat hour %d should be offpeak", h)
	}
	for h := 20; h < 22; h++ {
		assert.InDelta(t, 0.1645, prices[h].PriceEUR, 0.0001, "sat hour %d should be peak", h)
	}
	for h := 22; h < 24; h++ {
		assert.InDelta(t, 0.0951, prices[h].PriceEUR, 0.0001, "sat hour %d should be offpeak", h)
	}
}

func TestGenerateTOUPrices_CicloSemanal_Weekday(t *testing.T) {
	cfg := ptCicloSemanal()
	loc, _ := time.LoadLocation("Europe/Lisbon")

	// Monday 2026-03-09
	monStart := time.Date(2026, 3, 9, 0, 0, 0, 0, loc)
	prices := generateDay(cfg, monStart, loc)

	// Weekday: peak 08-22
	for h := 0; h < 8; h++ {
		assert.InDelta(t, 0.0951, prices[h].PriceEUR, 0.0001, "mon hour %d should be offpeak", h)
	}
	for h := 8; h < 22; h++ {
		assert.InDelta(t, 0.1645, prices[h].PriceEUR, 0.0001, "mon hour %d should be peak", h)
	}
	for h := 22; h < 24; h++ {
		assert.InDelta(t, 0.0951, prices[h].PriceEUR, 0.0001, "mon hour %d should be offpeak", h)
	}
}

func TestGenerateTOUPrices_OvernightPeak(t *testing.T) {
	// Night tariff: peak 22:00-06:00
	cfg := &TOUConfig{
		Rates: []TOURate{
			{ID: "offpeak", Name: "Day", Price: 0.10},
			{ID: "peak", Name: "Night", Price: 0.20},
		},
		Periods: []TOUPeriod{
			{Days: []int{0, 1, 2, 3, 4, 5, 6}, Start: "22:00", End: "06:00", RateID: "peak"},
		},
		Default: "offpeak",
	}
	loc := time.UTC
	monStart := time.Date(2026, 3, 9, 0, 0, 0, 0, loc) // Monday

	prices := generateDay(cfg, monStart, loc)

	// 00-05: peak (overnight continuation), 06-21: offpeak, 22-23: peak
	for h := 0; h < 6; h++ {
		assert.InDelta(t, 0.20, prices[h].PriceEUR, 0.0001, "hour %d should be peak (overnight)", h)
	}
	for h := 6; h < 22; h++ {
		assert.InDelta(t, 0.10, prices[h].PriceEUR, 0.0001, "hour %d should be offpeak", h)
	}
	for h := 22; h < 24; h++ {
		assert.InDelta(t, 0.20, prices[h].PriceEUR, 0.0001, "hour %d should be peak", h)
	}
}

func TestGenerateTOUPrices_TriHorario(t *testing.T) {
	// Tri-hourly: peak, mid-peak, off-peak
	cfg := &TOUConfig{
		Rates: []TOURate{
			{ID: "offpeak", Name: "Super Vazio", Price: 0.08},
			{ID: "midpeak", Name: "Cheias", Price: 0.14},
			{ID: "peak", Name: "Ponta", Price: 0.22},
		},
		Periods: []TOUPeriod{
			{Days: []int{1, 2, 3, 4, 5}, Start: "09:00", End: "12:00", RateID: "peak"},
			{Days: []int{1, 2, 3, 4, 5}, Start: "18:00", End: "21:00", RateID: "peak"},
			{Days: []int{1, 2, 3, 4, 5}, Start: "07:00", End: "09:00", RateID: "midpeak"},
			{Days: []int{1, 2, 3, 4, 5}, Start: "12:00", End: "18:00", RateID: "midpeak"},
			{Days: []int{1, 2, 3, 4, 5}, Start: "21:00", End: "23:00", RateID: "midpeak"},
		},
		Default: "offpeak",
	}

	loc := time.UTC
	monStart := time.Date(2026, 3, 9, 0, 0, 0, 0, loc) // Monday
	prices := generateDay(cfg, monStart, loc)

	assert.InDelta(t, 0.08, prices[0].PriceEUR, 0.0001, "00:00 offpeak")
	assert.InDelta(t, 0.14, prices[7].PriceEUR, 0.0001, "07:00 midpeak")
	assert.InDelta(t, 0.22, prices[9].PriceEUR, 0.0001, "09:00 peak")
	assert.InDelta(t, 0.14, prices[12].PriceEUR, 0.0001, "12:00 midpeak")
	assert.InDelta(t, 0.22, prices[18].PriceEUR, 0.0001, "18:00 peak")
	assert.InDelta(t, 0.14, prices[21].PriceEUR, 0.0001, "21:00 midpeak")
	assert.InDelta(t, 0.08, prices[23].PriceEUR, 0.0001, "23:00 offpeak")
}

func TestMatchesPeriod(t *testing.T) {
	p := TOUPeriod{Days: []int{1, 2, 3, 4, 5}, Start: "08:00", End: "22:00", RateID: "peak"}

	assert.True(t, matchesPeriod(p, 1, 8))   // Monday 08:00
	assert.True(t, matchesPeriod(p, 1, 21))  // Monday 21:00
	assert.False(t, matchesPeriod(p, 1, 22)) // Monday 22:00 (end exclusive)
	assert.False(t, matchesPeriod(p, 1, 7))  // Monday 07:00
	assert.False(t, matchesPeriod(p, 0, 12)) // Sunday 12:00 (wrong day)
}

func TestMatchesPeriod_Overnight(t *testing.T) {
	p := TOUPeriod{Days: []int{0, 1, 2, 3, 4, 5, 6}, Start: "22:00", End: "06:00", RateID: "peak"}

	assert.True(t, matchesPeriod(p, 1, 22))  // Monday 22:00
	assert.True(t, matchesPeriod(p, 1, 0))   // Monday 00:00
	assert.True(t, matchesPeriod(p, 1, 5))   // Monday 05:00
	assert.False(t, matchesPeriod(p, 1, 6))  // Monday 06:00 (end exclusive)
	assert.False(t, matchesPeriod(p, 1, 12)) // Monday 12:00
}
