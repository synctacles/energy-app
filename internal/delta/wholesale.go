package delta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// FetchWholesalePrices fetches day-ahead wholesale prices from the energy-data Worker.
// Returns prices for today and tomorrow (if available) in EUR/kWh.
func FetchWholesalePrices(ctx context.Context, zone string) ([]WholesalePrice, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	today := time.Now().UTC().Format("2006-01-02")
	tomorrow := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02")

	url := fmt.Sprintf("%s/api/v1/energy/prices?zone=%s&from=%s&to=%s",
		energyDataBaseURL, zone, today, tomorrow)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "SynctaclesEnergy/delta")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("prices endpoint returned %d", resp.StatusCode)
	}

	var result struct {
		Prices []struct {
			TS    int64   `json:"ts"`
			Price float64 `json:"price"` // EUR/MWh
		} `json:"prices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	prices := make([]WholesalePrice, 0, len(result.Prices))
	for _, p := range result.Prices {
		// Snap to hour boundary and convert MWh → kWh
		t := time.Unix(p.TS, 0).UTC().Truncate(time.Hour)
		prices = append(prices, WholesalePrice{
			Timestamp: t,
			PriceKWh:  p.Price / 1000.0,
		})
	}
	return prices, nil
}
