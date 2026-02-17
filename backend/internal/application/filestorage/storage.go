// Package filestorage provides file storage functionality with pluggable backends.
package filestorage

import (
	"context"
	"io"

	"github.com/smilemakc/mbflow/pkg/models"
)

// Provider defines the interface for storage backend implementations.
// Implementations include local disk storage, S3, GCS, etc.
type Provider interface {
	// Type returns the storage provider type
	Type() models.StorageType

	// Store stores a file and returns the path where it was stored
	Store(ctx context.Context, entry *models.FileEntry, reader io.Reader) (path string, err error)

	// Get retrieves a file by its path
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes a file by its path
	Delete(ctx context.Context, path string) error

	// Exists checks if a file exists at the given path
	Exists(ctx context.Context, path string) (bool, error)

	// GetUsage returns storage usage statistics
	GetUsage(ctx context.Context) (*models.StorageUsage, error)

	// Close closes the provider and releases resources
	Close() error
}

// Storage is the main interface for file storage operations.
// It uses a Provider for actual file operations and manages metadata.
type Storage interface {
	// Store stores a file with the given entry metadata
	Store(ctx context.Context, entry *models.FileEntry, reader io.Reader) (*models.FileEntry, error)

	// Get retrieves a file by ID
	Get(ctx context.Context, fileID string) (*models.FileEntry, io.ReadCloser, error)

	// Delete removes a file by ID
	Delete(ctx context.Context, fileID string) error

	// List lists files matching the query
	List(ctx context.Context, query *FileQuery) ([]*models.FileEntry, error)

	// Exists checks if a file exists
	Exists(ctx context.Context, fileID string) (bool, error)

	// GetMetadata retrieves file metadata without content
	GetMetadata(ctx context.Context, fileID string) (*models.FileEntry, error)

	// UpdateMetadata updates file metadata
	UpdateMetadata(ctx context.Context, fileID string, metadata map[string]any) error

	// UpdateTags updates file tags
	UpdateTags(ctx context.Context, fileID string, tags []string) error

	// GetUsage returns storage usage statistics
	GetUsage(ctx context.Context) (*models.StorageUsage, error)
}

// FileQuery defines query parameters for listing files
type FileQuery struct {
	StorageID   string             `json:"storage_id,omitempty"`
	WorkflowID  string             `json:"workflow_id,omitempty"`
	ExecutionID string             `json:"execution_id,omitempty"`
	MimeTypes   []string           `json:"mime_types,omitempty"`
	AccessScope models.AccessScope `json:"access_scope,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	NamePattern string             `json:"name_pattern,omitempty"` // LIKE pattern
	Expired     *bool              `json:"expired,omitempty"`      // Filter by expiration
	Limit       int                `json:"limit,omitempty"`
	Offset      int                `json:"offset,omitempty"`
	OrderBy     string             `json:"order_by,omitempty"`  // created_at, name, size
	OrderDir    string             `json:"order_dir,omitempty"` // asc, desc
}

// Manager manages multiple storage instances and observers
type Manager interface {
	// GetStorage returns a storage instance by ID
	GetStorage(storageID string) (Storage, error)

	// CreateStorage creates a new storage instance
	CreateStorage(storageID string, config *models.StorageConfig) (Storage, error)

	// DeleteStorage deletes a storage instance and all its files
	DeleteStorage(storageID string) error

	// ListStorages lists all storage IDs
	ListStorages() []string

	// HasStorage checks if a storage exists
	HasStorage(storageID string) bool

	// GetDefaultStorage returns the default storage instance
	GetDefaultStorage() (Storage, error)

	// RegisterObserver registers a file event observer
	RegisterObserver(observer FileObserver) error

	// UnregisterObserver removes an observer by name
	UnregisterObserver(name string) error

	// Cleanup removes expired files from all storages
	Cleanup(ctx context.Context) (removed int, err error)

	// Close closes the manager and all storages
	Close() error
}

// ProviderFactory creates storage providers
type ProviderFactory interface {
	// Create creates a provider with the given config
	Create(config *models.StorageConfig) (Provider, error)

	// Type returns the storage type this factory creates
	Type() models.StorageType
}
