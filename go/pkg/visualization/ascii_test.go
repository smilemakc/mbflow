package visualization

import (
	"strings"
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func TestASCIIRenderer_Format(t *testing.T) {
	renderer := NewASCIIRenderer()
	if got := renderer.Format(); got != "ascii" {
		t.Errorf("Format() = %v, want ascii", got)
	}
}

func TestASCIIRenderer_Render(t *testing.T) {
	tests := []struct {
		name     string
		workflow *models.Workflow
		opts     *RenderOptions
		want     []string // Expected substrings in output
		wantErr  bool
	}{
		{
			name:     "nil workflow",
			workflow: nil,
			opts:     DefaultRenderOptions(),
			wantErr:  true,
		},
		{
			name: "simple linear workflow",
			workflow: &models.Workflow{
				Name: "Simple Workflow",
				Nodes: []*models.Node{
					{ID: "a", Name: "Node A", Type: "http"},
					{ID: "b", Name: "Node B", Type: "http"},
				},
				Edges: []*models.Edge{
					{ID: "e1", From: "a", To: "b"},
				},
			},
			opts: &RenderOptions{
				CompactMode: true,
				UseColor:    false,
			},
			want: []string{
				"Simple Workflow",
				"a (http)",
				"└── b (http)",
			},
		},
		{
			name: "workflow with branching",
			workflow: &models.Workflow{
				Name: "Branching Workflow",
				Nodes: []*models.Node{
					{ID: "root", Name: "Root", Type: "http"},
					{ID: "child1", Name: "Child 1", Type: "http"},
					{ID: "child2", Name: "Child 2", Type: "http"},
				},
				Edges: []*models.Edge{
					{ID: "e1", From: "root", To: "child1"},
					{ID: "e2", From: "root", To: "child2"},
				},
			},
			opts: &RenderOptions{
				CompactMode: true,
				UseColor:    false,
			},
			want: []string{
				"Branching Workflow",
				"root (http)",
				"├── child1 (http)",
				"└── child2 (http)",
			},
		},
		{
			name: "workflow with detailed mode",
			workflow: &models.Workflow{
				Name: "Detailed Workflow",
				Nodes: []*models.Node{
					{ID: "a", Name: "Node A", Type: "http", Config: map[string]any{"method": "GET", "url": "/api/test"}},
					{ID: "b", Name: "Node B", Type: "http", Config: map[string]any{"method": "POST", "url": "/api/create"}},
				},
				Edges: []*models.Edge{
					{ID: "e1", From: "a", To: "b"},
				},
			},
			opts: &RenderOptions{
				CompactMode: false,
				UseColor:    false,
				ShowConfig:  true,
			},
			want: []string{
				"Detailed Workflow",
				"[a] Node A (http)",
				"│ GET /api/test",
				"[b] Node B (http)",
				"│ POST /api/create",
			},
		},
		{
			name: "workflow with multiple roots",
			workflow: &models.Workflow{
				Name: "Multi-Root",
				Nodes: []*models.Node{
					{ID: "root1", Name: "Root 1", Type: "http"},
					{ID: "root2", Name: "Root 2", Type: "http"},
					{ID: "child", Name: "Child", Type: "http"},
				},
				Edges: []*models.Edge{
					{ID: "e1", From: "root1", To: "child"},
				},
			},
			opts: &RenderOptions{
				CompactMode: true,
				UseColor:    false,
			},
			want: []string{
				"Multi-Root",
				"root1 (http)",
				"root2 (http)",
			},
		},
		{
			name: "empty workflow",
			workflow: &models.Workflow{
				Name:  "Empty",
				Nodes: []*models.Node{},
				Edges: []*models.Edge{},
			},
			opts: &RenderOptions{
				CompactMode: true,
				UseColor:    false,
			},
			want: []string{
				"Empty",
			},
		},
		{
			name: "workflow with deep nesting",
			workflow: &models.Workflow{
				Name: "Deep Nesting",
				Nodes: []*models.Node{
					{ID: "a", Name: "Level 1", Type: "http"},
					{ID: "b", Name: "Level 2", Type: "http"},
					{ID: "c", Name: "Level 3", Type: "http"},
					{ID: "d", Name: "Level 4", Type: "http"},
				},
				Edges: []*models.Edge{
					{ID: "e1", From: "a", To: "b"},
					{ID: "e2", From: "b", To: "c"},
					{ID: "e3", From: "c", To: "d"},
				},
			},
			opts: &RenderOptions{
				CompactMode: true,
				UseColor:    false,
			},
			want: []string{
				"Deep Nesting",
				"a (http)",
				"└── b (http)",
				"    └── c (http)",
				"        └── d (http)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewASCIIRenderer()
			got, err := renderer.Render(tt.workflow, tt.opts)

			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Check for expected substrings
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("Render() output missing expected substring:\nwant: %s\ngot: %s", want, got)
				}
			}
		})
	}
}

func TestASCIIRenderer_NodeFormatting(t *testing.T) {
	renderer := NewASCIIRenderer()

	tests := []struct {
		name        string
		node        *models.Node
		compactMode bool
		want        string
	}{
		{
			name:        "compact mode",
			node:        &models.Node{ID: "test", Name: "Test Node", Type: "http"},
			compactMode: true,
			want:        "test (http)",
		},
		{
			name:        "detailed mode",
			node:        &models.Node{ID: "test", Name: "Test Node", Type: "http"},
			compactMode: false,
			want:        "[test] Test Node (http)",
		},
		{
			name:        "node without name",
			node:        &models.Node{ID: "test", Type: "http"},
			compactMode: false,
			want:        "[test] (http)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &RenderOptions{
				CompactMode: tt.compactMode,
				UseColor:    false,
			}

			got := renderer.formatNode(tt.node, opts)

			if !strings.Contains(got, tt.want) {
				t.Errorf("formatNode() = %v, want substring %v", got, tt.want)
			}
		})
	}
}

func TestASCIIRenderer_ConfigExtraction(t *testing.T) {
	renderer := NewASCIIRenderer()

	tests := []struct {
		name string
		node *models.Node
		want string
	}{
		{
			name: "http node",
			node: &models.Node{
				ID:   "http1",
				Type: "http",
				Config: map[string]any{
					"method": "POST",
					"url":    "https://api.example.com/users",
				},
			},
			want: "POST https://api.example.com/users",
		},
		{
			name: "llm node",
			node: &models.Node{
				ID:   "llm1",
				Type: "llm",
				Config: map[string]any{
					"provider": "openai",
					"model":    "gpt-4",
				},
			},
			want: "openai / gpt-4",
		},
		{
			name: "transform node",
			node: &models.Node{
				ID:   "transform1",
				Type: "transform",
				Config: map[string]any{
					"type": "expression",
				},
			},
			want: "expression",
		},
		{
			name: "node with no config",
			node: &models.Node{
				ID:     "empty",
				Type:   "custom",
				Config: map[string]any{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.extractNodeConfig(tt.node)

			if got != tt.want {
				t.Errorf("extractNodeConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestASCIIRenderer_FindRootNodes(t *testing.T) {
	renderer := NewASCIIRenderer()

	tests := []struct {
		name     string
		workflow *models.Workflow
		wantIDs  []string
	}{
		{
			name: "single root",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "root"},
					{ID: "child"},
				},
				Edges: []*models.Edge{
					{From: "root", To: "child"},
				},
			},
			wantIDs: []string{"root"},
		},
		{
			name: "multiple roots",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "root1"},
					{ID: "root2"},
					{ID: "child"},
				},
				Edges: []*models.Edge{
					{From: "root1", To: "child"},
				},
			},
			wantIDs: []string{"root1", "root2"},
		},
		{
			name: "no edges",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "node1"},
					{ID: "node2"},
				},
				Edges: []*models.Edge{},
			},
			wantIDs: []string{"node1", "node2"},
		},
		{
			name: "cycle (all nodes have incoming)",
			workflow: &models.Workflow{
				Nodes: []*models.Node{
					{ID: "a"},
					{ID: "b"},
				},
				Edges: []*models.Edge{
					{From: "a", To: "b"},
					{From: "b", To: "a"},
				},
			},
			wantIDs: []string{}, // No root nodes in a cycle
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roots := renderer.findRootNodes(tt.workflow)

			if len(roots) != len(tt.wantIDs) {
				t.Errorf("findRootNodes() returned %d roots, want %d", len(roots), len(tt.wantIDs))
				return
			}

			gotIDs := make(map[string]bool)
			for _, root := range roots {
				gotIDs[root.ID] = true
			}

			for _, wantID := range tt.wantIDs {
				if !gotIDs[wantID] {
					t.Errorf("findRootNodes() missing expected root: %s", wantID)
				}
			}
		})
	}
}

func TestASCIIRenderer_Colorize(t *testing.T) {
	renderer := NewASCIIRenderer()

	tests := []struct {
		name    string
		text    string
		color   string
		enabled bool
		want    string
	}{
		{
			name:    "color enabled",
			text:    "test",
			color:   colorRed,
			enabled: true,
			want:    colorRed + "test" + colorReset,
		},
		{
			name:    "color disabled",
			text:    "test",
			color:   colorRed,
			enabled: false,
			want:    "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderer.colorize(tt.text, tt.color, tt.enabled)

			if got != tt.want {
				t.Errorf("colorize() = %v, want %v", got, tt.want)
			}
		})
	}
}
