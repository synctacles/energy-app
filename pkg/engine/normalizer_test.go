package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synctacles/energy-app/pkg/models"
)

func TestCalcStats(t *testing.T) {
	date := time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC)
	priceValues := []float64{0.20, 0.15, 0.10, 0.08, 0.12, 0.25, 0.30, 0.28,
		0.22, 0.18, 0.16, 0.14, 0.20, 0.22, 0.24, 0.26,
		0.28, 0.35, 0.40, 0.32, 0.24, 0.18, 0.14, 0.12}
	prices := makeHourlyPrices(date, priceValues)

	stats := CalcStats(prices)

	// Sum = 5.13, /24 = 0.21375
	assert.InDelta(t, 0.21375, stats.Average, 0.001)
	assert.InDelta(t, 0.08, stats.Min, 0.001)
	assert.InDelta(t, 0.40, stats.Max, 0.001)
	assert.Equal(t, "03:00", stats.CheapestHour)
	assert.Equal(t, "18:00", stats.ExpensiveHour)
	assert.Len(t, stats.Best4Hours, 4)
	// Best 4 should include hours 2,3,4,11 (0.10, 0.08, 0.12, 0.14 — cheapest)
	assert.Contains(t, stats.Best4Hours, "03:00")
}

func TestCalcStats_Empty(t *testing.T) {
	stats := CalcStats(nil)
	assert.Equal(t, 0.0, stats.Average)
}

// testTaxCache creates an in-memory TaxProfileCache with a NL zone profile.
func testTaxCache() *TaxProfileCache {
	cache := &TaxProfileCache{
		profiles: map[string]*WorkerTaxOverride{
			"NL": {
				VATRate:          0.21,
				EnergyTax:        0.09161,
				Surcharges:       0.0,
				NetworkTariffAvg: 0.0,
				Version:          "test-v1",
			},
		},
	}
	return cache
}

func TestNormalizer_MWhToConsumer(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)

	// 80 EUR/MWh = 0.08 EUR/kWh wholesale
	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "NL",
	}}

	consumer := norm.ToConsumer(prices)
	assert.Len(t, consumer, 1)
	assert.Equal(t, models.UnitKWh, consumer[0].Unit)
	// Expected: (0.08 + 0 + 0.09161 + 0 + 0) × 1.21 = 0.17161 × 1.21 ≈ 0.2076
	assert.InDelta(t, 0.2076, consumer[0].PriceEUR, 0.002)
}

func TestNormalizer_SkipsConsumerPrices(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)

	// Consumer price from Worker (already incl. VAT + taxes)
	consumerPrice := 0.2134
	prices := []models.HourlyPrice{{
		Timestamp:  time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:   consumerPrice,
		Unit:       models.UnitKWh,
		Source:     "synctacles",
		Zone:       "NL",
		IsConsumer: true,
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	assert.Equal(t, models.UnitKWh, result[0].Unit)
	// Price should be UNCHANGED — no tax applied
	assert.Equal(t, consumerPrice, result[0].PriceEUR)
	assert.True(t, result[0].IsConsumer)
}

func TestNormalizer_ConsumerPriceIgnoresSupplierMarkupOverride(t *testing.T) {
	cache := testTaxCache()
	// Even with a supplier markup override, consumer prices should not be touched
	norm := NewNormalizer(cache, 0.005)

	consumerPrice := 0.2134
	prices := []models.HourlyPrice{{
		Timestamp:  time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:   consumerPrice,
		Unit:       models.UnitKWh,
		Source:     "synctacles",
		Zone:       "NL",
		IsConsumer: true,
	}}

	result := norm.ToConsumer(prices)
	assert.Equal(t, consumerPrice, result[0].PriceEUR)
}

func TestNormalizer_MixedConsumerAndWholesale(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)

	ts := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	prices := []models.HourlyPrice{
		{Timestamp: ts, PriceEUR: 0.2134, Unit: models.UnitKWh, Source: "synctacles", Zone: "NL", IsConsumer: true},
		{Timestamp: ts, PriceEUR: 80.0, Unit: models.UnitMWh, Source: "energycharts", Zone: "NL", IsConsumer: false},
	}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 2)

	// First: consumer price unchanged
	assert.Equal(t, 0.2134, result[0].PriceEUR)

	// Second: wholesale normalized: (0.08 + 0.09161) × 1.21 ≈ 0.2076
	assert.InDelta(t, 0.2076, result[1].PriceEUR, 0.002)
}

func TestNormalizer_NoCacheFallsThrough(t *testing.T) {
	// No cache — wholesale prices should pass through as kWh
	norm := NewNormalizer(nil)

	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	// Converted to kWh but NOT to consumer (no tax data)
	assert.InDelta(t, 0.08, result[0].PriceEUR, 0.0001)
	assert.False(t, result[0].IsConsumer)
}

func TestNormalizer_UnknownZoneFallsThrough(t *testing.T) {
	cache := testTaxCache() // Only has "NL"
	norm := NewNormalizer(cache)

	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  50.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "XX", // Not in cache
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	// Converted to kWh but NOT to consumer
	assert.InDelta(t, 0.05, result[0].PriceEUR, 0.0001)
	assert.False(t, result[0].IsConsumer)
}

func TestNormalizer_ManualMode(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)
	norm.SetPricingMode("manual")
	norm.SetManualTaxProfile(&models.TaxProfile{
		VATRate:          0.21,
		SupplierMarkup:   0.01,
		EnergyTax:        []models.EnergyTaxEntry{{From: "2000-01-01", Rate: 0.05}},
		Surcharges:       0.0,
		NetworkTariffAvg: 0.08,
	})

	// Worker consumer price (80 EUR/MWh wholesale stored alongside)
	prices := []models.HourlyPrice{{
		Timestamp:    time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:     0.2076, // Worker consumer price (should be ignored in manual mode)
		WholesaleKWh: 0.08,   // 80 EUR/MWh = 0.08 EUR/kWh
		Unit:         models.UnitKWh,
		Source:       "synctacles",
		Zone:         "NL",
		IsConsumer:   true,
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	assert.True(t, result[0].IsConsumer)
	// Expected: (0.08 + 0.01 + 0.05 + 0.0 + 0.08) × 1.21 = 0.22 × 1.21 = 0.2662
	assert.InDelta(t, 0.2662, result[0].PriceEUR, 0.001)
}

func TestNormalizer_ManualMode_WholesaleInput(t *testing.T) {
	norm := NewNormalizer(nil) // No tax cache needed for manual
	norm.SetPricingMode("manual")
	norm.SetManualTaxProfile(&models.TaxProfile{
		VATRate:          0.19,
		SupplierMarkup:   0.0,
		EnergyTax:        []models.EnergyTaxEntry{{From: "2000-01-01", Rate: 0.02}},
		Surcharges:       0.03,
		NetworkTariffAvg: 0.09,
	})

	// Energy-Charts wholesale (no WholesaleKWh — PriceEUR IS the wholesale)
	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  60.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "DE-LU",
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	assert.True(t, result[0].IsConsumer)
	// wholesale = 0.06, total = (0.06 + 0.0 + 0.02 + 0.03 + 0.09) × 1.19 = 0.20 × 1.19 = 0.238
	assert.InDelta(t, 0.238, result[0].PriceEUR, 0.001)
}

func TestNormalizer_P1Mode_PassThrough(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)
	norm.SetPricingMode("p1_meter")

	// Wholesale prices should pass through in P1 mode (no consumer conversion)
	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "synctacles",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	// Converted to kWh but NOT consumer-normalized
	assert.InDelta(t, 0.08, result[0].PriceEUR, 0.0001)
	assert.False(t, result[0].IsConsumer)
}

func TestNormalizer_ManualMode_NoProfile(t *testing.T) {
	norm := NewNormalizer(nil)
	norm.SetPricingMode("manual")
	// No manual tax profile set — should pass through

	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	assert.InDelta(t, 0.08, result[0].PriceEUR, 0.0001)
	assert.False(t, result[0].IsConsumer)
}
