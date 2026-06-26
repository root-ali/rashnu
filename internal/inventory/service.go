package inventory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	repo   *PostgresRepository
	logger *zap.Logger
}

func NewService(repo *PostgresRepository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) ListDataCenters(ctx context.Context) ([]*DataCenter, error) {
	s.logger.Debug("service: listing datacenters")
	dcs, err := s.repo.GetDatacenters(ctx)
	if err != nil {
		s.logger.Error("service: failed to list datacenters", zap.Error(err))
		return nil, err
	}
	s.logger.Debug("service: successfully listed datacenters", zap.Int("count", len(dcs)))
	return dcs, nil
}

func (s *Service) GetDatacentersByName(ctx context.Context, name string) (*DataCenter, error) {
	s.logger.Debug("service: getting datacenter by name", zap.String("name", name))
	dc, err := s.repo.GetDatacentersByName(ctx, name)
	if err != nil {
		s.logger.Error("service: failed to get datacenter", zap.String("name", name), zap.Error(err))
		return nil, err
	}
	s.logger.Debug("service: successfully retrieved datacenter", zap.String("name", name))
	return dc, nil
}

func (s *Service) CreateDatacenter(ctx context.Context, cdr DatacenterCreateRequest) error {
	s.logger.Debug("service: creating datacenter", zap.String("name", cdr.Name), zap.String("region", cdr.Region))
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.Error("service: failed to generate UUID", zap.Error(err))
		return err
	}
	dc := &DataCenter{
		ID:             id,
		Name:           cdr.Name,
		Short:          cdr.Short,
		Provider:       cdr.Provider,
		Region:         cdr.Region,
		Racks:          cdr.Racks,
		PowerKW:        cdr.PowerKW,
		Tier:           cdr.Tier,
		MonthlyColoFee: cdr.MonthlyColoFee,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err = s.repo.CreateDatacenter(ctx, dc)
	if err != nil {
		s.logger.Error("service: failed to create datacenter", zap.String("name", cdr.Name), zap.Error(err))
		return err
	}
	s.logger.Debug("service: successfully created datacenter", zap.String("name", cdr.Name), zap.String("id", id.String()))
	return nil
}

func (s *Service) UpdateDatacenter(ctx context.Context, datacenter DatacenterUpdateRequest) error {
	s.logger.Debug("service: updating datacenter", zap.String("id", datacenter.ID.String()), zap.String("name", datacenter.Name))
	dc := &DataCenter{
		ID:             datacenter.ID,
		Name:           datacenter.Name,
		Short:          datacenter.Short,
		Provider:       datacenter.Provider,
		Region:         datacenter.Region,
		Racks:          datacenter.Racks,
		PowerKW:        datacenter.PowerKW,
		Tier:           datacenter.Tier,
		MonthlyColoFee: datacenter.MonthlyColoFee,
		UpdatedAt:      time.Now(),
	}
	err := s.repo.UpdateDatacenter(ctx, dc)
	if err != nil {
		s.logger.Error("service: failed to update datacenter", zap.String("id", datacenter.ID.String()), zap.Error(err))
		return err
	}
	s.logger.Debug("service: successfully updated datacenter", zap.String("id", datacenter.ID.String()))
	return nil
}

func (s *Service) DeleteDatacenter(ctx context.Context, id string) error {
	s.logger.Debug("service: deleting datacenter", zap.String("id", id))
	err := s.repo.DeleteDatacenter(ctx, id)
	if err != nil {
		s.logger.Error("service: failed to delete datacenter", zap.String("id", id), zap.Error(err))
		return err
	}
	s.logger.Debug("service: successfully deleted datacenter", zap.String("id", id))
	return nil
}

func (s *Service) ListServers(ctx context.Context) ([]*Server, error) {
	s.logger.Debug("service: listing servers")
	servers, err := s.repo.GetServers(ctx)
	if err != nil {
		s.logger.Error("service: failed to list servers", zap.Error(err))
		return nil, err
	}
	s.logger.Debug("service: successfully listed servers", zap.Int("count", len(servers)))
	return servers, nil
}

func (s *Service) GetServerByHostname(ctx context.Context, hostname string) (*Server, error) {
	s.logger.Debug("service: getting server by hostname", zap.String("hostname", hostname))
	server, err := s.repo.GetServerByHostname(ctx, hostname)
	if err != nil {
		s.logger.Error("service: failed to get server", zap.String("hostname", hostname), zap.Error(err))
		return nil, err
	}
	s.logger.Debug("service: successfully retrieved server", zap.String("hostname", hostname))
	return server, nil
}

func (s *Service) CreateServer(ctx context.Context, scr *ServerCreateRequest) error {
	s.logger.Debug("service: creating server", zap.String("hostname", scr.Hostname), zap.String("region", scr.Region))
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.Error("service: failed to generate UUID", zap.Error(err))
		return err
	}
	server := &Server{
		ID:            id,
		DataCenterID:  scr.DataCenterID,
		Hostname:      scr.Hostname,
		Region:        scr.Region,
		Vendor:        scr.Vendor,
		Model:         scr.Model,
		PurchaseDate:  scr.PurchaseDate.Time,
		PurchasePrice: scr.PurchasePrice,
		Role:          scr.Role,
		VCPUs:         scr.VCPUs,
		MemoryGB:      scr.MemoryGB,
		SSDCapacityGB: scr.SSDCapacityGB,
		HDDCapacityGB: scr.HDDCapacityGB,
		GPUs:          scr.GPUs,
		PowerWatts:    scr.PowerWatts,
		RackUnits:     scr.RackUnits,
		Status:        StatusActive,
		AmortMonths:   scr.AmortMonths,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err = s.repo.CreateServer(ctx, server)
	if err != nil {
		s.logger.Error("service: failed to create server", zap.String("hostname", scr.Hostname), zap.Error(err))
		return err
	}
	s.logger.Debug("service: successfully created server", zap.String("hostname", scr.Hostname), zap.String("id", id.String()))
	return nil
}

func (s *Service) UpdateServer(ctx context.Context, sur *ServerUpdateRequest) error {
	s.logger.Debug("service: updating server", zap.String("id", sur.ID.String()), zap.String("hostname", sur.Hostname))
	server := &Server{
		ID:            sur.ID,
		DataCenterID:  sur.DataCenterID,
		Hostname:      sur.Hostname,
		Region:        sur.Region,
		Vendor:        sur.Vendor,
		Model:         sur.Model,
		PurchaseDate:  sur.PurchaseDate.Time,
		PurchasePrice: sur.PurchasePrice,
		Role:          sur.Role,
		VCPUs:         sur.VCPUs,
		MemoryGB:      sur.MemoryGB,
		SSDCapacityGB: sur.SSDCapacityGB,
		HDDCapacityGB: sur.HDDCapacityGB,
		GPUs:          sur.GPUs,
		PowerWatts:    sur.PowerWatts,
		RackUnits:     sur.RackUnits,
		Status:        sur.Status,
		AmortMonths:   sur.AmortMonths,
		UpdatedAt:     time.Now(),
	}
	err := s.repo.UpdateServer(ctx, server)
	if err != nil {
		s.logger.Error("service: failed to update server", zap.String("id", sur.ID.String()), zap.Error(err))
		return err
	}
	s.logger.Debug("service: successfully updated server", zap.String("id", sur.ID.String()))
	return nil
}

func (s *Service) DeleteServer(ctx context.Context, id string) error {
	s.logger.Debug("service: deleting server", zap.String("id", id))
	err := s.repo.DeleteServer(ctx, id)
	if err != nil {
		s.logger.Error("service: failed to delete server", zap.String("id", id), zap.Error(err))
		return err
	}
	s.logger.Debug("service: successfully deleted server", zap.String("id", id))
	return nil
}

// Infrastructure Hardware Service Methods

func (s *Service) ListInfrastructureHardwares(ctx context.Context) ([]*InfrastructureHardware, error) {
	s.logger.Debug("service: listing infrastructure hardwares")
	hardwares, err := s.repo.GetInfrastructureHardwares(ctx)
	if err != nil {
		s.logger.Error("service: failed to list infrastructure hardwares", zap.Error(err))
		return nil, err
	}
	s.logger.Debug("service: successfully listed infrastructure hardwares", zap.Int("count", len(hardwares)))
	return hardwares, nil
}

func (s *Service) GetInfrastructureHardwareByID(ctx context.Context, id string) (*InfrastructureHardware, error) {
	s.logger.Debug("service: getting infrastructure hardware by ID", zap.String("id", id))
	hardware, err := s.repo.GetInfrastructureHardwareByID(ctx, id)
	if err != nil {
		s.logger.Error("service: failed to get infrastructure hardware", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	s.logger.Debug("service: successfully retrieved infrastructure hardware", zap.String("id", id))
	return hardware, nil
}

func (s *Service) CreateInfrastructureHardware(ctx context.Context, req *InfrastructureHardwareCreateRequest) error {
	s.logger.Debug("service: creating infrastructure hardware", zap.String("hostname", req.Hostname), zap.String("hardware_type", req.HardwareType))
	id, err := uuid.NewV7()
	if err != nil {
		s.logger.Error("service: failed to generate UUID", zap.Error(err))
		return err
	}
	hw := &InfrastructureHardware{
		ID:            id,
		DataCenterID:  req.DataCenterID,
		Hostname:      req.Hostname,
		HardwareType:  req.HardwareType,
		Region:        req.Region,
		PurchasePrice: req.PurchasePrice,
		PurchaseDate:  req.PurchaseDate.Time,
		AmortMonths:   req.AmortMonths,
		Model:         req.Model,
		Specs:         req.Specs,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err = s.repo.CreateInfrastructureHardware(ctx, hw)
	if err != nil {
		s.logger.Error("service: failed to create infrastructure hardware", zap.String("hostname", req.Hostname), zap.Error(err))
		return err
	}
	s.logger.Debug("service: successfully created infrastructure hardware", zap.String("hostname", req.Hostname), zap.String("id", id.String()))
	return nil
}

func (s *Service) UpdateInfrastructureHardware(ctx context.Context, req *InfrastructureHardwareUpdateRequest) error {
	s.logger.Debug("service: updating infrastructure hardware", zap.String("id", req.ID.String()), zap.String("hostname", req.Hostname))
	hw := &InfrastructureHardware{
		ID:            req.ID,
		DataCenterID:  req.DataCenterID,
		Hostname:      req.Hostname,
		HardwareType:  req.HardwareType,
		Region:        req.Region,
		PurchasePrice: req.PurchasePrice,
		PurchaseDate:  req.PurchaseDate.Time,
		AmortMonths:   req.AmortMonths,
		Model:         req.Model,
		Specs:         req.Specs,
		UpdatedAt:     time.Now(),
	}
	err := s.repo.UpdateInfrastructureHardware(ctx, hw)
	if err != nil {
		s.logger.Error("service: failed to update infrastructure hardware", zap.String("id", req.ID.String()), zap.Error(err))
		return err
	}
	s.logger.Debug("service: successfully updated infrastructure hardware", zap.String("id", req.ID.String()))
	return nil
}

func (s *Service) DeleteInfrastructureHardware(ctx context.Context, id string) error {
	s.logger.Debug("service: deleting infrastructure hardware", zap.String("id", id))
	err := s.repo.DeleteInfrastructureHardware(ctx, id)
	if err != nil {
		s.logger.Error("service: failed to delete infrastructure hardware", zap.String("id", id), zap.Error(err))
		return err
	}
	s.logger.Debug("service: successfully deleted infrastructure hardware", zap.String("id", id))
	return nil
}

// GetDatacentersWithHardware returns all datacenters with their associated servers and infrastructure hardware
func (s *Service) GetDatacentersWithHardware(ctx context.Context, nameFilter, regionFilter string) ([]*DatacenterWithHardware, error) {
	s.logger.Debug("service: getting datacenters with hardware",
		zap.String("name_filter", nameFilter),
		zap.String("region_filter", regionFilter))

	datacentersWithHW, err := s.repo.GetDatacentersWithHardware(ctx, nameFilter, regionFilter)
	if err != nil {
		s.logger.Error("service: failed to get datacenters with hardware", zap.Error(err))
		return nil, err
	}

	s.logger.Debug("service: successfully retrieved datacenters with hardware", zap.Int("count", len(datacentersWithHW)))
	return datacentersWithHW, nil
}
