package trigger

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
)

func newSyncTestManager(t *testing.T) (*Manager, *mockTriggerRepo) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc, err := cache.NewRedisCache(config.RedisConfig{URL: "redis://" + mr.Addr(), PoolSize: 2})
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })

	repo := &mockTriggerRepo{}
	m, err := NewManager(ManagerConfig{
		TriggerRepo:  repo,
		WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: &engine.ExecutionManager{},
		Cache:        rc,
	})
	require.NoError(t, err)
	return m, repo
}

func (cs *CronScheduler) hasEntry(id string) bool {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	_, ok := cs.entries[id]
	return ok
}

func TestManager_SyncTrigger_AppliesCronChanges(t *testing.T) {
	m, repo := newSyncTestManager(t)
	ctx := context.Background()
	id := uuid.New()
	tm := &storagemodels.TriggerModel{
		ID: id, WorkflowID: uuid.New(), Type: "cron", Enabled: true,
		Config: storagemodels.JSONBMap{"schedule": "0 0 9 * * *"},
	}

	// Created and enabled -> scheduled.
	repo.On("FindByID", mock.Anything, id).Return(tm, nil).Once()
	require.NoError(t, m.SyncTrigger(ctx, id.String()))
	assert.True(t, m.cronScheduler.hasEntry(id.String()))

	// Disabled -> unscheduled.
	disabled := *tm
	disabled.Enabled = false
	repo.On("FindByID", mock.Anything, id).Return(&disabled, nil).Once()
	require.NoError(t, m.SyncTrigger(ctx, id.String()))
	assert.False(t, m.cronScheduler.hasEntry(id.String()))

	// Re-enabled, then deleted -> unscheduled (deleted triggers must stop firing).
	repo.On("FindByID", mock.Anything, id).Return(tm, nil).Once()
	require.NoError(t, m.SyncTrigger(ctx, id.String()))
	require.True(t, m.cronScheduler.hasEntry(id.String()))
	repo.On("FindByID", mock.Anything, id).Return(nil, nil).Once()
	require.NoError(t, m.SyncTrigger(ctx, id.String()))
	assert.False(t, m.cronScheduler.hasEntry(id.String()))
}

func TestManager_SyncTrigger_AppliesWebhookChanges(t *testing.T) {
	m, repo := newSyncTestManager(t)
	ctx := context.Background()
	id := uuid.New()
	tm := &storagemodels.TriggerModel{ID: id, WorkflowID: uuid.New(), Type: "webhook", Enabled: true, Config: storagemodels.JSONBMap{}}

	repo.On("FindByID", mock.Anything, id).Return(tm, nil).Once()
	require.NoError(t, m.SyncTrigger(ctx, id.String()))
	_, ok := m.WebhookRegistry().GetWebhook(id.String())
	assert.True(t, ok, "enabled webhook must be callable without a restart")

	repo.On("FindByID", mock.Anything, id).Return(nil, nil).Once()
	require.NoError(t, m.SyncTrigger(ctx, id.String()))
	_, ok = m.WebhookRegistry().GetWebhook(id.String())
	assert.False(t, ok)
}

func TestParseCronSchedule_HonoursTimezone(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	utc, err := ParseCronSchedule("0 0 9 * * *", "")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC), utc.Next(from).UTC())

	// 09:00 in Moscow (UTC+3) is 06:00 UTC.
	msk, err := ParseCronSchedule("0 0 9 * * *", "Europe/Moscow")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 1, 1, 6, 0, 0, 0, time.UTC), msk.Next(from).UTC())

	_, err = ParseCronSchedule("0 0 9 * * *", "Mars/Base")
	assert.Error(t, err)
}

func TestValidateScheduleConfig(t *testing.T) {
	assert.NoError(t, ValidateScheduleConfig("cron", map[string]any{"schedule": "@hourly"}))
	assert.NoError(t, ValidateScheduleConfig("cron", map[string]any{"schedule": "0 9 * * 1-5"}), "standard 5-field cron as in the docs")
	assert.NoError(t, ValidateScheduleConfig("cron", map[string]any{"schedule": "*/10 * * * * *"}), "6-field cron with seconds")
	assert.NoError(t, ValidateScheduleConfig("interval", map[string]any{"interval": "30s"}))
	assert.NoError(t, ValidateScheduleConfig("webhook", nil))
	assert.Error(t, ValidateScheduleConfig("cron", map[string]any{"expression": "0 * * * *"}))
	assert.Error(t, ValidateScheduleConfig("interval", map[string]any{"interval": "-1s"}))
}
