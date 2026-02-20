package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
)

type ServiceAPIAuditHandlers struct {
	ops *serviceapi.Operations
}

func NewServiceAPIAuditHandlers(ops *serviceapi.Operations) *ServiceAPIAuditHandlers {
	return &ServiceAPIAuditHandlers{ops: ops}
}

func (h *ServiceAPIAuditHandlers) ListAuditLog(c *gin.Context) {
	params := serviceapi.ListAuditLogParams{
		Limit:  getQueryInt(c, "limit", 50),
		Offset: getQueryInt(c, "offset", 0),
	}

	if s := c.Query("service_name"); s != "" {
		params.ServiceName = &s
	}
	if a := c.Query("action"); a != "" {
		params.Action = &a
	}
	if rt := c.Query("resource_type"); rt != "" {
		params.ResourceType = &rt
	}
	if iuid := c.Query("impersonated_user_id"); iuid != "" {
		if parsed, err := uuid.Parse(iuid); err == nil {
			params.ImpersonatedUserID = &parsed
		}
	}
	if df := c.Query("date_from"); df != "" {
		if parsed, err := time.Parse(time.RFC3339, df); err == nil {
			params.DateFrom = &parsed
		}
	}
	if dt := c.Query("date_to"); dt != "" {
		if parsed, err := time.Parse(time.RFC3339, dt); err == nil {
			params.DateTo = &parsed
		}
	}

	result, err := h.ops.ListAuditLog(c.Request.Context(), params)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	respondList(c, http.StatusOK, result.AuditLogs, int(result.Total), params.Limit, params.Offset)
}
