package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/pkg/models"
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
	LogCredentialAccess(ctx context.Context, resourceID, action, actorID, actorType string, metadata map[string]any) error
}

// RentalKeyRepository defines the interface for rental key resource operations
type RentalKeyRepository interface {
	// CRUD operations
	CreateRentalKey(ctx context.Context, key *models.RentalKeyResource, plainAPIKey string) error
	GetRentalKey(ctx context.Context, resourceID string) (*models.RentalKeyResource, error)
	GetRentalKeysByOwner(ctx context.Context, ownerID string) ([]*models.RentalKeyResource, error)
	GetRentalKeysByProvider(ctx context.Context, ownerID string, provider models.LLMProviderType) ([]*models.RentalKeyResource, error)
	UpdateRentalKey(ctx context.Context, key *models.RentalKeyResource) error
	DeleteRentalKey(ctx context.Context, resourceID string) error

	// API key management (internal use only)
	GetDecryptedAPIKey(ctx context.Context, resourceID string) (string, error)
	RotateAPIKey(ctx context.Context, resourceID string, newPlainAPIKey string) error

	// Usage tracking
	RecordUsage(ctx context.Context, resourceID string, usage *models.RentalKeyUsageRecord) error
	GetUsageHistory(ctx context.Context, resourceID string, limit int, offset int) ([]*models.RentalKeyUsageRecord, error)
	GetUsageHistoryByTimeRange(ctx context.Context, resourceID string, from, to string) ([]*models.RentalKeyUsageRecord, error)
	GetUsageSummary(ctx context.Context, resourceID string) (*models.MultimodalUsage, int64, float64, error)

	// Usage reset (for scheduled jobs)
	ResetDailyUsage(ctx context.Context) error
	ResetMonthlyUsage(ctx context.Context) error

	// Admin operations
	GetAllRentalKeys(ctx context.Context, filter RentalKeyFilter) ([]*models.RentalKeyResource, int64, error)
	GetAllRentalKeysCount(ctx context.Context, filter RentalKeyFilter) (int64, error)
}

// RentalKeyFilter defines filter options for admin queries
type RentalKeyFilter struct {
	Provider  *models.LLMProviderType
	Status    *models.ResourceStatus
	OwnerID   *string
	CreatedBy *string
	Limit     int
	Offset    int
}

// ServiceKeyFilter defines filter options for listing service keys
type ServiceKeyFilter struct {
	UserID    *uuid.UUID
	Status    *string
	CreatedBy *uuid.UUID
	Limit     int
	Offset    int
}

// ServiceKeyRepository defines the interface for service key persistence
type ServiceKeyRepository interface {
	// Create creates a new service key
	Create(ctx context.Context, key *models.ServiceKey) error

	// FindByID finds a service key by ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.ServiceKey, error)

	// FindByPrefix finds a service key by its prefix (for validation)
	FindByPrefix(ctx context.Context, prefix string) ([]*models.ServiceKey, error)

	// FindByUserID returns all service keys for a user
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ServiceKey, error)

	// FindAll returns all service keys with optional filters
	FindAll(ctx context.Context, filter ServiceKeyFilter) ([]*models.ServiceKey, int64, error)

	// Update updates a service key
	Update(ctx context.Context, key *models.ServiceKey) error

	// Delete permanently deletes a service key
	Delete(ctx context.Context, id uuid.UUID) error

	// Revoke marks a service key as revoked
	Revoke(ctx context.Context, id uuid.UUID) error

	// UpdateLastUsed updates the last used timestamp and increments usage count
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error

	// CountByUserID returns the number of service keys for a user
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

// SystemKeyFilter defines filter options for listing system keys
type SystemKeyFilter struct {
	ServiceName *string
	Status      *string
	CreatedBy   *uuid.UUID
	Limit       int
	Offset      int
}

// SystemKeyRepository defines the interface for system key persistence
type SystemKeyRepository interface {
	Create(ctx context.Context, key *models.SystemKey) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.SystemKey, error)
	FindByPrefix(ctx context.Context, prefix string) ([]*models.SystemKey, error)
	FindAll(ctx context.Context, filter SystemKeyFilter) ([]*models.SystemKey, int64, error)
	Update(ctx context.Context, key *models.SystemKey) error
	Delete(ctx context.Context, id uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
}

// ServiceAuditLogFilter defines filter options for listing audit logs
type ServiceAuditLogFilter struct {
	SystemKeyID        *uuid.UUID
	ServiceName        *string
	Action             *string
	ResourceType       *string
	ImpersonatedUserID *uuid.UUID
	DateFrom           *time.Time
	DateTo             *time.Time
	Limit              int
	Offset             int
}

// ServiceAuditLogRepository defines the interface for audit log persistence
type ServiceAuditLogRepository interface {
	Create(ctx context.Context, log *models.ServiceAuditLog) error
	FindAll(ctx context.Context, filter ServiceAuditLogFilter) ([]*models.ServiceAuditLog, int64, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}
