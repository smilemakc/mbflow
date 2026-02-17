package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/smilemakc/mbflow/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== New() Tests ====================

func TestNew_JSONFormat_InfoLevel(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
	}

	logger := New(cfg)
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.logger)
}

func TestNew_TextFormat_DebugLevel(t *testing.T) {
	cfg := config.LoggingConfig{
		Level:  "debug",
		Format: "text",
	}

	logger := New(cfg)
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.logger)
}

func TestNew_AllLevels(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"Debug level", "debug"},
		{"Info level", "info"},
		{"Warn level", "warn"},
		{"Error level", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.LoggingConfig{
				Level:  tt.level,
				Format: "json",
			}

			logger := New(cfg)
			assert.NotNil(t, logger)
		})
	}
}

func TestNew_AllFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"JSON format", "json"},
		{"Text format", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.LoggingConfig{
				Level:  "info",
				Format: tt.format,
			}

			logger := New(cfg)
			assert.NotNil(t, logger)
		})
	}
}

// ==================== With() Tests ====================

func TestLogger_With_SingleAttribute(t *testing.T) {
	logger := New(config.LoggingConfig{
		Level:  "info",
		Format: "json",
	})

	childLogger := logger.With("key", "value")
	assert.NotNil(t, childLogger)
	assert.NotEqual(t, logger, childLogger)
}

func TestLogger_With_MultipleAttributes(t *testing.T) {
	logger := New(config.LoggingConfig{
		Level:  "info",
		Format: "json",
	})

	childLogger := logger.With("key1", "value1", "key2", "value2")
	assert.NotNil(t, childLogger)
	assert.NotEqual(t, logger, childLogger)
}

func TestLogger_With_ChainedCalls(t *testing.T) {
	logger := New(config.LoggingConfig{
		Level:  "info",
		Format: "json",
	})

	logger1 := logger.With("key1", "value1")
	logger2 := logger1.With("key2", "value2")
	logger3 := logger2.With("key3", "value3")

	assert.NotNil(t, logger1)
	assert.NotNil(t, logger2)
	assert.NotNil(t, logger3)

	// All should be different instances
	assert.NotEqual(t, logger, logger1)
	assert.NotEqual(t, logger1, logger2)
	assert.NotEqual(t, logger2, logger3)
}

// ==================== WithContext() Tests ====================

func TestLogger_WithContext_Success(t *testing.T) {
	logger := New(config.LoggingConfig{
		Level:  "info",
		Format: "json",
	})

	ctx := context.Background()
	contextLogger := logger.WithContext(ctx)
	assert.NotNil(t, contextLogger)
	// Currently returns same logger, but method should work
	assert.Equal(t, logger, contextLogger)
}

func TestLogger_WithContext_WithValues(t *testing.T) {
	logger := New(config.LoggingConfig{
		Level:  "info",
		Format: "json",
	})

	ctx := context.WithValue(context.Background(), "request_id", "123")
	contextLogger := logger.WithContext(ctx)
	assert.NotNil(t, contextLogger)
}

// ==================== Logging Methods Tests ====================

func TestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "debug", "json")

	logger.Debug("test debug message")

	output := buf.String()
	assert.Contains(t, output, "test debug message")
	assert.Contains(t, output, "DEBUG")
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "info", "json")

	logger.Info("test info message")

	output := buf.String()
	assert.Contains(t, output, "test info message")
	assert.Contains(t, output, "INFO")
}

func TestLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "warn", "json")

	logger.Warn("test warning message")

	output := buf.String()
	assert.Contains(t, output, "test warning message")
	assert.Contains(t, output, "WARN")
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "error", "json")

	logger.Error("test error message")

	output := buf.String()
	assert.Contains(t, output, "test error message")
	assert.Contains(t, output, "ERROR")
}

func TestLogger_DebugWithAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "debug", "json")

	logger.Debug("debug with attrs", "key1", "value1", "key2", 42)

	output := buf.String()
	assert.Contains(t, output, "debug with attrs")
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "key2")
	assert.Contains(t, output, "42")
}

func TestLogger_InfoWithAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "info", "json")

	logger.Info("info with attrs", "user", "alice", "count", 100)

	output := buf.String()
	assert.Contains(t, output, "info with attrs")
	assert.Contains(t, output, "user")
	assert.Contains(t, output, "alice")
	assert.Contains(t, output, "count")
	assert.Contains(t, output, "100")
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "warn", "json")

	// These should NOT be logged (below warn level)
	logger.Debug("debug message")
	logger.Info("info message")

	// These should be logged
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

// ==================== Context Methods Tests ====================

func TestLogger_DebugContext(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "debug", "json")

	ctx := context.Background()
	logger.DebugContext(ctx, "debug with context")

	output := buf.String()
	assert.Contains(t, output, "debug with context")
}

func TestLogger_InfoContext(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "info", "json")

	ctx := context.Background()
	logger.InfoContext(ctx, "info with context")

	output := buf.String()
	assert.Contains(t, output, "info with context")
}

func TestLogger_WarnContext(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "warn", "json")

	ctx := context.Background()
	logger.WarnContext(ctx, "warn with context")

	output := buf.String()
	assert.Contains(t, output, "warn with context")
}

func TestLogger_ErrorContext(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "error", "json")

	ctx := context.Background()
	logger.ErrorContext(ctx, "error with context")

	output := buf.String()
	assert.Contains(t, output, "error with context")
}

func TestLogger_ContextWithAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "info", "json")

	ctx := context.Background()
	logger.InfoContext(ctx, "context with attrs", "request_id", "abc123")

	output := buf.String()
	assert.Contains(t, output, "context with attrs")
	assert.Contains(t, output, "request_id")
	assert.Contains(t, output, "abc123")
}

// ==================== parseLevel Tests ====================

func TestParseLevel_Debug(t *testing.T) {
	level := parseLevel("debug")
	assert.Equal(t, slog.LevelDebug, level)
}

func TestParseLevel_Info(t *testing.T) {
	level := parseLevel("info")
	assert.Equal(t, slog.LevelInfo, level)
}

func TestParseLevel_Warn(t *testing.T) {
	level := parseLevel("warn")
	assert.Equal(t, slog.LevelWarn, level)
}

func TestParseLevel_Error(t *testing.T) {
	level := parseLevel("error")
	assert.Equal(t, slog.LevelError, level)
}

func TestParseLevel_Unknown(t *testing.T) {
	level := parseLevel("unknown")
	assert.Equal(t, slog.LevelInfo, level) // Should default to info
}

func TestParseLevel_Empty(t *testing.T) {
	level := parseLevel("")
	assert.Equal(t, slog.LevelInfo, level) // Should default to info
}

// ==================== Global Logger Tests ====================

func TestDefault_ReturnsLogger(t *testing.T) {
	logger := Default()
	assert.NotNil(t, logger)
}

func TestSetDefault_Success(t *testing.T) {
	originalLogger := Default()

	newLogger := New(config.LoggingConfig{
		Level:  "debug",
		Format: "text",
	})

	SetDefault(newLogger)

	currentLogger := Default()
	assert.Equal(t, newLogger, currentLogger)
	assert.NotEqual(t, originalLogger, currentLogger)

	// Restore original logger
	SetDefault(originalLogger)
}

func TestGlobalDebug(t *testing.T) {
	// This test verifies the global Debug function works
	// It uses the default logger
	Debug("global debug test") // Should not panic
}

func TestGlobalInfo(t *testing.T) {
	Info("global info test") // Should not panic
}

func TestGlobalWarn(t *testing.T) {
	Warn("global warn test") // Should not panic
}

func TestGlobalError(t *testing.T) {
	Error("global error test") // Should not panic
}

// ==================== JSON Format Tests ====================

func TestLogger_JSONFormat_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "info", "json")

	logger.Info("test message", "key", "value")

	output := buf.String()

	// Should be valid JSON
	var jsonData map[string]any
	err := json.Unmarshal([]byte(output), &jsonData)
	require.NoError(t, err)

	// Verify JSON fields
	assert.Equal(t, "INFO", jsonData["level"])
	assert.Equal(t, "test message", jsonData["msg"])
	assert.Equal(t, "value", jsonData["key"])
}

func TestLogger_JSONFormat_MultipleAttributes(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "info", "json")

	logger.Info("test", "str", "value", "num", 42, "bool", true)

	output := buf.String()

	var jsonData map[string]any
	err := json.Unmarshal([]byte(output), &jsonData)
	require.NoError(t, err)

	assert.Equal(t, "value", jsonData["str"])
	assert.Equal(t, float64(42), jsonData["num"])
	assert.Equal(t, true, jsonData["bool"])
}

// ==================== Text Format Tests ====================

func TestLogger_TextFormat_Output(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "info", "text")

	logger.Info("test message", "key", "value")

	output := buf.String()

	// Text format should contain message and attributes
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
	assert.Contains(t, output, "INFO")
}

// ==================== Integration Tests ====================

func TestLogger_Integration_CompleteFlow(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "debug", "json")

	// 1. Log at different levels
	logger.Debug("step 1", "action", "start")
	logger.Info("step 2", "action", "processing")
	logger.Warn("step 3", "action", "warning")
	logger.Error("step 4", "action", "error")

	output := buf.String()

	// All messages should be present
	assert.Contains(t, output, "step 1")
	assert.Contains(t, output, "step 2")
	assert.Contains(t, output, "step 3")
	assert.Contains(t, output, "step 4")

	// Should have 4 JSON objects
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 4)
}

func TestLogger_Integration_WithChaining(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := newTestLogger(&buf, "info", "json")

	// Create child loggers with additional context
	userLogger := baseLogger.With("user_id", "123")
	requestLogger := userLogger.With("request_id", "abc")

	requestLogger.Info("request completed", "status", 200)

	output := buf.String()

	var jsonData map[string]any
	err := json.Unmarshal([]byte(output), &jsonData)
	require.NoError(t, err)

	// Should contain all context
	assert.Equal(t, "123", jsonData["user_id"])
	assert.Equal(t, "abc", jsonData["request_id"])
	assert.Equal(t, float64(200), jsonData["status"])
}

func TestLogger_Integration_ContextPropagation(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf, "info", "json")

	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", "xyz789")

	logger.InfoContext(ctx, "operation completed")

	output := buf.String()
	assert.Contains(t, output, "operation completed")
}

// ==================== Helper Functions ====================

func newTestLogger(buf *bytes.Buffer, level, format string) *Logger {
	var handler slog.Handler

	parsedLevel := parseLevel(level)
	opts := &slog.HandlerOptions{
		Level:     parsedLevel,
		AddSource: level == "debug",
	}

	if format == "json" {
		handler = slog.NewJSONHandler(buf, opts)
	} else {
		handler = slog.NewTextHandler(buf, opts)
	}

	return &Logger{
		logger: slog.New(handler),
	}
}
