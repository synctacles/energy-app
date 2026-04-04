package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_BestWindowHours_Default(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 3, cfg.BestWindowHours)
}

func TestLoad_BestWindowHours_Clamp(t *testing.T) {
	// Too low
	os.Setenv("BEST_WINDOW_HOURS", "0")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.BestWindowHours)
	os.Unsetenv("BEST_WINDOW_HOURS")

	// Too high
	os.Setenv("BEST_WINDOW_HOURS", "12")
	cfg, err = Load()
	require.NoError(t, err)
	assert.Equal(t, 8, cfg.BestWindowHours)
	os.Unsetenv("BEST_WINDOW_HOURS")

	// Valid
	os.Setenv("BEST_WINDOW_HOURS", "5")
	cfg, err = Load()
	require.NoError(t, err)
	assert.Equal(t, 5, cfg.BestWindowHours)
	os.Unsetenv("BEST_WINDOW_HOURS")
}
