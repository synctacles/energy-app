package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/synctacles/energy-go/internal/models"
)

const energyChartsBaseURL = "https://api.energy-charts.info"

// ecSupportedZones lists bidding zones supported by Energy-Charts.
// The API uses zone codes directly as bzn parameter (e.g. ?bzn=NO1, ?bzn=DE-LU).
var ecSupportedZones = map[string]bool{
	"NL": true, "DE-LU": true, "AT": true, "BE": true, "FR": true,
	"NO1": true, "NO2": true, "NO3": true, "NO4": true, "NO5": true,
	"SE1": true, "SE2": true, "SE3": true, "SE4": true,
	"DK1": true, "DK2": true, "FI": true,
	"ES": true, "PT": true, "PL": true, "CZ": true, "CH": true, "HU": true,
	"IT-North": true, "IT-Centre-North": true, "IT-Centre-South": true,
	"IT-South": true, "IT-Sicily": true, "IT-Sardinia": true,
	"SI": true,
}

// EnergyCharts fetches wholesale electricity prices from Fraunhofer ISE.
// This is the universal EU fallback source — covers 16+ bidding zones, no auth.
type EnergyCharts struct{}

func (e *EnergyCharts) Name() string     { return "energycharts" }
func (e *EnergyCharts) RequiresKey() bool { return false }

func (e *EnergyCharts) Zones() []string {
	zones := make([]string, 0, len(ecSupportedZones))
	for z := range ecSupportedZones {
		zones = append(zones, z)
	}
	return zones
}

// FetchDayAhead fetches hourly wholesale prices for the given zone and date.
// Energy-Charts returns prices in EUR/MWh.
func (e *EnergyCharts) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	if !ecSupportedZones[zone] {
		return nil, fmt.Errorf("energycharts: unsupported zone %s", zone)
	}

	// Pass start/end dates to get the full day's data (ISO format: YYYY-MM-DD).
	// Without date params, the API only returns the most recent data points.
	dayStr := date.Format("2006-01-02")
	url := fmt.Sprintf("%s/price?bzn=%s&start=%s&end=%s", energyChartsBaseURL, zone, dayStr, dayStr)

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("energycharts fetch: %w", err)
	}

	var resp struct {
		UnixSeconds []int64    `json:"unix_seconds"`
		Price       []*float64 `json:"price"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("energycharts parse: %w", err)
	}

	if len(resp.UnixSeconds) != len(resp.Price) {
		return nil, fmt.Errorf("energycharts: timestamp/price array length mismatch (%d vs %d)",
			len(resp.UnixSeconds), len(resp.Price))
	}

	// Filter to requested date
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Energy-Charts may return 15-min intervals for some zones.
	// Filter to whole hours only (minute=0) for consistent 24-hour output.
	prices := make([]models.HourlyPrice, 0, 24)
	for i, unix := range resp.UnixSeconds {
		if resp.Price[i] == nil {
			continue
		}
		ts := time.Unix(unix, 0).UTC()
		if ts.Before(startOfDay) || !ts.Before(endOfDay) {
			continue
		}
		if ts.Minute() != 0 {
			continue // skip sub-hourly intervals
		}
		prices = append(prices, models.HourlyPrice{
			Timestamp: ts,
			PriceEUR:  *resp.Price[i],
			Unit:      models.UnitMWh,
			Source:    "energycharts",
			Quality:   "live",
			Zone:      zone,
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("energycharts: no prices for zone %s on %s", zone, date.Format("2006-01-02"))
	}
	return prices, nil
}
