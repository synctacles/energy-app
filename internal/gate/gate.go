// Package gate provides feature availability control.
// All features are enabled by default. When the install is purged (GDPR deletion),
// price fetching, actions, and API features are disabled.
package gate

import "sync/atomic"

// Gate controls which features are available.
type Gate struct {
	purged atomic.Bool
}

// New creates a Gate. All arguments are ignored — kept for compatibility.
func New(_ string, _ bool, _ string) *Gate {
	return &Gate{}
}

// SetPurged marks this install as purged. Disables price fetching and actions.
func (g *Gate) SetPurged() {
	g.purged.Store(true)
}

// IsPurged returns true if this install has been purged.
func (g *Gate) IsPurged() bool {
	return g.purged.Load()
}

func (g *Gate) CanFetchPrices() bool  { return !g.purged.Load() }
func (g *Gate) CanUseActions() bool   { return !g.purged.Load() }
func (g *Gate) CanUseTomorrow() bool  { return !g.purged.Load() }
func (g *Gate) CanUseFallback() bool  { return !g.purged.Load() }
func (g *Gate) Status() string {
	if g.purged.Load() {
		return "purged"
	}
	return "full"
}
func (g *Gate) HeartbeatOK() bool     { return true }
func (g *Gate) SetHeartbeatOK(_ bool) {}
func (g *Gate) IsEneverOnly() bool    { return false }
