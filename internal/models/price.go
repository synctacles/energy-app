// Package models defines core data types for the energy addon.
package models

import "time"

// PriceUnit represents the unit of a price value.
type PriceUnit string

const (
	UnitMWh PriceUnit = "EUR/MWh"
	UnitKWh PriceUnit = "EUR/kWh"
)

// HourlyPrice represents a single hour's electricity price.
type HourlyPrice struct {
	Timestamp  time.Time `json:"timestamp"`
	PriceEUR   float64   `json:"price_eur"`
	Unit       PriceUnit `json:"unit"`
	Source     string    `json:"source"`
	Quality    string    `json:"quality"` // "live", "estimated", "cached"
	Zone       string    `json:"zone"`
	IsConsumer bool      `json:"is_consumer"` // true = already consumer price (incl. tax), skip normalization
}

// ToKWh converts the price to EUR/kWh if it's in EUR/MWh.
func (p HourlyPrice) ToKWh() HourlyPrice {
	if p.Unit == UnitMWh {
		p.PriceEUR = p.PriceEUR / 1000.0
		p.Unit = UnitKWh
	}
	return p
}

// ToMWh converts the price to EUR/MWh if it's in EUR/kWh.
func (p HourlyPrice) ToMWh() HourlyPrice {
	if p.Unit == UnitKWh {
		p.PriceEUR = p.PriceEUR * 1000.0
		p.Unit = UnitMWh
	}
	return p
}

// PriceStats holds statistics calculated from a set of hourly prices.
type PriceStats struct {
	Average       float64  `json:"average"`
	Min           float64  `json:"min"`
	Max           float64  `json:"max"`
	CheapestHour  string   `json:"cheapest_hour"`
	ExpensiveHour string   `json:"expensive_hour"`
	Best4Hours    []string `json:"best_4_hours"`
}

// TaxProfile defines the tax components for a bidding zone.
type TaxProfile struct {
	VATRate     float64          `yaml:"vat_rate" json:"vat_rate"`
	Coefficient float64         `yaml:"coefficient" json:"coefficient"` // Markup on wholesale (1.0 = pass-through, 1.05 = 5% markup)
	EnergyTax   []EnergyTaxEntry `yaml:"energy_tax" json:"energy_tax"`
	Surcharges  float64         `yaml:"surcharges" json:"surcharges"`
}

// EffectiveCoefficient returns the coefficient, defaulting to 1.0 if not set.
func (tp *TaxProfile) EffectiveCoefficient() float64 {
	if tp.Coefficient == 0 {
		return 1.0
	}
	return tp.Coefficient
}

// EnergyTaxEntry is a time-bound energy tax rate.
type EnergyTaxEntry struct {
	From string  `yaml:"from" json:"from"` // "2024-01-01"
	Rate float64 `yaml:"rate" json:"rate"` // EUR/kWh
}

// ActiveEnergyTax returns the energy tax rate applicable at the given time.
func (tp *TaxProfile) ActiveEnergyTax(at time.Time) float64 {
	dateStr := at.Format("2006-01-02")
	var active float64
	for _, entry := range tp.EnergyTax {
		if entry.From <= dateStr {
			active = entry.Rate
		}
	}
	return active
}

// WholesaleToConsumer converts a wholesale EUR/kWh price to consumer price.
// Formula: consumer = (wholesale × coefficient + energy_tax + surcharges) × (1 + VAT)
func (tp *TaxProfile) WholesaleToConsumer(wholesaleKWh float64, at time.Time) float64 {
	energyTax := tp.ActiveEnergyTax(at)
	coeff := tp.EffectiveCoefficient()
	return (wholesaleKWh*coeff + energyTax + tp.Surcharges) * (1 + tp.VATRate)
}
