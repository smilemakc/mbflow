package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// toolTestExecutor returns a DAG executor whose "transform" nodes return their
// input plus the "token" resource their template context resolved, and whose
// "fail" nodes fail.
func toolTestExecutor(workflows map[string]*models.Workflow) *DAGExecutor {
	registry := executor.NewManager()
	_ = registry.Register("transform", &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, _ map[string]any, input any) (any, error) {
			out := map[string]any{"input": input}
			if data, ok := executor.GetExecutionContext(ctx); ok {
				out["resources"] = data.Resources
			}
			return out, nil
		},
	})
	_ = registry.Register("fail", &executor.ExecutorFunc{
		ExecuteFn: func(context.Context, map[string]any, any) (any, error) {
			return nil, errors.New("boom")
		},
	})
	return NewDAGExecutor(NewNodeExecutor(registry), NewExprConditionEvaluator(), NewNoOpNotifier(), NewMockWorkflowLoader(workflows))
}

func callerContext(owner string) context.Context {
	return executor.WithExecutionContext(context.Background(), &executor.ExecutionContextData{
		ExecutionID: "parent-exec",
		WorkflowID:  "parent-wf",
		NodeID:      "agent",
		OwnerID:     owner,
		Resources:   map[string]any{"anthropic": map[string]any{"api_key": "k"}},
	})
}

func TestExecuteWorkflow_RunsOwnedWorkflow(t *testing.T) {
	t.Parallel()
	dag := toolTestExecutor(map[string]*models.Workflow{
		"tool": {ID: "tool", Name: "Tool", CreatedBy: "alice", Nodes: []*models.Node{
			{ID: "only", Name: "Only", Type: "transform", Config: map[string]any{}},
		}},
	})

	out, err := dag.ExecuteWorkflow(callerContext("alice"), "tool", map[string]any{"country": "Brazil"})
	require.NoError(t, err)

	result, ok := out.(map[string]any)
	require.True(t, ok, "a single terminal node's output is returned as is: %T", out)
	assert.Equal(t, map[string]any{"country": "Brazil"}, result["input"])
	assert.Equal(t, map[string]any{"anthropic": map[string]any{"api_key": "k"}}, result["resources"],
		"the tool workflow inherits the caller's credentials")
}

func TestExecuteWorkflow_Errors(t *testing.T) {
	t.Parallel()
	dag := toolTestExecutor(map[string]*models.Workflow{
		"tool":   {ID: "tool", Name: "Tool", CreatedBy: "alice", Nodes: []*models.Node{{ID: "n", Name: "N", Type: "transform"}}},
		"broken": {ID: "broken", Name: "Broken", CreatedBy: "alice", Nodes: []*models.Node{{ID: "n", Name: "N", Type: "fail"}}},
	})

	t.Run("outside an execution", func(t *testing.T) {
		_, err := dag.ExecuteWorkflow(context.Background(), "tool", nil)
		require.ErrorContains(t, err, "outside of a workflow execution")
	})

	t.Run("another owner's workflow", func(t *testing.T) {
		_, err := dag.ExecuteWorkflow(callerContext("mallory"), "tool", nil)
		require.ErrorIs(t, err, ErrSubWorkflowNotAccessible)
	})

	t.Run("unknown workflow", func(t *testing.T) {
		_, err := dag.ExecuteWorkflow(callerContext("alice"), "missing", nil)
		require.ErrorContains(t, err, "failed to load workflow missing")
	})

	t.Run("failing workflow", func(t *testing.T) {
		_, err := dag.ExecuteWorkflow(callerContext("alice"), "broken", nil)
		require.ErrorContains(t, err, "boom")
	})

	t.Run("nesting limit", func(t *testing.T) {
		ctx := context.WithValue(callerContext("alice"), workflowToolDepthKey{}, maxWorkflowToolDepth)
		_, err := dag.ExecuteWorkflow(ctx, "tool", nil)
		require.ErrorIs(t, err, ErrWorkflowToolDepth)
	})

	t.Run("no loader", func(t *testing.T) {
		bare := NewDAGExecutor(NewNodeExecutor(executor.NewManager()), NewExprConditionEvaluator(), NewNoOpNotifier(), nil)
		_, err := bare.ExecuteWorkflow(callerContext("alice"), "tool", nil)
		require.ErrorContains(t, err, "workflow loader not configured")
	})
}
