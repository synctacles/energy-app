// Package heartbeat sends minimal installation pings for anonymous install counting.
// Fire-and-forget — no feature gating depends on heartbeat success.
//
// Uses conditional sending: checks every 6 hours but only sends if the payload
// changed or 24 hours have passed since the last successful send.
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

const (
	checkInterval = 6 * time.Hour  // How often to check if we should send
	forceInterval = 24 * time.Hour // Always send after this duration
)

// payload is the heartbeat JSON body.
type payload struct {
	InstallID string `json:"install_id"`
	Product   string `json:"product"`
	Version   string `json:"version"`
	Arch      string `json:"arch"`
}

// Sender sends heartbeats to the Synctacles API.
type Sender struct {
	installUUID  string
	product      string
	addonVersion string
	osArch       string

	lastPayload payload
	lastSentAt  time.Time
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

// Run sends a heartbeat immediately, then checks every 6 hours.
// Only sends if the payload changed or 24h have passed since the last send.
// Blocks until ctx is cancelled.
func (s *Sender) Run(ctx context.Context) {
	// Always send on startup
	s.send(ctx)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := s.currentPayload()
			if current != s.lastPayload || time.Since(s.lastSentAt) >= forceInterval {
				s.send(ctx)
			} else {
				slog.Debug("heartbeat: skipped, no changes")
			}
		}
	}
}

func (s *Sender) currentPayload() payload {
	return payload{
		InstallID: s.installUUID,
		Product:   s.product,
		Version:   s.addonVersion,
		Arch:      s.osArch,
	}
}

func (s *Sender) send(ctx context.Context) {
	p := s.currentPayload()

	body, _ := json.Marshal(p)
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
		s.lastPayload = p
		s.lastSentAt = time.Now()
		slog.Info("heartbeat sent", "product", s.product, "version", s.addonVersion)
	} else {
		slog.Debug("heartbeat: unexpected status", "status", resp.StatusCode)
	}
}
