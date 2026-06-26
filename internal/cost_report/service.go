package cost_report

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CatalogReader provides service + workload data needed for cost calculation.
type CatalogReader interface {
	ListServicesForCost(ctx context.Context) ([]*ServiceInput, error)
}

// PricingReader provides the per-unit daily prices per datacenter.
type PricingReader interface {
	GetDailyPricesForDate(ctx context.Context, date time.Time) ([]*DailyUnitPrice, error)
}

// Service is the public interface for cost reporting.
type Service interface {
	CalculateDailyServiceCosts(ctx context.Context, date time.Time) error
	FulfillServiceCosts(ctx context.Context, startTime time.Time) error
	GetMonthlyReportsForMonth(ctx context.Context, month time.Time) ([]*MonthlyReport, error)
	GetMonthlyTrend(ctx context.Context, months int) ([]TrendPoint, error)
	GetSpendSummary(ctx context.Context, month time.Time, groupBy, platform string) (*SpendSummaryResponse, error)
}

type costService struct {
	repo    *PostgresRepository
	catalog CatalogReader
	pricing PricingReader
	logger  *zap.Logger
}

func NewService(repo *PostgresRepository, catalog CatalogReader, pricing PricingReader, logger *zap.Logger) Service {
	return &costService{repo: repo, catalog: catalog, pricing: pricing, logger: logger}
}

// dcGroup accumulates cost for one datacenter within a single service.
type dcGroup struct {
	datacenterID   uuid.UUID
	datacenterName string
	workloadCount  int
	computeCost    int64
	cpuCost        int64
	ramCost        int64
	ssdCost        int64
	hddCost        int64
}

// CalculateDailyServiceCosts computes and persists the cost of every service for the given day,
// overwriting any existing reports for that day. Used by the daily scheduler and manual recalculation.
func (s *costService) CalculateDailyServiceCosts(ctx context.Context, date time.Time) error {
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	return s.calculateDailyServiceCosts(ctx, day, false)
}

// FulfillServiceCosts backfills daily service costs for every day from startTime up to today.
// Services that already have a report for a given day are skipped, so re-running after a new
// service is added only computes the missing rows.
func (s *costService) FulfillServiceCosts(ctx context.Context, startTime time.Time) error {
	start := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if start.After(end) {
		return fmt.Errorf("%w: start date %s is in the future", ErrInvalidDate, start.Format("2006-01-02"))
	}

	daysProcessed := 0
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		if err := s.calculateDailyServiceCosts(ctx, day, true); err != nil {
			return fmt.Errorf("fulfill %s: %w", day.Format("2006-01-02"), err)
		}
		daysProcessed++
	}

	s.logger.Info("fulfilled service costs",
		zap.Time("from", start),
		zap.Time("to", end),
		zap.Int("days", daysProcessed),
	)
	return nil
}

// calculateDailyServiceCosts computes and persists the cost of services for a single normalized day.
// Workloads are grouped by datacenter so multi-DC services produce a per-DC breakdown.
// When skipExisting is true, services that already have a report for the day are left untouched.
func (s *costService) calculateDailyServiceCosts(ctx context.Context, day time.Time, skipExisting bool) error {
	services, err := s.catalog.ListServicesForCost(ctx)
	if err != nil {
		return fmt.Errorf("list services: %w", err)
	}
	if len(services) == 0 {
		s.logger.Info("no services found, skipping cost calculation", zap.Time("date", day))
		return nil
	}

	var alreadyCalculated map[uuid.UUID]bool
	if skipExisting {
		alreadyCalculated, err = s.repo.getCalculatedServiceIDsForDate(ctx, day)
		if err != nil {
			return fmt.Errorf("get calculated service ids for %s: %w", day.Format("2006-01-02"), err)
		}
	}

	prices, err := s.pricing.GetDailyPricesForDate(ctx, day)
	if err != nil {
		return fmt.Errorf("get daily prices for %s: %w", day.Format("2006-01-02"), err)
	}

	priceMap := make(map[uuid.UUID]*DailyUnitPrice, len(prices))
	for _, p := range prices {
		priceMap[p.DatacenterID] = p
	}

	reports := make([]*DailyReport, 0, len(services))
	var dcRecords []*dcBreakdownRecord

	for _, svc := range services {
		if skipExisting && alreadyCalculated[svc.ID] {
			continue
		}
		// Kubernetes services spanning 2+ datacenters use averaged CPU/RAM unit prices.
		serviceDCs := uniqueWorkloadDatacenters(svc.Workloads)
		var cpuRAMBlend *blendedCPURAM
		if shouldBlendCPURAM(svc.Platform, len(serviceDCs)) {
			cpuRAMBlend = averageCPURAMPrices(serviceDCs, priceMap)
		}

		// Group workloads by datacenter, computing cost per workload.
		groups := make(map[uuid.UUID]*dcGroup)
		for _, w := range svc.Workloads {
			g, ok := groups[w.DatacenterID]
			if !ok {
				g = &dcGroup{
					datacenterID:   w.DatacenterID,
					datacenterName: w.DatacenterName,
				}
				groups[w.DatacenterID] = g
			}
			g.workloadCount++
			p, ok := priceMap[w.DatacenterID]
			if !ok {
				continue
			}
			wCPU, wRAM, wSSD, wHDD, wGPU := workloadResourceCosts(w, p, cpuRAMBlend)
			g.cpuCost += wCPU
			g.ramCost += wRAM
			g.ssdCost += wSSD
			g.hddCost += wHDD
			g.computeCost += wCPU + wRAM + wSSD + wHDD + wGPU
		}

		var totalCost, cpuCost, ramCost, ssdCost, hddCost int64
		for _, g := range groups {
			totalCost += g.computeCost
			cpuCost += g.cpuCost
			ramCost += g.ramCost
			ssdCost += g.ssdCost
			hddCost += g.hddCost
		}

		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		reports = append(reports, &DailyReport{
			ID:            id,
			ServiceID:     svc.ID,
			ServiceName:   svc.Name,
			Team:          svc.Team,
			Platform:      svc.Platform,
			WorkloadCount: len(svc.Workloads),
			Date:          day,
			ComputeCost:   totalCost,
			CPUCost:       cpuCost,
			RAMCost:       ramCost,
			SSDCost:       ssdCost,
			HDDCost:       hddCost,
		})

		for _, g := range groups {
			dcID, err := uuid.NewV7()
			if err != nil {
				return err
			}
			dcRecords = append(dcRecords, &dcBreakdownRecord{
				id:             dcID,
				serviceID:      svc.ID,
				datacenterID:   g.datacenterID,
				datacenterName: g.datacenterName,
				workloadCount:  g.workloadCount,
				date:           day,
				computeCost:    g.computeCost,
				cpuCost:        g.cpuCost,
				ramCost:        g.ramCost,
				ssdCost:        g.ssdCost,
				hddCost:        g.hddCost,
			})
		}
	}

	if len(reports) == 0 {
		s.logger.Info("no services to calculate for day, skipping",
			zap.Time("date", day),
			zap.Bool("skip_existing", skipExisting),
		)
		return nil
	}

	if err := s.repo.saveDailyReports(ctx, reports); err != nil {
		return fmt.Errorf("save daily reports: %w", err)
	}
	if len(dcRecords) > 0 {
		if err := s.repo.saveDcBreakdowns(ctx, dcRecords); err != nil {
			return fmt.Errorf("save dc breakdowns: %w", err)
		}
	}

	s.logger.Info("calculated daily service costs",
		zap.Time("date", day),
		zap.Int("services", len(reports)),
		zap.Int("dc_records", len(dcRecords)),
	)
	return nil
}

func (s *costService) GetMonthlyReportsForMonth(ctx context.Context, month time.Time) ([]*MonthlyReport, error) {
	return s.repo.getMonthlyReportsForMonth(ctx, month)
}

func (s *costService) GetMonthlyTrend(ctx context.Context, months int) ([]TrendPoint, error) {
	return s.repo.getMonthlyTrend(ctx, months)
}

func (s *costService) GetSpendSummary(ctx context.Context, month time.Time, groupBy, platform string) (*SpendSummaryResponse, error) {
	firstDay := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	items, total, err := s.repo.getSpendGrouped(ctx, firstDay, groupBy, platform)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []SpendItem{}
	}
	return &SpendSummaryResponse{
		Month:   firstDay,
		GroupBy: groupBy,
		Total:   total,
		Items:   items,
	}, nil
}
