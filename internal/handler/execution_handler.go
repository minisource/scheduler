package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/minisource/go-common/response"
	"github.com/minisource/scheduler/internal/models"
	"github.com/minisource/scheduler/internal/service"
)

// ExecutionHandler handles execution-related HTTP requests
type ExecutionHandler struct {
	executionService *service.ExecutionService
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(executionService *service.ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{
		executionService: executionService,
	}
}

// Get retrieves an execution by ID
// @Summary Get an execution
// @Description Get an execution by ID
// @Tags executions
// @Produce json
// @Param id path string true "Execution ID"
// @Success 200 {object} response.Response{data=models.JobExecution}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/executions/{id} [get]
func (h *ExecutionHandler) Get(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "BAD_REQUEST", "Invalid execution ID")
	}

	execution, err := h.executionService.GetByID(c.Context(), id)
	if err != nil {
		return response.NotFound(c, "Execution not found")
	}

	return response.OK(c, execution)
}

// List lists executions with filtering
// @Summary List executions
// @Description List executions with optional filtering
// @Tags executions
// @Produce json
// @Param job_id query string false "Filter by job ID"
// @Param status query string false "Filter by status"
// @Param start_time query string false "Filter by start time (RFC3339)"
// @Param end_time query string false "Filter by end time (RFC3339)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} response.Response{data=[]models.JobExecution}
// @Failure 500 {object} response.Response
// @Router /api/v1/executions [get]
func (h *ExecutionHandler) List(c *fiber.Ctx) error {
	tenantID := getTenantID(c)

	filter := models.ExecutionFilter{
		TenantID: &tenantID,
		Status:   models.ExecutionStatus(c.Query("status")),
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("page_size", 20),
	}

	// Parse job ID
	if jobIDStr := c.Query("job_id"); jobIDStr != "" {
		jobID, err := uuid.Parse(jobIDStr)
		if err == nil {
			filter.JobID = &jobID
		}
	}

	// Parse time filters
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	result, err := h.executionService.List(c.Context(), filter)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OKWithPagination(c, result.Executions, &response.Pagination{
		Page:    result.Page,
		PerPage: result.PageSize,
		Total:   result.TotalCount,
		HasNext: result.HasMore,
	})
}

// ListByJob lists executions for a specific job
// @Summary List executions by job
// @Description List executions for a specific job
// @Tags executions
// @Produce json
// @Param job_id path string true "Job ID"
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} response.Response{data=[]models.JobExecution}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/jobs/{job_id}/executions [get]
func (h *ExecutionHandler) ListByJob(c *fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return response.BadRequest(c, "BAD_REQUEST", "Invalid job ID")
	}

	limit := c.QueryInt("limit", 10)

	executions, err := h.executionService.GetByJobID(c.Context(), jobID, limit)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, executions)
}

// Cancel cancels an execution
// @Summary Cancel an execution
// @Description Cancel a pending or running execution
// @Tags executions
// @Param id path string true "Execution ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/executions/{id}/cancel [post]
func (h *ExecutionHandler) Cancel(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "BAD_REQUEST", "Invalid execution ID")
	}

	if err := h.executionService.Cancel(c.Context(), id); err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, map[string]bool{"cancelled": true})
}

// GetStats retrieves execution statistics
// @Summary Get execution statistics
// @Description Get statistics about executions
// @Tags executions
// @Produce json
// @Param start_time query string false "Start time (RFC3339)"
// @Param end_time query string false "End time (RFC3339)"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/executions/stats [get]
func (h *ExecutionHandler) GetStats(c *fiber.Ctx) error {
	tenantID := getTenantID(c)

	// Default to last 24 hours
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = t
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = t
		}
	}

	stats, err := h.executionService.GetStats(c.Context(), &tenantID, startTime, endTime)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, stats)
}
