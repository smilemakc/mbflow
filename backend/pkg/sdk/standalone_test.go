package sdk

import (
	"context"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/pkg/models"
)

func TestExecuteWorkflowStandalone(t *testing.T) {
	// Create standalone client (no database)
	client, err := NewStandaloneClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a simple workflow
	workflow := &models.Workflow{
		Name:        "Test Workflow",
		Description: "Simple test workflow",
		Variables: map[string]interface{}{
			"api_base": "https://jsonplaceholder.typicode.com",
		},
		Nodes: []*models.Node{
			{
				ID:   "http-node",
				Name: "HTTP Request",
				Type: "http",
				Config: map[string]interface{}{
					"method": "GET",
					"url":    "{{env.api_base}}/users/1",
				},
			},
			{
				ID:   "transform-node",
				Name: "Transform",
				Type: "transform",
				Config: map[string]interface{}{
					"type": "passthrough",
				},
			},
		},
		Edges: []*models.Edge{
			{
				ID:   "edge-1",
				From: "http-node",
				To:   "transform-node",
			},
		},
	}

	// Execute workflow standalone
	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, nil, nil)
	if err != nil {
		t.Fatalf("ExecuteWorkflowStandalone failed: %v", err)
	}

	// Verify execution
	if execution == nil {
		t.Fatal("Execution is nil")
	}

	if execution.Status != models.ExecutionStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.ExecutionStatusCompleted, execution.Status)
		if execution.Error != "" {
			t.Logf("Execution error: %s", execution.Error)
		}
	}

	if execution.WorkflowName != workflow.Name {
		t.Errorf("Expected workflow name %s, got %s", workflow.Name, execution.WorkflowName)
	}

	if len(execution.NodeExecutions) != 2 {
		t.Errorf("Expected 2 node executions, got %d", len(execution.NodeExecutions))
	}

	// Verify all nodes completed successfully
	for _, nodeExec := range execution.NodeExecutions {
		if nodeExec.Status != models.NodeExecutionStatusCompleted {
			t.Errorf("Node %s status: expected %s, got %s", nodeExec.NodeID, models.NodeExecutionStatusCompleted, nodeExec.Status)
			if nodeExec.Error != "" {
				t.Logf("Node error: %s", nodeExec.Error)
			}
		}
	}
}

func TestExecuteWorkflowStandalone_WithInput(t *testing.T) {
	client, err := NewStandaloneClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create workflow with input templates
	workflow := &models.Workflow{
		Name: "Test Input Template",
		Nodes: []*models.Node{
			{
				ID:   "transform",
				Name: "Transform",
				Type: "transform",
				Config: map[string]interface{}{
					"type": "passthrough",
				},
			},
		},
	}

	input := map[string]interface{}{
		"test_field": "test_value",
	}

	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, input, nil)
	if err != nil {
		t.Fatalf("ExecuteWorkflowStandalone failed: %v", err)
	}

	if execution.Status != models.ExecutionStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.ExecutionStatusCompleted, execution.Status)
	}

	// Verify input was passed
	if execution.Input["test_field"] != "test_value" {
		t.Errorf("Expected input field test_field=test_value, got %v", execution.Input["test_field"])
	}
}

func TestExecuteWorkflowStandalone_WithOptions(t *testing.T) {
	client, err := NewStandaloneClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	workflow := &models.Workflow{
		Name: "Test Options",
		Variables: map[string]interface{}{
			"workflow_var": "workflow_value",
		},
		Nodes: []*models.Node{
			{
				ID:   "transform",
				Name: "Transform",
				Type: "transform",
				Config: map[string]interface{}{
					"type": "passthrough",
				},
			},
		},
	}

	opts := &engine.ExecutionOptions{
		StrictMode:     false,
		MaxParallelism: 5,
		Variables: map[string]interface{}{
			"execution_var": "execution_value",
		},
	}

	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, nil, opts)
	if err != nil {
		t.Fatalf("ExecuteWorkflowStandalone failed: %v", err)
	}

	if execution.Status != models.ExecutionStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.ExecutionStatusCompleted, execution.Status)
	}

	// Verify variables were merged
	if execution.Variables["workflow_var"] != "workflow_value" {
		t.Errorf("Expected workflow_var=workflow_value, got %v", execution.Variables["workflow_var"])
	}
	if execution.Variables["execution_var"] != "execution_value" {
		t.Errorf("Expected execution_var=execution_value, got %v", execution.Variables["execution_var"])
	}
}

func TestExecuteWorkflowStandalone_FailedExecution(t *testing.T) {
	client, err := NewStandaloneClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create workflow with invalid node type
	workflow := &models.Workflow{
		Name: "Test Failed Execution",
		Nodes: []*models.Node{
			{
				ID:     "invalid-node",
				Name:   "Invalid Node",
				Type:   "nonexistent-executor-type",
				Config: map[string]interface{}{},
			},
		},
	}

	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, nil, nil)

	// Expect error
	if err == nil {
		t.Fatal("Expected error for invalid executor type, got nil")
	}

	// Execution should still be returned with failed status
	if execution == nil {
		t.Fatal("Expected execution to be returned even on failure")
	}

	if execution.Status != models.ExecutionStatusFailed {
		t.Errorf("Expected status %s, got %s", models.ExecutionStatusFailed, execution.Status)
	}

	if execution.Error == "" {
		t.Error("Expected error message to be set")
	}
}

func TestExecuteWorkflowStandalone_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Close client
	client.Close()

	ctx := context.Background()
	workflow := &models.Workflow{
		Name: "Test",
		Nodes: []*models.Node{
			{
				ID:   "test",
				Name: "Test",
				Type: "transform",
				Config: map[string]interface{}{
					"type": "passthrough",
				},
			},
		},
	}

	_, err = client.ExecuteWorkflowStandalone(ctx, workflow, nil, nil)
	if err == nil {
		t.Fatal("Expected error for closed client, got nil")
	}

	if err != models.ErrClientClosed {
		t.Errorf("Expected ErrClientClosed, got %v", err)
	}
}

func TestExecuteWorkflowStandalone_Duration(t *testing.T) {
	client, err := NewStandaloneClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Use HTTP node to ensure execution takes measurable time
	workflow := &models.Workflow{
		Name: "Test Duration",
		Nodes: []*models.Node{
			{
				ID:   "http-node",
				Name: "HTTP Request",
				Type: "http",
				Config: map[string]interface{}{
					"method": "GET",
					"url":    "https://jsonplaceholder.typicode.com/users/1",
				},
			},
		},
	}

	start := time.Now()
	execution, err := client.ExecuteWorkflowStandalone(ctx, workflow, nil, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ExecuteWorkflowStandalone failed: %v", err)
	}

	// Verify duration is non-negative (can be 0 for very fast executions)
	if execution.Duration < 0 {
		t.Errorf("Expected non-negative duration, got %d", execution.Duration)
	}

	// Duration should be approximately equal to elapsed time (within 200ms tolerance)
	diff := int64(elapsed.Milliseconds()) - execution.Duration
	if diff < 0 {
		diff = -diff
	}
	if diff > 200 {
		t.Errorf("Duration mismatch: execution.Duration=%dms, elapsed=%dms, diff=%dms", execution.Duration, elapsed.Milliseconds(), diff)
	}

	// Verify execution times are set
	if execution.StartedAt.IsZero() {
		t.Error("Expected StartedAt to be set")
	}

	if execution.CompletedAt == nil || execution.CompletedAt.IsZero() {
		t.Error("Expected CompletedAt to be set")
	}
}
