package trigger

import (
	"context"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) *cache.RedisCache {
	// Create a test Redis cache using the existing constructor
	cfg := config.RedisConfig{
		URL:      "redis://localhost:6379",
		Password: "",
		DB:       1, // Use test database
		PoolSize: 10,
	}

	redisCache, err := cache.NewRedisCache(cfg)
	if err != nil {
		t.Skip("Redis not available for testing")
	}

	// Clean test database
	ctx := context.Background()
	redisCache.Client().FlushDB(ctx)

	return redisCache
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
	t.Skip("Requires Redis connection - run with integration tests")

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
	t.Skip("Requires Redis connection - run with integration tests")

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
