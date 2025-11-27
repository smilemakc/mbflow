package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewWSEvent(t *testing.T) {
	before := time.Now()
	event := NewWSEvent(EventExecutionStarted, "wf-123", "exec-456")
	after := time.Now()

	assert.Equal(t, EventExecutionStarted, event.Type)
	assert.Equal(t, "wf-123", event.WorkflowID)
	assert.Equal(t, "exec-456", event.ExecutionID)
	assert.True(t, event.Timestamp.After(before) || event.Timestamp.Equal(before))
	assert.True(t, event.Timestamp.Before(after) || event.Timestamp.Equal(after))
}

func TestNewWSEvent_AllEventTypes(t *testing.T) {
	eventTypes := []string{
		EventExecutionStarted,
		EventExecutionCompleted,
		EventExecutionFailed,
		EventNodeStarted,
		EventNodeCompleted,
		EventNodeFailed,
		EventNodeRetrying,
		EventVariableSet,
		EventCallbackStarted,
		EventCallbackCompleted,
	}

	for _, eventType := range eventTypes {
		t.Run(eventType, func(t *testing.T) {
			event := NewWSEvent(eventType, "wf", "exec")
			assert.Equal(t, eventType, event.Type)
		})
	}
}

func TestNewSuccessResponse(t *testing.T) {
	resp := NewSuccessResponse(CmdSubscribe, "subscribed successfully")

	assert.Equal(t, CmdSubscribe, resp.Type)
	assert.True(t, resp.Success)
	assert.Equal(t, "subscribed successfully", resp.Message)
	assert.Empty(t, resp.Error)
}

func TestNewErrorResponse(t *testing.T) {
	resp := NewErrorResponse(CmdSubscribe, "invalid workflow_id")

	assert.Equal(t, CmdSubscribe, resp.Type)
	assert.False(t, resp.Success)
	assert.Empty(t, resp.Message)
	assert.Equal(t, "invalid workflow_id", resp.Error)
}

func TestWSEvent_JSONSerialization(t *testing.T) {
	event := NewWSEvent(EventNodeCompleted, "wf-123", "exec-456")
	event.NodeID = "node-789"
	event.NodeName = "process_data"
	event.NodeType = "action"
	event.DurationMs = 150
	event.Output = map[string]string{"result": "success"}

	data, err := json.Marshal(event)
	assert.NoError(t, err)

	var decoded WSEvent
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, event.Type, decoded.Type)
	assert.Equal(t, event.WorkflowID, decoded.WorkflowID)
	assert.Equal(t, event.ExecutionID, decoded.ExecutionID)
	assert.Equal(t, event.NodeID, decoded.NodeID)
	assert.Equal(t, event.NodeName, decoded.NodeName)
	assert.Equal(t, event.NodeType, decoded.NodeType)
	assert.Equal(t, event.DurationMs, decoded.DurationMs)
}

func TestWSEvent_JSONOmitEmpty(t *testing.T) {
	event := NewWSEvent(EventExecutionStarted, "wf-123", "exec-456")

	data, err := json.Marshal(event)
	assert.NoError(t, err)

	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	assert.NoError(t, err)

	// These fields should be present
	assert.Contains(t, m, "type")
	assert.Contains(t, m, "workflow_id")
	assert.Contains(t, m, "execution_id")
	assert.Contains(t, m, "timestamp")

	// These optional fields should be omitted when empty
	assert.NotContains(t, m, "node_id")
	assert.NotContains(t, m, "node_name")
	assert.NotContains(t, m, "node_type")
	assert.NotContains(t, m, "output")
	assert.NotContains(t, m, "error")
	assert.NotContains(t, m, "key")
	assert.NotContains(t, m, "value")
}

func TestWSCommand_JSONDeserialization(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected WSCommand
	}{
		{
			name:     "subscribe to workflow",
			json:     `{"action":"subscribe","workflow_id":"wf-123"}`,
			expected: WSCommand{Action: CmdSubscribe, WorkflowID: "wf-123"},
		},
		{
			name:     "subscribe to execution",
			json:     `{"action":"subscribe","execution_id":"exec-456"}`,
			expected: WSCommand{Action: CmdSubscribe, ExecutionID: "exec-456"},
		},
		{
			name:     "unsubscribe from workflow",
			json:     `{"action":"unsubscribe","workflow_id":"wf-123"}`,
			expected: WSCommand{Action: CmdUnsubscribe, WorkflowID: "wf-123"},
		},
		{
			name:     "cancel execution",
			json:     `{"action":"cancel","execution_id":"exec-456"}`,
			expected: WSCommand{Action: CmdCancel, ExecutionID: "exec-456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd WSCommand
			err := json.Unmarshal([]byte(tt.json), &cmd)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, cmd)
		})
	}
}

func TestWSResponse_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response *WSResponse
	}{
		{
			name:     "success response",
			response: NewSuccessResponse(CmdSubscribe, "subscribed"),
		},
		{
			name:     "error response",
			response: NewErrorResponse(CmdSubscribe, "invalid id"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			assert.NoError(t, err)

			var decoded WSResponse
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err)

			assert.Equal(t, tt.response.Type, decoded.Type)
			assert.Equal(t, tt.response.Success, decoded.Success)
			assert.Equal(t, tt.response.Message, decoded.Message)
			assert.Equal(t, tt.response.Error, decoded.Error)
		})
	}
}

func TestEventTypeConstants(t *testing.T) {
	// Verify event type constants have expected values
	assert.Equal(t, "execution.started", EventExecutionStarted)
	assert.Equal(t, "execution.completed", EventExecutionCompleted)
	assert.Equal(t, "execution.failed", EventExecutionFailed)
	assert.Equal(t, "node.started", EventNodeStarted)
	assert.Equal(t, "node.completed", EventNodeCompleted)
	assert.Equal(t, "node.failed", EventNodeFailed)
	assert.Equal(t, "node.retrying", EventNodeRetrying)
	assert.Equal(t, "variable.set", EventVariableSet)
	assert.Equal(t, "callback.started", EventCallbackStarted)
	assert.Equal(t, "callback.completed", EventCallbackCompleted)
}

func TestCommandTypeConstants(t *testing.T) {
	// Verify command type constants have expected values
	assert.Equal(t, "subscribe", CmdSubscribe)
	assert.Equal(t, "unsubscribe", CmdUnsubscribe)
	assert.Equal(t, "cancel", CmdCancel)
}
