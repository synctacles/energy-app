package delta

import (
	"testing"
	"time"
)

func TestParseForecastArray_NordPoolPattern(t *testing.T) {
	arr := []any{
		map[string]any{"start": "2026-03-15T14:00:00+01:00", "value": 0.2345},
		map[string]any{"start": "2026-03-15T15:00:00+01:00", "value": 0.1890},
		map[string]any{"start": "2026-03-15T16:00:00+01:00", "value": 0.3012},
	}

	prices := parseForecastArray(arr)
	if len(prices) != 3 {
		t.Fatalf("expected 3 prices, got %d", len(prices))
	}

	if prices[0].PriceKWh != 0.2345 {
		t.Errorf("expected 0.2345, got %f", prices[0].PriceKWh)
	}
	if prices[0].Timestamp.Hour() != 13 { // 14:00+01:00 = 13:00 UTC
		t.Errorf("expected hour 13 UTC, got %d", prices[0].Timestamp.Hour())
	}
}

func TestParseForecastArray_PriceKey(t *testing.T) {
	arr := []any{
		map[string]any{"datetime": "2026-03-15T12:00:00Z", "price": 0.15},
	}

	prices := parseForecastArray(arr)
	if len(prices) != 1 {
		t.Fatalf("expected 1 price, got %d", len(prices))
	}
	if prices[0].PriceKWh != 0.15 {
		t.Errorf("expected 0.15, got %f", prices[0].PriceKWh)
	}
}

func TestParseForecastArray_PlainFloats(t *testing.T) {
	arr := []any{0.21, 0.19, 0.18}

	prices := parseForecastArray(arr)
	if len(prices) != 3 {
		t.Fatalf("expected 3 prices, got %d", len(prices))
	}

	now := time.Now().UTC()
	if prices[0].Timestamp.Day() != now.Day() {
		t.Errorf("expected today, got %s", prices[0].Timestamp)
	}
	if prices[0].PriceKWh != 0.21 {
		t.Errorf("expected 0.21, got %f", prices[0].PriceKWh)
	}
}

func TestParseForecastArray_MissingTimestamp(t *testing.T) {
	arr := []any{
		map[string]any{"price": 0.15}, // no timestamp key
	}

	prices := parseForecastArray(arr)
	if len(prices) != 0 {
		t.Errorf("expected 0 prices for entry without timestamp, got %d", len(prices))
	}
}

func TestParseMapEntry_AllKeyVariants(t *testing.T) {
	tests := []struct {
		name   string
		m      map[string]any
		wantOK bool
	}{
		{"start+value", map[string]any{"start": "2026-03-15T10:00:00Z", "value": 0.1}, true},
		{"datetime+price", map[string]any{"datetime": "2026-03-15T10:00:00Z", "price": 0.2}, true},
		{"start_time+tariff", map[string]any{"start_time": "2026-03-15T10:00:00Z", "tariff": 0.3}, true},
		{"no_ts", map[string]any{"value": 0.1}, false},
		{"no_price", map[string]any{"start": "2026-03-15T10:00:00Z"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok := parseMapEntry(tt.m)
			if ok != tt.wantOK {
				t.Errorf("expected ok=%v, got %v", tt.wantOK, ok)
			}
		})
	}
}
