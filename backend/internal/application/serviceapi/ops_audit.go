package serviceapi

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/pkg/models"
)

// ListAuditLogParams contains parameters for listing audit logs.
type ListAuditLogParams struct {
	Limit              int
	Offset             int
	ServiceName        *string
	Action             *string
	ResourceType       *string
	ImpersonatedUserID *uuid.UUID
	DateFrom           *time.Time
	DateTo             *time.Time
}

// ListAuditLogResult contains the result of listing audit logs.
type ListAuditLogResult struct {
	AuditLogs []*models.ServiceAuditLog
	Total     int64
}

func (o *Operations) ListAuditLog(ctx context.Context, params ListAuditLogParams) (*ListAuditLogResult, error) {
	limit := params.Limit
	if limit > 100 {
		limit = 100
	}

	filter := repository.ServiceAuditLogFilter{
		Limit:              limit,
		Offset:             params.Offset,
		ServiceName:        params.ServiceName,
		Action:             params.Action,
		ResourceType:       params.ResourceType,
		ImpersonatedUserID: params.ImpersonatedUserID,
		DateFrom:           params.DateFrom,
		DateTo:             params.DateTo,
	}

	logs, total, err := o.AuditService.ListLogs(ctx, filter)
	if err != nil {
		o.Logger.Error("Failed to list audit logs", "error", err)
		return nil, err
	}

	return &ListAuditLogResult{
		AuditLogs: logs,
		Total:     total,
	}, nil
}
