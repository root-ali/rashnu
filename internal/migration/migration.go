// Package migration provides database migration functionality using golang-migrate
package migration

import (
	"context"
	"errors"
	"fmt"
	_ "io/fs"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/root-ali/rashnu/migrations"
	"go.uber.org/zap"
)

// Config holds migration configuration
type Config struct {
	DatabaseURL string
	Logger      *zap.Logger
}

// Migrator handles database migrations
type Migrator struct {
	migrate *migrate.Migrate
	logger  *zap.Logger
}

// New creates a new Migrator instance
func New(cfg Config) (*Migrator, error) {
	if cfg.DatabaseURL == "" {
		return nil, errors.New("database URL is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	// Create source from embedded filesystem
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{
		migrate: m,
		logger:  logger,
	}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up(ctx context.Context) error {
	m.logger.Info("running database migrations")

	done := make(chan error, 1)
	go func() {
		done <- m.migrate.Up()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			m.logger.Error("migration failed", zap.Error(err))
			return fmt.Errorf("migration up failed: %w", err)
		}
		if errors.Is(err, migrate.ErrNoChange) {
			m.logger.Info("no migrations to apply")
			return nil
		}
		m.logger.Info("migrations applied successfully")
		return nil
	}
}

// Down rolls back the last migration
func (m *Migrator) Down(ctx context.Context) error {
	m.logger.Info("rolling back last migration")

	done := make(chan error, 1)
	go func() {
		done <- m.migrate.Down()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			m.logger.Error("rollback failed", zap.Error(err))
			return fmt.Errorf("migration down failed: %w", err)
		}
		if errors.Is(err, migrate.ErrNoChange) {
			m.logger.Info("no migrations to roll back")
			return nil
		}
		m.logger.Info("migration rolled back successfully")
		return nil
	}
}

// Steps runs n migrations (positive for up, negative for down)
func (m *Migrator) Steps(ctx context.Context, n int) error {
	m.logger.Info("running migration steps", zap.Int("steps", n))

	done := make(chan error, 1)
	go func() {
		done <- m.migrate.Steps(n)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			m.logger.Error("migration steps failed", zap.Error(err), zap.Int("steps", n))
			return fmt.Errorf("migration steps failed: %w", err)
		}
		if errors.Is(err, migrate.ErrNoChange) {
			m.logger.Info("no migrations to apply")
			return nil
		}
		m.logger.Info("migration steps completed", zap.Int("steps", n))
		return nil
	}
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations
// Use with caution - typically only needed to recover from failed migrations
func (m *Migrator) Force(version int) error {
	m.logger.Warn("forcing migration version", zap.Int("version", version))
	if err := m.migrate.Force(version); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}
	m.logger.Info("migration version forced", zap.Int("version", version))
	return nil
}

// Drop drops all tables and the schema_migrations table
// WARNING: This is destructive and should only be used in development
func (m *Migrator) Drop(ctx context.Context) error {
	m.logger.Warn("dropping all database tables")

	done := make(chan error, 1)
	go func() {
		done <- m.migrate.Drop()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			m.logger.Error("drop failed", zap.Error(err))
			return fmt.Errorf("drop failed: %w", err)
		}
		m.logger.Info("all tables dropped")
		return nil
	}
}

// Close closes the migration instance
func (m *Migrator) Close() error {
	srcErr, dbErr := m.migrate.Close()
	if srcErr != nil {
		return fmt.Errorf("failed to close source: %w", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("failed to close database: %w", dbErr)
	}
	return nil
}

// MigrateUp is a convenience function to run migrations with a timeout
func MigrateUp(databaseURL string, logger *zap.Logger) error {
	migrator, err := New(Config{
		DatabaseURL: databaseURL,
		Logger:      logger,
	})
	if err != nil {
		return err
	}
	defer migrator.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return migrator.Up(ctx)
}

// MigrateDown is a convenience function to rollback the last migration
func MigrateDown(databaseURL string, logger *zap.Logger) error {
	migrator, err := New(Config{
		DatabaseURL: databaseURL,
		Logger:      logger,
	})
	if err != nil {
		return err
	}
	defer migrator.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return migrator.Down(ctx)
}

// GetVersion returns the current migration version
func GetVersion(databaseURL string) (uint, bool, error) {
	migrator, err := New(Config{
		DatabaseURL: databaseURL,
	})
	if err != nil {
		return 0, false, err
	}
	defer migrator.Close()

	return migrator.Version()
}
