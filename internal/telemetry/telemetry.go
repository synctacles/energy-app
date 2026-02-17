// Package telemetry sends periodic installation telemetry to the auth service.
// All fields are optional; the auth service accepts partial payloads.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
)

const (
	defaultBaseURL = "https://api.synctacles.com"
	sendInterval   = 24 * time.Hour
	uuidFile       = ".synctacles_uuid.json"
)

// payload matches the auth service TelemetryRequest.
type payload struct {
	InstallUUID    string         `json:"install_uuid"`
	Product        string         `json:"product"`
	Country        string         `json:"country,omitempty"`
	HAVersion      string         `json:"ha_version,omitempty"`
	EntityCount    int            `json:"entity_count,omitempty"`
	AddonVersion   string         `json:"addon_version,omitempty"`
	EnergyProvider string         `json:"energy_provider,omitempty"`

	// Extended fields (all optional)
	OSArch            string         `json:"os_arch,omitempty"`
	HAInstallType     string         `json:"ha_install_type,omitempty"`
	SupervisorChannel string         `json:"supervisor_channel,omitempty"`
	Locale            string         `json:"locale,omitempty"`
	ActiveSource      string         `json:"active_source,omitempty"`
	UptimeBucket      string         `json:"uptime_bucket,omitempty"`
	FallbackBucket    string         `json:"fallback_bucket,omitempty"`
	CacheHitBucket    string         `json:"cache_hit_bucket,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

// Deps holds all dependencies that the telemetry sender reads from.
// All fields are optional; nil/zero values are silently skipped.
type Deps struct {
	DataPath     string // persistent storage path for install UUID
	Version      string // addon version
	Zone         string // bidding zone (e.g. "NL")

	// Functions that return live data. Each may be nil.
	GetCoreInfo       func(ctx context.Context) (arch, haVersion, machine string, err error)
	GetSupervisorInfo func(ctx context.Context) (channel string, err error)
	GetHostOS         func(ctx context.Context) (operatingSystem string, err error)
	GetLocale         func(ctx context.Context) (locale string, err error)
	GetActiveSource   func() string
	GetSensorCount    func() int
	CheapestHoursOn   func() bool
	HasSupervisor     bool // true when running inside HA with Supervisor access
}

// Sender sends telemetry to the auth service once per interval.
type Sender struct {
	deps      Deps
	baseURL   string
	client    *http.Client
	startTime time.Time
	uuid      string
}

// NewSender creates a telemetry sender.
func NewSender(deps Deps) *Sender {
	baseURL := defaultBaseURL
	if v := os.Getenv("SYNCTACLES_AUTH_URL"); v != "" {
		baseURL = v
	}
	s := &Sender{
		deps:      deps,
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 15 * time.Second},
		startTime: time.Now(),
	}
	s.uuid = s.loadOrCreateUUID()
	return s
}

// RunBackground starts a goroutine that sends telemetry once daily.
// The first send happens 2 minutes after startup (to let data settle).
func (s *Sender) RunBackground(ctx context.Context) {
	go func() {
		// Initial delay to let sensors and sources initialize
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Minute):
		}

		s.sendOnce(ctx)

		ticker := time.NewTicker(sendInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.sendOnce(ctx)
			}
		}
	}()
}

// sendOnce collects and sends a single telemetry payload.
func (s *Sender) sendOnce(ctx context.Context) {
	p := payload{
		InstallUUID:  s.uuid,
		Product:      "energy",
		Country:      s.deps.Zone,
		AddonVersion: s.deps.Version,
		OSArch:       mapGoArch(runtime.GOARCH),
		UptimeBucket: uptimeBucket(time.Since(s.startTime)),
	}

	// Core info (HA version, arch from Supervisor)
	if s.deps.GetCoreInfo != nil {
		if arch, haVer, _, err := s.deps.GetCoreInfo(ctx); err == nil {
			p.HAVersion = haVer
			// Prefer Supervisor-reported arch over runtime.GOARCH
			if arch != "" {
				p.OSArch = arch
			}
		}
	}

	// Supervisor channel
	if s.deps.GetSupervisorInfo != nil {
		if channel, err := s.deps.GetSupervisorInfo(ctx); err == nil {
			p.SupervisorChannel = channel
		}
	}

	// HA install type (derived from host OS + supervisor presence)
	if s.deps.GetHostOS != nil {
		if osStr, err := s.deps.GetHostOS(ctx); err == nil {
			p.HAInstallType = FormatInstallType(osStr, s.deps.HasSupervisor)
		}
	} else if s.deps.HasSupervisor {
		p.HAInstallType = "supervised" // supervisor present but no host info
	}

	// Locale
	if s.deps.GetLocale != nil {
		if locale, err := s.deps.GetLocale(ctx); err == nil {
			p.Locale = locale
		}
	}

	// Active price source
	if s.deps.GetActiveSource != nil {
		p.ActiveSource = s.deps.GetActiveSource()
	}

	// Metadata
	meta := make(map[string]any)
	if s.deps.GetSensorCount != nil {
		meta["sensor_count"] = s.deps.GetSensorCount()
	}
	if s.deps.CheapestHoursOn != nil {
		meta["cheapest_hours_enabled"] = s.deps.CheapestHoursOn()
	}
	if len(meta) > 0 {
		p.Metadata = meta
	}

	// Send
	body, err := json.Marshal(p)
	if err != nil {
		slog.Debug("telemetry marshal failed", "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/auth/telemetry", bytes.NewReader(body))
	if err != nil {
		slog.Debug("telemetry request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		slog.Debug("telemetry send failed", "error", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Debug("telemetry sent", "uuid", s.uuid)
	} else {
		slog.Debug("telemetry rejected", "status", resp.StatusCode)
	}
}

// --- UUID persistence ---

type storedUUID struct {
	UUID      string `json:"uuid"`
	CreatedAt string `json:"created_at"`
}

func (s *Sender) loadOrCreateUUID() string {
	path := filepath.Join(s.deps.DataPath, uuidFile)

	data, err := os.ReadFile(path)
	if err == nil {
		var stored storedUUID
		if json.Unmarshal(data, &stored) == nil && stored.UUID != "" {
			return stored.UUID
		}
	}

	id := uuid.New().String()
	stored := storedUUID{
		UUID:      id,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if out, err := json.MarshalIndent(stored, "", "  "); err == nil {
		_ = os.WriteFile(path, out, 0600)
	}
	return id
}

// --- Bucket helpers ---

// uptimeBucket maps process uptime to a bucket string.
func uptimeBucket(d time.Duration) string {
	switch {
	case d < 24*time.Hour:
		return "<1d"
	case d < 7*24*time.Hour:
		return "1-7d"
	case d < 30*24*time.Hour:
		return "7-30d"
	default:
		return ">30d"
	}
}

// mapGoArch maps runtime.GOARCH values to the arch strings the auth service expects.
func mapGoArch(goarch string) string {
	switch goarch {
	case "arm64":
		return "aarch64"
	case "amd64":
		return "x86_64"
	case "arm":
		return "armv7"
	case "386":
		return "i386"
	default:
		return goarch
	}
}

// FallbackBucket maps a fallback count to a bucket string.
func FallbackBucket(count int) string {
	switch {
	case count == 0:
		return "0"
	case count <= 5:
		return "1-5"
	default:
		return "5+"
	}
}

// CacheHitBucket maps a cache hit ratio (0.0 - 1.0) to a bucket string.
func CacheHitBucket(ratio float64) string {
	switch {
	case ratio < 0.50:
		return "<50%"
	case ratio <= 0.90:
		return "50-90%"
	default:
		return ">90%"
	}
}

// InstallUUID returns the persistent install UUID (useful for other components).
func (s *Sender) InstallUUID() string {
	return s.uuid
}

// LoadInstallUUID reads or creates a persistent install UUID.
// Exported for use by other components (heartbeat, feature gate) without
// needing to create a full Sender.
func LoadInstallUUID(dataPath string) string {
	path := filepath.Join(dataPath, uuidFile)

	data, err := os.ReadFile(path)
	if err == nil {
		var stored storedUUID
		if json.Unmarshal(data, &stored) == nil && stored.UUID != "" {
			return stored.UUID
		}
	}

	id := uuid.New().String()
	stored := storedUUID{
		UUID:      id,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if out, err := json.MarshalIndent(stored, "", "  "); err == nil {
		_ = os.WriteFile(path, out, 0600)
	}
	return id
}

// OSArch returns the mapped architecture string for the current platform.
func OSArch() string {
	return mapGoArch(runtime.GOARCH)
}

// --- HA install type helpers ---

// InstallTypeFromOS tries to derive the HA installation type from the host OS string.
// The Supervisor /host/info returns operating_system like "Home Assistant OS 14.0".
func InstallTypeFromOS(operatingSystem string) string {
	switch {
	case len(operatingSystem) == 0:
		return ""
	case contains(operatingSystem, "Home Assistant OS"):
		return "os"
	default:
		// If supervisor is present but it's not HAOS, it's "supervised" or "container".
		// We can't distinguish further without more info, so return empty.
		return ""
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// FormatInstallType returns the HA install type based on available information.
// supervisorPresent indicates if SUPERVISOR_TOKEN is set (i.e., running as addon).
func FormatInstallType(operatingSystem string, supervisorPresent bool) string {
	if !supervisorPresent {
		return "core"
	}
	if t := InstallTypeFromOS(operatingSystem); t != "" {
		return t
	}
	return "supervised"
}
