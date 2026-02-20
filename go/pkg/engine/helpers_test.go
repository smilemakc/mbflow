package engine

import (
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func TestFindNodeByID(t *testing.T) {
	nodes := []*models.Node{
		{ID: "node-1", Name: "Node 1"},
		{ID: "node-2", Name: "Node 2"},
		{ID: "node-3", Name: "Node 3"},
	}

	tests := []struct {
		name     string
		nodeID   string
		expected *models.Node
	}{
		{
			name:     "find existing node",
			nodeID:   "node-2",
			expected: nodes[1],
		},
		{
			name:     "node not found",
			nodeID:   "node-999",
			expected: nil,
		},
		{
			name:     "empty id",
			nodeID:   "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindNodeByID(nodes, tt.nodeID)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCollectIncomingEdges(t *testing.T) {
	edges := []*models.Edge{
		{ID: "e1", From: "node-1", To: "node-2"},
		{ID: "e2", From: "node-1", To: "node-3"},
		{ID: "e3", From: "node-2", To: "node-3"},
		{ID: "e4", From: "node-3", To: "node-4"},
	}

	tests := []struct {
		name         string
		targetNodeID string
		expectedLen  int
		expectedIDs  []string
	}{
		{
			name:         "node with multiple incoming edges",
			targetNodeID: "node-3",
			expectedLen:  2,
			expectedIDs:  []string{"e2", "e3"},
		},
		{
			name:         "node with single incoming edge",
			targetNodeID: "node-2",
			expectedLen:  1,
			expectedIDs:  []string{"e1"},
		},
		{
			name:         "node with no incoming edges",
			targetNodeID: "node-1",
			expectedLen:  0,
			expectedIDs:  []string{},
		},
		{
			name:         "non-existent node",
			targetNodeID: "node-999",
			expectedLen:  0,
			expectedIDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CollectIncomingEdges(edges, tt.targetNodeID)
			if len(result) != tt.expectedLen {
				t.Errorf("expected %d edges, got %d", tt.expectedLen, len(result))
			}

			for _, expectedID := range tt.expectedIDs {
				found := false
				for _, edge := range result {
					if edge.ID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected edge %s not found", expectedID)
				}
			}
		})
	}
}

func TestCollectOutgoingEdges(t *testing.T) {
	edges := []*models.Edge{
		{ID: "e1", From: "node-1", To: "node-2"},
		{ID: "e2", From: "node-1", To: "node-3"},
		{ID: "e3", From: "node-2", To: "node-3"},
	}

	tests := []struct {
		name         string
		sourceNodeID string
		expectedLen  int
		expectedIDs  []string
	}{
		{
			name:         "node with multiple outgoing edges",
			sourceNodeID: "node-1",
			expectedLen:  2,
			expectedIDs:  []string{"e1", "e2"},
		},
		{
			name:         "node with single outgoing edge",
			sourceNodeID: "node-2",
			expectedLen:  1,
			expectedIDs:  []string{"e3"},
		},
		{
			name:         "node with no outgoing edges",
			sourceNodeID: "node-3",
			expectedLen:  0,
			expectedIDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CollectOutgoingEdges(edges, tt.sourceNodeID)
			if len(result) != tt.expectedLen {
				t.Errorf("expected %d edges, got %d", tt.expectedLen, len(result))
			}

			for _, expectedID := range tt.expectedIDs {
				found := false
				for _, edge := range result {
					if edge.ID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected edge %s not found", expectedID)
				}
			}
		})
	}
}

func TestGetNodePriority(t *testing.T) {
	tests := []struct {
		name     string
		node     *models.Node
		expected int
	}{
		{
			name: "node with int priority",
			node: &models.Node{
				ID:       "node-1",
				Metadata: map[string]any{"priority": 10},
			},
			expected: 10,
		},
		{
			name: "node with float64 priority",
			node: &models.Node{
				ID:       "node-2",
				Metadata: map[string]any{"priority": 5.0},
			},
			expected: 5,
		},
		{
			name: "node with int64 priority",
			node: &models.Node{
				ID:       "node-3",
				Metadata: map[string]any{"priority": int64(7)},
			},
			expected: 7,
		},
		{
			name: "node without priority",
			node: &models.Node{
				ID:       "node-4",
				Metadata: map[string]any{},
			},
			expected: DefaultNodePriority,
		},
		{
			name: "node with nil metadata",
			node: &models.Node{
				ID:       "node-5",
				Metadata: nil,
			},
			expected: DefaultNodePriority,
		},
		{
			name: "node with invalid priority type",
			node: &models.Node{
				ID:       "node-6",
				Metadata: map[string]any{"priority": "high"},
			},
			expected: DefaultNodePriority,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNodePriority(tt.node)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetNodeTimeout(t *testing.T) {
	tests := []struct {
		name     string
		node     *models.Node
		expected int64
	}{
		{
			name: "node with int timeout",
			node: &models.Node{
				ID:     "node-1",
				Config: map[string]any{"timeout": 5000},
			},
			expected: 5000,
		},
		{
			name: "node with int64 timeout",
			node: &models.Node{
				ID:     "node-2",
				Config: map[string]any{"timeout": int64(10000)},
			},
			expected: 10000,
		},
		{
			name: "node with float64 timeout",
			node: &models.Node{
				ID:     "node-3",
				Config: map[string]any{"timeout": 3000.0},
			},
			expected: 3000,
		},
		{
			name: "node without timeout",
			node: &models.Node{
				ID:     "node-4",
				Config: map[string]any{},
			},
			expected: 0,
		},
		{
			name: "node with nil config",
			node: &models.Node{
				ID:     "node-5",
				Config: nil,
			},
			expected: 0,
		},
		{
			name: "node with invalid timeout type",
			node: &models.Node{
				ID:     "node-6",
				Config: map[string]any{"timeout": "5s"},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNodeTimeout(tt.node)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
