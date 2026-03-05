// Package gate provides feature availability control.
// All features are always enabled.
package gate

// Gate controls which features are available.
type Gate struct{}

// New creates a Gate. All arguments are ignored — kept for compatibility.
func New(_ string, _ bool, _ string) *Gate {
	return &Gate{}
}

func (g *Gate) CanFetchPrices() bool  { return true }
func (g *Gate) CanUseActions() bool   { return true }
func (g *Gate) CanUseTomorrow() bool  { return true }
func (g *Gate) CanUseFallback() bool  { return true }
func (g *Gate) Status() string        { return "full" }
func (g *Gate) HeartbeatOK() bool     { return true }
func (g *Gate) SetHeartbeatOK(_ bool) {}
func (g *Gate) IsEneverOnly() bool    { return false }
