// Package gate controls feature availability.
//
// All features are unconditionally enabled — no registration, heartbeat, or lease required.
package gate

// Gate controls which features are available. All gates are permanently open.
type Gate struct{}

// New creates a feature gate. All features are always enabled.
func New() *Gate {
	return &Gate{}
}

// CanFetchPrices always returns true.
func (g *Gate) CanFetchPrices() bool { return true }

// CanUseActions always returns true — GO/WAIT/AVOID actions are free for everyone.
func (g *Gate) CanUseActions() bool { return true }

// CanUseTomorrow always returns true — tomorrow price preview is free for everyone.
func (g *Gate) CanUseTomorrow() bool { return true }

// CanUseFallback always returns true — local fallback scraping is free for everyone.
func (g *Gate) CanUseFallback() bool { return true }

// Status always returns "full" — all features enabled.
func (g *Gate) Status() string { return "full" }
