package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

var _ repository.SystemKeyRepository = (*SystemKeyRepoImpl)(nil)

// SystemKeyRepoImpl implements the SystemKeyRepository interface
type SystemKeyRepoImpl struct {
	db bun.IDB
}

// NewSystemKeyRepo creates a new SystemKeyRepoImpl
func NewSystemKeyRepo(db bun.IDB) *SystemKeyRepoImpl {
	return &SystemKeyRepoImpl{db: db}
}

// Create creates a new system key
func (r *SystemKeyRepoImpl) Create(ctx context.Context, key *pkgmodels.SystemKey) error {
	model := models.FromSystemKeyDomain(key)

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

// FindByID finds a system key by ID
func (r *SystemKeyRepoImpl) FindByID(ctx context.Context, id uuid.UUID) (*pkgmodels.SystemKey, error) {
	model := new(models.SystemKeyModel)

	err := r.db.NewSelect().
		Model(model).
		Where("syk.id = ?", id).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgmodels.ErrSystemKeyNotFound
		}
		return nil, err
	}

	return model.ToSystemKeyDomain(), nil
}

// FindByPrefix finds a system key by its prefix
func (r *SystemKeyRepoImpl) FindByPrefix(ctx context.Context, prefix string) ([]*pkgmodels.SystemKey, error) {
	var modelList []*models.SystemKeyModel

	err := r.db.NewSelect().
		Model(&modelList).
		Where("syk.key_prefix = ?", prefix).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	keys := make([]*pkgmodels.SystemKey, 0, len(modelList))
	for _, model := range modelList {
		keys = append(keys, model.ToSystemKeyDomain())
	}

	return keys, nil
}

// FindAll returns all system keys with optional filters
func (r *SystemKeyRepoImpl) FindAll(ctx context.Context, filter repository.SystemKeyFilter) ([]*pkgmodels.SystemKey, int64, error) {
	var modelList []*models.SystemKeyModel

	query := r.db.NewSelect().
		Model(&modelList)

	if filter.ServiceName != nil {
		query = query.Where("syk.service_name = ?", *filter.ServiceName)
	}

	if filter.Status != nil {
		query = query.Where("syk.status = ?", *filter.Status)
	}

	if filter.CreatedBy != nil {
		query = query.Where("syk.created_by = ?", *filter.CreatedBy)
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	query = query.Order("syk.created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, 0, err
	}

	keys := make([]*pkgmodels.SystemKey, 0, len(modelList))
	for _, model := range modelList {
		keys = append(keys, model.ToSystemKeyDomain())
	}

	return keys, int64(count), nil
}

// Update updates a system key
func (r *SystemKeyRepoImpl) Update(ctx context.Context, key *pkgmodels.SystemKey) error {
	keyID, err := uuid.Parse(key.ID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	model := models.FromSystemKeyDomain(key)

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

// Delete permanently deletes a system key
func (r *SystemKeyRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.SystemKeyModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// Revoke marks a system key as revoked
func (r *SystemKeyRepoImpl) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()

	_, err := r.db.NewUpdate().
		Model((*models.SystemKeyModel)(nil)).
		Set("status = ?", pkgmodels.SystemKeyStatusRevoked).
		Set("revoked_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// UpdateLastUsed updates the last used timestamp and increments usage count
func (r *SystemKeyRepoImpl) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()

	_, err := r.db.NewUpdate().
		Model((*models.SystemKeyModel)(nil)).
		Set("last_used_at = ?", now).
		Set("usage_count = usage_count + 1").
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// Count returns the total number of system keys
func (r *SystemKeyRepoImpl) Count(ctx context.Context) (int64, error) {
	count, err := r.db.NewSelect().
		Model((*models.SystemKeyModel)(nil)).
		Count(ctx)

	return int64(count), err
}
