package repository

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minisource/scheduler/internal/models"
	"gorm.io/gorm"
)

// JobRepository handles job persistence
type JobRepository struct {
	db *gorm.DB
}

// NewJobRepository creates a new job repository
func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

// Create creates a new job
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
	return r.db.WithContext(ctx).Create(job).Error
}

// Update updates a job
func (r *JobRepository) Update(ctx context.Context, job *models.Job) error {
	return r.db.WithContext(ctx).Save(job).Error
}

// FindByID retrieves a job by ID
func (r *JobRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	var job models.Job
	err := r.db.WithContext(ctx).First(&job, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// FindByTenantAndID retrieves a job by tenant and ID
func (r *JobRepository) FindByTenantAndID(ctx context.Context, tenantID, id uuid.UUID) (*models.Job, error) {
	var job models.Job
	err := r.db.WithContext(ctx).First(&job, "id = ? AND tenant_id = ?", id, tenantID).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// Query finds jobs matching the filter
func (r *JobRepository) Query(ctx context.Context, filter models.JobFilter) (*models.JobListResult, error) {
	var jobs []models.Job
	var total int64

	query := r.buildJobQuery(filter)

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
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&jobs).Error
	if err != nil {
		return nil, err
	}

	return &models.JobListResult{
		Jobs:       jobs,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
		HasMore:    int64((page)*pageSize) < total,
	}, nil
}

// buildJobQuery creates the GORM query from filter
func (r *JobRepository) buildJobQuery(filter models.JobFilter) *gorm.DB {
	query := r.db.Model(&models.Job{})

	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	} else {
		// Exclude deleted jobs by default
		query = query.Where("status != ?", models.JobStatusDeleted)
	}

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if filter.Name != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(filter.Name)+"%")
	}

	return query
}

// FindActiveJobs finds all active jobs
func (r *JobRepository) FindActiveJobs(ctx context.Context) ([]models.Job, error) {
	var jobs []models.Job
	err := r.db.WithContext(ctx).
		Where("status = ?", models.JobStatusActive).
		Find(&jobs).Error
	return jobs, err
}

// FindJobsDueForExecution finds jobs that are due to run
func (r *JobRepository) FindJobsDueForExecution(ctx context.Context, before time.Time, limit int) ([]models.Job, error) {
	var jobs []models.Job
	err := r.db.WithContext(ctx).
		Where("status = ?", models.JobStatusActive).
		Where("next_run_at <= ?", before).
		Order("priority DESC, next_run_at ASC").
		Limit(limit).
		Find(&jobs).Error
	return jobs, err
}

// UpdateNextRunAt updates the next run time for a job
func (r *JobRepository) UpdateNextRunAt(ctx context.Context, id uuid.UUID, nextRunAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.Job{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"next_run_at": nextRunAt,
			"updated_at":  time.Now(),
		}).Error
}

// UpdateLastRunAt updates the last run time and counters
func (r *JobRepository) UpdateLastRunAt(ctx context.Context, id uuid.UUID, success bool) error {
	updates := map[string]interface{}{
		"last_run_at": time.Now(),
		"updated_at":  time.Now(),
	}

	if success {
		updates["run_count"] = gorm.Expr("run_count + 1")
	} else {
		updates["fail_count"] = gorm.Expr("fail_count + 1")
	}

	return r.db.WithContext(ctx).
		Model(&models.Job{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateStatus updates job status
func (r *JobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.JobStatus) error {
	return r.db.WithContext(ctx).
		Model(&models.Job{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// Delete soft-deletes a job
func (r *JobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&models.Job{}).
		Where("id = ?", id).
		Update("status", models.JobStatusDeleted).Error
}

// GetStats retrieves job statistics
func (r *JobRepository) GetStats(ctx context.Context, tenantID *uuid.UUID) (*models.JobStats, error) {
	stats := &models.JobStats{
		JobsByType:   make(map[models.JobType]int64),
		JobsByStatus: make(map[models.JobStatus]int64),
	}

	query := r.db.WithContext(ctx).Model(&models.Job{})
	if tenantID != nil {
		query = query.Where("tenant_id = ?", tenantID)
	}

	// Total jobs (excluding deleted)
	query.Where("status != ?", models.JobStatusDeleted).Count(&stats.TotalJobs)

	// Active jobs
	r.db.Model(&models.Job{}).Where("status = ?", models.JobStatusActive).Count(&stats.ActiveJobs)

	// Paused jobs
	r.db.Model(&models.Job{}).Where("status = ?", models.JobStatusPaused).Count(&stats.PausedJobs)

	// Jobs by type
	var typeResults []struct {
		Type  models.JobType
		Count int64
	}
	r.db.Model(&models.Job{}).
		Select("type, COUNT(*) as count").
		Where("status != ?", models.JobStatusDeleted).
		Group("type").Scan(&typeResults)

	for _, tr := range typeResults {
		stats.JobsByType[tr.Type] = tr.Count
	}

	// Jobs by status
	var statusResults []struct {
		Status models.JobStatus
		Count  int64
	}
	r.db.Model(&models.Job{}).
		Select("status, COUNT(*) as count").
		Group("status").Scan(&statusResults)

	for _, sr := range statusResults {
		stats.JobsByStatus[sr.Status] = sr.Count
	}

	return stats, nil
}
