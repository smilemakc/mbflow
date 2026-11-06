package engine

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

const testSecret = "sk-test-0123456789"

func TestSecretRedactor_NilAndEmpty(t *testing.T) {
	t.Parallel()
	var nilR *SecretRedactor
	nilR.Add("whatever")
	assert.Zero(t, nilR.Len())
	assert.Equal(t, "a sk-test-0123456789", nilR.String("a "+testSecret))
	in := map[string]any{"k": testSecret}
	assert.Equal(t, in, nilR.Map(in))
	assert.Equal(t, testSecret, nilR.Value(testSecret))
	err := errors.New(testSecret)
	assert.Same(t, err, nilR.Error(err))
	ev := ExecutionEvent{Message: testSecret}
	assert.Equal(t, ev, nilR.Event(ev))

	empty := NewSecretRedactor("", "abc") // too short: ignored
	assert.Zero(t, empty.Len())
	assert.Equal(t, "abc def", empty.String("abc def"))
	assert.Nil(t, NewSecretRedactor(testSecret).Map(nil))
}

func TestSecretRedactor_String(t *testing.T) {
	t.Parallel()
	r := NewSecretRedactor("abcd", "abcdefgh", "abcd") // duplicate ignored
	r.Add("abcd")                                      // no change
	assert.Equal(t, 2, r.Len())
	// The longer secret is replaced whole, not as "***efgh".
	assert.Equal(t, "x *** y *** z", r.String("x abcdefgh y abcd z"))
	assert.Equal(t, "", r.String(""))
	assert.Equal(t, "nothing here", r.String("nothing here"))
}

type redactProbe struct {
	Token string `json:"token"`
	N     int    `json:"n"`
}

func TestSecretRedactor_ValueDeepCopy(t *testing.T) {
	t.Parallel()
	r := NewSecretRedactor(testSecret)

	original := map[string]any{
		"plain":    "Bearer " + testSecret,
		"nested":   map[string]any{"list": []any{testSecret, 1, true, nil}},
		"strings":  []string{testSecret, "ok"},
		"smap":     map[string]string{"key": testSecret},
		"maps":     []map[string]any{{"k": testSecret}},
		"num":      3.5,
		"jsonnum":  json.Number("42"),
		"struct":   redactProbe{Token: testSecret, N: 7},
		"err":      errors.New("failed with " + testSecret),
		testSecret: "secret as key",
	}
	snapshot, err := json.Marshal(original["nested"])
	require.NoError(t, err)

	got := r.Map(original)
	data, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(data), testSecret)
	assert.Equal(t, "Bearer ***", got["plain"])
	assert.Equal(t, []any{"***", 1, true, nil}, got["nested"].(map[string]any)["list"])
	assert.Equal(t, []string{"***", "ok"}, got["strings"])
	assert.Equal(t, map[string]string{"key": "***"}, got["smap"])
	assert.Equal(t, []map[string]any{{"k": "***"}}, got["maps"])
	assert.Equal(t, 3.5, got["num"])
	assert.Equal(t, json.Number("42"), got["jsonnum"])
	assert.Equal(t, map[string]any{"token": "***", "n": float64(7)}, got["struct"])
	assert.Equal(t, "failed with ***", got["err"])
	assert.Equal(t, "secret as key", got["***"])

	// The original is untouched.
	after, err := json.Marshal(original["nested"])
	require.NoError(t, err)
	assert.Equal(t, snapshot, after)
	assert.Equal(t, "Bearer "+testSecret, original["plain"])

	assert.Equal(t, "***", r.Value(testSecret))
	assert.Nil(t, r.Value(nil))
	// Values JSON cannot encode are redacted as text.
	assert.Equal(t, "***", r.Value(unmarshalable{s: testSecret}))
}

type unmarshalable struct{ s string }

func (u unmarshalable) MarshalJSON() ([]byte, error) { return nil, errors.New("no json") }
func (u unmarshalable) String() string               { return u.s }

func TestSecretRedactor_Error(t *testing.T) {
	t.Parallel()
	r := NewSecretRedactor(testSecret)
	assert.NoError(t, r.Error(nil))

	clean := errors.New("nothing secret")
	assert.Same(t, clean, r.Error(clean), "errors without secrets are returned as is")

	base := context.DeadlineExceeded
	wrapped := errors.Join(errors.New("call with key "+testSecret+" failed"), base)
	red := r.Error(wrapped)
	assert.NotContains(t, red.Error(), testSecret)
	assert.Contains(t, red.Error(), "call with key *** failed")
	assert.ErrorIs(t, red, context.DeadlineExceeded, "redacted errors still unwrap")
}

func TestSecretRedactor_Event(t *testing.T) {
	t.Parallel()
	r := NewSecretRedactor(testSecret)
	ev := r.Event(ExecutionEvent{
		Type:      EventTypeNodeFailed,
		Error:     errors.New("bad " + testSecret),
		Output:    map[string]any{"k": testSecret},
		Message:   "msg " + testSecret,
		Input:     map[string]any{"in": testSecret},
		Variables: map[string]any{"v": testSecret},
	})
	assert.Equal(t, "bad ***", ev.Error.Error())
	assert.Equal(t, map[string]any{"k": "***"}, ev.Output)
	assert.Equal(t, "msg ***", ev.Message)
	assert.Equal(t, map[string]any{"in": "***"}, ev.Input)
	assert.Equal(t, map[string]any{"v": "***"}, ev.Variables)

	bare := r.Event(ExecutionEvent{Type: EventTypeWaveStarted})
	assert.Nil(t, bare.Output)
	assert.Nil(t, bare.Input)
	assert.Nil(t, bare.Variables)
	assert.NoError(t, bare.Error)
}

func TestSecretRedactor_Concurrent(t *testing.T) {
	t.Parallel()
	r := NewSecretRedactor()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r.Add(strings.Repeat("s", 4+i))
			_ = r.String("ssssssssssss")
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 8, r.Len())
	assert.Equal(t, "***", r.String(strings.Repeat("s", 11)))
}

// eventCollector records every notified event.
type eventCollector struct {
	mu     sync.Mutex
	events []ExecutionEvent
}

func (c *eventCollector) Notify(_ context.Context, e ExecutionEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
}

// TestDAGExecutor_SecretsReachExecutorsButNotEvents checks that a template
// referencing a credential resource resolves to the real value for the
// executor and for downstream nodes, while emitted events are redacted.
func TestDAGExecutor_SecretsReachExecutorsButNotEvents(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	seenConfig := map[string]map[string]any{}
	seenInput := map[string]any{}
	mgr := executor.NewManager()
	require.NoError(t, mgr.Register("probe", &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, config map[string]any, input any) (any, error) {
			execCtx, _ := executor.GetExecutionContext(ctx)
			mu.Lock()
			seenConfig[execCtx.NodeID] = config
			seenInput[execCtx.NodeID] = input
			mu.Unlock()
			if config["fail"] == true {
				return nil, errors.New("upstream rejected key " + config["api_key"].(string))
			}
			return map[string]any{"echo": config["api_key"]}, nil
		},
	}))

	wf := &models.Workflow{
		ID: "wf-secrets",
		Nodes: []*models.Node{
			{ID: "a", Name: "A", Type: "probe", Config: map[string]any{"api_key": "{{resource.anthropic.api_key}}"}},
			{ID: "b", Name: "B", Type: "probe", Config: map[string]any{"api_key": "{{input.echo}}"}},
			{ID: "c", Name: "C", Type: "probe", Config: map[string]any{"api_key": "{{resource.anthropic.api_key}}", "fail": true}},
		},
		Edges: []*models.Edge{{ID: "e1", From: "a", To: "b"}, {ID: "e2", From: "b", To: "c"}},
	}

	state := NewExecutionState("exec-1", wf.ID, wf, map[string]any{}, map[string]any{})
	state.Resources = map[string]any{"anthropic": map[string]any{"api_key": testSecret}}
	state.Secrets = NewSecretRedactor(testSecret)

	collector := &eventCollector{}
	de := NewDAGExecutor(NewNodeExecutor(mgr), NewExprConditionEvaluator(), collector, NewNilWorkflowLoader())
	err := de.Execute(context.Background(), state, DefaultExecutionOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), testSecret, "the engine itself returns the raw error; callers redact it")

	mu.Lock()
	assert.Equal(t, testSecret, seenConfig["a"]["api_key"], "executor gets the real secret")
	assert.Equal(t, testSecret, seenConfig["b"]["api_key"], "downstream templates see real data")
	assert.Equal(t, testSecret, seenInput["b"].(map[string]any)["echo"], "downstream input is not redacted")
	mu.Unlock()

	// State keeps real values (it feeds downstream nodes).
	out, _ := state.GetNodeOutput("a")
	assert.Equal(t, testSecret, out.(map[string]any)["echo"])
	rc, _ := state.GetNodeResolvedConfig("a")
	assert.Equal(t, testSecret, rc["api_key"])

	collector.mu.Lock()
	defer collector.mu.Unlock()
	require.NotEmpty(t, collector.events)
	var sawFailed bool
	for _, e := range collector.events {
		data, mErr := json.Marshal(map[string]any{"output": e.Output, "message": e.Message, "input": e.Input})
		require.NoError(t, mErr)
		assert.NotContains(t, string(data), testSecret, "event %s leaks the secret", e.Type)
		if e.Error != nil {
			assert.NotContains(t, e.Error.Error(), testSecret)
		}
		if e.Type == EventTypeNodeFailed {
			sawFailed = true
			assert.Contains(t, e.Error.Error(), "upstream rejected key ***")
		}
		if e.Type == EventTypeNodeCompleted && e.NodeID == "a" {
			assert.Equal(t, map[string]any{"echo": "***"}, e.Output)
		}
	}
	assert.True(t, sawFailed)
}

func TestSubWorkflowChildInheritsSecrets(t *testing.T) {
	t.Parallel()
	mgr := executor.NewManager()
	require.NoError(t, mgr.Register("probe", &executor.ExecutorFunc{
		ExecuteFn: func(_ context.Context, config map[string]any, _ any) (any, error) {
			return map[string]any{"echo": config["api_key"]}, nil
		},
	}))
	child := &models.Workflow{
		ID:    "child",
		Nodes: []*models.Node{{ID: "c1", Name: "C1", Type: "probe", Config: map[string]any{"api_key": "{{resource.k.api_key}}"}}},
	}
	parent := &models.Workflow{
		ID: "parent",
		Nodes: []*models.Node{{ID: "fan", Name: "Fan", Type: "sub_workflow", Config: map[string]any{
			"workflow_id": "child", "for_each": "input.items",
		}}},
	}
	state := NewExecutionState("exec-p", parent.ID, parent, map[string]any{"items": []any{1}}, nil)
	state.Resources = map[string]any{"k": map[string]any{"api_key": testSecret}}
	state.Secrets = NewSecretRedactor(testSecret)

	collector := &eventCollector{}
	de := NewDAGExecutor(NewNodeExecutor(mgr), NewExprConditionEvaluator(), collector, NewMockWorkflowLoader(map[string]*models.Workflow{"child": child}))
	require.NoError(t, de.Execute(context.Background(), state, DefaultExecutionOptions()))

	out, ok := state.GetNodeOutput("fan")
	require.True(t, ok)
	data, _ := json.Marshal(out)
	assert.Contains(t, string(data), testSecret, "the child workflow resolved the real secret")

	collector.mu.Lock()
	defer collector.mu.Unlock()
	for _, e := range collector.events {
		data, _ := json.Marshal(map[string]any{"output": e.Output, "message": e.Message})
		assert.NotContains(t, string(data), testSecret, "child event %s leaks the secret", e.Type)
	}
}
