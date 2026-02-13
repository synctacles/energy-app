package web

import (
	"sync"

	"github.com/synctacles/energy-app/internal/hasensor"
)

// SensorData holds the latest sensor values for the web API.
// Thread-safe for concurrent read/write.
type SensorData struct {
	mu   sync.RWMutex
	data *hasensor.SensorSet
}

// NewSensorData creates an empty sensor data holder.
func NewSensorData() *SensorData {
	return &SensorData{}
}

// Update stores the latest sensor set.
func (s *SensorData) Update(set *hasensor.SensorSet) {
	s.mu.Lock()
	s.data = set
	s.mu.Unlock()
}

// Get returns the latest sensor set (may be nil if no data yet).
func (s *SensorData) Get() *hasensor.SensorSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}
