package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exerciseLoginLimiter checks the lockout policy shared by every LoginLimiter.
func exerciseLoginLimiter(t *testing.T, l LoginLimiter) {
	t.Helper()
	ctx := context.Background()
	key := LoginAttemptKey("10.0.0.1", " User@Example.com ")
	other := LoginAttemptKey("10.0.0.1", "other@example.com")

	assert.False(t, l.IsBlocked(ctx, key))
	assert.Equal(t, 3, l.RemainingAttempts(ctx, key))

	l.RecordFailedAttempt(ctx, key)
	l.RecordFailedAttempt(ctx, key)
	assert.Equal(t, 1, l.RemainingAttempts(ctx, key))
	assert.False(t, l.IsBlocked(ctx, key))

	l.RecordFailedAttempt(ctx, key)
	assert.True(t, l.IsBlocked(ctx, key), "locked out after max attempts")
	assert.Zero(t, l.RemainingAttempts(ctx, key))
	assert.False(t, l.IsBlocked(ctx, other), "other accounts are unaffected")

	l.RecordSuccessfulLogin(ctx, key)
	assert.False(t, l.IsBlocked(ctx, key))
	assert.Equal(t, 3, l.RemainingAttempts(ctx, key))
}

func TestMemoryLoginLimiter(t *testing.T) {
	exerciseLoginLimiter(t, NewMemoryLoginLimiter(NewLoginRateLimiter(3, time.Minute, time.Minute)))
}

func TestRedisLoginLimiter(t *testing.T) {
	_, client := setupMiniRedis(t)
	exerciseLoginLimiter(t, NewRedisLoginLimiter(NewRedisLoginRateLimiter(client, 3, time.Minute, time.Minute)))
}

func TestRedisLoginLimiter_SharedAcrossInstances(t *testing.T) {
	_, client := setupMiniRedis(t)
	ctx := context.Background()
	replicaA := NewRedisLoginLimiter(NewRedisLoginRateLimiter(client, 2, time.Minute, time.Minute))
	replicaB := NewRedisLoginLimiter(NewRedisLoginRateLimiter(client, 2, time.Minute, time.Minute))
	key := LoginAttemptKey("10.0.0.2", "a@b.c")

	replicaA.RecordFailedAttempt(ctx, key)
	replicaB.RecordFailedAttempt(ctx, key)
	assert.True(t, replicaA.IsBlocked(ctx, key), "a lockout recorded on one replica applies on the other")
}

func TestRedisLoginLimiter_FailsOpenWhenRedisIsDown(t *testing.T) {
	mr, client := setupMiniRedis(t)
	l := NewRedisLoginLimiter(NewRedisLoginRateLimiter(client, 3, time.Minute, time.Minute))
	mr.Close()

	ctx := context.Background()
	l.RecordFailedAttempt(ctx, "k")
	l.RecordSuccessfulLogin(ctx, "k")
	assert.False(t, l.IsBlocked(ctx, "k"))
	assert.Equal(t, 3, l.RemainingAttempts(ctx, "k"))
}

func TestRedisLoginRateLimiter_MiddlewarePerIPCap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, client := setupMiniRedis(t)
	lrl := NewRedisLoginRateLimiter(client, 1, time.Minute, time.Minute)
	l := NewRedisLoginLimiter(lrl)

	router := gin.New()
	router.POST("/login", l.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })

	// The per-IP request cap is maxAttempts*loginRequestsPerIPFactor, like
	// the in-memory limiter, and does not use the failed-attempt counters.
	for i := 0; i < loginRequestsPerIPFactor; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", nil))
		require.Equal(t, http.StatusOK, w.Code, "request %d", i+1)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", nil))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	blocked, err := lrl.IsBlocked(context.Background(), "192.0.2.1")
	require.NoError(t, err)
	assert.False(t, blocked, "per-IP request counters are separate from account lockouts")
}

func TestMemoryLoginLimiter_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	l := NewMemoryLoginLimiter(NewLoginRateLimiter(1, time.Minute, time.Minute))
	router := gin.New()
	router.POST("/login", l.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })

	codes := map[int]int{}
	for i := 0; i <= loginRequestsPerIPFactor; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/login", nil))
		codes[w.Code]++
	}
	assert.Equal(t, loginRequestsPerIPFactor, codes[http.StatusOK])
	assert.Equal(t, 1, codes[http.StatusTooManyRequests])
}

func TestNewAuthHandlers_LimiterWiring(t *testing.T) {
	assert.Nil(t, NewAuthHandlers(nil, nil, nil).rateLimiter, "a nil limiter disables lockout instead of wrapping a nil pointer")
	assert.NotNil(t, NewAuthHandlers(nil, nil, NewLoginRateLimiter(1, time.Minute, time.Minute)).rateLimiter)
	assert.Nil(t, NewAuthHandlersWithLimiter(nil, nil, nil).rateLimiter)
}
