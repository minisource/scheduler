package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/minisource/scheduler/internal/models"
	"github.com/minisource/scheduler/internal/service"
)

// JobHandler handles job-related HTTP requests
type JobHandler struct {
	jobService *service.JobService
}

// NewJobHandler creates a new job handler
func NewJobHandler(jobService *service.JobService) *JobHandler {
	return &JobHandler{
		jobService: jobService,
	}
}

// Create creates a new job
// @Summary Create a job
// @Description Create a new scheduled job
// @Tags jobs
// @Accept json
// @Produce json
// @Param request body models.CreateJobRequest true "Job creation request"
// @Success 201 {object} Response{data=models.Job}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/jobs [post]
func (h *JobHandler) Create(c *fiber.Ctx) error {
	var req models.CreateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	// Get tenant ID from context
	tenantID := getTenantID(c)

	job, err := h.jobService.Create(c.Context(), tenantID, &req)
	if err != nil {
		return InternalError(c, err.Error())
	}

	return Created(c, job)
}

// Get retrieves a job by ID
// @Summary Get a job
// @Description Get a job by ID
// @Tags jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} Response{data=models.Job}
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/jobs/{id} [get]
func (h *JobHandler) Get(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return BadRequest(c, "Invalid job ID")
	}

	tenantID := getTenantID(c)

	job, err := h.jobService.GetByID(c.Context(), tenantID, id)
	if err != nil {
		return NotFound(c, "Job not found")
	}

	return Success(c, job)
}

// List lists jobs with filtering
// @Summary List jobs
// @Description List jobs with optional filtering
// @Tags jobs
// @Produce json
// @Param status query string false "Filter by status"
// @Param type query string false "Filter by type"
// @Param name query string false "Filter by name"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} Response{data=[]models.Job}
// @Failure 500 {object} Response
// @Router /api/v1/jobs [get]
func (h *JobHandler) List(c *fiber.Ctx) error {
	tenantID := getTenantID(c)

	filter := models.JobFilter{
		TenantID: &tenantID,
		Status:   models.JobStatus(c.Query("status")),
		Type:     models.JobType(c.Query("type")),
		Name:     c.Query("name"),
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("page_size", 20),
	}

	result, err := h.jobService.List(c.Context(), filter)
	if err != nil {
		return InternalError(c, err.Error())
	}

	return SuccessWithMeta(c, result.Jobs, &Meta{
		Page:       result.Page,
		PageSize:   result.PageSize,
		TotalCount: result.TotalCount,
		HasMore:    result.HasMore,
	})
}

// Update updates a job
// @Summary Update a job
// @Description Update an existing job
// @Tags jobs
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param request body models.UpdateJobRequest true "Job update request"
// @Success 200 {object} Response{data=models.Job}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/jobs/{id} [put]
func (h *JobHandler) Update(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return BadRequest(c, "Invalid job ID")
	}

	var req models.UpdateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	tenantID := getTenantID(c)

	job, err := h.jobService.Update(c.Context(), tenantID, id, &req)
	if err != nil {
		return InternalError(c, err.Error())
	}

	return Success(c, job)
}

// Delete deletes a job
// @Summary Delete a job
// @Description Soft-delete a job
// @Tags jobs
// @Param id path string true "Job ID"
// @Success 204 "No Content"
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/jobs/{id} [delete]
func (h *JobHandler) Delete(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return BadRequest(c, "Invalid job ID")
	}

	tenantID := getTenantID(c)

	if err := h.jobService.Delete(c.Context(), tenantID, id); err != nil {
		return InternalError(c, err.Error())
	}

	return NoContent(c)
}

// Trigger manually triggers a job
// @Summary Trigger a job
// @Description Manually trigger a job execution
// @Tags jobs
// @Param id path string true "Job ID"
// @Success 200 {object} Response{data=models.JobExecution}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/jobs/{id}/trigger [post]
func (h *JobHandler) Trigger(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return BadRequest(c, "Invalid job ID")
	}

	tenantID := getTenantID(c)

	execution, err := h.jobService.Trigger(c.Context(), tenantID, id)
	if err != nil {
		return InternalError(c, err.Error())
	}

	return Success(c, execution)
}

// Pause pauses a job
// @Summary Pause a job
// @Description Pause a job from executing
// @Tags jobs
// @Param id path string true "Job ID"
// @Success 200 {object} Response{data=models.Job}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/jobs/{id}/pause [post]
func (h *JobHandler) Pause(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return BadRequest(c, "Invalid job ID")
	}

	tenantID := getTenantID(c)

	job, err := h.jobService.UpdateStatus(c.Context(), tenantID, id, models.JobStatusPaused)
	if err != nil {
		return InternalError(c, err.Error())
	}

	return Success(c, job)
}

// Resume resumes a paused job
// @Summary Resume a job
// @Description Resume a paused job
// @Tags jobs
// @Param id path string true "Job ID"
// @Success 200 {object} Response{data=models.Job}
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/jobs/{id}/resume [post]
func (h *JobHandler) Resume(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return BadRequest(c, "Invalid job ID")
	}

	tenantID := getTenantID(c)

	job, err := h.jobService.UpdateStatus(c.Context(), tenantID, id, models.JobStatusActive)
	if err != nil {
		return InternalError(c, err.Error())
	}

	return Success(c, job)
}

// GetStats retrieves job statistics
// @Summary Get job statistics
// @Description Get statistics about jobs
// @Tags jobs
// @Produce json
// @Success 200 {object} Response{data=models.JobStats}
// @Failure 500 {object} Response
// @Router /api/v1/jobs/stats [get]
func (h *JobHandler) GetStats(c *fiber.Ctx) error {
	tenantID := getTenantID(c)

	stats, err := h.jobService.GetStats(c.Context(), &tenantID)
	if err != nil {
		return InternalError(c, err.Error())
	}

	return Success(c, stats)
}

// getTenantID extracts the tenant ID from context
func getTenantID(c *fiber.Ctx) uuid.UUID {
	tenantIDStr := c.Get("X-Tenant-ID")
	if tenantIDStr == "" {
		tenantIDStr = c.Query("tenant_id")
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil
	}
	return tenantID
}
