package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

// buildTestWorkflow creates a simple 2-node linear workflow (node1 -> node2)
// that passes validation and uses the given executor type.
func buildTestWorkflow(executorType string) *models.Workflow {
	return &models.Workflow{
		ID:   "ephemeral-wf",
		Name: "Ephemeral Test Workflow",
		Nodes: []*models.Node{
			{ID: "node1", Name: "First", Type: executorType, Config: map[string]any{"nodeID": "node1"}},
			{ID: "node2", Name: "Second", Type: executorType, Config: map[string]any{"nodeID": "node2"}},
		},
		Edges: []*models.Edge{
			{ID: "edge1", From: "node1", To: "node2"},
		},
	}
}

// buildSingleNodeWorkflow creates a workflow with a single node.
func buildSingleNodeWorkflow(executorType string) *models.Workflow {
	return &models.Workflow{
		ID:   "ephemeral-wf-single",
		Name: "Single Node Workflow",
		Nodes: []*models.Node{
			{ID: "only-node", Name: "Only Node", Type: executorType, Config: map[string]any{"nodeID": "only-node"}},
		},
		Edges: []*models.Edge{},
	}
}

// newMockExecutorFunc creates a simple mock executor that returns the given output.
func newMockExecutorFunc(output any, err error) executor.Executor {
	return &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return output, err
		},
	}
}

// newEphemeralTestManager creates an ExecutionManager wired with a mock executor
// and no DB repositories (nil). The executor is registered under the given type name.
func newEphemeralTestManager(executorType string, exec executor.Executor) *ExecutionManager {
	registry := executor.NewManager()
	_ = registry.Register(executorType, exec)

	return &ExecutionManager{
		executorManager: registry,
		// No repos needed for ephemeral execution with persist=false
		workflowRepo:  nil,
		executionRepo: nil,
		eventRepo:     nil,
		resourceRepo:  nil,
		observerManager: nil,
	}
}

// --- tests ---

func TestEphemeralExecution_Sync_ShouldReturnCompletedExecution_WhenWorkflowSucceeds(t *testing.T) {
	// Arrange
	const execType = "test"
	mockExec := newMockExecutorFunc(map[string]any{"result": "ok", "value": 42}, nil)
	em := newEphemeralTestManager(execType, mockExec)

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: buildTestWorkflow(execType),
		Input:    map[string]any{"greeting": "hello"},
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, execution)

	assert.Equal(t, models.ExecutionStatusCompleted, execution.Status)
	assert.Equal(t, "inline", execution.WorkflowSource)
	assert.Equal(t, "Ephemeral Test Workflow", execution.WorkflowName)
	assert.NotEmpty(t, execution.ID)
	assert.NotNil(t, execution.CompletedAt)
	assert.Greater(t, execution.Duration, int64(-1))

	// Output should come from the leaf node (node2)
	require.NotNil(t, execution.Output)
	assert.Equal(t, "ok", execution.Output["result"])
	assert.Equal(t, 42, execution.Output["value"])

	// Input should be preserved
	assert.Equal(t, map[string]any{"greeting": "hello"}, execution.Input)

	// NodeExecutions should be populated for both nodes with logical IDs (not UUIDs)
	require.Len(t, execution.NodeExecutions, 2)

	nodeExecMap := make(map[string]*models.NodeExecution)
	for _, ne := range execution.NodeExecutions {
		nodeExecMap[ne.NodeID] = ne
	}

	node1Exec, ok := nodeExecMap["node1"]
	require.True(t, ok, "node1 execution should be present")
	assert.Equal(t, models.NodeExecutionStatusCompleted, node1Exec.Status)
	assert.Equal(t, "First", node1Exec.NodeName)
	assert.Equal(t, execType, node1Exec.NodeType)

	node2Exec, ok := nodeExecMap["node2"]
	require.True(t, ok, "node2 execution should be present")
	assert.Equal(t, models.NodeExecutionStatusCompleted, node2Exec.Status)
	assert.Equal(t, "Second", node2Exec.NodeName)
}

func TestEphemeralExecution_SyncWithError_ShouldReturnFailedExecution_WhenNodeFails(t *testing.T) {
	// Arrange
	const execType = "test"
	nodeError := fmt.Errorf("simulated node failure")
	mockExec := newMockExecutorFunc(nil, nodeError)
	em := newEphemeralTestManager(execType, mockExec)

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: buildTestWorkflow(execType),
		Input:    map[string]any{},
	}

	// Act
	execution, execErr := em.ExecuteEphemeral(context.Background(), opts)

	// Assert: ExecuteEphemeral returns both the execution and the execution error
	require.NotNil(t, execution)
	require.Error(t, execErr)

	assert.Equal(t, models.ExecutionStatusFailed, execution.Status)
	assert.Contains(t, execution.Error, "simulated node failure")
	assert.Equal(t, "inline", execution.WorkflowSource)
	assert.NotNil(t, execution.CompletedAt)
}

func TestEphemeralExecution_Async_ShouldReturnPendingExecution_WhenStarted(t *testing.T) {
	// Arrange: use a slow executor so async goroutine doesn't finish before our assertions
	const execType = "slow-test"
	slowExec := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			time.Sleep(200 * time.Millisecond)
			return map[string]any{"done": true}, nil
		},
	}
	em := newEphemeralTestManager(execType, slowExec)

	opts := &EphemeralExecutionOptions{
		Mode:     "async",
		Workflow: buildTestWorkflow(execType),
		Input:    map[string]any{"key": "val"},
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert: returns immediately with pending status
	require.NoError(t, err)
	require.NotNil(t, execution)

	assert.Equal(t, models.ExecutionStatusPending, execution.Status)
	assert.Equal(t, "inline", execution.WorkflowSource)
	assert.NotEmpty(t, execution.ID)
	assert.Equal(t, map[string]any{"key": "val"}, execution.Input)
}

func TestEphemeralExecution_WorkflowSourceIsInline_ShouldAlwaysSetInline(t *testing.T) {
	// Arrange
	const execType = "test"
	mockExec := newMockExecutorFunc(map[string]any{"ok": true}, nil)
	em := newEphemeralTestManager(execType, mockExec)

	tests := []struct {
		name string
		mode string
	}{
		{name: "sync mode", mode: "sync"},
		{name: "async mode", mode: "async"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &EphemeralExecutionOptions{
				Mode:     tt.mode,
				Workflow: buildTestWorkflow(execType),
			}

			execution, err := em.ExecuteEphemeral(context.Background(), opts)

			require.NoError(t, err)
			require.NotNil(t, execution)
			assert.Equal(t, "inline", execution.WorkflowSource)
		})
	}
}

func TestEphemeralExecution_ShouldReturnError_WhenWorkflowIsNil(t *testing.T) {
	// Arrange
	const execType = "test"
	mockExec := newMockExecutorFunc(nil, nil)
	em := newEphemeralTestManager(execType, mockExec)

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: nil,
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.Error(t, err)
	assert.Nil(t, execution)
	assert.Contains(t, err.Error(), "workflow is required")
}

func TestEphemeralExecution_ShouldReturnError_WhenWorkflowIsInvalid(t *testing.T) {
	// Arrange
	const execType = "test"
	mockExec := newMockExecutorFunc(nil, nil)
	em := newEphemeralTestManager(execType, mockExec)

	invalidWorkflow := &models.Workflow{
		// Missing name and nodes -- will fail Validate()
		Nodes: []*models.Node{},
		Edges: []*models.Edge{},
	}

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: invalidWorkflow,
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.Error(t, err)
	assert.Nil(t, execution)
	assert.Contains(t, err.Error(), "invalid workflow")
}

func TestEphemeralExecution_Sync_ShouldMergeVariables_WhenBothWorkflowAndOptsProvided(t *testing.T) {
	// Arrange: executor that captures the resolved variable from config template
	const execType = "test"
	var capturedConfig map[string]any
	templateExec := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			capturedConfig = config
			return map[string]any{"captured": true}, nil
		},
	}
	em := newEphemeralTestManager(execType, templateExec)

	wf := buildSingleNodeWorkflow(execType)
	wf.Variables = map[string]any{"baseKey": "from-workflow"}

	// node config uses the template variable
	wf.Nodes[0].Config = map[string]any{
		"url": "https://api.example.com/{{env.baseKey}}",
	}

	opts := &EphemeralExecutionOptions{
		Mode:      "sync",
		Workflow:  wf,
		Variables: map[string]any{"baseKey": "from-opts"},
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, execution)
	assert.Equal(t, models.ExecutionStatusCompleted, execution.Status)

	// Execution-level variables should override workflow-level variables
	assert.Equal(t, "from-opts", execution.Variables["baseKey"])

	// The template should have been resolved with the overridden value
	require.NotNil(t, capturedConfig)
	assert.Equal(t, "https://api.example.com/from-opts", capturedConfig["url"])
}

func TestEphemeralExecution_Sync_ShouldInitializeEmptyInput_WhenInputIsNil(t *testing.T) {
	// Arrange
	const execType = "test"
	mockExec := newMockExecutorFunc(map[string]any{"ok": true}, nil)
	em := newEphemeralTestManager(execType, mockExec)

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: buildSingleNodeWorkflow(execType),
		Input:    nil, // explicitly nil
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, execution)
	assert.NotNil(t, execution.Input, "input should be initialized to empty map, not nil")
	assert.Equal(t, map[string]any{}, execution.Input)
}

func TestEphemeralExecution_Sync_ShouldReturnSingleNodeOutput_WhenSingleLeafNode(t *testing.T) {
	// Arrange: single-node workflow -> output should be the node's output directly (not wrapped)
	const execType = "test"
	expectedOutput := map[string]any{"answer": "42", "metadata": map[string]any{"source": "test"}}
	mockExec := newMockExecutorFunc(expectedOutput, nil)
	em := newEphemeralTestManager(execType, mockExec)

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: buildSingleNodeWorkflow(execType),
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, execution)
	assert.Equal(t, models.ExecutionStatusCompleted, execution.Status)
	assert.Equal(t, "42", execution.Output["answer"])
}

func TestEphemeralExecution_Sync_ShouldHaveNodeExecutionsWithLogicalIDs(t *testing.T) {
	// Arrange: verify that ephemeral execution uses logical node IDs (not UUIDs from the DB)
	const execType = "test"
	mockExec := newMockExecutorFunc(map[string]any{"ok": true}, nil)
	em := newEphemeralTestManager(execType, mockExec)

	wf := &models.Workflow{
		ID:   "wf-logical",
		Name: "Logical ID Test",
		Nodes: []*models.Node{
			{ID: "my-custom-id-1", Name: "Step A", Type: execType, Config: map[string]any{}},
			{ID: "my-custom-id-2", Name: "Step B", Type: execType, Config: map[string]any{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "my-custom-id-1", To: "my-custom-id-2"},
		},
	}

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: wf,
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, execution)
	require.Len(t, execution.NodeExecutions, 2)

	nodeIDs := make([]string, len(execution.NodeExecutions))
	for i, ne := range execution.NodeExecutions {
		nodeIDs[i] = ne.NodeID
	}
	assert.Contains(t, nodeIDs, "my-custom-id-1")
	assert.Contains(t, nodeIDs, "my-custom-id-2")
}

func TestEphemeralExecution_Sync_ShouldHandleCancelledContext(t *testing.T) {
	// Arrange: cancel the context before execution
	const execType = "test"
	slowExec := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return map[string]any{"done": true}, nil
			}
		},
	}
	em := newEphemeralTestManager(execType, slowExec)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: buildSingleNodeWorkflow(execType),
	}

	// Act
	execution, execErr := em.ExecuteEphemeral(ctx, opts)

	// Assert
	require.NotNil(t, execution)
	require.Error(t, execErr)
	assert.Equal(t, models.ExecutionStatusCancelled, execution.Status)
}

func TestEphemeralExecution_Sync_ShouldPopulateNodeExecutionDetails(t *testing.T) {
	// Arrange: verify that node executions include input, output, config, timing
	const execType = "test"
	mockExec := &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			return map[string]any{"processed": true}, nil
		},
	}
	em := newEphemeralTestManager(execType, mockExec)

	wf := buildSingleNodeWorkflow(execType)
	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: wf,
		Input:    map[string]any{"data": "test-input"},
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, execution)
	require.Len(t, execution.NodeExecutions, 1)

	ne := execution.NodeExecutions[0]
	assert.Equal(t, "only-node", ne.NodeID)
	assert.Equal(t, "Only Node", ne.NodeName)
	assert.Equal(t, execType, ne.NodeType)
	assert.Equal(t, models.NodeExecutionStatusCompleted, ne.Status)
	assert.NotNil(t, ne.Output)
	assert.Equal(t, true, ne.Output["processed"])
	assert.False(t, ne.StartedAt.IsZero(), "StartedAt should be set")
	assert.NotNil(t, ne.CompletedAt, "CompletedAt should be set")
}

func TestEphemeralExecution_Sync_ShouldRecordErrorOnNodeExecution_WhenNodeFails(t *testing.T) {
	// Arrange
	const execType = "test"
	mockExec := newMockExecutorFunc(nil, fmt.Errorf("database connection lost"))
	em := newEphemeralTestManager(execType, mockExec)

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: buildSingleNodeWorkflow(execType),
	}

	// Act
	execution, execErr := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.NotNil(t, execution)
	require.Error(t, execErr)
	assert.Equal(t, models.ExecutionStatusFailed, execution.Status)

	// The failed node should have its error recorded
	require.Len(t, execution.NodeExecutions, 1)
	ne := execution.NodeExecutions[0]
	assert.Equal(t, models.NodeExecutionStatusFailed, ne.Status)
	assert.Contains(t, ne.Error, "database connection lost")
}

func TestEphemeralExecution_Sync_ShouldSetStrictMode_WhenOptsStrictModeTrue(t *testing.T) {
	// Arrange
	const execType = "test"
	mockExec := newMockExecutorFunc(map[string]any{"ok": true}, nil)
	em := newEphemeralTestManager(execType, mockExec)

	opts := &EphemeralExecutionOptions{
		Mode:       "sync",
		Workflow:   buildSingleNodeWorkflow(execType),
		StrictMode: true,
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, execution)
	assert.True(t, execution.StrictMode)
}

func TestEphemeralExecution_Sync_ShouldUseParallelBranches_WhenWorkflowHasFork(t *testing.T) {
	// Arrange: root -> branchA, root -> branchB (parallel)
	const execType = "test"
	mockExec := newMockExecutorFunc(map[string]any{"result": "done"}, nil)
	em := newEphemeralTestManager(execType, mockExec)

	wf := &models.Workflow{
		ID:   "wf-parallel",
		Name: "Parallel Branches",
		Nodes: []*models.Node{
			{ID: "root", Name: "Root", Type: execType, Config: map[string]any{}},
			{ID: "branchA", Name: "Branch A", Type: execType, Config: map[string]any{}},
			{ID: "branchB", Name: "Branch B", Type: execType, Config: map[string]any{}},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "root", To: "branchA"},
			{ID: "e2", From: "root", To: "branchB"},
		},
	}

	opts := &EphemeralExecutionOptions{
		Mode:     "sync",
		Workflow: wf,
	}

	// Act
	execution, err := em.ExecuteEphemeral(context.Background(), opts)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, execution)
	assert.Equal(t, models.ExecutionStatusCompleted, execution.Status)
	require.Len(t, execution.NodeExecutions, 3)

	// All nodes should be completed
	for _, ne := range execution.NodeExecutions {
		assert.Equal(t, models.NodeExecutionStatusCompleted, ne.Status,
			"node %s should be completed", ne.NodeID)
	}

	// Output should merge from both leaf nodes
	require.NotNil(t, execution.Output)
	assert.NotNil(t, execution.Output["branchA"])
	assert.NotNil(t, execution.Output["branchB"])
}
