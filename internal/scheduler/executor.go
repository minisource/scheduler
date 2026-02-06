package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/minisource/scheduler/config"
	"github.com/minisource/scheduler/internal/models"
)

// ExecutionResult represents the result of a job execution
type ExecutionResult struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	Duration   int64 // milliseconds
	Error      string
}

// Executor executes HTTP-based jobs
type Executor struct {
	config *config.Config
	client *http.Client
}

// NewExecutor creates a new executor
func NewExecutor(cfg *config.Config, client *http.Client) *Executor {
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &Executor{
		config: cfg,
		client: client,
	}
}

// Execute executes a job and returns the result
func (e *Executor) Execute(ctx context.Context, job *models.Job) (*ExecutionResult, error) {
	startTime := time.Now()
	result := &ExecutionResult{}

	// Build request
	req, err := e.buildRequest(ctx, job)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// Execute request
	resp, err := e.client.Do(req)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // Limit to 1MB
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	result.StatusCode = resp.StatusCode
	result.Body = body
	result.Headers = resp.Header
	result.Duration = time.Since(startTime).Milliseconds()

	// Check for error status codes
	if resp.StatusCode >= 400 {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		return result, fmt.Errorf("%s", result.Error)
	}

	return result, nil
}

// buildRequest builds an HTTP request from a job
func (e *Executor) buildRequest(ctx context.Context, job *models.Job) (*http.Request, error) {
	var body io.Reader

	// Parse payload
	if len(job.Payload) > 0 {
		body = bytes.NewReader(job.Payload)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, job.Method, job.Endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("User-Agent", "Minisource-Scheduler/1.0")
	req.Header.Set("X-Scheduler-Job-ID", job.ID.String())
	req.Header.Set("X-Scheduler-Tenant-ID", job.TenantID.String())

	// Set content type if payload exists
	if len(job.Payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	// Parse and apply custom headers
	if len(job.Headers) > 0 {
		var headers map[string]string
		if err := json.Unmarshal(job.Headers, &headers); err == nil {
			for key, value := range headers {
				req.Header.Set(key, value)
			}
		}
	}

	return req, nil
}

// ExecuteWithRetry executes a job with retry logic
func (e *Executor) ExecuteWithRetry(ctx context.Context, job *models.Job, maxRetries int, retryDelay time.Duration) (*ExecutionResult, error) {
	var lastErr error
	var result *ExecutionResult

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		result, lastErr = e.Execute(ctx, job)
		if lastErr == nil {
			return result, nil
		}

		// Check if error is retryable
		if !e.isRetryable(result) {
			return result, lastErr
		}
	}

	return result, lastErr
}

// isRetryable determines if an error is retryable
func (e *Executor) isRetryable(result *ExecutionResult) bool {
	if result == nil {
		return true // Network errors are retryable
	}

	// Server errors are retryable
	if result.StatusCode >= 500 {
		return true
	}

	// Rate limiting is retryable
	if result.StatusCode == 429 {
		return true
	}

	// Client errors are not retryable
	return false
}
