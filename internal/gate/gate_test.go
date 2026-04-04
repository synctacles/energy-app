package gate

import "testing"

func TestGate_AllFeaturesEnabledByDefault(t *testing.T) {
	g := New("NL", false, "")

	if !g.CanFetchPrices() {
		t.Error("CanFetchPrices should be true")
	}
	if !g.CanUseActions() {
		t.Error("CanUseActions should be true")
	}
	if !g.CanUseTomorrow() {
		t.Error("CanUseTomorrow should be true")
	}
	if !g.CanUseFallback() {
		t.Error("CanUseFallback should be true")
	}
	if g.Status() != "full" {
		t.Errorf("Status = %q, want full", g.Status())
	}
}
