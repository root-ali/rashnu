package service_catalog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PrometheusConfig struct {
	URL       string    `json:"url"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdatePrometheusConfigRequest struct {
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// PrometheusConfigRepository persists the single Prometheus config row.
type PrometheusConfigRepository interface {
	Get(ctx context.Context) (*PrometheusConfig, error)
	Upsert(ctx context.Context, cfg PrometheusConfig) error
}

type postgresPrometheusConfigRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgresPrometheusConfigRepository(pool *pgxpool.Pool, logger *zap.Logger) PrometheusConfigRepository {
	return &postgresPrometheusConfigRepo{pool: pool, logger: logger}
}

func (r *postgresPrometheusConfigRepo) Get(ctx context.Context) (*PrometheusConfig, error) {
	const q = `SELECT url, enabled, updated_at FROM prometheus_config WHERE id = 1`
	var c PrometheusConfig
	err := r.pool.QueryRow(ctx, q).Scan(&c.URL, &c.Enabled, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &PrometheusConfig{Enabled: false}, nil
		}
		r.logger.Error("get prometheus config", zap.Error(err))
		return nil, fmt.Errorf("get prometheus config: %w", err)
	}
	return &c, nil
}

func (r *postgresPrometheusConfigRepo) Upsert(ctx context.Context, cfg PrometheusConfig) error {
	const q = `
		INSERT INTO prometheus_config (id, url, enabled, updated_at)
		VALUES (1, $1, $2, NOW())
		ON CONFLICT (id) DO UPDATE
		SET url = $1, enabled = $2, updated_at = NOW()`
	if _, err := r.pool.Exec(ctx, q, cfg.URL, cfg.Enabled); err != nil {
		r.logger.Error("upsert prometheus config", zap.Error(err))
		return fmt.Errorf("upsert prometheus config: %w", err)
	}
	return nil
}
