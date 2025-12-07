package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== NewRedisCache Tests ====================

func TestNewRedisCache_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cfg := config.RedisConfig{
		URL:      "redis://" + s.Addr(),
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	cache, err := NewRedisCache(cfg)
	require.NoError(t, err)
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.Client())

	err = cache.Close()
	assert.NoError(t, err)
}

func TestNewRedisCache_WithPassword(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	// Set password on miniredis
	s.RequireAuth("secret")

	cfg := config.RedisConfig{
		URL:      "redis://" + s.Addr(),
		Password: "secret",
		DB:       0,
		PoolSize: 10,
	}

	cache, err := NewRedisCache(cfg)
	require.NoError(t, err)
	assert.NotNil(t, cache)

	err = cache.Close()
	assert.NoError(t, err)
}

func TestNewRedisCache_WithDB(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cfg := config.RedisConfig{
		URL:      "redis://" + s.Addr(),
		Password: "",
		DB:       1,
		PoolSize: 10,
	}

	cache, err := NewRedisCache(cfg)
	require.NoError(t, err)
	assert.NotNil(t, cache)

	err = cache.Close()
	assert.NoError(t, err)
}

func TestNewRedisCache_InvalidURL(t *testing.T) {
	cfg := config.RedisConfig{
		URL:      "invalid://url",
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	cache, err := NewRedisCache(cfg)
	assert.Error(t, err)
	assert.Nil(t, cache)
	assert.Contains(t, err.Error(), "failed to parse Redis URL")
}

func TestNewRedisCache_ConnectionFailure(t *testing.T) {
	cfg := config.RedisConfig{
		URL:      "redis://localhost:9999", // Non-existent server
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	cache, err := NewRedisCache(cfg)
	assert.Error(t, err)
	assert.Nil(t, cache)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

// ==================== Client Method Tests ====================

func TestRedisCache_Client(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cfg := config.RedisConfig{
		URL:      "redis://" + s.Addr(),
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	cache, err := NewRedisCache(cfg)
	require.NoError(t, err)
	defer cache.Close()

	client := cache.Client()
	assert.NotNil(t, client)

	// Verify client is functional
	ctx := context.Background()
	err = client.Ping(ctx).Err()
	assert.NoError(t, err)
}

// ==================== Health Method Tests ====================

func TestRedisCache_Health_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cfg := config.RedisConfig{
		URL:      "redis://" + s.Addr(),
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	cache, err := NewRedisCache(cfg)
	require.NoError(t, err)
	defer cache.Close()

	ctx := context.Background()
	err = cache.Health(ctx)
	assert.NoError(t, err)
}

func TestRedisCache_Health_AfterClose(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cfg := config.RedisConfig{
		URL:      "redis://" + s.Addr(),
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	cache, err := NewRedisCache(cfg)
	require.NoError(t, err)

	// Close the connection
	err = cache.Close()
	require.NoError(t, err)

	// Health check should fail
	ctx := context.Background()
	err = cache.Health(ctx)
	assert.Error(t, err)
}

// ==================== Set/Get Tests ====================

func TestRedisCache_Set_Get_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Set value
	err := cache.Set(ctx, "test_key", "test_value", 0)
	require.NoError(t, err)

	// Get value
	value, err := cache.Get(ctx, "test_key")
	require.NoError(t, err)
	assert.Equal(t, "test_value", value)
}

func TestRedisCache_Set_WithTTL(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Set value with TTL
	err := cache.Set(ctx, "ttl_key", "ttl_value", 1*time.Second)
	require.NoError(t, err)

	// Value should exist
	value, err := cache.Get(ctx, "ttl_key")
	require.NoError(t, err)
	assert.Equal(t, "ttl_value", value)

	// Fast-forward time in miniredis
	s.FastForward(2 * time.Second)

	// Value should be expired
	_, err = cache.Get(ctx, "ttl_key")
	assert.Error(t, err) // redis.Nil error
}

func TestRedisCache_Get_NonExistentKey(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	_, err := cache.Get(ctx, "non_existent")
	assert.Error(t, err) // redis.Nil error
}

func TestRedisCache_Set_OverwriteValue(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Set initial value
	err := cache.Set(ctx, "key", "value1", 0)
	require.NoError(t, err)

	// Overwrite with new value
	err = cache.Set(ctx, "key", "value2", 0)
	require.NoError(t, err)

	// Get value
	value, err := cache.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, "value2", value)
}

// ==================== Delete Tests ====================

func TestRedisCache_Delete_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Set value
	err := cache.Set(ctx, "delete_key", "value", 0)
	require.NoError(t, err)

	// Delete value
	err = cache.Delete(ctx, "delete_key")
	require.NoError(t, err)

	// Get should fail
	_, err = cache.Get(ctx, "delete_key")
	assert.Error(t, err)
}

func TestRedisCache_Delete_NonExistentKey(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Delete non-existent key (should not error)
	err := cache.Delete(ctx, "non_existent")
	assert.NoError(t, err)
}

func TestRedisCache_Delete_MultipleKeys(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Set multiple values
	err := cache.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)
	err = cache.Set(ctx, "key2", "value2", 0)
	require.NoError(t, err)
	err = cache.Set(ctx, "key3", "value3", 0)
	require.NoError(t, err)

	// Delete multiple keys
	err = cache.Delete(ctx, "key1", "key2", "key3")
	require.NoError(t, err)

	// All should be deleted
	_, err = cache.Get(ctx, "key1")
	assert.Error(t, err)
	_, err = cache.Get(ctx, "key2")
	assert.Error(t, err)
	_, err = cache.Get(ctx, "key3")
	assert.Error(t, err)
}

// ==================== Exists Tests ====================

func TestRedisCache_Exists_True(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Set value
	err := cache.Set(ctx, "exists_key", "value", 0)
	require.NoError(t, err)

	// Check existence
	count, err := cache.Exists(ctx, "exists_key")
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRedisCache_Exists_False(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Check non-existent key
	count, err := cache.Exists(ctx, "non_existent")
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestRedisCache_Exists_MultipleKeys(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Set two keys
	err := cache.Set(ctx, "key1", "value1", 0)
	require.NoError(t, err)
	err = cache.Set(ctx, "key2", "value2", 0)
	require.NoError(t, err)

	// Check existence of both keys
	count, err := cache.Exists(ctx, "key1", "key2", "key3")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count) // Only 2 exist
}

// ==================== Expire Tests ====================

func TestRedisCache_Expire_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Set value without TTL
	err := cache.Set(ctx, "expire_key", "value", 0)
	require.NoError(t, err)

	// Set expiration
	err = cache.Expire(ctx, "expire_key", 1*time.Second)
	require.NoError(t, err)

	// Value should exist
	value, err := cache.Get(ctx, "expire_key")
	require.NoError(t, err)
	assert.Equal(t, "value", value)

	// Fast-forward time
	s.FastForward(2 * time.Second)

	// Value should be expired
	_, err = cache.Get(ctx, "expire_key")
	assert.Error(t, err)
}

func TestRedisCache_Expire_NonExistentKey(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Try to expire non-existent key (should not error)
	err := cache.Expire(ctx, "non_existent", 1*time.Second)
	assert.NoError(t, err)
}

// ==================== Increment/Decrement Tests ====================

func TestRedisCache_Increment_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Increment non-existent key (starts at 0)
	value, err := cache.Increment(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), value)

	// Increment again
	value, err = cache.Increment(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(2), value)

	// Increment again
	value, err = cache.Increment(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(3), value)
}

func TestRedisCache_Decrement_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Set initial value
	_, err := cache.Increment(ctx, "counter")
	require.NoError(t, err)
	_, err = cache.Increment(ctx, "counter")
	require.NoError(t, err)
	_, err = cache.Increment(ctx, "counter")
	require.NoError(t, err)
	// counter = 3

	// Decrement
	value, err := cache.Decrement(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(2), value)

	// Decrement again
	value, err = cache.Decrement(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), value)
}

func TestRedisCache_Decrement_BelowZero(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Decrement non-existent key (starts at 0, goes to -1)
	value, err := cache.Decrement(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(-1), value)

	// Decrement again
	value, err = cache.Decrement(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(-2), value)
}

// ==================== Stats Tests ====================

func TestRedisCache_Stats_Success(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	stats := cache.Stats()
	assert.NotNil(t, stats)
	// Stats fields depend on actual Redis operations
	// Just verify the struct is returned
	assert.IsType(t, &CacheStats{}, stats)
}

// ==================== Integration Tests ====================

func TestRedisCache_Integration_CompleteFlow(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// 1. Set multiple keys
	err := cache.Set(ctx, "user:1", "Alice", 0)
	require.NoError(t, err)
	err = cache.Set(ctx, "user:2", "Bob", 0)
	require.NoError(t, err)
	err = cache.Set(ctx, "user:3", "Charlie", 0)
	require.NoError(t, err)

	// 2. Check existence
	count, err := cache.Exists(ctx, "user:1", "user:2", "user:3")
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// 3. Get values
	value, err := cache.Get(ctx, "user:1")
	require.NoError(t, err)
	assert.Equal(t, "Alice", value)

	// 4. Delete one key
	err = cache.Delete(ctx, "user:2")
	require.NoError(t, err)

	// 5. Verify deletion
	_, err = cache.Get(ctx, "user:2")
	assert.Error(t, err)

	// 6. Increment counter
	counter, err := cache.Increment(ctx, "total_users")
	require.NoError(t, err)
	assert.Equal(t, int64(1), counter)

	// 7. Set expiration on remaining keys
	err = cache.Expire(ctx, "user:1", 1*time.Second)
	require.NoError(t, err)

	// 8. Fast-forward time
	s.FastForward(2 * time.Second)

	// 9. Verify expiration
	_, err = cache.Get(ctx, "user:1")
	assert.Error(t, err)

	// 10. Get stats
	stats := cache.Stats()
	assert.NotNil(t, stats)
}

func TestRedisCache_Integration_ConcurrentAccess(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	cache := setupCache(t, s)
	defer cache.Close()

	ctx := context.Background()

	// Concurrent increments
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := cache.Increment(ctx, "concurrent_counter")
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Get final value
	value, err := cache.Get(ctx, "concurrent_counter")
	require.NoError(t, err)
	assert.Equal(t, "10", value)
}

// ==================== Helper Functions ====================

func setupCache(t *testing.T, s *miniredis.Miniredis) *RedisCache {
	cfg := config.RedisConfig{
		URL:      "redis://" + s.Addr(),
		Password: "",
		DB:       0,
		PoolSize: 10,
	}

	cache, err := NewRedisCache(cfg)
	require.NoError(t, err)
	return cache
}
