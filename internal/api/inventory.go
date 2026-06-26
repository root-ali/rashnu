package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/root-ali/rashnu/internal/inventory"
	"go.uber.org/zap"
)

type ListDatacentersResponse struct {
	Datacenters []*inventory.DataCenter `json:"datacenters"`
	Status      string                  `json:"status"`
}

func (h *Handler) ListDatacenterHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handler: listing datacenters")
	datacenters, err := h.inventory.ListDataCenters(r.Context())
	if err != nil {
		h.logger.Error("handler: failed to list datacenters", zap.Error(err))
		http.Error(w, "failed to list datacenters", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("handler: successfully listed datacenters", zap.Int("count", len(datacenters)))
	response := ListDatacentersResponse{
		Status:      "ok",
		Datacenters: datacenters,
	}
	writeJSON(w, 200, response)
}

func (h *Handler) GetDatacenterHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.logger.Debug("handler: getting datacenter by name", zap.String("name", name))
	dc, err := h.inventory.GetDatacentersByName(r.Context(), name)
	if err != nil {
		if errors.Is(err, inventory.ErrDatacenterNotFound) {
			h.logger.Debug("handler: datacenter not found", zap.String("name", name))
			http.Error(w, "datacenter not found", http.StatusNotFound)
			return
		}
		h.logger.Error("handler: failed to get datacenter", zap.String("name", name), zap.Error(err))
		http.Error(w, "failed to get datacenter", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("handler: successfully retrieved datacenter", zap.String("name", name))
	response := ListDatacentersResponse{
		Status:      "ok",
		Datacenters: []*inventory.DataCenter{dc},
	}
	writeJSON(w, 200, response)
}

func (h *Handler) CreateDatacenterHandler(w http.ResponseWriter, r *http.Request) {
	datacenter := inventory.DatacenterCreateRequest{}
	if err := json.NewDecoder(r.Body).Decode(&datacenter); err != nil {
		h.logger.Error("handler: failed to decode request", zap.Error(err))
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}
	h.logger.Debug("handler: creating datacenter", zap.String("name", datacenter.Name))
	err := h.inventory.CreateDatacenter(r.Context(), datacenter)
	if err != nil {
		if errors.Is(err, inventory.ErrDuplicateDatacenter) {
			h.logger.Warn("handler: duplicate datacenter", zap.String("name", datacenter.Name))
			writeError(w, http.StatusConflict, err)
			return
		}
		h.logger.Error("handler: failed to create datacenter", zap.String("name", datacenter.Name), zap.Error(err))
		http.Error(w, "failed to create datacenter", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("handler: successfully created datacenter", zap.String("name", datacenter.Name))
	writeJSON(w, 201, struct{}{})
}

func (h *Handler) UpdateDatacenterHandler(w http.ResponseWriter, r *http.Request) {
	datacenter := inventory.DatacenterUpdateRequest{}
	if err := json.NewDecoder(r.Body).Decode(&datacenter); err != nil {
		h.logger.Error("handler: failed to decode request", zap.Error(err))
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}
	h.logger.Debug("handler: updating datacenter", zap.String("id", datacenter.ID.String()))
	err := h.inventory.UpdateDatacenter(r.Context(), datacenter)
	if err != nil {
		if errors.Is(err, inventory.ErrDatacenterNotFound) {
			h.logger.Debug("handler: datacenter not found for update", zap.String("id", datacenter.ID.String()))
			http.Error(w, "datacenter not found", http.StatusNotFound)
			return
		}
		h.logger.Error("handler: failed to update datacenter", zap.String("id", datacenter.ID.String()), zap.Error(err))
		http.Error(w, "failed to update datacenter", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("handler: successfully updated datacenter", zap.String("id", datacenter.ID.String()))
	writeJSON(w, 200, struct{}{})
}

func (h *Handler) DeleteDatacenterHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.logger.Debug("handler: deleting datacenter", zap.String("id", id))
	err := h.inventory.DeleteDatacenter(r.Context(), id)
	if err != nil {
		if errors.Is(err, inventory.ErrDatacenterNotFound) {
			h.logger.Debug("handler: datacenter not found for deletion", zap.String("id", id))
			http.Error(w, "datacenter not found", http.StatusNotFound)
			return
		}
		h.logger.Error("handler: failed to delete datacenter", zap.String("id", id), zap.Error(err))
		http.Error(w, "failed to delete datacenter", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("handler: successfully deleted datacenter", zap.String("id", id))
	writeJSON(w, 200, struct{}{})
}

// ─── Server Handlers ──────────────────────────────────────────────────────────

type ListServersResponse struct {
	Servers []*inventory.Server `json:"servers"`
	Status  string              `json:"status"`
}

func (h *Handler) ListServersHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("handler: listing servers")
	servers, err := h.inventory.ListServers(r.Context())
	if err != nil {
		h.logger.Error("handler: failed to list servers", zap.Error(err))
		http.Error(w, "failed to list servers", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("handler: successfully listed servers", zap.Int("count", len(servers)))
	response := ListServersResponse{
		Status:  "ok",
		Servers: servers,
	}
	writeJSON(w, 200, response)
}

func (h *Handler) GetServerHandler(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "hostname")
	h.logger.Debug("handler: getting server by hostname", zap.String("hostname", hostname))
	server, err := h.inventory.GetServerByHostname(r.Context(), hostname)
	if err != nil {
		if errors.Is(err, inventory.ErrServerNotFound) {
			h.logger.Debug("handler: server not found", zap.String("hostname", hostname))
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		h.logger.Error("handler: failed to get server", zap.String("hostname", hostname), zap.Error(err))
		http.Error(w, "failed to get server", http.StatusInternalServerError)
		return
	}
	h.logger.Debug("handler: successfully retrieved server", zap.String("hostname", hostname))
	response := ListServersResponse{
		Status:  "ok",
		Servers: []*inventory.Server{server},
	}
	writeJSON(w, 200, response)
}

func (h *Handler) CreateServerHandler(w http.ResponseWriter, r *http.Request) {
	serverReq := &inventory.ServerCreateRequest{}
	if err := json.NewDecoder(r.Body).Decode(serverReq); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}
	err := h.inventory.CreateServer(r.Context(), serverReq)
	if err != nil {
		if errors.Is(err, inventory.ErrDuplicateServer) {
			h.logger.Warn("duplicate server", zap.String("hostname", serverReq.Hostname))
			writeError(w, http.StatusConflict, err)
			return
		}
		h.logger.Error("failed to create server", zap.Error(err))
		http.Error(w, "failed to create server", http.StatusInternalServerError)
		return
	}
	writeJSON(w, 201, struct{}{})
}

func (h *Handler) UpdateServerHandler(w http.ResponseWriter, r *http.Request) {
	serverReq := &inventory.ServerUpdateRequest{}
	if err := json.NewDecoder(r.Body).Decode(serverReq); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}
	err := h.inventory.UpdateServer(r.Context(), serverReq)
	if err != nil {
		h.logger.Error("failed to update server", zap.Error(err))
		http.Error(w, "failed to update server", http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, struct{}{})
}

func (h *Handler) DeleteServerHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.inventory.DeleteServer(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to delete server", zap.String("id", id), zap.Error(err))
		http.Error(w, "failed to delete server", http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, struct{}{})
}

// ─── Infrastructure Hardware Handlers ─────────────────────────────────────────

type ListInfrastructureHardwaresResponse struct {
	Hardwares []*inventory.InfrastructureHardware `json:"hardwares"`
	Status    string                              `json:"status"`
}

func (h *Handler) ListInfrastructureHardwaresHandler(w http.ResponseWriter, r *http.Request) {
	hardwares, err := h.inventory.ListInfrastructureHardwares(r.Context())
	if err != nil {
		h.logger.Error("failed to list infrastructure hardwares", zap.Error(err))
		http.Error(w, "failed to list infrastructure hardwares", http.StatusInternalServerError)
		return
	}
	h.logger.Info("listing infrastructure hardwares", zap.Any("hardwares", hardwares))
	response := ListInfrastructureHardwaresResponse{
		Status:    "ok",
		Hardwares: hardwares,
	}
	writeJSON(w, 200, response)
}

func (h *Handler) GetInfrastructureHardwareHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	hardware, err := h.inventory.GetInfrastructureHardwareByID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get infrastructure hardware", zap.String("id", id), zap.Error(err))
		http.Error(w, "failed to get infrastructure hardware", http.StatusInternalServerError)
		return
	}
	response := ListInfrastructureHardwaresResponse{
		Status:    "ok",
		Hardwares: []*inventory.InfrastructureHardware{hardware},
	}
	writeJSON(w, 200, response)
}

func (h *Handler) CreateInfrastructureHardwareHandler(w http.ResponseWriter, r *http.Request) {
	hwReq := &inventory.InfrastructureHardwareCreateRequest{}
	if err := json.NewDecoder(r.Body).Decode(hwReq); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}
	err := h.inventory.CreateInfrastructureHardware(r.Context(), hwReq)
	if err != nil {
		if errors.Is(err, inventory.ErrDuplicateInfrastructureHardware) {
			h.logger.Warn("duplicate infrastructure hardware", zap.String("hostname", hwReq.Hostname))
			writeError(w, http.StatusConflict, err)
			return
		}
		h.logger.Error("failed to create infrastructure hardware", zap.Error(err))
		http.Error(w, "failed to create infrastructure hardware", http.StatusInternalServerError)
		return
	}
	writeJSON(w, 201, struct{}{})
}

func (h *Handler) UpdateInfrastructureHardwareHandler(w http.ResponseWriter, r *http.Request) {
	hwReq := &inventory.InfrastructureHardwareUpdateRequest{}
	if err := json.NewDecoder(r.Body).Decode(hwReq); err != nil {
		h.logger.Error("failed to decode request", zap.Error(err))
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}
	err := h.inventory.UpdateInfrastructureHardware(r.Context(), hwReq)
	if err != nil {
		if errors.Is(err, inventory.ErrDuplicateInfrastructureHardware) {
			h.logger.Warn("duplicate infrastructure hardware", zap.String("hostname", hwReq.Hostname), zap.String("region", hwReq.Region))
			writeError(w, http.StatusConflict, err)
			return
		}
		if errors.Is(err, inventory.ErrInfrastructureHardwareNotFound) {
			h.logger.Debug("infrastructure hardware not found for update", zap.String("id", hwReq.ID.String()))
			http.Error(w, "infrastructure hardware not found", http.StatusNotFound)
			return
		}
		h.logger.Error("failed to update infrastructure hardware", zap.Error(err))
		http.Error(w, "failed to update infrastructure hardware", http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, struct{}{})
}

func (h *Handler) DeleteInfrastructureHardwareHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.inventory.DeleteInfrastructureHardware(r.Context(), id)
	if err != nil {
		if errors.Is(err, inventory.ErrInfrastructureHardwareNotFound) {
			h.logger.Debug("infrastructure hardware not found for delete", zap.String("id", id))
			writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("failed to delete infrastructure hardware", zap.String("id", id), zap.Error(err))
		http.Error(w, "failed to delete infrastructure hardware", http.StatusInternalServerError)
		return
	}
	writeJSON(w, 200, struct{}{})
}

// ─── Datacenter with Hardware Handlers ────────────────────────────────────────

type DatacentersWithHardwareResponse struct {
	Datacenters []*inventory.DatacenterWithHardware `json:"datacenters"`
	Status      string                              `json:"status"`
}

func (h *Handler) GetDatacentersWithHardwareHandler(w http.ResponseWriter, r *http.Request) {
	nameFilter := r.URL.Query().Get("name")
	regionFilter := r.URL.Query().Get("region")

	h.logger.Debug("handler: getting datacenters with hardware",
		zap.String("name_filter", nameFilter),
		zap.String("region_filter", regionFilter))

	datacenters, err := h.inventory.GetDatacentersWithHardware(r.Context(), nameFilter, regionFilter)
	if err != nil {
		h.logger.Error("handler: failed to get datacenters with hardware", zap.Error(err))
		http.Error(w, "failed to get datacenters with hardware", http.StatusInternalServerError)
		return
	}

	h.logger.Debug("handler: successfully retrieved datacenters with hardware", zap.Int("count", len(datacenters)))
	response := DatacentersWithHardwareResponse{
		Status:      "ok",
		Datacenters: datacenters,
	}
	writeJSON(w, 200, response)
}
