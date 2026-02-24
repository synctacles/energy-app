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
				{Code: "NL", EIC: "10YNL----------L", Name: "Netherlands", Country: "NL", Timezone: "Europe/Amsterdam"},
			},
		},
		{
			Country:  "DE",
			Name:     "Germany",
			Currency: "EUR",
			Zones: []ZoneInfo{
				{Code: "DE-LU", EIC: "10Y1001A1001A82H", Name: "Germany/Luxembourg", Country: "DE", Timezone: "Europe/Berlin"},
				{Code: "DE-AT-LU", EIC: "10Y1001A1001A63L", Name: "Germany/Austria/Luxembourg", Country: "DE", Timezone: "Europe/Berlin"},
			},
		},
		{
			Country:  "NO",
			Name:     "Norway",
			Currency: "NOK",
			Zones: []ZoneInfo{
				{Code: "NO1", EIC: "10YNO-1--------2", Name: "Oslo", Country: "NO", Timezone: "Europe/Oslo"},
				{Code: "NO2", EIC: "10YNO-2--------T", Name: "Kristiansand", Country: "NO", Timezone: "Europe/Oslo"},
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

	expected := []string{"DE-AT-LU", "DE-LU", "NL", "NO1", "NO2"}
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
