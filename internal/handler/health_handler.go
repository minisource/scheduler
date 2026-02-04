package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/minisource/scheduler/internal/scheduler"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db        *gorm.DB
	scheduler *scheduler.Scheduler
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *gorm.DB, sched *scheduler.Scheduler) *HealthHandler {
	return &HealthHandler{
		db:        db,
		scheduler: sched,
	}
}

// Health returns the service health status
// @Summary Health check
// @Description Check service health
// @Tags health
// @Produce json
// @Success 200 {object} Response
// @Failure 503 {object} Response
// @Router /health [get]
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	response := map[string]interface{}{
		"status":    "healthy",
		"scheduler": h.scheduler.IsRunning(),
	}

	// Check database connection
	sqlDB, err := h.db.DB()
	if err != nil {
		response["status"] = "unhealthy"
		response["database"] = "disconnected"
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{
			Success: false,
			Data:    response,
		})
	}

	if err := sqlDB.Ping(); err != nil {
		response["status"] = "unhealthy"
		response["database"] = "disconnected"
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{
			Success: false,
			Data:    response,
		})
	}

	response["database"] = "connected"

	return Success(c, response)
}

// Ready returns the service readiness status
// @Summary Readiness check
// @Description Check if service is ready to accept traffic
// @Tags health
// @Produce json
// @Success 200 {object} Response
// @Failure 503 {object} Response
// @Router /ready [get]
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	if !h.scheduler.IsRunning() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{
			Success: false,
			Error: &ErrorInfo{
				Code:    "NOT_READY",
				Message: "Scheduler is not running",
			},
		})
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{
			Success: false,
			Error: &ErrorInfo{
				Code:    "NOT_READY",
				Message: "Database connection error",
			},
		})
	}

	if err := sqlDB.Ping(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(Response{
			Success: false,
			Error: &ErrorInfo{
				Code:    "NOT_READY",
				Message: "Database ping failed",
			},
		})
	}

	return Success(c, map[string]string{"status": "ready"})
}

// Live returns the liveness status
// @Summary Liveness check
// @Description Check if service is alive
// @Tags health
// @Produce json
// @Success 200 {object} Response
// @Router /live [get]
func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return Success(c, map[string]string{"status": "alive"})
}
