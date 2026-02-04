# Scheduler Service

A distributed job scheduling microservice for Minisource platform. Supports cron jobs, one-time jobs, and interval-based scheduling with HTTP callback execution.

## Features

- **Multiple Job Types**: Cron expressions, one-time jobs, and interval-based scheduling
- **Distributed Execution**: Redis-based distributed locking for multi-instance deployments
- **Worker Pool**: Configurable worker pool for parallel job execution
- **HTTP Callbacks**: Execute jobs by calling HTTP endpoints with custom headers and payloads
- **Retry Logic**: Configurable retry attempts with delay between retries
- **Job History**: Daily aggregated statistics for job performance monitoring
- **Multi-tenancy**: Tenant-based job isolation
- **Observability**: OpenTelemetry tracing support

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL 16+
- Redis 7+
- Docker (optional)

### Running with Docker

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f scheduler
```

### Running Locally

```bash
# Copy environment file
cp .env.example .env

# Edit configuration
vim .env

# Run migrations
make migrate-up

# Start the service
make run
```

## API Endpoints

### Jobs

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/jobs` | List jobs |
| POST | `/api/v1/jobs` | Create job |
| GET | `/api/v1/jobs/:id` | Get job |
| PUT | `/api/v1/jobs/:id` | Update job |
| DELETE | `/api/v1/jobs/:id` | Delete job |
| POST | `/api/v1/jobs/:id/trigger` | Trigger job manually |
| POST | `/api/v1/jobs/:id/pause` | Pause job |
| POST | `/api/v1/jobs/:id/resume` | Resume job |
| GET | `/api/v1/jobs/stats` | Get job statistics |

### Executions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/executions` | List executions |
| GET | `/api/v1/executions/:id` | Get execution |
| POST | `/api/v1/executions/:id/cancel` | Cancel execution |
| GET | `/api/v1/executions/stats` | Get execution statistics |
| GET | `/api/v1/jobs/:job_id/executions` | List executions by job |

### History

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/history` | Get history by date range |
| GET | `/api/v1/history/stats` | Get aggregated statistics |
| GET | `/api/v1/jobs/:job_id/history` | Get job history |

### Health

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |
| GET | `/live` | Liveness check |

## Job Types

### Cron Jobs

Jobs that run on a cron schedule:

```json
{
  "name": "Daily Report",
  "type": "cron",
  "schedule": "0 0 9 * * *",
  "endpoint": "https://api.example.com/reports/generate",
  "method": "POST"
}
```

### One-Time Jobs

Jobs that run once at a specific time:

```json
{
  "name": "Send Welcome Email",
  "type": "one_time",
  "schedule": "2024-12-31T23:59:59Z",
  "endpoint": "https://api.example.com/emails/welcome",
  "method": "POST",
  "payload": {"user_id": "123"}
}
```

### Interval Jobs

Jobs that run at fixed intervals:

```json
{
  "name": "Health Check",
  "type": "interval",
  "schedule": "300",
  "endpoint": "https://api.example.com/health",
  "method": "GET"
}
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | HTTP server port | `5003` |
| `POSTGRES_HOST` | PostgreSQL host | `localhost` |
| `POSTGRES_PORT` | PostgreSQL port | `5432` |
| `POSTGRES_USER` | PostgreSQL user | `scheduler` |
| `POSTGRES_PASSWORD` | PostgreSQL password | - |
| `POSTGRES_DB` | PostgreSQL database | `scheduler` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `SCHEDULER_WORKER_COUNT` | Number of workers | `10` |
| `SCHEDULER_MAX_RETRIES` | Max retry attempts | `3` |
| `SCHEDULER_RETRY_DELAY_SECONDS` | Delay between retries | `60` |
| `SCHEDULER_LOCK_TTL_SECONDS` | Distributed lock TTL | `300` |
| `SCHEDULER_CLEANUP_DAYS` | Days to keep history | `30` |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Scheduler Service                       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  HTTP API   │  │  Scheduler  │  │  Worker     │         │
│  │  (Fiber)    │  │   Engine    │  │   Pool      │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
│         │                │                │                 │
│  ┌──────┴────────────────┴────────────────┴──────┐         │
│  │                 Service Layer                  │         │
│  └──────┬────────────────┬────────────────┬──────┘         │
│         │                │                │                 │
│  ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐         │
│  │ Job Repo    │  │ Exec Repo   │  │ History Repo│         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
└─────────┼────────────────┼────────────────┼─────────────────┘
          │                │                │
     ┌────┴────────────────┴────────────────┴────┐
     │              PostgreSQL                    │
     └───────────────────────────────────────────┘
          │
     ┌────┴─────┐
     │  Redis   │ (Distributed Locking)
     └──────────┘
```

## Development

```bash
# Install development tools
make install-tools

# Run with hot reload
make dev

# Run tests
make test

# Run linter
make lint

# Generate swagger docs
make swagger
```

## License

MIT License