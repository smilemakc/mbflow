package sdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_NewClient_StandaloneMode tests creating a client in standalone mode
func TestClient_NewClient_StandaloneMode(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	assert.Equal(t, ModeEmbedded, client.config.Mode)
	assert.Empty(t, client.config.DatabaseURL)
	assert.Empty(t, client.config.RedisURL)
	assert.NotNil(t, client.executorManager)
}

// TestClient_NewClient_EmbeddedMode tests creating a client in embedded mode
func TestClient_NewClient_EmbeddedMode(t *testing.T) {
	// Skip if no test database available
	if testing.Short() {
		t.Skip("Skipping embedded mode test in short mode")
	}

	databaseURL := "postgres://test:test@localhost:5432/mbflow_test?sslmode=disable"
	redisURL := "redis://localhost:6379"

	client, err := NewClient(WithEmbeddedMode(databaseURL, redisURL))

	// Note: This may fail if database is not available - that's expected
	if err != nil {
		t.Skipf("Skipping embedded mode test - database not available: %v", err)
	}

	require.NotNil(t, client)
	defer client.Close()

	assert.Equal(t, ModeEmbedded, client.config.Mode)
	assert.Equal(t, databaseURL, client.config.DatabaseURL)
	assert.Equal(t, redisURL, client.config.RedisURL)
}

// TestClient_NewClient_EmbeddedMode_NoDatabaseURL tests that embedded mode requires database URL
func TestClient_NewClient_EmbeddedMode_NoDatabaseURL(t *testing.T) {
	_, err := NewClient(WithEmbeddedMode("", ""))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database URL is required")
}

// TestClient_NewClient_RemoteMode tests creating a client in remote mode
func TestClient_NewClient_RemoteMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-api-key"),
	)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	assert.Equal(t, ModeRemote, client.config.Mode)
	assert.Equal(t, server.URL, client.config.BaseURL)
	assert.Equal(t, "test-api-key", client.config.APIKey)
}

// TestClient_NewClient_RemoteMode_NoBaseURL tests that remote mode requires base URL
func TestClient_NewClient_RemoteMode_NoBaseURL(t *testing.T) {
	_, err := NewClient(WithHTTPEndpoint(""))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// TestClient_NewClient_DefaultMode tests that default mode is embedded
func TestClient_NewClient_DefaultMode(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	assert.Equal(t, ModeEmbedded, client.config.Mode)
	assert.NotNil(t, client.executorManager)
}

// TestClient_WithAPIKey tests setting API key
func TestClient_WithAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithAPIKey("test-key-123"),
	)
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, "test-key-123", client.config.APIKey)
}

// TestClient_WithAPIKey_Empty tests that empty API key is rejected
func TestClient_WithAPIKey_Empty(t *testing.T) {
	_, err := NewClient(WithAPIKey(""))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// TestClient_WithTimeout tests setting custom timeout
func TestClient_WithTimeout(t *testing.T) {
	timeout := 60 * time.Second

	client, err := NewClient(
		WithStandaloneMode(),
		WithTimeout(timeout),
	)
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, timeout, client.config.Timeout)
	assert.Equal(t, timeout, client.httpClient.Timeout)
}

// TestClient_WithTimeout_Zero tests that zero timeout is rejected
func TestClient_WithTimeout_Zero(t *testing.T) {
	_, err := NewClient(WithTimeout(0))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

// TestClient_WithTimeout_Negative tests that negative timeout is rejected
func TestClient_WithTimeout_Negative(t *testing.T) {
	_, err := NewClient(WithTimeout(-1 * time.Second))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

// TestClient_WithHTTPClient tests setting custom HTTP client
func TestClient_WithHTTPClient(t *testing.T) {
	customClient := &http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns: 100,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(
		WithHTTPEndpoint(server.URL),
		WithHTTPClient(customClient),
	)
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, customClient, client.httpClient)
}

// TestClient_WithHTTPClient_Nil tests that nil HTTP client is rejected
func TestClient_WithHTTPClient_Nil(t *testing.T) {
	_, err := NewClient(WithHTTPClient(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestClient_WithExecutorManager tests setting custom executor manager
func TestClient_WithExecutorManager(t *testing.T) {
	customManager := executor.NewManager()

	client, err := NewClient(
		WithStandaloneMode(),
		WithExecutorManager(customManager),
	)
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, customManager, client.executorManager)
}

// TestClient_WithExecutorManager_Nil tests that nil executor manager is rejected
func TestClient_WithExecutorManager_Nil(t *testing.T) {
	_, err := NewClient(WithExecutorManager(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestClient_Close tests client closing
func TestClient_Close(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)

	// Second close should not error
	err = client.Close()
	assert.NoError(t, err)
}

// TestClient_Close_ChecksClosed tests that closed client returns error
func TestClient_Close_ChecksClosed(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)

	client.Close()

	err = client.checkClosed()
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestClient_RegisterExecutor tests registering custom executor
func TestClient_RegisterExecutor(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)
	defer client.Close()

	// Create mock executor
	mockExec := &mockExecutor{nodeType: "custom"}

	err = client.RegisterExecutor("custom", mockExec)
	assert.NoError(t, err)

	// Verify executor was registered
	exec, err := client.executorManager.Get("custom")
	assert.NoError(t, err)
	assert.Equal(t, mockExec, exec)
}

// TestClient_RegisterExecutor_RemoteMode tests that registering executor in remote mode fails
func TestClient_RegisterExecutor_RemoteMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(WithHTTPEndpoint(server.URL))
	require.NoError(t, err)
	defer client.Close()

	mockExec := &mockExecutor{nodeType: "custom"}

	err = client.RegisterExecutor("custom", mockExec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only available in embedded mode")
}

// TestClient_RegisterExecutor_ClosedClient tests that registering executor on closed client fails
func TestClient_RegisterExecutor_ClosedClient(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)
	client.Close()

	mockExec := &mockExecutor{nodeType: "custom"}

	err = client.RegisterExecutor("custom", mockExec)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestClient_Health tests health check
func TestClient_Health(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	health, err := client.Health(ctx)
	require.NoError(t, err)
	require.NotNil(t, health)

	assert.Equal(t, "ok", health.Status)
	assert.Equal(t, ModeEmbedded, health.Mode)
}

// TestClient_Health_ClosedClient tests health check on closed client
func TestClient_Health_ClosedClient(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()
	_, err = client.Health(ctx)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestClient_Workflows tests Workflows API accessor
func TestClient_Workflows(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)
	defer client.Close()

	workflows := client.Workflows()
	assert.NotNil(t, workflows)
}

// TestClient_Executions tests Executions API accessor
func TestClient_Executions(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)
	defer client.Close()

	executions := client.Executions()
	assert.NotNil(t, executions)
}

// TestClient_Triggers tests Triggers API accessor
func TestClient_Triggers(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)
	defer client.Close()

	triggers := client.Triggers()
	assert.NotNil(t, triggers)
}

// TestClient_MultipleOptions tests applying multiple options
func TestClient_MultipleOptions(t *testing.T) {
	timeout := 45 * time.Second

	client, err := NewClient(
		WithStandaloneMode(),
		WithTimeout(timeout),
	)
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, ModeEmbedded, client.config.Mode)
	assert.Equal(t, timeout, client.config.Timeout)
	assert.Empty(t, client.config.DatabaseURL)
}

// TestClient_ConcurrentAccess tests concurrent access to client
func TestClient_ConcurrentAccess(t *testing.T) {
	client, err := NewClient(WithStandaloneMode())
	require.NoError(t, err)
	defer client.Close()

	done := make(chan bool, 10)

	// Spawn 10 goroutines accessing client APIs
	for i := 0; i < 10; i++ {
		go func() {
			_ = client.Workflows()
			_ = client.Executions()
			_ = client.Triggers()
			_, _ = client.Health(context.Background())
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// mockExecutor is a simple executor for testing
type mockExecutor struct {
	nodeType string
}

func (m *mockExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	return map[string]any{"output": "test"}, nil
}

func (m *mockExecutor) Validate(config map[string]any) error {
	return nil
}

func (m *mockExecutor) Type() string {
	return m.nodeType
}

// TestClient_WithAutoMigrate tests auto-migrate option
func TestClient_WithAutoMigrate(t *testing.T) {
	client, err := NewClient(
		WithStandaloneMode(),
		WithAutoMigrate(),
	)
	require.NoError(t, err)
	defer client.Close()

	assert.True(t, client.config.AutoMigrate)
}

// TestClient_WithWebhookBaseURL tests webhook base URL option
func TestClient_WithWebhookBaseURL(t *testing.T) {
	client, err := NewClient(
		WithStandaloneMode(),
		WithWebhookBaseURL("https://api.example.com"),
	)
	require.NoError(t, err)
	defer client.Close()

	assert.Equal(t, "https://api.example.com", client.config.WebhookBaseURL)
}
