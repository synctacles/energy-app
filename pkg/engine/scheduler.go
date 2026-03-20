package engine

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// FeatureGate controls whether price fetching is allowed.
type FeatureGate interface {
	CanFetchPrices() bool
}

// Scheduler manages periodic price fetching and sensor updates.
type Scheduler struct {
	fallback     *FallbackManager
	normalizer   *Normalizer
	action       *ActionEngine
	zone         string
	hasWholesale bool // false for non-ENTSO-E zones (e.g. Madeira, Azores)
	pricingMode  string
	updateFn     func(ctx context.Context, prices []models.HourlyPrice, result *FetchResult) error
	stopCh       chan struct{}
	triggerCh    chan struct{}
	gate         FeatureGate

	// Event-driven fetch state: prices are fetched only when needed,
	// but re-normalized every tick for fresh delta application.
	hasTomorrowData bool         // true once tomorrow's prices are available
	lastResult      *FetchResult // cached result from last successful fetch
}

// NewScheduler creates a price fetch scheduler.
func NewScheduler(
	fallback *FallbackManager,
	normalizer *Normalizer,
	action *ActionEngine,
	zone string,
	updateFn func(ctx context.Context, prices []models.HourlyPrice, result *FetchResult) error,
	gate ...FeatureGate,
) *Scheduler {
	s := &Scheduler{
		fallback:   fallback,
		normalizer: normalizer,
		action:     action,
		zone:       zone,
		updateFn:   updateFn,
		stopCh:     make(chan struct{}),
		triggerCh:  make(chan struct{}, 1),
	}
	if len(gate) > 0 {
		s.gate = gate[0]
	}
	return s
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
			// Regular tick: re-normalize with fresh delta, only fetch if needed
			s.fetchAndUpdate(ctx)
		case <-hourTimer.C:
			// Hour boundary: re-normalize for fresh delta application
			slog.Info("hour boundary update")
			s.fetchAndUpdate(ctx)
			nextHour = time.Now().Truncate(time.Hour).Add(time.Hour)
			hourTimer.Reset(time.Until(nextHour))
		case <-dayAheadTimer.C:
			// Day-ahead publication window: always fetch fresh
			slog.Info("day-ahead fetch triggered")
			s.doFetchAndUpdate(ctx, true)
			dayAheadTimer = s.nextDayAheadTimer()
		case <-s.triggerCh:
			// Manual trigger: always fetch fresh
			slog.Info("manual fetch triggered")
			s.doFetchAndUpdate(ctx, true)
		}
	}
}

// Stop signals the scheduler to stop.
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

// TriggerFetch requests an immediate fetch cycle. Non-blocking.
func (s *Scheduler) TriggerFetch() {
	select {
	case s.triggerCh <- struct{}{}:
	default: // trigger already pending
	}
}

// SetZoneInfo configures whether the zone has wholesale data and the pricing mode.
// Non-wholesale zones in fixed/tou mode skip price fetching entirely.
func (s *Scheduler) SetZoneInfo(hasWholesale bool, pricingMode string) {
	s.hasWholesale = hasWholesale
	s.pricingMode = pricingMode
}

// needsFreshFetch decides whether we should hit the API or re-use cached prices.
// Day-ahead prices are published once (~13:00 CET) and don't change after that.
// We only need fresh fetches when:
//   - We don't have tomorrow's data yet AND we're in the day-ahead publication window (13-14 UTC)
//   - We've never fetched successfully (lastResult == nil)
//   - A manual trigger was used (always fetches fresh — handled by caller)
func (s *Scheduler) needsFreshFetch() bool {
	if s.lastResult == nil {
		return true
	}
	hour := time.Now().UTC().Hour()
	// During day-ahead window (13:00-14:00 UTC): fetch if we don't have tomorrow's data
	if hour >= 13 && hour < 14 && !s.hasTomorrowData {
		return true
	}
	return false
}

func (s *Scheduler) fetchAndUpdate(ctx context.Context) {
	s.doFetchAndUpdate(ctx, false)
}

func (s *Scheduler) doFetchAndUpdate(ctx context.Context, forceFresh bool) {
	if s.gate != nil && !s.gate.CanFetchPrices() {
		slog.Debug("price fetch skipped — install purged")
		return
	}
	// Non-wholesale zones with fixed/TOU pricing don't need external price data
	if !s.hasWholesale && (s.pricingMode == "fixed" || s.pricingMode == "tou") {
		slog.Debug("price fetch skipped — non-wholesale zone with regulated pricing", "zone", s.zone, "mode", s.pricingMode)
		return
	}

	if !forceFresh && !s.needsFreshFetch() && s.lastResult != nil {
		// Re-normalize existing prices with fresh delta (no API call)
		slog.Debug("re-normalizing cached prices (no fetch needed)", "zone", s.zone, "source", s.lastResult.Source)
		consumerPrices := s.normalizer.ToConsumer(s.lastResult.Prices)
		if s.updateFn != nil {
			if err := s.updateFn(ctx, consumerPrices, s.lastResult); err != nil {
				slog.Error("update callback failed", "error", err)
			}
		}
		return
	}

	result, err := s.fallback.Fetch(ctx, s.zone, time.Now().UTC())
	if err != nil {
		slog.Error("price fetch failed", "zone", s.zone, "error", err)
		return
	}

	// Track whether we have tomorrow's data
	s.lastResult = result
	s.hasTomorrowData = hasTomorrowPrices(result.Prices)

	// Normalize to consumer prices
	consumerPrices := s.normalizer.ToConsumer(result.Prices)

	if s.updateFn != nil {
		if err := s.updateFn(ctx, consumerPrices, result); err != nil {
			slog.Error("update callback failed", "error", err)
		}
	}
}

// hasTomorrowPrices checks if any price in the result is for tomorrow (UTC).
func hasTomorrowPrices(prices []models.HourlyPrice) bool {
	tomorrow := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	for _, p := range prices {
		if !p.Timestamp.Before(tomorrow) {
			return true
		}
	}
	return false
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
