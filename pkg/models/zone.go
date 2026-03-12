package models

import "math"

// ZoneInfo describes a bidding zone and its configuration.
type ZoneInfo struct {
	Code     string  `yaml:"code" json:"code"`         // "NL", "DE-LU", "NO1", etc.
	EIC      string  `yaml:"eic" json:"eic"`           // ENTSO-E EIC code
	Name     string  `yaml:"name" json:"name"`         // "Netherlands"
	Country  string  `yaml:"country" json:"country"`   // "NL", "DE", "NO"
	Timezone string  `yaml:"timezone" json:"timezone"` // "Europe/Amsterdam"
	Lat      float64 `yaml:"lat" json:"lat"`           // zone centroid latitude
	Lon      float64 `yaml:"lon" json:"lon"`           // zone centroid longitude

	// Zone-level overrides (optional). When set, these override the country-level defaults.
	TaxDefaults      *EmbeddedTaxDefaults `yaml:"tax_defaults,omitempty" json:"tax_defaults,omitempty"`
	RegulatedTariffs *RegulatedTariffs    `yaml:"regulated_tariffs,omitempty" json:"regulated_tariffs,omitempty"`
}

// HasWholesale returns true if this zone has ENTSO-E wholesale market data.
func (z ZoneInfo) HasWholesale() bool {
	return z.EIC != ""
}

// RegulatedTariffs defines pre-set tariff rates for zones with regulated (non-wholesale) pricing.
type RegulatedTariffs struct {
	Fixed float64                      `yaml:"fixed" json:"fixed"` // flat rate EUR/kWh
	TOU   map[string]RegulatedTOURates `yaml:"tou" json:"tou"`    // preset_id → rates
}

// RegulatedTOURates defines peak/offpeak rates for a regulated TOU preset.
type RegulatedTOURates struct {
	Peak    float64 `yaml:"peak" json:"peak"`       // EUR/kWh
	Offpeak float64 `yaml:"offpeak" json:"offpeak"` // EUR/kWh
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
// Zone-level tax_defaults override country-level when present.
func (r *ZoneRegistry) GetTaxDefaults(zoneCode string) *EmbeddedTaxDefaults {
	z, ok := r.GetZone(zoneCode)
	if !ok {
		return nil
	}
	// Zone-level override takes precedence
	if z.TaxDefaults != nil {
		return z.TaxDefaults
	}
	cc, ok := r.GetCountry(z.Country)
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

// ZoneDetectResult holds the result of zone auto-detection.
type ZoneDetectResult struct {
	Zone       ZoneInfo `json:"zone"`
	Country    string   `json:"country"`
	Method     string   `json:"method"`     // "coordinates", "timezone", "country"
	Mismatch   bool     `json:"mismatch"`   // true if HA country differs from detected
	HACountry  string   `json:"ha_country"` // original HA country setting
	Distance   float64  `json:"distance"`   // km from zone centroid (coordinates method)
}

// DetectZone determines the best matching zone using HA config signals.
// Priority: coordinates (strongest) > timezone > HA country (weakest).
// For coordinates, timezone is used as tiebreaker when multiple zones
// are within reasonable distance (e.g. NL vs BE vs DE borders).
func (r *ZoneRegistry) DetectZone(lat, lon float64, timezone, haCountry string) *ZoneDetectResult {
	// 1. Coordinates: find nearest zone, prefer timezone match as tiebreaker
	if lat != 0 || lon != 0 {
		best, dist := r.nearestZone(lat, lon)
		if best.Code != "" {
			// If timezone matches a different zone that's also close, prefer that
			if timezone != "" && best.Timezone != timezone {
				tzBest, tzDist := r.nearestZoneWithTimezone(lat, lon, timezone)
				if tzBest.Code != "" && tzDist < dist*1.5 {
					// Timezone-matching zone is within 50% of nearest → prefer it
					best = tzBest
					dist = tzDist
				}
			}
			return &ZoneDetectResult{
				Zone:      best,
				Country:   best.Country,
				Method:    "coordinates",
				Mismatch:  haCountry != "" && haCountry != best.Country,
				HACountry: haCountry,
				Distance:  dist,
			}
		}
	}

	// 2. Timezone match
	if timezone != "" {
		for _, code := range r.AllZones() {
			z, ok := r.GetZone(code)
			if !ok {
				continue
			}
			if z.Timezone == timezone {
				return &ZoneDetectResult{
					Zone:      z,
					Country:   z.Country,
					Method:    "timezone",
					Mismatch:  haCountry != "" && haCountry != z.Country,
					HACountry: haCountry,
				}
			}
		}
	}

	// 3. HA country as fallback
	if haCountry != "" {
		z, ok := r.GetZone(haCountry)
		if ok {
			return &ZoneDetectResult{
				Zone:      z,
				Country:   z.Country,
				Method:    "country",
				HACountry: haCountry,
			}
		}
		// Try alias (e.g. "DE" → "DE-LU")
		if alias, hasAlias := r.aliases[haCountry]; hasAlias {
			z, ok = r.GetZone(alias)
			if ok {
				return &ZoneDetectResult{
					Zone:      z,
					Country:   z.Country,
					Method:    "country",
					HACountry: haCountry,
				}
			}
		}
	}

	return nil
}

// nearestZone finds the zone with centroid closest to the given coordinates.
func (r *ZoneRegistry) nearestZone(lat, lon float64) (ZoneInfo, float64) {
	var best ZoneInfo
	bestDist := math.MaxFloat64
	for _, z := range r.zones {
		if z.Lat == 0 && z.Lon == 0 {
			continue
		}
		d := haversine(lat, lon, z.Lat, z.Lon)
		if d < bestDist {
			bestDist = d
			best = z
		}
	}
	return best, bestDist
}

// nearestZoneWithTimezone finds the nearest zone that matches the given timezone.
func (r *ZoneRegistry) nearestZoneWithTimezone(lat, lon float64, timezone string) (ZoneInfo, float64) {
	var best ZoneInfo
	bestDist := math.MaxFloat64
	for _, z := range r.zones {
		if z.Lat == 0 && z.Lon == 0 {
			continue
		}
		if z.Timezone != timezone {
			continue
		}
		d := haversine(lat, lon, z.Lat, z.Lon)
		if d < bestDist {
			bestDist = d
			best = z
		}
	}
	return best, bestDist
}

// haversine returns the great-circle distance in km between two points.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
