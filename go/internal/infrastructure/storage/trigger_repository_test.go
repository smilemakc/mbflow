package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func setupTriggerRepoTest(t *testing.T) (repository.TriggerRepository, bun.IDB, func()) {
	t.Helper()
	db, cleanup := testutil.SetupTestTx(t)
	return NewTriggerRepository(db), db, cleanup
}

func createTestWorkflowForTrigger(t *testing.T, db bun.IDB) *models.WorkflowModel {
	workflow := &models.WorkflowModel{
		ID:          uuid.New(),
		Name:        fmt.Sprintf("Test Workflow %s", uuid.New().String()[:8]), // Unique name
		Description: "Test workflow for triggers",
		Variables:   models.JSONBMap{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := db.NewInsert().
		Model(workflow).
		Exec(context.Background())
	require.NoError(t, err)

	return workflow
}

// Test Create Operations

func TestTriggerRepo_Create_ManualTrigger(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "manual",
		Config:     models.JSONBMap{},
		Enabled:    true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, trigger.ID)
	assert.False(t, trigger.CreatedAt.IsZero())
	assert.False(t, trigger.UpdatedAt.IsZero())
}

func TestTriggerRepo_Create_CronTrigger(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config: models.JSONBMap{
			"expression": "0 0 * * *",
		},
		Enabled: true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)
	assert.NotNil(t, trigger.Config)
	assert.Equal(t, "0 0 * * *", trigger.GetCronExpression())
}

func TestTriggerRepo_Create_WebhookTrigger(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "webhook",
		Config: models.JSONBMap{
			"url": "/webhooks/test-webhook-123",
		},
		Enabled: true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)
	assert.Equal(t, "/webhooks/test-webhook-123", trigger.GetWebhookURL())
}

func TestTriggerRepo_Create_IntervalTrigger(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "interval",
		Config: models.JSONBMap{
			"seconds": float64(3600), // 1 hour
		},
		Enabled: true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)
	assert.Equal(t, time.Hour, trigger.GetIntervalDuration())
}

// Test Update Operations

func TestTriggerRepo_Update_Config(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config: models.JSONBMap{
			"expression": "0 0 * * *",
		},
		Enabled: true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	// Update config
	trigger.Config["expression"] = "0 12 * * *"
	err = repo.Update(context.Background(), trigger)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(context.Background(), trigger.ID)
	require.NoError(t, err)
	assert.Equal(t, "0 12 * * *", updated.GetCronExpression())
}

func TestTriggerRepo_Update_UpdatedAt(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "manual",
		Config:     models.JSONBMap{},
		Enabled:    true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	originalUpdatedAt := trigger.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	// Update
	trigger.Config["test"] = "value"
	err = repo.Update(context.Background(), trigger)
	require.NoError(t, err)

	assert.True(t, trigger.UpdatedAt.After(originalUpdatedAt))
}

// Test Delete Operations

func TestTriggerRepo_Delete_Success(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "manual",
		Config:     models.JSONBMap{},
		Enabled:    true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	// Delete
	err = repo.Delete(context.Background(), trigger.ID)
	require.NoError(t, err)

	// Verify deleted
	deleted, err := repo.FindByID(context.Background(), trigger.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestTriggerRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	// Delete non-existent trigger
	err := repo.Delete(context.Background(), uuid.New())
	require.NoError(t, err) // Should not error, just no rows affected
}

// Test FindByID Operations

func TestTriggerRepo_FindByID_Success(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config: models.JSONBMap{
			"expression": "0 0 * * *",
		},
		Enabled: true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	// Find by ID
	found, err := repo.FindByID(context.Background(), trigger.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, trigger.ID, found.ID)
	assert.Equal(t, trigger.WorkflowID, found.WorkflowID)
	assert.Equal(t, "cron", found.Type)
	assert.True(t, found.IsCron())
}

func TestTriggerRepo_FindByID_NotFound(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	// Find non-existent trigger
	found, err := repo.FindByID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Nil(t, found)
}

// Test FindByWorkflowID Operations

func TestTriggerRepo_FindByWorkflowID_MultipleTriggers(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	// Create multiple triggers for the same workflow
	trigger1 := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "manual",
		Config:     models.JSONBMap{},
		Enabled:    true,
	}

	trigger2 := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config:     models.JSONBMap{"expression": "0 0 * * *"},
		Enabled:    true,
	}

	err := repo.Create(context.Background(), trigger1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different created_at

	err = repo.Create(context.Background(), trigger2)
	require.NoError(t, err)

	// Find all triggers for workflow
	triggers, err := repo.FindByWorkflowID(context.Background(), workflow.ID)
	require.NoError(t, err)
	assert.Len(t, triggers, 2)

	// Should be ordered by created_at DESC
	assert.Equal(t, trigger2.ID, triggers[0].ID)
	assert.Equal(t, trigger1.ID, triggers[1].ID)
}

func TestTriggerRepo_FindByWorkflowID_Empty(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	// Find triggers for non-existent workflow
	triggers, err := repo.FindByWorkflowID(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, triggers)
}

// Test FindByType Operations

func TestTriggerRepo_FindByType_WithPagination(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	// Create 5 cron triggers
	for i := 0; i < 5; i++ {
		trigger := &models.TriggerModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Type:       "cron",
			Config:     models.JSONBMap{"expression": fmt.Sprintf("0 %d * * *", i)},
			Enabled:    true,
		}
		err := repo.Create(context.Background(), trigger)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Get first page
	page1, err := repo.FindByType(context.Background(), "cron", 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Get second page
	page2, err := repo.FindByType(context.Background(), "cron", 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Verify different triggers
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func TestTriggerRepo_FindByType_Empty(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	triggers, err := repo.FindByType(context.Background(), "webhook", 10, 0)
	require.NoError(t, err)
	assert.Empty(t, triggers)
}

// Test FindEnabled Operations

func TestTriggerRepo_FindEnabled_OnlyEnabled(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	// Create enabled trigger
	enabled := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config:     models.JSONBMap{"expression": "0 0 * * *"},
		Enabled:    true,
	}

	// Create disabled trigger
	disabled := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "webhook",
		Config:     models.JSONBMap{"url": "/webhooks/test"},
		Enabled:    false,
	}

	err := repo.Create(context.Background(), enabled)
	require.NoError(t, err)

	err = repo.Create(context.Background(), disabled)
	require.NoError(t, err)

	// Verify enabled states after insertion
	verifyEnabled, err := repo.FindByID(context.Background(), enabled.ID)
	require.NoError(t, err)
	assert.True(t, verifyEnabled.Enabled, "enabled trigger should be enabled")

	verifyDisabled, err := repo.FindByID(context.Background(), disabled.ID)
	require.NoError(t, err)
	assert.False(t, verifyDisabled.Enabled, "disabled trigger should be disabled")

	// Find enabled triggers
	triggers, err := repo.FindEnabled(context.Background())
	require.NoError(t, err)
	require.Len(t, triggers, 1, "should only find 1 enabled trigger")
	assert.True(t, triggers[0].Enabled)
	assert.Equal(t, "cron", triggers[0].Type)
}

// Test FindEnabledByType Operations

func TestTriggerRepo_FindEnabledByType_FilterByBoth(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	// Create enabled cron trigger
	enabledCron := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config:     models.JSONBMap{"expression": "0 0 * * *"},
		Enabled:    true,
	}

	// Create disabled cron trigger
	disabledCron := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config:     models.JSONBMap{"expression": "0 12 * * *"},
		Enabled:    false,
	}

	// Create enabled webhook trigger
	enabledWebhook := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "webhook",
		Config:     models.JSONBMap{"url": "/webhooks/test"},
		Enabled:    true,
	}

	err := repo.Create(context.Background(), enabledCron)
	require.NoError(t, err)

	err = repo.Create(context.Background(), disabledCron)
	require.NoError(t, err)

	err = repo.Create(context.Background(), enabledWebhook)
	require.NoError(t, err)

	// Find enabled cron triggers
	triggers, err := repo.FindEnabledByType(context.Background(), "cron")
	require.NoError(t, err)
	assert.Len(t, triggers, 1)
	assert.True(t, triggers[0].Enabled)
	assert.Equal(t, "cron", triggers[0].Type)
	assert.Equal(t, "0 0 * * *", triggers[0].GetCronExpression())
}

// Test FindAll Operations

func TestTriggerRepo_FindAll_Pagination(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	// Create 5 triggers
	for i := 0; i < 5; i++ {
		trigger := &models.TriggerModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Type:       "manual",
			Config:     models.JSONBMap{},
			Enabled:    true,
		}
		err := repo.Create(context.Background(), trigger)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Get all with pagination
	page1, err := repo.FindAll(context.Background(), 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	page2, err := repo.FindAll(context.Background(), 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Verify ordering (DESC by created_at)
	assert.True(t, page1[0].CreatedAt.After(page1[1].CreatedAt) || page1[0].CreatedAt.Equal(page1[1].CreatedAt))
}

// Test Count Operations

func TestTriggerRepo_Count_Total(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	// Create 3 triggers
	for i := 0; i < 3; i++ {
		trigger := &models.TriggerModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Type:       "manual",
			Config:     models.JSONBMap{},
			Enabled:    true,
		}
		err := repo.Create(context.Background(), trigger)
		require.NoError(t, err)
	}

	count, err := repo.Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestTriggerRepo_CountByWorkflowID_FilterByWorkflow(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow1 := createTestWorkflowForTrigger(t, db)
	workflow2 := createTestWorkflowForTrigger(t, db)

	// Create 2 triggers for workflow1
	for i := 0; i < 2; i++ {
		trigger := &models.TriggerModel{
			ID:         uuid.New(),
			WorkflowID: workflow1.ID,
			Type:       "manual",
			Config:     models.JSONBMap{},
			Enabled:    true,
		}
		err := repo.Create(context.Background(), trigger)
		require.NoError(t, err)
	}

	// Create 1 trigger for workflow2
	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow2.ID,
		Type:       "cron",
		Config:     models.JSONBMap{"expression": "0 0 * * *"},
		Enabled:    true,
	}
	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	// Count by workflow1
	count1, err := repo.CountByWorkflowID(context.Background(), workflow1.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count1)

	// Count by workflow2
	count2, err := repo.CountByWorkflowID(context.Background(), workflow2.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count2)
}

func TestTriggerRepo_CountByType_FilterByType(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	// Create 2 cron triggers
	for i := 0; i < 2; i++ {
		trigger := &models.TriggerModel{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			Type:       "cron",
			Config:     models.JSONBMap{"expression": "0 0 * * *"},
			Enabled:    true,
		}
		err := repo.Create(context.Background(), trigger)
		require.NoError(t, err)
	}

	// Create 1 webhook trigger
	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "webhook",
		Config:     models.JSONBMap{"url": "/webhooks/test"},
		Enabled:    true,
	}
	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	// Count cron triggers
	cronCount, err := repo.CountByType(context.Background(), "cron")
	require.NoError(t, err)
	assert.Equal(t, 2, cronCount)

	// Count webhook triggers
	webhookCount, err := repo.CountByType(context.Background(), "webhook")
	require.NoError(t, err)
	assert.Equal(t, 1, webhookCount)
}

// Test Enable/Disable Operations

func TestTriggerRepo_Enable_Success(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config:     models.JSONBMap{"expression": "0 0 * * *"},
		Enabled:    false, // Start disabled
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	// Enable
	err = repo.Enable(context.Background(), trigger.ID)
	require.NoError(t, err)

	// Verify enabled
	enabled, err := repo.FindByID(context.Background(), trigger.ID)
	require.NoError(t, err)
	assert.True(t, enabled.Enabled)
}

func TestTriggerRepo_Disable_Success(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config:     models.JSONBMap{"expression": "0 0 * * *"},
		Enabled:    true, // Start enabled
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	// Disable
	err = repo.Disable(context.Background(), trigger.ID)
	require.NoError(t, err)

	// Verify disabled
	disabled, err := repo.FindByID(context.Background(), trigger.ID)
	require.NoError(t, err)
	assert.False(t, disabled.Enabled)
}

// Test MarkTriggered Operations

func TestTriggerRepo_MarkTriggered_UpdatesTimestamp(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config:     models.JSONBMap{"expression": "0 0 * * *"},
		Enabled:    true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	// Should initially be nil
	assert.Nil(t, trigger.LastTriggeredAt)

	// Mark as triggered
	err = repo.MarkTriggered(context.Background(), trigger.ID)
	require.NoError(t, err)

	// Verify timestamp updated
	triggered, err := repo.FindByID(context.Background(), trigger.ID)
	require.NoError(t, err)
	require.NotNil(t, triggered.LastTriggeredAt)
	assert.True(t, triggered.LastTriggeredAt.Before(time.Now().Add(time.Second)))
}

func TestTriggerRepo_MarkTriggered_MultipleTimes(t *testing.T) {
	t.Parallel()
	repo, db, cleanup := setupTriggerRepoTest(t)
	defer cleanup()

	workflow := createTestWorkflowForTrigger(t, db)

	trigger := &models.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Type:       "cron",
		Config:     models.JSONBMap{"expression": "0 0 * * *"},
		Enabled:    true,
	}

	err := repo.Create(context.Background(), trigger)
	require.NoError(t, err)

	// Mark as triggered first time
	err = repo.MarkTriggered(context.Background(), trigger.ID)
	require.NoError(t, err)

	first, err := repo.FindByID(context.Background(), trigger.ID)
	require.NoError(t, err)
	firstTime := first.LastTriggeredAt

	time.Sleep(100 * time.Millisecond)

	// Mark as triggered second time
	err = repo.MarkTriggered(context.Background(), trigger.ID)
	require.NoError(t, err)

	second, err := repo.FindByID(context.Background(), trigger.ID)
	require.NoError(t, err)
	secondTime := second.LastTriggeredAt

	// Second time should be after first time
	require.NotNil(t, secondTime)
	assert.True(t, secondTime.After(*firstTime))
}

// Test Model Helper Methods

func TestTriggerModel_TypeCheckers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		triggerType  string
		expectedFunc func(*models.TriggerModel) bool
	}{
		{"Manual", "manual", (*models.TriggerModel).IsManual},
		{"Cron", "cron", (*models.TriggerModel).IsCron},
		{"Webhook", "webhook", (*models.TriggerModel).IsWebhook},
		{"Event", "event", (*models.TriggerModel).IsEvent},
		{"Interval", "interval", (*models.TriggerModel).IsInterval},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := &models.TriggerModel{
				Type: tt.triggerType,
			}
			assert.True(t, tt.expectedFunc(trigger))
		})
	}
}

func TestTriggerModel_GetCronExpression_ValidCron(t *testing.T) {
	t.Parallel()
	trigger := &models.TriggerModel{
		Type: "cron",
		Config: models.JSONBMap{
			"expression": "0 0 * * *",
		},
	}

	expr := trigger.GetCronExpression()
	assert.Equal(t, "0 0 * * *", expr)
}

func TestTriggerModel_GetCronExpression_NotCronType(t *testing.T) {
	t.Parallel()
	trigger := &models.TriggerModel{
		Type: "webhook",
		Config: models.JSONBMap{
			"expression": "0 0 * * *",
		},
	}

	expr := trigger.GetCronExpression()
	assert.Empty(t, expr)
}

func TestTriggerModel_GetWebhookURL_ValidWebhook(t *testing.T) {
	t.Parallel()
	trigger := &models.TriggerModel{
		Type: "webhook",
		Config: models.JSONBMap{
			"url": "/webhooks/test-123",
		},
	}

	url := trigger.GetWebhookURL()
	assert.Equal(t, "/webhooks/test-123", url)
}

func TestTriggerModel_GetIntervalDuration_ValidInterval(t *testing.T) {
	t.Parallel()
	trigger := &models.TriggerModel{
		Type: "interval",
		Config: models.JSONBMap{
			"seconds": float64(300), // 5 minutes
		},
	}

	duration := trigger.GetIntervalDuration()
	assert.Equal(t, 5*time.Minute, duration)
}

func TestTriggerModel_MarkTriggered_SetsTimestamp(t *testing.T) {
	t.Parallel()
	trigger := &models.TriggerModel{
		Type: "cron",
	}

	before := time.Now()
	trigger.MarkTriggered()
	after := time.Now()

	require.NotNil(t, trigger.LastTriggeredAt)
	assert.True(t, trigger.LastTriggeredAt.After(before) || trigger.LastTriggeredAt.Equal(before))
	assert.True(t, trigger.LastTriggeredAt.Before(after) || trigger.LastTriggeredAt.Equal(after))
}
