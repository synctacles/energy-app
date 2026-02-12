package engine

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/synctacles/energy-go/internal/models"
)

// Scheduler manages periodic price fetching and sensor updates.
type Scheduler struct {
	fallback    *FallbackManager
	normalizer  *Normalizer
	action      *ActionEngine
	zone        string
	updateFn    func(ctx context.Context, prices []models.HourlyPrice, result *FetchResult) error
	stopCh      chan struct{}
}

// NewScheduler creates a price fetch scheduler.
func NewScheduler(
	fallback *FallbackManager,
	normalizer *Normalizer,
	action *ActionEngine,
	zone string,
	updateFn func(ctx context.Context, prices []models.HourlyPrice, result *FetchResult) error,
) *Scheduler {
	return &Scheduler{
		fallback:   fallback,
		normalizer: normalizer,
		action:     action,
		zone:       zone,
		updateFn:   updateFn,
		stopCh:     make(chan struct{}),
	}
}

// Run starts the scheduler loop. Blocks until Stop() is called or ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	// Initial fetch immediately
	s.fetchAndUpdate(ctx)

	// Calculate time until next hour boundary for instant updates
	nextHour := time.Now().Truncate(time.Hour).Add(time.Hour)
	hourTimer := time.NewTimer(time.Until(nextHour))
	defer hourTimer.Stop()

	// Regular 15-minute refresh
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	// Day-ahead fetch: 13:00 CET + random jitter (0-30 min)
	dayAheadTimer := s.nextDayAheadTimer()
	defer dayAheadTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.fetchAndUpdate(ctx)
		case <-hourTimer.C:
			// Instant update on hour boundary
			slog.Info("hour boundary update")
			s.fetchAndUpdate(ctx)
			nextHour = time.Now().Truncate(time.Hour).Add(time.Hour)
			hourTimer.Reset(time.Until(nextHour))
		case <-dayAheadTimer.C:
			slog.Info("day-ahead fetch triggered")
			s.fetchAndUpdate(ctx)
			dayAheadTimer = s.nextDayAheadTimer()
		}
	}
}

// Stop signals the scheduler to stop.
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) fetchAndUpdate(ctx context.Context) {
	result, err := s.fallback.Fetch(ctx, s.zone, time.Now().UTC())
	if err != nil {
		slog.Error("price fetch failed", "zone", s.zone, "error", err)
		return
	}

	// Normalize to consumer prices
	consumerPrices := s.normalizer.ToConsumer(result.Prices)

	if s.updateFn != nil {
		if err := s.updateFn(ctx, consumerPrices, result); err != nil {
			slog.Error("update callback failed", "error", err)
		}
	}
}

// nextDayAheadTimer returns a timer that fires at 13:00 CET + 0-30min jitter.
// This is when EPEX Spot publishes day-ahead prices.
func (s *Scheduler) nextDayAheadTimer() *time.Timer {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	target := time.Date(now.Year(), now.Month(), now.Day(), 13, 0, 0, 0, loc)

	// Add random jitter (0-30 minutes) to prevent thundering herd
	jitter := time.Duration(rand.IntN(30)) * time.Minute
	target = target.Add(jitter)

	// If target is in the past, schedule for tomorrow
	if target.Before(now) {
		target = target.Add(24 * time.Hour)
	}

	return time.NewTimer(time.Until(target))
}
