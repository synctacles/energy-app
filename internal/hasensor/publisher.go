// Package hasensor publishes energy data as HA sensors.
package hasensor

import (
	"context"
	"fmt"
	"time"

	"github.com/synctacles/energy-app/pkg/engine"
	"github.com/synctacles/energy-app/pkg/models"
)

// Publisher is the interface for publishing sensor state to Home Assistant.
type Publisher interface {
	// UpdateSensor sets the state and attributes of a sensor entity.
	UpdateSensor(ctx context.Context, entityID, state string, attrs map[string]any) error
}

// SensorSet holds all computed sensor values ready for publishing.
type SensorSet struct {
	Zone           string
	CurrentPrice   float64
	Action         models.ActionResult
	Stats          models.PriceStats
	BestWindow     *models.BestWindow
	Tomorrow       models.TomorrowResult
	TodayPrices    []models.HourlyPrice
	TomorrowPrices []models.HourlyPrice
	Source         string
	SourceTier     string // "worker", "energy_charts", "cache"
	Quality        string
	UpdatedAt      time.Time
}

// PublishAll publishes all sensor entities to HA via the given publisher.
// If a PowerTracker is provided (non-nil), sensors #9-11 are published when data is available.
func PublishAll(ctx context.Context, pub Publisher, s *SensorSet, power ...*PowerTracker) error {
	now := s.UpdatedAt.Format(time.RFC3339)

	// 1. Current price
	priceAttrs := map[string]any{
		"unit_of_measurement": "EUR/kWh",
		"source":              s.Source,
		"source_tier":         s.SourceTier,
		"quality":             s.Quality,
		"zone":                s.Zone,
		"friendly_name":       "Synctacles Energy Price",
		"icon":                "mdi:currency-eur",
		"device_class":        "monetary",
		"last_updated":        now,
	}
	if err := pub.UpdateSensor(ctx, "sensor.synctacles_energy_price",
		fmt.Sprintf("%.4f", s.CurrentPrice), priceAttrs,
	); err != nil {
		return fmt.Errorf("publish price: %w", err)
	}

	// 2. Cheapest hour
	if err := pub.UpdateSensor(ctx, "sensor.synctacles_cheapest_hour",
		s.Stats.CheapestHour,
		map[string]any{
			"price":         s.Stats.Min,
			"zone":          s.Zone,
			"friendly_name": "Synctacles Cheapest Hour",
			"icon":          "mdi:clock-check-outline",
			"last_updated":  now,
		},
	); err != nil {
		return fmt.Errorf("publish cheapest: %w", err)
	}

	// 3. Expensive hour
	if err := pub.UpdateSensor(ctx, "sensor.synctacles_expensive_hour",
		s.Stats.ExpensiveHour,
		map[string]any{
			"price":         s.Stats.Max,
			"zone":          s.Zone,
			"friendly_name": "Synctacles Expensive Hour",
			"icon":          "mdi:clock-alert-outline",
			"last_updated":  now,
		},
	); err != nil {
		return fmt.Errorf("publish expensive: %w", err)
	}

	// 4. Prices today — hourly array in attributes
	todayHourly := make([]map[string]any, 0, len(s.TodayPrices))
	for _, p := range s.TodayPrices {
		todayHourly = append(todayHourly, map[string]any{
			"hour":  p.Timestamp.Format("15:04"),
			"price": p.PriceEUR,
		})
	}
	if err := pub.UpdateSensor(ctx, "sensor.synctacles_prices_today",
		fmt.Sprintf("%d", len(s.TodayPrices)),
		map[string]any{
			"hourly":        todayHourly,
			"average":       s.Stats.Average,
			"min":           s.Stats.Min,
			"max":           s.Stats.Max,
			"zone":          s.Zone,
			"friendly_name": "Synctacles Prices Today",
			"icon":          "mdi:chart-line",
			"last_updated":  now,
		},
	); err != nil {
		return fmt.Errorf("publish prices today: %w", err)
	}

	// 5. Cheap hour binary sensor — ON when action is GO
	cheapState := "off"
	if s.Action.Action == models.ActionGo {
		cheapState = "on"
	}
	if err := pub.UpdateSensor(ctx, "binary_sensor.synctacles_cheap_hour",
		cheapState,
		map[string]any{
			"device_class":  "power",
			"friendly_name": "Synctacles Cheap Hour",
			"icon":          "mdi:flash",
			"current_price": s.CurrentPrice,
			"average_price": s.Stats.Average,
			"last_updated":  now,
		},
	); err != nil {
		return fmt.Errorf("publish cheap hour: %w", err)
	}

	// 6. Action
	if err := pub.UpdateSensor(ctx, "sensor.synctacles_energy_action",
		string(s.Action.Action),
		map[string]any{
			"reason":        s.Action.Reason,
			"deviation_pct": s.Action.DeviationPct,
			"current_price": s.Action.CurrentPrice,
			"average_price": s.Action.AveragePrice,
			"quality":       s.Action.Quality,
			"friendly_name": "Synctacles Energy Action",
			"icon":          actionIcon(s.Action.Action),
			"last_updated":  now,
		},
	); err != nil {
		return fmt.Errorf("publish action: %w", err)
	}

	// 7. Best window
	if s.BestWindow != nil {
		if err := pub.UpdateSensor(ctx, "sensor.synctacles_best_window",
			fmt.Sprintf("%s - %s", s.BestWindow.StartHour, s.BestWindow.EndHour),
			map[string]any{
				"avg_price":     s.BestWindow.AvgPrice,
				"duration":      s.BestWindow.Duration,
				"total_cost":    s.BestWindow.TotalCost,
				"friendly_name": "Synctacles Best Window",
				"icon":          "mdi:clock-star-four-points-outline",
				"last_updated":  now,
			},
		); err != nil {
			return fmt.Errorf("publish best window: %w", err)
		}
	}

	// 8. Tomorrow preview
	if err := pub.UpdateSensor(ctx, "sensor.synctacles_tomorrow_preview",
		string(s.Tomorrow.Status),
		map[string]any{
			"cheapest_hour":  s.Tomorrow.CheapestHour,
			"expensive_hour": s.Tomorrow.ExpensiveHour,
			"avg_price":      s.Tomorrow.AvgPrice,
			"comparison":     s.Tomorrow.Comparison,
			"friendly_name":  "Synctacles Tomorrow Preview",
			"icon":           "mdi:calendar-arrow-right",
			"last_updated":   now,
		},
	); err != nil {
		return fmt.Errorf("publish tomorrow: %w", err)
	}

	// 9. Prices tomorrow
	tomorrowHourly := make([]map[string]any, 0, len(s.TomorrowPrices))
	for _, p := range s.TomorrowPrices {
		tomorrowHourly = append(tomorrowHourly, map[string]any{
			"hour":  p.Timestamp.Format("15:04"),
			"price": p.PriceEUR,
		})
	}
	if err := pub.UpdateSensor(ctx, "sensor.synctacles_prices_tomorrow",
		fmt.Sprintf("%d", len(s.TomorrowPrices)),
		map[string]any{
			"hourly":        tomorrowHourly,
			"zone":          s.Zone,
			"friendly_name": "Synctacles Prices Tomorrow",
			"icon":          "mdi:chart-line-variant",
			"last_updated":  now,
		},
	); err != nil {
		return fmt.Errorf("publish prices tomorrow: %w", err)
	}

	// --- Power-based sensors (needs power sensor) ---
	if len(power) == 0 || power[0] == nil {
		return nil
	}
	pt := power[0]

	// 10. Live Cost (needs power sensor)
	if costEUR, powerW, ok := pt.LiveCost(s.CurrentPrice); ok {
		dailyTotal := float64(0)
		if savings, totalKWh, ok2 := pt.DailySavings(s.Stats.Average); ok2 {
			_ = savings
			dailyTotal = totalKWh * s.Stats.Average
		}
		if err := pub.UpdateSensor(ctx, "sensor.synctacles_live_cost",
			fmt.Sprintf("%.2f", costEUR),
			map[string]any{
				"unit_of_measurement": "EUR/h",
				"power_w":             powerW,
				"price_kwh":           s.CurrentPrice,
				"daily_total":         fmt.Sprintf("%.2f", dailyTotal),
				"friendly_name":       "Synctacles Live Cost",
				"icon":                "mdi:cash-clock",
				"device_class":        "monetary",
				"last_updated":        now,
			},
		); err != nil {
			return fmt.Errorf("publish live cost: %w", err)
		}
	}

	// 11. Savings (needs power sensor)
	if savingsEUR, totalKWh, ok := pt.DailySavings(s.Stats.Average); ok {
		if err := pub.UpdateSensor(ctx, "sensor.synctacles_savings",
			fmt.Sprintf("%.2f", savingsEUR),
			map[string]any{
				"unit_of_measurement": "EUR",
				"daily_kwh":           fmt.Sprintf("%.1f", totalKWh),
				"avg_price":           s.Stats.Average,
				"friendly_name":       "Synctacles Savings",
				"icon":                "mdi:piggy-bank-outline",
				"state_class":         "measurement",
				"last_updated":        now,
			},
		); err != nil {
			return fmt.Errorf("publish savings: %w", err)
		}
	}

	// 12. Usage Score (needs power sensor)
	if score, cheapPct, avgPct, expPct, ok := pt.UsageScore(s.Stats.Average); ok {
		if err := pub.UpdateSensor(ctx, "sensor.synctacles_usage_score",
			fmt.Sprintf("%d", score),
			map[string]any{
				"cheap_pct":     fmt.Sprintf("%.1f", cheapPct),
				"average_pct":   fmt.Sprintf("%.1f", avgPct),
				"expensive_pct": fmt.Sprintf("%.1f", expPct),
				"friendly_name": "Synctacles Usage Score",
				"icon":          "mdi:gauge",
				"state_class":   "measurement",
				"last_updated":  now,
			},
		); err != nil {
			return fmt.Errorf("publish usage score: %w", err)
		}
	}

	// 13. Daily Cost (needs power sensor)
	if costEUR, totalKWh, ok := pt.DailyCost(); ok {
		if err := pub.UpdateSensor(ctx, "sensor.synctacles_daily_cost",
			fmt.Sprintf("%.2f", costEUR),
			map[string]any{
				"unit_of_measurement": "EUR",
				"daily_kwh":           fmt.Sprintf("%.1f", totalKWh),
				"friendly_name":       "Synctacles Daily Cost",
				"icon":                "mdi:counter",
				"device_class":        "monetary",
				"state_class":         "total",
				"last_updated":        now,
			},
		); err != nil {
			return fmt.Errorf("publish daily cost: %w", err)
		}
	}

	return nil
}

func actionIcon(a models.Action) string {
	switch a {
	case models.ActionGo, models.ActionOffpeak:
		return "mdi:lightning-bolt"
	case models.ActionAvoid, models.ActionPeak:
		return "mdi:hand-back-left"
	case models.ActionFlat:
		return "mdi:minus-circle-outline"
	default:
		return "mdi:clock-outline"
	}
}

// ComputeSensorSet builds a SensorSet from fetched prices and engine results.
// bestWindowHours controls the best window duration (default 3, range 1-8).
// pricingMode controls action calculation: "fixed"/"tou" use regulated logic, others use wholesale.
func ComputeSensorSet(
	zone string,
	todayPrices []models.HourlyPrice,
	tomorrowPrices []models.HourlyPrice,
	actionEngine *engine.ActionEngine,
	fetchResult *engine.FetchResult,
	now time.Time,
	_ string, // deprecated: leverancier parameter (kept for backward compat)
	bestWindowHours int,
	pricingMode string,
) *SensorSet {
	stats := engine.CalcStats(todayPrices)

	// Current slot price (works for both PT60 hourly and PT15 quarter-hourly data)
	currentPrice, _, _ := engine.CurrentSlotPrice(todayPrices, now)

	// For PT15M data: use hourly average as display price.
	// Mathematically correct: avg(consumer_quarters) = (avg_wholesale + taxes) × (1+VAT)
	// because VAT is a linear multiplier. Any remaining difference with the sensor
	// is from the delta correction lag, not from the averaging method.
	if len(todayPrices) > 48 {
		currentHour := now.Truncate(time.Hour)
		nextHour := currentHour.Add(time.Hour)
		var hourSum float64
		var hourCount int
		for _, p := range todayPrices {
			if !p.Timestamp.Before(currentHour) && p.Timestamp.Before(nextHour) {
				hourSum += p.PriceEUR
				hourCount++
			}
		}
		if hourCount > 0 {
			currentPrice = hourSum / float64(hourCount)
		}
	}

	// Action: use regulated engine for fixed/tou, wholesale engine for dynamic modes
	var actionResult models.ActionResult
	if pricingMode == "fixed" || pricingMode == "tou" {
		actionResult = engine.CalculateRegulatedAction(todayPrices, now, pricingMode)
	} else {
		actionResult = actionEngine.Calculate(todayPrices, now, fetchResult.AllowGo())
	}
	actionResult.Quality = fetchResult.Quality

	// Best window: skip for fixed (all hours equal), use full offpeak block for TOU
	if bestWindowHours < 1 {
		bestWindowHours = 3
	}
	var bestWindow *models.BestWindow
	if pricingMode == "fixed" {
		// No best window for fixed rate — every hour is identical
		bestWindow = nil
	} else if pricingMode == "tou" {
		// TOU: find the full offpeak block instead of arbitrary N-hour window
		bestWindow = engine.FindOffpeakWindow(todayPrices, now)
	} else {
		bestWindow = engine.FindBestWindow(todayPrices, now, bestWindowHours)
	}

	// Tomorrow preview: skip for fixed/tou (always identical)
	var tomorrow models.TomorrowResult
	if pricingMode == "fixed" || pricingMode == "tou" {
		tomorrow = models.TomorrowResult{Status: models.PreviewPending}
	} else {
		tomorrow = engine.DetermineTomorrowPreview(todayPrices, tomorrowPrices)
	}

	// Map source tier to human-readable label for HA sensor attribute
	sourceTier := fetchResult.Source
	switch {
	case fetchResult.Tier == 4:
		sourceTier = "cache"
	case fetchResult.Source == "sensor":
		sourceTier = "sensor"
	case fetchResult.Source == "synctacles":
		sourceTier = "worker"
	case fetchResult.Source == "energycharts":
		sourceTier = "energy_charts"
	}

	return &SensorSet{
		Zone:           zone,
		CurrentPrice:   currentPrice,
		Action:         actionResult,
		Stats:          stats,
		BestWindow:     bestWindow,
		Tomorrow:       tomorrow,
		TodayPrices:    todayPrices,
		TomorrowPrices: tomorrowPrices,
		Source:         fetchResult.Source,
		SourceTier:     sourceTier,
		Quality:        fetchResult.Quality,
		UpdatedAt:      now,
	}
}
