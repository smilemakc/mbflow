package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/auth"
	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

// rcEngineConfig returns a development-mode config for an isolated database.
func rcEngineConfig(t *testing.T, dsn, redisURL string) *config.Config {
	t.Helper()
	cfg := &config.Config{
		Env: config.EnvDevelopment,
		Server: config.ServerConfig{
			Host:               "127.0.0.1",
			Port:               freeTCPPort(t),
			ReadTimeout:        5 * time.Second,
			WriteTimeout:       5 * time.Second,
			ShutdownTimeout:    5 * time.Second,
			MaxBodySize:        1 << 20,
			MaxMultipartMemory: 1 << 20,
		},
		Database: config.DatabaseConfig{URL: dsn, MaxConnections: 5, MinConnections: 1, MaxIdleTime: time.Minute, MaxConnLifetime: time.Minute},
		Redis:    config.RedisConfig{URL: redisURL, PoolSize: 2},
		Logging:  config.LoggingConfig{Level: "error", Format: "json"},
		Observer: config.ObserverConfig{EnableWebSocket: true, WebSocketBufferSize: 16, BufferSize: 16},
		Auth: config.AuthConfig{
			Mode:                   "builtin",
			JWTSecret:              "integration-test-secret-at-least-32-characters",
			JWTExpirationHours:     1,
			RefreshExpiryDays:      1,
			MinPasswordLength:      8,
			MaxLoginAttempts:       3,
			LockoutDuration:        time.Minute,
			BootstrapAdminEmail:    "admin@example.com",
			BootstrapAdminPassword: "Admin-Passw0rd-123",
		},
		ServiceKeys: config.ServiceKeysConfig{MaxKeysPerUser: 5, DefaultExpiryDays: 30},
		ServiceAPI:  config.SystemAPIConfig{MaxKeys: 5, BcryptCost: 4, DefaultExpiryDays: 30, SystemUserID: "00000000-0000-0000-0000-000000000000"},
		FileStorage: config.FileStorageConfig{StoragePath: t.TempDir(), MaxFileSize: 1 << 20},
	}
	require.NoError(t, cfg.Validate())
	return cfg
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

func newRCEngineServer(t *testing.T, cfg *config.Config) *Server {
	t.Helper()
	srv, err := New(WithConfig(cfg))
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})
	return srv
}

// seedStaleExecution inserts a running inline execution whose lease expired
// an hour ago, as left behind by a crashed process.
func seedStaleExecution(t *testing.T, dsn string) uuid.UUID {
	t.Helper()
	db, err := storage.NewDB(&storage.Config{DSN: dsn, MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	defer func() { _ = storage.Close(db) }()

	ctx := context.Background()
	exec := &storagemodels.ExecutionModel{
		ID: uuid.New(), Status: "running", WorkflowSource: "inline",
		WorkflowSnapshot: storagemodels.JSONBMap{"name": "crashed"},
	}
	require.NoError(t, storage.NewExecutionRepository(db).Create(ctx, exec))
	_, err = db.NewUpdate().Model((*storagemodels.ExecutionModel)(nil)).
		Set("updated_at = current_timestamp - interval '1 hour'").Where("id = ?", exec.ID).Exec(ctx)
	require.NoError(t, err)
	return exec.ID
}

func TestServer_RCEngine_FullFeatureSet(t *testing.T) {
	mr := miniredis.RunT(t)
	callbacks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer callbacks.Close()

	dsn := testutil.SetupTestDSN(t)
	staleID := seedStaleExecution(t, dsn)

	cfg := rcEngineConfig(t, dsn, "redis://"+mr.Addr())
	cfg.GRPCServiceAPI = config.GRPCServiceAPIConfig{Enabled: true, Address: "127.0.0.1:0"}
	cfg.Observer.EnableDatabase = true
	cfg.Observer.EnableLogger = true
	cfg.Observer.EnableHTTP = true
	cfg.Observer.HTTPCallbackURL = callbacks.URL
	cfg.Observer.HTTPMethod = http.MethodPost
	cfg.Observer.HTTPTimeout = time.Second
	cfg.Execution = config.ExecutionConfig{MaxConcurrentExecutions: 7, HeartbeatInterval: 50 * time.Millisecond, StaleAfter: time.Second}

	srv := newRCEngineServer(t, cfg)

	t.Run("startup sweep fails executions of a crashed process", func(t *testing.T) {
		row, err := srv.ExecutionRepository().FindByID(context.Background(), staleID)
		require.NoError(t, err)
		assert.Equal(t, "failed", row.Status)
		assert.Equal(t, engine.InterruptedExecutionError, row.Error)
	})

	t.Run("execution limit is wired from config", func(t *testing.T) {
		assert.Equal(t, 7, srv.ExecutionManager().MaxConcurrentExecutions())
	})

	t.Run("components are exposed", func(t *testing.T) {
		assert.NotNil(t, srv.ObserverManager())
		assert.NotNil(t, srv.FileStorageManager())
		assert.NotNil(t, srv.WorkflowRepository())
		assert.NotNil(t, srv.ServiceKeyService())
		assert.NotNil(t, srv.TriggerManager())
		assert.NotNil(t, srv.serviceAPI.GRPCServerInstance)
		assert.NotNil(t, srv.auth.EncryptionService)
	})

	t.Run("development endpoints and dependency checks", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, doRequest(srv, http.MethodGet, "/swagger/index.html", "").Code)

		w := doRequest(srv, http.MethodGet, "/ready", "")
		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), `"redis":"ok"`)

		w = doRequest(srv, http.MethodGet, "/metrics", "")
		require.Equal(t, http.StatusOK, w.Code)
		var metrics map[string]map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &metrics))
		assert.Contains(t, metrics["metrics"], "redis")
		assert.Contains(t, metrics["metrics"], "database")
	})

	t.Run("encryption-backed routes are registered", func(t *testing.T) {
		for _, path := range []string{"/api/v1/credentials", "/api/v1/rental-keys", "/api/v1/admin/rental-keys"} {
			assert.Equal(t, http.StatusUnauthorized, doRequest(srv, http.MethodGet, path, "").Code, path)
		}
	})

	t.Run("readiness reports a redis outage", func(t *testing.T) {
		mr.SetError("down")
		defer mr.SetError("")
		w := doRequest(srv, http.MethodGet, "/ready", "")
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), `"redis":"unavailable"`)
	})
}

func loginAttempt(srv *Server, password string) int {
	body := `{"email":"admin@example.com","password":"` + password + `"}`
	return doJSONRequest(srv, http.MethodPost, "/api/v1/auth/login", "", body).Code
}

func TestServer_RCEngine_LoginLockoutSharedViaRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	replicaA := newRCEngineServer(t, rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://"+mr.Addr()))
	replicaB := newRCEngineServer(t, rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://"+mr.Addr()))

	for i := 0; i < 3; i++ {
		assert.Equal(t, http.StatusUnauthorized, loginAttempt(replicaA, "wrong-password"), "attempt %d", i+1)
	}
	assert.Equal(t, http.StatusTooManyRequests, loginAttempt(replicaB, "Admin-Passw0rd-123"),
		"a lockout on one replica applies to the others")

	var keys []string
	for _, k := range mr.Keys() {
		if strings.HasPrefix(k, "ratelimit:login:") {
			keys = append(keys, k)
		}
	}
	assert.NotEmpty(t, keys, "lockout state lives in Redis")
}

func TestServer_RCEngine_LoginLockoutWithoutRedis(t *testing.T) {
	srv := newRCEngineServer(t, rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://127.0.0.1:1"))
	require.Nil(t, srv.data.RedisCache)

	for i := 0; i < 3; i++ {
		assert.Equal(t, http.StatusUnauthorized, loginAttempt(srv, "wrong-password"))
	}
	assert.Equal(t, http.StatusTooManyRequests, loginAttempt(srv, "Admin-Passw0rd-123"), "per-process lockout still applies")
}

func TestServer_RCEngine_InvalidRecoveryConfig(t *testing.T) {
	cfg := rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://127.0.0.1:1")
	cfg.Execution = config.ExecutionConfig{HeartbeatInterval: time.Second, StaleAfter: time.Millisecond}

	_, err := New(WithConfig(cfg))
	require.ErrorContains(t, err, "failed to start execution recovery")
}

func TestServer_RCEngine_RunStopsOnSignal(t *testing.T) {
	mr := miniredis.RunT(t)
	cfg := rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://"+mr.Addr())
	cfg.GRPCServiceAPI = config.GRPCServiceAPIConfig{Enabled: true, Address: "127.0.0.1:0"}
	srv := newRCEngineServer(t, cfg)

	// Keep SIGTERM from terminating the test binary even before Run has
	// registered its own handler.
	guard := make(chan os.Signal, 16)
	signal.Notify(guard, syscall.SIGTERM)
	defer signal.Stop(guard)

	done := make(chan error, 1)
	go func() { done <- srv.Run() }()

	addr := net.JoinHostPort(cfg.Server.Host, itoa(cfg.Server.Port))
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 5*time.Second, 10*time.Millisecond, "HTTP server starts listening")

	deadline := time.After(10 * time.Second)
	for {
		require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGTERM))
		select {
		case err := <-done:
			require.NoError(t, err)
			assert.True(t, srv.shuttingDown.Load())
			return
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatal("Run did not return after SIGTERM")
		}
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestServer_RCEngine_RunReturnsListenError(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer busy.Close()

	cfg := rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://127.0.0.1:1")
	cfg.Server.Port = busy.Addr().(*net.TCPAddr).Port
	srv := newRCEngineServer(t, cfg)

	done := make(chan error, 1)
	go func() { done <- srv.Run() }()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "server error")
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not fail on a busy port")
	}
}

func TestServer_RCEngine_GRPCListenerReleasedWithoutRun(t *testing.T) {
	cfg := rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://127.0.0.1:1")
	cfg.GRPCServiceAPI = config.GRPCServiceAPIConfig{Enabled: true, Address: "127.0.0.1:0"}
	srv := newRCEngineServer(t, cfg)
	addr := srv.serviceAPI.GRPCListener.Addr().String()

	require.NoError(t, srv.Shutdown(context.Background()))
	l, err := net.Listen("tcp", addr)
	require.NoError(t, err, "the gRPC port is free again after Shutdown")
	require.NoError(t, l.Close())
}

func TestServer_RCEngine_GRPCListenError(t *testing.T) {
	cfg := rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://127.0.0.1:1")
	cfg.GRPCServiceAPI = config.GRPCServiceAPIConfig{Enabled: true, Address: "127.0.0.1:not-a-port"}
	_, err := New(WithConfig(cfg))
	require.ErrorContains(t, err, "failed to setup gRPC server")
}

func TestServer_RCEngine_ShutdownDeadlineWithRunningExecution(t *testing.T) {
	cfg := rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://127.0.0.1:1")
	srv := newRCEngineServer(t, cfg)

	started := make(chan struct{})
	require.NoError(t, srv.RegisterExecutor("rc_block", &blockingNode{started: started}))

	_, err := srv.ExecutionManager().ExecuteEphemeral(context.Background(), &engine.EphemeralExecutionOptions{
		Mode: "async",
		Workflow: &models.Workflow{ID: "wf", Name: "block", Nodes: []*models.Node{
			{ID: "n", Name: "N", Type: "rc_block"},
		}},
	})
	require.NoError(t, err)
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = srv.Shutdown(ctx)
	require.Error(t, err, "the execution outlives the shutdown deadline")
	assert.Equal(t, err, srv.Shutdown(context.Background()), "Shutdown is idempotent")
}

// blockingNode runs until its context is cancelled.
type blockingNode struct{ started chan struct{} }

func (b *blockingNode) Execute(ctx context.Context, _ map[string]any, _ any) (any, error) {
	close(b.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func (b *blockingNode) Validate(map[string]any) error { return nil }

func TestServer_RCEngine_NewErrors(t *testing.T) {
	t.Run("option error", func(t *testing.T) {
		_, err := New(func(*Server) error { return assert.AnError })
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("database unreachable", func(t *testing.T) {
		cfg := rcEngineConfig(t, "postgres://u:p@127.0.0.1:1/none?sslmode=disable&connect_timeout=1", "redis://127.0.0.1:1")
		_, err := New(WithConfig(cfg))
		require.ErrorContains(t, err, "failed to initialize database")
	})

	t.Run("invalid trusted proxies", func(t *testing.T) {
		cfg := rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://127.0.0.1:1")
		cfg.Server.TrustedProxies = []string{"not-an-ip"}
		srv, err := New(WithConfig(cfg))
		require.ErrorContains(t, err, "failed to setup routes")
		assert.Nil(t, srv, "partially initialized resources are released, not returned")
	})

	t.Run("invalid environment configuration", func(t *testing.T) {
		t.Setenv("MBFLOW_ENV", "staging")
		_, err := New()
		require.ErrorContains(t, err, "failed to load configuration")
	})
}

func TestServer_RCEngine_ConfigFromEnvironment(t *testing.T) {
	dsn := testutil.SetupTestDSN(t)
	t.Setenv("MBFLOW_ENV", config.EnvDevelopment)
	t.Setenv("MBFLOW_DATABASE_URL", dsn)
	t.Setenv("MBFLOW_DB_MAX_CONNECTIONS", "5")
	t.Setenv("MBFLOW_DB_MIN_CONNECTIONS", "1")
	t.Setenv("MBFLOW_JWT_SECRET", "integration-test-secret-at-least-32-characters")
	t.Setenv("MBFLOW_REDIS_URL", "redis://127.0.0.1:1")
	t.Setenv("MBFLOW_LOG_LEVEL", "debug")
	t.Setenv("MBFLOW_LOG_FORMAT", "text")
	t.Setenv("MBFLOW_PORT", itoa(freeTCPPort(t)))
	t.Setenv("MBFLOW_FILE_STORAGE_PATH", t.TempDir())
	t.Setenv("MBFLOW_TRUSTED_PROXIES", "10.0.0.0/8")
	t.Setenv("MBFLOW_EXECUTION_HEARTBEAT_INTERVAL", "1m")
	t.Setenv("MBFLOW_MAX_CONCURRENT_EXECUTIONS", "4")

	srv, err := New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	assert.Equal(t, 4, srv.ExecutionManager().MaxConcurrentExecutions())
	assert.NotNil(t, srv.Logger())

	// Debug level in development allows any CORS origin and answers preflights.
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/workflows", nil)
	req.Header.Set("Origin", "https://anything.example")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestServer_RCEngine_CORSAllowList(t *testing.T) {
	cfg := rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://127.0.0.1:1")
	cfg.Server.CORS = true
	cfg.Server.CORSAllowedOrigins = []string{"https://app.example.com"}
	srv := newRCEngineServer(t, cfg)

	for origin, want := range map[string]string{"https://app.example.com": "https://app.example.com", "https://evil.example": ""} {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, want, w.Header().Get("Access-Control-Allow-Origin"), origin)
	}
}

func TestServer_RCEngine_WebSocketAccess(t *testing.T) {
	srv := newRCEngineServer(t, rcEngineConfig(t, testutil.SetupTestDSN(t), "redis://127.0.0.1:1"))
	userToken := loginNewUser(t, srv, "ws-user@example.com")

	assert.Equal(t, http.StatusForbidden, doRequest(srv, http.MethodGet, "/ws/executions", userToken).Code,
		"only administrators may subscribe to all executions")

	admin, err := srv.AuthService().Login(context.Background(), &auth.LoginRequest{
		Email: "admin@example.com", Password: "Admin-Passw0rd-123",
	}, "127.0.0.1", "test")
	require.NoError(t, err)
	w := doRequest(srv, http.MethodGet, "/ws/executions", admin.AccessToken)
	assert.NotEqual(t, http.StatusForbidden, w.Code, "admins pass the check and reach the upgrader")
	assert.Less(t, w.Code, 500)
}
