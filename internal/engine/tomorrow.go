package engine

import (
	"fmt"

	"github.com/synctacles/energy-go/internal/models"
)

// Thresholds for tomorrow preview determination.
const (
	favorableAbsoluteMax = 0.20 // EUR/kWh — avg below this = FAVORABLE
	expensiveAbsoluteMin = 0.30 // EUR/kWh — avg above this = EXPENSIVE
	favorableRelative    = 0.90 // tomorrow avg < today avg * 0.90 = FAVORABLE
	expensiveRelative    = 1.10 // tomorrow avg > today avg * 1.10 = EXPENSIVE
)

// DetermineTomorrowPreview computes the preview status for tomorrow's prices.
// todayPrices and tomorrowPrices should be in EUR/kWh.
func DetermineTomorrowPreview(todayPrices, tomorrowPrices []models.HourlyPrice) models.TomorrowResult {
	if len(tomorrowPrices) == 0 {
		return models.TomorrowResult{Status: models.PreviewPending}
	}

	tomorrowStats := CalcStats(tomorrowPrices)

	result := models.TomorrowResult{
		CheapestHour:  tomorrowStats.CheapestHour,
		ExpensiveHour: tomorrowStats.ExpensiveHour,
		AvgPrice:      tomorrowStats.Average,
	}

	// Determine status
	todayStats := CalcStats(todayPrices)

	switch {
	case tomorrowStats.Average < favorableAbsoluteMax:
		result.Status = models.PreviewFavorable
		result.Comparison = fmt.Sprintf("avg €%.3f/kWh (below €%.2f threshold)", tomorrowStats.Average, favorableAbsoluteMax)
	case len(todayPrices) > 0 && tomorrowStats.Average < todayStats.Average*favorableRelative:
		result.Status = models.PreviewFavorable
		result.Comparison = fmt.Sprintf("%.1f%% cheaper than today", (1-tomorrowStats.Average/todayStats.Average)*100)
	case tomorrowStats.Average > expensiveAbsoluteMin:
		result.Status = models.PreviewExpensive
		result.Comparison = fmt.Sprintf("avg €%.3f/kWh (above €%.2f threshold)", tomorrowStats.Average, expensiveAbsoluteMin)
	case len(todayPrices) > 0 && tomorrowStats.Average > todayStats.Average*expensiveRelative:
		result.Status = models.PreviewExpensive
		result.Comparison = fmt.Sprintf("%.1f%% more expensive than today", (tomorrowStats.Average/todayStats.Average-1)*100)
	default:
		result.Status = models.PreviewNormal
		if len(todayPrices) > 0 {
			result.Comparison = fmt.Sprintf("avg €%.3f/kWh (similar to today)", tomorrowStats.Average)
		} else {
			result.Comparison = fmt.Sprintf("avg €%.3f/kWh", tomorrowStats.Average)
		}
	}

	return result
}
