// Package state provides persistence for energy addon state.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// EnergyState tracks the last known energy state for change detection.
type EnergyState struct {
	Zone         string  `json:"zone"`
	CurrentPrice float64 `json:"current_price"`
	Action       string  `json:"action"`
	Quality      string  `json:"quality"`
	LastFetch    string  `json:"last_fetch"`
	LastPublish  string  `json:"last_publish"`
	PriceSource  string  `json:"price_source"`
}

// Store manages state persistence via atomic JSON writes.
type Store struct {
	path string
}

// NewStore creates a state store at the given config directory.
func NewStore(configPath string) *Store {
	return &Store{path: filepath.Join(configPath, "energy_state.json")}
}

// Load loads the current state. Returns empty state on error.
func (s *Store) Load() *EnergyState {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return &EnergyState{}
	}
	var state EnergyState
	if json.Unmarshal(data, &state) != nil {
		return &EnergyState{}
	}
	return &state
}

// Save persists the state atomically (write temp then rename).
func (s *Store) Save(state *EnergyState) error {
	state.LastPublish = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
