// Package collector — SynctaclesAPI is the primary price source (Tier 0).
// It fetches wholesale + consumer prices from the Synctacles Energy Data Worker
// at energy-data.synctacles.com. Consumer prices include taxes, surcharges,
// and network tariff averages per country.
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

const defaultBaseURL = "https://energy-data.synctacles.com"

// SynctaclesAPI fetches prices from the Synctacles Energy Data Worker.
// Primary source (Tier 0) — returns wholesale + consumer prices for all active zones.
type SynctaclesAPI struct {
	BaseURL string // defaults to energy-data.synctacles.com

	mu             sync.RWMutex
	lastTaxProfile *CachedTaxProfile
	lastStatus     string // day_ahead_status from last response
}

// CachedTaxProfile holds the tax profile returned by the Worker for version-based caching.
// Field names match the Worker /api/v1/energy/prices response (CC_INSTRUCTION §3).
type CachedTaxProfile struct {
	Version        string   `json:"version"`
	VatPct         float64  `json:"vat_pct"`
	EnergyTaxKWh   float64  `json:"energy_tax_kwh"`
	SurchargesKWh  float64  `json:"surcharges_kwh"`
	NetworkCostKWh *float64 `json:"network_cost_kwh"` // nil = unknown (consumer_price_estimated)
	ValidFrom      string   `json:"valid_from"`
}

func (s *SynctaclesAPI) Name() string     { return "synctacles" }
func (s *SynctaclesAPI) RequiresKey() bool { return false }

func (s *SynctaclesAPI) baseURL() string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	return defaultBaseURL
}

// LastTaxProfile returns the most recent tax profile from the Worker.
func (s *SynctaclesAPI) LastTaxProfile() *CachedTaxProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.lastTaxProfile == nil {
		return nil
	}
	cp := *s.lastTaxProfile
	return &cp
}

// LastDayAheadStatus returns the day_ahead_status from the last /prices response.
func (s *SynctaclesAPI) LastDayAheadStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastStatus
}

// Zones returns all supported zones. The Worker dynamically controls this via
// the is_active flag, but the collector accepts any zone and lets the Worker
// return 404/empty if the zone is inactive.
func (s *SynctaclesAPI) Zones() []string {
	return []string{
		"NL", "DE-LU", "AT", "BE", "FR", "GB",
		"CH", "CZ", "HU", "PL", "SI",
		"DK1", "DK2", "ES", "PT", "FI",
		"IT-NORTH",
		"NO1", "NO2", "NO3", "NO4", "NO5",
		"SE1", "SE2", "SE3", "SE4",
		"EE", "LT", "LV", "GR", "HR", "RO", "SK", "BG",
		"IE-SEM", "ME", "MK", "RS",
	}
}

// Wire format matching the Worker /prices response (ADR_008)
type workerPriceResponse struct {
	Zone              string             `json:"zone"`
	Resolution        string             `json:"resolution"`
	Currency          string             `json:"currency"`
	Source            string             `json:"source"`
	DayAheadStatus    string             `json:"day_ahead_status"`
	Prices            []workerPriceEntry `json:"prices"`
	TaxProfileVersion string             `json:"tax_profile_version,omitempty"`
	TaxProfile        *workerTaxProfile  `json:"tax_profile,omitempty"`
}

type workerPriceEntry struct {
	Timestamp              string   `json:"timestamp"`
	Price                  float64  `json:"price"`                             // EUR/MWh (wholesale)
	ConsumerPrice          *float64 `json:"consumer_price"`                    // EUR/kWh (with taxes)
	ConsumerPriceEstimated bool     `json:"consumer_price_estimated,omitempty"` // true when network_cost_kwh unknown
}

type workerTaxProfile struct {
	VatPct         float64  `json:"vat_pct"`
	EnergyTaxKWh   float64  `json:"energy_tax_kwh"`
	SurchargesKWh  float64  `json:"surcharges_kwh"`
	NetworkCostKWh *float64 `json:"network_cost_kwh"` // nil = unknown
	ValidFrom      string   `json:"valid_from"`
}

// FetchDayAhead fetches prices from the Worker for a single date.
// Returns consumer prices (IsConsumer=true) when the Worker provides them,
// otherwise falls back to wholesale prices (IsConsumer=false).
func (s *SynctaclesAPI) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	dateStr := date.Format("2006-01-02")
	// Let the Worker choose the best resolution per zone (PT15M for most EU zones,
	// PT30M for GB, PT60M for zones without quarter-hourly data).
	url := fmt.Sprintf("%s/api/v1/energy/prices?zone=%s&from=%s&to=%s",
		s.baseURL(), zone, dateStr, dateStr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("synctacles create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SynctaclesEnergy/1.0")

	httpResp, err := defaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("synctacles fetch: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("synctacles: HTTP %d for zone %s", httpResp.StatusCode, zone)
	}

	var resp workerPriceResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("synctacles parse: %w", err)
	}

	// Cache tax profile (version-based — only update if version changed)
	s.mu.Lock()
	s.lastStatus = resp.DayAheadStatus
	if resp.TaxProfile != nil && resp.TaxProfileVersion != "" {
		if s.lastTaxProfile == nil || s.lastTaxProfile.Version != resp.TaxProfileVersion {
			s.lastTaxProfile = &CachedTaxProfile{
				Version:        resp.TaxProfileVersion,
				VatPct:         resp.TaxProfile.VatPct,
				EnergyTaxKWh:   resp.TaxProfile.EnergyTaxKWh,
				SurchargesKWh:  resp.TaxProfile.SurchargesKWh,
				NetworkCostKWh: resp.TaxProfile.NetworkCostKWh,
				ValidFrom:      resp.TaxProfile.ValidFrom,
			}
		}
	}
	s.mu.Unlock()

	prices := make([]models.HourlyPrice, 0, len(resp.Prices))
	for _, p := range resp.Prices {
		ts, err := time.Parse(time.RFC3339, p.Timestamp)
		if err != nil {
			continue
		}

		hp := models.HourlyPrice{
			Timestamp: ts.UTC(),
			Source:    "synctacles",
			Quality:   "live",
			Zone:      zone,
		}

		// Prefer consumer price from Worker (includes taxes + network tariff).
		// Always store wholesale for manual mode to recompute with user-defined components.
		if p.ConsumerPrice != nil {
			hp.PriceEUR = *p.ConsumerPrice
			hp.Unit = models.UnitKWh
			hp.IsConsumer = true
			hp.WholesaleKWh = p.Price / 1000.0 // MWh → kWh
		} else {
			hp.PriceEUR = p.Price
			hp.Unit = models.UnitMWh
			hp.IsConsumer = false
		}

		prices = append(prices, hp)
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("synctacles: no prices for zone %s on %s", zone, dateStr)
	}

	// Aggregate sub-hourly prices (PT15M, PT30M) to hourly.
	// The app engine expects 24 hourly entries. Without aggregation,
	// PT15M zones get 96 entries which breaks best-window calculations
	// and shows quarter-hour timestamps in cheapest/expensive displays.
	if resp.Resolution == "PT15M" || resp.Resolution == "PT30M" {
		prices = aggregateToHourly(prices)
	}

	return prices, nil
}

// aggregateToHourly groups sub-hourly prices (PT15M, PT30M) into hourly averages.
// PT15M zones return 96 entries per day, PT30M returns 48 — the app expects 24.
func aggregateToHourly(prices []models.HourlyPrice) []models.HourlyPrice {
	type bucket struct {
		sumPrice     float64
		sumWholesale float64
		count        int
		first        models.HourlyPrice // template for metadata
	}

	buckets := make(map[time.Time]*bucket)
	var order []time.Time // preserve chronological order

	for _, p := range prices {
		hour := p.Timestamp.Truncate(time.Hour)
		b, ok := buckets[hour]
		if !ok {
			b = &bucket{first: p}
			buckets[hour] = b
			order = append(order, hour)
		}
		b.sumPrice += p.PriceEUR
		b.sumWholesale += p.WholesaleKWh
		b.count++
	}

	result := make([]models.HourlyPrice, 0, len(order))
	for _, hour := range order {
		b := buckets[hour]
		hp := b.first
		hp.Timestamp = hour
		hp.PriceEUR = b.sumPrice / float64(b.count)
		hp.WholesaleKWh = b.sumWholesale / float64(b.count)
		result = append(result, hp)
	}
	return result
}

// TaxSeedResponse is the response from the Worker /api/v1/energy/tax endpoint.
type TaxSeedResponse struct {
	Zone        string   `json:"zone"`
	CountryCode string   `json:"country_code"`
	CountryName string   `json:"country_name"`
	Currency    string   `json:"currency"`
	SeedNeeded  int      `json:"seed_needed"`
	TaxSeed     *TaxSeed `json:"tax_seed"`
}

// TaxSeed holds a single active tax seed entry from the Worker.
type TaxSeed struct {
	CountryCode    string   `json:"country_code"`
	VatPct         float64  `json:"vat_pct"`
	EnergyTaxKWh   float64  `json:"energy_tax_kwh"`
	SurchargesKWh  float64  `json:"surcharges_kwh"`
	NetworkCostKWh *float64 `json:"network_cost_kwh"`
	ValidFrom      string   `json:"valid_from"`
	ValidTo        *string  `json:"valid_to"`
	Notes          string   `json:"notes"`
}

// FetchTaxSeed fetches the active tax seed for a zone from the Worker.
// Used to refresh tax data when the user changes their zone.
func (s *SynctaclesAPI) FetchTaxSeed(ctx context.Context, zone string) (*TaxSeedResponse, error) {
	url := fmt.Sprintf("%s/api/v1/energy/tax?zone=%s", s.baseURL(), zone)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("synctacles tax seed request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SynctaclesEnergy/1.0")

	httpResp, err := defaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("synctacles tax seed fetch: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("synctacles tax seed: HTTP %d for zone %s", httpResp.StatusCode, zone)
	}

	var resp TaxSeedResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("synctacles tax seed parse: %w", err)
	}
	return &resp, nil
}
