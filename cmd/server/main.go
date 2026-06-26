package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/root-ali/rashnu/internal/api"
	"github.com/root-ali/rashnu/internal/config"
	"github.com/root-ali/rashnu/internal/cost_report"
	"github.com/root-ali/rashnu/internal/db"
	"github.com/root-ali/rashnu/internal/identity"
	"github.com/root-ali/rashnu/internal/inventory"
	"github.com/root-ali/rashnu/internal/migration"
	"github.com/root-ali/rashnu/internal/pricing"
	"github.com/root-ali/rashnu/internal/service_catalog"

	"go.uber.org/zap"
)

func main() {
	// Load config
	configFilePath := "config.yaml"
	cfg, err := config.NewConfig(configFilePath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initiate logger instance
	logger := initLogger(cfg.Logger.Level, cfg.Logger.Env, cfg.Logger.Encoding)
	defer logger.Sync()

	logger.Info("starting rashnu",
		zap.String("version", "1.0.0"),
		zap.String("description", "FinOps Platform for bare-metal infrastructure"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", cfg.Database.User, cfg.Database.Pass, cfg.Database.Host, cfg.Database.Port, cfg.Database.DB, cfg.Database.SSLMode)

	if err := migration.MigrateUp(dsn, logger); err != nil {
		logger.Fatal("failed to run database migrations", zap.Error(err))
	}

	// Initiate database instance
	dbCfg := db.Config{
		DSN:             dsn,
		MaxConns:        int32(cfg.Database.MaxConnections),
		MinConns:        int32(cfg.Database.MinConnections),
		MaxConnLifetime: time.Duration(cfg.Database.ConnMaxLifetime) * time.Minute,
		MaxConnIdleTime: time.Duration(cfg.Database.ConnMaxIdleTime) * time.Minute,
	}
	postgres, err := db.NewPostgres(ctx, dbCfg, logger)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer postgres.Close()

	// Initiate ّInventory Service
	inventoryRepo := inventory.NewPostgresRepository(postgres.Pool, logger)
	inventoryService := inventory.NewService(inventoryRepo, logger)

	// Initiate Pricing service
	pricingRepo := pricing.NewPostgresRepository(postgres.Pool, logger)
	pricingService := pricing.NewService(inventoryRepo, pricingRepo, logger)

	err = pricingService.CreateMonthlyPrice()
	if err != nil {
		logger.Error("failed to create monthly price", zap.Error(err))
	}

	// Backfill missing daily unit prices through today before cost reports run.
	pricingDailyScheduler := pricing.NewDailyScheduler(pricingService, logger)
	pricingDailyScheduler.RunOnStartup(ctx)
	go pricingDailyScheduler.Start(ctx)

	pricingMonthlyScheduler := pricing.NewMonthlyScheduler(pricingService, logger)
	go pricingMonthlyScheduler.Start(ctx)

	// Initiate identity Service
	jwtSecret := []byte(cfg.JWT.Secret)
	identityRepo := identity.NewPostgresRepository(postgres.Pool, logger)
	identityService := identity.NewService(identityRepo, jwtSecret, logger)

	// Define default admin user
	err = identityService.CreateAdminUser()
	if err != nil {
		if errors.Is(err, identity.ErrUserAlreadyExist) {
			logger.Debug("admin user already exists", zap.Error(err))
		} else {
			logger.Error("failed to create admin user", zap.Error(err))
			os.Exit(1)
		}
	}

	// Initiate Service Catalog
	catalogRepo := service_catalog.NewPostgresRepository(postgres.Pool, logger)
	promConfigRepo := service_catalog.NewPostgresPrometheusConfigRepository(postgres.Pool, logger)
	promRepo := service_catalog.NewHTTPPrometheusRepository(logger)
	catalogService := service_catalog.NewCatalog(catalogRepo, promConfigRepo, promRepo, logger)

	// Initiate Cost Report Service
	costReportRepo := cost_report.NewPostgresRepository(postgres.Pool, logger)
	costReportService := cost_report.NewService(
		costReportRepo,
		&catalogReaderAdapter{repo: catalogRepo},
		&pricingReaderAdapter{repo: pricingRepo},
		logger,
	)

	// Calculate today's costs on startup so the dashboard has data immediately.
	if err := costReportService.CalculateDailyServiceCosts(ctx, time.Now()); err != nil {
		logger.Error("failed to calculate daily service costs on startup", zap.Error(err))
	}

	// Run the daily cost calculation every day at 01:00.
	costReportScheduler := cost_report.NewScheduler(costReportService, logger)
	go costReportScheduler.Start(ctx)

	// Initiate HTTP Server
	handler := api.NewHandler(inventoryService, pricingService, identityService, catalogService, costReportService, jwtSecret, logger)

	httpAddr := fmt.Sprintf("%s:%s", cfg.Http.Host, cfg.Http.Port)
	srv := &http.Server{
		Addr:           httpAddr,
		Handler:        handler.Router(),
		ReadTimeout:    time.Duration(cfg.Http.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Http.WriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(cfg.Http.IdleTimeout) * time.Second,
		MaxHeaderBytes: int(cfg.Http.MaxHeaderBytes),
	}

	go func() {
		logger.Info("http server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	gracefulShutdown(srv, logger)
}

func initLogger(logLevel string, env string, encoding string) *zap.Logger {

	var config zap.Config
	var level zap.AtomicLevel

	// Parse log level
	switch logLevel {
	case "debug":
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn", "warning":
		level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		level = zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	// Check if we're in development mode
	if env == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Force JSON encoding for all environments
	config.Encoding = encoding
	config.Level = level

	logger, err := config.Build()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}

	logger.Info("logger initialized",
		zap.String("level", logLevel),
		zap.String("env", env),
	)

	return logger
}

// catalogReaderAdapter adapts service_catalog.Repository into cost_report.CatalogReader.
type catalogReaderAdapter struct {
	repo service_catalog.Repository
}

func (a *catalogReaderAdapter) ListServicesForCost(ctx context.Context) ([]*cost_report.ServiceInput, error) {
	svcs, err := a.repo.ListServices(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*cost_report.ServiceInput, 0, len(svcs))
	for _, s := range svcs {
		pods, err := a.repo.GetPodsByServiceID(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		vms, err := a.repo.GetVMsByServiceID(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		workloads := make([]cost_report.WorkloadInput, 0, len(pods)+len(vms))
		for _, p := range pods {
			workloads = append(workloads, cost_report.WorkloadInput{
				VCPU: p.VCPU, MemoryGB: p.MemoryGB, SSDGB: p.SSDGB,
				HDDGB: p.HDDGB, GPUs: p.GPUs,
				DatacenterID:   p.DatacenterID,
				DatacenterName: p.DatacenterName,
			})
		}
		for _, v := range vms {
			workloads = append(workloads, cost_report.WorkloadInput{
				VCPU: v.VCPU, MemoryGB: v.MemoryGB, SSDGB: v.SSDGB,
				HDDGB: v.HDDGB, GPUs: v.GPUs,
				DatacenterID:   v.DatacenterID,
				DatacenterName: v.DatacenterName,
			})
		}
		result = append(result, &cost_report.ServiceInput{
			ID:        s.ID,
			Name:      s.Name,
			Team:      s.Team,
			Platform:  s.Platform,
			Workloads: workloads,
		})
	}
	return result, nil
}

// pricingReaderAdapter adapts pricing.PostgresRepository into cost_report.PricingReader.
type pricingReaderAdapter struct {
	repo *pricing.PostgresRepository
}

func (a *pricingReaderAdapter) GetDailyPricesForDate(ctx context.Context, date time.Time) ([]*cost_report.DailyUnitPrice, error) {
	entries, err := a.repo.GetDailyPricesForDate(ctx, date)
	if err != nil {
		return nil, err
	}
	result := make([]*cost_report.DailyUnitPrice, 0, len(entries))
	for _, e := range entries {
		result = append(result, &cost_report.DailyUnitPrice{
			DatacenterID: e.DatacenterID,
			VCPUPrice:    e.VCPUPrice,
			MemoryPrice:  e.MemoryPrice,
			SSDPrice:     e.SSDPrice,
			HDDPrice:     e.HDDPrice,
			GPUPrice:     e.GPUPrice,
		})
	}
	return result, nil
}

func gracefulShutdown(srv *http.Server, logger *zap.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down http server and database connection with timeout of 20 seconds because of kill signal")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}
	logger.Info("rashnu stopped gracefully")
}
