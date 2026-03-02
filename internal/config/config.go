// Package config loads energy addon settings from environment variables.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds settings for the Energy HA app.
type Config struct {
	// HA
	SupervisorToken string `env:"SUPERVISOR_TOKEN"`
	IngressPort     int    `env:"INGRESS_PORT" envDefault:"8098"`

	// Pricing mode: "auto", "manual", "p1_meter", "enever"
	PricingMode string `env:"PRICING_MODE" envDefault:"auto"`

	// Zone
	BiddingZone string `env:"ENERGY_ZONE" envDefault:"NL"`
	Currency    string `env:"ENERGY_CURRENCY" envDefault:"EUR"`

	// Thresholds (user configurable via app options)
	GoThreshold    float64 `env:"ENERGY_GO_THRESHOLD" envDefault:"-15"`
	AvoidThreshold float64 `env:"ENERGY_AVOID_THRESHOLD" envDefault:"20"`

	// Enever (pricing mode "enever", NL only)
	EneverEnabled     bool   `env:"ENEVER_ENABLED" envDefault:"false"`
	EneverToken       string `env:"ENEVER_TOKEN"`
	EneverLeverancier string `env:"ENEVER_LEVERANCIER" envDefault:"frank"`

	// Supplier markup in EUR/kWh (used in auto + manual modes)
	SupplierMarkup float64 `env:"ENERGY_SUPPLIER_MARKUP" envDefault:"0"`

	// Manual mode: user-defined tax components
	ManualVATRate       float64 `env:"MANUAL_VAT_RATE" envDefault:"0"`
	ManualEnergyTax     float64 `env:"MANUAL_ENERGY_TAX" envDefault:"0"`
	ManualSurcharges    float64 `env:"MANUAL_SURCHARGES" envDefault:"0"`
	ManualNetworkTariff float64 `env:"MANUAL_NETWORK_TARIFF" envDefault:"0"`

	// Fixed-rate mode: user-defined flat rate (no dynamic pricing)
	FixedRatePrice float64 `env:"FIXED_RATE_PRICE" envDefault:"0"`

	// P1 mode: HA sensor entity for consumer tariff
	P1SensorEntity string `env:"P1_SENSOR_ENTITY"`

	// Best window duration in hours (1-8, default 3)
	BestWindowHours int `env:"BEST_WINDOW_HOURS" envDefault:"3"`

	// Power sensor for Live Cost calculation
	PowerSensorEntity string `env:"POWER_SENSOR_ENTITY"`

	// Price alerts — HA persistent notification when price drops below threshold
	AlertEnabled   bool    `env:"ENERGY_ALERTS_ENABLED" envDefault:"false"`
	AlertThreshold float64 `env:"ENERGY_ALERT_THRESHOLD" envDefault:"0"`

	// Debug
	DebugMode bool `env:"DEBUG_MODE" envDefault:"false"`
}

// Valid pricing modes.
const (
	ModeAuto         = "auto"
	ModeManual       = "manual"
	ModeP1Meter      = "p1_meter"      // Legacy name, kept for backward compat
	ModeMeterTariff  = "meter_tariff"   // New canonical name for smart meter mode
	ModeEnever       = "enever"
	ModeFixed        = "fixed"          // User-defined flat rate, no dynamic pricing
)

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
	// Validate pricing mode
	switch cfg.PricingMode {
	case ModeAuto, ModeManual, ModeP1Meter, ModeMeterTariff, ModeEnever, ModeFixed:
		// OK
	default:
		cfg.PricingMode = ModeAuto
	}
	return cfg, nil
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

// HasAlerts returns true if price alerts are enabled with a valid threshold.
func (c *Config) HasAlerts() bool {
	return c.AlertEnabled && c.AlertThreshold > 0
}

// IsFixedMode returns true if pricing mode is fixed-rate with a configured price.
func (c *Config) IsFixedMode() bool {
	return c.PricingMode == ModeFixed && c.FixedRatePrice > 0
}

// IsEneverMode returns true if pricing mode is Enever with valid credentials.
func (c *Config) IsEneverMode() bool {
	return c.PricingMode == ModeEnever && c.EneverToken != ""
}

// IsMeterTariffMode returns true if pricing mode is meter tariff (smart meter)
// with a configured sensor. Accepts both legacy "p1_meter" and new "meter_tariff".
func (c *Config) IsMeterTariffMode() bool {
	return (c.PricingMode == ModeP1Meter || c.PricingMode == ModeMeterTariff) && c.P1SensorEntity != ""
}

// ValidateTaxInputs validates user-entered tax components against CC_INSTRUCTION §10 ranges.
// Returns an error describing the first invalid value, or nil if all values are valid.
func ValidateTaxInputs(vatRate, energyTax, surcharges, networkTariff float64) error {
	if vatRate < 0 || vatRate > 0.50 {
		return fmt.Errorf("VAT rate must be between 0%% and 50%%")
	}
	if energyTax < 0 || energyTax > 0.50 {
		return fmt.Errorf("energy tax must be between 0 and 0.50 EUR/kWh")
	}
	if surcharges < 0 || surcharges > 0.50 {
		return fmt.Errorf("surcharges must be between 0 and 0.50 EUR/kWh")
	}
	if networkTariff < 0 || networkTariff > 0.50 {
		return fmt.Errorf("network tariff must be between 0 and 0.50 EUR/kWh")
	}
	return nil
}
