package websocket

import (
	"time"

	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
)

// Ensure SocketObserver implements ExecutionObserver
var _ monitoring.ExecutionObserver = (*SocketObserver)(nil)

// SocketObserver implements monitoring.ExecutionObserver and broadcasts
// events to WebSocket clients through the Broadcaster interface.
type SocketObserver struct {
	hub Broadcaster
}

// NewSocketObserver creates a new SocketObserver
func NewSocketObserver(hub Broadcaster) *SocketObserver {
	return &SocketObserver{
		hub: hub,
	}
}

// OnExecutionStarted is called when a workflow execution starts
func (so *SocketObserver) OnExecutionStarted(workflowID, executionID string) {
	event := NewWSEvent(EventExecutionStarted, workflowID, executionID)
	so.hub.Broadcast("", workflowID, executionID, event)
}

// OnExecutionCompleted is called when a workflow execution completes successfully
func (so *SocketObserver) OnExecutionCompleted(workflowID, executionID string, duration time.Duration) {
	event := NewWSEvent(EventExecutionCompleted, workflowID, executionID)
	event.DurationMs = duration.Milliseconds()
	so.hub.Broadcast("", workflowID, executionID, event)
}

// OnExecutionFailed is called when a workflow execution fails
func (so *SocketObserver) OnExecutionFailed(workflowID, executionID string, err error, duration time.Duration) {
	event := NewWSEvent(EventExecutionFailed, workflowID, executionID)
	event.DurationMs = duration.Milliseconds()
	if err != nil {
		event.Error = err.Error()
	}
	so.hub.Broadcast("", workflowID, executionID, event)
}

// OnNodeStarted is called when a node starts executing
func (so *SocketObserver) OnNodeStarted(workflowID, executionID string, node domain.Node, attemptNumber int) {
	event := NewWSEvent(EventNodeStarted, workflowID, executionID)
	event.AttemptNumber = attemptNumber
	populateNodeFields(event, node)
	so.hub.Broadcast("", workflowID, executionID, event)
}

// OnNodeCompleted is called when a node completes successfully
func (so *SocketObserver) OnNodeCompleted(workflowID, executionID string, node domain.Node, output any, duration time.Duration) {
	event := NewWSEvent(EventNodeCompleted, workflowID, executionID)
	event.DurationMs = duration.Milliseconds()
	event.Output = output
	populateNodeFields(event, node)
	so.hub.Broadcast("", workflowID, executionID, event)
}

// OnNodeFailed is called when a node fails
func (so *SocketObserver) OnNodeFailed(workflowID, executionID string, node domain.Node, err error, duration time.Duration, willRetry bool) {
	event := NewWSEvent(EventNodeFailed, workflowID, executionID)
	event.DurationMs = duration.Milliseconds()
	event.WillRetry = willRetry
	if err != nil {
		event.Error = err.Error()
	}
	populateNodeFields(event, node)
	so.hub.Broadcast("", workflowID, executionID, event)
}

// OnNodeRetrying is called when a node is being retried
func (so *SocketObserver) OnNodeRetrying(workflowID, executionID string, node domain.Node, attemptNumber int, delay time.Duration) {
	event := NewWSEvent(EventNodeRetrying, workflowID, executionID)
	event.AttemptNumber = attemptNumber
	event.DelayMs = delay.Milliseconds()
	populateNodeFields(event, node)
	so.hub.Broadcast("", workflowID, executionID, event)
}

// OnVariableSet is called when a variable is set in the execution context
func (so *SocketObserver) OnVariableSet(workflowID, executionID, key string, value any) {
	event := NewWSEvent(EventVariableSet, workflowID, executionID)
	event.Key = key
	event.Value = value
	so.hub.Broadcast("", workflowID, executionID, event)
}

// OnNodeCallbackStarted is called when a node callback starts processing
func (so *SocketObserver) OnNodeCallbackStarted(workflowID, executionID string, node domain.Node) {
	event := NewWSEvent(EventCallbackStarted, workflowID, executionID)
	populateNodeFields(event, node)
	so.hub.Broadcast("", workflowID, executionID, event)
}

// OnNodeCallbackCompleted is called when a node callback completes
func (so *SocketObserver) OnNodeCallbackCompleted(workflowID, executionID string, node domain.Node, err error, duration time.Duration) {
	event := NewWSEvent(EventCallbackCompleted, workflowID, executionID)
	event.DurationMs = duration.Milliseconds()
	if err != nil {
		event.Error = err.Error()
	}
	populateNodeFields(event, node)
	so.hub.Broadcast("", workflowID, executionID, event)
}

// populateNodeFields extracts node information and populates event fields
func populateNodeFields(event *WSEvent, node domain.Node) {
	if node == nil {
		return
	}
	event.NodeID = node.ID().String()
	event.NodeName = node.Name()
	event.NodeType = string(node.Type())
}
