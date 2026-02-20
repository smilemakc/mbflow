package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowAPI_Create_StandaloneMode tests creating workflow in standalone mode
func TestWorkflowAPI_Create_StandaloneMode(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name:        "Test Workflow",
		Description: "Test workflow for Create",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "node2",
				Name: "Node 2",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "edge1",
				From: "node1",
				To:   "node2",
			},
		},
	}

	created, err := client.Workflows().Create(ctx, workflow)
	require.NoError(t, err)
	require.NotNil(t, created)

	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Test Workflow", created.Name)
	assert.Equal(t, "Test workflow for Create", created.Description)
	assert.False(t, created.CreatedAt.IsZero())
	assert.False(t, created.UpdatedAt.IsZero())
	assert.Len(t, created.Nodes, 2)
	assert.Len(t, created.Edges, 1)
}

// TestWorkflowAPI_Create_WithEdges tests creating workflow with edges
func TestWorkflowAPI_Create_WithEdges(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Workflow With Edges",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "node2",
				Name: "Node 2",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "edge1",
				From: "node1",
				To:   "node2",
			},
		},
	}

	created, err := client.Workflows().Create(ctx, workflow)
	require.NoError(t, err)

	assert.Len(t, created.Nodes, 2)
	assert.Len(t, created.Edges, 1)
	assert.Equal(t, "node1", created.Edges[0].From)
	assert.Equal(t, "node2", created.Edges[0].To)
}

// TestWorkflowAPI_Create_EmptyName tests that empty name is rejected
func TestWorkflowAPI_Create_EmptyName(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name:  "",
		Nodes: []*models.Node{},
	}

	_, err = client.Workflows().Create(ctx, workflow)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

// TestWorkflowAPI_Create_NoNodes tests creating workflow with no nodes
func TestWorkflowAPI_Create_NoNodes(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name:  "No Nodes Workflow",
		Nodes: []*models.Node{},
	}

	_, err = client.Workflows().Create(ctx, workflow)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

// TestWorkflowAPI_Create_WithCycle tests that cycle detection works
func TestWorkflowAPI_Create_WithCycle(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Cyclic Workflow",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "node2",
				Name: "Node 2",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{ID: "edge1", From: "node1", To: "node2"},
			{ID: "edge2", From: "node2", To: "node1"}, // Creates cycle
		},
	}

	_, err = client.Workflows().Create(ctx, workflow)
	assert.Error(t, err)
	// Should fail with validation error (could be DAG or basic validation)
	assert.True(t, err != nil)
}

// TestWorkflowAPI_Create_InvalidEdge tests that invalid edges are rejected
func TestWorkflowAPI_Create_InvalidEdge(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Invalid Edge Workflow",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "edge1",
				From: "node1",
				To:   "nonexistent", // Invalid node reference
			},
		},
	}

	_, err = client.Workflows().Create(ctx, workflow)
	assert.Error(t, err)
	// Should fail with validation error
	assert.True(t, err != nil)
}

// TestWorkflowAPI_Create_ClosedClient tests that closed client returns error
func TestWorkflowAPI_Create_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Test",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
	}

	_, err = client.Workflows().Create(ctx, workflow)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestWorkflowAPI_Create_WithVariables tests creating workflow with variables
func TestWorkflowAPI_Create_WithVariables(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Workflow With Variables",
		Variables: map[string]any{
			"api_key": "secret",
			"timeout": 30,
		},
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "node2",
				Name: "Node 2",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{ID: "edge1", From: "node1", To: "node2"},
		},
	}

	created, err := client.Workflows().Create(ctx, workflow)
	require.NoError(t, err)

	assert.NotEmpty(t, created.Variables)
	assert.Equal(t, "secret", created.Variables["api_key"])
	assert.Equal(t, 30, created.Variables["timeout"])
}

// TestWorkflowAPI_Create_WithTags tests creating workflow with tags
func TestWorkflowAPI_Create_WithTags(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Tagged Workflow",
		Tags: []string{"test", "automation"},
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "node2",
				Name: "Node 2",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{ID: "edge1", From: "node1", To: "node2"},
		},
	}

	created, err := client.Workflows().Create(ctx, workflow)
	require.NoError(t, err)

	assert.Equal(t, []string{"test", "automation"}, created.Tags)
}

// TestWorkflowAPI_Get_NotAvailableInStandalone tests that Get is not available in standalone
func TestWorkflowAPI_Get_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Workflows().Get(ctx, "some-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

// TestWorkflowAPI_Get_EmptyID tests that empty ID is rejected
func TestWorkflowAPI_Get_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Workflows().Get(ctx, "")
	assert.ErrorIs(t, err, models.ErrInvalidWorkflowID)
}

// TestWorkflowAPI_Get_ClosedClient tests that closed client returns error
func TestWorkflowAPI_Get_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Workflows().Get(ctx, "some-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestWorkflowAPI_List_NotAvailableInStandalone tests that List is not available in standalone
func TestWorkflowAPI_List_NotAvailableInStandalone(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Workflows().List(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

// TestWorkflowAPI_List_ClosedClient tests that closed client returns error
func TestWorkflowAPI_List_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Workflows().List(ctx, nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestWorkflowAPI_Update_EmptyID tests that empty ID is rejected
func TestWorkflowAPI_Update_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		ID:   "",
		Name: "Test",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
	}

	_, err = client.Workflows().Update(ctx, workflow)
	assert.ErrorIs(t, err, models.ErrInvalidWorkflowID)
}

// TestWorkflowAPI_Update_ValidationFails tests that validation runs on update
func TestWorkflowAPI_Update_ValidationFails(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		ID:    "test-id",
		Name:  "", // Invalid
		Nodes: []*models.Node{},
	}

	_, err = client.Workflows().Update(ctx, workflow)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

// TestWorkflowAPI_Update_ClosedClient tests that closed client returns error
func TestWorkflowAPI_Update_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		ID:   "test-id",
		Name: "Test",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
	}

	_, err = client.Workflows().Update(ctx, workflow)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestWorkflowAPI_Delete_EmptyID tests that empty ID is rejected
func TestWorkflowAPI_Delete_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Workflows().Delete(ctx, "")
	assert.ErrorIs(t, err, models.ErrInvalidWorkflowID)
}

// TestWorkflowAPI_Delete_ClosedClient tests that closed client returns error
func TestWorkflowAPI_Delete_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	err = client.Workflows().Delete(ctx, "some-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestWorkflowAPI_ValidateDAG_Success tests successful DAG validation
func TestWorkflowAPI_ValidateDAG_Success(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Valid DAG",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "node2",
				Name: "Node 2",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{From: "node1", To: "node2"},
		},
	}

	result, err := client.Workflows().ValidateDAG(ctx, workflow)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

// TestWorkflowAPI_ValidateDAG_Cycle tests that cycle is detected
func TestWorkflowAPI_ValidateDAG_Cycle(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Cyclic DAG",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "node2",
				Name: "Node 2",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{From: "node1", To: "node2"},
			{From: "node2", To: "node1"},
		},
	}

	result, err := client.Workflows().ValidateDAG(ctx, workflow)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0], "cycle detected")
}

// TestWorkflowAPI_ValidateDAG_InvalidEdge tests that invalid edge is detected
func TestWorkflowAPI_ValidateDAG_InvalidEdge(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Invalid Edge DAG",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{From: "node1", To: "nonexistent"},
		},
	}

	result, err := client.Workflows().ValidateDAG(ctx, workflow)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// TestWorkflowAPI_ValidateDAG_ClosedClient tests that closed client returns error
func TestWorkflowAPI_ValidateDAG_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Test",
		Nodes: []*models.Node{
			{
				ID:   "node1",
				Name: "Node 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
	}

	_, err = client.Workflows().ValidateDAG(ctx, workflow)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestWorkflowAPI_Create_ComplexWorkflow tests creating a complex workflow
func TestWorkflowAPI_Create_ComplexWorkflow(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name:        "Complex Workflow",
		Description: "Multi-node parallel workflow",
		Variables: map[string]any{
			"api_base": "https://api.example.com",
		},
		Tags: []string{"complex", "parallel"},
		Nodes: []*models.Node{
			{
				ID:   "start",
				Name: "Start Node",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "parallel1",
				Name: "Parallel 1",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "parallel2",
				Name: "Parallel 2",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
			{
				ID:   "merge",
				Name: "Merge Node",
				Type: "transform",
				Config: map[string]any{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{ID: "edge1", From: "start", To: "parallel1"},
			{ID: "edge2", From: "start", To: "parallel2"},
			{ID: "edge3", From: "parallel1", To: "merge"},
			{ID: "edge4", From: "parallel2", To: "merge"},
		},
	}

	created, err := client.Workflows().Create(ctx, workflow)
	require.NoError(t, err)

	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Complex Workflow", created.Name)
	assert.Len(t, created.Nodes, 4)
	assert.Len(t, created.Edges, 4)
	assert.Len(t, created.Tags, 2)
	assert.NotEmpty(t, created.Variables)
}

// TestWorkflowAPI_Create_Concurrent tests concurrent workflow creation
func TestWorkflowAPI_Create_Concurrent(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(idx int) {
			workflow := &models.Workflow{
				Name: "Concurrent Workflow",
				Nodes: []*models.Node{
					{
						ID:   "node1",
						Name: "Node 1",
						Type: "transform",
						Config: map[string]any{
							"type": "passthrough",
						},
					},
					{
						ID:   "node2",
						Name: "Node 2",
						Type: "transform",
						Config: map[string]any{
							"type": "passthrough",
						},
					},
				},
				Edges: []*models.Edge{
					{ID: "edge1", From: "node1", To: "node2"},
				},
			}

			_, err := client.Workflows().Create(ctx, workflow)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	timeout := time.After(5 * time.Second)
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent creates")
		}
	}
}
