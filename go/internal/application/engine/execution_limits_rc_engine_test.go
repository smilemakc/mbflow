package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetMaxConcurrentExecutions(t *testing.T) {
	em := &ExecutionManager{}
	assert.Zero(t, em.MaxConcurrentExecutions(), "unlimited by default")
	assert.Zero(t, em.RunningExecutions())

	em.SetMaxConcurrentExecutions(3)
	assert.Equal(t, 3, em.MaxConcurrentExecutions())

	release, err := em.acquireSlot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, em.RunningExecutions())
	release()
	assert.Zero(t, em.RunningExecutions())

	em.SetMaxConcurrentExecutions(0)
	assert.Zero(t, em.MaxConcurrentExecutions())
	em.SetMaxConcurrentExecutions(-5)
	assert.Zero(t, em.MaxConcurrentExecutions())
}

func TestAcquireSlot_WaitsAndHonorsContextAndShutdown(t *testing.T) {
	em := &ExecutionManager{}
	em.SetMaxConcurrentExecutions(1)

	release, err := em.acquireSlot(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = em.acquireSlot(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	got := make(chan error, 1)
	go func() {
		r, err := em.acquireSlot(context.Background())
		if err == nil {
			r()
		}
		got <- err
	}()
	release()
	require.NoError(t, <-got, "a waiter gets the slot once it is released")

	// Shutdown wakes waiters with ErrShuttingDown.
	release, err = em.acquireSlot(context.Background())
	require.NoError(t, err)
	defer release()
	go func() {
		_, err := em.acquireSlot(context.Background())
		got <- err
	}()
	require.NoError(t, em.Shutdown(context.Background()))
	require.ErrorIs(t, <-got, ErrShuttingDown)
}

func TestExecuteAsync_QueuesBeyondLimit(t *testing.T) {
	block := newBlockingExecutor()
	f := newStoredFixture(t, block, newFakeExecutionRepo())
	f.em.SetMaxConcurrentExecutions(1)

	first, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)

	second, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)

	// The second execution must stay pending while the first holds the slot.
	select {
	case id := <-block.started:
		t.Fatalf("node %s started although the concurrency limit is reached", id)
	case <-time.After(50 * time.Millisecond):
	}
	assert.Equal(t, "pending", f.execRepo.status(second.ID))
	assert.Equal(t, 1, f.em.RunningExecutions())
	assert.Len(t, f.em.active.list(), 2, "queued executions keep their lease")

	close(block.release)
	eventually(t, func() bool {
		return f.execRepo.status(first.ID) == "completed" && f.execRepo.status(second.ID) == "completed"
	}, "both executions complete once slots free up")
}

func TestExecute_Sync_WaitsForSlotAndHonorsContext(t *testing.T) {
	block := newBlockingExecutor()
	f := newStoredFixture(t, block, newFakeExecutionRepo())
	f.em.SetMaxConcurrentExecutions(1)

	_, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	exec, err := f.em.Execute(ctx, f.wf.ID.String(), nil, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorContains(t, err, "waiting for an execution slot")
	assert.Nil(t, exec)
	assert.Equal(t, 1, f.execRepo.count(), "no record is created for a sync execution that never started")

	close(block.release)
}

func TestShutdown_CancelsQueuedExecutions(t *testing.T) {
	block := newBlockingExecutor()
	f := newStoredFixture(t, block, newFakeExecutionRepo())
	f.em.SetMaxConcurrentExecutions(1)

	running, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)
	queued, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	require.Error(t, f.em.Shutdown(ctx), "shutdown deadline expires with work in flight")

	row, _ := f.execRepo.get(queued.ID)
	assert.Equal(t, "cancelled", row.Status)
	assert.Contains(t, row.Error, "cancelled before it started")
	assert.NotNil(t, row.CompletedAt)
	assert.Equal(t, "failed", f.execRepo.status(running.ID), "the running execution is interrupted")
}

func TestEphemeral_ConcurrencyLimit(t *testing.T) {
	block := newBlockingExecutor()
	em := newEphemeralTestManager("block", block)
	em.ephemeralRegistry = NewEphemeralStreamRegistry(time.Minute)
	em.SetMaxConcurrentExecutions(1)

	first, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{Mode: "async", Workflow: buildSingleNodeWorkflow("block")})
	require.NoError(t, err)
	waitStarted(t, block)

	queued, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{Mode: "async", Workflow: buildSingleNodeWorkflow("block")})
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusPending, queued.Status)

	// A sync ephemeral execution waits for a slot and gives up with its ctx.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = em.ExecuteEphemeral(ctx, &EphemeralExecutionOptions{Mode: "sync", Workflow: buildSingleNodeWorkflow("block")})
	require.ErrorIs(t, err, context.DeadlineExceeded)

	sctx, scancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer scancel()
	require.Error(t, em.Shutdown(sctx))

	assert.True(t, em.IsEphemeralExecutionTerminal(first.ID))
	assert.True(t, em.IsEphemeralExecutionTerminal(queued.ID), "a queued execution cancelled at shutdown is terminal")
}

func TestEphemeral_Persisted(t *testing.T) {
	user := uuid.New()

	t.Run("sync", func(t *testing.T) {
		repo := newFakeExecutionRepo()
		em := newEphemeralTestManager("echo", echoExecutor())
		em.executionRepo = repo

		exec, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{
			Mode: "sync", PersistExecution: true, CreatedBy: &user, Workflow: buildTestWorkflow("echo"),
		})
		require.NoError(t, err)
		row, ok := repo.get(exec.ID)
		require.True(t, ok)
		assert.Equal(t, "completed", row.Status)
		assert.Equal(t, []string{"pending", "running", "completed"}, repo.statusHistory(exec.ID))
		assert.Empty(t, em.active.list())
	})

	t.Run("async", func(t *testing.T) {
		repo := newFakeExecutionRepo()
		em := newEphemeralTestManager("echo", echoExecutor())
		em.executionRepo = repo

		exec, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{
			Mode: "async", PersistExecution: true, CreatedBy: &user, Workflow: buildTestWorkflow("echo"),
		})
		require.NoError(t, err)
		require.NoError(t, em.Shutdown(context.Background()))

		row, ok := repo.get(exec.ID)
		require.True(t, ok)
		assert.Equal(t, "completed", row.Status)
		assert.Equal(t, "inline", row.WorkflowSource)
		assert.Empty(t, em.active.list())
	})

	errCases := []struct {
		name     string
		mode     string
		setup    func(*fakeExecutionRepo)
		wantErr  string
		asyncErr bool
	}{
		{name: "sync create", mode: "sync", setup: func(r *fakeExecutionRepo) { r.createErr = errors.New("db") }, wantErr: "failed to persist ephemeral execution"},
		{name: "sync running update", mode: "sync", setup: func(r *fakeExecutionRepo) { r.failNext = []error{errors.New("db")} }, wantErr: "failed to persist ephemeral execution status"},
		{name: "sync final update", mode: "sync", setup: func(r *fakeExecutionRepo) { r.failNext = []error{nil, errors.New("db")} }, wantErr: "failed to persist ephemeral execution"},
		{name: "async create", mode: "async", setup: func(r *fakeExecutionRepo) { r.createErr = errors.New("db") }, wantErr: "failed to persist ephemeral execution"},
		{name: "async running update", mode: "async", setup: func(r *fakeExecutionRepo) { r.failNext = []error{errors.New("db")} }, asyncErr: true},
		{name: "async final update", mode: "async", setup: func(r *fakeExecutionRepo) { r.failNext = []error{nil, errors.New("db")} }, asyncErr: true},
	}
	for _, tc := range errCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeExecutionRepo()
			tc.setup(repo)
			em := newEphemeralTestManager("echo", echoExecutor())
			em.executionRepo = repo
			em.ephemeralRegistry = NewEphemeralStreamRegistry(time.Minute)

			exec, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{
				Mode: tc.mode, PersistExecution: true, Workflow: buildTestWorkflow("echo"),
			})
			if !tc.asyncErr {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.NoError(t, em.Shutdown(context.Background()))
			assert.True(t, em.IsEphemeralExecutionTerminal(exec.ID), "failed persistence still ends the stream")
		})
	}
}
