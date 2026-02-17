package sdk

import (
	"context"
	"testing"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecutionAPI_Run_EmptyWorkflowID tests that empty workflow ID is rejected
func TestExecutionAPI_Run_EmptyWorkflowID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().Run(ctx, "", nil)
	assert.ErrorIs(t, err, models.ErrInvalidWorkflowID)
}

// TestExecutionAPI_Run_ClosedClient tests that closed client returns error
func TestExecutionAPI_Run_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Executions().Run(ctx, "test-workflow-id", nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_Run_NotAvailableInStandalone tests that Run requires persistence
func TestExecutionAPI_Run_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().Run(ctx, "test-workflow-id", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_RunSync_EmptyWorkflowID tests that empty workflow ID is rejected
func TestExecutionAPI_RunSync_EmptyWorkflowID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().RunSync(ctx, "", nil)
	assert.ErrorIs(t, err, models.ErrInvalidWorkflowID)
}

// TestExecutionAPI_RunSync_ClosedClient tests that closed client returns error
func TestExecutionAPI_RunSync_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Executions().RunSync(ctx, "test-workflow-id", nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_RunSync_NotAvailableInStandalone tests that RunSync requires persistence
func TestExecutionAPI_RunSync_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().RunSync(ctx, "test-workflow-id", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_Get_EmptyExecutionID tests that empty execution ID is rejected
func TestExecutionAPI_Get_EmptyExecutionID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().Get(ctx, "")
	assert.ErrorIs(t, err, models.ErrInvalidExecutionID)
}

// TestExecutionAPI_Get_ClosedClient tests that closed client returns error
func TestExecutionAPI_Get_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Executions().Get(ctx, "test-execution-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_Get_NotAvailableInStandalone tests that Get requires persistence
func TestExecutionAPI_Get_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().Get(ctx, "test-execution-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_List_ClosedClient tests that closed client returns error
func TestExecutionAPI_List_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Executions().List(ctx, nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_List_NotAvailableInStandalone tests that List requires persistence
func TestExecutionAPI_List_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().List(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_List_WithOptions tests listing with filter options
func TestExecutionAPI_List_WithOptions(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	opts := &ExecutionListOptions{
		WorkflowID: "test-workflow",
		Status:     "completed",
		Limit:      10,
		Offset:     0,
	}

	_, err = client.Executions().List(ctx, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_Cancel_EmptyExecutionID tests that empty execution ID is rejected
func TestExecutionAPI_Cancel_EmptyExecutionID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Executions().Cancel(ctx, "")
	assert.ErrorIs(t, err, models.ErrInvalidExecutionID)
}

// TestExecutionAPI_Cancel_ClosedClient tests that closed client returns error
func TestExecutionAPI_Cancel_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	err = client.Executions().Cancel(ctx, "test-execution-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_Cancel_NotAvailableInStandalone tests that Cancel requires persistence
func TestExecutionAPI_Cancel_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Executions().Cancel(ctx, "test-execution-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_Retry_EmptyExecutionID tests that empty execution ID is rejected
func TestExecutionAPI_Retry_EmptyExecutionID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().Retry(ctx, "")
	assert.ErrorIs(t, err, models.ErrInvalidExecutionID)
}

// TestExecutionAPI_Retry_ClosedClient tests that closed client returns error
func TestExecutionAPI_Retry_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Executions().Retry(ctx, "test-execution-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_Retry_NotAvailableInStandalone tests that Retry requires persistence
func TestExecutionAPI_Retry_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().Retry(ctx, "test-execution-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_Watch_EmptyExecutionID tests that empty execution ID is rejected
func TestExecutionAPI_Watch_EmptyExecutionID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().Watch(ctx, "")
	assert.ErrorIs(t, err, models.ErrInvalidExecutionID)
}

// TestExecutionAPI_Watch_ClosedClient tests that closed client returns error
func TestExecutionAPI_Watch_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Executions().Watch(ctx, "test-execution-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_Watch_NotAvailableInStandalone tests that Watch requires persistence
func TestExecutionAPI_Watch_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().Watch(ctx, "test-execution-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_GetLogs_EmptyExecutionID tests that empty execution ID is rejected
func TestExecutionAPI_GetLogs_EmptyExecutionID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().GetLogs(ctx, "", nil)
	assert.ErrorIs(t, err, models.ErrInvalidExecutionID)
}

// TestExecutionAPI_GetLogs_ClosedClient tests that closed client returns error
func TestExecutionAPI_GetLogs_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Executions().GetLogs(ctx, "test-execution-id", nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_GetLogs_NotAvailableInStandalone tests that GetLogs requires persistence
func TestExecutionAPI_GetLogs_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	opts := &LogOptions{
		NodeID: "node1",
		Level:  "error",
		Limit:  100,
	}

	_, err = client.Executions().GetLogs(ctx, "test-execution-id", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_StreamLogs_EmptyExecutionID tests that empty execution ID is rejected
func TestExecutionAPI_StreamLogs_EmptyExecutionID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().StreamLogs(ctx, "", nil)
	assert.ErrorIs(t, err, models.ErrInvalidExecutionID)
}

// TestExecutionAPI_StreamLogs_ClosedClient tests that closed client returns error
func TestExecutionAPI_StreamLogs_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Executions().StreamLogs(ctx, "test-execution-id", nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_GetNodeResult_EmptyExecutionID tests that empty execution ID is rejected
func TestExecutionAPI_GetNodeResult_EmptyExecutionID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().GetNodeResult(ctx, "", "node1")
	assert.ErrorIs(t, err, models.ErrInvalidExecutionID)
}

// TestExecutionAPI_GetNodeResult_EmptyNodeID tests that empty node ID is rejected
func TestExecutionAPI_GetNodeResult_EmptyNodeID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().GetNodeResult(ctx, "test-execution-id", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "node ID is required")
}

// TestExecutionAPI_GetNodeResult_ClosedClient tests that closed client returns error
func TestExecutionAPI_GetNodeResult_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Executions().GetNodeResult(ctx, "test-execution-id", "node1")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestExecutionAPI_GetNodeResult_NotAvailableInStandalone tests that GetNodeResult requires persistence
func TestExecutionAPI_GetNodeResult_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Executions().GetNodeResult(ctx, "test-execution-id", "node1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available in standalone mode")
}

// TestExecutionAPI_ExecutionRequest_Creation tests ExecutionRequest struct
func TestExecutionAPI_ExecutionRequest_Creation(t *testing.T) {
	req := &ExecutionRequest{
		WorkflowID: "test-workflow",
		Input: map[string]any{
			"key": "value",
		},
		Async: true,
	}

	assert.Equal(t, "test-workflow", req.WorkflowID)
	assert.Equal(t, "value", req.Input["key"])
	assert.True(t, req.Async)
}

// TestExecutionAPI_ExecutionListOptions_Creation tests ExecutionListOptions struct
func TestExecutionAPI_ExecutionListOptions_Creation(t *testing.T) {
	startTime := int64(1000)
	endTime := int64(2000)

	opts := &ExecutionListOptions{
		WorkflowID: "test-workflow",
		Status:     "completed",
		Limit:      10,
		Offset:     5,
		StartTime:  &startTime,
		EndTime:    &endTime,
	}

	assert.Equal(t, "test-workflow", opts.WorkflowID)
	assert.Equal(t, "completed", opts.Status)
	assert.Equal(t, 10, opts.Limit)
	assert.Equal(t, 5, opts.Offset)
	assert.Equal(t, int64(1000), *opts.StartTime)
	assert.Equal(t, int64(2000), *opts.EndTime)
}

// TestExecutionAPI_ExecutionUpdate_Creation tests ExecutionUpdate struct
func TestExecutionAPI_ExecutionUpdate_Creation(t *testing.T) {
	update := &ExecutionUpdate{
		ExecutionID: "exec-123",
		Status:      "running",
		NodeID:      "node-1",
		Event:       "node_started",
		Data: map[string]any{
			"progress": 50,
		},
		Timestamp: 1234567890,
	}

	assert.Equal(t, "exec-123", update.ExecutionID)
	assert.Equal(t, "running", update.Status)
	assert.Equal(t, "node-1", update.NodeID)
	assert.Equal(t, "node_started", update.Event)
	assert.Equal(t, 50, update.Data["progress"])
	assert.Equal(t, int64(1234567890), update.Timestamp)
}

// TestExecutionAPI_LogOptions_Creation tests LogOptions struct
func TestExecutionAPI_LogOptions_Creation(t *testing.T) {
	opts := &LogOptions{
		NodeID: "node-1",
		Level:  "error",
		Limit:  100,
	}

	assert.Equal(t, "node-1", opts.NodeID)
	assert.Equal(t, "error", opts.Level)
	assert.Equal(t, 100, opts.Limit)
}
