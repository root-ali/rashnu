package pricing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/root-ali/rashnu/internal/inventory"
	"go.uber.org/zap"
)

type InventoryRepositoryInterface interface {
	GetDatacentersWithHardware(ctx context.Context, nameFilter, regionFilter string) ([]*inventory.DatacenterWithHardware, error)
}

type Repository interface {
	createMonthlyPrice(ctx context.Context, prices []*perMonthPrice) error
	createDailyPrice(ctx context.Context, prices []*perDayPrice) error
	getMonthlyPrice(ctx context.Context, datacenterId uuid.UUID, month time.Time) (*Price, error)
	getAllPrices(ctx context.Context) ([]*Price, error)
	updateMonthlyPrice(ctx context.Context, datacenterId uuid.UUID, month time.Time, price *perMonthPrice) error
	getLatestDailyPriceDate(ctx context.Context) (*time.Time, error)
	hasDailyPricesForDate(ctx context.Context, date time.Time) (bool, error)
}

type Service interface {
	CreateMonthlyPrice() error
	FulfillPriceData(startTime time.Time) error
	EnsureDailyPricesFilled(ctx context.Context, through time.Time) error
	EnsureMonthlyPricesForMonth(ctx context.Context, month time.Time) error
	GetMonthlyPrice(datacenterId uuid.UUID, month time.Time) (*Price, error)
	GetAllPrices() ([]*Price, error)
}

type service struct {
	inventory  InventoryRepositoryInterface
	repository Repository
	logger     *zap.Logger
}

func NewService(inv InventoryRepositoryInterface, repository Repository, logger *zap.Logger) Service {
	return &service{inv, repository, logger}
}

func (s *service) FulfillPriceData(startTime time.Time) error {
	ctx := context.Background()

	// Get all datacenters with hardware
	s.logger.Info("fetching datacenters with hardware inventory")
	getInvCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	allInventory, err := s.inventory.GetDatacentersWithHardware(getInvCtx, "", "")
	if err != nil {
		s.logger.Error("failed to get datacenters inventory", zap.Error(err))
		return err
	}

	if len(allInventory) == 0 {
		s.logger.Warn("no datacenters found in inventory")
		return nil
	}

	s.logger.Info("found datacenters", zap.Int("count", len(allInventory)))

	// Normalize dates
	startMonth := normalizeToMonthStart(startTime)
	now := time.Now()
	currentMonth := normalizeToMonthStart(now)

	s.logger.Info("processing price data",
		zap.Time("from", startMonth),
		zap.Time("to", currentMonth))

	totalMonths := 0
	totalDays := 0

	// Process each month from start to current
	for monthDate := startMonth; !monthDate.After(currentMonth); monthDate = monthDate.AddDate(0, 1, 0) {
		s.logger.Debug("processing month", zap.Time("month", monthDate))

		// Calculate and save monthly prices
		if err := s.processMonthlyPrices(ctx, allInventory, monthDate); err != nil {
			s.logger.Error("failed to process monthly prices",
				zap.Time("month", monthDate),
				zap.Error(err))
			return err
		}
		totalMonths++

		// Calculate and save daily prices for this month
		daysProcessed, err := s.processDailyPrices(ctx, allInventory, monthDate, now)
		if err != nil {
			s.logger.Error("failed to process daily prices",
				zap.Time("month", monthDate),
				zap.Error(err))
			return err
		}
		totalDays += daysProcessed
	}

	s.logger.Info("successfully fulfilled price data",
		zap.Int("months_processed", totalMonths),
		zap.Int("days_processed", totalDays),
		zap.Time("from", startMonth),
		zap.Time("to", currentMonth))

	return nil
}

// EnsureDailyPricesFilled writes per-datacenter daily unit prices for every missing
// calendar day from the day after the latest stored date through through (inclusive).
// Days that already have rows are skipped.
func (s *service) EnsureDailyPricesFilled(ctx context.Context, through time.Time) error {
	throughDay := normalizeToDayStart(through)

	getInvCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	allInventory, err := s.inventory.GetDatacentersWithHardware(getInvCtx, "", "")
	if err != nil {
		s.logger.Error("failed to get datacenters inventory", zap.Error(err))
		return err
	}
	if len(allInventory) == 0 {
		s.logger.Warn("no datacenters found, skipping daily price fill")
		return nil
	}

	latestCtx, latestCancel := context.WithTimeout(ctx, 5*time.Second)
	defer latestCancel()

	latest, err := s.repository.getLatestDailyPriceDate(latestCtx)
	if err != nil {
		return err
	}

	startDay := firstDailyPriceDay(latest, throughDay)
	if startDay.After(throughDay) {
		s.logger.Debug("daily prices already up to date",
			zap.Time("through", throughDay),
			zap.Time("latest", derefTime(latest)))
		return nil
	}

	s.logger.Info("filling daily prices",
		zap.Time("from", startDay),
		zap.Time("through", throughDay))

	daysProcessed := 0
	for day := startDay; !day.After(throughDay); day = day.AddDate(0, 0, 1) {
		checkCtx, checkCancel := context.WithTimeout(ctx, 5*time.Second)
		exists, err := s.repository.hasDailyPricesForDate(checkCtx, day)
		checkCancel()
		if err != nil {
			return err
		}
		if exists {
			s.logger.Debug("daily prices already exist, skipping day", zap.Time("date", day))
			continue
		}

		if err := s.saveDailyPricesForDay(ctx, allInventory, day); err != nil {
			s.logger.Error("failed to save daily prices",
				zap.Time("date", day),
				zap.Error(err))
			return err
		}
		daysProcessed++
	}

	s.logger.Info("daily prices filled",
		zap.Int("days_processed", daysProcessed),
		zap.Time("from", startDay),
		zap.Time("through", throughDay))

	return nil
}

// EnsureMonthlyPricesForMonth calculates and stores monthly prices for every datacenter
// in the given calendar month.
func (s *service) EnsureMonthlyPricesForMonth(ctx context.Context, month time.Time) error {
	monthDate := normalizeToMonthStart(month)

	getInvCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	allInventory, err := s.inventory.GetDatacentersWithHardware(getInvCtx, "", "")
	if err != nil {
		s.logger.Error("failed to get datacenters inventory", zap.Error(err))
		return err
	}
	if len(allInventory) == 0 {
		s.logger.Warn("no datacenters found, skipping monthly price fill")
		return nil
	}

	s.logger.Info("ensuring monthly prices", zap.Time("month", monthDate))
	return s.processMonthlyPrices(ctx, allInventory, monthDate)
}

func (s *service) saveDailyPricesForDay(ctx context.Context, inventories []*inventory.DatacenterWithHardware, day time.Time) error {
	monthDate := normalizeToMonthStart(day)
	nextMonth := monthDate.AddDate(0, 1, 0)
	daysInMonth := int(nextMonth.Sub(monthDate).Hours() / 24)

	prices := make([]*perDayPrice, 0, len(inventories))
	for _, inv := range inventories {
		prices = append(prices, s.calculateDailyPriceForDatacenter(inv, day, daysInMonth))
	}

	saveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := s.repository.createDailyPrice(saveCtx, prices); err != nil {
		return err
	}

	s.logger.Debug("saved daily prices",
		zap.Time("date", day),
		zap.Int("count", len(prices)))

	return nil
}

func firstDailyPriceDay(latest *time.Time, through time.Time) time.Time {
	if latest != nil {
		return latest.AddDate(0, 0, 1)
	}
	return normalizeToMonthStart(through)
}

func normalizeToDayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// processMonthlyPrices calculates and stores monthly prices for all datacenters for a given month
func (s *service) processMonthlyPrices(ctx context.Context, inventories []*inventory.DatacenterWithHardware, monthDate time.Time) error {
	monthlyPrices := make([]*perMonthPrice, 0, len(inventories))

	for _, inv := range inventories {
		price := s.calculateMonthlyPriceForDatacenter(inv, monthDate)
		monthlyPrices = append(monthlyPrices, price)
	}

	saveCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := s.repository.createMonthlyPrice(saveCtx, monthlyPrices); err != nil {
		return err
	}

	s.logger.Debug("saved monthly prices",
		zap.Time("month", monthDate),
		zap.Int("count", len(monthlyPrices)))

	return nil
}

// processDailyPrices calculates and stores daily prices for all datacenters for a given month
func (s *service) processDailyPrices(ctx context.Context, inventories []*inventory.DatacenterWithHardware, monthDate time.Time, maxDate time.Time) (int, error) {
	nextMonth := monthDate.AddDate(0, 1, 0)
	daysInMonth := int(nextMonth.Sub(monthDate).Hours() / 24)

	// Batch daily prices to avoid too many DB calls
	allDailyPrices := make([]*perDayPrice, 0, len(inventories)*daysInMonth)
	daysProcessed := 0

	for day := 0; day < daysInMonth; day++ {
		currentDate := monthDate.AddDate(0, 0, day)

		// Skip dates in the future
		if currentDate.After(maxDate) {
			break
		}

		for _, inv := range inventories {
			price := s.calculateDailyPriceForDatacenter(inv, currentDate, daysInMonth)
			allDailyPrices = append(allDailyPrices, price)
		}

		daysProcessed++
	}

	if len(allDailyPrices) > 0 {
		saveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := s.repository.createDailyPrice(saveCtx, allDailyPrices); err != nil {
			return 0, err
		}

		s.logger.Debug("saved daily prices",
			zap.Time("month", monthDate),
			zap.Int("days", daysProcessed),
			zap.Int("total_records", len(allDailyPrices)))
	}

	return daysProcessed, nil
}

// calculateMonthlyPriceForDatacenter calculates the monthly price for a single datacenter
func (s *service) calculateMonthlyPriceForDatacenter(inv *inventory.DatacenterWithHardware, monthDate time.Time) *perMonthPrice {
	data := s.calculateBasePriceDataForMonth(inv, monthDate)

	return &perMonthPrice{
		DatacenterID: inv.DataCenter.ID,
		Region:       inv.DataCenter.Region,
		Date:         monthDate,
		TotalPrice:   data.totalMonthlyPrice,
		VCPUPrice:    calculateUnitPrice(data.cpuPortion, data.totalVCPUs),
		MemoryPrice:  calculateUnitPrice(data.memoryPortion, data.totalMemoryGB),
		SSDPrice:     calculateUnitPrice(data.ssdPortion, data.totalSSDGB),
		HDDPrice:     calculateUnitPrice(data.hddPortion, data.totalHDDGB),
		GPUPrice:     0,
	}
}

// calculateDailyPriceForDatacenter calculates the daily price for a single datacenter
func (s *service) calculateDailyPriceForDatacenter(inv *inventory.DatacenterWithHardware, date time.Time, daysInMonth int) *perDayPrice {
	monthDate := normalizeToMonthStart(date)
	data := s.calculateBasePriceDataForMonth(inv, monthDate)

	return &perDayPrice{
		DatacenterID: inv.DataCenter.ID,
		Region:       inv.DataCenter.Region,
		Date:         date,
		TotalPrice:   int64(float64(data.totalMonthlyPrice) / float64(daysInMonth)),
		VCPUPrice:    calculateDailyUnitPrice(data.cpuPortion, data.totalVCPUs, daysInMonth),
		MemoryPrice:  calculateDailyUnitPrice(data.memoryPortion, data.totalMemoryGB, daysInMonth),
		SSDPrice:     calculateDailyUnitPrice(data.ssdPortion, data.totalSSDGB, daysInMonth),
		HDDPrice:     calculateDailyUnitPrice(data.hddPortion, data.totalHDDGB, daysInMonth),
		GPUPrice:     0,
	}
}

// Helper functions
func normalizeToMonthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func calculateUnitPrice(portion float64, totalUnits int) int64 {
	if totalUnits == 0 {
		return 0
	}
	return int64(portion / float64(totalUnits))
}

func calculateDailyUnitPrice(monthlyPortion float64, totalUnits int, daysInMonth int) int64 {
	if totalUnits == 0 || daysInMonth == 0 {
		return 0
	}
	monthlyUnitPrice := monthlyPortion / float64(totalUnits)
	return int64(monthlyUnitPrice / float64(daysInMonth))
}

// calculateBasePriceDataForMonth calculates base price data for a specific month
func (s *service) calculateBasePriceDataForMonth(inv *inventory.DatacenterWithHardware, monthDate time.Time) *priceCalculationData {
	var totalHardwareCost int64

	for _, server := range inv.Servers {
		if server.AmortMonths > 0 {
			totalHardwareCost += server.PurchasePrice / int64(server.AmortMonths)
		}
	}

	for _, infra := range inv.InfrastructureHardwares {
		if infra.AmortMonths > 0 {
			totalHardwareCost += infra.PurchasePrice / int64(infra.AmortMonths)
		}
	}

	totalMonthlyPrice := totalHardwareCost + inv.DataCenter.MonthlyColoFee

	// Calculate total resources from all servers
	var totalVCPUs, totalMemoryGB, totalSSDGB, totalHDDGB int
	for _, server := range inv.Servers {
		totalVCPUs += server.VCPUs
		totalMemoryGB += server.MemoryGB
		totalSSDGB += server.SSDCapacityGB
		totalHDDGB += server.HDDCapacityGB
	}

	// Calculate price portions based on ratios
	// CPU: 50%, Memory: 35%, SSD: 10%, HDD: 5%
	cpuPortion := float64(totalMonthlyPrice) * 0.50
	memoryPortion := float64(totalMonthlyPrice) * 0.35
	ssdPortion := float64(totalMonthlyPrice) * 0.10
	hddPortion := float64(totalMonthlyPrice) * 0.05

	return &priceCalculationData{
		totalMonthlyPrice: totalMonthlyPrice,
		totalVCPUs:        totalVCPUs,
		totalMemoryGB:     totalMemoryGB,
		totalSSDGB:        totalSSDGB,
		totalHDDGB:        totalHDDGB,
		cpuPortion:        cpuPortion,
		memoryPortion:     memoryPortion,
		ssdPortion:        ssdPortion,
		hddPortion:        hddPortion,
		monthDate:         monthDate,
	}
}

func (s *service) CreateMonthlyPrice() error {
	ctx := context.Background()
	getInvCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	allInventory, err := s.inventory.GetDatacentersWithHardware(getInvCtx, "", "")
	if err != nil {
		s.logger.Warn("service: error getting datacenters inventory", zap.Error(err))
		return err
	}

	s.logger.Debug("service: got datacenters", zap.Any("datacenters", allInventory))

	monthlyPrice, err := s.calculatePerMonthPrice(allInventory)
	if err != nil {
		s.logger.Error("service: error calculating monthly prices", zap.Error(err))
		return err
	}

	s.logger.Debug("Per Month Prices calculated", zap.Any("prices", monthlyPrice))

	// Save prices to database
	saveCtx, saveCancel := context.WithTimeout(ctx, 10*time.Second)
	defer saveCancel()

	err = s.repository.createMonthlyPrice(saveCtx, monthlyPrice)
	if err != nil {
		s.logger.Error("service: error saving monthly prices", zap.Error(err))
		return err
	}

	s.logger.Info("service: successfully created monthly prices", zap.Int("count", len(monthlyPrice)))
	return nil
}

func (s *service) GetMonthlyPrice(datacenterId uuid.UUID, month time.Time) (*Price, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	price, err := s.repository.getMonthlyPrice(ctx, datacenterId, month)
	if err != nil {
		s.logger.Error("service: error getting monthly price", zap.Error(err))
		return nil, err
	}

	return price, nil
}

func (s *service) GetAllPrices() ([]*Price, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	prices, err := s.repository.getAllPrices(ctx)
	if err != nil {
		s.logger.Error("service: error getting all prices", zap.Error(err))
		return nil, err
	}

	return prices, nil
}

type priceCalculationData struct {
	totalMonthlyPrice int64
	totalVCPUs        int
	totalMemoryGB     int
	totalSSDGB        int
	totalHDDGB        int
	cpuPortion        float64
	memoryPortion     float64
	ssdPortion        float64
	hddPortion        float64
	monthDate         time.Time
}

func (s *service) calculateBasePriceData(inv *inventory.DatacenterWithHardware) *priceCalculationData {
	// Calculate total monthly hardware cost from amortization
	var totalHardwareCost int64

	// Add server monthly costs (purchase price / amortization months)
	for _, server := range inv.Servers {
		if server.AmortMonths > 0 {
			totalHardwareCost += server.PurchasePrice / int64(server.AmortMonths)
		}
	}

	// Add infrastructure hardware monthly costs
	for _, infra := range inv.InfrastructureHardwares {
		if infra.AmortMonths > 0 {
			totalHardwareCost += infra.PurchasePrice / int64(infra.AmortMonths)
		}
	}

	// Add colo fee to get total monthly price
	totalMonthlyPrice := totalHardwareCost + inv.DataCenter.MonthlyColoFee

	// Calculate total resources from all servers
	var totalVCPUs, totalMemoryGB, totalSSDGB, totalHDDGB int
	for _, server := range inv.Servers {
		totalVCPUs += server.VCPUs
		totalMemoryGB += server.MemoryGB
		totalSSDGB += server.SSDCapacityGB
		totalHDDGB += server.HDDCapacityGB
	}

	// Calculate price portions based on ratios
	// CPU: 50%, Memory: 35%, SSD: 10%, HDD: 5%
	cpuPortion := float64(totalMonthlyPrice) * 0.50
	memoryPortion := float64(totalMonthlyPrice) * 0.35
	ssdPortion := float64(totalMonthlyPrice) * 0.10
	hddPortion := float64(totalMonthlyPrice) * 0.05

	// Set date to first day of current month for month-based pricing
	now := time.Now()
	monthDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	return &priceCalculationData{
		totalMonthlyPrice: totalMonthlyPrice,
		totalVCPUs:        totalVCPUs,
		totalMemoryGB:     totalMemoryGB,
		totalSSDGB:        totalSSDGB,
		totalHDDGB:        totalHDDGB,
		cpuPortion:        cpuPortion,
		memoryPortion:     memoryPortion,
		ssdPortion:        ssdPortion,
		hddPortion:        hddPortion,
		monthDate:         monthDate,
	}
}

func (s *service) calculatePerMonthPrice(inventories []*inventory.DatacenterWithHardware) ([]*perMonthPrice, error) {
	results := make([]*perMonthPrice, 0, len(inventories))

	for _, inv := range inventories {
		data := s.calculateBasePriceData(inv)

		// Calculate per-unit prices
		var vcpuPrice, memoryPrice, ssdPrice, hddPrice int64

		if data.totalVCPUs > 0 {
			vcpuPrice = int64(data.cpuPortion / float64(data.totalVCPUs))
		}
		if data.totalMemoryGB > 0 {
			memoryPrice = int64(data.memoryPortion / float64(data.totalMemoryGB))
		}
		if data.totalSSDGB > 0 {
			ssdPrice = int64(data.ssdPortion / float64(data.totalSSDGB))
		}
		if data.totalHDDGB > 0 {
			hddPrice = int64(data.hddPortion / float64(data.totalHDDGB))
		}

		result := &perMonthPrice{
			DatacenterID: inv.DataCenter.ID,
			Region:       inv.DataCenter.Region,
			Date:         data.monthDate,
			TotalPrice:   data.totalMonthlyPrice,
			VCPUPrice:    vcpuPrice,
			MemoryPrice:  memoryPrice,
			SSDPrice:     ssdPrice,
			HDDPrice:     hddPrice,
			GPUPrice:     0,
		}

		results = append(results, result)
	}

	return results, nil
}

func (s *service) calculatePerDayPrice(inventories []*inventory.DatacenterWithHardware) ([]*perDayPrice, error) {
	results := make([]*perDayPrice, 0, len(inventories))

	for _, inv := range inventories {
		data := s.calculateBasePriceData(inv)

		// Calculate days in current m.onth
		nextMonth := data.monthDate.AddDate(0, 1, 0)
		daysInMonth := nextMonth.Sub(data.monthDate).Hours() / 24

		// Calculate per-unit daily prices
		var vcpuPrice, memoryPrice, ssdPrice, hddPrice int64

		if data.totalVCPUs > 0 {
			monthlyVCPUPrice := data.cpuPortion / float64(data.totalVCPUs)
			vcpuPrice = int64(monthlyVCPUPrice / daysInMonth)
		}
		if data.totalMemoryGB > 0 {
			monthlyMemoryPrice := data.memoryPortion / float64(data.totalMemoryGB)
			memoryPrice = int64(monthlyMemoryPrice / daysInMonth)
		}
		if data.totalSSDGB > 0 {
			monthlySSDPrice := data.ssdPortion / float64(data.totalSSDGB)
			ssdPrice = int64(monthlySSDPrice / daysInMonth)
		}
		if data.totalHDDGB > 0 {
			monthlyHDDPrice := data.hddPortion / float64(data.totalHDDGB)
			hddPrice = int64(monthlyHDDPrice / daysInMonth)
		}

		// Calculate total daily price
		totalDailyPrice := int64(float64(data.totalMonthlyPrice) / daysInMonth)

		result := &perDayPrice{
			DatacenterID: inv.DataCenter.ID,
			Region:       inv.DataCenter.Region,
			Date:         data.monthDate,
			TotalPrice:   totalDailyPrice,
			VCPUPrice:    vcpuPrice,
			MemoryPrice:  memoryPrice,
			SSDPrice:     ssdPrice,
			HDDPrice:     hddPrice,
			GPUPrice:     0,
		}

		results = append(results, result)
	}

	return results, nil
}
