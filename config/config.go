package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server    ServerConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	Scheduler SchedulerConfig
	Tracing   TracingConfig
}

type ServerConfig struct {
	Port            int
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type PostgresConfig struct {
	Host               string
	Port               string
	User               string
	Password           string
	DBName             string
	SSLMode            string
	MaxOpenConns       int
	MaxIdleConns       int
	MaxLifetimeMinutes int
	LogLevel           string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type SchedulerConfig struct {
	WorkerCount       int
	MaxRetries        int
	RetryDelaySeconds int
	LockTTLSeconds    int
	HeartbeatSeconds  int
	CleanupDays       int
	Timezone          string
}

type TracingConfig struct {
	Enabled     bool
	ServiceName string
	Endpoint    string
	SampleRate  float64
}

func LoadConfig() *Config {
	cfg, _ := Load()
	return cfg
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port:            getEnvInt("SERVER_PORT", 5003),
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:     getDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Postgres: PostgresConfig{
			Host:               getEnv("POSTGRES_HOST", "localhost"),
			Port:               getEnv("POSTGRES_PORT", "5432"),
			User:               getEnv("POSTGRES_USER", "scheduler_user"),
			Password:           getEnv("POSTGRES_PASSWORD", "scheduler_password"),
			DBName:             getEnv("POSTGRES_DB", "scheduler_db"),
			SSLMode:            getEnv("POSTGRES_SSL_MODE", "disable"),
			MaxOpenConns:       getEnvInt("POSTGRES_MAX_OPEN_CONNS", 25),
			MaxIdleConns:       getEnvInt("POSTGRES_MAX_IDLE_CONNS", 10),
			MaxLifetimeMinutes: getEnvInt("POSTGRES_MAX_LIFETIME_MINS", 30),
			LogLevel:           getEnv("POSTGRES_LOG_LEVEL", "warn"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 2),
		},
		Scheduler: SchedulerConfig{
			WorkerCount:       getEnvInt("SCHEDULER_WORKER_COUNT", 10),
			MaxRetries:        getEnvInt("SCHEDULER_MAX_RETRIES", 3),
			RetryDelaySeconds: getEnvInt("SCHEDULER_RETRY_DELAY_SECONDS", 60),
			LockTTLSeconds:    getEnvInt("SCHEDULER_LOCK_TTL_SECONDS", 300),
			HeartbeatSeconds:  getEnvInt("SCHEDULER_HEARTBEAT_SECONDS", 30),
			CleanupDays:       getEnvInt("SCHEDULER_CLEANUP_DAYS", 30),
			Timezone:          getEnv("SCHEDULER_TIMEZONE", "UTC"),
		},
		Tracing: TracingConfig{
			Enabled:     getEnvBool("TRACING_ENABLED", true),
			ServiceName: getEnv("SERVICE_NAME", "scheduler-service"),
			Endpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
			SampleRate:  getEnvFloat("TRACING_SAMPLE_RATE", 1.0),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
