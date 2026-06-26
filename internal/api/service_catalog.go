package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/root-ali/rashnu/internal/service_catalog"
	"go.uber.org/zap"
)

// ── Services ──────────────────────────────────────────────────────────────────

func (h *Handler) ListServicesHandler(w http.ResponseWriter, r *http.Request) {
	services, err := h.catalog.ListServices(r.Context())
	if err != nil {
		h.logger.Error("list services", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": services})
}

func (h *Handler) GetServiceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	svc, err := h.catalog.GetService(r.Context(), id)
	if err != nil {
		if errors.Is(err, service_catalog.ErrServiceNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("get service", zap.String("id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (h *Handler) CreateServiceHandler(w http.ResponseWriter, r *http.Request) {
	var req service_catalog.CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	id, err := h.catalog.CreateService(r.Context(), req)
	if err != nil {
		if errors.Is(err, service_catalog.ErrServiceDuplicate) {
			writeError(w, http.StatusConflict, err)
			return
		}
		if errors.Is(err, service_catalog.ErrInvalidPlatform) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		h.logger.Error("create service", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *Handler) UpdateServiceHandler(w http.ResponseWriter, r *http.Request) {
	var req service_catalog.UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.catalog.UpdateService(r.Context(), req); err != nil {
		if errors.Is(err, service_catalog.ErrServiceNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("update service", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.catalog.DeleteService(r.Context(), id); err != nil {
		if errors.Is(err, service_catalog.ErrServiceNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("delete service", zap.String("id", id), zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Pod workloads ─────────────────────────────────────────────────────────────

func (h *Handler) AddPodWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	var req service_catalog.AddPodWorkloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.catalog.AddPodWorkload(r.Context(), req); err != nil {
		if errors.Is(err, service_catalog.ErrDatacenterInvalid) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		h.logger.Error("add pod workload", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) UpdatePodWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	var req service_catalog.UpdatePodWorkloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.ServiceID = mustParseUUID(chi.URLParam(r, "id"))
	req.ID = mustParseUUID(chi.URLParam(r, "podId"))
	if err := h.catalog.UpdatePodWorkload(r.Context(), req); err != nil {
		if errors.Is(err, service_catalog.ErrPodNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("update pod workload", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeletePodWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "id")
	podID := chi.URLParam(r, "podId")
	if err := h.catalog.DeletePodWorkload(r.Context(), serviceID, podID); err != nil {
		if errors.Is(err, service_catalog.ErrPodNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("delete pod workload", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── VM workloads ──────────────────────────────────────────────────────────────

func (h *Handler) AddVMWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	var req service_catalog.AddVMWorkloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.catalog.AddVMWorkload(r.Context(), req); err != nil {
		if errors.Is(err, service_catalog.ErrDatacenterInvalid) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		h.logger.Error("add vm workload", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) UpdateVMWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	var req service_catalog.UpdateVMWorkloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.ServiceID = mustParseUUID(chi.URLParam(r, "id"))
	req.ID = mustParseUUID(chi.URLParam(r, "vmId"))
	if err := h.catalog.UpdateVMWorkload(r.Context(), req); err != nil {
		if errors.Is(err, service_catalog.ErrVMNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("update vm workload", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteVMWorkloadHandler(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "id")
	vmID := chi.URLParam(r, "vmId")
	if err := h.catalog.DeleteVMWorkload(r.Context(), serviceID, vmID); err != nil {
		if errors.Is(err, service_catalog.ErrVMNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("delete vm workload", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
