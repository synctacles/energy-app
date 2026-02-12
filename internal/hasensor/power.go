package hasensor

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/synctacles/energy-go/internal/ha"
)

// DetectPowerSensor queries HA for a power sensor (W) and returns the best match.
// Priority: 1) entity with "p1" in the name, 2) entity with "power" in the name
// (excluding phase-specific and generation sensors), 3) any sensor with unit W.
// Returns empty string if nothing suitable is found.
func DetectPowerSensor(ctx context.Context, supervisor *ha.SupervisorClient) string {
	states, err := supervisor.GetAllStates(ctx)
	if err != nil {
		slog.Warn("auto-detect power sensor: failed to get states", "error", err)
		return ""
	}

	type candidate struct {
		entityID string
		priority int // lower = better
	}

	var candidates []candidate

	for _, s := range states {
		eid, _ := s["entity_id"].(string)
		if !strings.HasPrefix(eid, "sensor.") {
			continue
		}

		attrs, _ := s["attributes"].(map[string]any)
		if attrs == nil {
			continue
		}

		unit, _ := attrs["unit_of_measurement"].(string)
		if unit != "W" {
			continue
		}

		// Skip phase-specific sensors (phase_1, phase_2, etc.)
		lower := strings.ToLower(eid)
		if strings.Contains(lower, "phase") {
			continue
		}
		// Skip generation/output sensors
		if strings.Contains(lower, "output") || strings.Contains(lower, "solar") ||
			strings.Contains(lower, "export") || strings.Contains(lower, "charging") ||
			strings.Contains(lower, "discharging") || strings.Contains(lower, "battery") {
			continue
		}

		// Verify it has a numeric state
		stateStr, _ := s["state"].(string)
		if _, err := strconv.ParseFloat(stateStr, 64); err != nil {
			continue
		}

		prio := 30 // default priority
		if strings.Contains(lower, "p1") && strings.Contains(lower, "power") {
			prio = 10 // P1 meter total power = best
		} else if strings.Contains(lower, "p1") {
			prio = 15
		} else if strings.Contains(lower, "power") && !strings.Contains(lower, "maximum") {
			prio = 20
		}

		candidates = append(candidates, candidate{eid, prio})
	}

	if len(candidates) == 0 {
		return ""
	}

	// Find best candidate
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.priority < best.priority {
			best = c
		}
	}

	return best.entityID
}

// PowerTracker reads a power sensor from HA and accumulates usage data
// for Live Cost, Savings, and Usage Score calculations.
type PowerTracker struct {
	entityID   string
	supervisor *ha.SupervisorClient

	mu          sync.RWMutex
	currentW    float64   // Latest power reading in Watts
	lastRead    time.Time // When we last read the sensor
	hourlyUsage []hourUsage
	resetDay    int // Day of month when we last reset
}

type hourUsage struct {
	Hour     int     // 0-23
	AvgWatts float64 // Average power during this hour
	PriceEUR float64 // Price during this hour (EUR/kWh)
	Samples  int     // Number of readings
}

// NewPowerTracker creates a tracker for the given HA power sensor entity.
func NewPowerTracker(entityID string, supervisor *ha.SupervisorClient) *PowerTracker {
	return &PowerTracker{
		entityID:    entityID,
		supervisor:  supervisor,
		hourlyUsage: make([]hourUsage, 24),
		resetDay:    time.Now().UTC().Day(),
	}
}

// ReadPower reads the current power from the HA sensor.
// Should be called periodically (e.g. every 15 minutes from the scheduler).
func (t *PowerTracker) ReadPower(ctx context.Context, currentPrice float64) {
	if t.supervisor == nil || t.entityID == "" {
		return
	}

	state, err := t.supervisor.GetState(ctx, t.entityID)
	if err != nil {
		slog.Debug("failed to read power sensor", "entity", t.entityID, "error", err)
		return
	}

	stateStr, _ := state["state"].(string)
	watts, err := strconv.ParseFloat(stateStr, 64)
	if err != nil {
		slog.Debug("invalid power sensor value", "entity", t.entityID, "value", stateStr)
		return
	}

	now := time.Now().UTC()

	t.mu.Lock()
	defer t.mu.Unlock()

	// Reset daily accumulator on day change
	if now.Day() != t.resetDay {
		t.hourlyUsage = make([]hourUsage, 24)
		t.resetDay = now.Day()
	}

	t.currentW = watts
	t.lastRead = now

	// Accumulate hourly usage
	h := now.Hour()
	u := &t.hourlyUsage[h]
	u.Hour = h
	u.PriceEUR = currentPrice
	u.Samples++
	// Running average
	u.AvgWatts = u.AvgWatts + (watts-u.AvgWatts)/float64(u.Samples)
}

// LiveCost returns the current instantaneous cost in EUR/hour.
func (t *PowerTracker) LiveCost(currentPrice float64) (eurPerHour float64, powerW float64, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.currentW == 0 && t.lastRead.IsZero() {
		return 0, 0, false
	}

	// Cost = Power(kW) * Price(EUR/kWh) = EUR/h
	kw := t.currentW / 1000.0
	return kw * currentPrice, t.currentW, true
}

// DailySavings calculates savings vs flat-rate consumption at average price.
// Returns savings in EUR and the total daily kWh consumed.
func (t *PowerTracker) DailySavings(avgPrice float64) (savingsEUR float64, totalKWh float64, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var actualCost, flatCost, totalWh float64
	hasData := false

	for _, u := range t.hourlyUsage {
		if u.Samples == 0 {
			continue
		}
		hasData = true
		kWh := u.AvgWatts / 1000.0 // Approximation: avg watts for 1 hour ≈ kWh
		totalWh += u.AvgWatts
		actualCost += kWh * u.PriceEUR
		flatCost += kWh * avgPrice
	}

	if !hasData {
		return 0, 0, false
	}

	totalKWh = totalWh / 1000.0
	savingsEUR = flatCost - actualCost // Positive = saved money
	return savingsEUR, totalKWh, true
}

// DailyCost returns the cumulative daily electricity cost in EUR.
// Resets at midnight (same as other daily accumulators).
func (t *PowerTracker) DailyCost() (costEUR float64, totalKWh float64, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var totalWh float64
	hasData := false

	for _, u := range t.hourlyUsage {
		if u.Samples == 0 {
			continue
		}
		hasData = true
		kWh := u.AvgWatts / 1000.0
		totalWh += u.AvgWatts
		costEUR += kWh * u.PriceEUR
	}

	if !hasData {
		return 0, 0, false
	}

	totalKWh = totalWh / 1000.0
	return costEUR, totalKWh, true
}

// UsageScore calculates a 0-100 score based on cheap-hour usage.
// 100 = all usage during cheap hours, 0 = all during expensive hours.
func (t *PowerTracker) UsageScore(avgPrice float64) (score int, cheapPct, avgPct, expPct float64, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var cheapW, avgW, expW, totalW float64
	hasData := false

	goThreshold := avgPrice * 0.85    // -15%
	avoidThreshold := avgPrice * 1.20 // +20%

	for _, u := range t.hourlyUsage {
		if u.Samples == 0 || u.PriceEUR == 0 {
			continue
		}
		hasData = true
		totalW += u.AvgWatts

		if u.PriceEUR <= goThreshold {
			cheapW += u.AvgWatts
		} else if u.PriceEUR >= avoidThreshold {
			expW += u.AvgWatts
		} else {
			avgW += u.AvgWatts
		}
	}

	if !hasData || totalW == 0 {
		return 0, 0, 0, 0, false
	}

	cheapPct = cheapW / totalW * 100
	avgPct = avgW / totalW * 100
	expPct = expW / totalW * 100

	// Score: 100 * cheap% - 50 * expensive%, clamped 0-100
	rawScore := cheapPct - expPct*0.5 + 50
	if rawScore > 100 {
		rawScore = 100
	}
	if rawScore < 0 {
		rawScore = 0
	}

	return int(rawScore), cheapPct, avgPct, expPct, true
}
