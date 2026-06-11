// Package config provides configuration management for all services
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server ServerConfig
	
	// Database configuration
	Database DatabaseConfig
	
	// Redis configuration
	Redis RedisConfig
	
	// JWT configuration  
	JWT JWTConfig
	
	// OAuth configuration
	OAuth OAuthConfig
	
	// Telemetry configuration
	Telemetry TelemetryConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            int
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	Environment     string // "development", "staging", "production"
	FrontendURL     string // Allowed CORS origin for browser clients
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// JWTConfig holds JWT token configuration
type JWTConfig struct {
	Secret          string
	ExpirationHours int
	Issuer          string
}

// OAuthConfig holds OAuth provider configuration
type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	CallbackURL        string
}

// TelemetryConfig holds OpenTelemetry configuration
type TelemetryConfig struct {
	ServiceName    string
	ServiceVersion string
	Endpoint       string
	Enabled        bool
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists (for development)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnvAsInt("PORT", 8000),
			Host:            getEnv("HOST", "0.0.0.0"),
			ReadTimeout:     getEnvAsDuration("READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvAsDuration("WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
			Environment:     getEnv("ENVIRONMENT", "development"),
			FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:3000"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Database:        getEnv("DB_NAME", "agent_obs"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", generateDefaultSecret()),
			ExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24*7), // 7 days default
			Issuer:          getEnv("JWT_ISSUER", "eyesite"),
		},
		OAuth: OAuthConfig{
			GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
			GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
			CallbackURL:        getEnv("OAUTH_CALLBACK_URL", "http://localhost:3000/auth/callback"),
		},
		Telemetry: TelemetryConfig{
			ServiceName:    getEnv("OTEL_SERVICE_NAME", "agent-obs-api"),
			ServiceVersion: getEnv("OTEL_SERVICE_VERSION", "0.1.0"),
			Endpoint:       getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
			Enabled:        getEnvAsBool("OTEL_ENABLED", false),
		},
	}

	// Validate required configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters")
	}

	return nil
}

// DatabaseConnectionString returns PostgreSQL connection string
func (c *Config) DatabaseConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// RedisAddress returns Redis address string
func (c *Config) RedisAddress() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// Helper functions for reading environment variables

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultVal
}

func generateDefaultSecret() string {
	// In production, this should never be used - always set JWT_SECRET env var
	// This is only for development convenience
	return "INSECURE_DEFAULT_SECRET_CHANGE_IN_PRODUCTION_MIN_32_CHARS"
}
