package trigger

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// WebhookRegistry manages webhook triggers
type WebhookRegistry struct {
	triggerRepo  repository.TriggerRepository
	workflowRepo repository.WorkflowRepository
	executionMgr *engine.ExecutionManager
	cache        *cache.RedisCache

	webhooks map[string]*models.Trigger // triggerID -> trigger
	mu       sync.RWMutex
}

// WebhookRegistryConfig holds configuration for webhook registry
type WebhookRegistryConfig struct {
	TriggerRepo  repository.TriggerRepository
	WorkflowRepo repository.WorkflowRepository
	ExecutionMgr *engine.ExecutionManager
	Cache        *cache.RedisCache
}

// NewWebhookRegistry creates a new webhook registry
func NewWebhookRegistry(cfg WebhookRegistryConfig) *WebhookRegistry {
	return &WebhookRegistry{
		triggerRepo:  cfg.TriggerRepo,
		workflowRepo: cfg.WorkflowRepo,
		executionMgr: cfg.ExecutionMgr,
		cache:        cfg.Cache,
		webhooks:     make(map[string]*models.Trigger),
	}
}

// RegisterAll registers all webhook triggers
func (wr *WebhookRegistry) RegisterAll(ctx context.Context, triggers []*storagemodels.TriggerModel) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	for _, trigger := range triggers {
		if trigger.Type == string(models.TriggerTypeWebhook) {
			domainTrigger := wr.modelToDomain(trigger)
			wr.webhooks[domainTrigger.ID] = domainTrigger
		}
	}

	return nil
}

// RegisterWebhook registers a new webhook trigger
func (wr *WebhookRegistry) RegisterWebhook(ctx context.Context, trigger *models.Trigger) error {
	if trigger.Type != models.TriggerTypeWebhook {
		return nil // Not a webhook trigger
	}

	wr.mu.Lock()
	defer wr.mu.Unlock()

	wr.webhooks[trigger.ID] = trigger
	return nil
}

// UnregisterWebhook unregisters a webhook trigger
func (wr *WebhookRegistry) UnregisterWebhook(ctx context.Context, triggerID string) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	delete(wr.webhooks, triggerID)
	return nil
}

// GetWebhook retrieves a webhook trigger by ID
func (wr *WebhookRegistry) GetWebhook(triggerID string) (*models.Trigger, bool) {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	trigger, exists := wr.webhooks[triggerID]
	return trigger, exists
}

// ExecuteWebhook executes a workflow triggered by a webhook
func (wr *WebhookRegistry) ExecuteWebhook(ctx context.Context, triggerID string, payload map[string]any, headers map[string]string, sourceIP string) (string, error) {
	// Get trigger
	trigger, exists := wr.GetWebhook(triggerID)
	if !exists {
		return "", fmt.Errorf("webhook trigger not found")
	}

	if !trigger.Enabled {
		return "", fmt.Errorf("webhook trigger is disabled")
	}

	// Validate signature if secret is configured
	if err := wr.validateSignature(trigger, payload, headers); err != nil {
		return "", fmt.Errorf("signature validation failed: %w", err)
	}

	// Check IP whitelist
	if err := wr.checkIPWhitelist(trigger, sourceIP); err != nil {
		return "", fmt.Errorf("IP not whitelisted: %w", err)
	}

	// Check rate limit
	if err := wr.checkRateLimit(ctx, triggerID); err != nil {
		return "", fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Merge trigger input with payload
	input := make(map[string]any)

	// First add trigger's default input
	if defaultInput, ok := trigger.Config["input"].(map[string]any); ok {
		for k, v := range defaultInput {
			input[k] = v
		}
	}

	// Then add payload (overrides trigger input)
	for k, v := range payload {
		input[k] = v
	}

	// Add webhook metadata
	input["_webhook"] = map[string]any{
		"trigger_id": triggerID,
		"headers":    headers,
		"source_ip":  sourceIP,
		"timestamp":  time.Now().Unix(),
	}

	// Execute workflow
	execution, err := wr.executionMgr.Execute(ctx, trigger.WorkflowID, input, nil)
	if err != nil {
		return "", fmt.Errorf("failed to execute workflow: %w", err)
	}

	// Update trigger state
	state, err := LoadTriggerState(ctx, wr.cache, triggerID)
	if err != nil {
		state = NewTriggerState(triggerID)
	}
	state.MarkExecuted()

	if err := state.Save(ctx, wr.cache); err != nil {
		fmt.Printf("failed to save trigger state: %v\n", err)
	}

	// Update last triggered timestamp in database
	triggerUUID, _ := uuid.Parse(triggerID)
	if err := wr.triggerRepo.MarkTriggered(ctx, triggerUUID); err != nil {
		fmt.Printf("failed to mark trigger as triggered: %v\n", err)
	}

	return execution.ID, nil
}

// validateSignature validates HMAC signature if configured
func (wr *WebhookRegistry) validateSignature(trigger *models.Trigger, payload map[string]any, headers map[string]string) error {
	secret, ok := trigger.Config["secret"].(string)
	if !ok || secret == "" {
		return nil // No signature validation required
	}

	// Get signature from header
	signature := headers["X-Webhook-Signature"]
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	// Compute expected signature
	expectedSignature := wr.computeSignature(secret, payload)

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// computeSignature computes HMAC-SHA256 signature
func (wr *WebhookRegistry) computeSignature(secret string, payload map[string]any) string {
	// Convert payload to JSON string (in production, use actual request body)
	// This is a simplified version
	payloadStr := fmt.Sprintf("%v", payload)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payloadStr))
	return hex.EncodeToString(h.Sum(nil))
}

// checkIPWhitelist checks if source IP is whitelisted
func (wr *WebhookRegistry) checkIPWhitelist(trigger *models.Trigger, sourceIP string) error {
	whitelist, ok := trigger.Config["ip_whitelist"].([]any)
	if !ok || len(whitelist) == 0 {
		return nil // No whitelist configured
	}

	// Parse source IP
	ip := net.ParseIP(sourceIP)
	if ip == nil {
		return fmt.Errorf("invalid source IP: %s", sourceIP)
	}

	// Check against whitelist
	for _, allowedIP := range whitelist {
		allowedStr, ok := allowedIP.(string)
		if !ok {
			continue
		}

		// Check if it's a CIDR range
		if _, ipNet, err := net.ParseCIDR(allowedStr); err == nil {
			if ipNet.Contains(ip) {
				return nil
			}
		} else {
			// Check exact IP match
			if sourceIP == allowedStr {
				return nil
			}
		}
	}

	return fmt.Errorf("IP %s not in whitelist", sourceIP)
}

// checkRateLimit checks if trigger has exceeded rate limit
func (wr *WebhookRegistry) checkRateLimit(ctx context.Context, triggerID string) error {
	// Get rate limit configuration
	// For now, use a simple fixed rate limit: 100 requests per minute
	key := fmt.Sprintf("trigger:%s:ratelimit", triggerID)

	// Increment counter
	count, err := wr.cache.Increment(ctx, key)
	if err != nil {
		// If error, allow request (fail open)
		return nil
	}

	// Set expiration on first increment
	if count == 1 {
		if err := wr.cache.Expire(ctx, key, time.Minute); err != nil {
			fmt.Printf("failed to set rate limit expiration: %v\n", err)
		}
	}

	// Check limit (100 per minute)
	if count > 100 {
		return fmt.Errorf("rate limit exceeded: %d requests in last minute", count)
	}

	return nil
}

// modelToDomain converts storage model to domain model
func (wr *WebhookRegistry) modelToDomain(tm *storagemodels.TriggerModel) *models.Trigger {
	trigger := &models.Trigger{
		ID:         tm.ID.String(),
		WorkflowID: tm.WorkflowID.String(),
		Type:       models.TriggerType(tm.Type),
		Config:     make(map[string]any),
		Enabled:    tm.Enabled,
		CreatedAt:  tm.CreatedAt,
		UpdatedAt:  tm.UpdatedAt,
	}

	if tm.Config != nil {
		trigger.Config = map[string]any(tm.Config)
	}

	if tm.LastTriggeredAt != nil {
		trigger.LastRun = tm.LastTriggeredAt
	}

	return trigger
}
