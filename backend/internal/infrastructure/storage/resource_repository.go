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

var _ repository.FileStorageRepository = (*ResourceRepositoryImpl)(nil)

type ResourceRepositoryImpl struct {
	db *bun.DB
}

func NewResourceRepository(db *bun.DB) *ResourceRepositoryImpl {
	return &ResourceRepositoryImpl{db: db}
}

func (r *ResourceRepositoryImpl) Create(ctx context.Context, resource pkgmodels.Resource) error {
	fsResource, ok := resource.(*pkgmodels.FileStorageResource)
	if !ok {
		return pkgmodels.ErrInvalidResourceType
	}

	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		resourceModel := &models.ResourceModel{
			ID:          uuid.New(),
			Type:        string(fsResource.Type),
			OwnerID:     uuid.MustParse(fsResource.OwnerID),
			Name:        fsResource.Name,
			Description: fsResource.Description,
			Status:      string(fsResource.Status),
			Metadata:    fsResource.Metadata,
		}

		if _, err := tx.NewInsert().Model(resourceModel).Exec(ctx); err != nil {
			return err
		}

		fileStorageModel := &models.FileStorageModel{
			ResourceID:        resourceModel.ID,
			StorageLimitBytes: fsResource.StorageLimitBytes,
			UsedStorageBytes:  fsResource.UsedStorageBytes,
			FileCount:         fsResource.FileCount,
		}
		if fsResource.PricingPlanID != "" {
			planID := uuid.MustParse(fsResource.PricingPlanID)
			fileStorageModel.PricingPlanID = &planID
		}

		_, err := tx.NewInsert().Model(fileStorageModel).Exec(ctx)
		if err != nil {
			return err
		}

		fsResource.ID = resourceModel.ID.String()
		fsResource.CreatedAt = resourceModel.CreatedAt
		fsResource.UpdatedAt = resourceModel.UpdatedAt

		return nil
	})
}

func (r *ResourceRepositoryImpl) GetByID(ctx context.Context, id string) (pkgmodels.Resource, error) {
	resourceID, err := uuid.Parse(id)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	resourceModel := new(models.ResourceModel)
	err = r.db.NewSelect().
		Model(resourceModel).
		Relation("FileStorage").
		Relation("Credentials").
		Relation("RentalKey").
		Where("r.id = ? AND r.deleted_at IS NULL", resourceID).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, pkgmodels.ErrResourceNotFound
		}
		return nil, err
	}

	return r.toResourceDomain(resourceModel), nil
}

func (r *ResourceRepositoryImpl) GetByOwner(ctx context.Context, ownerID string) ([]pkgmodels.Resource, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var resourceModels []*models.ResourceModel
	err = r.db.NewSelect().
		Model(&resourceModels).
		Relation("FileStorage").
		Relation("Credentials").
		Relation("RentalKey").
		Where("r.owner_id = ? AND r.deleted_at IS NULL", ownerUUID).
		Order("r.created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	resources := make([]pkgmodels.Resource, 0, len(resourceModels))
	for _, rm := range resourceModels {
		if res := r.toResourceDomain(rm); res != nil {
			resources = append(resources, res)
		}
	}

	return resources, nil
}

func (r *ResourceRepositoryImpl) GetByOwnerAndType(ctx context.Context, ownerID string, resourceType pkgmodels.ResourceType) ([]pkgmodels.Resource, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var resourceModels []*models.ResourceModel
	err = r.db.NewSelect().
		Model(&resourceModels).
		Relation("FileStorage").
		Relation("Credentials").
		Relation("RentalKey").
		Where("r.owner_id = ? AND r.type = ? AND r.deleted_at IS NULL", ownerUUID, string(resourceType)).
		Order("r.created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	resources := make([]pkgmodels.Resource, 0, len(resourceModels))
	for _, rm := range resourceModels {
		if res := r.toResourceDomain(rm); res != nil {
			resources = append(resources, res)
		}
	}

	return resources, nil
}

func (r *ResourceRepositoryImpl) Update(ctx context.Context, resource pkgmodels.Resource) error {
	resourceID, err := uuid.Parse(resource.GetID())
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		// Update base resource fields
		_, err := tx.NewUpdate().
			Model((*models.ResourceModel)(nil)).
			Set("name = ?", resource.GetName()).
			Set("description = ?", resource.GetDescription()).
			Set("status = ?", string(resource.GetStatus())).
			Set("metadata = ?", resource.GetMetadata()).
			Set("updated_at = ?", time.Now()).
			Where("id = ? AND deleted_at IS NULL", resourceID).
			Exec(ctx)

		if err != nil {
			return err
		}

		// Update type-specific fields
		switch res := resource.(type) {
		case *pkgmodels.FileStorageResource:
			_, err = tx.NewUpdate().
				Model((*models.FileStorageModel)(nil)).
				Set("storage_limit_bytes = ?", res.StorageLimitBytes).
				Set("used_storage_bytes = ?", res.UsedStorageBytes).
				Set("file_count = ?", res.FileCount).
				Where("resource_id = ?", resourceID).
				Exec(ctx)
		case *pkgmodels.CredentialsResource:
			encryptedData := make(models.JSONBMap)
			for k, v := range res.EncryptedData {
				encryptedData[k] = v
			}
			var provider *string
			if res.Provider != "" {
				provider = &res.Provider
			}
			_, err = tx.NewUpdate().
				Model((*models.CredentialsModel)(nil)).
				Set("credential_type = ?", string(res.CredentialType)).
				Set("encrypted_data = ?", encryptedData).
				Set("provider = ?", provider).
				Set("expires_at = ?", res.ExpiresAt).
				Where("resource_id = ?", resourceID).
				Exec(ctx)
		}

		return err
	})
}

func (r *ResourceRepositoryImpl) Delete(ctx context.Context, id string) error {
	resourceID, err := uuid.Parse(id)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.ResourceModel)(nil)).
		Set("deleted_at = ?", time.Now()).
		Set("status = ?", string(pkgmodels.ResourceStatusDeleted)).
		Where("id = ? AND deleted_at IS NULL", resourceID).
		Exec(ctx)

	return err
}

func (r *ResourceRepositoryImpl) HardDelete(ctx context.Context, id string) error {
	resourceID, err := uuid.Parse(id)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewDelete().
		Model((*models.ResourceModel)(nil)).
		Where("id = ?", resourceID).
		Exec(ctx)

	return err
}

func (r *ResourceRepositoryImpl) GetFileStorage(ctx context.Context, resourceID string) (*pkgmodels.FileStorageResource, error) {
	resource, err := r.GetByID(ctx, resourceID)
	if err != nil {
		return nil, err
	}

	fsResource, ok := resource.(*pkgmodels.FileStorageResource)
	if !ok {
		return nil, pkgmodels.ErrInvalidResourceType
	}

	return fsResource, nil
}

func (r *ResourceRepositoryImpl) UpdateUsage(ctx context.Context, resourceID string, usedBytes int64, fileCount int) error {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.FileStorageModel)(nil)).
		Set("used_storage_bytes = ?", usedBytes).
		Set("file_count = ?", fileCount).
		Where("resource_id = ?", resID).
		Exec(ctx)

	return err
}

func (r *ResourceRepositoryImpl) IncrementUsage(ctx context.Context, resourceID string, bytesAdded int64) error {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.FileStorageModel)(nil)).
		Set("used_storage_bytes = used_storage_bytes + ?", bytesAdded).
		Set("file_count = file_count + 1").
		Where("resource_id = ?", resID).
		Exec(ctx)

	return err
}

func (r *ResourceRepositoryImpl) DecrementUsage(ctx context.Context, resourceID string, bytesRemoved int64) error {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.FileStorageModel)(nil)).
		Set("used_storage_bytes = GREATEST(0, used_storage_bytes - ?)", bytesRemoved).
		Set("file_count = GREATEST(0, file_count - 1)").
		Where("resource_id = ?", resID).
		Exec(ctx)

	return err
}

// toResourceDomain converts a ResourceModel to the appropriate domain type based on resource type
func (r *ResourceRepositoryImpl) toResourceDomain(rm *models.ResourceModel) pkgmodels.Resource {
	if rm == nil {
		return nil
	}

	switch pkgmodels.ResourceType(rm.Type) {
	case pkgmodels.ResourceTypeFileStorage:
		if rm.FileStorage != nil {
			return models.ToFileStorageResourceDomain(rm, rm.FileStorage)
		}
	case pkgmodels.ResourceTypeCredentials:
		if rm.Credentials != nil {
			return models.ToCredentialsResourceDomain(rm, rm.Credentials)
		}
	case pkgmodels.ResourceTypeRentalKey:
		if rm.RentalKey != nil {
			return models.ToRentalKeyResourceDomain(rm, rm.RentalKey)
		}
	}

	return nil
}
