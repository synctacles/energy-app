// Package delta submits per-hour supplier correction factors to the energy-data
// Worker for crowdsourced price calibration (ADR_010).
package delta

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/synctacles/energy-app/internal/ha"
)

// ForecastPrice represents a single hour's consumer price from a sensor forecast.
type ForecastPrice struct {
	Timestamp time.Time
	PriceKWh  float64
}

// ReadSensorForecast reads day-ahead forecast prices from a HA sensor entity's
// attributes. Returns all known future hours. Different HA integrations store
// forecasts in different attribute formats — this function tries known patterns.
func ReadSensorForecast(ctx context.Context, sv *ha.SupervisorClient, entityID string) ([]ForecastPrice, error) {
	state, err := sv.GetState(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("read sensor state: %w", err)
	}

	attrs, _ := state["attributes"].(map[string]any)
	if attrs == nil {
		return nil, fmt.Errorf("sensor has no attributes")
	}

	var prices []ForecastPrice

	// Try known forecast attribute patterns
	for _, key := range []string{"forecast", "prices", "hourly"} {
		if arr, ok := attrs[key].([]any); ok && len(arr) > 0 {
			prices = parseForecastArray(arr)
			if len(prices) > 0 {
				slog.Debug("delta: forecast from attribute", "key", key, "hours", len(prices), "entity", entityID)
				break
			}
		}
	}

	// Try today/tomorrow arrays (Nord Pool, Energi Data Service, easyEnergy patterns)
	if len(prices) == 0 {
		for _, todayKey := range []string{"today", "today_prices", "raw_today"} {
			if arr, ok := attrs[todayKey].([]any); ok {
				prices = append(prices, parseForecastArray(arr)...)
			}
		}
		for _, tomKey := range []string{"tomorrow", "tomorrow_prices", "raw_tomorrow"} {
			if arr, ok := attrs[tomKey].([]any); ok {
				prices = append(prices, parseForecastArray(arr)...)
			}
		}
		if len(prices) > 0 {
			slog.Debug("delta: forecast from today/tomorrow arrays", "hours", len(prices), "entity", entityID)
		}
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("no forecast data found in sensor attributes")
	}

	// Sort by timestamp and deduplicate
	sort.Slice(prices, func(i, j int) bool { return prices[i].Timestamp.Before(prices[j].Timestamp) })
	deduped := prices[:0]
	var lastTS time.Time
	for _, p := range prices {
		if !p.Timestamp.Equal(lastTS) {
			deduped = append(deduped, p)
			lastTS = p.Timestamp
		}
	}

	return deduped, nil
}

// parseForecastArray tries to parse an array of forecast entries. Supports:
//   - {"start": "ISO8601", "value": float64}       (Nord Pool pattern)
//   - {"start": "ISO8601", "price": float64}
//   - {"hour": "HH:MM", "price": float64}
//   - {"datetime": "ISO8601", "price": float64}
//   - plain float64 array (indexed by hour 0-23)
func parseForecastArray(arr []any) []ForecastPrice {
	var prices []ForecastPrice

	for i, item := range arr {
		switch v := item.(type) {
		case map[string]any:
			ts, price, ok := parseMapEntry(v)
			if ok {
				prices = append(prices, ForecastPrice{Timestamp: ts, PriceKWh: price})
			}
		case float64:
			// Plain float array — assume index = hour of today
			now := time.Now().UTC()
			today := time.Date(now.Year(), now.Month(), now.Day(), i, 0, 0, 0, time.UTC)
			prices = append(prices, ForecastPrice{Timestamp: today, PriceKWh: v})
		}
	}
	return prices
}

// parseMapEntry extracts timestamp and price from a forecast map entry.
func parseMapEntry(m map[string]any) (time.Time, float64, bool) {
	// Find timestamp
	var ts time.Time
	for _, key := range []string{"start", "datetime", "start_time", "date", "time"} {
		if s, ok := m[key].(string); ok && s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				ts = t.UTC()
				break
			}
			if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
				ts = t.UTC()
				break
			}
		}
	}
	if ts.IsZero() {
		return ts, 0, false
	}

	// Find price
	for _, key := range []string{"value", "price", "tariff", "rate"} {
		if p, ok := m[key].(float64); ok {
			return ts, p, true
		}
	}
	return ts, 0, false
}
