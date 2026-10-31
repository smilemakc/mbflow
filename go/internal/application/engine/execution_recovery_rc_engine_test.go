package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryConfig_WithDefaults(t *testing.T) {
	cfg, err := RecoveryConfig{}.withDefaults()
	require.NoError(t, err)
	assert.Equal(t, DefaultHeartbeatInterval, cfg.HeartbeatInterval)
	assert.Equal(t, 3*DefaultHeartbeatInterval, cfg.StaleAfter)

	cfg, err = RecoveryConfig{HeartbeatInterval: time.Second}.withDefaults()
	require.NoError(t, err)
	assert.Equal(t, 3*time.Second, cfg.StaleAfter)

	_, err = RecoveryConfig{HeartbeatInterval: time.Second, StaleAfter: time.Second}.withDefaults()
	require.Error(t, err)
}

func TestStartRecovery_Errors(t *testing.T) {
	f := newStoredFixture(t, echoExecutor(), newFakeExecutionRepo())
	require.ErrorIs(t, f.em.StartRecovery(context.Background(), RecoveryConfig{}), ErrRecoveryUnsupported)

	lease := newFakeLeaseRepo()
	g := newStoredFixture(t, echoExecutor(), lease)
	require.Error(t, g.em.StartRecovery(context.Background(), RecoveryConfig{HeartbeatInterval: time.Minute, StaleAfter: time.Second}))
	require.NoError(t, g.em.StartRecovery(context.Background(), RecoveryConfig{HeartbeatInterval: time.Hour}))
	require.Error(t, g.em.StartRecovery(context.Background(), RecoveryConfig{HeartbeatInterval: time.Hour}), "recovery starts only once")
}

func TestStartRecovery_SweepsAtStartup(t *testing.T) {
	lease := newFakeLeaseRepo()
	lease.staleIDs = []uuid.UUID{uuid.New(), uuid.New()}
	f := newStoredFixture(t, echoExecutor(), lease)

	require.NoError(t, f.em.StartRecovery(context.Background(), RecoveryConfig{HeartbeatInterval: time.Hour, StaleAfter: 2 * time.Hour}))

	assert.Equal(t, 1, lease.sweepCount(), "the startup sweep runs before StartRecovery returns")
	assert.Equal(t, 2*time.Hour, lease.staleAfter)
	assert.Equal(t, InterruptedExecutionError, lease.errMsg)
}

func TestStartRecovery_HeartbeatsOwnedExecutions(t *testing.T) {
	block := newBlockingExecutor()
	lease := newFakeLeaseRepo()
	f := newStoredFixture(t, block, lease)
	f.em.SetMaxConcurrentExecutions(1)
	require.NoError(t, f.em.StartRecovery(context.Background(), RecoveryConfig{HeartbeatInterval: 5 * time.Millisecond}))

	running, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)
	queued, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)

	eventually(t, func() bool {
		return lease.touchCount(running.ID) >= 2 && lease.touchCount(queued.ID) >= 2
	}, "running and queued executions get their lease renewed")
	eventually(t, func() bool { return lease.sweepCount() >= 3 }, "the sweeper runs periodically")

	close(block.release)
	eventually(t, func() bool { return len(f.em.active.list()) == 0 }, "finished executions are released")

	require.NoError(t, f.em.Shutdown(context.Background()))
	sweeps := lease.sweepCount()
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, sweeps, lease.sweepCount(), "the recovery loop stops on shutdown")
}

func TestStartRecovery_SurvivesRepositoryErrors(t *testing.T) {
	block := newBlockingExecutor()
	lease := newFakeLeaseRepo()
	lease.touchErr = errors.New("touch failed")
	lease.sweepErr = errors.New("sweep failed")
	f := newStoredFixture(t, block, lease)

	require.NoError(t, f.em.StartRecovery(context.Background(), RecoveryConfig{HeartbeatInterval: 5 * time.Millisecond}))
	_, err := f.em.ExecuteAsync(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	waitStarted(t, block)

	eventually(t, func() bool { return lease.sweepCount() >= 3 }, "the loop keeps running after errors")
	close(block.release)
}

func TestStartRecovery_StopsWhenContextEnds(t *testing.T) {
	lease := newFakeLeaseRepo()
	f := newStoredFixture(t, echoExecutor(), lease)

	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, f.em.StartRecovery(ctx, RecoveryConfig{HeartbeatInterval: 5 * time.Millisecond}))
	eventually(t, func() bool { return lease.sweepCount() >= 2 }, "the loop runs")
	cancel()

	done := make(chan struct{})
	go func() {
		f.em.recoveryWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("recovery loop did not stop after its context was cancelled")
	}
}

func TestActiveExecutions_IgnoresInvalidIDs(t *testing.T) {
	var a activeExecutions
	untrack := a.add("not-a-uuid", nil)
	assert.Empty(t, a.list())
	untrack()

	id := uuid.NewString()
	untrack = a.add(id, nil)
	assert.Equal(t, []uuid.UUID{uuid.MustParse(id)}, a.list())
	untrack()
	assert.Empty(t, a.list())
}
