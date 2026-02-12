package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/synctacles/energy-go/internal/models"
)

const frankEnergieURL = "https://graphql.frankenergie.nl"

const frankQuery = `{
  marketPricesElectricity(startDate: "%s", endDate: "%s") {
    from
    till
    marketPrice
    marketPriceTax
    sourcingMarkupPrice
    energyTaxPrice
  }
}`

// FrankEnergie fetches Dutch consumer electricity prices via GraphQL.
type FrankEnergie struct{}

func (f *FrankEnergie) Name() string        { return "frank" }
func (f *FrankEnergie) Zones() []string      { return []string{"NL"} }
func (f *FrankEnergie) RequiresKey() bool    { return false }

// FetchDayAhead fetches hourly prices for the given date.
// Frank returns all price components in EUR/kWh; sum = consumer price.
func (f *FrankEnergie) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	startDate := date.Format("2006-01-02")
	endDate := date.Add(24 * time.Hour).Format("2006-01-02")

	query := fmt.Sprintf(frankQuery, startDate, endDate)
	payload := fmt.Sprintf(`{"query":%q}`, query)

	data, err := httpPost(ctx, frankEnergieURL, "application/json", strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("frank fetch: %w", err)
	}

	var resp struct {
		Data struct {
			MarketPricesElectricity []struct {
				From               string   `json:"from"`
				MarketPrice        *float64 `json:"marketPrice"`
				MarketPriceTax     *float64 `json:"marketPriceTax"`
				SourcingMarkupPrice *float64 `json:"sourcingMarkupPrice"`
				EnergyTaxPrice     *float64 `json:"energyTaxPrice"`
			} `json:"marketPricesElectricity"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("frank parse: %w", err)
	}

	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("frank API error: %s", resp.Errors[0].Message)
	}

	prices := make([]models.HourlyPrice, 0, len(resp.Data.MarketPricesElectricity))
	for _, r := range resp.Data.MarketPricesElectricity {
		ts, err := time.Parse(time.RFC3339, r.From)
		if err != nil {
			continue
		}

		// Sum all components; handle nil values
		total := ptrOr(r.MarketPrice) + ptrOr(r.MarketPriceTax) +
			ptrOr(r.SourcingMarkupPrice) + ptrOr(r.EnergyTaxPrice)

		prices = append(prices, models.HourlyPrice{
			Timestamp:  ts.UTC(),
			PriceEUR:   total,
			Unit:       models.UnitKWh,
			Source:     "frank",
			Quality:    "live",
			Zone:       zone,
			IsConsumer: true, // Frank returns sum of all components incl. VAT + energy tax
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("frank: no prices returned for %s", date.Format("2006-01-02"))
	}
	return prices, nil
}

func ptrOr(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
