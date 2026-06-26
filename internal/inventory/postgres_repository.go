package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PostgresRepository implements the Repository interface using PostgreSQL
type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresRepository creates a new PostgresRepository
func NewPostgresRepository(pool *pgxpool.Pool, logger *zap.Logger) *PostgresRepository {
	return &PostgresRepository{
		pool:   pool,
		logger: logger,
	}
}

const dcColumns = `id, name, short, provider, region, racks, power_kw, tier, monthly_colo_fee, created_at, updated_at`

func scanDC(row pgx.CollectableRow) (*DataCenter, error) {
	dc := new(DataCenter)
	return dc, row.Scan(
		&dc.ID, &dc.Name, &dc.Short, &dc.Provider, &dc.Region,
		&dc.Racks, &dc.PowerKW, &dc.Tier, &dc.MonthlyColoFee,
		&dc.CreatedAt, &dc.UpdatedAt,
	)
}

func scanDCRow(row pgx.Row) (*DataCenter, error) {
	dc := new(DataCenter)
	return dc, row.Scan(
		&dc.ID, &dc.Name, &dc.Short, &dc.Provider, &dc.Region,
		&dc.Racks, &dc.PowerKW, &dc.Tier, &dc.MonthlyColoFee,
		&dc.CreatedAt, &dc.UpdatedAt,
	)
}

func (r *PostgresRepository) GetDatacenters(ctx context.Context) ([]*DataCenter, error) {
	r.logger.Debug("getting all datacenters from database")
	rows, err := r.pool.Query(ctx, "SELECT "+dcColumns+" FROM datacenters")
	if err != nil {
		r.logger.Error("failed to query datacenters", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	dcs, err := pgx.CollectRows(rows, scanDC)
	if err != nil {
		r.logger.Error("failed to scan datacenters", zap.Error(err))
		return nil, err
	}
	r.logger.Debug("successfully retrieved datacenters", zap.Int("count", len(dcs)))
	return dcs, nil
}

func (r *PostgresRepository) GetDatacentersByName(ctx context.Context, name string) (*DataCenter, error) {
	r.logger.Debug("getting datacenter by name", zap.String("name", name))
	rows, err := r.pool.Query(ctx, "SELECT "+dcColumns+" FROM datacenters WHERE name=$1", name)
	if err != nil {
		r.logger.Error("failed to query datacenter by name", zap.String("name", name), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	dcs, err := pgx.CollectRows(rows, scanDC)
	if err != nil {
		r.logger.Error("failed to scan datacenter", zap.String("name", name), zap.Error(err))
		return nil, err
	}
	if len(dcs) == 0 {
		r.logger.Debug("datacenter not found", zap.String("name", name))
		return nil, ErrDatacenterNotFound
	}
	r.logger.Debug("successfully retrieved datacenter", zap.String("name", name))
	return dcs[0], nil
}

func (r *PostgresRepository) CreateDatacenter(ctx context.Context, dc *DataCenter) error {
	r.logger.Debug("creating datacenter", zap.String("name", dc.Name))
	_, err := r.pool.Exec(ctx, `
		INSERT INTO datacenters (id, name, short, provider, region, racks, power_kw, tier, monthly_colo_fee, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, dc.ID, dc.Name, dc.Short, dc.Provider, dc.Region,
		dc.Racks, dc.PowerKW, dc.Tier, dc.MonthlyColoFee,
		dc.CreatedAt, dc.UpdatedAt)
	if err != nil {
		if isDuplicateKeyError(err) {
			r.logger.Debug("duplicate datacenter detected", zap.String("name", dc.Name))
			return ErrDuplicateDatacenter
		}
		r.logger.Error("failed to create datacenter", zap.String("name", dc.Name), zap.Error(err))
		return err
	}
	r.logger.Debug("successfully created datacenter", zap.String("name", dc.Name))
	return nil
}

func (r *PostgresRepository) UpdateDatacenter(ctx context.Context, dc *DataCenter) error {
	r.logger.Debug("updating datacenter", zap.String("id", dc.ID.String()))
	result, err := r.pool.Exec(ctx, `
		UPDATE datacenters
		SET name=$2, short=$3, provider=$4, region=$5, racks=$6, power_kw=$7, tier=$8, monthly_colo_fee=$9, updated_at=$10
		WHERE id=$1`,
		dc.ID, dc.Name, dc.Short, dc.Provider, dc.Region,
		dc.Racks, dc.PowerKW, dc.Tier, dc.MonthlyColoFee,
		dc.UpdatedAt)
	if err != nil {
		r.logger.Error("failed to update datacenter", zap.String("id", dc.ID.String()), zap.Error(err))
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrDatacenterNotFound
	}
	r.logger.Debug("successfully updated datacenter", zap.String("id", dc.ID.String()))
	return nil
}

func (r *PostgresRepository) DeleteDatacenter(ctx context.Context, id string) error {
	r.logger.Debug("deleting datacenter", zap.String("id", id))
	result, err := r.pool.Exec(ctx, `DELETE FROM datacenters WHERE id=$1`, id)
	if err != nil {
		r.logger.Error("failed to delete datacenter", zap.String("id", id), zap.Error(err))
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrDatacenterNotFound
	}
	return nil
}

const serverColumns = `id, datacenter_id, hostname, region, vendor, model, purchase_price, purchase_date,
	amort_months, vcpus, memory_gb, ssd_capacity_gb, hdd_capacity_gb, gpus, power_watts, rack_units,
	role, status, created_at, updated_at`

func scanServer(row pgx.CollectableRow) (*Server, error) {
	var s Server
	return &s, row.Scan(
		&s.ID, &s.DataCenterID, &s.Hostname, &s.Region,
		&s.Vendor, &s.Model,
		&s.PurchasePrice, &s.PurchaseDate, &s.AmortMonths,
		&s.VCPUs, &s.MemoryGB, &s.SSDCapacityGB, &s.HDDCapacityGB, &s.GPUs,
		&s.PowerWatts, &s.RackUnits,
		&s.Role, &s.Status, &s.CreatedAt, &s.UpdatedAt,
	)
}

// GetServers retrieves all servers from the database.
func (r *PostgresRepository) GetServers(ctx context.Context) ([]*Server, error) {
	r.logger.Debug("getting all servers from database")
	rows, err := r.pool.Query(ctx, "SELECT "+serverColumns+" FROM servers")
	if err != nil {
		r.logger.Error("failed to query servers", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	servers, err := pgx.CollectRows(rows, scanServer)
	if err != nil {
		r.logger.Error("failed to scan servers", zap.Error(err))
		return nil, err
	}
	r.logger.Debug("successfully retrieved servers", zap.Int("count", len(servers)))
	return servers, nil
}

func (r *PostgresRepository) GetServerByHostname(ctx context.Context, hostname string) (*Server, error) {
	r.logger.Debug("getting server by hostname", zap.String("hostname", hostname))
	rows, err := r.pool.Query(ctx, "SELECT "+serverColumns+" FROM servers WHERE hostname=$1", hostname)
	if err != nil {
		r.logger.Error("failed to query server by hostname", zap.String("hostname", hostname), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	servers, err := pgx.CollectRows(rows, scanServer)
	if err != nil {
		r.logger.Error("failed to scan server", zap.String("hostname", hostname), zap.Error(err))
		return nil, err
	}
	if len(servers) == 0 {
		return nil, ErrServerNotFound
	}
	return servers[0], nil
}

func (r *PostgresRepository) CreateServer(ctx context.Context, s *Server) error {
	r.logger.Debug("creating server", zap.String("hostname", s.Hostname))
	_, err := r.pool.Exec(ctx, `
		INSERT INTO servers (id, datacenter_id, hostname, region, vendor, model,
			purchase_price, purchase_date, amort_months,
			vcpus, memory_gb, ssd_capacity_gb, hdd_capacity_gb, gpus,
			power_watts, rack_units, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, s.ID, s.DataCenterID, s.Hostname, s.Region, s.Vendor, s.Model,
		s.PurchasePrice, s.PurchaseDate, s.AmortMonths,
		s.VCPUs, s.MemoryGB, s.SSDCapacityGB, s.HDDCapacityGB, s.GPUs,
		s.PowerWatts, s.RackUnits, s.Role, s.Status,
		s.CreatedAt, s.UpdatedAt)
	if err != nil {
		if isDuplicateKeyError(err) {
			r.logger.Debug("duplicate server detected", zap.String("hostname", s.Hostname))
			return ErrDuplicateServer
		}
		r.logger.Error("failed to create server", zap.String("hostname", s.Hostname), zap.Error(err))
		return err
	}
	r.logger.Debug("successfully created server", zap.String("hostname", s.Hostname))
	return nil
}

func (r *PostgresRepository) UpdateServer(ctx context.Context, s *Server) error {
	r.logger.Debug("updating server", zap.String("id", s.ID.String()))
	result, err := r.pool.Exec(ctx, `
		UPDATE servers SET
			datacenter_id=$2, hostname=$3, region=$4,
			vendor=$5, model=$6,
			purchase_price=$7, purchase_date=$8, amort_months=$9,
			vcpus=$10, memory_gb=$11, ssd_capacity_gb=$12, hdd_capacity_gb=$13, gpus=$14,
			power_watts=$15, rack_units=$16, role=$17, status=$18, updated_at=$19
		WHERE id=$1`,
		s.ID, s.DataCenterID, s.Hostname, s.Region,
		s.Vendor, s.Model,
		s.PurchasePrice, s.PurchaseDate, s.AmortMonths,
		s.VCPUs, s.MemoryGB, s.SSDCapacityGB, s.HDDCapacityGB, s.GPUs,
		s.PowerWatts, s.RackUnits, s.Role, s.Status, time.Now())
	if err != nil {
		r.logger.Error("failed to update server", zap.String("id", s.ID.String()), zap.Error(err))
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrServerNotFound
	}
	r.logger.Debug("successfully updated server", zap.String("id", s.ID.String()))
	return nil
}

func (r *PostgresRepository) DeleteServer(ctx context.Context, id string) error {
	r.logger.Debug("deleting server", zap.String("id", id))
	result, err := r.pool.Exec(ctx, `DELETE FROM servers WHERE id=$1`, id)
	if err != nil {
		r.logger.Error("failed to delete server", zap.String("id", id), zap.Error(err))
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrServerNotFound
	}
	return nil
}

const hwColumns = `id, datacenter_id, hostname, hardware_type, purchase_price, purchase_date,
	region, model, specs, amort_months, created_at, updated_at`

func scanHW(row pgx.CollectableRow) (*InfrastructureHardware, error) {
	var hw InfrastructureHardware
	return &hw, row.Scan(
		&hw.ID, &hw.DataCenterID, &hw.Hostname, &hw.HardwareType,
		&hw.PurchasePrice, &hw.PurchaseDate,
		&hw.Region, &hw.Model, &hw.Specs, &hw.AmortMonths,
		&hw.CreatedAt, &hw.UpdatedAt,
	)
}

func (r *PostgresRepository) GetInfrastructureHardwares(ctx context.Context) ([]*InfrastructureHardware, error) {
	r.logger.Debug("getting all infrastructure hardwares from database")
	rows, err := r.pool.Query(ctx, "SELECT "+hwColumns+" FROM infrastructure_hardwares")
	if err != nil {
		r.logger.Error("failed to query infrastructure hardwares", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	hws, err := pgx.CollectRows(rows, scanHW)
	if err != nil {
		r.logger.Error("failed to scan infrastructure hardwares", zap.Error(err))
		return nil, err
	}
	r.logger.Debug("successfully retrieved infrastructure hardwares", zap.Int("count", len(hws)))
	return hws, nil
}

func (r *PostgresRepository) GetInfrastructureHardwareByID(ctx context.Context, id string) (*InfrastructureHardware, error) {
	r.logger.Debug("getting infrastructure hardware by ID", zap.String("id", id))
	rows, err := r.pool.Query(ctx, "SELECT "+hwColumns+" FROM infrastructure_hardwares WHERE id=$1", id)
	if err != nil {
		r.logger.Error("failed to query infrastructure hardware by ID", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	hws, err := pgx.CollectRows(rows, scanHW)
	if err != nil {
		r.logger.Error("failed to scan infrastructure hardware", zap.String("id", id), zap.Error(err))
		return nil, err
	}
	if len(hws) == 0 {
		return nil, ErrInfrastructureHardwareNotFound
	}
	return hws[0], nil
}

func (r *PostgresRepository) CreateInfrastructureHardware(ctx context.Context, hw *InfrastructureHardware) error {
	r.logger.Debug("creating infrastructure hardware", zap.String("hostname", hw.Hostname))
	_, err := r.pool.Exec(ctx, `
		INSERT INTO infrastructure_hardwares
			(id, datacenter_id, hostname, hardware_type, purchase_price, purchase_date,
			 region, model, specs, amort_months, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		hw.ID, hw.DataCenterID, hw.Hostname, hw.HardwareType,
		hw.PurchasePrice, hw.PurchaseDate,
		hw.Region, hw.Model, hw.Specs, hw.AmortMonths,
		hw.CreatedAt, hw.UpdatedAt)
	if err != nil {
		if isDuplicateKeyError(err) {
			r.logger.Debug("duplicate infrastructure hardware detected", zap.String("hostname", hw.Hostname))
			return ErrDuplicateInfrastructureHardware
		}
		r.logger.Error("failed to create infrastructure hardware", zap.String("hostname", hw.Hostname), zap.Error(err))
		return err
	}
	r.logger.Debug("successfully created infrastructure hardware", zap.String("hostname", hw.Hostname))
	return nil
}

func (r *PostgresRepository) UpdateInfrastructureHardware(ctx context.Context, hw *InfrastructureHardware) error {
	r.logger.Debug("updating infrastructure hardware", zap.String("id", hw.ID.String()))
	result, err := r.pool.Exec(ctx, `
		UPDATE infrastructure_hardwares SET
			datacenter_id=$2, hostname=$3, hardware_type=$4,
			purchase_price=$5, purchase_date=$6, region=$7,
			model=$8, specs=$9, amort_months=$10, updated_at=$11
		WHERE id=$1`,
		hw.ID, hw.DataCenterID, hw.Hostname, hw.HardwareType,
		hw.PurchasePrice, hw.PurchaseDate, hw.Region,
		hw.Model, hw.Specs, hw.AmortMonths, time.Now())
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateInfrastructureHardware
		}
		r.logger.Error("failed to update infrastructure hardware", zap.String("id", hw.ID.String()), zap.Error(err))
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInfrastructureHardwareNotFound
	}
	r.logger.Debug("successfully updated infrastructure hardware", zap.String("id", hw.ID.String()))
	return nil
}

func (r *PostgresRepository) DeleteInfrastructureHardware(ctx context.Context, id string) error {
	r.logger.Debug("deleting infrastructure hardware", zap.String("id", id))
	result, err := r.pool.Exec(ctx, `DELETE FROM infrastructure_hardwares WHERE id=$1`, id)
	if err != nil {
		r.logger.Error("failed to delete infrastructure hardware", zap.String("id", id), zap.Error(err))
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInfrastructureHardwareNotFound
	}
	return nil
}

func (r *PostgresRepository) GetDatacentersWithHardware(ctx context.Context, nameFilter, regionFilter string) ([]*DatacenterWithHardware, error) {
	r.logger.Debug("getting datacenters with hardware",
		zap.String("name_filter", nameFilter),
		zap.String("region_filter", regionFilter))

	query := "SELECT " + dcColumns + " FROM datacenters WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if nameFilter != "" {
		query += fmt.Sprintf(" AND name = $%d", argPos)
		args = append(args, nameFilter)
		argPos++
	}
	if regionFilter != "" {
		query += fmt.Sprintf(" AND region = $%d", argPos)
		args = append(args, regionFilter)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query datacenters", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	datacenters, err := pgx.CollectRows(rows, scanDC)
	if err != nil {
		r.logger.Error("failed to scan datacenters", zap.Error(err))
		return nil, err
	}

	result := make([]*DatacenterWithHardware, 0, len(datacenters))
	for _, dc := range datacenters {
		serverRows, err := r.pool.Query(ctx,
			"SELECT "+serverColumns+" FROM servers WHERE datacenter_id = $1", dc.ID)
		if err != nil {
			return nil, err
		}
		servers, err := pgx.CollectRows(serverRows, scanServer)
		serverRows.Close()
		if err != nil {
			return nil, err
		}

		hwRows, err := r.pool.Query(ctx,
			"SELECT "+hwColumns+" FROM infrastructure_hardwares WHERE datacenter_id = $1", dc.ID)
		if err != nil {
			return nil, err
		}
		hardwares, err := pgx.CollectRows(hwRows, scanHW)
		hwRows.Close()
		if err != nil {
			return nil, err
		}

		result = append(result, &DatacenterWithHardware{
			DataCenter:              dc,
			Servers:                 servers,
			InfrastructureHardwares: hardwares,
		})
	}

	r.logger.Debug("successfully retrieved datacenters with hardware", zap.Int("count", len(result)))
	return result, nil
}

func isDuplicateKeyError(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == "23505" || strings.Contains(pgErr.Message, "duplicate key")
	}
	return false
}
