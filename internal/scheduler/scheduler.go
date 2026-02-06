package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/minisource/scheduler/config"
	"github.com/minisource/scheduler/internal/models"
	"github.com/minisource/scheduler/internal/repository"
	"github.com/robfig/cron/v3"
)

// Scheduler is the core scheduler engine
type Scheduler struct {
	config        *config.Config
	jobRepo       *repository.JobRepository
	executionRepo *repository.ExecutionRepository
	historyRepo   *repository.HistoryRepository
	locker        *DistributedLocker
	executor      *Executor
	workerPool    *WorkerPool
	cronParser    cron.Parser

	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.RWMutex
}

// NewScheduler creates a new scheduler instance
func NewScheduler(
	cfg *config.Config,
	jobRepo *repository.JobRepository,
	executionRepo *repository.ExecutionRepository,
	historyRepo *repository.HistoryRepository,
	locker *DistributedLocker,
) *Scheduler {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	return &Scheduler{
		config:        cfg,
		jobRepo:       jobRepo,
		executionRepo: executionRepo,
		historyRepo:   historyRepo,
		locker:        locker,
		cronParser:    parser,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	s.mu.Unlock()

	// Initialize executor
	s.executor = NewExecutor(s.config, &http.Client{
		Timeout: time.Duration(s.config.Scheduler.LockTTLSeconds) * time.Second,
	})

	// Initialize worker pool
	s.workerPool = NewWorkerPool(s.config.Scheduler.WorkerCount, s.processJob)

	// Start worker pool
	s.workerPool.Start(s.ctx)

	// Start scheduler loops
	s.wg.Add(3)
	go s.schedulerLoop()
	go s.heartbeatLoop()
	go s.cleanupLoop()

	return nil
}

// Stop stops the scheduler gracefully
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	if s.workerPool != nil {
		s.workerPool.Stop()
	}

	s.wg.Wait()
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// schedulerLoop is the main scheduling loop
func (s *Scheduler) schedulerLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processScheduledJobs()
		}
	}
}

// processScheduledJobs processes jobs that are due
func (s *Scheduler) processScheduledJobs() {
	// Try to acquire leader lock
	lockKey := "scheduler:leader"
	acquired, err := s.locker.AcquireLock(s.ctx, lockKey, time.Duration(s.config.Scheduler.LockTTLSeconds)*time.Second)
	if err != nil || !acquired {
		return // Another instance is the leader
	}
	defer s.locker.ReleaseLock(s.ctx, lockKey)

	// Find jobs due for execution
	jobs, err := s.jobRepo.FindJobsDueForExecution(s.ctx, time.Now(), 100)
	if err != nil {
		return
	}

	for _, job := range jobs {
		// Create execution record
		execution := &models.JobExecution{
			ID:          uuid.New(),
			JobID:       job.ID,
			TenantID:    job.TenantID,
			Status:      models.ExecutionStatusPending,
			ScheduledAt: time.Now(),
			Attempt:     1,
		}

		if err := s.executionRepo.Create(s.ctx, execution); err != nil {
			continue
		}

		// Calculate next run time
		nextRunAt, err := s.CalculateNextRun(&job)
		if err == nil && nextRunAt != nil {
			s.jobRepo.UpdateNextRunAt(s.ctx, job.ID, *nextRunAt)
		}

		// Submit to worker pool
		s.workerPool.Submit(JobTask{
			Job:       job,
			Execution: *execution,
		})
	}
}

// processJob processes a single job execution
func (s *Scheduler) processJob(task JobTask) {
	ctx, cancel := context.WithTimeout(s.ctx, time.Duration(task.Job.Timeout)*time.Second)
	defer cancel()

	workerID := fmt.Sprintf("worker-%s", uuid.New().String()[:8])

	// Mark as running
	if err := s.executionRepo.MarkAsRunning(ctx, task.Execution.ID, workerID); err != nil {
		return
	}

	// Execute the job
	result, err := s.executor.Execute(ctx, &task.Job)

	if err != nil {
		s.handleExecutionFailure(ctx, &task, err, result)
		return
	}

	// Mark as completed
	var response []byte
	if result != nil {
		response = result.Body
	}

	statusCode := 0
	if result != nil {
		statusCode = result.StatusCode
	}

	if err := s.executionRepo.MarkAsCompleted(ctx, task.Execution.ID, statusCode, response); err != nil {
		return
	}

	// Update job counters
	s.jobRepo.UpdateLastRunAt(ctx, task.Job.ID, true)

	// Update history
	if result != nil {
		s.historyRepo.IncrementSuccess(ctx, task.Job.ID, time.Now(), result.Duration)
	}
}

// handleExecutionFailure handles a failed execution
func (s *Scheduler) handleExecutionFailure(ctx context.Context, task *JobTask, err error, result *ExecutionResult) {
	errMsg := err.Error()

	var statusCode *int
	if result != nil {
		statusCode = &result.StatusCode
	}

	// Check if we should retry
	if task.Execution.Attempt < task.Job.MaxRetries {
		s.executionRepo.MarkAsRetrying(ctx, task.Execution.ID, errMsg)

		// Schedule retry
		retryDelay := time.Duration(s.config.Scheduler.RetryDelaySeconds) * time.Second
		time.AfterFunc(retryDelay, func() {
			task.Execution.Attempt++
			s.workerPool.Submit(*task)
		})
		return
	}

	// Max retries exceeded
	s.executionRepo.MarkAsFailed(ctx, task.Execution.ID, errMsg, statusCode)
	s.jobRepo.UpdateLastRunAt(ctx, task.Job.ID, false)
	s.historyRepo.IncrementFailure(ctx, task.Job.ID, time.Now())
}

// heartbeatLoop maintains scheduler heartbeat
func (s *Scheduler) heartbeatLoop() {
	defer s.wg.Done()

	interval := time.Duration(s.config.Scheduler.HeartbeatSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.locker.RefreshLock(s.ctx, "scheduler:leader", time.Duration(s.config.Scheduler.LockTTLSeconds)*time.Second)
		}
	}
}

// cleanupLoop cleans up old data periodically
func (s *Scheduler) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup removes old execution records
func (s *Scheduler) cleanup() {
	cutoff := time.Now().AddDate(0, 0, -s.config.Scheduler.CleanupDays)
	s.executionRepo.CleanupOld(s.ctx, cutoff)
	s.historyRepo.CleanupOld(s.ctx, cutoff)
}

// CalculateNextRun calculates the next run time for a job
func (s *Scheduler) CalculateNextRun(job *models.Job) (*time.Time, error) {
	now := time.Now()

	switch job.Type {
	case models.JobTypeCron:
		schedule, err := s.cronParser.Parse(job.Schedule)
		if err != nil {
			return nil, fmt.Errorf("invalid cron expression: %w", err)
		}
		next := schedule.Next(now)
		return &next, nil

	case models.JobTypeInterval:
		var interval int
		if err := json.Unmarshal([]byte(job.Schedule), &interval); err != nil {
			return nil, fmt.Errorf("invalid interval: %w", err)
		}
		next := now.Add(time.Duration(interval) * time.Second)
		return &next, nil

	case models.JobTypeOneTime:
		// One-time jobs don't have a next run
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown job type: %s", job.Type)
	}
}

// TriggerJob manually triggers a job
func (s *Scheduler) TriggerJob(ctx context.Context, jobID uuid.UUID) (*models.JobExecution, error) {
	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return nil, err
	}

	execution := &models.JobExecution{
		ID:          uuid.New(),
		JobID:       job.ID,
		TenantID:    job.TenantID,
		Status:      models.ExecutionStatusPending,
		ScheduledAt: time.Now(),
		Attempt:     1,
	}

	if err := s.executionRepo.Create(ctx, execution); err != nil {
		return nil, err
	}

	// Submit to worker pool
	s.workerPool.Submit(JobTask{
		Job:       *job,
		Execution: *execution,
	})

	return execution, nil
}
