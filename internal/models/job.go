package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JobType represents the type of job
type JobType string

const (
	JobTypeCron     JobType = "cron"     // Recurring cron job
	JobTypeOneTime  JobType = "one_time" // One-time scheduled job
	JobTypeInterval JobType = "interval" // Fixed interval job
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusActive   JobStatus = "active"
	JobStatusPaused   JobStatus = "paused"
	JobStatusDisabled JobStatus = "disabled"
	JobStatusDeleted  JobStatus = "deleted"
)

// ExecutionStatus represents the status of a job execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusRetrying  ExecutionStatus = "retrying"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
	ExecutionStatusTimeout   ExecutionStatus = "timeout"
)

// Job represents a scheduled job
type Job struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID    uuid.UUID       `json:"tenant_id" gorm:"type:uuid;index:idx_jobs_tenant"`
	Name        string          `json:"name" gorm:"type:varchar(255);not null"`
	Description string          `json:"description,omitempty" gorm:"type:text"`
	Type        JobType         `json:"type" gorm:"type:varchar(20);not null;index:idx_jobs_type"`
	Status      JobStatus       `json:"status" gorm:"type:varchar(20);not null;default:'active';index:idx_jobs_status"`
	Schedule    string          `json:"schedule" gorm:"type:varchar(100)"` // Cron expression or interval
	Timezone    string          `json:"timezone" gorm:"type:varchar(50);default:'UTC'"`
	Endpoint    string          `json:"endpoint" gorm:"type:varchar(500);not null"`        // HTTP endpoint to call
	Method      string          `json:"method" gorm:"type:varchar(10);default:'POST'"`     // HTTP method
	Headers     json.RawMessage `json:"headers,omitempty" gorm:"type:jsonb"`               // HTTP headers
	Payload     json.RawMessage `json:"payload,omitempty" gorm:"type:jsonb"`               // Request body
	Timeout     int             `json:"timeout" gorm:"default:30"`                         // Timeout in seconds
	MaxRetries  int             `json:"max_retries" gorm:"default:3"`                      // Max retry attempts
	RetryDelay  int             `json:"retry_delay" gorm:"default:60"`                     // Delay between retries in seconds
	Priority    int             `json:"priority" gorm:"default:5;index:idx_jobs_priority"` // 1-10, higher is more important
	Tags        json.RawMessage `json:"tags,omitempty" gorm:"type:jsonb"`                  // Job tags for filtering
	Metadata    json.RawMessage `json:"metadata,omitempty" gorm:"type:jsonb"`              // Additional metadata
	NextRunAt   *time.Time      `json:"next_run_at,omitempty" gorm:"index:idx_jobs_next_run"`
	LastRunAt   *time.Time      `json:"last_run_at,omitempty"`
	RunCount    int64           `json:"run_count" gorm:"default:0"`
	FailCount   int64           `json:"fail_count" gorm:"default:0"`
	CreatedBy   *uuid.UUID      `json:"created_by,omitempty" gorm:"type:uuid"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the table name for GORM
func (Job) TableName() string {
	return "jobs"
}

// JobExecution represents a single execution of a job
type JobExecution struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	JobID       uuid.UUID       `json:"job_id" gorm:"type:uuid;not null;index:idx_executions_job"`
	TenantID    uuid.UUID       `json:"tenant_id" gorm:"type:uuid;index:idx_executions_tenant"`
	Status      ExecutionStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending';index:idx_executions_status"`
	ScheduledAt time.Time       `json:"scheduled_at" gorm:"not null;index:idx_executions_scheduled"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Duration    *int64          `json:"duration_ms,omitempty"`                        // Duration in milliseconds
	Attempt     int             `json:"attempt" gorm:"default:1"`                     // Current attempt number
	WorkerID    string          `json:"worker_id,omitempty" gorm:"type:varchar(100)"` // ID of worker executing
	Request     json.RawMessage `json:"request,omitempty" gorm:"type:jsonb"`          // Request sent
	Response    json.RawMessage `json:"response,omitempty" gorm:"type:jsonb"`         // Response received
	StatusCode  *int            `json:"status_code,omitempty"`                        // HTTP status code
	Error       string          `json:"error,omitempty" gorm:"type:text"`             // Error message
	TraceID     string          `json:"trace_id,omitempty" gorm:"type:varchar(64)"`   // Distributed trace ID
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the table name for GORM
func (JobExecution) TableName() string {
	return "job_executions"
}

// JobSchedule represents a calculated schedule entry
type JobSchedule struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	JobID       uuid.UUID  `json:"job_id" gorm:"type:uuid;not null;index:idx_schedule_job"`
	ScheduledAt time.Time  `json:"scheduled_at" gorm:"not null;index:idx_schedule_time"`
	Locked      bool       `json:"locked" gorm:"default:false"`
	LockedBy    string     `json:"locked_by,omitempty" gorm:"type:varchar(100)"`
	LockedAt    *time.Time `json:"locked_at,omitempty"`
	ExecutionID *uuid.UUID `json:"execution_id,omitempty" gorm:"type:uuid"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns the table name for GORM
func (JobSchedule) TableName() string {
	return "job_schedules"
}

// JobHistory represents historical job statistics
type JobHistory struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	JobID         uuid.UUID `json:"job_id" gorm:"type:uuid;not null;index:idx_history_job"`
	TenantID      uuid.UUID `json:"tenant_id" gorm:"type:uuid;index:idx_history_tenant"`
	Date          time.Time `json:"date" gorm:"type:date;not null;index:idx_history_date"`
	TotalRuns     int64     `json:"total_runs" gorm:"default:0"`
	SuccessCount  int64     `json:"success_count" gorm:"default:0"`
	FailureCount  int64     `json:"failure_count" gorm:"default:0"`
	TotalDuration int64     `json:"total_duration_ms" gorm:"default:0"`
	AvgDuration   int64     `json:"avg_duration_ms" gorm:"default:0"`
	MinDuration   int64     `json:"min_duration_ms"`
	MaxDuration   int64     `json:"max_duration_ms"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName returns the table name for GORM
func (JobHistory) TableName() string {
	return "job_history"
}

// CreateJobRequest represents a request to create a new job
type CreateJobRequest struct {
	Name        string          `json:"name" validate:"required,min=1,max=255"`
	Description string          `json:"description,omitempty"`
	Type        JobType         `json:"type" validate:"required,oneof=cron one_time interval"`
	Schedule    string          `json:"schedule" validate:"required"`
	Timezone    string          `json:"timezone,omitempty"`
	Endpoint    string          `json:"endpoint" validate:"required,url"`
	Method      string          `json:"method,omitempty"`
	Headers     json.RawMessage `json:"headers,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Timeout     int             `json:"timeout,omitempty"`
	MaxRetries  int             `json:"max_retries,omitempty"`
	RetryDelay  int             `json:"retry_delay,omitempty"`
	Priority    int             `json:"priority,omitempty"`
	Tags        json.RawMessage `json:"tags,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// UpdateJobRequest represents a request to update a job
type UpdateJobRequest struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Schedule    *string          `json:"schedule,omitempty"`
	Timezone    *string          `json:"timezone,omitempty"`
	Endpoint    *string          `json:"endpoint,omitempty"`
	Method      *string          `json:"method,omitempty"`
	Headers     *json.RawMessage `json:"headers,omitempty"`
	Payload     *json.RawMessage `json:"payload,omitempty"`
	Timeout     *int             `json:"timeout,omitempty"`
	MaxRetries  *int             `json:"max_retries,omitempty"`
	RetryDelay  *int             `json:"retry_delay,omitempty"`
	Priority    *int             `json:"priority,omitempty"`
	Tags        *json.RawMessage `json:"tags,omitempty"`
	Metadata    *json.RawMessage `json:"metadata,omitempty"`
}

// JobFilter represents query filters for jobs
type JobFilter struct {
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
	Status   JobStatus  `json:"status,omitempty"`
	Type     JobType    `json:"type,omitempty"`
	Name     string     `json:"name,omitempty"`
	Tags     []string   `json:"tags,omitempty"`
	Page     int        `json:"page,omitempty"`
	PageSize int        `json:"page_size,omitempty"`
}

// ExecutionFilter represents query filters for executions
type ExecutionFilter struct {
	JobID     *uuid.UUID      `json:"job_id,omitempty"`
	TenantID  *uuid.UUID      `json:"tenant_id,omitempty"`
	Status    ExecutionStatus `json:"status,omitempty"`
	StartTime *time.Time      `json:"start_time,omitempty"`
	EndTime   *time.Time      `json:"end_time,omitempty"`
	Page      int             `json:"page,omitempty"`
	PageSize  int             `json:"page_size,omitempty"`
}

// JobStats represents job statistics
type JobStats struct {
	TotalJobs     int64               `json:"total_jobs"`
	ActiveJobs    int64               `json:"active_jobs"`
	PausedJobs    int64               `json:"paused_jobs"`
	TotalRuns     int64               `json:"total_runs"`
	SuccessRate   float64             `json:"success_rate"`
	AvgDuration   float64             `json:"avg_duration_ms"`
	JobsByType    map[JobType]int64   `json:"jobs_by_type"`
	JobsByStatus  map[JobStatus]int64 `json:"jobs_by_status"`
	RunsToday     int64               `json:"runs_today"`
	FailuresToday int64               `json:"failures_today"`
}

// JobListResult represents paginated job results
type JobListResult struct {
	Jobs       []Job `json:"jobs"`
	TotalCount int64 `json:"total_count"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	HasMore    bool  `json:"has_more"`
}

// ExecutionListResult represents paginated execution results
type ExecutionListResult struct {
	Executions []JobExecution `json:"executions"`
	TotalCount int64          `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	HasMore    bool           `json:"has_more"`
}

// AggregatedHistoryStats contains aggregated statistics
type AggregatedHistoryStats struct {
	TotalSuccess  int64   `json:"total_success"`
	TotalFailure  int64   `json:"total_failure"`
	TotalDuration int64   `json:"total_duration"`
	AvgDuration   float64 `json:"avg_duration"`
	MinDuration   int64   `json:"min_duration"`
	MaxDuration   int64   `json:"max_duration"`
	SuccessRate   float64 `json:"success_rate"`
}
