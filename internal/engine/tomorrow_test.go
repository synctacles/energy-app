package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/synctacles/energy-go/internal/models"
)

func TestDetermineTomorrowPreview_Favorable_Absolute(t *testing.T) {
	// Tomorrow average < €0.20/kWh → FAVORABLE
	today := makeHourlyPrices(time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.25))
	tomorrow := makeHourlyPrices(time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.15))

	result := DetermineTomorrowPreview(today, tomorrow)
	assert.Equal(t, models.PreviewFavorable, result.Status)
	assert.InDelta(t, 0.15, result.AvgPrice, 0.001)
}

func TestDetermineTomorrowPreview_Favorable_Relative(t *testing.T) {
	// Tomorrow avg < today avg * 0.90 → FAVORABLE
	today := makeHourlyPrices(time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.28))
	tomorrow := makeHourlyPrices(time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.24))

	result := DetermineTomorrowPreview(today, tomorrow)
	assert.Equal(t, models.PreviewFavorable, result.Status)
}

func TestDetermineTomorrowPreview_Expensive_Absolute(t *testing.T) {
	// Tomorrow average > €0.30/kWh → EXPENSIVE
	today := makeHourlyPrices(time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.25))
	tomorrow := makeHourlyPrices(time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.35))

	result := DetermineTomorrowPreview(today, tomorrow)
	assert.Equal(t, models.PreviewExpensive, result.Status)
}

func TestDetermineTomorrowPreview_Expensive_Relative(t *testing.T) {
	// Tomorrow avg > today avg * 1.10 → EXPENSIVE (but below €0.30 absolute)
	today := makeHourlyPrices(time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.22))
	tomorrow := makeHourlyPrices(time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.26))

	result := DetermineTomorrowPreview(today, tomorrow)
	assert.Equal(t, models.PreviewExpensive, result.Status)
}

func TestDetermineTomorrowPreview_Normal(t *testing.T) {
	// Tomorrow similar to today → NORMAL
	today := makeHourlyPrices(time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.25))
	tomorrow := makeHourlyPrices(time.Date(2026, 2, 12, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.25))

	result := DetermineTomorrowPreview(today, tomorrow)
	assert.Equal(t, models.PreviewNormal, result.Status)
}

func TestDetermineTomorrowPreview_Pending(t *testing.T) {
	// No tomorrow prices → PENDING
	today := makeHourlyPrices(time.Date(2026, 2, 11, 0, 0, 0, 0, time.UTC), uniformPrices(24, 0.25))

	result := DetermineTomorrowPreview(today, nil)
	assert.Equal(t, models.PreviewPending, result.Status)
}

func uniformPrices(n int, price float64) []float64 {
	p := make([]float64, n)
	for i := range p {
		p[i] = price
	}
	return p
}
