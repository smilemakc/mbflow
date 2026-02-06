package rest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter provides Redis-backed rate limiting for distributed deployments
type RedisRateLimiter struct {
	client        redis.UniversalClient
	keyPrefix     string
	limit         int
	window        time.Duration
	blockDuration time.Duration
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
// client: Redis client
// keyPrefix: prefix for Redis keys (e.g., "ratelimit:api:")
// limit: max attempts per window
// window: time window for counting attempts
// blockDuration: how long to block after exceeding limit
func NewRedisRateLimiter(client redis.UniversalClient, keyPrefix string, limit int, window, blockDuration time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:        client,
		keyPrefix:     keyPrefix,
		limit:         limit,
		window:        window,
		blockDuration: blockDuration,
	}
}

// Middleware returns a gin middleware for rate limiting
func (rl *RedisRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		allowed, retryAfter, err := rl.Allow(c.Request.Context(), clientIP)
		if err != nil {
			// On Redis error, allow the request but log it
			c.Next()
			return
		}

		if !allowed {
			respondErrorWithDetails(c, http.StatusTooManyRequests, "too many requests", "RATE_LIMIT_EXCEEDED", map[string]interface{}{
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Allow checks if a request from the given key should be allowed
// Returns: allowed, retry_after_seconds, error
func (rl *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, int, error) {
	blockKey := rl.keyPrefix + "block:" + key
	countKey := rl.keyPrefix + "count:" + key

	// Check if blocked
	blocked, err := rl.client.Exists(ctx, blockKey).Result()
	if err != nil {
		return false, 0, fmt.Errorf("redis exists error: %w", err)
	}

	if blocked > 0 {
		ttl, err := rl.client.TTL(ctx, blockKey).Result()
		if err != nil {
			return false, int(rl.blockDuration.Seconds()), nil
		}
		return false, int(ttl.Seconds()), nil
	}

	// Increment counter
	count, err := rl.client.Incr(ctx, countKey).Result()
	if err != nil {
		return false, 0, fmt.Errorf("redis incr error: %w", err)
	}

	// Set expiry on first request
	if count == 1 {
		if err := rl.client.Expire(ctx, countKey, rl.window).Err(); err != nil {
			return false, 0, fmt.Errorf("redis expire error: %w", err)
		}
	}

	// Check if limit exceeded
	if int(count) > rl.limit {
		// Block the client
		if err := rl.client.Set(ctx, blockKey, "1", rl.blockDuration).Err(); err != nil {
			return false, 0, fmt.Errorf("redis set block error: %w", err)
		}
		return false, int(rl.blockDuration.Seconds()), nil
	}

	return true, 0, nil
}

// Reset resets the rate limit for a specific key
func (rl *RedisRateLimiter) Reset(ctx context.Context, key string) error {
	blockKey := rl.keyPrefix + "block:" + key
	countKey := rl.keyPrefix + "count:" + key

	pipe := rl.client.Pipeline()
	pipe.Del(ctx, blockKey)
	pipe.Del(ctx, countKey)
	_, err := pipe.Exec(ctx)
	return err
}

// RedisLoginRateLimiter provides specialized Redis-backed rate limiting for login attempts
type RedisLoginRateLimiter struct {
	client          redis.UniversalClient
	keyPrefix       string
	maxAttempts     int
	windowDuration  time.Duration
	lockoutDuration time.Duration
}

// NewRedisLoginRateLimiter creates a new Redis-backed login rate limiter
func NewRedisLoginRateLimiter(client redis.UniversalClient, maxAttempts int, windowDuration, lockoutDuration time.Duration) *RedisLoginRateLimiter {
	return &RedisLoginRateLimiter{
		client:          client,
		keyPrefix:       "ratelimit:login:",
		maxAttempts:     maxAttempts,
		windowDuration:  windowDuration,
		lockoutDuration: lockoutDuration,
	}
}

// Middleware returns the rate limiting middleware
func (lrl *RedisLoginRateLimiter) Middleware() gin.HandlerFunc {
	rl := NewRedisRateLimiter(lrl.client, lrl.keyPrefix, lrl.maxAttempts, lrl.windowDuration, lrl.lockoutDuration)
	return rl.Middleware()
}

// RecordFailedAttempt records a failed login attempt
func (lrl *RedisLoginRateLimiter) RecordFailedAttempt(ctx context.Context, key string) error {
	countKey := lrl.keyPrefix + "count:" + key
	blockKey := lrl.keyPrefix + "block:" + key

	count, err := lrl.client.Incr(ctx, countKey).Result()
	if err != nil {
		return fmt.Errorf("redis incr error: %w", err)
	}

	// Set expiry on first attempt
	if count == 1 {
		if err := lrl.client.Expire(ctx, countKey, lrl.windowDuration).Err(); err != nil {
			return fmt.Errorf("redis expire error: %w", err)
		}
	}

	// Block if max attempts reached
	if int(count) >= lrl.maxAttempts {
		if err := lrl.client.Set(ctx, blockKey, "1", lrl.lockoutDuration).Err(); err != nil {
			return fmt.Errorf("redis set block error: %w", err)
		}
	}

	return nil
}

// RecordSuccessfulLogin resets the rate limit for a successful login
func (lrl *RedisLoginRateLimiter) RecordSuccessfulLogin(ctx context.Context, key string) error {
	countKey := lrl.keyPrefix + "count:" + key
	blockKey := lrl.keyPrefix + "block:" + key

	pipe := lrl.client.Pipeline()
	pipe.Del(ctx, countKey)
	pipe.Del(ctx, blockKey)
	_, err := pipe.Exec(ctx)
	return err
}

// IsBlocked checks if the key is currently blocked
func (lrl *RedisLoginRateLimiter) IsBlocked(ctx context.Context, key string) (bool, error) {
	blockKey := lrl.keyPrefix + "block:" + key

	exists, err := lrl.client.Exists(ctx, blockKey).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists error: %w", err)
	}

	return exists > 0, nil
}

// GetRemainingAttempts returns the number of remaining login attempts
func (lrl *RedisLoginRateLimiter) GetRemainingAttempts(ctx context.Context, key string) (int, error) {
	countKey := lrl.keyPrefix + "count:" + key
	blockKey := lrl.keyPrefix + "block:" + key

	// Check if blocked first
	blocked, err := lrl.client.Exists(ctx, blockKey).Result()
	if err != nil {
		return 0, fmt.Errorf("redis exists error: %w", err)
	}
	if blocked > 0 {
		return 0, nil
	}

	// Get current count
	count, err := lrl.client.Get(ctx, countKey).Int()
	if err == redis.Nil {
		return lrl.maxAttempts, nil
	}
	if err != nil {
		return 0, fmt.Errorf("redis get error: %w", err)
	}

	remaining := lrl.maxAttempts - count
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}
