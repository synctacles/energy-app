// Package hasensor publishes energy data as HA sensors.
package hasensor

import (
	"context"
	"fmt"
	"time"

	"github.com/synctacles/energy-go/internal/engine"
	"github.com/synctacles/energy-go/internal/models"
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
	Quality        string
	Leverancier    string // Enever supplier name (e.g. "zonneplan"), empty when not using Enever
	UpdatedAt      time.Time
}

// PublishAll publishes all sensor entities to HA via the given publisher.
// If a PowerTracker is provided (non-nil), sensors #9-11 are published when data is available.
func PublishAll(ctx context.Context, pub Publisher, s *SensorSet, hasLicense bool, power ...*PowerTracker) error {
	now := s.UpdatedAt.Format(time.RFC3339)

	// 1. Current price (FREE)
	priceAttrs := map[string]any{
		"unit_of_measurement": "EUR/kWh",
		"source":              s.Source,
		"quality":             s.Quality,
		"zone":                s.Zone,
		"friendly_name":       "Synctacles Energy Price",
		"icon":                "mdi:currency-eur",
		"device_class":        "monetary",
		"state_class":         "measurement",
		"last_updated":        now,
	}
	if s.Leverancier != "" {
		priceAttrs["leverancier"] = s.Leverancier
	}
	if err := pub.UpdateSensor(ctx, "sensor.synctacles_energy_price",
		fmt.Sprintf("%.4f", s.CurrentPrice), priceAttrs,
	); err != nil {
		return fmt.Errorf("publish price: %w", err)
	}

	// 2. Cheapest hour (FREE)
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

	// 3. Expensive hour (FREE)
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

	// 4. Prices today (FREE) — hourly array in attributes
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

	// --- PRO sensors (require license) ---
	if !hasLicense {
		return nil
	}

	// 5. Action (PRO)
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

	// 6. Best window (PRO)
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

	// 7. Tomorrow preview (PRO)
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

	// 8. Prices tomorrow (PRO)
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

	// --- Power-based sensors (PRO, only when power sensor available) ---
	if len(power) == 0 || power[0] == nil {
		return nil
	}
	pt := power[0]

	// 9. Live Cost (PRO, needs power sensor)
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
				"state_class":         "measurement",
				"last_updated":        now,
			},
		); err != nil {
			return fmt.Errorf("publish live cost: %w", err)
		}
	}

	// 10. Savings (PRO, needs power sensor)
	if savingsEUR, totalKWh, ok := pt.DailySavings(s.Stats.Average); ok {
		if err := pub.UpdateSensor(ctx, "sensor.synctacles_savings",
			fmt.Sprintf("%.2f", savingsEUR),
			map[string]any{
				"unit_of_measurement": "EUR",
				"daily_kwh":           fmt.Sprintf("%.1f", totalKWh),
				"avg_price":           s.Stats.Average,
				"friendly_name":       "Synctacles Savings",
				"icon":                "mdi:piggy-bank-outline",
				"state_class":         "total_increasing",
				"last_updated":        now,
			},
		); err != nil {
			return fmt.Errorf("publish savings: %w", err)
		}
	}

	// 11. Usage Score (PRO, needs power sensor)
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

	return nil
}

func actionIcon(a models.Action) string {
	switch a {
	case models.ActionGo:
		return "mdi:lightning-bolt"
	case models.ActionAvoid:
		return "mdi:hand-back-left"
	default:
		return "mdi:clock-outline"
	}
}

// ComputeSensorSet builds a SensorSet from fetched prices and engine results.
// leverancier is the Enever supplier name (empty string when not using Enever).
func ComputeSensorSet(
	zone string,
	todayPrices []models.HourlyPrice,
	tomorrowPrices []models.HourlyPrice,
	actionEngine *engine.ActionEngine,
	fetchResult *engine.FetchResult,
	now time.Time,
	leverancier string,
) *SensorSet {
	stats := engine.CalcStats(todayPrices)

	// Current hour price
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	var currentPrice float64
	for _, p := range todayPrices {
		if p.Timestamp.Equal(currentHour) {
			currentPrice = p.PriceEUR
			break
		}
	}

	// Action
	actionResult := actionEngine.Calculate(todayPrices, now, fetchResult.AllowGo())
	actionResult.Quality = fetchResult.Quality

	// Best window (3 hours)
	bestWindow := engine.FindBestWindow(todayPrices, now, 3)

	// Tomorrow preview
	tomorrow := engine.DetermineTomorrowPreview(todayPrices, tomorrowPrices)

	// Only set leverancier when Enever is the active source
	lev := ""
	if fetchResult.Source == "enever" {
		lev = leverancier
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
		Quality:        fetchResult.Quality,
		Leverancier:    lev,
		UpdatedAt:      now,
	}
}
