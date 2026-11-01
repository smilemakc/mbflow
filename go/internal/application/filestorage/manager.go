package filestorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	storagerepo "github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
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
	logger      *logger.Logger
	metadata    MetadataRepository
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
func NewStorageManager(config *ManagerConfig, log *logger.Logger) *StorageManager {
	if config == nil {
		config = DefaultManagerConfig()
	}

	m := &StorageManager{
		config:      config,
		storages:    make(map[string]*managedStorage),
		factories:   make(map[models.StorageType]ProviderFactory),
		observers:   make(map[string]FileObserver),
		validator:   NewMimeValidator(),
		logger:      log,
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

// SetMetadataRepository attaches the repository holding file metadata
// records. With it, storages resolve file IDs (not only paths), and List,
// GetMetadata, UpdateMetadata, UpdateTags and Cleanup become available.
// Store does not create metadata records: callers persist them (the record
// may need fields such as owner or resource that the storage does not know).
func (m *StorageManager) SetMetadataRepository(repo MetadataRepository) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata = repo
}

// metadataRepository returns the configured metadata repository, if any.
func (m *StorageManager) metadataRepository() MetadataRepository {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metadata
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
			go func(o FileObserver) {
				if err := o.OnFileEvent(ctx, event); err != nil && m.logger != nil {
					m.logger.Error("File observer error",
						"observer", o.Name(),
						"event_type", event.Type,
						"storage_id", event.StorageID,
						"error", err,
					)
				}
			}(obs)
		}
	}
}

// cleanupBatchSize bounds how many expired files one Cleanup pass removes.
const cleanupBatchSize = 500

// Cleanup removes expired files: their content from the storage provider
// and their metadata record. Expiration is tracked in the metadata
// repository, so without one there is nothing to clean up and Cleanup
// returns 0.
func (m *StorageManager) Cleanup(ctx context.Context) (int, error) {
	repo := m.metadataRepository()
	if repo == nil {
		return 0, nil
	}

	expired, err := repo.FindExpired(ctx, cleanupBatchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to find expired files: %w", err)
	}

	deleted := 0
	var errs []error
	for _, file := range expired {
		if err := m.removeExpired(ctx, repo, file); err != nil {
			errs = append(errs, fmt.Errorf("file %s: %w", file.ID, err))
			continue
		}
		deleted++
	}

	if m.logger != nil && (deleted > 0 || len(errs) > 0) {
		m.logger.Info("File cleanup completed",
			"deleted_count", deleted,
			"failed_count", len(errs),
		)
	}

	return deleted, errors.Join(errs...)
}

// removeExpired deletes one expired file's content and metadata.
func (m *StorageManager) removeExpired(ctx context.Context, repo MetadataRepository, file *storagemodels.FileModel) error {
	// Storages are opened lazily, so after a restart the storage holding
	// an expired file is usually not open yet: open it the same way
	// readers do.
	store, err := m.GetStorage(file.StorageID)
	if err != nil {
		return fmt.Errorf("failed to open storage %s: %w", file.StorageID, err)
	}
	wrapper, ok := store.(*storageWrapper)
	if !ok {
		return fmt.Errorf("unexpected storage implementation %T", store)
	}
	if err := wrapper.provider.Delete(ctx, file.Path); err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}
	if err := repo.Delete(ctx, file.ID); err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	m.notifyObservers(ctx, NewFileEvent(EventFileExpired, file.StorageID, modelToEntry(file)))
	return nil
}

// cleanupRoutine runs periodic cleanup
func (m *StorageManager) cleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := m.Cleanup(context.Background()); err != nil && m.logger != nil {
				m.logger.Error("File cleanup failed", "error", err)
			}
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

	var errs []error
	for id, managed := range m.storages {
		if err := managed.provider.Close(); err != nil {
			errs = append(errs, fmt.Errorf("storage %s: %w", id, err))
		}
	}

	return errors.Join(errs...)
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

// resolve maps a file reference to its metadata record and storage path.
//
// A reference is either a file ID (UUID) or a storage path as returned by
// Store. IDs are resolved through the metadata repository and must belong to
// this storage; when no repository is configured, or the reference is not a
// UUID, it is treated as a path and the returned model is nil.
func (s *storageWrapper) resolve(ctx context.Context, fileRef string) (*storagemodels.FileModel, string, error) {
	repo := s.manager.metadataRepository()
	id, err := uuid.Parse(fileRef)
	if repo == nil || err != nil {
		return nil, fileRef, nil
	}

	model, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %s", ErrFileNotFound, fileRef)
	}
	if model.StorageID != s.storage.storageID {
		return nil, "", fmt.Errorf("%w: %s", ErrFileNotFound, fileRef)
	}
	return model, model.Path, nil
}

// requireModel resolves a file ID to its metadata record; it fails when no
// metadata repository is configured or the reference is not a known file ID.
func (s *storageWrapper) requireModel(ctx context.Context, fileID string) (*storagemodels.FileModel, error) {
	if s.manager.metadataRepository() == nil {
		return nil, ErrMetadataUnavailable
	}
	model, _, err := s.resolve(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, fmt.Errorf("%w: %s", ErrFileNotFound, fileID)
	}
	return model, nil
}

// Get retrieves a file by ID or by storage path (as returned by Store).
// When the file is resolved by ID the returned entry carries the full
// metadata; for a path only a minimal entry is returned.
func (s *storageWrapper) Get(ctx context.Context, fileID string) (*models.FileEntry, io.ReadCloser, error) {
	model, path, err := s.resolve(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	reader, err := s.provider.Get(ctx, path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file: %w", err)
	}
	if model != nil {
		return modelToEntry(model), reader, nil
	}
	entry := &models.FileEntry{
		ID:        fileID,
		StorageID: s.storage.storageID,
		Path:      path,
	}
	return entry, reader, nil
}

// Delete removes a file by ID or path. For a file resolved by ID its
// metadata record is removed as well.
func (s *storageWrapper) Delete(ctx context.Context, fileID string) error {
	model, path, err := s.resolve(ctx, fileID)
	if err != nil {
		return err
	}
	if err := s.provider.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	entry := &models.FileEntry{ID: fileID, StorageID: s.storage.storageID, Path: path}
	if model != nil {
		if err := s.manager.metadataRepository().Delete(ctx, model.ID); err != nil {
			return fmt.Errorf("failed to delete file metadata: %w", err)
		}
		entry = modelToEntry(model)
	}

	s.manager.notifyObservers(ctx, NewFileEvent(EventFileRemoved, s.storage.storageID, entry))
	return nil
}

// List lists the files of this storage matching the query. It requires a
// metadata repository.
func (s *storageWrapper) List(ctx context.Context, query *FileQuery) ([]*models.FileEntry, error) {
	repo := s.manager.metadataRepository()
	if repo == nil {
		return nil, ErrMetadataUnavailable
	}
	if query == nil {
		query = &FileQuery{}
	}

	repoQuery := &storagerepo.FileQuery{
		// A storage only ever lists its own files.
		StorageID:   s.storage.storageID,
		MimeTypes:   query.MimeTypes,
		AccessScope: string(query.AccessScope),
		Tags:        query.Tags,
		Expired:     query.Expired,
		Limit:       query.Limit,
		Offset:      query.Offset,
		OrderBy:     query.OrderBy,
		OrderDir:    query.OrderDir,
	}
	if query.WorkflowID != "" {
		id, err := uuid.Parse(query.WorkflowID)
		if err != nil {
			return nil, fmt.Errorf("invalid workflow ID %q: %w", query.WorkflowID, err)
		}
		repoQuery.WorkflowID = &id
	}
	if query.ExecutionID != "" {
		id, err := uuid.Parse(query.ExecutionID)
		if err != nil {
			return nil, fmt.Errorf("invalid execution ID %q: %w", query.ExecutionID, err)
		}
		repoQuery.ExecutionID = &id
	}

	files, err := repo.FindByQuery(ctx, repoQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	entries := make([]*models.FileEntry, 0, len(files))
	for _, f := range files {
		entries = append(entries, modelToEntry(f))
	}
	return entries, nil
}

// Exists checks if a file exists, by ID or path.
func (s *storageWrapper) Exists(ctx context.Context, fileID string) (bool, error) {
	_, path, err := s.resolve(ctx, fileID)
	if errors.Is(err, ErrFileNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return s.provider.Exists(ctx, path)
}

// GetMetadata retrieves file metadata by file ID. It requires a metadata
// repository.
func (s *storageWrapper) GetMetadata(ctx context.Context, fileID string) (*models.FileEntry, error) {
	model, err := s.requireModel(ctx, fileID)
	if err != nil {
		return nil, err
	}
	return modelToEntry(model), nil
}

// UpdateMetadata merges metadata into the file's metadata (a nil value
// removes the key). It requires a metadata repository.
func (s *storageWrapper) UpdateMetadata(ctx context.Context, fileID string, metadata map[string]any) error {
	model, err := s.requireModel(ctx, fileID)
	if err != nil {
		return err
	}
	if model.Metadata == nil {
		model.Metadata = make(storagemodels.JSONBMap, len(metadata))
	}
	for k, v := range metadata {
		if v == nil {
			delete(model.Metadata, k)
			continue
		}
		model.Metadata[k] = v
	}
	return s.saveModel(ctx, model)
}

// UpdateTags replaces the file's tags. It requires a metadata repository.
func (s *storageWrapper) UpdateTags(ctx context.Context, fileID string, tags []string) error {
	model, err := s.requireModel(ctx, fileID)
	if err != nil {
		return err
	}
	model.Tags = storagemodels.StringArray(append([]string(nil), tags...))
	return s.saveModel(ctx, model)
}

// saveModel persists an updated metadata record and notifies observers.
func (s *storageWrapper) saveModel(ctx context.Context, model *storagemodels.FileModel) error {
	if err := s.manager.metadataRepository().Update(ctx, model); err != nil {
		return fmt.Errorf("failed to update file metadata: %w", err)
	}
	s.manager.notifyObservers(ctx, NewFileEvent(EventFileUpdated, s.storage.storageID, modelToEntry(model)))
	return nil
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
