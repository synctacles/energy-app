package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHourlyPrice_ToKWh(t *testing.T) {
	p := HourlyPrice{PriceEUR: 100.0, Unit: UnitMWh}
	converted := p.ToKWh()
	assert.Equal(t, UnitKWh, converted.Unit)
	assert.InDelta(t, 0.1, converted.PriceEUR, 0.0001)

	// Already kWh — no change
	p2 := HourlyPrice{PriceEUR: 0.1, Unit: UnitKWh}
	assert.Equal(t, 0.1, p2.ToKWh().PriceEUR)
}

func TestHourlyPrice_ToMWh(t *testing.T) {
	p := HourlyPrice{PriceEUR: 0.1, Unit: UnitKWh}
	converted := p.ToMWh()
	assert.Equal(t, UnitMWh, converted.Unit)
	assert.InDelta(t, 100.0, converted.PriceEUR, 0.01)

	// Already MWh — no change
	p2 := HourlyPrice{PriceEUR: 100.0, Unit: UnitMWh}
	assert.InDelta(t, 100.0, p2.ToMWh().PriceEUR, 0.01)
}

func TestTaxProfile_ActiveEnergyTax(t *testing.T) {
	tp := TaxProfile{
		EnergyTax: []EnergyTaxEntry{
			{From: "2024-01-01", Rate: 0.10880},
			{From: "2026-01-01", Rate: 0.09161},
		},
	}

	// Before any entry
	assert.Equal(t, 0.0, tp.ActiveEnergyTax(time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)))
	// First period (2024 — NL EB excl. BTW)
	assert.InDelta(t, 0.10880, tp.ActiveEnergyTax(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)), 0.00001)
	// Second period (2026 — NL EB excl. BTW, verlaagd)
	assert.InDelta(t, 0.09161, tp.ActiveEnergyTax(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)), 0.00001)
}

func TestTaxProfile_WholesaleToConsumer_NL(t *testing.T) {
	// NL tax profile: VAT 21%, energy tax €0.09161/kWh excl BTW (2026)
	tp := TaxProfile{
		VATRate: 0.21,
		EnergyTax: []EnergyTaxEntry{
			{From: "2024-01-01", Rate: 0.10880},
			{From: "2026-01-01", Rate: 0.09161},
		},
		Surcharges: 0.0,
	}

	at := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	wholesale := 0.08 // EUR/kWh wholesale

	consumer := tp.WholesaleToConsumer(wholesale, at)
	// Expected: (0.08 + 0 + 0.09161 + 0 + 0) × 1.21 = 0.17161 × 1.21 ≈ 0.2076
	assert.InDelta(t, 0.2076, consumer, 0.002)
}

func TestTaxProfile_WholesaleToConsumer_WithSupplierMarkup(t *testing.T) {
	// DE profile with €0.005/kWh supplier markup
	tp := TaxProfile{
		VATRate:        0.19,
		SupplierMarkup: 0.005,
		EnergyTax: []EnergyTaxEntry{
			{From: "2024-01-01", Rate: 0.02050},
		},
		Surcharges: 0.02946,
	}

	at := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	wholesale := 0.08

	consumer := tp.WholesaleToConsumer(wholesale, at)
	// Expected: (0.08 + 0.005 + 0.02050 + 0.02946 + 0) × 1.19 = 0.13496 × 1.19 ≈ 0.16060
	assert.InDelta(t, 0.16060, consumer, 0.001)
}

func TestTaxProfile_CalculateBreakdown(t *testing.T) {
	tp := TaxProfile{
		VATRate:          0.21,
		SupplierMarkup:   0.003,
		EnergyTax:        []EnergyTaxEntry{{From: "2024-01-01", Rate: 0.09161}},
		Surcharges:       0.0,
		NetworkTariffAvg: 0.095,
	}

	at := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	bd := tp.CalculateBreakdown(0.08, at)

	assert.InDelta(t, 0.08, bd.Wholesale, 0.0001)
	assert.InDelta(t, 0.003, bd.SupplierMarkup, 0.0001)
	assert.InDelta(t, 0.09161, bd.EnergyTax, 0.0001)
	assert.InDelta(t, 0.095, bd.NetworkTariff, 0.0001) // informational only, NOT in subtotal
	// Subtotal: 0.08 + 0.003 + 0.09161 + 0 = 0.17461 (network tariff excluded — billed separately)
	assert.InDelta(t, 0.17461, bd.Subtotal, 0.001)
	// Consumer: 0.17461 × 1.21 ≈ 0.21128
	assert.InDelta(t, 0.21128, bd.ConsumerTotal, 0.002)
}
