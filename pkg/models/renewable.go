package models

import "time"

// RenewablePoint represents a single timestamp's renewable energy share data.
type RenewablePoint struct {
	Timestamp         time.Time `json:"timestamp"`
	RenShare          float64   `json:"ren_share"`           // Renewable share % (0-100)
	Signal            int       `json:"signal"`              // EC signal: 1=green, 2=yellow
	SolarShare        *float64  `json:"solar_share"`         // Solar % of total load
	WindOnshoreShare  *float64  `json:"wind_onshore_share"`  // Onshore wind %
	WindOffshoreShare *float64  `json:"wind_offshore_share"` // Offshore wind %
}

// SignalLabel returns a human-readable label for the EC signal value.
func (r RenewablePoint) SignalLabel() string {
	if r.Signal == 1 {
		return "green"
	}
	return "yellow"
}

// RenewableData holds the full renewable response from the Worker.
type RenewableData struct {
	Zone       string           `json:"zone"`
	Resolution string           `json:"resolution"`
	Source     string           `json:"source"`
	Current    *RenewablePoint  `json:"current"`
	Data       []RenewablePoint `json:"data"`
}
