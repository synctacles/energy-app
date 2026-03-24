package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

const energyChartsBaseURL = "https://api.energy-charts.info"

// ecSupportedZones lists bidding zones supported by Energy-Charts.
// Keys are our canonical ENTSO-E zone codes; values are the bzn parameter
// Energy-Charts expects (which differs for Italian zones).
var ecSupportedZones = map[string]string{
	"NL": "NL", "DE-LU": "DE-LU", "AT": "AT", "BE": "BE", "FR": "FR",
	"NO1": "NO1", "NO2": "NO2", "NO3": "NO3", "NO4": "NO4", "NO5": "NO5",
	"SE1": "SE1", "SE2": "SE2", "SE3": "SE3", "SE4": "SE4",
	"DK1": "DK1", "DK2": "DK2", "FI": "FI",
	"ES": "ES", "PT": "PT", "PL": "PL", "CZ": "CZ", "CH": "CH", "HU": "HU",
	"IT-NORTH": "IT-North", "IT-CNORTH": "IT-Centre-North", "IT-CSOUTH": "IT-Centre-South",
	"IT-SOUTH": "IT-South", "IT-Sicily": "IT-Sicily", "IT-Sardinia": "IT-Sardinia",
	"SI": "SI",
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
	ecZone, ok := ecSupportedZones[zone]
	if !ok {
		return nil, fmt.Errorf("energycharts: unsupported zone %s", zone)
	}

	// Pass start/end dates to get the full day's data (ISO format: YYYY-MM-DD).
	// Without date params, the API only returns the most recent data points.
	dayStr := date.Format("2006-01-02")
	url := fmt.Sprintf("%s/price?bzn=%s&start=%s&end=%s", energyChartsBaseURL, ecZone, dayStr, dayStr)

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

	// Energy-Charts returns PT15M for some zones (e.g. NL, DE-LU) and PT60M for others.
	// Collect all valid data points first, then decide resolution.
	var all []models.HourlyPrice
	for i, unix := range resp.UnixSeconds {
		if resp.Price[i] == nil {
			continue
		}
		ts := time.Unix(unix, 0).UTC()
		if ts.Before(startOfDay) || !ts.Before(endOfDay) {
			continue
		}
		all = append(all, models.HourlyPrice{
			Timestamp: ts,
			PriceEUR:  *resp.Price[i],
			Unit:      models.UnitMWh,
			Source:    "energycharts",
			Quality:   "live",
			Zone:      zone,
		})
	}

	// If we have sub-hourly data (>24 points), use PT15M as-is.
	// Otherwise fall back to PT60M (whole hours only).
	prices := all
	if len(all) <= 24 {
		prices = make([]models.HourlyPrice, 0, 24)
		for _, p := range all {
			if p.Timestamp.Minute() == 0 {
				prices = append(prices, p)
			}
		}
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("energycharts: no prices for zone %s on %s", zone, date.Format("2006-01-02"))
	}
	return prices, nil
}
