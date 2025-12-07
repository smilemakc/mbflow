package filestorage

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/pkg/models"
)

// DefaultManagerConfig returns default manager configuration
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		BasePath:        "./file_storage",
		MaxFileSize:     100 * 1024 * 1024, // 100MB
		MaxStorageSize:  0,                 // unlimited
		CleanupInterval: 1 * time.Hour,
	}
}

// ManagerConfig holds manager configuration
type ManagerConfig struct {
	BasePath        string        // Base path for all storages
	MaxFileSize     int64         // Maximum file size in bytes
	MaxStorageSize  int64         // Maximum storage size (0 = unlimited)
	DefaultTTL      time.Duration // Default TTL for files (0 = no expiration)
	CleanupInterval time.Duration // Interval for cleanup routine
}

// StorageManager manages multiple storages and observers
type StorageManager struct {
	config      *ManagerConfig
	storages    map[string]*managedStorage
	factories   map[models.StorageType]ProviderFactory
	observers   map[string]FileObserver
	validator   *MimeValidator
	mu          sync.RWMutex
	cleanupDone chan struct{}
}

// managedStorage wraps a storage with its provider and config
type managedStorage struct {
	provider  Provider
	config    *models.StorageConfig
	storageID string
}

// NewStorageManager creates a new storage manager
func NewStorageManager(config *ManagerConfig) *StorageManager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	m := &StorageManager{
		config:      config,
		storages:    make(map[string]*managedStorage),
		factories:   make(map[models.StorageType]ProviderFactory),
		observers:   make(map[string]FileObserver),
		validator:   NewMimeValidator(),
		cleanupDone: make(chan struct{}),
	}

	// Register default factories
	m.RegisterFactory(NewLocalProviderFactory())

	// Start cleanup routine if interval is set
	if config.CleanupInterval > 0 {
		go m.cleanupRoutine()
	}

	return m
}

// RegisterFactory registers a provider factory
func (m *StorageManager) RegisterFactory(factory ProviderFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.factories[factory.Type()] = factory
}

// GetStorage returns a storage by ID, creating if necessary with default config
func (m *StorageManager) GetStorage(storageID string) (Storage, error) {
	m.mu.RLock()
	storage, exists := m.storages[storageID]
	m.mu.RUnlock()

	if exists {
		return m.wrapStorage(storage), nil
	}

	// Create with default config
	return m.CreateStorage(storageID, &models.StorageConfig{
		Type:     models.StorageTypeLocal,
		BasePath: m.config.BasePath,
	})
}

// CreateStorage creates a new storage instance
func (m *StorageManager) CreateStorage(storageID string, config *models.StorageConfig) (Storage, error) {
	m.mu.Lock()

	// Check if already exists
	if _, exists := m.storages[storageID]; exists {
		wrapped := m.wrapStorage(m.storages[storageID])
		m.mu.Unlock()
		return wrapped, nil
	}

	// Get factory for storage type
	factory, ok := m.factories[config.Type]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("no factory registered for storage type: %s", config.Type)
	}

	// Update base path to include storage ID
	config.BasePath = fmt.Sprintf("%s/%s", m.config.BasePath, storageID)

	// Create provider
	provider, err := factory.Create(config)
	if err != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	managed := &managedStorage{
		provider:  provider,
		config:    config,
		storageID: storageID,
	}
	m.storages[storageID] = managed
	wrapped := m.wrapStorage(managed)

	// Unlock before notifying observers to avoid deadlock
	m.mu.Unlock()

	// Notify observers
	m.notifyObservers(context.Background(), NewFileEvent(EventStorageCreated, storageID, nil))

	return wrapped, nil
}

// wrapStorage creates a Storage wrapper with observer notifications
func (m *StorageManager) wrapStorage(managed *managedStorage) Storage {
	return &storageWrapper{
		manager:  m,
		storage:  managed,
		provider: managed.provider,
	}
}

// DeleteStorage deletes a storage and all its files
func (m *StorageManager) DeleteStorage(storageID string) error {
	m.mu.Lock()

	storage, exists := m.storages[storageID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("storage not found: %s", storageID)
	}

	// Close provider
	if err := storage.provider.Close(); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to close provider: %w", err)
	}

	delete(m.storages, storageID)

	// Unlock before notifying observers to avoid deadlock
	m.mu.Unlock()

	// Notify observers
	m.notifyObservers(context.Background(), NewFileEvent(EventStorageDeleted, storageID, nil))

	return nil
}

// ListStorages returns all storage IDs
func (m *StorageManager) ListStorages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.storages))
	for id := range m.storages {
		ids = append(ids, id)
	}
	return ids
}

// HasStorage checks if a storage exists
func (m *StorageManager) HasStorage(storageID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.storages[storageID]
	return exists
}

// GetDefaultStorage returns the default storage
func (m *StorageManager) GetDefaultStorage() (Storage, error) {
	return m.GetStorage("default")
}

// RegisterObserver registers a file event observer
func (m *StorageManager) RegisterObserver(observer FileObserver) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.observers[observer.Name()]; exists {
		return fmt.Errorf("observer already registered: %s", observer.Name())
	}

	m.observers[observer.Name()] = observer
	return nil
}

// UnregisterObserver removes an observer
func (m *StorageManager) UnregisterObserver(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.observers[name]; !exists {
		return fmt.Errorf("observer not found: %s", name)
	}

	delete(m.observers, name)
	return nil
}

// notifyObservers sends an event to all matching observers
func (m *StorageManager) notifyObservers(ctx context.Context, event *FileEvent) {
	m.mu.RLock()
	observers := make([]FileObserver, 0, len(m.observers))
	for _, obs := range m.observers {
		observers = append(observers, obs)
	}
	m.mu.RUnlock()

	for _, obs := range observers {
		filter := obs.Filter()
		if filter == nil || filter.ShouldNotify(event) {
			// Call observer in goroutine to avoid blocking
			go func(o FileObserver) {
				_ = o.OnFileEvent(ctx, event) // Ignore errors for now
			}(obs)
		}
	}
}

// Cleanup removes expired files from all storages
func (m *StorageManager) Cleanup(ctx context.Context) (int, error) {
	// TODO: Implement cleanup with repository integration
	return 0, nil
}

// cleanupRoutine runs periodic cleanup
func (m *StorageManager) cleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.Cleanup(context.Background())
		case <-m.cleanupDone:
			return
		}
	}
}

// Close closes the manager and all storages
func (m *StorageManager) Close() error {
	close(m.cleanupDone)

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, storage := range m.storages {
		storage.provider.Close()
	}

	return nil
}

// GetValidator returns the MIME validator
func (m *StorageManager) GetValidator() *MimeValidator {
	return m.validator
}

// storageWrapper wraps provider operations with observer notifications
type storageWrapper struct {
	manager  *StorageManager
	storage  *managedStorage
	provider Provider
}

// Store stores a file
func (s *storageWrapper) Store(ctx context.Context, entry *models.FileEntry, reader io.Reader) (*models.FileEntry, error) {
	// Validate MIME type
	if err := s.manager.validator.Validate(entry.MimeType); err != nil {
		return nil, err
	}

	// Validate file size (if known)
	if entry.Size > 0 && s.manager.config.MaxFileSize > 0 && entry.Size > s.manager.config.MaxFileSize {
		event := NewFileEvent(EventQuotaExceeded, s.storage.storageID, entry).
			WithMetadata("reason", "file_size_exceeded")
		s.manager.notifyObservers(ctx, event)
		return nil, fmt.Errorf("file size %d exceeds maximum %d", entry.Size, s.manager.config.MaxFileSize)
	}

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Set storage ID
	entry.StorageID = s.storage.storageID

	// Set timestamps
	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now

	// Apply default TTL
	if s.storage.config.DefaultTTL != nil && *s.storage.config.DefaultTTL > 0 && entry.ExpiresAt == nil {
		entry.SetTTL(*s.storage.config.DefaultTTL)
	}

	// Store file
	path, err := s.provider.Store(ctx, entry, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	entry.Path = path

	// Notify observers
	s.manager.notifyObservers(ctx, NewFileEvent(EventFileAdded, s.storage.storageID, entry))

	return entry, nil
}

// Get retrieves a file
func (s *storageWrapper) Get(ctx context.Context, fileID string) (*models.FileEntry, io.ReadCloser, error) {
	// TODO: Get metadata from repository
	// For now, this requires repository integration
	return nil, nil, fmt.Errorf("not implemented: requires repository integration")
}

// Delete removes a file
func (s *storageWrapper) Delete(ctx context.Context, fileID string) error {
	// TODO: Implement with repository integration
	return fmt.Errorf("not implemented: requires repository integration")
}

// List lists files
func (s *storageWrapper) List(ctx context.Context, query *FileQuery) ([]*models.FileEntry, error) {
	// TODO: Implement with repository integration
	return nil, fmt.Errorf("not implemented: requires repository integration")
}

// Exists checks if a file exists
func (s *storageWrapper) Exists(ctx context.Context, fileID string) (bool, error) {
	// TODO: Implement with repository integration
	return false, fmt.Errorf("not implemented: requires repository integration")
}

// GetMetadata retrieves file metadata
func (s *storageWrapper) GetMetadata(ctx context.Context, fileID string) (*models.FileEntry, error) {
	// TODO: Implement with repository integration
	return nil, fmt.Errorf("not implemented: requires repository integration")
}

// UpdateMetadata updates file metadata
func (s *storageWrapper) UpdateMetadata(ctx context.Context, fileID string, metadata map[string]interface{}) error {
	// TODO: Implement with repository integration
	return fmt.Errorf("not implemented: requires repository integration")
}

// UpdateTags updates file tags
func (s *storageWrapper) UpdateTags(ctx context.Context, fileID string, tags []string) error {
	// TODO: Implement with repository integration
	return fmt.Errorf("not implemented: requires repository integration")
}

// GetUsage returns storage usage
func (s *storageWrapper) GetUsage(ctx context.Context) (*models.StorageUsage, error) {
	usage, err := s.provider.GetUsage(ctx)
	if err != nil {
		return nil, err
	}
	usage.StorageID = s.storage.storageID
	usage.MaxSize = s.storage.config.MaxSize
	if usage.MaxSize > 0 {
		usage.UsagePercent = float64(usage.TotalSize) / float64(usage.MaxSize) * 100
	}
	return usage, nil
}
