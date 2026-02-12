package collector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExchangeRates_ConvertEUR(t *testing.T) {
	e := NewExchangeRates()
	// EUR to EUR should be identity
	assert.Equal(t, 100.0, e.Convert(100.0, "EUR"))
	assert.Equal(t, 100.0, e.ConvertToEUR(100.0, "EUR"))
}

func TestExchangeRates_ConvertWithRates(t *testing.T) {
	e := NewExchangeRates()
	e.rates["NOK"] = 11.0
	e.rates["SEK"] = 10.0

	// 100 EUR = 1100 NOK
	assert.InDelta(t, 1100.0, e.Convert(100.0, "NOK"), 0.01)
	// 1100 NOK = 100 EUR
	assert.InDelta(t, 100.0, e.ConvertToEUR(1100.0, "NOK"), 0.01)

	// Unknown currency returns original amount
	assert.Equal(t, 50.0, e.Convert(50.0, "GBP"))
	assert.Equal(t, 50.0, e.ConvertToEUR(50.0, "GBP"))
}

func TestExchangeRates_NeedsRefresh(t *testing.T) {
	e := NewExchangeRates()
	assert.True(t, e.NeedsRefresh())

	e.last = time.Now()
	assert.False(t, e.NeedsRefresh())

	e.last = time.Now().Add(-25 * time.Hour)
	assert.True(t, e.NeedsRefresh())
}

func TestExchangeRates_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	e := NewExchangeRates()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := e.Fetch(ctx, []string{"NOK", "SEK", "DKK"})
	if err != nil {
		t.Skipf("ECB API unavailable: %v", err)
	}

	require.False(t, e.NeedsRefresh())

	// NOK should be roughly 10-13 per EUR
	nok := e.Convert(1.0, "NOK")
	assert.Greater(t, nok, 8.0)
	assert.Less(t, nok, 15.0)

	// DKK is pegged to EUR at ~7.46
	dkk := e.Convert(1.0, "DKK")
	assert.Greater(t, dkk, 7.0)
	assert.Less(t, dkk, 8.0)
}
