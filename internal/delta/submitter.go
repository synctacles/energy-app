package delta

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/synctacles/energy-app/pkg/engine"
	"github.com/synctacles/energy-app/pkg/platform"
)

const energyDataBaseURL = "https://energy-data.synctacles.com"

// SubmitterConfig configures the delta submitter.
type SubmitterConfig struct {
	InstallUUID string
	Zone        string
	Source      string // "sensor" or "enever"
	TaxCache    *engine.TaxProfileCache

	// GetDayAheadPrices returns all known consumer prices with their timestamps.
	// For sensor mode: reads forecast from HA sensor attributes.
	// For Enever mode: returns prices from Enever API for a specific supplier.
	GetDayAheadPrices func(ctx context.Context, supplier string) ([]HourlyConsumerPrice, error)

	// GetWholesalePrices returns ENTSO-E day-ahead wholesale prices (EUR/kWh).
	GetWholesalePrices func(ctx context.Context, zone string) ([]WholesalePrice, error)

	// Suppliers returns the list of suppliers to submit deltas for.
	// For sensor mode: returns a single supplier.
	// For Enever mode: returns all 23 NL suppliers.
	Suppliers func() []string
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

// Submitter submits per-hour supplier deltas to the energy-data Worker.
type Submitter struct {
	cfg SubmitterConfig
}

// NewSubmitter creates a new delta submitter.
func NewSubmitter(cfg SubmitterConfig) *Submitter {
	return &Submitter{cfg: cfg}
}

// Run starts the delta submitter loop. It submits once after startup
// stabilization, then every hour.
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

	// Index wholesale by hour
	wsMap := make(map[string]float64, len(wholesale))
	for _, w := range wholesale {
		key := w.Timestamp.UTC().Format("2006-01-02T15")
		wsMap[key] = w.PriceKWh
	}

	for _, supplier := range s.cfg.Suppliers() {
		s.submitForSupplier(ctx, supplier, tax, wsMap)
	}
}

// submitForSupplier calculates and submits deltas for a single supplier.
func (s *Submitter) submitForSupplier(ctx context.Context, supplier string, tax *engine.WorkerTaxOverride, wsMap map[string]float64) {
	consumer, err := s.cfg.GetDayAheadPrices(ctx, supplier)
	if err != nil || len(consumer) == 0 {
		slog.Debug("delta: consumer prices unavailable", "supplier", supplier, "error", err)
		return
	}

	var deltas []deltaEntry

	for _, cp := range consumer {
		hourKey := cp.Timestamp.UTC().Format("2006-01-02T15")
		ws, ok := wsMap[hourKey]
		if !ok {
			continue
		}

		// delta = consumer/(1+VAT) - wholesale - energy_tax - surcharges
		exclVAT := cp.PriceKWh / (1 + tax.VATRate)
		d := exclVAT - ws - tax.EnergyTax - tax.Surcharges

		// Round to 6 decimals
		d = float64(int(d*1000000+0.5)) / 1000000

		ts := cp.Timestamp.UTC().Format("2006-01-02T15:04:05Z")
		deltas = append(deltas, deltaEntry{TS: ts, Delta: d})
	}

	if len(deltas) == 0 {
		slog.Debug("delta: no matching hours", "supplier", supplier)
		return
	}

	// Cap at 48 entries
	if len(deltas) > 48 {
		deltas = deltas[:48]
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
