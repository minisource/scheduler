package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/minisource/scheduler/internal/models"
	"gorm.io/gorm"
)

// ExecutionRepository handles job execution persistence
type ExecutionRepository struct {
	db *gorm.DB
}

// NewExecutionRepository creates a new execution repository
func NewExecutionRepository(db *gorm.DB) *ExecutionRepository {
	return &ExecutionRepository{db: db}
}

// Create creates a new execution record
func (r *ExecutionRepository) Create(ctx context.Context, execution *models.JobExecution) error {
	return r.db.WithContext(ctx).Create(execution).Error
}

// Update updates an execution record
func (r *ExecutionRepository) Update(ctx context.Context, execution *models.JobExecution) error {
	return r.db.WithContext(ctx).Save(execution).Error
}

// FindByID retrieves an execution by ID
func (r *ExecutionRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.JobExecution, error) {
	var execution models.JobExecution
	err := r.db.WithContext(ctx).First(&execution, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// Query finds executions matching the filter
func (r *ExecutionRepository) Query(ctx context.Context, filter models.ExecutionFilter) (*models.ExecutionListResult, error) {
	var executions []models.JobExecution
	var total int64

	query := r.buildQuery(filter)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	err := query.Order("scheduled_at DESC").Offset(offset).Limit(pageSize).Find(&executions).Error
	if err != nil {
		return nil, err
	}

	return &models.ExecutionListResult{
		Executions: executions,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
		HasMore:    int64((page)*pageSize) < total,
	}, nil
}

// buildQuery creates the GORM query from filter
func (r *ExecutionRepository) buildQuery(filter models.ExecutionFilter) *gorm.DB {
	query := r.db.Model(&models.JobExecution{})

	if filter.JobID != nil {
		query = query.Where("job_id = ?", filter.JobID)
	}

	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.StartTime != nil {
		query = query.Where("scheduled_at >= ?", filter.StartTime)
	}

	if filter.EndTime != nil {
		query = query.Where("scheduled_at <= ?", filter.EndTime)
	}

	return query
}

// FindByJobID retrieves executions for a job
func (r *ExecutionRepository) FindByJobID(ctx context.Context, jobID uuid.UUID, limit int) ([]models.JobExecution, error) {
	var executions []models.JobExecution
	err := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("scheduled_at DESC").
		Limit(limit).
		Find(&executions).Error
	return executions, err
}

// FindPending finds pending executions
func (r *ExecutionRepository) FindPending(ctx context.Context, before time.Time, limit int) ([]models.JobExecution, error) {
	var executions []models.JobExecution
	err := r.db.WithContext(ctx).
		Where("status = ?", models.ExecutionStatusPending).
		Where("scheduled_at <= ?", before).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&executions).Error
	return executions, err
}

// FindRunning finds running executions
func (r *ExecutionRepository) FindRunning(ctx context.Context) ([]models.JobExecution, error) {
	var executions []models.JobExecution
	err := r.db.WithContext(ctx).
		Where("status = ?", models.ExecutionStatusRunning).
		Find(&executions).Error
	return executions, err
}

// MarkAsRunning marks an execution as running
func (r *ExecutionRepository) MarkAsRunning(ctx context.Context, id uuid.UUID, workerID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.JobExecution{}).
		Where("id = ?", id).
		Where("status = ?", models.ExecutionStatusPending).
		Updates(map[string]interface{}{
			"status":     models.ExecutionStatusRunning,
			"started_at": now,
			"worker_id":  workerID,
			"updated_at": now,
		}).Error
}

// MarkAsCompleted marks an execution as completed
func (r *ExecutionRepository) MarkAsCompleted(ctx context.Context, id uuid.UUID, statusCode int, response []byte) error {
	now := time.Now()

	var execution models.JobExecution
	if err := r.db.WithContext(ctx).First(&execution, "id = ?", id).Error; err != nil {
		return err
	}

	var duration int64
	if execution.StartedAt != nil {
		duration = now.Sub(*execution.StartedAt).Milliseconds()
	}

	return r.db.WithContext(ctx).
		Model(&models.JobExecution{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       models.ExecutionStatusCompleted,
			"completed_at": now,
			"duration":     duration,
			"status_code":  statusCode,
			"response":     response,
			"updated_at":   now,
		}).Error
}

// MarkAsFailed marks an execution as failed
func (r *ExecutionRepository) MarkAsFailed(ctx context.Context, id uuid.UUID, errMsg string, statusCode *int) error {
	now := time.Now()

	var execution models.JobExecution
	if err := r.db.WithContext(ctx).First(&execution, "id = ?", id).Error; err != nil {
		return err
	}

	var duration int64
	if execution.StartedAt != nil {
		duration = now.Sub(*execution.StartedAt).Milliseconds()
	}

	updates := map[string]interface{}{
		"status":       models.ExecutionStatusFailed,
		"completed_at": now,
		"duration":     duration,
		"error":        errMsg,
		"updated_at":   now,
	}

	if statusCode != nil {
		updates["status_code"] = *statusCode
	}

	return r.db.WithContext(ctx).
		Model(&models.JobExecution{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// MarkAsRetrying marks an execution for retry
func (r *ExecutionRepository) MarkAsRetrying(ctx context.Context, id uuid.UUID, errMsg string) error {
	return r.db.WithContext(ctx).
		Model(&models.JobExecution{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     models.ExecutionStatusRetrying,
			"error":      errMsg,
			"attempt":    gorm.Expr("attempt + 1"),
			"updated_at": time.Now(),
		}).Error
}

// CancelExecution cancels an execution
func (r *ExecutionRepository) CancelExecution(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.JobExecution{}).
		Where("id = ?", id).
		Where("status IN ?", []models.ExecutionStatus{models.ExecutionStatusPending, models.ExecutionStatusRunning}).
		Updates(map[string]interface{}{
			"status":       models.ExecutionStatusCancelled,
			"completed_at": time.Now(),
			"updated_at":   time.Now(),
		}).Error
}

// CleanupOld removes old execution records
func (r *ExecutionRepository) CleanupOld(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Where("status IN ?", []models.ExecutionStatus{
			models.ExecutionStatusCompleted,
			models.ExecutionStatusFailed,
			models.ExecutionStatusCancelled,
		}).
		Delete(&models.JobExecution{})
	return result.RowsAffected, result.Error
}

// GetExecutionStats gets execution statistics for a time period
func (r *ExecutionRepository) GetExecutionStats(ctx context.Context, tenantID *uuid.UUID, startTime, endTime time.Time) (map[string]int64, error) {
	stats := make(map[string]int64)

	query := r.db.WithContext(ctx).Model(&models.JobExecution{}).
		Where("scheduled_at >= ? AND scheduled_at <= ?", startTime, endTime)

	if tenantID != nil {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// Total executions
	var total int64
	query.Count(&total)
	stats["total"] = total

	// By status
	for _, status := range []models.ExecutionStatus{
		models.ExecutionStatusCompleted,
		models.ExecutionStatusFailed,
		models.ExecutionStatusCancelled,
	} {
		var count int64
		r.db.Model(&models.JobExecution{}).
			Where("scheduled_at >= ? AND scheduled_at <= ?", startTime, endTime).
			Where("status = ?", status).
			Count(&count)
		stats[string(status)] = count
	}

	return stats, nil
}
