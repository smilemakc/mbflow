package trigger

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// executeCall records one WorkflowExecutor.Execute invocation.
type executeCall struct {
	WorkflowID string
	Input      map[string]any
	TriggerID  string
}

// fakeExecutor is a WorkflowExecutor that records calls.
type fakeExecutor struct {
	mu    sync.Mutex
	calls []executeCall
	err   error
}

func (f *fakeExecutor) Execute(_ context.Context, workflowID string, input map[string]any, opts *engine.ExecutionOptions) (*models.Execution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	call := executeCall{WorkflowID: workflowID, Input: input}
	if opts != nil {
		call.TriggerID = opts.TriggerID
	}
	f.calls = append(f.calls, call)
	if f.err != nil {
		return nil, f.err
	}
	return &models.Execution{ID: "exec-" + workflowID}, nil
}

func (f *fakeExecutor) Calls() []executeCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]executeCall(nil), f.calls...)
}

func (f *fakeExecutor) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

var _ WorkflowExecutor = (*fakeExecutor)(nil)
var _ WorkflowExecutor = (*engine.ExecutionManager)(nil)

func TestClaimKeys(t *testing.T) {
	tick := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	assert.Equal(t, "mbflow:trigger:claim:t1:tick:1767323045", cronClaimKey("t1", tick))
	assert.Equal(t, "mbflow:trigger:claim:t1:event:e1", eventClaimKey("t1", "e1"))

	// The tick key is the same whatever the location of the time value.
	moscow, err := time.LoadLocation("Europe/Moscow")
	require.NoError(t, err)
	assert.Equal(t, cronClaimKey("t1", tick), cronClaimKey("t1", tick.In(moscow)))
}

func TestEventDedupID(t *testing.T) {
	assert.Equal(t, "evt-1", eventDedupID(Event{ID: "evt-1"}, `{"id":"evt-1"}`))

	// Without an ID the raw payload identifies the delivery.
	a := eventDedupID(Event{}, `{"type":"x","data":{"n":1}}`)
	b := eventDedupID(Event{}, `{"type":"x","data":{"n":1}}`)
	c := eventDedupID(Event{}, `{"type":"x","data":{"n":2}}`)
	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
	assert.Contains(t, a, "sha256-")
}

func TestClaimOccurrence(t *testing.T) {
	ctx := context.Background()

	t.Run("nil cache always fires", func(t *testing.T) {
		assert.True(t, claimOccurrence(ctx, nil, "k"))
		assert.True(t, claimOccurrence(ctx, nil, "k"))
	})

	t.Run("exactly one replica wins a key", func(t *testing.T) {
		rc, mr := setupTestRedisServer(t)

		var wins int
		var mu sync.Mutex
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if claimOccurrence(ctx, rc, "mbflow:trigger:claim:x") {
					mu.Lock()
					wins++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		assert.Equal(t, 1, wins)

		assert.Equal(t, claimTTL, mr.TTL("mbflow:trigger:claim:x"))
		val, err := mr.Get("mbflow:trigger:claim:x")
		require.NoError(t, err)
		assert.Equal(t, replicaID, val)

		// A different occurrence is claimed independently.
		assert.True(t, claimOccurrence(ctx, rc, "mbflow:trigger:claim:y"))
	})

	t.Run("redis error fails open", func(t *testing.T) {
		rc, mr := setupTestRedisServer(t)
		mr.SetError("ERR redis unavailable")
		assert.True(t, claimOccurrence(ctx, rc, "k"))
	})
}

func TestNewManager_TypedNilExecutionManager(t *testing.T) {
	var em *engine.ExecutionManager
	_, err := NewManager(ManagerConfig{
		TriggerRepo:  &mockTriggerRepo{},
		WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: em,
		Cache:        setupTestRedis(t),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execution manager is required")
}

func TestNewManager_MissingCache(t *testing.T) {
	_, err := NewManager(ManagerConfig{
		TriggerRepo:  &mockTriggerRepo{},
		WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: &fakeExecutor{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis cache is required")
}

var errExec = errors.New("boom")
