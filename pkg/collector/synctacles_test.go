package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSynctaclesAPI_Metadata(t *testing.T) {
	s := &SynctaclesAPI{BaseURL: "https://energy.synctacles.com"}
	assert.Equal(t, "synctacles", s.Name())
	assert.False(t, s.RequiresKey())
	assert.Len(t, s.Zones(), 38)
	assert.Contains(t, s.Zones(), "NL")
	assert.Contains(t, s.Zones(), "DE-LU")
	assert.Contains(t, s.Zones(), "NO1")
}
