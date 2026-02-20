package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Config.Load() Tests ====================

func TestConfig_Load_DefaultValues(t *testing.T) {
	// Clear all environment variables
	clearEnv()
	os.Setenv("MBFLOW_JWT_SECRET", "test-secret-key-that-is-at-least-32-chars-long")
	defer os.Unsetenv("MBFLOW_JWT_SECRET")

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify default values
	assert.Equal(t, 8585, cfg.Server.Port)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 15*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.ShutdownTimeout)
	assert.True(t, cfg.Server.CORS)
	assert.Empty(t, cfg.Server.APIKeys)

	assert.Equal(t, "postgres://mbflow:mbflow@localhost:5432/mbflow?sslmode=disable", cfg.Database.URL)
	assert.Equal(t, 20, cfg.Database.MaxConnections)
	assert.Equal(t, 5, cfg.Database.MinConnections)
	assert.Equal(t, 30*time.Minute, cfg.Database.MaxIdleTime)
	assert.Equal(t, time.Hour, cfg.Database.MaxConnLifetime)

	assert.Equal(t, "redis://localhost:6379", cfg.Redis.URL)
	assert.Equal(t, "", cfg.Redis.Password)
	assert.Equal(t, 0, cfg.Redis.DB)
	assert.Equal(t, 10, cfg.Redis.PoolSize)

	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)

	assert.True(t, cfg.Observer.EnableDatabase)
	assert.False(t, cfg.Observer.EnableHTTP)
	assert.True(t, cfg.Observer.EnableLogger)
	assert.True(t, cfg.Observer.EnableWebSocket)
	assert.Equal(t, 256, cfg.Observer.WebSocketBufferSize)
	assert.Equal(t, 100, cfg.Observer.BufferSize)
}

func TestConfig_Load_CustomValues(t *testing.T) {
	clearEnv()
	os.Setenv("MBFLOW_JWT_SECRET", "test-secret-key-that-is-at-least-32-chars-long")

	// Set custom environment variables
	os.Setenv("MBFLOW_PORT", "9090")
	os.Setenv("MBFLOW_HOST", "127.0.0.1")
	os.Setenv("MBFLOW_READ_TIMEOUT", "30s")
	os.Setenv("MBFLOW_WRITE_TIMEOUT", "30s")
	os.Setenv("MBFLOW_SHUTDOWN_TIMEOUT", "60s")
	os.Setenv("MBFLOW_CORS_ENABLED", "false")
	os.Setenv("MBFLOW_API_KEYS", "key1,key2,key3")

	os.Setenv("MBFLOW_DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	os.Setenv("MBFLOW_DB_MAX_CONNECTIONS", "50")
	os.Setenv("MBFLOW_DB_MIN_CONNECTIONS", "10")
	os.Setenv("MBFLOW_DB_MAX_IDLE_TIME", "1h")
	os.Setenv("MBFLOW_DB_MAX_CONN_LIFETIME", "2h")

	os.Setenv("MBFLOW_REDIS_URL", "redis://localhost:6380")
	os.Setenv("MBFLOW_REDIS_PASSWORD", "secret")
	os.Setenv("MBFLOW_REDIS_DB", "1")
	os.Setenv("MBFLOW_REDIS_POOL_SIZE", "20")

	os.Setenv("MBFLOW_LOG_LEVEL", "debug")
	os.Setenv("MBFLOW_LOG_FORMAT", "text")

	os.Setenv("MBFLOW_OBSERVER_DB_ENABLED", "false")
	os.Setenv("MBFLOW_OBSERVER_HTTP_ENABLED", "true")
	os.Setenv("MBFLOW_OBSERVER_HTTP_URL", "http://example.com/webhook")
	os.Setenv("MBFLOW_OBSERVER_HTTP_METHOD", "PUT")
	os.Setenv("MBFLOW_OBSERVER_HTTP_TIMEOUT", "20s")
	os.Setenv("MBFLOW_OBSERVER_HTTP_MAX_RETRIES", "5")
	os.Setenv("MBFLOW_OBSERVER_HTTP_RETRY_DELAY", "2s")
	os.Setenv("MBFLOW_OBSERVER_HTTP_HEADERS", "Authorization:Bearer token,Content-Type:application/json")
	os.Setenv("MBFLOW_OBSERVER_LOGGER_ENABLED", "false")
	os.Setenv("MBFLOW_OBSERVER_WEBSOCKET_ENABLED", "false")
	os.Setenv("MBFLOW_OBSERVER_WEBSOCKET_BUFFER_SIZE", "512")
	os.Setenv("MBFLOW_OBSERVER_BUFFER_SIZE", "200")

	defer clearEnv()

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify custom values
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.False(t, cfg.Server.CORS)
	assert.Equal(t, []string{"key1", "key2", "key3"}, cfg.Server.APIKeys)

	assert.Equal(t, "postgres://user:pass@localhost:5432/testdb", cfg.Database.URL)
	assert.Equal(t, 50, cfg.Database.MaxConnections)
	assert.Equal(t, 10, cfg.Database.MinConnections)

	assert.Equal(t, "redis://localhost:6380", cfg.Redis.URL)
	assert.Equal(t, "secret", cfg.Redis.Password)
	assert.Equal(t, 1, cfg.Redis.DB)
	assert.Equal(t, 20, cfg.Redis.PoolSize)

	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "text", cfg.Logging.Format)

	assert.False(t, cfg.Observer.EnableDatabase)
	assert.True(t, cfg.Observer.EnableHTTP)
	assert.Equal(t, "http://example.com/webhook", cfg.Observer.HTTPCallbackURL)
	assert.Equal(t, "PUT", cfg.Observer.HTTPMethod)
	assert.Equal(t, 20*time.Second, cfg.Observer.HTTPTimeout)
	assert.Equal(t, 5, cfg.Observer.HTTPMaxRetries)
	assert.Equal(t, 2*time.Second, cfg.Observer.HTTPRetryDelay)
	assert.Equal(t, "Bearer token", cfg.Observer.HTTPHeaders["Authorization"])
	assert.Equal(t, "application/json", cfg.Observer.HTTPHeaders["Content-Type"])
	assert.False(t, cfg.Observer.EnableLogger)
	assert.False(t, cfg.Observer.EnableWebSocket)
	assert.Equal(t, 512, cfg.Observer.WebSocketBufferSize)
	assert.Equal(t, 200, cfg.Observer.BufferSize)
}

func TestConfig_Load_InvalidValuesUsesDefaults(t *testing.T) {
	clearEnv()
	os.Setenv("MBFLOW_JWT_SECRET", "test-secret-key-that-is-at-least-32-chars-long")

	// Set invalid environment variables (should use defaults)
	os.Setenv("MBFLOW_PORT", "invalid")
	os.Setenv("MBFLOW_DB_MAX_CONNECTIONS", "not_a_number")
	os.Setenv("MBFLOW_READ_TIMEOUT", "invalid_duration")
	os.Setenv("MBFLOW_CORS_ENABLED", "not_a_bool")

	defer clearEnv()

	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Should use default values when parsing fails
	assert.Equal(t, 8585, cfg.Server.Port)
	assert.Equal(t, 20, cfg.Database.MaxConnections)
	assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
	assert.True(t, cfg.Server.CORS)
}

// ==================== Config.Validate() Tests ====================

func TestConfig_Validate_Success(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			URL:            "postgres://localhost:5432/test",
			MaxConnections: 10,
			MinConnections: 5,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Auth: validAuthConfig(),
	}

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"Port too low", 0},
		{"Port negative", -1},
		{"Port too high", 65536},
		{"Port way too high", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: tt.port,
				},
				Database: DatabaseConfig{
					URL:            "postgres://localhost:5432/test",
					MaxConnections: 10,
					MinConnections: 5,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			}

			err := cfg.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid port")
		})
	}
}

func TestConfig_Validate_ValidPorts(t *testing.T) {
	tests := []int{1, 80, 443, 8080, 8585, 65535}

	for _, port := range tests {
		t.Run("Port "+string(rune(port)), func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: port,
				},
				Database: DatabaseConfig{
					URL:            "postgres://localhost:5432/test",
					MaxConnections: 10,
					MinConnections: 5,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
				Auth: validAuthConfig(),
			}

			err := cfg.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestConfig_Validate_EmptyDatabaseURL(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			URL:            "",
			MaxConnections: 10,
			MinConnections: 5,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database URL is required")
}

func TestConfig_Validate_InvalidMaxConnections(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			URL:            "postgres://localhost:5432/test",
			MaxConnections: 0,
			MinConnections: 5,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database max connections must be at least 1")
}

func TestConfig_Validate_InvalidMinConnections(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			URL:            "postgres://localhost:5432/test",
			MaxConnections: 10,
			MinConnections: 0,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database min connections must be at least 1")
}

func TestConfig_Validate_MinExceedsMax(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			URL:            "postgres://localhost:5432/test",
			MaxConnections: 5,
			MinConnections: 10,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database min connections cannot exceed max connections")
}

func TestConfig_Validate_InvalidLogLevel(t *testing.T) {
	tests := []string{"trace", "verbose", "critical", "invalid", ""}

	for _, level := range tests {
		t.Run("Level "+level, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					URL:            "postgres://localhost:5432/test",
					MaxConnections: 10,
					MinConnections: 5,
				},
				Logging: LoggingConfig{
					Level:  level,
					Format: "json",
				},
			}

			err := cfg.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid log level")
		})
	}
}

func TestConfig_Validate_ValidLogLevels(t *testing.T) {
	tests := []string{"debug", "info", "warn", "error"}

	for _, level := range tests {
		t.Run("Level "+level, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					URL:            "postgres://localhost:5432/test",
					MaxConnections: 10,
					MinConnections: 5,
				},
				Logging: LoggingConfig{
					Level:  level,
					Format: "json",
				},
				Auth: validAuthConfig(),
			}

			err := cfg.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestConfig_Validate_InvalidLogFormat(t *testing.T) {
	tests := []string{"xml", "yaml", "csv", "invalid", ""}

	for _, format := range tests {
		t.Run("Format "+format, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					URL:            "postgres://localhost:5432/test",
					MaxConnections: 10,
					MinConnections: 5,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: format,
				},
			}

			err := cfg.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid log format")
		})
	}
}

func TestConfig_Validate_ValidLogFormats(t *testing.T) {
	tests := []string{"json", "text"}

	for _, format := range tests {
		t.Run("Format "+format, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Port: 8080,
				},
				Database: DatabaseConfig{
					URL:            "postgres://localhost:5432/test",
					MaxConnections: 10,
					MinConnections: 5,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: format,
				},
				Auth: validAuthConfig(),
			}

			err := cfg.Validate()
			assert.NoError(t, err)
		})
	}
}

// ==================== Helper Functions Tests ====================

func TestGetEnv_WithValue(t *testing.T) {
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	result := getEnv("TEST_KEY", "default")
	assert.Equal(t, "test_value", result)
}

func TestGetEnv_WithoutValue(t *testing.T) {
	os.Unsetenv("TEST_KEY")

	result := getEnv("TEST_KEY", "default")
	assert.Equal(t, "default", result)
}

func TestGetEnvAsInt_ValidInteger(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	result := getEnvAsInt("TEST_INT", 10)
	assert.Equal(t, 42, result)
}

func TestGetEnvAsInt_InvalidInteger(t *testing.T) {
	os.Setenv("TEST_INT", "not_a_number")
	defer os.Unsetenv("TEST_INT")

	result := getEnvAsInt("TEST_INT", 10)
	assert.Equal(t, 10, result)
}

func TestGetEnvAsInt_EmptyString(t *testing.T) {
	os.Unsetenv("TEST_INT")

	result := getEnvAsInt("TEST_INT", 10)
	assert.Equal(t, 10, result)
}

func TestGetEnvAsInt_NegativeNumber(t *testing.T) {
	os.Setenv("TEST_INT", "-42")
	defer os.Unsetenv("TEST_INT")

	result := getEnvAsInt("TEST_INT", 10)
	assert.Equal(t, -42, result)
}

func TestGetEnvAsBool_True(t *testing.T) {
	tests := []string{"true", "True", "TRUE", "1", "t", "T"}

	for _, value := range tests {
		t.Run("Value "+value, func(t *testing.T) {
			os.Setenv("TEST_BOOL", value)
			defer os.Unsetenv("TEST_BOOL")

			result := getEnvAsBool("TEST_BOOL", false)
			assert.True(t, result)
		})
	}
}

func TestGetEnvAsBool_False(t *testing.T) {
	tests := []string{"false", "False", "FALSE", "0", "f", "F"}

	for _, value := range tests {
		t.Run("Value "+value, func(t *testing.T) {
			os.Setenv("TEST_BOOL", value)
			defer os.Unsetenv("TEST_BOOL")

			result := getEnvAsBool("TEST_BOOL", true)
			assert.False(t, result)
		})
	}
}

func TestGetEnvAsBool_Invalid(t *testing.T) {
	os.Setenv("TEST_BOOL", "invalid")
	defer os.Unsetenv("TEST_BOOL")

	result := getEnvAsBool("TEST_BOOL", true)
	assert.True(t, result)
}

func TestGetEnvAsBool_Empty(t *testing.T) {
	os.Unsetenv("TEST_BOOL")

	result := getEnvAsBool("TEST_BOOL", true)
	assert.True(t, result)
}

func TestGetEnvAsDuration_Valid(t *testing.T) {
	tests := []struct {
		value    string
		expected time.Duration
	}{
		{"1s", 1 * time.Second},
		{"1m", 1 * time.Minute},
		{"1h", 1 * time.Hour},
		{"30s", 30 * time.Second},
		{"1h30m", 90 * time.Minute},
		{"100ms", 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run("Duration "+tt.value, func(t *testing.T) {
			os.Setenv("TEST_DURATION", tt.value)
			defer os.Unsetenv("TEST_DURATION")

			result := getEnvAsDuration("TEST_DURATION", 10*time.Second)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvAsDuration_Invalid(t *testing.T) {
	os.Setenv("TEST_DURATION", "invalid")
	defer os.Unsetenv("TEST_DURATION")

	result := getEnvAsDuration("TEST_DURATION", 10*time.Second)
	assert.Equal(t, 10*time.Second, result)
}

func TestGetEnvAsDuration_Empty(t *testing.T) {
	os.Unsetenv("TEST_DURATION")

	result := getEnvAsDuration("TEST_DURATION", 10*time.Second)
	assert.Equal(t, 10*time.Second, result)
}

func TestGetEnvAsSlice_CommaSeparated(t *testing.T) {
	os.Setenv("TEST_SLICE", "value1,value2,value3")
	defer os.Unsetenv("TEST_SLICE")

	result := getEnvAsSlice("TEST_SLICE", []string{})
	assert.Equal(t, []string{"value1", "value2", "value3"}, result)
}

func TestGetEnvAsSlice_SingleValue(t *testing.T) {
	os.Setenv("TEST_SLICE", "single")
	defer os.Unsetenv("TEST_SLICE")

	result := getEnvAsSlice("TEST_SLICE", []string{})
	assert.Equal(t, []string{"single"}, result)
}

func TestGetEnvAsSlice_WithSpaces(t *testing.T) {
	os.Setenv("TEST_SLICE", "value1, value2, value3")
	defer os.Unsetenv("TEST_SLICE")

	result := getEnvAsSlice("TEST_SLICE", []string{})
	assert.Equal(t, []string{"value1", " value2", " value3"}, result)
}

func TestGetEnvAsSlice_Empty(t *testing.T) {
	os.Unsetenv("TEST_SLICE")

	result := getEnvAsSlice("TEST_SLICE", []string{"default1", "default2"})
	assert.Equal(t, []string{"default1", "default2"}, result)
}

func TestGetEnvAsSlice_EmptyString(t *testing.T) {
	os.Setenv("TEST_SLICE", "")
	defer os.Unsetenv("TEST_SLICE")

	result := getEnvAsSlice("TEST_SLICE", []string{"default1", "default2"})
	assert.Equal(t, []string{"default1", "default2"}, result)
}

func TestParseHTTPHeaders_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:  "Single header",
			input: "Authorization:Bearer token",
			expected: map[string]string{
				"Authorization": "Bearer token",
			},
		},
		{
			name:  "Multiple headers",
			input: "Authorization:Bearer token,Content-Type:application/json",
			expected: map[string]string{
				"Authorization": "Bearer token",
				"Content-Type":  "application/json",
			},
		},
		{
			name:  "Headers with spaces",
			input: "Authorization: Bearer token, Content-Type: application/json",
			expected: map[string]string{
				"Authorization": "Bearer token",
				"Content-Type":  "application/json",
			},
		},
		{
			name:  "Headers with extra spaces",
			input: "  Authorization : Bearer token  ,  Content-Type : application/json  ",
			expected: map[string]string{
				"Authorization": "Bearer token",
				"Content-Type":  "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHTTPHeaders(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseHTTPHeaders_Empty(t *testing.T) {
	result := parseHTTPHeaders("")
	assert.Empty(t, result)
	assert.NotNil(t, result)
}

func TestParseHTTPHeaders_InvalidFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"No colon", "Authorization Bearer token"},
		{"Only key", "Authorization"},
		{"Only comma", ",,,"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHTTPHeaders(tt.input)
			assert.NotNil(t, result)
			// Should return empty map or skip invalid entries
		})
	}
}

// ==================== Helper Functions ====================

// validAuthConfig returns an AuthConfig that passes validation.
func validAuthConfig() AuthConfig {
	return AuthConfig{
		Mode:              "builtin",
		JWTSecret:         "test-secret-key-that-is-at-least-32-chars-long",
		MinPasswordLength: 8,
	}
}

func clearEnv() {
	// Clear all MBFlow-related environment variables
	envVars := []string{
		"MBFLOW_PORT", "MBFLOW_HOST", "MBFLOW_READ_TIMEOUT", "MBFLOW_WRITE_TIMEOUT", "MBFLOW_SHUTDOWN_TIMEOUT",
		"MBFLOW_CORS_ENABLED", "MBFLOW_CORS_ALLOWED_ORIGINS", "MBFLOW_API_KEYS",
		"MBFLOW_MAX_BODY_SIZE", "MBFLOW_MAX_MULTIPART_MEMORY",
		"MBFLOW_DATABASE_URL", "MBFLOW_DB_MAX_CONNECTIONS", "MBFLOW_DB_MIN_CONNECTIONS",
		"MBFLOW_DB_MAX_IDLE_TIME", "MBFLOW_DB_MAX_CONN_LIFETIME",
		"MBFLOW_REDIS_URL", "MBFLOW_REDIS_PASSWORD", "MBFLOW_REDIS_DB", "MBFLOW_REDIS_POOL_SIZE",
		"MBFLOW_LOG_LEVEL", "MBFLOW_LOG_FORMAT",
		"MBFLOW_OBSERVER_DB_ENABLED", "MBFLOW_OBSERVER_HTTP_ENABLED", "MBFLOW_OBSERVER_HTTP_URL", "MBFLOW_OBSERVER_HTTP_METHOD",
		"MBFLOW_OBSERVER_HTTP_TIMEOUT", "MBFLOW_OBSERVER_HTTP_MAX_RETRIES", "MBFLOW_OBSERVER_HTTP_RETRY_DELAY", "MBFLOW_OBSERVER_HTTP_HEADERS",
		"MBFLOW_OBSERVER_LOGGER_ENABLED", "MBFLOW_OBSERVER_WEBSOCKET_ENABLED", "MBFLOW_OBSERVER_WEBSOCKET_BUFFER_SIZE", "MBFLOW_OBSERVER_BUFFER_SIZE",
		"MBFLOW_AUTH_MODE", "MBFLOW_JWT_SECRET", "MBFLOW_JWT_EXPIRATION_HOURS", "MBFLOW_JWT_REFRESH_DAYS",
		"MBFLOW_SESSION_DURATION", "MBFLOW_MAX_SESSIONS_PER_USER", "MBFLOW_MIN_PASSWORD_LENGTH",
		"MBFLOW_AUTH_GATEWAY_URL", "MBFLOW_AUTH_CLIENT_ID", "MBFLOW_AUTH_GRPC_ADDRESS",
	}

	for _, key := range envVars {
		os.Unsetenv(key)
	}
}
