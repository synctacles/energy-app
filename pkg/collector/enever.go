package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// Leveranciers maps Dutch supplier names to Enever API price codes.
var Leveranciers = map[string]string{
	"anwb":           "prijsANWB",
	"budget":         "prijsBE",
	"coolblue":       "prijsCB",
	"easyenergy":     "prijsEE",
	"energiedirect":  "prijsED",
	"energievanons":  "prijsEVO",
	"energiek":       "prijsEK",
	"energyzero":     "prijsEZ",
	"essent":         "prijsES",
	"frank":          "prijsFR",
	"groenestroom":   "prijsGL",
	"hegg":           "prijsHG",
	"innova":         "prijsIN",
	"mijndomein":     "prijsMDE",
	"nextenergy":     "prijsNE",
	"pureenergie":    "prijsPE",
	"quatt":          "prijsQT",
	"samsam":         "prijsSS",
	"tibber":         "prijsTI",
	"vandebron":      "prijsVDB",
	"vattenfall":     "prijsVF",
	"vrijopnaam":     "prijsVON",
	"wout":           "prijsWE",
	"zonneplan":      "prijsZP",
}

// Enever fetches NL consumer prices from enever.nl.
// Requires a user-provided API token and leverancier selection.
type Enever struct {
	Token       string
	Leverancier string // Key from Leveranciers map (e.g. "frank")
}

func (e *Enever) Name() string      { return "enever" }
func (e *Enever) Zones() []string   { return []string{"NL"} }
func (e *Enever) RequiresKey() bool { return true }

func (e *Enever) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	if e.Token == "" {
		return nil, fmt.Errorf("enever: no token configured")
	}

	priceField := Leveranciers[e.Leverancier]
	if priceField == "" {
		return nil, fmt.Errorf("enever: unknown leverancier %q", e.Leverancier)
	}

	// Determine endpoint (today or tomorrow)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var endpoint string
	if date.Before(today.Add(24 * time.Hour)) {
		endpoint = "stroomprijs_vandaag"
	} else {
		endpoint = "stroomprijs_morgen"
	}

	url := fmt.Sprintf("https://enever.nl/api/%s.php?token=%s", endpoint, e.Token)

	body, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("enever fetch: %w", err)
	}

	// Parse response — Enever returns array of objects with dynamic price field names
	var rawItems []map[string]any
	if err := json.Unmarshal(body, &rawItems); err != nil {
		// Try wrapped format
		var wrapped struct {
			Data []map[string]any `json:"data"`
		}
		if err2 := json.Unmarshal(body, &wrapped); err2 != nil {
			return nil, fmt.Errorf("enever parse: %w (also tried wrapped: %w)", err, err2)
		}
		rawItems = wrapped.Data
	}

	// Enever timestamps are in CET/CEST (Netherlands local time)
	nlLoc, _ := time.LoadLocation("Europe/Amsterdam")

	var prices []models.HourlyPrice
	for _, item := range rawItems {
		datum, _ := item["datum"].(string)
		if datum == "" {
			continue
		}

		// Extract price from the leverancier-specific field
		priceVal, ok := item[priceField]
		if !ok {
			continue
		}
		price, ok := priceVal.(float64)
		if !ok {
			continue
		}

		// Parse timestamp from the response.
		// Enever may return datetime in "datum" (e.g. "2026-03-03 00:00:00")
		// or split across "datum" + "uur"/"van" fields.
		var ts time.Time
		parsed := false
		for _, layout := range []string{
			"2006-01-02 15:04:05", // full datetime in datum
			"2006-01-02 15:04",    // datetime without seconds
		} {
			if t, err := time.ParseInLocation(layout, datum, nlLoc); err == nil {
				ts = t
				parsed = true
				break
			}
		}
		if !parsed {
			// Fallback: separate uur/van field
			uur, _ := item["uur"].(string)
			if uur == "" {
				uur, _ = item["van"].(string)
			}
			if t, err := time.ParseInLocation("2006-01-02 15:04", datum+" "+uur, nlLoc); err == nil {
				ts = t
				parsed = true
			}
		}
		if !parsed {
			continue
		}

		prices = append(prices, models.HourlyPrice{
			Timestamp:  ts.UTC(),
			PriceEUR:   price,
			Unit:       models.UnitKWh, // Enever returns consumer EUR/kWh
			Source:     "enever",
			Quality:    "live",
			Zone:       "NL",
			IsConsumer: true, // Enever returns leverancier-specific consumer prices incl. VAT
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("enever: no prices in response")
	}

	return prices, nil
}
