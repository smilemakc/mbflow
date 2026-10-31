package sdk

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// rcFailingManager is an executor.Manager whose Register always fails.
type rcFailingManager struct{ executor.Manager }

func (rcFailingManager) Register(string, executor.Executor) error {
	return errors.New("register refused")
}

type rcObserverManager struct{}

func (rcObserverManager) Notify(context.Context, *engine.Event) error { return nil }
func (rcObserverManager) Register(engine.Observer) error              { return nil }
func (rcObserverManager) Unregister(string) error                     { return nil }
func (rcObserverManager) Count() int                                  { return 0 }

type rcLogger struct{}

func (rcLogger) Debug(string, ...any) {}
func (rcLogger) Info(string, ...any)  {}
func (rcLogger) Warn(string, ...any)  {}
func (rcLogger) Error(string, ...any) {}

func TestRCOptions(t *testing.T) {
	cfg := &ClientConfig{}

	require.NoError(t, WithDatabase("postgres://x")(cfg))
	assert.Equal(t, "postgres://x", cfg.DatabaseURL)
	assert.Error(t, WithDatabase("")(cfg))

	require.NoError(t, WithRedis("redis://r")(cfg))
	assert.Equal(t, "redis://r", cfg.RedisURL)

	require.NoError(t, WithLogger(rcLogger{})(cfg))
	assert.NotNil(t, cfg.Logger)
	assert.Error(t, WithLogger(nil)(cfg))

	require.NoError(t, WithObserverManager(rcObserverManager{})(cfg))
	assert.NotNil(t, cfg.ObserverManager)
	assert.Error(t, WithObserverManager(nil)(cfg))

	require.NoError(t, WithEmbeddedMode("postgres://db", "redis://r2")(cfg))
	assert.Equal(t, ModeEmbedded, cfg.Mode)
	assert.Equal(t, "postgres://db", cfg.DatabaseURL)
	assert.Equal(t, "redis://r2", cfg.RedisURL)

	require.NoError(t, WithStandaloneMode()(cfg))
	assert.Empty(t, cfg.DatabaseURL)
	assert.Error(t, WithHTTPEndpoint("")(cfg))
}

func TestRCNewClient_Errors(t *testing.T) {
	_, err := NewClient(func(*ClientConfig) error { return errors.New("nope") })
	assert.ErrorContains(t, err, "failed to apply option: nope")

	_, err = NewClient(func(c *ClientConfig) error { c.Timeout = 0; return nil })
	assert.ErrorContains(t, err, "timeout must be positive")

	_, err = NewClient(func(c *ClientConfig) error { c.Mode = ClientMode(42); return nil })
	assert.ErrorContains(t, err, "unknown client mode")

	_, err = NewClient(WithExecutorManager(rcFailingManager{executor.NewManager()}))
	assert.ErrorContains(t, err, "failed to register built-in executors")

	// Remote health check failures.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	_, err = NewClient(WithHTTPEndpoint(srv.URL))
	assert.ErrorContains(t, err, "health check failed with status: 503")

	_, err = NewClient(WithHTTPEndpoint("http://bad host"))
	assert.ErrorContains(t, err, "failed to create health check request")

	_, err = NewClient(WithHTTPEndpoint(rcClosedURL(t)))
	assert.ErrorContains(t, err, "failed to connect to remote API")

	// initializeRemote is also guarded on its own.
	c := &Client{config: &ClientConfig{Mode: ModeRemote}}
	assert.ErrorContains(t, c.initializeRemote(), "base URL is required")
}

// rcClosedURL returns a URL on which nothing listens.
func rcClosedURL(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return "http://" + addr
}

func TestRCClient_EmbeddedObserverAndExecutor(t *testing.T) {
	c, err := NewClient(WithObserverManager(rcObserverManager{}))
	require.NoError(t, err)
	defer c.Close()
	assert.NotNil(t, c.observerManager)
	assert.NotNil(t, c.getStandaloneExecutor())

	h, err := c.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", h.Status)
	assert.Equal(t, ModeEmbedded, h.Mode)
	assert.Nil(t, h.Remote)

	c.executorManager = nil
	assert.ErrorContains(t, c.RegisterExecutor("x", nil), "executor manager not initialized")
}

func TestRCClient_HealthRemote(t *testing.T) {
	var status int
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer "+rcAPIKey, r.Header.Get("Authorization"))
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	status, body = http.StatusOK, `{"status":"ok","version":"2.3.4"}`
	c := rcRemoteClient(t, srv.URL)
	ctx := context.Background()

	h, err := c.Health(ctx)
	require.NoError(t, err)
	assert.Equal(t, "ok", h.Status)
	require.NotNil(t, h.Remote)
	assert.Equal(t, "healthy", h.Remote.Status)
	assert.Equal(t, "2.3.4", h.Remote.Version)

	body = "not json"
	h, err = c.Health(ctx)
	require.NoError(t, err)
	assert.Equal(t, "healthy", h.Remote.Status)
	assert.Empty(t, h.Remote.Version)

	status = http.StatusInternalServerError
	h, err = c.Health(ctx)
	require.NoError(t, err)
	assert.Equal(t, "unhealthy", h.Status)
	assert.Contains(t, h.Remote.Error, "500")

	srv.Close()
	h, err = c.Health(ctx)
	require.NoError(t, err)
	assert.Equal(t, "unhealthy", h.Status)
	assert.NotEmpty(t, h.Remote.Error)

	c.config.BaseURL = "http://bad host"
	_, err = c.Health(ctx)
	assert.ErrorContains(t, err, "failed to create health check request")
}

func TestRCMigrationAPI_NotAvailable(t *testing.T) {
	c, err := NewStandaloneClient()
	require.NoError(t, err)
	m := newMigrationAPI(c)
	ctx := context.Background()

	_, err = m.Up(ctx)
	assert.Equal(t, errMigrationsNotAvailable, err)
	_, err = m.Down(ctx)
	assert.Equal(t, errMigrationsNotAvailable, err)
	_, err = m.Status(ctx)
	assert.Equal(t, errMigrationsNotAvailable, err)
	assert.Equal(t, errMigrationsNotAvailable, m.Reset(ctx))

	require.NoError(t, c.Close())
	_, err = m.Up(ctx)
	assert.ErrorIs(t, err, models.ErrClientClosed)
	_, err = m.Down(ctx)
	assert.ErrorIs(t, err, models.ErrClientClosed)
	_, err = m.Status(ctx)
	assert.ErrorIs(t, err, models.ErrClientClosed)
	assert.ErrorIs(t, m.Reset(ctx), models.ErrClientClosed)
}

func TestRCEmbeddedMode_UnsupportedOperations(t *testing.T) {
	c, err := NewStandaloneClient(WithWebhookBaseURL("https://hooks.example"))
	require.NoError(t, err)
	defer c.Close()
	ctx := context.Background()

	wf := rcSampleWorkflow()
	wf.ID = "wf-1"
	_, err = c.Workflows().Update(ctx, wf)
	assert.Equal(t, errWorkflowPersistenceNotAvailable, err)
	assert.Equal(t, errWorkflowPersistenceNotAvailable, c.Workflows().Delete(ctx, "wf-1"))
	_, err = c.Workflows().GetTopology(ctx, "wf-1")
	assert.Equal(t, errWorkflowPersistenceNotAvailable, err)

	_, err = c.Executions().StreamLogs(ctx, "e1", nil)
	assert.Equal(t, errStandaloneModeNotSupported, err)
	_, err = c.Triggers().GetWebhookURL(ctx, "t1")
	assert.Equal(t, errTriggersNotAvailable, err)

	// DAG validation errors on update.
	bad := rcSampleWorkflow()
	bad.ID = "wf-1"
	bad.Edges = append(bad.Edges, &models.Edge{ID: "e2", From: "n2", To: "n1"})
	_, err = c.Workflows().Update(ctx, bad)
	assert.ErrorContains(t, err, "DAG validation failed")

	_, err = c.ExecuteWorkflowStandalone(ctx, nil, nil, nil)
	assert.ErrorContains(t, err, "workflow is required")

	c.standaloneExecutor = nil
	_, err = c.ExecuteWorkflowStandalone(ctx, wf, nil, nil)
	assert.ErrorContains(t, err, "standalone executor not initialized")
}

func TestRCRemoteMode_StandaloneAndValidation(t *testing.T) {
	_, srv := rcNewFakeAPI(t)
	c := rcRemoteClient(t, srv.URL)
	ctx := context.Background()

	_, err := c.ExecuteWorkflowStandalone(ctx, rcSampleWorkflow(), nil, nil)
	assert.ErrorContains(t, err, "only available in embedded mode")

	_, err = c.Workflows().GetTopology(ctx, "")
	assert.ErrorIs(t, err, models.ErrInvalidWorkflowID)

	require.NoError(t, c.Close())
	_, err = c.Workflows().GetTopology(ctx, "x")
	assert.ErrorIs(t, err, models.ErrClientClosed)
	_, err = c.Executions().StreamLogs(ctx, "x", nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
	_, err = c.Triggers().GetWebhookURL(ctx, "x")
	assert.ErrorIs(t, err, models.ErrClientClosed)
	_, err = c.Health(ctx)
	assert.ErrorIs(t, err, models.ErrClientClosed)
	_, err = c.Executions().StreamLogs(ctx, "", nil)
	assert.Error(t, err)
}

func TestRCStreamLogs_EmptyID(t *testing.T) {
	c, err := NewStandaloneClient()
	require.NoError(t, err)
	defer c.Close()
	_, err = c.Executions().StreamLogs(context.Background(), "", nil)
	assert.ErrorIs(t, err, models.ErrInvalidExecutionID)
}

func TestRCIsTerminalExecutionStatus(t *testing.T) {
	for _, s := range []models.ExecutionStatus{models.ExecutionStatusCompleted, models.ExecutionStatusFailed, models.ExecutionStatusCancelled, models.ExecutionStatusTimeout} {
		assert.True(t, isTerminalExecutionStatus(s), s)
	}
	for _, s := range []models.ExecutionStatus{models.ExecutionStatusPending, models.ExecutionStatusRunning, ""} {
		assert.False(t, isTerminalExecutionStatus(s), s)
	}
}

func TestRCTopologicalSort_Cycle(t *testing.T) {
	wf := &models.Workflow{
		Nodes: []*models.Node{{ID: "a"}, {ID: "b"}},
		Edges: []*models.Edge{{ID: "1", From: "a", To: "b"}, {ID: "2", From: "b", To: "a"}},
	}
	_, err := topologicalSort(wf)
	assert.Error(t, err)
}

func TestRCServiceClient_GRPCTransport(t *testing.T) {
	_, err := NewServiceClient(ServiceClientConfig{Endpoint: "http://x", SystemKey: "k", Transport: TransportGRPC})
	assert.ErrorContains(t, err, "gRPC address is required")

	// Without transport credentials grpc refuses to dial.
	_, err = NewServiceClient(ServiceClientConfig{Endpoint: "http://x", SystemKey: "k", Transport: TransportGRPC, GRPCAddress: "127.0.0.1:1"})
	assert.ErrorContains(t, err, "failed to create gRPC transport")

	// Dialing is lazy, so this succeeds without a server.
	c, err := NewServiceClient(ServiceClientConfig{
		Endpoint: "http://x", SystemKey: "sysk_1", Transport: TransportGRPC,
		GRPCAddress: "127.0.0.1:1", GRPCInsecure: true, OnBehalfOf: "u1",
	})
	require.NoError(t, err)
	require.NotNil(t, c.grpcTransport)
	assert.NotNil(t, c.Workflows)
	assert.NotNil(t, c.Credentials)

	md, ok := metadata.FromOutgoingContext(c.grpcTransport.contextWithAuth(context.Background()))
	require.True(t, ok)
	assert.Equal(t, []string{"sysk_1"}, md.Get("x-system-key"))
	assert.Equal(t, []string{"u1"}, md.Get("x-on-behalf-of"))

	as := c.As("u2")
	assert.Equal(t, "u2", as.config.OnBehalfOf)
	assert.Equal(t, "u2", as.grpcTransport.onBehalfOf)
	assert.Equal(t, "u1", c.grpcTransport.onBehalfOf)
	assert.Same(t, as, as.Workflows.client)

	md, _ = metadata.FromOutgoingContext((&grpcServiceTransport{systemKey: "k"}).contextWithAuth(context.Background()))
	assert.Empty(t, md.Get("x-on-behalf-of"))

	require.NoError(t, c.Close())
	assert.NoError(t, (&grpcServiceTransport{}).close())
}

func TestRCServiceClient_ConfigErrors(t *testing.T) {
	_, err := NewServiceClient(ServiceClientConfig{SystemKey: "k"})
	assert.ErrorContains(t, err, "endpoint is required")
	_, err = NewServiceClient(ServiceClientConfig{Endpoint: "http://x"})
	assert.ErrorContains(t, err, "system key is required")

	c, err := NewServiceClient(ServiceClientConfig{Endpoint: "http://x", SystemKey: "k"})
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, c.httpClient.Timeout)
	assert.NoError(t, c.Close())

	custom := &http.Client{}
	c, err = NewServiceClient(ServiceClientConfig{Endpoint: "http://x", SystemKey: "k", HTTPClient: custom, Timeout: time.Second})
	require.NoError(t, err)
	assert.Same(t, custom, c.httpClient)
}
