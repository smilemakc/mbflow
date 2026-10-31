package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
)

func rc2Logger() *logger.Logger {
	return logger.New(config.LoggingConfig{Level: "error", Format: "text"})
}

func TestRC2_LoggingMiddleware_RequestIDAndLevels(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	m := NewLoggingMiddleware(rc2Logger())
	r := gin.New()
	r.Use(m.RequestLogger())
	r.GET("/status/:code", func(c *gin.Context) {
		if c.Query("user") != "" {
			c.Set(ContextKeyUserID, c.Query("user"))
		}
		if c.Query("err") != "" {
			_ = c.Error(assert.AnError)
		}
		c.JSON(map[string]int{"ok": 200, "warn": 404, "fail": 500}[c.Param("code")], gin.H{"request_id": GetRequestID(c)})
	})

	// Valid client-supplied request ID is propagated.
	req := httptest.NewRequest(http.MethodGet, "/status/ok?token=secret&user=u1", nil)
	req.Header.Set(RequestIDHeader, "client-id_1.2:3")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "client-id_1.2:3", w.Header().Get(RequestIDHeader))
	assert.Equal(t, "client-id_1.2:3", rc2DecodeMap(t, w)["request_id"])

	// Invalid request IDs (log injection) are replaced by a generated UUID.
	for _, path := range []string{"/status/warn", "/status/fail?err=1"} {
		req = httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set(RequestIDHeader, "evil\nid")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		generated := w.Header().Get(RequestIDHeader)
		_, err := uuid.Parse(generated)
		assert.NoError(t, err, "expected generated UUID, got %q", generated)
		assert.Equal(t, generated, rc2DecodeMap(t, w)["request_id"])
	}
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRC2_GetRequestID_Missing(t *testing.T) {
	t.Parallel()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	assert.Equal(t, "", GetRequestID(c))
}

func TestRC2_RedactQuery_Unparseable(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "[unparseable]", redactQuery("a=%zz"))
}

func TestRC2_RecoveryMiddleware(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyRequestID, "rid-42")
		c.Set(ContextKeyUserID, "user-1")
		c.Next()
	})
	r.Use(NewRecoveryMiddleware(rc2Logger()).Recovery())
	r.GET("/panic", func(c *gin.Context) { panic("kaboom") })
	r.GET("/fine", func(c *gin.Context) { c.String(http.StatusOK, "fine") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)
	body := rc2DecodeMap(t, w)
	assert.Equal(t, "INTERNAL_ERROR", body["code"])
	assert.Contains(t, body["message"], "rid-42")
	assert.NotContains(t, w.Body.String(), "kaboom", "panic value must not leak to the client")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/fine", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "fine", w.Body.String())
}

func TestRC2_BodySizeMiddleware_AllowsSmallBodies(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewBodySizeMiddleware(rc2Logger(), 1024).LimitBodySize())
	r.POST("/echo", func(c *gin.Context) {
		var m map[string]any
		require.NoError(t, c.ShouldBindJSON(&m))
		c.JSON(http.StatusOK, m)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"a":"b"}`)))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "b", rc2DecodeMap(t, w)["a"])
}

func TestRC2_RateLimiter_MiddlewareBlocksAndRecovers(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	rl := NewRateLimiter(2, time.Hour, 30*time.Millisecond)
	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	do := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.1.2.3:1234"
		r.ServeHTTP(w, req)
		return w
	}

	assert.Equal(t, http.StatusNoContent, do().Code)
	assert.Equal(t, http.StatusNoContent, do().Code)
	w := do()
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	body := rc2DecodeMap(t, w)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", body["code"])
	assert.Contains(t, body, "details")
	// Still blocked while the block lasts.
	assert.Equal(t, http.StatusTooManyRequests, do().Code)

	// After the block duration the client is let through again.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, http.StatusNoContent, do().Code)
}

func TestRC2_RateLimiter_WindowExpiryResetsCount(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1, 20*time.Millisecond, time.Hour)
	assert.True(t, rl.Allow("k"))
	time.Sleep(30 * time.Millisecond)
	assert.True(t, rl.Allow("k"), "a new window starts after expiry")
	assert.False(t, rl.Allow("k"))
	rl.Reset("k")
	assert.True(t, rl.Allow("k"))
}

func TestRC2_LoginRateLimiter_ExpiryAndRemaining(t *testing.T) {
	t.Parallel()
	lrl := NewLoginRateLimiter(2, time.Hour, 20*time.Millisecond)
	key := LoginAttemptKey("1.1.1.1", "  User@Example.COM ")
	assert.Equal(t, "1.1.1.1|user@example.com", key)

	assert.False(t, lrl.IsBlocked(key))
	assert.Equal(t, 2, lrl.GetRemainingAttempts(key))
	lrl.RecordFailedAttempt(key)
	assert.Equal(t, 1, lrl.GetRemainingAttempts(key))
	lrl.RecordFailedAttempt(key)
	assert.True(t, lrl.IsBlocked(key))
	assert.Equal(t, 0, lrl.GetRemainingAttempts(key))

	time.Sleep(30 * time.Millisecond)
	assert.False(t, lrl.IsBlocked(key), "lockout expires")

	lrl.RecordSuccessfulLogin(key)
	assert.Equal(t, 2, lrl.GetRemainingAttempts(key))

	// Attempts beyond the maximum never yield a negative remainder.
	other := LoginAttemptKey("1.1.1.1", "other@example.com")
	lrl.rl.mu.Lock()
	lrl.rl.clients[other] = &clientInfo{attempts: 5, firstSeen: time.Now()}
	lrl.rl.mu.Unlock()
	assert.Equal(t, 0, lrl.GetRemainingAttempts(other))
}

func TestRC2_LoginRateLimiter_PerIPMiddleware(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	// 1 attempt * factor 20 = 20 login requests per IP.
	lrl := NewLoginRateLimiter(1, time.Hour, time.Hour)
	r := gin.New()
	r.POST("/login", lrl.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })

	codes := map[int]int{}
	for i := 0; i < loginRequestsPerIPFactor+1; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "192.0.2.10:999"
		r.ServeHTTP(w, req)
		codes[w.Code]++
	}
	assert.Equal(t, loginRequestsPerIPFactor, codes[http.StatusOK])
	assert.Equal(t, 1, codes[http.StatusTooManyRequests])
}

func TestRC2_RedisRateLimiter_MiddlewareAndErrors(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	mr, client := setupMiniRedis(t)
	rl := NewRedisRateLimiter(client, "rc2:", 1, time.Minute, time.Minute)

	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	do := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.5:1"
		r.ServeHTTP(w, req)
		return w
	}

	assert.Equal(t, http.StatusNoContent, do().Code)
	w := do()
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	details := rc2DecodeMap(t, w)["details"].(map[string]any)
	assert.Greater(t, details["retry_after"].(float64), float64(0))

	// Already blocked: TTL of the block key is reported.
	allowed, retry, err := rl.Allow(context.Background(), "203.0.113.5")
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Greater(t, retry, 0)

	// Redis failures fail open in the middleware and surface from Allow.
	mr.SetError("ERR rc2 injected")
	assert.Equal(t, http.StatusNoContent, do().Code)
	_, _, err = rl.Allow(context.Background(), "fresh")
	assert.Error(t, err)
	mr.SetError("")

	// Reset clears block and counter.
	require.NoError(t, rl.Reset(context.Background(), "203.0.113.5"))
	assert.Equal(t, http.StatusNoContent, do().Code)
}

func TestRC2_RedisLoginRateLimiter_ErrorsAndMiddleware(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	mr, client := setupMiniRedis(t)
	lrl := NewRedisLoginRateLimiter(client, 2, time.Minute, time.Minute)

	r := gin.New()
	r.POST("/login", lrl.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })
	// The per-IP cap is generous (a multiple of maxAttempts); count how
	// many requests pass before the IP is throttled.
	passed := 0
	blocked := false
	for i := 0; i < 1000 && !blocked; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "198.51.100.7:1"
		r.ServeHTTP(w, req)
		switch w.Code {
		case http.StatusOK:
			passed++
		case http.StatusTooManyRequests:
			blocked = true
		default:
			t.Fatalf("unexpected status %d", w.Code)
		}
	}
	assert.True(t, blocked, "login requests from one IP are eventually throttled")
	assert.GreaterOrEqual(t, passed, lrl.maxAttempts)
	// The per-IP counter does not consume per-account failed attempts.
	remainingAfterIP, err := lrl.GetRemainingAttempts(ctx, "198.51.100.7")
	require.NoError(t, err)
	assert.Equal(t, lrl.maxAttempts, remainingAfterIP)

	// Counter beyond the max yields zero remaining without being blocked.
	require.NoError(t, client.Set(ctx, "ratelimit:login:count:over", "10", time.Minute).Err())
	remaining, err := lrl.GetRemainingAttempts(ctx, "over")
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)

	// Non-integer counter is a get error.
	require.NoError(t, client.Set(ctx, "ratelimit:login:count:garbage", "x", time.Minute).Err())
	_, err = lrl.GetRemainingAttempts(ctx, "garbage")
	assert.Error(t, err)

	mr.SetError("ERR rc2 injected")
	defer mr.SetError("")
	assert.Error(t, lrl.RecordFailedAttempt(ctx, "k"))
	_, err = lrl.IsBlocked(ctx, "k")
	assert.Error(t, err)
	_, err = lrl.GetRemainingAttempts(ctx, "k")
	assert.Error(t, err)
	assert.Error(t, lrl.RecordSuccessfulLogin(ctx, "k"))
}
