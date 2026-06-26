package pricing

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const (
	// dailyScheduleHour and dailyScheduleMinute define when daily prices are calculated (00:30 local time).
	dailyScheduleHour   = 0
	dailyScheduleMinute = 30

	// monthlyScheduleHour and monthlyScheduleMinute define when monthly prices are finalized
	// on the first day of each month (00:30 local time, for the month that just ended).
	monthlyScheduleHour   = 0
	monthlyScheduleMinute = 30

	runTimeout = 10 * time.Minute
)

// DailyScheduler fills per-datacenter daily unit prices once per day at 00:30 local time.
type DailyScheduler struct {
	svc    Service
	logger *zap.Logger
}

// NewDailyScheduler builds a DailyScheduler for the given pricing service.
func NewDailyScheduler(svc Service, logger *zap.Logger) *DailyScheduler {
	return &DailyScheduler{svc: svc, logger: logger}
}

// Start blocks and runs the daily price fill at 00:30 until ctx is cancelled.
// It is intended to be launched in its own goroutine.
func (s *DailyScheduler) Start(ctx context.Context) {
	s.logger.Info("daily pricing scheduler started",
		zap.Int("hour", dailyScheduleHour),
		zap.Int("minute", dailyScheduleMinute),
	)

	for {
		next := nextDailyRun(time.Now())
		s.logger.Info("daily pricing scheduler waiting for next run", zap.Time("next_run", next))

		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Info("daily pricing scheduler stopped")
			return
		case <-timer.C:
			s.runOnce(ctx)
		}
	}
}

// RunOnStartup fills any missing daily prices through today. Call once before serving traffic.
func (s *DailyScheduler) RunOnStartup(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	through := normalizeToDayStart(time.Now())
	if err := s.svc.EnsureDailyPricesFilled(runCtx, through); err != nil {
		s.logger.Error("startup daily price fill failed", zap.Error(err))
		return
	}
	s.logger.Info("startup daily price fill completed", zap.Time("through", through))
}

func (s *DailyScheduler) runOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	through := normalizeToDayStart(time.Now())
	if err := s.svc.EnsureDailyPricesFilled(runCtx, through); err != nil {
		s.logger.Error("scheduled daily price fill failed", zap.Error(err))
		return
	}
	s.logger.Info("scheduled daily price fill completed", zap.Time("through", through))
}

func nextDailyRun(now time.Time) time.Time {
	loc := now.Location()
	next := time.Date(now.Year(), now.Month(), now.Day(), dailyScheduleHour, dailyScheduleMinute, 0, 0, loc)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// MonthlyScheduler finalizes monthly prices on the first day of each month at 00:30 local time,
// covering the calendar month that just ended.
type MonthlyScheduler struct {
	svc    Service
	logger *zap.Logger
}

// NewMonthlyScheduler builds a MonthlyScheduler for the given pricing service.
func NewMonthlyScheduler(svc Service, logger *zap.Logger) *MonthlyScheduler {
	return &MonthlyScheduler{svc: svc, logger: logger}
}

// Start blocks and runs the monthly price fill on the first of each month until ctx is cancelled.
// It is intended to be launched in its own goroutine.
func (s *MonthlyScheduler) Start(ctx context.Context) {
	s.logger.Info("monthly pricing scheduler started",
		zap.Int("hour", monthlyScheduleHour),
		zap.Int("minute", monthlyScheduleMinute),
	)

	for {
		next := nextMonthlyRun(time.Now())
		s.logger.Info("monthly pricing scheduler waiting for next run", zap.Time("next_run", next))

		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Info("monthly pricing scheduler stopped")
			return
		case <-timer.C:
			s.runOnce(ctx)
		}
	}
}

func (s *MonthlyScheduler) runOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	month := previousMonthStart(time.Now())
	if err := s.svc.EnsureMonthlyPricesForMonth(runCtx, month); err != nil {
		s.logger.Error("scheduled monthly price fill failed",
			zap.Time("month", month),
			zap.Error(err))
		return
	}
	s.logger.Info("scheduled monthly price fill completed", zap.Time("month", month))
}

func nextMonthlyRun(now time.Time) time.Time {
	loc := now.Location()
	year, month, _ := now.Date()
	firstThisMonth := time.Date(year, month, 1, monthlyScheduleHour, monthlyScheduleMinute, 0, 0, loc)
	if now.Before(firstThisMonth) {
		return firstThisMonth
	}
	return firstThisMonth.AddDate(0, 1, 0)
}

func previousMonthStart(now time.Time) time.Time {
	loc := now.Location()
	year, month, _ := now.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, loc).AddDate(0, -1, 0)
}
