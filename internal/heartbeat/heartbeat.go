// Package heartbeat sends minimal installation pings for anonymous install counting.
// Fire-and-forget — no feature gating depends on heartbeat success.
package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// Endpoint is the Cloudflare Worker heartbeat URL.
const Endpoint = "https://api.synctacles.com/api/v1/install/heartbeat"

// Sender sends heartbeats to the Synctacles API.
type Sender struct {
	installUUID  string
	product      string
	addonVersion string
	osArch       string
}

// Config for creating a heartbeat sender.
type Config struct {
	InstallUUID  string
	Product      string // "energy" or "care"
	AddonVersion string
	OSArch       string
	OnSuccess    func() // unused, kept for backward compatibility
	OnFailure    func() // unused, kept for backward compatibility
}

// NewSender creates a heartbeat sender.
func NewSender(cfg Config) *Sender {
	return &Sender{
		installUUID:  cfg.InstallUUID,
		product:      cfg.Product,
		addonVersion: cfg.AddonVersion,
		osArch:       cfg.OSArch,
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
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", Endpoint, bytes.NewReader(body))
	if err != nil {
		slog.Debug("heartbeat: create request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("heartbeat: send failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("heartbeat sent", "product", s.product, "version", s.addonVersion)
	} else {
		slog.Debug("heartbeat: unexpected status", "status", resp.StatusCode)
	}
}
