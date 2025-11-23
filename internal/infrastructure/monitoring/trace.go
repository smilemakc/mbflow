package monitoring

import (
	"fmt"
	"sync"
	"time"
)

// ExecutionTrace represents a trace of execution events.
// It can be used for debugging and visualization.
type ExecutionTrace struct {
	ExecutionID string
	WorkflowID  string
	Events      []*TraceEvent
	mu          sync.Mutex
}

// TraceEvent represents a single event in the execution trace.
type TraceEvent struct {
	Timestamp time.Time
	EventType string
	NodeID    string
	NodeType  string
	Message   string
	Data      map[string]interface{}
	Error     error
}

// NewExecutionTrace creates a new ExecutionTrace.
func NewExecutionTrace(executionID, workflowID string) *ExecutionTrace {
	return &ExecutionTrace{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		Events:      make([]*TraceEvent, 0),
	}
}

// AddEvent adds an event to the trace.
func (t *ExecutionTrace) AddEvent(eventType, nodeID, nodeType, message string, data map[string]interface{}, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	event := &TraceEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		NodeID:    nodeID,
		NodeType:  nodeType,
		Message:   message,
		Data:      data,
		Error:     err,
	}
	t.Events = append(t.Events, event)
}

// GetEvents returns all events in the trace.
func (t *ExecutionTrace) GetEvents() []*TraceEvent {
	t.mu.Lock()
	defer t.mu.Unlock()

	events := make([]*TraceEvent, len(t.Events))
	copy(events, t.Events)
	return events
}

// String returns a string representation of the trace.
func (t *ExecutionTrace) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := fmt.Sprintf("Execution Trace [%s]\n", t.ExecutionID)
	result += fmt.Sprintf("Workflow: %s\n", t.WorkflowID)
	result += fmt.Sprintf("Events: %d\n\n", len(t.Events))

	for i, event := range t.Events {
		result += fmt.Sprintf("%d. [%s] %s", i+1, event.Timestamp.Format("15:04:05.000"), event.EventType)
		if event.NodeID != "" {
			result += fmt.Sprintf(" node=%s", event.NodeID)
		}
		if event.NodeType != "" {
			result += fmt.Sprintf(" type=%s", event.NodeType)
		}
		if event.Message != "" {
			result += fmt.Sprintf(" - %s", event.Message)
		}
		if event.Error != nil {
			result += fmt.Sprintf(" [ERROR: %v]", event.Error)
		}
		result += "\n"
	}

	return result
}
