package engine

import (
	"fmt"
	"time"

	"github.com/synctacles/energy-go/internal/models"
)

// ActionEngine computes GO/WAIT/AVOID recommendations based on current price vs daily average.
type ActionEngine struct {
	goThreshold    float64 // e.g. -15 (% below average → GO)
	avoidThreshold float64 // e.g. 20 (% above average → AVOID)
}

// NewActionEngine creates an action engine with configurable thresholds.
func NewActionEngine(goThreshold, avoidThreshold float64) *ActionEngine {
	return &ActionEngine{
		goThreshold:    goThreshold,
		avoidThreshold: avoidThreshold,
	}
}

// Calculate computes the action for the current hour given today's prices.
// Expects prices in EUR/kWh. allowGo controls whether GO is permitted (only for live data).
func (e *ActionEngine) Calculate(prices []models.HourlyPrice, now time.Time, allowGo bool) models.ActionResult {
	if len(prices) == 0 {
		return models.ActionResult{
			Action: models.ActionWait,
			Reason: "no price data available",
		}
	}

	stats := CalcStats(prices)

	// Find current hour's price
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	var currentPrice float64
	found := false
	for _, p := range prices {
		if p.Timestamp.Equal(currentHour) {
			currentPrice = p.PriceEUR
			found = true
			break
		}
	}

	if !found {
		return models.ActionResult{
			Action:       models.ActionWait,
			Reason:       "current hour not found in price data",
			AveragePrice: stats.Average,
		}
	}

	// Calculate deviation from daily average
	var deviationPct float64
	if stats.Average != 0 {
		deviationPct = ((currentPrice - stats.Average) / stats.Average) * 100
	}

	// Check if current hour is in best 4 hours (always GO if allowed)
	isBest4 := false
	hourStr := currentHour.Format("15:04")
	for _, h := range stats.Best4Hours {
		if h == hourStr {
			isBest4 = true
			break
		}
	}

	// Determine action
	action := models.ActionWait
	var reason string

	switch {
	case isBest4 && allowGo:
		action = models.ActionGo
		reason = "cheapest 4 hours of the day"
	case deviationPct <= e.goThreshold && allowGo:
		action = models.ActionGo
		reason = fmt.Sprintf("%.1f%% below daily average", -deviationPct)
	case deviationPct >= e.avoidThreshold:
		action = models.ActionAvoid
		reason = fmt.Sprintf("%.1f%% above daily average", deviationPct)
	default:
		action = models.ActionWait
		reason = "price near daily average"
	}

	// Override: no GO on cached/stale data
	if !allowGo && action == models.ActionGo {
		action = models.ActionWait
		reason = "data not fresh enough for GO recommendation"
	}

	return models.ActionResult{
		Action:       action,
		Reason:       reason,
		DeviationPct: deviationPct,
		CurrentPrice: currentPrice,
		AveragePrice: stats.Average,
	}
}
