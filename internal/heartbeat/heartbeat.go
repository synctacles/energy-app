// Package heartbeat sends minimal installation pings to the auth service.
// Unlike telemetry (opt-in, detailed), heartbeat is always active when
// the user has consented via the addon config option.
package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Endpoint is the auth service heartbeat URL.
const Endpoint = "https://api.synctacles.com/auth/heartbeat"

// Sender sends heartbeats to the auth service.
type Sender struct {
	installUUID  string
	product      string
	addonVersion string
	osArch       string
	onSuccess    func()
	onFailure    func()
}

// Config for creating a heartbeat sender.
type Config struct {
	InstallUUID  string
	Product      string // "energy" or "care"
	AddonVersion string
	OSArch       string
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
		"install_uuid":  s.installUUID,
		"product":       s.product,
		"addon_version": s.addonVersion,
		"os_arch":       s.osArch,
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("heartbeat: send failed", "error", err)
		if s.onFailure != nil {
			s.onFailure()
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("heartbeat sent", "product", s.product, "version", s.addonVersion)
		if s.onSuccess != nil {
			s.onSuccess()
		}
	} else {
		slog.Warn("heartbeat: unexpected status", "status", resp.StatusCode)
		if s.onFailure != nil {
			s.onFailure()
		}
	}
}
