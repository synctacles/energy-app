package collector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synctacles/energy-app/pkg/models"
)

func TestAWATTar_Metadata(t *testing.T) {
	a := &AWATTar{}
	assert.Equal(t, "awattar", a.Name())
	assert.False(t, a.RequiresKey())
	assert.Contains(t, a.Zones(), "DE-LU")
	assert.Contains(t, a.Zones(), "AT")
}

func TestAWATTar_UnsupportedZone(t *testing.T) {
	a := &AWATTar{}
	ctx := context.Background()
	_, err := a.FetchDayAhead(ctx, "NL", time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported zone")
}

func TestAWATTar_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	a := &AWATTar{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prices, err := a.FetchDayAhead(ctx, "DE-LU", time.Now().UTC())
	if err != nil {
		t.Skipf("aWATTar API unavailable: %v", err)
	}

	assert.NotEmpty(t, prices)
	for _, p := range prices {
		assert.Equal(t, models.UnitMWh, p.Unit)
		assert.Equal(t, "awattar", p.Source)
		assert.Equal(t, "DE-LU", p.Zone)
	}
}
