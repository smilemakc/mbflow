package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func newRegistry(t *testing.T, types map[string]executor.Executor) executor.Manager {
	t.Helper()
	m := executor.NewManager()
	for name, exec := range types {
		require.NoError(t, m.Register(name, exec))
	}
	return m
}

func echoConfigExecutor() *mockExecutor {
	return &mockExecutor{executeFn: func(_ context.Context, config map[string]any, _ any) (any, error) {
		return map[string]any{"id": config["id"]}, nil
	}}
}

func TestStandaloneExecutor_Success(t *testing.T) {
	t.Parallel()
	se := NewStandaloneExecutor(newRegistry(t, map[string]executor.Executor{"echo": echoConfigExecutor()}))

	wf := &models.Workflow{
		Name:      "fan-out",
		Variables: map[string]any{"a": 1},
		Nodes: []*models.Node{
			{ID: "root", Name: "Root", Type: "echo", Config: map[string]any{"id": "root"}},
			{ID: "left", Name: "Left", Type: "echo", Config: map[string]any{"id": "left"}},
			{ID: "right", Name: "Right", Type: "echo", Config: map[string]any{"id": "right"}},
		},
		Edges: []*models.Edge{{ID: "e1", From: "root", To: "left"}, {ID: "e2", From: "root", To: "right"}},
	}

	exec, err := se.ExecuteStandalone(context.Background(), wf, nil, &ExecutionOptions{Timeout: 5 * time.Second, Variables: map[string]any{"b": 2}})
	require.NoError(t, err)
	assert.NotEmpty(t, wf.ID, "a missing workflow ID is generated")
	assert.Equal(t, models.ExecutionStatusCompleted, exec.Status)
	assert.Equal(t, map[string]any{"a": 1, "b": 2}, exec.Variables)
	assert.Equal(t, map[string]any{}, exec.Input)
	// Several leaves: outputs are namespaced by node ID.
	assert.Equal(t, map[string]any{"left": map[string]any{"id": "left"}, "right": map[string]any{"id": "right"}}, exec.Output)
	require.Len(t, exec.NodeExecutions, 3)
	for _, ne := range exec.NodeExecutions {
		assert.Equal(t, models.NodeExecutionStatusCompleted, ne.Status)
		assert.NotNil(t, ne.Config)
		assert.NotNil(t, ne.ResolvedConfig)
		assert.False(t, ne.StartedAt.IsZero())
		assert.NotNil(t, ne.CompletedAt)
	}
}

func TestStandaloneExecutor_SingleLeafAndFailure(t *testing.T) {
	t.Parallel()
	failing := &mockExecutor{executeFn: func(context.Context, map[string]any, any) (any, error) {
		return nil, errors.New("kaput")
	}}
	se := NewStandaloneExecutor(newRegistry(t, map[string]executor.Executor{"echo": echoConfigExecutor(), "fail": failing}))

	ok, err := se.ExecuteStandalone(context.Background(), &models.Workflow{
		ID:    "wf",
		Nodes: []*models.Node{{ID: "only", Name: "Only", Type: "echo", Config: map[string]any{"id": "only"}}},
	}, map[string]any{"x": 1}, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"id": "only"}, ok.Output)

	failed, err := se.ExecuteStandalone(context.Background(), &models.Workflow{
		ID:    "wf",
		Nodes: []*models.Node{{ID: "bad", Name: "Bad", Type: "fail"}},
	}, nil, nil)
	require.Error(t, err)
	assert.Equal(t, models.ExecutionStatusFailed, failed.Status)
	assert.Contains(t, failed.Error, "kaput")
	require.Len(t, failed.NodeExecutions, 1)
	assert.Contains(t, failed.NodeExecutions[0].Error, "kaput")

	_, err = se.ExecuteStandalone(context.Background(), nil, nil, nil)
	require.Error(t, err)
}

func TestGetFinalOutputFromState_NoLeaves(t *testing.T) {
	t.Parallel()
	wf := &models.Workflow{}
	assert.Nil(t, getFinalOutputFromState(NewExecutionState("e", "w", wf, nil, nil), wf))
}

func TestSimpleConditionEvaluator(t *testing.T) {
	t.Parallel()
	e := NewSimpleConditionEvaluator()
	for _, tc := range []struct {
		cond   string
		output any
		want   bool
	}{
		{"", nil, true},
		{"true", nil, true},
		{"false", "x", false},
		{"output.ok", map[string]any{}, true},
		{"output.ok", nil, false},
	} {
		got, err := e.Evaluate(tc.cond, tc.output)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, "condition %q", tc.cond)
	}
}

func TestDAGUtils(t *testing.T) {
	t.Parallel()
	a, b, c := &models.Node{ID: "a"}, &models.Node{ID: "b"}, &models.Node{ID: "c"}
	wf := &models.Workflow{Nodes: []*models.Node{a, b, c}, Edges: []*models.Edge{{From: "a", To: "b"}}}

	assert.Equal(t, []string{"a", "b", "c"}, FlattenWaves([][]*models.Node{{a}, {b, c}}))
	assert.Nil(t, FlattenWaves(nil))
	assert.Equal(t, []*models.Node{b, c}, FindLeafNodes(wf))
	assert.Same(t, c, GetNodeByID(wf, "c"))
	assert.Nil(t, GetNodeByID(wf, "missing"))
	assert.Equal(t, -1, findNodeWave([][]*models.Node{{a}}, "missing"))
}

func TestExecutionState_Accessors(t *testing.T) {
	t.Parallel()
	es := NewExecutionState("e", "w", &models.Workflow{}, nil, nil)
	now := time.Now()

	_, ok := es.GetNodeError("n")
	assert.False(t, ok)

	es.SetNodeError("n", errors.New("bad"))
	es.SetNodeStartTime("n", now)
	es.SetNodeEndTime("n", now.Add(time.Second))
	es.SetNodeInput("n", map[string]any{"in": 1})
	es.SetNodeConfig("n", map[string]any{"c": 1})
	es.SetNodeResolvedConfig("n", map[string]any{"c": 2})
	es.SetNodeOutput("n", map[string]any{"out": "abc"})

	gotErr, ok := es.GetNodeError("n")
	assert.True(t, ok)
	assert.EqualError(t, gotErr, "bad")
	start, _ := es.GetNodeStartTime("n")
	assert.Equal(t, now, start)
	end, _ := es.GetNodeEndTime("n")
	assert.Equal(t, now.Add(time.Second), end)
	in, _ := es.GetNodeInput("n")
	assert.Equal(t, map[string]any{"in": 1}, in)
	cfg, _ := es.GetNodeConfig("n")
	assert.Equal(t, map[string]any{"c": 1}, cfg)
	rcfg, _ := es.GetNodeResolvedConfig("n")
	assert.Equal(t, map[string]any{"c": 2}, rcfg)

	assert.Positive(t, es.GetTotalMemoryUsage())
	es.ClearNodeOutput("n")
	_, ok = es.GetNodeOutput("n")
	assert.False(t, ok)
	assert.Zero(t, es.GetTotalMemoryUsage())
}

func TestToMapInterface(t *testing.T) {
	t.Parallel()
	type payload struct {
		Name string `json:"name"`
	}
	assert.Nil(t, ToMapInterface(nil))
	m := map[string]any{"k": "v"}
	assert.Equal(t, m, ToMapInterface(m))
	assert.Equal(t, map[string]any{"name": "x"}, ToMapInterface(payload{Name: "x"}))
	assert.Equal(t, map[string]any{"value": []int{1, 2}}, ToMapInterface([]int{1, 2}), "non-object JSON is wrapped")
	ch := make(chan int)
	assert.Equal(t, map[string]any{"value": ch}, ToMapInterface(ch), "unmarshalable values are wrapped")
}

func TestMergeVariables_Precedence(t *testing.T) {
	t.Parallel()
	got := MergeVariables(map[string]any{"a": 1, "b": 1}, map[string]any{"b": 2})
	assert.Equal(t, map[string]any{"a": 1, "b": 2}, got)
	assert.Equal(t, map[string]any{}, MergeVariables(nil, nil))
}

type panickingNotifier struct{}

func (panickingNotifier) Notify(context.Context, ExecutionEvent) { panic("notifier exploded") }

func TestDAGExecutor_RecoversNodeAndNotifierPanics(t *testing.T) {
	t.Parallel()
	panicky := &mockExecutor{executeFn: func(context.Context, map[string]any, any) (any, error) {
		panic("executor exploded")
	}}
	nodeExec := NewNodeExecutor(newRegistry(t, map[string]executor.Executor{"panic": panicky}))
	de := NewDAGExecutor(nodeExec, NewExprConditionEvaluator(), panickingNotifier{}, NewNilWorkflowLoader())

	wf := &models.Workflow{ID: "wf", Nodes: []*models.Node{{ID: "p", Name: "P", Type: "panic"}}}
	es := NewExecutionState("e", "wf", wf, nil, nil)

	err := de.Execute(context.Background(), es, DefaultExecutionOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "node panicked")
	nodeErr, ok := es.GetNodeError("p")
	require.True(t, ok)
	assert.Contains(t, nodeErr.Error(), "executor exploded")
	status, _ := es.GetNodeStatus("p")
	assert.Equal(t, models.NodeExecutionStatusFailed, status, "a panicked node is not left running")
	_, ok = es.GetNodeEndTime("p")
	assert.True(t, ok)
}

func TestNoOpNotifier(t *testing.T) {
	t.Parallel()
	NewNoOpNotifier().Notify(context.Background(), ExecutionEvent{Type: "x"})
}

func TestConvertRetryPolicy(t *testing.T) {
	t.Parallel()
	assert.Equal(t, NoInternalRetryPolicy(), convertRetryPolicy(nil))
	for in, want := range map[BackoffStrategy]InternalBackoffStrategy{
		BackoffConstant:    InternalBackoffConstant,
		BackoffLinear:      InternalBackoffLinear,
		BackoffExponential: InternalBackoffExponential,
	} {
		got := convertRetryPolicy(&RetryPolicy{MaxAttempts: 2, BackoffStrategy: in, RetryOn: []string{"x"}})
		assert.Equal(t, want, got.BackoffStrategy)
		assert.Equal(t, 2, got.MaxAttempts)
		assert.Equal(t, []string{"x"}, got.RetryableErrors)
	}
}

func TestToSlice(t *testing.T) {
	t.Parallel()
	_, err := toSlice(nil)
	require.Error(t, err)

	got, err := toSlice([]any{1, "a"})
	require.NoError(t, err)
	assert.Equal(t, []any{1, "a"}, got)

	got, err = toSlice([]string{"a", "b"})
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b"}, got)

	got, err = toSlice([2]int{1, 2})
	require.NoError(t, err)
	assert.Equal(t, []any{1, 2}, got)

	_, err = toSlice(map[string]any{})
	require.Error(t, err)
}

func TestParseSubWorkflowConfig_Variants(t *testing.T) {
	t.Parallel()
	_, err := parseSubWorkflowConfig(&models.Node{Config: map[string]any{}})
	require.ErrorContains(t, err, "workflow_id")
	_, err = parseSubWorkflowConfig(&models.Node{Config: map[string]any{"workflow_id": "w"}})
	require.ErrorContains(t, err, "for_each")

	cfg, err := parseSubWorkflowConfig(&models.Node{Config: map[string]any{
		"workflow_id": "w", "for_each": "input.items", "item_var": "it",
		"max_parallelism": 3, "on_error": "continue", "timeout_per_item": 250,
	}})
	require.NoError(t, err)
	assert.Equal(t, "it", cfg.ItemVar)
	assert.Equal(t, 3, cfg.MaxParallelism)
	assert.Equal(t, "continue", cfg.OnError)
	assert.Equal(t, 250*time.Millisecond, cfg.TimeoutPerItem)

	cfg, err = parseSubWorkflowConfig(&models.Node{Config: map[string]any{
		"workflow_id": "w", "for_each": "input.items", "max_parallelism": float64(2), "timeout_per_item": float64(100),
	}})
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.MaxParallelism)
	assert.Equal(t, 100*time.Millisecond, cfg.TimeoutPerItem)
	assert.Equal(t, SubWorkflowDefaultItemVar, cfg.ItemVar)
}

type tempErr struct{ temporary, timeout bool }

func (e tempErr) Error() string   { return "net" }
func (e tempErr) Temporary() bool { return e.temporary }
func (e tempErr) Timeout() bool   { return e.timeout }

type timeoutOnlyErr struct{ timeout bool }

func (e timeoutOnlyErr) Error() string { return "timeout" }
func (e timeoutOnlyErr) Timeout() bool { return e.timeout }

func TestIsRetryableError_Classification(t *testing.T) {
	t.Parallel()
	assert.False(t, IsRetryableError(nil))
	assert.False(t, IsRetryableError(context.Canceled))
	assert.False(t, IsRetryableError(context.DeadlineExceeded))
	assert.True(t, IsRetryableError(tempErr{temporary: true}))
	assert.False(t, IsRetryableError(tempErr{temporary: false, timeout: true}), "Temporary() takes precedence")
	assert.True(t, IsRetryableError(timeoutOnlyErr{timeout: true}))
	assert.False(t, IsRetryableError(timeoutOnlyErr{}))
	assert.True(t, IsRetryableError(errors.New("other")))
}
