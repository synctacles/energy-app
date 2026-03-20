package delta

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/synctacles/energy-app/pkg/engine"
	"github.com/synctacles/energy-app/pkg/platform"
)

// energyDataBaseURL is the base URL for the Synctacles Energy Data Worker.
var energyDataBaseURL = platform.EnergyDataBaseURL

// SubmitterConfig configures the delta submitter.
type SubmitterConfig struct {
	InstallUUID string
	Zone        string
	Source      string // "sensor"
	TaxCache    *engine.TaxProfileCache

	// GetDayAheadPrices returns all known consumer prices with their timestamps.
	// Reads forecast from HA sensor attributes.
	GetDayAheadPrices func(ctx context.Context, supplier string) ([]HourlyConsumerPrice, error)

	// GetWholesalePrices returns day-ahead wholesale prices (EUR/kWh).
	GetWholesalePrices func(ctx context.Context, zone string) ([]WholesalePrice, error)

	// Suppliers returns the list of suppliers to submit deltas for.
	// Returns a single supplier for sensor mode.
	Suppliers func() []string

	// ReadLivePrice reads the current live sensor price (EUR/kWh incl. VAT).
	// Optional — when set, enables event-driven live correction.
	ReadLivePrice func(ctx context.Context) (float64, error)
}

// HourlyConsumerPrice is a consumer price at a specific hour.
type HourlyConsumerPrice struct {
	Timestamp time.Time
	PriceKWh  float64 // EUR/kWh incl. VAT
}

// WholesalePrice is a wholesale price at a specific hour.
type WholesalePrice struct {
	Timestamp time.Time
	PriceKWh  float64 // EUR/kWh
}

// liveCorrectionThreshold: 0.25 ct/kWh — only submit correction when deviation exceeds this
const liveCorrectionThreshold = 0.0025

// Submitter submits per-hour supplier deltas to the energy-data Worker.
type Submitter struct {
	cfg       SubmitterConfig
	wsCache   map[string]float64 // cached wholesale: hourKey → EUR/kWh
	lastDelta map[string]float64 // last submitted delta per hourKey
}

// NewSubmitter creates a new delta submitter.
func NewSubmitter(cfg SubmitterConfig) *Submitter {
	return &Submitter{
		cfg:       cfg,
		wsCache:   make(map[string]float64),
		lastDelta: make(map[string]float64),
	}
}

// Run starts the delta submitter loop. It submits once after startup
// stabilization, then every hour. submitAll includes a live correction
// for the current hour when a sensor is available.
func (s *Submitter) Run(ctx context.Context) {
	// Wait for prices to load and stabilize
	select {
	case <-ctx.Done():
		return
	case <-time.After(3 * time.Minute):
	}

	s.submitAll(ctx)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.submitAll(ctx)
		}
	}
}

// submitAll calculates and submits deltas for all configured suppliers.
func (s *Submitter) submitAll(ctx context.Context) {
	tax := s.cfg.TaxCache.Get(s.cfg.Zone)
	if tax == nil {
		slog.Debug("delta: no tax profile", "zone", s.cfg.Zone)
		return
	}

	// Fetch wholesale prices
	wholesale, err := s.cfg.GetWholesalePrices(ctx, s.cfg.Zone)
	if err != nil || len(wholesale) == 0 {
		slog.Debug("delta: wholesale unavailable", "zone", s.cfg.Zone, "error", err)
		return
	}

	// Index wholesale by hour (average of quarters) + cache for live corrections.
	// PT15M data has 4 entries per hour — we need the hourly average to match
	// how consumer prices are averaged on the display side.
	wsSums := make(map[string]float64, len(wholesale))
	wsCounts := make(map[string]int, len(wholesale))
	for _, w := range wholesale {
		key := w.Timestamp.UTC().Format("2006-01-02T15")
		wsSums[key] += w.PriceKWh
		wsCounts[key]++
	}
	wsMap := make(map[string]float64, len(wsSums))
	for key, sum := range wsSums {
		wsMap[key] = sum / float64(wsCounts[key])
	}
	s.wsCache = wsMap

	for _, supplier := range s.cfg.Suppliers() {
		s.submitForSupplier(ctx, supplier, tax, wsMap)
	}

	// Immediately apply live correction for current hour if sensor available.
	// The forecast may differ from the live sensor — correct it now instead
	// of waiting for the 15-min live check ticker.
	s.checkLiveCorrection(ctx)
}

// checkLiveCorrection reads the live sensor price and submits a corrected
// delta for the current hour if it deviates from the forecast-based delta
// by more than the threshold (0.5 ct/kWh).
func (s *Submitter) checkLiveCorrection(ctx context.Context) {
	if s.cfg.ReadLivePrice == nil {
		return
	}

	tax := s.cfg.TaxCache.Get(s.cfg.Zone)
	if tax == nil {
		slog.Debug("delta: live check skipped — no tax profile")
		return
	}

	livePrice, err := s.cfg.ReadLivePrice(ctx)
	if err != nil || livePrice <= 0 {
		slog.Info("delta: live check — sensor read failed", "error", err, "price", livePrice)
		return
	}

	// Current hour wholesale
	now := time.Now().UTC()
	hourKey := now.Format("2006-01-02T15")
	ws, ok := s.wsCache[hourKey]
	if !ok {
		slog.Info("delta: live check — no wholesale for hour", "hour", hourKey, "cache_size", len(s.wsCache))
		return
	}

	slog.Info("delta: live check", "hour", hourKey, "live_price", livePrice, "wholesale", ws)

	// Calculate live delta: (live_price / (1+VAT)) - wholesale - taxes
	liveDelta := livePrice/(1+tax.VATRate) - ws - tax.EnergyTax - tax.Surcharges
	liveDelta = float64(int(liveDelta*1000000+0.5)) / 1000000

	// Compare with last submitted delta — only submit if meaningfully different
	lastDelta, hasLast := s.lastDelta[hourKey]
	if hasLast && math.Abs(liveDelta-lastDelta) < liveCorrectionThreshold {
		return
	}

	// Submit correction for current hour only
	hourTS := now.Truncate(time.Hour).Format("2006-01-02T15:04:05Z")
	suppliers := s.cfg.Suppliers()
	if len(suppliers) == 0 {
		return
	}

	s.submit(ctx, suppliers[0], []deltaEntry{{TS: hourTS, Delta: liveDelta}})
	s.lastDelta[hourKey] = liveDelta

	slog.Info("delta: live correction submitted",
		"supplier", suppliers[0],
		"hour", hourKey,
		"old_delta", lastDelta,
		"new_delta", liveDelta,
		"deviation", math.Abs(liveDelta-lastDelta))
}


// submitForSupplier calculates and submits deltas for a single supplier.
func (s *Submitter) submitForSupplier(ctx context.Context, supplier string, tax *engine.WorkerTaxOverride, wsMap map[string]float64) {
	consumer, err := s.cfg.GetDayAheadPrices(ctx, supplier)
	if err != nil || len(consumer) == 0 {
		slog.Debug("delta: consumer prices unavailable", "supplier", supplier, "error", err)
		return
	}

	// Collect deltas per hour (PT15M data may have multiple entries per hour)
	type hourAccum struct {
		sum   float64
		count int
	}
	hourMap := make(map[string]*hourAccum)
	var hourOrder []string

	for _, cp := range consumer {
		hourKey := cp.Timestamp.UTC().Format("2006-01-02T15")
		ws, ok := wsMap[hourKey]
		if !ok {
			continue
		}

		// delta = consumer/(1+VAT) - wholesale - energy_tax - surcharges
		exclVAT := cp.PriceKWh / (1 + tax.VATRate)
		d := exclVAT - ws - tax.EnergyTax - tax.Surcharges

		hourTS := cp.Timestamp.UTC().Truncate(time.Hour).Format("2006-01-02T15:04:05Z")
		if _, exists := hourMap[hourTS]; !exists {
			hourMap[hourTS] = &hourAccum{}
			hourOrder = append(hourOrder, hourTS)
		}
		hourMap[hourTS].sum += d
		hourMap[hourTS].count++
	}

	var deltas []deltaEntry
	for _, ts := range hourOrder {
		acc := hourMap[ts]
		avg := acc.sum / float64(acc.count)
		// Round to 6 decimals
		avg = float64(int(avg*1000000+0.5)) / 1000000
		deltas = append(deltas, deltaEntry{TS: ts, Delta: avg})
	}

	if len(deltas) == 0 {
		slog.Debug("delta: no matching hours", "supplier", supplier)
		return
	}

	// Cap at 48 entries
	if len(deltas) > 48 {
		deltas = deltas[:48]
	}

	// Update lastDelta cache (for live correction threshold comparison)
	for _, d := range deltas {
		s.lastDelta[d.TS[:13]] = d.Delta // "2006-01-02T15" from "2006-01-02T15:04:05Z"
	}

	s.submit(ctx, supplier, deltas)
}

type deltaEntry struct {
	TS    string  `json:"ts"`
	Delta float64 `json:"delta"`
}

// submit sends the delta batch to the energy-data Worker.
func (s *Submitter) submit(ctx context.Context, supplier string, deltas []deltaEntry) {
	payload := map[string]any{
		"install_uuid": s.cfg.InstallUUID,
		"zone":         s.cfg.Zone,
		"supplier":     supplier,
		"source":       s.cfg.Source,
		"deltas":       deltas,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Debug("delta: marshal failed", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST",
		energyDataBaseURL+"/api/v1/energy/supplier-deltas", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SynctaclesEnergy/delta")
	platform.SignRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Debug("delta: submit failed", "supplier", supplier, "error", err)
		return
	}
	defer resp.Body.Close()

	slog.Info("delta: submitted",
		"source", s.cfg.Source,
		"zone", s.cfg.Zone,
		"supplier", supplier,
		"hours", len(deltas),
		"status", resp.StatusCode)
}
