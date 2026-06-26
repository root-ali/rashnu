package cost_report

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// dcBreakdownRecord is the internal representation saved to service_cost_reports_by_dc.
type dcBreakdownRecord struct {
	id             uuid.UUID
	serviceID      uuid.UUID
	datacenterID   uuid.UUID
	datacenterName string
	workloadCount  int
	date           time.Time
	computeCost    int64
	cpuCost        int64
	ramCost        int64
	ssdCost        int64
	hddCost        int64
}

// PostgresRepository persists and reads cost reports.
type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgresRepository(pool *pgxpool.Pool, logger *zap.Logger) *PostgresRepository {
	return &PostgresRepository{pool: pool, logger: logger}
}

func (r *PostgresRepository) saveDailyReports(ctx context.Context, reports []*DailyReport) error {
	const q = `
		INSERT INTO service_cost_reports
			(id, service_id, service_name, team, platform, workload_count, date,
			 compute_cost, cpu_cost, ram_cost, ssd_cost, hdd_cost)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (service_id, date) DO UPDATE SET
			service_name   = EXCLUDED.service_name,
			team           = EXCLUDED.team,
			workload_count = EXCLUDED.workload_count,
			compute_cost   = EXCLUDED.compute_cost,
			cpu_cost       = EXCLUDED.cpu_cost,
			ram_cost       = EXCLUDED.ram_cost,
			ssd_cost       = EXCLUDED.ssd_cost,
			hdd_cost       = EXCLUDED.hdd_cost`

	batch := &pgx.Batch{}
	for _, rep := range reports {
		batch.Queue(q,
			rep.ID, rep.ServiceID, rep.ServiceName, rep.Team,
			rep.Platform, rep.WorkloadCount, rep.Date, rep.ComputeCost,
			rep.CPUCost, rep.RAMCost, rep.SSDCost, rep.HDDCost,
		)
	}
	return r.execBatch(ctx, batch, len(reports), "save daily report")
}

func (r *PostgresRepository) saveDcBreakdowns(ctx context.Context, recs []*dcBreakdownRecord) error {
	const q = `
		INSERT INTO service_cost_reports_by_dc
			(id, service_id, datacenter_id, datacenter_name, workload_count, date,
			 compute_cost, cpu_cost, ram_cost, ssd_cost, hdd_cost)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (service_id, datacenter_id, date) DO UPDATE SET
			datacenter_name = EXCLUDED.datacenter_name,
			workload_count  = EXCLUDED.workload_count,
			compute_cost    = EXCLUDED.compute_cost,
			cpu_cost        = EXCLUDED.cpu_cost,
			ram_cost        = EXCLUDED.ram_cost,
			ssd_cost        = EXCLUDED.ssd_cost,
			hdd_cost        = EXCLUDED.hdd_cost`

	batch := &pgx.Batch{}
	for _, rec := range recs {
		batch.Queue(q,
			rec.id, rec.serviceID, rec.datacenterID, rec.datacenterName,
			rec.workloadCount, rec.date, rec.computeCost,
			rec.cpuCost, rec.ramCost, rec.ssdCost, rec.hddCost,
		)
	}
	return r.execBatch(ctx, batch, len(recs), "save dc breakdown")
}

func (r *PostgresRepository) execBatch(ctx context.Context, batch *pgx.Batch, n int, label string) error {
	res := r.pool.SendBatch(ctx, batch)
	defer res.Close()
	for i := 0; i < n; i++ {
		if _, err := res.Exec(); err != nil {
			r.logger.Error(label, zap.Error(err))
			return err
		}
	}
	return nil
}

// getCalculatedServiceIDsForDate returns the set of service IDs that already have a
// daily report for the given day, used to skip recomputation during backfills.
func (r *PostgresRepository) getCalculatedServiceIDsForDate(ctx context.Context, date time.Time) (map[uuid.UUID]bool, error) {
	const q = `SELECT service_id FROM service_cost_reports WHERE date = $1`

	rows, err := r.pool.Query(ctx, q, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

func (r *PostgresRepository) getMonthlyReportsForMonth(ctx context.Context, month time.Time) ([]*MonthlyReport, error) {
	firstDay := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, 0).Add(-time.Nanosecond)

	const q = `
		SELECT
			service_id,
			MAX(service_name)         AS service_name,
			MAX(team)                 AS team,
			MAX(platform)             AS platform,
			SUM(compute_cost)         AS total_cost,
			AVG(compute_cost)::BIGINT AS daily_avg,
			COUNT(*)                  AS days_recorded,
			SUM(cpu_cost)             AS cpu_cost,
			SUM(ram_cost)             AS ram_cost,
			SUM(ssd_cost)             AS ssd_cost,
			SUM(hdd_cost)             AS hdd_cost
		FROM service_cost_reports
		WHERE date BETWEEN $1 AND $2
		GROUP BY service_id
		ORDER BY total_cost DESC`

	rows, err := r.pool.Query(ctx, q, firstDay, lastDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (*MonthlyReport, error) {
		var m MonthlyReport
		m.Month = firstDay
		err := row.Scan(
			&m.ServiceID, &m.ServiceName, &m.Team, &m.Platform,
			&m.TotalCost, &m.DailyAvg, &m.DaysRecorded,
			&m.CPUCost, &m.RAMCost, &m.SSDCost, &m.HDDCost,
		)
		return &m, err
	})
	if err != nil {
		return nil, err
	}
	return r.attachMonthlyDcBreakdown(ctx, firstDay, lastDay, reports)
}

// attachMonthlyDcBreakdown fetches per-DC monthly aggregates and attaches them to monthly reports.
func (r *PostgresRepository) attachMonthlyDcBreakdown(ctx context.Context, firstDay, lastDay time.Time, reports []*MonthlyReport) ([]*MonthlyReport, error) {
	if len(reports) == 0 {
		return reports, nil
	}
	const q = `
		SELECT
			service_id,
			datacenter_id,
			MAX(datacenter_name)       AS datacenter_name,
			SUM(compute_cost)          AS total_cost,
			AVG(compute_cost)::BIGINT  AS daily_avg,
			SUM(cpu_cost)              AS cpu_cost,
			SUM(ram_cost)              AS ram_cost,
			SUM(ssd_cost)              AS ssd_cost,
			SUM(hdd_cost)              AS hdd_cost
		FROM service_cost_reports_by_dc
		WHERE date BETWEEN $1 AND $2
		GROUP BY service_id, datacenter_id
		ORDER BY service_id, total_cost DESC`

	rows, err := r.pool.Query(ctx, q, firstDay, lastDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byService := make(map[uuid.UUID][]MonthlyDatacenterCost)
	for rows.Next() {
		var svcID uuid.UUID
		var dc MonthlyDatacenterCost
		if err := rows.Scan(&svcID, &dc.DatacenterID, &dc.DatacenterName, &dc.TotalCost, &dc.DailyAvg,
			&dc.CPUCost, &dc.RAMCost, &dc.SSDCost, &dc.HDDCost); err != nil {
			return nil, err
		}
		byService[svcID] = append(byService[svcID], dc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, rep := range reports {
		if breakdown, ok := byService[rep.ServiceID]; ok {
			rep.DatacenterBreakdown = breakdown
		} else {
			rep.DatacenterBreakdown = []MonthlyDatacenterCost{}
		}
	}
	return reports, nil
}

func (r *PostgresRepository) getMonthlyTrend(ctx context.Context, months int) ([]TrendPoint, error) {
	const q = `
		SELECT
			DATE_TRUNC('month', date)::TIMESTAMPTZ AS month,
			SUM(compute_cost)                      AS total_cost
		FROM service_cost_reports
		WHERE date >= NOW() - ($1 || ' months')::INTERVAL
		GROUP BY DATE_TRUNC('month', date)
		ORDER BY month ASC`

	rows, err := r.pool.Query(ctx, q, months)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []TrendPoint
	for rows.Next() {
		var p TrendPoint
		if err := rows.Scan(&p.Month, &p.TotalCost); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if points == nil {
		points = []TrendPoint{}
	}
	return points, nil
}

func (r *PostgresRepository) getSpendGrouped(ctx context.Context, firstDay time.Time, groupBy, platform string) ([]SpendItem, int64, error) {
	lastDay := firstDay.AddDate(0, 1, 0).Add(-time.Nanosecond)

	var q string
	switch groupBy {
	case "team":
		q = `
			SELECT team, team, SUM(compute_cost), COUNT(DISTINCT service_id)
			FROM service_cost_reports
			WHERE date BETWEEN $1 AND $2
			  AND ($3 = '' OR platform = $3)
			GROUP BY team
			ORDER BY SUM(compute_cost) DESC`
	case "platform":
		q = `
			SELECT platform, platform, SUM(compute_cost), COUNT(DISTINCT service_id)
			FROM service_cost_reports
			WHERE date BETWEEN $1 AND $2
			  AND ($3 = '' OR platform = $3)
			GROUP BY platform
			ORDER BY SUM(compute_cost) DESC`
	case "service":
		q = `
			SELECT service_id::TEXT, MAX(service_name), SUM(compute_cost), 1
			FROM service_cost_reports
			WHERE date BETWEEN $1 AND $2
			  AND ($3 = '' OR platform = $3)
			GROUP BY service_id
			ORDER BY SUM(compute_cost) DESC`
	case "datacenter":
		q = `
			SELECT dc.datacenter_id::TEXT, MAX(dc.datacenter_name), SUM(dc.compute_cost), COUNT(DISTINCT dc.service_id)
			FROM service_cost_reports_by_dc dc
			JOIN service_cost_reports scr ON scr.service_id = dc.service_id AND scr.date = dc.date
			WHERE dc.date BETWEEN $1 AND $2
			  AND ($3 = '' OR scr.platform = $3)
			GROUP BY dc.datacenter_id
			ORDER BY SUM(dc.compute_cost) DESC`
	default:
		return nil, 0, ErrInvalidGroupBy
	}

	rows, err := r.pool.Query(ctx, q, firstDay, lastDay, platform)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []SpendItem
	var total int64
	for rows.Next() {
		var item SpendItem
		if err := rows.Scan(&item.Key, &item.Label, &item.TotalCost, &item.ServiceCount); err != nil {
			return nil, 0, err
		}
		if groupBy == "platform" {
			if item.Label == "vm" {
				item.Label = "VM"
			} else if item.Label == "kubernetes" {
				item.Label = "Kubernetes"
			}
		}
		total += item.TotalCost
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
