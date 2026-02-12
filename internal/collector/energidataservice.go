package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/synctacles/energy-go/internal/models"
)

const energiDataServiceURL = "https://api.energidataservice.dk/dataset/Elspotprices"

// EnergiDataService fetches Nordic/Baltic wholesale electricity prices.
// Covers DK1, DK2, NO2, SE3, SE4 and more. No auth required.
type EnergiDataService struct{}

func (e *EnergiDataService) Name() string     { return "energidataservice" }
func (e *EnergiDataService) RequiresKey() bool { return false }

func (e *EnergiDataService) Zones() []string {
	return []string{"DK1", "DK2", "NO1", "NO2", "NO3", "NO4", "NO5", "SE1", "SE2", "SE3", "SE4", "FI", "EE", "LT", "LV"}
}

// FetchDayAhead fetches hourly wholesale prices for the given zone and date.
// Returns prices in EUR/MWh.
func (e *EnergiDataService) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	filter := fmt.Sprintf(`{"PriceArea":["%s"]}`, zone)
	params := url.Values{
		"start":  {start.Format("2006-01-02T15:04")},
		"end":    {end.Format("2006-01-02T15:04")},
		"filter": {filter},
		"sort":   {"HourUTC asc"},
		"limit":  {"48"},
	}

	reqURL := energiDataServiceURL + "?" + params.Encode()
	data, err := httpGet(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("energidataservice fetch: %w", err)
	}

	var resp struct {
		Records []struct {
			HourUTC      string   `json:"HourUTC"`
			PriceArea    string   `json:"PriceArea"`
			SpotPriceEUR *float64 `json:"SpotPriceEUR"`
		} `json:"records"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("energidataservice parse: %w", err)
	}

	prices := make([]models.HourlyPrice, 0, len(resp.Records))
	for _, r := range resp.Records {
		if r.SpotPriceEUR == nil {
			continue
		}
		ts, err := time.Parse("2006-01-02T15:04:05", r.HourUTC)
		if err != nil {
			continue
		}
		prices = append(prices, models.HourlyPrice{
			Timestamp: ts.UTC(),
			PriceEUR:  *r.SpotPriceEUR,
			Unit:      models.UnitMWh,
			Source:    "energidataservice",
			Quality:   "live",
			Zone:      zone,
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("energidataservice: no prices for zone %s on %s", zone, date.Format("2006-01-02"))
	}
	return prices, nil
}
