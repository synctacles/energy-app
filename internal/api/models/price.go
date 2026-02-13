package models

import "time"

// Price represents an energy price at a specific time and zone.
type Price struct {
	Zone       string    `json:"zone"`
	Timestamp  time.Time `json:"timestamp"`
	PriceEUR   float64   `json:"price_eur"`
	Unit       string    `json:"unit"`
	Source     string    `json:"source"`
	Quality    string    `json:"quality"`
	IsConsumer bool      `json:"is_consumer"`
	FetchedAt  time.Time `json:"fetched_at"`
}

// PricesResponse is the API response for /prices endpoint.
type PricesResponse struct {
	Zone      string    `json:"zone"`
	Prices    []Price   `json:"prices"`
	Count     int       `json:"count"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// DashboardResponse is the API response for /dashboard endpoint (Pro tier).
type DashboardResponse struct {
	Zone         string    `json:"zone"`
	CurrentPrice float64   `json:"current_price"`
	CurrentTime  time.Time `json:"current_time"`
	TodayAvg     float64   `json:"today_avg"`
	TodayMin     float64   `json:"today_min"`
	TodayMax     float64   `json:"today_max"`
	NextHour     *Price    `json:"next_hour,omitempty"`
	Cheapest     *Price    `json:"cheapest_today,omitempty"`
	MostExpensive *Price   `json:"most_expensive_today,omitempty"`
}

// BestWindowResponse is the API response for /best-window endpoint (Pro tier).
type BestWindowResponse struct {
	Zone       string    `json:"zone"`
	WindowSize int       `json:"window_size_hours"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	AvgPrice   float64   `json:"avg_price_eur"`
	Savings    float64   `json:"savings_eur"`
}

// EnergyAction represents GO/WAIT/AVOID recommendation.
type EnergyAction string

const (
	ActionGo    EnergyAction = "GO"
	ActionWait  EnergyAction = "WAIT"
	ActionAvoid EnergyAction = "AVOID"
)

// EnergyActionResponse is the API response for /energy-action endpoint (Pro tier).
type EnergyActionResponse struct {
	Zone      string       `json:"zone"`
	Action    EnergyAction `json:"action"`
	CurrentPrice float64   `json:"current_price_eur"`
	TodayAvg  float64      `json:"today_avg_eur"`
	Reason    string       `json:"reason"`
	Timestamp time.Time    `json:"timestamp"`
}
