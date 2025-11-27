package websocket

import (
	"time"
)

// Event types (server -> client)
const (
	EventExecutionStarted   = "execution.started"
	EventExecutionCompleted = "execution.completed"
	EventExecutionFailed    = "execution.failed"
	EventNodeStarted        = "node.started"
	EventNodeCompleted      = "node.completed"
	EventNodeFailed         = "node.failed"
	EventNodeRetrying       = "node.retrying"
	EventVariableSet        = "variable.set"
	EventCallbackStarted    = "callback.started"
	EventCallbackCompleted  = "callback.completed"
)

// Command types (client -> server)
const (
	CmdSubscribe   = "subscribe"
	CmdUnsubscribe = "unsubscribe"
	CmdCancel      = "cancel"
)

// WSEvent represents an event sent from server to client
type WSEvent struct {
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
	WorkflowID  string    `json:"workflow_id"`
	ExecutionID string    `json:"execution_id"`

	// Node-specific fields (optional)
	NodeID        string `json:"node_id,omitempty"`
	NodeName      string `json:"node_name,omitempty"`
	NodeType      string `json:"node_type,omitempty"`
	DurationMs    int64  `json:"duration_ms,omitempty"`
	Output        any    `json:"output,omitempty"`
	Error         string `json:"error,omitempty"`
	AttemptNumber int    `json:"attempt_number,omitempty"`
	WillRetry     bool   `json:"will_retry,omitempty"`
	DelayMs       int64  `json:"delay_ms,omitempty"`

	// Variable-specific
	Key   string `json:"key,omitempty"`
	Value any    `json:"value,omitempty"`
}

// WSCommand represents a command sent from client to server
type WSCommand struct {
	Action      string `json:"action"`
	ExecutionID string `json:"execution_id,omitempty"`
	WorkflowID  string `json:"workflow_id,omitempty"`
}

// WSResponse represents a response to a client command
type WSResponse struct {
	Type    string `json:"type"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// NewWSEvent creates a new WSEvent with the given type and IDs
func NewWSEvent(eventType, workflowID, executionID string) *WSEvent {
	return &WSEvent{
		Type:        eventType,
		Timestamp:   time.Now(),
		WorkflowID:  workflowID,
		ExecutionID: executionID,
	}
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(responseType, message string) *WSResponse {
	return &WSResponse{
		Type:    responseType,
		Success: true,
		Message: message,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(responseType, errorMsg string) *WSResponse {
	return &WSResponse{
		Type:    responseType,
		Success: false,
		Error:   errorMsg,
	}
}
