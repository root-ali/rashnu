package service_catalog

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Catalog interface {
	CreateService(ctx context.Context, req CreateServiceRequest) (uuid.UUID, error)
	GetService(ctx context.Context, id string) (*Service, error)
	ListServices(ctx context.Context) ([]*Service, error)
	UpdateService(ctx context.Context, req UpdateServiceRequest) error
	DeleteService(ctx context.Context, id string) error

	AddPodWorkload(ctx context.Context, req AddPodWorkloadRequest) error
	UpdatePodWorkload(ctx context.Context, req UpdatePodWorkloadRequest) error
	DeletePodWorkload(ctx context.Context, serviceID, podID string) error

	AddVMWorkload(ctx context.Context, req AddVMWorkloadRequest) error
	UpdateVMWorkload(ctx context.Context, req UpdateVMWorkloadRequest) error
	DeleteVMWorkload(ctx context.Context, serviceID, vmID string) error

	GetPrometheusConfig(ctx context.Context) (*PrometheusConfig, error)
	UpdatePrometheusConfig(ctx context.Context, req UpdatePrometheusConfigRequest) error
}

type catalog struct {
	repo           Repository
	promConfigRepo PrometheusConfigRepository
	promRepo       PrometheusRepository
	logger         *zap.Logger
}

func NewCatalog(repo Repository, promConfigRepo PrometheusConfigRepository, promRepo PrometheusRepository, logger *zap.Logger) Catalog {
	return &catalog{
		repo:           repo,
		promConfigRepo: promConfigRepo,
		promRepo:       promRepo,
		logger:         logger,
	}
}

func (c *catalog) CreateService(ctx context.Context, req CreateServiceRequest) (uuid.UUID, error) {
	if req.Platform != PlatformKubernetes && req.Platform != PlatformVM {
		return uuid.UUID{}, ErrInvalidPlatform
	}
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.UUID{}, err
	}
	now := time.Now()
	s := Service{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Platform:    req.Platform,
		Team:        req.Team,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	c.logger.Debug("creating service", zap.String("name", s.Name), zap.String("platform", s.Platform))
	return id, c.repo.CreateService(ctx, s)
}

func (c *catalog) GetService(ctx context.Context, id string) (*Service, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrServiceNotFound
	}
	s, err := c.repo.GetServiceByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	s.Resources = computeResources(s.Pods, s.VMs)
	c.enrichWithPrometheus(ctx, s)
	return s, nil
}

// enrichWithPrometheus attaches live pod metrics when Prometheus is enabled.
// Errors are logged and silently ignored — stored values remain as fallback.
func (c *catalog) enrichWithPrometheus(ctx context.Context, s *Service) {
	if s.Platform != PlatformKubernetes || len(s.Pods) == 0 {
		return
	}
	cfg, err := c.promConfigRepo.Get(ctx)
	if err != nil || !cfg.Enabled || cfg.URL == "" {
		return
	}
	namespace := s.Pods[0].Namespace
	if namespace == "" {
		return
	}
	metrics, err := c.promRepo.GetPodMetrics(ctx, cfg.URL, namespace, s.Name)
	if err != nil {
		c.logger.Warn("prometheus metrics unavailable, using stored values",
			zap.String("service", s.Name), zap.Error(err))
		return
	}
	s.LivePods = metrics
}

func (c *catalog) GetPrometheusConfig(ctx context.Context) (*PrometheusConfig, error) {
	return c.promConfigRepo.Get(ctx)
}

func (c *catalog) UpdatePrometheusConfig(ctx context.Context, req UpdatePrometheusConfigRequest) error {
	return c.promConfigRepo.Upsert(ctx, PrometheusConfig{URL: req.URL, Enabled: req.Enabled})
}

func (c *catalog) ListServices(ctx context.Context) ([]*Service, error) {
	services, err := c.repo.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range services {
		pods, err := c.repo.GetPodsByServiceID(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		vms, err := c.repo.GetVMsByServiceID(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		s.Pods = pods
		s.VMs = vms
		s.Resources = computeResources(pods, vms)
	}
	return services, nil
}

func (c *catalog) UpdateService(ctx context.Context, req UpdateServiceRequest) error {
	s := &Service{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Team:        req.Team,
	}
	c.logger.Debug("updating service", zap.String("id", req.ID.String()))
	return c.repo.UpdateService(ctx, s)
}

func (c *catalog) DeleteService(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return ErrServiceNotFound
	}
	c.logger.Debug("deleting service", zap.String("id", id))
	return c.repo.DeleteService(ctx, uid)
}

// ── Pod workloads ─────────────────────────────────────────────────────────────

func (c *catalog) AddPodWorkload(ctx context.Context, req AddPodWorkloadRequest) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	now := time.Now()
	pod := PodSpec{
		ID:            id,
		ServiceID:     req.ServiceID,
		Name:          req.Name,
		Namespace:     req.Namespace,
		VCPU:          req.VCPU,
		MemoryGB:      req.MemoryGB,
		SSDGB:         req.SSDGB,
		HDDGB:         req.HDDGB,
		GPUs:          req.GPUs,
		CPUUsage:      req.CPUUsage,
		MemoryUsageGB: req.MemoryUsageGB,
		DatacenterID:  req.DatacenterID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	c.logger.Debug("adding pod workload", zap.String("service_id", req.ServiceID.String()))
	return c.repo.AddPodWorkload(ctx, pod)
}

func (c *catalog) UpdatePodWorkload(ctx context.Context, req UpdatePodWorkloadRequest) error {
	pod := PodSpec{
		ID:            req.ID,
		ServiceID:     req.ServiceID,
		Name:          req.Name,
		Namespace:     req.Namespace,
		VCPU:          req.VCPU,
		MemoryGB:      req.MemoryGB,
		SSDGB:         req.SSDGB,
		HDDGB:         req.HDDGB,
		GPUs:          req.GPUs,
		CPUUsage:      req.CPUUsage,
		MemoryUsageGB: req.MemoryUsageGB,
		DatacenterID:  req.DatacenterID,
	}
	c.logger.Debug("updating pod workload", zap.String("id", req.ID.String()))
	return c.repo.UpdatePodWorkload(ctx, pod)
}

func (c *catalog) DeletePodWorkload(ctx context.Context, serviceID, podID string) error {
	sUID, err := uuid.Parse(serviceID)
	if err != nil {
		return ErrServiceNotFound
	}
	pUID, err := uuid.Parse(podID)
	if err != nil {
		return ErrPodNotFound
	}
	c.logger.Debug("deleting pod workload", zap.String("pod_id", podID))
	return c.repo.DeletePodWorkload(ctx, sUID, pUID)
}

// ── VM workloads ──────────────────────────────────────────────────────────────

func (c *catalog) AddVMWorkload(ctx context.Context, req AddVMWorkloadRequest) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	now := time.Now()
	vm := VMSpec{
		ID:            id,
		ServiceID:     req.ServiceID,
		Name:          req.Name,
		VCPU:          req.VCPU,
		MemoryGB:      req.MemoryGB,
		SSDGB:         req.SSDGB,
		HDDGB:         req.HDDGB,
		GPUs:          req.GPUs,
		CPUUsage:      req.CPUUsage,
		MemoryUsageGB: req.MemoryUsageGB,
		DatacenterID:  req.DatacenterID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	c.logger.Debug("adding vm workload", zap.String("service_id", req.ServiceID.String()))
	return c.repo.AddVMWorkload(ctx, vm)
}

func (c *catalog) UpdateVMWorkload(ctx context.Context, req UpdateVMWorkloadRequest) error {
	vm := VMSpec{
		ID:            req.ID,
		ServiceID:     req.ServiceID,
		Name:          req.Name,
		VCPU:          req.VCPU,
		MemoryGB:      req.MemoryGB,
		SSDGB:         req.SSDGB,
		HDDGB:         req.HDDGB,
		GPUs:          req.GPUs,
		CPUUsage:      req.CPUUsage,
		MemoryUsageGB: req.MemoryUsageGB,
		DatacenterID:  req.DatacenterID,
	}
	c.logger.Debug("updating vm workload", zap.String("id", req.ID.String()))
	return c.repo.UpdateVMWorkload(ctx, vm)
}

func (c *catalog) DeleteVMWorkload(ctx context.Context, serviceID, vmID string) error {
	sUID, err := uuid.Parse(serviceID)
	if err != nil {
		return ErrServiceNotFound
	}
	vUID, err := uuid.Parse(vmID)
	if err != nil {
		return ErrVMNotFound
	}
	c.logger.Debug("deleting vm workload", zap.String("vm_id", vmID))
	return c.repo.DeleteVMWorkload(ctx, sUID, vUID)
}
