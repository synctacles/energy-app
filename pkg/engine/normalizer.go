// Package engine provides core business logic for price analysis and recommendations.
package engine

import (
	"sync"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// PriceLookup returns the per-hour consumer price for a given timestamp.
// Returns (consumer_kwh, true) when available, (0, false) when not.
// Source: GET /api/v1/energy/supplier-prices (ADR_016).
type PriceLookup func(t time.Time) (float64, bool)

// Normalizer converts wholesale prices to consumer prices.
// Behavior depends on PricingMode:
//   - "auto":     Worker consumer prices pass through; wholesale normalized via tax cache
//   - "manual":   All prices normalized using user-defined ManualTaxProfile
//   - "p1_meter": All prices pass through (consumer price comes from HA sensor, not here)
type Normalizer struct {
	mu                    sync.RWMutex
	taxCache              *TaxProfileCache       // Worker-provided tax profiles (keyed by zone)
	zoneRegistry          *models.ZoneRegistry   // Embedded fallback tax defaults
	supplierMarkupOverride float64               // User calibration override (EUR/kWh)
	pricingMode           string                 // "auto", "manual", "p1_meter"
	manualTaxProfile      *models.TaxProfile     // User-defined components for manual mode
	lastTaxSource         string                 // "worker", "embedded", "none" — for degraded banner
	priceLookup           PriceLookup            // ADR_016: per-hour consumer price (replaces delta reconstruction)
	priceIsSupplierSpecific bool                 // true = from known supplier, false = _average
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
	n.mu.Lock()
	defer n.mu.Unlock()
	n.zoneRegistry = reg
}

// SetPricingMode sets the active pricing mode.
func (n *Normalizer) SetPricingMode(mode string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.pricingMode = mode
}

// SetManualTaxProfile sets user-defined tax components for manual mode.
func (n *Normalizer) SetManualTaxProfile(tp *models.TaxProfile) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.manualTaxProfile = tp
}

// SetSupplierMarkup updates the supplier markup override at runtime.
func (n *Normalizer) SetSupplierMarkup(markup float64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.supplierMarkupOverride = markup
}

// SetPriceLookup sets the per-hour consumer price lookup function (ADR_016).
// When set, prices from the Worker are used directly — no delta reconstruction.
// supplierSpecific indicates whether prices are from a known supplier (true) or _average (false).
func (n *Normalizer) SetPriceLookup(fn PriceLookup, supplierSpecific bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.priceLookup = fn
	n.priceIsSupplierSpecific = supplierSpecific
}

// TaxSource returns the tax data source used for the last normalization.
// "worker" = live Worker data, "embedded" = fallback defaults, "none" = no tax data.
func (n *Normalizer) TaxSource() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.lastTaxSource
}

// normalizerSnapshot holds a point-in-time copy of mutable Normalizer fields.
// Used to avoid holding the lock during external calls (e.g., deltaLookup).
type normalizerSnapshot struct {
	taxCache               *TaxProfileCache
	zoneRegistry           *models.ZoneRegistry
	supplierMarkupOverride float64
	pricingMode            string
	manualTaxProfile       *models.TaxProfile
	priceLookup            PriceLookup
	priceIsSupplierSpecific bool
}

func (n *Normalizer) snapshot() normalizerSnapshot {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return normalizerSnapshot{
		taxCache:               n.taxCache,
		zoneRegistry:           n.zoneRegistry,
		supplierMarkupOverride: n.supplierMarkupOverride,
		pricingMode:            n.pricingMode,
		manualTaxProfile:       n.manualTaxProfile,
		priceLookup:            n.priceLookup,
		priceIsSupplierSpecific: n.priceIsSupplierSpecific,
	}
}

// ToConsumer converts a slice of wholesale prices to consumer prices using the zone's tax profile.
// Prices already marked as consumer (IsConsumer=true) are only unit-converted to kWh.
// If the zone has no tax profile, prices are returned as-is in kWh.
func (n *Normalizer) ToConsumer(prices []models.HourlyPrice) []models.HourlyPrice {
	snap := n.snapshot()
	result := make([]models.HourlyPrice, len(prices))
	for i, p := range prices {
		result[i] = n.normalizeOne(snap, p)
	}
	return result
}

func (n *Normalizer) normalizeOne(snap normalizerSnapshot, p models.HourlyPrice) models.HourlyPrice {
	// First convert to EUR/kWh
	p = p.ToKWh()

	switch snap.pricingMode {
	case "manual":
		return n.normalizeManual(snap, p)
	case "external_sensor", "p1_meter", "meter_tariff":
		// External sensor mode: chart uses Worker prices + supplier delta.
		// Run through normalizeAuto for delta application on consumer prices.
		return n.normalizeAuto(snap, p)
	default:
		// "auto": consumer prices pass through, wholesale normalized via tax cache.
		return n.normalizeAuto(snap, p)
	}
}

// setLastTaxSource updates lastTaxSource under write lock.
func (n *Normalizer) setLastTaxSource(src string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastTaxSource = src
}

// setLastTaxSourceIfNone updates lastTaxSource only if currently "none".
func (n *Normalizer) setLastTaxSourceIfNone(src string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.lastTaxSource == "none" {
		n.lastTaxSource = src
	}
}

// normalizeAuto handles auto mode: consumer prices pass through, wholesale → tax cache.
// ADR_016: When priceLookup is available, use pre-computed consumer price directly.
func (n *Normalizer) normalizeAuto(snap normalizerSnapshot, p models.HourlyPrice) models.HourlyPrice {
	// ADR_016: If we have a pre-computed consumer price for this hour, use it directly.
	// This replaces the delta reconstruction (ADR_010). The Worker already computed:
	// consumer_kwh = (wholesale + markup + energy_tax) × (1 + VAT)
	isSensorMode := snap.pricingMode == "external_sensor" || snap.pricingMode == "p1_meter" || snap.pricingMode == "meter_tariff"
	if snap.priceLookup != nil && (!isSensorMode || snap.priceIsSupplierSpecific) {
		if consumerPrice, ok := snap.priceLookup(p.Timestamp); ok {
			p.PriceEUR = consumerPrice
			p.IsConsumer = true
			n.setLastTaxSource("worker")
			return p
		}
	}

	// Fallback: if consumer price not available in cache, use existing tax-based normalization
	if p.IsConsumer {
		n.setLastTaxSourceIfNone("consumer")
		if !isSensorMode && snap.supplierMarkupOverride > 0 {
			vatRate := n.vatRateForZone(snap, p.Zone)
			p.PriceEUR += snap.supplierMarkupOverride * (1 + vatRate)
		}
		return p
	}

	// Try Worker tax cache first (live data)
	var override *WorkerTaxOverride
	if snap.taxCache != nil {
		override = snap.taxCache.Get(p.Zone)
	}

	if override != nil {
		n.setLastTaxSource("worker")
		tp := models.TaxProfile{
			VATRate:          override.VATRate,
			SupplierMarkup:   override.SupplierMarkup,
			EnergyTax:        []models.EnergyTaxEntry{{From: "2000-01-01", Rate: override.EnergyTax}},
			Surcharges:       override.Surcharges,
			NetworkTariffAvg: override.NetworkTariffAvg,
		}
		if snap.supplierMarkupOverride > 0 {
			tp.SupplierMarkup = snap.supplierMarkupOverride
		}
		p.PriceEUR = tp.WholesaleToConsumer(p.PriceEUR, p.Timestamp)
		p.IsConsumer = true
		return p
	}

	// Fallback: embedded tax defaults from zone registry
	if snap.zoneRegistry != nil {
		defaults := snap.zoneRegistry.GetTaxDefaults(p.Zone)
		if defaults != nil {
			n.setLastTaxSource("embedded")
			tp := models.TaxProfile{
				VATRate:          defaults.VATRate,
				EnergyTax:        []models.EnergyTaxEntry{{From: "2000-01-01", Rate: defaults.EnergyTax}},
				Surcharges:       defaults.Surcharges,
				NetworkTariffAvg: defaults.NetworkTariffAvg,
			}
			if snap.supplierMarkupOverride > 0 {
				tp.SupplierMarkup = snap.supplierMarkupOverride
			}
			p.PriceEUR = tp.WholesaleToConsumer(p.PriceEUR, p.Timestamp)
			p.IsConsumer = true
			return p
		}
	}

	n.setLastTaxSource("none")
	return p
}

// normalizeManual handles manual mode: always use wholesale + user-defined tax components.
func (n *Normalizer) normalizeManual(snap normalizerSnapshot, p models.HourlyPrice) models.HourlyPrice {
	if snap.manualTaxProfile == nil {
		return p
	}

	// Get wholesale price: prefer stored wholesale, else use current price (already kWh)
	wholesale := p.PriceEUR
	if p.IsConsumer && p.WholesaleKWh > 0 {
		wholesale = p.WholesaleKWh
	}

	p.PriceEUR = snap.manualTaxProfile.WholesaleToConsumer(wholesale, p.Timestamp)
	p.IsConsumer = true
	return p
}

// vatRateForZone returns the VAT rate for a zone from cache or embedded defaults.
func (n *Normalizer) vatRateForZone(snap normalizerSnapshot, zone string) float64 {
	if snap.taxCache != nil {
		if override := snap.taxCache.Get(zone); override != nil {
			return override.VATRate
		}
	}
	if snap.zoneRegistry != nil {
		if defaults := snap.zoneRegistry.GetTaxDefaults(zone); defaults != nil {
			return defaults.VATRate
		}
	}
	return 0
}

// DetectSlotDuration returns the duration between consecutive price entries.
// Returns time.Hour for hourly, 15*time.Minute for PT15M, 30*time.Minute for PT30M.
// Falls back to time.Hour if unable to determine (fewer than 2 entries).
func DetectSlotDuration(prices []models.HourlyPrice) time.Duration {
	if len(prices) < 2 {
		return time.Hour
	}
	// Find two earliest timestamps to determine gap
	min1, min2 := prices[0].Timestamp, prices[1].Timestamp
	if min2.Before(min1) {
		min1, min2 = min2, min1
	}
	for _, p := range prices[2:] {
		if p.Timestamp.Before(min1) {
			min2 = min1
			min1 = p.Timestamp
		} else if p.Timestamp.Before(min2) {
			min2 = p.Timestamp
		}
	}
	gap := min2.Sub(min1)
	if gap > 0 && gap <= time.Hour {
		return gap
	}
	return time.Hour
}

// CalcStats computes price statistics from a set of prices (hourly or PT15).
// Expects prices in EUR/kWh. "Best 4" scales with resolution: 4 slots for PT60,
// 16 slots for PT15 — so it always represents the cheapest 4 hours of the day.
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

	// Find best 4 hours (cheapest) — scale N by slots-per-hour for PT15/PT30
	slotDur := DetectSlotDuration(prices)
	slotsPerHour := int(time.Hour / slotDur) // 1 for PT60, 4 for PT15
	best4 := findCheapestN(prices, 4*slotsPerHour)

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
