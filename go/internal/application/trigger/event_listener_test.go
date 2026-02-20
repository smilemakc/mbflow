package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
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
				Data: map[string]any{
					"user_id": "123",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
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
				Data: map[string]any{
					"user_id": "123",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "user.created",
					"filter": map[string]any{
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
				Data: map[string]any{
					"user_id": "123",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "user.created",
					"filter": map[string]any{
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
				Data: map[string]any{
					"user_id": "123",
					"role":    "admin",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "user.created",
					"filter": map[string]any{
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
				Data: map[string]any{
					"user_id": "123",
					"role":    "user",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "user.created",
					"filter": map[string]any{
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
				Data: map[string]any{
					"user_id": "123",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "user.created",
					"filter": map[string]any{
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
		Config: map[string]any{
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
		Data: map[string]any{
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

func TestEventListener_StopWithoutStart(t *testing.T) {
	// Test that Stop doesn't hang when listener was never started
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	// Create a timeout to ensure Stop doesn't hang
	done := make(chan bool)
	go func() {
		err := el.Stop()
		assert.NoError(t, err)
		done <- true
	}()

	select {
	case <-done:
		// Success - Stop completed
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() hung - did not complete within timeout")
	}
}

func TestEventListener_StopWithNoTriggersStarted(t *testing.T) {
	// Test that Stop doesn't hang when Start was called but no triggers were added
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Start with empty trigger list (no triggers)
	// This simulates the case where listener goroutine is NOT started
	err = el.Start(ctx, []*storagemodels.TriggerModel{})
	assert.NoError(t, err)

	// Create a timeout to ensure Stop doesn't hang
	done := make(chan bool)
	go func() {
		err := el.Stop()
		assert.NoError(t, err)
		done <- true
	}()

	select {
	case <-done:
		// Success - Stop completed
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() hung - did not complete within timeout")
	}
}

// ==================== NEW COMPREHENSIVE TESTS ====================

func TestEventListener_AddTriggerNonEventType(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Add non-event trigger (should be ignored)
	trigger := &models.Trigger{
		ID:         uuid.New().String(),
		WorkflowID: uuid.New().String(),
		Type:       models.TriggerTypeCron,
		Config: map[string]any{
			"schedule": "0 0 * * *",
		},
		Enabled: true,
	}

	err = el.AddTrigger(ctx, trigger)
	assert.NoError(t, err)

	// Verify trigger was not added to event triggers
	el.mu.RLock()
	assert.Len(t, el.triggers, 0)
	el.mu.RUnlock()
}

func TestEventListener_AddTriggerMissingEventType(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Add event trigger without event_type in config
	trigger := &models.Trigger{
		ID:         uuid.New().String(),
		WorkflowID: uuid.New().String(),
		Type:       models.TriggerTypeEvent,
		Config:     map[string]any{}, // Missing event_type
		Enabled:    true,
	}

	err = el.AddTrigger(ctx, trigger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event_type not found")
}

func TestEventListener_AddMultipleTriggersSameEventType(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Add 3 triggers for the same event type
	eventType := "user.created"
	for i := 0; i < 3; i++ {
		trigger := &models.Trigger{
			ID:         fmt.Sprintf("trigger-%d", i+1),
			WorkflowID: uuid.New().String(),
			Type:       models.TriggerTypeEvent,
			Config: map[string]any{
				"event_type": eventType,
			},
			Enabled: true,
		}

		err = el.AddTrigger(ctx, trigger)
		assert.NoError(t, err)
	}

	// Verify all 3 triggers were added
	el.mu.RLock()
	triggers := el.triggers[eventType]
	el.mu.RUnlock()

	assert.Len(t, triggers, 3)
	assert.Equal(t, "trigger-1", triggers[0].ID)
	assert.Equal(t, "trigger-2", triggers[1].ID)
	assert.Equal(t, "trigger-3", triggers[2].ID)
}

func TestEventListener_RemoveTriggerNonExistent(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Try to remove non-existent trigger (should not error)
	err = el.RemoveTrigger(ctx, "non-existent-trigger")
	assert.NoError(t, err)

	// Verify triggers map is still empty
	el.mu.RLock()
	assert.Len(t, el.triggers, 0)
	el.mu.RUnlock()
}

func TestEventListener_RemoveLastTriggerForEventType(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Add trigger
	trigger := &models.Trigger{
		ID:         "test-trigger",
		WorkflowID: uuid.New().String(),
		Type:       models.TriggerTypeEvent,
		Config: map[string]any{
			"event_type": "user.created",
		},
		Enabled: true,
	}

	err = el.AddTrigger(ctx, trigger)
	assert.NoError(t, err)

	// Verify trigger was added
	el.mu.RLock()
	assert.Len(t, el.triggers["user.created"], 1)
	el.mu.RUnlock()

	// Remove the trigger
	err = el.RemoveTrigger(ctx, "test-trigger")
	assert.NoError(t, err)

	// Verify event type was removed from map
	el.mu.RLock()
	_, exists := el.triggers["user.created"]
	assert.False(t, exists)
	el.mu.RUnlock()
}

func TestEventListener_ConcurrentOperations(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	const numGoroutines = 10
	const triggersPerGoroutine = 5

	var wg sync.WaitGroup

	// Concurrent Add operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < triggersPerGoroutine; j++ {
				trigger := &models.Trigger{
					ID:         fmt.Sprintf("trigger-%d-%d", routineID, j),
					WorkflowID: uuid.New().String(),
					Type:       models.TriggerTypeEvent,
					Config: map[string]any{
						"event_type": fmt.Sprintf("event.type.%d", routineID%3),
					},
					Enabled: true,
				}
				_ = el.AddTrigger(ctx, trigger)
			}
		}(i)
	}
	wg.Wait()

	// Verify all triggers were added
	el.mu.RLock()
	totalTriggers := 0
	for _, triggers := range el.triggers {
		totalTriggers += len(triggers)
	}
	el.mu.RUnlock()

	assert.Equal(t, numGoroutines*triggersPerGoroutine, totalTriggers)

	// Concurrent Remove operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < triggersPerGoroutine; j++ {
				triggerID := fmt.Sprintf("trigger-%d-%d", routineID, j)
				_ = el.RemoveTrigger(ctx, triggerID)
			}
		}(i)
	}
	wg.Wait()

	// Verify all triggers were removed
	el.mu.RLock()
	totalTriggersAfterRemove := 0
	for _, triggers := range el.triggers {
		totalTriggersAfterRemove += len(triggers)
	}
	el.mu.RUnlock()

	assert.Equal(t, 0, totalTriggersAfterRemove)
}

func TestEventListener_ComplexFilterMatching(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	tests := []struct {
		name     string
		event    Event
		trigger  *models.Trigger
		expected bool
	}{
		{
			name: "multiple field filters - all match",
			event: Event{
				Type:   "user.updated",
				Source: "api",
				Data: map[string]any{
					"user_id": "123",
					"role":    "admin",
					"status":  "active",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "user.updated",
					"filter": map[string]any{
						"source": "api",
						"role":   "admin",
						"status": "active",
					},
				},
			},
			expected: true,
		},
		{
			name: "multiple field filters - partial match",
			event: Event{
				Type:   "user.updated",
				Source: "api",
				Data: map[string]any{
					"user_id": "123",
					"role":    "admin",
					"status":  "inactive",
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "user.updated",
					"filter": map[string]any{
						"source": "api",
						"role":   "admin",
						"status": "active",
					},
				},
			},
			expected: false,
		},
		{
			name: "numeric field filter match",
			event: Event{
				Type: "order.created",
				Data: map[string]any{
					"order_id": "456",
					"amount":   100,
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "order.created",
					"filter": map[string]any{
						"amount": 100,
					},
				},
			},
			expected: true,
		},
		{
			name: "boolean field filter match",
			event: Event{
				Type: "task.completed",
				Data: map[string]any{
					"task_id": "789",
					"success": true,
				},
			},
			trigger: &models.Trigger{
				Type: models.TriggerTypeEvent,
				Config: map[string]any{
					"event_type": "task.completed",
					"filter": map[string]any{
						"success": true,
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := el.matchesFilter(tt.event, tt.trigger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEventListener_GetChannels(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Add triggers for multiple event types
	eventTypes := []string{"user.created", "order.completed", "task.updated"}
	for _, eventType := range eventTypes {
		trigger := &models.Trigger{
			ID:         uuid.New().String(),
			WorkflowID: uuid.New().String(),
			Type:       models.TriggerTypeEvent,
			Config: map[string]any{
				"event_type": eventType,
			},
			Enabled: true,
		}
		err = el.AddTrigger(ctx, trigger)
		require.NoError(t, err)
	}

	// Get channels
	channels := el.getChannels()

	// Verify all channels are present
	assert.Len(t, channels, 3)
	expectedChannels := map[string]bool{
		"mbflow:events:user.created":    false,
		"mbflow:events:order.completed": false,
		"mbflow:events:task.updated":    false,
	}

	for _, channel := range channels {
		expectedChannels[channel] = true
	}

	for channel, found := range expectedChannels {
		assert.True(t, found, "Channel %s not found", channel)
	}
}

func TestEventListener_ModelToDomain(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	now := time.Now()
	triggerID := uuid.New()
	workflowID := uuid.New()

	tm := &storagemodels.TriggerModel{
		ID:         triggerID,
		WorkflowID: workflowID,
		Type:       string(models.TriggerTypeEvent),
		Config: map[string]any{
			"event_type": "user.created",
			"filter": map[string]any{
				"source": "api",
			},
		},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Convert to domain model
	trigger := el.modelToDomain(tm)

	// Verify conversion
	assert.Equal(t, triggerID.String(), trigger.ID)
	assert.Equal(t, workflowID.String(), trigger.WorkflowID)
	assert.Equal(t, models.TriggerTypeEvent, trigger.Type)
	assert.Equal(t, "user.created", trigger.Config["event_type"])
	assert.True(t, trigger.Enabled)
	assert.Equal(t, now, trigger.CreatedAt)
	assert.Equal(t, now, trigger.UpdatedAt)
}

func TestEventListener_ModelToDomainWithLastTriggered(t *testing.T) {
	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	now := time.Now()
	lastTriggered := now.Add(-1 * time.Hour)
	triggerID := uuid.New()
	workflowID := uuid.New()

	tm := &storagemodels.TriggerModel{
		ID:              triggerID,
		WorkflowID:      workflowID,
		Type:            string(models.TriggerTypeEvent),
		Config:          map[string]any{"event_type": "user.created"},
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastTriggeredAt: &lastTriggered,
	}

	// Convert to domain model
	trigger := el.modelToDomain(tm)

	// Verify LastRun is set
	assert.NotNil(t, trigger.LastRun)
	assert.Equal(t, lastTriggered, *trigger.LastRun)
}

func TestEventListener_StartWithMixedTriggers(t *testing.T) {
	t.Skip("Requires Redis connection")

	el, err := NewEventListener(EventListenerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Create mixed trigger models (event and non-event)
	triggers := []*storagemodels.TriggerModel{
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeEvent),
			Config: map[string]any{
				"event_type": "user.created",
			},
			Enabled: true,
		},
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeCron),
			Config: map[string]any{
				"schedule": "0 0 * * *",
			},
			Enabled: true,
		},
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeEvent),
			Config: map[string]any{
				"event_type": "order.completed",
			},
			Enabled: true,
		},
	}

	// Start should only register event triggers
	err = el.Start(ctx, triggers)
	assert.NoError(t, err)

	// Verify only 2 event triggers were added
	el.mu.RLock()
	assert.Len(t, el.triggers, 2)
	assert.Len(t, el.triggers["user.created"], 1)
	assert.Len(t, el.triggers["order.completed"], 1)
	el.mu.RUnlock()
}
