package collector

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// OMIE provides Spanish and Portuguese day-ahead electricity prices.
// Data is available as semicolon-delimited text files (no JSON API).
// As of 2025, OMIE returns quarter-hourly data (H1Q1-H24Q4, 96 values).
type OMIE struct{}

func (o *OMIE) Name() string     { return "omie" }
func (o *OMIE) RequiresKey() bool { return false }
func (o *OMIE) Zones() []string   { return []string{"ES", "PT"} }

// omieRowPrefix maps zone to the prefix of the data row in the OMIE file.
var omieRowPrefix = map[string]string{
	"ES": "Precio marginal en el sistema espa",
	"PT": "Precio marginal en el sistema portugu",
}

// FetchDayAhead fetches hourly prices from the OMIE daily text file.
// Returns prices in EUR/MWh (averaged from quarter-hourly data).
func (o *OMIE) FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error) {
	prefix, ok := omieRowPrefix[zone]
	if !ok {
		return nil, fmt.Errorf("omie: unsupported zone %s", zone)
	}

	// URL: /dados/AGNO_YYYY/MES_MM/TXT/INT_PBC_EV_H_1_DD_MM_YYYY_DD_MM_YYYY.TXT
	dd := fmt.Sprintf("%02d", date.Day())
	mm := fmt.Sprintf("%02d", date.Month())
	yyyy := fmt.Sprintf("%d", date.Year())

	url := fmt.Sprintf(
		"https://www.omie.es/sites/default/files/dados/AGNO_%s/MES_%s/TXT/INT_PBC_EV_H_1_%s_%s_%s_%s_%s_%s.TXT",
		yyyy, mm, dd, mm, yyyy, dd, mm, yyyy,
	)

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("omie fetch: %w", err)
	}

	// Parse semicolon-delimited text file.
	// Data rows start with descriptive text (e.g. "Precio marginal en el sistema español").
	// Format: 96 quarter-hourly values (H1Q1, H1Q2, H1Q3, H1Q4, H2Q1, ..., H24Q4).
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var quarterPrices []float64

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		fields := strings.Split(line, ";")
		// First field is the description, remaining are 96 quarter-hourly prices
		for i := 1; i < len(fields); i++ {
			val := strings.TrimSpace(fields[i])
			if val == "" {
				continue
			}
			// OMIE uses comma as decimal separator
			val = strings.ReplaceAll(val, ",", ".")
			price, err := strconv.ParseFloat(val, 64)
			if err != nil {
				continue
			}
			quarterPrices = append(quarterPrices, price)
		}
		break
	}

	if len(quarterPrices) == 0 {
		return nil, fmt.Errorf("omie: no prices parsed for zone %s on %s", zone, date.Format("2006-01-02"))
	}

	// Average quarter-hourly prices into hourly prices.
	// 96 quarters → 24 hours (4 quarters per hour).
	numHours := len(quarterPrices) / 4
	if numHours > 24 {
		numHours = 24
	}

	prices := make([]models.HourlyPrice, 0, numHours)
	for h := 0; h < numHours; h++ {
		start := h * 4
		end := start + 4
		if end > len(quarterPrices) {
			end = len(quarterPrices)
		}

		var sum float64
		count := 0
		for _, qp := range quarterPrices[start:end] {
			sum += qp
			count++
		}
		if count == 0 {
			continue
		}
		avgPrice := sum / float64(count)

		ts := time.Date(date.Year(), date.Month(), date.Day(), h, 0, 0, 0, time.UTC)
		prices = append(prices, models.HourlyPrice{
			Timestamp: ts,
			PriceEUR:  avgPrice,
			Unit:      models.UnitMWh,
			Source:    "omie",
			Quality:   "live",
			Zone:      zone,
		})
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("omie: no hourly prices for zone %s on %s", zone, date.Format("2006-01-02"))
	}
	return prices, nil
}
