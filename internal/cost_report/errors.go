package cost_report

import "errors"

var (
	// ErrReportNotFound is returned when a cost report cannot be found.
	ErrReportNotFound = errors.New("cost report not found")

	// ErrInvalidDate is returned when a date string cannot be parsed.
	ErrInvalidDate = errors.New("invalid date format, expected YYYY-MM-DD")

	// ErrInvalidMonth is returned when a month string cannot be parsed.
	ErrInvalidMonth = errors.New("invalid month format, expected YYYY-MM")

	// ErrNoPricingData is returned when no pricing data exists for the requested date.
	ErrNoPricingData = errors.New("no pricing data available for the requested date")

	// ErrInvalidGroupBy is returned when group_by query param is not supported.
	ErrInvalidGroupBy = errors.New("invalid group_by, expected team|platform|service|datacenter")
)
