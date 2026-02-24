package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// aWATTar API endpoints per country.
var awattarURLs = map[string]string{
	"DE-LU": "https://api.awattar.de/v1/marketdata",
	"AT":    "https://api.awattar.at/v1/marketdata",
}

// AWATTar fetches German and Austrian wholesale electricity prices.
// No auth required. Rate limit: 100 requests/day.
type AWATTar struct{}

func (a *AWATTar) Name() string     { return "awattar" }
func (a *AWATTar) RequiresKey() bool { return false }

func (a *AWATTar) Zones() []string {
	return []string{"DE-LU", "AT"}
}

// FetchDayAhead fetches hourly wholesale prices for the given zone and date.
// Returns prices in EUR/MWh.
func (a *AWATTar) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	baseURL, ok := awattarURLs[zone]
	if !ok {
		return nil, fmt.Errorf("awattar: unsupported zone %s", zone)
	}

	// aWATTar uses Unix millisecond timestamps
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf("%s?start=%d&end=%d",
		baseURL,
		start.UnixMilli(),
		end.UnixMilli(),
	)

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("awattar fetch: %w", err)
	}

	var resp struct {
		Data []struct {
			StartTimestamp int64   `json:"start_timestamp"` // Unix ms
			EndTimestamp   int64   `json:"end_timestamp"`
			Marketprice    float64 `json:"marketprice"`
			Unit           string  `json:"unit"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("awattar parse: %w", err)
	}

	prices := make([]models.HourlyPrice, 0, len(resp.Data))
	for _, r := range resp.Data {
		ts := time.UnixMilli(r.StartTimestamp).UTC()
		prices = append(prices, models.HourlyPrice{
			Timestamp: ts,
			PriceEUR:  r.Marketprice,
			Unit:      models.UnitMWh,
			Source:    "awattar",
			Quality:   "live",
			Zone:      zone,
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("awattar: no prices for zone %s on %s", zone, date.Format("2006-01-02"))
	}
	return prices, nil
}
