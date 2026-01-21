package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

var _ repository.CredentialsRepository = (*CredentialsRepositoryImpl)(nil)

// CredentialsRepositoryImpl implements the CredentialsRepository interface
type CredentialsRepositoryImpl struct {
	db *bun.DB
}

// NewCredentialsRepository creates a new CredentialsRepositoryImpl
func NewCredentialsRepository(db *bun.DB) *CredentialsRepositoryImpl {
	return &CredentialsRepositoryImpl{db: db}
}

// CreateCredentials creates a new credentials resource
func (r *CredentialsRepositoryImpl) CreateCredentials(ctx context.Context, cred *pkgmodels.CredentialsResource) error {
	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		resourceModel := &models.ResourceModel{
			ID:          uuid.New(),
			Type:        string(pkgmodels.ResourceTypeCredentials),
			OwnerID:     uuid.MustParse(cred.OwnerID),
			Name:        cred.Name,
			Description: cred.Description,
			Status:      string(cred.Status),
			Metadata:    cred.Metadata,
		}

		if _, err := tx.NewInsert().Model(resourceModel).Exec(ctx); err != nil {
			return err
		}

		// Convert encrypted data to JSONBMap
		encryptedData := make(models.JSONBMap)
		for k, v := range cred.EncryptedData {
			encryptedData[k] = v
		}

		var provider *string
		if cred.Provider != "" {
			provider = &cred.Provider
		}

		credentialsModel := &models.CredentialsModel{
			ResourceID:     resourceModel.ID,
			CredentialType: string(cred.CredentialType),
			EncryptedData:  encryptedData,
			Provider:       provider,
			ExpiresAt:      cred.ExpiresAt,
			UsageCount:     0,
		}

		if _, err := tx.NewInsert().Model(credentialsModel).Exec(ctx); err != nil {
			return err
		}

		// Update the domain model with generated values
		cred.ID = resourceModel.ID.String()
		cred.CreatedAt = resourceModel.CreatedAt
		cred.UpdatedAt = resourceModel.UpdatedAt

		// Log creation
		if err := r.logAccessInTx(ctx, tx, resourceModel.ID.String(), "created", cred.OwnerID, "user", nil); err != nil {
			return err
		}

		return nil
	})
}

// GetCredentials retrieves credentials by resource ID
func (r *CredentialsRepositoryImpl) GetCredentials(ctx context.Context, resourceID string) (*pkgmodels.CredentialsResource, error) {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	resourceModel := new(models.ResourceModel)
	err = r.db.NewSelect().
		Model(resourceModel).
		Relation("Credentials").
		Where("r.id = ? AND r.deleted_at IS NULL", resID).
		Where("r.type = ?", string(pkgmodels.ResourceTypeCredentials)).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, pkgmodels.ErrResourceNotFound
		}
		return nil, err
	}

	if resourceModel.Credentials == nil {
		return nil, pkgmodels.ErrResourceNotFound
	}

	return models.ToCredentialsResourceDomain(resourceModel, resourceModel.Credentials), nil
}

// GetCredentialsByOwner retrieves all credentials for an owner
func (r *CredentialsRepositoryImpl) GetCredentialsByOwner(ctx context.Context, ownerID string) ([]*pkgmodels.CredentialsResource, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var resourceModels []*models.ResourceModel
	err = r.db.NewSelect().
		Model(&resourceModels).
		Relation("Credentials").
		Where("r.owner_id = ? AND r.deleted_at IS NULL", ownerUUID).
		Where("r.type = ?", string(pkgmodels.ResourceTypeCredentials)).
		Order("r.created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	credentials := make([]*pkgmodels.CredentialsResource, 0, len(resourceModels))
	for _, rm := range resourceModels {
		if rm.Credentials != nil {
			credentials = append(credentials, models.ToCredentialsResourceDomain(rm, rm.Credentials))
		}
	}

	return credentials, nil
}

// GetCredentialsByProvider retrieves credentials by provider for an owner
func (r *CredentialsRepositoryImpl) GetCredentialsByProvider(ctx context.Context, ownerID, provider string) ([]*pkgmodels.CredentialsResource, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var resourceModels []*models.ResourceModel
	err = r.db.NewSelect().
		Model(&resourceModels).
		Relation("Credentials", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("rc.provider = ?", provider)
		}).
		Where("r.owner_id = ? AND r.deleted_at IS NULL", ownerUUID).
		Where("r.type = ?", string(pkgmodels.ResourceTypeCredentials)).
		Order("r.created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	credentials := make([]*pkgmodels.CredentialsResource, 0, len(resourceModels))
	for _, rm := range resourceModels {
		if rm.Credentials != nil {
			credentials = append(credentials, models.ToCredentialsResourceDomain(rm, rm.Credentials))
		}
	}

	return credentials, nil
}

// UpdateCredentials updates credentials resource
func (r *CredentialsRepositoryImpl) UpdateCredentials(ctx context.Context, cred *pkgmodels.CredentialsResource) error {
	resourceID, err := uuid.Parse(cred.ID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		// Update base resource
		_, err := tx.NewUpdate().
			Model((*models.ResourceModel)(nil)).
			Set("name = ?", cred.Name).
			Set("description = ?", cred.Description).
			Set("status = ?", string(cred.Status)).
			Set("metadata = ?", cred.Metadata).
			Set("updated_at = ?", time.Now()).
			Where("id = ? AND deleted_at IS NULL", resourceID).
			Exec(ctx)

		if err != nil {
			return err
		}

		// Convert encrypted data
		encryptedData := make(models.JSONBMap)
		for k, v := range cred.EncryptedData {
			encryptedData[k] = v
		}

		var provider *string
		if cred.Provider != "" {
			provider = &cred.Provider
		}

		// Update credentials-specific data
		_, err = tx.NewUpdate().
			Model((*models.CredentialsModel)(nil)).
			Set("credential_type = ?", string(cred.CredentialType)).
			Set("encrypted_data = ?", encryptedData).
			Set("provider = ?", provider).
			Set("expires_at = ?", cred.ExpiresAt).
			Where("resource_id = ?", resourceID).
			Exec(ctx)

		if err != nil {
			return err
		}

		// Log update
		return r.logAccessInTx(ctx, tx, cred.ID, "updated", cred.OwnerID, "user", nil)
	})
}

// UpdateEncryptedData updates only the encrypted data
func (r *CredentialsRepositoryImpl) UpdateEncryptedData(ctx context.Context, resourceID string, encryptedData map[string]string) error {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	// Convert to JSONBMap
	data := make(models.JSONBMap)
	for k, v := range encryptedData {
		data[k] = v
	}

	_, err = r.db.NewUpdate().
		Model((*models.CredentialsModel)(nil)).
		Set("encrypted_data = ?", data).
		Where("resource_id = ?", resID).
		Exec(ctx)

	return err
}

// DeleteCredentials soft-deletes a credentials resource
func (r *CredentialsRepositoryImpl) DeleteCredentials(ctx context.Context, resourceID string) error {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		// Get owner ID for audit log
		var resource models.ResourceModel
		err := tx.NewSelect().
			Model(&resource).
			Column("owner_id").
			Where("id = ?", resID).
			Scan(ctx)
		if err != nil {
			return err
		}

		// Soft delete the resource
		_, err = tx.NewUpdate().
			Model((*models.ResourceModel)(nil)).
			Set("deleted_at = ?", time.Now()).
			Set("status = ?", string(pkgmodels.ResourceStatusDeleted)).
			Where("id = ? AND deleted_at IS NULL", resID).
			Exec(ctx)

		if err != nil {
			return err
		}

		// Log deletion
		return r.logAccessInTx(ctx, tx, resourceID, "deleted", resource.OwnerID.String(), "user", nil)
	})
}

// IncrementUsageCount increments the usage counter and updates last_used_at
func (r *CredentialsRepositoryImpl) IncrementUsageCount(ctx context.Context, resourceID string) error {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.CredentialsModel)(nil)).
		Set("usage_count = usage_count + 1").
		Set("last_used_at = ?", time.Now()).
		Where("resource_id = ?", resID).
		Exec(ctx)

	return err
}

// LogCredentialAccess logs an access event to the audit log
func (r *CredentialsRepositoryImpl) LogCredentialAccess(ctx context.Context, resourceID, action, actorID, actorType string, metadata map[string]interface{}) error {
	return r.logAccessInTx(ctx, r.db, resourceID, action, actorID, actorType, metadata)
}

// CredentialAuditLogModel represents the audit log entry
type CredentialAuditLogModel struct {
	bun.BaseModel `bun:"table:mbflow_credential_audit_log,alias:cal"`

	ID           uuid.UUID       `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
	CredentialID uuid.UUID       `bun:"credential_id,type:uuid,notnull"`
	Action       string          `bun:"action,notnull"`
	ActorID      *uuid.UUID      `bun:"actor_id,type:uuid"`
	ActorType    string          `bun:"actor_type,notnull,default:'user'"`
	IPAddress    *string         `bun:"ip_address"`
	UserAgent    *string         `bun:"user_agent"`
	Metadata     models.JSONBMap `bun:"metadata,type:jsonb,default:'{}'"`
	CreatedAt    time.Time       `bun:"created_at,notnull,default:current_timestamp"`
}

func (r *CredentialsRepositoryImpl) logAccessInTx(ctx context.Context, db bun.IDB, resourceID, action, actorID, actorType string, metadata map[string]interface{}) error {
	credID, err := uuid.Parse(resourceID)
	if err != nil {
		return err
	}

	var actorUUID *uuid.UUID
	if actorID != "" {
		parsed, err := uuid.Parse(actorID)
		if err == nil {
			actorUUID = &parsed
		}
	}

	var meta models.JSONBMap
	if metadata != nil {
		meta = models.JSONBMap(metadata)
	}

	log := &CredentialAuditLogModel{
		ID:           uuid.New(),
		CredentialID: credID,
		Action:       action,
		ActorID:      actorUUID,
		ActorType:    actorType,
		Metadata:     meta,
		CreatedAt:    time.Now(),
	}

	_, err = db.NewInsert().Model(log).Exec(ctx)
	return err
}
