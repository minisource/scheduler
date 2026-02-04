package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/minisource/scheduler/internal/models"
	"github.com/minisource/scheduler/internal/repository"
)

// ExecutionService handles execution business logic
type ExecutionService struct {
	executionRepo *repository.ExecutionRepository
}

// NewExecutionService creates a new execution service
func NewExecutionService(executionRepo *repository.ExecutionRepository) *ExecutionService {
	return &ExecutionService{
		executionRepo: executionRepo,
	}
}

// GetByID retrieves an execution by ID
func (s *ExecutionService) GetByID(ctx context.Context, id uuid.UUID) (*models.JobExecution, error) {
	return s.executionRepo.FindByID(ctx, id)
}

// List lists executions with filtering
func (s *ExecutionService) List(ctx context.Context, filter models.ExecutionFilter) (*models.ExecutionListResult, error) {
	return s.executionRepo.Query(ctx, filter)
}

// GetByJobID retrieves executions for a job
func (s *ExecutionService) GetByJobID(ctx context.Context, jobID uuid.UUID, limit int) ([]models.JobExecution, error) {
	return s.executionRepo.FindByJobID(ctx, jobID, limit)
}

// Cancel cancels an execution
func (s *ExecutionService) Cancel(ctx context.Context, id uuid.UUID) error {
	return s.executionRepo.CancelExecution(ctx, id)
}

// GetStats retrieves execution statistics
func (s *ExecutionService) GetStats(ctx context.Context, tenantID *uuid.UUID, startTime, endTime time.Time) (map[string]int64, error) {
	return s.executionRepo.GetExecutionStats(ctx, tenantID, startTime, endTime)
}

// GetRunning retrieves running executions
func (s *ExecutionService) GetRunning(ctx context.Context) ([]models.JobExecution, error) {
	return s.executionRepo.FindRunning(ctx)
}

// GetPending retrieves pending executions
func (s *ExecutionService) GetPending(ctx context.Context, before time.Time, limit int) ([]models.JobExecution, error) {
	return s.executionRepo.FindPending(ctx, before, limit)
}
