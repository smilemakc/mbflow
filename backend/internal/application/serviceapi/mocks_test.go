package serviceapi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/application/systemkey"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/crypto"
	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// --- Mock: WorkflowRepository ---

type mockWorkflowRepo struct {
	mock.Mock
}

func (m *mockWorkflowRepo) Create(ctx context.Context, workflow *storagemodels.WorkflowModel) error {
	return m.Called(ctx, workflow).Error(0)
}

func (m *mockWorkflowRepo) Update(ctx context.Context, workflow *storagemodels.WorkflowModel) error {
	return m.Called(ctx, workflow).Error(0)
}

func (m *mockWorkflowRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockWorkflowRepo) HardDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockWorkflowRepo) FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, id)
	wm, _ := args.Get(0).(*storagemodels.WorkflowModel)
	return wm, args.Error(1)
}

func (m *mockWorkflowRepo) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, id)
	wm, _ := args.Get(0).(*storagemodels.WorkflowModel)
	return wm, args.Error(1)
}

func (m *mockWorkflowRepo) FindByName(ctx context.Context, name string, version int) (*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, name, version)
	wm, _ := args.Get(0).(*storagemodels.WorkflowModel)
	return wm, args.Error(1)
}

func (m *mockWorkflowRepo) FindAll(ctx context.Context, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, limit, offset)
	wms, _ := args.Get(0).([]*storagemodels.WorkflowModel)
	return wms, args.Error(1)
}

func (m *mockWorkflowRepo) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	args := m.Called(ctx, status, limit, offset)
	wms, _ := args.Get(0).([]*storagemodels.WorkflowModel)
	return wms, args.Error(1)
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
	wms, _ := args.Get(0).([]*storagemodels.WorkflowModel)
	return wms, args.Error(1)
}

func (m *mockWorkflowRepo) CountWithFilters(ctx context.Context, filters repository.WorkflowFilters) (int, error) {
	args := m.Called(ctx, filters)
	return args.Int(0), args.Error(1)
}

func (m *mockWorkflowRepo) CreateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	return m.Called(ctx, node).Error(0)
}

func (m *mockWorkflowRepo) UpdateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	return m.Called(ctx, node).Error(0)
}

func (m *mockWorkflowRepo) DeleteNode(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockWorkflowRepo) FindNodeByID(ctx context.Context, id uuid.UUID) (*storagemodels.NodeModel, error) {
	args := m.Called(ctx, id)
	nm, _ := args.Get(0).(*storagemodels.NodeModel)
	return nm, args.Error(1)
}

func (m *mockWorkflowRepo) FindNodesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.NodeModel, error) {
	args := m.Called(ctx, workflowID)
	nms, _ := args.Get(0).([]*storagemodels.NodeModel)
	return nms, args.Error(1)
}

func (m *mockWorkflowRepo) CreateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	return m.Called(ctx, edge).Error(0)
}

func (m *mockWorkflowRepo) UpdateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	return m.Called(ctx, edge).Error(0)
}

func (m *mockWorkflowRepo) DeleteEdge(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockWorkflowRepo) FindEdgeByID(ctx context.Context, id uuid.UUID) (*storagemodels.EdgeModel, error) {
	args := m.Called(ctx, id)
	em, _ := args.Get(0).(*storagemodels.EdgeModel)
	return em, args.Error(1)
}

func (m *mockWorkflowRepo) FindEdgesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.EdgeModel, error) {
	args := m.Called(ctx, workflowID)
	ems, _ := args.Get(0).([]*storagemodels.EdgeModel)
	return ems, args.Error(1)
}

func (m *mockWorkflowRepo) ValidateDAG(ctx context.Context, workflowID uuid.UUID) error {
	return m.Called(ctx, workflowID).Error(0)
}

func (m *mockWorkflowRepo) AssignResource(ctx context.Context, workflowID uuid.UUID, resource *storagemodels.WorkflowResourceModel, assignedBy *uuid.UUID) error {
	return m.Called(ctx, workflowID, resource, assignedBy).Error(0)
}

func (m *mockWorkflowRepo) UnassignResource(ctx context.Context, workflowID, resourceID uuid.UUID) error {
	return m.Called(ctx, workflowID, resourceID).Error(0)
}

func (m *mockWorkflowRepo) UnassignResourceFromAllWorkflows(ctx context.Context, resourceID uuid.UUID) (int64, error) {
	args := m.Called(ctx, resourceID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockWorkflowRepo) GetWorkflowResources(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.WorkflowResourceModel, error) {
	args := m.Called(ctx, workflowID)
	wrs, _ := args.Get(0).([]*storagemodels.WorkflowResourceModel)
	return wrs, args.Error(1)
}

func (m *mockWorkflowRepo) UpdateResourceAlias(ctx context.Context, workflowID, resourceID uuid.UUID, newAlias string) error {
	return m.Called(ctx, workflowID, resourceID, newAlias).Error(0)
}

func (m *mockWorkflowRepo) ResourceExists(ctx context.Context, workflowID, resourceID uuid.UUID) (bool, error) {
	args := m.Called(ctx, workflowID, resourceID)
	return args.Bool(0), args.Error(1)
}

func (m *mockWorkflowRepo) GetResourceByAlias(ctx context.Context, workflowID uuid.UUID, alias string) (*storagemodels.WorkflowResourceModel, error) {
	args := m.Called(ctx, workflowID, alias)
	wrm, _ := args.Get(0).(*storagemodels.WorkflowResourceModel)
	return wrm, args.Error(1)
}

// --- Mock: ExecutionRepository ---

type mockExecutionRepo struct {
	mock.Mock
}

func (m *mockExecutionRepo) Create(ctx context.Context, execution *storagemodels.ExecutionModel) error {
	return m.Called(ctx, execution).Error(0)
}

func (m *mockExecutionRepo) Update(ctx context.Context, execution *storagemodels.ExecutionModel) error {
	return m.Called(ctx, execution).Error(0)
}

func (m *mockExecutionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockExecutionRepo) FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.ExecutionModel, error) {
	args := m.Called(ctx, id)
	em, _ := args.Get(0).(*storagemodels.ExecutionModel)
	return em, args.Error(1)
}

func (m *mockExecutionRepo) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*storagemodels.ExecutionModel, error) {
	args := m.Called(ctx, id)
	em, _ := args.Get(0).(*storagemodels.ExecutionModel)
	return em, args.Error(1)
}

func (m *mockExecutionRepo) FindByWorkflowID(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*storagemodels.ExecutionModel, error) {
	args := m.Called(ctx, workflowID, limit, offset)
	ems, _ := args.Get(0).([]*storagemodels.ExecutionModel)
	return ems, args.Error(1)
}

func (m *mockExecutionRepo) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*storagemodels.ExecutionModel, error) {
	args := m.Called(ctx, status, limit, offset)
	ems, _ := args.Get(0).([]*storagemodels.ExecutionModel)
	return ems, args.Error(1)
}

func (m *mockExecutionRepo) FindAll(ctx context.Context, limit, offset int) ([]*storagemodels.ExecutionModel, error) {
	args := m.Called(ctx, limit, offset)
	ems, _ := args.Get(0).([]*storagemodels.ExecutionModel)
	return ems, args.Error(1)
}

func (m *mockExecutionRepo) FindRunning(ctx context.Context) ([]*storagemodels.ExecutionModel, error) {
	args := m.Called(ctx)
	ems, _ := args.Get(0).([]*storagemodels.ExecutionModel)
	return ems, args.Error(1)
}

func (m *mockExecutionRepo) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *mockExecutionRepo) CountByWorkflowID(ctx context.Context, workflowID uuid.UUID) (int, error) {
	args := m.Called(ctx, workflowID)
	return args.Int(0), args.Error(1)
}

func (m *mockExecutionRepo) CountByStatus(ctx context.Context, status string) (int, error) {
	args := m.Called(ctx, status)
	return args.Int(0), args.Error(1)
}

func (m *mockExecutionRepo) CreateNodeExecution(ctx context.Context, nodeExecution *storagemodels.NodeExecutionModel) error {
	return m.Called(ctx, nodeExecution).Error(0)
}

func (m *mockExecutionRepo) UpdateNodeExecution(ctx context.Context, nodeExecution *storagemodels.NodeExecutionModel) error {
	return m.Called(ctx, nodeExecution).Error(0)
}

func (m *mockExecutionRepo) DeleteNodeExecution(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockExecutionRepo) FindNodeExecutionByID(ctx context.Context, id uuid.UUID) (*storagemodels.NodeExecutionModel, error) {
	args := m.Called(ctx, id)
	nem, _ := args.Get(0).(*storagemodels.NodeExecutionModel)
	return nem, args.Error(1)
}

func (m *mockExecutionRepo) FindNodeExecutionsByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*storagemodels.NodeExecutionModel, error) {
	args := m.Called(ctx, executionID)
	nems, _ := args.Get(0).([]*storagemodels.NodeExecutionModel)
	return nems, args.Error(1)
}

func (m *mockExecutionRepo) FindNodeExecutionsByWave(ctx context.Context, executionID uuid.UUID, wave int) ([]*storagemodels.NodeExecutionModel, error) {
	args := m.Called(ctx, executionID, wave)
	nems, _ := args.Get(0).([]*storagemodels.NodeExecutionModel)
	return nems, args.Error(1)
}

func (m *mockExecutionRepo) FindNodeExecutionsByStatus(ctx context.Context, executionID uuid.UUID, status string) ([]*storagemodels.NodeExecutionModel, error) {
	args := m.Called(ctx, executionID, status)
	nems, _ := args.Get(0).([]*storagemodels.NodeExecutionModel)
	return nems, args.Error(1)
}

func (m *mockExecutionRepo) GetEvents(ctx context.Context, executionID uuid.UUID) ([]*storagemodels.EventModel, error) {
	args := m.Called(ctx, executionID)
	evts, _ := args.Get(0).([]*storagemodels.EventModel)
	return evts, args.Error(1)
}

func (m *mockExecutionRepo) GetStatistics(ctx context.Context, workflowID *uuid.UUID, from, to time.Time) (*repository.ExecutionStatistics, error) {
	args := m.Called(ctx, workflowID, from, to)
	stats, _ := args.Get(0).(*repository.ExecutionStatistics)
	return stats, args.Error(1)
}

// --- Mock: TriggerRepository ---

type mockTriggerRepo struct {
	mock.Mock
}

func (m *mockTriggerRepo) Create(ctx context.Context, trigger *storagemodels.TriggerModel) error {
	return m.Called(ctx, trigger).Error(0)
}

func (m *mockTriggerRepo) Update(ctx context.Context, trigger *storagemodels.TriggerModel) error {
	return m.Called(ctx, trigger).Error(0)
}

func (m *mockTriggerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockTriggerRepo) FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, id)
	tm, _ := args.Get(0).(*storagemodels.TriggerModel)
	return tm, args.Error(1)
}

func (m *mockTriggerRepo) FindByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, workflowID)
	tms, _ := args.Get(0).([]*storagemodels.TriggerModel)
	return tms, args.Error(1)
}

func (m *mockTriggerRepo) FindByType(ctx context.Context, triggerType string, limit, offset int) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, triggerType, limit, offset)
	tms, _ := args.Get(0).([]*storagemodels.TriggerModel)
	return tms, args.Error(1)
}

func (m *mockTriggerRepo) FindEnabled(ctx context.Context) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx)
	tms, _ := args.Get(0).([]*storagemodels.TriggerModel)
	return tms, args.Error(1)
}

func (m *mockTriggerRepo) FindEnabledByType(ctx context.Context, triggerType string) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, triggerType)
	tms, _ := args.Get(0).([]*storagemodels.TriggerModel)
	return tms, args.Error(1)
}

func (m *mockTriggerRepo) FindAll(ctx context.Context, limit, offset int) ([]*storagemodels.TriggerModel, error) {
	args := m.Called(ctx, limit, offset)
	tms, _ := args.Get(0).([]*storagemodels.TriggerModel)
	return tms, args.Error(1)
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
	return m.Called(ctx, id).Error(0)
}

func (m *mockTriggerRepo) Disable(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *mockTriggerRepo) MarkTriggered(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- Mock: CredentialsRepository ---

type mockCredentialsRepo struct {
	mock.Mock
}

func (m *mockCredentialsRepo) CreateCredentials(ctx context.Context, cred *models.CredentialsResource) error {
	return m.Called(ctx, cred).Error(0)
}

func (m *mockCredentialsRepo) GetCredentials(ctx context.Context, resourceID string) (*models.CredentialsResource, error) {
	args := m.Called(ctx, resourceID)
	cr, _ := args.Get(0).(*models.CredentialsResource)
	return cr, args.Error(1)
}

func (m *mockCredentialsRepo) GetCredentialsByOwner(ctx context.Context, ownerID string) ([]*models.CredentialsResource, error) {
	args := m.Called(ctx, ownerID)
	crs, _ := args.Get(0).([]*models.CredentialsResource)
	return crs, args.Error(1)
}

func (m *mockCredentialsRepo) GetCredentialsByProvider(ctx context.Context, ownerID, provider string) ([]*models.CredentialsResource, error) {
	args := m.Called(ctx, ownerID, provider)
	crs, _ := args.Get(0).([]*models.CredentialsResource)
	return crs, args.Error(1)
}

func (m *mockCredentialsRepo) UpdateCredentials(ctx context.Context, cred *models.CredentialsResource) error {
	return m.Called(ctx, cred).Error(0)
}

func (m *mockCredentialsRepo) UpdateEncryptedData(ctx context.Context, resourceID string, encryptedData map[string]string) error {
	return m.Called(ctx, resourceID, encryptedData).Error(0)
}

func (m *mockCredentialsRepo) DeleteCredentials(ctx context.Context, resourceID string) error {
	return m.Called(ctx, resourceID).Error(0)
}

func (m *mockCredentialsRepo) IncrementUsageCount(ctx context.Context, resourceID string) error {
	return m.Called(ctx, resourceID).Error(0)
}

func (m *mockCredentialsRepo) LogCredentialAccess(ctx context.Context, resourceID, action, actorID, actorType string, metadata map[string]interface{}) error {
	return m.Called(ctx, resourceID, action, actorID, actorType, metadata).Error(0)
}

// --- Mock: ExecutorManager ---

type mockExecutorManager struct {
	registeredTypes map[string]bool
}

func newMockExecutorManager(types ...string) executor.Manager {
	m := &mockExecutorManager{registeredTypes: make(map[string]bool)}
	for _, t := range types {
		m.registeredTypes[t] = true
	}
	return m
}

func (m *mockExecutorManager) Register(_ string, _ executor.Executor) error { return nil }

func (m *mockExecutorManager) Get(_ string) (executor.Executor, error) { return nil, nil }

func (m *mockExecutorManager) Has(nodeType string) bool {
	return m.registeredTypes[nodeType]
}

func (m *mockExecutorManager) List() []string { return nil }

func (m *mockExecutorManager) Unregister(_ string) error { return nil }

// --- Mock: ServiceAuditLogRepository (for AuditService) ---

type mockAuditLogRepo struct {
	mock.Mock
}

func (m *mockAuditLogRepo) Create(ctx context.Context, log *models.ServiceAuditLog) error {
	return m.Called(ctx, log).Error(0)
}

func (m *mockAuditLogRepo) FindAll(ctx context.Context, filter repository.ServiceAuditLogFilter) ([]*models.ServiceAuditLog, int64, error) {
	args := m.Called(ctx, filter)
	logs, _ := args.Get(0).([]*models.ServiceAuditLog)
	return logs, args.Get(1).(int64), args.Error(2)
}

func (m *mockAuditLogRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	args := m.Called(ctx, before)
	return args.Get(0).(int64), args.Error(1)
}

// --- Mock: ExecutionManager ---

type mockExecutionManager struct {
	mock.Mock
}

func (m *mockExecutionManager) ExecuteAsync(ctx context.Context, workflowID string, input map[string]interface{}, opts *engine.ExecutionOptions) (*models.Execution, error) {
	args := m.Called(ctx, workflowID, input, opts)
	exec, _ := args.Get(0).(*models.Execution)
	return exec, args.Error(1)
}

// --- Helpers ---

func newTestLogger() *logger.Logger {
	return logger.Default()
}

func newTestEncryptionSvc() *crypto.EncryptionService {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	svc, _ := crypto.NewEncryptionService(key)
	return svc
}

func newTestOperations(
	wfRepo *mockWorkflowRepo,
	execRepo *mockExecutionRepo,
	trigRepo *mockTriggerRepo,
	credRepo *mockCredentialsRepo,
	auditLogRepo *mockAuditLogRepo,
	execMgr *mockExecutionManager,
	executorMgr executor.Manager,
) *Operations {
	var auditSvc *systemkey.AuditService
	if auditLogRepo != nil {
		auditSvc = systemkey.NewAuditService(auditLogRepo, 90)
	}

	ops := &Operations{
		Logger:        newTestLogger(),
		EncryptionSvc: newTestEncryptionSvc(),
	}

	if wfRepo != nil {
		ops.WorkflowRepo = wfRepo
	}
	if execRepo != nil {
		ops.ExecutionRepo = execRepo
	}
	if trigRepo != nil {
		ops.TriggerRepo = trigRepo
	}
	if credRepo != nil {
		ops.CredentialsRepo = credRepo
	}
	if auditSvc != nil {
		ops.AuditService = auditSvc
	}
	if executorMgr != nil {
		ops.ExecutorManager = executorMgr
	}

	return ops
}

// Compile-time interface checks.
var (
	_ repository.WorkflowRepository      = (*mockWorkflowRepo)(nil)
	_ repository.ExecutionRepository      = (*mockExecutionRepo)(nil)
	_ repository.TriggerRepository        = (*mockTriggerRepo)(nil)
	_ repository.CredentialsRepository    = (*mockCredentialsRepo)(nil)
	_ repository.ServiceAuditLogRepository = (*mockAuditLogRepo)(nil)
)
