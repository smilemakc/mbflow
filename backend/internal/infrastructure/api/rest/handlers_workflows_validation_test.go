package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/executor/builtin"
	"github.com/smilemakc/mbflow/testutil"
)

// TestValidateNodes_ValidTypes tests that all registered executor types are valid
func TestValidateNodes_ValidTypes(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Cleanup(t)

	workflowRepo := storage.NewWorkflowRepository(testDB.DB)
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	executorManager := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executorManager))
	require.NoError(t, builtin.RegisterAdapters(executorManager))

	handlers := NewWorkflowHandlers(workflowRepo, log, executorManager)

	// Test valid executor types
	validTypes := []string{
		"http", "transform", "llm", "conditional", "merge",
		"html_clean", "rss_parser", "base64_to_bytes",
	}

	for _, nodeType := range validTypes {
		t.Run("valid_"+nodeType, func(t *testing.T) {
			nodes := []NodeRequest{
				{
					ID:   "node-1",
					Name: "Test Node",
					Type: nodeType,
				},
			}
			err := handlers.validateNodes(nodes)
			assert.NoError(t, err, "Type %s should be valid", nodeType)
		})
	}
}

// TestValidateNodes_UIOnlyTypes tests that UI-only types are valid
func TestValidateNodes_UIOnlyTypes(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Cleanup(t)

	workflowRepo := storage.NewWorkflowRepository(testDB.DB)
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	executorManager := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executorManager))

	handlers := NewWorkflowHandlers(workflowRepo, log, executorManager)

	nodes := []NodeRequest{
		{
			ID:   "comment-1",
			Name: "Comment Node",
			Type: "comment",
		},
	}

	err := handlers.validateNodes(nodes)
	assert.NoError(t, err, "Comment type should be valid (UI-only)")
}

// TestValidateNodes_InvalidType tests that unregistered types are rejected
func TestValidateNodes_InvalidType(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Cleanup(t)

	workflowRepo := storage.NewWorkflowRepository(testDB.DB)
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	executorManager := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executorManager))

	handlers := NewWorkflowHandlers(workflowRepo, log, executorManager)

	nodes := []NodeRequest{
		{
			ID:   "node-1",
			Name: "Invalid Node",
			Type: "nonexistent_type",
		},
	}

	err := handlers.validateNodes(nodes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type 'nonexistent_type'")
}

// TestValidateNodes_MixedTypes tests validation with both valid and invalid types
func TestValidateNodes_MixedTypes(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Cleanup(t)

	workflowRepo := storage.NewWorkflowRepository(testDB.DB)
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	executorManager := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executorManager))
	require.NoError(t, builtin.RegisterAdapters(executorManager))

	handlers := NewWorkflowHandlers(workflowRepo, log, executorManager)

	// Valid nodes
	validNodes := []NodeRequest{
		{ID: "node-1", Name: "HTTP Node", Type: "http"},
		{ID: "node-2", Name: "Transform Node", Type: "transform"},
		{ID: "node-3", Name: "Comment Node", Type: "comment"},
		{ID: "node-4", Name: "RSS Node", Type: "rss_parser"},
	}

	err := handlers.validateNodes(validNodes)
	assert.NoError(t, err)

	// Invalid node mixed with valid ones
	invalidNodes := []NodeRequest{
		{ID: "node-1", Name: "HTTP Node", Type: "http"},
		{ID: "node-2", Name: "Invalid Node", Type: "invalid_type"},
	}

	err = handlers.validateNodes(invalidNodes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type 'invalid_type'")
}

// TestValidateNodes_RequiredFields tests required field validation
func TestValidateNodes_RequiredFields(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Cleanup(t)

	workflowRepo := storage.NewWorkflowRepository(testDB.DB)
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	executorManager := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executorManager))

	handlers := NewWorkflowHandlers(workflowRepo, log, executorManager)

	tests := []struct {
		name        string
		node        NodeRequest
		expectedErr string
	}{
		{
			name:        "missing_id",
			node:        NodeRequest{Name: "Test", Type: "http"},
			expectedErr: "id is required",
		},
		{
			name:        "missing_name",
			node:        NodeRequest{ID: "node-1", Type: "http"},
			expectedErr: "name is required",
		},
		{
			name:        "missing_type",
			node:        NodeRequest{ID: "node-1", Name: "Test"},
			expectedErr: "type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := []NodeRequest{tt.node}
			err := handlers.validateNodes(nodes)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// TestValidateNodes_DuplicateIDs tests duplicate node ID validation
func TestValidateNodes_DuplicateIDs(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Cleanup(t)

	workflowRepo := storage.NewWorkflowRepository(testDB.DB)
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	executorManager := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executorManager))

	handlers := NewWorkflowHandlers(workflowRepo, log, executorManager)

	nodes := []NodeRequest{
		{ID: "node-1", Name: "First", Type: "http"},
		{ID: "node-1", Name: "Duplicate", Type: "transform"},
	}

	err := handlers.validateNodes(nodes)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate node id: node-1")
}

// TestValidateNodes_FieldLengths tests field length validation
func TestValidateNodes_FieldLengths(t *testing.T) {
	testDB := testutil.SetupTestDB(t)
	defer testDB.Cleanup(t)

	workflowRepo := storage.NewWorkflowRepository(testDB.DB)
	log := logger.New(config.LoggingConfig{
		Level:  "error",
		Format: "text",
	})

	executorManager := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executorManager))

	handlers := NewWorkflowHandlers(workflowRepo, log, executorManager)

	tests := []struct {
		name        string
		node        NodeRequest
		expectedErr string
	}{
		{
			name: "id_too_long",
			node: NodeRequest{
				ID:   string(make([]byte, 101)), // 101 chars
				Name: "Test",
				Type: "http",
			},
			expectedErr: "node id too long",
		},
		{
			name: "name_too_long",
			node: NodeRequest{
				ID:   "node-1",
				Name: string(make([]byte, 256)), // 256 chars
				Type: "http",
			},
			expectedErr: "name too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := []NodeRequest{tt.node}
			err := handlers.validateNodes(nodes)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}
