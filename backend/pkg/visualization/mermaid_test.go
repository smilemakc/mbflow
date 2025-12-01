package visualization

import (
	"strings"
	"testing"

	"github.com/smilemakc/mbflow/pkg/models"
)

func TestMermaidRenderer_Format(t *testing.T) {
	renderer := NewMermaidRenderer()
	if got := renderer.Format(); got != "mermaid" {
		t.Errorf("Format() = %v, want mermaid", got)
	}
}

func TestMermaidRenderer_Render(t *testing.T) {
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
					{ID: "a", Name: "Node A", Type: "http", Config: map[string]interface{}{"method": "GET", "url": "/api/a"}},
					{ID: "b", Name: "Node B", Type: "http", Config: map[string]interface{}{"method": "POST", "url": "/api/b"}},
				},
				Edges: []*models.Edge{
					{ID: "e1", From: "a", To: "b"},
				},
			},
			opts: DefaultRenderOptions(),
			want: []string{
				"flowchart TB",
				`a["HTTP: GET: Node A`,
				`b["HTTP: POST: Node B`,
				"a --> b",
			},
		},
		{
			name: "workflow with different node types",
			workflow: &models.Workflow{
				Name: "Mixed Types",
				Nodes: []*models.Node{
					{ID: "http_node", Name: "HTTP Call", Type: "http"},
					{ID: "llm_node", Name: "LLM Process", Type: "llm"},
					{ID: "transform_node", Name: "Transform Data", Type: "transform"},
					{ID: "conditional_node", Name: "Check Condition", Type: "conditional"},
					{ID: "merge_node", Name: "Merge Results", Type: "merge"},
				},
				Edges: []*models.Edge{
					{ID: "e1", From: "http_node", To: "llm_node"},
					{ID: "e2", From: "llm_node", To: "transform_node"},
					{ID: "e3", From: "transform_node", To: "conditional_node"},
					{ID: "e4", From: "conditional_node", To: "merge_node"},
				},
			},
			opts: DefaultRenderOptions(),
			want: []string{
				"flowchart TB",
				`http_node["HTTP: HTTP Call"]`,                  // Rectangle
				`llm_node(["LLM: LLM Process"])`,                // Stadium
				`transform_node[/"Transform: Transform Data"/]`, // Trapezoid
				`conditional_node{"If: Check Condition"}`,       // Diamond
				`merge_node{{"Merge: Merge Results"}}`,          // Hexagon
			},
		},
		{
			name: "workflow with conditional edges",
			workflow: &models.Workflow{
				Name: "Conditional Flow",
				Nodes: []*models.Node{
					{ID: "check", Name: "Check Status", Type: "conditional"},
					{ID: "success", Name: "Success Path", Type: "http"},
					{ID: "failure", Name: "Failure Path", Type: "http"},
				},
				Edges: []*models.Edge{
					{ID: "e1", From: "check", To: "success", Condition: "status == 200"},
					{ID: "e2", From: "check", To: "failure", Condition: "status != 200"},
				},
			},
			opts: DefaultRenderOptions(),
			want: []string{
				`check -- "status == 200" --> success`,
				`check -- "status != 200" --> failure`,
			},
		},
		{
			name: "workflow without config display",
			workflow: &models.Workflow{
				Name: "No Config",
				Nodes: []*models.Node{
					{ID: "a", Name: "Node A", Type: "http", Config: map[string]interface{}{"url": "/api/test"}},
				},
				Edges: []*models.Edge{},
			},
			opts: &RenderOptions{
				ShowConfig:     false,
				ShowConditions: true,
				Direction:      "LR",
			},
			want: []string{
				"flowchart LR",
				`a["HTTP: Node A"]`,
			},
		},
		{
			name: "workflow with LR direction",
			workflow: &models.Workflow{
				Name: "Left to Right",
				Nodes: []*models.Node{
					{ID: "a", Name: "Start", Type: "http"},
				},
				Edges: []*models.Edge{},
			},
			opts: &RenderOptions{
				Direction: "LR",
			},
			want: []string{
				"flowchart LR",
			},
		},
		{
			name: "empty workflow",
			workflow: &models.Workflow{
				Name:  "Empty",
				Nodes: []*models.Node{},
				Edges: []*models.Edge{},
			},
			opts: DefaultRenderOptions(),
			want: []string{
				"flowchart TB",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewMermaidRenderer()
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

func TestMermaidRenderer_NodeShapes(t *testing.T) {
	renderer := NewMermaidRenderer()
	opts := DefaultRenderOptions()

	tests := []struct {
		nodeType      string
		expectedShape string // Part of the shape syntax to check
	}{
		{"http", `["HTTP:`},
		{"llm", `(["LLM:`},
		{"transform", `[/"Transform:`},
		{"conditional", `{"If:`},
		{"merge", `{{"Merge:`},
		{"custom_type", `["CUSTOM_TYPE:`}, // Default to rectangle
	}

	for _, tt := range tests {
		t.Run(tt.nodeType, func(t *testing.T) {
			workflow := &models.Workflow{
				Name: "Test",
				Nodes: []*models.Node{
					{ID: "test", Name: "Test Node", Type: tt.nodeType, Config: map[string]interface{}{}},
				},
				Edges: []*models.Edge{},
			}

			got, err := renderer.Render(workflow, opts)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if !strings.Contains(got, tt.expectedShape) {
				t.Errorf("Expected shape %q not found in output:\n%s", tt.expectedShape, got)
			}
		})
	}
}

func TestMermaidRenderer_ConfigExtraction(t *testing.T) {
	renderer := NewMermaidRenderer()
	opts := DefaultRenderOptions()

	tests := []struct {
		name             string
		node             *models.Node
		expectedInOutput string
	}{
		{
			name: "http node with URL",
			node: &models.Node{
				ID:   "http1",
				Name: "HTTP Call",
				Type: "http",
				Config: map[string]interface{}{
					"method": "POST",
					"url":    "https://api.example.com/users",
				},
			},
			expectedInOutput: "https://api.example.com/users",
		},
		{
			name: "llm node with model",
			node: &models.Node{
				ID:   "llm1",
				Name: "LLM Call",
				Type: "llm",
				Config: map[string]interface{}{
					"provider": "openai",
					"model":    "gpt-4",
				},
			},
			expectedInOutput: "gpt-4",
		},
		{
			name: "transform node with type",
			node: &models.Node{
				ID:   "transform1",
				Name: "Transform",
				Type: "transform",
				Config: map[string]interface{}{
					"type": "expression",
				},
			},
			expectedInOutput: "expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := &models.Workflow{
				Name:  "Test",
				Nodes: []*models.Node{tt.node},
				Edges: []*models.Edge{},
			}

			got, err := renderer.Render(workflow, opts)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			if !strings.Contains(got, tt.expectedInOutput) {
				t.Errorf("Expected config value %q not found in output:\n%s", tt.expectedInOutput, got)
			}
		})
	}
}

func TestMermaidRenderer_ClassAssignment(t *testing.T) {
	renderer := NewMermaidRenderer()
	opts := DefaultRenderOptions()

	workflow := &models.Workflow{
		Name: "Class Assignment Test",
		Nodes: []*models.Node{
			{ID: "http1", Name: "HTTP Node", Type: "http"},
			{ID: "llm1", Name: "LLM Node", Type: "llm"},
			{ID: "transform1", Name: "Transform Node", Type: "transform"},
			{ID: "cond1", Name: "Conditional Node", Type: "conditional"},
			{ID: "merge1", Name: "Merge Node", Type: "merge"},
		},
		Edges: []*models.Edge{},
	}

	got, err := renderer.Render(workflow, opts)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Check that classDef declarations exist
	expectedClassDefs := []string{
		"classDef httpNode",
		"classDef llmNode",
		"classDef transformNode",
		"classDef conditionalNode",
		"classDef mergeNode",
	}

	for _, classDef := range expectedClassDefs {
		if !strings.Contains(got, classDef) {
			t.Errorf("Expected classDef %q not found in output", classDef)
		}
	}

	// Check that classes are assigned to nodes
	expectedAssignments := []string{
		"class http1 httpNode",
		"class llm1 llmNode",
		"class transform1 transformNode",
		"class cond1 conditionalNode",
		"class merge1 mergeNode",
	}

	for _, assignment := range expectedAssignments {
		if !strings.Contains(got, assignment) {
			t.Errorf("Expected class assignment %q not found in output:\n%s", assignment, got)
		}
	}
}

func TestMermaidRenderer_ClassAssignment_WithoutConfig(t *testing.T) {
	renderer := NewMermaidRenderer()
	opts := &RenderOptions{
		ShowConfig:     false, // Classes should not be applied when ShowConfig is false
		ShowConditions: true,
		Direction:      "TB",
	}

	workflow := &models.Workflow{
		Name: "No Classes Test",
		Nodes: []*models.Node{
			{ID: "http1", Name: "HTTP Node", Type: "http"},
		},
		Edges: []*models.Edge{},
	}

	got, err := renderer.Render(workflow, opts)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// When ShowConfig is false, no classes should be defined or assigned
	if strings.Contains(got, "classDef") {
		t.Error("classDef should not be present when ShowConfig is false")
	}
	if strings.Contains(got, "class http1") {
		t.Error("class assignment should not be present when ShowConfig is false")
	}
}
