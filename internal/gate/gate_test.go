package gate

import "testing"

func TestGate_AllFeaturesEnabledByDefault(t *testing.T) {
	g := New("NL", false, "")

	if !g.CanFetchPrices() {
		t.Error("CanFetchPrices should be true by default")
	}
	if !g.CanUseActions() {
		t.Error("CanUseActions should be true by default")
	}
	if !g.CanUseTomorrow() {
		t.Error("CanUseTomorrow should be true by default")
	}
	if !g.CanUseFallback() {
		t.Error("CanUseFallback should be true by default")
	}
	if g.Status() != "full" {
		t.Errorf("Status = %q, want full", g.Status())
	}
	if g.IsPurged() {
		t.Error("IsPurged should be false by default")
	}
}

func TestGate_PurgedDisablesFeatures(t *testing.T) {
	g := New("NL", false, "")
	g.SetPurged()

	if g.CanFetchPrices() {
		t.Error("CanFetchPrices should be false after purge")
	}
	if g.CanUseActions() {
		t.Error("CanUseActions should be false after purge")
	}
	if g.CanUseTomorrow() {
		t.Error("CanUseTomorrow should be false after purge")
	}
	if g.CanUseFallback() {
		t.Error("CanUseFallback should be false after purge")
	}
	if g.Status() != "purged" {
		t.Errorf("Status = %q, want purged", g.Status())
	}
	if !g.IsPurged() {
		t.Error("IsPurged should be true after purge")
	}
}
