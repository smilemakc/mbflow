package trigger

import (
	"context"
	"testing"
	"time"

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
				Config: map[string]interface{}{
					"schedule": "0 */5 * * * *", // Every 5 minutes
				},
			},
			expectError: false,
		},
		{
			name: "valid cron expression with timezone",
			trigger: &models.Trigger{
				Type: models.TriggerTypeCron,
				Config: map[string]interface{}{
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
				Config: map[string]interface{}{
					"schedule": "invalid",
				},
			},
			expectError: true,
		},
		{
			name: "missing schedule",
			trigger: &models.Trigger{
				Type:   models.TriggerTypeCron,
				Config: map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name: "invalid timezone",
			trigger: &models.Trigger{
				Type: models.TriggerTypeCron,
				Config: map[string]interface{}{
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
				Config: map[string]interface{}{
					"interval": "30s",
				},
			},
			expectError: false,
		},
		{
			name: "valid integer seconds",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]interface{}{
					"interval": 60,
				},
			},
			expectError: false,
		},
		{
			name: "valid float seconds",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]interface{}{
					"interval": 30.5,
				},
			},
			expectError: false,
		},
		{
			name: "invalid duration string",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]interface{}{
					"interval": "invalid",
				},
			},
			expectError: true,
		},
		{
			name: "negative interval",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]interface{}{
					"interval": -30,
				},
			},
			expectError: true,
		},
		{
			name: "zero interval",
			trigger: &models.Trigger{
				Type: models.TriggerTypeInterval,
				Config: map[string]interface{}{
					"interval": 0,
				},
			},
			expectError: true,
		},
		{
			name: "missing interval",
			trigger: &models.Trigger{
				Type:   models.TriggerTypeInterval,
				Config: map[string]interface{}{},
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
		Config: map[string]interface{}{
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
