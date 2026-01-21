package rest

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/logger"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/executor"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

// testWorkflowRepository is a simple test implementation of WorkflowRepository.
type testWorkflowRepository struct {
	workflows       map[uuid.UUID]*storagemodels.WorkflowModel
	createErr       error
	findByIDErr     error
	hardDeleteCalls []uuid.UUID
}

func newTestWorkflowRepository() *testWorkflowRepository {
	return &testWorkflowRepository{
		workflows:       make(map[uuid.UUID]*storagemodels.WorkflowModel),
		hardDeleteCalls: make([]uuid.UUID, 0),
	}
}

func (r *testWorkflowRepository) Create(ctx context.Context, workflow *storagemodels.WorkflowModel) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.workflows[workflow.ID] = workflow
	return nil
}

func (r *testWorkflowRepository) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	wf, ok := r.workflows[id]
	if !ok {
		return nil, pkgmodels.ErrWorkflowNotFound
	}
	return wf, nil
}

func (r *testWorkflowRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	r.hardDeleteCalls = append(r.hardDeleteCalls, id)
	delete(r.workflows, id)
	return nil
}

// Minimal interface implementations
func (r *testWorkflowRepository) Update(ctx context.Context, workflow *storagemodels.WorkflowModel) error {
	return nil
}
func (r *testWorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *testWorkflowRepository) FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) FindByName(ctx context.Context, name string, version int) (*storagemodels.WorkflowModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) FindAll(ctx context.Context, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) FindByStatus(ctx context.Context, status string, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) Count(ctx context.Context) (int, error) { return 0, nil }
func (r *testWorkflowRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	return 0, nil
}
func (r *testWorkflowRepository) FindAllWithFilters(ctx context.Context, filters repository.WorkflowFilters, limit, offset int) ([]*storagemodels.WorkflowModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) CountWithFilters(ctx context.Context, filters repository.WorkflowFilters) (int, error) {
	return 0, nil
}
func (r *testWorkflowRepository) CreateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	return nil
}
func (r *testWorkflowRepository) UpdateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	return nil
}
func (r *testWorkflowRepository) DeleteNode(ctx context.Context, id uuid.UUID) error { return nil }
func (r *testWorkflowRepository) FindNodeByID(ctx context.Context, id uuid.UUID) (*storagemodels.NodeModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) FindNodesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.NodeModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) CreateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	return nil
}
func (r *testWorkflowRepository) UpdateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	return nil
}
func (r *testWorkflowRepository) DeleteEdge(ctx context.Context, id uuid.UUID) error { return nil }
func (r *testWorkflowRepository) FindEdgeByID(ctx context.Context, id uuid.UUID) (*storagemodels.EdgeModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) FindEdgesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.EdgeModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) ValidateDAG(ctx context.Context, workflowID uuid.UUID) error {
	return nil
}
func (r *testWorkflowRepository) AssignResource(ctx context.Context, workflowID uuid.UUID, resource *storagemodels.WorkflowResourceModel, assignedBy *uuid.UUID) error {
	return nil
}
func (r *testWorkflowRepository) UnassignResource(ctx context.Context, workflowID, resourceID uuid.UUID) error {
	return nil
}
func (r *testWorkflowRepository) UnassignResourceFromAllWorkflows(ctx context.Context, resourceID uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *testWorkflowRepository) GetWorkflowResources(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.WorkflowResourceModel, error) {
	return nil, nil
}
func (r *testWorkflowRepository) UpdateResourceAlias(ctx context.Context, workflowID, resourceID uuid.UUID, newAlias string) error {
	return nil
}
func (r *testWorkflowRepository) ResourceExists(ctx context.Context, workflowID, resourceID uuid.UUID) (bool, error) {
	return false, nil
}
func (r *testWorkflowRepository) GetResourceByAlias(ctx context.Context, workflowID uuid.UUID, alias string) (*storagemodels.WorkflowResourceModel, error) {
	return nil, nil
}

// testTriggerRepository is a simple test implementation of TriggerRepository.
type testTriggerRepository struct {
	triggers  map[uuid.UUID]*storagemodels.TriggerModel
	createErr error
}

func newTestTriggerRepository() *testTriggerRepository {
	return &testTriggerRepository{
		triggers: make(map[uuid.UUID]*storagemodels.TriggerModel),
	}
}

func (r *testTriggerRepository) Create(ctx context.Context, trigger *storagemodels.TriggerModel) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.triggers[trigger.ID] = trigger
	return nil
}

func (r *testTriggerRepository) FindByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.TriggerModel, error) {
	var result []*storagemodels.TriggerModel
	for _, t := range r.triggers {
		if t.WorkflowID == workflowID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *testTriggerRepository) Update(ctx context.Context, trigger *storagemodels.TriggerModel) error {
	return nil
}
func (r *testTriggerRepository) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *testTriggerRepository) FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.TriggerModel, error) {
	return nil, nil
}
func (r *testTriggerRepository) FindByType(ctx context.Context, triggerType string, limit, offset int) ([]*storagemodels.TriggerModel, error) {
	return nil, nil
}
func (r *testTriggerRepository) FindEnabled(ctx context.Context) ([]*storagemodels.TriggerModel, error) {
	return nil, nil
}
func (r *testTriggerRepository) FindEnabledByType(ctx context.Context, triggerType string) ([]*storagemodels.TriggerModel, error) {
	return nil, nil
}
func (r *testTriggerRepository) FindAll(ctx context.Context, limit, offset int) ([]*storagemodels.TriggerModel, error) {
	return nil, nil
}
func (r *testTriggerRepository) Count(ctx context.Context) (int, error) { return 0, nil }
func (r *testTriggerRepository) CountByWorkflowID(ctx context.Context, workflowID uuid.UUID) (int, error) {
	return 0, nil
}
func (r *testTriggerRepository) CountByType(ctx context.Context, triggerType string) (int, error) {
	return 0, nil
}
func (r *testTriggerRepository) Enable(ctx context.Context, id uuid.UUID) error        { return nil }
func (r *testTriggerRepository) Disable(ctx context.Context, id uuid.UUID) error       { return nil }
func (r *testTriggerRepository) MarkTriggered(ctx context.Context, id uuid.UUID) error { return nil }

// testExecutorManager is a simple test implementation of executor.Manager.
type testExecutorManager struct {
	types map[string]bool
}

func newTestExecutorManager(types ...string) *testExecutorManager {
	m := &testExecutorManager{types: make(map[string]bool)}
	for _, t := range types {
		m.types[t] = true
	}
	return m
}

func (m *testExecutorManager) Has(nodeType string) bool { return m.types[nodeType] }
func (m *testExecutorManager) List() []string {
	result := make([]string, 0, len(m.types))
	for t := range m.types {
		result = append(result, t)
	}
	return result
}
func (m *testExecutorManager) Register(string, executor.Executor) error { return nil }
func (m *testExecutorManager) Get(string) (executor.Executor, error)    { return nil, nil }
func (m *testExecutorManager) Unregister(string) error                  { return nil }

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestHandleImportWorkflow_MultipartUpload(t *testing.T) {
	yamlContent := `
metadata:
  name: "Test Import Workflow"
  description: "Testing import"
nodes:
  - id: n1
    name: "HTTP Request"
    type: http
    config:
      url: "https://api.example.com"
`

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.yaml")
	require.NoError(t, err)
	_, err = part.Write([]byte(yamlContent))
	require.NoError(t, err)
	writer.Close()

	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http", "transform", "llm")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.POST("/workflows/import", handlers.HandleImportWorkflow)

	req := httptest.NewRequest(http.MethodPost, "/workflows/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.Contains(t, resp.Body.String(), "Test Import Workflow")
	assert.Contains(t, resp.Body.String(), "workflow_id")
	assert.Len(t, workflowRepo.workflows, 1)
}

func TestHandleImportWorkflow_RawYAMLBody(t *testing.T) {
	yamlContent := `
metadata:
  name: "Raw YAML Workflow"
nodes:
  - id: n1
    name: "Node"
    type: transform
`

	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http", "transform")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.POST("/workflows/import", handlers.HandleImportWorkflow)

	req := httptest.NewRequest(http.MethodPost, "/workflows/import", bytes.NewBufferString(yamlContent))
	req.Header.Set("Content-Type", "application/x-yaml")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.Contains(t, resp.Body.String(), "Raw YAML Workflow")
}

func TestHandleImportWorkflow_WithTrigger(t *testing.T) {
	yamlContent := `
metadata:
  name: "Workflow with Trigger"
nodes:
  - id: n1
    name: "Node"
    type: http
trigger:
  name: "Daily Trigger"
  type: cron
  config:
    schedule: "0 9 * * *"
`

	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.POST("/workflows/import", handlers.HandleImportWorkflow)

	req := httptest.NewRequest(http.MethodPost, "/workflows/import", bytes.NewBufferString(yamlContent))
	req.Header.Set("Content-Type", "text/yaml")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.Contains(t, resp.Body.String(), "trigger_id")
	assert.Len(t, triggerRepo.triggers, 1)
}

func TestHandleImportWorkflow_InvalidContentType(t *testing.T) {
	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.POST("/workflows/import", handlers.HandleImportWorkflow)

	req := httptest.NewRequest(http.MethodPost, "/workflows/import", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "INVALID_CONTENT_TYPE")
}

func TestHandleImportWorkflow_ValidationError(t *testing.T) {
	yamlContent := `
metadata:
  name: ""
nodes: []
`

	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.POST("/workflows/import", handlers.HandleImportWorkflow)

	req := httptest.NewRequest(http.MethodPost, "/workflows/import", bytes.NewBufferString(yamlContent))
	req.Header.Set("Content-Type", "application/x-yaml")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "IMPORT_ERROR")
}

func TestHandleImportWorkflow_InvalidFileExtension(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "test.json")
	require.NoError(t, err)
	_, err = part.Write([]byte("{}"))
	require.NoError(t, err)
	writer.Close()

	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.POST("/workflows/import", handlers.HandleImportWorkflow)

	req := httptest.NewRequest(http.MethodPost, "/workflows/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "INVALID_FILE_TYPE")
}

func TestHandleGetSupportedTypes(t *testing.T) {
	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http", "transform", "llm", "conditional")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.GET("/workflows/import/types", handlers.HandleGetSupportedTypes)

	req := httptest.NewRequest(http.MethodGet, "/workflows/import/types", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "node_types")
}

func TestHandleExportWorkflow_YAML(t *testing.T) {
	workflowID := uuid.New()

	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	now := time.Now()
	workflowModel := &storagemodels.WorkflowModel{
		ID:          workflowID,
		Name:        "Export Test",
		Description: "Testing export",
		Status:      "draft",
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		Nodes: []*storagemodels.NodeModel{
			{
				ID:         uuid.New(),
				NodeID:     "n1",
				WorkflowID: workflowID,
				Name:       "Node 1",
				Type:       "http",
				Config:     storagemodels.JSONBMap{"url": "https://example.com"},
			},
		},
		Edges: []*storagemodels.EdgeModel{},
	}
	workflowRepo.workflows[workflowID] = workflowModel

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.GET("/workflows/:workflow_id/export", handlers.HandleExportWorkflow)

	req := httptest.NewRequest(http.MethodGet, "/workflows/"+workflowID.String()+"/export?format=yaml", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "application/x-yaml", resp.Header().Get("Content-Type"))
	assert.Contains(t, resp.Header().Get("Content-Disposition"), "workflow.yaml")
}

func TestHandleExportWorkflow_JSON(t *testing.T) {
	workflowID := uuid.New()

	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	now := time.Now()
	workflowModel := &storagemodels.WorkflowModel{
		ID:        workflowID,
		Name:      "Export JSON Test",
		Status:    "draft",
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
		Nodes:     []*storagemodels.NodeModel{},
		Edges:     []*storagemodels.EdgeModel{},
	}
	workflowRepo.workflows[workflowID] = workflowModel

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.GET("/workflows/:workflow_id/export", handlers.HandleExportWorkflow)

	req := httptest.NewRequest(http.MethodGet, "/workflows/"+workflowID.String()+"/export?format=json", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")
}

func TestHandleExportWorkflow_InvalidFormat(t *testing.T) {
	workflowID := uuid.New()

	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	now := time.Now()
	workflowModel := &storagemodels.WorkflowModel{
		ID:        workflowID,
		Name:      "Test",
		Status:    "draft",
		CreatedAt: now,
		UpdatedAt: now,
		Nodes:     []*storagemodels.NodeModel{},
		Edges:     []*storagemodels.EdgeModel{},
	}
	workflowRepo.workflows[workflowID] = workflowModel

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.GET("/workflows/:workflow_id/export", handlers.HandleExportWorkflow)

	req := httptest.NewRequest(http.MethodGet, "/workflows/"+workflowID.String()+"/export?format=xml", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "INVALID_FORMAT")
}

func TestHandleExportWorkflow_NotFound(t *testing.T) {
	workflowID := uuid.New()

	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.GET("/workflows/:workflow_id/export", handlers.HandleExportWorkflow)

	req := httptest.NewRequest(http.MethodGet, "/workflows/"+workflowID.String()+"/export", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.NotEqual(t, http.StatusOK, resp.Code)
}

func TestHandleExportWorkflow_InvalidID(t *testing.T) {
	workflowRepo := newTestWorkflowRepository()
	triggerRepo := newTestTriggerRepository()
	executorManager := newTestExecutorManager("http")
	log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})

	handlers := NewImportHandlers(workflowRepo, triggerRepo, log, executorManager)

	router := setupTestRouter()
	router.GET("/workflows/:workflow_id/export", handlers.HandleExportWorkflow)

	req := httptest.NewRequest(http.MethodGet, "/workflows/not-a-uuid/export", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}
