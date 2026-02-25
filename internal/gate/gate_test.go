package gate

import "testing"

func TestGate_AllFeaturesAlwaysEnabled(t *testing.T) {
	g := New("NL", false, "")

	if !g.CanFetchPrices() {
		t.Error("CanFetchPrices should always be true")
	}
	if !g.CanUseActions() {
		t.Error("CanUseActions should always be true")
	}
	if !g.CanUseTomorrow() {
		t.Error("CanUseTomorrow should always be true")
	}
	if !g.CanUseFallback() {
		t.Error("CanUseFallback should always be true")
	}
	if g.Status() != "full" {
		t.Errorf("Status = %q, want full", g.Status())
	}
	if !g.HeartbeatOK() {
		t.Error("HeartbeatOK should always be true")
	}
}
