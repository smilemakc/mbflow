package builtin

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/application/filestorage"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Storage implementation for adapter tests
type adapterMockStorage struct {
	entries map[string]*models.FileEntry
	files   map[string][]byte
}

func newAdapterMockStorage() *adapterMockStorage {
	return &adapterMockStorage{
		entries: make(map[string]*models.FileEntry),
		files:   make(map[string][]byte),
	}
}

func (m *adapterMockStorage) Store(ctx context.Context, entry *models.FileEntry, reader io.Reader) (*models.FileEntry, error) {
	// Read data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Store data and entry
	m.files[entry.ID] = data
	storedEntry := *entry
	storedEntry.Size = int64(len(data))
	storedEntry.Checksum = "mock-checksum"
	m.entries[entry.ID] = &storedEntry

	return &storedEntry, nil
}

func (m *adapterMockStorage) Get(ctx context.Context, fileID string) (*models.FileEntry, io.ReadCloser, error) {
	entry, ok := m.entries[fileID]
	if !ok {
		return nil, nil, assert.AnError
	}

	data, ok := m.files[fileID]
	if !ok {
		return nil, nil, assert.AnError
	}

	return entry, io.NopCloser(bytes.NewReader(data)), nil
}

func (m *adapterMockStorage) Delete(ctx context.Context, fileID string) error {
	delete(m.files, fileID)
	delete(m.entries, fileID)
	return nil
}

func (m *adapterMockStorage) List(ctx context.Context, query *filestorage.FileQuery) ([]*models.FileEntry, error) {
	var entries []*models.FileEntry
	for _, entry := range m.entries {
		entries = append(entries, entry)
	}
	return entries, nil
}

func (m *adapterMockStorage) Exists(ctx context.Context, fileID string) (bool, error) {
	_, ok := m.entries[fileID]
	return ok, nil
}

func (m *adapterMockStorage) GetMetadata(ctx context.Context, fileID string) (*models.FileEntry, error) {
	entry, ok := m.entries[fileID]
	if !ok {
		return nil, assert.AnError
	}
	return entry, nil
}

func (m *adapterMockStorage) UpdateMetadata(ctx context.Context, fileID string, metadata map[string]any) error {
	entry, ok := m.entries[fileID]
	if !ok {
		return assert.AnError
	}
	entry.Metadata = metadata
	return nil
}

func (m *adapterMockStorage) UpdateTags(ctx context.Context, fileID string, tags []string) error {
	entry, ok := m.entries[fileID]
	if !ok {
		return assert.AnError
	}
	entry.Tags = tags
	return nil
}

func (m *adapterMockStorage) GetUsage(ctx context.Context) (*models.StorageUsage, error) {
	var totalSize int64
	for _, entry := range m.entries {
		totalSize += entry.Size
	}
	return &models.StorageUsage{
		StorageID:    "default",
		TotalSize:    totalSize,
		FileCount:    int64(len(m.entries)),
		MaxSize:      0,
		UsagePercent: 0,
	}, nil
}

// Mock Manager implementation for adapter tests
type adapterMockManager struct {
	storage *adapterMockStorage
}

func newAdapterMockManager() *adapterMockManager {
	return &adapterMockManager{
		storage: newAdapterMockStorage(),
	}
}

func (m *adapterMockManager) GetStorage(storageID string) (filestorage.Storage, error) {
	if storageID != "default" && storageID != "test-storage" {
		return nil, assert.AnError
	}
	return m.storage, nil
}

func (m *adapterMockManager) CreateStorage(storageID string, config *models.StorageConfig) (filestorage.Storage, error) {
	return m.storage, nil
}

func (m *adapterMockManager) DeleteStorage(storageID string) error {
	return nil
}

func (m *adapterMockManager) ListStorages() []string {
	return []string{"default"}
}

func (m *adapterMockManager) HasStorage(storageID string) bool {
	return storageID == "default" || storageID == "test-storage"
}

func (m *adapterMockManager) GetDefaultStorage() (filestorage.Storage, error) {
	return m.storage, nil
}

func (m *adapterMockManager) RegisterObserver(observer filestorage.FileObserver) error {
	return nil
}

func (m *adapterMockManager) UnregisterObserver(name string) error {
	return nil
}

func (m *adapterMockManager) Cleanup(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *adapterMockManager) Close() error {
	return nil
}

func TestFileToBytesExecutor_Execute(t *testing.T) {
	manager := newAdapterMockManager()
	executor := NewFileToBytesExecutor(manager)
	ctx := context.Background()

	// Setup test file
	testData := []byte("test file content")
	fileID := uuid.New().String()
	entry := &models.FileEntry{
		ID:        fileID,
		StorageID: "default",
		Name:      "test.txt",
		MimeType:  "text/plain",
		Size:      int64(len(testData)),
		Checksum:  "checksum",
	}
	manager.storage.files[fileID] = testData
	manager.storage.entries[fileID] = entry

	tests := []struct {
		name          string
		config        map[string]any
		input         any
		expectError   bool
		errorContains string
		validateFunc  func(t *testing.T, result map[string]any)
	}{
		{
			name: "read file as base64 (default)",
			config: map[string]any{
				"storage_id":    "default",
				"file_id":       fileID,
				"output_format": "base64",
			},
			input:       nil,
			expectError: false,
			validateFunc: func(t *testing.T, result map[string]any) {
				assert.True(t, result["success"].(bool))
				assert.Equal(t, base64.StdEncoding.EncodeToString(testData), result["result"].(string))
				assert.Equal(t, fileID, result["file_id"].(string))
				assert.Equal(t, "test.txt", result["file_name"].(string))
				assert.Equal(t, "text/plain", result["mime_type"].(string))
				assert.Equal(t, int64(len(testData)), result["size"].(int64))
				assert.Equal(t, "base64", result["format"].(string))
			},
		},
		{
			name: "read file as raw bytes",
			config: map[string]any{
				"storage_id":    "default",
				"file_id":       fileID,
				"output_format": "raw",
			},
			input:       nil,
			expectError: false,
			validateFunc: func(t *testing.T, result map[string]any) {
				assert.True(t, result["success"].(bool))
				assert.Equal(t, testData, result["result"].([]byte))
				assert.Equal(t, "raw", result["format"].(string))
			},
		},
		{
			name: "file_id from input string",
			config: map[string]any{
				"storage_id": "default",
			},
			input:       fileID,
			expectError: false,
			validateFunc: func(t *testing.T, result map[string]any) {
				assert.True(t, result["success"].(bool))
				assert.Equal(t, fileID, result["file_id"].(string))
			},
		},
		{
			name: "file_id from input map",
			config: map[string]any{
				"storage_id": "default",
			},
			input: map[string]any{
				"file_id": fileID,
			},
			expectError: false,
			validateFunc: func(t *testing.T, result map[string]any) {
				assert.True(t, result["success"].(bool))
				assert.Equal(t, fileID, result["file_id"].(string))
			},
		},
		{
			name: "invalid output format",
			config: map[string]any{
				"storage_id":    "default",
				"file_id":       fileID,
				"output_format": "invalid",
			},
			input:         nil,
			expectError:   true,
			errorContains: "invalid output_format",
		},
		{
			name: "storage not found",
			config: map[string]any{
				"storage_id": "non-existent",
				"file_id":    fileID,
			},
			input:         nil,
			expectError:   true,
			errorContains: "failed to get storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, tt.config, tt.input)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			resultMap, ok := result.(map[string]any)
			require.True(t, ok, "result should be a map")

			if tt.validateFunc != nil {
				tt.validateFunc(t, resultMap)
			}
		})
	}
}

func TestFileToBytesExecutor_Validate(t *testing.T) {
	manager := newAdapterMockManager()
	executor := NewFileToBytesExecutor(manager)

	tests := []struct {
		name          string
		config        map[string]any
		expectError   bool
		errorContains string
	}{
		{
			name: "valid config with base64",
			config: map[string]any{
				"storage_id":    "default",
				"file_id":       "test-id",
				"output_format": "base64",
			},
			expectError: false,
		},
		{
			name: "valid config with raw",
			config: map[string]any{
				"storage_id":    "default",
				"file_id":       "test-id",
				"output_format": "raw",
			},
			expectError: false,
		},
		{
			name: "missing file_id",
			config: map[string]any{
				"storage_id": "default",
			},
			expectError:   true,
			errorContains: "file_id is required",
		},
		{
			name: "invalid output_format",
			config: map[string]any{
				"storage_id":    "default",
				"file_id":       "test-id",
				"output_format": "json",
			},
			expectError:   true,
			errorContains: "invalid output_format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestBytesToFileExecutor_Execute(t *testing.T) {
	manager := newAdapterMockManager()
	executor := NewBytesToFileExecutor(manager)
	ctx := context.Background()

	testData := []byte("test file content")

	tests := []struct {
		name          string
		config        map[string]any
		input         any
		expectError   bool
		errorContains string
		validateFunc  func(t *testing.T, result map[string]any)
	}{
		{
			name: "save bytes to file",
			config: map[string]any{
				"storage_id":   "default",
				"file_name":    "output.txt",
				"mime_type":    "text/plain",
				"access_scope": "workflow",
				"ttl":          0,
			},
			input:       testData,
			expectError: false,
			validateFunc: func(t *testing.T, result map[string]any) {
				assert.True(t, result["success"].(bool))
				assert.NotEmpty(t, result["file_id"].(string))
				assert.Equal(t, "default", result["storage_id"].(string))
				assert.Equal(t, "output.txt", result["file_name"].(string))
				assert.Equal(t, "text/plain", result["mime_type"].(string))
				assert.Equal(t, int64(len(testData)), result["size"].(int64))
				assert.Equal(t, "workflow", result["access_scope"].(string))
			},
		},
		{
			name: "save base64 string to file",
			config: map[string]any{
				"storage_id": "default",
				"file_name":  "encoded.bin",
			},
			input:       base64.StdEncoding.EncodeToString(testData),
			expectError: false,
			validateFunc: func(t *testing.T, result map[string]any) {
				assert.True(t, result["success"].(bool))
				// Should auto-decode base64
				fileID := result["file_id"].(string)
				storedData := manager.storage.files[fileID]
				assert.Equal(t, testData, storedData)
			},
		},
		{
			name: "save with auto mime type detection",
			config: map[string]any{
				"storage_id": "default",
				"file_name":  "data.json",
			},
			input:       []byte(`{"test":true}`),
			expectError: false,
			validateFunc: func(t *testing.T, result map[string]any) {
				assert.True(t, result["success"].(bool))
				// MIME type should be auto-detected
				assert.NotEmpty(t, result["mime_type"].(string))
			},
		},
		{
			name: "save with edge access scope",
			config: map[string]any{
				"storage_id":   "default",
				"file_name":    "edge.bin",
				"access_scope": "edge",
			},
			input:       testData,
			expectError: false,
			validateFunc: func(t *testing.T, result map[string]any) {
				assert.Equal(t, "edge", result["access_scope"].(string))
			},
		},
		{
			name: "save with tags",
			config: map[string]any{
				"storage_id": "default",
				"file_name":  "tagged.bin",
				"tags":       []any{"processed", "output"},
			},
			input:       testData,
			expectError: false,
		},
		{
			name: "input from map with data field",
			config: map[string]any{
				"storage_id": "default",
				"file_name":  "from-map.bin",
			},
			input: map[string]any{
				"data": testData,
			},
			expectError: false,
		},
		{
			name: "missing file_name",
			config: map[string]any{
				"storage_id": "default",
			},
			input:         testData,
			expectError:   true,
			errorContains: "file_name is required",
		},
		{
			name: "invalid storage",
			config: map[string]any{
				"storage_id": "non-existent",
				"file_name":  "test.bin",
			},
			input:         testData,
			expectError:   true,
			errorContains: "failed to get storage",
		},
		{
			name: "invalid access_scope",
			config: map[string]any{
				"storage_id":   "default",
				"file_name":    "test.bin",
				"access_scope": "invalid",
			},
			input:         testData,
			expectError:   true,
			errorContains: "invalid access_scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, tt.config, tt.input)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			resultMap, ok := result.(map[string]any)
			require.True(t, ok, "result should be a map")

			if tt.validateFunc != nil {
				tt.validateFunc(t, resultMap)
			}
		})
	}
}

func TestBytesToFileExecutor_Validate(t *testing.T) {
	manager := newAdapterMockManager()
	executor := NewBytesToFileExecutor(manager)

	tests := []struct {
		name          string
		config        map[string]any
		expectError   bool
		errorContains string
	}{
		{
			name: "valid config",
			config: map[string]any{
				"storage_id":   "default",
				"file_name":    "test.txt",
				"access_scope": "workflow",
				"ttl":          3600,
			},
			expectError: false,
		},
		{
			name: "missing file_name",
			config: map[string]any{
				"storage_id": "default",
			},
			expectError:   true,
			errorContains: "file_name is required",
		},
		{
			name: "invalid access_scope",
			config: map[string]any{
				"storage_id":   "default",
				"file_name":    "test.txt",
				"access_scope": "invalid",
			},
			expectError:   true,
			errorContains: "invalid access_scope",
		},
		{
			name: "negative TTL",
			config: map[string]any{
				"storage_id": "default",
				"file_name":  "test.txt",
				"ttl":        -100,
			},
			expectError:   true,
			errorContains: "ttl must be >= 0",
		},
		{
			name: "valid edge scope",
			config: map[string]any{
				"storage_id":   "default",
				"file_name":    "test.txt",
				"access_scope": "edge",
			},
			expectError: false,
		},
		{
			name: "valid result scope",
			config: map[string]any{
				"storage_id":   "default",
				"file_name":    "test.txt",
				"access_scope": "result",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestBytesToFileExecutor_ExtractBytes(t *testing.T) {
	manager := newAdapterMockManager()
	executor := NewBytesToFileExecutor(manager)

	tests := []struct {
		name          string
		input         any
		expected      []byte
		expectError   bool
		errorContains string
	}{
		{
			name:     "direct byte slice",
			input:    []byte("test"),
			expected: []byte("test"),
		},
		{
			name:     "string input (not base64)",
			input:    "hello",
			expected: []byte("hello"),
		},
		{
			name:     "base64 string input",
			input:    base64.StdEncoding.EncodeToString([]byte("decoded")),
			expected: []byte("decoded"),
		},
		{
			name:     "map with data field (bytes)",
			input:    map[string]any{"data": []byte("test")},
			expected: []byte("test"),
		},
		{
			name:     "map with data field (string)",
			input:    map[string]any{"data": "plaintext"},
			expected: []byte("plaintext"),
		},
		{
			name:          "map without data field",
			input:         map[string]any{"other": "value"},
			expectError:   true,
			errorContains: "expected 'data' field",
		},
		{
			name:          "unsupported type",
			input:         12345,
			expectError:   true,
			errorContains: "unsupported input type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.extractBytes(tt.input)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
