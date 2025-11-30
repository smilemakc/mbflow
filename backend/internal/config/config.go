// Package config provides configuration management for MBFlow.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Logging  LoggingConfig
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port            int
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	CORS            bool
	APIKeys         []string
}

// DatabaseConfig holds database-related configuration.
type DatabaseConfig struct {
	URL             string
	MaxConnections  int
	MinConnections  int
	MaxIdleTime     time.Duration
	MaxConnLifetime time.Duration
}

// RedisConfig holds Redis-related configuration.
type RedisConfig struct {
	URL      string
	Password string
	DB       int
	PoolSize int
}

// LoggingConfig holds logging-related configuration.
type LoggingConfig struct {
	Level  string
	Format string // "json" or "text"
}

// Load loads the configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnvAsInt("PORT", 8181),
			Host:            getEnv("HOST", "0.0.0.0"),
			ReadTimeout:     getEnvAsDuration("READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvAsDuration("WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
			CORS:            getEnvAsBool("CORS_ENABLED", true),
			APIKeys:         getEnvAsSlice("API_KEYS", []string{}),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://mbflow:mbflow@localhost:5432/mbflow?sslmode=disable"),
			MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 20),
			MinConnections:  getEnvAsInt("DB_MIN_CONNECTIONS", 5),
			MaxIdleTime:     getEnvAsDuration("DB_MAX_IDLE_TIME", 30*time.Minute),
			MaxConnLifetime: getEnvAsDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
			PoolSize: getEnvAsInt("REDIS_POOL_SIZE", 10),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	if c.Database.MaxConnections < 1 {
		return fmt.Errorf("database max connections must be at least 1")
	}

	if c.Database.MinConnections < 1 {
		return fmt.Errorf("database min connections must be at least 1")
	}

	if c.Database.MinConnections > c.Database.MaxConnections {
		return fmt.Errorf("database min connections cannot exceed max connections")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	if c.Logging.Format != "json" && c.Logging.Format != "text" {
		return fmt.Errorf("invalid log format: %s (must be json or text)", c.Logging.Format)
	}

	return nil
}

// Helper functions for environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	// Simple comma-separated parsing
	var result []string
	current := ""
	for _, ch := range valueStr {
		if ch == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
