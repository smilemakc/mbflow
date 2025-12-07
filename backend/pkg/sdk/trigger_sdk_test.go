package sdk

import (
	"context"
	"testing"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTriggerAPI_Create_ValidationError tests that Create validates trigger
func TestTriggerAPI_Create_ValidationError(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Invalid trigger (missing required fields)
	trigger := &models.Trigger{}

	_, err = client.Triggers().Create(ctx, trigger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger validation failed")
}

// TestTriggerAPI_Create_ClosedClient tests that closed client returns error
func TestTriggerAPI_Create_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	trigger := &models.Trigger{
		Name:       "Test Trigger",
		WorkflowID: "test-workflow-id",
		Type:       models.TriggerTypeManual,
		Enabled:    true,
	}

	_, err = client.Triggers().Create(ctx, trigger)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_Create_NotImplemented tests that Create returns not implemented
func TestTriggerAPI_Create_NotImplemented(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	trigger := &models.Trigger{
		Name:       "Test Trigger",
		WorkflowID: "test-workflow-id",
		Type:       models.TriggerTypeManual,
		Enabled:    true,
	}

	_, err = client.Triggers().Create(ctx, trigger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_Get_EmptyID tests that empty trigger ID is rejected
func TestTriggerAPI_Get_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Triggers().Get(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger ID is required")
}

// TestTriggerAPI_Get_ClosedClient tests that closed client returns error
func TestTriggerAPI_Get_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Triggers().Get(ctx, "test-trigger-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_Get_NotImplemented tests that Get returns not implemented
func TestTriggerAPI_Get_NotImplemented(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Triggers().Get(ctx, "test-trigger-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_List_ClosedClient tests that closed client returns error
func TestTriggerAPI_List_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Triggers().List(ctx, nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_List_NotImplemented tests that List returns not implemented
func TestTriggerAPI_List_NotImplemented(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Triggers().List(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_List_WithOptions tests listing with filter options
func TestTriggerAPI_List_WithOptions(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	enabled := true
	opts := &TriggerListOptions{
		WorkflowID: "test-workflow",
		Type:       string(models.TriggerTypeCron),
		Enabled:    &enabled,
		Limit:      10,
		Offset:     0,
	}

	_, err = client.Triggers().List(ctx, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_Update_EmptyID tests that empty trigger ID is rejected
func TestTriggerAPI_Update_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	trigger := &models.Trigger{
		ID:         "",
		Name:       "Test Trigger",
		WorkflowID: "test-workflow-id",
		Type:       models.TriggerTypeManual,
		Enabled:    true,
	}

	_, err = client.Triggers().Update(ctx, trigger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger ID is required")
}

// TestTriggerAPI_Update_ValidationError tests that Update validates trigger
func TestTriggerAPI_Update_ValidationError(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Invalid trigger (missing required fields)
	trigger := &models.Trigger{
		ID: "test-trigger-id",
	}

	_, err = client.Triggers().Update(ctx, trigger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger validation failed")
}

// TestTriggerAPI_Update_ClosedClient tests that closed client returns error
func TestTriggerAPI_Update_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	trigger := &models.Trigger{
		ID:         "test-trigger-id",
		Name:       "Updated Trigger",
		WorkflowID: "test-workflow-id",
		Type:       models.TriggerTypeManual,
		Enabled:    true,
	}

	_, err = client.Triggers().Update(ctx, trigger)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_Update_NotImplemented tests that Update returns not implemented
func TestTriggerAPI_Update_NotImplemented(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	trigger := &models.Trigger{
		ID:         "test-trigger-id",
		Name:       "Updated Trigger",
		WorkflowID: "test-workflow-id",
		Type:       models.TriggerTypeManual,
		Enabled:    true,
	}

	_, err = client.Triggers().Update(ctx, trigger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_Delete_EmptyID tests that empty trigger ID is rejected
func TestTriggerAPI_Delete_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Triggers().Delete(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger ID is required")
}

// TestTriggerAPI_Delete_ClosedClient tests that closed client returns error
func TestTriggerAPI_Delete_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	err = client.Triggers().Delete(ctx, "test-trigger-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_Delete_NotImplemented tests that Delete returns not implemented
func TestTriggerAPI_Delete_NotImplemented(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Triggers().Delete(ctx, "test-trigger-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_Enable_EmptyID tests that empty trigger ID is rejected
func TestTriggerAPI_Enable_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Triggers().Enable(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger ID is required")
}

// TestTriggerAPI_Enable_ClosedClient tests that closed client returns error
func TestTriggerAPI_Enable_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	err = client.Triggers().Enable(ctx, "test-trigger-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_Enable_NotImplemented tests that Enable returns not implemented
func TestTriggerAPI_Enable_NotImplemented(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Triggers().Enable(ctx, "test-trigger-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_Disable_EmptyID tests that empty trigger ID is rejected
func TestTriggerAPI_Disable_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Triggers().Disable(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger ID is required")
}

// TestTriggerAPI_Disable_ClosedClient tests that closed client returns error
func TestTriggerAPI_Disable_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	err = client.Triggers().Disable(ctx, "test-trigger-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_Disable_NotImplemented tests that Disable returns not implemented
func TestTriggerAPI_Disable_NotImplemented(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	err = client.Triggers().Disable(ctx, "test-trigger-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_Trigger_EmptyID tests that empty trigger ID is rejected
func TestTriggerAPI_Trigger_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Triggers().Trigger(ctx, "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger ID is required")
}

// TestTriggerAPI_Trigger_ClosedClient tests that closed client returns error
func TestTriggerAPI_Trigger_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Triggers().Trigger(ctx, "test-trigger-id", nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_Trigger_NotImplemented tests that Trigger returns not implemented
func TestTriggerAPI_Trigger_NotImplemented(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	input := map[string]interface{}{
		"key": "value",
	}

	_, err = client.Triggers().Trigger(ctx, "test-trigger-id", input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_GetWebhookURL_EmptyID tests that empty trigger ID is rejected
func TestTriggerAPI_GetWebhookURL_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Triggers().GetWebhookURL(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger ID is required")
}

// TestTriggerAPI_GetWebhookURL_ClosedClient tests that closed client returns error
func TestTriggerAPI_GetWebhookURL_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Triggers().GetWebhookURL(ctx, "test-trigger-id")
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_GetHistory_EmptyID tests that empty trigger ID is rejected
func TestTriggerAPI_GetHistory_EmptyID(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Triggers().GetHistory(ctx, "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger ID is required")
}

// TestTriggerAPI_GetHistory_ClosedClient tests that closed client returns error
func TestTriggerAPI_GetHistory_ClosedClient(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	client.Close()

	ctx := context.Background()

	_, err = client.Triggers().GetHistory(ctx, "test-trigger-id", nil)
	assert.ErrorIs(t, err, models.ErrClientClosed)
}

// TestTriggerAPI_GetHistory_NotImplemented tests that GetHistory returns not implemented
func TestTriggerAPI_GetHistory_NotImplemented(t *testing.T) {
	client, err := NewStandaloneClient()
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	opts := &TriggerHistoryOptions{
		Limit:  10,
		Offset: 0,
		Status: "completed",
	}

	_, err = client.Triggers().GetHistory(ctx, "test-trigger-id", opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedded mode not implemented yet")
}

// TestTriggerAPI_TriggerListOptions_Creation tests TriggerListOptions struct
func TestTriggerAPI_TriggerListOptions_Creation(t *testing.T) {
	enabled := true

	opts := &TriggerListOptions{
		WorkflowID: "test-workflow",
		Type:       string(models.TriggerTypeCron),
		Enabled:    &enabled,
		Limit:      20,
		Offset:     10,
	}

	assert.Equal(t, "test-workflow", opts.WorkflowID)
	assert.Equal(t, string(models.TriggerTypeCron), opts.Type)
	assert.NotNil(t, opts.Enabled)
	assert.True(t, *opts.Enabled)
	assert.Equal(t, 20, opts.Limit)
	assert.Equal(t, 10, opts.Offset)
}

// TestTriggerAPI_TriggerHistoryOptions_Creation tests TriggerHistoryOptions struct
func TestTriggerAPI_TriggerHistoryOptions_Creation(t *testing.T) {
	startTime := int64(1000)
	endTime := int64(2000)

	opts := &TriggerHistoryOptions{
		Limit:     50,
		Offset:    5,
		StartTime: &startTime,
		EndTime:   &endTime,
		Status:    "completed",
	}

	assert.Equal(t, 50, opts.Limit)
	assert.Equal(t, 5, opts.Offset)
	assert.Equal(t, int64(1000), *opts.StartTime)
	assert.Equal(t, int64(2000), *opts.EndTime)
	assert.Equal(t, "completed", opts.Status)
}
