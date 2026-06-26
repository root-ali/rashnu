package api

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/root-ali/rashnu/internal/pricing"
)

// FulfillPriceDataHandler handles POST /api/v1/pricing/fulfill
func (h *Handler) FulfillPriceDataHandler(w http.ResponseWriter, r *http.Request) {
	var req pricing.FulfillPriceDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// Parse the start time
	startTime, err := time.Parse("2006-01-02", req.StartTime)
	if err != nil {
		h.logger.Error("invalid start time format", zap.Error(err), zap.String("start_time", req.StartTime))
		writeError(w, http.StatusBadRequest, err)
		return
	}

	// Call the service to fulfill price data
	if err := h.pricing.FulfillPriceData(startTime); err != nil {
		h.logger.Error("failed to fulfill price data", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"message":      "Successfully fulfilled price data",
		"start_time":   startTime,
		"completed_at": time.Now(),
	}

	writeJSON(w, http.StatusOK, response)
}

// CreateMonthlyPriceHandler handles POST /api/v1/pricing/monthly
func (h *Handler) CreateMonthlyPriceHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.pricing.CreateMonthlyPrice(); err != nil {
		h.logger.Error("failed to create monthly price", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	response := map[string]interface{}{
		"message":    "Successfully created monthly prices",
		"created_at": time.Now(),
	}

	writeJSON(w, http.StatusOK, response)
}

// GetAllPricesHandler handles GET /api/v1/pricing
func (h *Handler) GetAllPricesHandler(w http.ResponseWriter, r *http.Request) {
	prices, err := h.pricing.GetAllPrices()
	if err != nil {
		h.logger.Error("failed to get all prices", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	response := pricing.ToPriceResponseList(prices)
	writeJSON(w, http.StatusOK, response)
}
