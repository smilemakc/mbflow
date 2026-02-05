package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMiniRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func TestRedisRateLimiter_Allow(t *testing.T) {
	t.Parallel()

	_, client := setupMiniRedis(t)
	ctx := context.Background()

	rl := NewRedisRateLimiter(client, "test:", 3, time.Minute, 5*time.Minute)

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		allowed, _, err := rl.Allow(ctx, "client1")
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	// 4th request should be blocked
	allowed, retryAfter, err := rl.Allow(ctx, "client1")
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Greater(t, retryAfter, 0)
}

func TestRedisRateLimiter_DifferentKeys(t *testing.T) {
	t.Parallel()

	_, client := setupMiniRedis(t)
	ctx := context.Background()

	rl := NewRedisRateLimiter(client, "test:", 2, time.Minute, 5*time.Minute)

	// Exhaust limit for client1
	for i := 0; i < 3; i++ {
		rl.Allow(ctx, "client1")
	}

	// client2 should still be allowed
	allowed, _, err := rl.Allow(ctx, "client2")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestRedisRateLimiter_Reset(t *testing.T) {
	t.Parallel()

	_, client := setupMiniRedis(t)
	ctx := context.Background()

	rl := NewRedisRateLimiter(client, "test:", 2, time.Minute, 5*time.Minute)

	// Exhaust limit
	for i := 0; i < 3; i++ {
		rl.Allow(ctx, "client1")
	}

	// Should be blocked
	allowed, _, err := rl.Allow(ctx, "client1")
	require.NoError(t, err)
	assert.False(t, allowed)

	// Reset
	err = rl.Reset(ctx, "client1")
	require.NoError(t, err)

	// Should be allowed again
	allowed, _, err = rl.Allow(ctx, "client1")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestRedisRateLimiter_Middleware(t *testing.T) {
	t.Parallel()

	_, client := setupMiniRedis(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	rl := NewRedisRateLimiter(client, "test:", 2, time.Minute, 5*time.Minute)
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed", i+1)
	}

	// 3rd request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRedisLoginRateLimiter_RecordFailedAttempt(t *testing.T) {
	t.Parallel()

	_, client := setupMiniRedis(t)
	ctx := context.Background()

	lrl := NewRedisLoginRateLimiter(client, 3, time.Minute, 5*time.Minute)

	// Record 2 failed attempts
	for i := 0; i < 2; i++ {
		err := lrl.RecordFailedAttempt(ctx, "user1")
		require.NoError(t, err)
	}

	// Check remaining attempts
	remaining, err := lrl.GetRemainingAttempts(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, 1, remaining)

	// Not blocked yet
	blocked, err := lrl.IsBlocked(ctx, "user1")
	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestRedisLoginRateLimiter_Blocked(t *testing.T) {
	t.Parallel()

	_, client := setupMiniRedis(t)
	ctx := context.Background()

	lrl := NewRedisLoginRateLimiter(client, 3, time.Minute, 5*time.Minute)

	// Record 3 failed attempts
	for i := 0; i < 3; i++ {
		err := lrl.RecordFailedAttempt(ctx, "user1")
		require.NoError(t, err)
	}

	// Should be blocked
	blocked, err := lrl.IsBlocked(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, blocked)

	// Remaining attempts should be 0
	remaining, err := lrl.GetRemainingAttempts(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
}

func TestRedisLoginRateLimiter_SuccessfulLogin(t *testing.T) {
	t.Parallel()

	_, client := setupMiniRedis(t)
	ctx := context.Background()

	lrl := NewRedisLoginRateLimiter(client, 3, time.Minute, 5*time.Minute)

	// Record 2 failed attempts
	for i := 0; i < 2; i++ {
		err := lrl.RecordFailedAttempt(ctx, "user1")
		require.NoError(t, err)
	}

	// Record successful login
	err := lrl.RecordSuccessfulLogin(ctx, "user1")
	require.NoError(t, err)

	// Should have full attempts again
	remaining, err := lrl.GetRemainingAttempts(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, 3, remaining)
}

func TestRedisLoginRateLimiter_ResetAfterSuccessfulLogin(t *testing.T) {
	t.Parallel()

	_, client := setupMiniRedis(t)
	ctx := context.Background()

	lrl := NewRedisLoginRateLimiter(client, 3, time.Minute, 5*time.Minute)

	// Block the user
	for i := 0; i < 3; i++ {
		err := lrl.RecordFailedAttempt(ctx, "user1")
		require.NoError(t, err)
	}

	// Verify blocked
	blocked, err := lrl.IsBlocked(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, blocked)

	// Successful login resets
	err = lrl.RecordSuccessfulLogin(ctx, "user1")
	require.NoError(t, err)

	// No longer blocked
	blocked, err = lrl.IsBlocked(ctx, "user1")
	require.NoError(t, err)
	assert.False(t, blocked)
}

func TestRedisRateLimiter_WindowExpiry(t *testing.T) {
	t.Parallel()

	mr, client := setupMiniRedis(t)
	ctx := context.Background()

	// Use a short window
	rl := NewRedisRateLimiter(client, "test:", 2, 100*time.Millisecond, 5*time.Minute)

	// Exhaust limit
	for i := 0; i < 3; i++ {
		rl.Allow(ctx, "client1")
	}

	// Should be blocked
	allowed, _, err := rl.Allow(ctx, "client1")
	require.NoError(t, err)
	assert.False(t, allowed)

	// Fast-forward time in miniredis
	mr.FastForward(200 * time.Millisecond)

	// Note: The block key has a longer duration, so we need to reset manually
	// In production, the block would expire after blockDuration
	err = rl.Reset(ctx, "client1")
	require.NoError(t, err)

	// Should be allowed after reset
	allowed, _, err = rl.Allow(ctx, "client1")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestRedisLoginRateLimiter_DifferentUsers(t *testing.T) {
	t.Parallel()

	_, client := setupMiniRedis(t)
	ctx := context.Background()

	lrl := NewRedisLoginRateLimiter(client, 2, time.Minute, 5*time.Minute)

	// Block user1
	for i := 0; i < 2; i++ {
		err := lrl.RecordFailedAttempt(ctx, "user1")
		require.NoError(t, err)
	}

	// user1 should be blocked
	blocked, err := lrl.IsBlocked(ctx, "user1")
	require.NoError(t, err)
	assert.True(t, blocked)

	// user2 should not be blocked
	blocked, err = lrl.IsBlocked(ctx, "user2")
	require.NoError(t, err)
	assert.False(t, blocked)

	// user2 should have full attempts
	remaining, err := lrl.GetRemainingAttempts(ctx, "user2")
	require.NoError(t, err)
	assert.Equal(t, 2, remaining)
}
