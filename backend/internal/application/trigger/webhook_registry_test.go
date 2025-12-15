package trigger

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookRegistry_ComputeSignature(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	payload := map[string]interface{}{
		"user_id": "123",
		"action":  "created",
	}

	secret := "test-secret"

	// Compute signature
	signature1 := wr.computeSignature(secret, payload)
	assert.NotEmpty(t, signature1)

	// Same payload should produce same signature
	signature2 := wr.computeSignature(secret, payload)
	assert.Equal(t, signature1, signature2)

	// Different secret should produce different signature
	signature3 := wr.computeSignature("different-secret", payload)
	assert.NotEqual(t, signature1, signature3)
}

func TestWebhookRegistry_CheckIPWhitelist(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	tests := []struct {
		name        string
		trigger     *models.Trigger
		sourceIP    string
		expectError bool
	}{
		{
			name: "no whitelist - allow all",
			trigger: &models.Trigger{
				Type:   models.TriggerTypeWebhook,
				Config: map[string]interface{}{},
			},
			sourceIP:    "192.168.1.100",
			expectError: false,
		},
		{
			name: "exact IP match",
			trigger: &models.Trigger{
				Type: models.TriggerTypeWebhook,
				Config: map[string]interface{}{
					"ip_whitelist": []interface{}{
						"192.168.1.100",
						"10.0.0.1",
					},
				},
			},
			sourceIP:    "192.168.1.100",
			expectError: false,
		},
		{
			name: "exact IP no match",
			trigger: &models.Trigger{
				Type: models.TriggerTypeWebhook,
				Config: map[string]interface{}{
					"ip_whitelist": []interface{}{
						"192.168.1.100",
						"10.0.0.1",
					},
				},
			},
			sourceIP:    "192.168.1.200",
			expectError: true,
		},
		{
			name: "CIDR range match",
			trigger: &models.Trigger{
				Type: models.TriggerTypeWebhook,
				Config: map[string]interface{}{
					"ip_whitelist": []interface{}{
						"192.168.1.0/24",
					},
				},
			},
			sourceIP:    "192.168.1.150",
			expectError: false,
		},
		{
			name: "CIDR range no match",
			trigger: &models.Trigger{
				Type: models.TriggerTypeWebhook,
				Config: map[string]interface{}{
					"ip_whitelist": []interface{}{
						"192.168.1.0/24",
					},
				},
			},
			sourceIP:    "192.168.2.100",
			expectError: true,
		},
		{
			name: "mixed exact and CIDR",
			trigger: &models.Trigger{
				Type: models.TriggerTypeWebhook,
				Config: map[string]interface{}{
					"ip_whitelist": []interface{}{
						"10.0.0.1",
						"192.168.1.0/24",
					},
				},
			},
			sourceIP:    "10.0.0.1",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wr.checkIPWhitelist(tt.trigger, tt.sourceIP)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhookRegistry_RegisterUnregister(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	trigger := &models.Trigger{
		ID:         "webhook-1",
		WorkflowID: "workflow-1",
		Type:       models.TriggerTypeWebhook,
		Config: map[string]interface{}{
			"secret": "test-secret",
		},
		Enabled: true,
	}

	// Register webhook
	err := wr.RegisterWebhook(nil, trigger)
	require.NoError(t, err)

	// Verify webhook was registered
	retrieved, exists := wr.GetWebhook(trigger.ID)
	assert.True(t, exists)
	assert.Equal(t, trigger.ID, retrieved.ID)

	// Unregister webhook
	err = wr.UnregisterWebhook(nil, trigger.ID)
	require.NoError(t, err)

	// Verify webhook was unregistered
	_, exists = wr.GetWebhook(trigger.ID)
	assert.False(t, exists)
}

func TestWebhookRegistry_ValidateSignature(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	payload := map[string]interface{}{
		"data": "test",
	}

	tests := []struct {
		name        string
		trigger     *models.Trigger
		headers     map[string]string
		expectError bool
	}{
		{
			name: "no secret - skip validation",
			trigger: &models.Trigger{
				Type:   models.TriggerTypeWebhook,
				Config: map[string]interface{}{},
			},
			headers:     map[string]string{},
			expectError: false,
		},
		{
			name: "secret configured but no header",
			trigger: &models.Trigger{
				Type: models.TriggerTypeWebhook,
				Config: map[string]interface{}{
					"secret": "test-secret",
				},
			},
			headers:     map[string]string{},
			expectError: true,
		},
		{
			name: "valid signature",
			trigger: &models.Trigger{
				Type: models.TriggerTypeWebhook,
				Config: map[string]interface{}{
					"secret": "test-secret",
				},
			},
			headers: map[string]string{
				"X-Webhook-Signature": wr.computeSignature("test-secret", payload),
			},
			expectError: false,
		},
		{
			name: "invalid signature",
			trigger: &models.Trigger{
				Type: models.TriggerTypeWebhook,
				Config: map[string]interface{}{
					"secret": "test-secret",
				},
			},
			headers: map[string]string{
				"X-Webhook-Signature": "invalid-signature",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wr.validateSignature(tt.trigger, payload, tt.headers)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhookRegistry_ExecuteWebhook(t *testing.T) {
	t.Skip("Requires full integration test with execution manager and Redis")

	// This test would verify:
	// 1. Signature validation
	// 2. IP whitelist checking
	// 3. Rate limiting
	// 4. Workflow execution
	// 5. Trigger state updates
}

// TestWebhookRegistry_RegisterAll tests registering multiple webhooks
func TestWebhookRegistry_RegisterAll(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	triggers := []*storagemodels.TriggerModel{
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeWebhook),
			Config: map[string]interface{}{
				"path": "/webhook/1",
			},
			Enabled: true,
		},
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeWebhook),
			Config: map[string]interface{}{
				"path": "/webhook/2",
			},
			Enabled: true,
		},
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeCron), // Should be ignored
			Config: map[string]interface{}{
				"schedule": "0 0 * * * *",
			},
			Enabled: true,
		},
	}

	err := wr.RegisterAll(nil, triggers)
	require.NoError(t, err)

	// Verify only webhook triggers were registered
	wr.mu.RLock()
	numWebhooks := len(wr.webhooks)
	wr.mu.RUnlock()
	assert.Equal(t, 2, numWebhooks, "only webhook triggers should be registered")
}

// TestWebhookRegistry_RegisterNonWebhookTrigger tests that non-webhook triggers are ignored
func TestWebhookRegistry_RegisterNonWebhookTrigger(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	cronTrigger := &models.Trigger{
		ID:         "cron-1",
		WorkflowID: "wf-1",
		Type:       models.TriggerTypeCron,
		Config: map[string]interface{}{
			"schedule": "0 0 * * * *",
		},
		Enabled: true,
	}

	err := wr.RegisterWebhook(nil, cronTrigger)
	assert.NoError(t, err)

	// Verify webhook was not registered
	_, exists := wr.GetWebhook(cronTrigger.ID)
	assert.False(t, exists, "non-webhook trigger should not be registered")
}

// TestWebhookRegistry_GetWebhookNotFound tests getting non-existent webhook
func TestWebhookRegistry_GetWebhookNotFound(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	_, exists := wr.GetWebhook("non-existent-id")
	assert.False(t, exists)
}

// TestWebhookRegistry_ConcurrentOperations tests concurrent register/unregister
func TestWebhookRegistry_ConcurrentOperations(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	const numGoroutines = 10
	const webhooksPerGoroutine = 5

	// Register webhooks concurrently
	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < webhooksPerGoroutine; j++ {
				trigger := &models.Trigger{
					ID:         fmt.Sprintf("webhook-%d-%d", goroutineID, j),
					WorkflowID: fmt.Sprintf("wf-%d-%d", goroutineID, j),
					Type:       models.TriggerTypeWebhook,
					Config: map[string]interface{}{
						"path": fmt.Sprintf("/webhook/%d/%d", goroutineID, j),
					},
					Enabled: true,
				}
				_ = wr.RegisterWebhook(nil, trigger)
			}
		}(i)
	}
	wg.Wait()

	// Verify webhooks were registered
	wr.mu.RLock()
	totalWebhooks := len(wr.webhooks)
	wr.mu.RUnlock()
	assert.Equal(t, numGoroutines*webhooksPerGoroutine, totalWebhooks)

	// Unregister webhooks concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < webhooksPerGoroutine; j++ {
				triggerID := fmt.Sprintf("webhook-%d-%d", goroutineID, j)
				_ = wr.UnregisterWebhook(nil, triggerID)
			}
		}(i)
	}
	wg.Wait()

	// Verify all webhooks were unregistered
	wr.mu.RLock()
	remainingWebhooks := len(wr.webhooks)
	wr.mu.RUnlock()
	assert.Equal(t, 0, remainingWebhooks)
}

// TestWebhookRegistry_DuplicateRegistration tests registering same webhook multiple times
func TestWebhookRegistry_DuplicateRegistration(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	triggerID := "webhook-dup"

	trigger1 := &models.Trigger{
		ID:         triggerID,
		WorkflowID: "wf-1",
		Type:       models.TriggerTypeWebhook,
		Config: map[string]interface{}{
			"path": "/webhook/original",
		},
		Enabled: true,
	}

	trigger2 := &models.Trigger{
		ID:         triggerID,
		WorkflowID: "wf-2", // Different workflow
		Type:       models.TriggerTypeWebhook,
		Config: map[string]interface{}{
			"path": "/webhook/updated",
		},
		Enabled: true,
	}

	// Register first trigger
	err := wr.RegisterWebhook(nil, trigger1)
	require.NoError(t, err)

	// Register second trigger with same ID (should replace)
	err = wr.RegisterWebhook(nil, trigger2)
	require.NoError(t, err)

	// Verify only one webhook exists
	wr.mu.RLock()
	numWebhooks := len(wr.webhooks)
	wr.mu.RUnlock()
	assert.Equal(t, 1, numWebhooks)

	// Verify it's the updated one
	retrieved, exists := wr.GetWebhook(triggerID)
	assert.True(t, exists)
	assert.Equal(t, "wf-2", retrieved.WorkflowID)
}

// TestWebhookRegistry_UnregisterNonExistent tests unregistering non-existent webhook
func TestWebhookRegistry_UnregisterNonExistent(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	// Unregistering non-existent webhook should not error
	err := wr.UnregisterWebhook(nil, "non-existent")
	assert.NoError(t, err)
}

// TestWebhookRegistry_EmptyRegisterAll tests registering empty list
func TestWebhookRegistry_EmptyRegisterAll(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	err := wr.RegisterAll(nil, []*storagemodels.TriggerModel{})
	require.NoError(t, err)

	wr.mu.RLock()
	numWebhooks := len(wr.webhooks)
	wr.mu.RUnlock()
	assert.Equal(t, 0, numWebhooks)
}

// TestWebhookRegistry_SignatureWithEmptyPayload tests signature with empty payload
func TestWebhookRegistry_SignatureWithEmptyPayload(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	emptyPayload := map[string]interface{}{}
	secret := "test-secret"

	signature := wr.computeSignature(secret, emptyPayload)
	assert.NotEmpty(t, signature)

	// Empty payload should still produce consistent signature
	signature2 := wr.computeSignature(secret, emptyPayload)
	assert.Equal(t, signature, signature2)
}

// TestWebhookRegistry_IPWhitelistInvalidIP tests IP whitelist with invalid IP
func TestWebhookRegistry_IPWhitelistInvalidIP(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	trigger := &models.Trigger{
		Type: models.TriggerTypeWebhook,
		Config: map[string]interface{}{
			"ip_whitelist": []interface{}{
				"invalid-ip-format",
			},
		},
	}

	// Should handle invalid IP gracefully
	err := wr.checkIPWhitelist(trigger, "192.168.1.1")
	// Should error because source IP doesn't match invalid entry
	assert.Error(t, err)
}

// TestWebhookRegistry_IPWhitelistIPv6 tests IP whitelist with IPv6 addresses
func TestWebhookRegistry_IPWhitelistIPv6(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	tests := []struct {
		name        string
		whitelist   []interface{}
		sourceIP    string
		expectError bool
	}{
		{
			name: "IPv6 exact match",
			whitelist: []interface{}{
				"2001:db8::1",
			},
			sourceIP:    "2001:db8::1",
			expectError: false,
		},
		{
			name: "IPv6 CIDR match",
			whitelist: []interface{}{
				"2001:db8::/32",
			},
			sourceIP:    "2001:db8::100",
			expectError: false,
		},
		{
			name: "IPv6 no match",
			whitelist: []interface{}{
				"2001:db8::1",
			},
			sourceIP:    "2001:db8::2",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := &models.Trigger{
				Type: models.TriggerTypeWebhook,
				Config: map[string]interface{}{
					"ip_whitelist": tt.whitelist,
				},
			}

			err := wr.checkIPWhitelist(trigger, tt.sourceIP)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestWebhookRegistry_SignatureWithNestedPayload tests signature with nested structures
func TestWebhookRegistry_SignatureWithNestedPayload(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	nestedPayload := map[string]interface{}{
		"user": map[string]interface{}{
			"id":   123,
			"name": "John Doe",
			"profile": map[string]interface{}{
				"email": "john@example.com",
			},
		},
		"items": []interface{}{
			map[string]interface{}{"id": 1, "name": "Item 1"},
			map[string]interface{}{"id": 2, "name": "Item 2"},
		},
	}

	secret := "test-secret"

	signature1 := wr.computeSignature(secret, nestedPayload)
	assert.NotEmpty(t, signature1)

	// Same nested structure should produce same signature
	signature2 := wr.computeSignature(secret, nestedPayload)
	assert.Equal(t, signature1, signature2)
}

// TestWebhookRegistry_CheckIPWhitelistInvalidSourceIP tests checkIPWhitelist with invalid source IP format
func TestWebhookRegistry_CheckIPWhitelistInvalidSourceIP(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	trigger := &models.Trigger{
		Type: models.TriggerTypeWebhook,
		Config: map[string]interface{}{
			"ip_whitelist": []interface{}{
				"192.168.1.100",
			},
		},
	}

	// Test with invalid source IP
	err := wr.checkIPWhitelist(trigger, "not-an-ip-address")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source IP")
}

// TestWebhookRegistry_CheckIPWhitelistNonStringEntry tests whitelist with non-string entries
func TestWebhookRegistry_CheckIPWhitelistNonStringEntry(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	trigger := &models.Trigger{
		Type: models.TriggerTypeWebhook,
		Config: map[string]interface{}{
			"ip_whitelist": []interface{}{
				123,           // Non-string entry (should be skipped)
				"192.168.1.1", // Valid entry
			},
		},
	}

	// Should match the valid entry and ignore the non-string
	err := wr.checkIPWhitelist(trigger, "192.168.1.1")
	assert.NoError(t, err)

	// Should not match if only non-string entries in whitelist
	trigger2 := &models.Trigger{
		Type: models.TriggerTypeWebhook,
		Config: map[string]interface{}{
			"ip_whitelist": []interface{}{
				123,  // Non-string entry
				true, // Non-string entry
			},
		},
	}

	err = wr.checkIPWhitelist(trigger2, "192.168.1.1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in whitelist")
}

// TestWebhookRegistry_modelToDomain tests conversion from storage model to domain model
func TestWebhookRegistry_modelToDomain(t *testing.T) {
	wr := NewWebhookRegistry(WebhookRegistryConfig{})

	t.Run("complete model", func(t *testing.T) {
		triggerID := uuid.New()
		workflowID := uuid.New()
		createdAt := time.Now().Add(-1 * time.Hour)
		updatedAt := time.Now()
		lastTriggeredAt := time.Now().Add(-30 * time.Minute)

		storageModel := &storagemodels.TriggerModel{
			ID:         triggerID,
			WorkflowID: workflowID,
			Type:       string(models.TriggerTypeWebhook),
			Config: storagemodels.JSONBMap{
				"path":   "/webhook/test",
				"secret": "test-secret",
			},
			Enabled:         true,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
			LastTriggeredAt: &lastTriggeredAt,
		}

		result := wr.modelToDomain(storageModel)

		assert.Equal(t, triggerID.String(), result.ID)
		assert.Equal(t, workflowID.String(), result.WorkflowID)
		assert.Equal(t, models.TriggerTypeWebhook, result.Type)
		assert.True(t, result.Enabled)
		assert.Equal(t, createdAt, result.CreatedAt)
		assert.Equal(t, updatedAt, result.UpdatedAt)
		assert.NotNil(t, result.LastRun)
		assert.Equal(t, lastTriggeredAt, *result.LastRun)
		assert.Equal(t, "/webhook/test", result.Config["path"])
		assert.Equal(t, "test-secret", result.Config["secret"])
	})

	t.Run("nil config", func(t *testing.T) {
		storageModel := &storagemodels.TriggerModel{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeWebhook),
			Config:     nil,
			Enabled:    true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		result := wr.modelToDomain(storageModel)

		assert.NotNil(t, result.Config)
		assert.Empty(t, result.Config)
	})

	t.Run("nil LastTriggeredAt", func(t *testing.T) {
		storageModel := &storagemodels.TriggerModel{
			ID:              uuid.New(),
			WorkflowID:      uuid.New(),
			Type:            string(models.TriggerTypeWebhook),
			Config:          storagemodels.JSONBMap{"path": "/webhook/test"},
			Enabled:         true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			LastTriggeredAt: nil,
		}

		result := wr.modelToDomain(storageModel)

		assert.Nil(t, result.LastRun)
	})

	t.Run("disabled trigger", func(t *testing.T) {
		storageModel := &storagemodels.TriggerModel{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeWebhook),
			Config:     storagemodels.JSONBMap{"path": "/webhook/test"},
			Enabled:    false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		result := wr.modelToDomain(storageModel)

		assert.False(t, result.Enabled)
	})
}
