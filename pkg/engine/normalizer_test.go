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

func TestNormalizer_ConsumerPriceAppliesSupplierMarkup(t *testing.T) {
	cache := testTaxCache()
	// Supplier markup is pre-VAT, so consumer price += markup × (1 + VAT)
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
	// 0.2134 + 0.005 × (1 + 0.21) = 0.2134 + 0.00605 = 0.21945
	assert.InDelta(t, 0.21945, result[0].PriceEUR, 0.00001)
}

func TestNormalizer_ConsumerPriceNoMarkupWhenZero(t *testing.T) {
	cache := testTaxCache()
	// No supplier markup → consumer price unchanged
	norm := NewNormalizer(cache)

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
	// Expected: (0.08 + 0.01 + 0.05 + 0.0) × 1.21 = 0.14 × 1.21 = 0.1694
	// Network tariff (0.08) excluded — billed separately by grid operator
	assert.InDelta(t, 0.1694, result[0].PriceEUR, 0.001)
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
	// wholesale = 0.06, total = (0.06 + 0.0 + 0.02 + 0.03) × 1.19 = 0.11 × 1.19 = 0.1309
	// Network tariff (0.09) excluded — billed separately by grid operator
	assert.InDelta(t, 0.1309, result[0].PriceEUR, 0.001)
}

func TestNormalizer_P1Mode_NormalizesWholesale(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)
	norm.SetPricingMode("p1_meter")

	// P1/sensor mode now normalizes wholesale → consumer (same as auto mode)
	// This enables chart calibration with supplier-specific deltas
	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "synctacles",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	// Wholesale (0.08 kWh) normalized to consumer via tax cache
	// (0.08 + 0.09161) × 1.21 ≈ 0.2076
	assert.True(t, result[0].IsConsumer)
	assert.InDelta(t, 0.2076, result[0].PriceEUR, 0.01)
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

// ============================================================================
// Pipeline coverage: negative prices, delta application, sensor mode,
// concurrent safety, and zone-loop invariants.
// ============================================================================

func TestNormalizer_ToConsumer_NegativeWholesale(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)

	// Negative wholesale price (common during high wind/solar) must NOT be clamped.
	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 6, 15, 14, 0, 0, 0, time.UTC),
		PriceEUR:  -20.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	assert.True(t, result[0].IsConsumer)
	// (-0.02 + 0 + 0.09161 + 0) × 1.21 = 0.07161 × 1.21 ≈ 0.08665
	assert.InDelta(t, 0.08665, result[0].PriceEUR, 0.002)
	// Consumer price should be lower than a positive wholesale scenario
	assert.Less(t, result[0].PriceEUR, 0.20)
}

func TestNormalizer_ToConsumer_PriceLookupApplied(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)

	// ADR_016: PriceLookup returns the full consumer price directly
	norm.SetPriceLookup(func(t time.Time) (float64, bool) {
		return 0.2758, true // pre-computed consumer price
	}, true)

	ts := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)

	prices := []models.HourlyPrice{{
		Timestamp: ts,
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	assert.True(t, result[0].IsConsumer)
	// Price lookup overrides — uses consumer price directly
	assert.InDelta(t, 0.2758, result[0].PriceEUR, 0.0001)
}

func TestNormalizer_ToConsumer_PriceLookupDoesNotOverrideConsumer(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)

	// ADR_016: PriceLookup must NOT override existing consumer prices (sensor is truth)
	norm.SetPriceLookup(func(t time.Time) (float64, bool) {
		return 0.2255, true
	}, true)

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
	// Consumer price passes through unchanged — price lookup only applies to wholesale
	assert.Equal(t, consumerPrice, result[0].PriceEUR)
}

func TestNormalizer_ToConsumer_SensorMode_ConsumerPricePreserved(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)
	norm.SetPricingMode("external_sensor")

	// Even with supplier-specific price lookup, existing consumer prices are preserved
	// (sensor is the ground truth in sensor mode)
	norm.SetPriceLookup(func(t time.Time) (float64, bool) {
		return 0.30, true
	}, true) // supplierSpecific = true

	consumerPrice := 0.25
	prices := []models.HourlyPrice{{
		Timestamp:  time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:   consumerPrice,
		Unit:       models.UnitKWh,
		Source:     "synctacles",
		Zone:       "NL",
		IsConsumer: true,
	}}

	result := norm.ToConsumer(prices)
	// Sensor consumer price preserved — NOT overridden by price lookup
	assert.Equal(t, consumerPrice, result[0].PriceEUR)
}

func TestNormalizer_ToConsumer_SensorMode_AveragePrice_Skipped(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)
	norm.SetPricingMode("external_sensor")

	// _average price: should be SKIPPED in sensor mode (sensor is more accurate)
	norm.SetPriceLookup(func(t time.Time) (float64, bool) {
		return 0.30, true
	}, false) // supplierSpecific = false (_average)

	consumerPrice := 0.25
	prices := []models.HourlyPrice{{
		Timestamp:  time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:   consumerPrice,
		Unit:       models.UnitKWh,
		Source:     "synctacles",
		Zone:       "NL",
		IsConsumer: true,
	}}

	result := norm.ToConsumer(prices)
	// _average skipped in sensor mode → price unchanged
	assert.Equal(t, consumerPrice, result[0].PriceEUR)
}

func TestNormalizer_ToConsumer_SensorMode_FixedMarkup_Skipped(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache, 0.005) // fixed markup
	norm.SetPricingMode("p1_meter")

	consumerPrice := 0.25
	prices := []models.HourlyPrice{{
		Timestamp:  time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:   consumerPrice,
		Unit:       models.UnitKWh,
		Source:     "synctacles",
		Zone:       "NL",
		IsConsumer: true,
	}}

	result := norm.ToConsumer(prices)
	// Fixed markup skipped in sensor mode → unchanged
	assert.Equal(t, consumerPrice, result[0].PriceEUR)
}

func TestNormalizer_ToConsumer_PriceLookupReturnsLowPrice(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)

	// Even a very low price should be used directly
	norm.SetPriceLookup(func(t time.Time) (float64, bool) {
		return 0.05, true
	}, true)

	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	assert.InDelta(t, 0.05, result[0].PriceEUR, 0.0001)
	assert.True(t, result[0].IsConsumer)
}

func TestNormalizer_ToConsumer_NilPriceLookup(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)
	// No SetPriceLookup call — priceLookup is nil

	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	// No price lookup → falls back to tax-based normalization
	// (0.08 + 0 + 0.09161 + 0) × 1.21 ≈ 0.2076
	assert.InDelta(t, 0.2076, result[0].PriceEUR, 0.002)
	assert.True(t, result[0].IsConsumer)
}

func TestNormalizer_ToConsumer_PriceNotAvailable_FallsBackToMarkup(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache, 0.003) // fixed markup as fallback

	// Price lookup exists but returns false for this hour
	norm.SetPriceLookup(func(t time.Time) (float64, bool) {
		return 0, false
	}, true)

	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	// Price unavailable → falls back to tax-based normalization with markup
	// (0.08 + 0.003 + 0.09161 + 0) × 1.21 ≈ 0.2113
	assert.InDelta(t, 0.2113, result[0].PriceEUR, 0.002)
}

func TestNormalizer_SetPricingMode_Concurrent(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)

	modes := []string{"auto", "manual", "p1_meter", "external_sensor", "meter_tariff"}
	done := make(chan struct{})

	// 100 goroutines setting modes concurrently
	for i := 0; i < 100; i++ {
		go func(idx int) {
			norm.SetPricingMode(modes[idx%len(modes)])
			_ = norm.TaxSource() // concurrent read
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	// If we get here without a race condition, the test passes.
	// Run with -race flag to verify.
}

func TestNormalizer_SetSupplierMarkup_Concurrent(t *testing.T) {
	cache := testTaxCache()
	norm := NewNormalizer(cache)

	done := make(chan struct{})

	// 100 goroutines writing markups while others normalize
	for i := 0; i < 50; i++ {
		go func(idx int) {
			norm.SetSupplierMarkup(float64(idx) * 0.001)
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 50; i++ {
		go func() {
			prices := []models.HourlyPrice{{
				Timestamp: time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
				PriceEUR:  80.0,
				Unit:      models.UnitMWh,
				Source:    "energycharts",
				Zone:      "NL",
			}}
			_ = norm.ToConsumer(prices) // concurrent read + normalize
			done <- struct{}{}
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestNormalizer_FindCheapestN_WithNegative(t *testing.T) {
	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	// Mix of negative and positive prices
	values := []float64{-0.05, 0.10, -0.02, 0.20, 0.05, 0.30, -0.08, 0.15}
	prices := makeHourlyPrices(date, values)

	result := findCheapestN(prices, 3)
	assert.Len(t, result, 3)
	// Cheapest 3: hour 6 (-0.08), hour 0 (-0.05), hour 2 (-0.02), sorted by time
	assert.Equal(t, []string{"00:00", "02:00", "06:00"}, result)
}

// ============================================================================
// Zone-loop tests: structural invariants across ALL registered zones
// ============================================================================

func TestNormalizer_AllZones_TaxPositive(t *testing.T) {
	configs, err := loadAllCountryConfigs()
	if err != nil {
		t.Skipf("could not load country configs: %v", err)
	}
	reg := models.NewZoneRegistry(configs)

	for _, zoneCode := range reg.AllZones() {
		defaults := reg.GetTaxDefaults(zoneCode)
		if defaults == nil {
			continue // non-wholesale zones may not have tax defaults
		}
		if defaults.VATRate <= 0 {
			t.Errorf("zone %s: VAT rate should be positive, got %f", zoneCode, defaults.VATRate)
		}
	}
}

func TestNormalizer_AllZones_ConsumerGTWholesale(t *testing.T) {
	configs, err := loadAllCountryConfigs()
	if err != nil {
		t.Skipf("could not load country configs: %v", err)
	}
	reg := models.NewZoneRegistry(configs)

	ts := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	wholesale := 0.08 // 80 EUR/MWh — typical positive wholesale price

	for _, zoneCode := range reg.AllZones() {
		defaults := reg.GetTaxDefaults(zoneCode)
		if defaults == nil {
			continue
		}

		tp := models.TaxProfile{
			VATRate:    defaults.VATRate,
			EnergyTax:  []models.EnergyTaxEntry{{From: "2000-01-01", Rate: defaults.EnergyTax}},
			Surcharges: defaults.Surcharges,
		}
		consumer := tp.WholesaleToConsumer(wholesale, ts)
		if consumer <= wholesale {
			t.Errorf("zone %s: consumer (%f) should be > wholesale (%f) for positive prices",
				zoneCode, consumer, wholesale)
		}
	}
}

func TestNormalizer_AllZones_NonWholesale_HasPresetsOrFixed(t *testing.T) {
	configs, err := loadAllCountryConfigs()
	if err != nil {
		t.Skipf("could not load country configs: %v", err)
	}
	reg := models.NewZoneRegistry(configs)

	var missing []string
	for _, zoneCode := range reg.AllZones() {
		z, ok := reg.GetZone(zoneCode)
		if !ok || z.HasWholesale() {
			continue // skip wholesale zones
		}
		// Non-wholesale zones should have regulated tariffs or TOU presets
		hasTOU := len(reg.GetTOUPresets(zoneCode)) > 0
		hasRegulated := z.RegulatedTariffs != nil
		if !hasTOU && !hasRegulated {
			missing = append(missing, zoneCode)
		}
	}
	if len(missing) > 0 {
		t.Logf("non-wholesale zones without TOU presets or regulated tariffs: %v (informational — not all zones have presets yet)", missing)
	}
}

func TestNormalizer_EmbeddedFallback(t *testing.T) {
	// No tax cache (simulating Worker unreachable), but embedded zone registry available.
	configs, err := loadAllCountryConfigs()
	if err != nil {
		t.Skipf("could not load country configs: %v", err)
	}
	reg := models.NewZoneRegistry(configs)

	norm := NewNormalizer(nil) // No Worker cache
	norm.SetZoneRegistry(reg)

	prices := []models.HourlyPrice{{
		Timestamp: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
		PriceEUR:  80.0,
		Unit:      models.UnitMWh,
		Source:    "energycharts",
		Zone:      "NL",
	}}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 1)
	assert.True(t, result[0].IsConsumer)
	// Should use embedded tax data — consumer price > 0.08 (wholesale)
	assert.Greater(t, result[0].PriceEUR, 0.08)
	assert.Equal(t, "embedded", norm.TaxSource())
}

// loadAllCountryConfigs loads country configs via the countries package.
func loadAllCountryConfigs() ([]*models.CountryConfig, error) {
	// Import the countries package dynamically isn't possible in Go,
	// so we use the same embed pattern. This test file is in package engine,
	// so we construct a minimal registry inline for the zone-loop tests.
	// We use the countries.LoadAll function via a test helper.
	return countriesLoadAll()
}
