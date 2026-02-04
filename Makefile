.PHONY: build run test clean docker-build docker-up docker-down migrate-up migrate-down lint swagger

# Application
APP_NAME=scheduler
MAIN_PATH=./cmd/main.go
BUILD_DIR=./bin

# Docker
DOCKER_IMAGE=minisource/scheduler
DOCKER_TAG=latest

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

# Run the application
run:
	@go run $(MAIN_PATH)

# Run with hot reload (requires air)
dev:
	@air -c .air.toml

# Run tests
test:
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Format code
fmt:
	@go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@golangci-lint run

# Download dependencies
deps:
	@go mod download
	@go mod tidy

# Generate swagger documentation (requires swag)
swagger:
	@swag init -g $(MAIN_PATH) -o ./docs

# Docker commands
docker-build:
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-up:
	@docker-compose up -d

docker-down:
	@docker-compose down

docker-dev:
	@docker-compose -f docker-compose.dev.yml up -d

docker-dev-down:
	@docker-compose -f docker-compose.dev.yml down

docker-logs:
	@docker-compose logs -f scheduler

# Database migrations (requires golang-migrate)
migrate-up:
	@migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	@migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@migrate create -ext sql -dir ./migrations -seq $(name)

# Install development tools
install-tools:
	@go install github.com/air-verse/air@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Help
help:
	@echo "Available commands:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  dev            - Run with hot reload"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  clean          - Clean build artifacts"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  deps           - Download dependencies"
	@echo "  swagger        - Generate swagger docs"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-up      - Start Docker containers"
	@echo "  docker-down    - Stop Docker containers"
	@echo "  docker-dev     - Start dev Docker containers"
	@echo "  docker-logs    - View Docker logs"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback last migration"
	@echo "  install-tools  - Install development tools"
