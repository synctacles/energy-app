package collector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synctacles/energy-app/pkg/models"
)

func TestFrankEnergie_Metadata(t *testing.T) {
	f := &FrankEnergie{}
	assert.Equal(t, "frank", f.Name())
	assert.Equal(t, []string{"NL"}, f.Zones())
	assert.False(t, f.RequiresKey())
}

func TestPtrOr(t *testing.T) {
	v := 1.23
	assert.Equal(t, 1.23, ptrOr(&v))
	assert.Equal(t, 0.0, ptrOr(nil))
}

func TestFrankEnergie_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	f := &FrankEnergie{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prices, err := f.FetchDayAhead(ctx, "NL", time.Now().UTC())
	if err != nil {
		t.Skipf("Frank Energie API unavailable: %v", err)
	}

	assert.NotEmpty(t, prices)
	for _, p := range prices {
		assert.Equal(t, models.UnitKWh, p.Unit)
		assert.Equal(t, "frank", p.Source)
		assert.Equal(t, "NL", p.Zone)
		assert.Greater(t, p.PriceEUR, -1.0)
		assert.Less(t, p.PriceEUR, 5.0)
	}
}
