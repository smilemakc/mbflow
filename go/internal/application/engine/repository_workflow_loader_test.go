package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// --- Mock: WorkflowRepository (local to engine package tests) ---

type mockEngineWorkflowRepo struct {
	mock.Mock
}

func (m *mockEngineWorkflowRepo) Create(ctx context.Context, workflow *storagemodels.WorkflowModel) error {
	return m.Called(ctx, workflow).Error(0)
}

func (m *mockEngineWorkflowRepo) Update(ctx context.Context, workflow *storagemodels.WorkflowModel) error {
	return m.Called(ctx, workflow).Error(0)
}

func (m *mockEngineWorkflowRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockEngineWorkflowRepo) HardDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockEngineWorkflowRepo) FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, id)
	wm, _ := args.Get(0).(*storagemodels.WorkflowModel)
	return wm, args.Error(1)
}

func (m *mockEngineWorkflowRepo) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, id)
	wm, _ := args.Get(0).(*storagemodels.WorkflowModel)
	return wm, args.Error(1)
}

func (m *mockEngineWorkflowRepo) FindByName(ctx context.Context, name string, version int) (*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, name, version)
	wm, _ := args.Get(0).(*storagemodels.WorkflowModel)
	return wm, args.Error(1)
}

func (m *mockEngineWorkflowRepo) FindAll(ctx context.Context, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, limit, offset)
	wms, _ := args.Get(0).([]*storagemodels.WorkflowModel)
	return wms, args.Error(1)
}

func (m *mockEngineWorkflowRepo) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, status, limit, offset)
	wms, _ := args.Get(0).([]*storagemodels.WorkflowModel)
	return wms, args.Error(1)
}

func (m *mockEngineWorkflowRepo) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *mockEngineWorkflowRepo) CountByStatus(ctx context.Context, status string) (int, error) {
	args := m.Called(ctx, status)
	return args.Int(0), args.Error(1)
}

func (m *mockEngineWorkflowRepo) FindAllWithFilters(ctx context.Context, filters repository.WorkflowFilters, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, filters, limit, offset)
	wms, _ := args.Get(0).([]*storagemodels.WorkflowModel)
	return wms, args.Error(1)
}

func (m *mockEngineWorkflowRepo) CountWithFilters(ctx context.Context, filters repository.WorkflowFilters) (int, error) {
	args := m.Called(ctx, filters)
	return args.Int(0), args.Error(1)
}

func (m *mockEngineWorkflowRepo) CreateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	return m.Called(ctx, node).Error(0)
}

func (m *mockEngineWorkflowRepo) UpdateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	return m.Called(ctx, node).Error(0)
}

func (m *mockEngineWorkflowRepo) DeleteNode(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockEngineWorkflowRepo) FindNodeByID(ctx context.Context, id uuid.UUID) (*storagemodels.NodeModel, error) {
	args := m.Called(ctx, id)
	nm, _ := args.Get(0).(*storagemodels.NodeModel)
	return nm, args.Error(1)
}

func (m *mockEngineWorkflowRepo) FindNodesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.NodeModel, error) {
	args := m.Called(ctx, workflowID)
	nms, _ := args.Get(0).([]*storagemodels.NodeModel)
	return nms, args.Error(1)
}

func (m *mockEngineWorkflowRepo) CreateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	return m.Called(ctx, edge).Error(0)
}

func (m *mockEngineWorkflowRepo) UpdateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	return m.Called(ctx, edge).Error(0)
}

func (m *mockEngineWorkflowRepo) DeleteEdge(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockEngineWorkflowRepo) FindEdgeByID(ctx context.Context, id uuid.UUID) (*storagemodels.EdgeModel, error) {
	args := m.Called(ctx, id)
	em, _ := args.Get(0).(*storagemodels.EdgeModel)
	return em, args.Error(1)
}

func (m *mockEngineWorkflowRepo) FindEdgesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.EdgeModel, error) {
	args := m.Called(ctx, workflowID)
	ems, _ := args.Get(0).([]*storagemodels.EdgeModel)
	return ems, args.Error(1)
}

func (m *mockEngineWorkflowRepo) ValidateDAG(ctx context.Context, workflowID uuid.UUID) error {
	return m.Called(ctx, workflowID).Error(0)
}

func (m *mockEngineWorkflowRepo) AssignResource(ctx context.Context, workflowID uuid.UUID, resource *storagemodels.WorkflowResourceModel, assignedBy *uuid.UUID) error {
	return m.Called(ctx, workflowID, resource, assignedBy).Error(0)
}

func (m *mockEngineWorkflowRepo) UnassignResource(ctx context.Context, workflowID, resourceID uuid.UUID) error {
	return m.Called(ctx, workflowID, resourceID).Error(0)
}

func (m *mockEngineWorkflowRepo) UnassignResourceFromAllWorkflows(ctx context.Context, resourceID uuid.UUID) (int64, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockEngineWorkflowRepo) GetWorkflowResources(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.WorkflowResourceModel, error) {
	args := m.Called(ctx, workflowID)
	wrs, _ := args.Get(0).([]*storagemodels.WorkflowResourceModel)
	return wrs, args.Error(1)
}

func (m *mockEngineWorkflowRepo) UpdateResourceAlias(ctx context.Context, workflowID, resourceID uuid.UUID, newAlias string) error {
	return m.Called(ctx, workflowID, resourceID, newAlias).Error(0)
}

func (m *mockEngineWorkflowRepo) ResourceExists(ctx context.Context, workflowID, resourceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, workflowID, resourceID)
	return args.Bool(0), args.Error(1)
}

func (m *mockEngineWorkflowRepo) GetResourceByAlias(ctx context.Context, workflowID uuid.UUID, alias string) (*storagemodels.WorkflowResourceModel, error) {
	args := m.Called(ctx, workflowID, alias)
	wrm, _ := args.Get(0).(*storagemodels.WorkflowResourceModel)
	return wrm, args.Error(1)
}

// Compile-time interface check.
var _ repository.WorkflowRepository = (*mockEngineWorkflowRepo)(nil)

// --- Tests ---

func TestRepositoryWorkflowLoader_LoadWorkflow_Success(t *testing.T) {
	// Arrange
	wfID := uuid.New()
	nodeID := uuid.New()

	workflowModel := &storagemodels.WorkflowModel{
		ID:          wfID,
		Name:        "My Workflow",
		Description: "Integration workflow",
		Status:      "active",
		Version:     2,
		Variables:   storagemodels.JSONBMap{"timeout": float64(30)},
		Nodes: []*storagemodels.NodeModel{
			{
				ID:     nodeID,
				NodeID: "node-start",
				Name:   "Start",
				Type:   "http",
				Config: storagemodels.JSONBMap{"url": "https://example.com"},
			},
		},
		Edges: []*storagemodels.EdgeModel{
			{
				EdgeID:       "edge-1",
				FromNodeID:   "node-start",
				ToNodeID:     "node-end",
				SourceHandle: "true",
				Loop:         storagemodels.JSONBMap{"max_iterations": float64(3)},
			},
		},
	}

	repo := new(mockEngineWorkflowRepo)
	repo.
		On("FindByIDWithRelations", mock.Anything, wfID).
		Return(workflowModel, nil).
		Once()

	loader := NewRepositoryWorkflowLoader(repo)

	// Act
	result, err := loader.LoadWorkflow(context.Background(), wfID.String())

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, wfID.String(), result.ID)
	assert.Equal(t, "My Workflow", result.Name)
	assert.Equal(t, "Integration workflow", result.Description)
	assert.Equal(t, models.WorkflowStatus("active"), result.Status)

	require.Len(t, result.Nodes, 1)
	assert.Equal(t, "node-start", result.Nodes[0].ID)
	assert.Equal(t, "Start", result.Nodes[0].Name)
	assert.Equal(t, "http", result.Nodes[0].Type)

	require.Len(t, result.Edges, 1)
	assert.Equal(t, "edge-1", result.Edges[0].ID)
	assert.Equal(t, "node-start", result.Edges[0].From)
	assert.Equal(t, "node-end", result.Edges[0].To)
	assert.Equal(t, "true", result.Edges[0].SourceHandle)
	require.NotNil(t, result.Edges[0].Loop)
	assert.Equal(t, 3, result.Edges[0].Loop.MaxIterations)

	repo.AssertExpectations(t)
}

func TestRepositoryWorkflowLoader_LoadWorkflow_InvalidID(t *testing.T) {
	// Arrange
	repo := new(mockEngineWorkflowRepo)
	loader := NewRepositoryWorkflowLoader(repo)

	// Act
	result, err := loader.LoadWorkflow(context.Background(), "not-a-uuid")

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid workflow ID")

	repo.AssertNotCalled(t, "FindByIDWithRelations", mock.Anything, mock.Anything)
}

func TestRepositoryWorkflowLoader_LoadWorkflow_NotFound(t *testing.T) {
	// Arrange
	wfID := uuid.New()
	repoErr := errors.New("workflow not found")

	repo := new(mockEngineWorkflowRepo)
	repo.
		On("FindByIDWithRelations", mock.Anything, wfID).
		Return((*storagemodels.WorkflowModel)(nil), repoErr).
		Once()

	loader := NewRepositoryWorkflowLoader(repo)

	// Act
	result, err := loader.LoadWorkflow(context.Background(), wfID.String())

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, repoErr)

	repo.AssertExpectations(t)
}

func TestRepositoryWorkflowLoader_LoadWorkflow_WithLoopEdge(t *testing.T) {
	// Arrange
	wfID := uuid.New()

	workflowModel := &storagemodels.WorkflowModel{
		ID:     wfID,
		Name:   "Loop Workflow",
		Status: "active",
		Nodes: []*storagemodels.NodeModel{
			{NodeID: "node-a", Name: "A", Type: "task"},
			{NodeID: "node-b", Name: "B", Type: "task"},
		},
		Edges: []*storagemodels.EdgeModel{
			{
				EdgeID:     "loop-edge",
				FromNodeID: "node-b",
				ToNodeID:   "node-a",
				// max_iterations stored as float64 in JSONBMap (JSON numeric type)
				Loop: storagemodels.JSONBMap{"max_iterations": float64(5)},
			},
		},
	}

	repo := new(mockEngineWorkflowRepo)
	repo.
		On("FindByIDWithRelations", mock.Anything, wfID).
		Return(workflowModel, nil).
		Once()

	loader := NewRepositoryWorkflowLoader(repo)

	// Act
	result, err := loader.LoadWorkflow(context.Background(), wfID.String())

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Edges, 1)

	edge := result.Edges[0]
	assert.Equal(t, "loop-edge", edge.ID)
	require.NotNil(t, edge.Loop, "loop edge must have a Loop config")
	assert.Equal(t, 5, edge.Loop.MaxIterations)
	assert.True(t, edge.IsLoop())

	repo.AssertExpectations(t)
}

func TestRepositoryWorkflowLoader_LoadWorkflow_WithSourceHandle(t *testing.T) {
	// Arrange
	wfID := uuid.New()

	workflowModel := &storagemodels.WorkflowModel{
		ID:     wfID,
		Name:   "Conditional Workflow",
		Status: "active",
		Nodes: []*storagemodels.NodeModel{
			{NodeID: "node-cond", Name: "Condition", Type: "condition"},
			{NodeID: "node-false", Name: "FalseBranch", Type: "task"},
		},
		Edges: []*storagemodels.EdgeModel{
			{
				EdgeID:       "edge-false-branch",
				FromNodeID:   "node-cond",
				ToNodeID:     "node-false",
				SourceHandle: "false",
			},
		},
	}

	repo := new(mockEngineWorkflowRepo)
	repo.
		On("FindByIDWithRelations", mock.Anything, wfID).
		Return(workflowModel, nil).
		Once()

	loader := NewRepositoryWorkflowLoader(repo)

	// Act
	result, err := loader.LoadWorkflow(context.Background(), wfID.String())

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Edges, 1)

	edge := result.Edges[0]
	assert.Equal(t, "edge-false-branch", edge.ID)
	assert.Equal(t, "false", edge.SourceHandle)
	assert.Nil(t, edge.Loop)

	repo.AssertExpectations(t)
}
