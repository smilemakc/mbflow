package serviceapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// --- triggerModelToDomain ---

func TestTriggerModelToDomain_ShouldReturnNil_WhenNilInput(t *testing.T) {
	result := triggerModelToDomain(nil, "", "")

	assert.Nil(t, result)
}

func TestTriggerModelToDomain_ShouldMapFields_WhenValidModel(t *testing.T) {
	// Arrange
	trigID := uuid.New()
	wfID := uuid.New()
	now := time.Now()
	lastTriggered := now.Add(-1 * time.Hour)

	tm := &storagemodels.TriggerModel{
		ID:              trigID,
		WorkflowID:      wfID,
		Type:            "cron",
		Config:          storagemodels.JSONBMap{"expression": "0 * * * *"},
		Enabled:         true,
		LastTriggeredAt: &lastTriggered,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Act
	result := triggerModelToDomain(tm, "My Trigger", "Runs hourly")

	// Assert
	require.NotNil(t, result)
	assert.Equal(t, trigID.String(), result.ID)
	assert.Equal(t, wfID.String(), result.WorkflowID)
	assert.Equal(t, models.TriggerType("cron"), result.Type)
	assert.True(t, result.Enabled)
	assert.Equal(t, "My Trigger", result.Name)
	assert.Equal(t, "Runs hourly", result.Description)
	assert.Equal(t, &lastTriggered, result.LastRun)
	assert.Equal(t, "0 * * * *", result.Config["expression"])
}

func TestTriggerModelToDomain_ShouldUseConfigName_WhenNameParamEmpty(t *testing.T) {
	tm := &storagemodels.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Type:       "manual",
		Config:     storagemodels.JSONBMap{"name": "Config Name", "description": "Config Desc"},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	result := triggerModelToDomain(tm, "", "")

	assert.Equal(t, "Config Name", result.Name)
	assert.Equal(t, "Config Desc", result.Description)
}

func TestTriggerModelToDomain_ShouldPreferParamName_OverConfigName(t *testing.T) {
	tm := &storagemodels.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Type:       "manual",
		Config:     storagemodels.JSONBMap{"name": "Config Name"},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	result := triggerModelToDomain(tm, "Explicit Name", "")

	assert.Equal(t, "Explicit Name", result.Name)
}

func TestTriggerModelToDomain_ShouldHandleNilLastTriggeredAt(t *testing.T) {
	tm := &storagemodels.TriggerModel{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Type:       "webhook",
		Config:     storagemodels.JSONBMap{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	result := triggerModelToDomain(tm, "", "")

	assert.Nil(t, result.LastRun)
}

// --- isValidTriggerType ---

func TestIsValidTriggerType_ShouldReturnTrue_ForValidTypes(t *testing.T) {
	validTypes := []string{"manual", "cron", "webhook", "event", "interval"}
	for _, typ := range validTypes {
		t.Run(typ, func(t *testing.T) {
			assert.True(t, isValidTriggerType(typ))
		})
	}
}

func TestIsValidTriggerType_ShouldReturnFalse_ForInvalidTypes(t *testing.T) {
	invalidTypes := []string{"timer", "schedule", "api", "", "CRON", "Manual"}
	for _, typ := range invalidTypes {
		t.Run(typ, func(t *testing.T) {
			assert.False(t, isValidTriggerType(typ))
		})
	}
}

// --- ListTriggers ---

func TestListTriggers_ShouldFindAll_WhenNoFilters(t *testing.T) {
	// Arrange
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	triggerModels := []*storagemodels.TriggerModel{
		{ID: uuid.New(), WorkflowID: uuid.New(), Type: "manual", Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), WorkflowID: uuid.New(), Type: "cron", Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	trigRepo.On("FindAll", mock.Anything, 10, 0).Return(triggerModels, nil)
	trigRepo.On("Count", mock.Anything).Return(2, nil)

	// Act
	result, err := ops.ListTriggers(context.Background(), ListTriggersParams{Limit: 10, Offset: 0})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Triggers, 2)
	assert.Equal(t, 2, result.Total)
	trigRepo.AssertExpectations(t)
}

func TestListTriggers_ShouldFilterByWorkflowID_WhenProvided(t *testing.T) {
	// Arrange
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	wfID := uuid.New()
	triggerModels := []*storagemodels.TriggerModel{
		{ID: uuid.New(), WorkflowID: wfID, Type: "cron", Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	trigRepo.On("FindByWorkflowID", mock.Anything, wfID).Return(triggerModels, nil)
	trigRepo.On("CountByWorkflowID", mock.Anything, wfID).Return(1, nil)

	// Act
	result, err := ops.ListTriggers(context.Background(), ListTriggersParams{WorkflowID: &wfID})

	// Assert
	require.NoError(t, err)
	assert.Len(t, result.Triggers, 1)
	assert.Equal(t, 1, result.Total)
}

func TestListTriggers_ShouldFilterByType_WhenProvided(t *testing.T) {
	// Arrange
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigType := "webhook"
	triggerModels := []*storagemodels.TriggerModel{
		{ID: uuid.New(), WorkflowID: uuid.New(), Type: "webhook", Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	trigRepo.On("FindByType", mock.Anything, "webhook", 20, 5).Return(triggerModels, nil)
	trigRepo.On("CountByType", mock.Anything, "webhook").Return(1, nil)

	// Act
	result, err := ops.ListTriggers(context.Background(), ListTriggersParams{
		Limit:  20,
		Offset: 5,
		Type:   &trigType,
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, result.Triggers, 1)
}

func TestListTriggers_ShouldWorkflowIDTakePrecedence_OverType(t *testing.T) {
	// Arrange
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	wfID := uuid.New()
	trigType := "cron"
	trigRepo.On("FindByWorkflowID", mock.Anything, wfID).Return([]*storagemodels.TriggerModel{}, nil)
	trigRepo.On("CountByWorkflowID", mock.Anything, wfID).Return(0, nil)

	// Act
	result, err := ops.ListTriggers(context.Background(), ListTriggersParams{
		WorkflowID: &wfID,
		Type:       &trigType,
	})

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result.Triggers)
	trigRepo.AssertNotCalled(t, "FindByType")
}

func TestListTriggers_ShouldReturnError_WhenRepoFails(t *testing.T) {
	// Arrange
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigRepo.On("FindAll", mock.Anything, 10, 0).Return(([]*storagemodels.TriggerModel)(nil), errors.New("query failed"))

	// Act
	result, err := ops.ListTriggers(context.Background(), ListTriggersParams{Limit: 10, Offset: 0})

	// Assert
	assert.Nil(t, result)
	require.Error(t, err)
}

func TestListTriggers_ShouldFallbackToLen_WhenCountFails(t *testing.T) {
	// Arrange
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	triggerModels := []*storagemodels.TriggerModel{
		{ID: uuid.New(), WorkflowID: uuid.New(), Type: "manual", Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	trigRepo.On("FindAll", mock.Anything, 10, 0).Return(triggerModels, nil)
	trigRepo.On("Count", mock.Anything).Return(0, errors.New("count failed"))

	// Act
	result, err := ops.ListTriggers(context.Background(), ListTriggersParams{Limit: 10, Offset: 0})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total) // Falls back to len(triggers)
}

// --- CreateTrigger ---

func TestCreateTrigger_ShouldReturnError_WhenWorkflowIDEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	result, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
		WorkflowID: "",
		Name:       "Test",
		Type:       "manual",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "WORKFLOW_ID_REQUIRED", opErr.Code)
}

func TestCreateTrigger_ShouldReturnError_WhenNameEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	result, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
		WorkflowID: uuid.New().String(),
		Name:       "",
		Type:       "manual",
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NAME_REQUIRED", opErr.Code)
}

func TestCreateTrigger_ShouldReturnError_WhenTypeEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	result, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
		WorkflowID: uuid.New().String(),
		Name:       "Test",
		Type:       "",
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "TYPE_REQUIRED", opErr.Code)
}

func TestCreateTrigger_ShouldReturnError_WhenWorkflowIDNotUUID(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	result, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
		WorkflowID: "not-a-uuid",
		Name:       "Test",
		Type:       "manual",
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_ID", opErr.Code)
}

func TestCreateTrigger_ShouldReturnError_WhenInvalidTriggerType(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(wfRepo, nil, trigRepo, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)

	result, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
		WorkflowID: wfID.String(),
		Name:       "Test",
		Type:       "invalid_type",
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_TRIGGER_TYPE", opErr.Code)
}

func TestCreateTrigger_ShouldReturnError_WhenWorkflowNotFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	result, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
		WorkflowID: wfID.String(),
		Name:       "Test",
		Type:       "manual",
	})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestCreateTrigger_ShouldReturnTrigger_WhenValidParams(t *testing.T) {
	// Arrange
	wfRepo := new(mockWorkflowRepo)
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(wfRepo, nil, trigRepo, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	trigRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TriggerModel")).Return(nil)

	// Act
	result, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
		WorkflowID:  wfID.String(),
		Name:        "My Cron Trigger",
		Description: "Runs every hour",
		Type:        "cron",
		Config:      map[string]any{"expression": "0 * * * *"},
		Enabled:     true,
	})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, wfID.String(), result.WorkflowID)
	assert.Equal(t, models.TriggerType("cron"), result.Type)
	assert.Equal(t, "My Cron Trigger", result.Name)
	assert.Equal(t, "Runs every hour", result.Description)
	assert.True(t, result.Enabled)
}

func TestCreateTrigger_ShouldReturnError_WhenRepoCreateFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(wfRepo, nil, trigRepo, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	trigRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("create failed"))

	result, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
		WorkflowID: wfID.String(),
		Name:       "Test",
		Type:       "manual",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create failed")
}

func TestCreateTrigger_ShouldAcceptAllValidTypes(t *testing.T) {
	validTypes := []string{"manual", "cron", "webhook", "event", "interval"}
	for _, typ := range validTypes {
		t.Run(typ, func(t *testing.T) {
			wfRepo := new(mockWorkflowRepo)
			trigRepo := new(mockTriggerRepo)
			ops := newTestOperations(wfRepo, nil, trigRepo, nil, nil, nil, nil)

			wfID := uuid.New()
			wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
			trigRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

			result, err := ops.CreateTrigger(context.Background(), CreateTriggerParams{
				WorkflowID: wfID.String(),
				Name:       "Test",
				Type:       typ,
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, models.TriggerType(typ), result.Type)
		})
	}
}

// --- GetTrigger ---

func TestGetTrigger_ShouldReturnTrigger_WhenFound(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	wfID := uuid.New()
	tm := &storagemodels.TriggerModel{
		ID: trigID, WorkflowID: wfID, Type: "manual",
		Config: storagemodels.JSONBMap{"name": "Existing"}, Enabled: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	trigRepo.On("FindByID", mock.Anything, trigID).Return(tm, nil)

	result, err := ops.GetTrigger(context.Background(), GetTriggerParams{TriggerID: trigID})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, trigID.String(), result.ID)
}

func TestGetTrigger_ShouldReturnNotFound_WhenNilModel(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("FindByID", mock.Anything, trigID).Return((*storagemodels.TriggerModel)(nil), nil)

	result, err := ops.GetTrigger(context.Background(), GetTriggerParams{TriggerID: trigID})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrTriggerNotFound)
}

func TestGetTrigger_ShouldReturnNotFound_WhenRepoErrors(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("FindByID", mock.Anything, trigID).Return((*storagemodels.TriggerModel)(nil), errors.New("not found"))

	result, err := ops.GetTrigger(context.Background(), GetTriggerParams{TriggerID: trigID})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, models.ErrTriggerNotFound)
}

// --- UpdateTrigger ---

func TestUpdateTrigger_ShouldReturnNotFound_WhenTriggerMissing(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("FindByID", mock.Anything, trigID).Return((*storagemodels.TriggerModel)(nil), nil)

	result, err := ops.UpdateTrigger(context.Background(), UpdateTriggerParams{TriggerID: trigID})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, models.ErrTriggerNotFound)
}

func TestUpdateTrigger_ShouldUpdateType_WhenProvided(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	tm := &storagemodels.TriggerModel{
		ID: trigID, WorkflowID: uuid.New(), Type: "manual",
		Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	trigRepo.On("FindByID", mock.Anything, trigID).Return(tm, nil)
	trigRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	result, err := ops.UpdateTrigger(context.Background(), UpdateTriggerParams{
		TriggerID: trigID,
		Type:      "cron",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, models.TriggerType("cron"), result.Type)
}

func TestUpdateTrigger_ShouldReturnError_WhenInvalidType(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	tm := &storagemodels.TriggerModel{
		ID: trigID, WorkflowID: uuid.New(), Type: "manual",
		Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	trigRepo.On("FindByID", mock.Anything, trigID).Return(tm, nil)

	result, err := ops.UpdateTrigger(context.Background(), UpdateTriggerParams{
		TriggerID: trigID,
		Type:      "bad_type",
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_TRIGGER_TYPE", opErr.Code)
}

func TestUpdateTrigger_ShouldUpdateConfig_WhenProvided(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	tm := &storagemodels.TriggerModel{
		ID: trigID, WorkflowID: uuid.New(), Type: "cron",
		Config: storagemodels.JSONBMap{"expression": "old"}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	trigRepo.On("FindByID", mock.Anything, trigID).Return(tm, nil)
	trigRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *storagemodels.TriggerModel) bool {
		return m.Config["expression"] == "0 0 * * *"
	})).Return(nil)

	result, err := ops.UpdateTrigger(context.Background(), UpdateTriggerParams{
		TriggerID: trigID,
		Config:    map[string]any{"expression": "0 0 * * *"},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	trigRepo.AssertExpectations(t)
}

func TestUpdateTrigger_ShouldUpdateEnabled_WhenProvided(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	tm := &storagemodels.TriggerModel{
		ID: trigID, WorkflowID: uuid.New(), Type: "manual",
		Config: storagemodels.JSONBMap{}, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	trigRepo.On("FindByID", mock.Anything, trigID).Return(tm, nil)

	disabled := false
	trigRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *storagemodels.TriggerModel) bool {
		return !m.Enabled
	})).Return(nil)

	result, err := ops.UpdateTrigger(context.Background(), UpdateTriggerParams{
		TriggerID: trigID,
		Enabled:   &disabled,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Enabled)
}

func TestUpdateTrigger_ShouldReturnError_WhenRepoUpdateFails(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	tm := &storagemodels.TriggerModel{
		ID: trigID, WorkflowID: uuid.New(), Type: "manual",
		Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	trigRepo.On("FindByID", mock.Anything, trigID).Return(tm, nil)
	trigRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("update failed"))

	enabled := true
	result, err := ops.UpdateTrigger(context.Background(), UpdateTriggerParams{
		TriggerID: trigID,
		Enabled:   &enabled,
	})

	assert.Nil(t, result)
	require.Error(t, err)
}

// --- DeleteTrigger ---

func TestDeleteTrigger_ShouldReturnNil_WhenSuccess(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("Delete", mock.Anything, trigID).Return(nil)

	err := ops.DeleteTrigger(context.Background(), DeleteTriggerParams{TriggerID: trigID})

	require.NoError(t, err)
	trigRepo.AssertExpectations(t)
}

func TestDeleteTrigger_ShouldReturnError_WhenRepoFails(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("Delete", mock.Anything, trigID).Return(errors.New("delete failed"))

	err := ops.DeleteTrigger(context.Background(), DeleteTriggerParams{TriggerID: trigID})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
}

// --- EnableTrigger ---

func TestEnableTrigger_ShouldReturnTrigger_WhenSuccess(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("Enable", mock.Anything, trigID).Return(nil)
	trigRepo.On("FindByID", mock.Anything, trigID).Return(&storagemodels.TriggerModel{
		ID: trigID, WorkflowID: uuid.New(), Type: "cron", Enabled: true,
		Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)

	result, err := ops.EnableTrigger(context.Background(), EnableTriggerParams{TriggerID: trigID})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Enabled)
}

func TestEnableTrigger_ShouldReturnError_WhenEnableFails(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("Enable", mock.Anything, trigID).Return(errors.New("enable failed"))

	result, err := ops.EnableTrigger(context.Background(), EnableTriggerParams{TriggerID: trigID})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestEnableTrigger_ShouldReturnNotFound_WhenFindAfterEnableFails(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("Enable", mock.Anything, trigID).Return(nil)
	trigRepo.On("FindByID", mock.Anything, trigID).Return((*storagemodels.TriggerModel)(nil), nil)

	result, err := ops.EnableTrigger(context.Background(), EnableTriggerParams{TriggerID: trigID})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, models.ErrTriggerNotFound)
}

// --- DisableTrigger ---

func TestDisableTrigger_ShouldReturnTrigger_WhenSuccess(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("Disable", mock.Anything, trigID).Return(nil)
	trigRepo.On("FindByID", mock.Anything, trigID).Return(&storagemodels.TriggerModel{
		ID: trigID, WorkflowID: uuid.New(), Type: "cron", Enabled: false,
		Config: storagemodels.JSONBMap{}, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)

	result, err := ops.DisableTrigger(context.Background(), DisableTriggerParams{TriggerID: trigID})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Enabled)
}

func TestDisableTrigger_ShouldReturnError_WhenDisableFails(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("Disable", mock.Anything, trigID).Return(errors.New("disable failed"))

	result, err := ops.DisableTrigger(context.Background(), DisableTriggerParams{TriggerID: trigID})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestDisableTrigger_ShouldReturnNotFound_WhenFindAfterDisableFails(t *testing.T) {
	trigRepo := new(mockTriggerRepo)
	ops := newTestOperations(nil, nil, trigRepo, nil, nil, nil, nil)

	trigID := uuid.New()
	trigRepo.On("Disable", mock.Anything, trigID).Return(nil)
	trigRepo.On("FindByID", mock.Anything, trigID).Return((*storagemodels.TriggerModel)(nil), nil)

	result, err := ops.DisableTrigger(context.Background(), DisableTriggerParams{TriggerID: trigID})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, models.ErrTriggerNotFound)
}
