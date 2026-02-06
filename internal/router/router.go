package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/swagger"
	"github.com/minisource/scheduler/internal/handler"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	Job       *handler.JobHandler
	Execution *handler.ExecutionHandler
	History   *handler.HistoryHandler
	Health    *handler.HealthHandler
}

// SetupRouter configures the Fiber router
func SetupRouter(app *fiber.App, h *Handlers) {
	// Middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Tenant-ID,X-Request-ID",
	}))

	// Swagger route
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Health check routes (no prefix)
	app.Get("/health", h.Health.Health)
	app.Get("/ready", h.Health.Ready)
	app.Get("/live", h.Health.Live)

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Job routes
	jobs := v1.Group("/jobs")
	jobs.Get("/stats", h.Job.GetStats)
	jobs.Get("/", h.Job.List)
	jobs.Post("/", h.Job.Create)
	jobs.Get("/:id", h.Job.Get)
	jobs.Put("/:id", h.Job.Update)
	jobs.Delete("/:id", h.Job.Delete)
	jobs.Post("/:id/trigger", h.Job.Trigger)
	jobs.Post("/:id/pause", h.Job.Pause)
	jobs.Post("/:id/resume", h.Job.Resume)
	jobs.Get("/:job_id/executions", h.Execution.ListByJob)
	jobs.Get("/:job_id/history", h.History.GetByJob)

	// Execution routes
	executions := v1.Group("/executions")
	executions.Get("/stats", h.Execution.GetStats)
	executions.Get("/", h.Execution.List)
	executions.Get("/:id", h.Execution.Get)
	executions.Post("/:id/cancel", h.Execution.Cancel)

	// History routes
	history := v1.Group("/history")
	history.Get("/stats", h.History.GetAggregated)
	history.Get("/", h.History.GetDateRange)
}
