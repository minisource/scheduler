package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/minisource/go-common/response"
	_ "github.com/minisource/scheduler/internal/models"
	"github.com/minisource/scheduler/internal/service"
)

// HistoryHandler handles history-related HTTP requests
type HistoryHandler struct {
	historyService *service.HistoryService
}

// NewHistoryHandler creates a new history handler
func NewHistoryHandler(historyService *service.HistoryService) *HistoryHandler {
	return &HistoryHandler{
		historyService: historyService,
	}
}

// GetByJob retrieves history for a job
// @Summary Get job history
// @Description Get execution history for a specific job
// @Tags history
// @Produce json
// @Param job_id path string true "Job ID"
// @Param days query int false "Number of days" default(30)
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/jobs/{job_id}/history [get]
func (h *HistoryHandler) GetByJob(c *fiber.Ctx) error {
	jobIDStr := c.Params("job_id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return response.BadRequest(c, "BAD_REQUEST", "Invalid job ID")
	}

	days := c.QueryInt("days", 30)

	history, err := h.historyService.GetByJobID(c.Context(), jobID, days)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, history)
}

// GetAggregated retrieves aggregated history stats
// @Summary Get aggregated history
// @Description Get aggregated execution statistics
// @Tags history
// @Produce json
// @Param job_id query string false "Filter by job ID"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} response.Response{data=models.AggregatedHistoryStats}
// @Failure 500 {object} response.Response
// @Router /api/v1/history/stats [get]
func (h *HistoryHandler) GetAggregated(c *fiber.Ctx) error {
	var jobID *uuid.UUID

	if jobIDStr := c.Query("job_id"); jobIDStr != "" {
		id, err := uuid.Parse(jobIDStr)
		if err == nil {
			jobID = &id
		}
	}

	// Default to last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = t
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = t
		}
	}

	stats, err := h.historyService.GetAggregated(c.Context(), jobID, startDate, endDate)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, stats)
}

// GetDateRange retrieves history for a date range
// @Summary Get history by date range
// @Description Get execution history for a date range
// @Tags history
// @Produce json
// @Param start_date query string true "Start date (YYYY-MM-DD)"
// @Param end_date query string true "End date (YYYY-MM-DD)"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/history [get]
func (h *HistoryHandler) GetDateRange(c *fiber.Ctx) error {
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		return response.BadRequest(c, "BAD_REQUEST", "start_date and end_date are required")
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return response.BadRequest(c, "BAD_REQUEST", "Invalid start_date format (use YYYY-MM-DD)")
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return response.BadRequest(c, "BAD_REQUEST", "Invalid end_date format (use YYYY-MM-DD)")
	}

	history, err := h.historyService.GetByDateRange(c.Context(), startDate, endDate)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, history)
}
