package engine

import (
	"fmt"
	"math"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// floatEpsilon is the tolerance for float64 price comparisons.
const floatEpsilon = 0.0001

// CalculateRegulatedAction computes the action for non-wholesale (regulated) pricing modes.
// For fixed-rate: always FLAT (every hour is the same price).
// For TOU: OFFPEAK or PEAK based on current rate, with countdown to next transition.
func CalculateRegulatedAction(prices []models.HourlyPrice, now time.Time, pricingMode string) models.ActionResult {
	if len(prices) == 0 {
		return models.ActionResult{
			Action: models.ActionWait,
			Reason: "no price data available",
		}
	}

	currentPrice, _, found := CurrentSlotPrice(prices, now)
	if !found {
		return models.ActionResult{
			Action: models.ActionWait,
			Reason: "current hour not found in price data",
		}
	}

	stats := CalcStats(prices)

	// Fixed-rate mode: all prices identical, no action recommendation needed
	if pricingMode == "fixed" {
		return models.ActionResult{
			Action:       models.ActionFlat,
			Reason:       "fixed rate — all hours equal",
			CurrentPrice: currentPrice,
			AveragePrice: stats.Average,
		}
	}

	// TOU mode: determine if current hour is peak or offpeak
	// We detect this by comparing current price to min price:
	// if current == min → offpeak, else → peak (or midpeak treated as peak)
	isPeak := currentPrice > stats.Min+floatEpsilon

	// Find next transition: scan future hours for price change
	nextTransition, nextRate := findNextTransition(prices, now, currentPrice, stats)

	if isPeak {
		result := models.ActionResult{
			Action:         models.ActionPeak,
			CurrentPrice:   currentPrice,
			AveragePrice:   stats.Average,
			NextTransition: nextTransition,
			NextRate:       nextRate,
		}
		if nextTransition != "" {
			result.Reason = fmt.Sprintf("peak rate — offpeak starts at %s", nextTransition)
		} else {
			result.Reason = "peak rate"
		}
		return result
	}

	result := models.ActionResult{
		Action:         models.ActionOffpeak,
		CurrentPrice:   currentPrice,
		AveragePrice:   stats.Average,
		NextTransition: nextTransition,
		NextRate:       nextRate,
	}
	if nextTransition != "" {
		result.Reason = fmt.Sprintf("offpeak rate — peak starts at %s", nextTransition)
	} else {
		result.Reason = "offpeak rate"
	}
	return result
}

// findNextTransition scans prices after 'now' to find the first hour where the price
// differs from the current price. Returns "HH:MM" and rate label, or "" if no change found.
func findNextTransition(prices []models.HourlyPrice, now time.Time, currentPrice float64, stats models.PriceStats) (string, string) {
	for _, p := range prices {
		if !p.Timestamp.After(now) {
			continue
		}
		// Price changed — this is the transition point
		if math.Abs(p.PriceEUR-currentPrice) > floatEpsilon {
			label := "offpeak"
			if p.PriceEUR > stats.Min+floatEpsilon {
				label = "peak"
			}
			return p.Timestamp.Format("15:04"), label
		}
	}
	return "", ""
}
