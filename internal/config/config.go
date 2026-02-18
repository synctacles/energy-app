// Package config loads energy addon settings from environment variables.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds settings for the Energy HA addon.
type Config struct {
	// HA
	SupervisorToken string `env:"SUPERVISOR_TOKEN"`
	IngressPort     int    `env:"INGRESS_PORT" envDefault:"8098"`

	// Zone
	BiddingZone string `env:"ENERGY_ZONE" envDefault:"NL"`
	Currency    string `env:"ENERGY_CURRENCY" envDefault:"EUR"`

	// Thresholds (user configurable via addon options)
	GoThreshold    float64 `env:"ENERGY_GO_THRESHOLD" envDefault:"-15"`
	AvoidThreshold float64 `env:"ENERGY_AVOID_THRESHOLD" envDefault:"20"`

	// License
	LicenseKey string `env:"SYNCTACLES_LICENSE_KEY"`

	// Optional: Enever BYO keys (NL only, consumer prices)
	EneverEnabled     bool   `env:"ENEVER_ENABLED" envDefault:"false"`
	EneverToken       string `env:"ENEVER_TOKEN"`
	EneverLeverancier string `env:"ENEVER_LEVERANCIER" envDefault:"frank"`

	// Optional: price coefficient override (0 = use country default, e.g. 1.05 = 5% markup)
	Coefficient float64 `env:"ENERGY_COEFFICIENT" envDefault:"0"`

	// Best window duration in hours (1-8, default 3)
	BestWindowHours int `env:"BEST_WINDOW_HOURS" envDefault:"3"`

	// Optional: P1 power sensor for Live Cost calculation
	PowerSensorEntity string `env:"POWER_SENSOR_ENTITY"`

	// Synctacles central price server (primary source, Tier 0).
	// When set, the addon fetches pre-computed consumer prices from this server.
	// All other collectors (Energy-Charts, etc.) become emergency fallback only.
	SynctaclesURL string `env:"SYNCTACLES_URL" envDefault:"https://energy.synctacles.com"`

	// Heartbeat (anonymous install counting — enables registered features)
	HeartbeatEnabled bool `env:"HEARTBEAT_ENABLED" envDefault:"true"`

	// Price alerts — HA persistent notification when price drops below threshold
	AlertEnabled   bool    `env:"ENERGY_ALERTS_ENABLED" envDefault:"false"`
	AlertThreshold float64 `env:"ENERGY_ALERT_THRESHOLD" envDefault:"0"`

	// Debug
	DebugMode bool `env:"DEBUG_MODE" envDefault:"false"`
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse energy config: %w", err)
	}
	// Clamp best window hours to valid range
	if cfg.BestWindowHours < 1 {
		cfg.BestWindowHours = 1
	} else if cfg.BestWindowHours > 8 {
		cfg.BestWindowHours = 8
	}
	return cfg, nil
}

// HasLicense returns true if a license key (API key) is configured.
// Actual tier validation is done by the license.Validator.
func (c *Config) HasLicense() bool {
	return c.LicenseKey != ""
}

// HasSupervisor returns true if running inside HA with Supervisor access.
func (c *Config) HasSupervisor() bool {
	return c.SupervisorToken != ""
}

// HasEnever returns true if Enever is enabled with a valid token.
func (c *Config) HasEnever() bool {
	return c.EneverEnabled && c.EneverToken != ""
}

// HasPowerSensor returns true if a power sensor is configured for live cost.
func (c *Config) HasPowerSensor() bool {
	return c.PowerSensorEntity != ""
}

// HasSynctaclesServer returns true if a Synctacles server URL is configured.
func (c *Config) HasSynctaclesServer() bool {
	return c.SynctaclesURL != ""
}

// HasAlerts returns true if price alerts are enabled with a valid threshold.
func (c *Config) HasAlerts() bool {
	return c.AlertEnabled && c.AlertThreshold > 0
}
