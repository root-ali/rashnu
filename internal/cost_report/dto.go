package cost_report

import "time"

// ── Request types ─────────────────────────────────────────────────────────────

// CalculateDailyCostsRequest carries the date for which costs should be calculated.
type CalculateDailyCostsRequest struct {
	Date string `json:"date"` // YYYY-MM-DD; empty means today
}

// FulfillCostReportsRequest carries the start date for a cost backfill.
type FulfillCostReportsRequest struct {
	StartTime string `json:"start_time"` // YYYY-MM-DD; backfills from this date up to today
}

// GetMonthlyReportsRequest selects monthly cost reports for a specific month.
type GetMonthlyReportsRequest struct {
	Month string `json:"month"` // YYYY-MM; empty means current month
}

// ── Response types ────────────────────────────────────────────────────────────

// MonthlyReportsResponse wraps a slice of MonthlyReport with summary fields.
type MonthlyReportsResponse struct {
	Reports []*MonthlyReport `json:"reports"`
	Month   time.Time        `json:"month"`
	Total   int64            `json:"total"`
}

// TrendPoint is a single data point in a monthly cost trend series.
type TrendPoint struct {
	Month     time.Time `json:"month"`
	TotalCost int64     `json:"total_cost"`
}

// TrendResponse wraps a slice of TrendPoint for the trend endpoint.
type TrendResponse struct {
	Points []TrendPoint `json:"points"`
}

// SpendItem is a single row in a grouped spend summary.
type SpendItem struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	TotalCost    int64  `json:"total_cost"`
	ServiceCount int    `json:"service_count,omitempty"`
	Platform     string `json:"platform,omitempty"`
	Team         string `json:"team,omitempty"`
}

// SpendSummaryResponse wraps grouped spend for charts and breakdowns.
type SpendSummaryResponse struct {
	Month   time.Time   `json:"month"`
	GroupBy string      `json:"group_by"`
	Total   int64       `json:"total"`
	Items   []SpendItem `json:"items"`
}
