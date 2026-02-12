package models

// Action represents the energy usage recommendation.
type Action string

const (
	ActionGo    Action = "GO"
	ActionWait  Action = "WAIT"
	ActionAvoid Action = "AVOID"
)

// ActionResult holds the computed action with its reasoning.
type ActionResult struct {
	Action       Action  `json:"action"`
	Reason       string  `json:"reason"`
	DeviationPct float64 `json:"deviation_pct"`
	CurrentPrice float64 `json:"current_price"`
	AveragePrice float64 `json:"average_price"`
	Quality      string  `json:"quality"`
}

// BestWindow represents the optimal usage window.
type BestWindow struct {
	StartHour string  `json:"start_hour"`
	EndHour   string  `json:"end_hour"`
	AvgPrice  float64 `json:"avg_price"`
	Duration  int     `json:"duration_hours"`
	TotalCost float64 `json:"total_cost"`
}

// TomorrowPreview represents the preview status for next-day prices.
type TomorrowPreview string

const (
	PreviewFavorable TomorrowPreview = "FAVORABLE"
	PreviewNormal    TomorrowPreview = "NORMAL"
	PreviewExpensive TomorrowPreview = "EXPENSIVE"
	PreviewPending   TomorrowPreview = "PENDING"
)

// TomorrowResult holds the tomorrow preview with context.
type TomorrowResult struct {
	Status        TomorrowPreview `json:"status"`
	CheapestHour  string          `json:"cheapest_hour,omitempty"`
	ExpensiveHour string          `json:"expensive_hour,omitempty"`
	AvgPrice      float64         `json:"avg_price,omitempty"`
	Comparison    string          `json:"comparison,omitempty"`
}
