package rest

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/servicekey"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
)

func TestRC2_AuthMiddleware_ServiceKeys(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	authEnv := rc2NewAuthEnv(t, nil)
	ctx := context.Background()

	userID := uuid.MustParse(env.user(t, false))
	sks := servicekey.NewService(storage.NewServiceKeyRepository(env.db), servicekey.Config{BcryptCost: 4})
	mw := NewAuthMiddleware(authEnv.pm, authEnv.svc, sks)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/p", mw.RequireAuth(), func(c *gin.Context) {
		uid, _ := GetUserID(c)
		keyID, _ := GetServiceKeyID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": uid, "key_id": keyID, "service_key": IsServiceKeyAuth(c), "admin": IsAdmin(c)})
	})
	call := func(header, value string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set(header, value)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	created, err := sks.CreateKey(ctx, userID, "ci", "", userID, nil)
	require.NoError(t, err)

	for _, h := range []struct{ name, value string }{
		{"X-Service-Key", created.PlainKey},
		{"Authorization", "Bearer " + created.PlainKey},
	} {
		w := call(h.name, h.value)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		body := rc2DecodeMap(t, w)
		assert.Equal(t, userID.String(), body["user_id"])
		assert.Equal(t, created.Key.ID, body["key_id"])
		assert.Equal(t, true, body["service_key"])
		assert.Equal(t, false, body["admin"], "service keys never grant admin")
	}

	w := call("X-Service-Key", "sk_"+strings.Repeat("x", 40))
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "invalid service key", rc2DecodeMap(t, w)["message"])

	// Expired key.
	expiring, err := sks.CreateKey(ctx, userID, "old", "", userID, nil)
	require.NoError(t, err)
	_, err = env.db.NewUpdate().Table("mbflow_service_keys").
		Set("expires_at = ?", time.Now().Add(-time.Hour)).
		Where("id = ?", uuid.MustParse(expiring.Key.ID)).Exec(ctx)
	require.NoError(t, err)
	w = call("X-Service-Key", expiring.PlainKey)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "service key has expired", rc2DecodeMap(t, w)["message"])

	// Revoked key.
	require.NoError(t, sks.RevokeKey(ctx, uuid.MustParse(created.Key.ID)))
	w = call("X-Service-Key", created.PlainKey)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "service key has been revoked", rc2DecodeMap(t, w)["message"])
}

// rc2FailHook makes selected Redis commands fail.
type rc2FailHook struct{ cmds map[string]bool }

func (h rc2FailHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) { return next(ctx, network, addr) }
}

func (h rc2FailHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.cmds[cmd.Name()] {
			err := errors.New("rc2 injected " + cmd.Name() + " failure")
			cmd.SetErr(err)
			return err
		}
		return next(ctx, cmd)
	}
}

func (h rc2FailHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func rc2FailingRedis(t *testing.T, cmds ...string) redis.UniversalClient {
	t.Helper()
	_, client := rc2FailingRedisWithServer(t, cmds...)
	return client
}

func rc2FailingRedisWithServer(t *testing.T, cmds ...string) (*miniredis.Miniredis, redis.UniversalClient) {
	t.Helper()
	mr, _ := setupMiniRedis(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	set := map[string]bool{}
	for _, c := range cmds {
		set[c] = true
	}
	client.AddHook(rc2FailHook{cmds: set})
	return mr, client
}

func TestRC2_RedisRateLimiter_CommandFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// TTL failure while blocked falls back to the configured block duration.
	mr, client := rc2FailingRedisWithServer(t, "ttl")
	rl := NewRedisRateLimiter(client, "rc2:", 1, time.Minute, 42*time.Second)
	require.NoError(t, mr.Set("rc2:block:k", "1"))
	allowed, retry, err := rl.Allow(ctx, "k")
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 42, retry)

	for _, tc := range []struct {
		cmd   string
		count string
	}{
		{"incr", ""},
		{"expire", ""},
		{"set", "5"},
	} {
		mr, client := rc2FailingRedisWithServer(t, tc.cmd)
		if tc.count != "" {
			require.NoError(t, mr.Set("rc2:count:k", tc.count))
		}
		rl := NewRedisRateLimiter(client, "rc2:", 1, time.Minute, time.Minute)
		_, _, err := rl.Allow(ctx, "k")
		assert.ErrorContains(t, err, tc.cmd, tc.cmd)
	}

	// Login limiter: expire and block failures.
	lrl := NewRedisLoginRateLimiter(rc2FailingRedis(t, "expire"), 1, time.Minute, time.Minute)
	assert.ErrorContains(t, lrl.RecordFailedAttempt(ctx, "k"), "expire")
	lrl = NewRedisLoginRateLimiter(rc2FailingRedis(t, "set"), 1, time.Minute, time.Minute)
	assert.ErrorContains(t, lrl.RecordFailedAttempt(ctx, "k"), "set block")
}
