package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/synctacles/energy-app/pkg/models"
)

// TOUConfig holds a flexible time-of-use schedule.
// Users define rate levels and periods (day-of-week + time ranges).
// Hours not covered by any period fall back to the Default rate.
type TOUConfig struct {
	Rates   []TOURate   `json:"rates"`   // 2-3 rate levels (e.g. peak, offpeak, midpeak)
	Periods []TOUPeriod `json:"periods"` // when each rate applies
	Default string      `json:"default"` // rate ID for uncovered hours (usually "offpeak")
}

// TOURate defines a named price level.
type TOURate struct {
	ID    string  `json:"id"`    // "peak", "offpeak", "midpeak"
	Name  string  `json:"name"`  // display name
	Price float64 `json:"price"` // EUR/kWh (consumer price incl. taxes)
}

// TOUPeriod defines when a rate applies.
// Days uses ISO weekday numbering: 0=Sunday, 1=Monday, ..., 6=Saturday.
// Overnight ranges (Start > End, e.g. 22:00-06:00) are supported.
type TOUPeriod struct {
	Days   []int  `json:"days"`    // 0=Sun, 1=Mon, ..., 6=Sat
	Start  string `json:"start"`   // "08:00"
	End    string `json:"end"`     // "22:00"
	RateID string `json:"rate_id"` // references TOURate.ID
}

// ParseTOUConfig parses a JSON string into a TOUConfig.
func ParseTOUConfig(jsonStr string) (*TOUConfig, error) {
	if jsonStr == "" {
		return nil, fmt.Errorf("empty TOU config")
	}
	var cfg TOUConfig
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		return nil, fmt.Errorf("parse TOU config: %w", err)
	}
	return &cfg, cfg.Validate()
}

// Validate checks the TOUConfig for consistency.
func (c *TOUConfig) Validate() error {
	if len(c.Rates) < 2 {
		return fmt.Errorf("TOU config needs at least 2 rates")
	}
	rateIDs := make(map[string]bool)
	for _, r := range c.Rates {
		if r.ID == "" {
			return fmt.Errorf("TOU rate missing id")
		}
		if r.Price < 0 || r.Price > 5.0 {
			return fmt.Errorf("TOU rate %q price out of range (0-5.0)", r.ID)
		}
		rateIDs[r.ID] = true
	}
	if c.Default == "" {
		return fmt.Errorf("TOU config missing default rate")
	}
	if !rateIDs[c.Default] {
		return fmt.Errorf("TOU default rate %q not found in rates", c.Default)
	}
	for i, p := range c.Periods {
		if !rateIDs[p.RateID] {
			return fmt.Errorf("period %d references unknown rate %q", i, p.RateID)
		}
		if _, err := parseHHMM(p.Start); err != nil {
			return fmt.Errorf("period %d invalid start %q: %w", i, p.Start, err)
		}
		if _, err := parseHHMM(p.End); err != nil {
			return fmt.Errorf("period %d invalid end %q: %w", i, p.End, err)
		}
		for _, d := range p.Days {
			if d < 0 || d > 6 {
				return fmt.Errorf("period %d invalid day %d (must be 0-6)", i, d)
			}
		}
	}
	return nil
}

// GenerateTOUPrices generates synthetic hourly prices for today and tomorrow
// based on the TOU schedule. Each hour gets the rate matching its day-of-week
// and time, or the default rate if no period matches.
func GenerateTOUPrices(cfg *TOUConfig, loc *time.Location) (today, tomorrow []models.HourlyPrice) {
	now := time.Now().In(loc)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	tomorrowStart := todayStart.Add(24 * time.Hour)

	today = generateDay(cfg, todayStart, loc)
	tomorrow = generateDay(cfg, tomorrowStart, loc)
	return today, tomorrow
}

func generateDay(cfg *TOUConfig, dayStart time.Time, loc *time.Location) []models.HourlyPrice {
	prices := make([]models.HourlyPrice, 24)
	defaultPrice := ratePrice(cfg, cfg.Default)

	for h := 0; h < 24; h++ {
		ts := dayStart.Add(time.Duration(h) * time.Hour)
		price := defaultPrice

		// Check periods — last matching period wins
		dow := goWeekdayToISO(ts.Weekday())
		for _, p := range cfg.Periods {
			if matchesPeriod(p, dow, h) {
				price = ratePrice(cfg, p.RateID)
			}
		}

		prices[h] = models.HourlyPrice{
			Timestamp:  ts.UTC(),
			PriceEUR:   price,
			Unit:       models.UnitKWh,
			Source:     "tou",
			Quality:    "live",
			Zone:       "",
			IsConsumer: true,
		}
	}
	return prices
}

// matchesPeriod checks if a given day-of-week and hour falls within a period.
func matchesPeriod(p TOUPeriod, dow, hour int) bool {
	// Check day-of-week
	dayMatch := false
	for _, d := range p.Days {
		if d == dow {
			dayMatch = true
			break
		}
	}
	if !dayMatch {
		return false
	}

	startH, _ := parseHHMM(p.Start)
	endH, _ := parseHHMM(p.End)

	if startH <= endH {
		// Normal range: e.g. 08:00-22:00
		return hour >= startH && hour < endH
	}
	// Overnight range: e.g. 22:00-06:00
	return hour >= startH || hour < endH
}

// ratePrice looks up a rate's price by ID.
func ratePrice(cfg *TOUConfig, rateID string) float64 {
	for _, r := range cfg.Rates {
		if r.ID == rateID {
			return r.Price
		}
	}
	return 0
}

// parseHHMM parses "HH:MM" and returns the hour component.
func parseHHMM(s string) (int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("expected HH:MM format")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, fmt.Errorf("invalid hour: %s", parts[0])
	}
	return h, nil
}

// goWeekdayToISO converts Go's time.Weekday (Sunday=0) to our ISO convention (Sunday=0).
// They happen to match, but this makes intent explicit.
func goWeekdayToISO(wd time.Weekday) int {
	return int(wd)
}
