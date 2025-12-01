package trigger

import (
	"testing"

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
