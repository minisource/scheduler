package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/minisource/scheduler/internal/models"
	"gorm.io/gorm"
)

// HistoryRepository handles job history persistence
type HistoryRepository struct {
	db *gorm.DB
}

// NewHistoryRepository creates a new history repository
func NewHistoryRepository(db *gorm.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

// Upsert creates or updates a history record
func (r *HistoryRepository) Upsert(ctx context.Context, history *models.JobHistory) error {
	return r.db.WithContext(ctx).
		Where("job_id = ? AND date = ?", history.JobID, history.Date).
		Assign(*history).
		FirstOrCreate(history).Error
}

// IncrementSuccess increments the success count for a job on a date
func (r *HistoryRepository) IncrementSuccess(ctx context.Context, jobID uuid.UUID, date time.Time, duration int64) error {
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	
	var history models.JobHistory
	err := r.db.WithContext(ctx).
		Where("job_id = ? AND date = ?", jobID, dateOnly).
		First(&history).Error

	if err == gorm.ErrRecordNotFound {
		history = models.JobHistory{
			ID:            uuid.New(),
			JobID:         jobID,
			Date:          dateOnly,
			SuccessCount:  1,
			TotalDuration: duration,
			MinDuration:   duration,
			MaxDuration:   duration,
		}
		return r.db.WithContext(ctx).Create(&history).Error
	}

	if err != nil {
		return err
	}

	// Update statistics
	newCount := history.SuccessCount + 1
	totalDuration := history.TotalDuration + duration
	avgDuration := float64(totalDuration) / float64(newCount+history.FailureCount)

	minDuration := history.MinDuration
	if duration < minDuration || minDuration == 0 {
		minDuration = duration
	}

	maxDuration := history.MaxDuration
	if duration > maxDuration {
		maxDuration = duration
	}

	return r.db.WithContext(ctx).
		Model(&models.JobHistory{}).
		Where("id = ?", history.ID).
		Updates(map[string]interface{}{
			"success_count":  newCount,
			"total_duration": totalDuration,
			"avg_duration":   avgDuration,
			"min_duration":   minDuration,
			"max_duration":   maxDuration,
		}).Error
}

// IncrementFailure increments the failure count for a job on a date
func (r *HistoryRepository) IncrementFailure(ctx context.Context, jobID uuid.UUID, date time.Time) error {
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	var history models.JobHistory
	err := r.db.WithContext(ctx).
		Where("job_id = ? AND date = ?", jobID, dateOnly).
		First(&history).Error

	if err == gorm.ErrRecordNotFound {
		history = models.JobHistory{
			ID:           uuid.New(),
			JobID:        jobID,
			Date:         dateOnly,
			FailureCount: 1,
		}
		return r.db.WithContext(ctx).Create(&history).Error
	}

	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&models.JobHistory{}).
		Where("id = ?", history.ID).
		Update("failure_count", gorm.Expr("failure_count + 1")).Error
}

// FindByJobID retrieves history records for a job
func (r *HistoryRepository) FindByJobID(ctx context.Context, jobID uuid.UUID, days int) ([]models.JobHistory, error) {
	var history []models.JobHistory
	startDate := time.Now().AddDate(0, 0, -days)
	
	err := r.db.WithContext(ctx).
		Where("job_id = ? AND date >= ?", jobID, startDate).
		Order("date DESC").
		Find(&history).Error
	return history, err
}

// FindByDateRange retrieves history records for a date range
func (r *HistoryRepository) FindByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.JobHistory, error) {
	var history []models.JobHistory
	err := r.db.WithContext(ctx).
		Where("date >= ? AND date <= ?", startDate, endDate).
		Order("date DESC, job_id").
		Find(&history).Error
	return history, err
}

// GetAggregatedStats gets aggregated statistics for a period
func (r *HistoryRepository) GetAggregatedStats(ctx context.Context, jobID *uuid.UUID, startDate, endDate time.Time) (*models.AggregatedHistoryStats, error) {
	query := r.db.WithContext(ctx).Model(&models.JobHistory{}).
		Where("date >= ? AND date <= ?", startDate, endDate)

	if jobID != nil {
		query = query.Where("job_id = ?", jobID)
	}

	var result struct {
		TotalSuccess   int64
		TotalFailure   int64
		TotalDuration  int64
		MinDuration    int64
		MaxDuration    int64
	}

	err := query.Select(`
		COALESCE(SUM(success_count), 0) as total_success,
		COALESCE(SUM(failure_count), 0) as total_failure,
		COALESCE(SUM(total_duration), 0) as total_duration,
		COALESCE(MIN(min_duration), 0) as min_duration,
		COALESCE(MAX(max_duration), 0) as max_duration
	`).Scan(&result).Error

	if err != nil {
		return nil, err
	}

	totalExecutions := result.TotalSuccess + result.TotalFailure
	var avgDuration float64
	if totalExecutions > 0 {
		avgDuration = float64(result.TotalDuration) / float64(totalExecutions)
	}

	stats := &models.AggregatedHistoryStats{
		TotalSuccess:  result.TotalSuccess,
		TotalFailure:  result.TotalFailure,
		TotalDuration: result.TotalDuration,
		AvgDuration:   avgDuration,
		MinDuration:   result.MinDuration,
		MaxDuration:   result.MaxDuration,
	}

	if totalExecutions > 0 {
		stats.SuccessRate = float64(result.TotalSuccess) / float64(totalExecutions) * 100
	}

	return stats, nil
}

// CleanupOld removes old history records
func (r *HistoryRepository) CleanupOld(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("date < ?", before).
		Delete(&models.JobHistory{})
	return result.RowsAffected, result.Error
}
