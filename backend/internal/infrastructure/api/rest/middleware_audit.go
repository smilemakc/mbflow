package rest

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/smilemakc/mbflow/internal/application/systemkey"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
)

type AuditMiddleware struct {
	auditService *systemkey.AuditService
	logger       *logger.Logger
}

func NewAuditMiddleware(auditService *systemkey.AuditService, log *logger.Logger) *AuditMiddleware {
	return &AuditMiddleware{
		auditService: auditService,
		logger:       log,
	}
}

func (m *AuditMiddleware) RecordAction() gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				requestBody = string(bodyBytes)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		c.Next()

		systemKeyIDVal, exists := c.Get(ContextKeySystemKeyID)
		if !exists {
			return
		}
		systemKeyID := systemKeyIDVal.(string)

		serviceNameVal, _ := c.Get(ContextKeyServiceName)
		serviceName := serviceNameVal.(string)

		var impersonatedUserID *string
		if isImpersonated, exists := c.Get(ContextKeyImpersonated); exists && isImpersonated.(bool) {
			if userIDVal, exists := c.Get(ContextKeyUserID); exists {
				userID := userIDVal.(string)
				impersonatedUserID = &userID
			}
		}

		action, resourceType, resourceID := parseServiceAPIPath(c.Request.URL.Path, c.Request.Method)

		var body *string
		if requestBody != "" {
			body = &requestBody
		}

		clientIP := c.ClientIP()
		status := c.Writer.Status()

		go func() {
			if err := m.auditService.LogAction(
				c.Request.Context(),
				systemKeyID,
				serviceName,
				action,
				resourceType,
				resourceID,
				impersonatedUserID,
				c.Request.Method,
				c.Request.URL.Path,
				body,
				clientIP,
				status,
			); err != nil {
				m.logger.Error("Failed to log audit action", "error", err, "action", action, "resource_type", resourceType)
			}
		}()
	}
}

func parseServiceAPIPath(path, method string) (action, resourceType string, resourceID *string) {
	trimmed := strings.TrimPrefix(path, "/api/v1/service/")
	parts := strings.SplitN(trimmed, "/", 3)

	if len(parts) == 0 {
		return "unknown", "unknown", nil
	}

	resourceType = strings.TrimSuffix(parts[0], "s")

	resourceType = strings.ReplaceAll(resourceType, "-", "_")

	switch method {
	case http.MethodGet:
		if len(parts) >= 2 && parts[1] != "" {
			action = resourceType + ".get"
		} else {
			action = resourceType + ".list"
		}
	case http.MethodPost:
		if len(parts) >= 3 {
			action = resourceType + "." + parts[2]
		} else {
			action = resourceType + ".create"
		}
	case http.MethodPut:
		action = resourceType + ".update"
	case http.MethodDelete:
		action = resourceType + ".delete"
	default:
		action = resourceType + "." + strings.ToLower(method)
	}

	if len(parts) >= 2 && parts[1] != "" {
		if len(parts[1]) >= 32 {
			resourceID = &parts[1]
		}
	}

	return action, resourceType, resourceID
}
