package monitoring

import (
	"time"

	"github.com/smilemakc/mbflow/internal/domain"
)

// EventType represents the type of log event.
type EventType string

// Event type constants
const (
	// Execution level events
	EventExecutionStarted   EventType = "execution_started"
	EventExecutionCompleted EventType = "execution_completed"
	EventExecutionFailed    EventType = "execution_failed"

	// Node level events
	EventNodeStarted   EventType = "node_started"
	EventNodeCompleted EventType = "node_completed"
	EventNodeFailed    EventType = "node_failed"
	EventNodeRetrying  EventType = "node_retrying"
	EventNodeSkipped   EventType = "node_skipped"

	// Variable events
	EventVariableSet EventType = "variable_set"

	// State transition events
	EventStateTransition EventType = "state_transition"

	// Callback events
	EventCallbackStarted   EventType = "callback_started"
	EventCallbackCompleted EventType = "callback_completed"

	// General events
	EventInfo  EventType = "info"
	EventDebug EventType = "debug"
	EventError EventType = "error"
)

// LogLevel represents the severity level of a log event.
type LogLevel string

const (
	LevelDebug   LogLevel = "debug"
	LevelInfo    LogLevel = "info"
	LevelWarning LogLevel = "warning"
	LevelError   LogLevel = "error"
)

// LogEvent represents a single log event with all relevant information.
type LogEvent struct {
	// Core fields
	Timestamp   time.Time `json:"timestamp"`
	Type        EventType `json:"type"`
	Level       LogLevel  `json:"level"`
	Message     string    `json:"message"`
	ExecutionID string    `json:"execution_id"`
	WorkflowID  string    `json:"workflow_id"`

	// Node fields (optional)
	NodeID   string         `json:"node_id,omitempty"`
	NodeType string         `json:"node_type,omitempty"`
	NodeName string         `json:"node_name,omitempty"`
	Config   map[string]any `json:"config,omitempty"`

	// Timing fields (optional)
	Duration time.Duration `json:"duration,omitempty"`

	// Retry fields (optional)
	AttemptNumber int           `json:"attempt_number,omitempty"`
	WillRetry     bool          `json:"will_retry,omitempty"`
	RetryDelay    time.Duration `json:"retry_delay,omitempty"`

	// Error fields (optional)
	Error        error  `json:"-"`                       // Original error
	ErrorMessage string `json:"error_message,omitempty"` // Error as string

	// Variable fields (optional)
	VariableKey   string      `json:"variable_key,omitempty"`
	VariableValue interface{} `json:"variable_value,omitempty"`

	// State transition fields (optional)
	FromState string `json:"from_state,omitempty"`
	ToState   string `json:"to_state,omitempty"`

	// Additional metadata (optional)
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Output data (optional)
	Output interface{} `json:"output,omitempty"`

	// Skip reason (optional)
	Reason string `json:"reason,omitempty"`
}

// Helper functions to create log events

// NewExecutionStartedEvent creates an execution started event.
func NewExecutionStartedEvent(workflowID, executionID string) *LogEvent {
	return &LogEvent{
		Timestamp:   time.Now(),
		Type:        EventExecutionStarted,
		Level:       LevelInfo,
		Message:     "Execution started",
		WorkflowID:  workflowID,
		ExecutionID: executionID,
	}
}

// NewExecutionCompletedEvent creates an execution completed event.
func NewExecutionCompletedEvent(workflowID, executionID string, duration time.Duration) *LogEvent {
	return &LogEvent{
		Timestamp:   time.Now(),
		Type:        EventExecutionCompleted,
		Level:       LevelInfo,
		Message:     "Execution completed",
		WorkflowID:  workflowID,
		ExecutionID: executionID,
		Duration:    duration,
	}
}

// NewExecutionFailedEvent creates an execution failed event.
func NewExecutionFailedEvent(workflowID, executionID string, err error, duration time.Duration) *LogEvent {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	return &LogEvent{
		Timestamp:    time.Now(),
		Type:         EventExecutionFailed,
		Level:        LevelError,
		Message:      "Execution failed",
		WorkflowID:   workflowID,
		ExecutionID:  executionID,
		Duration:     duration,
		Error:        err,
		ErrorMessage: errorMsg,
	}
}

// NewNodeStartedEvent creates a node started event from a domain.Node.
func NewNodeStartedEvent(workflowID, executionID string, node domain.Node, attemptNumber int) *LogEvent {
	if node == nil {
		return &LogEvent{
			Timestamp:     time.Now(),
			Type:          EventNodeStarted,
			Level:         LevelInfo,
			Message:       "Node started",
			ExecutionID:   executionID,
			WorkflowID:    workflowID,
			AttemptNumber: attemptNumber,
		}
	}

	return NewNodeStartedEventFromConfig(
		executionID,
		node.ID().String(),
		workflowID,
		string(node.Type()),
		node.Name(),
		node.Config(),
		attemptNumber,
	)
}

// NewNodeStartedEventFromConfig creates a node started event from configuration.
func NewNodeStartedEventFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int) *LogEvent {
	message := "Node started"
	if attemptNumber > 1 {
		message = "Node started (retry)"
	}

	return &LogEvent{
		Timestamp:     time.Now(),
		Type:          EventNodeStarted,
		Level:         LevelInfo,
		Message:       message,
		ExecutionID:   executionID,
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		NodeType:      nodeType,
		NodeName:      name,
		Config:        config,
		AttemptNumber: attemptNumber,
	}
}

// NewNodeCompletedEvent creates a node completed event from a domain.Node.
func NewNodeCompletedEvent(workflowID, executionID string, node domain.Node, output any, duration time.Duration) *LogEvent {
	if node == nil {
		return &LogEvent{
			Timestamp:   time.Now(),
			Type:        EventNodeCompleted,
			Level:       LevelInfo,
			Message:     "Node completed",
			ExecutionID: executionID,
			Duration:    duration,
			Output:      output,
		}
	}

	return NewNodeCompletedEventFromConfig(
		executionID,
		node.ID().String(),
		workflowID,
		string(node.Type()),
		node.Name(),
		node.Config(),
		duration,
	)
}

// NewNodeCompletedEventFromConfig creates a node completed event from configuration.
func NewNodeCompletedEventFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, duration time.Duration) *LogEvent {
	return &LogEvent{
		Timestamp:   time.Now(),
		Type:        EventNodeCompleted,
		Level:       LevelInfo,
		Message:     "Node completed",
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		NodeName:    name,
		Config:      config,
		Duration:    duration,
	}
}

// NewNodeFailedEvent creates a node failed event from a domain.Node.
func NewNodeFailedEvent(workflowID, executionID string, node domain.Node, err error, duration time.Duration, willRetry bool) *LogEvent {
	if node == nil {
		errorMsg := ""
		if err != nil {
			errorMsg = err.Error()
		}

		message := "Node failed"
		if willRetry {
			message = "Node failed (will retry)"
		}

		return &LogEvent{
			Timestamp:    time.Now(),
			Type:         EventNodeFailed,
			Level:        LevelError,
			Message:      message,
			ExecutionID:  executionID,
			Duration:     duration,
			Error:        err,
			ErrorMessage: errorMsg,
			WillRetry:    willRetry,
		}
	}

	return NewNodeFailedEventFromConfig(
		executionID,
		node.ID().String(),
		"",
		string(node.Type()),
		node.Name(),
		node.Config(),
		err,
		duration,
		willRetry,
	)
}

// NewNodeFailedEventFromConfig creates a node failed event from configuration.
func NewNodeFailedEventFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, err error, duration time.Duration, willRetry bool) *LogEvent {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	message := "Node failed"
	if willRetry {
		message = "Node failed (will retry)"
	}

	return &LogEvent{
		Timestamp:    time.Now(),
		Type:         EventNodeFailed,
		Level:        LevelError,
		Message:      message,
		ExecutionID:  executionID,
		WorkflowID:   workflowID,
		NodeID:       nodeID,
		NodeType:     nodeType,
		NodeName:     name,
		Config:       config,
		Duration:     duration,
		Error:        err,
		ErrorMessage: errorMsg,
		WillRetry:    willRetry,
	}
}

// NewNodeRetryingEvent creates a node retrying event from a domain.Node.
func NewNodeRetryingEvent(workflowID, executionID string, node domain.Node, attemptNumber int, delay time.Duration) *LogEvent {
	if node == nil {
		return &LogEvent{
			Timestamp:     time.Now(),
			Type:          EventNodeRetrying,
			Level:         LevelWarning,
			Message:       "Node retrying",
			ExecutionID:   executionID,
			AttemptNumber: attemptNumber,
			RetryDelay:    delay,
		}
	}

	return NewNodeRetryingEventFromConfig(
		executionID,
		node.ID().String(),
		workflowID,
		string(node.Type()),
		node.Name(),
		node.Config(),
		attemptNumber,
		delay,
	)
}

// NewNodeRetryingEventFromConfig creates a node retrying event from configuration.
func NewNodeRetryingEventFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, attemptNumber int, delay time.Duration) *LogEvent {
	return &LogEvent{
		Timestamp:     time.Now(),
		Type:          EventNodeRetrying,
		Level:         LevelWarning,
		Message:       "Node retrying",
		ExecutionID:   executionID,
		WorkflowID:    workflowID,
		NodeID:        nodeID,
		NodeType:      nodeType,
		NodeName:      name,
		Config:        config,
		AttemptNumber: attemptNumber,
		RetryDelay:    delay,
	}
}

// NewNodeSkippedEvent creates a node skipped event from a domain.Node.
func NewNodeSkippedEvent(workflowID, executionID string, node domain.Node, reason string) *LogEvent {
	if node == nil {
		return &LogEvent{
			Timestamp:   time.Now(),
			Type:        EventNodeSkipped,
			Level:       LevelInfo,
			Message:     "Node skipped",
			ExecutionID: executionID,
			Reason:      reason,
		}
	}

	return NewNodeSkippedEventFromConfig(
		executionID,
		node.ID().String(),
		workflowID,
		string(node.Type()),
		node.Name(),
		node.Config(),
		reason,
	)
}

// NewNodeSkippedEventFromConfig creates a node skipped event from configuration.
func NewNodeSkippedEventFromConfig(executionID, nodeID, workflowID, nodeType, name string, config map[string]any, reason string) *LogEvent {
	return &LogEvent{
		Timestamp:   time.Now(),
		Type:        EventNodeSkipped,
		Level:       LevelInfo,
		Message:     "Node skipped: " + reason,
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		NodeName:    name,
		Config:      config,
		Reason:      reason,
	}
}

// NewVariableSetEvent creates a variable set event.
func NewVariableSetEvent(workflowID, executionID, key string, value interface{}) *LogEvent {
	return &LogEvent{
		Timestamp:     time.Now(),
		Type:          EventVariableSet,
		Level:         LevelDebug,
		Message:       "Variable set: " + key,
		ExecutionID:   executionID,
		VariableKey:   key,
		VariableValue: value,
		WorkflowID:    workflowID,
	}
}

// NewStateTransitionEvent creates a state transition event.
func NewStateTransitionEvent(workflowID, executionID, nodeID, fromState, toState string) *LogEvent {
	return &LogEvent{
		Timestamp:   time.Now(),
		Type:        EventStateTransition,
		Level:       LevelDebug,
		Message:     "State transition: " + fromState + " -> " + toState,
		ExecutionID: executionID,
		NodeID:      nodeID,
		FromState:   fromState,
		ToState:     toState,
		WorkflowID:  workflowID,
	}
}

// NewInfoEvent creates an info level event.
func NewInfoEvent(workflowID, executionID, message string) *LogEvent {
	return &LogEvent{
		Timestamp:   time.Now(),
		Type:        EventInfo,
		Level:       LevelInfo,
		Message:     message,
		ExecutionID: executionID,
		WorkflowID:  workflowID,
	}
}

// NewDebugEvent creates a debug level event.
func NewDebugEvent(workflowID, executionID, message string) *LogEvent {
	return &LogEvent{
		Timestamp:   time.Now(),
		Type:        EventDebug,
		Level:       LevelDebug,
		Message:     message,
		ExecutionID: executionID,
		WorkflowID:  workflowID,
	}
}

// NewErrorEvent creates an error level event.
func NewErrorEvent(workflowID, executionID, message string, err error) *LogEvent {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	return &LogEvent{
		Timestamp:    time.Now(),
		Type:         EventError,
		Level:        LevelError,
		Message:      message,
		ExecutionID:  executionID,
		Error:        err,
		ErrorMessage: errorMsg,
		WorkflowID:   workflowID,
	}
}

func NewNodeCallbackStartedEvent(workflowID, executionID string, node domain.Node) *LogEvent {
	if node == nil {
		return &LogEvent{
			Timestamp:   time.Now(),
			Type:        EventCallbackStarted,
			Level:       LevelInfo,
			Message:     "Node callback started. Node is nil",
			ExecutionID: executionID,
			WorkflowID:  workflowID,
		}
	}
	return &LogEvent{
		Timestamp:   time.Now(),
		Type:        EventCallbackStarted,
		Level:       LevelInfo,
		Message:     "Node callback started",
		ExecutionID: executionID,
		NodeID:      node.ID().String(),
		NodeType:    string(node.Type()),
		NodeName:    node.Name(),
		Config:      node.Config(),
		WorkflowID:  workflowID,
	}
}

func NewNodeCallbackCompletedEvent(workflowID, executionID string, node domain.Node, output any, duration time.Duration) *LogEvent {
	if node == nil {
		return &LogEvent{
			Timestamp:   time.Now(),
			Type:        EventCallbackCompleted,
			Level:       LevelInfo,
			Message:     "Node callback completed",
			ExecutionID: executionID,
			Duration:    duration,
			Output:      output,
			WorkflowID:  workflowID,
		}
	}
	return &LogEvent{
		Timestamp:   time.Now(),
		Type:        EventCallbackCompleted,
		Level:       LevelInfo,
		Message:     "Node callback completed",
		ExecutionID: executionID,
		NodeID:      node.ID().String(),
		NodeType:    string(node.Type()),
		NodeName:    node.Name(),
		Config:      node.Config(),
		WorkflowID:  workflowID,
	}
}
