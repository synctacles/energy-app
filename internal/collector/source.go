// Package collector defines the PriceSource interface and collector implementations.
package collector

import (
	"context"
	"time"

	"github.com/synctacles/energy-go/internal/models"
)

// PriceSource is the interface that all price collectors implement.
type PriceSource interface {
	// Name returns the source identifier (e.g. "easyenergy", "frank", "energycharts").
	Name() string

	// Zones returns the bidding zone codes this source supports.
	Zones() []string

	// RequiresKey returns true if this source needs an API key.
	RequiresKey() bool

	// FetchDayAhead fetches hourly prices for the given zone and date.
	// Returns prices in the source's native unit (MWh or kWh).
	FetchDayAhead(ctx context.Context, zone string, date time.Time) ([]models.HourlyPrice, error)
}
