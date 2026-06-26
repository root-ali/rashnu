package pricing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

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

func (r *PostgresRepository) createMonthlyPrice(ctx context.Context, prices []*perMonthPrice) error {
	query := `
		INSERT INTO pricing (id, datacenter_id, month, cpu_price, memory_price, ssd_price, hdd_price, gpu_price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (datacenter_id, month)
		DO UPDATE SET
			cpu_price = EXCLUDED.cpu_price,
			memory_price = EXCLUDED.memory_price,
			ssd_price = EXCLUDED.ssd_price,
			hdd_price = EXCLUDED.hdd_price,
			gpu_price = EXCLUDED.gpu_price,
			updated_at = now()
	`

	batch := &pgx.Batch{}
	for _, price := range prices {
		batch.Queue(
			query,
			uuid.New(),
			price.DatacenterID,
			price.Date,
			price.VCPUPrice,
			price.MemoryPrice,
			price.SSDPrice,
			price.HDDPrice,
			price.GPUPrice,
		)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < len(prices); i++ {
		_, err := results.Exec()
		if err != nil {
			r.logger.Error("failed to insert price", zap.Error(err))
			return err
		}
	}

	return nil
}

func (r *PostgresRepository) getMonthlyPrice(ctx context.Context, datacenterId uuid.UUID, month time.Time) (*Price, error) {
	query := `
		SELECT id, datacenter_id, month, cpu_price, memory_price, ssd_price, hdd_price, gpu_price, created_at, updated_at
		FROM pricing
		WHERE datacenter_id = $1 AND month = $2
	`

	var price Price
	err := r.pool.QueryRow(ctx, query, datacenterId, month).Scan(
		&price.ID,
		&price.DatacenterID,
		&price.Date,
		&price.CPUPrice,
		&price.MemoryPrice,
		&price.SSDPrice,
		&price.HDDPrice,
		&price.GPUPrice,
		&price.CreatedAt,
		&price.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("failed to get price", zap.Error(err))
		return nil, err
	}

	return &price, nil
}

func (r *PostgresRepository) getAllPrices(ctx context.Context) ([]*Price, error) {
	query := `
		SELECT id, datacenter_id, month, cpu_price, memory_price, ssd_price, hdd_price, gpu_price, created_at, updated_at
		FROM pricing
		ORDER BY month DESC, datacenter_id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to query prices", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var prices []*Price
	for rows.Next() {
		var price Price
		err := rows.Scan(
			&price.ID,
			&price.DatacenterID,
			&price.Date,
			&price.CPUPrice,
			&price.MemoryPrice,
			&price.SSDPrice,
			&price.HDDPrice,
			&price.GPUPrice,
			&price.CreatedAt,
			&price.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan price", zap.Error(err))
			return nil, err
		}
		prices = append(prices, &price)
	}

	return prices, nil
}

func (r *PostgresRepository) updateMonthlyPrice(ctx context.Context, datacenterId uuid.UUID, month time.Time, price *perMonthPrice) error {
	query := `
		UPDATE pricing
		SET cpu_price = $1, memory_price = $2, ssd_price = $3, hdd_price = $4, gpu_price = $5, updated_at = now()
		WHERE datacenter_id = $6 AND month = $7
	`

	result, err := r.pool.Exec(ctx, query,
		price.VCPUPrice,
		price.MemoryPrice,
		price.SSDPrice,
		price.HDDPrice,
		price.GPUPrice,
		datacenterId,
		month,
	)

	if err != nil {
		r.logger.Error("failed to update price", zap.Error(err))
		return err
	}

	if result.RowsAffected() == 0 {
		r.logger.Warn("no rows updated", zap.Any("datacenter_id", datacenterId), zap.Any("month", month))
	}

	return nil
}

// GetDailyPricesForDate returns all per-unit prices for every datacenter on the given day.
func (r *PostgresRepository) GetDailyPricesForDate(ctx context.Context, date time.Time) ([]*DailyPriceEntry, error) {
	const q = `
		SELECT datacenter_id, cpu_price, memory_price, ssd_price, hdd_price, gpu_price
		FROM daily_pricing
		WHERE date = $1`

	rows, err := r.pool.Query(ctx, q, date)
	if err != nil {
		r.logger.Error("get daily prices for date", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var entries []*DailyPriceEntry
	for rows.Next() {
		var e DailyPriceEntry
		if err := rows.Scan(&e.DatacenterID, &e.VCPUPrice, &e.MemoryPrice, &e.SSDPrice, &e.HDDPrice, &e.GPUPrice); err != nil {
			r.logger.Error("scan daily price entry", zap.Error(err))
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

func (r *PostgresRepository) getLatestDailyPriceDate(ctx context.Context) (*time.Time, error) {
	const q = `SELECT MAX(date) FROM daily_pricing`

	var latest *time.Time
	if err := r.pool.QueryRow(ctx, q).Scan(&latest); err != nil {
		r.logger.Error("get latest daily price date", zap.Error(err))
		return nil, err
	}
	if latest == nil {
		return nil, nil
	}

	day := time.Date(latest.Year(), latest.Month(), latest.Day(), 0, 0, 0, 0, time.UTC)
	return &day, nil
}

func (r *PostgresRepository) hasDailyPricesForDate(ctx context.Context, date time.Time) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM daily_pricing WHERE date = $1)`

	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	var exists bool
	if err := r.pool.QueryRow(ctx, q, day).Scan(&exists); err != nil {
		r.logger.Error("check daily prices for date", zap.Error(err), zap.Time("date", day))
		return false, err
	}
	return exists, nil
}

func (r *PostgresRepository) createDailyPrice(ctx context.Context, prices []*perDayPrice) error {
	query := `
		INSERT INTO daily_pricing (id, datacenter_id, date, cpu_price, memory_price, ssd_price, hdd_price, gpu_price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (datacenter_id, date)
		DO UPDATE SET
			cpu_price = EXCLUDED.cpu_price,
			memory_price = EXCLUDED.memory_price,
			ssd_price = EXCLUDED.ssd_price,
			hdd_price = EXCLUDED.hdd_price,
			gpu_price = EXCLUDED.gpu_price,
			updated_at = now()
	`

	batch := &pgx.Batch{}
	for _, price := range prices {
		batch.Queue(
			query,
			uuid.New(),
			price.DatacenterID,
			price.Date,
			price.VCPUPrice,
			price.MemoryPrice,
			price.SSDPrice,
			price.HDDPrice,
			price.GPUPrice,
		)
	}

	results := r.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < len(prices); i++ {
		_, err := results.Exec()
		if err != nil {
			r.logger.Error("failed to insert daily price", zap.Error(err))
			return err
		}
	}

	return nil
}
