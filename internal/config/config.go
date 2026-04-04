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

	// Pricing mode: "auto", "manual", "external_sensor", "fixed"
	PricingMode string `env:"PRICING_MODE" envDefault:"auto"`

	// Zone
	BiddingZone string `env:"ENERGY_ZONE" envDefault:"NL"`
	Currency    string `env:"ENERGY_CURRENCY" envDefault:"EUR"`

	// Thresholds (user configurable via app options)
	GoThreshold    float64 `env:"ENERGY_GO_THRESHOLD" envDefault:"-15"`
	AvoidThreshold float64 `env:"ENERGY_AVOID_THRESHOLD" envDefault:"20"`

	// Supplier markup in EUR/kWh (used in auto + manual modes)
	SupplierMarkup float64 `env:"ENERGY_SUPPLIER_MARKUP" envDefault:"0"`

	// Selected supplier ID (e.g. "tibber", "awattar") — pre-fills SupplierMarkup
	SupplierID string `env:"ENERGY_SUPPLIER_ID" envDefault:""`

	// Manual mode: user-defined tax components
	ManualVATRate       float64 `env:"MANUAL_VAT_RATE" envDefault:"0"`
	ManualEnergyTax     float64 `env:"MANUAL_ENERGY_TAX" envDefault:"0"`
	ManualSurcharges    float64 `env:"MANUAL_SURCHARGES" envDefault:"0"`
	ManualNetworkTariff float64 `env:"MANUAL_NETWORK_TARIFF" envDefault:"0"`

	// Fixed-rate mode: user-defined flat rate (no dynamic pricing)
	FixedRatePrice float64 `env:"FIXED_RATE_PRICE" envDefault:"0"`

	// Time-of-Use mode: flexible schedule with peak/off-peak rates (JSON)
	TOUConfigJSON string `env:"TOU_CONFIG" envDefault:""`

	// External sensor mode: HA sensor entity for consumer tariff (env kept as P1_SENSOR_ENTITY for backward compat)
	P1SensorEntity string `env:"P1_SENSOR_ENTITY"`

	// Best window duration in hours (1-8, default 3)
	BestWindowHours int `env:"BEST_WINDOW_HOURS" envDefault:"3"`

	// Power sensor for Live Cost calculation
	PowerSensorEntity string `env:"POWER_SENSOR_ENTITY"`

	// Price alerts — HA persistent notification when price drops below threshold
	AlertEnabled   bool    `env:"ENERGY_ALERTS_ENABLED" envDefault:"false"`
	AlertThreshold float64 `env:"ENERGY_ALERT_THRESHOLD" envDefault:"0"`

	// Telemetry opt-out (default: enabled, user can disable via Settings)
	TelemetryEnabled bool `env:"TELEMETRY_ENABLED" envDefault:"true"`

	// Consent flags (persisted in HA Supervisor options)
	DisclaimerAccepted  bool `env:"DISCLAIMER_ACCEPTED" envDefault:"false"`
	PrivacyAccepted     bool `env:"PRIVACY_ACCEPTED" envDefault:"false"`
	OnboardingCompleted bool `env:"ONBOARDING_COMPLETED" envDefault:"false"`

	// Debug
	DebugMode bool `env:"DEBUG_MODE" envDefault:"false"`
}

// Valid pricing modes.
//
// Legacy mode migration (do NOT remove):
//   - "p1_meter" and "meter_tariff" are legacy names for "external_sensor".
//     They exist in user configs from pre-0.4.0 installations. The Load()
//     function accepts them as valid modes, and IsExternalSensorMode() treats
//     all three identically. Removing them would break existing users on update.
//   - When adding new modes, use a new canonical name and keep legacy aliases
//     in the switch statement in Load() and in IsExternalSensorMode().
const (
	ModeAuto           = "auto"
	ModeManual         = "manual"
	ModeExternalSensor = "external_sensor" // Canonical: any HA sensor with EUR/kWh tariff
	ModeP1Meter        = "p1_meter"        // Legacy alias for external_sensor (pre-0.4.0)
	ModeMeterTariff    = "meter_tariff"    // Legacy alias for external_sensor (pre-0.4.0)
	ModeFixed          = "fixed"           // User-defined flat rate, no dynamic pricing
	ModeTOU            = "tou"             // Time-of-use schedule (bi-horário / tri-horário)
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
	// Sanitize bashio "null" strings (bashio returns literal "null" for empty values)
	if cfg.P1SensorEntity == "null" {
		cfg.P1SensorEntity = ""
	}
	if cfg.PowerSensorEntity == "null" {
		cfg.PowerSensorEntity = ""
	}
	if cfg.SupplierID == "null" {
		cfg.SupplierID = ""
	}
	// Validate pricing mode
	switch cfg.PricingMode {
	case ModeAuto, ModeManual, ModeExternalSensor, ModeP1Meter, ModeMeterTariff, ModeFixed, ModeTOU:
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

// IsTOUMode returns true if pricing mode is time-of-use with a valid config.
func (c *Config) IsTOUMode() bool {
	return c.PricingMode == ModeTOU && c.TOUConfigJSON != ""
}

// IsExternalSensorMode returns true if pricing mode uses an external HA sensor
// for the consumer tariff. Accepts canonical "external_sensor" and legacy "p1_meter"/"meter_tariff".
func (c *Config) IsExternalSensorMode() bool {
	return (c.PricingMode == ModeExternalSensor || c.PricingMode == ModeP1Meter || c.PricingMode == ModeMeterTariff) && c.P1SensorEntity != ""
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
