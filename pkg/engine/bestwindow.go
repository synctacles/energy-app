package engine

import (
	"sort"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// slidingWindowResult holds the output of the sliding window search.
type slidingWindowResult struct {
	startIdx int
	avgPrice float64
}

// slidingWindowMin finds the contiguous block of slotsNeeded with the lowest average price.
// Returns the start index and average price. Prices must already be sorted and filtered.
func slidingWindowMin(prices []models.HourlyPrice, slotsNeeded int) slidingWindowResult {
	bestSum := float64(0)
	for i := 0; i < slotsNeeded; i++ {
		bestSum += prices[i].PriceEUR
	}

	currentSum := bestSum
	bestStart := 0

	for i := 1; i <= len(prices)-slotsNeeded; i++ {
		currentSum = currentSum - prices[i-1].PriceEUR + prices[i+slotsNeeded-1].PriceEUR
		if currentSum < bestSum {
			bestSum = currentSum
			bestStart = i
		}
	}

	return slidingWindowResult{
		startIdx: bestStart,
		avgPrice: bestSum / float64(slotsNeeded),
	}
}

// buildWindow constructs a BestWindow from a sliding window result.
func buildWindow(future []models.HourlyPrice, result slidingWindowResult, slotsNeeded, durationHours int, slotDur time.Duration) *models.BestWindow {
	startTime := future[result.startIdx].Timestamp
	endTime := future[result.startIdx+slotsNeeded-1].Timestamp.Add(slotDur)

	return &models.BestWindow{
		StartHour: startTime.Format("15:04"),
		EndHour:   endTime.Format("15:04"),
		AvgPrice:  result.avgPrice,
		Duration:  durationHours,
		TotalCost: result.avgPrice * float64(durationHours),
	}
}

// sortAndFilterFuture sorts prices by timestamp and returns only future slots.
func sortAndFilterFuture(prices []models.HourlyPrice, now time.Time) []models.HourlyPrice {
	sorted := make([]models.HourlyPrice, len(prices))
	copy(sorted, prices)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	var future []models.HourlyPrice
	for _, p := range sorted {
		if !p.Timestamp.Before(currentHour) {
			future = append(future, p)
		}
	}
	return future
}

// FindBestWindow finds the cheapest contiguous window of the given duration
// from the remaining slots in the day (starting from now).
// Duration is in hours. Handles both PT60 and PT15 data automatically.
// Returns nil if not enough slots remain.
func FindBestWindow(prices []models.HourlyPrice, now time.Time, durationHours int) *models.BestWindow {
	if durationHours <= 0 || len(prices) == 0 {
		return nil
	}

	slotDur := DetectSlotDuration(prices)
	slotsPerHour := int(time.Hour / slotDur)
	slotsNeeded := durationHours * slotsPerHour

	if len(prices) < slotsNeeded {
		return nil
	}

	future := sortAndFilterFuture(prices, now)
	if len(future) < slotsNeeded {
		return nil
	}

	result := slidingWindowMin(future, slotsNeeded)
	return buildWindow(future, result, slotsNeeded, durationHours, slotDur)
}

// FindBestWindows returns both the best and runner-up windows.
func FindBestWindows(prices []models.HourlyPrice, now time.Time, durationHours int) (best *models.BestWindow, runnerUp *models.BestWindow) {
	best = FindBestWindow(prices, now, durationHours)
	if best == nil {
		return nil, nil
	}

	slotDur := DetectSlotDuration(prices)
	slotsPerHour := int(time.Hour / slotDur)
	slotsNeeded := durationHours * slotsPerHour

	// Find runner-up: exclude the best window's slots and find next best
	future := sortAndFilterFuture(prices, now)

	var remaining []models.HourlyPrice
	for _, p := range future {
		h := p.Timestamp.Format("15:04")
		if h >= best.StartHour && h < best.EndHour {
			continue
		}
		remaining = append(remaining, p)
	}

	if len(remaining) < slotsNeeded {
		return best, nil
	}

	result := slidingWindowMin(remaining, slotsNeeded)
	runnerUp = buildWindow(remaining, result, slotsNeeded, durationHours, slotDur)
	return best, runnerUp
}
