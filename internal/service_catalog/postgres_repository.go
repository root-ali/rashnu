package service_catalog

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Repository interface {
	CreateService(ctx context.Context, s Service) error
	GetServiceByID(ctx context.Context, id uuid.UUID) (*Service, error)
	ListServices(ctx context.Context) ([]*Service, error)
	UpdateService(ctx context.Context, s *Service) error
	DeleteService(ctx context.Context, id uuid.UUID) error

	AddPodWorkload(ctx context.Context, pod PodSpec) error
	GetPodsByServiceID(ctx context.Context, serviceID uuid.UUID) ([]PodSpec, error)
	UpdatePodWorkload(ctx context.Context, pod PodSpec) error
	DeletePodWorkload(ctx context.Context, serviceID, podID uuid.UUID) error

	AddVMWorkload(ctx context.Context, vm VMSpec) error
	GetVMsByServiceID(ctx context.Context, serviceID uuid.UUID) ([]VMSpec, error)
	UpdateVMWorkload(ctx context.Context, vm VMSpec) error
	DeleteVMWorkload(ctx context.Context, serviceID, vmID uuid.UUID) error
}

type postgresRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgresRepository(pool *pgxpool.Pool, logger *zap.Logger) Repository {
	return &postgresRepository{pool: pool, logger: logger}
}

func (r *postgresRepository) CreateService(ctx context.Context, s Service) error {
	const q = `
		INSERT INTO services (id, name, description, platform, team, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, q, s.ID, s.Name, s.Description, s.Platform, s.Team, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		if isDuplicate(err) {
			return ErrServiceDuplicate
		}
		r.logger.Error("create service", zap.String("name", s.Name), zap.Error(err))
		return fmt.Errorf("create service: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetServiceByID(ctx context.Context, id uuid.UUID) (*Service, error) {
	const q = `
		SELECT id, name, description, platform, team, created_at, updated_at, deleted_at
		FROM services
		WHERE id = $1 AND deleted_at IS NULL`
	var s Service
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.Name, &s.Description, &s.Platform, &s.Team,
		&s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrServiceNotFound
		}
		r.logger.Error("get service by id", zap.String("id", id.String()), zap.Error(err))
		return nil, fmt.Errorf("get service: %w", err)
	}

	pods, err := r.GetPodsByServiceID(ctx, id)
	if err != nil {
		return nil, err
	}
	vms, err := r.GetVMsByServiceID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.Pods = pods
	s.VMs = vms
	return &s, nil
}

func (r *postgresRepository) ListServices(ctx context.Context) ([]*Service, error) {
	const q = `
		SELECT id, name, description, platform, team, created_at, updated_at
		FROM services
		WHERE deleted_at IS NULL
		ORDER BY name`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		r.logger.Error("list services", zap.Error(err))
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (*Service, error) {
		var s Service
		err := row.Scan(&s.ID, &s.Name, &s.Description, &s.Platform, &s.Team, &s.CreatedAt, &s.UpdatedAt)
		return &s, err
	})
}

func (r *postgresRepository) UpdateService(ctx context.Context, s *Service) error {
	const q = `
		UPDATE services
		SET name = $2, description = $3, team = $4, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id`
	err := r.pool.QueryRow(ctx, q, s.ID, s.Name, s.Description, s.Team).Scan(&s.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrServiceNotFound
		}
		r.logger.Error("update service", zap.String("id", s.ID.String()), zap.Error(err))
		return fmt.Errorf("update service: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteService(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE services SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id`
	err := r.pool.QueryRow(ctx, q, id).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrServiceNotFound
		}
		r.logger.Error("delete service", zap.String("id", id.String()), zap.Error(err))
		return fmt.Errorf("delete service: %w", err)
	}
	return nil
}

func (r *postgresRepository) AddPodWorkload(ctx context.Context, pod PodSpec) error {
	const q = `
		INSERT INTO service_pods (id, service_id, name, namespace, vcpu, memory_gb, ssd_gb, hdd_gb, gpus, cpu_usage, memory_usage_gb, datacenter_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	_, err := r.pool.Exec(ctx, q,
		pod.ID, pod.ServiceID, pod.Name, pod.Namespace,
		pod.VCPU, pod.MemoryGB, pod.SSDGB, pod.HDDGB, pod.GPUs,
		pod.CPUUsage, pod.MemoryUsageGB,
		pod.DatacenterID, pod.CreatedAt, pod.UpdatedAt,
	)
	if err != nil {
		r.logger.Error("add pod workload", zap.String("service_id", pod.ServiceID.String()), zap.Error(err))
		return fmt.Errorf("add pod workload: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetPodsByServiceID(ctx context.Context, serviceID uuid.UUID) ([]PodSpec, error) {
	const q = `
		SELECT p.id, p.service_id, p.name, p.namespace, p.vcpu, p.memory_gb, p.ssd_gb, p.hdd_gb, p.gpus,
		       p.cpu_usage, p.memory_usage_gb, p.datacenter_id, d.name, p.created_at, p.updated_at
		FROM service_pods p
		JOIN datacenters d ON d.id = p.datacenter_id
		WHERE p.service_id = $1`
	rows, err := r.pool.Query(ctx, q, serviceID)
	if err != nil {
		r.logger.Error("get pods by service", zap.String("service_id", serviceID.String()), zap.Error(err))
		return nil, fmt.Errorf("get pods: %w", err)
	}
	defer rows.Close()
	pods, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (PodSpec, error) {
		var p PodSpec
		err := row.Scan(
			&p.ID, &p.ServiceID, &p.Name, &p.Namespace,
			&p.VCPU, &p.MemoryGB, &p.SSDGB, &p.HDDGB, &p.GPUs,
			&p.CPUUsage, &p.MemoryUsageGB,
			&p.DatacenterID, &p.DatacenterName, &p.CreatedAt, &p.UpdatedAt,
		)
		return p, err
	})
	if pods == nil {
		pods = []PodSpec{}
	}
	return pods, err
}

func (r *postgresRepository) UpdatePodWorkload(ctx context.Context, pod PodSpec) error {
	const q = `
		UPDATE service_pods
		SET name = $3, namespace = $4, vcpu = $5, memory_gb = $6, ssd_gb = $7, hdd_gb = $8, gpus = $9,
		    cpu_usage = $10, memory_usage_gb = $11, datacenter_id = $12, updated_at = NOW()
		WHERE id = $1 AND service_id = $2
		RETURNING id`
	err := r.pool.QueryRow(ctx, q,
		pod.ID, pod.ServiceID, pod.Name, pod.Namespace,
		pod.VCPU, pod.MemoryGB, pod.SSDGB, pod.HDDGB, pod.GPUs,
		pod.CPUUsage, pod.MemoryUsageGB, pod.DatacenterID,
	).Scan(&pod.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPodNotFound
		}
		r.logger.Error("update pod workload", zap.String("id", pod.ID.String()), zap.Error(err))
		return fmt.Errorf("update pod workload: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeletePodWorkload(ctx context.Context, serviceID, podID uuid.UUID) error {
	const q = `DELETE FROM service_pods WHERE id = $1 AND service_id = $2 RETURNING id`
	err := r.pool.QueryRow(ctx, q, podID, serviceID).Scan(&podID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPodNotFound
		}
		r.logger.Error("delete pod workload", zap.String("id", podID.String()), zap.Error(err))
		return fmt.Errorf("delete pod workload: %w", err)
	}
	return nil
}

func (r *postgresRepository) AddVMWorkload(ctx context.Context, vm VMSpec) error {
	const q = `
		INSERT INTO service_vms (id, service_id, name, vcpu, memory_gb, ssd_gb, hdd_gb, gpus, cpu_usage, memory_usage_gb, datacenter_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err := r.pool.Exec(ctx, q,
		vm.ID, vm.ServiceID, vm.Name,
		vm.VCPU, vm.MemoryGB, vm.SSDGB, vm.HDDGB, vm.GPUs,
		vm.CPUUsage, vm.MemoryUsageGB,
		vm.DatacenterID, vm.CreatedAt, vm.UpdatedAt,
	)
	if err != nil {
		r.logger.Error("add vm workload", zap.String("service_id", vm.ServiceID.String()), zap.Error(err))
		return fmt.Errorf("add vm workload: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetVMsByServiceID(ctx context.Context, serviceID uuid.UUID) ([]VMSpec, error) {
	const q = `
		SELECT v.id, v.service_id, v.name, v.vcpu, v.memory_gb, v.ssd_gb, v.hdd_gb, v.gpus,
		       v.cpu_usage, v.memory_usage_gb, v.datacenter_id, d.name, v.created_at, v.updated_at
		FROM service_vms v
		JOIN datacenters d ON d.id = v.datacenter_id
		WHERE v.service_id = $1`
	rows, err := r.pool.Query(ctx, q, serviceID)
	if err != nil {
		r.logger.Error("get vms by service", zap.String("service_id", serviceID.String()), zap.Error(err))
		return nil, fmt.Errorf("get vms: %w", err)
	}
	defer rows.Close()
	vms, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (VMSpec, error) {
		var v VMSpec
		err := row.Scan(
			&v.ID, &v.ServiceID, &v.Name,
			&v.VCPU, &v.MemoryGB, &v.SSDGB, &v.HDDGB, &v.GPUs,
			&v.CPUUsage, &v.MemoryUsageGB,
			&v.DatacenterID, &v.DatacenterName, &v.CreatedAt, &v.UpdatedAt,
		)
		return v, err
	})
	if vms == nil {
		vms = []VMSpec{}
	}
	return vms, err
}

func (r *postgresRepository) UpdateVMWorkload(ctx context.Context, vm VMSpec) error {
	const q = `
		UPDATE service_vms
		SET name = $3, vcpu = $4, memory_gb = $5, ssd_gb = $6, hdd_gb = $7, gpus = $8,
		    cpu_usage = $9, memory_usage_gb = $10, datacenter_id = $11, updated_at = NOW()
		WHERE id = $1 AND service_id = $2
		RETURNING id`
	err := r.pool.QueryRow(ctx, q,
		vm.ID, vm.ServiceID, vm.Name,
		vm.VCPU, vm.MemoryGB, vm.SSDGB, vm.HDDGB, vm.GPUs,
		vm.CPUUsage, vm.MemoryUsageGB, vm.DatacenterID,
	).Scan(&vm.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrVMNotFound
		}
		r.logger.Error("update vm workload", zap.String("id", vm.ID.String()), zap.Error(err))
		return fmt.Errorf("update vm workload: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteVMWorkload(ctx context.Context, serviceID, vmID uuid.UUID) error {
	const q = `DELETE FROM service_vms WHERE id = $1 AND service_id = $2 RETURNING id`
	err := r.pool.QueryRow(ctx, q, vmID, serviceID).Scan(&vmID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrVMNotFound
		}
		r.logger.Error("delete vm workload", zap.String("id", vmID.String()), zap.Error(err))
		return fmt.Errorf("delete vm workload: %w", err)
	}
	return nil
}

func isDuplicate(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key"))
}
