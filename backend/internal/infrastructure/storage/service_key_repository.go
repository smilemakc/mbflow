package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

var _ repository.ServiceKeyRepository = (*ServiceKeyRepositoryImpl)(nil)

// ServiceKeyRepositoryImpl implements the ServiceKeyRepository interface
type ServiceKeyRepositoryImpl struct {
	db *bun.DB
}

// NewServiceKeyRepository creates a new ServiceKeyRepositoryImpl
func NewServiceKeyRepository(db *bun.DB) *ServiceKeyRepositoryImpl {
	return &ServiceKeyRepositoryImpl{db: db}
}

// Create creates a new service key
func (r *ServiceKeyRepositoryImpl) Create(ctx context.Context, key *pkgmodels.ServiceKey) error {
	model := models.FromServiceKeyDomain(key)

	_, err := r.db.NewInsert().
		Model(model).
		Exec(ctx)

	if err != nil {
		return err
	}

	key.ID = model.ID.String()
	key.CreatedAt = model.CreatedAt
	key.UpdatedAt = model.UpdatedAt

	return nil
}

// FindByID finds a service key by ID
func (r *ServiceKeyRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*pkgmodels.ServiceKey, error) {
	model := new(models.ServiceKeyModel)

	err := r.db.NewSelect().
		Model(model).
		Where("sk.id = ?", id).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgmodels.ErrServiceKeyNotFound
		}
		return nil, err
	}

	return model.ToServiceKeyDomain(), nil
}

// FindByPrefix finds a service key by its prefix
func (r *ServiceKeyRepositoryImpl) FindByPrefix(ctx context.Context, prefix string) ([]*pkgmodels.ServiceKey, error) {
	var modelList []*models.ServiceKeyModel

	err := r.db.NewSelect().
		Model(&modelList).
		Where("sk.key_prefix = ?", prefix).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	keys := make([]*pkgmodels.ServiceKey, 0, len(modelList))
	for _, model := range modelList {
		keys = append(keys, model.ToServiceKeyDomain())
	}

	return keys, nil
}

// FindByUserID returns all service keys for a user
func (r *ServiceKeyRepositoryImpl) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*pkgmodels.ServiceKey, error) {
	var modelList []*models.ServiceKeyModel

	err := r.db.NewSelect().
		Model(&modelList).
		Where("sk.user_id = ?", userID).
		Order("sk.created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	keys := make([]*pkgmodels.ServiceKey, 0, len(modelList))
	for _, model := range modelList {
		keys = append(keys, model.ToServiceKeyDomain())
	}

	return keys, nil
}

// FindAll returns all service keys with optional filters
func (r *ServiceKeyRepositoryImpl) FindAll(ctx context.Context, filter repository.ServiceKeyFilter) ([]*pkgmodels.ServiceKey, int64, error) {
	var modelList []*models.ServiceKeyModel

	query := r.db.NewSelect().
		Model(&modelList)

	if filter.UserID != nil {
		query = query.Where("sk.user_id = ?", *filter.UserID)
	}

	if filter.Status != nil {
		query = query.Where("sk.status = ?", *filter.Status)
	}

	if filter.CreatedBy != nil {
		query = query.Where("sk.created_by = ?", *filter.CreatedBy)
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	query = query.Order("sk.created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, 0, err
	}

	keys := make([]*pkgmodels.ServiceKey, 0, len(modelList))
	for _, model := range modelList {
		keys = append(keys, model.ToServiceKeyDomain())
	}

	return keys, int64(count), nil
}

// Update updates a service key
func (r *ServiceKeyRepositoryImpl) Update(ctx context.Context, key *pkgmodels.ServiceKey) error {
	keyID, err := uuid.Parse(key.ID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	model := models.FromServiceKeyDomain(key)

	_, err = r.db.NewUpdate().
		Model(model).
		Column("name", "description", "expires_at", "updated_at").
		Where("id = ?", keyID).
		Exec(ctx)

	if err != nil {
		return err
	}

	key.UpdatedAt = time.Now()

	return nil
}

// Delete permanently deletes a service key
func (r *ServiceKeyRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.ServiceKeyModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// Revoke marks a service key as revoked
func (r *ServiceKeyRepositoryImpl) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()

	_, err := r.db.NewUpdate().
		Model((*models.ServiceKeyModel)(nil)).
		Set("status = ?", pkgmodels.ServiceKeyStatusRevoked).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// UpdateLastUsed updates the last used timestamp and increments usage count
func (r *ServiceKeyRepositoryImpl) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()

	_, err := r.db.NewUpdate().
		Model((*models.ServiceKeyModel)(nil)).
		Set("last_used_at = ?", now).
		Set("usage_count = usage_count + 1").
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// CountByUserID returns the number of service keys for a user
func (r *ServiceKeyRepositoryImpl) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := r.db.NewSelect().
		Model((*models.ServiceKeyModel)(nil)).
		Where("user_id = ?", userID).
		Count(ctx)

	return int64(count), err
}
