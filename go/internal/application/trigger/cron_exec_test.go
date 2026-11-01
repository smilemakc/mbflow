package trigger

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// newReplicaScheduler creates a cron scheduler as one API replica would,
// sharing the given Redis with the other replicas.
func newReplicaScheduler(t *testing.T, rc *cache.RedisCache, exec WorkflowExecutor) (*CronScheduler, *mockTriggerRepo) {
	t.Helper()
	repo := &mockTriggerRepo{}
	repo.On("MarkTriggered", mock.Anything, mock.Anything).Return(nil).Maybe()
	cs, err := NewCronScheduler(CronSchedulerConfig{
		TriggerRepo:  repo,
		WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: exec,
		Cache:        rc,
	})
	require.NoError(t, err)
	return cs, repo
}

func cronTrigger(id string) *models.Trigger {
	return &models.Trigger{
		ID:         id,
		WorkflowID: "wf-" + id,
		Type:       models.TriggerTypeCron,
		Enabled:    true,
		Config: map[string]any{
			"schedule": "0 0 9 * * *",
			"input":    map[string]any{"greeting": "hi"},
		},
	}
}

func TestAlignedIntervalSchedule(t *testing.T) {
	s := alignedIntervalSchedule{Delay: 30 * time.Second}

	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, base.Add(30*time.Second), s.Next(base))
	assert.Equal(t, base.Add(30*time.Second), s.Next(base.Add(1*time.Second)))
	assert.Equal(t, base.Add(30*time.Second), s.Next(base.Add(29*time.Second+999*time.Millisecond)))
	assert.Equal(t, base.Add(60*time.Second), s.Next(base.Add(30*time.Second)))

	// Replicas started at different moments compute the same ticks, which
	// is what per-tick claims rely on.
	replicaA := s.Next(base.Add(3 * time.Second))
	replicaB := s.Next(base.Add(17 * time.Second))
	assert.Equal(t, replicaA, replicaB)
}

func TestParseIntervalSchedule_AlignedAndRounded(t *testing.T) {
	cs := &CronScheduler{}

	sched, err := cs.parseIntervalSchedule(&models.Trigger{Config: map[string]any{"interval": "1m"}})
	require.NoError(t, err)
	assert.Equal(t, alignedIntervalSchedule{Delay: time.Minute}, sched)

	sched, err = cs.parseIntervalSchedule(&models.Trigger{Config: map[string]any{"interval": "100ms"}})
	require.NoError(t, err)
	assert.Equal(t, alignedIntervalSchedule{Delay: time.Second}, sched, "sub-second intervals are clamped")

	sched, err = cs.parseIntervalSchedule(&models.Trigger{Config: map[string]any{"interval": "1500ms"}})
	require.NoError(t, err)
	assert.Equal(t, alignedIntervalSchedule{Delay: 2 * time.Second}, sched)

	_, err = cs.parseIntervalSchedule(&models.Trigger{Config: map[string]any{"interval": true}})
	assert.ErrorContains(t, err, "invalid interval type")
}

func TestParseSchedule_UnsupportedType(t *testing.T) {
	cs := &CronScheduler{}
	_, err := cs.parseSchedule(&models.Trigger{Type: models.TriggerTypeWebhook})
	assert.ErrorContains(t, err, "unsupported trigger type")
}

func TestCronScheduler_RunTick_OnlyOneReplicaFires(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	exec := &fakeExecutor{}
	replicaA, repoA := newReplicaScheduler(t, rc, exec)
	replicaB, repoB := newReplicaScheduler(t, rc, exec)

	id := uuid.NewString()
	trig := cronTrigger(id)
	tick := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

	firedA := replicaA.runTick(trig, tick)
	firedB := replicaB.runTick(trig, tick)
	assert.True(t, firedA != firedB, "exactly one replica must fire a tick")
	require.Equal(t, 1, exec.Count())

	call := exec.Calls()[0]
	assert.Equal(t, "wf-"+id, call.WorkflowID)
	assert.Equal(t, id, call.TriggerID)
	assert.Equal(t, map[string]any{"greeting": "hi"}, call.Input)
	assert.True(t, mr.Exists(cronClaimKey(id, tick)))

	// The next tick is a new occurrence.
	assert.True(t, replicaB.runTick(trig, tick.Add(24*time.Hour)))
	assert.Equal(t, 2, exec.Count())

	// State and DB bookkeeping happened for each fired tick.
	state, err := LoadTriggerState(context.Background(), rc, id)
	require.NoError(t, err)
	assert.Equal(t, int64(2), state.ExecutionCount)
	repoA.AssertNumberOfCalls(t, "MarkTriggered", 1)
	repoB.AssertNumberOfCalls(t, "MarkTriggered", 1)
}

func TestCronScheduler_RunTick_FailsOpenWithoutRedis(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	exec := &fakeExecutor{}
	cs, repo := newReplicaScheduler(t, rc, exec)
	mr.SetError("ERR redis unavailable")

	assert.True(t, cs.runTick(cronTrigger(uuid.NewString()), time.Now()))
	assert.Equal(t, 1, exec.Count(), "redis outage must not drop scheduled runs")
	repo.AssertNumberOfCalls(t, "MarkTriggered", 1)
}

func TestCronScheduler_ExecuteTrigger_ExecutionError(t *testing.T) {
	rc := setupTestRedis(t)
	exec := &fakeExecutor{err: errExec}
	cs, repo := newReplicaScheduler(t, rc, exec)

	err := cs.executeTrigger(context.Background(), cronTrigger(uuid.NewString()))
	require.ErrorIs(t, err, errExec)
	repo.AssertNotCalled(t, "MarkTriggered", mock.Anything, mock.Anything)

	// A failed execution still counts as a claimed (fired) tick.
	assert.True(t, cs.runTick(cronTrigger(uuid.NewString()), time.Now()))
}

func TestCronScheduler_ExecuteTrigger_RecordsNextExecution(t *testing.T) {
	rc := setupTestRedis(t)
	exec := &fakeExecutor{}
	cs, repo := newReplicaScheduler(t, rc, exec)
	ctx := context.Background()

	id := uuid.NewString()
	trig := cronTrigger(id)
	trig.Config = map[string]any{"schedule": "0 0 9 * * *"} // no default input
	require.NoError(t, cs.AddTrigger(ctx, trig))

	// MarkTriggered failures are logged, not returned.
	repo.ExpectedCalls = nil
	repo.On("MarkTriggered", mock.Anything, mock.Anything).Return(errExec).Once()

	require.NoError(t, cs.executeTrigger(ctx, trig))
	assert.Equal(t, map[string]any{}, exec.Calls()[0].Input)

	state, err := LoadTriggerState(ctx, rc, id)
	require.NoError(t, err)
	assert.Equal(t, int64(1), state.ExecutionCount)
	assert.False(t, state.NextExecution.IsZero())
	repo.AssertExpectations(t)
}

func TestCronScheduler_ScheduledTick(t *testing.T) {
	cs, _ := newReplicaScheduler(t, nil, &fakeExecutor{})
	now := time.Date(2026, 3, 1, 9, 0, 0, 400*int(time.Millisecond), time.UTC)

	// Unknown trigger: now rounded to the scheduler precision.
	assert.Equal(t, time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC), cs.scheduledTick("missing", now))

	// Known trigger that has not fired yet (no Prev): same fallback.
	trig := cronTrigger(uuid.NewString())
	require.NoError(t, cs.AddTrigger(context.Background(), trig))
	assert.Equal(t, time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC), cs.scheduledTick(trig.ID, now))
}

func TestCronScheduler_EndToEnd_TwoReplicasFireOncePerTick(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	exec := &fakeExecutor{}
	replicaA, _ := newReplicaScheduler(t, rc, exec)
	replicaB, _ := newReplicaScheduler(t, rc, exec)

	id := uuid.NewString()
	trig := &models.Trigger{
		ID: id, WorkflowID: "wf", Type: models.TriggerTypeInterval, Enabled: true,
		Config: map[string]any{"interval": "1s"},
	}
	ctx := context.Background()
	require.NoError(t, replicaA.Start(ctx, nil))
	require.NoError(t, replicaB.Start(ctx, nil))
	require.NoError(t, replicaA.AddTrigger(ctx, trig))
	require.NoError(t, replicaB.AddTrigger(ctx, trig))

	require.Eventually(t, func() bool { return exec.Count() >= 1 }, 3*time.Second, 10*time.Millisecond)
	require.NoError(t, replicaA.Stop())
	require.NoError(t, replicaB.Stop())

	// The fired tick was keyed by the scheduled (aligned, whole-second)
	// time recorded by the cron runner.
	for _, k := range mr.Keys() {
		if strings.HasPrefix(k, claimKeyPrefix+id+":tick:") {
			ts, err := strconv.ParseInt(strings.TrimPrefix(k, claimKeyPrefix+id+":tick:"), 10, 64)
			require.NoError(t, err)
			assert.WithinDuration(t, time.Now(), time.Unix(ts, 0), 5*time.Second)
		}
	}

	ticks := 0
	for _, k := range mr.Keys() {
		if strings.HasPrefix(k, claimKeyPrefix+id+":tick:") {
			ticks++
		}
	}
	assert.Equal(t, ticks, exec.Count(), "each tick must be executed by exactly one replica")
}

func TestCronScheduler_StartSkipsInvalidTriggers(t *testing.T) {
	cs, _ := newReplicaScheduler(t, setupTestRedis(t), &fakeExecutor{})
	bad := &storagemodels.TriggerModel{
		ID: uuid.New(), WorkflowID: uuid.New(), Type: "cron", Enabled: true,
		Config: storagemodels.JSONBMap{"schedule": "61 * * * *"},
	}
	good := &storagemodels.TriggerModel{
		ID: uuid.New(), WorkflowID: uuid.New(), Type: "interval", Enabled: true,
		Config: storagemodels.JSONBMap{"interval": "5m"},
	}
	require.NoError(t, cs.Start(context.Background(), []*storagemodels.TriggerModel{bad, good}))
	t.Cleanup(func() { _ = cs.Stop() })
	assert.False(t, cs.hasEntry(bad.ID.String()))
	assert.True(t, cs.hasEntry(good.ID.String()))
}
