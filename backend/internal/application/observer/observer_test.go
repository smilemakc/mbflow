package observer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventTypeFilter_ShouldNotify(t *testing.T) {
	tests := []struct {
		name         string
		allowedTypes []EventType
		event        Event
		shouldNotify bool
	}{
		{
			name:         "nil filter allows all events",
			allowedTypes: nil,
			event: Event{
				Type: EventTypeExecutionStarted,
			},
			shouldNotify: true,
		},
		{
			name:         "empty filter allows all events",
			allowedTypes: []EventType{},
			event: Event{
				Type: EventTypeNodeCompleted,
			},
			shouldNotify: true,
		},
		{
			name:         "filter allows execution.started",
			allowedTypes: []EventType{EventTypeExecutionStarted},
			event: Event{
				Type: EventTypeExecutionStarted,
			},
			shouldNotify: true,
		},
		{
			name:         "filter blocks execution.started",
			allowedTypes: []EventType{EventTypeNodeCompleted},
			event: Event{
				Type: EventTypeExecutionStarted,
			},
			shouldNotify: false,
		},
		{
			name: "filter allows multiple event types",
			allowedTypes: []EventType{
				EventTypeExecutionStarted,
				EventTypeExecutionCompleted,
				EventTypeExecutionFailed,
			},
			event: Event{
				Type: EventTypeExecutionCompleted,
			},
			shouldNotify: true,
		},
		{
			name: "filter blocks unlisted event type",
			allowedTypes: []EventType{
				EventTypeExecutionStarted,
				EventTypeExecutionCompleted,
			},
			event: Event{
				Type: EventTypeNodeFailed,
			},
			shouldNotify: false,
		},
		{
			name: "filter allows node events only",
			allowedTypes: []EventType{
				EventTypeNodeStarted,
				EventTypeNodeCompleted,
				EventTypeNodeFailed,
			},
			event: Event{
				Type: EventTypeNodeCompleted,
			},
			shouldNotify: true,
		},
		{
			name: "filter blocks wave events when only node events allowed",
			allowedTypes: []EventType{
				EventTypeNodeStarted,
				EventTypeNodeCompleted,
			},
			event: Event{
				Type: EventTypeWaveStarted,
			},
			shouldNotify: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filter EventFilter
			if tt.allowedTypes != nil {
				filter = NewEventTypeFilter(tt.allowedTypes...)
			}

			result := filter == nil || filter.ShouldNotify(tt.event)
			assert.Equal(t, tt.shouldNotify, result, "Filter decision mismatch")
		})
	}
}

func TestNewEventTypeFilter_NoTypes(t *testing.T) {
	// When no types are provided, should return nil (all events allowed)
	filter := NewEventTypeFilter()
	assert.Nil(t, filter, "Expected nil filter when no types provided")
}

func TestNewEventTypeFilter_SingleType(t *testing.T) {
	filter := NewEventTypeFilter(EventTypeExecutionStarted)
	assert.NotNil(t, filter, "Expected non-nil filter")

	typeFilter, ok := filter.(*EventTypeFilter)
	assert.True(t, ok, "Expected EventTypeFilter type")
	assert.Len(t, typeFilter.allowedTypes, 1, "Expected 1 allowed type")
	assert.True(t, typeFilter.allowedTypes[EventTypeExecutionStarted], "Expected execution.started to be allowed")
}

func TestNewEventTypeFilter_MultipleTypes(t *testing.T) {
	types := []EventType{
		EventTypeExecutionStarted,
		EventTypeExecutionCompleted,
		EventTypeNodeStarted,
		EventTypeNodeCompleted,
	}

	filter := NewEventTypeFilter(types...)
	assert.NotNil(t, filter, "Expected non-nil filter")

	typeFilter, ok := filter.(*EventTypeFilter)
	assert.True(t, ok, "Expected EventTypeFilter type")
	assert.Len(t, typeFilter.allowedTypes, 4, "Expected 4 allowed types")

	for _, eventType := range types {
		assert.True(t, typeFilter.allowedTypes[eventType], "Expected %s to be allowed", eventType)
	}
}

func TestEvent_AllFields(t *testing.T) {
	// Test that Event struct can hold all expected fields
	nodeID := "node-123"
	nodeName := "HTTP Request"
	nodeType := "http"
	waveIndex := 2
	nodeCount := 5
	durationMs := int64(1500)
	retryCount := 3
	testErr := assert.AnError

	event := Event{
		Type:        EventTypeNodeCompleted,
		ExecutionID: "exec-uuid-123",
		WorkflowID:  "wf-uuid-456",
		Timestamp:   time.Now(),
		NodeID:      &nodeID,
		NodeName:    &nodeName,
		NodeType:    &nodeType,
		WaveIndex:   &waveIndex,
		NodeCount:   &nodeCount,
		Status:      "completed",
		Error:       testErr,
		Input: map[string]interface{}{
			"url": "https://api.example.com",
		},
		Output: map[string]interface{}{
			"status": 200,
			"data":   "response",
		},
		Variables: map[string]interface{}{
			"user_id": "123",
		},
		DurationMs: &durationMs,
		RetryCount: &retryCount,
		Metadata: map[string]interface{}{
			"custom": "value",
		},
	}

	// Verify all fields are accessible
	assert.Equal(t, EventTypeNodeCompleted, event.Type)
	assert.Equal(t, "exec-uuid-123", event.ExecutionID)
	assert.Equal(t, "wf-uuid-456", event.WorkflowID)
	assert.NotNil(t, event.Timestamp)
	assert.Equal(t, "node-123", *event.NodeID)
	assert.Equal(t, "HTTP Request", *event.NodeName)
	assert.Equal(t, "http", *event.NodeType)
	assert.Equal(t, 2, *event.WaveIndex)
	assert.Equal(t, 5, *event.NodeCount)
	assert.Equal(t, "completed", event.Status)
	assert.Equal(t, testErr, event.Error)
	assert.NotNil(t, event.Input)
	assert.NotNil(t, event.Output)
	assert.NotNil(t, event.Variables)
	assert.Equal(t, int64(1500), *event.DurationMs)
	assert.Equal(t, 3, *event.RetryCount)
	assert.NotNil(t, event.Metadata)
}

func TestEventType_Constants(t *testing.T) {
	// Verify all event type constants are defined correctly
	assert.Equal(t, EventType("execution.started"), EventTypeExecutionStarted)
	assert.Equal(t, EventType("execution.completed"), EventTypeExecutionCompleted)
	assert.Equal(t, EventType("execution.failed"), EventTypeExecutionFailed)
	assert.Equal(t, EventType("wave.started"), EventTypeWaveStarted)
	assert.Equal(t, EventType("wave.completed"), EventTypeWaveCompleted)
	assert.Equal(t, EventType("node.started"), EventTypeNodeStarted)
	assert.Equal(t, EventType("node.completed"), EventTypeNodeCompleted)
	assert.Equal(t, EventType("node.failed"), EventTypeNodeFailed)
	assert.Equal(t, EventType("node.retrying"), EventTypeNodeRetrying)
}

func TestEventTypeFilter_NilSafety(t *testing.T) {
	// Test that nil filter safely allows all events
	var filter *EventTypeFilter
	event := Event{Type: EventTypeExecutionStarted}

	// Should not panic
	result := filter.ShouldNotify(event)
	assert.True(t, result, "Nil filter should allow all events")
}

func TestEventTypeFilter_ThreadSafety(t *testing.T) {
	// Test concurrent access to filter
	filter := NewEventTypeFilter(
		EventTypeExecutionStarted,
		EventTypeExecutionCompleted,
		EventTypeNodeCompleted,
	)

	done := make(chan bool, 10)

	// Launch 10 goroutines that concurrently check the filter
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				event := Event{Type: EventTypeExecutionStarted}
				filter.ShouldNotify(event)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// No assertion needed - test passes if no race condition detected
}
