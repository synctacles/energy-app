package delta

import (
	"testing"
	"time"
)

func TestSubmitter_PruneStale(t *testing.T) {
	s := &Submitter{
		wsCache:   make(map[string]float64),
		lastDelta: make(map[string]float64),
	}

	now := time.Now().UTC()

	// Add entries: some recent, some old
	recentKey := now.Add(-1 * time.Hour).Format("2006-01-02T15")
	staleKey := now.Add(-49 * time.Hour).Format("2006-01-02T15")
	veryOldKey := now.Add(-100 * time.Hour).Format("2006-01-02T15")

	s.wsCache[recentKey] = 0.08
	s.wsCache[staleKey] = 0.07
	s.wsCache[veryOldKey] = 0.06

	s.lastDelta[recentKey] = 0.005
	s.lastDelta[staleKey] = 0.004
	// Also test the longer key format used in lastDelta
	staleFullKey := now.Add(-50 * time.Hour).Truncate(time.Hour).Format("2006-01-02T15:04:05Z")
	s.lastDelta[staleFullKey] = 0.003

	s.pruneStale()

	// Recent entries should survive
	if _, ok := s.wsCache[recentKey]; !ok {
		t.Error("recent wsCache entry should survive pruning")
	}
	if _, ok := s.lastDelta[recentKey]; !ok {
		t.Error("recent lastDelta entry should survive pruning")
	}

	// Stale entries (>48h) should be removed
	if _, ok := s.wsCache[staleKey]; ok {
		t.Error("stale wsCache entry should be pruned")
	}
	if _, ok := s.wsCache[veryOldKey]; ok {
		t.Error("very old wsCache entry should be pruned")
	}
	if _, ok := s.lastDelta[staleKey]; ok {
		t.Error("stale lastDelta entry should be pruned")
	}
	if _, ok := s.lastDelta[staleFullKey]; ok {
		t.Error("stale full-format lastDelta entry should be pruned")
	}
}

func TestSubmitter_PruneKeepsRecent(t *testing.T) {
	s := &Submitter{
		wsCache:   make(map[string]float64),
		lastDelta: make(map[string]float64),
	}

	now := time.Now().UTC()

	// All entries within 48h window
	for h := 0; h < 47; h++ {
		key := now.Add(-time.Duration(h) * time.Hour).Format("2006-01-02T15")
		s.wsCache[key] = 0.08 + float64(h)*0.001
		s.lastDelta[key] = 0.005 + float64(h)*0.0001
	}

	beforeWS := len(s.wsCache)
	beforeDelta := len(s.lastDelta)

	s.pruneStale()

	if len(s.wsCache) != beforeWS {
		t.Errorf("wsCache: expected %d entries, got %d (no recent entries should be pruned)",
			beforeWS, len(s.wsCache))
	}
	if len(s.lastDelta) != beforeDelta {
		t.Errorf("lastDelta: expected %d entries, got %d (no recent entries should be pruned)",
			beforeDelta, len(s.lastDelta))
	}
}

func TestSubmitter_PruneEmptyMaps(t *testing.T) {
	s := &Submitter{
		wsCache:   make(map[string]float64),
		lastDelta: make(map[string]float64),
	}

	// Should not panic on empty maps
	s.pruneStale()

	if len(s.wsCache) != 0 || len(s.lastDelta) != 0 {
		t.Error("empty maps should remain empty after pruning")
	}
}

func TestSubmitter_PruneBoundary(t *testing.T) {
	s := &Submitter{
		wsCache:   make(map[string]float64),
		lastDelta: make(map[string]float64),
	}

	now := time.Now().UTC()

	// Entry exactly at 48h boundary — should be pruned (cutoff is exclusive)
	boundaryKey := now.Add(-48 * time.Hour).Format("2006-01-02T15")
	s.wsCache[boundaryKey] = 0.07

	// Entry just inside the window
	insideKey := now.Add(-47 * time.Hour).Format("2006-01-02T15")
	s.wsCache[insideKey] = 0.08

	s.pruneStale()

	if _, ok := s.wsCache[boundaryKey]; ok {
		t.Error("entry at exactly 48h should be pruned")
	}
	if _, ok := s.wsCache[insideKey]; !ok {
		t.Error("entry at 47h should survive")
	}
}
