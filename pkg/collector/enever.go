package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	"groenestroom":   "prijsGSL",
	"hegg":           "prijsHE",
	"innova":         "prijsIN",
	"mijndomein":     "prijsMDE",
	"nextenergy":     "prijsNE",
	"pureenergie":    "prijsPE",
	"quatt":          "prijsQU",
	"samsam":         "prijsSS",
	"tibber":         "prijsTI",
	"vandebron":      "prijsVDB",
	"vattenfall":     "prijsVF",
	"vrijopnaam":     "prijsVON",
	"wout":           "prijsWE",
	"zonneplan":      "prijsZP",
}

// Enever fetches NL consumer prices from enever.nl (API v3).
// Requires a user-provided API token and leverancier selection.
// Returns PT15 (96 entries per day) when available.
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

	// API v3: resolution=15 for PT15, price filter to reduce payload
	url := fmt.Sprintf("https://enever.nl/apiv3/%s.php?token=%s&resolution=15&price=prijs,%s",
		endpoint, e.Token, priceField)

	body, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("enever fetch: %w", err)
	}

	// API v3 always returns {"status": bool, "data": [...], "code": int}
	var resp struct {
		Status bool              `json:"status"`
		Data   []json.RawMessage `json:"data"`
		Code   int               `json:"code"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("enever parse: %w", err)
	}

	var prices []models.HourlyPrice
	for _, raw := range resp.Data {
		var item map[string]any
		if json.Unmarshal(raw, &item) != nil {
			continue
		}

		datum, _ := item["datum"].(string)
		if datum == "" {
			continue
		}

		// Extract price — Enever returns prices as strings (e.g. "0.238174")
		priceVal, ok := item[priceField]
		if !ok {
			continue
		}
		var price float64
		switch v := priceVal.(type) {
		case float64:
			price = v
		case string:
			if p, err := strconv.ParseFloat(v, 64); err == nil {
				price = p
			} else {
				continue
			}
		default:
			continue
		}

		// API v3 timestamps are ISO 8601 with timezone offset: "2026-03-03T00:00:00+01:00"
		ts, err := time.Parse(time.RFC3339, datum)
		if err != nil {
			// Fallback: v2 format "2026-03-03 00:00:00" (CET/CEST)
			nlLoc, _ := time.LoadLocation("Europe/Amsterdam")
			if t, err2 := time.ParseInLocation("2006-01-02 15:04:05", datum, nlLoc); err2 == nil {
				ts = t
			} else {
				continue
			}
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
