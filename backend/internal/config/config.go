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
	FileStorage    FileStorageConfig
	ServiceKeys    ServiceKeysConfig
	ServiceAPI     SystemAPIConfig
	GRPCServiceAPI GRPCServiceAPIConfig
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port               int
	Host               string
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	ShutdownTimeout    time.Duration
	CORS               bool
	CORSAllowedOrigins []string
	APIKeys            []string
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

	GRPCAddress       string
	GRPCTimeout       time.Duration
	GRPCApplicationID string
	GRPCClientName    string
	GRPCClientVersion string
	GRPCPlatform      string
	GRPCEnvironment   string

	EnableFallback bool
	FallbackMode   string
}

// FileStorageConfig holds file storage configuration.
type FileStorageConfig struct {
	MaxFileSize int64
	StoragePath string
}

// ServiceKeysConfig holds service key configuration.
type ServiceKeysConfig struct {
	MaxKeysPerUser    int
	DefaultExpiryDays int
}

// SystemAPIConfig holds system API configuration.
type SystemAPIConfig struct {
	MaxKeys            int    `mapstructure:"max_keys" yaml:"max_keys"`
	BcryptCost         int    `mapstructure:"bcrypt_cost" yaml:"bcrypt_cost"`
	DefaultExpiryDays  int    `mapstructure:"default_expiry_days" yaml:"default_expiry_days"`
	AuditRetentionDays int    `mapstructure:"audit_retention_days" yaml:"audit_retention_days"`
	SystemUserID       string `mapstructure:"system_user_id" yaml:"system_user_id"`
}

// GRPCServiceAPIConfig holds gRPC Service API configuration.
type GRPCServiceAPIConfig struct {
	Enabled bool
	Address string
}

// Load loads the configuration from environment variables.
func Load() (*Config, error) {
	godotenv.Load()
	cfg := &Config{
		Server: ServerConfig{
			Port:               getEnvAsInt("MBFLOW_PORT", 8585),
			Host:               getEnv("MBFLOW_HOST", "0.0.0.0"),
			ReadTimeout:        getEnvAsDuration("MBFLOW_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:       getEnvAsDuration("MBFLOW_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout:    getEnvAsDuration("MBFLOW_SHUTDOWN_TIMEOUT", 30*time.Second),
			CORS:               getEnvAsBool("MBFLOW_CORS_ENABLED", true),
			CORSAllowedOrigins: getEnvAsSlice("MBFLOW_CORS_ALLOWED_ORIGINS", []string{}),
			APIKeys:            getEnvAsSlice("MBFLOW_API_KEYS", []string{}),
		},
		Database: DatabaseConfig{
			URL:             getEnv("MBFLOW_DATABASE_URL", "postgres://mbflow:mbflow@localhost:5432/mbflow?sslmode=disable"),
			MaxConnections:  getEnvAsInt("MBFLOW_DB_MAX_CONNECTIONS", 20),
			MinConnections:  getEnvAsInt("MBFLOW_DB_MIN_CONNECTIONS", 5),
			MaxIdleTime:     getEnvAsDuration("MBFLOW_DB_MAX_IDLE_TIME", 30*time.Minute),
			MaxConnLifetime: getEnvAsDuration("MBFLOW_DB_MAX_CONN_LIFETIME", time.Hour),
		},
		Redis: RedisConfig{
			URL:      getEnv("MBFLOW_REDIS_URL", "redis://localhost:6379"),
			Password: getEnv("MBFLOW_REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("MBFLOW_REDIS_DB", 0),
			PoolSize: getEnvAsInt("MBFLOW_REDIS_POOL_SIZE", 10),
		},
		Logging: LoggingConfig{
			Level:  getEnv("MBFLOW_LOG_LEVEL", "info"),
			Format: getEnv("MBFLOW_LOG_FORMAT", "json"),
		},
		Observer: ObserverConfig{
			EnableDatabase:      getEnvAsBool("MBFLOW_OBSERVER_DB_ENABLED", true),
			EnableHTTP:          getEnvAsBool("MBFLOW_OBSERVER_HTTP_ENABLED", false),
			HTTPCallbackURL:     getEnv("MBFLOW_OBSERVER_HTTP_URL", ""),
			HTTPMethod:          getEnv("MBFLOW_OBSERVER_HTTP_METHOD", "POST"),
			HTTPTimeout:         getEnvAsDuration("MBFLOW_OBSERVER_HTTP_TIMEOUT", 10*time.Second),
			HTTPMaxRetries:      getEnvAsInt("MBFLOW_OBSERVER_HTTP_MAX_RETRIES", 3),
			HTTPRetryDelay:      getEnvAsDuration("MBFLOW_OBSERVER_HTTP_RETRY_DELAY", 1*time.Second),
			HTTPHeaders:         parseHTTPHeaders(getEnv("MBFLOW_OBSERVER_HTTP_HEADERS", "")),
			EnableLogger:        getEnvAsBool("MBFLOW_OBSERVER_LOGGER_ENABLED", true),
			EnableWebSocket:     getEnvAsBool("MBFLOW_OBSERVER_WEBSOCKET_ENABLED", true),
			WebSocketBufferSize: getEnvAsInt("MBFLOW_OBSERVER_WEBSOCKET_BUFFER_SIZE", 256),
			BufferSize:          getEnvAsInt("MBFLOW_OBSERVER_BUFFER_SIZE", 100),
		},
		Auth: AuthConfig{
			Mode:                getEnv("MBFLOW_AUTH_MODE", "builtin"),
			JWTSecret:           getEnv("MBFLOW_JWT_SECRET", ""),
			JWTExpirationHours:  getEnvAsInt("MBFLOW_JWT_EXPIRATION_HOURS", 24),
			RefreshExpiryDays:   getEnvAsInt("MBFLOW_JWT_REFRESH_DAYS", 30),
			SessionDuration:     getEnvAsDuration("MBFLOW_SESSION_DURATION", 24*time.Hour),
			MaxSessionsPerUser:  getEnvAsInt("MBFLOW_MAX_SESSIONS_PER_USER", 5),
			MinPasswordLength:   getEnvAsInt("MBFLOW_MIN_PASSWORD_LENGTH", 8),
			RequireSpecialChars: getEnvAsBool("MBFLOW_REQUIRE_SPECIAL_CHARS", false),
			RequireUppercase:    getEnvAsBool("MBFLOW_REQUIRE_UPPERCASE", false),
			RequireNumbers:      getEnvAsBool("MBFLOW_REQUIRE_NUMBERS", false),
			EnableRateLimit:     getEnvAsBool("MBFLOW_ENABLE_RATE_LIMIT", true),
			MaxLoginAttempts:    getEnvAsInt("MBFLOW_MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:     getEnvAsDuration("MBFLOW_LOCKOUT_DURATION", 15*time.Minute),
			AllowRegistration:   getEnvAsBool("MBFLOW_ALLOW_REGISTRATION", true),
			GatewayURL:          getEnv("MBFLOW_AUTH_GATEWAY_URL", ""),
			ClientID:            getEnv("MBFLOW_AUTH_CLIENT_ID", ""),
			ClientSecret:        getEnv("MBFLOW_AUTH_CLIENT_SECRET", ""),
			IssuerURL:           getEnv("MBFLOW_AUTH_ISSUER_URL", ""),
			JWKSURL:             getEnv("MBFLOW_AUTH_JWKS_URL", ""),
			RedirectURL:         getEnv("MBFLOW_AUTH_REDIRECT_URL", ""),
			GRPCAddress:         getEnv("MBFLOW_AUTH_GRPC_ADDRESS", ""),
			GRPCTimeout:         getEnvAsDuration("MBFLOW_AUTH_GRPC_TIMEOUT", 10*time.Second),
			GRPCApplicationID:   getEnv("MBFLOW_AUTH_APPLICATION_ID", ""),
			GRPCClientName:      getEnv("MBFLOW_AUTH_CLIENT_NAME", "mbflow"),
			GRPCClientVersion:   getEnv("MBFLOW_AUTH_CLIENT_VERSION", ""),
			GRPCPlatform:        getEnv("MBFLOW_AUTH_PLATFORM", ""),
			GRPCEnvironment:     getEnv("MBFLOW_AUTH_ENVIRONMENT", ""),
			EnableFallback:      getEnvAsBool("MBFLOW_AUTH_ENABLE_FALLBACK", false),
			FallbackMode:        getEnv("MBFLOW_AUTH_FALLBACK_MODE", "builtin"),
		},
		FileStorage: FileStorageConfig{
			MaxFileSize: getEnvAsInt64("MBFLOW_FILE_STORAGE_MAX_FILE_SIZE", 10*1024*1024),
			StoragePath: getEnv("MBFLOW_FILE_STORAGE_PATH", "./data/storage"),
		},
		ServiceKeys: ServiceKeysConfig{
			MaxKeysPerUser:    getEnvAsInt("MBFLOW_SERVICE_KEYS_MAX_PER_USER", 10),
			DefaultExpiryDays: getEnvAsInt("MBFLOW_SERVICE_KEYS_DEFAULT_EXPIRY_DAYS", 365),
		},
		ServiceAPI: SystemAPIConfig{
			MaxKeys:            getEnvAsInt("MBFLOW_SERVICE_API_MAX_KEYS", 100),
			BcryptCost:         getEnvAsInt("MBFLOW_SERVICE_API_BCRYPT_COST", 10),
			DefaultExpiryDays:  getEnvAsInt("MBFLOW_SERVICE_API_DEFAULT_EXPIRY_DAYS", 365),
			AuditRetentionDays: getEnvAsInt("MBFLOW_SERVICE_API_AUDIT_RETENTION_DAYS", 90),
			SystemUserID:       getEnv("MBFLOW_SERVICE_API_SYSTEM_USER_ID", "00000000-0000-0000-0000-000000000000"),
		},
		GRPCServiceAPI: GRPCServiceAPIConfig{
			Enabled: getEnvAsBool("GRPC_SERVICE_API_ENABLED", false),
			Address: getEnv("GRPC_SERVICE_API_ADDRESS", ":50051"),
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
	validModes := map[string]bool{
		"builtin": true, "gateway": true, "hybrid": true, "grpc": true, "grpc_hybrid": true,
	}
	if !validModes[c.Auth.Mode] {
		return fmt.Errorf("invalid MBFLOW_AUTH_MODE: %s (must be builtin, gateway, hybrid, grpc, or grpc_hybrid)", c.Auth.Mode)
	}

	// Modes that require JWT secret for local token generation
	if c.Auth.Mode == "builtin" || c.Auth.Mode == "hybrid" || c.Auth.Mode == "grpc_hybrid" {
		if c.Auth.JWTSecret == "" {
			return fmt.Errorf("MBFLOW_JWT_SECRET is required for %s mode", c.Auth.Mode)
		}
		if len(c.Auth.JWTSecret) < 32 {
			return fmt.Errorf("MBFLOW_JWT_SECRET must be at least 32 characters")
		}
	}

	if c.Auth.Mode == "gateway" || c.Auth.Mode == "hybrid" {
		if c.Auth.GatewayURL == "" || c.Auth.ClientID == "" {
			return fmt.Errorf("MBFLOW_AUTH_GATEWAY_URL and MBFLOW_AUTH_CLIENT_ID are required for %s mode", c.Auth.Mode)
		}
	}

	if c.Auth.Mode == "grpc" {
		if c.Auth.GRPCAddress == "" {
			return fmt.Errorf("MBFLOW_AUTH_GRPC_ADDRESS is required for grpc mode")
		}
	}

	if c.Auth.MinPasswordLength < 8 {
		return fmt.Errorf("MBFLOW_MIN_PASSWORD_LENGTH must be at least 8")
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
