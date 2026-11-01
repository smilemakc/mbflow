package serviceapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// cancelTestRepo is a small stateful execution repository with the sticky
// cancellation of the real one.
type cancelTestRepo struct {
	repository.ExecutionRepository

	mu      sync.Mutex
	rows    map[uuid.UUID]storagemodels.ExecutionModel
	findErr error
}

func newCancelTestRepo() *cancelTestRepo {
	return &cancelTestRepo{rows: map[uuid.UUID]storagemodels.ExecutionModel{}}
}

func (r *cancelTestRepo) put(m storagemodels.ExecutionModel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[m.ID] = m
}

func (r *cancelTestRepo) row(id uuid.UUID) storagemodels.ExecutionModel {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rows[id]
}

func (r *cancelTestRepo) FindByID(_ context.Context, id uuid.UUID) (*storagemodels.ExecutionModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.findErr != nil {
		return nil, r.findErr
	}
	row, ok := r.rows[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", models.ErrExecutionNotFound, id)
	}
	return &row, nil
}

func (r *cancelTestRepo) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*storagemodels.ExecutionModel, error) {
	return r.FindByID(ctx, id)
}

func (r *cancelTestRepo) Create(_ context.Context, m *storagemodels.ExecutionModel) error {
	r.put(*m)
	return nil
}

func (r *cancelTestRepo) Update(_ context.Context, m *storagemodels.ExecutionModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.rows[m.ID]; ok {
		if old.Status == "cancelled" && m.Status != "cancelled" {
			return repository.ErrExecutionAlreadyCancelled
		}
		// Like the real repository, Update leaves these columns alone.
		m.InputData, m.WorkflowID, m.TriggerID, m.Metadata = old.InputData, old.WorkflowID, old.TriggerID, old.Metadata
	}
	r.rows[m.ID] = *m
	return nil
}

// cancelLeaseRepo adds the lease methods.
type cancelLeaseRepo struct {
	*cancelTestRepo
	cancelErr  error
	onCancel   func(id uuid.UUID) // replaces the default behavior when set
	cancelHits int
}

func (r *cancelLeaseRepo) TouchExecutions(_ context.Context, ids []uuid.UUID) (int, error) {
	return len(ids), nil
}

func (r *cancelLeaseRepo) FailStaleExecutions(context.Context, time.Duration, string) ([]uuid.UUID, error) {
	return nil, nil
}

func (r *cancelLeaseRepo) CancelExecution(_ context.Context, id uuid.UUID, errMsg string) (bool, error) {
	r.cancelHits++
	if r.cancelErr != nil {
		return false, r.cancelErr
	}
	if r.onCancel != nil {
		r.onCancel(id)
		return false, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[id]
	if !ok || !isActiveExecutionStatus(row.Status) {
		return false, nil
	}
	row.Status = "cancelled"
	row.Error = errMsg
	r.rows[id] = row
	return true, nil
}

func (r *cancelLeaseRepo) CancelledExecutions(context.Context, []uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

const cancelTestType = "cancel-test"

// cancelFixture wires Operations to a real ExecutionManager.
type cancelFixture struct {
	ops     *Operations
	em      *engine.ExecutionManager
	repo    *cancelTestRepo
	wfID    uuid.UUID
	started chan struct{}
	release chan struct{}
}

// newCancelFixture registers an executor that blocks until released or
// cancelled; ignoreCancel makes it succeed once its context ends.
func newCancelFixture(t *testing.T, repo repository.ExecutionRepository, base *cancelTestRepo, ignoreCancel bool) *cancelFixture {
	t.Helper()
	f := &cancelFixture{repo: base, wfID: uuid.New(), started: make(chan struct{}, 8), release: make(chan struct{})}
	executors := executor.NewManager()
	require.NoError(t, executors.Register(cancelTestType, &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, _ map[string]any, _ any) (any, error) {
			f.started <- struct{}{}
			select {
			case <-f.release:
				return map[string]any{"ok": true}, nil
			case <-ctx.Done():
				if ignoreCancel {
					return map[string]any{"ok": true}, nil
				}
				return nil, ctx.Err()
			}
		},
	}))

	wfRepo := new(mockWorkflowRepo)
	wfRepo.On("FindByIDWithRelations", mock.Anything, f.wfID).Return(&storagemodels.WorkflowModel{
		ID: f.wfID, Name: "wf", Status: "active", Version: 1,
		Variables: storagemodels.JSONBMap{"base": "wf"},
		Nodes: []*storagemodels.NodeModel{
			{ID: uuid.New(), NodeID: "n1", WorkflowID: f.wfID, Name: "N1", Type: cancelTestType},
		},
	}, nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, mock.Anything).Return(nil, models.ErrWorkflowNotFound)

	f.em = engine.NewExecutionManager(executors, wfRepo, repo, nil, nil, nil)
	f.ops = newTestOperations(wfRepo, nil, nil, nil, nil, nil, executors)
	f.ops.ExecutionRepo = repo
	f.ops.ExecutionMgr = f.em
	t.Cleanup(func() {
		select {
		case <-f.release:
		default:
			close(f.release)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = f.em.Shutdown(ctx)
	})
	return f
}

func (f *cancelFixture) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-f.started:
	case <-time.After(5 * time.Second):
		t.Fatal("node did not start")
	}
}

func requireOpError(t *testing.T, err error, status int, code string) *OperationError {
	t.Helper()
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, status, opErr.HTTPStatus)
	assert.Equal(t, code, opErr.Code)
	return opErr
}

func TestCancelExecution_Local(t *testing.T) {
	repo := newCancelTestRepo()
	f := newCancelFixture(t, repo, repo, false)

	exec, err := f.ops.StartExecution(context.Background(), StartExecutionParams{WorkflowID: f.wfID.String()})
	require.NoError(t, err)
	f.waitStarted(t)

	res, err := f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: uuid.MustParse(exec.ID)})
	require.NoError(t, err)
	assert.False(t, res.Pending)
	assert.Equal(t, models.ExecutionStatusCancelled, res.Execution.Status)
	assert.Equal(t, engine.CancelledExecutionError, res.Execution.Error)

	// Canceling again is a conflict.
	_, err = f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: uuid.MustParse(exec.ID)})
	opErr := requireOpError(t, err, http.StatusConflict, CodeExecutionNotCancellable)
	assert.Contains(t, opErr.Message, "already cancelled")
}

func TestCancelExecution_Remote(t *testing.T) {
	base := newCancelTestRepo()
	lease := &cancelLeaseRepo{cancelTestRepo: base}
	f := newCancelFixture(t, lease, base, false)

	id := uuid.New()
	base.put(storagemodels.ExecutionModel{ID: id, Status: "running", WorkflowSource: "stored"})

	res, err := f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: id})
	require.NoError(t, err)
	assert.True(t, res.Pending, "another replica stops it on its next heartbeat")
	assert.Equal(t, models.ExecutionStatusCancelled, res.Execution.Status)
}

func TestCancelExecution_Errors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		repo := newCancelTestRepo()
		f := newCancelFixture(t, repo, repo, false)
		_, err := f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: uuid.New()})
		requireOpError(t, err, http.StatusNotFound, CodeExecutionNotFound)
	})

	t.Run("lookup failure", func(t *testing.T) {
		repo := newCancelTestRepo()
		repo.findErr = errors.New("db down")
		f := newCancelFixture(t, repo, repo, false)
		_, err := f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: uuid.New()})
		require.EqualError(t, err, "db down")
	})

	t.Run("terminal", func(t *testing.T) {
		repo := newCancelTestRepo()
		f := newCancelFixture(t, repo, repo, false)
		id := uuid.New()
		repo.put(storagemodels.ExecutionModel{ID: id, Status: "completed"})
		_, err := f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: id})
		opErr := requireOpError(t, err, http.StatusConflict, CodeExecutionNotCancellable)
		assert.Equal(t, "execution is already completed and cannot be cancelled", opErr.Message)
	})

	t.Run("no execution manager", func(t *testing.T) {
		repo := newCancelTestRepo()
		f := newCancelFixture(t, repo, repo, false)
		f.ops.ExecutionMgr = nil
		id := uuid.New()
		repo.put(storagemodels.ExecutionModel{ID: id, Status: "running"})
		_, err := f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: id})
		requireOpError(t, err, http.StatusServiceUnavailable, "EXECUTION_MANAGER_UNAVAILABLE")
	})

	t.Run("not running here without lease support", func(t *testing.T) {
		repo := newCancelTestRepo()
		f := newCancelFixture(t, repo, repo, false)
		id := uuid.New()
		repo.put(storagemodels.ExecutionModel{ID: id, Status: "running"})
		_, err := f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: id})
		opErr := requireOpError(t, err, http.StatusConflict, CodeExecutionNotCancellable)
		assert.Contains(t, opErr.Message, "not running on this server")
	})

	t.Run("finished between lookup and cancel", func(t *testing.T) {
		base := newCancelTestRepo()
		lease := &cancelLeaseRepo{cancelTestRepo: base}
		f := newCancelFixture(t, lease, base, false)
		id := uuid.New()
		base.put(storagemodels.ExecutionModel{ID: id, Status: "running"})
		lease.onCancel = func(id uuid.UUID) { base.put(storagemodels.ExecutionModel{ID: id, Status: "failed"}) }

		_, err := f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: id})
		opErr := requireOpError(t, err, http.StatusConflict, CodeExecutionNotCancellable)
		assert.Contains(t, opErr.Message, "already failed")

		base.put(storagemodels.ExecutionModel{ID: id, Status: "running"})
		lease.onCancel = func(uuid.UUID) {
			base.mu.Lock()
			base.findErr = errors.New("gone")
			base.mu.Unlock()
		}
		_, err = f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: id})
		requireOpError(t, err, http.StatusConflict, CodeExecutionNotCancellable)
	})

	t.Run("repository failure", func(t *testing.T) {
		base := newCancelTestRepo()
		lease := &cancelLeaseRepo{cancelTestRepo: base, cancelErr: errors.New("write failed")}
		f := newCancelFixture(t, lease, base, false)
		id := uuid.New()
		base.put(storagemodels.ExecutionModel{ID: id, Status: "pending"})
		_, err := f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: id})
		require.ErrorContains(t, err, "write failed")
	})

	t.Run("completed before the cancellation took effect", func(t *testing.T) {
		repo := newCancelTestRepo()
		f := newCancelFixture(t, repo, repo, true)
		exec, err := f.ops.StartExecution(context.Background(), StartExecutionParams{WorkflowID: f.wfID.String()})
		require.NoError(t, err)
		f.waitStarted(t)

		_, err = f.ops.CancelExecution(context.Background(), CancelExecutionParams{ExecutionID: uuid.MustParse(exec.ID)})
		opErr := requireOpError(t, err, http.StatusConflict, CodeExecutionNotCancellable)
		assert.Contains(t, opErr.Message, "finished with status completed")
	})
}

func TestRetryExecution(t *testing.T) {
	repo := newCancelTestRepo()
	f := newCancelFixture(t, repo, repo, false)
	close(f.release)

	original := uuid.New()
	triggerID := uuid.New()
	repo.put(storagemodels.ExecutionModel{
		ID: original, Status: "failed", WorkflowID: &f.wfID, WorkflowSource: "stored", TriggerID: &triggerID,
		InputData: storagemodels.JSONBMap{"in": 1},
		Variables: storagemodels.JSONBMap{"base": "original", "extra": "x"},
	})

	exec, err := f.ops.RetryExecution(context.Background(), RetryExecutionParams{ExecutionID: original})
	require.NoError(t, err)
	assert.NotEqual(t, original.String(), exec.ID)
	assert.Equal(t, f.wfID.String(), exec.WorkflowID)
	assert.Equal(t, models.ExecutionStatusPending, exec.Status)
	assert.Equal(t, map[string]any{"in": 1}, exec.Input)
	assert.Equal(t, "original", exec.Variables["base"], "the original variables win over the workflow's")
	assert.Equal(t, "x", exec.Variables["extra"])
	assert.Equal(t, original.String(), exec.Metadata[MetadataKeyRetryOf])

	require.NoError(t, f.em.Shutdown(context.Background()))
	row := repo.row(uuid.MustParse(exec.ID))
	assert.Equal(t, "completed", row.Status)
	assert.Equal(t, original.String(), row.Metadata[MetadataKeyRetryOf])
	require.NotNil(t, row.TriggerID)
	assert.Equal(t, triggerID, *row.TriggerID)
}

func TestRetryExecution_Errors(t *testing.T) {
	repo := newCancelTestRepo()
	f := newCancelFixture(t, repo, repo, false)
	ctx := context.Background()

	_, err := f.ops.RetryExecution(ctx, RetryExecutionParams{ExecutionID: uuid.New()})
	requireOpError(t, err, http.StatusNotFound, CodeExecutionNotFound)

	for _, status := range []string{"pending", "running", "completed"} {
		id := uuid.New()
		repo.put(storagemodels.ExecutionModel{ID: id, Status: status, WorkflowID: &f.wfID, WorkflowSource: "stored"})
		_, err = f.ops.RetryExecution(ctx, RetryExecutionParams{ExecutionID: id})
		opErr := requireOpError(t, err, http.StatusConflict, CodeExecutionNotRetryable)
		assert.Contains(t, opErr.Message, "execution is "+status)
	}

	inline := uuid.New()
	repo.put(storagemodels.ExecutionModel{ID: inline, Status: "cancelled", WorkflowSource: "inline"})
	_, err = f.ops.RetryExecution(ctx, RetryExecutionParams{ExecutionID: inline})
	opErr := requireOpError(t, err, http.StatusConflict, CodeExecutionNotRetryable)
	assert.Contains(t, opErr.Message, "inline executions")

	missingWorkflow := uuid.New()
	gone := uuid.New()
	repo.put(storagemodels.ExecutionModel{ID: missingWorkflow, Status: "timeout", WorkflowID: &gone, WorkflowSource: "stored"})
	_, err = f.ops.RetryExecution(ctx, RetryExecutionParams{ExecutionID: missingWorkflow})
	require.ErrorIs(t, err, models.ErrWorkflowNotFound)

	f.ops.ExecutionMgr = nil
	_, err = f.ops.RetryExecution(ctx, RetryExecutionParams{ExecutionID: missingWorkflow})
	requireOpError(t, err, http.StatusServiceUnavailable, "EXECUTION_MANAGER_UNAVAILABLE")
}

func TestNewConflictAndNotFoundErrors(t *testing.T) {
	assert.Equal(t, http.StatusConflict, NewConflictError("C", "m").HTTPStatus)
	assert.Equal(t, http.StatusNotFound, NewNotFoundError("N", "m").HTTPStatus)
}
