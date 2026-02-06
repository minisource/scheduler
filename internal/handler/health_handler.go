package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/minisource/go-common/response"
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
// @Success 200 {object} response.Response
// @Failure 503 {object} response.Response
// @Router /health [get]
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	healthData := map[string]interface{}{
		"status":    "healthy",
		"scheduler": h.scheduler.IsRunning(),
	}

	// Check database connection
	sqlDB, err := h.db.DB()
	if err != nil {
		healthData["status"] = "unhealthy"
		healthData["database"] = "disconnected"
		return response.ServiceUnavailable(c, "Database connection error")
	}

	if err := sqlDB.Ping(); err != nil {
		healthData["status"] = "unhealthy"
		healthData["database"] = "disconnected"
		return response.ServiceUnavailable(c, "Database ping failed")
	}

	healthData["database"] = "connected"

	return response.OK(c, healthData)
}

// Ready returns the service readiness status
// @Summary Readiness check
// @Description Check if service is ready to accept traffic
// @Tags health
// @Produce json
// @Success 200 {object} response.Response
// @Failure 503 {object} response.Response
// @Router /ready [get]
func (h *HealthHandler) Ready(c *fiber.Ctx) error {
	if !h.scheduler.IsRunning() {
		return response.ServiceUnavailable(c, "Scheduler is not running")
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		return response.ServiceUnavailable(c, "Database connection error")
	}

	if err := sqlDB.Ping(); err != nil {
		return response.ServiceUnavailable(c, "Database ping failed")
	}

	return response.OK(c, map[string]string{"status": "ready"})
}

// Live returns the liveness status
// @Summary Liveness check
// @Description Check if service is alive
// @Tags health
// @Produce json
// @Success 200 {object} response.Response
// @Router /live [get]
func (h *HealthHandler) Live(c *fiber.Ctx) error {
	return response.OK(c, map[string]string{"status": "alive"})
}
