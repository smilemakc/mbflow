package trigger

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
)

// Mock repositories for testing
type mockTriggerRepo struct {
	mock.Mock
}

func (m *mockTriggerRepo) Create(ctx context.Context, trigger *storagemodels.TriggerModel) error {
	args := m.Called(ctx, trigger)
	return args.Error(0)
}

func (m *mockTriggerRepo) FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storagemodels.TriggerModel), args.Error(1)
}

func (m *mockTriggerRepo) FindByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.TriggerModel), args.Error(1)
}

func (m *mockTriggerRepo) FindByType(ctx context.Context, triggerType string, limit, offset int) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, triggerType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.TriggerModel), args.Error(1)
}

func (m *mockTriggerRepo) FindEnabled(ctx context.Context) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.TriggerModel), args.Error(1)
}

func (m *mockTriggerRepo) Update(ctx context.Context, trigger *storagemodels.TriggerModel) error {
	args := m.Called(ctx, trigger)
	return args.Error(0)
}

func (m *mockTriggerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockTriggerRepo) MarkTriggered(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockTriggerRepo) FindEnabledByType(ctx context.Context, triggerType string) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, triggerType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.TriggerModel), args.Error(1)
}

func (m *mockTriggerRepo) FindAll(ctx context.Context, limit, offset int) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.TriggerModel), args.Error(1)
}

func (m *mockTriggerRepo) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *mockTriggerRepo) CountByWorkflowID(ctx context.Context, workflowID uuid.UUID) (int, error) {
	args := m.Called(ctx, workflowID)
	return args.Int(0), args.Error(1)
}

func (m *mockTriggerRepo) CountByType(ctx context.Context, triggerType string) (int, error) {
	args := m.Called(ctx, triggerType)
	return args.Int(0), args.Error(1)
}

func (m *mockTriggerRepo) Enable(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockTriggerRepo) Disable(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockWorkflowRepo struct {
	mock.Mock
}

func (m *mockWorkflowRepo) Create(ctx context.Context, workflow *storagemodels.WorkflowModel) error {
	args := m.Called(ctx, workflow)
	return args.Error(0)
}

func (m *mockWorkflowRepo) FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storagemodels.WorkflowModel), args.Error(1)
}

func (m *mockWorkflowRepo) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storagemodels.WorkflowModel), args.Error(1)
}

func (m *mockWorkflowRepo) FindAll(ctx context.Context, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.WorkflowModel), args.Error(1)
}

func (m *mockWorkflowRepo) Update(ctx context.Context, workflow *storagemodels.WorkflowModel) error {
	args := m.Called(ctx, workflow)
	return args.Error(0)
}

func (m *mockWorkflowRepo) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockWorkflowRepo) HardDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockWorkflowRepo) FindByName(ctx context.Context, name string, version int) (*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, name, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storagemodels.WorkflowModel), args.Error(1)
}

func (m *mockWorkflowRepo) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, status, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.WorkflowModel), args.Error(1)
}

func (m *mockWorkflowRepo) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *mockWorkflowRepo) CountByStatus(ctx context.Context, status string) (int, error) {
	args := m.Called(ctx, status)
	return args.Int(0), args.Error(1)
}

func (m *mockWorkflowRepo) FindAllWithFilters(ctx context.Context, filters repository.WorkflowFilters, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.WorkflowModel), args.Error(1)
}

func (m *mockWorkflowRepo) CountWithFilters(ctx context.Context, filters repository.WorkflowFilters) (int, error) {
	args := m.Called(ctx, filters)
	return args.Int(0), args.Error(1)
}

func (m *mockWorkflowRepo) CreateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	args := m.Called(ctx, node)
	return args.Error(0)
}

func (m *mockWorkflowRepo) UpdateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	args := m.Called(ctx, node)
	return args.Error(0)
}

func (m *mockWorkflowRepo) DeleteNode(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockWorkflowRepo) FindNodeByID(ctx context.Context, id uuid.UUID) (*storagemodels.NodeModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storagemodels.NodeModel), args.Error(1)
}

func (m *mockWorkflowRepo) FindNodesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.NodeModel, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.NodeModel), args.Error(1)
}

func (m *mockWorkflowRepo) CreateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	args := m.Called(ctx, edge)
	return args.Error(0)
}

func (m *mockWorkflowRepo) UpdateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	args := m.Called(ctx, edge)
	return args.Error(0)
}

func (m *mockWorkflowRepo) DeleteEdge(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockWorkflowRepo) FindEdgeByID(ctx context.Context, id uuid.UUID) (*storagemodels.EdgeModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storagemodels.EdgeModel), args.Error(1)
}

func (m *mockWorkflowRepo) FindEdgesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.EdgeModel, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.EdgeModel), args.Error(1)
}

func (m *mockWorkflowRepo) ValidateDAG(ctx context.Context, workflowID uuid.UUID) error {
	args := m.Called(ctx, workflowID)
	return args.Error(0)
}

// Resource management methods
func (m *mockWorkflowRepo) AssignResource(ctx context.Context, workflowID uuid.UUID, resource *storagemodels.WorkflowResourceModel, assignedBy *uuid.UUID) error {
	args := m.Called(ctx, workflowID, resource, assignedBy)
	return args.Error(0)
}

func (m *mockWorkflowRepo) UnassignResource(ctx context.Context, workflowID, resourceID uuid.UUID) error {
	args := m.Called(ctx, workflowID, resourceID)
	return args.Error(0)
}

func (m *mockWorkflowRepo) UnassignResourceFromAllWorkflows(ctx context.Context, resourceID uuid.UUID) (int64, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockWorkflowRepo) GetWorkflowResources(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.WorkflowResourceModel, error) {
	args := m.Called(ctx, workflowID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.WorkflowResourceModel), args.Error(1)
}

func (m *mockWorkflowRepo) UpdateResourceAlias(ctx context.Context, workflowID, resourceID uuid.UUID, newAlias string) error {
	args := m.Called(ctx, workflowID, resourceID, newAlias)
	return args.Error(0)
}

func (m *mockWorkflowRepo) ResourceExists(ctx context.Context, workflowID, resourceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, workflowID, resourceID)
	return args.Bool(0), args.Error(1)
}

func (m *mockWorkflowRepo) GetResourceByAlias(ctx context.Context, workflowID uuid.UUID, alias string) (*storagemodels.WorkflowResourceModel, error) {
	args := m.Called(ctx, workflowID, alias)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storagemodels.WorkflowResourceModel), args.Error(1)
}

// Compile-time interface checks
var _ repository.TriggerRepository = (*mockTriggerRepo)(nil)
var _ repository.WorkflowRepository = (*mockWorkflowRepo)(nil)

func (m *mockTriggerRepo) FindWithFilters(ctx context.Context, filters repository.TriggerFilters, limit, offset int) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storagemodels.TriggerModel), args.Error(1)
}

func (m *mockTriggerRepo) CountWithFilters(ctx context.Context, filters repository.TriggerFilters) (int, error) {
	args := m.Called(ctx, filters)
	return args.Int(0), args.Error(1)
}
