package trigger

import (
	"context"
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

func TestParseCronSchedule(t *testing.T) {
	tests := []struct {
		name        string
		trigger     *models.Trigger
		expectError bool
	}{
		{
			name: "valid cron expression with seconds",
			trigger: &models.Trigger{
				Type: models.TriggerTypeCron,
				Config: map[string]any{
					"schedule": "0 */5 * * * *", // Every 5 minutes
				},
			},
			expectError: false,
		},
		{
			name: "valid cron expression with timezone",
			trigger: &models.Trigger{
				Type: models.TriggerTypeCron,
				Config: map[string]any{
					"schedule": "0 0 9 * * *", // 9 AM daily
					"timezone": "America/New_York",
				},
			},
			expectError: false,
		},
		{
			name: "invalid cron expression",
			trigger: &models.Trigger{
				Type: models.TriggerTypeCron,
				Config: map[string]any{
					"schedule": "invalid",
				},
			},
			expectError: true,
		},
		{
			name: "missing schedule",
			trigger: &models.Trigger{
				Type:   models.TriggerTypeCron,
				Config: map[string]any{},
			},
			expectError: true,
		},
		{
			name: "invalid timezone",
			trigger: &models.Trigger{
				Type: models.TriggerTypeCron,
				Config: map[string]any{
					"schedule": "0 0 9 * * *",
					"timezone": "Invalid/Timezone",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := NewCronScheduler(CronSchedulerConfig{})
			require.NoError(t, err)

			schedule, err := cs.parseCronSchedule(tt.trigger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, schedule)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schedule)
			}
		})
	}
}

func TestParseIntervalSchedule(t *testing.T) {
	tests := []struct {
		name        string
		trigger     *models.Trigger
		expectError bool
	}{
		{
			name: "valid duration string",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]any{
					"interval": "30s",
				},
			},
			expectError: false,
		},
		{
			name: "valid integer seconds",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]any{
					"interval": 60,
				},
			},
			expectError: false,
		},
		{
			name: "valid float seconds",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]any{
					"interval": 30.5,
				},
			},
			expectError: false,
		},
		{
			name: "invalid duration string",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]any{
					"interval": "invalid",
				},
			},
			expectError: true,
		},
		{
			name: "negative interval",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]any{
					"interval": -30,
				},
			},
			expectError: true,
		},
		{
			name: "zero interval",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]any{
					"interval": 0,
				},
			},
			expectError: true,
		},
		{
			name: "missing interval",
			trigger: &models.Trigger{
				Type:   models.TriggerTypeInterval,
				Config: map[string]any{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := NewCronScheduler(CronSchedulerConfig{})
			require.NoError(t, err)

			schedule, err := cs.parseIntervalSchedule(tt.trigger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, schedule)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schedule)
			}
		})
	}
}

func TestCronScheduler_AddRemoveTrigger(t *testing.T) {
	t.Skip("Requires full integration test setup")

	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	trigger := &models.Trigger{
		ID:         "test-trigger-1",
		WorkflowID: "test-workflow-1",
		Type:       models.TriggerTypeCron,
		Config: map[string]any{
			"schedule": "0 */5 * * * *", // Every 5 minutes
		},
		Enabled: true,
	}

	// Add trigger
	err = cs.AddTrigger(ctx, trigger)
	assert.NoError(t, err)

	// Verify trigger was added
	cs.mu.RLock()
	_, exists := cs.entries[trigger.ID]
	cs.mu.RUnlock()
	assert.True(t, exists)

	// Remove trigger
	err = cs.RemoveTrigger(ctx, trigger.ID)
	assert.NoError(t, err)

	// Verify trigger was removed
	cs.mu.RLock()
	_, exists = cs.entries[trigger.ID]
	cs.mu.RUnlock()
	assert.False(t, exists)
}

func TestCronScheduler_ScheduleExecution(t *testing.T) {
	t.Skip("Requires full integration test with execution manager")

	// This test would verify:
	// 1. Cron job is scheduled correctly
	// 2. Job executes at the right time
	// 3. Workflow execution is triggered
	// 4. Trigger state is updated
}

func TestCronScheduler_GracefulShutdown(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	err = cs.Start(ctx, nil)
	require.NoError(t, err)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop should complete without error
	err = cs.Stop()
	assert.NoError(t, err)
}

// TestCronScheduler_AddMultipleTriggers tests adding multiple triggers
func TestCronScheduler_AddMultipleTriggers(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	err = cs.Start(ctx, nil)
	require.NoError(t, err)
	defer cs.Stop()

	triggers := []*models.Trigger{
		{
			ID:         "cron-1",
			WorkflowID: "wf-1",
			Type:       models.TriggerTypeCron,
			Config: map[string]any{
				"schedule": "0 0 * * * *", // Every hour
			},
			Enabled: true,
		},
		{
			ID:         "interval-1",
			WorkflowID: "wf-2",
			Type:       models.TriggerTypeInterval,
			Config: map[string]any{
				"interval": "30s",
			},
			Enabled: true,
		},
		{
			ID:         "cron-2",
			WorkflowID: "wf-3",
			Type:       models.TriggerTypeCron,
			Config: map[string]any{
				"schedule": "0 */15 * * * *", // Every 15 minutes
			},
			Enabled: true,
		},
	}

	// Add all triggers
	for _, trigger := range triggers {
		err := cs.AddTrigger(ctx, trigger)
		assert.NoError(t, err)
	}

	// Verify all triggers were added
	cs.mu.RLock()
	assert.Len(t, cs.entries, 3)
	for _, trigger := range triggers {
		_, exists := cs.entries[trigger.ID]
		assert.True(t, exists, "trigger %s should exist", trigger.ID)
	}
	cs.mu.RUnlock()
}

// TestCronScheduler_RemoveNonExistentTrigger tests removing a trigger that doesn't exist
func TestCronScheduler_RemoveNonExistentTrigger(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Removing non-existent trigger should not error
	err = cs.RemoveTrigger(ctx, "non-existent-id")
	assert.NoError(t, err)
}

// TestCronScheduler_UpdateTrigger tests updating a trigger (remove + add)
func TestCronScheduler_UpdateTrigger(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	err = cs.Start(ctx, nil)
	require.NoError(t, err)
	defer cs.Stop()

	triggerID := "update-test"

	// Add initial trigger
	trigger := &models.Trigger{
		ID:         triggerID,
		WorkflowID: "wf-1",
		Type:       models.TriggerTypeCron,
		Config: map[string]any{
			"schedule": "0 0 * * * *", // Every hour
		},
		Enabled: true,
	}

	err = cs.AddTrigger(ctx, trigger)
	require.NoError(t, err)

	// Verify added
	cs.mu.RLock()
	firstEntryID := cs.entries[triggerID]
	cs.mu.RUnlock()
	assert.NotZero(t, firstEntryID)

	// Update trigger with new schedule
	trigger.Config["schedule"] = "0 */30 * * * *" // Every 30 minutes
	err = cs.AddTrigger(ctx, trigger)
	require.NoError(t, err)

	// Verify entry was replaced (new entry ID)
	cs.mu.RLock()
	secondEntryID := cs.entries[triggerID]
	cs.mu.RUnlock()
	assert.NotZero(t, secondEntryID)
	assert.NotEqual(t, firstEntryID, secondEntryID, "entry ID should change after update")
}

// TestCronScheduler_IgnoreNonCronTriggers tests that non-cron triggers are ignored
func TestCronScheduler_IgnoreNonCronTriggers(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	err = cs.Start(ctx, nil)
	require.NoError(t, err)
	defer cs.Stop()

	// Webhook trigger should be ignored
	webhookTrigger := &models.Trigger{
		ID:         "webhook-1",
		WorkflowID: "wf-1",
		Type:       models.TriggerTypeWebhook,
		Config: map[string]any{
			"path": "/webhook/test",
		},
		Enabled: true,
	}

	err = cs.AddTrigger(ctx, webhookTrigger)
	assert.NoError(t, err)

	// Verify no entry was created
	cs.mu.RLock()
	_, exists := cs.entries[webhookTrigger.ID]
	cs.mu.RUnlock()
	assert.False(t, exists, "webhook trigger should not be added to cron scheduler")
}

// TestCronScheduler_ConcurrentOperations tests concurrent add/remove operations
func TestCronScheduler_ConcurrentOperations(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	err = cs.Start(ctx, nil)
	require.NoError(t, err)
	defer cs.Stop()

	const numGoroutines = 10
	const triggersPerGoroutine = 5

	// Add triggers concurrently
	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < triggersPerGoroutine; j++ {
				trigger := &models.Trigger{
					ID:         fmt.Sprintf("trigger-%d-%d", goroutineID, j),
					WorkflowID: fmt.Sprintf("wf-%d-%d", goroutineID, j),
					Type:       models.TriggerTypeCron,
					Config: map[string]any{
						"schedule": "0 0 * * * *",
					},
					Enabled: true,
				}
				_ = cs.AddTrigger(ctx, trigger)
			}
		}(i)
	}
	wg.Wait()

	// Verify triggers were added
	cs.mu.RLock()
	totalTriggers := len(cs.entries)
	cs.mu.RUnlock()
	assert.Greater(t, totalTriggers, 0, "at least some triggers should be added")
	assert.LessOrEqual(t, totalTriggers, numGoroutines*triggersPerGoroutine)

	// Remove triggers concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < triggersPerGoroutine; j++ {
				triggerID := fmt.Sprintf("trigger-%d-%d", goroutineID, j)
				_ = cs.RemoveTrigger(ctx, triggerID)
			}
		}(i)
	}
	wg.Wait()

	// Verify all triggers were removed
	cs.mu.RLock()
	remainingTriggers := len(cs.entries)
	cs.mu.RUnlock()
	assert.Equal(t, 0, remainingTriggers, "all triggers should be removed")
}

// TestCronScheduler_StartWithInitialTriggers tests starting with pre-loaded triggers
func TestCronScheduler_StartWithInitialTriggers(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Create initial triggers
	initialTriggers := []*storagemodels.TriggerModel{
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeCron),
			Config: map[string]any{
				"schedule": "0 0 * * * *",
			},
			Enabled: true,
		},
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeInterval),
			Config: map[string]any{
				"interval": "1m",
			},
			Enabled: true,
		},
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeWebhook), // Should be ignored
			Config: map[string]any{
				"path": "/webhook",
			},
			Enabled: true,
		},
	}

	// Start with initial triggers
	err = cs.Start(ctx, initialTriggers)
	require.NoError(t, err)
	defer cs.Stop()

	// Verify only cron and interval triggers were added
	cs.mu.RLock()
	numEntries := len(cs.entries)
	cs.mu.RUnlock()
	assert.Equal(t, 2, numEntries, "only cron and interval triggers should be loaded")
}

// TestCronScheduler_InvalidScheduleDoesNotCrash tests that invalid schedules don't crash
func TestCronScheduler_InvalidScheduleDoesNotCrash(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	err = cs.Start(ctx, nil)
	require.NoError(t, err)
	defer cs.Stop()

	invalidTrigger := &models.Trigger{
		ID:         "invalid-1",
		WorkflowID: "wf-1",
		Type:       models.TriggerTypeCron,
		Config: map[string]any{
			"schedule": "this is not a valid cron expression",
		},
		Enabled: true,
	}

	// Should return error but not crash
	err = cs.AddTrigger(ctx, invalidTrigger)
	assert.Error(t, err)

	// Verify no entry was created
	cs.mu.RLock()
	_, exists := cs.entries[invalidTrigger.ID]
	cs.mu.RUnlock()
	assert.False(t, exists)
}

// TestCronScheduler_MultipleStartStop tests multiple start/stop cycles
func TestCronScheduler_MultipleStartStop(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	// Multiple start/stop cycles
	for i := 0; i < 3; i++ {
		err = cs.Start(ctx, nil)
		assert.NoError(t, err)

		time.Sleep(50 * time.Millisecond)

		err = cs.Stop()
		assert.NoError(t, err)
	}
}

// TestCronScheduler_LargeInterval tests interval trigger with large duration
func TestCronScheduler_LargeInterval(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	trigger := &models.Trigger{
		Type: models.TriggerTypeInterval,
		Config: map[string]any{
			"interval": "24h", // 1 day
		},
	}

	schedule, err := cs.parseIntervalSchedule(trigger)
	assert.NoError(t, err)
	assert.NotNil(t, schedule)
}

// TestCronScheduler_SmallInterval tests interval trigger with small duration
func TestCronScheduler_SmallInterval(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	trigger := &models.Trigger{
		Type: models.TriggerTypeInterval,
		Config: map[string]any{
			"interval": "1s", // 1 second
		},
	}

	schedule, err := cs.parseIntervalSchedule(trigger)
	assert.NoError(t, err)
	assert.NotNil(t, schedule)
}

// TestCronScheduler_ComplexCronExpressions tests complex cron expressions
func TestCronScheduler_ComplexCronExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		timezone   string
		expectErr  bool
	}{
		{
			name:       "every weekday at 9am",
			expression: "0 0 9 * * MON-FRI",
			timezone:   "America/New_York",
			expectErr:  false,
		},
		{
			name:       "first day of month at midnight",
			expression: "0 0 0 1 * *",
			timezone:   "UTC",
			expectErr:  false,
		},
		{
			name:       "every 15 seconds",
			expression: "*/15 * * * * *",
			timezone:   "UTC",
			expectErr:  false,
		},
		{
			name:       "last day of month",
			expression: "0 0 0 L * *", // Some cron implementations support L
			timezone:   "UTC",
			expectErr:  true, // robfig/cron doesn't support L
		},
	}

	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := &models.Trigger{
				Type: models.TriggerTypeCron,
				Config: map[string]any{
					"schedule": tt.expression,
					"timezone": tt.timezone,
				},
			}

			schedule, err := cs.parseCronSchedule(trigger)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, schedule)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schedule)
			}
		})
	}
}

// TestCronScheduler_DuplicateTriggerID tests adding trigger with duplicate ID
func TestCronScheduler_DuplicateTriggerID(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	err = cs.Start(ctx, nil)
	require.NoError(t, err)
	defer cs.Stop()

	triggerID := "duplicate-test"

	// Add first trigger
	trigger1 := &models.Trigger{
		ID:         triggerID,
		WorkflowID: "wf-1",
		Type:       models.TriggerTypeCron,
		Config: map[string]any{
			"schedule": "0 0 * * * *",
		},
		Enabled: true,
	}

	err = cs.AddTrigger(ctx, trigger1)
	require.NoError(t, err)

	// Add second trigger with same ID (should replace)
	trigger2 := &models.Trigger{
		ID:         triggerID,
		WorkflowID: "wf-2", // Different workflow
		Type:       models.TriggerTypeCron,
		Config: map[string]any{
			"schedule": "0 */30 * * * *", // Different schedule
		},
		Enabled: true,
	}

	err = cs.AddTrigger(ctx, trigger2)
	assert.NoError(t, err)

	// Should only have one entry
	cs.mu.RLock()
	numEntries := len(cs.entries)
	cs.mu.RUnlock()
	assert.Equal(t, 1, numEntries, "duplicate ID should replace existing entry")
}

// TestCronScheduler_modelToDomain tests conversion from storage model to domain model
func TestCronScheduler_modelToDomain(t *testing.T) {
	cs, err := NewCronScheduler(CronSchedulerConfig{})
	require.NoError(t, err)

	t.Run("complete model", func(t *testing.T) {
		triggerID := uuid.New()
		workflowID := uuid.New()
		createdAt := time.Now().Add(-1 * time.Hour)
		updatedAt := time.Now()
		lastTriggeredAt := time.Now().Add(-30 * time.Minute)

		storageModel := &storagemodels.TriggerModel{
			ID:         triggerID,
			WorkflowID: workflowID,
			Type:       string(models.TriggerTypeCron),
			Config: storagemodels.JSONBMap{
				"schedule": "0 0 * * * *",
				"timezone": "UTC",
			},
			Enabled:         true,
			CreatedAt:       createdAt,
			UpdatedAt:       updatedAt,
			LastTriggeredAt: &lastTriggeredAt,
		}

		result := cs.modelToDomain(storageModel)

		assert.Equal(t, triggerID.String(), result.ID)
		assert.Equal(t, workflowID.String(), result.WorkflowID)
		assert.Equal(t, models.TriggerTypeCron, result.Type)
		assert.True(t, result.Enabled)
		assert.Equal(t, createdAt, result.CreatedAt)
		assert.Equal(t, updatedAt, result.UpdatedAt)
		assert.NotNil(t, result.LastRun)
		assert.Equal(t, lastTriggeredAt, *result.LastRun)
		assert.Equal(t, "0 0 * * * *", result.Config["schedule"])
		assert.Equal(t, "UTC", result.Config["timezone"])
	})

	t.Run("nil config", func(t *testing.T) {
		storageModel := &storagemodels.TriggerModel{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeInterval),
			Config:     nil,
			Enabled:    true,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		result := cs.modelToDomain(storageModel)

		assert.NotNil(t, result.Config)
		assert.Empty(t, result.Config)
	})

	t.Run("nil LastTriggeredAt", func(t *testing.T) {
		storageModel := &storagemodels.TriggerModel{
			ID:              uuid.New(),
			WorkflowID:      uuid.New(),
			Type:            string(models.TriggerTypeCron),
			Config:          storagemodels.JSONBMap{"schedule": "0 0 * * * *"},
			Enabled:         true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			LastTriggeredAt: nil,
		}

		result := cs.modelToDomain(storageModel)

		assert.Nil(t, result.LastRun)
	})

	t.Run("disabled trigger", func(t *testing.T) {
		storageModel := &storagemodels.TriggerModel{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Type:       string(models.TriggerTypeCron),
			Config:     storagemodels.JSONBMap{"schedule": "0 0 * * * *"},
			Enabled:    false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		result := cs.modelToDomain(storageModel)

		assert.False(t, result.Enabled)
	})
}

// TestCronScheduler_updateNextExecution tests updating next execution time
func TestCronScheduler_updateNextExecution(t *testing.T) {
	t.Run("nil cache - should not error", func(t *testing.T) {
		cs, err := NewCronScheduler(CronSchedulerConfig{
			Cache: nil,
		})
		require.NoError(t, err)

		ctx := context.Background()
		nextTime := time.Now().Add(1 * time.Hour)

		// Should not error when cache is nil
		err = cs.updateNextExecution(ctx, "trigger-1", nextTime)
		assert.NoError(t, err)
	})

	t.Run("with cache - skip test if Redis not available", func(t *testing.T) {
		// This test would require Redis mock or real Redis instance
		// Skipping for now as it requires cache infrastructure
		t.Skip("Requires Redis cache setup")
	})
}

// Note: executeTrigger and createJob functions require integration test setup
// with ExecutionManager, TriggerRepository, and potentially Redis cache.
// These functions are best tested in integration tests with full infrastructure.
// They are currently at 0% and 20% coverage respectively and will remain so
// until proper integration test infrastructure is available.
