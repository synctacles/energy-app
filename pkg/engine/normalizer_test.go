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

func TestNormalizer_MWhToConsumer(t *testing.T) {
	// NL tax profile
	nlConfig := &models.CountryConfig{
		Country: "NL",
		Zones: []models.ZoneInfo{{Code: "NL", Country: "NL"}},
		TaxProfile: models.TaxProfile{
			VATRate:    0.21,
			EnergyTax:  []models.EnergyTaxEntry{{From: "2024-01-01", Rate: 0.09161}},
			Surcharges: 0.0,
		},
	}

	registry := models.NewZoneRegistry([]*models.CountryConfig{nlConfig})
	norm := NewNormalizer(registry)

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
	// Expected: (0.08 + 0.09161) * 1.21 = 0.17161 * 1.21 ≈ 0.2076
	assert.InDelta(t, 0.2076, consumer[0].PriceEUR, 0.002)
}

func TestNormalizer_SkipsConsumerPrices(t *testing.T) {
	// NL tax profile — would change price if applied
	nlConfig := &models.CountryConfig{
		Country: "NL",
		Zones:   []models.ZoneInfo{{Code: "NL", Country: "NL"}},
		TaxProfile: models.TaxProfile{
			VATRate:    0.21,
			EnergyTax:  []models.EnergyTaxEntry{{From: "2024-01-01", Rate: 0.09161}},
			Surcharges: 0.0,
		},
	}

	registry := models.NewZoneRegistry([]*models.CountryConfig{nlConfig})
	norm := NewNormalizer(registry)

	// Consumer price from EasyEnergy (already incl. VAT + taxes)
	consumerPrice := 0.2134
	prices := []models.HourlyPrice{{
		Timestamp:  time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:   consumerPrice,
		Unit:       models.UnitKWh,
		Source:     "easyenergy",
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

func TestNormalizer_ConsumerPriceIgnoresCoeffOverride(t *testing.T) {
	nlConfig := &models.CountryConfig{
		Country: "NL",
		Zones:   []models.ZoneInfo{{Code: "NL", Country: "NL"}},
		TaxProfile: models.TaxProfile{
			VATRate:    0.21,
			EnergyTax:  []models.EnergyTaxEntry{{From: "2024-01-01", Rate: 0.09161}},
			Surcharges: 0.0,
		},
	}

	registry := models.NewZoneRegistry([]*models.CountryConfig{nlConfig})
	// Even with a coefficient override, consumer prices should not be touched
	norm := NewNormalizer(registry, 1.10)

	consumerPrice := 0.2134
	prices := []models.HourlyPrice{{
		Timestamp:  time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC),
		PriceEUR:   consumerPrice,
		Unit:       models.UnitKWh,
		Source:     "frank",
		Zone:       "NL",
		IsConsumer: true,
	}}

	result := norm.ToConsumer(prices)
	assert.Equal(t, consumerPrice, result[0].PriceEUR)
}

func TestNormalizer_MixedConsumerAndWholesale(t *testing.T) {
	nlConfig := &models.CountryConfig{
		Country: "NL",
		Zones:   []models.ZoneInfo{{Code: "NL", Country: "NL"}},
		TaxProfile: models.TaxProfile{
			VATRate:    0.21,
			EnergyTax:  []models.EnergyTaxEntry{{From: "2024-01-01", Rate: 0.09161}},
			Surcharges: 0.0,
		},
	}

	registry := models.NewZoneRegistry([]*models.CountryConfig{nlConfig})
	norm := NewNormalizer(registry)

	ts := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	prices := []models.HourlyPrice{
		{Timestamp: ts, PriceEUR: 0.2134, Unit: models.UnitKWh, Source: "easyenergy", Zone: "NL", IsConsumer: true},
		{Timestamp: ts, PriceEUR: 80.0, Unit: models.UnitMWh, Source: "energycharts", Zone: "NL", IsConsumer: false},
	}

	result := norm.ToConsumer(prices)
	assert.Len(t, result, 2)

	// First: consumer price unchanged
	assert.Equal(t, 0.2134, result[0].PriceEUR)

	// Second: wholesale normalized: (0.08 + 0.09161) * 1.21 ≈ 0.2076
	assert.InDelta(t, 0.2076, result[1].PriceEUR, 0.002)
}
