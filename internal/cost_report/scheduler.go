package cost_report

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const (
	// scheduleHour and scheduleMinute define when the daily calculation runs (01:00 local time).
	scheduleHour   = 1
	scheduleMinute = 0

	// runTimeout bounds a single scheduled calculation run.
	runTimeout = 10 * time.Minute
)

// Scheduler triggers the daily service cost calculation once per day at a fixed time.
type Scheduler struct {
	svc    Service
	logger *zap.Logger
}

// NewScheduler builds a Scheduler that drives the given cost report service.
func NewScheduler(svc Service, logger *zap.Logger) *Scheduler {
	return &Scheduler{svc: svc, logger: logger}
}

// Start blocks and runs the daily calculation at scheduleHour:scheduleMinute until ctx is cancelled.
// It is intended to be launched in its own goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("cost report scheduler started",
		zap.Int("hour", scheduleHour),
		zap.Int("minute", scheduleMinute),
	)

	for {
		next := nextRun(time.Now())
		s.logger.Info("cost report scheduler waiting for next run", zap.Time("next_run", next))

		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			s.logger.Info("cost report scheduler stopped")
			return
		case <-timer.C:
			s.runOnce(ctx)
		}
	}
}

// runOnce executes a single daily calculation with its own bounded context.
func (s *Scheduler) runOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	if err := s.svc.CalculateDailyServiceCosts(runCtx, time.Now()); err != nil {
		s.logger.Error("scheduled daily cost calculation failed", zap.Error(err))
		return
	}
	s.logger.Info("scheduled daily cost calculation completed")
}

// nextRun returns the next occurrence of scheduleHour:scheduleMinute strictly after now.
func nextRun(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), scheduleHour, scheduleMinute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
