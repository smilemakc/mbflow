package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/internal/application/systemkey"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
)

type ServiceAPIAuditHandlers struct {
	auditService *systemkey.AuditService
	logger       *logger.Logger
}

func NewServiceAPIAuditHandlers(auditService *systemkey.AuditService, log *logger.Logger) *ServiceAPIAuditHandlers {
	return &ServiceAPIAuditHandlers{
		auditService: auditService,
		logger:       log,
	}
}

func (h *ServiceAPIAuditHandlers) ListAuditLog(c *gin.Context) {
	limit := getQueryInt(c, "limit", 50)
	offset := getQueryInt(c, "offset", 0)

	if limit > 100 {
		limit = 100
	}

	filter := repository.ServiceAuditLogFilter{
		Limit:  limit,
		Offset: offset,
	}

	if serviceName := c.Query("service_name"); serviceName != "" {
		filter.ServiceName = &serviceName
	}
	if action := c.Query("action"); action != "" {
		filter.Action = &action
	}
	if resourceType := c.Query("resource_type"); resourceType != "" {
		filter.ResourceType = &resourceType
	}
	if impersonatedUserID := c.Query("impersonated_user_id"); impersonatedUserID != "" {
		if parsed, err := uuid.Parse(impersonatedUserID); err == nil {
			filter.ImpersonatedUserID = &parsed
		}
	}
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
			filter.DateFrom = &parsed
		}
	}
	if dateToStr := c.Query("date_to"); dateToStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateToStr); err == nil {
			filter.DateTo = &parsed
		}
	}

	logs, total, err := h.auditService.ListLogs(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list audit logs", "error", err)
		respondError(c, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_logs": logs,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}
