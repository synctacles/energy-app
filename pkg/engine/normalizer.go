// Package engine provides core business logic for price analysis and recommendations.
package engine

import (
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// Normalizer converts wholesale prices to consumer prices using country tax profiles.
type Normalizer struct {
	registry         *models.ZoneRegistry
	coeffOverride    float64 // If > 0, overrides the YAML coefficient for all zones
}

// NewNormalizer creates a normalizer with zone registry for tax lookups.
// coeffOverride overrides the per-country coefficient if > 0 (0 = use YAML default).
func NewNormalizer(registry *models.ZoneRegistry, coeffOverride ...float64) *Normalizer {
	n := &Normalizer{registry: registry}
	if len(coeffOverride) > 0 && coeffOverride[0] > 0 {
		n.coeffOverride = coeffOverride[0]
	}
	return n
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

	// Consumer prices (e.g. EasyEnergy, Frank, Enever) already include
	// VAT, energy tax, and supplier markup — no normalization needed.
	if p.IsConsumer {
		return p
	}

	// Look up tax profile for this zone
	cc, ok := n.registry.GetCountryForZone(p.Zone)
	if !ok {
		return p // No tax profile — return wholesale in kWh
	}

	// Apply coefficient override if set
	tp := cc.TaxProfile
	if n.coeffOverride > 0 {
		tp.Coefficient = n.coeffOverride
	}

	// Apply tax profile: wholesale → consumer
	p.PriceEUR = tp.WholesaleToConsumer(p.PriceEUR, p.Timestamp)
	p.IsConsumer = true
	return p
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
