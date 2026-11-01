package rest

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
)

// LoginLimiter is the login lockout policy used by the auth handlers. The
// in-memory implementation keeps counters per process; the Redis-backed one
// shares them between all replicas.
type LoginLimiter interface {
	// Middleware caps the total number of login requests per client IP.
	Middleware() gin.HandlerFunc
	// IsBlocked reports whether key (see LoginAttemptKey) is locked out.
	IsBlocked(ctx context.Context, key string) bool
	// RecordFailedAttempt counts a failed login for key.
	RecordFailedAttempt(ctx context.Context, key string)
	// RecordSuccessfulLogin clears the failed attempts of key.
	RecordSuccessfulLogin(ctx context.Context, key string)
	// RemainingAttempts returns how many failed logins key has left.
	RemainingAttempts(ctx context.Context, key string) int
}

// NewMemoryLoginLimiter adapts the per-process LoginRateLimiter to LoginLimiter.
func NewMemoryLoginLimiter(l *LoginRateLimiter) LoginLimiter {
	return memoryLoginLimiter{l: l}
}

type memoryLoginLimiter struct{ l *LoginRateLimiter }

func (m memoryLoginLimiter) Middleware() gin.HandlerFunc { return m.l.Middleware() }

func (m memoryLoginLimiter) IsBlocked(_ context.Context, key string) bool {
	return m.l.IsBlocked(key)
}

func (m memoryLoginLimiter) RecordFailedAttempt(_ context.Context, key string) {
	m.l.RecordFailedAttempt(key)
}

func (m memoryLoginLimiter) RecordSuccessfulLogin(_ context.Context, key string) {
	m.l.RecordSuccessfulLogin(key)
}

func (m memoryLoginLimiter) RemainingAttempts(_ context.Context, key string) int {
	return m.l.GetRemainingAttempts(key)
}

// NewRedisLoginLimiter adapts RedisLoginRateLimiter to LoginLimiter. Redis
// errors are logged and fail open (the login proceeds to credential checks),
// consistent with RedisRateLimiter.Middleware.
func NewRedisLoginLimiter(l *RedisLoginRateLimiter) LoginLimiter {
	return redisLoginLimiter{l: l}
}

type redisLoginLimiter struct{ l *RedisLoginRateLimiter }

func (r redisLoginLimiter) Middleware() gin.HandlerFunc { return r.l.Middleware() }

func (r redisLoginLimiter) IsBlocked(ctx context.Context, key string) bool {
	blocked, err := r.l.IsBlocked(ctx, key)
	if err != nil {
		slog.Warn("login rate limiter unavailable", "op", "is_blocked", "error", err)
		return false
	}
	return blocked
}

func (r redisLoginLimiter) RecordFailedAttempt(ctx context.Context, key string) {
	if err := r.l.RecordFailedAttempt(ctx, key); err != nil {
		slog.Warn("login rate limiter unavailable", "op", "record_failed", "error", err)
	}
}

func (r redisLoginLimiter) RecordSuccessfulLogin(ctx context.Context, key string) {
	if err := r.l.RecordSuccessfulLogin(ctx, key); err != nil {
		slog.Warn("login rate limiter unavailable", "op", "record_success", "error", err)
	}
}

func (r redisLoginLimiter) RemainingAttempts(ctx context.Context, key string) int {
	remaining, err := r.l.GetRemainingAttempts(ctx, key)
	if err != nil {
		slog.Warn("login rate limiter unavailable", "op", "remaining_attempts", "error", err)
		return r.l.maxAttempts
	}
	return remaining
}
