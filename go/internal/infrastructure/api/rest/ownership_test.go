package rest

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/filestorage"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/executor/builtin"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

// ownershipEnv wires the workflow-related handlers the same way
// pkg/server/routes.go does: auth context first, then the ownership guard.
type ownershipEnv struct {
	db           bun.IDB
	workflowRepo *storage.WorkflowRepository
	execRepo     *storage.ExecutionRepository
	fileRepo     *storage.FileRepository
	userRepo     *storage.UserRepository
	guard        *OwnershipGuard
	workflows    *WorkflowHandlers
	nodes        *NodeHandlers
	executions   *ExecutionHandlers
	triggers     *TriggerHandlers
	files        *FileHandlers
}

func newOwnershipEnv(t *testing.T) *ownershipEnv {
	t.Helper()

	db, _ := testutil.SetupTestTx(t)
	log := logger.New(config.LoggingConfig{Level: "error", Format: "text"})

	executorManager := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(executorManager))
	require.NoError(t, builtin.RegisterAdapters(executorManager))

	workflowRepo := storage.NewWorkflowRepository(db)
	execRepo := storage.NewExecutionRepository(db)
	triggerRepo := storage.NewTriggerRepository(db)
	fileRepo := storage.NewFileRepository(db)
	resourceRepo := storage.NewResourceRepository(db)

	execManager := engine.NewExecutionManager(executorManager, workflowRepo, execRepo,
		storage.NewEventRepository(db), resourceRepo, nil)

	ops := &serviceapi.Operations{
		WorkflowRepo:    workflowRepo,
		ExecutionRepo:   execRepo,
		TriggerRepo:     triggerRepo,
		ExecutionMgr:    execManager,
		ExecutorManager: executorManager,
		Logger:          log,
	}

	storageManager := filestorage.NewStorageManager(&filestorage.ManagerConfig{BasePath: t.TempDir()}, log)
	t.Cleanup(func() { _ = storageManager.Close() })

	guard := NewOwnershipGuard(workflowRepo, execRepo, triggerRepo, fileRepo, resourceRepo, log)

	return &ownershipEnv{
		db:           db,
		workflowRepo: workflowRepo,
		execRepo:     execRepo,
		fileRepo:     fileRepo,
		userRepo:     storage.NewUserRepository(db),
		guard:        guard,
		workflows:    NewWorkflowHandlers(ops, log).WithOwnershipGuard(guard),
		nodes:        NewNodeHandlers(workflowRepo, log),
		executions:   NewExecutionHandlers(ops, log),
		triggers:     NewTriggerHandlers(ops, log).WithOwnershipGuard(guard),
		files:        NewFileHandlers(fileRepo, storageManager, log).WithOwnershipGuard(guard),
	}
}

// createUser inserts a user so created_by foreign keys are satisfied.
func (e *ownershipEnv) createUser(t *testing.T, admin bool) string {
	t.Helper()
	id := uuid.New()
	email := fmt.Sprintf("user-%s@example.com", id.String()[:8])
	require.NoError(t, e.userRepo.Create(context.Background(), &storagemodels.UserModel{
		ID:           id,
		Email:        email,
		Username:     email,
		PasswordHash: "x",
		IsActive:     true,
		IsAdmin:      admin,
	}))
	return id.String()
}

// routerFor builds a router authenticated as userID.
func (e *ownershipEnv) routerFor(userID string, isAdmin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(testutil.MockAuthMiddleware(userID, isAdmin))
	api := router.Group("/api/v1")

	workflows := api.Group("/workflows")
	workflows.Use(e.guard.RequireWorkflowAccess("workflow_id"))
	workflows.POST("", e.workflows.HandleCreateWorkflow)
	workflows.GET("", e.workflows.HandleListWorkflows)
	workflows.GET("/:workflow_id", e.workflows.HandleGetWorkflow)
	workflows.PUT("/:workflow_id", e.workflows.HandleUpdateWorkflow)
	workflows.DELETE("/:workflow_id", e.workflows.HandleDeleteWorkflow)
	workflows.POST("/:workflow_id/nodes", e.nodes.HandleAddNode)
	workflows.GET("/:workflow_id/nodes", e.nodes.HandleListNodes)
	workflows.POST("/:workflow_id/resources", e.workflows.AttachWorkflowResource)

	executions := api.Group("/executions")
	executions.Use(e.guard.RequireExecutionAccess("id"))
	executions.Use(e.guard.RequireWorkflowAccess("workflow_id"))
	executions.GET("", e.executions.HandleListExecutions)
	executions.GET("/:id", e.executions.HandleGetExecution)
	executions.POST("/run/:workflow_id", e.executions.HandleRunExecution)

	triggers := api.Group("/triggers")
	triggers.Use(e.guard.RequireTriggerAccess("id"))
	triggers.POST("", e.triggers.HandleCreateTrigger)
	triggers.GET("", e.triggers.HandleListTriggers)
	triggers.GET("/:id", e.triggers.HandleGetTrigger)
	triggers.DELETE("/:id", e.triggers.HandleDeleteTrigger)

	files := api.Group("/files")
	files.Use(e.guard.RequireFileAccess("id"))
	files.POST("", e.files.HandleUploadFile)
	files.GET("", e.files.HandleListFiles)
	files.GET("/:id/metadata", e.files.HandleGetFileMetadata)
	files.DELETE("/:id", e.files.HandleDeleteFile)

	return router
}

func createWorkflowAs(t *testing.T, router *gin.Engine, name string) string {
	t.Helper()
	w := testutil.MakeRequest(t, router, "POST", "/api/v1/workflows", map[string]any{"name": name})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var wf map[string]any
	testutil.ParseResponse(t, w, &wf)
	return wf["id"].(string)
}

func listIDs(t *testing.T, router *gin.Engine, path string) []string {
	t.Helper()
	w := testutil.MakeRequest(t, router, "GET", path, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var items []map[string]any
	testutil.ParseListResponse(t, w, &items)
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, fmt.Sprint(it["id"]))
	}
	return ids
}

func TestOwnership_UserCannotSeeOrDeleteOtherUsersWorkflow(t *testing.T) {
	t.Parallel()
	env := newOwnershipEnv(t)

	userA := env.createUser(t, false)
	userB := env.createUser(t, false)
	admin := env.createUser(t, true)
	routerA := env.routerFor(userA, false)
	routerB := env.routerFor(userB, false)
	routerAdmin := env.routerFor(admin, true)

	wfA := createWorkflowAs(t, routerA, "A's workflow")
	wfB := createWorkflowAs(t, routerB, "B's workflow")

	// B cannot read, modify, extend or delete A's workflow: 404, not 403.
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/v1/workflows/" + wfA},
		{"PUT", "/api/v1/workflows/" + wfA},
		{"DELETE", "/api/v1/workflows/" + wfA},
		{"GET", "/api/v1/workflows/" + wfA + "/nodes"},
		{"POST", "/api/v1/workflows/" + wfA + "/nodes"},
		{"POST", "/api/v1/executions/run/" + wfA},
	} {
		w := testutil.MakeRequest(t, routerB, tc.method, tc.path, map[string]any{"name": "hijack"})
		assert.Equal(t, http.StatusNotFound, w.Code, "%s %s: %s", tc.method, tc.path, w.Body.String())
	}

	// A's workflow survived B's delete attempt and is unchanged.
	w := testutil.MakeRequest(t, routerA, "GET", "/api/v1/workflows/"+wfA, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var wf map[string]any
	testutil.ParseResponse(t, w, &wf)
	assert.Equal(t, "A's workflow", wf["name"])

	// Listings only contain the caller's workflows; admins see everything.
	assert.ElementsMatch(t, []string{wfA}, listIDs(t, routerA, "/api/v1/workflows"))
	assert.ElementsMatch(t, []string{wfB}, listIDs(t, routerB, "/api/v1/workflows"))
	adminIDs := listIDs(t, routerAdmin, "/api/v1/workflows")
	assert.Contains(t, adminIDs, wfA)
	assert.Contains(t, adminIDs, wfB)

	// Admin can read and delete anyone's workflow.
	assert.Equal(t, http.StatusOK, testutil.MakeRequest(t, routerAdmin, "GET", "/api/v1/workflows/"+wfA, nil).Code)
	assert.Equal(t, http.StatusOK, testutil.MakeRequest(t, routerAdmin, "DELETE", "/api/v1/workflows/"+wfB, nil).Code)
}

func TestOwnership_UnownedWorkflowVisibleToAdminOnly(t *testing.T) {
	t.Parallel()
	env := newOwnershipEnv(t)

	user := env.createUser(t, false)
	admin := env.createUser(t, true)

	legacy := &storagemodels.WorkflowModel{ID: uuid.New(), Name: "legacy", Status: "draft", Version: 1}
	require.NoError(t, env.workflowRepo.Create(context.Background(), legacy))

	routerUser := env.routerFor(user, false)
	assert.Equal(t, http.StatusNotFound, testutil.MakeRequest(t, routerUser, "GET", "/api/v1/workflows/"+legacy.ID.String(), nil).Code)
	assert.NotContains(t, listIDs(t, routerUser, "/api/v1/workflows"), legacy.ID.String())

	routerAdmin := env.routerFor(admin, true)
	assert.Equal(t, http.StatusOK, testutil.MakeRequest(t, routerAdmin, "GET", "/api/v1/workflows/"+legacy.ID.String(), nil).Code)
}

func TestOwnership_ExecutionsIsolated(t *testing.T) {
	t.Parallel()
	env := newOwnershipEnv(t)
	ctx := context.Background()

	userA := env.createUser(t, false)
	userB := env.createUser(t, false)
	routerA := env.routerFor(userA, false)
	routerB := env.routerFor(userB, false)

	wfA := uuid.MustParse(createWorkflowAs(t, routerA, "A"))
	now := time.Now()

	stored := &storagemodels.ExecutionModel{ID: uuid.New(), WorkflowID: &wfA, WorkflowSource: "stored", Status: "completed", StartedAt: &now}
	require.NoError(t, env.execRepo.Create(ctx, stored))

	ownerA := uuid.MustParse(userA)
	inline := &storagemodels.ExecutionModel{
		ID: uuid.New(), WorkflowSource: "inline", Status: "completed", StartedAt: &now,
		WorkflowSnapshot: storagemodels.JSONBMap{"name": "inline"}, CreatedBy: &ownerA,
	}
	require.NoError(t, env.execRepo.Create(ctx, inline))

	for _, id := range []uuid.UUID{stored.ID, inline.ID} {
		assert.Equal(t, http.StatusOK, testutil.MakeRequest(t, routerA, "GET", "/api/v1/executions/"+id.String(), nil).Code)
		assert.Equal(t, http.StatusNotFound, testutil.MakeRequest(t, routerB, "GET", "/api/v1/executions/"+id.String(), nil).Code)
	}

	assert.ElementsMatch(t, []string{stored.ID.String(), inline.ID.String()}, listIDs(t, routerA, "/api/v1/executions"))
	assert.Empty(t, listIDs(t, routerB, "/api/v1/executions"))
	// Filtering by someone else's workflow yields nothing.
	assert.Empty(t, listIDs(t, routerB, "/api/v1/executions?workflow_id="+wfA.String()))
}

func TestOwnership_TriggersIsolated(t *testing.T) {
	t.Parallel()
	env := newOwnershipEnv(t)

	userA := env.createUser(t, false)
	userB := env.createUser(t, false)
	routerA := env.routerFor(userA, false)
	routerB := env.routerFor(userB, false)

	wfA := createWorkflowAs(t, routerA, "A")
	body := map[string]any{"workflow_id": wfA, "name": "t", "type": "manual", "config": map[string]any{}}

	// B cannot attach a trigger to A's workflow.
	w := testutil.MakeRequest(t, routerB, "POST", "/api/v1/triggers", body)
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	w = testutil.MakeRequest(t, routerA, "POST", "/api/v1/triggers", body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var trigger map[string]any
	testutil.ParseResponse(t, w, &trigger)
	triggerID := trigger["id"].(string)

	assert.Equal(t, http.StatusNotFound, testutil.MakeRequest(t, routerB, "GET", "/api/v1/triggers/"+triggerID, nil).Code)
	assert.Equal(t, http.StatusNotFound, testutil.MakeRequest(t, routerB, "DELETE", "/api/v1/triggers/"+triggerID, nil).Code)
	assert.Empty(t, listIDs(t, routerB, "/api/v1/triggers"))
	assert.Equal(t, []string{triggerID}, listIDs(t, routerA, "/api/v1/triggers"))
}

func TestOwnership_FilesIsolated(t *testing.T) {
	t.Parallel()
	env := newOwnershipEnv(t)

	userA := env.createUser(t, false)
	userB := env.createUser(t, false)
	routerA := env.routerFor(userA, false)
	routerB := env.routerFor(userB, false)

	wfA := createWorkflowAs(t, routerA, "A")
	upload := map[string]any{
		"file_name": "a.txt",
		"file_data": "aGVsbG8=", // "hello"
		"mime_type": "text/plain",
	}

	// B cannot attach a file to A's workflow.
	withWorkflow := map[string]any{"workflow_id": wfA}
	for k, v := range upload {
		withWorkflow[k] = v
	}
	w := testutil.MakeRequest(t, routerB, "POST", "/api/v1/files", withWorkflow)
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	w = testutil.MakeRequest(t, routerA, "POST", "/api/v1/files", upload)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var file map[string]any
	testutil.ParseResponse(t, w, &file)
	fileID := file["id"].(string)

	stored, err := env.fileRepo.FindByID(context.Background(), uuid.MustParse(fileID))
	require.NoError(t, err)
	require.NotNil(t, stored.CreatedBy)
	assert.Equal(t, userA, stored.CreatedBy.String())

	assert.Equal(t, http.StatusNotFound, testutil.MakeRequest(t, routerB, "GET", "/api/v1/files/"+fileID+"/metadata", nil).Code)
	assert.Equal(t, http.StatusNotFound, testutil.MakeRequest(t, routerB, "DELETE", "/api/v1/files/"+fileID, nil).Code)
	assert.Equal(t, http.StatusOK, testutil.MakeRequest(t, routerA, "GET", "/api/v1/files/"+fileID+"/metadata", nil).Code)

	var listB struct {
		Files []map[string]any `json:"files"`
	}
	w = testutil.MakeRequest(t, routerB, "GET", "/api/v1/files", nil)
	require.Equal(t, http.StatusOK, w.Code)
	testutil.ParseResponse(t, w, &listB)
	assert.Empty(t, listB.Files)
}

func TestOwnershipGuard_RequiresAuthenticatedCaller(t *testing.T) {
	t.Parallel()
	env := newOwnershipEnv(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/workflows/:workflow_id", env.guard.RequireWorkflowAccess("workflow_id"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := testutil.MakeRequest(t, router, "GET", "/workflows/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOwnership_CannotAttachOtherUsersResource(t *testing.T) {
	t.Parallel()
	env := newOwnershipEnv(t)

	userA := env.createUser(t, false)
	userB := env.createUser(t, false)
	routerA := env.routerFor(userA, false)
	routerB := env.routerFor(userB, false)

	resourceRepo := storage.NewResourceRepository(env.db)
	resA := pkgmodels.NewFileStorageResource(userA, "A storage")
	require.NoError(t, resourceRepo.Create(context.Background(), resA))

	wfB := createWorkflowAs(t, routerB, "B")
	wfA := createWorkflowAs(t, routerA, "A")
	body := map[string]any{"resource_id": resA.GetID(), "alias": "store"}

	// B cannot attach A's resource to B's own workflow.
	w := testutil.MakeRequest(t, routerB, "POST", "/api/v1/workflows/"+wfB+"/resources", body)
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// The owner can.
	w = testutil.MakeRequest(t, routerA, "POST", "/api/v1/workflows/"+wfA+"/resources", body)
	assert.Less(t, w.Code, 300, w.Body.String())
}
