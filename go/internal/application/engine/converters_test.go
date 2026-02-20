package engine

import (
	"testing"
	"time"

	"github.com/google/uuid"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowModelToDomain_Nil tests conversion of nil workflow
func TestWorkflowModelToDomain_Nil(t *testing.T) {
	result := storagemodels.WorkflowModelToDomain(nil)
	assert.Nil(t, result)
}

// TestWorkflowModelToDomain_Complete tests conversion with all fields populated
func TestWorkflowModelToDomain_Complete(t *testing.T) {
	wfID := uuid.New()
	nodeID1 := uuid.New()
	nodeID2 := uuid.New()
	createdAt := time.Now().Add(-1 * time.Hour)
	updatedAt := time.Now()

	storageWorkflow := &storagemodels.WorkflowModel{
		ID:          wfID,
		Name:        "Test Workflow",
		Description: "Test Description",
		Status:      "active",
		Variables: storagemodels.JSONBMap{
			"api_key": "secret",
			"timeout": 30,
		},
		Metadata: storagemodels.JSONBMap{
			"version": "1.0",
			"author":  "test",
		},
		Nodes: []*storagemodels.NodeModel{
			{
				ID:       nodeID1,
				NodeID:   "node1",
				Name:     "Node 1",
				Type:     "http",
				Config:   storagemodels.JSONBMap{"url": "https://example.com"},
				Position: storagemodels.JSONBMap{"x": 100.0, "y": 200.0},
			},
			{
				ID:     nodeID2,
				NodeID: "node2",
				Name:   "Node 2",
				Type:   "transform",
				Config: storagemodels.JSONBMap{"type": "passthrough"},
			},
		},
		Edges: []*storagemodels.EdgeModel{
			{
				EdgeID:     "edge1",
				FromNodeID: "node1",
				ToNodeID:   "node2",
				Condition:  storagemodels.JSONBMap{"expression": "output.status == 200"},
			},
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	result := storagemodels.WorkflowModelToDomain(storageWorkflow)
	require.NotNil(t, result)

	assert.Equal(t, wfID.String(), result.ID)
	assert.Equal(t, "Test Workflow", result.Name)
	assert.Equal(t, "Test Description", result.Description)
	assert.Equal(t, models.WorkflowStatus("active"), result.Status)

	// Check variables
	assert.Equal(t, "secret", result.Variables["api_key"])
	assert.Equal(t, 30, result.Variables["timeout"])

	// Check metadata
	assert.Equal(t, "1.0", result.Metadata["version"])
	assert.Equal(t, "test", result.Metadata["author"])

	// Check nodes
	assert.Len(t, result.Nodes, 2)
	assert.Equal(t, "node1", result.Nodes[0].ID)
	assert.Equal(t, "Node 1", result.Nodes[0].Name)
	assert.Equal(t, "http", result.Nodes[0].Type)
	assert.NotNil(t, result.Nodes[0].Position)
	assert.Equal(t, 100.0, result.Nodes[0].Position.X)
	assert.Equal(t, 200.0, result.Nodes[0].Position.Y)

	// Check edges
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, "edge1", result.Edges[0].ID)
	assert.Equal(t, "node1", result.Edges[0].From)
	assert.Equal(t, "node2", result.Edges[0].To)
	assert.Equal(t, "output.status == 200", result.Edges[0].Condition)

	// Check timestamps
	assert.Equal(t, createdAt, result.CreatedAt)
	assert.Equal(t, updatedAt, result.UpdatedAt)
}

// TestWorkflowModelToDomain_EmptyCollections tests workflow with empty nodes and edges
func TestWorkflowModelToDomain_EmptyCollections(t *testing.T) {
	wfID := uuid.New()

	storageWorkflow := &storagemodels.WorkflowModel{
		ID:    wfID,
		Name:  "Empty Workflow",
		Nodes: []*storagemodels.NodeModel{},
		Edges: []*storagemodels.EdgeModel{},
	}

	result := storagemodels.WorkflowModelToDomain(storageWorkflow)
	require.NotNil(t, result)

	assert.Empty(t, result.Nodes)
	assert.Empty(t, result.Edges)
	assert.NotNil(t, result.Variables)
	assert.NotNil(t, result.Metadata)
}

// TestNodeModelToDomain_Nil tests conversion of nil node
func TestNodeModelToDomain_Nil(t *testing.T) {
	result := storagemodels.NodeModelToDomain(nil)
	assert.Nil(t, result)
}

// TestNodeModelToDomain_WithPosition tests node conversion with position
func TestNodeModelToDomain_WithPosition(t *testing.T) {
	nodeID := uuid.New()

	storageNode := &storagemodels.NodeModel{
		ID:       nodeID,
		NodeID:   "test-node",
		Name:     "Test Node",
		Type:     "http",
		Config:   storagemodels.JSONBMap{"url": "https://api.example.com"},
		Position: storagemodels.JSONBMap{"x": 150.5, "y": 250.75},
	}

	result := storagemodels.NodeModelToDomain(storageNode)
	require.NotNil(t, result)

	assert.Equal(t, "test-node", result.ID)
	assert.Equal(t, "Test Node", result.Name)
	assert.Equal(t, "http", result.Type)
	assert.Equal(t, "https://api.example.com", result.Config["url"])
	require.NotNil(t, result.Position)
	assert.Equal(t, 150.5, result.Position.X)
	assert.Equal(t, 250.75, result.Position.Y)
}

// TestNodeModelToDomain_WithoutPosition tests node conversion without position
func TestNodeModelToDomain_WithoutPosition(t *testing.T) {
	storageNode := &storagemodels.NodeModel{
		ID:     uuid.New(),
		NodeID: "node-no-pos",
		Name:   "Node",
		Type:   "transform",
		Config: storagemodels.JSONBMap{"type": "passthrough"},
	}

	result := storagemodels.NodeModelToDomain(storageNode)
	require.NotNil(t, result)

	assert.Nil(t, result.Position)
}

// TestNodeModelToDomain_InvalidPosition tests node with invalid position data
func TestNodeModelToDomain_InvalidPosition(t *testing.T) {
	storageNode := &storagemodels.NodeModel{
		ID:       uuid.New(),
		NodeID:   "node-bad-pos",
		Name:     "Node",
		Type:     "transform",
		Position: storagemodels.JSONBMap{"x": "invalid", "y": 100.0},
	}

	result := storagemodels.NodeModelToDomain(storageNode)
	require.NotNil(t, result)

	// Invalid position should result in nil Position
	assert.Nil(t, result.Position)
}

// TestEdgeModelToDomain_Nil tests conversion of nil edge
func TestEdgeModelToDomain_Nil(t *testing.T) {
	result := storagemodels.EdgeModelToDomain(nil)
	assert.Nil(t, result)
}

// TestEdgeModelToDomain_WithCondition tests edge conversion with condition
func TestEdgeModelToDomain_WithCondition(t *testing.T) {
	storageEdge := &storagemodels.EdgeModel{
		EdgeID:     "edge-1",
		FromNodeID: "source",
		ToNodeID:   "target",
		Condition:  storagemodels.JSONBMap{"expression": "output.value > 10"},
	}

	result := storagemodels.EdgeModelToDomain(storageEdge)
	require.NotNil(t, result)

	assert.Equal(t, "edge-1", result.ID)
	assert.Equal(t, "source", result.From)
	assert.Equal(t, "target", result.To)
	assert.Equal(t, "output.value > 10", result.Condition)
}

// TestEdgeModelToDomain_WithoutCondition tests edge conversion without condition
func TestEdgeModelToDomain_WithoutCondition(t *testing.T) {
	storageEdge := &storagemodels.EdgeModel{
		EdgeID:     "edge-2",
		FromNodeID: "node1",
		ToNodeID:   "node2",
	}

	result := storagemodels.EdgeModelToDomain(storageEdge)
	require.NotNil(t, result)

	assert.Equal(t, "edge-2", result.ID)
	assert.Equal(t, "", result.Condition)
}

// TestEdgeModelToDomain_InvalidCondition tests edge with invalid condition format
func TestEdgeModelToDomain_InvalidCondition(t *testing.T) {
	storageEdge := &storagemodels.EdgeModel{
		EdgeID:     "edge-3",
		FromNodeID: "a",
		ToNodeID:   "b",
		Condition:  storagemodels.JSONBMap{"invalid": 123},
	}

	result := storagemodels.EdgeModelToDomain(storageEdge)
	require.NotNil(t, result)

	// Invalid condition should result in empty string
	assert.Equal(t, "", result.Condition)
}

// TestExecutionModelToDomain_Nil tests conversion of nil execution
func TestExecutionModelToDomain_Nil(t *testing.T) {
	result := storagemodels.ExecutionModelToDomain(nil)
	assert.Nil(t, result)
}

// TestExecutionModelToDomain_Complete tests complete execution conversion
func TestExecutionModelToDomain_Complete(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()
	nodeExecID := uuid.New()
	nodeID := uuid.New()
	startedAt := time.Now().Add(-5 * time.Minute)
	completedAt := time.Now()

	storageExec := &storagemodels.ExecutionModel{
		ID:          execID,
		WorkflowID:  wfID,
		Status:      "completed",
		StartedAt:   &startedAt,
		CompletedAt: &completedAt,
		InputData:   storagemodels.JSONBMap{"user_id": 123},
		OutputData:  storagemodels.JSONBMap{"result": "success"},
		Variables:   storagemodels.JSONBMap{"env": "production"},
		Error:       "",
		NodeExecutions: []*storagemodels.NodeExecutionModel{
			{
				ID:          nodeExecID,
				ExecutionID: execID,
				NodeID:      nodeID,
				Status:      "completed",
				InputData:   storagemodels.JSONBMap{"input": "value"},
				OutputData:  storagemodels.JSONBMap{"output": "result"},
				RetryCount:  0,
			},
		},
	}

	result := storagemodels.ExecutionModelToDomain(storageExec)
	require.NotNil(t, result)

	assert.Equal(t, execID.String(), result.ID)
	assert.Equal(t, wfID.String(), result.WorkflowID)
	assert.Equal(t, models.ExecutionStatus("completed"), result.Status)
	assert.Equal(t, startedAt, result.StartedAt)
	assert.NotNil(t, result.CompletedAt)
	assert.Equal(t, completedAt, *result.CompletedAt)

	assert.Equal(t, 123, result.Input["user_id"])
	assert.Equal(t, "success", result.Output["result"])
	assert.Equal(t, "production", result.Variables["env"])
	assert.Equal(t, "", result.Error)

	assert.Len(t, result.NodeExecutions, 1)
	assert.Equal(t, nodeExecID.String(), result.NodeExecutions[0].ID)
}

// TestExecutionModelToDomain_WithError tests execution with error
func TestExecutionModelToDomain_WithError(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()
	startedAt := time.Now()

	storageExec := &storagemodels.ExecutionModel{
		ID:         execID,
		WorkflowID: wfID,
		Status:     "failed",
		StartedAt:  &startedAt,
		Error:      "node execution failed: timeout",
	}

	result := storagemodels.ExecutionModelToDomain(storageExec)
	require.NotNil(t, result)

	assert.Equal(t, models.ExecutionStatus("failed"), result.Status)
	assert.Equal(t, "node execution failed: timeout", result.Error)
	assert.Nil(t, result.CompletedAt)
}

// TestExecutionDomainToModel_Nil tests conversion of nil execution
func TestExecutionDomainToModel_Nil(t *testing.T) {
	result := storagemodels.ExecutionDomainToModel(nil)
	assert.Nil(t, result)
}

// TestExecutionDomainToModel_Complete tests complete execution conversion
func TestExecutionDomainToModel_Complete(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()
	startedAt := time.Now().Add(-2 * time.Minute)
	completedAt := time.Now()

	domainExec := &models.Execution{
		ID:          execID.String(),
		WorkflowID:  wfID.String(),
		Status:      models.ExecutionStatusCompleted,
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
		Input:       map[string]any{"test": "input"},
		Output:      map[string]any{"test": "output"},
		Variables:   map[string]any{"var": "value"},
		Error:       "",
		NodeExecutions: []*models.NodeExecution{
			{
				ID:          uuid.New().String(),
				ExecutionID: execID.String(),
				NodeID:      "node-1",
				Status:      models.NodeExecutionStatusCompleted,
			},
		},
	}

	result := storagemodels.ExecutionDomainToModel(domainExec)
	require.NotNil(t, result)

	assert.Equal(t, execID, result.ID)
	assert.Equal(t, wfID, result.WorkflowID)
	assert.Equal(t, "completed", result.Status)
	assert.NotNil(t, result.StartedAt)
	assert.NotNil(t, result.CompletedAt)

	assert.Equal(t, "input", result.InputData["test"])
	assert.Equal(t, "output", result.OutputData["test"])
	assert.Equal(t, "value", result.Variables["var"])

	assert.Len(t, result.NodeExecutions, 1)
}

// TestExecutionDomainToModel_InvalidUUIDs tests handling of invalid UUID strings
func TestExecutionDomainToModel_InvalidUUIDs(t *testing.T) {
	domainExec := &models.Execution{
		ID:         "invalid-uuid",
		WorkflowID: "also-invalid",
		Status:     models.ExecutionStatusRunning,
		StartedAt:  time.Now(),
	}

	result := storagemodels.ExecutionDomainToModel(domainExec)
	require.NotNil(t, result)

	// Invalid UUIDs should result in zero UUID
	assert.Equal(t, uuid.Nil, result.ID)
	assert.Equal(t, uuid.Nil, result.WorkflowID)
}

// TestNodeExecutionModelToDomain_Nil tests conversion of nil node execution
func TestNodeExecutionModelToDomain_Nil(t *testing.T) {
	result := storagemodels.NodeExecutionModelToDomain(nil)
	assert.Nil(t, result)
}

// TestNodeExecutionModelToDomain_Complete tests complete node execution conversion
func TestNodeExecutionModelToDomain_Complete(t *testing.T) {
	neID := uuid.New()
	execID := uuid.New()
	nodeID := uuid.New()
	startedAt := time.Now().Add(-1 * time.Minute)
	completedAt := time.Now()

	storageNE := &storagemodels.NodeExecutionModel{
		ID:             neID,
		ExecutionID:    execID,
		NodeID:         nodeID,
		Status:         "completed",
		InputData:      storagemodels.JSONBMap{"input": "data"},
		OutputData:     storagemodels.JSONBMap{"output": "data"},
		Config:         storagemodels.JSONBMap{"config": "original"},
		ResolvedConfig: storagemodels.JSONBMap{"config": "resolved"},
		StartedAt:      &startedAt,
		CompletedAt:    &completedAt,
		RetryCount:     2,
		Error:          "",
	}

	result := storagemodels.NodeExecutionModelToDomain(storageNE)
	require.NotNil(t, result)

	assert.Equal(t, neID.String(), result.ID)
	assert.Equal(t, execID.String(), result.ExecutionID)
	assert.Equal(t, nodeID.String(), result.NodeID)
	assert.Equal(t, models.NodeExecutionStatusCompleted, result.Status)

	assert.Equal(t, "data", result.Input["input"])
	assert.Equal(t, "data", result.Output["output"])
	assert.Equal(t, "original", result.Config["config"])
	assert.Equal(t, "resolved", result.ResolvedConfig["config"])

	assert.Equal(t, startedAt, result.StartedAt)
	assert.NotNil(t, result.CompletedAt)
	assert.Equal(t, 2, result.RetryCount)
}

// TestNodeExecutionDomainToModel_Nil tests conversion of nil node execution
func TestNodeExecutionDomainToModel_Nil(t *testing.T) {
	result := storagemodels.NodeExecutionDomainToModel(nil)
	assert.Nil(t, result)
}

// TestNodeExecutionDomainToModel_Complete tests complete node execution conversion
func TestNodeExecutionDomainToModel_Complete(t *testing.T) {
	neID := uuid.New()
	execID := uuid.New()
	nodeID := uuid.New()
	startedAt := time.Now().Add(-30 * time.Second)
	completedAt := time.Now()

	domainNE := &models.NodeExecution{
		ID:             neID.String(),
		ExecutionID:    execID.String(),
		NodeID:         nodeID.String(),
		Status:         models.NodeExecutionStatusCompleted,
		Input:          map[string]any{"test": "input"},
		Output:         map[string]any{"test": "output"},
		Config:         map[string]any{"key": "value"},
		ResolvedConfig: map[string]any{"key": "resolved"},
		StartedAt:      startedAt,
		CompletedAt:    &completedAt,
		RetryCount:     1,
		Error:          "",
	}

	result := storagemodels.NodeExecutionDomainToModel(domainNE)
	require.NotNil(t, result)

	assert.Equal(t, neID, result.ID)
	assert.Equal(t, execID, result.ExecutionID)
	assert.Equal(t, nodeID, result.NodeID)
	assert.Equal(t, "completed", result.Status)

	assert.Equal(t, "input", result.InputData["test"])
	assert.Equal(t, "output", result.OutputData["test"])
	assert.Equal(t, "value", result.Config["key"])
	assert.Equal(t, "resolved", result.ResolvedConfig["key"])

	assert.NotNil(t, result.StartedAt)
	assert.NotNil(t, result.CompletedAt)
	assert.Equal(t, 1, result.RetryCount)
}

// TestNodeExecutionDomainToModel_GeneratesID tests ID generation for empty/invalid IDs
func TestNodeExecutionDomainToModel_GeneratesID(t *testing.T) {
	domainNE := &models.NodeExecution{
		ID:          "", // Empty ID
		ExecutionID: uuid.New().String(),
		NodeID:      uuid.New().String(),
		Status:      models.NodeExecutionStatusRunning,
		StartedAt:   time.Now(),
	}

	result := storagemodels.NodeExecutionDomainToModel(domainNE)
	require.NotNil(t, result)

	// Should generate a new UUID for empty ID
	assert.NotEqual(t, uuid.Nil, result.ID)
}

// TestNodeExecutionDomainToModel_InvalidID tests handling of invalid ID
func TestNodeExecutionDomainToModel_InvalidID(t *testing.T) {
	domainNE := &models.NodeExecution{
		ID:          "invalid-uuid",
		ExecutionID: uuid.New().String(),
		NodeID:      uuid.New().String(),
		Status:      models.NodeExecutionStatusRunning,
		StartedAt:   time.Now(),
	}

	result := storagemodels.NodeExecutionDomainToModel(domainNE)
	require.NotNil(t, result)

	// Should generate a new UUID for invalid ID
	assert.NotEqual(t, uuid.Nil, result.ID)
}

// TestNodeExecutionDomainToModel_ZeroTimestamps tests handling of zero timestamps
func TestNodeExecutionDomainToModel_ZeroTimestamps(t *testing.T) {
	domainNE := &models.NodeExecution{
		ID:          uuid.New().String(),
		ExecutionID: uuid.New().String(),
		NodeID:      uuid.New().String(),
		Status:      models.NodeExecutionStatusPending,
		StartedAt:   time.Time{}, // Zero time
		CompletedAt: nil,
	}

	result := storagemodels.NodeExecutionDomainToModel(domainNE)
	require.NotNil(t, result)

	// Zero timestamps should not be set
	assert.Nil(t, result.StartedAt)
	assert.Nil(t, result.CompletedAt)
}
