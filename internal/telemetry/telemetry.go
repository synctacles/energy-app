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
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/synctacles/energy-app/pkg/platform"
)

var defaultBaseURL = platform.APIBaseURL

const (
	sendInterval    = 24 * time.Hour
	uuidFile        = ".synctacles_uuid.json"          // legacy (per-app)
	sharedUUIDFile  = ".synctacles_install_id"          // shared across apps via /config
)

// payload matches the auth service TelemetryRequest.
type payload struct {
	InstallUUID string         `json:"install_uuid"`
	Product     string         `json:"product"`
	Country     string         `json:"country,omitempty"`
	HAVersion   string         `json:"ha_version,omitempty"`
	EntityCount int            `json:"entity_count,omitempty"`
	AddonCount  int            `json:"addon_count,omitempty"`
	AddonVersion string        `json:"addon_version,omitempty"`

	// Extended fields (all optional)
	OSArch            string         `json:"os_arch,omitempty"`
	HAInstallType     string         `json:"ha_install_type,omitempty"`
	SupervisorChannel string         `json:"supervisor_channel,omitempty"`
	Locale            string         `json:"locale,omitempty"`
	UptimeBucket      string         `json:"uptime_bucket,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

// Deps holds all dependencies that the telemetry sender reads from.
// All fields are optional; nil/zero values are silently skipped.
type Deps struct {
	DataPath     string // persistent storage path (legacy UUID location)
	ConfigPath   string // shared HA config path (shared UUID location)
	Version      string // addon version
	Zone         string // bidding zone (e.g. "NL")

	// Functions that return live data. Each may be nil.
	GetCoreInfo       func(ctx context.Context) (arch, haVersion, machine string, err error)
	GetSupervisorInfo func(ctx context.Context) (channel string, err error)
	GetHostOS         func(ctx context.Context) (operatingSystem string, err error)
	GetLocale         func(ctx context.Context) (locale string, err error)
	GetActiveSource   func() string
	GetSensorCount    func() int
	GetEntityCount    func(ctx context.Context) int // entity count from HA states
	GetAddonCount     func(ctx context.Context) int // addon count from Supervisor
	CheapestHoursOn   func() bool
	HasSupervisor     bool // true when running inside HA with Supervisor access

	// Observability: tax/price source tracking
	GetTaxSource      func() string // "worker", "embedded", "none"
	GetFallbackCount  func() int    // number of fallback events since startup
	GetCacheHitRatio  func() float64 // 0.0-1.0 cache hit ratio

	// Config snapshot for remote troubleshooting
	GetConfigSnapshot func() map[string]any // pricing_mode, supplier, thresholds, etc.
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
		s.sendSourceHealth(ctx)

		ticker := time.NewTicker(sendInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.sendOnce(ctx)
				s.sendSourceHealth(ctx)
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

	// Entity and addon counts
	if s.deps.GetEntityCount != nil {
		p.EntityCount = s.deps.GetEntityCount(ctx)
	}
	if s.deps.GetAddonCount != nil {
		p.AddonCount = s.deps.GetAddonCount(ctx)
	}

	// Metadata (energy-specific fields stored here to match Worker schema)
	meta := make(map[string]any)
	if s.deps.GetActiveSource != nil {
		meta["active_source"] = s.deps.GetActiveSource()
	}
	if s.deps.GetFallbackCount != nil {
		meta["fallback_bucket"] = FallbackBucket(s.deps.GetFallbackCount())
	}
	if s.deps.GetCacheHitRatio != nil {
		meta["cache_hit_bucket"] = CacheHitBucket(s.deps.GetCacheHitRatio())
	}
	if s.deps.GetSensorCount != nil {
		meta["sensor_count"] = s.deps.GetSensorCount()
	}
	if s.deps.CheapestHoursOn != nil {
		meta["cheapest_hours_enabled"] = s.deps.CheapestHoursOn()
	}
	if s.deps.GetTaxSource != nil {
		meta["tax_source"] = s.deps.GetTaxSource()
	}
	if s.deps.GetConfigSnapshot != nil {
		if snap := s.deps.GetConfigSnapshot(); snap != nil {
			for k, v := range snap {
				meta["cfg_"+k] = v
			}
		}
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/api/v1/telemetry", bytes.NewReader(body))
	if err != nil {
		slog.Debug("telemetry request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	platform.SignRequest(req, body)

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

// sendSourceHealth reports price source health to the platform API.
// Runs after each telemetry send to piggyback on the same schedule.
func (s *Sender) sendSourceHealth(ctx context.Context) {
	if s.deps.GetActiveSource == nil {
		return
	}

	report := map[string]any{
		"install_id":     s.uuid,
		"report_date":    time.Now().UTC().Format("2006-01-02"),
		"country":        s.deps.Zone,
		"active_source":  s.deps.GetActiveSource(),
		"fallback_events": 0,
	}
	if s.deps.GetFallbackCount != nil {
		report["fallback_events"] = s.deps.GetFallbackCount()
	}
	if s.deps.GetTaxSource != nil {
		report["sources"] = []map[string]string{
			{"name": "tax", "status": s.deps.GetTaxSource()},
		}
	}

	body, err := json.Marshal(report)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/api/v1/energy/source-health", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	platform.SignRequest(req, body)

	resp, err := s.client.Do(req)
	if err != nil {
		slog.Debug("source health report failed", "error", err)
		return
	}
	resp.Body.Close()
	slog.Debug("source health reported", "status", resp.StatusCode)
}

// --- UUID persistence ---

type storedUUID struct {
	UUID      string `json:"uuid"`
	CreatedAt string `json:"created_at"`
}

func (s *Sender) loadOrCreateUUID() string {
	return LoadInstallUUID(s.deps.ConfigPath, s.deps.DataPath)
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
// It prefers the shared location (/config/.synctacles_install_id) so both
// Energy and Care apps use the same UUID per HA installation.
// Falls back to the legacy location (/data/.synctacles_uuid.json) and migrates.
func LoadInstallUUID(configPath, dataPath string) string {
	// 1. Shared location (preferred — shared with Care app)
	if configPath != "" {
		sharedPath := filepath.Join(configPath, sharedUUIDFile)
		if data, err := os.ReadFile(sharedPath); err == nil {
			if id := strings.TrimSpace(string(data)); id != "" {
				return id
			}
		}
	}

	// 2. Legacy location (Energy) — migrate to shared if possible
	legacyPath := filepath.Join(dataPath, uuidFile)
	if data, err := os.ReadFile(legacyPath); err == nil {
		var stored storedUUID
		if json.Unmarshal(data, &stored) == nil && stored.UUID != "" {
			if configPath != "" {
				if err := os.WriteFile(filepath.Join(configPath, sharedUUIDFile), []byte(stored.UUID), 0644); err == nil {
					slog.Info("install UUID migrated to shared location", "uuid", stored.UUID)
				}
			}
			return stored.UUID
		}
	}

	// 2b. Legacy location (Care app) — adopt Care's UUID if it exists
	if configPath != "" {
		careLegacy := filepath.Join(configPath, ".care_install_id")
		if data, err := os.ReadFile(careLegacy); err == nil {
			if id := strings.TrimSpace(string(data)); id != "" {
				if err := os.WriteFile(filepath.Join(configPath, sharedUUIDFile), []byte(id), 0644); err == nil {
					slog.Info("install UUID adopted from Care app", "uuid", id)
				}
				return id
			}
		}
	}

	// 3. Generate new UUID at shared location (fallback to legacy)
	id := uuid.New().String()
	if configPath != "" {
		if err := os.WriteFile(filepath.Join(configPath, sharedUUIDFile), []byte(id), 0644); err == nil {
			return id
		}
		slog.Warn("could not write shared UUID, falling back to legacy", "error", "config path not writable")
	}
	stored := storedUUID{UUID: id, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	if out, err := json.MarshalIndent(stored, "", "  "); err == nil {
		_ = os.WriteFile(legacyPath, out, 0600)
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
