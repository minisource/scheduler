package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/minisource/scheduler/internal/models"
	"github.com/minisource/scheduler/internal/repository"
)

// HistoryService handles history business logic
type HistoryService struct {
	historyRepo *repository.HistoryRepository
}

// NewHistoryService creates a new history service
func NewHistoryService(historyRepo *repository.HistoryRepository) *HistoryService {
	return &HistoryService{
		historyRepo: historyRepo,
	}
}

// GetByJobID retrieves history for a job
func (s *HistoryService) GetByJobID(ctx context.Context, jobID uuid.UUID, days int) ([]models.JobHistory, error) {
	return s.historyRepo.FindByJobID(ctx, jobID, days)
}

// GetByDateRange retrieves history for a date range
func (s *HistoryService) GetByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.JobHistory, error) {
	return s.historyRepo.FindByDateRange(ctx, startDate, endDate)
}

// GetAggregated retrieves aggregated history stats
func (s *HistoryService) GetAggregated(ctx context.Context, jobID *uuid.UUID, startDate, endDate time.Time) (*models.AggregatedHistoryStats, error) {
	return s.historyRepo.GetAggregatedStats(ctx, jobID, startDate, endDate)
}

// RecordSuccess records a successful execution in history
func (s *HistoryService) RecordSuccess(ctx context.Context, jobID uuid.UUID, date time.Time, duration int64) error {
	return s.historyRepo.IncrementSuccess(ctx, jobID, date, duration)
}

// RecordFailure records a failed execution in history
func (s *HistoryService) RecordFailure(ctx context.Context, jobID uuid.UUID, date time.Time) error {
	return s.historyRepo.IncrementFailure(ctx, jobID, date)
}

// Cleanup removes old history records
func (s *HistoryService) Cleanup(ctx context.Context, before time.Time) (int64, error) {
	return s.historyRepo.CleanupOld(ctx, before)
}
