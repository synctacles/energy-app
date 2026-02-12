package engine

import (
	"sort"
	"time"

	"github.com/synctacles/energy-go/internal/models"
)

// FindBestWindow finds the cheapest contiguous window of the given duration
// from the remaining hours in the day (starting from now).
// Duration is in hours. Returns nil if not enough hours remain.
func FindBestWindow(prices []models.HourlyPrice, now time.Time, durationHours int) *models.BestWindow {
	if durationHours <= 0 || len(prices) < durationHours {
		return nil
	}

	// Sort by timestamp
	sorted := make([]models.HourlyPrice, len(prices))
	copy(sorted, prices)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Filter to future hours only
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	var future []models.HourlyPrice
	for _, p := range sorted {
		if !p.Timestamp.Before(currentHour) {
			future = append(future, p)
		}
	}

	if len(future) < durationHours {
		return nil
	}

	// Sliding window: find the contiguous block with lowest average price
	bestSum := float64(0)
	for i := 0; i < durationHours; i++ {
		bestSum += future[i].PriceEUR
	}

	currentSum := bestSum
	bestStart := 0

	for i := 1; i <= len(future)-durationHours; i++ {
		currentSum = currentSum - future[i-1].PriceEUR + future[i+durationHours-1].PriceEUR
		if currentSum < bestSum {
			bestSum = currentSum
			bestStart = i
		}
	}

	avgPrice := bestSum / float64(durationHours)
	startTime := future[bestStart].Timestamp
	endTime := future[bestStart+durationHours-1].Timestamp.Add(time.Hour)

	return &models.BestWindow{
		StartHour: startTime.Format("15:04"),
		EndHour:   endTime.Format("15:04"),
		AvgPrice:  avgPrice,
		Duration:  durationHours,
		TotalCost: bestSum,
	}
}

// FindBestWindows returns both the best and runner-up windows.
func FindBestWindows(prices []models.HourlyPrice, now time.Time, durationHours int) (best *models.BestWindow, runnerUp *models.BestWindow) {
	best = FindBestWindow(prices, now, durationHours)
	if best == nil {
		return nil, nil
	}

	// Find runner-up: exclude the best window's hours and find next best
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	sorted := make([]models.HourlyPrice, len(prices))
	copy(sorted, prices)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Parse best window times to exclude
	bestStartH := best.StartHour
	bestEndH := best.EndHour

	var remaining []models.HourlyPrice
	for _, p := range sorted {
		if p.Timestamp.Before(currentHour) {
			continue
		}
		h := p.Timestamp.Format("15:04")
		if h >= bestStartH && h < bestEndH {
			continue
		}
		remaining = append(remaining, p)
	}

	if len(remaining) >= durationHours {
		// Find best window in remaining
		bestSum := float64(0)
		for i := 0; i < durationHours; i++ {
			bestSum += remaining[i].PriceEUR
		}
		currentSum := bestSum
		bestStart := 0

		for i := 1; i <= len(remaining)-durationHours; i++ {
			currentSum = currentSum - remaining[i-1].PriceEUR + remaining[i+durationHours-1].PriceEUR
			if currentSum < bestSum {
				bestSum = currentSum
				bestStart = i
			}
		}

		avgPrice := bestSum / float64(durationHours)
		startTime := remaining[bestStart].Timestamp
		endTime := remaining[bestStart+durationHours-1].Timestamp.Add(time.Hour)

		runnerUp = &models.BestWindow{
			StartHour: startTime.Format("15:04"),
			EndHour:   endTime.Format("15:04"),
			AvgPrice:  avgPrice,
			Duration:  durationHours,
			TotalCost: bestSum,
		}
	}

	return best, runnerUp
}
