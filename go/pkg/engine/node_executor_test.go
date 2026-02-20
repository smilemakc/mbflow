package engine

import (
	"context"
	"testing"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// mockExecutor is a simple mock executor for testing
type mockExecutor struct {
	validateFn func(config map[string]any) error
	executeFn  func(ctx context.Context, config map[string]any, input any) (any, error)
}

func (m *mockExecutor) Validate(config map[string]any) error {
	if m.validateFn != nil {
		return m.validateFn(config)
	}
	return nil
}

func (m *mockExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, config, input)
	}
	return map[string]any{"status": "ok"}, nil
}

// TestNodeExecutor_Execute_TemplateResolution tests that templates are resolved correctly
func TestNodeExecutor_Execute_TemplateResolution(t *testing.T) {
	t.Parallel()
	// Create mock executor that verifies templates were resolved
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			// Verify that templates were resolved
			if url, ok := config["url"].(string); ok {
				if url != "https://api.com/12345" {
					t.Errorf("expected URL to be 'https://api.com/12345', got '%s'", url)
				}
			} else {
				t.Error("URL not found in config")
			}

			if apiKey, ok := config["api_key"].(string); ok {
				if apiKey != "secret-key" {
					t.Errorf("expected api_key to be 'secret-key', got '%s'", apiKey)
				}
			} else {
				t.Error("api_key not found in config")
			}

			return map[string]any{"result": "success"}, nil
		},
	}

	// Create executor registry and register mock
	registry := executor.NewManager()
	if err := registry.Register("http", mockExec); err != nil {
		t.Fatalf("failed to register executor: %v", err)
	}

	// Create node executor
	nodeExec := NewNodeExecutor(registry)

	// Create node context with templates
	nodeCtx := &NodeContext{
		ExecutionID: "exec-123",
		NodeID:      "node-1",
		Node: &models.Node{
			ID:   "node-1",
			Name: "Test Node",
			Type: "http",
			Config: map[string]any{
				"url":     "https://api.com/{{input.userId}}",
				"api_key": "{{env.apiKey}}",
			},
		},
		WorkflowVariables: map[string]any{
			"apiKey": "secret-key",
		},
		ExecutionVariables: map[string]any{},
		DirectParentOutput: map[string]any{
			"userId": "12345",
		},
		StrictMode: false,
	}

	// Execute
	execResult, err := nodeExec.Execute(context.Background(), nodeCtx)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Verify result
	if execResult == nil {
		t.Fatalf("expected NodeExecutionResult, got nil")
	}

	resultMap, ok := execResult.Output.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", execResult.Output)
	}

	if resultMap["result"] != "success" {
		t.Errorf("unexpected result: %v", execResult.Output)
	}

	// Verify metadata
	if execResult.Config == nil {
		t.Error("expected Config to be set")
	}
	if execResult.ResolvedConfig == nil {
		t.Error("expected ResolvedConfig to be set")
	}
	if execResult.Input == nil {
		t.Error("expected Input to be set")
	}
}

// TestNodeExecutor_Execute_ExecutionVariablesOverride tests that execution variables override workflow variables
func TestNodeExecutor_Execute_ExecutionVariablesOverride(t *testing.T) {
	t.Parallel()
	mockExec := &mockExecutor{
		executeFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			// Verify that execution var overrode workflow var
			if val, ok := config["key"].(string); ok {
				if val != "execution-value" {
					t.Errorf("expected 'execution-value', got '%s'", val)
				}
			} else {
				t.Error("key not found in config")
			}
			return map[string]any{"result": "ok"}, nil
		},
	}

	registry := executor.NewManager()
	registry.Register("test", mockExec)
	nodeExec := NewNodeExecutor(registry)

	nodeCtx := &NodeContext{
		ExecutionID: "exec-123",
		NodeID:      "node-1",
		Node: &models.Node{
			ID:   "node-1",
			Type: "test",
			Config: map[string]any{
				"key": "{{env.testKey}}",
			},
		},
		WorkflowVariables: map[string]any{
			"testKey": "workflow-value",
		},
		ExecutionVariables: map[string]any{
			"testKey": "execution-value", // This should override workflow value
		},
		DirectParentOutput: map[string]any{},
		StrictMode:         false,
	}

	result, err := nodeExec.Execute(context.Background(), nodeCtx)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}
}

// TestNodeExecutor_Execute_MultipleParentOutputs tests namespace collision avoidance with multiple parents
func TestPrepareNodeContext_MultipleParents(t *testing.T) {
	t.Parallel()
	execState := &ExecutionState{
		ExecutionID: "exec-123",
		WorkflowID:  "wf-123",
		Workflow: &models.Workflow{
			ID:        "wf-123",
			Name:      "Test",
			Variables: map[string]any{"key": "value"},
		},
		NodeOutputs: map[string]any{
			"parent1": map[string]any{"field1": "value1"},
			"parent2": map[string]any{"field2": "value2"},
		},
	}

	node := &models.Node{
		ID:   "node-3",
		Name: "Child",
		Type: "test",
	}

	parentNodes := []*models.Node{
		{ID: "parent1", Name: "Parent 1"},
		{ID: "parent2", Name: "Parent 2"},
	}

	opts := DefaultExecutionOptions()

	nodeCtx := PrepareNodeContext(execState, node, parentNodes, opts)

	// Verify DirectParentOutput has namespaced outputs
	if nodeCtx.DirectParentOutput == nil {
		t.Fatal("DirectParentOutput is nil")
	}

	// Check that outputs are namespaced by parent ID
	parent1Output, ok := nodeCtx.DirectParentOutput["parent1"].(map[string]any)
	if !ok {
		t.Error("parent1 output not found or wrong type")
	} else {
		if parent1Output["field1"] != "value1" {
			t.Errorf("expected field1=value1, got %v", parent1Output["field1"])
		}
	}

	parent2Output, ok := nodeCtx.DirectParentOutput["parent2"].(map[string]any)
	if !ok {
		t.Error("parent2 output not found or wrong type")
	} else {
		if parent2Output["field2"] != "value2" {
			t.Errorf("expected field2=value2, got %v", parent2Output["field2"])
		}
	}
}

// TestNodeExecutor_Execute_NoParents tests that execution input is used when node has no parents
func TestPrepareNodeContext_NoParents(t *testing.T) {
	t.Parallel()
	execState := &ExecutionState{
		ExecutionID: "exec-123",
		WorkflowID:  "wf-123",
		Workflow: &models.Workflow{
			ID:        "wf-123",
			Name:      "Test",
			Variables: map[string]any{},
		},
		Input: map[string]any{
			"initialData": "test-value",
		},
	}

	node := &models.Node{
		ID:   "node-1",
		Name: "Start",
		Type: "test",
	}

	nodeCtx := PrepareNodeContext(execState, node, []*models.Node{}, DefaultExecutionOptions())

	// Verify DirectParentOutput equals execution Input
	if nodeCtx.DirectParentOutput == nil {
		t.Fatal("DirectParentOutput is nil")
	}

	if val, ok := nodeCtx.DirectParentOutput["initialData"]; !ok || val != "test-value" {
		t.Errorf("expected initialData=test-value, got %v", val)
	}
}
