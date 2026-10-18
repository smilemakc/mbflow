package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/auth"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/testutil"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return newTestServerWithRedis(t, "redis://127.0.0.1:1")
}

// newTestServerWithRedis builds a production-mode server; redisURL may point
// to an unreachable address (Redis is optional) or to a miniredis instance.
func newTestServerWithRedis(t *testing.T, redisURL string) *Server {
	t.Helper()

	cfg := &config.Config{
		Env: config.EnvProduction,
		Server: config.ServerConfig{
			Host:               "127.0.0.1",
			Port:               8585,
			ReadTimeout:        5 * time.Second,
			WriteTimeout:       5 * time.Second,
			ShutdownTimeout:    5 * time.Second,
			CORS:               true,
			CORSAllowedOrigins: []string{"https://app.example.com"},
			MaxBodySize:        1 << 20,
			MaxMultipartMemory: 1 << 20,
		},
		Database: config.DatabaseConfig{
			URL:             testutil.SetupTestDSN(t),
			MaxConnections:  5,
			MinConnections:  1,
			MaxIdleTime:     time.Minute,
			MaxConnLifetime: time.Minute,
		},
		Redis:    config.RedisConfig{URL: redisURL, PoolSize: 2},
		Logging:  config.LoggingConfig{Level: "error", Format: "json"},
		Observer: config.ObserverConfig{EnableWebSocket: true, WebSocketBufferSize: 16, BufferSize: 16},
		Auth: config.AuthConfig{
			Mode:                   "builtin",
			JWTSecret:              "integration-test-secret-at-least-32-characters",
			JWTExpirationHours:     1,
			RefreshExpiryDays:      1,
			MinPasswordLength:      8,
			MaxLoginAttempts:       5,
			LockoutDuration:        time.Minute,
			BootstrapAdminEmail:    "admin@example.com",
			BootstrapAdminPassword: "Admin-Passw0rd-123",
		},
		ServiceKeys: config.ServiceKeysConfig{MaxKeysPerUser: 5, DefaultExpiryDays: 30},
		ServiceAPI:  config.SystemAPIConfig{MaxKeys: 5, BcryptCost: 4, DefaultExpiryDays: 30, SystemUserID: "00000000-0000-0000-0000-000000000000"},
	}
	require.NoError(t, cfg.Validate())

	srv, err := New(WithConfig(cfg))
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	return srv
}

func doRequest(srv *Server, method, path, token string) *httptest.ResponseRecorder {
	return doJSONRequest(srv, method, path, token, "")
}

func doJSONRequest(srv *Server, method, path, token, body string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	return w
}

func TestServer_ProductionHardening(t *testing.T) {
	srv := newTestServer(t)

	t.Run("liveness does not depend on redis", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, doRequest(srv, "GET", "/health", "").Code)
		assert.Equal(t, http.StatusOK, doRequest(srv, "GET", "/livez", "").Code)
	})

	t.Run("protected API requires authentication", func(t *testing.T) {
		for _, path := range []string{
			"/api/v1/workflows",
			"/api/v1/executions",
			"/api/v1/triggers",
			"/api/v1/files",
		} {
			assert.Equal(t, http.StatusUnauthorized, doRequest(srv, "GET", path, "").Code, path)
		}
		assert.Equal(t, http.StatusUnauthorized, doRequest(srv, "POST", "/api/v1/executions/ephemeral", "").Code)
		assert.Equal(t, http.StatusUnauthorized, doRequest(srv, "GET", "/ws/executions?execution_id=x", "").Code)
	})

	t.Run("swagger is disabled in production", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, doRequest(srv, "GET", "/swagger/index.html", "").Code)
	})

	t.Run("bootstrap admin can log in and use the API", func(t *testing.T) {
		result, err := srv.AuthService().Login(context.Background(), &auth.LoginRequest{
			Email: "admin@example.com", Password: "Admin-Passw0rd-123",
		}, "127.0.0.1", "test")
		require.NoError(t, err)
		require.True(t, result.User.IsAdmin)

		w := doRequest(srv, "GET", "/api/v1/workflows", result.AccessToken)
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})

	t.Run("readiness reports redis outage and shutdown", func(t *testing.T) {
		// Redis is unreachable in this test, so if a cache was configured the
		// server must report not ready; otherwise the DB check alone passes.
		w := doRequest(srv, "GET", "/ready", "")
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		if srv.data.RedisCache == nil {
			assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		} else {
			assert.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
		}

		srv.shuttingDown.Store(true)
		defer srv.shuttingDown.Store(false)
		assert.Equal(t, http.StatusServiceUnavailable, doRequest(srv, "GET", "/ready", "").Code)
	})
}

// loginNewUser creates a regular user and returns an access token.
func loginNewUser(t *testing.T, srv *Server, email string) string {
	t.Helper()
	ctx := context.Background()
	_, err := srv.AuthService().CreateUser(ctx, &auth.RegisterRequest{Email: email, Password: "User-Passw0rd-123"}, false)
	require.NoError(t, err)
	result, err := srv.AuthService().Login(ctx, &auth.LoginRequest{Email: email, Password: "User-Passw0rd-123"}, "127.0.0.1", "test")
	require.NoError(t, err)
	return result.AccessToken
}

func TestServer_PerUserIsolation(t *testing.T) {
	srv := newTestServer(t)
	tokenA := loginNewUser(t, srv, "alice@example.com")
	tokenB := loginNewUser(t, srv, "bob@example.com")

	w := doJSONRequest(srv, "POST", "/api/v1/workflows", tokenA, `{"name":"alice workflow"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	wfID := created["id"].(string)

	// Bob can neither see nor delete Alice's workflow.
	assert.Equal(t, http.StatusNotFound, doRequest(srv, "GET", "/api/v1/workflows/"+wfID, tokenB).Code)
	assert.Equal(t, http.StatusNotFound, doRequest(srv, "DELETE", "/api/v1/workflows/"+wfID, tokenB).Code)
	assert.Equal(t, http.StatusNotFound, doRequest(srv, "POST", "/api/v1/executions/run/"+wfID, tokenB).Code)
	assert.Equal(t, http.StatusNotFound, doRequest(srv, "GET", "/ws/executions?execution_id="+uuid.NewString(), tokenB).Code)

	w = doRequest(srv, "GET", "/api/v1/workflows", tokenB)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), wfID)

	// Alice still has it.
	assert.Equal(t, http.StatusOK, doRequest(srv, "GET", "/api/v1/workflows/"+wfID, tokenA).Code)
	w = doRequest(srv, "GET", "/api/v1/workflows", tokenA)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), wfID)
}

// countExecutions returns how many executions of workflowID were started by a
// trigger (trigger_id set).
func countTriggeredExecutions(t *testing.T, srv *Server, workflowID string) int {
	t.Helper()
	n, err := srv.DB().NewSelect().Table("mbflow_executions").
		Where("workflow_id = ?", workflowID).Where("trigger_id IS NOT NULL").Count(context.Background())
	require.NoError(t, err)
	return n
}

func TestServer_TriggersApplyWithoutRestart(t *testing.T) {
	mr := miniredis.RunT(t)
	srv := newTestServerWithRedis(t, "redis://"+mr.Addr())
	require.NotNil(t, srv.TriggerManager(), "trigger manager needs Redis")
	token := loginNewUser(t, srv, "carol@example.com")

	// Nodes sent on create must be stored (they used to be dropped silently).
	w := doJSONRequest(srv, "POST", "/api/v1/workflows", token, `{"name":"triggered",
		"nodes":[{"id":"t","name":"T","type":"transform","config":{"type":"expression","expression":"1 + 1"}}]}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var wf map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wf))
	wfID := wf["id"].(string)
	require.Len(t, wf["nodes"], 1)

	createTrigger := func(body string) string {
		t.Helper()
		w := doJSONRequest(srv, "POST", "/api/v1/triggers", token, body)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		var tr map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tr))
		return tr["id"].(string)
	}

	// Invalid schedules are rejected instead of silently never firing.
	w = doJSONRequest(srv, "POST", "/api/v1/triggers", token,
		`{"workflow_id":"`+wfID+`","name":"bad","type":"cron","enabled":true,"config":{"expression":"0 * * * *"}}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Webhook trigger is callable right after creation (no restart).
	hookID := createTrigger(`{"workflow_id":"` + wfID + `","name":"hook","type":"webhook","enabled":true,"config":{}}`)
	w = doJSONRequest(srv, "POST", "/api/v1/webhooks/"+hookID, "", `{"x":1}`)
	assert.Less(t, w.Code, 300, "webhook call: %d %s", w.Code, w.Body.String())

	// Interval trigger starts firing without a restart and records trigger_id.
	intervalID := createTrigger(`{"workflow_id":"` + wfID + `","name":"tick","type":"interval","enabled":true,"config":{"interval":"1s"}}`)
	require.Eventually(t, func() bool { return countTriggeredExecutions(t, srv, wfID) >= 2 }, 10*time.Second, 200*time.Millisecond)

	// After deletion it stops firing.
	require.Equal(t, http.StatusOK, doRequest(srv, "DELETE", "/api/v1/triggers/"+intervalID, token).Code)
	time.Sleep(1200 * time.Millisecond) // let an in-flight tick finish
	before := countTriggeredExecutions(t, srv, wfID)
	time.Sleep(2500 * time.Millisecond)
	assert.Equal(t, before, countTriggeredExecutions(t, srv, wfID), "deleted trigger must not keep firing")
}

func TestServer_PayloadTooLarge(t *testing.T) {
	srv := newTestServer(t)
	token := loginNewUser(t, srv, "dave@example.com")
	big := `{"name":"` + strings.Repeat("a", 2<<20) + `"}` // > MaxBodySize (1 MiB in tests)
	assert.Equal(t, http.StatusRequestEntityTooLarge, doJSONRequest(srv, "POST", "/api/v1/workflows", token, big).Code)
}
