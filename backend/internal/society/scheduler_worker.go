package society

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SchedulerWorker executes due recurring billing schedules in the background.
type SchedulerWorker struct {
	repo           *Repository
	logger         *slog.Logger
	workerID       string
	stopChan       chan struct{}
	mu             sync.RWMutex
	isRunning      bool
	lastPollAt     *time.Time
	lastExecutionAt *time.Time
}

// SchedulerWorkerAccessor defines the read-only operational telemetry contract for the scheduler worker.
type SchedulerWorkerAccessor interface {
	GetWorkerHealth() (status string, id string, lastPoll, lastExec *time.Time)
}

// NewSchedulerWorker initializes a billing scheduler background worker instance.
func NewSchedulerWorker(repo *Repository, logger *slog.Logger) *SchedulerWorker {
	if logger == nil {
		logger = slog.Default()
	}
	workerID := fmt.Sprintf("sched-worker-%s-%d", uuid.New().String()[:8], time.Now().UnixNano())
	return &SchedulerWorker{
		repo:     repo,
		logger:   logger,
		workerID: workerID,
		stopChan: make(chan struct{}),
	}
}

// GetWorkerHealth returns current operational status and heartbeat timestamps.
func (w *SchedulerWorker) GetWorkerHealth() (string, string, *time.Time, *time.Time) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	status := "STOPPED"
	if w.isRunning {
		status = "RUNNING"
	}
	return status, w.workerID, w.lastPollAt, w.lastExecutionAt
}

// Start initiates the polling loop at the specified interval.
func (w *SchedulerWorker) Start(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.mu.Lock()
	w.isRunning = true
	w.mu.Unlock()

	w.logger.Info("billing scheduler worker started", "worker_id", w.workerID, "interval", interval.String())

	defer func() {
		w.mu.Lock()
		w.isRunning = false
		w.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("billing scheduler worker stopping via context", "worker_id", w.workerID)
			return
		case <-w.stopChan:
			w.logger.Info("billing scheduler worker stopped via stop signal", "worker_id", w.workerID)
			return
		case <-ticker.C:
			w.ProcessDueSchedules(ctx, 5)
		}
	}
}

// Stop signals the worker loop to terminate.
func (w *SchedulerWorker) Stop() {
	close(w.stopChan)
}

// ProcessDueSchedules claims and executes due schedules. Returns the count executed.
func (w *SchedulerWorker) ProcessDueSchedules(ctx context.Context, limit int) int {
	if w.repo == nil {
		return 0
	}

	pollTime := time.Now()
	w.mu.Lock()
	w.lastPollAt = &pollTime
	w.mu.Unlock()

	schedules, err := w.repo.ClaimDueSchedules(ctx, w.workerID, limit)
	if err != nil {
		w.logger.Error("failed to claim due billing schedules", "error", err)
		return 0
	}

	if len(schedules) == 0 {
		return 0
	}

	processedCount := 0
	for _, s := range schedules {
		w.logger.Info("executing due billing schedule", "schedule_id", s.ID, "society_id", s.SocietyID, "name", s.Name)
		exec, execErr := w.repo.ExecuteScheduleRun(
			ctx, s.SocietyID, s.ID, "SCHEDULED", s.CreatedBy, "127.0.0.1", "billing-scheduler-worker",
		)
		if execErr != nil {
			w.logger.Error("failed to execute billing schedule", "schedule_id", s.ID, "error", execErr)
		} else {
			w.logger.Info("billing schedule executed successfully",
				"schedule_id", s.ID, "batch_id", exec.BatchID, "period_name", exec.PeriodName)
			processedCount++
		}
	}

	if processedCount > 0 {
		execTime := time.Now()
		w.mu.Lock()
		w.lastExecutionAt = &execTime
		w.mu.Unlock()
	}

	return processedCount
}
