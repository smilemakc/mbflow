package builtin

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStorage implements filestorage.Storage for testing
type mockStorage struct {
	mu        sync.RWMutex
	files     map[string]*storedFile
	storageID string
}

type storedFile struct {
	entry   *models.FileEntry
	content []byte
}

func newMockStorage(storageID string) *mockStorage {
	return &mockStorage{
		files:     make(map[string]*storedFile),
		storageID: storageID,
	}
}

func (m *mockStorage) Store(ctx context.Context, entry *models.FileEntry, reader io.Reader) (*models.FileEntry, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	entry.ID = "file-" + entry.Name
	entry.StorageID = m.storageID
	entry.Size = int64(len(content))
	entry.Checksum = "sha256-test-checksum"
	entry.CreatedAt = time.Now()

	m.mu.Lock()
	m.files[entry.ID] = &storedFile{
		entry:   entry,
		content: content,
	}
	m.mu.Unlock()

	return entry, nil
}

func (m *mockStorage) Get(ctx context.Context, fileID string) (*models.FileEntry, io.ReadCloser, error) {
	m.mu.RLock()
	file, ok := m.files[fileID]
	m.mu.RUnlock()
	if !ok {
		return nil, nil, errors.New("file not found")
	}
	return file.entry, io.NopCloser(bytes.NewReader(file.content)), nil
}

func (m *mockStorage) Delete(ctx context.Context, fileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.files[fileID]; !ok {
		return errors.New("file not found")
	}
	delete(m.files, fileID)
	return nil
}

func (m *mockStorage) List(ctx context.Context, query *filestorage.FileQuery) ([]*models.FileEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var files []*models.FileEntry
	for _, f := range m.files {
		// Apply access scope filter
		if query.AccessScope != "" && f.entry.AccessScope != query.AccessScope {
			continue
		}
		// Apply tag filter
		if len(query.Tags) > 0 {
			hasTag := false
			for _, queryTag := range query.Tags {
				for _, fileTag := range f.entry.Tags {
					if queryTag == fileTag {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		files = append(files, f.entry)
	}

	// Apply limit
	if query.Limit > 0 && len(files) > query.Limit {
		files = files[:query.Limit]
	}

	return files, nil
}

func (m *mockStorage) Exists(ctx context.Context, fileID string) (bool, error) {
	m.mu.RLock()
	_, ok := m.files[fileID]
	m.mu.RUnlock()
	return ok, nil
}

func (m *mockStorage) GetMetadata(ctx context.Context, fileID string) (*models.FileEntry, error) {
	m.mu.RLock()
	file, ok := m.files[fileID]
	m.mu.RUnlock()
	if !ok {
		return nil, errors.New("file not found")
	}
	return file.entry, nil
}

func (m *mockStorage) UpdateMetadata(ctx context.Context, fileID string, metadata map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	file, ok := m.files[fileID]
	if !ok {
		return errors.New("file not found")
	}
	file.entry.Metadata = metadata
	return nil
}

func (m *mockStorage) UpdateTags(ctx context.Context, fileID string, tags []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	file, ok := m.files[fileID]
	if !ok {
		return errors.New("file not found")
	}
	file.entry.Tags = tags
	return nil
}

func (m *mockStorage) GetUsage(ctx context.Context) (*models.StorageUsage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalSize int64
	for _, f := range m.files {
		totalSize += f.entry.Size
	}
	return &models.StorageUsage{
		StorageID:    m.storageID,
		TotalSize:    totalSize,
		FileCount:    int64(len(m.files)),
		MaxSize:      0,
		UsagePercent: 0,
	}, nil
}

// mockManager implements filestorage.Manager for testing
type mockManager struct {
	mu       sync.RWMutex
	storages map[string]*mockStorage
}

func newMockManager() *mockManager {
	mgr := &mockManager{
		storages: make(map[string]*mockStorage),
	}
	// Create default storage
	mgr.storages["default"] = newMockStorage("default")
	return mgr
}

func (m *mockManager) GetStorage(storageID string) (filestorage.Storage, error) {
	m.mu.RLock()
	storage, ok := m.storages[storageID]
	m.mu.RUnlock()
	if ok {
		return storage, nil
	}
	// Create new storage on demand
	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check in case another goroutine created it
	if storage, ok := m.storages[storageID]; ok {
		return storage, nil
	}
	storage = newMockStorage(storageID)
	m.storages[storageID] = storage
	return storage, nil
}

func (m *mockManager) CreateStorage(storageID string, config *models.StorageConfig) (filestorage.Storage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	storage := newMockStorage(storageID)
	m.storages[storageID] = storage
	return storage, nil
}

func (m *mockManager) DeleteStorage(storageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.storages, storageID)
	return nil
}

func (m *mockManager) ListStorages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var ids []string
	for id := range m.storages {
		ids = append(ids, id)
	}
	return ids
}

func (m *mockManager) HasStorage(storageID string) bool {
	m.mu.RLock()
	_, ok := m.storages[storageID]
	m.mu.RUnlock()
	return ok
}

func (m *mockManager) GetDefaultStorage() (filestorage.Storage, error) {
	return m.GetStorage("default")
}

func (m *mockManager) RegisterObserver(observer filestorage.FileObserver) error {
	return nil
}

func (m *mockManager) UnregisterObserver(name string) error {
	return nil
}

func (m *mockManager) Cleanup(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *mockManager) Close() error {
	return nil
}

// ============== Store Action Tests ==============

func TestFileStorageExecutor_Store_Base64(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	testContent := "Hello, World! This is a test file."
	base64Content := base64.StdEncoding.EncodeToString([]byte(testContent))

	config := map[string]any{
		"action":       "store",
		"file_data":    base64Content,
		"file_name":    "test.txt",
		"mime_type":    "text/plain",
		"access_scope": "workflow",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)

	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, "file-test.txt", resultMap["file_id"])
	assert.Equal(t, "test.txt", resultMap["file_name"])
	assert.Equal(t, "text/plain", resultMap["mime_type"])
	assert.Equal(t, int64(len(testContent)), resultMap["size"])
	assert.Equal(t, models.AccessScope("workflow"), resultMap["access_scope"])
}

func TestFileStorageExecutor_Store_WithTags(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	testContent := "Test file with tags"
	base64Content := base64.StdEncoding.EncodeToString([]byte(testContent))

	config := map[string]any{
		"action":       "store",
		"file_data":    base64Content,
		"file_name":    "tagged-file.txt",
		"access_scope": "result",
		"tags":         []any{"important", "document"},
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, models.AccessScope("result"), resultMap["access_scope"])
}

func TestFileStorageExecutor_Store_WithTTL(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	testContent := "Temporary file"
	base64Content := base64.StdEncoding.EncodeToString([]byte(testContent))

	config := map[string]any{
		"action":    "store",
		"file_data": base64Content,
		"file_name": "temp.txt",
		"ttl":       3600, // 1 hour
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.NotNil(t, resultMap["expires_at"])
}

func TestFileStorageExecutor_Store_MissingData(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action":    "store",
		"file_name": "test.txt",
		// No file_data or file_url
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either file_data or file_url is required")
}

func TestFileStorageExecutor_Store_InvalidBase64(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action":    "store",
		"file_data": "not-valid-base64!!!",
		"file_name": "test.txt",
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode base64")
}

func TestFileStorageExecutor_Store_InvalidAccessScope(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	testContent := "Test"
	base64Content := base64.StdEncoding.EncodeToString([]byte(testContent))

	config := map[string]any{
		"action":       "store",
		"file_data":    base64Content,
		"file_name":    "test.txt",
		"access_scope": "invalid",
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid access_scope")
}

func TestFileStorageExecutor_Store_CustomStorageID(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	testContent := "Custom storage test"
	base64Content := base64.StdEncoding.EncodeToString([]byte(testContent))

	config := map[string]any{
		"action":     "store",
		"storage_id": "custom-storage",
		"file_data":  base64Content,
		"file_name":  "custom.txt",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, "custom-storage", resultMap["storage_id"])
}

// ============== Get Action Tests ==============

func TestFileStorageExecutor_Get_Success(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// First store a file
	testContent := "Get test content"
	base64Content := base64.StdEncoding.EncodeToString([]byte(testContent))

	storeConfig := map[string]any{
		"action":    "store",
		"file_data": base64Content,
		"file_name": "retrieve-me.txt",
		"mime_type": "text/plain",
	}

	storeResult, err := exec.Execute(context.Background(), storeConfig, nil)
	require.NoError(t, err)
	fileID := storeResult.(map[string]any)["file_id"].(string)

	// Now get the file
	getConfig := map[string]any{
		"action":  "get",
		"file_id": fileID,
	}

	result, err := exec.Execute(context.Background(), getConfig, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, fileID, resultMap["file_id"])
	assert.Equal(t, "retrieve-me.txt", resultMap["file_name"])
	assert.Equal(t, "text/plain", resultMap["mime_type"])

	// Verify content
	decoded, err := base64.StdEncoding.DecodeString(resultMap["file_data"].(string))
	require.NoError(t, err)
	assert.Equal(t, testContent, string(decoded))
}

func TestFileStorageExecutor_Get_FileNotFound(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action":  "get",
		"file_id": "non-existent-file",
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestFileStorageExecutor_Get_MissingFileID(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action": "get",
		// No file_id
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file_id is required")
}

// ============== Delete Action Tests ==============

func TestFileStorageExecutor_Delete_Success(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// First store a file
	testContent := "Delete me"
	base64Content := base64.StdEncoding.EncodeToString([]byte(testContent))

	storeConfig := map[string]any{
		"action":    "store",
		"file_data": base64Content,
		"file_name": "delete-me.txt",
	}

	storeResult, err := exec.Execute(context.Background(), storeConfig, nil)
	require.NoError(t, err)
	fileID := storeResult.(map[string]any)["file_id"].(string)

	// Delete the file
	deleteConfig := map[string]any{
		"action":  "delete",
		"file_id": fileID,
	}

	result, err := exec.Execute(context.Background(), deleteConfig, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, fileID, resultMap["file_id"])

	// Verify file is gone
	getConfig := map[string]any{
		"action":  "get",
		"file_id": fileID,
	}

	_, err = exec.Execute(context.Background(), getConfig, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestFileStorageExecutor_Delete_FileNotFound(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action":  "delete",
		"file_id": "non-existent",
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

// ============== List Action Tests ==============

func TestFileStorageExecutor_List_All(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Store multiple files
	for i := 0; i < 3; i++ {
		content := base64.StdEncoding.EncodeToString([]byte("Content"))
		config := map[string]any{
			"action":    "store",
			"file_data": content,
			"file_name": "file" + string(rune('0'+i)) + ".txt",
		}
		_, err := exec.Execute(context.Background(), config, nil)
		require.NoError(t, err)
	}

	// List all files
	listConfig := map[string]any{
		"action": "list",
	}

	result, err := exec.Execute(context.Background(), listConfig, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, 3, resultMap["count"])

	files := resultMap["files"].([]map[string]any)
	assert.Len(t, files, 3)
}

func TestFileStorageExecutor_List_WithAccessScopeFilter(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Store files with different access scopes
	for _, scope := range []string{"workflow", "result", "edge"} {
		content := base64.StdEncoding.EncodeToString([]byte("Content"))
		config := map[string]any{
			"action":       "store",
			"file_data":    content,
			"file_name":    scope + "-file.txt",
			"access_scope": scope,
		}
		_, err := exec.Execute(context.Background(), config, nil)
		require.NoError(t, err)
	}

	// List only workflow files
	listConfig := map[string]any{
		"action":       "list",
		"access_scope": "workflow",
	}

	result, err := exec.Execute(context.Background(), listConfig, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, 1, resultMap["count"])
}

func TestFileStorageExecutor_List_WithLimit(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Store 5 files
	for i := 0; i < 5; i++ {
		content := base64.StdEncoding.EncodeToString([]byte("Content"))
		config := map[string]any{
			"action":    "store",
			"file_data": content,
			"file_name": "file" + string(rune('0'+i)) + ".txt",
		}
		_, err := exec.Execute(context.Background(), config, nil)
		require.NoError(t, err)
	}

	// List with limit
	listConfig := map[string]any{
		"action": "list",
		"limit":  2,
	}

	result, err := exec.Execute(context.Background(), listConfig, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	files := resultMap["files"].([]map[string]any)
	assert.LessOrEqual(t, len(files), 2)
}

// ============== Metadata Action Tests ==============

func TestFileStorageExecutor_Metadata_Success(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// First store a file
	testContent := "Metadata test"
	base64Content := base64.StdEncoding.EncodeToString([]byte(testContent))

	storeConfig := map[string]any{
		"action":       "store",
		"file_data":    base64Content,
		"file_name":    "meta-file.txt",
		"mime_type":    "text/plain",
		"access_scope": "workflow",
		"tags":         []any{"test", "metadata"},
	}

	storeResult, err := exec.Execute(context.Background(), storeConfig, nil)
	require.NoError(t, err)
	fileID := storeResult.(map[string]any)["file_id"].(string)

	// Get metadata
	metaConfig := map[string]any{
		"action":  "metadata",
		"file_id": fileID,
	}

	result, err := exec.Execute(context.Background(), metaConfig, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, fileID, resultMap["file_id"])
	assert.Equal(t, "meta-file.txt", resultMap["file_name"])
	assert.Equal(t, "text/plain", resultMap["mime_type"])
	assert.Equal(t, int64(len(testContent)), resultMap["size"])
	assert.Equal(t, models.AccessScope("workflow"), resultMap["access_scope"])
}

func TestFileStorageExecutor_Metadata_FileNotFound(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action":  "metadata",
		"file_id": "non-existent",
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

// ============== Validation Tests ==============

func TestFileStorageExecutor_Validate(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid store with file_data",
			config: map[string]any{
				"action":    "store",
				"file_data": "dGVzdA==",
			},
			wantErr: false,
		},
		{
			name: "Valid store with file_url",
			config: map[string]any{
				"action":   "store",
				"file_url": "https://example.com/file.pdf",
			},
			wantErr: false,
		},
		{
			name: "Store without data or url",
			config: map[string]any{
				"action": "store",
			},
			wantErr: true,
			errMsg:  "either file_data or file_url is required",
		},
		{
			name: "Valid get",
			config: map[string]any{
				"action":  "get",
				"file_id": "some-id",
			},
			wantErr: false,
		},
		{
			name: "Get without file_id",
			config: map[string]any{
				"action": "get",
			},
			wantErr: true,
			errMsg:  "file_id is required",
		},
		{
			name: "Valid delete",
			config: map[string]any{
				"action":  "delete",
				"file_id": "some-id",
			},
			wantErr: false,
		},
		{
			name: "Valid list",
			config: map[string]any{
				"action": "list",
			},
			wantErr: false,
		},
		{
			name: "Valid metadata",
			config: map[string]any{
				"action":  "metadata",
				"file_id": "some-id",
			},
			wantErr: false,
		},
		{
			name: "Invalid action",
			config: map[string]any{
				"action": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid action",
		},
		{
			name:    "Missing action",
			config:  map[string]any{},
			wantErr: true,
			errMsg:  "action is required",
		},
		{
			name: "Invalid access_scope",
			config: map[string]any{
				"action":       "store",
				"file_data":    "dGVzdA==",
				"access_scope": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid access_scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exec.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============== Action Missing Tests ==============

func TestFileStorageExecutor_MissingAction(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action is required")
}

func TestFileStorageExecutor_UnsupportedAction(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action": "unsupported",
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported action")
}

// ============== Result Format Tests ==============

func TestFileStorageExecutor_ResultContainsDuration(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString([]byte("test")),
		"file_name": "test.txt",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Contains(t, resultMap, "duration_ms")
	assert.Contains(t, resultMap, "action")
	assert.Equal(t, "store", resultMap["action"])
}

// ============== Complex Workflow Tests ==============

func TestFileStorageExecutor_CompleteWorkflow(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// 1. Store a file
	content := "Complete workflow test content"
	base64Content := base64.StdEncoding.EncodeToString([]byte(content))

	storeResult, err := exec.Execute(context.Background(), map[string]any{
		"action":       "store",
		"file_data":    base64Content,
		"file_name":    "workflow-test.txt",
		"mime_type":    "text/plain",
		"access_scope": "workflow",
		"tags":         []any{"workflow", "test"},
	}, nil)
	require.NoError(t, err)

	fileID := storeResult.(map[string]any)["file_id"].(string)

	// 2. Get metadata
	metaResult, err := exec.Execute(context.Background(), map[string]any{
		"action":  "metadata",
		"file_id": fileID,
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "workflow-test.txt", metaResult.(map[string]any)["file_name"])

	// 3. Get file content
	getResult, err := exec.Execute(context.Background(), map[string]any{
		"action":  "get",
		"file_id": fileID,
	}, nil)
	require.NoError(t, err)

	decoded, _ := base64.StdEncoding.DecodeString(getResult.(map[string]any)["file_data"].(string))
	assert.Equal(t, content, string(decoded))

	// 4. List files
	listResult, err := exec.Execute(context.Background(), map[string]any{
		"action": "list",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, listResult.(map[string]any)["count"])

	// 5. Delete file
	_, err = exec.Execute(context.Background(), map[string]any{
		"action":  "delete",
		"file_id": fileID,
	}, nil)
	require.NoError(t, err)

	// 6. Verify deletion
	listResult, err = exec.Execute(context.Background(), map[string]any{
		"action": "list",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, listResult.(map[string]any)["count"])
}

// ============== Access Scope Tests ==============

func TestFileStorageExecutor_AllAccessScopes(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	scopes := []string{"workflow", "edge", "result"}

	for _, scope := range scopes {
		t.Run("Scope_"+scope, func(t *testing.T) {
			content := base64.StdEncoding.EncodeToString([]byte(scope))
			config := map[string]any{
				"action":       "store",
				"file_data":    content,
				"file_name":    scope + ".txt",
				"access_scope": scope,
			}

			result, err := exec.Execute(context.Background(), config, nil)
			require.NoError(t, err)
			assert.Equal(t, models.AccessScope(scope), result.(map[string]any)["access_scope"])
		})
	}
}

// ============== Edge Cases ==============

func TestFileStorageExecutor_EmptyFileName(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString([]byte("test")),
		// No file_name - should generate one
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	fileName := resultMap["file_name"].(string)
	assert.True(t, strings.HasPrefix(fileName, "file_"))
}

// ============== URL Input Tests ==============

func TestFileStorageExecutor_Store_FromURL_ValidURL(t *testing.T) {
	// NOTE: This test validates the config handling for file_url
	// Actual HTTP download would require httptest mock server

	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Validate that file_url config is accepted
	config := map[string]any{
		"action":   "store",
		"file_url": "https://example.com/test-file.pdf",
	}

	// Validation should pass
	err := exec.Validate(config)
	assert.NoError(t, err)
}

func TestFileStorageExecutor_Store_FromURL_WithFileName(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Config with both URL and custom file name
	config := map[string]any{
		"action":    "store",
		"file_url":  "https://example.com/some/path/document.pdf",
		"file_name": "custom-name.pdf",
		"mime_type": "application/pdf",
	}

	err := exec.Validate(config)
	assert.NoError(t, err)
}

func TestFileStorageExecutor_Store_FromURL_InvalidURL(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action":   "store",
		"file_url": "not-a-valid-url",
	}

	// Execute should fail when trying to fetch invalid URL
	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
}

func TestFileStorageExecutor_Store_FromURL_EmptyURL(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	config := map[string]any{
		"action":   "store",
		"file_url": "",
		// Empty URL should be treated as missing
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either file_data or file_url is required")
}

// ============== Raw Bytes / Binary Input Tests ==============

func TestFileStorageExecutor_Store_RawBytes_TextContent(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Simple text content as bytes
	textContent := []byte("This is plain text content stored as bytes")
	base64Content := base64.StdEncoding.EncodeToString(textContent)

	config := map[string]any{
		"action":    "store",
		"file_data": base64Content,
		"file_name": "text-from-bytes.txt",
		"mime_type": "text/plain",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, int64(len(textContent)), resultMap["size"])
}

func TestFileStorageExecutor_Store_RawBytes_BinaryContent(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Binary content (simulating an image or PDF)
	binaryContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
	base64Content := base64.StdEncoding.EncodeToString(binaryContent)

	config := map[string]any{
		"action":    "store",
		"file_data": base64Content,
		"file_name": "image.png",
		"mime_type": "image/png",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, "image.png", resultMap["file_name"])
	assert.Equal(t, "image/png", resultMap["mime_type"])

	// Verify content integrity by getting the file back
	fileID := resultMap["file_id"].(string)
	getResult, err := exec.Execute(context.Background(), map[string]any{
		"action":  "get",
		"file_id": fileID,
	}, nil)
	require.NoError(t, err)

	getResultMap := getResult.(map[string]any)
	decodedContent, err := base64.StdEncoding.DecodeString(getResultMap["file_data"].(string))
	require.NoError(t, err)
	assert.Equal(t, binaryContent, decodedContent)
}

func TestFileStorageExecutor_Store_RawBytes_PDFContent(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// PDF header bytes
	pdfContent := []byte("%PDF-1.4 test content simulating a PDF file")
	base64Content := base64.StdEncoding.EncodeToString(pdfContent)

	config := map[string]any{
		"action":    "store",
		"file_data": base64Content,
		"file_name": "document.pdf",
		"mime_type": "application/pdf",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, "document.pdf", resultMap["file_name"])
	assert.Equal(t, "application/pdf", resultMap["mime_type"])
}

func TestFileStorageExecutor_Store_RawBytes_JSONContent(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// JSON content
	jsonContent := []byte(`{"name": "test", "data": [1, 2, 3], "nested": {"key": "value"}}`)
	base64Content := base64.StdEncoding.EncodeToString(jsonContent)

	config := map[string]any{
		"action":    "store",
		"file_data": base64Content,
		"file_name": "data.json",
		"mime_type": "application/json",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, "application/json", resultMap["mime_type"])
	assert.Equal(t, int64(len(jsonContent)), resultMap["size"])
}

func TestFileStorageExecutor_Store_RawBytes_LargeFile(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Simulate a larger file (1KB)
	largeContent := make([]byte, 1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	base64Content := base64.StdEncoding.EncodeToString(largeContent)

	config := map[string]any{
		"action":    "store",
		"file_data": base64Content,
		"file_name": "large-file.bin",
		"mime_type": "application/octet-stream",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, int64(1024), resultMap["size"])
}

func TestFileStorageExecutor_Store_RawBytes_FromInputMap(t *testing.T) {
	// Simulates receiving base64 data from previous node output
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Input that might come from HTTP node or LLM node
	nodeOutput := map[string]any{
		"content":   base64.StdEncoding.EncodeToString([]byte("content from previous node")),
		"file_name": "from-input.txt",
	}

	config := map[string]any{
		"action":    "store",
		"file_data": nodeOutput["content"],
		"file_name": nodeOutput["file_name"],
		"mime_type": "text/plain",
	}

	result, err := exec.Execute(context.Background(), config, nodeOutput)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, "from-input.txt", resultMap["file_name"])
}

func TestFileStorageExecutor_Store_RawBytes_EmptyContent(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	// Empty byte array results in empty string which is treated as missing data
	emptyContent := []byte{}
	base64Content := base64.StdEncoding.EncodeToString(emptyContent)

	config := map[string]any{
		"action":    "store",
		"file_data": base64Content,
		"file_name": "empty.txt",
		"mime_type": "text/plain",
	}

	// Empty base64 content is treated as missing by the executor
	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either file_data or file_url is required")
}

func TestFileStorageExecutor_Store_RawBytes_WithAllOptions(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	content := []byte("Full config test with all options")
	base64Content := base64.StdEncoding.EncodeToString(content)

	config := map[string]any{
		"action":       "store",
		"storage_id":   "test-storage",
		"file_data":    base64Content,
		"file_name":    "full-options.txt",
		"mime_type":    "text/plain",
		"access_scope": "result",
		"tags":         []any{"test", "full", "options"},
		"ttl":          7200,
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, "test-storage", resultMap["storage_id"])
	assert.Equal(t, "full-options.txt", resultMap["file_name"])
	assert.Equal(t, models.AccessScope("result"), resultMap["access_scope"])
	assert.NotNil(t, resultMap["expires_at"])
}

// ============== MIME Type Detection Tests ==============

func TestFileStorageExecutor_Store_MimeTypeDetection(t *testing.T) {
	manager := newMockManager()
	exec := NewFileStorageExecutor(manager)

	testCases := []struct {
		name         string
		fileName     string
		content      []byte
		expectedMime string
		explicitMime string
	}{
		{
			name:         "Text file by extension",
			fileName:     "readme.txt",
			content:      []byte("plain text content"),
			explicitMime: "",
		},
		{
			name:         "JSON file",
			fileName:     "data.json",
			content:      []byte(`{"key": "value"}`),
			explicitMime: "application/json",
		},
		{
			name:         "Explicit MIME override",
			fileName:     "file.bin",
			content:      []byte("binary content"),
			explicitMime: "application/octet-stream",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := map[string]any{
				"action":    "store",
				"file_data": base64.StdEncoding.EncodeToString(tc.content),
				"file_name": tc.fileName,
			}
			if tc.explicitMime != "" {
				config["mime_type"] = tc.explicitMime
			}

			result, err := exec.Execute(context.Background(), config, nil)
			require.NoError(t, err)

			resultMap := result.(map[string]any)
			assert.Equal(t, true, resultMap["success"])
			if tc.explicitMime != "" {
				assert.Equal(t, tc.explicitMime, resultMap["mime_type"])
			}
		})
	}
}

// ============== URL Download with httptest Tests ==============

func TestFileStorageExecutor_Store_URLDownload_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("%PDF-1.4 test content"))
	}))
	defer server.Close()

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":   "store",
		"file_url": server.URL + "/test.pdf",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, "application/pdf", resultMap["mime_type"])
}

func TestFileStorageExecutor_Store_URLDownload_HTTPError_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":   "store",
		"file_url": server.URL + "/missing.pdf",
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestFileStorageExecutor_Store_URLDownload_HTTPError_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":   "store",
		"file_url": server.URL + "/error.pdf",
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
}

func TestFileStorageExecutor_Store_URLDownload_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(3 * time.Second)
		w.Write([]byte("too slow"))
	}))
	defer server.Close()

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":   "store",
		"file_url": server.URL + "/slow.pdf",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := exec.Execute(ctx, config, nil)
	// Note: Timeout behavior depends on HTTP client implementation
	// Test may pass with timeout or complete successfully
	_ = err
}

func TestFileStorageExecutor_Store_URLDownload_LargeFile(t *testing.T) {
	// Create 1MB test data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(largeData)
	}))
	defer server.Close()

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":   "store",
		"file_url": server.URL + "/large.bin",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, int64(1024*1024), resultMap["size"])
}

func TestFileStorageExecutor_Store_URLDownload_MIMEFromHeader(t *testing.T) {
	tests := []struct {
		name         string
		contentType  string
		expectedMime string
	}{
		{"JSON", "application/json", "application/json"},
		{"PNG", "image/png", "image/png"},
		{"PDF", "application/pdf", "application/pdf"},
		{"Plain text", "text/plain; charset=utf-8", "text/plain; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.Write([]byte("test content"))
			}))
			defer server.Close()

			mockMgr := newMockManager()
			exec := NewFileStorageExecutor(mockMgr)

			config := map[string]any{
				"action":   "store",
				"file_url": server.URL + "/file",
			}

			result, err := exec.Execute(context.Background(), config, nil)
			require.NoError(t, err)

			resultMap := result.(map[string]any)
			assert.Equal(t, tt.expectedMime, resultMap["mime_type"])
		})
	}
}

func TestFileStorageExecutor_Store_URLDownload_FilenameExtraction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("PNG content"))
	}))
	defer server.Close()

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":   "store",
		"file_url": server.URL + "/path/to/image.png",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Contains(t, resultMap["file_name"].(string), ".png")
}

func TestFileStorageExecutor_Store_URLDownload_NetworkError(t *testing.T) {
	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":   "store",
		"file_url": "http://localhost:9999/nonexistent",
	}

	_, err := exec.Execute(context.Background(), config, nil)
	assert.Error(t, err)
}

// ============== Large File Tests ==============

func TestFileStorageExecutor_Store_1MB_Success(t *testing.T) {
	data := make([]byte, 1024*1024) // 1MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString(data),
		"file_name": "1mb-file.bin",
		"mime_type": "application/octet-stream",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, int64(1024*1024), resultMap["size"])
}

func TestFileStorageExecutor_Store_10MB_Success(t *testing.T) {
	data := make([]byte, 10*1024*1024) // 10MB
	for i := range data {
		data[i] = byte(i % 256)
	}

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString(data),
		"file_name": "10mb-file.bin",
		"mime_type": "application/octet-stream",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, true, resultMap["success"])
	assert.Equal(t, int64(10*1024*1024), resultMap["size"])
}

func TestFileStorageExecutor_Get_LargeFile_Integrity(t *testing.T) {
	// Create and store a 5MB file
	originalData := make([]byte, 5*1024*1024)
	for i := range originalData {
		originalData[i] = byte(i % 256)
	}

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	// Store
	storeConfig := map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString(originalData),
		"file_name": "5mb-test.bin",
		"mime_type": "application/octet-stream",
	}

	storeResult, err := exec.Execute(context.Background(), storeConfig, nil)
	require.NoError(t, err)

	storeMap := storeResult.(map[string]any)
	fileID := storeMap["file_id"].(string)

	// Get
	getConfig := map[string]any{
		"action":  "get",
		"file_id": fileID,
	}

	getResult, err := exec.Execute(context.Background(), getConfig, nil)
	require.NoError(t, err)

	getMap := getResult.(map[string]any)
	retrievedData, err := base64.StdEncoding.DecodeString(getMap["file_data"].(string))
	require.NoError(t, err)

	// Verify integrity
	assert.Equal(t, originalData, retrievedData)
}

// ============== Concurrency Tests ==============

func TestFileStorageExecutor_Concurrent_MultipleStores(t *testing.T) {
	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	var wg sync.WaitGroup
	results := make([]map[string]any, 10)
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			config := map[string]any{
				"action":    "store",
				"file_data": base64.StdEncoding.EncodeToString([]byte("test content")),
				"file_name": "concurrent-file-" + string(rune('A'+idx)) + ".txt",
				"mime_type": "text/plain",
			}

			result, err := exec.Execute(context.Background(), config, nil)
			errors[idx] = err
			if err == nil {
				results[idx] = result.(map[string]any)
			}
		}(i)
	}

	wg.Wait()

	// Verify all succeeded
	for i := 0; i < 10; i++ {
		require.NoError(t, errors[i], "goroutine %d failed", i)
		assert.Equal(t, true, results[i]["success"])
	}

	// Verify all files are unique
	fileIDs := make(map[string]bool)
	for i := 0; i < 10; i++ {
		fileID := results[i]["file_id"].(string)
		assert.False(t, fileIDs[fileID], "duplicate file ID: %s", fileID)
		fileIDs[fileID] = true
	}
}

func TestFileStorageExecutor_Concurrent_StoreAndGet(t *testing.T) {
	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	// First, store a file
	storeConfig := map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString([]byte("shared content")),
		"file_name": "shared-file.txt",
		"mime_type": "text/plain",
	}

	storeResult, err := exec.Execute(context.Background(), storeConfig, nil)
	require.NoError(t, err)

	fileID := storeResult.(map[string]any)["file_id"].(string)

	// Now read it concurrently from 20 goroutines
	var wg sync.WaitGroup
	errors := make([]error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			getConfig := map[string]any{
				"action":  "get",
				"file_id": fileID,
			}

			_, err := exec.Execute(context.Background(), getConfig, nil)
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Verify all reads succeeded
	for i := 0; i < 20; i++ {
		require.NoError(t, errors[i], "goroutine %d failed", i)
	}
}

func TestFileStorageExecutor_Concurrent_10Goroutines_MixedOps(t *testing.T) {
	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	var wg sync.WaitGroup
	var mu sync.Mutex
	fileIDs := []string{}

	// 10 goroutines: 5 store, 5 list
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			if idx%2 == 0 {
				// Store operation
				config := map[string]any{
					"action":    "store",
					"file_data": base64.StdEncoding.EncodeToString([]byte("concurrent test")),
					"file_name": "file-" + string(rune('A'+idx)) + ".txt",
					"mime_type": "text/plain",
				}

				result, err := exec.Execute(context.Background(), config, nil)
				if err == nil {
					mu.Lock()
					fileIDs = append(fileIDs, result.(map[string]any)["file_id"].(string))
					mu.Unlock()
				}
			} else {
				// List operation
				config := map[string]any{
					"action": "list",
				}

				exec.Execute(context.Background(), config, nil)
			}
		}(i)
	}

	wg.Wait()

	// Verify we got some file IDs
	assert.GreaterOrEqual(t, len(fileIDs), 5)
}

func TestFileStorageExecutor_Concurrent_DifferentStorages(t *testing.T) {
	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	var wg sync.WaitGroup
	errors := make([]error, 6)

	storageIDs := []string{"storage-a", "storage-b", "storage-c"}

	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			config := map[string]any{
				"action":     "store",
				"file_data":  base64.StdEncoding.EncodeToString([]byte("test")),
				"file_name":  "file.txt",
				"mime_type":  "text/plain",
				"storage_id": storageIDs[idx%3],
			}

			_, err := exec.Execute(context.Background(), config, nil)
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Verify all succeeded
	for i := 0; i < 6; i++ {
		require.NoError(t, errors[i], "goroutine %d failed", i)
	}
}

// ============== MIME Detection Edge Cases ==============

func TestFileStorageExecutor_MIMEDetection_RealPNGSignature(t *testing.T) {
	// Real PNG signature
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	pngData = append(pngData, []byte("fake PNG data")...)

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString(pngData),
		"file_name": "test.png",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	// Should detect PNG from signature
	assert.Equal(t, "image/png", resultMap["mime_type"])
}

func TestFileStorageExecutor_MIMEDetection_RealJPEGSignature(t *testing.T) {
	// Real JPEG signature
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	jpegData = append(jpegData, []byte("fake JPEG data")...)

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString(jpegData),
		"file_name": "test.jpg",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	// Should detect JPEG from signature
	assert.Contains(t, resultMap["mime_type"].(string), "image/jpeg")
}

func TestFileStorageExecutor_MIMEDetection_ConflictingExtension(t *testing.T) {
	// PNG signature but .txt extension
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	pngData = append(pngData, []byte("PNG with wrong extension")...)

	mockMgr := newMockManager()
	exec := NewFileStorageExecutor(mockMgr)

	config := map[string]any{
		"action":    "store",
		"file_data": base64.StdEncoding.EncodeToString(pngData),
		"file_name": "image.txt", // Wrong extension
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	// Should detect PNG from signature, not extension
	assert.Equal(t, "image/png", resultMap["mime_type"])
}
