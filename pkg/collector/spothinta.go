package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

const spotHintaBaseURL = "https://api.spot-hinta.fi"

// SpotHinta fetches Finnish and Nordic electricity prices.
// Covers FI, NO1-5, SE1-4, DK1-2, EE, LT, LV. No auth required.
// Rate limit: 1 req/min, 1440/day.
type SpotHinta struct{}

func (s *SpotHinta) Name() string     { return "spothinta" }
func (s *SpotHinta) RequiresKey() bool { return false }

func (s *SpotHinta) Zones() []string {
	return []string{"FI", "NO1", "NO2", "NO3", "NO4", "NO5", "SE1", "SE2", "SE3", "SE4", "DK1", "DK2", "EE", "LT", "LV"}
}

// FetchDayAhead fetches hourly prices for the given zone and date.
// SpotHinta returns prices in EUR/kWh (PriceNoTax = wholesale excl. tax).
func (s *SpotHinta) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	targetDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	var endpoint string
	if targetDay.Equal(today) {
		endpoint = "/Today"
	} else if targetDay.Equal(today.Add(24 * time.Hour)) {
		endpoint = "/DayForward"
	} else {
		// spot-hinta.fi only provides today and tomorrow
		return nil, fmt.Errorf("spothinta: only today and tomorrow supported, requested %s", date.Format("2006-01-02"))
	}

	url := fmt.Sprintf("%s%s?region=%s&priceResolution=60", spotHintaBaseURL, endpoint, zone)

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("spothinta fetch: %w", err)
	}

	var raw []struct {
		Rank         int     `json:"Rank"`
		DateTime     string  `json:"DateTime"`
		PriceNoTax   float64 `json:"PriceNoTax"`
		PriceWithTax float64 `json:"PriceWithTax"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("spothinta parse: %w", err)
	}

	prices := make([]models.HourlyPrice, 0, len(raw))
	for _, r := range raw {
		ts, err := time.Parse(time.RFC3339, r.DateTime)
		if err != nil {
			continue
		}
		prices = append(prices, models.HourlyPrice{
			Timestamp: ts.UTC(),
			PriceEUR:  r.PriceNoTax, // EUR/kWh wholesale excl. tax
			Unit:      models.UnitKWh,
			Source:    "spothinta",
			Quality:   "live",
			Zone:      zone,
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("spothinta: no prices for zone %s on %s", zone, date.Format("2006-01-02"))
	}
	return prices, nil
}
