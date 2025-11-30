// Package logger provides structured logging functionality.
package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/smilemakc/mbflow/internal/config"
)

// Logger wraps slog.Logger with additional context.
type Logger struct {
	logger *slog.Logger
}

// New creates a new logger based on the configuration.
func New(cfg config.LoggingConfig) *Logger {
	var handler slog.Handler

	// Parse log level
	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.Level == "debug",
	}

	// Create handler based on format
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return &Logger{
		logger: slog.New(handler),
	}
}

// With creates a new logger with the given attributes.
func (l *Logger) With(args ...interface{}) *Logger {
	return &Logger{
		logger: l.logger.With(args...),
	}
}

// WithContext creates a new logger with context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract context values if needed
	// For now, just return the same logger
	return l
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

// DebugContext logs a debug message with context.
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...interface{}) {
	l.logger.DebugContext(ctx, msg, args...)
}

// InfoContext logs an info message with context.
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...interface{}) {
	l.logger.InfoContext(ctx, msg, args...)
}

// WarnContext logs a warning message with context.
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...interface{}) {
	l.logger.WarnContext(ctx, msg, args...)
}

// ErrorContext logs an error message with context.
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...interface{}) {
	l.logger.ErrorContext(ctx, msg, args...)
}

// parseLevel parses a log level string to slog.Level.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Global logger for convenience
var defaultLogger *Logger

func init() {
	defaultLogger = New(config.LoggingConfig{
		Level:  "info",
		Format: "json",
	})
}

// Default returns the default logger.
func Default() *Logger {
	return defaultLogger
}

// SetDefault sets the default logger.
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// Debug logs a debug message using the default logger.
func Debug(msg string, args ...interface{}) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an info message using the default logger.
func Info(msg string, args ...interface{}) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning message using the default logger.
func Warn(msg string, args ...interface{}) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message using the default logger.
func Error(msg string, args ...interface{}) {
	defaultLogger.Error(msg, args...)
}
