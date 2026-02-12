package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/synctacles/energy-go/internal/models"
)

const easyEnergyBaseURL = "https://mijn.easyenergy.com/nl/api/tariff/getapxtariffs"

// EasyEnergy fetches Dutch wholesale electricity prices from the APX exchange.
// TariffUsage = wholesale (APX/EPEX) + supplier markup in EUR/kWh.
// Does NOT include energy tax or VAT — normalizer must apply those.
type EasyEnergy struct{}

func (e *EasyEnergy) Name() string        { return "easyenergy" }
func (e *EasyEnergy) Zones() []string      { return []string{"NL"} }
func (e *EasyEnergy) RequiresKey() bool    { return false }

// FetchDayAhead fetches hourly prices for the given date.
// EasyEnergy returns wholesale + supplier markup in EUR/kWh (excl. energy tax and VAT).
func (e *EasyEnergy) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	url := fmt.Sprintf("%s?startTimestamp=%s&endTimestamp=%s",
		easyEnergyBaseURL,
		start.Format("2006-01-02T15:04:05.000Z"),
		end.Format("2006-01-02T15:04:05.000Z"),
	)

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("easyenergy fetch: %w", err)
	}

	var raw []struct {
		Timestamp    string  `json:"Timestamp"`
		TariffUsage  float64 `json:"TariffUsage"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("easyenergy parse: %w", err)
	}

	prices := make([]models.HourlyPrice, 0, len(raw))
	for _, r := range raw {
		ts, err := time.Parse(time.RFC3339, r.Timestamp)
		if err != nil {
			// Try alternative format
			ts, err = time.Parse("2006-01-02T15:04:05-07:00", r.Timestamp)
			if err != nil {
				continue
			}
		}
		prices = append(prices, models.HourlyPrice{
			Timestamp: ts.UTC(),
			PriceEUR:  r.TariffUsage,
			Unit:      models.UnitKWh,
			Source:    "easyenergy",
			Quality:   "live",
			Zone:      zone,
			// IsConsumer: false — TariffUsage ≈ wholesale + margin, needs energy tax + VAT
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("easyenergy: no prices returned for %s", date.Format("2006-01-02"))
	}
	return prices, nil
}
