package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/metrics"
	"github.com/google/uuid"
	"github.com/root-ali/rashnu/internal/cost_report"
	"github.com/root-ali/rashnu/internal/identity"
	"github.com/root-ali/rashnu/internal/service_catalog"
	"go.uber.org/zap"

	"github.com/root-ali/rashnu/internal/inventory"
	"github.com/root-ali/rashnu/internal/pricing"
)

type Handler struct {
	inventory  InventoryService
	pricing    PricingService
	identity   IdentityService
	catalog    ServiceCatalogService
	costReport CostReportService
	jwtSecret  []byte
	logger     *zap.Logger
}

type InventoryService interface {
	// Datacenters - List of datacenter and servers on different hosts
	ListDataCenters(ctx context.Context) ([]*inventory.DataCenter, error)
	GetDatacentersByName(ctx context.Context, name string) (*inventory.DataCenter, error)
	CreateDatacenter(context.Context, inventory.DatacenterCreateRequest) error
	UpdateDatacenter(ctx context.Context, datacenter inventory.DatacenterUpdateRequest) error
	DeleteDatacenter(ctx context.Context, id string) error
	GetDatacentersWithHardware(ctx context.Context, nameFilter, regionFilter string) ([]*inventory.DatacenterWithHardware, error)

	// Servers - Production servers and non-infrastructure hardware like servers storages.
	ListServers(ctx context.Context) ([]*inventory.Server, error)
	GetServerByHostname(ctx context.Context, hostname string) (*inventory.Server, error)
	CreateServer(ctx context.Context, scr *inventory.ServerCreateRequest) error
	UpdateServer(ctx context.Context, sur *inventory.ServerUpdateRequest) error
	DeleteServer(ctx context.Context, id string) error

	// Infrastructure Hardware - switches, routers, firewalls, etc.
	ListInfrastructureHardwares(ctx context.Context) ([]*inventory.InfrastructureHardware, error)
	GetInfrastructureHardwareByID(ctx context.Context, id string) (*inventory.InfrastructureHardware, error)
	CreateInfrastructureHardware(ctx context.Context, req *inventory.InfrastructureHardwareCreateRequest) error
	UpdateInfrastructureHardware(ctx context.Context, req *inventory.InfrastructureHardwareUpdateRequest) error
	DeleteInfrastructureHardware(ctx context.Context, id string) error
}

type PricingService interface {
	FulfillPriceData(startTime time.Time) error
	CreateMonthlyPrice() error
	GetAllPrices() ([]*pricing.Price, error)
}

type ServiceCatalogService interface {
	CreateService(ctx context.Context, req service_catalog.CreateServiceRequest) (uuid.UUID, error)
	GetService(ctx context.Context, id string) (*service_catalog.Service, error)
	ListServices(ctx context.Context) ([]*service_catalog.Service, error)
	UpdateService(ctx context.Context, req service_catalog.UpdateServiceRequest) error
	DeleteService(ctx context.Context, id string) error

	AddPodWorkload(ctx context.Context, req service_catalog.AddPodWorkloadRequest) error
	UpdatePodWorkload(ctx context.Context, req service_catalog.UpdatePodWorkloadRequest) error
	DeletePodWorkload(ctx context.Context, serviceID, podID string) error

	AddVMWorkload(ctx context.Context, req service_catalog.AddVMWorkloadRequest) error
	UpdateVMWorkload(ctx context.Context, req service_catalog.UpdateVMWorkloadRequest) error
	DeleteVMWorkload(ctx context.Context, serviceID, vmID string) error

	GetPrometheusConfig(ctx context.Context) (*service_catalog.PrometheusConfig, error)
	UpdatePrometheusConfig(ctx context.Context, req service_catalog.UpdatePrometheusConfigRequest) error
}

type CostReportService interface {
	CalculateDailyServiceCosts(ctx context.Context, date time.Time) error
	FulfillServiceCosts(ctx context.Context, startTime time.Time) error
	GetMonthlyReportsForMonth(ctx context.Context, month time.Time) ([]*cost_report.MonthlyReport, error)
	GetMonthlyTrend(ctx context.Context, months int) ([]cost_report.TrendPoint, error)
	GetSpendSummary(ctx context.Context, month time.Time, groupBy, platform string) (*cost_report.SpendSummaryResponse, error)
}

type IdentityService interface {
	Create(context.Context, identity.CreateUserRequest) error
	Login(context.Context, identity.LoginRequest) (identity.LoginResponse, error)
	Logout(ctx context.Context, email string) error
	Update(ctx context.Context, u identity.UpdateUserRequest) error
	UpdatePassword(ctx context.Context, upr identity.UpdateUserPasswordRequest) error
	Delete(ctx context.Context, id string) error
	GetAll(ctx context.Context) ([]*identity.User, error)
	IsTokenValid(email, token string) bool
}

func NewHandler(inventory InventoryService, pricing PricingService, identity IdentityService, catalog ServiceCatalogService, costReport CostReportService, jwtSecret []byte, logger *zap.Logger) *Handler {
	return &Handler{
		inventory:  inventory,
		pricing:    pricing,
		identity:   identity,
		catalog:    catalog,
		costReport: costReport,
		jwtSecret:  jwtSecret,
		logger:     logger,
	}
}

// mustParseUUID parses a URL param UUID; returns zero UUID on invalid input
// (the service layer will return not-found for a zero UUID).
func mustParseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	// Configure middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.Compress(5, "bzip"))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(h.loggingMiddleware)

	// metrics middleware
	r.Use(metrics.Collector(metrics.CollectorOpts{
		Host:  false,
		Proto: true,
		Skip: func(r *http.Request) bool {
			return r.Method != "OPTIONS"
		},
	}))

	r.Handle("/metrics", metrics.Handler())

	// healthechk api
	r.Get("/healthz", h.healthz)

	// Serve static files for the frontend
	fileServer(r, "/", http.Dir("./frontend/dist/"))

	// Api handler for version 1
	r.Route("/api/v1", func(r chi.Router) {
		// Public identity routes (no auth required)
		r.Post("/user/login", h.LoginHandler)

		// Protected routes — token required for all groups below
		r.Group(func(r chi.Router) {
			r.Use(h.AuthMiddleware)

			// Admin-only: write operations
			r.Group(func(r chi.Router) {
				r.Use(h.RequireRole("admin"))

				r.Post("/user/new", h.CreateUserHandler)
				r.Put("/user/{id}", h.UpdateUserHandler)
				r.Get("/user", h.GetAllUserHandler)
				r.Delete("/user/{id}", h.DeleteUserHandler)

				r.Post("/datacenters", h.CreateDatacenterHandler)
				r.Put("/datacenters", h.UpdateDatacenterHandler)
				r.Delete("/datacenters/{id}", h.DeleteDatacenterHandler)

				r.Post("/servers", h.CreateServerHandler)
				r.Put("/servers", h.UpdateServerHandler)
				r.Delete("/servers/{id}", h.DeleteServerHandler)

				r.Post("/infrastructure-hardwares", h.CreateInfrastructureHardwareHandler)
				r.Put("/infrastructure-hardwares", h.UpdateInfrastructureHardwareHandler)
				r.Delete("/infrastructure-hardwares/{id}", h.DeleteInfrastructureHardwareHandler)

				r.Post("/pricing/fulfill", h.FulfillPriceDataHandler)
				r.Post("/pricing/monthly", h.CreateMonthlyPriceHandler)

				// Cost reports — trigger recalculation / backfill (admin only)
				r.Post("/cost-reports/calculate", h.CalculateDailyCostsHandler)
				r.Post("/cost-reports/fulfill", h.FulfillCostReportsHandler)

				// Prometheus config (admin write)
				r.Put("/prometheus/config", h.UpdatePrometheusConfigHandler)

				// Service catalog — write operations
				r.Post("/services", h.CreateServiceHandler)
				r.Put("/services/{id}", h.UpdateServiceHandler)
				r.Delete("/services/{id}", h.DeleteServiceHandler)
				r.Post("/services/{id}/pods", h.AddPodWorkloadHandler)
				r.Put("/services/{id}/pods/{podId}", h.UpdatePodWorkloadHandler)
				r.Delete("/services/{id}/pods/{podId}", h.DeletePodWorkloadHandler)
				r.Post("/services/{id}/vms", h.AddVMWorkloadHandler)
				r.Put("/services/{id}/vms/{vmId}", h.UpdateVMWorkloadHandler)
				r.Delete("/services/{id}/vms/{vmId}", h.DeleteVMWorkloadHandler)
			})

			// Viewer + admin: read-only operations
			r.Group(func(r chi.Router) {
				r.Use(h.RequireRole("admin", "viewer"))

				r.Post("/user/logout", h.LogoutHandler)
				r.Put("/user/{id}/password", h.UpdateUserPasswordHandler)

				r.Get("/datacenters", h.ListDatacenterHandler)
				r.Get("/datacenters/{name}", h.GetDatacenterHandler)
				r.Get("/datacenters-with-hardware", h.GetDatacentersWithHardwareHandler)

				r.Get("/servers", h.ListServersHandler)
				r.Get("/servers/{hostname}", h.GetServerHandler)

				r.Get("/infrastructure-hardwares", h.ListInfrastructureHardwaresHandler)
				r.Get("/infrastructure-hardwares/{id}", h.GetInfrastructureHardwareHandler)

				r.Get("/pricing", h.GetAllPricesHandler)

				// Service catalog
				r.Get("/services", h.ListServicesHandler)
				r.Get("/services/{id}", h.GetServiceHandler)

				// Prometheus config (read)
				r.Get("/prometheus/config", h.GetPrometheusConfigHandler)

				// Cost reports
				r.Get("/cost-reports/monthly", h.GetMonthlyCostReportsHandler)
				r.Get("/cost-reports/trend", h.GetCostTrendHandler)
				r.Get("/cost-reports/spend", h.GetCostSpendHandler)

			})
		})
	})

	

	return r
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "rashnu"})
}

func (h *Handler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		h.logger.Info("request",
			zap.String("Real IP", r.RemoteAddr),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("User Agent", r.Header.Get("User-Agent")),
			zap.Int("status", ww.Status()),
			zap.Duration("duration", time.Since(start)),
		)
	})
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("fileServer does not permit URL parameters")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, req *http.Request) {
		// SPA fallback: serve index.html if the file doesn't exist
		requested := filepath.Join("./frontend/dist", req.URL.Path)
		if _, err := os.Stat(requested); os.IsNotExist(err) && filepath.Ext(req.URL.Path) == "" {
			http.ServeFile(w, req, "./frontend/dist/index.html")
			return
		}
		fs.ServeHTTP(w, req)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
