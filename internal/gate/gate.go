// Package gate controls feature availability based on heartbeat status and lease validity.
//
// Feature matrix for Energy App:
//
//	Without heartbeat: NOTHING works (except Enever prices for NL, without actions)
//	With heartbeat:    Everything works. Local fallback requires a valid server-signed lease.
package gate

import (
	"log/slog"
	"sync"

	"github.com/synctacles/energy-backend/pkg/lease"
)

// LeasePublicKey is the Ed25519 public key for verifying fallback leases.
// This is compiled into the binary — the matching private key is on the energy-server.
const LeasePublicKey = "qVXQZowCyCDPV/2q+cGY3IhBc+BYJvhX7lbkuKfiDFg="

// Gate controls which features are available based on heartbeat and lease status.
type Gate struct {
	mu            sync.RWMutex
	heartbeatOK   bool
	isNL          bool
	hasEnever     bool
	leaseVerifier *lease.Verifier
	currentLease  *lease.Lease
}

// New creates a feature gate for the given zone configuration.
func New(biddingZone string, eneverEnabled bool) *Gate {
	v, err := lease.NewVerifier(LeasePublicKey)
	if err != nil {
		slog.Error("failed to create lease verifier", "error", err)
	}
	return &Gate{
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
// NL with Enever: always allowed (local Dutch source, user's own API key).
// All others: requires heartbeat.
func (g *Gate) CanFetchPrices() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.isNL && g.hasEnever {
		return true
	}
	return g.heartbeatOK
}

// CanUseActions returns whether GO/WAIT/AVOID actions and automations are allowed.
// Always requires heartbeat, even for NL Enever users.
func (g *Gate) CanUseActions() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.heartbeatOK
}

// CanUseFallback returns whether local fallback scraping is allowed.
// Requires heartbeat + valid server-signed lease.
// NL Enever: Enever itself is always allowed (it's the user's own API key),
// but scraping other sources (Energy-Charts etc.) requires a lease.
func (g *Gate) CanUseFallback() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.heartbeatOK {
		return false
	}
	if g.currentLease == nil || g.leaseVerifier == nil {
		return false
	}
	return g.leaseVerifier.Verify(*g.currentLease)
}

// IsEneverOnly returns true if only Enever (NL) is available without heartbeat.
// When true, the source chain should be limited to Enever only.
func (g *Gate) IsEneverOnly() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.isNL && g.hasEnever && !g.heartbeatOK
}

// Status returns a human-readable status for the web UI.
func (g *Gate) Status() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.heartbeatOK {
		if g.currentLease != nil && g.leaseVerifier != nil && g.leaseVerifier.Verify(*g.currentLease) {
			return "full"
		}
		return "active"
	}
	if g.isNL && g.hasEnever {
		return "enever_only"
	}
	return "disabled"
}
