package trigger

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventListener_MatchesFilter(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	tests := []struct {
		name     string
		event    Event
		trigger  *models.Trigger
		expected bool
	}{
		{
			name: "no filter - matches all",
			event: Event{
				Type: "user.created",
				Data: map[string]interface{}{
					"user_id": "123",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]interface{}{
					"event_type": "user.created",
				},
			},
			expected: true,
		},
		{
			name: "source filter match",
			event: Event{
				Type:   "user.created",
				Source: "api",
				Data: map[string]interface{}{
					"user_id": "123",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]interface{}{
					"event_type": "user.created",
					"filter": map[string]interface{}{
						"source": "api",
					},
				},
			},
			expected: true,
		},
		{
			name: "source filter no match",
			event: Event{
				Type:   "user.created",
				Source: "webhook",
				Data: map[string]interface{}{
					"user_id": "123",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]interface{}{
					"event_type": "user.created",
					"filter": map[string]interface{}{
						"source": "api",
					},
				},
			},
			expected: false,
		},
		{
			name: "custom field filter match",
			event: Event{
				Type: "user.created",
				Data: map[string]interface{}{
					"user_id": "123",
					"role":    "admin",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]interface{}{
					"event_type": "user.created",
					"filter": map[string]interface{}{
						"role": "admin",
					},
				},
			},
			expected: true,
		},
		{
			name: "custom field filter no match",
			event: Event{
				Type: "user.created",
				Data: map[string]interface{}{
					"user_id": "123",
					"role":    "user",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]interface{}{
					"event_type": "user.created",
					"filter": map[string]interface{}{
						"role": "admin",
					},
				},
			},
			expected: false,
		},
		{
			name: "missing field in event data",
			event: Event{
				Type: "user.created",
				Data: map[string]interface{}{
					"user_id": "123",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]interface{}{
					"event_type": "user.created",
					"filter": map[string]interface{}{
						"role": "admin",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := el.matchesFilter(tt.event, tt.trigger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEventListener_GetEventChannel(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	tests := []struct {
		eventType string
		expected  string
	}{
		{
			eventType: "user.created",
			expected:  "mbflow:events:user.created",
		},
		{
			eventType: "order.completed",
			expected:  "mbflow:events:order.completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			channel := el.getEventChannel(tt.eventType)
			assert.Equal(t, tt.expected, channel)
		})
	}
}

func TestEventListener_AddRemoveTrigger(t *testing.T) {
	t.Skip("Requires Redis connection")

	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	trigger := &models.Trigger{
		ID:         "test-trigger-1",
		WorkflowID: "test-workflow-1",
		Type:       models.TriggerTypeEvent,
		Config: map[string]interface{}{
			"event_type": "user.created",
		},
		Enabled: true,
	}

	// Add trigger
	err = el.AddTrigger(ctx, trigger)
	assert.NoError(t, err)

	// Verify trigger was added
	el.mu.RLock()
	triggers := el.triggers["user.created"]
	el.mu.RUnlock()
	assert.Len(t, triggers, 1)
	assert.Equal(t, trigger.ID, triggers[0].ID)

	// Remove trigger
	err = el.RemoveTrigger(ctx, trigger.ID)
	assert.NoError(t, err)

	// Verify trigger was removed
	el.mu.RLock()
	triggers = el.triggers["user.created"]
	el.mu.RUnlock()
	assert.Len(t, triggers, 0)
}

func TestEvent_JSONSerialization(t *testing.T) {
	event := Event{
		Type:   "user.created",
		Source: "api",
		Data: map[string]interface{}{
			"user_id": "123",
			"email":   "user@example.com",
		},
		Timestamp: time.Now(),
	}

	// Serialize
	data, err := json.Marshal(event)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Deserialize
	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.Type, decoded.Type)
	assert.Equal(t, event.Source, decoded.Source)
	assert.Equal(t, event.Data["user_id"], decoded.Data["user_id"])
	assert.Equal(t, event.Data["email"], decoded.Data["email"])
}

func TestEventListener_StartStop(t *testing.T) {
	t.Skip("Requires Redis connection")

	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Start with no triggers
	err = el.Start(ctx, nil)
	assert.NoError(t, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop should complete without error
	err = el.Stop()
	assert.NoError(t, err)
}
