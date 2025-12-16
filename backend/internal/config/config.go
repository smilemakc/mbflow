// Package config provides configuration management for MBFlow.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	Logging     LoggingConfig
	Observer    ObserverConfig
	Auth        AuthConfig
	FileStorage FileStorageConfig
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

// ObserverConfig holds observer-related configuration.
type ObserverConfig struct {
	// Database observer
	EnableDatabase bool

	// HTTP callback observer
	EnableHTTP      bool
	HTTPCallbackURL string
	HTTPMethod      string
	HTTPTimeout     time.Duration
	HTTPMaxRetries  int
	HTTPRetryDelay  time.Duration
	HTTPHeaders     map[string]string

	// Logger observer
	EnableLogger bool

	// WebSocket observer
	EnableWebSocket     bool
	WebSocketBufferSize int

	// General settings
	BufferSize int
}

// AuthConfig holds authentication and authorization configuration.
type AuthConfig struct {
	Mode string

	JWTSecret          string
	JWTExpirationHours int
	RefreshExpiryDays  int

	SessionDuration    time.Duration
	MaxSessionsPerUser int

	MinPasswordLength   int
	RequireSpecialChars bool
	RequireUppercase    bool
	RequireNumbers      bool

	EnableRateLimit  bool
	MaxLoginAttempts int
	LockoutDuration  time.Duration

	AllowRegistration bool

	GatewayURL   string
	ClientID     string
	ClientSecret string
	IssuerURL    string
	JWKSURL      string
	RedirectURL  string

	EnableFallback bool
	FallbackMode   string
}

// FileStorageConfig holds file storage configuration.
type FileStorageConfig struct {
	MaxFileSize int64
	StoragePath string
}

// Load loads the configuration from environment variables.
func Load() (*Config, error) {
	godotenv.Load()
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
		Observer: ObserverConfig{
			EnableDatabase:      getEnvAsBool("OBSERVER_DB_ENABLED", true),
			EnableHTTP:          getEnvAsBool("OBSERVER_HTTP_ENABLED", false),
			HTTPCallbackURL:     getEnv("OBSERVER_HTTP_URL", ""),
			HTTPMethod:          getEnv("OBSERVER_HTTP_METHOD", "POST"),
			HTTPTimeout:         getEnvAsDuration("OBSERVER_HTTP_TIMEOUT", 10*time.Second),
			HTTPMaxRetries:      getEnvAsInt("OBSERVER_HTTP_MAX_RETRIES", 3),
			HTTPRetryDelay:      getEnvAsDuration("OBSERVER_HTTP_RETRY_DELAY", 1*time.Second),
			HTTPHeaders:         parseHTTPHeaders(getEnv("OBSERVER_HTTP_HEADERS", "")),
			EnableLogger:        getEnvAsBool("OBSERVER_LOGGER_ENABLED", true),
			EnableWebSocket:     getEnvAsBool("OBSERVER_WEBSOCKET_ENABLED", true),
			WebSocketBufferSize: getEnvAsInt("OBSERVER_WEBSOCKET_BUFFER_SIZE", 256),
			BufferSize:          getEnvAsInt("OBSERVER_BUFFER_SIZE", 100),
		},
		Auth: AuthConfig{
			Mode:                getEnv("AUTH_MODE", "builtin"),
			JWTSecret:           getEnv("JWT_SECRET", ""),
			JWTExpirationHours:  getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
			RefreshExpiryDays:   getEnvAsInt("JWT_REFRESH_DAYS", 30),
			SessionDuration:     getEnvAsDuration("SESSION_DURATION", 24*time.Hour),
			MaxSessionsPerUser:  getEnvAsInt("MAX_SESSIONS_PER_USER", 5),
			MinPasswordLength:   getEnvAsInt("MIN_PASSWORD_LENGTH", 8),
			RequireSpecialChars: getEnvAsBool("REQUIRE_SPECIAL_CHARS", false),
			RequireUppercase:    getEnvAsBool("REQUIRE_UPPERCASE", false),
			RequireNumbers:      getEnvAsBool("REQUIRE_NUMBERS", false),
			EnableRateLimit:     getEnvAsBool("ENABLE_RATE_LIMIT", true),
			MaxLoginAttempts:    getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:     getEnvAsDuration("LOCKOUT_DURATION", 15*time.Minute),
			AllowRegistration:   getEnvAsBool("ALLOW_REGISTRATION", true),
			GatewayURL:          getEnv("AUTH_GATEWAY_URL", ""),
			ClientID:            getEnv("AUTH_CLIENT_ID", ""),
			ClientSecret:        getEnv("AUTH_CLIENT_SECRET", ""),
			IssuerURL:           getEnv("AUTH_ISSUER_URL", ""),
			JWKSURL:             getEnv("AUTH_JWKS_URL", ""),
			RedirectURL:         getEnv("AUTH_REDIRECT_URL", ""),
			EnableFallback:      getEnvAsBool("AUTH_ENABLE_FALLBACK", false),
			FallbackMode:        getEnv("AUTH_FALLBACK_MODE", "builtin"),
		},
		FileStorage: FileStorageConfig{
			MaxFileSize: getEnvAsInt64("FILE_STORAGE_MAX_FILE_SIZE", 10*1024*1024),
			StoragePath: getEnv("FILE_STORAGE_PATH", "./data/storage"),
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

	if err := c.validateAuth(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateAuth() error {
	if c.Auth.Mode != "builtin" && c.Auth.Mode != "gateway" && c.Auth.Mode != "hybrid" {
		return fmt.Errorf("invalid AUTH_MODE: %s (must be builtin, gateway, or hybrid)", c.Auth.Mode)
	}

	if c.Auth.Mode == "builtin" || c.Auth.Mode == "hybrid" {
		if c.Auth.JWTSecret == "" {
			return fmt.Errorf("JWT_SECRET is required for %s mode", c.Auth.Mode)
		}
		if len(c.Auth.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters")
		}
	}

	if c.Auth.Mode == "gateway" || c.Auth.Mode == "hybrid" {
		if c.Auth.GatewayURL == "" || c.Auth.ClientID == "" {
			return fmt.Errorf("AUTH_GATEWAY_URL and AUTH_CLIENT_ID are required for %s mode", c.Auth.Mode)
		}
	}

	if c.Auth.MinPasswordLength < 8 {
		return fmt.Errorf("MIN_PASSWORD_LENGTH must be at least 8")
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

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}

	return value
}

// parseHTTPHeaders parses HTTP headers from environment variable
// Format: "Key1:Value1,Key2:Value2"
func parseHTTPHeaders(headersStr string) map[string]string {
	headers := make(map[string]string)
	if headersStr == "" {
		return headers
	}

	pairs := strings.Split(headersStr, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return headers
}
