package hasensor

import (
	"context"
	"log/slog"
	"net"
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
