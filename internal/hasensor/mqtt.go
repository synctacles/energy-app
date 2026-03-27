package hasensor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

// MQTTPublisher publishes sensor state via MQTT with HA auto-discovery.
// When MQTT discovery messages are published, HA automatically groups sensors
// under a "Synctacles Energy" device in the UI.
type MQTTPublisher struct {
	host       string
	port       int
	clientID   string
	username   string
	password   string
	conn       net.Conn
	mu         sync.Mutex
	discovered map[string]bool // track which entities have been discovered
	stopPing   chan struct{}   // signals the keepalive goroutine to stop
}

// NewMQTTPublisher creates an MQTT publisher.
// Uses raw TCP + minimal MQTT protocol — no external dependency needed.
func NewMQTTPublisher(host string, port int, username, password string) *MQTTPublisher {
	return &MQTTPublisher{
		host:       host,
		port:       port,
		clientID:   "synctacles-energy",
		username:   username,
		password:   password,
		discovered: make(map[string]bool),
	}
}

// Connect establishes the TCP connection and sends MQTT CONNECT.
func (p *MQTTPublisher) Connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connectLocked()
}

// connectLocked dials and performs the MQTT handshake. Must be called with p.mu held.
func (p *MQTTPublisher) connectLocked() error {
	// Stop any existing keepalive goroutine
	if p.stopPing != nil {
		close(p.stopPing)
		p.stopPing = nil
	}
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}

	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("mqtt connect to %s: %w", addr, err)
	}
	p.conn = conn

	// Send MQTT CONNECT packet
	if err := p.sendConnect(); err != nil {
		conn.Close()
		p.conn = nil
		return fmt.Errorf("mqtt handshake: %w", err)
	}

	// Read CONNACK
	if err := p.readConnack(); err != nil {
		conn.Close()
		p.conn = nil
		return fmt.Errorf("mqtt connack: %w", err)
	}

	// Start keepalive ping goroutine (sends PINGREQ every 30s for 60s keepalive)
	p.stopPing = make(chan struct{})
	go p.keepAlive(p.stopPing, p.conn)

	slog.Info("MQTT connected", "broker", addr)
	return nil
}

// keepAlive sends MQTT PINGREQ packets at half the keepalive interval.
// Stops when the stop channel is closed or a write fails.
func (p *MQTTPublisher) keepAlive(stop chan struct{}, conn net.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, err := conn.Write([]byte{0xC0, 0x00}); err != nil {
				slog.Debug("mqtt: keepalive ping failed", "error", err)
				return
			}
		}
	}
}

// CleanupStaleTopics removes stale retained MQTT discovery messages from
// legacy topic names. Runs exactly ONCE per installation using a marker file
// in /data/ (persistent HA app storage). The one-time empty retained publish
// clears the broker's retained message permanently.
func (p *MQTTPublisher) CleanupStaleTopics() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return
	}

	// Clear any retained discovery topics from a previous session that may
	// not have been cleaned up (e.g. SIGKILL during shutdown, uninstall
	// without graceful stop). This runs on every startup to guarantee no
	// orphaned ghost sensors survive across restarts.
	allTopics := []string{
		// Legacy (rc34)
		"homeassistant/sensor/synctacles_energy/binary_sensor.synctacles_cheap_hour/config",
		// Current entities — discovery + state
		"homeassistant/sensor/synctacles_energy/current_price/config",
		"homeassistant/sensor/synctacles_energy/next_hour_price/config",
		"homeassistant/sensor/synctacles_energy/today_average/config",
		"homeassistant/sensor/synctacles_energy/today_min/config",
		"homeassistant/sensor/synctacles_energy/today_max/config",
		"homeassistant/sensor/synctacles_energy/tomorrow_average/config",
		"homeassistant/sensor/synctacles_energy/price_ratio/config",
		"homeassistant/sensor/synctacles_energy/hours_until_go/config",
		"homeassistant/sensor/synctacles_energy/best_start/config",
		"homeassistant/sensor/synctacles_energy/price_trend/config",
		"homeassistant/sensor/synctacles_energy/renewable_share/config",
		"homeassistant/binary_sensor/synctacles_energy/green_energy/config",
		"homeassistant/binary_sensor/synctacles_energy/green_window/config",
		"homeassistant/binary_sensor/synctacles_energy/cheap_hour/config",
		"homeassistant/binary_sensor/synctacles_energy/price_alert/config",
	}

	cleared := 0
	for _, topic := range allTopics {
		if err := p.doPublish(topic, []byte{}, true); err != nil {
			slog.Debug("mqtt: failed to clear stale topic", "topic", topic, "error", err)
			continue
		}
		cleared++
	}

	if cleared > 0 {
		slog.Info("mqtt: cleared stale discovery topics on startup", "count", cleared)
	}
}

// RemoveAllDiscovery publishes empty retained messages to all previously
// discovered entity topics, removing them from HA. Call before Close()
// during app uninstall/shutdown to prevent orphaned ghost sensors.
func (p *MQTTPublisher) RemoveAllDiscovery() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil || len(p.discovered) == 0 {
		return
	}

	cleared := 0
	for entityID := range p.discovered {
		component := "sensor"
		objectID := strings.TrimPrefix(entityID, "sensor.synctacles_")
		if strings.HasPrefix(entityID, "binary_sensor.") {
			component = "binary_sensor"
			objectID = strings.TrimPrefix(entityID, "binary_sensor.synctacles_")
		}

		// Clear discovery config (empty retained = HA removes the entity)
		discoveryTopic := fmt.Sprintf("homeassistant/%s/synctacles_energy/%s/config", component, objectID)
		if err := p.doPublish(discoveryTopic, []byte{}, true); err != nil {
			slog.Debug("mqtt: failed to clear discovery", "topic", discoveryTopic, "error", err)
			continue
		}

		// Clear state
		stateTopic := fmt.Sprintf("synctacles/energy/%s/state", objectID)
		_ = p.doPublish(stateTopic, []byte{}, true)

		cleared++
	}

	slog.Info("mqtt: removed all discovery topics", "count", cleared)
}

// Close disconnects from the broker.
func (p *MQTTPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopPing != nil {
		close(p.stopPing)
		p.stopPing = nil
	}
	if p.conn != nil {
		// Send DISCONNECT packet
		_, _ = p.conn.Write([]byte{0xE0, 0x00})
		p.conn.Close()
		p.conn = nil
	}
}

// UpdateSensor publishes sensor state to MQTT.
// On first call per entity, sends auto-discovery config.
func (p *MQTTPublisher) UpdateSensor(ctx context.Context, entityID, state string, attrs map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return fmt.Errorf("mqtt not connected")
	}

	// Entity ID format: sensor.synctacles_energy_price → object_id = energy_price
	// binary_sensor.synctacles_cheap_hour → component = binary_sensor, object_id = cheap_hour
	component := "sensor"
	objectID := strings.TrimPrefix(entityID, "sensor.synctacles_")
	if strings.HasPrefix(entityID, "binary_sensor.") {
		component = "binary_sensor"
		objectID = strings.TrimPrefix(entityID, "binary_sensor.synctacles_")
	}

	// Send discovery config if not yet done
	if !p.discovered[entityID] {
		if err := p.publishDiscovery(component, objectID, entityID, attrs); err != nil {
			return fmt.Errorf("mqtt discovery for %s: %w", entityID, err)
		}
		p.discovered[entityID] = true
	}

	// Publish state
	stateTopic := fmt.Sprintf("synctacles/energy/%s/state", objectID)
	payload := map[string]any{
		"state":      state,
		"attributes": attrs,
	}
	data, _ := json.Marshal(payload)
	return p.publish(stateTopic, data, true)
}

// publishDiscovery sends an HA MQTT discovery config message.
func (p *MQTTPublisher) publishDiscovery(component, objectID, entityID string, attrs map[string]any) error {
	discoveryTopic := fmt.Sprintf("homeassistant/%s/synctacles_energy/%s/config", component, objectID)

	friendlyName, _ := attrs["friendly_name"].(string)
	icon, _ := attrs["icon"].(string)
	unit, _ := attrs["unit_of_measurement"].(string)
	deviceClass, _ := attrs["device_class"].(string)
	stateClass, _ := attrs["state_class"].(string)

	config := map[string]any{
		"name":                friendlyName,
		"unique_id":          fmt.Sprintf("synctacles_energy_%s", objectID),
		"default_entity_id":  fmt.Sprintf("synctacles_%s", objectID),
		"state_topic":        fmt.Sprintf("synctacles/energy/%s/state", objectID),
		"value_template":     "{{ value_json.state }}",
		"json_attributes_topic": fmt.Sprintf("synctacles/energy/%s/state", objectID),
		"json_attributes_template": "{{ value_json.attributes | tojson }}",
		"device": map[string]any{
			"identifiers":  []string{"synctacles_energy"},
			"name":         "Synctacles Energy",
			"manufacturer": "Synctacles",
			"model":        "Energy App",
		},
	}
	if icon != "" {
		config["icon"] = icon
	}
	if unit != "" {
		config["unit_of_measurement"] = unit
	}
	if deviceClass != "" {
		config["device_class"] = deviceClass
	}
	if stateClass != "" {
		config["state_class"] = stateClass
	}
	if component == "binary_sensor" {
		config["payload_on"] = "on"
		config["payload_off"] = "off"
	}

	data, _ := json.Marshal(config)
	return p.publish(discoveryTopic, data, true)
}

// publish sends an MQTT PUBLISH packet (QoS 0).
// On write failure it reconnects once and retries.
func (p *MQTTPublisher) publish(topic string, payload []byte, retain bool) error {
	if err := p.doPublish(topic, payload, retain); err == nil {
		return nil
	}
	// Connection lost — reconnect and retry once.
	slog.Warn("MQTT publish failed, reconnecting...", "topic", topic)
	if err := p.connectLocked(); err != nil {
		return fmt.Errorf("mqtt reconnect: %w", err)
	}
	// Reset discovery state so all entities are re-announced after reconnect.
	p.discovered = make(map[string]bool)
	return p.doPublish(topic, payload, retain)
}

// doPublish writes a single PUBLISH packet to the current connection.
func (p *MQTTPublisher) doPublish(topic string, payload []byte, retain bool) error {
	if p.conn == nil {
		return fmt.Errorf("mqtt not connected")
	}

	var flags byte = 0x30 // PUBLISH, QoS 0
	if retain {
		flags |= 0x01
	}

	topicBytes := []byte(topic)
	remainingLen := 2 + len(topicBytes) + len(payload)

	var packet []byte
	packet = append(packet, flags)
	packet = append(packet, encodeRemainingLength(remainingLen)...)
	packet = append(packet, byte(len(topicBytes)>>8), byte(len(topicBytes)))
	packet = append(packet, topicBytes...)
	packet = append(packet, payload...)

	_ = p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := p.conn.Write(packet)
	return err
}

// sendConnect sends MQTT CONNECT packet with optional authentication.
func (p *MQTTPublisher) sendConnect() error {
	clientIDBytes := []byte(p.clientID)
	usernameBytes := []byte(p.username)
	passwordBytes := []byte(p.password)

	// Connect flags: Clean Session (0x02)
	var connectFlags byte = 0x02
	remainingLen := 10 + 2 + len(clientIDBytes)

	if p.username != "" {
		connectFlags |= 0x80 // Username flag
		remainingLen += 2 + len(usernameBytes)
	}
	if p.password != "" {
		connectFlags |= 0x40 // Password flag
		remainingLen += 2 + len(passwordBytes)
	}

	var packet []byte
	packet = append(packet, 0x10) // CONNECT
	packet = append(packet, encodeRemainingLength(remainingLen)...)

	// Variable header
	packet = append(packet, 0x00, 0x04) // Protocol name length
	packet = append(packet, []byte("MQTT")...)
	packet = append(packet, 0x04)         // Protocol level (4 = MQTT 3.1.1)
	packet = append(packet, connectFlags) // Connect flags
	packet = append(packet, 0x00, 0x3C)  // Keep alive (60 seconds)

	// Payload: client ID
	packet = append(packet, byte(len(clientIDBytes)>>8), byte(len(clientIDBytes)))
	packet = append(packet, clientIDBytes...)

	// Payload: username (if set)
	if p.username != "" {
		packet = append(packet, byte(len(usernameBytes)>>8), byte(len(usernameBytes)))
		packet = append(packet, usernameBytes...)
	}

	// Payload: password (if set)
	if p.password != "" {
		packet = append(packet, byte(len(passwordBytes)>>8), byte(len(passwordBytes)))
		packet = append(packet, passwordBytes...)
	}

	_ = p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := p.conn.Write(packet)
	return err
}

// readConnack reads and validates the CONNACK packet.
func (p *MQTTPublisher) readConnack() error {
	_ = p.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4)
	_, err := io.ReadFull(p.conn, buf)
	if err != nil {
		return err
	}
	if buf[0] != 0x20 { // CONNACK
		return fmt.Errorf("expected CONNACK, got 0x%02x", buf[0])
	}
	if buf[3] != 0x00 { // Return code 0 = accepted
		return fmt.Errorf("connection refused, code: %d", buf[3])
	}
	return nil
}

// SensorCount returns the number of MQTT sensors that have been discovered/published.
func (p *MQTTPublisher) SensorCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.discovered)
}

// encodeRemainingLength encodes an MQTT remaining length field.
func encodeRemainingLength(length int) []byte {
	var encoded []byte
	for {
		b := byte(length % 128)
		length /= 128
		if length > 0 {
			b |= 0x80
		}
		encoded = append(encoded, b)
		if length == 0 {
			break
		}
	}
	return encoded
}
