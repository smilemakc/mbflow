package server

import (
	"context"
	"testing"

	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/pkg/executor"
)

func TestWithConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	s := &Server{}
	opt := WithConfig(cfg)

	err := opt(s)
	if err != nil {
		t.Fatalf("WithConfig returned error: %v", err)
	}

	if s.config != cfg {
		t.Error("WithConfig did not set config")
	}
	if s.config.Server.Host != "localhost" {
		t.Errorf("Expected host localhost, got %s", s.config.Server.Host)
	}
	if s.config.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", s.config.Server.Port)
	}
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	l := logger.New(config.LoggingConfig{
		Level:  "info",
		Format: "json",
	})

	s := &Server{}
	opt := WithLogger(l)

	err := opt(s)
	if err != nil {
		t.Fatalf("WithLogger returned error: %v", err)
	}

	if s.logger != l {
		t.Error("WithLogger did not set logger")
	}
}

func TestWithExecutorManager(t *testing.T) {
	t.Parallel()

	mgr := executor.NewRegistry()

	s := &Server{}
	opt := WithExecutorManager(mgr)

	err := opt(s)
	if err != nil {
		t.Fatalf("WithExecutorManager returned error: %v", err)
	}

	if s.execution.ExecutorManager != mgr {
		t.Error("WithExecutorManager did not set executor manager")
	}
}

func TestServer_Config(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 9090,
		},
	}

	s := &Server{config: cfg}

	result := s.Config()
	if result != cfg {
		t.Error("Config() did not return the correct config")
	}
}

func TestServer_Logger(t *testing.T) {
	t.Parallel()

	l := logger.New(config.LoggingConfig{
		Level:  "debug",
		Format: "text",
	})

	s := &Server{logger: l}

	result := s.Logger()
	if result != l {
		t.Error("Logger() did not return the correct logger")
	}
}

func TestServer_ExecutorManager(t *testing.T) {
	t.Parallel()

	mgr := executor.NewRegistry()

	s := &Server{
		execution: ExecutionLayer{ExecutorManager: mgr},
	}

	result := s.ExecutorManager()
	if result != mgr {
		t.Error("ExecutorManager() did not return the correct manager")
	}
}

func TestServer_Router_Nil(t *testing.T) {
	t.Parallel()

	s := &Server{}

	result := s.Router()
	if result != nil {
		t.Error("Router() should return nil when not initialized")
	}
}

func TestServer_DB_Nil(t *testing.T) {
	t.Parallel()

	s := &Server{}

	result := s.DB()
	if result != nil {
		t.Error("DB() should return nil when not initialized")
	}
}

func TestServer_RegisterExecutor_NilManager(t *testing.T) {
	t.Parallel()

	s := &Server{}

	err := s.RegisterExecutor("test", nil)
	if err == nil {
		t.Error("RegisterExecutor should return error when executor manager is nil")
	}
	if err.Error() != "executor manager not initialized" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestServer_RegisterExecutor_Success(t *testing.T) {
	t.Parallel()

	mgr := executor.NewRegistry()
	s := &Server{
		execution: ExecutionLayer{ExecutorManager: mgr},
	}

	mockExec := &mockExecutor{}
	err := s.RegisterExecutor("test-type", mockExec)
	if err != nil {
		t.Fatalf("RegisterExecutor returned error: %v", err)
	}

	// Verify executor was registered
	_, err = mgr.Get("test-type")
	if err != nil {
		t.Fatalf("Get registered executor failed: %v", err)
	}

	// Verify the type was registered
	if !mgr.Has("test-type") {
		t.Error("Executor was not registered properly")
	}
}

// mockExecutor is a simple mock for testing
type mockExecutor struct{}

func (m *mockExecutor) Execute(_ context.Context, _ map[string]any, _ any) (any, error) {
	return nil, nil
}

func (m *mockExecutor) Validate(_ map[string]any) error {
	return nil
}
