package serviceapi

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func TestRunTrigger_NotFoundAndDisabled(t *testing.T) {
	ctx := context.Background()
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	missing := uuid.New()
	trigRepo.On("FindByID", mock.Anything, missing).Return(nil, nil)
	_, err := ops.RunTrigger(ctx, RunTriggerParams{TriggerID: missing})
	assert.ErrorIs(t, err, models.ErrTriggerNotFound)

	failing := uuid.New()
	trigRepo.On("FindByID", mock.Anything, failing).Return(nil, errors.New("db down"))
	_, err = ops.RunTrigger(ctx, RunTriggerParams{TriggerID: failing})
	assert.ErrorIs(t, err, models.ErrTriggerNotFound)

	disabled := uuid.New()
	trigRepo.On("FindByID", mock.Anything, disabled).Return(&storagemodels.TriggerModel{ID: disabled, WorkflowID: uuid.New(), Type: "manual"}, nil)
	_, err = ops.RunTrigger(ctx, RunTriggerParams{TriggerID: disabled})
	assert.ErrorIs(t, err, models.ErrTriggerDisabled)
}

func TestRunTrigger_NoExecutionManager(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)
	id := uuid.New()
	trigRepo.On("FindByID", mock.Anything, id).Return(&storagemodels.TriggerModel{ID: id, WorkflowID: uuid.New(), Type: "manual", Enabled: true}, nil)

	_, err := ops.RunTrigger(context.Background(), RunTriggerParams{TriggerID: id})
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "EXECUTION_UNAVAILABLE", opErr.Code)
	assert.Equal(t, http.StatusServiceUnavailable, opErr.HTTPStatus)
}

func TestRunTrigger_StartsAsyncExecution(t *testing.T) {
	ctx := context.Background()
	wfRepo := new(mockWorkflowRepo)
	execRepo := new(mockExecutionRepo)
	trigRepo := new(mockTriggerRepo)

	wfID := uuid.New()
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{
		ID: wfID, Name: "wf", Status: "active", Version: 1,
		Nodes: []*storagemodels.NodeModel{
			{ID: uuid.New(), NodeID: "n1", WorkflowID: wfID, Name: "N1", Type: rcType, Config: storagemodels.JSONBMap{"value": "v"}},
		},
	}, nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, mock.Anything).Return(nil, models.ErrWorkflowNotFound)

	var mu sync.Mutex
	var created *storagemodels.ExecutionModel
	execRepo.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		mu.Lock()
		defer mu.Unlock()
		created = args.Get(1).(*storagemodels.ExecutionModel)
	}).Return(nil)
	execRepo.On("Update", mock.Anything, mock.Anything).Return(nil).Maybe()
	execRepo.On("CreateNodeExecution", mock.Anything, mock.Anything).Return(nil).Maybe()
	execRepo.On("UpdateNodeExecution", mock.Anything, mock.Anything).Return(nil).Maybe()
	execRepo.On("FindNodeExecutionsByExecutionID", mock.Anything, mock.Anything).Return(nil, nil).Maybe()

	trigID := uuid.New()
	trigRepo.On("FindByID", mock.Anything, trigID).Return(&storagemodels.TriggerModel{
		ID: trigID, WorkflowID: wfID, Type: "manual", Enabled: true,
		Config: storagemodels.JSONBMap{"input": map[string]any{"a": "default", "b": "default"}},
	}, nil)
	// A failure to record last_run does not fail the run.
	trigRepo.On("MarkTriggered", mock.Anything, trigID).Return(errors.New("db hiccup")).Once()
	trigRepo.On("MarkTriggered", mock.Anything, trigID).Return(nil)

	executors := rcExecutors(false)
	ops := newTestOperations(wfRepo, execRepo, trigRepo, nil, nil, nil, executors)
	em := engine.NewExecutionManager(executors, wfRepo, execRepo, nil, nil, nil)
	ops.ExecutionMgr = em

	exec, err := ops.RunTrigger(ctx, RunTriggerParams{TriggerID: trigID, Input: map[string]any{"b": "request"}})
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusPending, exec.Status)
	assert.Equal(t, wfID.String(), exec.WorkflowID)

	_, err = ops.RunTrigger(ctx, RunTriggerParams{TriggerID: trigID})
	require.NoError(t, err)

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	require.NoError(t, em.Shutdown(shutdownCtx))

	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, created)
	require.NotNil(t, created.TriggerID)
	assert.Equal(t, trigID, *created.TriggerID)
	trigRepo.AssertNumberOfCalls(t, "MarkTriggered", 2)

	// The workflow vanished: the error is returned and last_run untouched.
	goneID := uuid.New()
	trigRepo.On("FindByID", mock.Anything, goneID).Return(&storagemodels.TriggerModel{ID: goneID, WorkflowID: uuid.New(), Type: "manual", Enabled: true}, nil)
	em2 := engine.NewExecutionManager(executors, wfRepo, execRepo, nil, nil, nil)
	ops.ExecutionMgr = em2
	_, err = ops.RunTrigger(ctx, RunTriggerParams{TriggerID: goneID})
	assert.ErrorIs(t, err, models.ErrWorkflowNotFound)
	trigRepo.AssertNotCalled(t, "MarkTriggered", mock.Anything, goneID)
}

func TestMergeTriggerInput(t *testing.T) {
	defaults := map[string]any{"a": 1, "b": 2}
	config := map[string]any{"input": defaults}
	input := map[string]any{"b": 3, "c": 4}

	merged := mergeTriggerInput(config, input)
	assert.Equal(t, map[string]any{"a": 1, "b": 3, "c": 4}, merged)
	assert.Equal(t, map[string]any{"a": 1, "b": 2}, defaults, "config input is not modified")
	assert.Equal(t, map[string]any{"b": 3, "c": 4}, input, "request input is not modified")

	assert.Equal(t, map[string]any{}, mergeTriggerInput(nil, nil))
	assert.Equal(t, map[string]any{"x": 1}, mergeTriggerInput(map[string]any{"input": "not a map"}, map[string]any{"x": 1}))
}

func TestCreateTrigger_PersistsNameAndDescription(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(wfRepo, nil, trigRepo, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	var stored *storagemodels.TriggerModel
	trigRepo.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		stored = args.Get(1).(*storagemodels.TriggerModel)
	}).Return(nil)

	res, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
		WorkflowID: wfID.String(), Name: "Nightly", Description: "desc", Type: "manual", Config: map[string]any{},
	})
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "Nightly", stored.Name)
	assert.Equal(t, "desc", stored.Description)
	assert.Equal(t, "Nightly", res.Name)
	assert.Nil(t, res.NextRun)
}

func TestUpdateTrigger_NameAndDescription(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	id := uuid.New()
	tm := &storagemodels.TriggerModel{ID: id, WorkflowID: uuid.New(), Type: "manual", Name: "old", Description: "old desc", Config: storagemodels.JSONBMap{}}
	trigRepo.On("FindByID", mock.Anything, id).Return(tm, nil)
	trigRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	// Omitted fields keep their stored values.
	res, err := ops.UpdateTrigger(context.Background(), UpdateTriggerParams{TriggerID: id})
	require.NoError(t, err)
	assert.Equal(t, "old", res.Name)
	assert.Equal(t, "old desc", res.Description)

	empty := ""
	res, err = ops.UpdateTrigger(context.Background(), UpdateTriggerParams{TriggerID: id, Name: "new", Description: &empty})
	require.NoError(t, err)
	assert.Equal(t, "new", res.Name)
	assert.Empty(t, res.Description)
	assert.Equal(t, "new", tm.Name)
	assert.Empty(t, tm.Description)
}

func TestTriggerModelToDomain_NextRun(t *testing.T) {
	fixed := time.Date(2026, 10, 31, 10, 7, 30, 0, time.UTC)
	orig := timeNow
	timeNow = func() time.Time { return fixed }
	t.Cleanup(func() { timeNow = orig })

	interval := triggerModelToDomain(&storagemodels.TriggerModel{
		ID: uuid.New(), WorkflowID: uuid.New(), Type: "interval", Enabled: true,
		Config: storagemodels.JSONBMap{"interval": "10m"},
	})
	require.NotNil(t, interval.NextRun)
	assert.True(t, interval.NextRun.Equal(time.Date(2026, 10, 31, 10, 10, 0, 0, time.UTC)))

	cron := triggerModelToDomain(&storagemodels.TriggerModel{
		ID: uuid.New(), WorkflowID: uuid.New(), Type: "cron", Enabled: true,
		Config: storagemodels.JSONBMap{"schedule": "30 12 * * *", "timezone": "UTC"},
	})
	require.NotNil(t, cron.NextRun)
	assert.True(t, cron.NextRun.Equal(time.Date(2026, 10, 31, 12, 30, 0, 0, time.UTC)))

	disabled := triggerModelToDomain(&storagemodels.TriggerModel{
		ID: uuid.New(), WorkflowID: uuid.New(), Type: "cron",
		Config: storagemodels.JSONBMap{"schedule": "30 12 * * *"},
	})
	assert.Nil(t, disabled.NextRun)
}
