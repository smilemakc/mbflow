package filestorage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock provider for testing
type mockProvider struct {
	storeFn  func(context.Context, *models.FileEntry, io.Reader) (string, error)
	getFn    func(context.Context, string) (io.ReadCloser, error)
	deleteFn func(context.Context, string) error
	existsFn func(context.Context, string) (bool, error)
	usageFn  func(context.Context) (*models.StorageUsage, error)
	closeFn  func() error
	typeFn   func() models.StorageType
	mu       sync.Mutex
	closed   bool
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		typeFn: func() models.StorageType {
			return models.StorageTypeLocal
		},
	}
}

func (m *mockProvider) Type() models.StorageType {
	if m.typeFn != nil {
		return m.typeFn()
	}
	return models.StorageTypeLocal
}

func (m *mockProvider) Store(ctx context.Context, entry *models.FileEntry, reader io.Reader) (string, error) {
	if m.storeFn != nil {
		return m.storeFn(ctx, entry, reader)
	}
	entry.Path = "mock/path/" + entry.Name
	entry.Size = 100
	entry.Checksum = "mock-checksum"
	return entry.Path, nil
}

func (m *mockProvider) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	if m.getFn != nil {
		return m.getFn(ctx, path)
	}
	return io.NopCloser(bytes.NewReader([]byte("mock content"))), nil
}

func (m *mockProvider) Delete(ctx context.Context, path string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, path)
	}
	return nil
}

func (m *mockProvider) Exists(ctx context.Context, path string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, path)
	}
	return true, nil
}

func (m *mockProvider) GetUsage(ctx context.Context) (*models.StorageUsage, error) {
	if m.usageFn != nil {
		return m.usageFn(ctx)
	}
	return &models.StorageUsage{
		TotalSize: 1000,
		FileCount: 10,
	}, nil
}

func (m *mockProvider) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

func (m *mockProvider) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// Mock factory
type mockFactory struct {
	createFn func(*models.StorageConfig) (Provider, error)
	typeFn   func() models.StorageType
}

func newMockFactory() *mockFactory {
	return &mockFactory{
		typeFn: func() models.StorageType {
			return models.StorageTypeLocal
		},
		createFn: func(config *models.StorageConfig) (Provider, error) {
			return newMockProvider(), nil
		},
	}
}

func (f *mockFactory) Type() models.StorageType {
	if f.typeFn != nil {
		return f.typeFn()
	}
	return models.StorageTypeLocal
}

func (f *mockFactory) Create(config *models.StorageConfig) (Provider, error) {
	if f.createFn != nil {
		return f.createFn(config)
	}
	return newMockProvider(), nil
}

// ============== A. Initialization Tests ==============

func TestStorageManager_New_DefaultConfig(t *testing.T) {
	manager := NewStorageManager(nil, nil)

	require.NotNil(t, manager)
	assert.NotNil(t, manager.config)
	assert.Equal(t, "./file_storage", manager.config.BasePath)
	assert.Equal(t, int64(100*1024*1024), manager.config.MaxFileSize)
	assert.Equal(t, int64(0), manager.config.MaxStorageSize)
	assert.NotNil(t, manager.validator)

	manager.Close()
}

func TestStorageManager_New_CustomConfig(t *testing.T) {
	config := &ManagerConfig{
		BasePath:        "/custom/path",
		MaxFileSize:     50 * 1024 * 1024,
		MaxStorageSize:  1024 * 1024 * 1024,
		DefaultTTL:      time.Hour,
		CleanupInterval: 30 * time.Minute,
	}

	manager := NewStorageManager(config, nil)

	require.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
	assert.Equal(t, "/custom/path", manager.config.BasePath)
	assert.Equal(t, int64(50*1024*1024), manager.config.MaxFileSize)
	assert.Equal(t, time.Hour, manager.config.DefaultTTL)

	manager.Close()
}

func TestStorageManager_RegisterFactory_Success(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	factory := newMockFactory()
	manager.RegisterFactory(factory)

	// Verify factory was registered
	assert.Contains(t, manager.factories, factory.Type())
}

func TestStorageManager_RegisterFactory_Duplicate(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	factory1 := newMockFactory()
	factory2 := newMockFactory()

	manager.RegisterFactory(factory1)
	manager.RegisterFactory(factory2) // Should overwrite

	// Both should work, second should overwrite first
	assert.Contains(t, manager.factories, factory1.Type())
}

func TestStorageManager_GetValidator_NotNil(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	validator := manager.GetValidator()

	assert.NotNil(t, validator)
	assert.IsType(t, &MimeValidator{}, validator)
}

// ============== B. Storage Lifecycle Tests ==============

func TestStorageManager_CreateStorage_Local_Success(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	// Register mock factory
	manager.RegisterFactory(newMockFactory())

	config := &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "/test/path",
	}

	storage, err := manager.CreateStorage("test-storage", config)

	require.NoError(t, err)
	require.NotNil(t, storage)
	assert.True(t, manager.HasStorage("test-storage"))
}

func TestStorageManager_CreateStorage_AlreadyExists(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	config := &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "/test/path",
	}

	// Create first time
	storage1, err := manager.CreateStorage("test-storage", config)
	require.NoError(t, err)

	// Create second time - should return existing
	storage2, err := manager.CreateStorage("test-storage", config)
	require.NoError(t, err)
	assert.Equal(t, storage1, storage2)
}

func TestStorageManager_CreateStorage_InvalidType(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	config := &models.StorageConfig{
		Type:     "invalid-type",
		BasePath: "/test/path",
	}

	storage, err := manager.CreateStorage("test-storage", config)

	assert.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "no factory registered")
}

func TestStorageManager_CreateStorage_FactoryError(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	// Register factory that returns error
	factory := newMockFactory()
	factory.createFn = func(config *models.StorageConfig) (Provider, error) {
		return nil, errors.New("factory creation error")
	}
	manager.RegisterFactory(factory)

	config := &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "/test/path",
	}

	storage, err := manager.CreateStorage("test-storage", config)

	assert.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "failed to create provider")
}

func TestStorageManager_GetStorage_Exists(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	// Create storage
	config := &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "/test/path",
	}
	created, err := manager.CreateStorage("test-storage", config)
	require.NoError(t, err)

	// Get existing storage
	storage, err := manager.GetStorage("test-storage")

	require.NoError(t, err)
	assert.Equal(t, created, storage)
}

func TestStorageManager_GetStorage_CreateOnDemand(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	// Get non-existent storage - should create with default config
	storage, err := manager.GetStorage("auto-created")

	require.NoError(t, err)
	require.NotNil(t, storage)
	assert.True(t, manager.HasStorage("auto-created"))
}

func TestStorageManager_GetStorage_Concurrent(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	var wg sync.WaitGroup
	goroutineCount := 20
	results := make([]Storage, goroutineCount)

	// Concurrent GetStorage calls
	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			storage, err := manager.GetStorage("concurrent-storage")
			assert.NoError(t, err)
			results[idx] = storage
		}(i)
	}

	wg.Wait()

	// All should get the same storage instance
	for i := 1; i < goroutineCount; i++ {
		assert.Equal(t, results[0], results[i])
	}
}

func TestStorageManager_DeleteStorage_Success(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	mockProv := newMockProvider()
	factory := newMockFactory()
	factory.createFn = func(config *models.StorageConfig) (Provider, error) {
		return mockProv, nil
	}
	manager.RegisterFactory(factory)

	// Create storage
	_, err := manager.CreateStorage("test-storage", &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "/test",
	})
	require.NoError(t, err)

	// Delete storage
	err = manager.DeleteStorage("test-storage")

	require.NoError(t, err)
	assert.False(t, manager.HasStorage("test-storage"))
	assert.True(t, mockProv.isClosed(), "Provider should be closed")
}

func TestStorageManager_DeleteStorage_NotFound(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	err := manager.DeleteStorage("non-existent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage not found")
}

func TestStorageManager_DeleteStorage_ProviderCloseError(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	mockProv := newMockProvider()
	mockProv.closeFn = func() error {
		return errors.New("close error")
	}

	factory := newMockFactory()
	factory.createFn = func(config *models.StorageConfig) (Provider, error) {
		return mockProv, nil
	}
	manager.RegisterFactory(factory)

	_, err := manager.CreateStorage("test-storage", &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "/test",
	})
	require.NoError(t, err)

	// Delete should return close error
	err = manager.DeleteStorage("test-storage")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to close provider")
}

func TestStorageManager_ListStorages_Empty(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	storages := manager.ListStorages()

	assert.Empty(t, storages)
}

func TestStorageManager_ListStorages_Multiple(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	// Create multiple storages
	storageIDs := []string{"storage-1", "storage-2", "storage-3"}
	for _, id := range storageIDs {
		_, err := manager.CreateStorage(id, &models.StorageConfig{
			Type:     models.StorageTypeLocal,
			BasePath: "/test",
		})
		require.NoError(t, err)
	}

	list := manager.ListStorages()

	assert.Len(t, list, 3)
	for _, id := range storageIDs {
		assert.Contains(t, list, id)
	}
}

func TestStorageManager_HasStorage_True(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	_, err := manager.CreateStorage("test-storage", &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "/test",
	})
	require.NoError(t, err)

	assert.True(t, manager.HasStorage("test-storage"))
	assert.False(t, manager.HasStorage("non-existent"))
}

func TestStorageManager_GetDefaultStorage_Success(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	storage, err := manager.GetDefaultStorage()

	require.NoError(t, err)
	require.NotNil(t, storage)
	assert.True(t, manager.HasStorage("default"))
}

// ============== C. Observer Management Tests ==============

func TestStorageManager_RegisterObserver_Success(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	observer := newTestObserver("test-observer", nil)

	err := manager.RegisterObserver(observer)

	assert.NoError(t, err)
	assert.Contains(t, manager.observers, "test-observer")
}

func TestStorageManager_RegisterObserver_Duplicate(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	observer := newTestObserver("test-observer", nil)

	err := manager.RegisterObserver(observer)
	require.NoError(t, err)

	// Register again - should fail
	err = manager.RegisterObserver(observer)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "observer already registered")
}

func TestStorageManager_UnregisterObserver_Success(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	observer := newTestObserver("test-observer", nil)
	err := manager.RegisterObserver(observer)
	require.NoError(t, err)

	// Unregister
	err = manager.UnregisterObserver("test-observer")

	assert.NoError(t, err)
	assert.NotContains(t, manager.observers, "test-observer")
}

func TestStorageManager_UnregisterObserver_NotFound(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	err := manager.UnregisterObserver("non-existent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "observer not found")
}

func TestStorageManager_NotifyObservers_FileAdded(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	observer := newTestObserver("test-observer", nil)
	err := manager.RegisterObserver(observer)
	require.NoError(t, err)

	event := NewFileEvent(EventFileAdded, "storage-1", &models.FileEntry{ID: "file-1"})
	manager.notifyObservers(context.Background(), event)

	// Give goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	events := observer.getEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, EventFileAdded, events[0].Type)
}

func TestStorageManager_NotifyObservers_WithFilter(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	// Observer that only receives EventFileAdded
	filter := NewEventTypeFilter(EventFileAdded)
	observer := newTestObserver("filtered-observer", filter)
	err := manager.RegisterObserver(observer)
	require.NoError(t, err)

	// Send matching event
	event1 := NewFileEvent(EventFileAdded, "storage-1", nil)
	manager.notifyObservers(context.Background(), event1)

	// Send non-matching event
	event2 := NewFileEvent(EventFileRemoved, "storage-1", nil)
	manager.notifyObservers(context.Background(), event2)

	time.Sleep(100 * time.Millisecond)

	events := observer.getEvents()
	assert.Len(t, events, 1) // Only EventFileAdded should be received
	assert.Equal(t, EventFileAdded, events[0].Type)
}

func TestStorageManager_NotifyObservers_Concurrent(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	observer := newTestObserver("test-observer", nil)
	err := manager.RegisterObserver(observer)
	require.NoError(t, err)

	// Concurrent notifications
	var wg sync.WaitGroup
	eventCount := 50

	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			event := NewFileEvent(EventFileAdded, "storage-1", &models.FileEntry{
				ID: "file-" + string(rune(idx)),
			})
			manager.notifyObservers(context.Background(), event)
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// All events should be received
	assert.Equal(t, eventCount, observer.getCallCount())
}

func TestStorageManager_NotifyObservers_Async(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	var notified atomic.Bool
	observer := NewFuncObserver("test", nil, func(ctx context.Context, event *FileEvent) error {
		time.Sleep(50 * time.Millisecond) // Simulate slow observer
		notified.Store(true)
		return nil
	})
	err := manager.RegisterObserver(observer)
	require.NoError(t, err)

	event := NewFileEvent(EventFileAdded, "storage-1", nil)

	// notifyObservers should not block
	start := time.Now()
	manager.notifyObservers(context.Background(), event)
	duration := time.Since(start)

	// Should return immediately (not wait for observer)
	assert.Less(t, duration, 10*time.Millisecond)

	// Wait for observer to complete
	time.Sleep(100 * time.Millisecond)
	assert.True(t, notified.Load())
}

// ============== D. Storage Wrapper Tests ==============

func TestStorageWrapper_Store_MIMEValidation(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())
	storage, err := manager.GetStorage("test-storage")
	require.NoError(t, err)

	// Valid MIME type
	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "test-storage",
		Name:      "test.txt",
		MimeType:  "text/plain",
	}

	_, err = storage.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
	assert.NoError(t, err)

	// Invalid MIME type
	entry2 := &models.FileEntry{
		ID:        "file-2",
		StorageID: "test-storage",
		Name:      "test.exe",
		MimeType:  "application/x-msdownload",
	}

	_, err = storage.Store(context.Background(), entry2, bytes.NewReader([]byte("test")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MIME type not allowed")
}

func TestStorageWrapper_Store_FileSizeExceeded(t *testing.T) {
	config := &ManagerConfig{
		BasePath:    t.TempDir(),
		MaxFileSize: 100, // 100 bytes limit
	}
	manager := NewStorageManager(config, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())
	storage, err := manager.GetStorage("test-storage")
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "test-storage",
		Name:      "large.txt",
		MimeType:  "text/plain",
		Size:      200, // Exceeds limit
	}

	_, err = storage.Store(context.Background(), entry, bytes.NewReader(make([]byte, 200)))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestStorageWrapper_Store_GenerateID(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())
	storage, err := manager.GetStorage("test-storage")
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "", // Empty ID should be generated
		StorageID: "test-storage",
		Name:      "test.txt",
		MimeType:  "text/plain",
	}

	_, err = storage.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
	assert.NoError(t, err)
	assert.NotEmpty(t, entry.ID)
}

func TestStorageWrapper_Store_SetTimestamps(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())
	storage, err := manager.GetStorage("test-storage")
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "test-storage",
		Name:      "test.txt",
		MimeType:  "text/plain",
	}

	before := time.Now()
	_, err = storage.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
	after := time.Now()

	assert.NoError(t, err)
	assert.True(t, entry.CreatedAt.After(before) || entry.CreatedAt.Equal(before))
	assert.True(t, entry.CreatedAt.Before(after) || entry.CreatedAt.Equal(after))
	assert.Equal(t, entry.CreatedAt, entry.UpdatedAt)
}

func TestStorageWrapper_Store_ApplyDefaultTTL(t *testing.T) {
	config := &ManagerConfig{
		BasePath:   t.TempDir(),
		DefaultTTL: time.Hour,
	}
	manager := NewStorageManager(config, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	// Create storage with default TTL
	storageConfig := &models.StorageConfig{
		Type:       models.StorageTypeLocal,
		BasePath:   "/test",
		DefaultTTL: new(time.Duration),
	}
	*storageConfig.DefaultTTL = time.Hour

	storage, err := manager.CreateStorage("test-storage", storageConfig)
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "test-storage",
		Name:      "test.txt",
		MimeType:  "text/plain",
	}

	_, err = storage.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
	assert.NoError(t, err)

	// Should have ExpiresAt set
	assert.NotNil(t, entry.ExpiresAt)
}

func TestStorageWrapper_Store_ObserverNotification(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	// Use EventTypeFilter to only receive EventFileAdded events
	filter := NewEventTypeFilter(EventFileAdded)
	observer := newTestObserver("test-observer", filter)
	err := manager.RegisterObserver(observer)
	require.NoError(t, err)

	manager.RegisterFactory(newMockFactory())
	storage, err := manager.GetStorage("test-storage")
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "test-storage",
		Name:      "test.txt",
		MimeType:  "text/plain",
	}

	_, err = storage.Store(context.Background(), entry, bytes.NewReader([]byte("test")))
	assert.NoError(t, err)

	// Wait for async notification
	time.Sleep(100 * time.Millisecond)

	events := observer.getEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, EventFileAdded, events[0].Type)
}

func TestStorageWrapper_Store_QuotaExceeded_Event(t *testing.T) {
	config := &ManagerConfig{
		BasePath:    t.TempDir(),
		MaxFileSize: 50,
	}
	manager := NewStorageManager(config, nil)
	defer manager.Close()

	// Use EventTypeFilter to only receive QuotaExceeded events
	filter := NewEventTypeFilter(EventQuotaExceeded)
	observer := newTestObserver("test-observer", filter)
	err := manager.RegisterObserver(observer)
	require.NoError(t, err)

	manager.RegisterFactory(newMockFactory())
	storage, err := manager.GetStorage("test-storage")
	require.NoError(t, err)

	entry := &models.FileEntry{
		ID:        "file-1",
		StorageID: "test-storage",
		Name:      "large.txt",
		MimeType:  "text/plain",
		Size:      100,
	}

	_, err = storage.Store(context.Background(), entry, bytes.NewReader(make([]byte, 100)))
	assert.Error(t, err)

	// Should have QuotaExceeded event
	time.Sleep(100 * time.Millisecond)
	events := observer.getEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, EventQuotaExceeded, events[0].Type)
}

func TestStorageWrapper_GetUsage_Success(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())
	storage, err := manager.GetStorage("test-storage")
	require.NoError(t, err)

	usage, err := storage.GetUsage(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, usage)
	assert.Equal(t, "test-storage", usage.StorageID)
}

func TestStorageWrapper_GetUsage_MaxSizeCalculation(t *testing.T) {
	config := &ManagerConfig{
		BasePath:       t.TempDir(),
		MaxStorageSize: 10000,
	}
	manager := NewStorageManager(config, nil)
	defer manager.Close()

	mockProv := newMockProvider()
	mockProv.usageFn = func(ctx context.Context) (*models.StorageUsage, error) {
		return &models.StorageUsage{
			TotalSize: 5000,
			FileCount: 10,
		}, nil
	}

	factory := newMockFactory()
	factory.createFn = func(cfg *models.StorageConfig) (Provider, error) {
		return mockProv, nil
	}
	manager.RegisterFactory(factory)

	storageConfig := &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "/test",
		MaxSize:  10000,
	}

	storage, err := manager.CreateStorage("test-storage", storageConfig)
	require.NoError(t, err)

	usage, err := storage.GetUsage(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, int64(10000), usage.MaxSize)
	assert.Equal(t, float64(50), usage.UsagePercent) // 5000/10000 = 50%
}

func TestStorageWrapper_GetUsage_UsagePercent(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	mockProv := newMockProvider()
	mockProv.usageFn = func(ctx context.Context) (*models.StorageUsage, error) {
		return &models.StorageUsage{
			TotalSize: 7500,
			FileCount: 100,
		}, nil
	}

	factory := newMockFactory()
	factory.createFn = func(cfg *models.StorageConfig) (Provider, error) {
		return mockProv, nil
	}
	manager.RegisterFactory(factory)

	storageConfig := &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: "/test",
		MaxSize:  10000,
	}

	storage, err := manager.CreateStorage("test-storage", storageConfig)
	require.NoError(t, err)

	usage, err := storage.GetUsage(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, int64(7500), usage.TotalSize)
	assert.Equal(t, int64(10000), usage.MaxSize)
	assert.Equal(t, float64(75), usage.UsagePercent) // 7500/10000 = 75%
}

// ============== E. Cleanup & Close Tests ==============

func TestStorageManager_CleanupRoutine_Starts(t *testing.T) {
	config := &ManagerConfig{
		BasePath:        t.TempDir(),
		CleanupInterval: 100 * time.Millisecond,
	}
	manager := NewStorageManager(config, nil)
	defer manager.Close()

	// Cleanup routine should start automatically
	// Just verify manager was created successfully
	assert.NotNil(t, manager)
}

func TestStorageManager_Close_StopsCleanup(t *testing.T) {
	config := &ManagerConfig{
		BasePath:        t.TempDir(),
		CleanupInterval: 10 * time.Millisecond,
	}
	manager := NewStorageManager(config, nil)

	// Close should stop cleanup routine
	err := manager.Close()
	assert.NoError(t, err)

	// Give time for cleanup routine to exit
	time.Sleep(50 * time.Millisecond)
}

func TestStorageManager_Close_ClosesAllStorages(t *testing.T) {
	manager := NewStorageManager(nil, nil)

	providers := make([]*mockProvider, 3)
	for i := range providers {
		providers[i] = newMockProvider()
	}

	factory := newMockFactory()
	idx := 0
	factory.createFn = func(cfg *models.StorageConfig) (Provider, error) {
		prov := providers[idx]
		idx++
		return prov, nil
	}
	manager.RegisterFactory(factory)

	// Create multiple storages
	for i := 0; i < 3; i++ {
		_, err := manager.CreateStorage(string(rune('a'+i)), &models.StorageConfig{
			Type:     models.StorageTypeLocal,
			BasePath: "/test",
		})
		require.NoError(t, err)
	}

	// Close manager
	err := manager.Close()
	assert.NoError(t, err)

	// All providers should be closed
	for i, prov := range providers {
		assert.True(t, prov.isClosed(), "Provider %d should be closed", i)
	}
}

// ============== F. Concurrency Tests ==============

func TestStorageManager_Concurrent_CreateMultipleStorages(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	var wg sync.WaitGroup
	storageCount := 20

	for i := 0; i < storageCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := "storage-" + string(rune('a'+idx))
			_, err := manager.CreateStorage(id, &models.StorageConfig{
				Type:     models.StorageTypeLocal,
				BasePath: "/test",
			})
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// All storages should be created
	list := manager.ListStorages()
	assert.Len(t, list, storageCount)
}

func TestStorageManager_Concurrent_ObserverRegistration(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	var wg sync.WaitGroup
	observerCount := 20

	for i := 0; i < observerCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			observer := newTestObserver("observer-"+string(rune('a'+idx)), nil)
			err := manager.RegisterObserver(observer)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	assert.Len(t, manager.observers, observerCount)
}

func TestStorageManager_Concurrent_MixedOperations(t *testing.T) {
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	var wg sync.WaitGroup

	// Concurrent creates
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = manager.CreateStorage("storage-"+string(rune('a'+idx)), &models.StorageConfig{
				Type:     models.StorageTypeLocal,
				BasePath: "/test",
			})
		}(i)
	}

	// Concurrent gets
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = manager.GetStorage("storage-" + string(rune('a'+idx%5)))
		}(i)
	}

	// Concurrent observer registrations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			observer := newTestObserver("obs-"+string(rune('a'+idx)), nil)
			_ = manager.RegisterObserver(observer)
		}(i)
	}

	wg.Wait()
}

func TestStorageManager_RaceDetection(t *testing.T) {
	// This test is designed to be run with -race flag
	manager := NewStorageManager(nil, nil)
	defer manager.Close()

	manager.RegisterFactory(newMockFactory())

	var wg sync.WaitGroup

	// Multiple concurrent operations that might race
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			switch idx % 5 {
			case 0:
				manager.CreateStorage("storage", &models.StorageConfig{Type: models.StorageTypeLocal, BasePath: "/test"})
			case 1:
				manager.GetStorage("storage")
			case 2:
				manager.HasStorage("storage")
			case 3:
				manager.ListStorages()
			case 4:
				obs := newTestObserver("obs-"+string(rune(idx)), nil)
				manager.RegisterObserver(obs)
			}
		}(i)
	}

	wg.Wait()
}
