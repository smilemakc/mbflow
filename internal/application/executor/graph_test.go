package executor

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

// testGraphNode is a simple test node implementation
type testGraphNode struct {
	id       uuid.UUID
	name     string
	nodeType domain.NodeType
}

func (n *testGraphNode) ID() uuid.UUID                  { return n.id }
func (n *testGraphNode) Name() string                   { return n.name }
func (n *testGraphNode) Type() domain.NodeType          { return n.nodeType }
func (n *testGraphNode) Config() map[string]any         { return nil }
func (n *testGraphNode) IOSchema() *domain.NodeIOSchema { return nil }
func (n *testGraphNode) InputBindingConfig() *domain.InputBindingConfig {
	return &domain.InputBindingConfig{
		AutoBind:          true,
		Mappings:          make(map[string]string),
		CollisionStrategy: domain.CollisionStrategyNamespaceByParent,
	}
}

// Helper to create a simple node
func createSimpleNode(name string, nodeType domain.NodeType) domain.Node {
	return &testGraphNode{
		id:       uuid.New(),
		name:     name,
		nodeType: nodeType,
	}
}

// testGraphEdge2 is a test edge implementation for graph_test.go
// (Note: testGraphEdge is already defined in variable_binder_test.go with different fields)
type testGraphEdge2 struct {
	id         uuid.UUID
	fromNodeID uuid.UUID
	toNodeID   uuid.UUID
	edgeType   domain.EdgeType
	config     map[string]any
}

func (e *testGraphEdge2) ID() uuid.UUID          { return e.id }
func (e *testGraphEdge2) FromNodeID() uuid.UUID  { return e.fromNodeID }
func (e *testGraphEdge2) ToNodeID() uuid.UUID    { return e.toNodeID }
func (e *testGraphEdge2) Type() domain.EdgeType  { return e.edgeType }
func (e *testGraphEdge2) Config() map[string]any { return e.config }

// Helper to create an edge
func createSimpleEdge(fromID, toID uuid.UUID, edgeType domain.EdgeType, config map[string]any) domain.Edge {
	return &testGraphEdge2{
		id:         uuid.New(),
		fromNodeID: fromID,
		toNodeID:   toID,
		edgeType:   edgeType,
		config:     config,
	}
}

// testGraphWorkflow is a simple test workflow implementation
type testGraphWorkflow struct {
	id    uuid.UUID
	name  string
	nodes []domain.Node
	edges []domain.Edge
}

func (w *testGraphWorkflow) ID() uuid.UUID                                { return w.id }
func (w *testGraphWorkflow) Name() string                                 { return w.name }
func (w *testGraphWorkflow) Version() string                              { return "1.0" }
func (w *testGraphWorkflow) Description() string                          { return "Test" }
func (w *testGraphWorkflow) Spec() map[string]any                         { return nil }
func (w *testGraphWorkflow) CreatedAt() time.Time                         { return time.Time{} }
func (w *testGraphWorkflow) UpdatedAt() time.Time                         { return time.Time{} }
func (w *testGraphWorkflow) GetAllNodes() []domain.Node                   { return w.nodes }
func (w *testGraphWorkflow) GetAllEdges() []domain.Edge                   { return w.edges }
func (w *testGraphWorkflow) GetAllTriggers() []domain.Trigger             { return nil }
func (w *testGraphWorkflow) GetNode(uuid.UUID) (domain.Node, error)       { return nil, nil }
func (w *testGraphWorkflow) GetEdge(uuid.UUID) (domain.Edge, error)       { return nil, nil }
func (w *testGraphWorkflow) GetTrigger(uuid.UUID) (domain.Trigger, error) { return nil, nil }
func (w *testGraphWorkflow) RemoveNode(uuid.UUID) error                   { return nil }
func (w *testGraphWorkflow) RemoveEdge(uuid.UUID) error                   { return nil }
func (w *testGraphWorkflow) RemoveTrigger(uuid.UUID) error                { return nil }
func (w *testGraphWorkflow) UseNode(domain.Node) error {
	return nil
}
func (w *testGraphWorkflow) AddNode(domain.NodeType, string, map[string]any) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (w *testGraphWorkflow) AddEdge(uuid.UUID, uuid.UUID, domain.EdgeType, map[string]any) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (w *testGraphWorkflow) AddTrigger(domain.TriggerType, map[string]any) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (w *testGraphWorkflow) Validate() error                      { return nil }
func (w *testGraphWorkflow) GetUncommittedEvents() []domain.Event { return nil }
func (w *testGraphWorkflow) MarkEventsAsCommitted()               {}
func (w *testGraphWorkflow) ApplyEvent(domain.Event) error        { return nil }

// Helper function to create a test workflow with nodes and edges
func createTestWorkflowForGraph(nodes []domain.Node, edges []domain.Edge) domain.Workflow {
	return &testGraphWorkflow{
		id:    uuid.New(),
		name:  "TestWorkflow",
		nodes: nodes,
		edges: edges,
	}
}

// TestGetNodeByName tests the GetNodeByName method
func TestGetNodeByName(t *testing.T) {
	t.Run("Found", func(t *testing.T) {
		// Create a valid connected workflow: node1 -> node2 -> node3
		node1 := createSimpleNode("node1", domain.NodeTypeStart)
		node2 := createSimpleNode("node2", domain.NodeTypeTransform)
		node3 := createSimpleNode("node3", domain.NodeTypeEnd)

		edge1 := createSimpleEdge(node1.ID(), node2.ID(), domain.EdgeTypeDirect, nil)
		edge2 := createSimpleEdge(node2.ID(), node3.ID(), domain.EdgeTypeDirect, nil)

		workflow := createTestWorkflowForGraph([]domain.Node{node1, node2, node3}, []domain.Edge{edge1, edge2})

		graph, err := NewWorkflowGraph(workflow)
		if err != nil {
			t.Fatalf("Failed to create graph: %v", err)
		}

		// Test finding existing node
		found, err := graph.GetNodeByName("node2")
		if err != nil {
			t.Fatalf("Expected to find node2, got error: %v", err)
		}
		if found.Name() != "node2" {
			t.Errorf("Expected node name 'node2', got '%s'", found.Name())
		}
		if found.Type() != domain.NodeTypeTransform {
			t.Errorf("Expected node type transform, got %s", found.Type())
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		// Create a valid single-node workflow
		node1 := createSimpleNode("node1", domain.NodeTypeStart)
		node2 := createSimpleNode("node2", domain.NodeTypeEnd)
		edge := createSimpleEdge(node1.ID(), node2.ID(), domain.EdgeTypeDirect, nil)

		workflow := createTestWorkflowForGraph([]domain.Node{node1, node2}, []domain.Edge{edge})

		graph, err := NewWorkflowGraph(workflow)
		if err != nil {
			t.Fatalf("Failed to create graph: %v", err)
		}

		// Test finding non-existent node
		_, err = graph.GetNodeByName("nonexistent")
		if err == nil {
			t.Error("Expected error when finding non-existent node, got nil")
		}
	})
}

// TestIsAncestor tests the IsAncestor method
func TestIsAncestor(t *testing.T) {
	t.Run("DirectParent", func(t *testing.T) {
		// Create workflow: A -> B
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeEnd)
		edge := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, nil)

		workflow := createTestWorkflowForGraph([]domain.Node{nodeA, nodeB}, []domain.Edge{edge})
		graph, _ := NewWorkflowGraph(workflow)

		// A is ancestor of B
		if !graph.IsAncestor(nodeA.ID(), nodeB.ID()) {
			t.Error("Expected A to be ancestor of B")
		}

		// B is not ancestor of A
		if graph.IsAncestor(nodeB.ID(), nodeA.ID()) {
			t.Error("Expected B not to be ancestor of A")
		}
	})

	t.Run("IndirectAncestor", func(t *testing.T) {
		// Create workflow: A -> B -> C -> D
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeTransform)
		nodeC := createSimpleNode("C", domain.NodeTypeTransform)
		nodeD := createSimpleNode("D", domain.NodeTypeEnd)

		edge1 := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, nil)
		edge2 := createSimpleEdge(nodeB.ID(), nodeC.ID(), domain.EdgeTypeDirect, nil)
		edge3 := createSimpleEdge(nodeC.ID(), nodeD.ID(), domain.EdgeTypeDirect, nil)

		workflow := createTestWorkflowForGraph(
			[]domain.Node{nodeA, nodeB, nodeC, nodeD},
			[]domain.Edge{edge1, edge2, edge3},
		)
		graph, _ := NewWorkflowGraph(workflow)

		// A is ancestor of D (through B and C)
		if !graph.IsAncestor(nodeA.ID(), nodeD.ID()) {
			t.Error("Expected A to be ancestor of D")
		}

		// B is ancestor of D (through C)
		if !graph.IsAncestor(nodeB.ID(), nodeD.ID()) {
			t.Error("Expected B to be ancestor of D")
		}

		// D is not ancestor of A
		if graph.IsAncestor(nodeD.ID(), nodeA.ID()) {
			t.Error("Expected D not to be ancestor of A")
		}
	})

	t.Run("NotAncestor_Parallel", func(t *testing.T) {
		// Create workflow: A -> B, A -> C (B and C are parallel)
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeTransform)
		nodeC := createSimpleNode("C", domain.NodeTypeTransform)

		edge1 := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, nil)
		edge2 := createSimpleEdge(nodeA.ID(), nodeC.ID(), domain.EdgeTypeDirect, nil)

		workflow := createTestWorkflowForGraph(
			[]domain.Node{nodeA, nodeB, nodeC},
			[]domain.Edge{edge1, edge2},
		)
		graph, _ := NewWorkflowGraph(workflow)

		// B is not ancestor of C (they are parallel)
		if graph.IsAncestor(nodeB.ID(), nodeC.ID()) {
			t.Error("Expected B not to be ancestor of C (parallel branches)")
		}

		// C is not ancestor of B
		if graph.IsAncestor(nodeC.ID(), nodeB.ID()) {
			t.Error("Expected C not to be ancestor of B (parallel branches)")
		}
	})

	t.Run("SelfReference", func(t *testing.T) {
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		workflow := createTestWorkflowForGraph([]domain.Node{nodeA}, nil)
		graph, _ := NewWorkflowGraph(workflow)

		// A is not ancestor of itself
		if graph.IsAncestor(nodeA.ID(), nodeA.ID()) {
			t.Error("Expected node not to be ancestor of itself")
		}
	})
}

// TestValidateEdgeDataSources tests the ValidateEdgeDataSources method
func TestValidateEdgeDataSources(t *testing.T) {
	t.Run("ValidAncestor", func(t *testing.T) {
		// Create workflow: A -> B -> C
		// Edge B->C includes A in include_outputs_from
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeTransform)
		nodeC := createSimpleNode("C", domain.NodeTypeEnd)

		edge1 := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, nil)
		edge2 := createSimpleEdge(nodeB.ID(), nodeC.ID(), domain.EdgeTypeDirect, map[string]any{
			"include_outputs_from": []string{"A"},
		})

		workflow := createTestWorkflowForGraph(
			[]domain.Node{nodeA, nodeB, nodeC},
			[]domain.Edge{edge1, edge2},
		)
		graph, _ := NewWorkflowGraph(workflow)

		// Should not error - A is ancestor of C
		err := graph.ValidateEdgeDataSources(edge2)
		if err != nil {
			t.Errorf("Expected no error for valid ancestor reference, got: %v", err)
		}
	})

	t.Run("NodeNotFound", func(t *testing.T) {
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeEnd)

		edge := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, map[string]any{
			"include_outputs_from": []string{"NonExistent"},
		})

		workflow := createTestWorkflowForGraph([]domain.Node{nodeA, nodeB}, []domain.Edge{edge})

		// Creating graph should fail validation due to non-existent node reference
		_, err := NewWorkflowGraph(workflow)
		if err == nil {
			t.Error("Expected error for non-existent node, got nil")
		}
	})

	t.Run("ForwardReference", func(t *testing.T) {
		// Create workflow: A -> B -> C
		// Try to make edge A->B include C (forward reference)
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeTransform)
		nodeC := createSimpleNode("C", domain.NodeTypeEnd)

		edge1 := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, map[string]any{
			"include_outputs_from": []string{"C"}, // C comes after B, invalid!
		})
		edge2 := createSimpleEdge(nodeB.ID(), nodeC.ID(), domain.EdgeTypeDirect, nil)

		workflow := createTestWorkflowForGraph(
			[]domain.Node{nodeA, nodeB, nodeC},
			[]domain.Edge{edge1, edge2},
		)

		// Creating graph should fail validation due to forward reference
		_, err := NewWorkflowGraph(workflow)
		if err == nil {
			t.Error("Expected error for forward reference, got nil")
		}
	})

	t.Run("SelfReference", func(t *testing.T) {
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeEnd)

		edge := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, map[string]any{
			"include_outputs_from": []string{"B"}, // B references itself
		})

		workflow := createTestWorkflowForGraph([]domain.Node{nodeA, nodeB}, []domain.Edge{edge})

		// Creating graph should fail validation due to self-reference
		_, err := NewWorkflowGraph(workflow)
		if err == nil {
			t.Error("Expected error for self-reference, got nil")
		}
	})

	t.Run("NoIncludeOutputsFrom", func(t *testing.T) {
		// Edge without include_outputs_from should pass validation
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeEnd)

		edge := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, nil)

		workflow := createTestWorkflowForGraph([]domain.Node{nodeA, nodeB}, []domain.Edge{edge})
		graph, _ := NewWorkflowGraph(workflow)

		err := graph.ValidateEdgeDataSources(edge)
		if err != nil {
			t.Errorf("Expected no error for edge without include_outputs_from, got: %v", err)
		}
	})

	t.Run("InvalidType_NotArray", func(t *testing.T) {
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeEnd)

		edge := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, map[string]any{
			"include_outputs_from": "not_an_array", // Should be []string
		})

		workflow := createTestWorkflowForGraph([]domain.Node{nodeA, nodeB}, []domain.Edge{edge})
		graph, _ := NewWorkflowGraph(workflow)

		err := graph.ValidateEdgeDataSources(edge)
		if err == nil {
			t.Error("Expected error for invalid type, got nil")
		}
	})

	t.Run("MultipleValidSources", func(t *testing.T) {
		// Create workflow: A -> B -> C -> D
		// Edge C->D includes both A and B
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeTransform)
		nodeC := createSimpleNode("C", domain.NodeTypeTransform)
		nodeD := createSimpleNode("D", domain.NodeTypeEnd)

		edge1 := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, nil)
		edge2 := createSimpleEdge(nodeB.ID(), nodeC.ID(), domain.EdgeTypeDirect, nil)
		edge3 := createSimpleEdge(nodeC.ID(), nodeD.ID(), domain.EdgeTypeDirect, map[string]any{
			"include_outputs_from": []string{"A", "B"},
		})

		workflow := createTestWorkflowForGraph(
			[]domain.Node{nodeA, nodeB, nodeC, nodeD},
			[]domain.Edge{edge1, edge2, edge3},
		)
		graph, _ := NewWorkflowGraph(workflow)

		err := graph.ValidateEdgeDataSources(edge3)
		if err != nil {
			t.Errorf("Expected no error for multiple valid sources, got: %v", err)
		}
	})
}

// TestWorkflowValidation_WithEdgeDataSources tests that workflow validation calls edge validation
func TestWorkflowValidation_WithEdgeDataSources(t *testing.T) {
	t.Run("ValidWorkflow", func(t *testing.T) {
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeTransform)
		nodeC := createSimpleNode("C", domain.NodeTypeEnd)

		edge1 := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, nil)
		edge2 := createSimpleEdge(nodeB.ID(), nodeC.ID(), domain.EdgeTypeDirect, map[string]any{
			"include_outputs_from": []string{"A"},
		})

		workflow := createTestWorkflowForGraph(
			[]domain.Node{nodeA, nodeB, nodeC},
			[]domain.Edge{edge1, edge2},
		)

		// Graph creation calls Validate()
		_, err := NewWorkflowGraph(workflow)
		if err != nil {
			t.Errorf("Expected valid workflow to pass validation, got: %v", err)
		}
	})

	t.Run("InvalidWorkflow_ForwardReference", func(t *testing.T) {
		nodeA := createSimpleNode("A", domain.NodeTypeStart)
		nodeB := createSimpleNode("B", domain.NodeTypeTransform)
		nodeC := createSimpleNode("C", domain.NodeTypeEnd)

		edge1 := createSimpleEdge(nodeA.ID(), nodeB.ID(), domain.EdgeTypeDirect, map[string]any{
			"include_outputs_from": []string{"C"}, // Forward reference!
		})
		edge2 := createSimpleEdge(nodeB.ID(), nodeC.ID(), domain.EdgeTypeDirect, nil)

		workflow := createTestWorkflowForGraph(
			[]domain.Node{nodeA, nodeB, nodeC},
			[]domain.Edge{edge1, edge2},
		)

		// Graph creation should fail validation
		_, err := NewWorkflowGraph(workflow)
		if err == nil {
			t.Error("Expected workflow validation to fail for forward reference, got nil")
		}
	})
}
