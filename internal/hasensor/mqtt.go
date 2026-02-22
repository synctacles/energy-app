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
// Includes automatic reconnection on broken pipe or connection loss.
type MQTTPublisher struct {
	host       string
	port       int
	clientID   string
	username   string
	password   string
	conn       net.Conn
	mu         sync.Mutex
	discovered map[string]bool // track which entities have been discovered
	connected  bool
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

// connectLocked performs the actual connection (must hold p.mu).
func (p *MQTTPublisher) connectLocked() error {
	// Close any existing connection
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}
	p.connected = false

	addr := fmt.Sprintf("%s:%d", p.host, p.port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("mqtt connect to %s: %w", addr, err)
	}
	p.conn = conn

	if err := p.sendConnect(); err != nil {
		conn.Close()
		p.conn = nil
		return fmt.Errorf("mqtt handshake: %w", err)
	}

	if err := p.readConnack(); err != nil {
		conn.Close()
		p.conn = nil
		return fmt.Errorf("mqtt connack: %w", err)
	}

	p.connected = true
	// Reset discovery state so sensors are re-registered after reconnect
	p.discovered = make(map[string]bool)
	slog.Info("MQTT connected", "broker", addr)
	return nil
}

// reconnect attempts to re-establish the MQTT connection (must hold p.mu).
func (p *MQTTPublisher) reconnect() error {
	slog.Warn("MQTT reconnecting...")
	if err := p.connectLocked(); err != nil {
		slog.Error("MQTT reconnect failed", "error", err)
		return err
	}
	slog.Info("MQTT reconnected successfully")
	return nil
}

// Close disconnects from the broker.
func (p *MQTTPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.connected = false
	if p.conn != nil {
		_, _ = p.conn.Write([]byte{0xE0, 0x00})
		p.conn.Close()
		p.conn = nil
	}
}

// UpdateSensor publishes sensor state to MQTT.
// On first call per entity, sends auto-discovery config.
// Automatically reconnects on broken pipe or connection loss.
func (p *MQTTPublisher) UpdateSensor(ctx context.Context, entityID, state string, attrs map[string]any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil || !p.connected {
		if err := p.reconnect(); err != nil {
			return fmt.Errorf("mqtt not connected: %w", err)
		}
	}

	objectID := strings.TrimPrefix(entityID, "sensor.synctacles_")

	if !p.discovered[entityID] {
		if err := p.publishDiscovery(objectID, entityID, attrs); err != nil {
			// Try reconnect once on discovery failure
			if reconnErr := p.reconnect(); reconnErr != nil {
				return fmt.Errorf("mqtt discovery for %s: %w", entityID, err)
			}
			if err := p.publishDiscovery(objectID, entityID, attrs); err != nil {
				return fmt.Errorf("mqtt discovery for %s after reconnect: %w", entityID, err)
			}
		}
		p.discovered[entityID] = true
	}

	stateTopic := fmt.Sprintf("synctacles/energy/%s/state", objectID)
	payload := map[string]any{
		"state":      state,
		"attributes": attrs,
	}
	data, _ := json.Marshal(payload)

	if err := p.publish(stateTopic, data, true); err != nil {
		// Broken pipe or write error — reconnect and retry once
		slog.Warn("MQTT publish failed, reconnecting", "error", err, "topic", stateTopic)
		if reconnErr := p.reconnect(); reconnErr != nil {
			return fmt.Errorf("mqtt publish %s: %w (reconnect also failed: %v)", stateTopic, err, reconnErr)
		}
		// Re-send discovery after reconnect
		if err := p.publishDiscovery(objectID, entityID, attrs); err != nil {
			return fmt.Errorf("mqtt re-discovery after reconnect: %w", err)
		}
		p.discovered[entityID] = true
		return p.publish(stateTopic, data, true)
	}
	return nil
}

// publishDiscovery sends an HA MQTT discovery config message.
func (p *MQTTPublisher) publishDiscovery(objectID, entityID string, attrs map[string]any) error {
	discoveryTopic := fmt.Sprintf("homeassistant/sensor/synctacles_energy/%s/config", objectID)

	friendlyName, _ := attrs["friendly_name"].(string)
	icon, _ := attrs["icon"].(string)
	unit, _ := attrs["unit_of_measurement"].(string)
	deviceClass, _ := attrs["device_class"].(string)
	stateClass, _ := attrs["state_class"].(string)

	config := map[string]any{
		"name":                friendlyName,
		"unique_id":          fmt.Sprintf("synctacles_energy_%s", objectID),
		"object_id":          fmt.Sprintf("synctacles_%s", objectID),
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

	data, _ := json.Marshal(config)
	return p.publish(discoveryTopic, data, true)
}

// publish sends an MQTT PUBLISH packet (QoS 0).
func (p *MQTTPublisher) publish(topic string, payload []byte, retain bool) error {
	// MQTT PUBLISH packet (QoS 0)
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
	packet = append(packet, 0x00, 0x3C)  // Keep alive (60s)

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
