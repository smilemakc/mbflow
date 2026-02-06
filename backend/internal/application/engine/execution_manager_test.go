package engine

import (
	"testing"

	pkgengine "github.com/smilemakc/mbflow/pkg/engine"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
)

// ==================== MergeVariables Tests ====================

func TestMergeVariables(t *testing.T) {
	tests := []struct {
		name          string
		workflowVars  map[string]interface{}
		executionVars map[string]interface{}
		expected      map[string]interface{}
	}{
		{
			name:          "both empty",
			workflowVars:  map[string]interface{}{},
			executionVars: map[string]interface{}{},
			expected:      map[string]interface{}{},
		},
		{
			name:          "only workflow vars",
			workflowVars:  map[string]interface{}{"key1": "value1", "key2": 42},
			executionVars: map[string]interface{}{},
			expected:      map[string]interface{}{"key1": "value1", "key2": 42},
		},
		{
			name:          "only execution vars",
			workflowVars:  map[string]interface{}{},
			executionVars: map[string]interface{}{"key3": "value3"},
			expected:      map[string]interface{}{"key3": "value3"},
		},
		{
			name:          "no overlap",
			workflowVars:  map[string]interface{}{"key1": "value1"},
			executionVars: map[string]interface{}{"key2": "value2"},
			expected:      map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			name:          "execution vars override workflow vars",
			workflowVars:  map[string]interface{}{"key1": "workflow", "key2": "keep"},
			executionVars: map[string]interface{}{"key1": "execution", "key3": "new"},
			expected:      map[string]interface{}{"key1": "execution", "key2": "keep", "key3": "new"},
		},
		{
			name:          "nil workflow vars",
			workflowVars:  nil,
			executionVars: map[string]interface{}{"key1": "value1"},
			expected:      map[string]interface{}{"key1": "value1"},
		},
		{
			name:          "nil execution vars",
			workflowVars:  map[string]interface{}{"key1": "value1"},
			executionVars: nil,
			expected:      map[string]interface{}{"key1": "value1"},
		},
		{
			name:          "both nil",
			workflowVars:  nil,
			executionVars: nil,
			expected:      map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pkgengine.MergeVariables(tt.workflowVars, tt.executionVars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ==================== FindLeafNodes Tests ====================

func TestFindLeafNodes(t *testing.T) {
	tests := []struct {
		name     string
		workflow *models.Workflow
		expected []string // node IDs
	}{
		{
			name: "single node (is leaf)",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "node1", Name: "Node 1"},
				},
				Edges: []*models.Edge{},
			},
			expected: []string{"node1"},
		},
		{
			name: "linear chain - last node is leaf",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "node1", Name: "Node 1"},
					{ID: "node2", Name: "Node 2"},
					{ID: "node3", Name: "Node 3"},
				},
				Edges: []*models.Edge{
					{From: "node1", To: "node2"},
					{From: "node2", To: "node3"},
				},
			},
			expected: []string{"node3"},
		},
		{
			name: "parallel branches - multiple leaves",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "node1", Name: "Root"},
					{ID: "node2", Name: "Branch A"},
					{ID: "node3", Name: "Branch B"},
				},
				Edges: []*models.Edge{
					{From: "node1", To: "node2"},
					{From: "node1", To: "node3"},
				},
			},
			expected: []string{"node2", "node3"},
		},
		{
			name: "merge pattern - single leaf",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "node1", Name: "Node 1"},
					{ID: "node2", Name: "Node 2"},
					{ID: "node3", Name: "Merge"},
				},
				Edges: []*models.Edge{
					{From: "node1", To: "node3"},
					{From: "node2", To: "node3"},
				},
			},
			expected: []string{"node3"},
		},
		{
			name: "complex DAG - multiple leaves",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "n1", Name: "N1"},
					{ID: "n2", Name: "N2"},
					{ID: "n3", Name: "N3"},
					{ID: "n4", Name: "N4"},
					{ID: "n5", Name: "N5"},
				},
				Edges: []*models.Edge{
					{From: "n1", To: "n2"},
					{From: "n1", To: "n3"},
					{From: "n2", To: "n4"},
					// n3 is leaf, n4 is leaf, n5 is leaf (isolated)
				},
			},
			expected: []string{"n3", "n4", "n5"},
		},
		{
			name: "empty workflow",
			workflow: &models.Workflow{
				Nodes: []*models.Node{},
				Edges: []*models.Edge{},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaves := pkgengine.FindLeafNodes(tt.workflow)

			// Extract IDs for comparison
			leafIDs := make([]string, len(leaves))
			for i, leaf := range leaves {
				leafIDs[i] = leaf.ID
			}

			// Sort for consistent comparison
			assert.ElementsMatch(t, tt.expected, leafIDs)
		})
	}
}

// ==================== getFinalOutput Tests ====================

func TestExecutionManager_GetFinalOutput(t *testing.T) {
	em := &ExecutionManager{}

	tests := []struct {
		name      string
		execState *pkgengine.ExecutionState
		workflow  *models.Workflow
		expected  map[string]interface{}
	}{
		{
			name: "single leaf node",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "node1"},
					{ID: "node2"},
				},
				Edges: []*models.Edge{
					{From: "node1", To: "node2"},
				},
			},
			execState: func() *pkgengine.ExecutionState {
				state := pkgengine.NewExecutionState("exec-1", "workflow-1", &models.Workflow{}, nil, nil)
				state.SetNodeOutput("node2", map[string]interface{}{"result": "success", "count": 42})
				return state
			}(),
			expected: map[string]interface{}{"result": "success", "count": 42},
		},
		{
			name: "multiple leaf nodes",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "root"},
					{ID: "leaf1"},
					{ID: "leaf2"},
				},
				Edges: []*models.Edge{
					{From: "root", To: "leaf1"},
					{From: "root", To: "leaf2"},
				},
			},
			execState: func() *pkgengine.ExecutionState {
				state := pkgengine.NewExecutionState("exec-1", "workflow-1", &models.Workflow{}, nil, nil)
				state.SetNodeOutput("leaf1", map[string]interface{}{"data": "A"})
				state.SetNodeOutput("leaf2", map[string]interface{}{"data": "B"})
				return state
			}(),
			expected: map[string]interface{}{
				"leaf1": map[string]interface{}{"data": "A"},
				"leaf2": map[string]interface{}{"data": "B"},
			},
		},
		{
			name: "no leaf nodes",
			workflow: &models.Workflow{
				Nodes: []*models.Node{},
				Edges: []*models.Edge{},
			},
			execState: pkgengine.NewExecutionState("exec-1", "workflow-1", &models.Workflow{}, nil, nil),
			expected:  nil,
		},
		{
			name: "leaf node with non-map output",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "node1"},
				},
				Edges: []*models.Edge{},
			},
			execState: func() *pkgengine.ExecutionState {
				state := pkgengine.NewExecutionState("exec-1", "workflow-1", &models.Workflow{}, nil, nil)
				state.SetNodeOutput("node1", "string output")
				return state
			}(),
			expected: map[string]interface{}{
				"value": "string output",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set workflow in exec state for findLeafNodes
			tt.execState.Workflow = tt.workflow

			result := em.getFinalOutput(tt.execState)
			assert.Equal(t, tt.expected, result)
		})
	}
}
