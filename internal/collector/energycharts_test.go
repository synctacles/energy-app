package collector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synctacles/energy-go/internal/models"
)

func TestEnergyCharts_Metadata(t *testing.T) {
	e := &EnergyCharts{}
	assert.Equal(t, "energycharts", e.Name())
	assert.False(t, e.RequiresKey())
	assert.Contains(t, e.Zones(), "NL")
	assert.Contains(t, e.Zones(), "DE-LU")
	assert.Contains(t, e.Zones(), "FR")
}

func TestECSupportedZones(t *testing.T) {
	supported := []string{"NL", "DE-LU", "NO1", "SE3", "FI", "ES", "PT", "IT-North", "SI"}
	for _, zone := range supported {
		assert.True(t, ecSupportedZones[zone], "zone %s should be supported", zone)
	}
	assert.False(t, ecSupportedZones["XX"], "unknown zone should not be supported")
}

func TestEnergyCharts_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	e := &EnergyCharts{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test NL prices
	prices, err := e.FetchDayAhead(ctx, "NL", time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		t.Skipf("Energy-Charts API unavailable: %v", err)
	}

	assert.NotEmpty(t, prices)
	// Should be exactly 24 hourly prices (sub-hourly filtered out)
	assert.LessOrEqual(t, len(prices), 24)
	for _, p := range prices {
		assert.Equal(t, models.UnitMWh, p.Unit)
		assert.Equal(t, "energycharts", p.Source)
		assert.Equal(t, "NL", p.Zone)
		assert.False(t, p.IsConsumer, "wholesale source should not be consumer")
		// Only whole hours (no 15-min intervals)
		assert.Equal(t, 0, p.Timestamp.Minute(), "expected whole hours only, got %s", p.Timestamp.Format(time.RFC3339))
		// Wholesale prices in EUR/MWh, typically -50 to 500
		assert.Greater(t, p.PriceEUR, -500.0)
		assert.Less(t, p.PriceEUR, 1000.0)
	}
}
