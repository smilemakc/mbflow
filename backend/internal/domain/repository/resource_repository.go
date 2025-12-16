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

// CredentialsRepository defines the interface for credentials resource operations
type CredentialsRepository interface {
	// CreateCredentials creates a new credentials resource with encrypted data
	CreateCredentials(ctx context.Context, cred *models.CredentialsResource) error

	// GetCredentials retrieves credentials by resource ID (encrypted data only)
	GetCredentials(ctx context.Context, resourceID string) (*models.CredentialsResource, error)

	// GetCredentialsByOwner retrieves all credentials for an owner (encrypted data only)
	GetCredentialsByOwner(ctx context.Context, ownerID string) ([]*models.CredentialsResource, error)

	// GetCredentialsByProvider retrieves credentials by provider for an owner
	GetCredentialsByProvider(ctx context.Context, ownerID, provider string) ([]*models.CredentialsResource, error)

	// UpdateCredentials updates credentials resource
	UpdateCredentials(ctx context.Context, cred *models.CredentialsResource) error

	// UpdateEncryptedData updates only the encrypted data
	UpdateEncryptedData(ctx context.Context, resourceID string, encryptedData map[string]string) error

	// DeleteCredentials soft-deletes a credentials resource
	DeleteCredentials(ctx context.Context, resourceID string) error

	// IncrementUsageCount increments the usage counter and updates last_used_at
	IncrementUsageCount(ctx context.Context, resourceID string) error

	// LogCredentialAccess logs an access event to the audit log
	LogCredentialAccess(ctx context.Context, resourceID, action, actorID, actorType string, metadata map[string]interface{}) error
}
