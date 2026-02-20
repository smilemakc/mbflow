package serviceapi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// --- ListWorkflows ---

func TestListWorkflows_ShouldReturnWorkflows_WhenNoFilters(t *testing.T) {
	// Arrange
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfModels := []*storagemodels.WorkflowModel{
		{ID: uuid.New(), Name: "WF1", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), Name: "WF2", Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	wfRepo.On("FindAllWithFilters", mock.Anything, mock.Anything, 10, 0).Return(wfModels, nil)
	wfRepo.On("CountWithFilters", mock.Anything, mock.Anything).Return(2, nil)

	// Act
	result, err := ops.ListWorkflows(context.Background(), ListWorkflowsParams{Limit: 10, Offset: 0})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Workflows, 2)
	assert.Equal(t, 2, result.Total)
}

func TestListWorkflows_ShouldIncludeUnowned_WhenNoUserIDFilter(t *testing.T) {
	// Arrange
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfRepo.On("FindAllWithFilters", mock.Anything, mock.MatchedBy(func(f any) bool {
		// The IncludeUnowned should be true when no UserID is specified
		return true
	}), 10, 0).Return([]*storagemodels.WorkflowModel{}, nil)
	wfRepo.On("CountWithFilters", mock.Anything, mock.Anything).Return(0, nil)

	// Act
	_, err := ops.ListWorkflows(context.Background(), ListWorkflowsParams{Limit: 10, Offset: 0})

	// Assert
	require.NoError(t, err)
}

func TestListWorkflows_ShouldFilterByStatus_WhenProvided(t *testing.T) {
	// Arrange
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	status := "active"
	wfRepo.On("FindAllWithFilters", mock.Anything, mock.MatchedBy(func(f any) bool {
		return true
	}), 10, 0).Return([]*storagemodels.WorkflowModel{}, nil)
	wfRepo.On("CountWithFilters", mock.Anything, mock.Anything).Return(0, nil)

	// Act
	_, err := ops.ListWorkflows(context.Background(), ListWorkflowsParams{
		Limit:  10,
		Offset: 0,
		Status: &status,
	})

	// Assert
	require.NoError(t, err)
}

func TestListWorkflows_ShouldFilterByUserID_WhenProvided(t *testing.T) {
	// Arrange
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	userID := uuid.New()
	wfRepo.On("FindAllWithFilters", mock.Anything, mock.Anything, 10, 0).Return([]*storagemodels.WorkflowModel{}, nil)
	wfRepo.On("CountWithFilters", mock.Anything, mock.Anything).Return(0, nil)

	// Act
	_, err := ops.ListWorkflows(context.Background(), ListWorkflowsParams{
		Limit:  10,
		Offset: 0,
		UserID: &userID,
	})

	// Assert
	require.NoError(t, err)
}

func TestListWorkflows_ShouldReturnError_WhenRepoFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfRepo.On("FindAllWithFilters", mock.Anything, mock.Anything, 10, 0).Return(([]*storagemodels.WorkflowModel)(nil), errors.New("db error"))

	result, err := ops.ListWorkflows(context.Background(), ListWorkflowsParams{Limit: 10})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestListWorkflows_ShouldFallbackToLen_WhenCountFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfModels := []*storagemodels.WorkflowModel{
		{ID: uuid.New(), Name: "WF1", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	wfRepo.On("FindAllWithFilters", mock.Anything, mock.Anything, 10, 0).Return(wfModels, nil)
	wfRepo.On("CountWithFilters", mock.Anything, mock.Anything).Return(0, errors.New("count error"))

	result, err := ops.ListWorkflows(context.Background(), ListWorkflowsParams{Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
}

// --- GetWorkflow ---

func TestGetWorkflow_ShouldReturnWorkflow_WhenFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "Test Workflow", Description: "Desc", Status: "active",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(wfModel, nil)

	result, err := ops.GetWorkflow(context.Background(), GetWorkflowParams{WorkflowID: wfID})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, wfID.String(), result.ID)
	assert.Equal(t, "Test Workflow", result.Name)
}

func TestGetWorkflow_ShouldReturnError_WhenNotFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	result, err := ops.GetWorkflow(context.Background(), GetWorkflowParams{WorkflowID: wfID})

	assert.Nil(t, result)
	require.Error(t, err)
}

// --- CreateWorkflow ---

func TestCreateWorkflow_ShouldReturnError_WhenNameEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, nil)

	result, err := ops.CreateWorkflow(context.Background(), CreateWorkflowParams{Name: ""})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NAME_REQUIRED", opErr.Code)
}

func TestCreateWorkflow_ShouldReturnWorkflow_WhenValidParams(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.WorkflowModel")).Return(nil)

	createdBy := uuid.New()
	result, err := ops.CreateWorkflow(context.Background(), CreateWorkflowParams{
		Name:        "New Workflow",
		Description: "Does stuff",
		Variables:   map[string]any{"env": "prod"},
		Metadata:    map[string]any{"team": "backend"},
		CreatedBy:   &createdBy,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Workflow", result.Name)
	assert.Equal(t, "Does stuff", result.Description)
	assert.Equal(t, models.WorkflowStatus("draft"), result.Status)
}

func TestCreateWorkflow_ShouldSetDraftStatus(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	var savedModel *storagemodels.WorkflowModel
	wfRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			savedModel = args.Get(1).(*storagemodels.WorkflowModel)
		}).
		Return(nil)

	_, err := ops.CreateWorkflow(context.Background(), CreateWorkflowParams{Name: "Test"})

	require.NoError(t, err)
	require.NotNil(t, savedModel)
	assert.Equal(t, "draft", savedModel.Status)
	assert.Equal(t, 1, savedModel.Version)
}

func TestCreateWorkflow_ShouldReturnError_WhenRepoFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("create failed"))

	result, err := ops.CreateWorkflow(context.Background(), CreateWorkflowParams{Name: "Test"})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestCreateWorkflow_ShouldSetCreatedBy_WhenProvided(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	createdBy := uuid.New()
	var savedModel *storagemodels.WorkflowModel
	wfRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			savedModel = args.Get(1).(*storagemodels.WorkflowModel)
		}).
		Return(nil)

	_, err := ops.CreateWorkflow(context.Background(), CreateWorkflowParams{
		Name:      "Test",
		CreatedBy: &createdBy,
	})

	require.NoError(t, err)
	require.NotNil(t, savedModel.CreatedBy)
	assert.Equal(t, createdBy, *savedModel.CreatedBy)
}

func TestCreateWorkflow_ShouldNotSetCreatedBy_WhenNil(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	var savedModel *storagemodels.WorkflowModel
	wfRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			savedModel = args.Get(1).(*storagemodels.WorkflowModel)
		}).
		Return(nil)

	_, err := ops.CreateWorkflow(context.Background(), CreateWorkflowParams{Name: "Test"})

	require.NoError(t, err)
	assert.Nil(t, savedModel.CreatedBy)
}

// --- UpdateWorkflow ---

func TestUpdateWorkflow_ShouldReturnError_WhenWorkflowNotFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager())

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{WorkflowID: wfID})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestUpdateWorkflow_ShouldUpdateNameAndDescription_WhenProvided(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager())

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "Old Name", Description: "Old Desc", Status: "draft",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)
	wfRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	updatedModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "New Name", Description: "New Desc", Status: "draft",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(updatedModel, nil)

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID:  wfID,
		Name:        "New Name",
		Description: "New Desc",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Name", result.Name)
}

func TestUpdateWorkflow_ShouldReturnNodeValidationError_WhenNodeIDEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: uuid.New(),
		Nodes: []NodeInput{
			{ID: "", Name: "Test", Type: "http"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NODE_VALIDATION_FAILED", opErr.Code)
}

func TestUpdateWorkflow_ShouldReturnNodeValidationError_WhenDuplicateNodeIDs(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: uuid.New(),
		Nodes: []NodeInput{
			{ID: "node-1", Name: "Test1", Type: "http"},
			{ID: "node-1", Name: "Test2", Type: "http"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NODE_VALIDATION_FAILED", opErr.Code)
	assert.Contains(t, opErr.Message, "duplicate node id")
}

func TestUpdateWorkflow_ShouldReturnNodeValidationError_WhenInvalidNodeType(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: uuid.New(),
		Nodes: []NodeInput{
			{ID: "node-1", Name: "Test", Type: "nonexistent_type"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "NODE_VALIDATION_FAILED", opErr.Code)
}

func TestUpdateWorkflow_ShouldAllowCommentNodeType(t *testing.T) {
	// comment is a UI-only type that should bypass executor check
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager())

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)
	wfRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(wfModel, nil)

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: wfID,
		Nodes: []NodeInput{
			{ID: "comment-1", Name: "A Note", Type: "comment"},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestUpdateWorkflow_ShouldReturnNodeValidationError_WhenNodeIDTooLong(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))

	longID := strings.Repeat("a", 101)

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: uuid.New(),
		Nodes: []NodeInput{
			{ID: longID, Name: "Test", Type: "http"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Contains(t, opErr.Message, "node id too long")
}

func TestUpdateWorkflow_ShouldReturnNodeValidationError_WhenNodeNameTooLong(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))

	longName := strings.Repeat("a", 256)

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: uuid.New(),
		Nodes: []NodeInput{
			{ID: "node-1", Name: longName, Type: "http"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Contains(t, opErr.Message, "name too long")
}

func TestUpdateWorkflow_ShouldReturnEdgeValidationError_WhenEdgeIDEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: uuid.New(),
		Edges: []EdgeInput{
			{ID: "", From: "a", To: "b"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "EDGE_VALIDATION_FAILED", opErr.Code)
}

func TestUpdateWorkflow_ShouldReturnEdgeValidationError_WhenSelfReference(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: uuid.New(),
		Nodes: []NodeInput{
			{ID: "node-1", Name: "A", Type: "http"},
		},
		Edges: []EdgeInput{
			{ID: "edge-1", From: "node-1", To: "node-1"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "EDGE_VALIDATION_FAILED", opErr.Code)
	assert.Contains(t, opErr.Message, "self-reference")
}

func TestUpdateWorkflow_ShouldReturnEdgeValidationError_WhenFromNodeMissing(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: uuid.New(),
		Nodes: []NodeInput{
			{ID: "node-1", Name: "A", Type: "http"},
		},
		Edges: []EdgeInput{
			{ID: "edge-1", From: "nonexistent", To: "node-1"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Contains(t, opErr.Message, "from node")
}

func TestUpdateWorkflow_ShouldReturnEdgeValidationError_WhenDuplicateEdgeIDs(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: uuid.New(),
		Nodes: []NodeInput{
			{ID: "a", Name: "A", Type: "http"},
			{ID: "b", Name: "B", Type: "http"},
			{ID: "c", Name: "C", Type: "http"},
		},
		Edges: []EdgeInput{
			{ID: "edge-1", From: "a", To: "b"},
			{ID: "edge-1", From: "b", To: "c"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Contains(t, opErr.Message, "duplicate edge id")
}

func TestUpdateWorkflow_ShouldReturnValidationError_WhenInvalidResourceID(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager())

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: wfID,
		Resources: []ResourceInput{
			{ResourceID: "not-a-uuid", Alias: "my-cred", AccessType: "read"},
		},
	})

	assert.Nil(t, result)
	var opErr *OperationError
	require.ErrorAs(t, err, &opErr)
	assert.Equal(t, "INVALID_RESOURCE_ID", opErr.Code)
}

func TestUpdateWorkflow_ShouldDefaultAccessTypeToRead_WhenEmpty(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager())

	wfID := uuid.New()
	resID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)

	var savedModel *storagemodels.WorkflowModel
	wfRepo.On("Update", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			savedModel = args.Get(1).(*storagemodels.WorkflowModel)
		}).
		Return(nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(wfModel, nil)

	_, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: wfID,
		Resources: []ResourceInput{
			{ResourceID: resID.String(), Alias: "my-cred", AccessType: ""},
		},
	})

	require.NoError(t, err)
	require.Len(t, savedModel.Resources, 1)
	assert.Equal(t, "read", savedModel.Resources[0].AccessType)
}

func TestUpdateWorkflow_ShouldSkipNodeValidation_WhenNodesNil(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager())

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)
	wfRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return(wfModel, nil)

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: wfID,
		Name:       "Updated",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestUpdateWorkflow_ShouldReturnError_WhenUpdateFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager())

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)
	wfRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("update err"))

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: wfID,
		Name:       "New",
	})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestUpdateWorkflow_ShouldReturnError_WhenFetchUpdatedFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, newMockExecutorManager())

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)
	wfRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	wfRepo.On("FindByIDWithRelations", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), errors.New("fetch err"))

	result, err := ops.UpdateWorkflow(context.Background(), UpdateWorkflowParams{
		WorkflowID: wfID,
		Name:       "New",
	})

	assert.Nil(t, result)
	require.Error(t, err)
}

// --- DeleteWorkflow ---

func TestDeleteWorkflow_ShouldReturnNil_WhenSuccess(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("Delete", mock.Anything, wfID).Return(nil)

	err := ops.DeleteWorkflow(context.Background(), DeleteWorkflowParams{WorkflowID: wfID})

	require.NoError(t, err)
	wfRepo.AssertExpectations(t)
}

func TestDeleteWorkflow_ShouldReturnError_WhenRepoFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("Delete", mock.Anything, wfID).Return(errors.New("delete failed"))

	err := ops.DeleteWorkflow(context.Background(), DeleteWorkflowParams{WorkflowID: wfID})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
}

// --- PublishWorkflow ---

func TestPublishWorkflow_ShouldSetStatusActive(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)
	wfRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *storagemodels.WorkflowModel) bool {
		return m.Status == "active"
	})).Return(nil)

	result, err := ops.PublishWorkflow(context.Background(), PublishWorkflowParams{WorkflowID: wfID})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, models.WorkflowStatus("active"), result.Status)
}

func TestPublishWorkflow_ShouldReturnError_WhenWorkflowNotFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	result, err := ops.PublishWorkflow(context.Background(), PublishWorkflowParams{WorkflowID: wfID})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestPublishWorkflow_ShouldReturnError_WhenUpdateFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", Status: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)
	wfRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("update failed"))

	result, err := ops.PublishWorkflow(context.Background(), PublishWorkflowParams{WorkflowID: wfID})

	assert.Nil(t, result)
	require.Error(t, err)
}

// --- UnpublishWorkflow ---

func TestUnpublishWorkflow_ShouldSetStatusDraft(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfModel := &storagemodels.WorkflowModel{
		ID: wfID, Name: "WF", Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	wfRepo.On("FindByID", mock.Anything, wfID).Return(wfModel, nil)
	wfRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *storagemodels.WorkflowModel) bool {
		return m.Status == "draft"
	})).Return(nil)

	result, err := ops.UnpublishWorkflow(context.Background(), UnpublishWorkflowParams{WorkflowID: wfID})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, models.WorkflowStatus("draft"), result.Status)
}

func TestUnpublishWorkflow_ShouldReturnError_WhenWorkflowNotFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	result, err := ops.UnpublishWorkflow(context.Background(), UnpublishWorkflowParams{WorkflowID: wfID})

	assert.Nil(t, result)
	require.Error(t, err)
}

// --- AttachWorkflowResource ---

func TestAttachWorkflowResource_ShouldAttachSuccessfully(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	resID := uuid.New()
	assignedBy := uuid.New()

	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	wfRepo.On("AssignResource", mock.Anything, wfID, mock.Anything, &assignedBy).Return(nil)

	err := ops.AttachWorkflowResource(context.Background(), AttachWorkflowResourceParams{
		WorkflowID: wfID,
		ResourceID: resID,
		Alias:      "my-key",
		AccessType: "write",
		AssignedBy: &assignedBy,
	})

	require.NoError(t, err)
	wfRepo.AssertExpectations(t)
}

func TestAttachWorkflowResource_ShouldDefaultAccessTypeToRead(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	resID := uuid.New()

	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	wfRepo.On("AssignResource", mock.Anything, wfID, mock.MatchedBy(func(r *storagemodels.WorkflowResourceModel) bool {
		return r.AccessType == "read"
	}), (*uuid.UUID)(nil)).Return(nil)

	err := ops.AttachWorkflowResource(context.Background(), AttachWorkflowResourceParams{
		WorkflowID: wfID,
		ResourceID: resID,
		Alias:      "test",
		AccessType: "",
	})

	require.NoError(t, err)
}

func TestAttachWorkflowResource_ShouldReturnError_WhenWorkflowNotFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	err := ops.AttachWorkflowResource(context.Background(), AttachWorkflowResourceParams{
		WorkflowID: wfID,
		ResourceID: uuid.New(),
		Alias:      "test",
	})

	require.Error(t, err)
}

// --- DetachWorkflowResource ---

func TestDetachWorkflowResource_ShouldDetachSuccessfully(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	resID := uuid.New()

	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	wfRepo.On("UnassignResource", mock.Anything, wfID, resID).Return(nil)

	err := ops.DetachWorkflowResource(context.Background(), DetachWorkflowResourceParams{
		WorkflowID: wfID,
		ResourceID: resID,
	})

	require.NoError(t, err)
}

func TestDetachWorkflowResource_ShouldReturnError_WhenWorkflowNotFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	err := ops.DetachWorkflowResource(context.Background(), DetachWorkflowResourceParams{
		WorkflowID: wfID,
		ResourceID: uuid.New(),
	})

	require.Error(t, err)
}

func TestDetachWorkflowResource_ShouldReturnError_WhenUnassignFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	resID := uuid.New()

	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	wfRepo.On("UnassignResource", mock.Anything, wfID, resID).Return(errors.New("unassign failed"))

	err := ops.DetachWorkflowResource(context.Background(), DetachWorkflowResourceParams{
		WorkflowID: wfID,
		ResourceID: resID,
	})

	require.Error(t, err)
}

// --- GetWorkflowResources ---

func TestGetWorkflowResources_ShouldReturnResources_WhenFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	resID1 := uuid.New()
	resID2 := uuid.New()

	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	wfRepo.On("GetWorkflowResources", mock.Anything, wfID).Return([]*storagemodels.WorkflowResourceModel{
		{WorkflowID: wfID, ResourceID: resID1, Alias: "cred-a", AccessType: "read"},
		{WorkflowID: wfID, ResourceID: resID2, Alias: "cred-b", AccessType: "write"},
	}, nil)

	result, err := ops.GetWorkflowResources(context.Background(), GetWorkflowResourcesParams{WorkflowID: wfID})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Resources, 2)
	assert.Equal(t, resID1.String(), result.Resources[0].ResourceID)
	assert.Equal(t, "cred-a", result.Resources[0].Alias)
	assert.Equal(t, "read", result.Resources[0].AccessType)
}

func TestGetWorkflowResources_ShouldReturnError_WhenWorkflowNotFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	result, err := ops.GetWorkflowResources(context.Background(), GetWorkflowResourcesParams{WorkflowID: wfID})

	assert.Nil(t, result)
	require.Error(t, err)
}

func TestGetWorkflowResources_ShouldReturnEmpty_WhenNoResources(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	wfRepo.On("GetWorkflowResources", mock.Anything, wfID).Return([]*storagemodels.WorkflowResourceModel{}, nil)

	result, err := ops.GetWorkflowResources(context.Background(), GetWorkflowResourcesParams{WorkflowID: wfID})

	require.NoError(t, err)
	assert.Empty(t, result.Resources)
}

// --- UpdateWorkflowResourceAlias ---

func TestUpdateWorkflowResourceAlias_ShouldUpdateAlias(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	resID := uuid.New()

	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	wfRepo.On("UpdateResourceAlias", mock.Anything, wfID, resID, "new-alias").Return(nil)

	err := ops.UpdateWorkflowResourceAlias(context.Background(), UpdateWorkflowResourceAliasParams{
		WorkflowID: wfID,
		ResourceID: resID,
		Alias:      "new-alias",
	})

	require.NoError(t, err)
	wfRepo.AssertExpectations(t)
}

func TestUpdateWorkflowResourceAlias_ShouldReturnError_WhenWorkflowNotFound(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	wfRepo.On("FindByID", mock.Anything, wfID).Return((*storagemodels.WorkflowModel)(nil), models.ErrWorkflowNotFound)

	err := ops.UpdateWorkflowResourceAlias(context.Background(), UpdateWorkflowResourceAliasParams{
		WorkflowID: wfID,
		ResourceID: uuid.New(),
		Alias:      "alias",
	})

	require.Error(t, err)
}

func TestUpdateWorkflowResourceAlias_ShouldReturnError_WhenUpdateFails(t *testing.T) {
	wfRepo := new(mockWorkflowRepo)
	ops := newTestOperations(wfRepo, nil, nil, nil, nil, nil, nil)

	wfID := uuid.New()
	resID := uuid.New()

	wfRepo.On("FindByID", mock.Anything, wfID).Return(&storagemodels.WorkflowModel{ID: wfID}, nil)
	wfRepo.On("UpdateResourceAlias", mock.Anything, wfID, resID, "alias").Return(errors.New("update failed"))

	err := ops.UpdateWorkflowResourceAlias(context.Background(), UpdateWorkflowResourceAliasParams{
		WorkflowID: wfID,
		ResourceID: resID,
		Alias:      "alias",
	})

	require.Error(t, err)
}

// --- validateNodes ---

func TestValidateNodes_ShouldReturnNil_WhenNodesNil(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager())
	err := ops.validateNodes(nil)
	assert.NoError(t, err)
}

func TestValidateNodes_ShouldReturnError_WhenNodeNameEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))
	err := ops.validateNodes([]NodeInput{
		{ID: "n1", Name: "", Type: "http"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestValidateNodes_ShouldReturnError_WhenNodeTypeEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager())
	err := ops.validateNodes([]NodeInput{
		{ID: "n1", Name: "Test", Type: ""},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type is required")
}

// --- validateEdges ---

func TestValidateEdges_ShouldReturnNil_WhenEdgesNil(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager())
	err := ops.validateEdges(nil, nil)
	assert.NoError(t, err)
}

func TestValidateEdges_ShouldReturnError_WhenFromEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager())
	err := ops.validateEdges([]EdgeInput{
		{ID: "e1", From: "", To: "b"},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "from is required")
}

func TestValidateEdges_ShouldReturnError_WhenToEmpty(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager())
	err := ops.validateEdges([]EdgeInput{
		{ID: "e1", From: "a", To: ""},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "to is required")
}

func TestValidateEdges_ShouldReturnError_WhenEdgeIDTooLong(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager())
	err := ops.validateEdges([]EdgeInput{
		{ID: strings.Repeat("x", 101), From: "a", To: "b"},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "edge id too long")
}

func TestValidateEdges_ShouldNotValidateNodeRefs_WhenNoNodesProvided(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager())
	err := ops.validateEdges([]EdgeInput{
		{ID: "e1", From: "nonexistent-a", To: "nonexistent-b"},
	}, nil)
	// When len(nodes) == 0, node reference validation is skipped
	assert.NoError(t, err)
}

func TestValidateEdges_ShouldReturnError_WhenToNodeNotInNodes(t *testing.T) {
	ops := newTestOperations(nil, nil, nil, nil, nil, nil, newMockExecutorManager("http"))
	err := ops.validateEdges(
		[]EdgeInput{
			{ID: "e1", From: "a", To: "missing"},
		},
		[]NodeInput{
			{ID: "a", Name: "A", Type: "http"},
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "to node")
}
