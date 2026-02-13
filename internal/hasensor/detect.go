package hasensor

import (
	"context"
	"log/slog"
	"strings"

	"github.com/synctacles/energy-app/internal/ha"
)

// mqttAddonSlugs are known MQTT broker addon slugs in HA.
var mqttAddonSlugs = []string{
	"core_mosquitto",       // Official Mosquitto addon
	"a0d7b954_mqtt",        // Community Mosquitto
	"45df7312_mqtt-broker", // Alternative broker
}

// DetectMQTTBroker checks if an MQTT broker addon is installed and running.
// Returns the broker host and port if found.
func DetectMQTTBroker(ctx context.Context, supervisor *ha.SupervisorClient) (host string, found bool) {
	if supervisor == nil {
		return "", false
	}

	addons, err := supervisor.ListAddons(ctx)
	if err != nil {
		slog.Debug("could not list addons for MQTT detection", "error", err)
		return "", false
	}

	for _, addon := range addons {
		for _, slug := range mqttAddonSlugs {
			if addon.Slug == slug && addon.State == "started" {
				slog.Info("MQTT broker detected", "addon", addon.Name, "slug", addon.Slug)
				return "core-mosquitto", true
			}
		}
		// Also check for any addon with "mqtt" in the slug that's running
		if strings.Contains(strings.ToLower(addon.Slug), "mosquitto") && addon.State == "started" {
			slog.Info("MQTT broker detected (fuzzy match)", "addon", addon.Name, "slug", addon.Slug)
			return "core-mosquitto", true
		}
	}

	return "", false
}
