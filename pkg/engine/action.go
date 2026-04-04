package engine

import (
	"fmt"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
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

// CurrentSlotPrice finds the price for the current time slot from a list of prices.
// Works for both hourly (PT60) and quarter-hourly (PT15) data by selecting the
// most recent entry at or before now. Returns the price, slot timestamp, and whether
// a matching slot was found.
func CurrentSlotPrice(prices []models.HourlyPrice, now time.Time) (price float64, slot time.Time, found bool) {
	for _, p := range prices {
		if !p.Timestamp.After(now) && p.Timestamp.After(slot) {
			price = p.PriceEUR
			slot = p.Timestamp
			found = true
		}
	}
	return
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

	// Find current slot's price (works for PT60 and PT15)
	currentPrice, currentSlot, found := CurrentSlotPrice(prices, now)

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

	// Check if current slot is in best 4 (always GO if allowed)
	isBest4 := false
	slotStr := currentSlot.Format("15:04")
	for _, h := range stats.Best4Hours {
		if h == slotStr {
			isBest4 = true
			break
		}
	}

	// Determine action
	var action models.Action
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
