package hasensor

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/synctacles/energy-app/internal/ha"
)

// MQTTCredentials holds broker connection details.
type MQTTCredentials struct {
	Host     string
	Port     int
	Username string
	Password string
}

// mqttBrokerHosts are Docker DNS names for known HA MQTT broker addons.
var mqttBrokerHosts = []string{
	"core-mosquitto",       // Official Mosquitto addon
	"a0d7b954-mqtt",        // Community Mosquitto
	"45df7312-mqtt-broker", // Alternative broker
}

// DetectMQTTBroker finds an MQTT broker and returns connection credentials.
// First tries the Supervisor services API (provides auth credentials).
// Falls back to TCP probing known broker hostnames.
func DetectMQTTBroker(ctx context.Context, supervisor *ha.SupervisorClient) (*MQTTCredentials, bool) {
	// Try Supervisor services API first (gives us auth credentials)
	if supervisor != nil {
		if creds, err := supervisor.GetMQTTService(ctx); err == nil {
			slog.Info("MQTT credentials from Supervisor", "host", creds.Host, "port", creds.Port)
			return &MQTTCredentials{
				Host:     creds.Host,
				Port:     creds.Port,
				Username: creds.Username,
				Password: creds.Password,
			}, true
		} else {
			slog.Debug("MQTT service not available from Supervisor", "error", err)
		}
	}

	// Fallback: TCP probe known broker hostnames (no credentials)
	for _, h := range mqttBrokerHosts {
		addr := h + ":1883"
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			slog.Debug("MQTT probe failed", "host", h, "error", err)
			continue
		}
		conn.Close()
		slog.Info("MQTT broker detected via TCP probe", "host", h)
		return &MQTTCredentials{Host: h, Port: 1883}, true
	}
	return nil, false
}

// DetectTariffSensor queries HA for a tariff/price sensor and returns the best match.
// Detects consumer tariff sensors from known integrations:
//   - UK: Glow CAD (unit_rate), Octopus Energy (current_rate)
//   - NL: P1 Monitor (consumption_price)
//   - NO/SE/DE: Tibber (electricity_price)
//   - EU: Nord Pool, EPEX Spot, easyEnergy, Energi Data Service
//
// Returns empty string if nothing suitable is found.
func DetectTariffSensor(ctx context.Context, supervisor *ha.SupervisorClient) string {
	states, err := supervisor.GetAllStates(ctx)
	if err != nil {
		slog.Warn("auto-detect tariff sensor: failed to get states", "error", err)
		return ""
	}
	return matchTariffSensor(states)
}

// matchTariffSensor finds the best tariff sensor from a list of HA entity states.
func matchTariffSensor(states []map[string]any) string {
	type candidate struct {
		entityID string
		priority int // lower = better
	}

	var candidates []candidate

	for _, s := range states {
		eid, _ := s["entity_id"].(string)
		if !strings.HasPrefix(eid, "sensor.") {
			continue
		}

		// Must have a price-like unit
		attrs, _ := s["attributes"].(map[string]any)
		unit := ""
		if attrs != nil {
			if u, ok := attrs["unit_of_measurement"].(string); ok {
				unit = u
			}
		}
		unitLower := strings.ToLower(unit)
		isPriceUnit := strings.Contains(unitLower, "/kwh") ||
			strings.Contains(unitLower, "/wh") ||
			unitLower == "eur" || unitLower == "gbp" ||
			unitLower == "nok" || unitLower == "sek" ||
			unitLower == "dkk" || unitLower == "czk" ||
			unitLower == "pln" || unitLower == "chf" ||
			unitLower == "huf"
		if unit != "" && !isPriceUnit {
			continue
		}

		// Must have a numeric state
		stateStr, _ := s["state"].(string)
		if _, err := strconv.ParseFloat(stateStr, 64); err != nil {
			continue
		}

		lower := strings.ToLower(eid)

		// Exclude non-tariff sensors
		if strings.Contains(lower, "gas") || strings.Contains(lower, "standing") ||
			strings.Contains(lower, "daily") || strings.Contains(lower, "monthly") ||
			strings.Contains(lower, "cost") || strings.Contains(lower, "yesterday") {
			continue
		}

		prio := 0

		// Priority 10: Direct consumer tariff from meter/provider
		switch {
		case strings.Contains(lower, "unit_rate") && !strings.Contains(lower, "gas"):
			prio = 10 // UK Glow CAD
		case strings.Contains(lower, "current_rate") && strings.Contains(lower, "octopus"):
			prio = 10 // UK Octopus Energy
		case strings.Contains(lower, "consumption_price"):
			prio = 10 // NL P1 Monitor

		// Priority 15: Provider with full consumer price
		case strings.Contains(lower, "electricity_price") && strings.Contains(lower, "tibber"):
			prio = 15 // Tibber (NO/SE/DE)

		// Priority 20: Generic tariff patterns
		case strings.Contains(lower, "tariff") || strings.Contains(lower, "tarief"):
			prio = 20
		case strings.Contains(lower, "energy_price"):
			prio = 20
		case strings.Contains(lower, "electricity_price"):
			prio = 20

		// Priority 25: Spot price integrations
		case strings.Contains(lower, "current_price") && strings.Contains(lower, "nord_pool"):
			prio = 25
		case strings.Contains(lower, "net_price") && strings.Contains(lower, "epex"):
			prio = 25
		case strings.Contains(lower, "energidataservice"):
			prio = 25
		case strings.Contains(lower, "current_hour_price"):
			prio = 25 // easyEnergy

		default:
			continue // Not a known tariff pattern
		}

		// Adjust priority based on last_changed — dynamic sensors update every 1-2h,
		// fixed-rate sensors rarely change. This helps deprioritize fixed tariffs.
		if lc, ok := s["last_changed"].(string); ok && lc != "" {
			if lastChanged, err := time.Parse(time.RFC3339, lc); err == nil {
				hoursSince := time.Since(lastChanged).Hours()
				if hoursSince > 24 {
					prio += 15 // likely fixed rate — deprioritize
				} else if hoursSince < 4 {
					prio -= 5 // recently changed — likely dynamic
				}
			}
		}

		candidates = append(candidates, candidate{eid, prio})
	}

	if len(candidates) == 0 {
		return ""
	}

	// Find best candidate
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.priority < best.priority {
			best = c
		}
	}

	slog.Debug("tariff sensor detection", "best", best.entityID, "priority", best.priority, "candidates", len(candidates))
	return best.entityID
}

// supplierPatterns maps entity ID substrings to supplier names.
var supplierPatterns = []struct {
	pattern  string
	supplier string
}{
	{"tibber", "tibber"},
	{"octopus", "octopus"},
	{"nord_pool", "nordpool"},
	{"epex", "epex"},
	{"energidataservice", "energidataservice"},
	{"easyenergy", "easyenergy"},
	{"energyzero", "energyzero"},
	{"zonneplan", "zonneplan"},
	{"frank_energie", "frank"},
	{"amber", "amber"},
	{"glow", "glow"},
	{"p1_monitor", "p1monitor"},
}

// SupplierHintFromEntity extracts a supplier name from a tariff sensor entity ID.
// Returns empty string if no known supplier pattern is found.
func SupplierHintFromEntity(entityID string) string {
	lower := strings.ToLower(entityID)
	for _, p := range supplierPatterns {
		if strings.Contains(lower, p.pattern) {
			return p.supplier
		}
	}
	return ""
}
