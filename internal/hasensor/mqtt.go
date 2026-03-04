package hasensor

import (
	"context"
	"encoding/json"
	"fmt"
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

	slog.Info("MQTT connected", "broker", addr)
	return nil
}

// CleanupStaleTopics removes retained MQTT messages from legacy discovery topics.
// Call once after Connect to clear topics that used incorrect naming.
func (p *MQTTPublisher) CleanupStaleTopics() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Legacy topic used "binary_sensor.synctacles_cheap_hour" as object_id (with dot).
	// Fixed in rc34: now uses component=binary_sensor, object_id=cheap_hour.
	staleTopics := []string{
		"homeassistant/sensor/synctacles_energy/binary_sensor.synctacles_cheap_hour/config",
	}
	for _, topic := range staleTopics {
		if err := p.doPublish(topic, []byte{}, true); err != nil {
			slog.Debug("mqtt: failed to clear stale topic", "topic", topic, "error", err)
		} else {
			slog.Info("mqtt: cleared stale discovery topic", "topic", topic)
		}
	}
}

// Close disconnects from the broker.
func (p *MQTTPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
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
	packet = append(packet, 0x00, 0x00)  // Keep alive (0 = disabled)

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
	_, err := p.conn.Read(buf)
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
