// Package heartbeat sends minimal installation pings to the Synctacles API.
// Unlike telemetry (opt-in, detailed), heartbeat is always active.
package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/synctacles/energy-app/pkg/platform"
)

// Endpoint is the Synctacles API heartbeat URL.
const Endpoint = "https://api.synctacles.com/api/v1/install/heartbeat"

// Sender sends heartbeats to the Synctacles API.
type Sender struct {
	installUUID  string
	product      string
	addonVersion string
	osArch       string
	haVersion    string
	onSuccess    func()
	onFailure    func()
}

// Config for creating a heartbeat sender.
type Config struct {
	InstallUUID  string
	Product      string // "energy" or "care"
	AddonVersion string
	OSArch       string
	HAVersion    string // HA Core version (e.g. "2026.3.1")
	OnSuccess    func() // called after successful heartbeat
	OnFailure    func() // called after failed heartbeat
}

// NewSender creates a heartbeat sender.
func NewSender(cfg Config) *Sender {
	return &Sender{
		installUUID:  cfg.InstallUUID,
		product:      cfg.Product,
		addonVersion: cfg.AddonVersion,
		osArch:       cfg.OSArch,
		haVersion:    cfg.HAVersion,
		onSuccess:    cfg.OnSuccess,
		onFailure:    cfg.OnFailure,
	}
}

// Run sends a heartbeat immediately, then every 6 hours.
// Blocks until ctx is cancelled.
func (s *Sender) Run(ctx context.Context) {
	s.send(ctx)

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.send(ctx)
		}
	}
}

func (s *Sender) send(ctx context.Context) {
	payload := map[string]string{
		"install_id": s.installUUID,
		"product":    s.product,
		"version":    s.addonVersion,
		"arch":       s.osArch,
		"ha_version": s.haVersion,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", Endpoint, bytes.NewReader(body))
	if err != nil {
		slog.Warn("heartbeat: create request failed", "error", err)
		if s.onFailure != nil {
			s.onFailure()
		}
		return
	}
	req.Header.Set("Content-Type", "application/json")
	platform.SignRequest(req, body)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("heartbeat: send failed", "error", err)
		if s.onFailure != nil {
			s.onFailure()
		}
		return
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("heartbeat: unexpected status", "status", resp.StatusCode)
		if s.onFailure != nil {
			s.onFailure()
		}
		return
	}

	slog.Info("heartbeat sent", "product", s.product, "version", s.addonVersion)
	if s.onSuccess != nil {
		s.onSuccess()
	}
}
