package engine

import (
	"sort"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// FindBestWindow finds the cheapest contiguous window of the given duration
// from the remaining slots in the day (starting from now).
// Duration is in hours. Handles both PT60 and PT15 data automatically.
// Returns nil if not enough slots remain.
func FindBestWindow(prices []models.HourlyPrice, now time.Time, durationHours int) *models.BestWindow {
	if durationHours <= 0 || len(prices) == 0 {
		return nil
	}

	// Detect slot duration (PT15 = 15min, PT60 = 1h)
	slotDur := DetectSlotDuration(prices)
	slotsPerHour := int(time.Hour / slotDur)
	slotsNeeded := durationHours * slotsPerHour

	if len(prices) < slotsNeeded {
		return nil
	}

	// Sort by timestamp
	sorted := make([]models.HourlyPrice, len(prices))
	copy(sorted, prices)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Filter to future slots only
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	var future []models.HourlyPrice
	for _, p := range sorted {
		if !p.Timestamp.Before(currentHour) {
			future = append(future, p)
		}
	}

	if len(future) < slotsNeeded {
		return nil
	}

	// Sliding window: find the contiguous block with lowest average price
	bestSum := float64(0)
	for i := 0; i < slotsNeeded; i++ {
		bestSum += future[i].PriceEUR
	}

	currentSum := bestSum
	bestStart := 0

	for i := 1; i <= len(future)-slotsNeeded; i++ {
		currentSum = currentSum - future[i-1].PriceEUR + future[i+slotsNeeded-1].PriceEUR
		if currentSum < bestSum {
			bestSum = currentSum
			bestStart = i
		}
	}

	avgPrice := bestSum / float64(slotsNeeded)
	startTime := future[bestStart].Timestamp
	endTime := future[bestStart+slotsNeeded-1].Timestamp.Add(slotDur)

	return &models.BestWindow{
		StartHour: startTime.Format("15:04"),
		EndHour:   endTime.Format("15:04"),
		AvgPrice:  avgPrice,
		Duration:  durationHours,
		TotalCost: avgPrice * float64(durationHours),
	}
}

// FindBestWindows returns both the best and runner-up windows.
func FindBestWindows(prices []models.HourlyPrice, now time.Time, durationHours int) (best *models.BestWindow, runnerUp *models.BestWindow) {
	best = FindBestWindow(prices, now, durationHours)
	if best == nil {
		return nil, nil
	}

	// Detect slot duration for runner-up calculation
	slotDur := DetectSlotDuration(prices)
	slotsPerHour := int(time.Hour / slotDur)
	slotsNeeded := durationHours * slotsPerHour

	// Find runner-up: exclude the best window's slots and find next best
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

	if len(remaining) >= slotsNeeded {
		// Find best window in remaining
		bestSum := float64(0)
		for i := 0; i < slotsNeeded; i++ {
			bestSum += remaining[i].PriceEUR
		}
		currentSum := bestSum
		bestStart := 0

		for i := 1; i <= len(remaining)-slotsNeeded; i++ {
			currentSum = currentSum - remaining[i-1].PriceEUR + remaining[i+slotsNeeded-1].PriceEUR
			if currentSum < bestSum {
				bestSum = currentSum
				bestStart = i
			}
		}

		avgPrice := bestSum / float64(slotsNeeded)
		startTime := remaining[bestStart].Timestamp
		endTime := remaining[bestStart+slotsNeeded-1].Timestamp.Add(slotDur)

		runnerUp = &models.BestWindow{
			StartHour: startTime.Format("15:04"),
			EndHour:   endTime.Format("15:04"),
			AvgPrice:  avgPrice,
			Duration:  durationHours,
			TotalCost: avgPrice * float64(durationHours),
		}
	}

	return best, runnerUp
}
