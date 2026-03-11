package models

// ZoneInfo describes a bidding zone and its configuration.
type ZoneInfo struct {
	Code     string `yaml:"code" json:"code"`         // "NL", "DE-LU", "NO1", etc.
	EIC      string `yaml:"eic" json:"eic"`           // ENTSO-E EIC code
	Name     string `yaml:"name" json:"name"`         // "Netherlands"
	Country  string `yaml:"country" json:"country"`   // "NL", "DE", "NO"
	Timezone string `yaml:"timezone" json:"timezone"` // "Europe/Amsterdam"
}

// EmbeddedTaxDefaults provides fallback tax data for cold-start scenarios when
// the Worker tax profile hasn't been fetched yet. These values are intentionally
// conservative estimates that are overridden by live Worker data when available.
type EmbeddedTaxDefaults struct {
	VATRate          float64 `yaml:"vat_rate" json:"vat_rate"`
	EnergyTax        float64 `yaml:"energy_tax" json:"energy_tax"`                 // EUR/kWh
	Surcharges       float64 `yaml:"surcharges" json:"surcharges"`                 // EUR/kWh
	NetworkTariffAvg float64 `yaml:"network_tariff_avg" json:"network_tariff_avg"` // EUR/kWh
	ValidFrom        string  `yaml:"valid_from" json:"valid_from"`                 // "2026-01-01"
}

// Supplier describes a known electricity supplier for a country.
type Supplier struct {
	ID     string  `yaml:"id" json:"id"`         // "tibber", "awattar"
	Name   string  `yaml:"name" json:"name"`     // "Tibber"
	Markup float64 `yaml:"markup" json:"markup"` // markup per kWh in local currency
}

// TOUPresetPeriod defines a time range within a TOU preset.
type TOUPresetPeriod struct {
	Days  []int  `yaml:"days" json:"days"`   // 0=Sun, 1=Mon, ..., 6=Sat
	Start string `yaml:"start" json:"start"` // "08:00"
	End   string `yaml:"end" json:"end"`     // "22:00"
	Rate  string `yaml:"rate" json:"rate"`   // "peak", "midpeak"
}

// TOUPreset defines a country-specific time-of-use schedule template.
// Users select a preset to auto-fill the schedule; they only need to enter rates.
type TOUPreset struct {
	ID      string            `yaml:"id" json:"id"`           // "ciclo_diario"
	Name    string            `yaml:"name" json:"name"`       // "Ciclo diário"
	Periods []TOUPresetPeriod `yaml:"periods" json:"periods"` // schedule template
}

// CountryConfig defines the full configuration for a country.
// Live tax data comes from the Worker (see TaxProfileCache).
// TaxDefaults provides embedded fallback for cold-start when Worker is unreachable.
type CountryConfig struct {
	Country     string               `yaml:"country" json:"country"`
	Name        string               `yaml:"name" json:"name"`
	Currency    string               `yaml:"currency" json:"currency"`
	Zones       []ZoneInfo           `yaml:"zones" json:"zones"`
	TaxDefaults *EmbeddedTaxDefaults `yaml:"tax_defaults,omitempty" json:"tax_defaults,omitempty"`
	Suppliers   []Supplier           `yaml:"suppliers,omitempty" json:"suppliers,omitempty"`
	TOUPresets  []TOUPreset          `yaml:"tou_presets,omitempty" json:"tou_presets,omitempty"`
}

// ZoneRegistry provides lookup for bidding zones.
type ZoneRegistry struct {
	zones     map[string]ZoneInfo
	aliases   map[string]string // country code → bidding zone code (e.g. "DE" → "DE-LU")
	countries map[string]*CountryConfig
}

// NewZoneRegistry creates a registry from country configs.
func NewZoneRegistry(configs []*CountryConfig) *ZoneRegistry {
	r := &ZoneRegistry{
		zones:     make(map[string]ZoneInfo),
		aliases:   make(map[string]string),
		countries: make(map[string]*CountryConfig),
	}
	for _, cc := range configs {
		r.countries[cc.Country] = cc
		for _, z := range cc.Zones {
			r.zones[z.Code] = z
		}
		// Register country code as alias for the first zone if different.
		// This allows "DE" to resolve to "DE-LU", "NO" to "NO1", etc.
		if len(cc.Zones) > 0 && cc.Country != cc.Zones[0].Code {
			r.aliases[cc.Country] = cc.Zones[0].Code
		}
	}
	return r
}

// GetZone returns zone info by code, or false if not found.
// Also checks country-code aliases (e.g. "DE" resolves to "DE-LU").
func (r *ZoneRegistry) GetZone(code string) (ZoneInfo, bool) {
	z, ok := r.zones[code]
	if ok {
		return z, true
	}
	if alias, hasAlias := r.aliases[code]; hasAlias {
		z, ok = r.zones[alias]
		return z, ok
	}
	return ZoneInfo{}, false
}

// GetCountry returns country config by country code.
func (r *ZoneRegistry) GetCountry(code string) (*CountryConfig, bool) {
	cc, ok := r.countries[code]
	return cc, ok
}

// GetCountryForZone returns the country config for a given zone code.
// Also resolves country-code aliases (e.g. "DE" resolves to "DE-LU").
func (r *ZoneRegistry) GetCountryForZone(zoneCode string) (*CountryConfig, bool) {
	z, ok := r.GetZone(zoneCode)
	if !ok {
		return nil, false
	}
	return r.GetCountry(z.Country)
}

// GetTaxDefaults returns embedded fallback tax defaults for a zone, or nil if unavailable.
func (r *ZoneRegistry) GetTaxDefaults(zoneCode string) *EmbeddedTaxDefaults {
	cc, ok := r.GetCountryForZone(zoneCode)
	if !ok || cc.TaxDefaults == nil {
		return nil
	}
	return cc.TaxDefaults
}

// AllZones returns all registered zone codes.
func (r *ZoneRegistry) AllZones() []string {
	codes := make([]string, 0, len(r.zones))
	for code := range r.zones {
		codes = append(codes, code)
	}
	return codes
}
