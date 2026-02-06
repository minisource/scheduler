package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/minisource/scheduler/internal/models"
	"github.com/minisource/scheduler/internal/repository"
	"github.com/minisource/scheduler/internal/scheduler"
	"github.com/robfig/cron/v3"
)

// JobService handles job business logic
type JobService struct {
	jobRepo    *repository.JobRepository
	scheduler  *scheduler.Scheduler
	cronParser cron.Parser
}

// NewJobService creates a new job service
func NewJobService(
	jobRepo *repository.JobRepository,
	sched *scheduler.Scheduler,
) *JobService {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	return &JobService{
		jobRepo:    jobRepo,
		scheduler:  sched,
		cronParser: parser,
	}
}

// Create creates a new job
func (s *JobService) Create(ctx context.Context, tenantID uuid.UUID, req *models.CreateJobRequest) (*models.Job, error) {
	// Validate job type and schedule
	if err := s.validateSchedule(req.Type, req.Schedule); err != nil {
		return nil, err
	}

	// Parse headers
	var headers json.RawMessage
	if req.Headers != nil {
		h, _ := json.Marshal(req.Headers)
		headers = h
	}

	// Parse payload
	var payload json.RawMessage
	if req.Payload != nil {
		p, _ := json.Marshal(req.Payload)
		payload = p
	}

	// Parse metadata
	var metadata json.RawMessage
	if req.Metadata != nil {
		m, _ := json.Marshal(req.Metadata)
		metadata = m
	}

	// Set defaults
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30
	}

	maxRetries := req.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	priority := req.Priority
	if priority == 0 {
		priority = 5
	}

	method := req.Method
	if method == "" {
		method = "POST"
	}

	job := &models.Job{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Status:      models.JobStatusActive,
		Schedule:    req.Schedule,
		Timezone:    req.Timezone,
		Endpoint:    req.Endpoint,
		Method:      method,
		Headers:     headers,
		Payload:     payload,
		Timeout:     timeout,
		MaxRetries:  maxRetries,
		Priority:    priority,
		Tags:        req.Tags,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Calculate next run time
	nextRunAt, err := s.calculateNextRun(job)
	if err == nil && nextRunAt != nil {
		job.NextRunAt = nextRunAt
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return job, nil
}

// GetByID retrieves a job by ID
func (s *JobService) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Job, error) {
	return s.jobRepo.FindByTenantAndID(ctx, tenantID, id)
}

// List lists jobs with filtering
func (s *JobService) List(ctx context.Context, filter models.JobFilter) (*models.JobListResult, error) {
	return s.jobRepo.Query(ctx, filter)
}

// Update updates a job
func (s *JobService) Update(ctx context.Context, tenantID, id uuid.UUID, req *models.UpdateJobRequest) (*models.Job, error) {
	job, err := s.jobRepo.FindByTenantAndID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil && *req.Name != "" {
		job.Name = *req.Name
	}
	if req.Description != nil && *req.Description != "" {
		job.Description = *req.Description
	}
	if req.Schedule != nil && *req.Schedule != "" {
		if err := s.validateSchedule(job.Type, *req.Schedule); err != nil {
			return nil, err
		}
		job.Schedule = *req.Schedule
	}
	if req.Endpoint != nil && *req.Endpoint != "" {
		job.Endpoint = *req.Endpoint
	}
	if req.Method != nil && *req.Method != "" {
		job.Method = *req.Method
	}
	if req.Headers != nil {
		job.Headers = *req.Headers
	}
	if req.Payload != nil {
		job.Payload = *req.Payload
	}
	if req.Timeout != nil && *req.Timeout > 0 {
		job.Timeout = *req.Timeout
	}
	if req.MaxRetries != nil && *req.MaxRetries > 0 {
		job.MaxRetries = *req.MaxRetries
	}
	if req.Priority != nil && *req.Priority > 0 {
		job.Priority = *req.Priority
	}
	if req.Tags != nil {
		job.Tags = *req.Tags
	}

	job.UpdatedAt = time.Now()

	// Recalculate next run time if schedule changed
	if req.Schedule != nil && *req.Schedule != "" {
		nextRunAt, err := s.calculateNextRun(job)
		if err == nil && nextRunAt != nil {
			job.NextRunAt = nextRunAt
		}
	}

	if err := s.jobRepo.Update(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	return job, nil
}

// Delete soft-deletes a job
func (s *JobService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	job, err := s.jobRepo.FindByTenantAndID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	return s.jobRepo.Delete(ctx, job.ID)
}

// Trigger manually triggers a job
func (s *JobService) Trigger(ctx context.Context, tenantID, id uuid.UUID) (*models.JobExecution, error) {
	job, err := s.jobRepo.FindByTenantAndID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if job.Status != models.JobStatusActive && job.Status != models.JobStatusPaused {
		return nil, fmt.Errorf("job cannot be triggered in status: %s", job.Status)
	}

	return s.scheduler.TriggerJob(ctx, job.ID)
}

// UpdateStatus updates job status
func (s *JobService) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status models.JobStatus) (*models.Job, error) {
	job, err := s.jobRepo.FindByTenantAndID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	job.Status = status
	job.UpdatedAt = time.Now()

	if err := s.jobRepo.Update(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}

// GetStats retrieves job statistics
func (s *JobService) GetStats(ctx context.Context, tenantID *uuid.UUID) (*models.JobStats, error) {
	return s.jobRepo.GetStats(ctx, tenantID)
}

// validateSchedule validates the schedule based on job type
func (s *JobService) validateSchedule(jobType models.JobType, schedule string) error {
	switch jobType {
	case models.JobTypeCron:
		if _, err := s.cronParser.Parse(schedule); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	case models.JobTypeInterval:
		var interval int
		if err := json.Unmarshal([]byte(schedule), &interval); err != nil {
			return fmt.Errorf("invalid interval (should be seconds as integer): %w", err)
		}
		if interval < 1 {
			return fmt.Errorf("interval must be at least 1 second")
		}
	case models.JobTypeOneTime:
		// One-time jobs don't need schedule validation
	default:
		return fmt.Errorf("unknown job type: %s", jobType)
	}
	return nil
}

// calculateNextRun calculates the next run time for a job
func (s *JobService) calculateNextRun(job *models.Job) (*time.Time, error) {
	return s.scheduler.CalculateNextRun(job)
}
