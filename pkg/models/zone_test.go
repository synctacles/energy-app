package models

import (
	"sort"
	"testing"
)

func testRegistry() *ZoneRegistry {
	configs := []*CountryConfig{
		{
			Country:  "NL",
			Name:     "Netherlands",
			Currency: "EUR",
			Zones: []ZoneInfo{
				{Code: "NL", EIC: "10YNL----------L", Name: "Netherlands", Country: "NL", Timezone: "Europe/Amsterdam", Lat: 52.37, Lon: 5.30},
			},
		},
		{
			Country:  "DE",
			Name:     "Germany",
			Currency: "EUR",
			Zones: []ZoneInfo{
				{Code: "DE-LU", EIC: "10Y1001A1001A82H", Name: "Germany/Luxembourg", Country: "DE", Timezone: "Europe/Berlin", Lat: 51.16, Lon: 10.45},
				{Code: "DE-AT-LU", EIC: "10Y1001A1001A63L", Name: "Germany/Austria/Luxembourg", Country: "DE", Timezone: "Europe/Berlin"},
			},
		},
		{
			Country:  "NO",
			Name:     "Norway",
			Currency: "NOK",
			Zones: []ZoneInfo{
				{Code: "NO1", EIC: "10YNO-1--------2", Name: "Oslo", Country: "NO", Timezone: "Europe/Oslo", Lat: 59.91, Lon: 10.75},
				{Code: "NO2", EIC: "10YNO-2--------T", Name: "Kristiansand", Country: "NO", Timezone: "Europe/Oslo", Lat: 58.16, Lon: 7.99},
			},
		},
		{
			Country:  "PT",
			Name:     "Portugal",
			Currency: "EUR",
			Zones: []ZoneInfo{
				{Code: "PT", EIC: "10YPT-REN------W", Name: "Portugal", Country: "PT", Timezone: "Europe/Lisbon", Lat: 39.60, Lon: -8.00},
			},
		},
	}
	return NewZoneRegistry(configs)
}

func TestZoneRegistry_GetZone(t *testing.T) {
	r := testRegistry()

	z, ok := r.GetZone("NL")
	if !ok {
		t.Fatal("expected NL zone to exist")
	}
	if z.Country != "NL" || z.Timezone != "Europe/Amsterdam" {
		t.Errorf("NL zone: country=%s, tz=%s", z.Country, z.Timezone)
	}

	z, ok = r.GetZone("DE-LU")
	if !ok {
		t.Fatal("expected DE-LU zone to exist")
	}
	if z.Country != "DE" {
		t.Errorf("DE-LU zone: country=%s, want DE", z.Country)
	}
}

func TestZoneRegistry_GetZone_NotFound(t *testing.T) {
	r := testRegistry()
	_, ok := r.GetZone("XX")
	if ok {
		t.Error("expected unknown zone to not be found")
	}
}

func TestZoneRegistry_GetCountry(t *testing.T) {
	r := testRegistry()

	cc, ok := r.GetCountry("NO")
	if !ok {
		t.Fatal("expected NO country to exist")
	}
	if cc.Name != "Norway" || cc.Currency != "NOK" {
		t.Errorf("NO country: name=%s, currency=%s", cc.Name, cc.Currency)
	}
}

func TestZoneRegistry_GetCountry_NotFound(t *testing.T) {
	r := testRegistry()
	_, ok := r.GetCountry("XX")
	if ok {
		t.Error("expected unknown country to not be found")
	}
}

func TestZoneRegistry_GetCountryForZone(t *testing.T) {
	r := testRegistry()

	cc, ok := r.GetCountryForZone("NO1")
	if !ok {
		t.Fatal("expected country for NO1 to be found")
	}
	if cc.Country != "NO" {
		t.Errorf("country for NO1: got %s, want NO", cc.Country)
	}

	cc, ok = r.GetCountryForZone("DE-LU")
	if !ok {
		t.Fatal("expected country for DE-LU to be found")
	}
	if cc.Country != "DE" {
		t.Errorf("country for DE-LU: got %s, want DE", cc.Country)
	}
}

func TestZoneRegistry_GetCountryForZone_NotFound(t *testing.T) {
	r := testRegistry()
	_, ok := r.GetCountryForZone("FAKE")
	if ok {
		t.Error("expected unknown zone to return no country")
	}
}

func TestZoneRegistry_AllZones(t *testing.T) {
	r := testRegistry()
	zones := r.AllZones()
	sort.Strings(zones)

	expected := []string{"DE-AT-LU", "DE-LU", "NL", "NO1", "NO2", "PT"}
	if len(zones) != len(expected) {
		t.Fatalf("AllZones: got %d zones, want %d: %v", len(zones), len(expected), zones)
	}
	for i, z := range zones {
		if z != expected[i] {
			t.Errorf("AllZones[%d] = %s, want %s", i, z, expected[i])
		}
	}
}

func TestZoneRegistry_Empty(t *testing.T) {
	r := NewZoneRegistry(nil)
	zones := r.AllZones()
	if len(zones) != 0 {
		t.Errorf("empty registry should have 0 zones, got %d", len(zones))
	}
	_, ok := r.GetZone("NL")
	if ok {
		t.Error("empty registry should not find any zone")
	}
}

func TestZoneRegistry_MultipleZonesPerCountry(t *testing.T) {
	r := testRegistry()
	// Norway has 2 zones
	no1, ok1 := r.GetZone("NO1")
	no2, ok2 := r.GetZone("NO2")
	if !ok1 || !ok2 {
		t.Fatal("expected both NO zones to exist")
	}
	if no1.Country != "NO" || no2.Country != "NO" {
		t.Error("both NO zones should have country=NO")
	}
	if no1.Name == no2.Name {
		t.Error("NO1 and NO2 should have different names")
	}
}

func TestDetectZone_Coordinates_NL(t *testing.T) {
	r := testRegistry()
	// Barendrecht, NL (Hetzner HA)
	result := r.DetectZone(51.84, 4.54, "Europe/Amsterdam", "NL")
	if result == nil {
		t.Fatal("expected detection result")
	}
	if result.Zone.Code != "NL" {
		t.Errorf("zone: got %s, want NL", result.Zone.Code)
	}
	if result.Method != "coordinates" {
		t.Errorf("method: got %s, want coordinates", result.Method)
	}
	if result.Mismatch {
		t.Error("should not be a mismatch for NL coords + NL country")
	}
}

func TestDetectZone_Coordinates_Mismatch(t *testing.T) {
	r := testRegistry()
	// Madeira, PT coordinates but HA country=NL (Laptop HA case)
	result := r.DetectZone(32.65, -16.83, "Europe/Lisbon", "NL")
	if result == nil {
		t.Fatal("expected detection result")
	}
	if result.Zone.Code != "PT" {
		t.Errorf("zone: got %s, want PT", result.Zone.Code)
	}
	if result.Method != "coordinates" {
		t.Errorf("method: got %s, want coordinates", result.Method)
	}
	if !result.Mismatch {
		t.Error("should detect mismatch: PT coords but NL country")
	}
	if result.HACountry != "NL" {
		t.Errorf("ha_country: got %s, want NL", result.HACountry)
	}
}

func TestDetectZone_Norway_NearestZone(t *testing.T) {
	r := testRegistry()
	// Bergen (NO5 area) — should pick NO2 (Kristiansand, closer) or NO1 (Oslo)
	// depending on coordinates
	result := r.DetectZone(60.39, 5.32, "Europe/Oslo", "NO")
	if result == nil {
		t.Fatal("expected detection result")
	}
	if result.Country != "NO" {
		t.Errorf("country: got %s, want NO", result.Country)
	}
	if result.Method != "coordinates" {
		t.Errorf("method: got %s, want coordinates", result.Method)
	}
}

func TestDetectZone_TimezoneOnly(t *testing.T) {
	r := testRegistry()
	// No coordinates, timezone match
	result := r.DetectZone(0, 0, "Europe/Amsterdam", "")
	if result == nil {
		t.Fatal("expected detection result")
	}
	if result.Zone.Code != "NL" {
		t.Errorf("zone: got %s, want NL", result.Zone.Code)
	}
	if result.Method != "timezone" {
		t.Errorf("method: got %s, want timezone", result.Method)
	}
}

func TestDetectZone_CountryFallback(t *testing.T) {
	r := testRegistry()
	// No coordinates, no matching timezone, country only
	result := r.DetectZone(0, 0, "Unknown/TZ", "NL")
	if result == nil {
		t.Fatal("expected detection result")
	}
	if result.Zone.Code != "NL" {
		t.Errorf("zone: got %s, want NL", result.Zone.Code)
	}
	if result.Method != "country" {
		t.Errorf("method: got %s, want country", result.Method)
	}
}

func TestDetectZone_NoMatch(t *testing.T) {
	r := testRegistry()
	result := r.DetectZone(0, 0, "Asia/Tokyo", "JP")
	if result != nil {
		t.Errorf("expected nil for unsupported region, got zone=%s", result.Zone.Code)
	}
}
