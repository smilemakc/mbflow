package rest

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	mu      sync.RWMutex
	clients map[string]*clientInfo
	limit   int
	window  time.Duration
	cleanup time.Duration
}

type clientInfo struct {
	attempts  int
	firstSeen time.Time
	blocked   bool
	blockedAt time.Time
}

// NewRateLimiter creates a new rate limiter
// limit: max attempts per window
// window: time window for counting attempts
// blockDuration: how long to block after exceeding limit
func NewRateLimiter(limit int, window, blockDuration time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*clientInfo),
		limit:   limit,
		window:  window,
		cleanup: blockDuration,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Middleware returns a gin middleware for rate limiting
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !rl.Allow(clientIP) {
			respondErrorWithDetails(c, http.StatusTooManyRequests, "too many requests", "RATE_LIMIT_EXCEEDED", map[string]interface{}{
				"retry_after": int(rl.cleanup.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Allow checks if a request from the given key should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[key]

	if !exists {
		rl.clients[key] = &clientInfo{
			attempts:  1,
			firstSeen: now,
		}
		return true
	}

	// Check if blocked
	if client.blocked {
		if now.Sub(client.blockedAt) > rl.cleanup {
			// Unblock after cleanup period
			client.blocked = false
			client.attempts = 1
			client.firstSeen = now
			return true
		}
		return false
	}

	// Check if window has expired
	if now.Sub(client.firstSeen) > rl.window {
		client.attempts = 1
		client.firstSeen = now
		return true
	}

	// Increment attempts
	client.attempts++

	// Check if limit exceeded
	if client.attempts > rl.limit {
		client.blocked = true
		client.blockedAt = now
		return false
	}

	return true
}

// Reset resets the rate limit for a specific key
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.clients, key)
}

// cleanupLoop periodically removes expired entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, client := range rl.clients {
			// Remove if not blocked and window expired
			if !client.blocked && now.Sub(client.firstSeen) > rl.window {
				delete(rl.clients, key)
			}
			// Remove if blocked period expired
			if client.blocked && now.Sub(client.blockedAt) > rl.cleanup*2 {
				delete(rl.clients, key)
			}
		}
		rl.mu.Unlock()
	}
}

// LoginRateLimiter provides specialized rate limiting for login attempts
type LoginRateLimiter struct {
	rl              *RateLimiter
	maxAttempts     int
	lockoutDuration time.Duration
}

// NewLoginRateLimiter creates a new login-specific rate limiter
func NewLoginRateLimiter(maxAttempts int, windowDuration, lockoutDuration time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{
		rl:              NewRateLimiter(maxAttempts, windowDuration, lockoutDuration),
		maxAttempts:     maxAttempts,
		lockoutDuration: lockoutDuration,
	}
}

// Middleware returns the rate limiting middleware
func (lrl *LoginRateLimiter) Middleware() gin.HandlerFunc {
	return lrl.rl.Middleware()
}

// RecordFailedAttempt records a failed login attempt
func (lrl *LoginRateLimiter) RecordFailedAttempt(key string) {
	lrl.rl.mu.Lock()
	defer lrl.rl.mu.Unlock()

	client, exists := lrl.rl.clients[key]
	if !exists {
		lrl.rl.clients[key] = &clientInfo{
			attempts:  1,
			firstSeen: time.Now(),
		}
		return
	}

	client.attempts++
	if client.attempts >= lrl.maxAttempts {
		client.blocked = true
		client.blockedAt = time.Now()
	}
}

// RecordSuccessfulLogin resets the rate limit for a successful login
func (lrl *LoginRateLimiter) RecordSuccessfulLogin(key string) {
	lrl.rl.Reset(key)
}

// IsBlocked checks if the key is currently blocked
func (lrl *LoginRateLimiter) IsBlocked(key string) bool {
	lrl.rl.mu.RLock()
	defer lrl.rl.mu.RUnlock()

	client, exists := lrl.rl.clients[key]
	if !exists {
		return false
	}

	if !client.blocked {
		return false
	}

	// Check if block has expired
	if time.Since(client.blockedAt) > lrl.lockoutDuration {
		return false
	}

	return true
}

// GetRemainingAttempts returns the number of remaining login attempts
func (lrl *LoginRateLimiter) GetRemainingAttempts(key string) int {
	lrl.rl.mu.RLock()
	defer lrl.rl.mu.RUnlock()

	client, exists := lrl.rl.clients[key]
	if !exists {
		return lrl.maxAttempts
	}

	if client.blocked {
		return 0
	}

	remaining := lrl.maxAttempts - client.attempts
	if remaining < 0 {
		return 0
	}
	return remaining
}
