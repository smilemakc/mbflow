package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Setup creates and configures a new logger instance.
// This is an infrastructure component that provides logging functionality.
func Setup(level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: l,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}

// Logger creates a default logger with info level.
func Logger() *slog.Logger {
	return Setup("info")
}
