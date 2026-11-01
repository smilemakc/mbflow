package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/filestorage"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/executor/builtin"
	"github.com/smilemakc/mbflow/go/testutil"
)

// rc2Env wires the workflow-related handlers against a real (embedded)
// PostgreSQL database, the same way pkg/server/routes.go does.
type rc2Env struct {
	db           *bun.DB
	log          *logger.Logger
	workflowRepo *storage.WorkflowRepository
	execRepo     *storage.ExecutionRepository
	triggerRepo  repository.TriggerRepository
	fileRepo     *storage.FileRepository
	userRepo     *storage.UserRepository
	resourceRepo *storage.ResourceRepositoryImpl
	executors    executor.Manager
	execMgr      *engine.ExecutionManager
	ops          *serviceapi.Operations
	guard        *OwnershipGuard
	storageMgr   filestorage.Manager
	storageBase  string
}

func rc2NewEnv(t *testing.T) *rc2Env {
	t.Helper()
	idb, _ := testutil.SetupTestTx(t)
	db, ok := idb.(*bun.DB)
	require.True(t, ok)
	log := rc2Logger()

	executors := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executors))
	require.NoError(t, builtin.RegisterAdapters(executors))

	env := &rc2Env{
		db:           db,
		log:          log,
		workflowRepo: storage.NewWorkflowRepository(db),
		execRepo:     storage.NewExecutionRepository(db),
		triggerRepo:  storage.NewTriggerRepository(db),
		fileRepo:     storage.NewFileRepository(db),
		userRepo:     storage.NewUserRepository(db),
		resourceRepo: storage.NewResourceRepository(db),
		executors:    executors,
		storageBase:  t.TempDir(),
	}
	env.execMgr = engine.NewExecutionManager(executors, env.workflowRepo, env.execRepo,
		storage.NewEventRepository(db), env.resourceRepo, nil)
	// Wait for background executions before the database is dropped.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = env.execMgr.Shutdown(ctx)
	})
	env.ops = &serviceapi.Operations{
		WorkflowRepo:    env.workflowRepo,
		ExecutionRepo:   env.execRepo,
		TriggerRepo:     env.triggerRepo,
		ExecutionMgr:    env.execMgr,
		ExecutorManager: executors,
		Logger:          log,
	}
	sm := filestorage.NewStorageManager(&filestorage.ManagerConfig{BasePath: env.storageBase}, log)
	t.Cleanup(func() { _ = sm.Close() })
	env.storageMgr = sm
	env.guard = NewOwnershipGuard(env.workflowRepo, env.execRepo, env.triggerRepo, env.fileRepo, env.resourceRepo, log)
	return env
}

func (e *rc2Env) user(t *testing.T, admin bool) string {
	t.Helper()
	id := uuid.New()
	email := fmt.Sprintf("rc2-%s@example.com", id.String()[:8])
	require.NoError(t, e.userRepo.Create(context.Background(), &storagemodels.UserModel{
		ID: id, Email: email, Username: email, PasswordHash: "x", IsActive: true, IsAdmin: admin,
	}))
	return id.String()
}

// router returns a router authenticated as userID (no auth context when
// userID is empty). With guarded=false the ownership guard is left out so
// handler-level validation and error paths can be reached directly.
func (e *rc2Env) router(userID string, admin, guarded bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if userID != "" {
		r.Use(testutil.MockAuthMiddleware(userID, admin))
	}

	wfh := NewWorkflowHandlers(e.ops, e.log)
	exh := NewExecutionHandlers(e.ops, e.log)
	trh := NewTriggerHandlers(e.ops, e.log)
	fh := NewFileHandlers(e.fileRepo, e.storageMgr, e.log)
	imp := NewImportHandlers(e.workflowRepo, e.triggerRepo, e.log, e.executors)
	nodes := NewNodeHandlers(e.workflowRepo, e.log)
	edges := NewEdgeHandlers(e.workflowRepo, e.log)
	noop := func(c *gin.Context) { c.Next() }
	wfGuard, exGuard, trGuard, fileGuard := gin.HandlerFunc(noop), gin.HandlerFunc(noop), gin.HandlerFunc(noop), gin.HandlerFunc(noop)
	if guarded {
		wfh.WithOwnershipGuard(e.guard)
		trh.WithOwnershipGuard(e.guard)
		fh.WithOwnershipGuard(e.guard)
		wfGuard = e.guard.RequireWorkflowAccess("workflow_id")
		exGuard = e.guard.RequireExecutionAccess("id")
		trGuard = e.guard.RequireTriggerAccess("id")
		fileGuard = e.guard.RequireFileAccess("id")
	}

	api := r.Group("/api/v1")
	wf := api.Group("/workflows", wfGuard)
	wf.POST("", wfh.HandleCreateWorkflow)
	wf.GET("", wfh.HandleListWorkflows)
	wf.GET("/:workflow_id", wfh.HandleGetWorkflow)
	wf.PUT("/:workflow_id", wfh.HandleUpdateWorkflow)
	wf.POST("/:workflow_id/execute", exh.HandleRunExecution)
	wf.DELETE("/:workflow_id", wfh.HandleDeleteWorkflow)
	wf.POST("/:workflow_id/publish", wfh.HandlePublishWorkflow)
	wf.POST("/:workflow_id/unpublish", wfh.HandleUnpublishWorkflow)
	wf.GET("/:workflow_id/diagram", wfh.HandleGetWorkflowDiagram)
	wf.POST("/:workflow_id/resources", wfh.AttachWorkflowResource)
	wf.GET("/:workflow_id/resources", wfh.GetWorkflowResources)
	wf.PUT("/:workflow_id/resources/:resource_id", wfh.UpdateWorkflowResourceAlias)
	wf.DELETE("/:workflow_id/resources/:resource_id", wfh.DetachWorkflowResource)
	wf.POST("/:workflow_id/nodes", nodes.HandleAddNode)
	wf.GET("/:workflow_id/nodes/:node_id", nodes.HandleGetNode)
	wf.POST("/:workflow_id/edges", edges.HandleAddEdge)
	wf.POST("/import", imp.HandleImportWorkflow)
	wf.GET("/import/types", imp.HandleGetSupportedTypes)
	wf.GET("/:workflow_id/export", imp.HandleExportWorkflow)

	ex := api.Group("/executions", exGuard)
	if guarded {
		ex.Use(e.guard.RequireWorkflowAccess("workflow_id"))
	}
	ex.POST("/run/:workflow_id", exh.HandleRunExecution)
	ex.POST("", exh.HandleRunExecution)
	ex.POST("/ephemeral", exh.HandleRunEphemeralExecution)
	ex.GET("", exh.HandleListExecutions)
	ex.GET("/:id", exh.HandleGetExecution)
	ex.GET("/:id/logs", exh.HandleGetLogs)
	ex.GET("/:id/nodes/:node_id/result", exh.HandleGetNodeResult)
	ex.POST("/:id/cancel", exh.HandleCancelExecution)
	ex.POST("/:id/retry", exh.HandleRetryExecution)
	ex.GET("/:id/watch", exh.HandleWatchExecution)
	ex.GET("/:id/stream", exh.HandleStreamLogs)

	tr := api.Group("/triggers", trGuard)
	tr.POST("", trh.HandleCreateTrigger)
	tr.GET("", trh.HandleListTriggers)
	tr.GET("/:id", trh.HandleGetTrigger)
	tr.PUT("/:id", trh.HandleUpdateTrigger)
	tr.DELETE("/:id", trh.HandleDeleteTrigger)
	tr.POST("/:id/enable", trh.HandleEnableTrigger)
	tr.POST("/:id/disable", trh.HandleDisableTrigger)
	tr.POST("/:id/execute", trh.HandleTriggerManual)

	files := api.Group("/files", fileGuard)
	files.POST("", fh.HandleUploadFile)
	files.GET("", fh.HandleListFiles)
	files.GET("/:id", fh.HandleGetFile)
	files.GET("/:id/metadata", fh.HandleGetFileMetadata)
	files.DELETE("/:id", fh.HandleDeleteFile)
	files.GET("/storage/:storage_id/usage", fh.HandleGetStorageUsage)
	files.GET("/usage", fh.HandleGetStorageUsage)

	// Routes lacking the path parameters the handlers expect.
	np := r.Group("/np")
	np.GET("/wf", wfh.HandleGetWorkflow)
	np.PUT("/wf", wfh.HandleUpdateWorkflow)
	np.DELETE("/wf", wfh.HandleDeleteWorkflow)
	np.POST("/publish", wfh.HandlePublishWorkflow)
	np.POST("/unpublish", wfh.HandleUnpublishWorkflow)
	np.GET("/diagram", wfh.HandleGetWorkflowDiagram)
	np.POST("/attach", wfh.AttachWorkflowResource)
	np.DELETE("/detach", wfh.DetachWorkflowResource)
	np.DELETE("/detach/:workflow_id", wfh.DetachWorkflowResource)
	np.GET("/resources", wfh.GetWorkflowResources)
	np.PUT("/alias", wfh.UpdateWorkflowResourceAlias)
	np.PUT("/alias/:workflow_id", wfh.UpdateWorkflowResourceAlias)
	np.GET("/exec", exh.HandleGetExecution)
	np.GET("/logs", exh.HandleGetLogs)
	np.GET("/noderesult", exh.HandleGetNodeResult)
	np.GET("/trigger", trh.HandleGetTrigger)
	np.PUT("/trigger", trh.HandleUpdateTrigger)
	np.DELETE("/trigger", trh.HandleDeleteTrigger)
	np.POST("/enable", trh.HandleEnableTrigger)
	np.POST("/disable", trh.HandleDisableTrigger)
	np.POST("/manual", trh.HandleTriggerManual)
	np.GET("/file", fh.HandleGetFile)
	np.GET("/filemeta", fh.HandleGetFileMetadata)
	np.DELETE("/file", fh.HandleDeleteFile)
	np.GET("/export", imp.HandleExportWorkflow)
	np.GET("/exq", e.guard.RequireExecutionQueryAccess("execution_id"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func (e *rc2Env) createWorkflow(t *testing.T, r http.Handler, body map[string]any) map[string]any {
	t.Helper()
	w := rc2Do(t, r, http.MethodPost, "/api/v1/workflows", body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	return rc2DecodeMap(t, w)
}

// rc2SimpleWorkflow is a request body for a valid two-node workflow.
func rc2SimpleWorkflow(name string) map[string]any {
	return map[string]any{
		"name":        name,
		"description": "rc2",
		"variables":   map[string]any{"v": 1},
		"nodes": []map[string]any{
			{"id": "n1", "name": "First", "type": "transform", "config": map[string]any{"type": "passthrough"}},
			{"id": "n2", "name": "Second", "type": "transform", "config": map[string]any{"type": "passthrough"}},
		},
		"edges": []map[string]any{{"id": "e1", "from": "n1", "to": "n2"}},
	}
}

func rc2DoHeaders(t *testing.T, r http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
