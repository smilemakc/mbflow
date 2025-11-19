package monitoring

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ExecutionLogger provides structured logging for workflow execution.
// It logs node transitions, errors, and execution events with context.
type ExecutionLogger struct {
	// prefix is prepended to all log messages
	prefix string
	// verbose enables verbose logging
	verbose bool
	// mu protects concurrent writes
	mu sync.Mutex
}

// NewExecutionLogger creates a new ExecutionLogger.
func NewExecutionLogger(prefix string, verbose bool) *ExecutionLogger {
	return &ExecutionLogger{
		prefix:  prefix,
		verbose: verbose,
	}
}

// LogExecutionStarted logs when a workflow execution starts.
func (l *ExecutionLogger) LogExecutionStarted(workflowID, executionID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Execution started: workflow=%s execution=%s", l.prefix, workflowID, executionID)
}

// LogExecutionCompleted logs when a workflow execution completes successfully.
func (l *ExecutionLogger) LogExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Execution completed: workflow=%s execution=%s duration=%s",
		l.prefix, workflowID, executionID, duration)
}

// LogExecutionFailed logs when a workflow execution fails.
func (l *ExecutionLogger) LogExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Execution failed: workflow=%s execution=%s duration=%s error=%v",
		l.prefix, workflowID, executionID, duration, err)
}

// LogNodeStarted logs when a node starts executing.
func (l *ExecutionLogger) LogNodeStarted(executionID, nodeID, nodeType string, attemptNumber int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if attemptNumber > 1 {
		log.Printf("[%s] Node started (retry %d): execution=%s node=%s type=%s",
			l.prefix, attemptNumber, executionID, nodeID, nodeType)
	} else {
		log.Printf("[%s] Node started: execution=%s node=%s type=%s",
			l.prefix, executionID, nodeID, nodeType)
	}
}

// LogNodeCompleted logs when a node completes successfully.
func (l *ExecutionLogger) LogNodeCompleted(executionID, nodeID, nodeType string, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Node completed: execution=%s node=%s type=%s duration=%s",
		l.prefix, executionID, nodeID, nodeType, duration)
}

// LogNodeFailed logs when a node fails.
func (l *ExecutionLogger) LogNodeFailed(executionID, nodeID, nodeType string, err error, duration time.Duration, willRetry bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if willRetry {
		log.Printf("[%s] Node failed (will retry): execution=%s node=%s type=%s duration=%s error=%v",
			l.prefix, executionID, nodeID, nodeType, duration, err)
	} else {
		log.Printf("[%s] Node failed: execution=%s node=%s type=%s duration=%s error=%v",
			l.prefix, executionID, nodeID, nodeType, duration, err)
	}
}

// LogNodeRetrying logs when a node is being retried.
func (l *ExecutionLogger) LogNodeRetrying(executionID, nodeID string, attemptNumber int, delay time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Node retrying: execution=%s node=%s attempt=%d delay=%s",
		l.prefix, executionID, nodeID, attemptNumber, delay)
}

// LogNodeSkipped logs when a node is skipped.
func (l *ExecutionLogger) LogNodeSkipped(executionID, nodeID, nodeType, reason string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Node skipped: execution=%s node=%s type=%s reason=%s",
		l.prefix, executionID, nodeID, nodeType, reason)
}

// LogVariableSet logs when a variable is set (verbose mode only).
func (l *ExecutionLogger) LogVariableSet(executionID, key string, value interface{}) {
	if !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Variable set: execution=%s key=%s value=%v",
		l.prefix, executionID, key, value)
}

// LogError logs a general error.
func (l *ExecutionLogger) LogError(executionID string, message string, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Error: execution=%s message=%s error=%v",
		l.prefix, executionID, message, err)
}

// LogInfo logs an informational message.
func (l *ExecutionLogger) LogInfo(executionID string, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Info: execution=%s message=%s", l.prefix, executionID, message)
}

// LogDebug logs a debug message (verbose mode only).
func (l *ExecutionLogger) LogDebug(executionID string, message string) {
	if !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] Debug: execution=%s message=%s", l.prefix, executionID, message)
}

// LogTransition logs a state transition.
func (l *ExecutionLogger) LogTransition(executionID, nodeID, fromState, toState string) {
	if !l.verbose {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	log.Printf("[%s] State transition: execution=%s node=%s from=%s to=%s",
		l.prefix, executionID, nodeID, fromState, toState)
}

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
