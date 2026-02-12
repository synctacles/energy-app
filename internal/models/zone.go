package models

// ZoneInfo describes a bidding zone and its configuration.
type ZoneInfo struct {
	Code     string `yaml:"code" json:"code"`         // "NL", "DE-LU", "NO1", etc.
	EIC      string `yaml:"eic" json:"eic"`           // ENTSO-E EIC code
	Name     string `yaml:"name" json:"name"`         // "Netherlands"
	Country  string `yaml:"country" json:"country"`   // "NL", "DE", "NO"
	Timezone string `yaml:"timezone" json:"timezone"` // "Europe/Amsterdam"
}

// CountryConfig defines the full configuration for a country.
type CountryConfig struct {
	Country    string           `yaml:"country" json:"country"`
	Name       string           `yaml:"name" json:"name"`
	Currency   string           `yaml:"currency" json:"currency"`
	Zones      []ZoneInfo       `yaml:"zones" json:"zones"`
	TaxProfile TaxProfile       `yaml:"tax_profile" json:"tax_profile"`
	Sources    []SourcePriority `yaml:"sources" json:"sources"`
}

// SourcePriority maps a price source to its priority for this country.
type SourcePriority struct {
	Name     string `yaml:"name" json:"name"`
	Priority int    `yaml:"priority" json:"priority"`
}

// ZoneRegistry provides lookup for bidding zones.
type ZoneRegistry struct {
	zones     map[string]ZoneInfo
	countries map[string]*CountryConfig
}

// NewZoneRegistry creates a registry from country configs.
func NewZoneRegistry(configs []*CountryConfig) *ZoneRegistry {
	r := &ZoneRegistry{
		zones:     make(map[string]ZoneInfo),
		countries: make(map[string]*CountryConfig),
	}
	for _, cc := range configs {
		r.countries[cc.Country] = cc
		for _, z := range cc.Zones {
			r.zones[z.Code] = z
		}
	}
	return r
}

// GetZone returns zone info by code, or false if not found.
func (r *ZoneRegistry) GetZone(code string) (ZoneInfo, bool) {
	z, ok := r.zones[code]
	return z, ok
}

// GetCountry returns country config by country code.
func (r *ZoneRegistry) GetCountry(code string) (*CountryConfig, bool) {
	cc, ok := r.countries[code]
	return cc, ok
}

// GetCountryForZone returns the country config for a given zone code.
func (r *ZoneRegistry) GetCountryForZone(zoneCode string) (*CountryConfig, bool) {
	z, ok := r.zones[zoneCode]
	if !ok {
		return nil, false
	}
	return r.GetCountry(z.Country)
}

// AllZones returns all registered zone codes.
func (r *ZoneRegistry) AllZones() []string {
	codes := make([]string, 0, len(r.zones))
	for code := range r.zones {
		codes = append(codes, code)
	}
	return codes
}
