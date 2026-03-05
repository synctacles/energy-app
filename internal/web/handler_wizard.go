package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const energyDataBaseURL = "https://energy-data.synctacles.com"

// handleWizardData returns bundled data for the onboarding wizard:
// zones grouped by country, suppliers, tax defaults, current config, and detected zone.
func (s *Server) handleWizardData(w http.ResponseWriter, r *http.Request) {
	type zoneEntry struct {
		Code     string `json:"code"`
		Name     string `json:"name"`
		Timezone string `json:"timezone"`
	}
	type countryEntry struct {
		Code        string      `json:"code"`
		Name        string      `json:"name"`
		Currency    string      `json:"currency"`
		Zones       []zoneEntry `json:"zones"`
		Suppliers   []any       `json:"suppliers"`
		TaxDefaults any         `json:"tax_defaults"`
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
			entry = &countryEntry{
				Code:        cc.Country,
				Name:        cc.Name,
				Currency:    cc.Currency,
				Zones:       []zoneEntry{},
				Suppliers:   suppliers,
				TaxDefaults: taxDefaults,
			}
			countriesMap[z.Country] = entry
		}
		entry.Zones = append(entry.Zones, zoneEntry{
			Code:     z.Code,
			Name:     z.Name,
			Timezone: z.Timezone,
		})
	}

	// Sort countries by name
	countries := make([]*countryEntry, 0, len(countriesMap))
	for _, entry := range countriesMap {
		countries = append(countries, entry)
	}
	sort.Slice(countries, func(i, j int) bool {
		return countries[i].Name < countries[j].Name
	})

	// Detect zone from HA timezone
	var detectedZone string
	if s.supervisor != nil {
		haConfig, err := s.supervisor.GetConfig(r.Context())
		if err == nil {
			if tz, ok := haConfig["time_zone"].(string); ok && tz != "" {
				for _, code := range s.zoneRegistry.AllZones() {
					z, ok := s.zoneRegistry.GetZone(code)
					if ok && z.Timezone == tz {
						detectedZone = z.Code
						break
					}
				}
			}
		}
	}

	// Detect tariff sensors (Zonneplan, Tibber, Octopus, P1 Monitor, etc.)
	type sensorInfo struct {
		EntityID string  `json:"entity_id"`
		Name     string  `json:"name"`
		State    float64 `json:"state"`
		Unit     string  `json:"unit"`
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
				if attrs != nil {
					if fn, ok := attrs["friendly_name"].(string); ok {
						name = fn
					}
				}
				si := sensorInfo{EntityID: entityID, Name: name, State: val, Unit: unit}
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
		// Fallback: pick first sensor if no best detected
		if bestSensor == nil && len(tariffSensors) > 0 {
			bestSensor = &tariffSensors[0]
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

	// Fetch approved crowdsource suppliers from energy-data Worker (best-effort)
	crowdsourceSuppliers := fetchCrowdsourceSuppliers(r.Context(), s.cfg.BiddingZone)

	resp := map[string]any{
		"countries":     countries,
		"detected_zone": detectedZone,
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
	writeJSON(w, resp)
}

// handleCrowdsourceSubmit proxies tax verification data to the energy-data Worker.
func (s *Server) handleCrowdsourceSubmit(w http.ResponseWriter, r *http.Request) {
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

	body, _ := io.ReadAll(resp.Body)

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

// fetchCrowdsourceSuppliers fetches approved crowdsource suppliers for a zone
// from the energy-data Worker. Returns nil on any error (best-effort).
func fetchCrowdsourceSuppliers(ctx context.Context, zone string) []map[string]any {
	if zone == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET",
		energyDataBaseURL+"/api/v1/energy/suppliers?zone="+zone, nil)
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

	var result struct {
		Suppliers []map[string]any `json:"suppliers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}
	return result.Suppliers
}
