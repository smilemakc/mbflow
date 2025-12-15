package sdk

import (
	"testing"

	"github.com/google/uuid"
	pkgModels "github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowToStorageForCreate tests workflow conversion for create operation
func TestWorkflowToStorageForCreate(t *testing.T) {
	workflow := &pkgModels.Workflow{
		Name:        "Test Workflow",
		Description: "Test Description",
		Version:     1,
		Status:      pkgModels.WorkflowStatusDraft,
		Tags:        []string{"test", "helper"},
		Variables: map[string]interface{}{
			"var1": "value1",
			"var2": 42,
		},
		Nodes: []*pkgModels.Node{
			{
				ID:   "node1",
				Name: "HTTP Node",
				Type: "http",
				Config: map[string]interface{}{
					"url":    "https://api.example.com",
					"method": "GET",
				},
				Position: &pkgModels.Position{X: 100, Y: 100},
			},
			{
				ID:   "node2",
				Name: "Transform Node",
				Type: "transform",
				Config: map[string]interface{}{
					"type": "passthrough",
				},
				Position: &pkgModels.Position{X: 300, Y: 100},
			},
		},
		Edges: []*pkgModels.Edge{
			{
				ID:   "edge1",
				From: "node1",
				To:   "node2",
			},
		},
	}

	storageWorkflow, err := workflowToStorageForCreate(workflow)
	require.NoError(t, err)
	require.NotNil(t, storageWorkflow)

	// Verify ID was generated
	assert.NotEqual(t, uuid.Nil, storageWorkflow.ID)

	// Verify fields were copied correctly
	assert.Equal(t, "Test Workflow", storageWorkflow.Name)
	assert.Equal(t, "Test Description", storageWorkflow.Description)
	assert.Equal(t, 1, storageWorkflow.Version)
	assert.Equal(t, string(pkgModels.WorkflowStatusDraft), storageWorkflow.Status)

	// Verify variables
	assert.NotNil(t, storageWorkflow.Variables)
	assert.Equal(t, "value1", storageWorkflow.Variables["var1"])
	assert.Equal(t, 42, storageWorkflow.Variables["var2"])

	// Verify nodes
	assert.Len(t, storageWorkflow.Nodes, 2)
	assert.Equal(t, "node1", storageWorkflow.Nodes[0].NodeID)
	assert.Equal(t, "HTTP Node", storageWorkflow.Nodes[0].Name)
	assert.Equal(t, "node2", storageWorkflow.Nodes[1].NodeID)
	assert.Equal(t, "Transform Node", storageWorkflow.Nodes[1].Name)

	// Verify edges
	assert.Len(t, storageWorkflow.Edges, 1)
	assert.Equal(t, "edge1", storageWorkflow.Edges[0].EdgeID)
	assert.Equal(t, "node1", storageWorkflow.Edges[0].FromNodeID)
	assert.Equal(t, "node2", storageWorkflow.Edges[0].ToNodeID)

	// Verify tags are in metadata
	assert.NotNil(t, storageWorkflow.Metadata)
	tagsVal, ok := storageWorkflow.Metadata["tags"]
	assert.True(t, ok)
	tagsSlice, ok := tagsVal.([]string)
	assert.True(t, ok)
	assert.Equal(t, []string{"test", "helper"}, tagsSlice)
}

// TestWorkflowToStorageForUpdate_Success tests workflow conversion for update operation
func TestWorkflowToStorageForUpdate_Success(t *testing.T) {
	existingID := uuid.New()
	workflow := &pkgModels.Workflow{
		ID:          existingID.String(),
		Name:        "Updated Workflow",
		Description: "Updated Description",
		Version:     2,
		Status:      pkgModels.WorkflowStatusActive,
		Tags:        []string{"updated"},
		Variables: map[string]interface{}{
			"updated_var": "updated_value",
		},
		Nodes: []*pkgModels.Node{
			{
				ID:   "node1",
				Name: "Updated Node",
				Type: "http",
				Config: map[string]interface{}{
					"url": "https://api.updated.com",
				},
				Position: &pkgModels.Position{X: 200, Y: 200},
			},
		},
		Edges: []*pkgModels.Edge{},
	}

	storageWorkflow, err := workflowToStorageForUpdate(workflow)
	require.NoError(t, err)
	require.NotNil(t, storageWorkflow)

	// Verify ID was preserved
	assert.Equal(t, existingID, storageWorkflow.ID)

	// Verify fields were updated
	assert.Equal(t, "Updated Workflow", storageWorkflow.Name)
	assert.Equal(t, "Updated Description", storageWorkflow.Description)
	assert.Equal(t, 2, storageWorkflow.Version)
	assert.Equal(t, string(pkgModels.WorkflowStatusActive), storageWorkflow.Status)

	// Verify updated variables
	assert.NotNil(t, storageWorkflow.Variables)
	assert.Equal(t, "updated_value", storageWorkflow.Variables["updated_var"])

	// Verify nodes
	assert.Len(t, storageWorkflow.Nodes, 1)
	assert.Equal(t, "node1", storageWorkflow.Nodes[0].NodeID)
	assert.Equal(t, "Updated Node", storageWorkflow.Nodes[0].Name)

	// Verify no edges
	assert.Len(t, storageWorkflow.Edges, 0)

	// Verify tags in metadata
	assert.NotNil(t, storageWorkflow.Metadata)
	tagsVal, ok := storageWorkflow.Metadata["tags"]
	assert.True(t, ok)
	tagsSlice, ok := tagsVal.([]string)
	assert.True(t, ok)
	assert.Equal(t, []string{"updated"}, tagsSlice)
}

// TestWorkflowToStorageForUpdate_InvalidID tests that invalid ID returns error
func TestWorkflowToStorageForUpdate_InvalidID(t *testing.T) {
	workflow := &pkgModels.Workflow{
		ID:          "invalid-uuid",
		Name:        "Test Workflow",
		Description: "Test Description",
	}

	_, err := workflowToStorageForUpdate(workflow)
	assert.ErrorIs(t, err, pkgModels.ErrInvalidWorkflowID)
}

// TestWorkflowToStorageForUpdate_EmptyID tests that empty ID returns error
func TestWorkflowToStorageForUpdate_EmptyID(t *testing.T) {
	workflow := &pkgModels.Workflow{
		ID:          "",
		Name:        "Test Workflow",
		Description: "Test Description",
	}

	_, err := workflowToStorageForUpdate(workflow)
	assert.ErrorIs(t, err, pkgModels.ErrInvalidWorkflowID)
}

// TestWorkflowFromStorage tests converting storage model to domain model
func TestWorkflowFromStorage(t *testing.T) {
	// This test is covered by storage models tests, but we test it here for completeness
	// and to ensure the helper function works correctly

	workflowID := uuid.New()
	workflow := &pkgModels.Workflow{
		Name:        "Original Workflow",
		Description: "Original Description",
		Version:     1,
		Status:      pkgModels.WorkflowStatusDraft,
		Tags:        []string{"original"},
		Variables: map[string]interface{}{
			"var1": "value1",
		},
		Nodes: []*pkgModels.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "http",
				Config: map[string]interface{}{
					"url": "https://example.com",
				},
				Position: &pkgModels.Position{X: 100, Y: 100},
			},
		},
		Edges: []*pkgModels.Edge{},
	}

	// Convert to storage
	storageWorkflow, err := workflowToStorageForCreate(workflow)
	require.NoError(t, err)

	// Update the ID to use our known UUID for testing
	storageWorkflow.ID = workflowID

	// Convert back to domain
	domainWorkflow := workflowFromStorage(storageWorkflow)
	require.NotNil(t, domainWorkflow)

	// Verify round-trip conversion preserves data
	assert.Equal(t, workflowID.String(), domainWorkflow.ID)
	assert.Equal(t, "Original Workflow", domainWorkflow.Name)
	assert.Equal(t, "Original Description", domainWorkflow.Description)
	assert.Equal(t, 1, domainWorkflow.Version)
	assert.Equal(t, pkgModels.WorkflowStatusDraft, domainWorkflow.Status)
	assert.Equal(t, []string{"original"}, domainWorkflow.Tags)

	// Verify variables
	assert.NotNil(t, domainWorkflow.Variables)
	assert.Equal(t, "value1", domainWorkflow.Variables["var1"])

	// Verify nodes
	assert.Len(t, domainWorkflow.Nodes, 1)
	assert.Equal(t, "node1", domainWorkflow.Nodes[0].ID)
	assert.Equal(t, "Node 1", domainWorkflow.Nodes[0].Name)
	assert.Equal(t, "http", domainWorkflow.Nodes[0].Type)
	assert.NotNil(t, domainWorkflow.Nodes[0].Position)
	assert.Equal(t, 100.0, domainWorkflow.Nodes[0].Position.X)
	assert.Equal(t, 100.0, domainWorkflow.Nodes[0].Position.Y)

	// Verify edges
	assert.Len(t, domainWorkflow.Edges, 0)
}

// TestWorkflowFromStorage_ComplexWorkflow tests converting complex workflow
func TestWorkflowFromStorage_ComplexWorkflow(t *testing.T) {
	workflowID := uuid.New()
	workflow := &pkgModels.Workflow{
		Name:        "Complex Workflow",
		Description: "Multi-node parallel workflow",
		Version:     3,
		Status:      pkgModels.WorkflowStatusActive,
		Tags:        []string{"complex", "parallel", "production"},
		Variables: map[string]interface{}{
			"api_base":   "https://api.example.com",
			"timeout":    30,
			"retry_max":  3,
			"debug_mode": false,
		},
		Metadata: map[string]interface{}{
			"created_by": "test_user",
			"department": "engineering",
		},
		Nodes: []*pkgModels.Node{
			{
				ID:   "start",
				Name: "Start Node",
				Type: "http",
				Config: map[string]interface{}{
					"url":    "{{api_base}}/start",
					"method": "GET",
				},
				Position: &pkgModels.Position{X: 100, Y: 200},
			},
			{
				ID:   "parallel1",
				Name: "Parallel Branch 1",
				Type: "transform",
				Config: map[string]interface{}{
					"type":       "expression",
					"expression": "input.data * 2",
				},
				Position: &pkgModels.Position{X: 300, Y: 100},
			},
			{
				ID:   "parallel2",
				Name: "Parallel Branch 2",
				Type: "transform",
				Config: map[string]interface{}{
					"type":       "expression",
					"expression": "input.data + 10",
				},
				Position: &pkgModels.Position{X: 300, Y: 300},
			},
			{
				ID:   "merge",
				Name: "Merge Results",
				Type: "merge",
				Config: map[string]interface{}{
					"strategy": "all",
				},
				Position: &pkgModels.Position{X: 500, Y: 200},
			},
		},
		Edges: []*pkgModels.Edge{
			{
				ID:   "edge1",
				From: "start",
				To:   "parallel1",
			},
			{
				ID:   "edge2",
				From: "start",
				To:   "parallel2",
			},
			{
				ID:   "edge3",
				From: "parallel1",
				To:   "merge",
			},
			{
				ID:   "edge4",
				From: "parallel2",
				To:   "merge",
			},
		},
	}

	// Convert to storage
	storageWorkflow, err := workflowToStorageForCreate(workflow)
	require.NoError(t, err)
	storageWorkflow.ID = workflowID

	// Convert back to domain
	domainWorkflow := workflowFromStorage(storageWorkflow)
	require.NotNil(t, domainWorkflow)

	// Verify all fields
	assert.Equal(t, workflowID.String(), domainWorkflow.ID)
	assert.Equal(t, "Complex Workflow", domainWorkflow.Name)
	assert.Equal(t, "Multi-node parallel workflow", domainWorkflow.Description)
	assert.Equal(t, 3, domainWorkflow.Version)
	assert.Equal(t, pkgModels.WorkflowStatusActive, domainWorkflow.Status)
	assert.Equal(t, []string{"complex", "parallel", "production"}, domainWorkflow.Tags)

	// Verify variables (JSONBMap preserves types)
	assert.Equal(t, "https://api.example.com", domainWorkflow.Variables["api_base"])
	assert.Equal(t, 30, domainWorkflow.Variables["timeout"])
	assert.Equal(t, 3, domainWorkflow.Variables["retry_max"])
	assert.Equal(t, false, domainWorkflow.Variables["debug_mode"])

	// Verify metadata
	assert.Equal(t, "test_user", domainWorkflow.Metadata["created_by"])
	assert.Equal(t, "engineering", domainWorkflow.Metadata["department"])

	// Verify all nodes preserved
	assert.Len(t, domainWorkflow.Nodes, 4)
	nodeIDs := make(map[string]bool)
	for _, node := range domainWorkflow.Nodes {
		nodeIDs[node.ID] = true
	}
	assert.True(t, nodeIDs["start"])
	assert.True(t, nodeIDs["parallel1"])
	assert.True(t, nodeIDs["parallel2"])
	assert.True(t, nodeIDs["merge"])

	// Verify all edges preserved
	assert.Len(t, domainWorkflow.Edges, 4)
	edgeIDs := make(map[string]bool)
	for _, edge := range domainWorkflow.Edges {
		edgeIDs[edge.ID] = true
	}
	assert.True(t, edgeIDs["edge1"])
	assert.True(t, edgeIDs["edge2"])
	assert.True(t, edgeIDs["edge3"])
	assert.True(t, edgeIDs["edge4"])

	// Verify edge connections
	edgeMap := make(map[string]*pkgModels.Edge)
	for _, edge := range domainWorkflow.Edges {
		edgeMap[edge.ID] = edge
	}
	assert.Equal(t, "start", edgeMap["edge1"].From)
	assert.Equal(t, "parallel1", edgeMap["edge1"].To)
	assert.Equal(t, "start", edgeMap["edge2"].From)
	assert.Equal(t, "parallel2", edgeMap["edge2"].To)
	assert.Equal(t, "parallel1", edgeMap["edge3"].From)
	assert.Equal(t, "merge", edgeMap["edge3"].To)
	assert.Equal(t, "parallel2", edgeMap["edge4"].From)
	assert.Equal(t, "merge", edgeMap["edge4"].To)
}
