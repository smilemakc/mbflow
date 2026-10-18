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

var _ repository.APIClientRepository = (*APIClientRepositoryImpl)(nil)

// APIClientRepositoryImpl implements repository.APIClientRepository.
type APIClientRepositoryImpl struct {
	db bun.IDB
}

// NewAPIClientRepository creates a new APIClientRepositoryImpl.
func NewAPIClientRepository(db bun.IDB) *APIClientRepositoryImpl {
	return &APIClientRepositoryImpl{db: db}
}

// Settings columns are written on every update, specification columns only with a new document.
var (
	apiClientSettingsColumns = []string{"name", "description", "base_url", "auth", "default_headers"}
	apiClientSpecColumns     = []string{
		"title", "api_version", "spec_version", "servers", "security_schemes",
		"operations", "schemas", "warnings", "operation_count", "spec_raw", "spec_size",
	}
)

// Create stores a new API client.
func (r *APIClientRepositoryImpl) Create(ctx context.Context, client *pkgmodels.APIClient) error {
	model, err := models.FromAPIClientDomain(client)
	if err != nil {
		return err
	}
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now

	if _, err := r.db.NewInsert().Model(model).Exec(ctx); err != nil {
		return err
	}

	client.ID = model.ID.String()
	client.OperationCount = model.OperationCount
	client.CreatedAt = model.CreatedAt
	client.UpdatedAt = model.UpdatedAt
	return nil
}

// Update stores the client settings and, when SpecRaw is set, the specification.
func (r *APIClientRepositoryImpl) Update(ctx context.Context, client *pkgmodels.APIClient) error {
	model, err := models.FromAPIClientDomain(client)
	if err != nil {
		return err
	}
	if model.ID == uuid.Nil {
		return pkgmodels.ErrInvalidID
	}
	model.UpdatedAt = time.Now()

	columns := append([]string{"updated_at"}, apiClientSettingsColumns...)
	if client.SpecRaw != "" {
		columns = append(columns, apiClientSpecColumns...)
	}

	res, err := r.db.NewUpdate().
		Model(model).
		Column(columns...).
		Where("ac.id = ? AND ac.deleted_at IS NULL", model.ID).
		Exec(ctx)
	if err != nil {
		return err
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return pkgmodels.ErrAPIClientNotFound
	}

	client.UpdatedAt = model.UpdatedAt
	if client.SpecRaw != "" {
		client.OperationCount = model.OperationCount
	}
	return nil
}

// Delete marks the API client as deleted.
func (r *APIClientRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.NewUpdate().
		Model((*models.APIClientModel)(nil)).
		Set("deleted_at = ?", time.Now()).
		Where("ac.id = ? AND ac.deleted_at IS NULL", id).
		Exec(ctx)
	if err != nil {
		return err
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return pkgmodels.ErrAPIClientNotFound
	}
	return nil
}

// FindByID returns the API client without the original document.
func (r *APIClientRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*pkgmodels.APIClient, error) {
	model := new(models.APIClientModel)
	err := r.db.NewSelect().
		Model(model).
		ExcludeColumn("spec_raw").
		Where("ac.id = ? AND ac.deleted_at IS NULL", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgmodels.ErrAPIClientNotFound
		}
		return nil, err
	}
	return model.ToAPIClientDomain(), nil
}

// FindSpec returns the original document of the API client.
func (r *APIClientRepositoryImpl) FindSpec(ctx context.Context, id uuid.UUID) (string, error) {
	var spec string
	err := r.db.NewSelect().
		Model((*models.APIClientModel)(nil)).
		Column("spec_raw").
		Where("ac.id = ? AND ac.deleted_at IS NULL", id).
		Scan(ctx, &spec)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", pkgmodels.ErrAPIClientNotFound
		}
		return "", err
	}
	return spec, nil
}

// FindUpdatedAt returns the last modification time of the API client.
func (r *APIClientRepositoryImpl) FindUpdatedAt(ctx context.Context, id uuid.UUID) (time.Time, error) {
	var updatedAt time.Time
	err := r.db.NewSelect().
		Model((*models.APIClientModel)(nil)).
		Column("updated_at").
		Where("ac.id = ? AND ac.deleted_at IS NULL", id).
		Scan(ctx, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, pkgmodels.ErrAPIClientNotFound
		}
		return time.Time{}, err
	}
	return updatedAt, nil
}

// FindByOwner lists the API clients of a user without the heavy columns.
func (r *APIClientRepositoryImpl) FindByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*pkgmodels.APIClient, int64, error) {
	var list []*models.APIClientModel
	query := r.db.NewSelect().
		Model(&list).
		ExcludeColumn("operations", "schemas", "spec_raw").
		Where("ac.owner_id = ? AND ac.deleted_at IS NULL", ownerID)

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	query = query.Order("ac.created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, 0, err
	}

	clients := make([]*pkgmodels.APIClient, 0, len(list))
	for _, model := range list {
		clients = append(clients, model.ToAPIClientDomain())
	}
	return clients, int64(total), nil
}
