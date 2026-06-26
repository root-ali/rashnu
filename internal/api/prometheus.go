package api

import (
	"encoding/json"
	"net/http"

	"github.com/root-ali/rashnu/internal/service_catalog"
	"go.uber.org/zap"
)

func (h *Handler) GetPrometheusConfigHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.catalog.GetPrometheusConfig(r.Context())
	if err != nil {
		h.logger.Error("get prometheus config", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *Handler) UpdatePrometheusConfigHandler(w http.ResponseWriter, r *http.Request) {
	var req service_catalog.UpdatePrometheusConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.catalog.UpdatePrometheusConfig(r.Context(), req); err != nil {
		h.logger.Error("update prometheus config", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	h.logger.Info("prometheus config updated", zap.String("url", req.URL), zap.Bool("enabled", req.Enabled))
	w.WriteHeader(http.StatusNoContent)
}
