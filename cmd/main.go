package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/minisource/scheduler/config"
	"github.com/minisource/scheduler/internal/database"
	"github.com/minisource/scheduler/internal/handler"
	"github.com/minisource/scheduler/internal/repository"
	"github.com/minisource/scheduler/internal/router"
	"github.com/minisource/scheduler/internal/scheduler"
	"github.com/minisource/scheduler/internal/service"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	db, err := database.NewPostgresConnection(&cfg.Postgres)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	// Auto-migrate models
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to auto-migrate: %v", err)
	}

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize repositories
	jobRepo := repository.NewJobRepository(db)
	executionRepo := repository.NewExecutionRepository(db)
	historyRepo := repository.NewHistoryRepository(db)

	// Initialize distributed locker
	workerID := fmt.Sprintf("worker-%s", uuid.New().String()[:8])
	locker := scheduler.NewDistributedLocker(redisClient, workerID)

	// Initialize scheduler
	sched := scheduler.NewScheduler(cfg, jobRepo, executionRepo, historyRepo, locker)

	// Initialize services
	jobService := service.NewJobService(jobRepo, sched)
	executionService := service.NewExecutionService(executionRepo)
	historyService := service.NewHistoryService(historyRepo)

	// Initialize handlers
	handlers := &router.Handlers{
		Job:       handler.NewJobHandler(jobService),
		Execution: handler.NewExecutionHandler(executionService),
		History:   handler.NewHistoryHandler(historyService),
		Health:    handler.NewHealthHandler(db, sched),
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Minisource Scheduler",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Setup routes
	router.SetupRouter(app, handlers)

	// Start scheduler
	if err := sched.Start(ctx); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Server.Port)
		log.Printf("Starting scheduler service on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down scheduler service...")

	// Stop scheduler
	sched.Stop()

	// Shutdown server with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Scheduler service stopped")
}
