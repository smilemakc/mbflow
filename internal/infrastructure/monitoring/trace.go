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

// GetDuration returns the total duration of the execution based on first and last events.
func (t *ExecutionTrace) GetDuration() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.Events) < 2 {
		return 0
	}

	first := t.Events[0].Timestamp
	last := t.Events[len(t.Events)-1].Timestamp
	return last.Sub(first)
}

// GetEventsByType returns all events of a specific type.
func (t *ExecutionTrace) GetEventsByType(eventType string) []*TraceEvent {
	t.mu.Lock()
	defer t.mu.Unlock()

	var filtered []*TraceEvent
	for _, event := range t.Events {
		if event.EventType == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// GetEventsByNodeID returns all events for a specific node.
func (t *ExecutionTrace) GetEventsByNodeID(nodeID string) []*TraceEvent {
	t.mu.Lock()
	defer t.mu.Unlock()

	var filtered []*TraceEvent
	for _, event := range t.Events {
		if event.NodeID == nodeID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// GetErrorEvents returns all events that have an error.
func (t *ExecutionTrace) GetErrorEvents() []*TraceEvent {
	t.mu.Lock()
	defer t.mu.Unlock()

	var errors []*TraceEvent
	for _, event := range t.Events {
		if event.Error != nil {
			errors = append(errors, event)
		}
	}
	return errors
}

// HasErrors returns true if the trace contains any error events.
func (t *ExecutionTrace) HasErrors() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, event := range t.Events {
		if event.Error != nil {
			return true
		}
	}
	return false
}

// Summary returns a summary of the trace.
type TraceSummary struct {
	ExecutionID   string
	WorkflowID    string
	TotalEvents   int
	ErrorCount    int
	Duration      time.Duration
	StartTime     time.Time
	EndTime       time.Time
	EventTypes    map[string]int
	NodeIDs       []string
}

// GetSummary returns a summary of the trace.
func (t *ExecutionTrace) GetSummary() *TraceSummary {
	t.mu.Lock()
	defer t.mu.Unlock()

	summary := &TraceSummary{
		ExecutionID: t.ExecutionID,
		WorkflowID:  t.WorkflowID,
		TotalEvents: len(t.Events),
		EventTypes:  make(map[string]int),
		NodeIDs:     make([]string, 0),
	}

	nodeIDMap := make(map[string]bool)

	for i, event := range t.Events {
		// Track first and last timestamps
		if i == 0 {
			summary.StartTime = event.Timestamp
		}
		if i == len(t.Events)-1 {
			summary.EndTime = event.Timestamp
		}

		// Count event types
		summary.EventTypes[event.EventType]++

		// Count errors
		if event.Error != nil {
			summary.ErrorCount++
		}

		// Collect unique node IDs
		if event.NodeID != "" && !nodeIDMap[event.NodeID] {
			nodeIDMap[event.NodeID] = true
			summary.NodeIDs = append(summary.NodeIDs, event.NodeID)
		}
	}

	if len(t.Events) >= 2 {
		summary.Duration = summary.EndTime.Sub(summary.StartTime)
	}

	return summary
}
