package observer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewLoggerObserver(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		obs := NewLoggerObserver()

		assert.NotNil(t, obs)
		assert.Equal(t, "logger", obs.Name())
		assert.Nil(t, obs.Filter())
		assert.Nil(t, obs.logger)
	})

	t.Run("with logger instance", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		})
		obs := NewLoggerObserver(WithLoggerInstance(log))

		assert.NotNil(t, obs)
		assert.NotNil(t, obs.logger)
	})

	t.Run("with filter", func(t *testing.T) {
		filter := NewEventTypeFilter(EventTypeExecutionStarted)
		obs := NewLoggerObserver(WithLoggerFilter(filter))

		assert.NotNil(t, obs)
		assert.NotNil(t, obs.Filter())
	})

	t.Run("with multiple options", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		})
		filter := NewEventTypeFilter(EventTypeExecutionStarted)
		obs := NewLoggerObserver(
			WithLoggerInstance(log),
			WithLoggerFilter(filter),
		)

		assert.NotNil(t, obs)
		assert.NotNil(t, obs.logger)
		assert.NotNil(t, obs.Filter())
	})
}

func TestLoggerObserver_Name(t *testing.T) {
	obs := NewLoggerObserver()
	assert.Equal(t, "logger", obs.Name())
}

func TestLoggerObserver_Filter(t *testing.T) {
	t.Run("no filter by default", func(t *testing.T) {
		obs := NewLoggerObserver()
		assert.Nil(t, obs.Filter())
	})

	t.Run("with filter", func(t *testing.T) {
		filter := NewEventTypeFilter(EventTypeExecutionStarted)
		obs := NewLoggerObserver(WithLoggerFilter(filter))
		assert.NotNil(t, obs.Filter())
	})
}

func TestLoggerObserver_OnEvent(t *testing.T) {
	t.Run("logs execution started event without error", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "info",
			Format: "json",
		})
		obs := NewLoggerObserver(WithLoggerInstance(log))

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "running",
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("logs failed event without error", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "info",
			Format: "json",
		})
		obs := NewLoggerObserver(WithLoggerInstance(log))

		testErr := errors.New("execution failed")
		event := Event{
			Type:        EventTypeExecutionFailed,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "failed",
			Error:       testErr,
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("logs node event with node details", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "info",
			Format: "json",
		})
		obs := NewLoggerObserver(WithLoggerInstance(log))

		nodeID := "node-123"
		nodeName := "HTTP Request"
		nodeType := "http"

		event := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			NodeID:      &nodeID,
			NodeName:    &nodeName,
			NodeType:    &nodeType,
			Status:      "completed",
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("logs wave event with wave details", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "info",
			Format: "json",
		})
		obs := NewLoggerObserver(WithLoggerInstance(log))

		waveIndex := 2
		nodeCount := 5

		event := Event{
			Type:        EventTypeWaveStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			WaveIndex:   &waveIndex,
			NodeCount:   &nodeCount,
			Status:      "running",
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("logs event with duration", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "info",
			Format: "json",
		})
		obs := NewLoggerObserver(WithLoggerInstance(log))

		durationMs := int64(1500)

		event := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "completed",
			DurationMs:  &durationMs,
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("handles nil logger gracefully", func(t *testing.T) {
		obs := NewLoggerObserver() // No logger configured

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "running",
		}

		// Should not panic, should return nil
		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("logs all event types correctly", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "info",
			Format: "json",
		})

		eventTypes := []struct {
			eventType EventType
			hasError  bool
		}{
			{EventTypeExecutionStarted, false},
			{EventTypeExecutionCompleted, false},
			{EventTypeExecutionFailed, true},
			{EventTypeWaveStarted, false},
			{EventTypeWaveCompleted, false},
			{EventTypeNodeStarted, false},
			{EventTypeNodeCompleted, false},
			{EventTypeNodeFailed, true},
			{EventTypeNodeRetrying, false},
		}

		for _, tt := range eventTypes {
			t.Run(string(tt.eventType), func(t *testing.T) {
				obs := NewLoggerObserver(WithLoggerInstance(log))

				event := Event{
					Type:        tt.eventType,
					ExecutionID: "exec-123",
					WorkflowID:  "wf-456",
					Timestamp:   time.Now(),
					Status:      "running",
				}

				if tt.hasError {
					event.Error = errors.New("test error")
					event.Status = "failed"
				}

				err := obs.OnEvent(context.Background(), event)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("includes all fields in log call", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		})
		obs := NewLoggerObserver(WithLoggerInstance(log))

		nodeID := "node-123"
		nodeName := "Transform"
		nodeType := "transform"
		waveIndex := 1
		nodeCount := 3
		durationMs := int64(500)

		event := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			NodeID:      &nodeID,
			NodeName:    &nodeName,
			NodeType:    &nodeType,
			WaveIndex:   &waveIndex,
			NodeCount:   &nodeCount,
			Status:      "completed",
			DurationMs:  &durationMs,
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})

	t.Run("logs in text format", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{
			Level:  "info",
			Format: "text",
		})
		obs := NewLoggerObserver(WithLoggerInstance(log))

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "running",
		}

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
	})
}
