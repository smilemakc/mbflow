package monitoring

import (
	"fmt"
	"log"
	"sync"
	"time"

	"mbflow/internal/domain"
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
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeStartedFromConfig should be used instead.
func (l *ExecutionLogger) LogNodeStarted(executionID string, node *domain.Node, attemptNumber int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		log.Printf("[%s] Node started: execution=%s node=<nil> attempt=%d",
			l.prefix, executionID, attemptNumber)
		return
	}

	if attemptNumber > 1 {
		log.Printf("[%s] Node started (retry %d): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
			l.prefix, attemptNumber, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config())
	} else {
		log.Printf("[%s] Node started: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
			l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config())
	}
}

// LogNodeStartedFromConfig logs when a node starts executing from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ExecutionLogger) LogNodeStartedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if attemptNumber > 1 {
		log.Printf("[%s] Node started (retry %d): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
			l.prefix, attemptNumber, executionID, nodeID, workflowID, nodeType, name, config)
	} else {
		log.Printf("[%s] Node started: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
			l.prefix, executionID, nodeID, workflowID, nodeType, name, config)
	}
}

// LogNodeCompleted logs when a node completes successfully.
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeCompletedFromConfig should be used instead.
func (l *ExecutionLogger) LogNodeCompleted(executionID string, node *domain.Node, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		log.Printf("[%s] Node completed: execution=%s node=<nil> duration=%s",
			l.prefix, executionID, duration)
		return
	}

	log.Printf("[%s] Node completed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s",
		l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), duration)
}

// LogNodeCompletedFromConfig logs when a node completes successfully from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ExecutionLogger) LogNodeCompletedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	log.Printf("[%s] Node completed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s",
		l.prefix, executionID, nodeID, workflowID, nodeType, name, config, duration)
}

// LogNodeFailed logs when a node fails.
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeFailedFromConfig should be used instead.
func (l *ExecutionLogger) LogNodeFailed(executionID string, node *domain.Node, err error, duration time.Duration, willRetry bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		if willRetry {
			log.Printf("[%s] Node failed (will retry): execution=%s node=<nil> duration=%s error=%v",
				l.prefix, executionID, duration, err)
		} else {
			log.Printf("[%s] Node failed: execution=%s node=<nil> duration=%s error=%v",
				l.prefix, executionID, duration, err)
		}
		return
	}

	if willRetry {
		log.Printf("[%s] Node failed (will retry): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
			l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), duration, err)
	} else {
		log.Printf("[%s] Node failed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
			l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), duration, err)
	}
}

// LogNodeFailedFromConfig logs when a node fails from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ExecutionLogger) LogNodeFailedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, err error, duration time.Duration, willRetry bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if willRetry {
		log.Printf("[%s] Node failed (will retry): execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
			l.prefix, executionID, nodeID, workflowID, nodeType, name, config, duration, err)
	} else {
		log.Printf("[%s] Node failed: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v duration=%s error=%v",
			l.prefix, executionID, nodeID, workflowID, nodeType, name, config, duration, err)
	}
}

// LogNodeRetrying logs when a node is being retried.
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeRetryingFromConfig should be used instead.
func (l *ExecutionLogger) LogNodeRetrying(executionID string, node *domain.Node, attemptNumber int, delay time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		log.Printf("[%s] Node retrying: execution=%s node=<nil> attempt=%d delay=%s",
			l.prefix, executionID, attemptNumber, delay)
		return
	}

	log.Printf("[%s] Node retrying: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v attempt=%d delay=%s",
		l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), attemptNumber, delay)
}

// LogNodeRetryingFromConfig logs when a node is being retried from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ExecutionLogger) LogNodeRetryingFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int, delay time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	log.Printf("[%s] Node retrying: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v attempt=%d delay=%s",
		l.prefix, executionID, nodeID, workflowID, nodeType, name, config, attemptNumber, delay)
}

// LogNodeSkipped logs when a node is skipped.
// It accepts either a domain.Node or its configuration.
// If node is nil, LogNodeSkippedFromConfig should be used instead.
func (l *ExecutionLogger) LogNodeSkipped(executionID string, node *domain.Node, reason string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		log.Printf("[%s] Node skipped: execution=%s node=<nil> reason=%s",
			l.prefix, executionID, reason)
		return
	}

	log.Printf("[%s] Node skipped: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v reason=%s",
		l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config(), reason)
}

// LogNodeSkippedFromConfig logs when a node is skipped from its configuration.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ExecutionLogger) LogNodeSkippedFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, reason string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	log.Printf("[%s] Node skipped: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v reason=%s",
		l.prefix, executionID, nodeID, workflowID, nodeType, name, config, reason)
}

// LogNode logs all fields of a node.
// It accepts either a domain.Node or its configuration.
// If node is provided, all fields are extracted from it.
// If node is nil, LogNodeFromConfig should be used instead.
func (l *ExecutionLogger) LogNode(executionID string, node *domain.Node) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node == nil {
		log.Printf("[%s] Node info: execution=%s node=<nil>", l.prefix, executionID)
		return
	}

	log.Printf("[%s] Node info: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
		l.prefix, executionID, node.ID(), node.WorkflowID(), node.Type(), node.Name(), node.Config())
}

// LogNodeFromConfig logs all fields of a node from its configuration and metadata.
// This method is used when you have the node configuration but not the full domain.Node object.
func (l *ExecutionLogger) LogNodeFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	log.Printf("[%s] Node info: execution=%s node_id=%s workflow_id=%s node_type=%s name=%s config=%v",
		l.prefix, executionID, nodeID, workflowID, nodeType, name, config)
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
