package models

import (
	"testing"
	"time"
)

func TestWorkflow_Validate(t *testing.T) {
	tests := []struct {
		name     string
		workflow *Workflow
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid workflow",
			workflow: &Workflow{
				ID:   "wf-1",
				Name: "Test Workflow",
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			workflow: &Workflow{
				ID: "wf-1",
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
				},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "no nodes",
			workflow: &Workflow{
				ID:    "wf-1",
				Name:  "Test Workflow",
				Nodes: []*Node{},
			},
			wantErr: true,
			errMsg:  "at least one node is required",
		},
		{
			name: "duplicate node IDs",
			workflow: &Workflow{
				ID:   "wf-1",
				Name: "Test Workflow",
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
					{ID: "node-1", Name: "Node 2", Type: "http", Config: map[string]interface{}{}},
				},
			},
			wantErr: true,
			errMsg:  "duplicate node ID",
		},
		{
			name: "edge references non-existent source node",
			workflow: &Workflow{
				ID:   "wf-1",
				Name: "Test Workflow",
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
				},
				Edges: []*Edge{
					{ID: "edge-1", From: "non-existent", To: "node-1"},
				},
			},
			wantErr: true,
			errMsg:  "edge references non-existent source node",
		},
		{
			name: "edge references non-existent target node",
			workflow: &Workflow{
				ID:   "wf-1",
				Name: "Test Workflow",
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
				},
				Edges: []*Edge{
					{ID: "edge-1", From: "node-1", To: "non-existent"},
				},
			},
			wantErr: true,
			errMsg:  "edge references non-existent target node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.workflow.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNode_Validate(t *testing.T) {
	tests := []struct {
		name    string
		node    *Node
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid node",
			node: &Node{
				ID:     "node-1",
				Name:   "Test Node",
				Type:   "http",
				Config: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			node: &Node{
				Name:   "Test Node",
				Type:   "http",
				Config: map[string]interface{}{},
			},
			wantErr: true,
			errMsg:  "node ID is required",
		},
		{
			name: "missing name",
			node: &Node{
				ID:     "node-1",
				Type:   "http",
				Config: map[string]interface{}{},
			},
			wantErr: true,
			errMsg:  "node name is required",
		},
		{
			name: "missing type",
			node: &Node{
				ID:     "node-1",
				Name:   "Test Node",
				Config: map[string]interface{}{},
			},
			wantErr: true,
			errMsg:  "node type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEdge_Validate(t *testing.T) {
	tests := []struct {
		name    string
		edge    *Edge
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid edge",
			edge: &Edge{
				ID:   "edge-1",
				From: "node-1",
				To:   "node-2",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			edge: &Edge{
				From: "node-1",
				To:   "node-2",
			},
			wantErr: true,
			errMsg:  "edge ID is required",
		},
		{
			name: "missing source",
			edge: &Edge{
				ID: "edge-1",
				To: "node-2",
			},
			wantErr: true,
			errMsg:  "edge source is required",
		},
		{
			name: "missing target",
			edge: &Edge{
				ID:   "edge-1",
				From: "node-1",
			},
			wantErr: true,
			errMsg:  "edge target is required",
		},
		{
			name: "self-loop",
			edge: &Edge{
				ID:   "edge-1",
				From: "node-1",
				To:   "node-1",
			},
			wantErr: true,
			errMsg:  "self-loop edges are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.edge.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWorkflow_GetNode(t *testing.T) {
	workflow := &Workflow{
		Nodes: []*Node{
			{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
			{ID: "node-2", Name: "Node 2", Type: "http", Config: map[string]interface{}{}},
		},
	}

	tests := []struct {
		name    string
		nodeID  string
		wantErr bool
	}{
		{
			name:    "existing node",
			nodeID:  "node-1",
			wantErr: false,
		},
		{
			name:    "non-existent node",
			nodeID:  "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := workflow.GetNode(tt.nodeID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if err != ErrNodeNotFound {
					t.Errorf("expected ErrNodeNotFound, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if node == nil {
					t.Error("node is nil")
				}
				if node.ID != tt.nodeID {
					t.Errorf("expected node ID %s, got %s", tt.nodeID, node.ID)
				}
			}
		})
	}
}

func TestWorkflow_GetEdge(t *testing.T) {
	workflow := &Workflow{
		Edges: []*Edge{
			{ID: "edge-1", From: "node-1", To: "node-2"},
			{ID: "edge-2", From: "node-2", To: "node-3"},
		},
	}

	tests := []struct {
		name    string
		edgeID  string
		wantErr bool
	}{
		{
			name:    "existing edge",
			edgeID:  "edge-1",
			wantErr: false,
		},
		{
			name:    "non-existent edge",
			edgeID:  "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge, err := workflow.GetEdge(tt.edgeID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if err != ErrEdgeNotFound {
					t.Errorf("expected ErrEdgeNotFound, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if edge == nil {
					t.Error("edge is nil")
				}
				if edge.ID != tt.edgeID {
					t.Errorf("expected edge ID %s, got %s", tt.edgeID, edge.ID)
				}
			}
		})
	}
}

func TestWorkflow_AddNode(t *testing.T) {
	tests := []struct {
		name     string
		workflow *Workflow
		newNode  *Node
		wantErr  bool
		errMsg   string
	}{
		{
			name: "add valid node",
			workflow: &Workflow{
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
				},
			},
			newNode: &Node{
				ID:     "node-2",
				Name:   "Node 2",
				Type:   "http",
				Config: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "add node with duplicate ID",
			workflow: &Workflow{
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
				},
			},
			newNode: &Node{
				ID:     "node-1",
				Name:   "Node 2",
				Type:   "http",
				Config: map[string]interface{}{},
			},
			wantErr: true,
			errMsg:  "node ID already exists",
		},
		{
			name: "add invalid node",
			workflow: &Workflow{
				Nodes: []*Node{},
			},
			newNode: &Node{
				Name: "Invalid Node",
				Type: "http",
			},
			wantErr: true,
			errMsg:  "node ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialLen := len(tt.workflow.Nodes)
			err := tt.workflow.AddNode(tt.newNode)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(tt.workflow.Nodes) != initialLen+1 {
					t.Errorf("expected %d nodes, got %d", initialLen+1, len(tt.workflow.Nodes))
				}
			}
		})
	}
}

func TestWorkflow_AddEdge(t *testing.T) {
	tests := []struct {
		name     string
		workflow *Workflow
		newEdge  *Edge
		wantErr  bool
		errMsg   string
	}{
		{
			name: "add valid edge",
			workflow: &Workflow{
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
					{ID: "node-2", Name: "Node 2", Type: "http", Config: map[string]interface{}{}},
				},
				Edges: []*Edge{},
			},
			newEdge: &Edge{
				ID:   "edge-1",
				From: "node-1",
				To:   "node-2",
			},
			wantErr: false,
		},
		{
			name: "add edge with non-existent source",
			workflow: &Workflow{
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
				},
			},
			newEdge: &Edge{
				ID:   "edge-1",
				From: "non-existent",
				To:   "node-1",
			},
			wantErr: true,
			errMsg:  "source node does not exist",
		},
		{
			name: "add edge with duplicate ID",
			workflow: &Workflow{
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
					{ID: "node-2", Name: "Node 2", Type: "http", Config: map[string]interface{}{}},
				},
				Edges: []*Edge{
					{ID: "edge-1", From: "node-1", To: "node-2"},
				},
			},
			newEdge: &Edge{
				ID:   "edge-1",
				From: "node-1",
				To:   "node-2",
			},
			wantErr: true,
			errMsg:  "edge ID already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialLen := len(tt.workflow.Edges)
			err := tt.workflow.AddEdge(tt.newEdge)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(tt.workflow.Edges) != initialLen+1 {
					t.Errorf("expected %d edges, got %d", initialLen+1, len(tt.workflow.Edges))
				}
			}
		})
	}
}

func TestWorkflow_RemoveNode(t *testing.T) {
	tests := []struct {
		name          string
		workflow      *Workflow
		nodeID        string
		wantErr       bool
		expectedNodes int
		expectedEdges int
	}{
		{
			name: "remove existing node",
			workflow: &Workflow{
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
					{ID: "node-2", Name: "Node 2", Type: "http", Config: map[string]interface{}{}},
				},
				Edges: []*Edge{
					{ID: "edge-1", From: "node-1", To: "node-2"},
				},
			},
			nodeID:        "node-1",
			wantErr:       false,
			expectedNodes: 1,
			expectedEdges: 0, // Edge should be removed
		},
		{
			name: "remove non-existent node",
			workflow: &Workflow{
				Nodes: []*Node{
					{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{}},
				},
			},
			nodeID:  "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.workflow.RemoveNode(tt.nodeID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if err != ErrNodeNotFound {
					t.Errorf("expected ErrNodeNotFound, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(tt.workflow.Nodes) != tt.expectedNodes {
					t.Errorf("expected %d nodes, got %d", tt.expectedNodes, len(tt.workflow.Nodes))
				}
				if len(tt.workflow.Edges) != tt.expectedEdges {
					t.Errorf("expected %d edges, got %d", tt.expectedEdges, len(tt.workflow.Edges))
				}
			}
		})
	}
}

func TestWorkflow_RemoveEdge(t *testing.T) {
	tests := []struct {
		name     string
		workflow *Workflow
		edgeID   string
		wantErr  bool
	}{
		{
			name: "remove existing edge",
			workflow: &Workflow{
				Edges: []*Edge{
					{ID: "edge-1", From: "node-1", To: "node-2"},
					{ID: "edge-2", From: "node-2", To: "node-3"},
				},
			},
			edgeID:  "edge-1",
			wantErr: false,
		},
		{
			name: "remove non-existent edge",
			workflow: &Workflow{
				Edges: []*Edge{
					{ID: "edge-1", From: "node-1", To: "node-2"},
				},
			},
			edgeID:  "non-existent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialLen := len(tt.workflow.Edges)
			err := tt.workflow.RemoveEdge(tt.edgeID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if err != ErrEdgeNotFound {
					t.Errorf("expected ErrEdgeNotFound, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(tt.workflow.Edges) != initialLen-1 {
					t.Errorf("expected %d edges, got %d", initialLen-1, len(tt.workflow.Edges))
				}
			}
		})
	}
}

func TestWorkflow_Clone(t *testing.T) {
	original := &Workflow{
		ID:          "wf-1",
		Name:        "Original",
		Description: "Test workflow",
		Version:     1,
		Status:      WorkflowStatusActive,
		Tags:        []string{"test", "clone"},
		Nodes: []*Node{
			{ID: "node-1", Name: "Node 1", Type: "http", Config: map[string]interface{}{"key": "value"}},
		},
		Edges: []*Edge{
			{ID: "edge-1", From: "node-1", To: "node-2"},
		},
		Variables: map[string]interface{}{
			"var1": "value1",
		},
		Metadata: map[string]interface{}{
			"meta1": "metavalue1",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	clone, err := original.Clone()
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	if clone == nil {
		t.Fatal("Clone is nil")
	}

	// Verify basic fields
	if clone.ID != original.ID {
		t.Errorf("expected ID %s, got %s", original.ID, clone.ID)
	}
	if clone.Name != original.Name {
		t.Errorf("expected Name %s, got %s", original.Name, clone.Name)
	}
	if clone.Version != original.Version {
		t.Errorf("expected Version %d, got %d", original.Version, clone.Version)
	}

	// Verify nodes
	if len(clone.Nodes) != len(original.Nodes) {
		t.Errorf("expected %d nodes, got %d", len(original.Nodes), len(clone.Nodes))
	}

	// Verify edges
	if len(clone.Edges) != len(original.Edges) {
		t.Errorf("expected %d edges, got %d", len(original.Edges), len(clone.Edges))
	}

	// Modify clone and verify original is unchanged
	clone.Name = "Modified"
	if original.Name == "Modified" {
		t.Error("modifying clone affected original")
	}
}

func TestWorkflowStatus_Values(t *testing.T) {
	statuses := []WorkflowStatus{
		WorkflowStatusDraft,
		WorkflowStatusActive,
		WorkflowStatusInactive,
		WorkflowStatusArchived,
	}

	expectedValues := []string{"draft", "active", "inactive", "archived"}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("expected status %s, got %s", expectedValues[i], string(status))
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
