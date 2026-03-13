package engine

import (
	"sort"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// FindOffpeakWindow finds the longest contiguous offpeak block remaining today.
// For TOU pricing, "offpeak" = the minimum price in the dataset.
// Returns nil if no future offpeak slots exist.
func FindOffpeakWindow(prices []models.HourlyPrice, now time.Time) *models.BestWindow {
	if len(prices) == 0 {
		return nil
	}

	stats := CalcStats(prices)

	// Sort by timestamp
	sorted := make([]models.HourlyPrice, len(prices))
	copy(sorted, prices)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Find contiguous offpeak blocks in future slots
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	type block struct {
		start    time.Time
		end      time.Time
		avgPrice float64
		slots    int
	}

	var blocks []block
	var cur *block

	for _, p := range sorted {
		if p.Timestamp.Before(currentHour) {
			continue
		}
		isOffpeak := p.PriceEUR <= stats.Min+0.0001
		if isOffpeak {
			if cur == nil {
				cur = &block{start: p.Timestamp, end: p.Timestamp.Add(time.Hour), avgPrice: p.PriceEUR, slots: 1}
			} else {
				cur.end = p.Timestamp.Add(time.Hour)
				cur.avgPrice = (cur.avgPrice*float64(cur.slots) + p.PriceEUR) / float64(cur.slots+1)
				cur.slots++
			}
		} else {
			if cur != nil {
				blocks = append(blocks, *cur)
				cur = nil
			}
		}
	}
	if cur != nil {
		blocks = append(blocks, *cur)
	}

	if len(blocks) == 0 {
		return nil
	}

	// Return the longest offpeak block (or first if tied)
	best := blocks[0]
	for _, b := range blocks[1:] {
		if b.slots > best.slots {
			best = b
		}
	}

	return &models.BestWindow{
		StartHour: best.start.Format("15:04"),
		EndHour:   best.end.Format("15:04"),
		AvgPrice:  best.avgPrice,
		Duration:  best.slots,
		TotalCost: best.avgPrice * float64(best.slots),
	}
}
