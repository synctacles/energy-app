package hasensor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synctacles/energy-app/pkg/engine"
	"github.com/synctacles/energy-app/pkg/models"
)

// mockPublisher records all sensor updates.
type mockPublisher struct {
	updates []sensorUpdate
}

type sensorUpdate struct {
	entityID string
	state    string
	attrs    map[string]any
}

func (m *mockPublisher) UpdateSensor(_ context.Context, entityID, state string, attrs map[string]any) error {
	m.updates = append(m.updates, sensorUpdate{entityID, state, attrs})
	return nil
}

func findUpdate(updates []sensorUpdate, entityID string) (sensorUpdate, bool) {
	for _, u := range updates {
		if u.entityID == entityID {
			return u, true
		}
	}
	return sensorUpdate{}, false
}

func TestPublishAll_AllSensors(t *testing.T) {
	pub := &mockPublisher{}
	s := &SensorSet{
		Zone:         "NL",
		CurrentPrice: 0.2134,
		Action:       models.ActionResult{Action: models.ActionGo},
		Stats:        models.PriceStats{Average: 0.20, Min: 0.08, Max: 0.35, CheapestHour: "03:00", ExpensiveHour: "18:00"},
		BestWindow:   &models.BestWindow{StartHour: "02:00", EndHour: "05:00", AvgPrice: 0.09, Duration: 3},
		Tomorrow:     models.TomorrowResult{Status: models.PreviewFavorable},
		TodayPrices:  make([]models.HourlyPrice, 24),
		UpdatedAt:    time.Now(),
	}

	err := PublishAll(context.Background(), pub, s)
	require.NoError(t, err)

	// All 9 sensors published (no license gate)
	assert.Len(t, pub.updates, 9)
	entityIDs := make([]string, len(pub.updates))
	for i, u := range pub.updates {
		entityIDs[i] = u.entityID
	}
	assert.Contains(t, entityIDs, "sensor.synctacles_energy_price")
	assert.Contains(t, entityIDs, "sensor.synctacles_cheapest_hour")
	assert.Contains(t, entityIDs, "sensor.synctacles_expensive_hour")
	assert.Contains(t, entityIDs, "sensor.synctacles_prices_today")
	assert.Contains(t, entityIDs, "binary_sensor.synctacles_cheap_hour")
	assert.Contains(t, entityIDs, "sensor.synctacles_energy_action")
	assert.Contains(t, entityIDs, "sensor.synctacles_best_window")
	assert.Contains(t, entityIDs, "sensor.synctacles_tomorrow_preview")
	assert.Contains(t, entityIDs, "sensor.synctacles_prices_tomorrow")
}

func TestPublishAll_CheapHour_OnDuringGO(t *testing.T) {
	pub := &mockPublisher{}
	s := &SensorSet{
		Zone:         "NL",
		CurrentPrice: 0.10,
		Action:       models.ActionResult{Action: models.ActionGo},
		Stats:        models.PriceStats{Average: 0.20, Min: 0.08, Max: 0.35, CheapestHour: "03:00", ExpensiveHour: "18:00"},
		TodayPrices:  make([]models.HourlyPrice, 24),
		UpdatedAt:    time.Now(),
	}

	err := PublishAll(context.Background(), pub, s)
	require.NoError(t, err)

	u, found := findUpdate(pub.updates, "binary_sensor.synctacles_cheap_hour")
	require.True(t, found)
	assert.Equal(t, "on", u.state)
}

func TestPublishAll_CheapHour_OffDuringWait(t *testing.T) {
	pub := &mockPublisher{}
	s := &SensorSet{
		Zone:         "NL",
		CurrentPrice: 0.20,
		Action:       models.ActionResult{Action: models.ActionWait},
		Stats:        models.PriceStats{Average: 0.20, Min: 0.08, Max: 0.35, CheapestHour: "03:00", ExpensiveHour: "18:00"},
		TodayPrices:  make([]models.HourlyPrice, 24),
		UpdatedAt:    time.Now(),
	}

	err := PublishAll(context.Background(), pub, s)
	require.NoError(t, err)

	u, found := findUpdate(pub.updates, "binary_sensor.synctacles_cheap_hour")
	require.True(t, found)
	assert.Equal(t, "off", u.state)
}

func TestPublishAll_CheapHour_OffDuringAvoid(t *testing.T) {
	pub := &mockPublisher{}
	s := &SensorSet{
		Zone:         "NL",
		CurrentPrice: 0.35,
		Action:       models.ActionResult{Action: models.ActionAvoid},
		Stats:        models.PriceStats{Average: 0.20, Min: 0.08, Max: 0.35, CheapestHour: "03:00", ExpensiveHour: "18:00"},
		TodayPrices:  make([]models.HourlyPrice, 24),
		UpdatedAt:    time.Now(),
	}

	err := PublishAll(context.Background(), pub, s)
	require.NoError(t, err)

	u, found := findUpdate(pub.updates, "binary_sensor.synctacles_cheap_hour")
	require.True(t, found)
	assert.Equal(t, "off", u.state)
}

func TestComputeSensorSet(t *testing.T) {
	now := time.Date(2026, 2, 11, 14, 0, 0, 0, time.UTC)
	prices := make([]models.HourlyPrice, 24)
	for i := range prices {
		prices[i] = models.HourlyPrice{
			Timestamp: time.Date(2026, 2, 11, i, 0, 0, 0, time.UTC),
			PriceEUR:  0.20,
			Unit:      models.UnitKWh,
			Zone:      "NL",
		}
	}
	prices[14].PriceEUR = 0.10 // Current hour is cheap

	ae := engine.NewActionEngine(-15, 20)
	fr := &engine.FetchResult{Source: "easyenergy", Tier: 1, Quality: "live"}

	ss := ComputeSensorSet("NL", prices, nil, ae, fr, now, "", 3)

	assert.Equal(t, "NL", ss.Zone)
	assert.InDelta(t, 0.10, ss.CurrentPrice, 0.001)
	assert.Equal(t, "easyenergy", ss.Source)
	assert.Equal(t, models.PreviewPending, ss.Tomorrow.Status)
}

func TestComputeSensorSet_BestWindowHours(t *testing.T) {
	now := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	prices := make([]models.HourlyPrice, 24)
	for i := range prices {
		prices[i] = models.HourlyPrice{
			Timestamp: time.Date(2026, 2, 11, i, 0, 0, 0, time.UTC),
			PriceEUR:  0.25,
			Unit:      models.UnitKWh,
			Zone:      "NL",
		}
	}
	// Make hours 12-17 cheap
	for i := 12; i <= 17; i++ {
		prices[i].PriceEUR = 0.10
	}

	ae := engine.NewActionEngine(-15, 20)
	fr := &engine.FetchResult{Source: "easyenergy", Tier: 1, Quality: "live"}

	// 3-hour window
	ss3 := ComputeSensorSet("NL", prices, nil, ae, fr, now, "", 3)
	require.NotNil(t, ss3.BestWindow)
	assert.Equal(t, 3, ss3.BestWindow.Duration)

	// 5-hour window
	ss5 := ComputeSensorSet("NL", prices, nil, ae, fr, now, "", 5)
	require.NotNil(t, ss5.BestWindow)
	assert.Equal(t, 5, ss5.BestWindow.Duration)

	// Default (0 falls back to 3)
	ss0 := ComputeSensorSet("NL", prices, nil, ae, fr, now, "", 0)
	require.NotNil(t, ss0.BestWindow)
	assert.Equal(t, 3, ss0.BestWindow.Duration)
}
