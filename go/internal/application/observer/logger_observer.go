package observer

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
)

// LoggerObserver logs execution events to structured logger (slog)
type LoggerObserver struct {
	name   string
	logger *logger.Logger
	filter EventFilter
}

// LoggerObserverOption configures LoggerObserver
type LoggerObserverOption func(*LoggerObserver)

// WithLoggerInstance sets the logger instance
func WithLoggerInstance(l *logger.Logger) LoggerObserverOption {
	return func(o *LoggerObserver) {
		o.logger = l
	}
}

// WithLoggerFilter sets event filter
func WithLoggerFilter(filter EventFilter) LoggerObserverOption {
	return func(o *LoggerObserver) {
		o.filter = filter
	}
}

// NewLoggerObserver creates a new logger observer
func NewLoggerObserver(opts ...LoggerObserverOption) *LoggerObserver {
	obs := &LoggerObserver{
		name:   "logger",
		filter: nil, // No filter by default
	}

	for _, opt := range opts {
		opt(obs)
	}

	return obs
}

// Name returns the observer's name
func (o *LoggerObserver) Name() string {
	return o.name
}

// Filter returns the event filter
func (o *LoggerObserver) Filter() EventFilter {
	return o.filter
}

// OnEvent handles event logging
func (o *LoggerObserver) OnEvent(ctx context.Context, event Event) error {
	if o.logger == nil {
		return nil // No logger configured, skip silently
	}

	fields := []any{
		"event_type", string(event.Type),
		"execution_id", event.ExecutionID,
		"workflow_id", event.WorkflowID,
		"status", event.Status,
	}

	// Add node fields
	if event.NodeID != nil {
		fields = append(fields, "node_id", *event.NodeID)
		fields = append(fields, "node_name", *event.NodeName)
		fields = append(fields, "node_type", *event.NodeType)
	}

	// Add wave fields
	if event.WaveIndex != nil {
		fields = append(fields, "wave_index", *event.WaveIndex)
	}

	if event.NodeCount != nil {
		fields = append(fields, "node_count", *event.NodeCount)
	}

	// Add timing
	if event.DurationMs != nil {
		fields = append(fields, "duration_ms", *event.DurationMs)
	}

	// Build message
	msg := fmt.Sprintf("Workflow event: %s", event.Type)

	// Use appropriate log level
	if event.Error != nil {
		fields = append(fields, "error", event.Error.Error())
		o.logger.ErrorContext(ctx, msg, fields...)
	} else {
		o.logger.InfoContext(ctx, msg, fields...)
	}

	return nil
}
