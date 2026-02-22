// Package sourcehealth tracks price source usage and sends daily reports
// to the Synctacles API for monitoring fallback rates and source reliability.
package sourcehealth

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/synctacles/energy-backend/pkg/engine"
)

const endpoint = "https://api.synctacles.com/api/v1/energy/source-health"

// Config for creating a source health tracker.
type Config struct {
	InstallUUID string
	Zone        string   // bidding zone (used as country)
	SourceNames []string // ordered source chain names
}

type sourceEntry struct {
	Name    string `json:"name"`
	Fetches int    `json:"fetches"`
	Tier    int    `json:"tier"`
}

// Tracker accumulates price source usage per day and sends periodic reports.
type Tracker struct {
	cfg Config

	mu           sync.Mutex
	currentDate  string
	sources      map[string]*sourceEntry
	fallbacks    int
	activeSource string
}

// NewTracker creates a source health tracker.
func NewTracker(cfg Config) *Tracker {
	return &Tracker{
		cfg:     cfg,
		sources: make(map[string]*sourceEntry),
	}
}

// Record registers a successful price fetch result.
// Called from the scheduler update callback on every fetch cycle.
func (t *Tracker) Record(result *engine.FetchResult) {
	if result == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	today := time.Now().UTC().Format("2006-01-02")
	if t.currentDate != today {
		t.currentDate = today
		t.sources = make(map[string]*sourceEntry)
		t.fallbacks = 0
	}

	se, ok := t.sources[result.Source]
	if !ok {
		se = &sourceEntry{Name: result.Source, Tier: result.Tier}
		t.sources[result.Source] = se
	}
	se.Fetches++

	if result.Tier > 1 {
		t.fallbacks++
	}

	t.activeSource = result.Source
}

// Run sends source health reports every 6 hours. Blocks until ctx is cancelled.
func (t *Tracker) Run(ctx context.Context) {
	// Wait 5 minutes for initial data accumulation
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Minute):
	}

	t.send(ctx)

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.send(ctx)
		}
	}
}

func (t *Tracker) send(ctx context.Context) {
	t.mu.Lock()
	date := t.currentDate
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}

	// Build sources list — include all configured sources (even unused ones)
	seen := make(map[string]bool)
	var sources []sourceEntry
	for _, se := range t.sources {
		sources = append(sources, *se)
		seen[se.Name] = true
	}
	for i, name := range t.cfg.SourceNames {
		if !seen[name] {
			sources = append(sources, sourceEntry{Name: name, Tier: i + 1})
		}
	}

	fallbacks := t.fallbacks
	active := t.activeSource
	t.mu.Unlock()

	body, _ := json.Marshal(map[string]any{
		"install_id":      t.cfg.InstallUUID,
		"report_date":     date,
		"country":         t.cfg.Zone,
		"active_source":   active,
		"fallback_events": fallbacks,
		"sources":         sources,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		slog.Debug("sourcehealth: request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		slog.Debug("sourcehealth: send failed", "error", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Info("source health sent", "date", date, "sources", len(sources), "fallbacks", fallbacks)
	} else {
		slog.Debug("sourcehealth: unexpected status", "status", resp.StatusCode)
	}
}
