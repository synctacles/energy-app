package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/synctacles/energy-app/internal/hasensor"
	"github.com/synctacles/energy-app/pkg/platform"
)

var energyDataBaseURL = platform.EnergyDataBaseURL

// profileResponse matches the synctacles-api GET /api/v1/energy/install-profile response.
type profileResponse struct {
	Profile *installProfile `json:"profile"`
}

type installProfile struct {
	Zone             string   `json:"zone"`
	Timezone         string   `json:"timezone"`
	SupplierDomain   string   `json:"supplier_domain"`
	SupplierName     string   `json:"supplier_name"`
	ContractType     string   `json:"contract_type"`
	HasSolar         int      `json:"has_solar"`
	HasBattery       int      `json:"has_battery"`
	HasGas           int      `json:"has_gas"`
	HasGridMeter     int      `json:"has_grid_meter"`
	WholesaleKWh     *float64 `json:"wholesale_reading_kwh"`
	SupplierMarkup   *float64 `json:"supplier_markup_kwh"`
}

var platformAPIBaseURL = platform.APIBaseURL

// fetchHarvestedProfile fetches the Care-harvested install profile for wizard pre-fill.
// Returns nil on any error (best-effort).
func fetchHarvestedProfile(ctx context.Context, installUUID string) *installProfile {
	if installUUID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET",
		platformAPIBaseURL+"/api/v1/energy/install-profile?install_uuid="+installUUID, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "SynctaclesEnergy/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var result profileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	return result.Profile
}

// handleWizardData returns bundled data for the onboarding wizard:
// zones grouped by country, suppliers, tax defaults, current config, and detected zone.
func (s *Server) handleWizardData(w http.ResponseWriter, r *http.Request) {
	type zoneEntry struct {
		Code             string `json:"code"`
		Name             string `json:"name"`
		Timezone         string `json:"timezone"`
		HasWholesale     bool   `json:"has_wholesale"`
		TaxDefaults      any    `json:"tax_defaults,omitempty"`
		RegulatedTariffs any    `json:"regulated_tariffs,omitempty"`
		TOUPresets       []any  `json:"tou_presets,omitempty"`
	}
	type countryEntry struct {
		Code        string      `json:"code"`
		Name        string      `json:"name"`
		Currency    string      `json:"currency"`
		Zones       []zoneEntry `json:"zones"`
		Suppliers   []any       `json:"suppliers"`
		TaxDefaults any         `json:"tax_defaults"`
		TOUPresets  []any       `json:"tou_presets"`
	}

	// Build countries list from zone registry
	countriesMap := make(map[string]*countryEntry)
	for _, code := range s.zoneRegistry.AllZones() {
		z, ok := s.zoneRegistry.GetZone(code)
		if !ok {
			continue
		}
		cc, ok := s.zoneRegistry.GetCountryForZone(code)
		if !ok {
			continue
		}
		entry, exists := countriesMap[z.Country]
		if !exists {
			suppliers := make([]any, 0, len(cc.Suppliers))
			for _, sup := range cc.Suppliers {
				suppliers = append(suppliers, map[string]any{
					"id":     sup.ID,
					"name":   sup.Name,
					"markup": sup.Markup,
				})
			}
			var taxDefaults any
			if cc.TaxDefaults != nil {
				taxDefaults = cc.TaxDefaults
			}
			touPresets := make([]any, 0, len(cc.TOUPresets))
			for _, p := range cc.TOUPresets {
				touPresets = append(touPresets, p)
			}
			entry = &countryEntry{
				Code:        cc.Country,
				Name:        cc.Name,
				Currency:    cc.Currency,
				Zones:       []zoneEntry{},
				Suppliers:   suppliers,
				TaxDefaults: taxDefaults,
				TOUPresets:  touPresets,
			}
			countriesMap[z.Country] = entry
		}
		ze := zoneEntry{
			Code:         z.Code,
			Name:         z.Name,
			Timezone:     z.Timezone,
			HasWholesale: z.HasWholesale(),
		}
		// Zone-level tax defaults override country-level
		if z.TaxDefaults != nil {
			ze.TaxDefaults = z.TaxDefaults
		}
		if z.RegulatedTariffs != nil {
			ze.RegulatedTariffs = z.RegulatedTariffs
		}
		if len(z.TOUPresets) > 0 {
			for _, p := range z.TOUPresets {
				ze.TOUPresets = append(ze.TOUPresets, p)
			}
		}
		entry.Zones = append(entry.Zones, ze)
	}

	// Sort countries by name
	countries := make([]*countryEntry, 0, len(countriesMap))
	for _, entry := range countriesMap {
		countries = append(countries, entry)
	}
	sort.Slice(countries, func(i, j int) bool {
		return countries[i].Name < countries[j].Name
	})

	// Detect zone from HA config (coordinates > timezone > country)
	var detectedZone string
	var zoneMismatch bool
	var detectedCountryName string
	var haCountry string
	var detectMethod string
	var locationCorrections []map[string]any
	if s.supervisor != nil {
		haConfig, err := s.supervisor.GetConfig(r.Context())
		if err == nil {
			lat, _ := haConfig["latitude"].(float64)
			lon, _ := haConfig["longitude"].(float64)
			tz, _ := haConfig["time_zone"].(string)
			haCountry, _ = haConfig["country"].(string)

			result := s.zoneRegistry.DetectZone(lat, lon, tz, haCountry)
			if result != nil {
				detectedZone = result.Zone.Code
				zoneMismatch = result.Mismatch
				detectMethod = result.Method
				if result.Mismatch {
					cc, _ := s.zoneRegistry.GetCountry(result.Country)
					if cc != nil {
						detectedCountryName = cc.Name
					}
				}
				// Suggest timezone/country corrections
				if result.Zone.Timezone != "" && tz != result.Zone.Timezone {
					locationCorrections = append(locationCorrections, map[string]any{
						"field":     "time_zone",
						"current":   tz,
						"suggested": result.Zone.Timezone,
					})
				}
				if result.Country != "" && haCountry != result.Country {
					locationCorrections = append(locationCorrections, map[string]any{
						"field":     "country",
						"current":   haCountry,
						"suggested": result.Country,
					})
				}
				// Currency correction
				haCurrency, _ := haConfig["currency"].(string)
				if cc, ok := s.zoneRegistry.GetCountry(result.Country); ok && cc.Currency != "" && haCurrency != cc.Currency {
					locationCorrections = append(locationCorrections, map[string]any{
						"field":     "currency",
						"current":   haCurrency,
						"suggested": cc.Currency,
					})
				}
			}
		}
	}

	// Detect tariff sensors — generic scan for any sensor with /kWh unit
	// that looks like an electricity tariff. Checks forecast attribute to
	// identify day-ahead capable sensors (Zonneplan, Tibber, Octopus, etc.)
	type sensorInfo struct {
		EntityID      string  `json:"entity_id"`
		Name          string  `json:"name"`
		State         float64 `json:"state"`
		Unit          string  `json:"unit"`
		SupplierHint  string  `json:"supplier_hint,omitempty"`
		HasForecast   bool    `json:"has_forecast,omitempty"`
		ForecastHours int     `json:"forecast_hours,omitempty"`
	}
	var tariffSensors []sensorInfo
	var bestSensor *sensorInfo
	if s.supervisor != nil {
		states, err := s.supervisor.GetAllStates(r.Context())
		if err == nil {
			for _, state := range states {
				entityID, _ := state["entity_id"].(string)
				if entityID == "" || !strings.HasPrefix(entityID, "sensor.") {
					continue
				}
				lower := strings.ToLower(entityID)
				if strings.Contains(lower, "synctacles") || strings.Contains(lower, "gas") {
					continue
				}
				attrs, _ := state["attributes"].(map[string]any)
				unit := ""
				if attrs != nil {
					if u, ok := attrs["unit_of_measurement"].(string); ok {
						unit = u
					}
				}
				if !strings.Contains(strings.ToLower(unit), "/kwh") {
					continue
				}
				stateVal, _ := state["state"].(string)
				val, err := strconv.ParseFloat(stateVal, 64)
				if err != nil || val <= 0 {
					continue
				}
				isTariff := strings.Contains(lower, "tariff") || strings.Contains(lower, "tarief") ||
					strings.Contains(lower, "electricity_price") || strings.Contains(lower, "energy_price") ||
					strings.Contains(lower, "unit_rate") || strings.Contains(lower, "current_rate") ||
					strings.Contains(lower, "current_price") || strings.Contains(lower, "net_price") ||
					strings.Contains(lower, "current_hour_price") || strings.Contains(lower, "consumption_price") ||
					strings.Contains(lower, "energidataservice")
				if !isTariff {
					continue
				}
				name := entityID
				var hasForecast bool
				var forecastHours int
				if attrs != nil {
					if fn, ok := attrs["friendly_name"].(string); ok {
						name = fn
					}
					// Check for forecast attribute (day-ahead prices)
					if forecast, ok := attrs["forecast"].([]any); ok && len(forecast) > 0 {
						hasForecast = true
						now := time.Now()
						for _, f := range forecast {
							fm, ok := f.(map[string]any)
							if !ok {
								forecastHours = len(forecast)
								break
							}
							dt, _ := fm["datetime"].(string)
							if dt == "" {
								forecastHours = len(forecast)
								break
							}
							if ts, err := time.Parse(time.RFC3339Nano, dt); err == nil && ts.After(now) {
								forecastHours++
							}
						}
					}
				}
				si := sensorInfo{
					EntityID:      entityID,
					Name:          name,
					State:         val,
					Unit:          unit,
					SupplierHint:  hasensor.SupplierHintFromEntity(entityID),
					HasForecast:   hasForecast,
					ForecastHours: forecastHours,
				}
				tariffSensors = append(tariffSensors, si)
			}
		}
		// Use the already-detected best sensor if available
		if s.detectedTariffSensor != "" {
			for i := range tariffSensors {
				if tariffSensors[i].EntityID == s.detectedTariffSensor {
					bestSensor = &tariffSensors[i]
					break
				}
			}
		}
		// Fallback: prefer sensor with day-ahead forecast, then first
		if bestSensor == nil && len(tariffSensors) > 0 {
			for i := range tariffSensors {
				if tariffSensors[i].HasForecast {
					bestSensor = &tariffSensors[i]
					break
				}
			}
			if bestSensor == nil {
				bestSensor = &tariffSensors[0]
			}
		}
	}

	// Get current wholesale price for markup auto-calculation
	var wholesaleKWh float64
	if data := s.sensorData.Get(); data != nil {
		// If we have consumer prices with wholesale data, use the stored wholesale
		if len(data.TodayPrices) > 0 {
			for _, p := range data.TodayPrices {
				if p.WholesaleKWh > 0 {
					wholesaleKWh = p.WholesaleKWh
					break
				}
			}
			// Fallback: use current price as wholesale if no separate wholesale
			if wholesaleKWh == 0 && data.CurrentPrice > 0 {
				wholesaleKWh = data.CurrentPrice
			}
		}
	}

	// Fetch Care-harvested profile for pre-fill (best-effort)
	harvestedProfile := fetchHarvestedProfile(r.Context(), s.installUUID)

	// Use harvested zone for supplier lookup if current config has no zone
	supplierZone := s.cfg.BiddingZone
	if supplierZone == "" && harvestedProfile != nil && harvestedProfile.Zone != "" {
		supplierZone = harvestedProfile.Zone
	}

	// Build markup parameter for supplier suggestions
	var markupParam string
	if harvestedProfile != nil && harvestedProfile.SupplierMarkup != nil {
		markupParam = strconv.FormatFloat(*harvestedProfile.SupplierMarkup, 'f', 6, 64)
	}

	// Fetch approved crowdsource suppliers from energy-data Worker (best-effort)
	crowdsourceSuppliers := fetchCrowdsourceSuppliers(r.Context(), supplierZone, markupParam)

	resp := map[string]any{
		"countries":     countries,
		"detected_zone": detectedZone,
		"detect_method": detectMethod,
		"zone_mismatch": zoneMismatch,
		"ha_country":    haCountry,
		"current_config": map[string]any{
			"zone":                 s.cfg.BiddingZone,
			"pricing_mode":         s.cfg.PricingMode,
			"supplier_id":          s.cfg.SupplierID,
			"onboarding_completed": s.cfg.OnboardingCompleted,
		},
		"tariff_sensors":        tariffSensors,
		"wholesale_kwh":         wholesaleKWh,
		"crowdsource_suppliers": crowdsourceSuppliers,
	}
	if bestSensor != nil {
		resp["detected_sensor"] = bestSensor
	}
	if harvestedProfile != nil {
		resp["harvested_profile"] = harvestedProfile
	}
	if detectedCountryName != "" {
		resp["detected_country_name"] = detectedCountryName
	}
	if len(locationCorrections) > 0 {
		resp["location_corrections"] = locationCorrections
	}
	writeJSON(w, resp)
}

// handleCrowdsourceSubmit proxies tax verification data to the energy-data Worker.
func (s *Server) handleCrowdsourceSubmit(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024) // 256KB max

	if s.installUUID == "" {
		writeError(w, http.StatusServiceUnavailable, "install UUID not available")
		return
	}

	var incoming map[string]any
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Add install_uuid to the submission
	incoming["install_uuid"] = s.installUUID

	jsonData, err := json.Marshal(incoming)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build request")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", energyDataBaseURL+"/api/v1/energy/submit-price", bytes.NewReader(jsonData))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SynctaclesEnergy/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to contact server")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("failed to read response body", "error", err)
	}

	if resp.StatusCode >= 400 {
		writeError(w, http.StatusBadGateway, "server returned error: "+string(body))
		return
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		writeError(w, http.StatusInternalServerError, "invalid response from server")
		return
	}

	writeJSON(w, result)
}

// handleZoneRequest proxies unsupported-region crowdsource submissions to the Worker.
func (s *Server) handleZoneRequest(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024) // 256KB max

	if s.installUUID == "" {
		writeError(w, http.StatusServiceUnavailable, "install UUID not available")
		return
	}

	var incoming map[string]any
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Add install_uuid
	incoming["install_uuid"] = s.installUUID

	// Use HA country code as the country field (the form sends free-text, Worker expects ISO code)
	if s.supervisor != nil {
		if haConfig, err := s.supervisor.GetConfig(r.Context()); err == nil {
			if haCountry, ok := haConfig["country"].(string); ok && haCountry != "" {
				incoming["country"] = haCountry
			}
			if tz, ok := haConfig["time_zone"].(string); ok && tz != "" {
				incoming["timezone"] = tz
			}
		}
	}

	// Auto-detect HA energy integrations (best-effort)
	if s.supervisor != nil {
		var integrations []string
		if addons, err := s.supervisor.ListAddons(r.Context()); err == nil {
			for _, a := range addons {
				name := strings.ToLower(a.Name)
				if strings.Contains(name, "energy") || strings.Contains(name, "solar") ||
					strings.Contains(name, "power") || strings.Contains(name, "grid") {
					integrations = append(integrations, a.Name)
				}
			}
		}
		if len(integrations) > 0 {
			incoming["ha_integrations"] = integrations
		}
	}

	jsonData, err := json.Marshal(incoming)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build request")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", energyDataBaseURL+"/api/v1/energy/zone-request", bytes.NewReader(jsonData))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SynctaclesEnergy/1.0")
	platform.SignRequest(req, jsonData)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to contact server")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("failed to read response body", "error", err)
	}
	if resp.StatusCode >= 400 {
		slog.Warn("zone-request proxy error", "status", resp.StatusCode, "body", string(body))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// fetchCrowdsourceSuppliers fetches approved crowdsource suppliers for a zone
// from the energy-data Worker. If markup is non-empty, includes it for suggestion matching.
// Returns nil on any error (best-effort).
func fetchCrowdsourceSuppliers(ctx context.Context, zone, markup string) map[string]any {
	if zone == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	u := energyDataBaseURL + "/api/v1/energy/suppliers?zone=" + zone
	if markup != "" {
		u += "&markup=" + markup
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "SynctaclesEnergy/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	return result
}
