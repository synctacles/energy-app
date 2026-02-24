// Package gate controls feature availability based on heartbeat status and lease validity.
//
// Feature matrix for Energy App:
//
//	Without heartbeat: Prices work (Synctacles server + Enever NL). No actions, no fallback.
//	With heartbeat:    Everything works. Local fallback requires a valid server-signed lease.
package gate

import (
	"log/slog"
	"sync"

	"github.com/synctacles/energy-app/pkg/lease"
)

// LeasePublicKey is the Ed25519 public key for verifying fallback leases.
// This is compiled into the binary — the matching private key is on the energy-server.
const LeasePublicKey = "i7kSvrX2E3m2B/5AsUEn/mV9OhWdZvku0APYlvZMQ1o="

// Gate controls which features are available based on heartbeat and lease status.
type Gate struct {
	mu            sync.RWMutex
	heartbeatOK   bool
	isRegistered  bool // true when a Synctacles API key is configured
	isNL          bool
	hasEnever     bool
	leaseVerifier *lease.Verifier
	currentLease  *lease.Lease
}

// New creates a feature gate for the given zone configuration.
// apiKey is the Synctacles license/API key — non-empty means the user is registered.
func New(biddingZone string, eneverEnabled bool, apiKey string) *Gate {
	v, err := lease.NewVerifier(LeasePublicKey)
	if err != nil {
		slog.Error("failed to create lease verifier", "error", err)
	}
	return &Gate{
		isRegistered:  apiKey != "",
		isNL:          biddingZone == "NL",
		hasEnever:     eneverEnabled,
		leaseVerifier: v,
	}
}

// SetHeartbeatOK marks the heartbeat as succeeded.
func (g *Gate) SetHeartbeatOK(ok bool) {
	g.mu.Lock()
	g.heartbeatOK = ok
	g.mu.Unlock()
	slog.Debug("gate: heartbeat status updated", "ok", ok)
}

// HeartbeatOK returns whether the last heartbeat succeeded.
func (g *Gate) HeartbeatOK() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.heartbeatOK
}

// UpdateLease stores a new lease received from the energy server.
func (g *Gate) UpdateLease(l *lease.Lease) {
	if l == nil {
		return
	}
	g.mu.Lock()
	g.currentLease = l
	g.mu.Unlock()
	slog.Debug("gate: lease updated", "expires_at", l.ExpiresAt)
}

// CanFetchPrices returns whether the app is allowed to fetch prices at all.
// Always allowed — price display is a basic feature that works without heartbeat.
// Only actions and fallback require heartbeat.
func (g *Gate) CanFetchPrices() bool {
	return true
}

// CanUseActions returns whether GO/WAIT/AVOID actions and automations are allowed.
// Requires both a registered API key and a live heartbeat.
func (g *Gate) CanUseActions() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.isRegistered && g.heartbeatOK
}

// CanUseTomorrow returns whether tomorrow price preview is allowed.
// Requires a registered API key (heartbeat not required).
func (g *Gate) CanUseTomorrow() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.isRegistered
}

// CanUseFallback returns whether local fallback scraping is allowed.
// Requires registration (API key) + heartbeat + valid server-signed lease.
// NL Enever: Enever itself is always allowed (it's the user's own API key),
// but scraping other sources (Energy-Charts etc.) requires a lease.
func (g *Gate) CanUseFallback() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.isRegistered || !g.heartbeatOK {
		return false
	}
	if g.currentLease == nil || g.leaseVerifier == nil {
		return false
	}
	return g.leaseVerifier.Verify(*g.currentLease)
}

// IsEneverOnly returns true if NL Enever is available but heartbeat is off.
// When true, the source chain includes Enever but excludes fallback sources.
func (g *Gate) IsEneverOnly() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.isNL && g.hasEnever && !g.heartbeatOK
}

// Status returns a human-readable status for the web UI.
// "full"        — registered + heartbeat + valid lease (all features)
// "registered"  — registered + heartbeat, no lease (actions work, no fallback)
// "prices_only" — no API key (basic prices only)
func (g *Gate) Status() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.isRegistered && g.heartbeatOK {
		if g.currentLease != nil && g.leaseVerifier != nil && g.leaseVerifier.Verify(*g.currentLease) {
			return "full"
		}
		return "registered"
	}
	return "prices_only"
}
