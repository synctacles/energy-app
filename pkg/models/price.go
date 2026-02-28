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
	Timestamp    time.Time `json:"timestamp"`
	PriceEUR     float64   `json:"price_eur"`
	WholesaleKWh float64   `json:"wholesale_kwh,omitempty"` // Raw wholesale EUR/kWh (set when source provides both wholesale + consumer)
	Unit         PriceUnit `json:"unit"`
	Source       string    `json:"source"`
	Quality      string    `json:"quality"` // "live", "estimated", "cached"
	Zone         string    `json:"zone"`
	IsConsumer   bool      `json:"is_consumer"` // true = already consumer price (incl. tax), skip normalization
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
	VATRate          float64          `yaml:"vat_rate" json:"vat_rate"`
	SupplierMarkup   float64         `yaml:"supplier_markup" json:"supplier_markup"`       // Fixed EUR/kWh (NOT percentage) — from calibration
	EnergyTax        []EnergyTaxEntry `yaml:"energy_tax" json:"energy_tax"`
	Surcharges       float64         `yaml:"surcharges" json:"surcharges"`
	NetworkTariffAvg float64         `yaml:"network_tariff_avg" json:"network_tariff_avg"` // EUR/kWh country avg (user can override)
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

// PriceBreakdown shows the detailed composition of a consumer price.
type PriceBreakdown struct {
	Wholesale        float64 `json:"wholesale"`          // Original wholesale price (EUR/kWh)
	SupplierMarkup   float64 `json:"supplier_markup"`    // Fixed EUR/kWh (from calibration)
	EnergyTax        float64 `json:"energy_tax"`         // Energy excise tax (EUR/kWh excl. VAT)
	Surcharges       float64 `json:"surcharges"`         // Additional levies (EUR/kWh excl. VAT)
	NetworkTariff    float64 `json:"network_tariff"`     // Network transport cost (EUR/kWh excl. VAT)
	Subtotal         float64 `json:"subtotal"`           // Sum before VAT
	VATRate          float64 `json:"vat_rate"`           // VAT rate as decimal (0.21 = 21%)
	VATAmount        float64 `json:"vat_amount"`         // VAT amount (EUR/kWh)
	ConsumerTotal    float64 `json:"consumer_total"`     // Final consumer price (EUR/kWh)
}

// CalculateBreakdown returns detailed price breakdown showing all tax components.
// Formula: (wholesale + supplier_markup + energy_tax + surcharges + network_tariff) × (1 + VAT)
func (tp *TaxProfile) CalculateBreakdown(wholesaleKWh float64, at time.Time) PriceBreakdown {
	energyTax := tp.ActiveEnergyTax(at)

	subtotal := wholesaleKWh + tp.SupplierMarkup + energyTax + tp.Surcharges + tp.NetworkTariffAvg
	vatAmount := subtotal * tp.VATRate
	consumerTotal := subtotal * (1 + tp.VATRate)

	return PriceBreakdown{
		Wholesale:      wholesaleKWh,
		SupplierMarkup: tp.SupplierMarkup,
		EnergyTax:      energyTax,
		Surcharges:     tp.Surcharges,
		NetworkTariff:  tp.NetworkTariffAvg,
		Subtotal:       subtotal,
		VATRate:        tp.VATRate,
		VATAmount:      vatAmount,
		ConsumerTotal:  consumerTotal,
	}
}

// WholesaleToConsumer converts a wholesale EUR/kWh price to consumer price.
// Formula: consumer = (wholesale + supplier_markup + energy_tax + surcharges + network_tariff) × (1 + VAT)
func (tp *TaxProfile) WholesaleToConsumer(wholesaleKWh float64, at time.Time) float64 {
	energyTax := tp.ActiveEnergyTax(at)
	return (wholesaleKWh + tp.SupplierMarkup + energyTax + tp.Surcharges + tp.NetworkTariffAvg) * (1 + tp.VATRate)
}

// CacheEntry holds cached prices with provenance metadata.
// Used by the smart cache to preserve original source tier and quality across reboots.
type CacheEntry struct {
	Prices       []HourlyPrice
	OriginalTier int
	FetchedAt    time.Time
}
