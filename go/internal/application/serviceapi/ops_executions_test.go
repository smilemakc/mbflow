package serviceapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// --- ListExecutions ---

func TestListExecutions_ShouldFindAll_WhenNoFilters(t *testing.T) {
	// Arrange
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	now := time.Now()
	execModels := []*storagemodels.ExecutionModel{
		{ID: uuid.New(), WorkflowID: uuid.New(), Status: "completed", StartedAt: &now, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), WorkflowID: uuid.New(), Status: "running", StartedAt: &now, CreatedAt: now, UpdatedAt: now},
	}
	execRepo.On("FindAll", mock.Anything, 10, 0).Return(execModels, nil)

	// Act
	result, err := ops.ListExecutions(context.Background(), ListExecutionsParams{Limit: 10, Offset: 0})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Executions, 2)
	assert.Equal(t, 2, result.Total)
	execRepo.AssertExpectations(t)
}

func TestListExecutions_ShouldFilterByWorkflowID_WhenProvided(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	now := time.Now()
	execModels := []*storagemodels.ExecutionModel{
		{ID: uuid.New(), WorkflowID: wfID, Status: "completed", StartedAt: &now, CreatedAt: now, UpdatedAt: now},
	}
	execRepo.On("FindByWorkflowID", mock.Anything, wfID, 10, 0).Return(execModels, nil)

	result, err := ops.ListExecutions(context.Background(), ListExecutionsParams{
		Limit:      10,
		Offset:     0,
		WorkflowID: &wfID,
	})

	require.NoError(t, err)
	assert.Len(t, result.Executions, 1)
}

func TestListExecutions_ShouldFilterByStatus_WhenProvided(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	status := "failed"
	now := time.Now()
	execModels := []*storagemodels.ExecutionModel{
		{ID: uuid.New(), WorkflowID: uuid.New(), Status: "failed", StartedAt: &now, CreatedAt: now, UpdatedAt: now},
	}
	execRepo.On("FindByStatus", mock.Anything, "failed", 20, 5).Return(execModels, nil)

	result, err := ops.ListExecutions(context.Background(), ListExecutionsParams{
		Limit:  20,
		Offset: 5,
		Status: &status,
	})

	require.NoError(t, err)
	assert.Len(t, result.Executions, 1)
}

func TestListExecutions_ShouldWorkflowIDTakePrecedence_OverStatus(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	status := "running"
	execRepo.On("FindByWorkflowID", mock.Anything, wfID, 10, 0).Return([]*storagemodels.ExecutionModel{}, nil)

	result, err := ops.ListExecutions(context.Background(), ListExecutionsParams{
		Limit:      10,
		WorkflowID: &wfID,
		Status:     &status,
	})

	require.NoError(t, err)
	assert.Empty(t, result.Executions)
	execRepo.AssertNotCalled(t, "FindByStatus")
}

func TestListExecutions_ShouldReturnError_WhenRepoFails(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	execRepo.On("FindAll", mock.Anything, 10, 0).Return(([]*storagemodels.ExecutionModel)(nil), errors.New("db error"))

	result, err := ops.ListExecutions(context.Background(), ListExecutionsParams{Limit: 10})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestListExecutions_ShouldReturnEmptyList_WhenNoneFound(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	execRepo.On("FindAll", mock.Anything, 10, 0).Return([]*storagemodels.ExecutionModel{}, nil)

	result, err := ops.ListExecutions(context.Background(), ListExecutionsParams{Limit: 10})

	require.NoError(t, err)
	assert.Empty(t, result.Executions)
	assert.Equal(t, 0, result.Total)
}

// --- GetExecution ---

func TestGetExecution_ShouldReturnExecution_WhenFound(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	wfID := uuid.New()
	now := time.Now()

	execModel := &storagemodels.ExecutionModel{
		ID: execID, WorkflowID: wfID, Status: "completed", StartedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	}
	execRepo.On("FindByIDWithRelations", mock.Anything, execID).Return(execModel, nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{
		ID: wfID, Name: "Test WF", CreatedAt: now, UpdatedAt: now,
	}, nil)

	result, err := ops.GetExecution(context.Background(), GetExecutionParams{ExecutionID: execID})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, execID.String(), result.ID)
	assert.Equal(t, models.ExecutionStatus("completed"), result.Status)
}

func TestGetExecution_ShouldReturnError_WhenNotFound(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	execRepo.On("FindByIDWithRelations", mock.Anything, execID).Return((*storagemodels.ExecutionModel)(nil), models.ErrExecutionNotFound)

	result, err := ops.GetExecution(context.Background(), GetExecutionParams{ExecutionID: execID})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestGetExecution_ShouldEnrichNodeExecutions_WithWorkflowNodeInfo(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	wfID := uuid.New()
	nodeUUID := uuid.New()
	now := time.Now()

	execModel := &storagemodels.ExecutionModel{
		ID: execID, WorkflowID: wfID, Status: "completed", StartedAt: &now,
		CreatedAt: now, UpdatedAt: now,
		NodeExecutions: []*storagemodels.NodeExecutionModel{
			{
				ID: uuid.New(), ExecutionID: execID, NodeID: nodeUUID, Status: "completed",
				StartedAt: &now, CreatedAt: now, UpdatedAt: now,
			},
		},
	}
	execRepo.On("FindByIDWithRelations", mock.Anything, execID).Return(execModel, nil)

	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", CreatedAt: now, UpdatedAt: now,
		Nodes: []*storagemodels.NodeModel{
			{ID: nodeUUID, NodeID: "my-http-node", Name: "HTTP Request", Type: "http", WorkflowID: wfID, CreatedAt: now, UpdatedAt: now},
		},
	}
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(wfModel, nil)

	result, err := ops.GetExecution(context.Background(), GetExecutionParams{ExecutionID: execID})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.NodeExecutions, 1)
	// The node execution's NodeID should be replaced with the logical ID
	assert.Equal(t, "my-http-node", result.NodeExecutions[0].NodeID)
}

func TestGetExecution_ShouldHandleMissingWorkflow_Gracefully(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	wfID := uuid.New()
	now := time.Now()

	execModel := &storagemodels.ExecutionModel{
		ID: execID, WorkflowID: wfID, Status: "completed", StartedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	}
	execRepo.On("FindByIDWithRelations", mock.Anything, execID).Return(execModel, nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	// Should not error -- workflow lookup failure is non-fatal for GetExecution
	result, err := ops.GetExecution(context.Background(), GetExecutionParams{ExecutionID: execID})

	require.NoError(t, err)
	require.NotNil(t, result)
}

// --- CancelExecution ---

func TestCancelExecution_ShouldReturnNotImplementedError(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	err := ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: uuid.New()})

	require.Error(t, err)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NOT_IMPLEMENTED", opErr.Code)
}

// --- RetryExecution ---

func TestRetryExecution_ShouldReturnNotImplementedError(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	err := ops.RetryExecution(context.Background(), RetryExecutionParams{ExecutionID: uuid.New()})

	require.Error(t, err)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NOT_IMPLEMENTED", opErr.Code)
}

// --- GetExecutionLogs ---

func TestGetExecutionLogs_ShouldReturnLogs_WhenEventsExist(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	now := time.Now()
	events := []*storagemodels.EventModel{
		{ID: uuid.New(), ExecutionID: execID, EventType: "execution.started", Payload: storagemodels.JSONBMap{}, CreatedAt: now},
		{ID: uuid.New(), ExecutionID: execID, EventType: "node.started", Payload: storagemodels.JSONBMap{"node_name": "HTTP Step"}, CreatedAt: now.Add(1 * time.Second)},
		{ID: uuid.New(), ExecutionID: execID, EventType: "node.completed", Payload: storagemodels.JSONBMap{"node_name": "HTTP Step", "duration_ms": float64(150)}, CreatedAt: now.Add(2 * time.Second)},
		{ID: uuid.New(), ExecutionID: execID, EventType: "execution.completed", Payload: storagemodels.JSONBMap{"duration_ms": float64(500)}, CreatedAt: now.Add(3 * time.Second)},
	}
	execRepo.On("GetEvents", mock.Anything, execID).Return(events, nil)

	result, err := ops.GetExecutionLogs(context.Background(), GetExecutionLogsParams{ExecutionID: execID})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Logs, 4)
	assert.Equal(t, 4, result.Total)
	assert.Equal(t, "Execution started", result.Logs[0].Message)
	assert.Equal(t, "info", result.Logs[0].Level)
	assert.Equal(t, "Node 'HTTP Step' started", result.Logs[1].Message)
	assert.Equal(t, "Node 'HTTP Step' completed in 150ms", result.Logs[2].Message)
	assert.Equal(t, "success", result.Logs[2].Level)
	assert.Equal(t, "Execution completed in 500ms", result.Logs[3].Message)
}

func TestGetExecutionLogs_ShouldReturnEmptyLogs_WhenErrorFetchingEvents(t *testing.T) {
	// The method returns empty logs on error, not an error
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	execRepo.On("GetEvents", mock.Anything, execID).Return(([]*storagemodels.EventModel)(nil), errors.New("db error"))

	result, err := ops.GetExecutionLogs(context.Background(), GetExecutionLogsParams{ExecutionID: execID})

	require.NoError(t, err)
	assert.Empty(t, result.Logs)
	assert.Equal(t, 0, result.Total)
}

func TestGetExecutionLogs_ShouldReturnEmptyLogs_WhenNoEvents(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	execRepo.On("GetEvents", mock.Anything, execID).Return([]*storagemodels.EventModel{}, nil)

	result, err := ops.GetExecutionLogs(context.Background(), GetExecutionLogsParams{ExecutionID: execID})

	require.NoError(t, err)
	assert.Empty(t, result.Logs)
	assert.Equal(t, 0, result.Total)
}

// --- GetNodeResult ---

func TestGetNodeResult_ShouldReturnNodeExecution_WhenFound(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	wfID := uuid.New()
	nodeUUID := uuid.New()
	now := time.Now()

	execModel := &storagemodels.ExecutionModel{
		ID: execID, WorkflowID: wfID, Status: "completed", StartedAt: &now,
		CreatedAt: now, UpdatedAt: now,
		NodeExecutions: []*storagemodels.NodeExecutionModel{
			{
				ID: uuid.New(), ExecutionID: execID, NodeID: nodeUUID, Status: "completed",
				StartedAt: &now, OutputData: storagemodels.JSONBMap{"result": "ok"},
				CreatedAt: now, UpdatedAt: now,
			},
		},
	}
	execRepo.On("FindByIDWithRelations", mock.Anything, execID).Return(execModel, nil)

	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", CreatedAt: now, UpdatedAt: now,
		Nodes: []*storagemodels.NodeModel{
			{ID: nodeUUID, NodeID: "my-node", Name: "Node A", Type: "http", CreatedAt: now, UpdatedAt: now},
		},
	}
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(wfModel, nil)

	result, err := ops.GetNodeResult(context.Background(), GetNodeResultParams{
		ExecutionID: execID,
		NodeID:      "my-node",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "my-node", result.NodeID)
	assert.Equal(t, "ok", result.Output["result"])
}

func TestGetNodeResult_ShouldReturnError_WhenNodeNotFound(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	wfID := uuid.New()
	nodeUUID := uuid.New()
	now := time.Now()

	execModel := &storagemodels.ExecutionModel{
		ID: execID, WorkflowID: wfID, Status: "completed", StartedAt: &now,
		CreatedAt: now, UpdatedAt: now,
		NodeExecutions: []*storagemodels.NodeExecutionModel{
			{
				ID: uuid.New(), ExecutionID: execID, NodeID: nodeUUID, Status: "completed",
				StartedAt: &now, CreatedAt: now, UpdatedAt: now,
			},
		},
	}
	execRepo.On("FindByIDWithRelations", mock.Anything, execID).Return(execModel, nil)

	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", CreatedAt: now, UpdatedAt: now,
		Nodes: []*storagemodels.NodeModel{
			{ID: nodeUUID, NodeID: "some-other-node", Name: "Node B", Type: "http", CreatedAt: now, UpdatedAt: now},
		},
	}
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(wfModel, nil)

	result, err := ops.GetNodeResult(context.Background(), GetNodeResultParams{
		ExecutionID: execID,
		NodeID:      "nonexistent-node",
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NODE_EXECUTION_NOT_FOUND", opErr.Code)
}

func TestGetNodeResult_ShouldReturnError_WhenExecutionNotFound(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	ops := newTestOperations(nil, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	execRepo.On("FindByIDWithRelations", mock.Anything, execID).Return((*storagemodels.ExecutionModel)(nil), models.ErrExecutionNotFound)

	result, err := ops.GetNodeResult(context.Background(), GetNodeResultParams{
		ExecutionID: execID,
		NodeID:      "any-node",
	})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestGetNodeResult_ShouldReturnError_WhenWorkflowNotFound(t *testing.T) {
	execRepo := new(mockExecutionRepo)
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, execRepo, nil, nil, nil, nil, nil)

	execID := uuid.New()
	wfID := uuid.New()
	now := time.Now()

	execModel := &storagemodels.ExecutionModel{
		ID: execID, WorkflowID: wfID, Status: "completed", StartedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	}
	execRepo.On("FindByIDWithRelations", mock.Anything, execID).Return(execModel, nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	result, err := ops.GetNodeResult(context.Background(), GetNodeResultParams{
		ExecutionID: execID,
		NodeID:      "any",
	})

	assert.Nil(t, result)
	require.Error(t, err)
}

// --- getLogLevel ---

func TestGetLogLevel_ShouldReturnCorrectLevels(t *testing.T) {
	tests := []struct {
		eventType string
		expected  string
	}{
		{"execution.failed", "error"},
		{"node.failed", "error"},
		{"execution.completed", "success"},
		{"node.completed", "success"},
		{"wave.completed", "success"},
		{"execution.started", "info"},
		{"node.started", "info"},
		{"wave.started", "info"},
		{"node.retrying", "warning"},
		{"unknown.event", "info"},
		{"", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			assert.Equal(t, tt.expected, getLogLevel(tt.eventType))
		})
	}
}

// --- formatLogMessage ---

func TestFormatLogMessage_ShouldFormatAllEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		payload   map[string]any
		expected  string
	}{
		{
			name:      "execution started",
			eventType: "execution.started",
			payload:   map[string]any{},
			expected:  "Execution started",
		},
		{
			name:      "execution completed with duration",
			eventType: "execution.completed",
			payload:   map[string]any{"duration_ms": float64(1500)},
			expected:  "Execution completed in 1500ms",
		},
		{
			name:      "execution completed without duration",
			eventType: "execution.completed",
			payload:   map[string]any{},
			expected:  "Execution completed",
		},
		{
			name:      "execution failed with error",
			eventType: "execution.failed",
			payload:   map[string]any{"error": "timeout"},
			expected:  "Execution failed: timeout",
		},
		{
			name:      "execution failed without error",
			eventType: "execution.failed",
			payload:   map[string]any{},
			expected:  "Execution failed",
		},
		{
			name:      "wave started with index and count",
			eventType: "wave.started",
			payload:   map[string]any{"wave_index": float64(0), "node_count": float64(3)},
			expected:  "Wave 0 started with 3 nodes",
		},
		{
			name:      "wave started with index only",
			eventType: "wave.started",
			payload:   map[string]any{"wave_index": float64(1)},
			expected:  "Wave 1 started",
		},
		{
			name:      "wave started without payload",
			eventType: "wave.started",
			payload:   map[string]any{},
			expected:  "Wave started",
		},
		{
			name:      "wave completed with index",
			eventType: "wave.completed",
			payload:   map[string]any{"wave_index": float64(2)},
			expected:  "Wave 2 completed",
		},
		{
			name:      "wave completed without index",
			eventType: "wave.completed",
			payload:   map[string]any{},
			expected:  "Wave completed",
		},
		{
			name:      "node started with name",
			eventType: "node.started",
			payload:   map[string]any{"node_name": "HTTP Step"},
			expected:  "Node 'HTTP Step' started",
		},
		{
			name:      "node started without name",
			eventType: "node.started",
			payload:   map[string]any{},
			expected:  "Node started",
		},
		{
			name:      "node completed with name and duration",
			eventType: "node.completed",
			payload:   map[string]any{"node_name": "Transform", "duration_ms": float64(42)},
			expected:  "Node 'Transform' completed in 42ms",
		},
		{
			name:      "node completed with name only",
			eventType: "node.completed",
			payload:   map[string]any{"node_name": "Transform"},
			expected:  "Node 'Transform' completed",
		},
		{
			name:      "node completed without name",
			eventType: "node.completed",
			payload:   map[string]any{},
			expected:  "Node completed",
		},
		{
			name:      "node failed with name and error",
			eventType: "node.failed",
			payload:   map[string]any{"node_name": "API Call", "error": "timeout"},
			expected:  "Node 'API Call' failed: timeout",
		},
		{
			name:      "node failed with name only",
			eventType: "node.failed",
			payload:   map[string]any{"node_name": "API Call"},
			expected:  "Node 'API Call' failed",
		},
		{
			name:      "node failed without name",
			eventType: "node.failed",
			payload:   map[string]any{},
			expected:  "Node failed",
		},
		{
			name:      "node retrying with name",
			eventType: "node.retrying",
			payload:   map[string]any{"node_name": "Retry Step"},
			expected:  "Node 'Retry Step' retrying",
		},
		{
			name:      "node retrying without name",
			eventType: "node.retrying",
			payload:   map[string]any{},
			expected:  "Node retrying",
		},
		{
			name:      "unknown event type",
			eventType: "custom.event",
			payload:   map[string]any{},
			expected:  "custom.event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLogMessage(tt.eventType, tt.payload)
			assert.Equal(t, tt.expected, result)
		})
	}
}
