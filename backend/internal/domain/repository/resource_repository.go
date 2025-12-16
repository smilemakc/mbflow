package repository

import (
	"context"

	"github.com/smilemakc/mbflow/pkg/models"
)

// ResourceRepository defines the interface for resource persistence operations
type ResourceRepository interface {
	// Create creates a new resource
	Create(ctx context.Context, resource models.Resource) error

	// GetByID retrieves a resource by its ID
	GetByID(ctx context.Context, id string) (models.Resource, error)

	// GetByOwner retrieves all resources for a specific owner
	GetByOwner(ctx context.Context, ownerID string) ([]models.Resource, error)

	// GetByOwnerAndType retrieves resources of specific type for an owner
	GetByOwnerAndType(ctx context.Context, ownerID string, resourceType models.ResourceType) ([]models.Resource, error)

	// Update updates an existing resource
	Update(ctx context.Context, resource models.Resource) error

	// Delete soft-deletes a resource
	Delete(ctx context.Context, id string) error

	// HardDelete permanently removes a resource
	HardDelete(ctx context.Context, id string) error
}

// FileStorageRepository defines the interface for file storage resource operations
type FileStorageRepository interface {
	ResourceRepository

	// GetFileStorage retrieves file storage specific data by resource ID
	GetFileStorage(ctx context.Context, resourceID string) (*models.FileStorageResource, error)

	// UpdateUsage updates storage usage metrics
	UpdateUsage(ctx context.Context, resourceID string, usedBytes int64, fileCount int) error

	// IncrementUsage atomically increments storage usage
	IncrementUsage(ctx context.Context, resourceID string, bytesAdded int64) error

	// DecrementUsage atomically decrements storage usage
	DecrementUsage(ctx context.Context, resourceID string, bytesRemoved int64) error
}
