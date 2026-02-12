package hasensor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synctacles/energy-go/internal/engine"
	"github.com/synctacles/energy-go/internal/models"
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

func TestPublishAll_FreeOnly(t *testing.T) {
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

	err := PublishAll(context.Background(), pub, s, false)
	require.NoError(t, err)

	// Free tier: only 4 sensors (price, cheapest, expensive, prices_today)
	assert.Len(t, pub.updates, 4)
	assert.Equal(t, "sensor.synctacles_energy_price", pub.updates[0].entityID)
	assert.Equal(t, "sensor.synctacles_cheapest_hour", pub.updates[1].entityID)
	assert.Equal(t, "sensor.synctacles_expensive_hour", pub.updates[2].entityID)
	assert.Equal(t, "sensor.synctacles_prices_today", pub.updates[3].entityID)
}

func TestPublishAll_Pro(t *testing.T) {
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

	err := PublishAll(context.Background(), pub, s, true)
	require.NoError(t, err)

	// Pro tier: 4 free + 4 pro = 8 sensors
	assert.Len(t, pub.updates, 8)
	// Check pro sensors are included
	entityIDs := make([]string, len(pub.updates))
	for i, u := range pub.updates {
		entityIDs[i] = u.entityID
	}
	assert.Contains(t, entityIDs, "sensor.synctacles_energy_action")
	assert.Contains(t, entityIDs, "sensor.synctacles_best_window")
	assert.Contains(t, entityIDs, "sensor.synctacles_tomorrow_preview")
	assert.Contains(t, entityIDs, "sensor.synctacles_prices_tomorrow")
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

	ss := ComputeSensorSet("NL", prices, nil, ae, fr, now)

	assert.Equal(t, "NL", ss.Zone)
	assert.InDelta(t, 0.10, ss.CurrentPrice, 0.001)
	assert.Equal(t, "easyenergy", ss.Source)
	assert.Equal(t, models.PreviewPending, ss.Tomorrow.Status)
}
