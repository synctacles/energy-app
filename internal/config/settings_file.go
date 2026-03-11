package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

const settingsFileName = "web-settings.json"

// SettingsFilePath returns the path to the web settings backup file.
func SettingsFilePath(dataPath string) string {
	return filepath.Join(dataPath, settingsFileName)
}

// SaveSettingsFile writes all non-schema config fields to a backup file.
// Called by the web UI on every settings save.
func SaveSettingsFile(path string, settings map[string]any) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadSettingsFile reads the settings backup file.
func LoadSettingsFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// RestoreFromSettingsFile reads the backup file and applies non-schema fields
// to the config. Schema fields (zone, pricing_mode, debug_mode, telemetry_enabled)
// always come from HA addon options and are NOT overridden.
func RestoreFromSettingsFile(cfg *Config, dataPath string) {
	path := SettingsFilePath(dataPath)
	settings, err := LoadSettingsFile(path)
	if err != nil {
		return // no backup yet — first run or fresh install
	}

	restored := 0

	// Thresholds
	if v, ok := toFloat(settings["go_threshold"]); ok {
		cfg.GoThreshold = v
		restored++
	}
	if v, ok := toFloat(settings["avoid_threshold"]); ok {
		cfg.AvoidThreshold = v
		restored++
	}
	if v, ok := settings["best_window_hours"].(float64); ok && int(v) >= 1 && int(v) <= 8 {
		cfg.BestWindowHours = int(v)
		restored++
	}

	// Enever (NL only)
	if v, ok := settings["enever_token"].(string); ok {
		cfg.EneverToken = v
		restored++
	}
	if v, ok := settings["enever_leverancier"].(string); ok {
		cfg.EneverLeverancier = v
		restored++
	}
	if v, ok := settings["enever_enabled"].(bool); ok {
		cfg.EneverEnabled = v
	}

	// Supplier
	if v, ok := settings["supplier_id"].(string); ok {
		cfg.SupplierID = v
		restored++
	}
	if v, ok := toFloat(settings["supplier_markup"]); ok {
		cfg.SupplierMarkup = v
		restored++
	}

	// Manual tax
	if v, ok := toFloat(settings["manual_vat_rate"]); ok {
		cfg.ManualVATRate = v
		restored++
	}
	if v, ok := toFloat(settings["manual_energy_tax"]); ok {
		cfg.ManualEnergyTax = v
		restored++
	}
	if v, ok := toFloat(settings["manual_surcharges"]); ok {
		cfg.ManualSurcharges = v
		restored++
	}
	if v, ok := toFloat(settings["manual_network_tariff"]); ok {
		cfg.ManualNetworkTariff = v
		restored++
	}

	// Fixed rate
	if v, ok := toFloat(settings["fixed_rate_price"]); ok {
		cfg.FixedRatePrice = v
		restored++
	}

	// TOU config (stored as JSON string)
	if v, ok := settings["tou_config"].(string); ok && v != "" {
		cfg.TOUConfigJSON = v
		restored++
	}

	// External sensor
	if v, ok := settings["p1_sensor_entity"].(string); ok {
		cfg.P1SensorEntity = v
		restored++
	}

	// Power sensor
	if v, ok := settings["power_sensor"].(string); ok {
		cfg.PowerSensorEntity = v
		restored++
	}

	// Alerts
	if v, ok := settings["alerts_enabled"].(bool); ok {
		cfg.AlertEnabled = v
	}
	if v, ok := toFloat(settings["alert_threshold"]); ok {
		cfg.AlertThreshold = v
		restored++
	}

	// Consent flags (non-schema in new version)
	if v, ok := settings["disclaimer_accepted"].(bool); ok {
		cfg.DisclaimerAccepted = v
	}
	if v, ok := settings["privacy_accepted"].(bool); ok {
		cfg.PrivacyAccepted = v
	}
	if v, ok := settings["onboarding_completed"].(bool); ok {
		cfg.OnboardingCompleted = v
	}

	if restored > 0 {
		slog.Info("restored settings from backup", "fields", restored)
	}
}

// BuildSettingsMap creates a map of all non-schema fields from the current config.
// Used for saving the backup file.
func BuildSettingsMap(cfg *Config) map[string]any {
	return map[string]any{
		"go_threshold":         cfg.GoThreshold,
		"avoid_threshold":      cfg.AvoidThreshold,
		"best_window_hours":    cfg.BestWindowHours,
		"enever_enabled":       cfg.EneverEnabled,
		"enever_token":         cfg.EneverToken,
		"enever_leverancier":   cfg.EneverLeverancier,
		"supplier_id":          cfg.SupplierID,
		"supplier_markup":      cfg.SupplierMarkup,
		"manual_vat_rate":      cfg.ManualVATRate,
		"manual_energy_tax":    cfg.ManualEnergyTax,
		"manual_surcharges":    cfg.ManualSurcharges,
		"manual_network_tariff": cfg.ManualNetworkTariff,
		"fixed_rate_price":     cfg.FixedRatePrice,
		"tou_config":           cfg.TOUConfigJSON,
		"p1_sensor_entity":     cfg.P1SensorEntity,
		"power_sensor":         cfg.PowerSensorEntity,
		"alerts_enabled":       cfg.AlertEnabled,
		"alert_threshold":      cfg.AlertThreshold,
		"disclaimer_accepted":  cfg.DisclaimerAccepted,
		"privacy_accepted":     cfg.PrivacyAccepted,
		"onboarding_completed": cfg.OnboardingCompleted,
	}
}

// --- Consent file: dedicated storage for consent flags ---
// Separate from settings backup to avoid Supervisor options sync issues.

const consentFileName = "consent.json"

// ConsentFilePath returns the path to the consent file.
func ConsentFilePath(dataPath string) string {
	return filepath.Join(dataPath, consentFileName)
}

// ConsentState holds the user's consent flags.
type ConsentState struct {
	DisclaimerAccepted  bool `json:"disclaimer_accepted"`
	PrivacyAccepted     bool `json:"privacy_accepted"`
	OnboardingCompleted bool `json:"onboarding_completed"`
}

// SaveConsent writes consent flags to a dedicated file.
func SaveConsent(dataPath string, state ConsentState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConsentFilePath(dataPath), data, 0644)
}

// LoadConsent reads consent flags from the dedicated file.
func LoadConsent(dataPath string) (ConsentState, error) {
	data, err := os.ReadFile(ConsentFilePath(dataPath))
	if err != nil {
		return ConsentState{}, err
	}
	var state ConsentState
	if err := json.Unmarshal(data, &state); err != nil {
		return ConsentState{}, err
	}
	return state, nil
}

// RestoreConsent applies consent flags from the dedicated file to the config.
// Called on startup after config.Load().
func RestoreConsent(cfg *Config, dataPath string) {
	state, err := LoadConsent(dataPath)
	if err != nil {
		return // no consent file yet — first run
	}
	if state.DisclaimerAccepted {
		cfg.DisclaimerAccepted = true
	}
	if state.PrivacyAccepted {
		cfg.PrivacyAccepted = true
	}
	if state.OnboardingCompleted {
		cfg.OnboardingCompleted = true
	}
	slog.Info("restored consent from file", "disclaimer", state.DisclaimerAccepted, "privacy", state.PrivacyAccepted, "onboarding", state.OnboardingCompleted)
}

// toFloat handles JSON numbers (always float64 in Go's json package).
func toFloat(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}
