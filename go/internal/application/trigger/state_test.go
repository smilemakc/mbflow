package trigger

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
)

// setupTestRedis returns a cache backed by an in-process miniredis server.
func setupTestRedis(t *testing.T) *cache.RedisCache {
	t.Helper()
	rc, _ := setupTestRedisServer(t)
	return rc
}

// setupTestRedisServer is setupTestRedis that also returns the server, for
// tests that inspect keys or simulate an outage.
func setupTestRedisServer(t *testing.T) (*cache.RedisCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc, err := cache.NewRedisCache(config.RedisConfig{URL: "redis://" + mr.Addr(), PoolSize: 4})
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	return rc, mr
}

func TestNewTriggerState(t *testing.T) {
	triggerID := "test-trigger-123"
	state := NewTriggerState(triggerID)

	assert.Equal(t, triggerID, state.TriggerID)
	assert.Equal(t, int64(0), state.ExecutionCount)
	assert.False(t, state.UpdatedAt.IsZero())
}

func TestTriggerState_MarkExecuted(t *testing.T) {
	state := NewTriggerState("test-trigger")
	initialCount := state.ExecutionCount

	time.Sleep(10 * time.Millisecond)
	state.MarkExecuted()

	assert.Equal(t, initialCount+1, state.ExecutionCount)
	assert.False(t, state.LastExecuted.IsZero())
	assert.False(t, state.UpdatedAt.IsZero())
}

func TestTriggerState_SetNextExecution(t *testing.T) {
	state := NewTriggerState("test-trigger")
	nextTime := time.Now().Add(1 * time.Hour)

	state.SetNextExecution(nextTime)

	assert.Equal(t, nextTime.Unix(), state.NextExecution.Unix())
	assert.False(t, state.UpdatedAt.IsZero())
}

func TestTriggerState_SaveAndLoad(t *testing.T) {
	cache := setupTestRedis(t)
	ctx := context.Background()
	triggerID := "test-trigger-456"

	// Create and save state
	state := NewTriggerState(triggerID)
	state.MarkExecuted()
	state.SetNextExecution(time.Now().Add(1 * time.Hour))

	err := state.Save(ctx, cache)
	require.NoError(t, err)

	// Load state
	loaded, err := LoadTriggerState(ctx, cache, triggerID)
	require.NoError(t, err)

	assert.Equal(t, state.TriggerID, loaded.TriggerID)
	assert.Equal(t, state.ExecutionCount, loaded.ExecutionCount)
	assert.Equal(t, state.LastExecuted.Unix(), loaded.LastExecuted.Unix())
	assert.Equal(t, state.NextExecution.Unix(), loaded.NextExecution.Unix())
}

func TestDeleteTriggerState(t *testing.T) {
	cache := setupTestRedis(t)
	ctx := context.Background()
	triggerID := "test-trigger-789"

	// Create and save state
	state := NewTriggerState(triggerID)
	err := state.Save(ctx, cache)
	require.NoError(t, err)

	// Delete state
	err = DeleteTriggerState(ctx, cache, triggerID)
	require.NoError(t, err)

	// Verify deletion
	_, err = LoadTriggerState(ctx, cache, triggerID)
	assert.Error(t, err)
}

func TestGetTriggerStateKey(t *testing.T) {
	triggerID := "test-123"
	expected := "trigger:test-123:state"

	key := getTriggerStateKey(triggerID)
	assert.Equal(t, expected, key)
}

func TestLoadTriggerState_CorruptValue(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	require.NoError(t, mr.Set(getTriggerStateKey("t"), "{not json"))
	_, err := LoadTriggerState(context.Background(), rc, "t")
	assert.ErrorContains(t, err, "failed to unmarshal trigger state")

	mr.SetError("ERR redis unavailable")
	assert.ErrorContains(t, NewTriggerState("t").Save(context.Background(), rc), "failed to save trigger state")
}
