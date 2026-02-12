package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const ecbBaseURL = "https://data-api.ecb.europa.eu/service/data/EXR"

// ExchangeRates provides EUR to foreign currency conversion.
// Fetches from ECB (European Central Bank). Updated daily on workdays at ~16:00 CET.
type ExchangeRates struct {
	mu    sync.RWMutex
	rates map[string]float64 // "NOK" → 11.265 (1 EUR = X units)
	last  time.Time
}

// NewExchangeRates creates an exchange rate fetcher.
func NewExchangeRates() *ExchangeRates {
	return &ExchangeRates{
		rates: make(map[string]float64),
	}
}

// Fetch retrieves latest EUR exchange rates for the given currencies.
// Uses the ECB SDMX JSON API (httpGet sends Accept: application/json).
func (e *ExchangeRates) Fetch(ctx context.Context, currencies []string) error {
	if len(currencies) == 0 {
		return nil
	}

	joined := strings.Join(currencies, "+")
	url := fmt.Sprintf("%s/D.%s.EUR.SP00.A?lastNObservations=1", ecbBaseURL, joined)

	data, err := httpGet(ctx, url)
	if err != nil {
		return fmt.Errorf("ecb fetch: %w", err)
	}

	// Parse SDMX-JSON response
	var resp ecbResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("ecb parse: %w", err)
	}

	// Find CURRENCY dimension to map series keys to currency codes
	var currencyValues []ecbDimValue
	for _, dim := range resp.Structure.Dimensions.Series {
		if dim.ID == "CURRENCY" {
			currencyValues = dim.Values
			break
		}
	}
	if len(currencyValues) == 0 {
		return fmt.Errorf("ecb: CURRENCY dimension not found")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Series keys are like "0:0:0:0:0" where the second index is CURRENCY
	// (FREQ:CURRENCY:CURRENCY_DENOM:EXR_TYPE:EXR_SUFFIX)
	for _, ds := range resp.DataSets {
		for key, series := range ds.Series {
			parts := strings.Split(key, ":")
			if len(parts) < 2 {
				continue
			}
			currIdx := 0
			fmt.Sscanf(parts[1], "%d", &currIdx)
			if currIdx >= len(currencyValues) {
				continue
			}
			code := currencyValues[currIdx].ID

			// observations is {"0": [value, ...]}
			for _, obs := range series.Observations {
				if len(obs) > 0 {
					e.rates[code] = obs[0]
				}
			}
		}
	}

	e.last = time.Now()
	return nil
}

// ECB SDMX-JSON response types (minimal)
type ecbResponse struct {
	DataSets  []ecbDataSet  `json:"dataSets"`
	Structure ecbStructure  `json:"structure"`
}

type ecbDataSet struct {
	Series map[string]ecbSeries `json:"series"`
}

type ecbSeries struct {
	Observations map[string][]float64 `json:"observations"`
}

type ecbStructure struct {
	Dimensions ecbDimensions `json:"dimensions"`
}

type ecbDimensions struct {
	Series []ecbDimension `json:"series"`
}

type ecbDimension struct {
	ID     string        `json:"id"`
	Values []ecbDimValue `json:"values"`
}

type ecbDimValue struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Convert converts an amount in EUR to the target currency.
// Returns the original amount if the currency is EUR or rate is unknown.
func (e *ExchangeRates) Convert(amountEUR float64, toCurrency string) float64 {
	if toCurrency == "EUR" {
		return amountEUR
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	rate, ok := e.rates[toCurrency]
	if !ok {
		return amountEUR
	}
	return amountEUR * rate
}

// ConvertToEUR converts from a foreign currency to EUR.
// Returns the original amount if the currency is EUR or rate is unknown.
func (e *ExchangeRates) ConvertToEUR(amount float64, fromCurrency string) float64 {
	if fromCurrency == "EUR" {
		return amount
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	rate, ok := e.rates[fromCurrency]
	if !ok || rate == 0 {
		return amount
	}
	return amount / rate
}

// LastFetch returns when rates were last fetched.
func (e *ExchangeRates) LastFetch() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.last
}

// NeedsRefresh returns true if rates are stale (> 24 hours old).
func (e *ExchangeRates) NeedsRefresh() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.last.IsZero() || time.Since(e.last) > 24*time.Hour
}
