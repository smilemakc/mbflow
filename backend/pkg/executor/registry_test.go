package executor

import (
	"context"
	"testing"

	"github.com/smilemakc/mbflow/pkg/models"
)

// mockExecutor is a simple mock for testing
type mockExecutor struct {
	validateFn func(config map[string]interface{}) error
	executeFn  func(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error)
}

func (m *mockExecutor) Validate(config map[string]interface{}) error {
	if m.validateFn != nil {
		return m.validateFn(config)
	}
	return nil
}

func (m *mockExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, config, input)
	}
	return map[string]interface{}{"status": "ok"}, nil
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if registry.executors == nil {
		t.Error("registry.executors is nil")
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name     string
		nodeType string
		executor Executor
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "register valid executor",
			nodeType: "http",
			executor: &mockExecutor{},
			wantErr:  false,
		},
		{
			name:     "register with empty node type",
			nodeType: "",
			executor: &mockExecutor{},
			wantErr:  true,
			errMsg:   "node type cannot be empty",
		},
		{
			name:     "register nil executor",
			nodeType: "http",
			executor: nil,
			wantErr:  true,
			errMsg:   "executor cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			err := registry.Register(tt.nodeType, tt.executor)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	mockExec := &mockExecutor{}

	// Register an executor
	err := registry.Register("http", mockExec)
	if err != nil {
		t.Fatalf("failed to register executor: %v", err)
	}

	tests := []struct {
		name     string
		nodeType string
		wantErr  bool
		wantNil  bool
	}{
		{
			name:     "get existing executor",
			nodeType: "http",
			wantErr:  false,
			wantNil:  false,
		},
		{
			name:     "get non-existent executor",
			nodeType: "nonexistent",
			wantErr:  true,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec, err := registry.Get(tt.nodeType)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				// Check if error is ErrExecutorNotFound
				if !containsError(err, models.ErrExecutorNotFound) {
					t.Errorf("expected ErrExecutorNotFound, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.wantNil {
				if exec != nil {
					t.Error("expected nil executor")
				}
			} else {
				if exec == nil {
					t.Error("executor is nil")
				}
			}
		})
	}
}

func TestRegistry_Has(t *testing.T) {
	registry := NewRegistry()
	mockExec := &mockExecutor{}

	// Register an executor
	registry.Register("http", mockExec)

	tests := []struct {
		name     string
		nodeType string
		expected bool
	}{
		{
			name:     "has existing executor",
			nodeType: "http",
			expected: true,
		},
		{
			name:     "does not have non-existent executor",
			nodeType: "nonexistent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has := registry.Has(tt.nodeType)
			if has != tt.expected {
				t.Errorf("Has(%s) = %v, want %v", tt.nodeType, has, tt.expected)
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Empty registry
	list := registry.List()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}

	// Register some executors
	registry.Register("http", &mockExecutor{})
	registry.Register("transform", &mockExecutor{})
	registry.Register("llm", &mockExecutor{})

	list = registry.List()
	if len(list) != 3 {
		t.Errorf("expected 3 items, got %d", len(list))
	}

	// Verify all types are present
	types := make(map[string]bool)
	for _, nodeType := range list {
		types[nodeType] = true
	}

	expectedTypes := []string{"http", "transform", "llm"}
	for _, expected := range expectedTypes {
		if !types[expected] {
			t.Errorf("expected type %s not found in list", expected)
		}
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()
	mockExec := &mockExecutor{}

	// Register an executor
	registry.Register("http", mockExec)

	tests := []struct {
		name     string
		nodeType string
		wantErr  bool
	}{
		{
			name:     "unregister existing executor",
			nodeType: "http",
			wantErr:  false,
		},
		{
			name:     "unregister non-existent executor",
			nodeType: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Unregister(tt.nodeType)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if !containsError(err, models.ErrExecutorNotFound) {
					t.Errorf("expected ErrExecutorNotFound, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Verify executor is actually removed
				if registry.Has(tt.nodeType) {
					t.Errorf("executor %s still exists after unregister", tt.nodeType)
				}
			}
		})
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	registry := NewRegistry()
	done := make(chan bool)

	// Test concurrent registrations
	go func() {
		for i := 0; i < 100; i++ {
			registry.Register("type1", &mockExecutor{})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			registry.Get("type1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			registry.Has("type1")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			registry.List()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify registry is still functional
	if !registry.Has("type1") {
		t.Error("registry corrupted after concurrent access")
	}
}

func TestGlobalRegistry_Register(t *testing.T) {
	// Note: This test uses the global registry, so it may affect other tests
	// In a real scenario, you might want to reset the global registry between tests

	mockExec := &mockExecutor{}
	err := Register("test-global", mockExec)
	if err != nil {
		t.Errorf("global Register() failed: %v", err)
	}

	// Clean up
	defer Unregister("test-global")
}

func TestGlobalRegistry_Get(t *testing.T) {
	mockExec := &mockExecutor{}
	Register("test-global-get", mockExec)
	defer Unregister("test-global-get")

	exec, err := Get("test-global-get")
	if err != nil {
		t.Errorf("global Get() failed: %v", err)
	}
	if exec == nil {
		t.Error("global Get() returned nil executor")
	}
}

func TestGlobalRegistry_Has(t *testing.T) {
	mockExec := &mockExecutor{}
	Register("test-global-has", mockExec)
	defer Unregister("test-global-has")

	if !Has("test-global-has") {
		t.Error("global Has() returned false for registered executor")
	}
}

func TestGlobalRegistry_List(t *testing.T) {
	// Register a test executor
	mockExec := &mockExecutor{}
	Register("test-global-list", mockExec)
	defer Unregister("test-global-list")

	list := List()
	found := false
	for _, nodeType := range list {
		if nodeType == "test-global-list" {
			found = true
			break
		}
	}
	if !found {
		t.Error("global List() did not contain registered executor")
	}
}

func TestGlobalRegistry_Unregister(t *testing.T) {
	mockExec := &mockExecutor{}
	Register("test-global-unreg", mockExec)

	err := Unregister("test-global-unreg")
	if err != nil {
		t.Errorf("global Unregister() failed: %v", err)
	}

	if Has("test-global-unreg") {
		t.Error("executor still exists after global Unregister()")
	}
}

// Helper function to check if error contains another error
func containsError(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	// Simple contains check for error messages
	return err.Error() != "" && target.Error() != "" &&
		len(err.Error()) >= len(target.Error()) &&
		contains(err.Error(), target.Error())
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
