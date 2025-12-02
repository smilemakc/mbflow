package rest

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/smilemakc/mbflow/internal/application/trigger"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
)

// WebhookHandlers provides HTTP handlers for webhook trigger endpoints
type WebhookHandlers struct {
	webhookRegistry *trigger.WebhookRegistry
	logger          *logger.Logger
}

// NewWebhookHandlers creates a new WebhookHandlers instance
func NewWebhookHandlers(webhookRegistry *trigger.WebhookRegistry, log *logger.Logger) *WebhookHandlers {
	return &WebhookHandlers{
		webhookRegistry: webhookRegistry,
		logger:          log,
	}
}

// HandleWebhook handles POST /api/v1/webhooks/{trigger_id}
func (h *WebhookHandlers) HandleWebhook(c *gin.Context) {
	triggerID := c.Param("trigger_id")
	if triggerID == "" {
		respondError(c, http.StatusBadRequest, "trigger_id is required")
		return
	}

	// Parse request body as JSON
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Error("Failed to bind JSON in HandleWebhook", "error", err, "trigger_id", triggerID)
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Extract headers
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Get source IP
	sourceIP := getSourceIP(c)

	// Execute webhook
	executionID, err := h.webhookRegistry.ExecuteWebhook(
		c.Request.Context(),
		triggerID,
		payload,
		headers,
		sourceIP,
	)
	if err != nil {
		// Determine appropriate status code
		statusCode := http.StatusInternalServerError
		errorMsg := err.Error()

		if strings.Contains(errorMsg, "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(errorMsg, "disabled") {
			statusCode = http.StatusForbidden
		} else if strings.Contains(errorMsg, "signature validation failed") {
			statusCode = http.StatusUnauthorized
		} else if strings.Contains(errorMsg, "IP not whitelisted") {
			statusCode = http.StatusForbidden
		} else if strings.Contains(errorMsg, "rate limit exceeded") {
			statusCode = http.StatusTooManyRequests
		}

		h.logger.Error("Failed to execute webhook", "error", err, "trigger_id", triggerID, "source_ip", sourceIP, "status_code", statusCode)
		respondError(c, statusCode, errorMsg)
		return
	}

	// Return 202 Accepted with execution ID
	c.JSON(http.StatusAccepted, gin.H{
		"execution_id": executionID,
		"message":      "workflow execution started",
	})
}

// HandleWebhookGet handles GET /api/v1/webhooks/{trigger_id}
// Returns webhook configuration and status
func (h *WebhookHandlers) HandleWebhookGet(c *gin.Context) {
	triggerID := c.Param("trigger_id")
	if triggerID == "" {
		respondError(c, http.StatusBadRequest, "trigger_id is required")
		return
	}

	trigger, exists := h.webhookRegistry.GetWebhook(triggerID)
	if !exists {
		h.logger.Error("Webhook trigger not found", "trigger_id", triggerID)
		respondError(c, http.StatusNotFound, "webhook trigger not found")
		return
	}

	// Return webhook info (excluding sensitive data like secrets)
	webhookInfo := gin.H{
		"trigger_id":  trigger.ID,
		"workflow_id": trigger.WorkflowID,
		"enabled":     trigger.Enabled,
		"created_at":  trigger.CreatedAt,
		"updated_at":  trigger.UpdatedAt,
	}

	if trigger.LastRun != nil {
		webhookInfo["last_run"] = trigger.LastRun
	}

	// Include non-sensitive config
	config := make(map[string]interface{})
	if ipWhitelist, ok := trigger.Config["ip_whitelist"]; ok {
		config["ip_whitelist_enabled"] = true
		config["ip_whitelist"] = ipWhitelist
	}
	if _, ok := trigger.Config["secret"]; ok {
		config["signature_validation_enabled"] = true
	}

	webhookInfo["config"] = config

	c.JSON(http.StatusOK, webhookInfo)
}

// getSourceIP extracts the client IP address from the request
func getSourceIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := c.ClientIP()
	return ip
}
