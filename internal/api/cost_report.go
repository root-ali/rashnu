package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/root-ali/rashnu/internal/cost_report"
	"go.uber.org/zap"
)

func normalizeDayUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (h *Handler) CalculateDailyCostsHandler(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	var date time.Time
	if dateStr == "" {
		date = time.Now().UTC()
	} else {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, cost_report.ErrInvalidDate)
			return
		}
	}
	date = normalizeDayUTC(date)

	if err := h.costReport.CalculateDailyServiceCosts(r.Context(), date); err != nil {
		h.logger.Error("calculate daily service costs", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) FulfillCostReportsHandler(w http.ResponseWriter, r *http.Request) {
	var req cost_report.FulfillCostReportsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	startTime, err := time.Parse("2006-01-02", req.StartTime)
	if err != nil {
		writeError(w, http.StatusBadRequest, cost_report.ErrInvalidDate)
		return
	}

	if err := h.costReport.FulfillServiceCosts(r.Context(), startTime); err != nil {
		h.logger.Error("fulfill cost reports", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetMonthlyCostReportsHandler(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	var month time.Time
	if monthStr == "" {
		now := time.Now().UTC()
		month = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		var err error
		month, err = time.Parse("2006-01", monthStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, cost_report.ErrInvalidMonth)
			return
		}
	}

	reports, err := h.costReport.GetMonthlyReportsForMonth(r.Context(), month)
	if err != nil {
		h.logger.Error("get monthly cost reports", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	var total int64
	for _, rep := range reports {
		total += rep.TotalCost
	}

	writeJSON(w, http.StatusOK, cost_report.MonthlyReportsResponse{
		Reports: reports,
		Month:   time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC),
		Total:   total,
	})
}

func (h *Handler) GetCostTrendHandler(w http.ResponseWriter, r *http.Request) {
	months := 12
	if s := r.URL.Query().Get("months"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			if n > 60 {
				n = 60
			}
			months = n
		}
	}

	points, err := h.costReport.GetMonthlyTrend(r.Context(), months)
	if err != nil {
		h.logger.Error("get cost trend", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, cost_report.TrendResponse{Points: points})
}

func (h *Handler) GetCostSpendHandler(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	var month time.Time
	if monthStr == "" {
		now := time.Now().UTC()
		month = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		var err error
		month, err = time.Parse("2006-01", monthStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, cost_report.ErrInvalidMonth)
			return
		}
	}

	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "service"
	}

	platform := r.URL.Query().Get("platform")
	if platform == "all" {
		platform = ""
	}

	summary, err := h.costReport.GetSpendSummary(r.Context(), month, groupBy, platform)
	if err != nil {
		if err == cost_report.ErrInvalidGroupBy {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		h.logger.Error("get cost spend summary", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}
