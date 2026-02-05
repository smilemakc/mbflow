package storage

import (
	"context"
	"time"

	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

var _ repository.ServiceAuditLogRepository = (*ServiceAuditLogRepoImpl)(nil)

// ServiceAuditLogRepoImpl implements the ServiceAuditLogRepository interface
type ServiceAuditLogRepoImpl struct {
	db *bun.DB
}

// NewServiceAuditLogRepo creates a new ServiceAuditLogRepoImpl
func NewServiceAuditLogRepo(db *bun.DB) *ServiceAuditLogRepoImpl {
	return &ServiceAuditLogRepoImpl{db: db}
}

// Create creates a new audit log entry
func (r *ServiceAuditLogRepoImpl) Create(ctx context.Context, log *pkgmodels.ServiceAuditLog) error {
	model := models.FromServiceAuditLogDomain(log)

	_, err := r.db.NewInsert().
		Model(model).
		Exec(ctx)

	if err != nil {
		return err
	}

	log.ID = model.ID.String()
	log.CreatedAt = model.CreatedAt

	return nil
}

// FindAll returns all audit logs with optional filters
func (r *ServiceAuditLogRepoImpl) FindAll(ctx context.Context, filter repository.ServiceAuditLogFilter) ([]*pkgmodels.ServiceAuditLog, int64, error) {
	var modelList []*models.ServiceAuditLogModel

	query := r.db.NewSelect().
		Model(&modelList)

	if filter.SystemKeyID != nil {
		query = query.Where("sal.system_key_id = ?", *filter.SystemKeyID)
	}

	if filter.ServiceName != nil {
		query = query.Where("sal.service_name = ?", *filter.ServiceName)
	}

	if filter.Action != nil {
		query = query.Where("sal.action = ?", *filter.Action)
	}

	if filter.ResourceType != nil {
		query = query.Where("sal.resource_type = ?", *filter.ResourceType)
	}

	if filter.ImpersonatedUserID != nil {
		query = query.Where("sal.impersonated_user_id = ?", *filter.ImpersonatedUserID)
	}

	if filter.DateFrom != nil {
		query = query.Where("sal.created_at >= ?", *filter.DateFrom)
	}

	if filter.DateTo != nil {
		query = query.Where("sal.created_at <= ?", *filter.DateTo)
	}

	count, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	query = query.Order("sal.created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, 0, err
	}

	logs := make([]*pkgmodels.ServiceAuditLog, 0, len(modelList))
	for _, model := range modelList {
		logs = append(logs, model.ToServiceAuditLogDomain())
	}

	return logs, int64(count), nil
}

// DeleteOlderThan deletes audit logs older than the specified time
func (r *ServiceAuditLogRepoImpl) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.NewDelete().
		Model((*models.ServiceAuditLogModel)(nil)).
		Where("created_at < ?", before).
		Exec(ctx)

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
