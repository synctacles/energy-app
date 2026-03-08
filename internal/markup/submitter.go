// Package markup submits supplier markup measurements to the energy-data Worker
// for crowdsourced EMA tracking. Supports multiple sources (energy-app#40):
//   - "enever": NL consumer prices from Enever API (23 suppliers)
//   - "sensor": tariff sensor readings from HA integrations (Tibber, Octopus, etc.)
//   - "manual": user-entered calibration (planned, stap 3)
package markup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/synctacles/energy-app/pkg/engine"
	"github.com/synctacles/energy-app/pkg/platform"
)

const energyDataBaseURL = "https://energy-data.synctacles.com"

// SubmitterConfig holds the configuration for the markup submitter.
type SubmitterConfig struct {
	InstallUUID string
	Zone        string
	Supplier    string // Supplier key (e.g. "anwb", "tibber", "octopus")
	Source      string // "enever" or "sensor" — sent to Worker for alpha selection

	// GetConsumerPrice returns the current consumer price (EUR/kWh incl VAT).
	// Returns (price, true) when a valid price is available, (0, false) otherwise.
	GetConsumerPrice func() (price float64, ok bool)

	// TaxCache provides the Worker-provided tax profile for markup calculation.
	TaxCache *engine.TaxProfileCache
}

// Submitter periodically calculates supplier markup from consumer prices
// and submits it to the energy-data Worker for EMA aggregation.
type Submitter struct {
	cfg SubmitterConfig
}

// NewSubmitter creates a new markup submitter.
func NewSubmitter(cfg SubmitterConfig) *Submitter {
	return &Submitter{cfg: cfg}
}

// Run starts the submitter loop. It waits for the app to stabilize,
// then runs every 3 hours (matching the Care harvester interval).
func (s *Submitter) Run(ctx context.Context) {
	// Wait for prices to load and stabilize
	select {
	case <-ctx.Done():
		return
	case <-time.After(3 * time.Minute):
	}

	// First measurement
	s.measure(ctx)

	ticker := time.NewTicker(3 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.measure(ctx)
		}
	}
}

// measure calculates the supplier markup and submits it to the energy-data Worker.
func (s *Submitter) measure(ctx context.Context) {
	price, ok := s.cfg.GetConsumerPrice()
	if !ok || price <= 0 {
		slog.Debug("markup: skipping, no valid consumer price", "source", s.cfg.Source)
		return
	}

	// Get tax profile for markup calculation
	tax := s.cfg.TaxCache.Get(s.cfg.Zone)
	if tax == nil {
		slog.Debug("markup: skipping, no tax profile cached", "zone", s.cfg.Zone)
		return
	}

	// Fetch current wholesale price from energy-data Worker
	wholesaleKWh, err := s.fetchCurrentWholesale(ctx, s.cfg.Zone)
	if err != nil || wholesaleKWh == nil {
		slog.Debug("markup: wholesale unavailable, skipping", "zone", s.cfg.Zone, "error", err)
		return
	}

	// Calculate markup:
	//   markup = consumer / (1 + VAT) - wholesale - energy_tax - surcharges
	exclVAT := price / (1 + tax.VATRate)
	markup := exclVAT - *wholesaleKWh - tax.EnergyTax - tax.Surcharges

	// Round to 6 decimals
	markup = float64(int(markup*1000000+0.5)) / 1000000

	slog.Debug("markup: calculated",
		"source", s.cfg.Source,
		"consumer", price,
		"wholesale_kwh", *wholesaleKWh,
		"vat", tax.VATRate,
		"energy_tax", tax.EnergyTax,
		"surcharges", tax.Surcharges,
		"markup", markup)

	s.submitMarkup(ctx, markup)
}

// pricesResponse matches the energy-data Worker GET /api/v1/energy/prices response.
type pricesResponse struct {
	Prices []struct {
		Price float64 `json:"price"` // EUR/MWh
	} `json:"prices"`
}

// fetchCurrentWholesale fetches the current wholesale price from the energy-data Worker.
// Returns the price in EUR/kWh (converted from EUR/MWh).
func (s *Submitter) fetchCurrentWholesale(ctx context.Context, zone string) (*float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	today := time.Now().UTC().Format("2006-01-02")
	req, err := http.NewRequestWithContext(ctx, "GET",
		energyDataBaseURL+"/api/v1/energy/prices?zone="+zone+"&from="+today+"&to="+today, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "SynctaclesEnergy/markup")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("prices endpoint returned %d", resp.StatusCode)
	}

	var result pricesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Prices) == 0 {
		return nil, fmt.Errorf("no prices available for zone %s", zone)
	}

	// Find the price for the current hour
	hour := time.Now().UTC().Hour()
	idx := hour
	if idx >= len(result.Prices) {
		idx = len(result.Prices) - 1
	}
	kwh := result.Prices[idx].Price / 1000.0
	return &kwh, nil
}

// SubmitOnce sends a single markup measurement to the energy-data Worker (fire-and-forget).
// Used for manual calibration — runs in a background goroutine to avoid blocking the HTTP handler.
func SubmitOnce(installUUID, zone, supplier string, markupKWh float64) {
	go func() {
		s := &Submitter{cfg: SubmitterConfig{
			InstallUUID: installUUID,
			Zone:        zone,
			Supplier:    supplier,
			Source:      "manual",
		}}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.submitMarkup(ctx, markupKWh)
	}()
}

// submitMarkup sends the markup measurement to the energy-data Worker for EMA tracking.
func (s *Submitter) submitMarkup(ctx context.Context, markup float64) {
	payload := map[string]any{
		"install_uuid":     s.cfg.InstallUUID,
		"zone":             s.cfg.Zone,
		"contract_type":    "dynamic",
		"supplier_name":    s.cfg.Supplier,
		"total_markup_kwh": markup,
		"source":           s.cfg.Source,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Debug("markup: marshal failed", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST",
		energyDataBaseURL+"/api/v1/energy/submit-price", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SynctaclesEnergy/markup")
	platform.SignRequest(req, body)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Debug("markup: submit failed", "error", err)
		return
	}
	defer resp.Body.Close()

	slog.Info("markup: submitted to energy-data",
		"source", s.cfg.Source,
		"zone", s.cfg.Zone,
		"supplier", s.cfg.Supplier,
		"markup", markup,
		"status", resp.StatusCode)
}
