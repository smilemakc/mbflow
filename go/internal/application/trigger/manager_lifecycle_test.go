package trigger

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// newTestManager builds a Manager on miniredis with a recording executor.
func newTestManager(t *testing.T) (*Manager, *mockTriggerRepo, *fakeExecutor) {
	t.Helper()
	rc := setupTestRedis(t)
	repo := &mockTriggerRepo{}
	repo.On("MarkTriggered", mock.Anything, mock.Anything).Return(nil).Maybe()
	exec := &fakeExecutor{}
	m, err := NewManager(ManagerConfig{
		TriggerRepo:  repo,
		WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: exec,
		Cache:        rc,
	})
	require.NoError(t, err)
	return m, repo, exec
}

func TestManager_StartStop(t *testing.T) {
	m, repo, _ := newTestManager(t)

	cronTM := &storagemodels.TriggerModel{
		ID: uuid.New(), WorkflowID: uuid.New(), Type: "cron", Enabled: true,
		Config: storagemodels.JSONBMap{"schedule": "0 0 9 * * *"},
	}
	eventTM := eventTriggerModel("doc.created")
	webhookTM := &storagemodels.TriggerModel{
		ID: uuid.New(), WorkflowID: uuid.New(), Type: "webhook", Enabled: true,
		Config: storagemodels.JSONBMap{},
	}
	repo.On("FindEnabled", mock.Anything).
		Return([]*storagemodels.TriggerModel{cronTM, eventTM, webhookTM}, nil).Once()

	require.NoError(t, m.Start())
	assert.True(t, m.cronScheduler.hasEntry(cronTM.ID.String()))
	assert.Len(t, m.eventListener.triggers["doc.created"], 1)
	_, ok := m.WebhookRegistry().GetWebhook(webhookTM.ID.String())
	assert.True(t, ok)

	// The startup schedule records the next run in the trigger state.
	state, err := LoadTriggerState(context.Background(), m.cache, cronTM.ID.String())
	require.NoError(t, err)
	assert.False(t, state.NextExecution.IsZero())

	done := make(chan error, 1)
	go func() { done <- m.Stop() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Stop hung")
	}
	assert.Error(t, m.ctx.Err(), "Stop cancels the manager context")
}

func TestManager_StartFailsWhenTriggersCannotBeLoaded(t *testing.T) {
	m, repo, _ := newTestManager(t)
	repo.On("FindEnabled", mock.Anything).Return(nil, errExec).Once()
	require.ErrorIs(t, m.Start(), errExec)
}

func TestManager_TriggerManual(t *testing.T) {
	m, _, exec := newTestManager(t)
	ctx := context.Background()
	id := uuid.NewString()

	execID, err := m.TriggerManual(ctx, id, "wf-1", map[string]any{"a": 1})
	require.NoError(t, err)
	assert.Equal(t, "exec-wf-1", execID)
	assert.Equal(t, id, exec.Calls()[0].TriggerID)

	state, err := LoadTriggerState(ctx, m.cache, id)
	require.NoError(t, err)
	assert.Equal(t, int64(1), state.ExecutionCount)

	exec.err = errExec
	_, err = m.TriggerManual(ctx, id, "wf-1", nil)
	require.ErrorIs(t, err, errExec)
}

func TestManager_TriggerManual_StateSaveFailureIsNotFatal(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	exec := &fakeExecutor{}
	m, err := NewManager(ManagerConfig{
		TriggerRepo: &mockTriggerRepo{}, WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: exec, Cache: rc,
	})
	require.NoError(t, err)

	mr.SetError("ERR redis unavailable")
	execID, err := m.TriggerManual(context.Background(), uuid.NewString(), "wf", nil)
	require.NoError(t, err)
	assert.Equal(t, "exec-wf", execID)
	assert.Error(t, m.updateTriggerState(context.Background(), "t"))
}

func TestManager_OnTriggerCreated_AllTypes(t *testing.T) {
	m, _, _ := newTestManager(t)
	ctx := context.Background()

	interval := &models.Trigger{
		ID: uuid.NewString(), WorkflowID: "wf", Type: models.TriggerTypeInterval, Enabled: true,
		Config: map[string]any{"interval": "1h"},
	}
	require.NoError(t, m.OnTriggerCreated(ctx, interval))
	assert.True(t, m.cronScheduler.hasEntry(interval.ID))

	event := m.eventListener.modelToDomain(eventTriggerModel("e1"))
	require.NoError(t, m.OnTriggerCreated(ctx, event))
	assert.Len(t, m.eventListener.triggers["e1"], 1)

	manual := &models.Trigger{ID: uuid.NewString(), Type: models.TriggerTypeManual, Enabled: true}
	require.NoError(t, m.OnTriggerCreated(ctx, manual))

	disabled := &models.Trigger{
		ID: uuid.NewString(), Type: models.TriggerTypeCron, Enabled: false,
		Config: map[string]any{"schedule": "@hourly"},
	}
	require.NoError(t, m.OnTriggerCreated(ctx, disabled))
	assert.False(t, m.cronScheduler.hasEntry(disabled.ID))

	// Invalid cron expressions are rejected by the scheduler path.
	bad := &models.Trigger{
		ID: uuid.NewString(), Type: models.TriggerTypeCron, Enabled: true,
		Config: map[string]any{"schedule": "not a cron"},
	}
	assert.ErrorContains(t, m.OnTriggerCreated(ctx, bad), "invalid cron expression")

	// Deleting removes the trigger from every handler.
	require.NoError(t, m.OnTriggerDeleted(ctx, event.ID))
	assert.Empty(t, m.eventListener.triggers["e1"])
	require.NoError(t, m.OnTriggerDeleted(ctx, interval.ID))
	assert.False(t, m.cronScheduler.hasEntry(interval.ID))
}

func TestManager_OnTriggerDeleted_ClearStateFailureIsLogged(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	m, err := NewManager(ManagerConfig{
		TriggerRepo: &mockTriggerRepo{}, WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: &fakeExecutor{}, Cache: rc,
	})
	require.NoError(t, err)
	mr.SetError("ERR redis unavailable")
	assert.NoError(t, m.OnTriggerDeleted(context.Background(), uuid.NewString()))
}

func TestManager_SyncTrigger_Errors(t *testing.T) {
	m, repo, _ := newTestManager(t)
	ctx := context.Background()

	assert.ErrorContains(t, m.SyncTrigger(ctx, "not-a-uuid"), "invalid trigger ID")

	id := uuid.New()
	repo.On("FindByID", mock.Anything, id).Return(nil, errExec).Once()
	assert.ErrorIs(t, m.SyncTrigger(ctx, id.String()), errExec)
}
