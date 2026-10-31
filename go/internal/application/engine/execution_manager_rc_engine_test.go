package engine

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/observer"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgengine "github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func echoExecutor() executor.Executor {
	return &executor.ExecutorFunc{ExecuteFn: func(_ context.Context, config map[string]any, _ any) (any, error) {
		return map[string]any{"from": config["nodeID"]}, nil
	}}
}

func TestExecute_Stored_Success(t *testing.T) {
	f := newStoredFixture(t, echoExecutor(), newFakeExecutionRepo())

	opts := DefaultExecutionOptions()
	opts.Variables = map[string]any{"run": "r"}
	opts.TriggerID = uuid.NewString()

	exec, err := f.em.Execute(context.Background(), f.wf.ID.String(), map[string]any{"in": 1}, opts)
	require.NoError(t, err)

	assert.Equal(t, models.ExecutionStatusCompleted, exec.Status)
	assert.Equal(t, "stored", exec.WorkflowSource)
	assert.Equal(t, map[string]any{"from": "n2"}, exec.Output)
	assert.Equal(t, map[string]any{"wfvar": "w", "run": "r"}, exec.Variables)
	require.Len(t, exec.NodeExecutions, 2)
	for i, ne := range exec.NodeExecutions {
		assert.Equal(t, f.wf.Nodes[i].ID.String(), ne.NodeID, "node executions reference node UUIDs")
		assert.Equal(t, models.NodeExecutionStatusCompleted, ne.Status)
		assert.NotNil(t, ne.CompletedAt)
		assert.NotNil(t, ne.Output)
	}

	row, ok := f.execRepo.get(exec.ID)
	require.True(t, ok)
	assert.Equal(t, "completed", row.Status)
	assert.Equal(t, []string{"running", "completed"}, f.execRepo.statusHistory(exec.ID))

	eventually(t, func() bool {
		types := f.recorder.types(exec.ID)
		return contains(types, observer.EventTypeExecutionStarted) && contains(types, observer.EventTypeExecutionCompleted)
	}, "started and completed events are emitted")
	assert.Zero(t, f.em.RunningExecutions())
	assert.Empty(t, f.em.active.list(), "finished executions are no longer leased")
}

func contains(types []observer.EventType, want observer.EventType) bool {
	for _, t := range types {
		if t == want {
			return true
		}
	}
	return false
}

func TestExecute_Stored_NodeFailure(t *testing.T) {
	failing := &executor.ExecutorFunc{ExecuteFn: func(context.Context, map[string]any, any) (any, error) {
		return nil, errors.New("node broke")
	}}
	f := newStoredFixture(t, failing, newFakeExecutionRepo())

	exec, err := f.em.Execute(context.Background(), f.wf.ID.String(), nil, nil)
	require.Error(t, err)
	require.NotNil(t, exec)
	assert.Equal(t, models.ExecutionStatusFailed, exec.Status)
	assert.Contains(t, exec.Error, "node broke")
	assert.Equal(t, "failed", f.execRepo.status(exec.ID))
	eventually(t, func() bool {
		return contains(f.recorder.types(exec.ID), observer.EventTypeExecutionFailed)
	}, "failed event is emitted")
}

func TestExecute_Stored_PrepareErrors(t *testing.T) {
	t.Run("invalid workflow id", func(t *testing.T) {
		f := newStoredFixture(t, echoExecutor(), newFakeExecutionRepo())
		_, err := f.em.Execute(context.Background(), "not-a-uuid", nil, nil)
		require.ErrorContains(t, err, "invalid workflow ID")
	})
	t.Run("workflow load error", func(t *testing.T) {
		f := newStoredFixture(t, echoExecutor(), newFakeExecutionRepo())
		_, err := f.em.ExecuteAsync(context.Background(), uuid.NewString(), nil, nil)
		require.ErrorContains(t, err, "failed to load workflow")
	})
	t.Run("create error", func(t *testing.T) {
		repo := newFakeExecutionRepo()
		repo.createErr = errors.New("db down")
		f := newStoredFixture(t, echoExecutor(), repo)
		_, err := f.em.Execute(context.Background(), f.wf.ID.String(), nil, nil)
		require.ErrorContains(t, err, "failed to create execution")
	})
	t.Run("finalize error", func(t *testing.T) {
		repo := newFakeExecutionRepo()
		repo.failNext = []error{errors.New("db down")}
		f := newStoredFixture(t, echoExecutor(), repo)
		exec, err := f.em.Execute(context.Background(), f.wf.ID.String(), nil, nil)
		require.ErrorContains(t, err, "failed to update execution")
		assert.Nil(t, exec)
	})
}

func TestExecuteAsync_Stored_Lifecycle(t *testing.T) {
	block := newBlockingExecutor()
	f := newStoredFixture(t, block, newFakeExecutionRepo())

	snapshot, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), map[string]any{"x": 1}, nil)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusPending, snapshot.Status, "callers get the pending snapshot")

	waitStarted(t, block)
	assert.Equal(t, "running", f.execRepo.status(snapshot.ID))
	assert.Len(t, f.em.active.list(), 1)

	close(block.release)
	waitStarted(t, block)
	eventually(t, func() bool { return f.execRepo.status(snapshot.ID) == "completed" }, "execution completes")
	assert.Equal(t, []string{"pending", "running", "completed"}, f.execRepo.statusHistory(snapshot.ID))
	eventually(t, func() bool { return len(f.em.active.list()) == 0 }, "lease released")
}

func TestExecuteAsync_Stored_PersistErrors(t *testing.T) {
	t.Run("running status update fails", func(t *testing.T) {
		repo := newFakeExecutionRepo()
		repo.failNext = []error{errors.New("db down")}
		f := newStoredFixture(t, echoExecutor(), repo)

		exec, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
		require.NoError(t, err)
		eventually(t, func() bool {
			return contains(f.recorder.types(exec.ID), observer.EventTypeExecutionFailed)
		}, "an execution error event is emitted")
		require.NoError(t, f.em.Shutdown(context.Background()))
		assert.Equal(t, "pending", repo.status(exec.ID))
	})
	t.Run("finalize fails", func(t *testing.T) {
		repo := newFakeExecutionRepo()
		repo.failNext = []error{nil, errors.New("db down")}
		f := newStoredFixture(t, echoExecutor(), repo)

		exec, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
		require.NoError(t, err)
		require.NoError(t, f.em.Shutdown(context.Background()))
		assert.Equal(t, "running", repo.status(exec.ID))
		eventually(t, func() bool {
			return contains(f.recorder.types(exec.ID), observer.EventTypeExecutionFailed)
		}, "an execution error event is emitted")
	})
}

func TestExecuteAsync_RecoversPanic(t *testing.T) {
	repo := newFakeExecutionRepo()
	repo.panicUpdate = true
	f := newStoredFixture(t, echoExecutor(), repo)

	exec, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	require.NoError(t, f.em.Shutdown(context.Background()))

	row, _ := repo.get(exec.ID)
	assert.Equal(t, "failed", row.Status)
	assert.Contains(t, row.Error, "execution panicked")
}

func TestExecute_Stored_Resources(t *testing.T) {
	owner := uuid.New()
	newRes := func(status models.ResourceStatus, ownerID string) *models.FileStorageResource {
		r := models.NewFileStorageResource(ownerID, "files")
		r.ID = uuid.NewString()
		r.Status = status
		return r
	}

	cases := []struct {
		name    string
		res     *models.FileStorageResource
		known   bool
		wantErr string
	}{
		{name: "active and owned", res: newRes(models.ResourceStatusActive, owner.String()), known: true},
		{name: "missing", res: newRes(models.ResourceStatusActive, owner.String()), wantErr: "failed to load resource"},
		{name: "foreign owner", res: newRes(models.ResourceStatusActive, uuid.NewString()), known: true, wantErr: "resource access denied"},
		{name: "inactive", res: newRes(models.ResourceStatusSuspended, owner.String()), known: true, wantErr: "is not active"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newStoredFixture(t, echoExecutor(), newFakeExecutionRepo())
			f.wf.CreatedBy = &owner
			f.wf.Resources = []*storagemodels.WorkflowResourceModel{{ResourceID: uuid.MustParse(tc.res.ID), Alias: "store", AccessType: "read"}}
			if tc.known {
				f.resources.resources[tc.res.ID] = tc.res
			}

			exec, err := f.em.Execute(context.Background(), f.wf.ID.String(), nil, nil)
			if tc.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, models.ExecutionStatusCompleted, exec.Status)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
			assert.Equal(t, models.ExecutionStatusFailed, exec.Status)
		})
	}
}

func TestExecute_Stored_WebhookObservers(t *testing.T) {
	received := make(chan string, 32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Get("X-Test")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := newStoredFixture(t, echoExecutor(), newFakeExecutionRepo())
	baseCount := f.om.Count()

	opts := DefaultExecutionOptions()
	opts.Webhooks = []WebhookSubscription{
		{URL: srv.URL, Events: []string{string(observer.EventTypeExecutionCompleted)}, NodeIDs: []string{"n1"}, Headers: map[string]string{"X-Test": "yes"}},
		{URL: "ftp://example.invalid/hook"}, // rejected by the SSRF pre-flight
	}

	_, err := f.em.Execute(context.Background(), f.wf.ID.String(), nil, opts)
	require.NoError(t, err)
	assert.Equal(t, baseCount, f.om.Count(), "per-execution observers are unregistered afterwards")

	names := f.em.registerWebhookObservers("exec-1", opts)
	assert.Equal(t, []string{"webhook_exec-1_0"}, names, "only the allowed URL is registered")
	assert.Equal(t, baseCount+1, f.om.Count())
	f.em.unregisterWebhookObservers(names)
	assert.Equal(t, baseCount, f.om.Count())

	// A duplicate name fails registration and is skipped.
	names = f.em.registerWebhookObservers("dup", &ExecutionOptions{Webhooks: []WebhookSubscription{{URL: srv.URL}}})
	again := f.em.registerWebhookObservers("dup", &ExecutionOptions{Webhooks: []WebhookSubscription{{URL: srv.URL}}})
	assert.Empty(t, again)
	f.em.unregisterWebhookObservers(names)

	assert.Nil(t, f.em.registerWebhookObservers("x", nil))
}

func TestExecutionManager_Accessors(t *testing.T) {
	registry := NewEphemeralStreamRegistry(time.Minute)
	om := observer.NewObserverManager()
	em := NewExecutionManager(executor.NewManager(), nil, nil, nil, nil, om, registry)
	assert.Same(t, om, em.ObserverManager())

	assert.False(t, em.HasEphemeralExecution("e1"))
	registry.Register("e1", NewEphemeralNotifier(nil, NewEventRedactor()))
	assert.True(t, em.HasEphemeralExecution("e1"))
	assert.False(t, em.IsEphemeralExecutionTerminal("e1"))
	registry.MarkTerminal("e1")
	assert.True(t, em.IsEphemeralExecutionTerminal("e1"))

	bare := NewExecutionManager(executor.NewManager(), nil, nil, nil, nil, nil)
	assert.False(t, bare.HasEphemeralExecution("e1"))
	assert.False(t, bare.IsEphemeralExecutionTerminal("e1"))
	assert.Nil(t, bare.ObserverManager())
}

func TestConvertToPkgOptions(t *testing.T) {
	assert.Equal(t, pkgengine.DefaultExecutionOptions(), convertToPkgOptions(nil))

	for strategy, want := range map[BackoffStrategy]pkgengine.BackoffStrategy{
		BackoffConstant:    pkgengine.BackoffConstant,
		BackoffLinear:      pkgengine.BackoffLinear,
		BackoffExponential: pkgengine.BackoffExponential,
	} {
		opts := &ExecutionOptions{
			Timeout:        time.Second,
			MaxParallelism: 3,
			RetryPolicy: &RetryPolicy{
				MaxAttempts: 4, InitialDelay: time.Millisecond, MaxDelay: time.Second,
				BackoffStrategy: strategy, RetryableErrors: []string{"timeout"},
			},
		}
		got := convertToPkgOptions(opts)
		assert.Equal(t, time.Second, got.Timeout)
		assert.Equal(t, 3, got.MaxParallelism)
		require.NotNil(t, got.RetryPolicy)
		assert.Equal(t, want, got.RetryPolicy.BackoffStrategy)
		assert.Equal(t, 4, got.RetryPolicy.MaxAttempts)
		assert.Equal(t, []string{"timeout"}, got.RetryPolicy.RetryOn)
	}
}

func TestConvertEphemeralToPkgOptions(t *testing.T) {
	got := convertEphemeralToPkgOptions(&EphemeralExecutionOptions{Timeout: 2 * time.Second, NodeTimeout: time.Second, StrictMode: true, ContinueOnError: true})
	assert.Equal(t, 2*time.Second, got.Timeout)
	assert.Equal(t, time.Second, got.NodeTimeout)
	assert.True(t, got.StrictMode)
	assert.True(t, got.ContinueOnError)

	def := convertEphemeralToPkgOptions(&EphemeralExecutionOptions{})
	assert.Equal(t, pkgengine.DefaultExecutionOptions().Timeout, def.Timeout)
}

func TestEphemeralStreamRegistry_StartCleanupStopsOnCancel(t *testing.T) {
	r := NewEphemeralStreamRegistry(time.Millisecond)
	r.Register("gone", NewEphemeralNotifier(nil, NewEventRedactor()))
	r.MarkTerminal("gone")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		r.StartCleanup(ctx)
		close(done)
	}()
	eventually(t, func() bool { _, ok := r.Get("gone"); return !ok }, "terminal entries are cleaned up")
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("StartCleanup did not return after cancel")
	}
}

func TestObserverNotifier_NilManager(t *testing.T) {
	NewObserverNotifier(nil).Notify(context.Background(), pkgengine.ExecutionEvent{Type: "x"})
}
