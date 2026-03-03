// Package engine provides core business logic for price analysis and recommendations.
package engine

import (
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// Normalizer converts wholesale prices to consumer prices.
// Behavior depends on PricingMode:
//   - "auto":     Worker consumer prices pass through; wholesale normalized via tax cache
//   - "manual":   All prices normalized using user-defined ManualTaxProfile
//   - "p1_meter": All prices pass through (consumer price comes from HA sensor, not here)
//   - "enever":   Same as auto (Enever prices are already consumer)
type Normalizer struct {
	taxCache              *TaxProfileCache       // Worker-provided tax profiles (keyed by zone)
	zoneRegistry          *models.ZoneRegistry   // Embedded fallback tax defaults
	supplierMarkupOverride float64               // User calibration override (EUR/kWh)
	pricingMode           string                 // "auto", "manual", "p1_meter", "enever"
	manualTaxProfile      *models.TaxProfile     // User-defined components for manual mode
	lastTaxSource         string                 // "worker", "embedded", "none" — for degraded banner
}

// NewNormalizer creates a normalizer backed by the Worker tax profile cache.
// supplierMarkupOverride overrides the per-zone supplier_markup if > 0 (0 = use Worker default).
func NewNormalizer(taxCache *TaxProfileCache, supplierMarkupOverride ...float64) *Normalizer {
	n := &Normalizer{taxCache: taxCache, pricingMode: "auto", lastTaxSource: "none"}
	if len(supplierMarkupOverride) > 0 && supplierMarkupOverride[0] > 0 {
		n.supplierMarkupOverride = supplierMarkupOverride[0]
	}
	return n
}

// SetZoneRegistry sets the zone registry for embedded tax fallback.
func (n *Normalizer) SetZoneRegistry(reg *models.ZoneRegistry) {
	n.zoneRegistry = reg
}

// SetPricingMode sets the active pricing mode.
func (n *Normalizer) SetPricingMode(mode string) {
	n.pricingMode = mode
}

// SetManualTaxProfile sets user-defined tax components for manual mode.
func (n *Normalizer) SetManualTaxProfile(tp *models.TaxProfile) {
	n.manualTaxProfile = tp
}

// SetSupplierMarkup updates the supplier markup override at runtime.
func (n *Normalizer) SetSupplierMarkup(markup float64) {
	n.supplierMarkupOverride = markup
}

// TaxSource returns the tax data source used for the last normalization.
// "worker" = live Worker data, "embedded" = fallback defaults, "none" = no tax data.
func (n *Normalizer) TaxSource() string {
	return n.lastTaxSource
}

// ToConsumer converts a slice of wholesale prices to consumer prices using the zone's tax profile.
// Prices already marked as consumer (IsConsumer=true) are only unit-converted to kWh.
// If the zone has no tax profile, prices are returned as-is in kWh.
func (n *Normalizer) ToConsumer(prices []models.HourlyPrice) []models.HourlyPrice {
	result := make([]models.HourlyPrice, len(prices))
	for i, p := range prices {
		result[i] = n.normalizeOne(p)
	}
	return result
}

func (n *Normalizer) normalizeOne(p models.HourlyPrice) models.HourlyPrice {
	// First convert to EUR/kWh
	p = p.ToKWh()

	switch n.pricingMode {
	case "manual":
		return n.normalizeManual(p)
	case "p1_meter", "meter_tariff":
		// Meter tariff mode: consumer price comes from HA sensor, not normalizer.
		// Wholesale prices pass through for GO/WAIT/AVOID relative calculations.
		return p
	default:
		// "auto" and "enever": consumer prices pass through, wholesale normalized via tax cache.
		return n.normalizeAuto(p)
	}
}

// normalizeAuto handles auto/enever mode: consumer prices pass through, wholesale → tax cache.
func (n *Normalizer) normalizeAuto(p models.HourlyPrice) models.HourlyPrice {
	if p.IsConsumer {
		// Consumer prices already include taxes — no normalization needed.
		// Update lastTaxSource so the degraded banner doesn't trigger.
		if n.lastTaxSource == "none" {
			n.lastTaxSource = "consumer"
		}
		// Apply supplier markup if set: markup is pre-VAT, so add markup × (1 + VAT).
		if n.supplierMarkupOverride > 0 {
			vatRate := n.vatRateForZone(p.Zone)
			p.PriceEUR += n.supplierMarkupOverride * (1 + vatRate)
		}
		return p
	}

	// Try Worker tax cache first (live data)
	var override *WorkerTaxOverride
	if n.taxCache != nil {
		override = n.taxCache.Get(p.Zone)
	}

	if override != nil {
		n.lastTaxSource = "worker"
		tp := models.TaxProfile{
			VATRate:          override.VATRate,
			SupplierMarkup:   override.SupplierMarkup,
			EnergyTax:        []models.EnergyTaxEntry{{From: "2000-01-01", Rate: override.EnergyTax}},
			Surcharges:       override.Surcharges,
			NetworkTariffAvg: override.NetworkTariffAvg,
		}
		if n.supplierMarkupOverride > 0 {
			tp.SupplierMarkup = n.supplierMarkupOverride
		}
		p.PriceEUR = tp.WholesaleToConsumer(p.PriceEUR, p.Timestamp)
		p.IsConsumer = true
		return p
	}

	// Fallback: embedded tax defaults from zone registry
	if n.zoneRegistry != nil {
		defaults := n.zoneRegistry.GetTaxDefaults(p.Zone)
		if defaults != nil {
			n.lastTaxSource = "embedded"
			tp := models.TaxProfile{
				VATRate:          defaults.VATRate,
				EnergyTax:        []models.EnergyTaxEntry{{From: "2000-01-01", Rate: defaults.EnergyTax}},
				Surcharges:       defaults.Surcharges,
				NetworkTariffAvg: defaults.NetworkTariffAvg,
			}
			if n.supplierMarkupOverride > 0 {
				tp.SupplierMarkup = n.supplierMarkupOverride
			}
			p.PriceEUR = tp.WholesaleToConsumer(p.PriceEUR, p.Timestamp)
			p.IsConsumer = true
			return p
		}
	}

	n.lastTaxSource = "none"
	return p
}

// normalizeManual handles manual mode: always use wholesale + user-defined tax components.
func (n *Normalizer) normalizeManual(p models.HourlyPrice) models.HourlyPrice {
	if n.manualTaxProfile == nil {
		return p
	}

	// Get wholesale price: prefer stored wholesale, else use current price (already kWh)
	wholesale := p.PriceEUR
	if p.IsConsumer && p.WholesaleKWh > 0 {
		wholesale = p.WholesaleKWh
	}

	p.PriceEUR = n.manualTaxProfile.WholesaleToConsumer(wholesale, p.Timestamp)
	p.IsConsumer = true
	return p
}

// vatRateForZone returns the VAT rate for a zone from cache or embedded defaults.
func (n *Normalizer) vatRateForZone(zone string) float64 {
	if n.taxCache != nil {
		if override := n.taxCache.Get(zone); override != nil {
			return override.VATRate
		}
	}
	if n.zoneRegistry != nil {
		if defaults := n.zoneRegistry.GetTaxDefaults(zone); defaults != nil {
			return defaults.VATRate
		}
	}
	return 0
}

// CalcStats computes price statistics from a set of hourly prices.
// Expects prices in EUR/kWh.
func CalcStats(prices []models.HourlyPrice) models.PriceStats {
	if len(prices) == 0 {
		return models.PriceStats{}
	}

	var sum float64
	minPrice := prices[0].PriceEUR
	maxPrice := prices[0].PriceEUR
	var minHour, maxHour time.Time
	minHour = prices[0].Timestamp
	maxHour = prices[0].Timestamp

	for _, p := range prices {
		sum += p.PriceEUR
		if p.PriceEUR < minPrice {
			minPrice = p.PriceEUR
			minHour = p.Timestamp
		}
		if p.PriceEUR > maxPrice {
			maxPrice = p.PriceEUR
			maxHour = p.Timestamp
		}
	}

	avg := sum / float64(len(prices))

	// Find best 4 hours (cheapest)
	best4 := findCheapestN(prices, 4)

	return models.PriceStats{
		Average:       avg,
		Min:           minPrice,
		Max:           maxPrice,
		CheapestHour:  minHour.Format("15:04"),
		ExpensiveHour: maxHour.Format("15:04"),
		Best4Hours:    best4,
	}
}

// findCheapestN returns the N cheapest hours as "HH:MM" strings, sorted by time.
func findCheapestN(prices []models.HourlyPrice, n int) []string {
	if n > len(prices) {
		n = len(prices)
	}

	// Copy and sort by price
	type indexed struct {
		price float64
		ts    time.Time
	}
	items := make([]indexed, len(prices))
	for i, p := range prices {
		items[i] = indexed{price: p.PriceEUR, ts: p.Timestamp}
	}

	// Simple selection of cheapest N
	for i := 0; i < n; i++ {
		minIdx := i
		for j := i + 1; j < len(items); j++ {
			if items[j].price < items[minIdx].price {
				minIdx = j
			}
		}
		items[i], items[minIdx] = items[minIdx], items[i]
	}

	// Sort the cheapest N by time
	cheapest := items[:n]
	for i := 0; i < len(cheapest); i++ {
		for j := i + 1; j < len(cheapest); j++ {
			if cheapest[j].ts.Before(cheapest[i].ts) {
				cheapest[i], cheapest[j] = cheapest[j], cheapest[i]
			}
		}
	}

	result := make([]string, n)
	for i, c := range cheapest {
		result[i] = c.ts.Format("15:04")
	}
	return result
}
