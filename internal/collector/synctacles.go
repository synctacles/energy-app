// Package collector — SynctaclesAPI is the primary price source for the addon.
// It fetches pre-computed consumer prices from the Synctacles central server.
// All other collectors (Energy-Charts, aWATTar, etc.) serve as emergency fallback
// and are only used when the Synctacles server is unreachable.
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/synctacles/energy-go/internal/models"
)

// SynctaclesAPI fetches prices from the Synctacles central price server.
// This is the addon's primary source (Tier 0). It returns pre-computed
// consumer prices including taxes and surcharges for all 30 EU zones.
type SynctaclesAPI struct {
	BaseURL string // e.g. "https://energy.synctacles.com"
}

func (s *SynctaclesAPI) Name() string     { return "synctacles" }
func (s *SynctaclesAPI) RequiresKey() bool { return false }

// Zones returns all supported zones. The Synctacles server covers all 30 EU zones.
func (s *SynctaclesAPI) Zones() []string {
	return []string{
		"NL", "DE-LU", "AT", "BE", "CH", "CZ", "FR", "HU", "PL", "SI",
		"DK1", "DK2", "ES", "PT", "FI",
		"IT-North", "IT-Centre-North", "IT-Centre-South", "IT-South", "IT-Sicily", "IT-Sardinia",
		"NO1", "NO2", "NO3", "NO4", "NO5",
		"SE1", "SE2", "SE3", "SE4",
	}
}

// synctaclesPriceResponse is the wire format from the Synctacles price API.
type synctaclesPriceResponse struct {
	Zone    string                  `json:"zone"`
	Source  string                  `json:"source"`
	Quality string                 `json:"quality"`
	Prices  []synctaclesPriceEntry `json:"prices"`
}

type synctaclesPriceEntry struct {
	Timestamp  string  `json:"timestamp"`
	PriceEUR   float64 `json:"price_eur"`
	Unit       string  `json:"unit"`
	IsConsumer bool    `json:"is_consumer"`
}

// FetchDayAhead fetches pre-computed consumer prices from the Synctacles server.
func (s *SynctaclesAPI) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	url := fmt.Sprintf("%s/api/v1/prices?zone=%s&date=%s",
		s.BaseURL, zone, date.Format("2006-01-02"))

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("synctacles fetch: %w", err)
	}

	var resp synctaclesPriceResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("synctacles parse: %w", err)
	}

	prices := make([]models.HourlyPrice, 0, len(resp.Prices))
	for _, p := range resp.Prices {
		ts, err := time.Parse(time.RFC3339, p.Timestamp)
		if err != nil {
			continue
		}
		unit := models.UnitKWh
		if p.Unit == "EUR/MWh" {
			unit = models.UnitMWh
		}
		prices = append(prices, models.HourlyPrice{
			Timestamp:  ts.UTC(),
			PriceEUR:   p.PriceEUR,
			Unit:       unit,
			Source:     "synctacles",
			Quality:    resp.Quality,
			Zone:       zone,
			IsConsumer: p.IsConsumer,
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("synctacles: no prices for zone %s on %s", zone, date.Format("2006-01-02"))
	}
	return prices, nil
}
