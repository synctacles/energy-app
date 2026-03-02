package hasensor

import (
	"testing"
	"time"
)

func makeState(entityID, state, unit string, lastChanged time.Time) map[string]any {
	return map[string]any{
		"entity_id":    entityID,
		"state":        state,
		"last_changed": lastChanged.Format(time.RFC3339),
		"attributes": map[string]any{
			"unit_of_measurement": unit,
		},
	}
}

func TestMatchTariffSensor_PrefersRecentlyChanged(t *testing.T) {
	now := time.Now()

	// Two sensors with same name pattern priority, but one updated recently, one stale
	states := []map[string]any{
		makeState("sensor.fixed_tariff", "0.25", "EUR/kWh", now.Add(-48*time.Hour)),        // stale — fixed
		makeState("sensor.dynamic_energy_price", "0.18", "EUR/kWh", now.Add(-1*time.Hour)), // recent — dynamic
	}

	// Both match "tariff"/"energy_price" pattern (prio 20)
	// Fixed gets +15 penalty (>24h) = 35, dynamic gets -5 bonus (<4h) = 15
	result := matchTariffSensor(states)

	if result != "sensor.dynamic_energy_price" {
		t.Errorf("expected dynamic sensor, got %q", result)
	}
}

func TestMatchTariffSensor_FixedRateDeprioritized(t *testing.T) {
	now := time.Now()

	states := []map[string]any{
		makeState("sensor.my_tariff", "0.30", "EUR/kWh", now.Add(-72*time.Hour)),                 // stale tariff (prio 20+15=35)
		makeState("sensor.nord_pool_current_price", "0.05", "EUR/kWh", now.Add(-30*time.Minute)), // dynamic spot (prio 25-5=20)
	}

	result := matchTariffSensor(states)

	if result != "sensor.nord_pool_current_price" {
		t.Errorf("expected nord pool sensor to win over stale tariff, got %q", result)
	}
}

func TestMatchTariffSensor_NoLastChanged(t *testing.T) {
	// Sensors without last_changed should still work (no adjustment)
	states := []map[string]any{
		{
			"entity_id": "sensor.unit_rate_electricity",
			"state":     "0.28",
			"attributes": map[string]any{
				"unit_of_measurement": "GBP/kWh",
			},
		},
	}

	result := matchTariffSensor(states)

	if result != "sensor.unit_rate_electricity" {
		t.Errorf("expected sensor without last_changed to still match, got %q", result)
	}
}

func TestMatchTariffSensor_Empty(t *testing.T) {
	result := matchTariffSensor(nil)
	if result != "" {
		t.Errorf("expected empty string for nil states, got %q", result)
	}
}

func TestMatchTariffSensor_HighPrioBeatsLastChanged(t *testing.T) {
	now := time.Now()

	// UK Glow (prio 10) should still beat a recently-changed generic tariff (prio 20-5=15)
	states := []map[string]any{
		makeState("sensor.glow_unit_rate", "0.28", "GBP/kWh", now.Add(-12*time.Hour)),     // prio 10, no adjustment (4-24h)
		makeState("sensor.some_energy_price", "0.05", "EUR/kWh", now.Add(-30*time.Minute)), // prio 20-5=15
	}

	result := matchTariffSensor(states)

	if result != "sensor.glow_unit_rate" {
		t.Errorf("expected high-priority sensor to still win, got %q", result)
	}
}
