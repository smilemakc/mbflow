package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/observer"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// onlyActive waits until exactly one execution is tracked and returns it.
func onlyActive(t *testing.T, em *ExecutionManager) string {
	t.Helper()
	eventually(t, func() bool { return len(em.active.list()) == 1 }, "one execution is tracked")
	return em.active.list()[0].String()
}

func TestCancelExecution_LocalRunningAsync(t *testing.T) {
	block := newBlockingExecutor()
	f := newStoredFixture(t, block, newFakeLeaseRepo())

	exec, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), map[string]any{"in": 1}, nil)
	require.NoError(t, err)
	waitStarted(t, block)

	outcome, err := f.em.CancelExecution(context.Background(), exec.ID)
	require.NoError(t, err)
	assert.Equal(t, CancelFinished, outcome)

	row, ok := f.execRepo.get(exec.ID)
	require.True(t, ok)
	assert.Equal(t, "cancelled", row.Status)
	assert.Equal(t, CancelledExecutionError, row.Error)
	assert.NotNil(t, row.CompletedAt)
	assert.NotEmpty(t, row.NodeExecutions, "node executions are recorded")
	assert.Empty(t, f.em.active.list())
	eventually(t, func() bool {
		return contains(f.recorder.types(exec.ID), observer.EventTypeExecutionCancelled)
	}, "a cancelled event is emitted")
	assert.NotContains(t, f.recorder.types(exec.ID), observer.EventTypeExecutionFailed)

	// A finished execution is not cancellable any more.
	_, err = f.em.CancelExecution(context.Background(), exec.ID)
	require.ErrorIs(t, err, ErrExecutionNotCancellable)
}

func TestCancelExecution_LocalQueued(t *testing.T) {
	block := newBlockingExecutor()
	f := newStoredFixture(t, block, newFakeExecutionRepo())
	f.em.SetMaxConcurrentExecutions(1)

	running, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)
	queued, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)

	outcome, err := f.em.CancelExecution(context.Background(), queued.ID)
	require.NoError(t, err)
	assert.Equal(t, CancelFinished, outcome)

	row, _ := f.execRepo.get(queued.ID)
	assert.Equal(t, "cancelled", row.Status)
	assert.Equal(t, CancelledExecutionError, row.Error)
	assert.Equal(t, []string{"pending", "cancelled"}, f.execRepo.statusHistory(queued.ID), "it never ran")
	eventually(t, func() bool {
		return contains(f.recorder.types(queued.ID), observer.EventTypeExecutionCancelled)
	}, "a cancelled event is emitted")

	close(block.release)
	eventually(t, func() bool { return f.execRepo.status(running.ID) == "completed" }, "the running execution is unaffected")
}

func TestCancelExecution_LocalSync(t *testing.T) {
	block := newBlockingExecutor()
	f := newStoredFixture(t, block, newFakeExecutionRepo())

	type result struct {
		exec *models.Execution
		err  error
	}
	done := make(chan result, 1)
	go func() {
		exec, err := f.em.Execute(context.Background(), f.wf.ID.String(), nil, nil)
		done <- result{exec, err}
	}()
	waitStarted(t, block)
	id := onlyActive(t, f.em)

	outcome, err := f.em.CancelExecution(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, CancelFinished, outcome)

	res := <-done
	require.ErrorIs(t, res.err, context.Canceled)
	assert.Equal(t, models.ExecutionStatusCancelled, res.exec.Status)
	assert.Equal(t, "cancelled", f.execRepo.status(id))
}

func TestCancelExecution_EphemeralPersisted(t *testing.T) {
	for _, mode := range []string{"sync", "async"} {
		t.Run(mode, func(t *testing.T) {
			block := newBlockingExecutor()
			repo := newFakeExecutionRepo()
			em := newEphemeralTestManager("block", block)
			em.executionRepo = repo
			recorder := &recordingObserver{}
			em.observerManager = observer.NewObserverManager()
			require.NoError(t, em.observerManager.Register(recorder))

			done := make(chan error, 1)
			go func() {
				_, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{
					Mode: mode, PersistExecution: true, Workflow: buildSingleNodeWorkflow("block"),
				})
				done <- err
			}()
			waitStarted(t, block)
			id := onlyActive(t, em)

			outcome, err := em.CancelExecution(context.Background(), id)
			require.NoError(t, err)
			assert.Equal(t, CancelFinished, outcome)
			if mode == "sync" {
				require.Error(t, <-done)
			} else {
				require.NoError(t, <-done)
			}

			row, _ := repo.get(id)
			assert.Equal(t, "cancelled", row.Status)
			assert.Equal(t, CancelledExecutionError, row.Error)
			assert.NotEmpty(t, row.NodeExecutions)
			eventually(t, func() bool {
				return contains(recorder.types(id), observer.EventTypeExecutionCancelled)
			}, "a cancelled event is emitted")
			require.NoError(t, em.Shutdown(context.Background()))
		})
	}
}

func TestCancelExecution_Remote(t *testing.T) {
	lease := newFakeLeaseRepo()
	f := newStoredFixture(t, echoExecutor(), lease)

	// Executions owned by another replica exist only in the database.
	remote := uuid.New()
	finished := uuid.New()
	require.NoError(t, lease.Create(context.Background(), &storagemodels.ExecutionModel{ID: remote, Status: "running"}))
	require.NoError(t, lease.Create(context.Background(), &storagemodels.ExecutionModel{ID: finished, Status: "completed"}))

	outcome, err := f.em.CancelExecution(context.Background(), remote.String())
	require.NoError(t, err)
	assert.Equal(t, CancelPending, outcome)
	row, _ := lease.get(remote.String())
	assert.Equal(t, "cancelled", row.Status)
	assert.Equal(t, CancelledExecutionError, row.Error)

	_, err = f.em.CancelExecution(context.Background(), finished.String())
	require.ErrorIs(t, err, ErrExecutionNotCancellable)
	_, err = f.em.CancelExecution(context.Background(), uuid.NewString())
	require.ErrorIs(t, err, ErrExecutionNotCancellable)
	_, err = f.em.CancelExecution(context.Background(), "not-a-uuid")
	require.ErrorContains(t, err, "invalid execution ID")

	lease.cancelErr = errors.New("db down")
	_, err = f.em.CancelExecution(context.Background(), uuid.NewString())
	require.ErrorContains(t, err, "db down")
}

func TestCancelExecution_RemoteUnsupported(t *testing.T) {
	f := newStoredFixture(t, echoExecutor(), newFakeExecutionRepo())
	_, err := f.em.CancelExecution(context.Background(), uuid.NewString())
	require.ErrorIs(t, err, ErrCancelUnsupported)
}

func TestCancelExecution_HeartbeatStopsRemotelyCancelled(t *testing.T) {
	block := newBlockingExecutor()
	lease := newFakeLeaseRepo()
	f := newStoredFixture(t, block, lease)
	require.NoError(t, f.em.StartRecovery(context.Background(), RecoveryConfig{HeartbeatInterval: 5 * time.Millisecond}))

	exec, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)

	// Another replica cancels it in the database.
	changed, err := lease.CancelExecution(context.Background(), uuid.MustParse(exec.ID), CancelledExecutionError)
	require.NoError(t, err)
	require.True(t, changed)

	eventually(t, func() bool { return len(f.em.active.list()) == 0 }, "the heartbeat stops the execution")
	row, _ := lease.get(exec.ID)
	assert.Equal(t, "cancelled", row.Status)
	assert.NotEmpty(t, row.NodeExecutions, "the owner records node executions")
	eventually(t, func() bool {
		return contains(f.recorder.types(exec.ID), observer.EventTypeExecutionCancelled)
	}, "the owner emits the cancelled event")
}

func TestHeartbeat_SkipsCancelCheckWhenAllActiveAndSurvivesErrors(t *testing.T) {
	block := newBlockingExecutor()
	lease := newFakeLeaseRepo()
	f := newStoredFixture(t, block, lease)

	exec, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)

	f.em.heartbeat(context.Background(), lease)
	assert.Zero(t, lease.cancelCheckCount(), "no cancel check while every execution is still running")

	_, err = lease.CancelExecution(context.Background(), uuid.MustParse(exec.ID), CancelledExecutionError)
	require.NoError(t, err)
	lease.cancelledErr = errors.New("query failed")
	f.em.heartbeat(context.Background(), lease)
	assert.Equal(t, 1, lease.cancelCheckCount())
	assert.Len(t, f.em.active.list(), 1, "nothing is cancelled when the check fails")

	lease.cancelledErr = nil
	f.em.heartbeat(context.Background(), lease)
	eventually(t, func() bool { return len(f.em.active.list()) == 0 }, "the next heartbeat cancels it")
}

func TestCancelExecution_FinishedBeforeNoticingRemoteCancel(t *testing.T) {
	block := newBlockingExecutor()
	lease := newFakeLeaseRepo()
	f := newStoredFixture(t, block, lease)

	exec, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)

	// Cancelled elsewhere; without heartbeats the owner runs it to the end.
	_, err = lease.CancelExecution(context.Background(), uuid.MustParse(exec.ID), CancelledExecutionError)
	require.NoError(t, err)
	close(block.release)

	eventually(t, func() bool { return len(f.em.active.list()) == 0 }, "the execution finishes")
	row, _ := lease.get(exec.ID)
	assert.Equal(t, "cancelled", row.Status, "cancellation is sticky")
	assert.Equal(t, CancelledExecutionError, row.Error)
	assert.Nil(t, row.OutputData)
	assert.NotEmpty(t, row.NodeExecutions)
	eventually(t, func() bool {
		return contains(f.recorder.types(exec.ID), observer.EventTypeExecutionCancelled)
	}, "a cancelled event is emitted")
	assert.NotContains(t, f.recorder.types(exec.ID), observer.EventTypeExecutionCompleted)
}

func TestCancelExecution_QueuedRemoteCancelNeverRuns(t *testing.T) {
	block := newBlockingExecutor()
	lease := newFakeLeaseRepo()
	f := newStoredFixture(t, block, lease)
	f.em.SetMaxConcurrentExecutions(1)

	running, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)
	queued, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)

	_, err = lease.CancelExecution(context.Background(), uuid.MustParse(queued.ID), CancelledExecutionError)
	require.NoError(t, err)
	close(block.release)

	eventually(t, func() bool { return len(f.em.active.list()) == 0 }, "both executions finish")
	assert.Equal(t, "completed", lease.status(running.ID))
	assert.Equal(t, []string{"pending", "cancelled", "cancelled"}, lease.statusHistory(queued.ID), "the queued execution never ran")
}

// cancelOnCreateRepo cancels every execution right after creating it, as if
// another replica cancelled it immediately.
type cancelOnCreateRepo struct {
	*fakeLeaseRepo
}

func (r *cancelOnCreateRepo) Create(ctx context.Context, m *storagemodels.ExecutionModel) error {
	if err := r.fakeLeaseRepo.Create(ctx, m); err != nil {
		return err
	}
	_, err := r.CancelExecution(ctx, m.ID, CancelledExecutionError)
	return err
}

func TestCancelExecution_EphemeralCancelledBeforeRunning(t *testing.T) {
	for _, mode := range []string{"sync", "async"} {
		t.Run(mode, func(t *testing.T) {
			ran := make(chan struct{}, 1)
			exec := &executor.ExecutorFunc{ExecuteFn: func(context.Context, map[string]any, any) (any, error) {
				ran <- struct{}{}
				return map[string]any{}, nil
			}}
			repo := &cancelOnCreateRepo{newFakeLeaseRepo()}
			em := newEphemeralTestManager("x", exec)
			em.executionRepo = repo

			got, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{
				Mode: mode, PersistExecution: true, Workflow: buildSingleNodeWorkflow("x"),
			})
			if mode == "sync" {
				require.ErrorIs(t, err, models.ErrExecutionCancelled)
				assert.Equal(t, models.ExecutionStatusCancelled, got.Status)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, em.Shutdown(context.Background()))

			row, _ := repo.get(got.ID)
			assert.Equal(t, "cancelled", row.Status)
			assert.Equal(t, CancelledExecutionError, row.Error)
			select {
			case <-ran:
				t.Fatal("a cancelled execution must not run")
			default:
			}
		})
	}
}

func TestCancelExecution_WaitBounded(t *testing.T) {
	// The node ignores cancellation until released.
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	stubborn := &executor.ExecutorFunc{ExecuteFn: func(context.Context, map[string]any, any) (any, error) {
		started <- struct{}{}
		<-release
		return map[string]any{}, nil
	}}
	f := newStoredFixture(t, stubborn, newFakeExecutionRepo())
	defer close(release)

	old := cancelWait
	cancelWait = 20 * time.Millisecond
	defer func() { cancelWait = old }()

	exec, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	<-started

	outcome, err := f.em.CancelExecution(context.Background(), exec.ID)
	require.NoError(t, err)
	assert.Equal(t, CancelPending, outcome, "gives up waiting after cancelWait")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outcome, err = f.em.CancelExecution(ctx, exec.ID)
	require.NoError(t, err)
	assert.Equal(t, CancelPending, outcome, "gives up waiting when the request ends")
}

func TestActiveExecutions_Cancel(t *testing.T) {
	var a activeExecutions
	assert.Nil(t, a.cancel(uuid.New()), "unknown executions are not cancelled")

	id := uuid.NewString()
	untrack := a.add(id, nil)
	done := a.cancel(uuid.MustParse(id))
	require.NotNil(t, done, "an execution without a cancel func is still reported")

	// Re-adding the same ID replaces the entry; the stale untrack leaves it.
	untrack2 := a.add(id, nil)
	untrack()
	assert.Len(t, a.list(), 1)
	select {
	case <-done:
	default:
		t.Fatal("untrack closes the done channel")
	}
	untrack2()
	assert.Empty(t, a.list())
}

func TestCancellationReason(t *testing.T) {
	cancelled, _ := cancellationReason(nil) //nolint:staticcheck // a nil context must be handled
	assert.False(t, cancelled)
	cancelled, _ = cancellationReason(context.Background())
	assert.False(t, cancelled)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled, _ = cancellationReason(ctx)
	assert.False(t, cancelled, "a plain context cancellation is not a cancel request")

	ctx, cancelCause := context.WithCancelCause(context.Background())
	cancelCause(errCancelRequested)
	cancelled, msg := cancellationReason(ctx)
	assert.True(t, cancelled)
	assert.Equal(t, CancelledExecutionError, msg)
	assert.Equal(t, errCancelRequested, queuedCancelCause(ctx, context.Canceled))

	ctx, cancelCause = context.WithCancelCause(context.Background())
	cancelCause(ErrShuttingDown)
	cancelled, msg = cancellationReason(ctx)
	assert.True(t, cancelled)
	assert.Equal(t, ShutdownCancelledError, msg)
	assert.Equal(t, ErrShuttingDown, queuedCancelCause(ctx, ErrShuttingDown))
}

func TestPersistFinalState_PropagatesErrors(t *testing.T) {
	repo := newFakeLeaseRepo()
	em := &ExecutionManager{executionRepo: repo}
	exec := &models.Execution{ID: uuid.NewString(), Status: models.ExecutionStatusCompleted}
	repo.failNext = []error{errors.New("write failed")}
	require.ErrorContains(t, em.persistFinalState(context.Background(), exec), "write failed")
	assert.Equal(t, models.ExecutionStatusCompleted, exec.Status)
}

func TestExecute_StoresMetadata(t *testing.T) {
	f := newStoredFixture(t, echoExecutor(), newFakeExecutionRepo())
	opts := DefaultExecutionOptions()
	opts.Metadata = map[string]any{"retry_of": "abc"}

	exec, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, opts)
	require.NoError(t, err)
	assert.Equal(t, "abc", exec.Metadata["retry_of"])
	row, _ := f.execRepo.get(exec.ID)
	assert.Equal(t, "abc", row.Metadata["retry_of"])
}
