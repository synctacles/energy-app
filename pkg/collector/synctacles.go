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
type CachedTaxProfile struct {
	Version          string  `json:"version"`
	VATRate          float64 `json:"vat_rate"`
	EnergyTax        float64 `json:"energy_tax"`
	Surcharges       float64 `json:"surcharges"`
	NetworkTariffAvg float64 `json:"network_tariff_avg"`
	ValidFrom        string  `json:"valid_from"`
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
	Timestamp     string   `json:"timestamp"`
	Price         float64  `json:"price"`          // EUR/MWh (wholesale)
	ConsumerPrice *float64 `json:"consumer_price"` // EUR/kWh (with taxes)
}

type workerTaxProfile struct {
	VATRate          float64 `json:"vat_rate"`
	EnergyTax        float64 `json:"energy_tax"`
	Surcharges       float64 `json:"surcharges"`
	NetworkTariffAvg float64 `json:"network_tariff_avg"`
	ValidFrom        string  `json:"valid_from"`
}

// FetchDayAhead fetches prices from the Worker for a single date.
// Returns consumer prices (IsConsumer=true) when the Worker provides them,
// otherwise falls back to wholesale prices (IsConsumer=false).
func (s *SynctaclesAPI) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	dateStr := date.Format("2006-01-02")
	resolution := "PT60M"
	if zone == "GB" {
		resolution = "PT30M"
	}
	url := fmt.Sprintf("%s/prices?zone=%s&from=%s&to=%s&resolution=%s",
		s.baseURL(), zone, dateStr, dateStr, resolution)

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
				Version:          resp.TaxProfileVersion,
				VATRate:          resp.TaxProfile.VATRate,
				EnergyTax:        resp.TaxProfile.EnergyTax,
				Surcharges:       resp.TaxProfile.Surcharges,
				NetworkTariffAvg: resp.TaxProfile.NetworkTariffAvg,
				ValidFrom:        resp.TaxProfile.ValidFrom,
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
	return prices, nil
}
