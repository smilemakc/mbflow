package importer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// mockExecutorManager implements executor.Manager for testing.
type mockExecutorManager struct {
	registeredTypes map[string]bool
}

func newMockExecutorManager(types ...string) *mockExecutorManager {
	m := &mockExecutorManager{
		registeredTypes: make(map[string]bool),
	}
	for _, t := range types {
		m.registeredTypes[t] = true
	}
	return m
}

func (m *mockExecutorManager) Has(nodeType string) bool {
	return m.registeredTypes[nodeType]
}

func (m *mockExecutorManager) List() []string {
	types := make([]string, 0, len(m.registeredTypes))
	for t := range m.registeredTypes {
		types = append(types, t)
	}
	return types
}

func (m *mockExecutorManager) Register(string, executor.Executor) error { return nil }
func (m *mockExecutorManager) Get(string) (executor.Executor, error)    { return nil, nil }
func (m *mockExecutorManager) Unregister(string) error                  { return nil }

// mockExecutor implements executor.Executor for testing.
type mockExecutor struct{}

func (m *mockExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	return nil, nil
}

func (m *mockExecutor) Validate(config map[string]any) error {
	return nil
}

func TestYAMLImporter_ImportFromYAML_BasicWorkflow(t *testing.T) {
	yaml := `
metadata:
  name: "Test Workflow"
  description: "A test workflow"
  version: 1
  tags:
    - test
    - example

nodes:
  - id: node_1
    name: "HTTP Request"
    type: http
    config:
      url: "https://api.example.com"
      method: "GET"
    position:
      x: 100
      y: 200

  - id: node_2
    name: "Transform Data"
    type: transform
    config:
      mapping:
        result: "$.data"

edges:
  - id: e1
    from: node_1
    to: node_2
`

	manager := newMockExecutorManager("http", "transform")
	importer := NewYAMLImporter(manager)

	result, err := importer.ImportFromYAML([]byte(yaml))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Workflow)

	workflow := result.Workflow
	assert.Equal(t, "Test Workflow", workflow.Name)
	assert.Equal(t, "A test workflow", workflow.Description)
	assert.Equal(t, 1, workflow.Version)
	assert.Equal(t, []string{"test", "example"}, workflow.Tags)
	assert.Equal(t, models.WorkflowStatusDraft, workflow.Status)

	assert.Len(t, workflow.Nodes, 2)
	assert.Equal(t, "node_1", workflow.Nodes[0].ID)
	assert.Equal(t, "HTTP Request", workflow.Nodes[0].Name)
	assert.Equal(t, "http", workflow.Nodes[0].Type)
	assert.NotNil(t, workflow.Nodes[0].Position)
	assert.Equal(t, float64(100), workflow.Nodes[0].Position.X)
	assert.Equal(t, float64(200), workflow.Nodes[0].Position.Y)

	assert.Len(t, workflow.Edges, 1)
	assert.Equal(t, "e1", workflow.Edges[0].ID)
	assert.Equal(t, "node_1", workflow.Edges[0].From)
	assert.Equal(t, "node_2", workflow.Edges[0].To)

	assert.Equal(t, 2, result.NodesCount)
	assert.Equal(t, 1, result.EdgesCount)
	assert.Nil(t, result.Trigger)
}

func TestYAMLImporter_ImportFromYAML_WithTrigger(t *testing.T) {
	yaml := `
metadata:
  name: "Scheduled Workflow"

nodes:
  - id: start
    name: "Start"
    type: http
    config:
      url: "https://api.example.com"

trigger:
  name: "Daily Schedule"
  description: "Runs every day at 9 AM"
  type: cron
  enabled: true
  config:
    schedule: "0 9 * * *"
    timezone: "UTC"
`

	manager := newMockExecutorManager("http")
	importer := NewYAMLImporter(manager)

	result, err := importer.ImportFromYAML([]byte(yaml))

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Trigger)

	trigger := result.Trigger
	assert.Equal(t, "Daily Schedule", trigger.Name)
	assert.Equal(t, "Runs every day at 9 AM", trigger.Description)
	assert.Equal(t, models.TriggerTypeCron, trigger.Type)
	assert.True(t, trigger.Enabled)
	assert.Equal(t, "0 9 * * *", trigger.Config["schedule"])
	assert.Equal(t, "UTC", trigger.Config["timezone"])
}

func TestYAMLImporter_ImportFromYAML_WithVariables(t *testing.T) {
	yaml := `
metadata:
  name: "Workflow with Variables"

variables:
  api_key: "{{env.API_KEY}}"
  base_url: "https://api.example.com"
  timeout: 30

nodes:
  - id: request
    name: "API Request"
    type: http
    config:
      url: "{{variables.base_url}}/data"
`

	manager := newMockExecutorManager("http")
	importer := NewYAMLImporter(manager)

	result, err := importer.ImportFromYAML([]byte(yaml))

	require.NoError(t, err)
	require.NotNil(t, result)

	workflow := result.Workflow
	assert.NotNil(t, workflow.Variables)
	assert.Equal(t, "{{env.API_KEY}}", workflow.Variables["api_key"])
	assert.Equal(t, "https://api.example.com", workflow.Variables["base_url"])
	assert.Equal(t, 30, workflow.Variables["timeout"])
}

func TestYAMLImporter_ImportFromYAML_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectedErr string
	}{
		{
			name: "missing workflow name",
			yaml: `
metadata:
  description: "No name"
nodes:
  - id: n1
    name: "Node"
    type: http
`,
			expectedErr: "metadata.name",
		},
		{
			name: "no nodes",
			yaml: `
metadata:
  name: "Empty Workflow"
`,
			expectedErr: "nodes",
		},
		{
			name: "missing node id",
			yaml: `
metadata:
  name: "Test"
nodes:
  - name: "Node without ID"
    type: http
`,
			expectedErr: "nodes[0].id",
		},
		{
			name: "missing node name",
			yaml: `
metadata:
  name: "Test"
nodes:
  - id: n1
    type: http
`,
			expectedErr: "nodes[0].name",
		},
		{
			name: "missing node type",
			yaml: `
metadata:
  name: "Test"
nodes:
  - id: n1
    name: "Node"
`,
			expectedErr: "nodes[0].type",
		},
		{
			name: "duplicate node ids",
			yaml: `
metadata:
  name: "Test"
nodes:
  - id: n1
    name: "Node 1"
    type: http
  - id: n1
    name: "Node 2"
    type: http
`,
			expectedErr: "duplicate node ID",
		},
		{
			name: "edge references non-existent source",
			yaml: `
metadata:
  name: "Test"
nodes:
  - id: n1
    name: "Node"
    type: http
edges:
  - id: e1
    from: nonexistent
    to: n1
`,
			expectedErr: "non-existent source",
		},
		{
			name: "edge references non-existent target",
			yaml: `
metadata:
  name: "Test"
nodes:
  - id: n1
    name: "Node"
    type: http
edges:
  - id: e1
    from: n1
    to: nonexistent
`,
			expectedErr: "non-existent target",
		},
		{
			name: "self-loop edge",
			yaml: `
metadata:
  name: "Test"
nodes:
  - id: n1
    name: "Node"
    type: http
edges:
  - id: e1
    from: n1
    to: n1
`,
			expectedErr: "self-loop",
		},
		{
			name: "invalid trigger type",
			yaml: `
metadata:
  name: "Test"
nodes:
  - id: n1
    name: "Node"
    type: http
trigger:
  name: "Bad Trigger"
  type: invalid_type
`,
			expectedErr: "invalid trigger type",
		},
	}

	manager := newMockExecutorManager("http")
	importer := NewYAMLImporter(manager)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := importer.ImportFromYAML([]byte(tt.yaml))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestYAMLImporter_ImportFromYAML_UnknownNodeType(t *testing.T) {
	yaml := `
metadata:
  name: "Test"
nodes:
  - id: n1
    name: "Node"
    type: unknown_executor
`

	manager := newMockExecutorManager("http", "transform")
	importer := NewYAMLImporter(manager)

	_, err := importer.ImportFromYAML([]byte(yaml))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown executor type")
}

func TestYAMLImporter_ImportFromYAML_WithConditions(t *testing.T) {
	yaml := `
metadata:
  name: "Conditional Workflow"

nodes:
  - id: start
    name: "Start"
    type: http
  - id: condition
    name: "Check"
    type: conditional
  - id: success
    name: "Success"
    type: http
  - id: failure
    name: "Failure"
    type: http

edges:
  - id: e1
    from: start
    to: condition
  - id: e2
    from: condition
    to: success
    condition: "output == 'success'"
  - id: e3
    from: condition
    to: failure
    condition: "output == 'failure'"
`

	manager := newMockExecutorManager("http", "conditional")
	importer := NewYAMLImporter(manager)

	result, err := importer.ImportFromYAML([]byte(yaml))

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Len(t, result.Workflow.Edges, 3)

	// Find edge with condition
	var conditionalEdge *models.Edge
	for _, e := range result.Workflow.Edges {
		if e.ID == "e2" {
			conditionalEdge = e
			break
		}
	}

	require.NotNil(t, conditionalEdge)
	assert.Equal(t, "output == 'success'", conditionalEdge.Condition)
}

func TestYAMLImporter_ExportToYAML(t *testing.T) {
	workflow := &models.Workflow{
		ID:          "test-id",
		Name:        "Exported Workflow",
		Description: "A workflow for export testing",
		Version:     2,
		Tags:        []string{"export", "test"},
		Variables: map[string]any{
			"api_key": "secret",
		},
		Nodes: []*models.Node{
			{
				ID:   "n1",
				Name: "Node 1",
				Type: "http",
				Config: map[string]any{
					"url": "https://api.example.com",
				},
				Position: &models.Position{X: 100, Y: 200},
			},
			{
				ID:   "n2",
				Name: "Node 2",
				Type: "transform",
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "e1",
				From: "n1",
				To:   "n2",
			},
		},
	}

	trigger := &models.Trigger{
		ID:          "trigger-id",
		WorkflowID:  "test-id",
		Name:        "Test Trigger",
		Description: "A test trigger",
		Type:        models.TriggerTypeCron,
		Enabled:     true,
		Config: map[string]any{
			"schedule": "0 * * * *",
		},
	}

	importer := NewYAMLImporter(nil)

	yamlData, err := importer.ExportToYAML(workflow, trigger)

	require.NoError(t, err)
	assert.NotEmpty(t, yamlData)

	// Parse it back to verify round-trip
	manager := newMockExecutorManager("http", "transform")
	importer2 := NewYAMLImporter(manager)

	result, err := importer2.ImportFromYAML(yamlData)

	require.NoError(t, err)
	assert.Equal(t, "Exported Workflow", result.Workflow.Name)
	assert.Equal(t, "A workflow for export testing", result.Workflow.Description)
	assert.Equal(t, 2, result.Workflow.Version)
	assert.Len(t, result.Workflow.Nodes, 2)
	assert.Len(t, result.Workflow.Edges, 1)
	assert.NotNil(t, result.Trigger)
	assert.Equal(t, "Test Trigger", result.Trigger.Name)
}

func TestYAMLImporter_ExportToYAML_NoTrigger(t *testing.T) {
	workflow := &models.Workflow{
		ID:   "test-id",
		Name: "Simple Workflow",
		Nodes: []*models.Node{
			{
				ID:   "n1",
				Name: "Node",
				Type: "http",
			},
		},
	}

	importer := NewYAMLImporter(nil)

	yamlData, err := importer.ExportToYAML(workflow, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, yamlData)
	assert.NotContains(t, string(yamlData), "trigger:")
}

func TestYAMLImporter_GetSupportedNodeTypes(t *testing.T) {
	manager := newMockExecutorManager("http", "transform", "llm", "conditional")
	importer := NewYAMLImporter(manager)

	types := importer.GetSupportedNodeTypes()

	assert.Len(t, types, 4)
	assert.Contains(t, types, "http")
	assert.Contains(t, types, "transform")
	assert.Contains(t, types, "llm")
	assert.Contains(t, types, "conditional")
}

func TestYAMLImporter_ValidateNodeType(t *testing.T) {
	manager := newMockExecutorManager("http", "transform")
	importer := NewYAMLImporter(manager)

	assert.True(t, importer.ValidateNodeType("http"))
	assert.True(t, importer.ValidateNodeType("transform"))
	assert.False(t, importer.ValidateNodeType("unknown"))
}

func TestParseYAMLContent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "normal content",
			input:   "metadata:\n  name: test",
			wantErr: false,
		},
		{
			name:    "content with BOM",
			input:   "\xef\xbb\xbfmetadata:\n  name: test",
			wantErr: false,
		},
		{
			name:    "content with leading whitespace",
			input:   "   \n\nmetadata:\n  name: test",
			wantErr: false,
		},
		{
			name:    "empty content",
			input:   "",
			wantErr: true,
		},
		{
			name:    "only whitespace",
			input:   "   \n\n   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseYAMLContent([]byte(tt.input))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestYAMLImporter_AllTriggerTypes(t *testing.T) {
	triggerTypes := []struct {
		triggerType string
		config      string
	}{
		{"manual", ""},
		{"cron", "\n    schedule: \"0 * * * *\""},
		{"webhook", "\n    path: \"/hooks/test\""},
		{"event", "\n    event_type: \"user.created\""},
		{"interval", "\n    interval: \"5m\""},
	}

	manager := newMockExecutorManager("http")

	for _, tt := range triggerTypes {
		t.Run(tt.triggerType, func(t *testing.T) {
			yaml := `
metadata:
  name: "Test"
nodes:
  - id: n1
    name: "Node"
    type: http
trigger:
  name: "Test Trigger"
  type: ` + tt.triggerType
			if tt.config != "" {
				yaml += "\n  config:" + tt.config
			}

			importer := NewYAMLImporter(manager)
			result, err := importer.ImportFromYAML([]byte(yaml))

			require.NoError(t, err)
			require.NotNil(t, result.Trigger)
			assert.Equal(t, models.TriggerType(tt.triggerType), result.Trigger.Type)
		})
	}
}

func TestYAMLImporter_DefaultValues(t *testing.T) {
	yaml := `
metadata:
  name: "Minimal Workflow"
nodes:
  - id: n1
    name: "Node"
    type: http
trigger:
  name: "Trigger"
  type: manual
`

	manager := newMockExecutorManager("http")
	importer := NewYAMLImporter(manager)

	result, err := importer.ImportFromYAML([]byte(yaml))

	require.NoError(t, err)

	// Check default version
	assert.Equal(t, 1, result.Workflow.Version)

	// Check default trigger enabled
	assert.True(t, result.Trigger.Enabled)

	// Check default status
	assert.Equal(t, models.WorkflowStatusDraft, result.Workflow.Status)
}
