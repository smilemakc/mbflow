package trigger

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestNewManager_Success tests successful manager creation
func TestNewManager_Success(t *testing.T) {
	triggerRepo := &mockTriggerRepo{}
	workflowRepo := &mockWorkflowRepo{}
	redisCache := &cache.RedisCache{}

	cfg := ManagerConfig{
		TriggerRepo:  triggerRepo,
		WorkflowRepo: workflowRepo,
		ExecutionMgr: nil, // Will fail validation
		Cache:        redisCache,
	}

	_, err := NewManager(cfg)
	// Should fail because ExecutionMgr is nil
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution manager is required")
}

// TestNewManager_MissingTriggerRepo tests that trigger repo is required
func TestNewManager_MissingTriggerRepo(t *testing.T) {
	cfg := ManagerConfig{
		TriggerRepo:  nil,
		WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: nil,
		Cache:        &cache.RedisCache{},
	}

	_, err := NewManager(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trigger repository is required")
}

// TestNewManager_MissingWorkflowRepo tests that workflow repo is required
func TestNewManager_MissingWorkflowRepo(t *testing.T) {
	cfg := ManagerConfig{
		TriggerRepo:  &mockTriggerRepo{},
		WorkflowRepo: nil,
		ExecutionMgr: nil,
		Cache:        &cache.RedisCache{},
	}

	_, err := NewManager(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow repository is required")
}

// TestNewManager_MissingExecutionMgr tests that execution manager is required
func TestNewManager_MissingExecutionMgr(t *testing.T) {
	cfg := ManagerConfig{
		TriggerRepo:  &mockTriggerRepo{},
		WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: nil,
		Cache:        &cache.RedisCache{},
	}

	_, err := NewManager(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution manager is required")
}

// Note: Additional Manager tests are in manager_lifecycle_test.go
// which uses real ExecutionManager instances as ExecutionManager is a concrete type

// Additional tests for repository mocks to ensure they compile
func TestMockRepositories_CompileCheck(t *testing.T) {
	// This test just verifies that our mocks compile correctly
	triggerRepo := &mockTriggerRepo{}
	workflowRepo := &mockWorkflowRepo{}

	// Setup some mock expectations
	triggerRepo.On("FindEnabled", mock.Anything).Return([]*storagemodels.TriggerModel{}, nil)
	workflowRepo.On("FindByID", mock.Anything, mock.Anything).Return(nil, nil)

	// Call methods to verify signatures
	_, _ = triggerRepo.FindEnabled(context.Background())
	_, _ = workflowRepo.FindByID(context.Background(), uuid.New())

	// Verify expectations were met
	triggerRepo.AssertExpectations(t)
	workflowRepo.AssertExpectations(t)
}

// TestTriggerRepository_AllMethods tests all trigger repo methods compile
func TestTriggerRepository_AllMethods(t *testing.T) {
	repo := &mockTriggerRepo{}
	ctx := context.Background()
	id := uuid.New()
	triggerType := "cron"

	// Setup mocks
	repo.On("Create", ctx, mock.Anything).Return(nil)
	repo.On("FindByID", ctx, id).Return(nil, nil)
	repo.On("FindByWorkflowID", ctx, id).Return([]*storagemodels.TriggerModel{}, nil)
	repo.On("FindByType", ctx, triggerType, 10, 0).Return([]*storagemodels.TriggerModel{}, nil)
	repo.On("FindEnabled", ctx).Return([]*storagemodels.TriggerModel{}, nil)
	repo.On("FindEnabledByType", ctx, triggerType).Return([]*storagemodels.TriggerModel{}, nil)
	repo.On("FindAll", ctx, 10, 0).Return([]*storagemodels.TriggerModel{}, nil)
	repo.On("Update", ctx, mock.Anything).Return(nil)
	repo.On("Delete", ctx, id).Return(nil)
	repo.On("MarkTriggered", ctx, id).Return(nil)
	repo.On("Count", ctx).Return(0, nil)
	repo.On("CountByWorkflowID", ctx, id).Return(0, nil)
	repo.On("CountByType", ctx, triggerType).Return(0, nil)
	repo.On("Enable", ctx, id).Return(nil)
	repo.On("Disable", ctx, id).Return(nil)

	// Call all methods
	_ = repo.Create(ctx, &storagemodels.TriggerModel{})
	_, _ = repo.FindByID(ctx, id)
	_, _ = repo.FindByWorkflowID(ctx, id)
	_, _ = repo.FindByType(ctx, triggerType, 10, 0)
	_, _ = repo.FindEnabled(ctx)
	_, _ = repo.FindEnabledByType(ctx, triggerType)
	_, _ = repo.FindAll(ctx, 10, 0)
	_ = repo.Update(ctx, &storagemodels.TriggerModel{})
	_ = repo.Delete(ctx, id)
	_ = repo.MarkTriggered(ctx, id)
	_, _ = repo.Count(ctx)
	_, _ = repo.CountByWorkflowID(ctx, id)
	_, _ = repo.CountByType(ctx, triggerType)
	_ = repo.Enable(ctx, id)
	_ = repo.Disable(ctx, id)

	repo.AssertExpectations(t)
}

// TestWorkflowRepository_AllMethods tests all workflow repo methods compile
func TestWorkflowRepository_AllMethods(t *testing.T) {
	repo := &mockWorkflowRepo{}
	ctx := context.Background()
	id := uuid.New()
	status := "active"

	// Setup mocks for all methods
	repo.On("Create", ctx, mock.Anything).Return(nil)
	repo.On("FindByID", ctx, id).Return(nil, nil)
	repo.On("FindByIDWithRelations", ctx, id).Return(nil, nil)
	repo.On("FindByName", ctx, "test", 1).Return(nil, nil)
	repo.On("FindAll", ctx, 10, 0).Return([]*storagemodels.WorkflowModel{}, nil)
	repo.On("FindByStatus", ctx, status, 10, 0).Return([]*storagemodels.WorkflowModel{}, nil)
	repo.On("Update", ctx, mock.Anything).Return(nil)
	repo.On("Delete", ctx, id).Return(nil)
	repo.On("HardDelete", ctx, id).Return(nil)
	repo.On("Count", ctx).Return(0, nil)
	repo.On("CountByStatus", ctx, status).Return(0, nil)
	repo.On("CreateNode", ctx, mock.Anything).Return(nil)
	repo.On("UpdateNode", ctx, mock.Anything).Return(nil)
	repo.On("DeleteNode", ctx, id).Return(nil)
	repo.On("FindNodeByID", ctx, id).Return(nil, nil)
	repo.On("FindNodesByWorkflowID", ctx, id).Return([]*storagemodels.NodeModel{}, nil)
	repo.On("CreateEdge", ctx, mock.Anything).Return(nil)
	repo.On("UpdateEdge", ctx, mock.Anything).Return(nil)
	repo.On("DeleteEdge", ctx, id).Return(nil)
	repo.On("FindEdgeByID", ctx, id).Return(nil, nil)
	repo.On("FindEdgesByWorkflowID", ctx, id).Return([]*storagemodels.EdgeModel{}, nil)
	repo.On("ValidateDAG", ctx, id).Return(nil)

	// Call all methods
	_ = repo.Create(ctx, &storagemodels.WorkflowModel{})
	_, _ = repo.FindByID(ctx, id)
	_, _ = repo.FindByIDWithRelations(ctx, id)
	_, _ = repo.FindByName(ctx, "test", 1)
	_, _ = repo.FindAll(ctx, 10, 0)
	_, _ = repo.FindByStatus(ctx, status, 10, 0)
	_ = repo.Update(ctx, &storagemodels.WorkflowModel{})
	_ = repo.Delete(ctx, id)
	_ = repo.HardDelete(ctx, id)
	_, _ = repo.Count(ctx)
	_, _ = repo.CountByStatus(ctx, status)
	_ = repo.CreateNode(ctx, &storagemodels.NodeModel{})
	_ = repo.UpdateNode(ctx, &storagemodels.NodeModel{})
	_ = repo.DeleteNode(ctx, id)
	_, _ = repo.FindNodeByID(ctx, id)
	_, _ = repo.FindNodesByWorkflowID(ctx, id)
	_ = repo.CreateEdge(ctx, &storagemodels.EdgeModel{})
	_ = repo.UpdateEdge(ctx, &storagemodels.EdgeModel{})
	_ = repo.DeleteEdge(ctx, id)
	_, _ = repo.FindEdgeByID(ctx, id)
	_, _ = repo.FindEdgesByWorkflowID(ctx, id)
	_ = repo.ValidateDAG(ctx, id)

	repo.AssertExpectations(t)
}
