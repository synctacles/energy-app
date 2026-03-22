package engine

import (
	"fmt"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// GreenWindow represents the greenest contiguous time window.
type GreenWindow struct {
	StartHour string  `json:"start_hour"`
	EndHour   string  `json:"end_hour"`
	AvgShare  float64 `json:"avg_share"`  // average renewable % in window
	Duration  int     `json:"duration_h"` // hours
}

// FindGreenestWindow finds the contiguous window with the highest average
// renewable share from the forecast data. Duration is in hours.
// Returns nil if not enough data points.
func FindGreenestWindow(data []models.RenewablePoint, now time.Time, durationHours int) *GreenWindow {
	if durationHours <= 0 || len(data) == 0 {
		return nil
	}

	// Detect slot duration from data spacing
	slotDur := detectRenewableSlotDuration(data)
	slotsPerHour := int(time.Hour / slotDur)
	slotsNeeded := durationHours * slotsPerHour

	// Filter to future slots only
	future := filterFutureRenewable(data, now)
	if len(future) < slotsNeeded {
		return nil
	}

	// Sliding window maximum
	bestSum := float64(0)
	for i := 0; i < slotsNeeded; i++ {
		bestSum += future[i].RenShare
	}
	currentSum := bestSum
	bestStart := 0

	for i := 1; i <= len(future)-slotsNeeded; i++ {
		currentSum = currentSum - future[i-1].RenShare + future[i+slotsNeeded-1].RenShare
		if currentSum > bestSum {
			bestSum = currentSum
			bestStart = i
		}
	}

	startTime := future[bestStart].Timestamp
	endTime := future[bestStart+slotsNeeded-1].Timestamp.Add(slotDur)

	return &GreenWindow{
		StartHour: startTime.Format("15:04"),
		EndHour:   endTime.Format("15:04"),
		AvgShare:  bestSum / float64(slotsNeeded),
		Duration:  durationHours,
	}
}

// FindGreenCheapOverlap finds hours that are in BOTH the cheapest price window
// AND the greenest renewable window. Returns start, end, and count of overlapping hours.
// Returns nil if no overlap.
func FindGreenCheapOverlap(
	bestWindow *models.BestWindow,
	greenWindow *GreenWindow,
) *OverlapWindow {
	if bestWindow == nil || greenWindow == nil {
		return nil
	}

	// Build sets of hours covered by each window
	cheapHours := hoursInRange(bestWindow.StartHour, bestWindow.EndHour)
	greenHours := hoursInRange(greenWindow.StartHour, greenWindow.EndHour)

	// Find intersection
	var overlap []string
	greenSet := make(map[string]bool, len(greenHours))
	for _, h := range greenHours {
		greenSet[h] = true
	}
	for _, h := range cheapHours {
		if greenSet[h] {
			overlap = append(overlap, h)
		}
	}

	if len(overlap) == 0 {
		return nil
	}

	return &OverlapWindow{
		StartHour: overlap[0],
		EndHour:   addOneHour(overlap[len(overlap)-1]),
		Hours:     len(overlap),
	}
}

// OverlapWindow represents hours that are both cheap and green.
type OverlapWindow struct {
	StartHour string `json:"start_hour"`
	EndHour   string `json:"end_hour"`
	Hours     int    `json:"hours"`
}

// hoursInRange returns all HH:00 hours from start to end (exclusive).
func hoursInRange(start, end string) []string {
	sh, _ := parseHHMM(start)
	eh, _ := parseHHMM(end)

	var hours []string
	for h := sh; h != eh; h = (h + 1) % 24 {
		hours = append(hours, fmt.Sprintf("%02d:00", h))
		if len(hours) > 24 {
			break // safety
		}
	}
	return hours
}

// addOneHour adds one hour to HH:MM string.
func addOneHour(hhmm string) string {
	h, _ := parseHHMM(hhmm)
	return fmt.Sprintf("%02d:00", (h+1)%24)
}

// ParseHHMMPublic parses "HH:MM" and returns the hour component.
// Public wrapper for cross-package use.
func ParseHHMMPublic(s string) (int, error) {
	return parseHHMM(s)
}

func detectRenewableSlotDuration(data []models.RenewablePoint) time.Duration {
	if len(data) < 2 {
		return time.Hour
	}
	diff := data[1].Timestamp.Sub(data[0].Timestamp)
	if diff == 15*time.Minute {
		return 15 * time.Minute
	}
	return time.Hour
}

func filterFutureRenewable(data []models.RenewablePoint, now time.Time) []models.RenewablePoint {
	currentSlot := now.Truncate(15 * time.Minute)
	var future []models.RenewablePoint
	for _, d := range data {
		if !d.Timestamp.Before(currentSlot) {
			future = append(future, d)
		}
	}
	return future
}
