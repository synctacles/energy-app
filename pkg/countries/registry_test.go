package countries

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAll(t *testing.T) {
	configs, err := LoadAll()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(configs), 14, "should have at least 14 country configs")

	// Verify all have required fields (zone metadata only — tax data comes from Worker)
	for _, cc := range configs {
		assert.NotEmpty(t, cc.Country, "country code required")
		assert.NotEmpty(t, cc.Zones, "at least one zone required for %s", cc.Country)
		assert.NotEmpty(t, cc.Currency, "currency required for %s", cc.Country)

		for _, z := range cc.Zones {
			assert.NotEmpty(t, z.Code, "zone code required in %s", cc.Country)
			assert.NotEmpty(t, z.Name, "zone name required in %s", cc.Country)
		}
	}
}

func TestLoadRegistry(t *testing.T) {
	registry, err := LoadRegistry()
	require.NoError(t, err)

	zones := registry.AllZones()
	assert.GreaterOrEqual(t, len(zones), 25, "should have at least 25 zones across all countries")

	// Spot check key zones
	for _, zone := range []string{"NL", "DE-LU", "NO1", "SE3", "DK1", "FI", "BE", "FR", "AT", "ES", "PT"} {
		_, ok := registry.GetZone(zone)
		assert.True(t, ok, "zone %s should be in registry", zone)
	}

	// Verify NL zone metadata
	cc, ok := registry.GetCountryForZone("NL")
	require.True(t, ok)
	assert.Equal(t, "NL", cc.Country)
	assert.Equal(t, "Netherlands", cc.Name)
}
