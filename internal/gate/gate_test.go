package gate

import "testing"

func TestGate_NoAPIKey(t *testing.T) {
	g := New("NL", false, "")

	if g.CanUseActions() {
		t.Error("CanUseActions should be false without API key")
	}
	if g.CanUseTomorrow() {
		t.Error("CanUseTomorrow should be false without API key")
	}
	if !g.CanFetchPrices() {
		t.Error("CanFetchPrices should always be true")
	}
	if g.Status() != "prices_only" {
		t.Errorf("Status = %q, want prices_only", g.Status())
	}
}

func TestGate_WithAPIKeyNoHeartbeat(t *testing.T) {
	g := New("NL", false, "sk_pro_test")

	if g.CanUseActions() {
		t.Error("CanUseActions should require heartbeat")
	}
	if !g.CanUseTomorrow() {
		t.Error("CanUseTomorrow should work with API key even without heartbeat")
	}
	if g.Status() != "prices_only" {
		t.Errorf("Status = %q, want prices_only (no heartbeat)", g.Status())
	}
}

func TestGate_WithAPIKeyAndHeartbeat(t *testing.T) {
	g := New("NL", false, "sk_pro_test")
	g.SetHeartbeatOK(true)

	if !g.CanUseActions() {
		t.Error("CanUseActions should be true with API key + heartbeat")
	}
	if !g.CanUseTomorrow() {
		t.Error("CanUseTomorrow should be true with API key")
	}
	if g.Status() != "registered" {
		t.Errorf("Status = %q, want registered", g.Status())
	}
}

func TestGate_HeartbeatWithoutAPIKey(t *testing.T) {
	g := New("NL", false, "")
	g.SetHeartbeatOK(true) // heartbeat OK but no API key

	if g.CanUseActions() {
		t.Error("CanUseActions should require API key, not just heartbeat")
	}
	if g.CanUseTomorrow() {
		t.Error("CanUseTomorrow should require API key")
	}
	if g.Status() != "prices_only" {
		t.Errorf("Status = %q, want prices_only (no API key)", g.Status())
	}
}
