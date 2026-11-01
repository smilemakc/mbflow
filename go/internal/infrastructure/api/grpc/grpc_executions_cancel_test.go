package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pb "github.com/smilemakc/mbflow/go/api/proto/serviceapipb"
	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// rcLeaseExecutionRepo adds the writes and lease methods the execution
// manager needs for cancel and retry.
type rcLeaseExecutionRepo struct {
	*rcExecutionRepo
}

func (r *rcLeaseExecutionRepo) Create(_ context.Context, ex *storagemodels.ExecutionModel) error {
	r.put(ex)
	return nil
}

func (r *rcLeaseExecutionRepo) Update(_ context.Context, ex *storagemodels.ExecutionModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.executions[ex.ID]; ok {
		ex.WorkflowID, ex.Metadata = old.WorkflowID, old.Metadata
	}
	r.executions[ex.ID] = ex
	return nil
}

func (r *rcLeaseExecutionRepo) TouchExecutions(_ context.Context, ids []uuid.UUID) (int, error) {
	return len(ids), nil
}

func (r *rcLeaseExecutionRepo) FailStaleExecutions(context.Context, time.Duration, string) ([]uuid.UUID, error) {
	return nil, nil
}

func (r *rcLeaseExecutionRepo) CancelExecution(_ context.Context, id uuid.UUID, errMsg string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ex, ok := r.executions[id]
	if !ok || (ex.Status != "pending" && ex.Status != "running") {
		return false, nil
	}
	ex.Status, ex.Error = "cancelled", errMsg
	return true, nil
}

func (r *rcLeaseExecutionRepo) CancelledExecutions(context.Context, []uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}

func rcCancelEnv(t *testing.T) (*rcEnv, *rcLeaseExecutionRepo, *storagemodels.WorkflowModel) {
	t.Helper()
	env := rcNewEnv()
	repo := &rcLeaseExecutionRepo{env.execs}
	env.ops.ExecutionRepo = repo
	em := engine.NewExecutionManager(env.executors, env.workflows, repo, nil, nil, nil)
	env.ops.ExecutionMgr = em
	t.Cleanup(func() { _ = em.Shutdown(context.Background()) })

	id := uuid.New()
	wf := &storagemodels.WorkflowModel{
		ID: id, Name: "single", Status: "active", Version: 1,
		Nodes: []*storagemodels.NodeModel{{ID: uuid.New(), NodeID: "a", WorkflowID: id, Name: "A", Type: rcNodeType}},
	}
	env.workflows.put(wf)
	return env, repo, wf
}

func TestRCExecutions_CancelRunningElsewhere(t *testing.T) {
	ctx := context.Background()
	env, repo, wf := rcCancelEnv(t)

	running := &storagemodels.ExecutionModel{ID: uuid.New(), WorkflowID: &wf.ID, WorkflowSource: "stored", Status: "running"}
	repo.put(running)

	resp, err := env.srv.CancelExecution(ctx, &pb.CancelExecutionRequest{Id: running.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, running.ID.String(), resp.Execution.Id)
	assert.Equal(t, string(models.ExecutionStatusCancelled), resp.Execution.Status)

	_, err = env.srv.CancelExecution(ctx, &pb.CancelExecutionRequest{Id: running.ID.String()})
	assert.Equal(t, codes.FailedPrecondition, rcCode(t, err))
}

func TestRCExecutions_Retry(t *testing.T) {
	ctx := context.Background()
	env, repo, wf := rcCancelEnv(t)

	failed := &storagemodels.ExecutionModel{
		ID: uuid.New(), WorkflowID: &wf.ID, WorkflowSource: "stored", Status: "failed",
		InputData: storagemodels.JSONBMap{"in": "x"},
	}
	repo.put(failed)

	resp, err := env.srv.RetryExecution(ctx, &pb.RetryExecutionRequest{Id: failed.ID.String()})
	require.NoError(t, err)
	assert.NotEqual(t, failed.ID.String(), resp.Execution.Id)
	assert.Equal(t, wf.ID.String(), resp.Execution.WorkflowId)
	assert.Equal(t, "x", resp.Execution.Input.AsMap()["in"])

	require.NoError(t, env.ops.ExecutionMgr.Shutdown(ctx))
	created, err := repo.FindByID(ctx, uuid.MustParse(resp.Execution.Id))
	require.NoError(t, err)
	assert.Equal(t, failed.ID.String(), created.Metadata[serviceapi.MetadataKeyRetryOf])

	inline := &storagemodels.ExecutionModel{ID: uuid.New(), WorkflowSource: "inline", Status: "failed"}
	repo.put(inline)
	_, err = env.srv.RetryExecution(ctx, &pb.RetryExecutionRequest{Id: inline.ID.String()})
	assert.Equal(t, codes.FailedPrecondition, rcCode(t, err))
}

func TestMapError_StateConflictsAreFailedPrecondition(t *testing.T) {
	err := mapError(serviceapi.NewConflictError(serviceapi.CodeExecutionNotCancellable, "done"))
	assert.Equal(t, codes.FailedPrecondition, rcCode(t, err))
	err = mapError(serviceapi.NewConflictError("OTHER", "exists"))
	assert.Equal(t, codes.AlreadyExists, rcCode(t, err), "other conflicts keep AlreadyExists")
	err = mapError(serviceapi.NewNotFoundError(serviceapi.CodeExecutionNotFound, "missing"))
	assert.Equal(t, codes.NotFound, rcCode(t, err))
}
