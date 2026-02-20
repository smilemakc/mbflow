package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== TriggerType Tests ====================

func TestTriggerType_Constants(t *testing.T) {
	assert.Equal(t, TriggerType("manual"), TriggerTypeManual)
	assert.Equal(t, TriggerType("cron"), TriggerTypeCron)
	assert.Equal(t, TriggerType("webhook"), TriggerTypeWebhook)
	assert.Equal(t, TriggerType("event"), TriggerTypeEvent)
	assert.Equal(t, TriggerType("interval"), TriggerTypeInterval)
}

// ==================== Trigger.Validate Tests ====================

func TestTrigger_Validate_ManualTrigger_Success(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Manual Trigger",
		Type:       TriggerTypeManual,
		Config:     map[string]any{},
		Enabled:    true,
	}

	err := trigger.Validate()
	assert.NoError(t, err)
}

func TestTrigger_Validate_MissingWorkflowID(t *testing.T) {
	trigger := &Trigger{
		Name:    "Test Trigger",
		Type:    TriggerTypeManual,
		Config:  map[string]any{},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow ID is required")
}

func TestTrigger_Validate_MissingName(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Type:       TriggerTypeManual,
		Config:     map[string]any{},
		Enabled:    true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger name is required")
}

func TestTrigger_Validate_MissingType(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Test Trigger",
		Config:     map[string]any{},
		Enabled:    true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger type is required")
}

func TestTrigger_Validate_InvalidType(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Test Trigger",
		Type:       TriggerType("invalid"),
		Config:     map[string]any{},
		Enabled:    true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid trigger type")
}

// ==================== Cron Trigger Tests ====================

func TestTrigger_Validate_CronTrigger_Success(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Daily Cron",
		Type:       TriggerTypeCron,
		Config: map[string]any{
			"schedule": "0 0 * * *", // Daily at midnight
			"timezone": "UTC",
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.NoError(t, err)
}

func TestTrigger_Validate_CronTrigger_MissingSchedule(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Daily Cron",
		Type:       TriggerTypeCron,
		Config:     map[string]any{},
		Enabled:    true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cron schedule is required")
}

func TestTrigger_Validate_CronTrigger_EmptySchedule(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Daily Cron",
		Type:       TriggerTypeCron,
		Config: map[string]any{
			"schedule": "",
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cron schedule is required")
}

func TestTrigger_Validate_CronTrigger_InvalidScheduleType(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Daily Cron",
		Type:       TriggerTypeCron,
		Config: map[string]any{
			"schedule": 123, // Should be string
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cron schedule is required")
}

// ==================== Webhook Trigger Tests ====================

func TestTrigger_Validate_WebhookTrigger_Success(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Webhook Trigger",
		Type:       TriggerTypeWebhook,
		Config: map[string]any{
			"secret": "my-secret-token",
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.NoError(t, err)
}

func TestTrigger_Validate_WebhookTrigger_NoConfig(t *testing.T) {
	// Webhook config is optional
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Webhook Trigger",
		Type:       TriggerTypeWebhook,
		Config:     map[string]any{},
		Enabled:    true,
	}

	err := trigger.Validate()
	assert.NoError(t, err)
}

// ==================== Event Trigger Tests ====================

func TestTrigger_Validate_EventTrigger_Success(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Event Trigger",
		Type:       TriggerTypeEvent,
		Config: map[string]any{
			"event_type": "user.created",
			"source":     "user-service",
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.NoError(t, err)
}

func TestTrigger_Validate_EventTrigger_MissingEventType(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Event Trigger",
		Type:       TriggerTypeEvent,
		Config:     map[string]any{},
		Enabled:    true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event type is required")
}

func TestTrigger_Validate_EventTrigger_EmptyEventType(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Event Trigger",
		Type:       TriggerTypeEvent,
		Config: map[string]any{
			"event_type": "",
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event type is required")
}

func TestTrigger_Validate_EventTrigger_InvalidEventTypeType(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Event Trigger",
		Type:       TriggerTypeEvent,
		Config: map[string]any{
			"event_type": 123, // Should be string
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event type is required")
}

// ==================== Interval Trigger Tests ====================

func TestTrigger_Validate_IntervalTrigger_NumberSuccess(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Interval Trigger",
		Type:       TriggerTypeInterval,
		Config: map[string]any{
			"interval": float64(30), // 30 seconds
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.NoError(t, err)
}

func TestTrigger_Validate_IntervalTrigger_DurationStringSuccess(t *testing.T) {
	tests := []string{
		"30s",   // 30 seconds
		"5m",    // 5 minutes
		"1h",    // 1 hour
		"24h",   // 24 hours
		"1h30m", // 1.5 hours
	}

	for _, interval := range tests {
		t.Run(interval, func(t *testing.T) {
			trigger := &Trigger{
				WorkflowID: "wf_123",
				Name:       "Interval Trigger",
				Type:       TriggerTypeInterval,
				Config: map[string]any{
					"interval": interval,
				},
				Enabled: true,
			}

			err := trigger.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestTrigger_Validate_IntervalTrigger_MissingInterval(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Interval Trigger",
		Type:       TriggerTypeInterval,
		Config:     map[string]any{},
		Enabled:    true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interval is required")
}

func TestTrigger_Validate_IntervalTrigger_ZeroInterval(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Interval Trigger",
		Type:       TriggerTypeInterval,
		Config: map[string]any{
			"interval": float64(0),
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interval must be positive")
}

func TestTrigger_Validate_IntervalTrigger_NegativeInterval(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Interval Trigger",
		Type:       TriggerTypeInterval,
		Config: map[string]any{
			"interval": float64(-10),
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interval must be positive")
}

func TestTrigger_Validate_IntervalTrigger_InvalidDurationFormat(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Interval Trigger",
		Type:       TriggerTypeInterval,
		Config: map[string]any{
			"interval": "invalid duration",
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration format")
}

func TestTrigger_Validate_IntervalTrigger_InvalidType(t *testing.T) {
	trigger := &Trigger{
		WorkflowID: "wf_123",
		Name:       "Interval Trigger",
		Type:       TriggerTypeInterval,
		Config: map[string]any{
			"interval": true, // Invalid type (bool)
		},
		Enabled: true,
	}

	err := trigger.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "interval must be a number or duration string")
}

// ==================== Trigger JSON Tests ====================

func TestTrigger_JSONMarshaling(t *testing.T) {
	now := time.Now()
	lastRun := now.Add(-1 * time.Hour)
	nextRun := now.Add(1 * time.Hour)

	trigger := &Trigger{
		ID:          "trigger_123",
		WorkflowID:  "wf_123",
		Name:        "Test Trigger",
		Description: "Test trigger description",
		Type:        TriggerTypeCron,
		Config: map[string]any{
			"schedule": "0 * * * *",
		},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
		LastRun:   &lastRun,
		NextRun:   &nextRun,
		Metadata: map[string]any{
			"author": "system",
		},
	}

	data, err := json.Marshal(trigger)
	require.NoError(t, err)

	var unmarshaled Trigger
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, trigger.ID, unmarshaled.ID)
	assert.Equal(t, trigger.WorkflowID, unmarshaled.WorkflowID)
	assert.Equal(t, trigger.Name, unmarshaled.Name)
	assert.Equal(t, trigger.Type, unmarshaled.Type)
	assert.Equal(t, trigger.Enabled, unmarshaled.Enabled)
	assert.NotNil(t, unmarshaled.LastRun)
	assert.NotNil(t, unmarshaled.NextRun)
}

// ==================== CronConfig Tests ====================

func TestCronConfig_JSONMarshaling(t *testing.T) {
	config := &CronConfig{
		Schedule: "0 0 * * *",
		Timezone: "America/New_York",
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled CronConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.Schedule, unmarshaled.Schedule)
	assert.Equal(t, config.Timezone, unmarshaled.Timezone)
}

func TestCronConfig_NoTimezone(t *testing.T) {
	config := &CronConfig{
		Schedule: "*/5 * * * *", // Every 5 minutes
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled CronConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.Schedule, unmarshaled.Schedule)
	assert.Empty(t, unmarshaled.Timezone)
}

// ==================== WebhookConfig Tests ====================

func TestWebhookConfig_JSONMarshaling(t *testing.T) {
	config := &WebhookConfig{
		Secret: "my-webhook-secret",
		Headers: map[string]string{
			"X-Custom-Header": "value",
			"Authorization":   "Bearer token",
		},
		ContentType: "application/json",
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled WebhookConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.Secret, unmarshaled.Secret)
	assert.Equal(t, config.ContentType, unmarshaled.ContentType)
	assert.Len(t, unmarshaled.Headers, 2)
	assert.Equal(t, "value", unmarshaled.Headers["X-Custom-Header"])
}

func TestWebhookConfig_Empty(t *testing.T) {
	config := &WebhookConfig{}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled WebhookConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Empty(t, unmarshaled.Secret)
	assert.Empty(t, unmarshaled.ContentType)
}

// ==================== EventConfig Tests ====================

func TestEventConfig_JSONMarshaling(t *testing.T) {
	config := &EventConfig{
		EventType: "user.created",
		Filter: map[string]any{
			"status": "active",
			"role":   "admin",
		},
		Source: "user-service",
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled EventConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.EventType, unmarshaled.EventType)
	assert.Equal(t, config.Source, unmarshaled.Source)
	assert.NotNil(t, unmarshaled.Filter)
	assert.Equal(t, "active", unmarshaled.Filter["status"])
}

func TestEventConfig_NoFilter(t *testing.T) {
	config := &EventConfig{
		EventType: "order.created",
		Source:    "order-service",
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled EventConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.EventType, unmarshaled.EventType)
	assert.Nil(t, unmarshaled.Filter)
}

// ==================== IntervalConfig Tests ====================

func TestIntervalConfig_JSONMarshaling(t *testing.T) {
	config := &IntervalConfig{
		Interval: "5m",
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var unmarshaled IntervalConfig
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, config.Interval, unmarshaled.Interval)
}

func TestIntervalConfig_VariousDurations(t *testing.T) {
	durations := []string{
		"30s",
		"5m",
		"1h",
		"24h",
		"1h30m45s",
	}

	for _, duration := range durations {
		t.Run(duration, func(t *testing.T) {
			config := &IntervalConfig{
				Interval: duration,
			}

			data, err := json.Marshal(config)
			require.NoError(t, err)

			var unmarshaled IntervalConfig
			err = json.Unmarshal(data, &unmarshaled)
			require.NoError(t, err)

			assert.Equal(t, duration, unmarshaled.Interval)

			// Verify it's parseable
			_, err = time.ParseDuration(unmarshaled.Interval)
			assert.NoError(t, err)
		})
	}
}

// ==================== Complex Integration Tests ====================

func TestTrigger_AllTypes_Validation(t *testing.T) {
	tests := []struct {
		name    string
		trigger *Trigger
		wantErr bool
	}{
		{
			name: "Valid Manual Trigger",
			trigger: &Trigger{
				WorkflowID: "wf_123",
				Name:       "Manual",
				Type:       TriggerTypeManual,
				Config:     map[string]any{},
				Enabled:    true,
			},
			wantErr: false,
		},
		{
			name: "Valid Cron Trigger",
			trigger: &Trigger{
				WorkflowID: "wf_123",
				Name:       "Hourly Cron",
				Type:       TriggerTypeCron,
				Config: map[string]any{
					"schedule": "0 * * * *",
				},
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "Valid Webhook Trigger",
			trigger: &Trigger{
				WorkflowID: "wf_123",
				Name:       "Webhook",
				Type:       TriggerTypeWebhook,
				Config:     map[string]any{},
				Enabled:    true,
			},
			wantErr: false,
		},
		{
			name: "Valid Event Trigger",
			trigger: &Trigger{
				WorkflowID: "wf_123",
				Name:       "Event",
				Type:       TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "user.created",
				},
				Enabled: true,
			},
			wantErr: false,
		},
		{
			name: "Valid Interval Trigger",
			trigger: &Trigger{
				WorkflowID: "wf_123",
				Name:       "Interval",
				Type:       TriggerTypeInterval,
				Config: map[string]any{
					"interval": "5m",
				},
				Enabled: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trigger.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
