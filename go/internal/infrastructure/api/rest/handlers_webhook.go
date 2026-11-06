package rest

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/smilemakc/mbflow/go/internal/application/trigger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
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
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Error("Failed to bind JSON in HandleWebhook", "error", err, "trigger_id", triggerID)
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	// Extract headers. They become part of the execution input, which is
	// stored and shown to anyone who can read the execution, so credentials
	// sent by the caller are masked.
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = webhookHeaderValue(key, values[0])
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

		switch {
		case strings.Contains(errorMsg, "not found"):
			statusCode = http.StatusNotFound
		case strings.Contains(errorMsg, "disabled"):
			statusCode = http.StatusForbidden
		case strings.Contains(errorMsg, "signature validation failed"):
			statusCode = http.StatusUnauthorized
		case strings.Contains(errorMsg, "IP not whitelisted"):
			statusCode = http.StatusForbidden
		case strings.Contains(errorMsg, "rate limit exceeded"):
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

	webhookTrigger, exists := h.webhookRegistry.GetWebhook(triggerID)
	if !exists {
		h.logger.Error("Webhook trigger not found", "trigger_id", triggerID)
		respondError(c, http.StatusNotFound, "webhook trigger not found")
		return
	}

	// Return webhook info (excluding sensitive data like secrets)
	webhookInfo := gin.H{
		"trigger_id":  webhookTrigger.ID,
		"workflow_id": webhookTrigger.WorkflowID,
		"enabled":     webhookTrigger.Enabled,
		"created_at":  webhookTrigger.CreatedAt,
		"updated_at":  webhookTrigger.UpdatedAt,
	}

	if webhookTrigger.LastRun != nil {
		webhookInfo["last_run"] = webhookTrigger.LastRun
	}

	// Include non-sensitive config
	config := make(map[string]any)
	if ipWhitelist, ok := webhookTrigger.Config["ip_whitelist"]; ok {
		config["ip_whitelist_enabled"] = true
		config["ip_whitelist"] = ipWhitelist
	}
	if _, ok := webhookTrigger.Config["secret"]; ok {
		config["signature_validation_enabled"] = true
	}

	webhookInfo["config"] = config

	c.JSON(http.StatusOK, webhookInfo)
}

// getSourceIP returns the client IP used for the trigger's IP whitelist.
//
// It relies on gin's ClientIP, which honors X-Forwarded-For / X-Real-IP only
// when the request comes from a configured trusted proxy. Reading those
// headers directly would let any client spoof a whitelisted address.
func getSourceIP(c *gin.Context) string {
	return c.ClientIP()
}

// webhookHeaderValue masks credential-bearing headers. The HMAC signature
// header is kept: the trigger verifies it and it reveals no secret.
func webhookHeaderValue(name, value string) string {
	lower := strings.ToLower(name)
	switch lower {
	case "x-webhook-signature":
		return value
	case "authorization", "proxy-authorization", "cookie", "set-cookie":
		return "***"
	}
	for _, marker := range []string{"token", "secret", "password", "api-key", "apikey", "auth"} {
		if strings.Contains(lower, marker) {
			return "***"
		}
	}
	return value
}
