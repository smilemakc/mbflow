package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/internal/application/systemkey"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/executor/builtin"
	"github.com/smilemakc/mbflow/go/testutil"
)

// ---------------------------------------------------------------------------
// Shared rc1 helpers
// ---------------------------------------------------------------------------

func rc1Logger() *logger.Logger {
	return logger.New(config.LoggingConfig{Level: "error", Format: "text"})
}

// rc1DB returns an isolated, migrated database for the test.
func rc1DB(t *testing.T) *bun.DB {
	t.Helper()
	idb, _ := testutil.SetupTestTx(t)
	db, ok := idb.(*bun.DB)
	require.True(t, ok)
	return db
}

// rc1ClosedDB returns a database handle whose connection pool is closed, so
// every query fails. It drives the handlers' storage-failure branches.
func rc1ClosedDB(t *testing.T) *bun.DB {
	t.Helper()
	db := rc1DB(t)
	require.NoError(t, db.Close())
	return db
}

func rc1CreateUser(t *testing.T, db bun.IDB, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, storage.NewUserRepository(db).Create(context.Background(), &storagemodels.UserModel{
		ID: id, Email: name + "-" + id.String()[:8] + "@test.local", Username: name + "-" + id.String()[:8],
		PasswordHash: "x", IsActive: true,
	}))
	return id
}

func rc1Encryption(t *testing.T) *crypto.EncryptionService {
	t.Helper()
	svc, err := crypto.NewEncryptionService([]byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)
	return svc
}

// rc1Do performs a JSON request with optional headers.
func rc1Do(t *testing.T, h http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	switch b := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(b))
	case []byte:
		reader = bytes.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func rc1JSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v), w.Body.String())
	return v
}

// rc1UserFromHeader puts the X-Test-User header value into the context as the
// authenticated user, mimicking the auth middleware.
func rc1UserFromHeader() gin.HandlerFunc {
	return func(c *gin.Context) {
		if u := c.GetHeader("X-Test-User"); u != "" {
			c.Set(ContextKeyUserID, u)
		}
		if c.GetHeader("X-Test-Admin") == "true" {
			c.Set(ContextKeyIsAdmin, true)
		}
		c.Next()
	}
}

// ---------------------------------------------------------------------------
// Service API environment: real system key auth, impersonation and audit
// middleware in front of the service API handlers, as wired in pkg/server.
// ---------------------------------------------------------------------------

type rc1ServiceEnv struct {
	db           *bun.DB
	router       *gin.Engine
	keySvc       *systemkey.Service
	auditRepo    *storage.ServiceAuditLogRepoImpl
	workflowRepo *storage.WorkflowRepository
	triggerRepo  repository.TriggerRepository
	credRepo     *storage.CredentialsRepositoryImpl
	execRepo     *storage.ExecutionRepository
	systemUserID uuid.UUID
	adminID      uuid.UUID
	userID       uuid.UUID
	plainKey     string
	keyID        string
}

func rc1NewServiceEnv(t *testing.T, db *bun.DB, withEncryption bool) *rc1ServiceEnv {
	t.Helper()
	log := rc1Logger()
	env := &rc1ServiceEnv{db: db}

	keyRepo := storage.NewSystemKeyRepo(db)
	env.auditRepo = storage.NewServiceAuditLogRepo(db)
	env.keySvc = systemkey.NewService(keyRepo, systemkey.Config{MaxKeys: 3, BcryptCost: 4})
	auditSvc := systemkey.NewAuditService(env.auditRepo, 30)
	env.workflowRepo = storage.NewWorkflowRepository(db)
	env.triggerRepo = storage.NewTriggerRepository(db)
	env.credRepo = storage.NewCredentialsRepository(db)
	env.execRepo = storage.NewExecutionRepository(db)
	userRepo := storage.NewUserRepository(db)

	execMgr := executor.NewManager()
	require.NoError(t, builtin.RegisterBuiltins(execMgr))
	executionManager := engine.NewExecutionManager(execMgr, env.workflowRepo, env.execRepo, storage.NewEventRepository(db), nil, nil)

	ops := &serviceapi.Operations{
		WorkflowRepo:    env.workflowRepo,
		ExecutionRepo:   env.execRepo,
		TriggerRepo:     env.triggerRepo,
		CredentialsRepo: env.credRepo,
		ExecutionMgr:    executionManager,
		ExecutorManager: execMgr,
		AuditService:    auditSvc,
		Logger:          log,
	}
	if withEncryption {
		ops.EncryptionSvc = rc1Encryption(t)
	}

	env.systemUserID = uuid.New()
	if db.Ping() == nil {
		// The configured system user must exist: workflows and credentials
		// created without impersonation are owned by it.
		env.systemUserID = rc1CreateUser(t, db, "system")
	}
	sysAuth := NewSystemAuthMiddleware(env.keySvc, userRepo, env.systemUserID.String(), log)
	audit := NewAuditMiddleware(auditSvc, log)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")

	keyHandlers := NewServiceAPISystemKeyHandlers(env.keySvc, log)
	admin := api.Group("/service/system-keys", rc1UserFromHeader())
	admin.POST("", keyHandlers.CreateSystemKey)
	admin.GET("", keyHandlers.ListSystemKeys)
	admin.GET("/:id", keyHandlers.GetSystemKey)
	admin.DELETE("/:id", keyHandlers.DeleteSystemKey)
	admin.POST("/:id/revoke", keyHandlers.RevokeSystemKey)

	register := func(g *gin.RouterGroup) {
		wfh := NewServiceAPIWorkflowHandlers(ops)
		g.GET("/workflows", wfh.ListWorkflows)
		g.GET("/workflows/:id", wfh.GetWorkflow)
		g.POST("/workflows", wfh.CreateWorkflow)
		g.PUT("/workflows/:id", wfh.UpdateWorkflow)
		g.DELETE("/workflows/:id", wfh.DeleteWorkflow)

		exh := NewServiceAPIExecutionHandlers(ops)
		g.GET("/executions", exh.ListExecutions)
		g.GET("/executions/:id", exh.GetExecution)
		g.POST("/workflows/:id/execute", exh.StartExecution)
		g.POST("/executions/ephemeral", exh.StartEphemeralExecution)
		g.POST("/executions/:id/cancel", exh.CancelExecution)
		g.POST("/executions/:id/retry", exh.RetryExecution)

		trh := NewServiceAPITriggerHandlers(ops)
		g.GET("/triggers", trh.ListTriggers)
		g.POST("/triggers", trh.CreateTrigger)
		g.PUT("/triggers/:id", trh.UpdateTrigger)
		g.DELETE("/triggers/:id", trh.DeleteTrigger)

		crh := NewServiceAPICredentialHandlers(ops)
		g.GET("/credentials", crh.ListCredentials)
		g.POST("/credentials", crh.CreateCredential)
		g.PUT("/credentials/:id", crh.UpdateCredential)
		g.DELETE("/credentials/:id", crh.DeleteCredential)

		auh := NewServiceAPIAuditHandlers(ops)
		g.GET("/audit-log", auh.ListAuditLog)

		// Routes without the :id parameter exercise the missing-parameter guard.
		g.GET("/noparam/workflow", wfh.GetWorkflow)
		g.PUT("/noparam/workflow", wfh.UpdateWorkflow)
		g.DELETE("/noparam/workflow", wfh.DeleteWorkflow)
		g.GET("/noparam/execution", exh.GetExecution)
		g.POST("/noparam/execute", exh.StartExecution)
		g.PUT("/noparam/trigger", trh.UpdateTrigger)
		g.DELETE("/noparam/trigger", trh.DeleteTrigger)
		g.PUT("/noparam/credential", crh.UpdateCredential)
		g.DELETE("/noparam/credential", crh.DeleteCredential)
	}

	service := api.Group("/service")
	service.Use(sysAuth.RequireSystemAccess(), sysAuth.HandleImpersonation(), audit.RecordAction())
	register(service)

	// The same handlers behind a fake auth layer, for databases on which the
	// real system key check cannot succeed.
	register(r.Group("/raw/service", rc1UserFromHeader()))

	admin.GET("/noparam", keyHandlers.GetSystemKey)
	admin.DELETE("/noparam", keyHandlers.DeleteSystemKey)
	admin.POST("/noparam/revoke", keyHandlers.RevokeSystemKey)

	env.router = r
	return env
}

// rc1Provision creates users and an active system key on a live database.
func (e *rc1ServiceEnv) rc1Provision(t *testing.T) {
	t.Helper()
	e.adminID = rc1CreateUser(t, e.db, "admin")
	e.userID = rc1CreateUser(t, e.db, "user")
	res, err := e.keySvc.CreateKey(context.Background(), "svc", "", "billing", e.adminID, nil)
	require.NoError(t, err)
	e.plainKey = res.PlainKey
	e.keyID = res.Key.ID
}

func (e *rc1ServiceEnv) rc1Svc(t *testing.T, method, path string, body any, extra map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	h := map[string]string{"X-System-Key": e.plainKey}
	for k, v := range extra {
		h[k] = v
	}
	return rc1Do(t, e.router, method, "/api/v1/service"+path, body, h)
}

func (e *rc1ServiceEnv) rc1WaitAudit(t *testing.T, action string) *storagemodels.ServiceAuditLogModel {
	t.Helper()
	var found *storagemodels.ServiceAuditLogModel
	require.Eventually(t, func() bool {
		var rows []*storagemodels.ServiceAuditLogModel
		if err := e.db.NewSelect().Model(&rows).Where("action = ?", action).Scan(context.Background()); err != nil {
			return false
		}
		if len(rows) == 0 {
			return false
		}
		found = rows[0]
		return true
	}, 5*time.Second, 20*time.Millisecond, "audit entry %q not written", action)
	return found
}

// ---------------------------------------------------------------------------
// System auth middleware
// ---------------------------------------------------------------------------

func TestRC1SystemAuth_RejectsMissingOrInvalidKeys(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	env.rc1Provision(t)

	cases := []struct {
		name    string
		headers map[string]string
		msg     string
	}{
		{"no key", nil, "system key required"},
		{"bearer non-system token", map[string]string{"Authorization": "Bearer abc"}, "system key required"},
		{"basic auth", map[string]string{"Authorization": "Basic abc"}, "system key required"},
		{"x-system-key wrong prefix", map[string]string{"X-System-Key": "nope_123"}, "system key required"},
		{"unknown key", map[string]string{"X-System-Key": "sysk_" + strings.Repeat("a", 40)}, "invalid system key"},
		{"wrong secret same prefix", map[string]string{"X-System-Key": env.plainKey[:12] + strings.Repeat("z", len(env.plainKey)-12)}, "invalid system key"},
	}
	for _, tc := range cases {
		w := rc1Do(t, env.router, http.MethodGet, "/api/v1/service/workflows", nil, tc.headers)
		assert.Equal(t, http.StatusUnauthorized, w.Code, tc.name)
		assert.Equal(t, tc.msg, rc1JSON(t, w)["message"], tc.name)
	}

	// Bearer form with a valid key is accepted.
	w := rc1Do(t, env.router, http.MethodGet, "/api/v1/service/workflows", nil,
		map[string]string{"Authorization": "Bearer " + env.plainKey})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestRC1SystemAuth_RevokedAndExpiredKeys(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	env.rc1Provision(t)
	ctx := context.Background()

	revoked, err := env.keySvc.CreateKey(ctx, "revoked", "", "svc", env.adminID, nil)
	require.NoError(t, err)
	require.NoError(t, env.keySvc.RevokeKey(ctx, uuid.MustParse(revoked.Key.ID)))
	w := rc1Do(t, env.router, http.MethodGet, "/api/v1/service/workflows", nil, map[string]string{"X-System-Key": revoked.PlainKey})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "system key has been revoked", rc1JSON(t, w)["message"])

	days := 1
	expired, err := env.keySvc.CreateKey(ctx, "expired", "", "svc", env.adminID, &days)
	require.NoError(t, err)
	_, err = env.db.NewUpdate().Table("mbflow_system_keys").Set("expires_at = ?", time.Now().Add(-time.Hour)).
		Where("id = ?", expired.Key.ID).Exec(ctx)
	require.NoError(t, err)
	w = rc1Do(t, env.router, http.MethodGet, "/api/v1/service/workflows", nil, map[string]string{"X-System-Key": expired.PlainKey})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "system key has expired", rc1JSON(t, w)["message"])
}

func TestRC1SystemAuth_Impersonation(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	env.rc1Provision(t)

	w := env.rc1Svc(t, http.MethodGet, "/workflows", nil, map[string]string{"X-On-Behalf-Of": "not-a-uuid"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid user ID format", rc1JSON(t, w)["message"])

	w = env.rc1Svc(t, http.MethodGet, "/workflows", nil, map[string]string{"X-On-Behalf-Of": uuid.NewString()})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "user not found for impersonation", rc1JSON(t, w)["message"])

	// Impersonated creation records the impersonated user as the owner.
	w = env.rc1Svc(t, http.MethodPost, "/workflows", map[string]any{"name": "on behalf"},
		map[string]string{"X-On-Behalf-Of": env.userID.String()})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	wfID := uuid.MustParse(rc1JSON(t, w)["id"].(string))
	stored, err := env.workflowRepo.FindByID(context.Background(), wfID)
	require.NoError(t, err)
	require.NotNil(t, stored.CreatedBy)
	assert.Equal(t, env.userID, *stored.CreatedBy)

	entry := env.rc1WaitAudit(t, "workflow.create")
	assert.Equal(t, "billing", entry.ServiceName)
	assert.Equal(t, "workflow", entry.ResourceType)
	require.NotNil(t, entry.ImpersonatedUserID)
	assert.Equal(t, env.userID, *entry.ImpersonatedUserID)
	assert.Equal(t, http.StatusCreated, entry.ResponseStatus)
	assert.Equal(t, "/api/v1/service/workflows", entry.RequestPath)
	assert.Equal(t, http.MethodPost, entry.RequestMethod)

	// Without impersonation the configured system user owns the workflow.
	w = env.rc1Svc(t, http.MethodPost, "/workflows", map[string]any{"name": "as system"}, nil)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	stored, err = env.workflowRepo.FindByID(context.Background(), uuid.MustParse(rc1JSON(t, w)["id"].(string)))
	require.NoError(t, err)
	require.NotNil(t, stored.CreatedBy)
	assert.Equal(t, env.systemUserID, *stored.CreatedBy)
}

func TestRC1SystemAuth_ImpersonationLookupFailure(t *testing.T) {
	t.Parallel()
	log := rc1Logger()
	db := rc1ClosedDB(t)
	m := NewSystemAuthMiddleware(nil, storage.NewUserRepository(db), "sys", log)

	r := gin.New()
	r.GET("/x", m.HandleImpersonation(), func(c *gin.Context) { c.Status(http.StatusOK) })
	w := rc1Do(t, r, http.MethodGet, "/x", nil, map[string]string{"X-On-Behalf-Of": uuid.NewString()})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "failed to validate user", rc1JSON(t, w)["message"])
}

// ---------------------------------------------------------------------------
// Audit middleware
// ---------------------------------------------------------------------------

func TestRC1AuditMiddleware_RecordsActionsAndSkipsUnauthenticated(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	env.rc1Provision(t)

	// Requests without a system key in context are passed through untouched.
	audit := NewAuditMiddleware(systemkey.NewAuditService(env.auditRepo, 0), rc1Logger())
	r := gin.New()
	r.POST("/plain", audit.RecordAction(), func(c *gin.Context) {
		var body map[string]any
		require.NoError(t, c.ShouldBindJSON(&body))
		c.JSON(http.StatusOK, body)
	})
	w := rc1Do(t, r, http.MethodPost, "/plain", map[string]any{"k": "v"}, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "v", rc1JSON(t, w)["k"], "body must still be readable downstream")

	// Secrets in audited request bodies are not stored verbatim.
	w = env.rc1Svc(t, http.MethodPost, "/credentials", map[string]any{
		"name": "c", "credential_type": "api_key", "data": map[string]string{"api_key": "super-secret"},
	}, nil)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	credID := rc1JSON(t, w)["ID"]
	entry := env.rc1WaitAudit(t, "credential.create")
	require.NotNil(t, entry.RequestBody)
	assert.NotContains(t, *entry.RequestBody, "super-secret")
	assert.Nil(t, entry.ImpersonatedUserID)
	assert.Equal(t, env.keyID, entry.SystemKeyID.String())
	assert.NotNil(t, credID)

	// A failure on the audit store is logged, not surfaced to the client.
	broken := NewAuditMiddleware(systemkey.NewAuditService(storage.NewServiceAuditLogRepo(rc1ClosedDB(t)), 0), rc1Logger())
	r2 := gin.New()
	r2.GET("/api/v1/service/workflows", func(c *gin.Context) {
		c.Set(ContextKeySystemKeyID, env.keyID)
		c.Set(ContextKeyServiceName, "svc")
		c.Set(ContextKeyImpersonated, true)
		c.Set(ContextKeyUserID, env.userID.String())
		c.Next()
	}, broken.RecordAction(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	w = rc1Do(t, r2, http.MethodGet, "/api/v1/service/workflows", nil, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRC1ParseServiceAPIPath(t *testing.T) {
	t.Parallel()
	longID := uuid.NewString()
	cases := []struct {
		path, method, action, resource string
		id                             *string
	}{
		{"/api/v1/service/workflows", http.MethodGet, "workflow.list", "workflow", nil},
		{"/api/v1/service/workflows/" + longID, http.MethodGet, "workflow.get", "workflow", &longID},
		{"/api/v1/service/workflows", http.MethodPost, "workflow.create", "workflow", nil},
		{"/api/v1/service/workflows/" + longID + "/execute", http.MethodPost, "workflow.execute", "workflow", &longID},
		{"/api/v1/service/triggers/" + longID, http.MethodPut, "trigger.update", "trigger", &longID},
		{"/api/v1/service/credentials/short", http.MethodDelete, "credential.delete", "credential", nil},
		{"/api/v1/service/audit-log", http.MethodGet, "audit_log.list", "audit_log", nil},
		{"/api/v1/service/workflows/x", http.MethodPatch, "workflow.patch", "workflow", nil},
	}
	for _, tc := range cases {
		action, resource, id := parseServiceAPIPath(tc.path, tc.method)
		assert.Equal(t, tc.action, action, tc.path)
		assert.Equal(t, tc.resource, resource, tc.path)
		if tc.id == nil {
			assert.Nil(t, id, tc.path)
		} else {
			require.NotNil(t, id, tc.path)
			assert.Equal(t, *tc.id, *id)
		}
	}
}

// ---------------------------------------------------------------------------
// System key admin handlers
// ---------------------------------------------------------------------------

func TestRC1SystemKeyHandlers_Lifecycle(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	adminID := rc1CreateUser(t, env.db, "admin")
	admin := map[string]string{"X-Test-User": adminID.String()}
	base := "/api/v1/service/system-keys"

	// Authentication and validation.
	w := rc1Do(t, env.router, http.MethodPost, base, map[string]any{"name": "n", "service_name": "s"}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, env.router, http.MethodPost, base, map[string]any{"name": "n"}, admin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, rc1JSON(t, w)["message"], "servicename is required")

	days := 7
	w = rc1Do(t, env.router, http.MethodPost, base, map[string]any{
		"name": "billing key", "description": "d", "service_name": "billing", "expires_in_days": days,
	}, admin)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	created := rc1JSON(t, w)
	assert.True(t, strings.HasPrefix(created["key"].(string), "sysk_"))
	assert.Equal(t, "billing", created["service_name"])
	assert.Equal(t, adminID.String(), created["created_by"])
	assert.NotEmpty(t, created["expires_at"])
	assert.NotEmpty(t, created["warning"])
	keyID := created["id"].(string)

	// The plain key authenticates against the service API.
	w = rc1Do(t, env.router, http.MethodGet, "/api/v1/service/workflows", nil, map[string]string{"X-System-Key": created["key"].(string)})
	assert.Equal(t, http.StatusOK, w.Code)

	for _, n := range []string{"two", "three"} {
		w = rc1Do(t, env.router, http.MethodPost, base, map[string]any{"name": n, "service_name": "other"}, admin)
		require.Equal(t, http.StatusCreated, w.Code)
	}
	// MaxKeys is 3.
	w = rc1Do(t, env.router, http.MethodPost, base, map[string]any{"name": "four", "service_name": "other"}, admin)
	assert.Equal(t, http.StatusConflict, w.Code)

	// List with filters; the plain key and hash are never listed.
	w = rc1Do(t, env.router, http.MethodGet, base+"?limit=500&service_name=billing&status=active", nil, admin)
	require.Equal(t, http.StatusOK, w.Code)
	list := rc1JSON(t, w)
	assert.EqualValues(t, 100, list["limit"])
	assert.EqualValues(t, 1, list["total"])
	data := list["data"].([]any)
	require.Len(t, data, 1)
	item := data[0].(map[string]any)
	assert.Equal(t, keyID, item["id"])
	assert.NotContains(t, item, "key")
	assert.NotContains(t, item, "key_hash")

	// Get.
	w = rc1Do(t, env.router, http.MethodGet, base+"/"+keyID, nil, admin)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "billing key", rc1JSON(t, w)["name"])
	w = rc1Do(t, env.router, http.MethodGet, base+"/bad", nil, admin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, env.router, http.MethodGet, base+"/"+uuid.NewString(), nil, admin)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, env.router, http.MethodGet, base+"/noparam", nil, admin)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Revoke.
	w = rc1Do(t, env.router, http.MethodPost, base+"/"+keyID+"/revoke", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, env.router, http.MethodPost, base+"/bad/revoke", nil, admin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, env.router, http.MethodPost, base+"/"+uuid.NewString()+"/revoke", nil, admin)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, env.router, http.MethodPost, base+"/noparam/revoke", nil, admin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, env.router, http.MethodPost, base+"/"+keyID+"/revoke", nil, admin)
	require.Equal(t, http.StatusOK, w.Code)
	w = rc1Do(t, env.router, http.MethodGet, base+"/"+keyID, nil, admin)
	assert.Equal(t, "revoked", rc1JSON(t, w)["status"])
	w = rc1Do(t, env.router, http.MethodGet, "/api/v1/service/workflows", nil, map[string]string{"X-System-Key": created["key"].(string)})
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Delete.
	w = rc1Do(t, env.router, http.MethodDelete, base+"/"+keyID, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, env.router, http.MethodDelete, base+"/bad", nil, admin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, env.router, http.MethodDelete, base+"/noparam", nil, admin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, env.router, http.MethodDelete, base+"/"+keyID, nil, admin)
	require.Equal(t, http.StatusOK, w.Code)
	w = rc1Do(t, env.router, http.MethodDelete, base+"/"+keyID, nil, admin)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRC1SystemKeyHandlers_StorageFailures(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1ClosedDB(t), true)
	admin := map[string]string{"X-Test-User": uuid.NewString()}
	base := "/api/v1/service/system-keys"
	id := uuid.NewString()

	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, base},
		{http.MethodGet, base},
		{http.MethodGet, base + "/" + id},
		{http.MethodDelete, base + "/" + id},
		{http.MethodPost, base + "/" + id + "/revoke"},
	} {
		w := rc1Do(t, env.router, tc.method, tc.path, map[string]any{"name": "n", "service_name": "s"}, admin)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "%s %s", tc.method, tc.path)
	}
	// A broken key store rejects all system keys.
	w := rc1Do(t, env.router, http.MethodGet, "/api/v1/service/workflows", nil, map[string]string{"X-System-Key": "sysk_" + strings.Repeat("a", 40)})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// Service API workflows / executions / triggers / credentials / audit
// ---------------------------------------------------------------------------

func TestRC1ServiceAPIWorkflows(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	env.rc1Provision(t)

	// Create.
	w := env.rc1Svc(t, http.MethodPost, "/workflows", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPost, "/workflows", map[string]any{
		"name": "wf", "description": "d", "variables": map[string]any{"a": 1}, "metadata": map[string]any{"m": "x"},
	}, map[string]string{"X-On-Behalf-Of": env.userID.String()})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	wfID := rc1JSON(t, w)["id"].(string)

	// Update with nodes, edges and an invalid resource reference.
	nodes := []map[string]any{
		{"id": "a", "name": "A", "type": "transform", "config": map[string]any{"type": "passthrough"}},
		{"id": "b", "name": "B", "type": "transform", "config": map[string]any{"type": "passthrough"}},
	}
	edges := []map[string]any{{"id": "e1", "from": "a", "to": "b"}}
	w = env.rc1Svc(t, http.MethodPut, "/workflows/"+wfID, map[string]any{
		"name": "wf2", "nodes": nodes, "edges": edges,
	}, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	updated := rc1JSON(t, w)
	assert.Equal(t, "wf2", updated["name"])
	assert.Len(t, updated["nodes"], 2)
	assert.Len(t, updated["edges"], 1)

	w = env.rc1Svc(t, http.MethodPut, "/workflows/"+wfID, map[string]any{
		"resources": []map[string]any{{"resource_id": "not-a-uuid", "alias": "x"}},
	}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_RESOURCE_ID", rc1JSON(t, w)["code"])

	w = env.rc1Svc(t, http.MethodPut, "/workflows/"+wfID, map[string]any{
		"nodes": []map[string]any{{"id": "a", "name": "A", "type": "no-such-executor"}},
	}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "NODE_VALIDATION_FAILED", rc1JSON(t, w)["code"])

	w = env.rc1Svc(t, http.MethodPut, "/workflows/"+wfID, "{bad", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPut, "/workflows/bad-id", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPut, "/workflows/"+uuid.NewString(), map[string]any{"name": "x"}, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Get.
	w = env.rc1Svc(t, http.MethodGet, "/workflows/"+wfID, nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "wf2", rc1JSON(t, w)["name"])
	w = env.rc1Svc(t, http.MethodGet, "/workflows/bad-id", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodGet, "/workflows/"+uuid.NewString(), nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// List with filters.
	w = env.rc1Svc(t, http.MethodGet, "/workflows?user_id="+env.userID.String()+"&status=draft&limit=10", nil, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	list := rc1JSON(t, w)
	assert.EqualValues(t, 10, list["limit"])
	assert.Len(t, list["data"], 1)
	w = env.rc1Svc(t, http.MethodGet, "/workflows?user_id="+uuid.NewString(), nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["data"], 0)
	w = env.rc1Svc(t, http.MethodGet, "/workflows?user_id=nope", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_USER_ID", rc1JSON(t, w)["code"])

	// Delete.
	w = env.rc1Svc(t, http.MethodDelete, "/workflows/bad-id", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodDelete, "/workflows/"+wfID, nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	w = env.rc1Svc(t, http.MethodGet, "/workflows/"+wfID, nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Missing path parameters.
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/noparam/workflow"}, {http.MethodPut, "/noparam/workflow"}, {http.MethodDelete, "/noparam/workflow"},
	} {
		w = env.rc1Svc(t, tc.method, tc.path, map[string]any{}, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, tc.path)
		assert.Equal(t, "MISSING_PARAMETER", rc1JSON(t, w)["code"])
	}

	env.rc1WaitAudit(t, "workflow.update")
	env.rc1WaitAudit(t, "workflow.delete")
}

func TestRC1ServiceAPIExecutions(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	env.rc1Provision(t)

	wf := testutil.WorkflowDomainToModel(testutil.CreateSimpleWorkflow())
	require.NoError(t, env.workflowRepo.Create(context.Background(), wf))

	// Start (async) and fetch.
	w := env.rc1Svc(t, http.MethodPost, "/workflows/"+wf.ID.String()+"/execute", map[string]any{"input": map[string]any{"x": 1}}, nil)
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	execID := rc1JSON(t, w)["id"].(string)
	w = env.rc1Svc(t, http.MethodPost, "/workflows/"+wf.ID.String()+"/execute", "{bad", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPost, "/workflows/"+uuid.NewString()+"/execute", map[string]any{}, nil)
	assert.GreaterOrEqual(t, w.Code, 400)

	require.Eventually(t, func() bool {
		w := env.rc1Svc(t, http.MethodGet, "/executions/"+execID, nil, nil)
		return w.Code == http.StatusOK && rc1JSON(t, w)["status"] == "completed"
	}, 10*time.Second, 50*time.Millisecond)
	w = env.rc1Svc(t, http.MethodGet, "/executions/bad", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodGet, "/executions/"+uuid.NewString(), nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// List.
	w = env.rc1Svc(t, http.MethodGet, "/executions?workflow_id="+wf.ID.String(), nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["data"], 1)
	w = env.rc1Svc(t, http.MethodGet, "/executions?status=completed", nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["data"], 1)
	w = env.rc1Svc(t, http.MethodGet, "/executions?workflow_id=bad", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Ephemeral: validation.
	inline := map[string]any{
		"name":  "inline",
		"nodes": []map[string]any{{"id": "n1", "name": "N1", "type": "transform", "config": map[string]any{"type": "passthrough"}}},
		"edges": []map[string]any{},
	}
	for _, tc := range []struct {
		body any
		code int
		ec   string
	}{
		{map[string]any{"workflow": inline}, http.StatusBadRequest, "MODE_REQUIRED"},
		{map[string]any{"workflow": inline, "mode": "later"}, http.StatusBadRequest, "INVALID_MODE"},
		{map[string]any{"workflow": "not an object", "mode": "sync"}, http.StatusBadRequest, "INVALID_WORKFLOW"},
		{map[string]any{"workflow": map[string]any{"name": strings.Repeat("x", maxWorkflowSnapshotSize)}, "mode": "sync"}, http.StatusRequestEntityTooLarge, "WORKFLOW_TOO_LARGE"},
		{map[string]any{"workflow": inline, "mode": "sync", "credential_ids": []string{"x"}}, http.StatusBadRequest, "UNSUPPORTED_CREDENTIAL_IDS"},
		{map[string]any{"workflow": inline, "mode": "async", "webhooks": []map[string]any{{"url": ""}}}, http.StatusBadRequest, "INVALID_WEBHOOK"},
	} {
		w = env.rc1Svc(t, http.MethodPost, "/executions/ephemeral", tc.body, nil)
		assert.Equal(t, tc.code, w.Code, w.Body.String())
		assert.Equal(t, tc.ec, rc1JSON(t, w)["code"])
	}
	w = env.rc1Svc(t, http.MethodPost, "/executions/ephemeral", "{bad", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Ephemeral sync run completes in the request; edges without IDs get one.
	inline2 := map[string]any{
		"name": "inline2",
		"nodes": []map[string]any{
			{"id": "n1", "name": "N1", "type": "transform", "config": map[string]any{"type": "passthrough"}},
			{"id": "n2", "name": "N2", "type": "transform", "config": map[string]any{"type": "passthrough"}},
		},
		"edges": []map[string]any{{"from": "n1", "to": "n2"}},
	}
	w = env.rc1Svc(t, http.MethodPost, "/executions/ephemeral", map[string]any{
		"workflow": inline2, "mode": "sync", "input": map[string]any{"v": 1},
	}, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "completed", rc1JSON(t, w)["status"])

	w = env.rc1Svc(t, http.MethodPost, "/executions/ephemeral", map[string]any{
		"workflow": inline, "mode": "async",
		"webhooks": []map[string]any{{"url": rc1HookURL(t), "events": []string{"execution.completed"}, "headers": map[string]string{"a": "b"}, "node_ids": []string{"n1"}}},
	}, nil)
	assert.Equal(t, http.StatusAccepted, w.Code, w.Body.String())

	// Not implemented endpoints.
	w = env.rc1Svc(t, http.MethodPost, "/executions/"+execID+"/cancel", nil, nil)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	w = env.rc1Svc(t, http.MethodPost, "/executions/"+execID+"/retry", nil, nil)
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/noparam/execution"}, {http.MethodPost, "/noparam/execute"},
	} {
		w = env.rc1Svc(t, tc.method, tc.path, map[string]any{}, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, tc.path)
	}
}

func TestRC1ServiceAPITriggers(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	env.rc1Provision(t)

	wf := testutil.WorkflowDomainToModel(testutil.CreateSimpleWorkflow())
	require.NoError(t, env.workflowRepo.Create(context.Background(), wf))

	w := env.rc1Svc(t, http.MethodPost, "/triggers", map[string]any{"name": "t"}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPost, "/triggers", map[string]any{"workflow_id": wf.ID.String(), "name": "t", "type": "bogus"}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_TRIGGER_TYPE", rc1JSON(t, w)["code"])
	w = env.rc1Svc(t, http.MethodPost, "/triggers", map[string]any{"workflow_id": uuid.NewString(), "name": "t", "type": "manual"}, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = env.rc1Svc(t, http.MethodPost, "/triggers", map[string]any{
		"workflow_id": wf.ID.String(), "name": "t", "description": "d", "type": "manual", "enabled": true,
	}, nil)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	trigger := rc1JSON(t, w)
	trID := trigger["id"].(string)
	assert.Equal(t, "manual", trigger["type"])

	// List.
	for _, q := range []string{"", "?workflow_id=" + wf.ID.String(), "?type=manual"} {
		w = env.rc1Svc(t, http.MethodGet, "/triggers"+q, nil, nil)
		require.Equal(t, http.StatusOK, w.Code, q)
		assert.Len(t, rc1JSON(t, w)["data"], 1, q)
	}
	w = env.rc1Svc(t, http.MethodGet, "/triggers?workflow_id=bad", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Update.
	disabled := false
	w = env.rc1Svc(t, http.MethodPut, "/triggers/"+trID, map[string]any{"name": "t2", "enabled": disabled}, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, false, rc1JSON(t, w)["enabled"])
	w = env.rc1Svc(t, http.MethodPut, "/triggers/"+trID, map[string]any{"type": "bogus"}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPut, "/triggers/"+trID, "{bad", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPut, "/triggers/bad", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPut, "/triggers/"+uuid.NewString(), map[string]any{}, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Delete.
	w = env.rc1Svc(t, http.MethodDelete, "/triggers/bad", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodDelete, "/triggers/"+trID, nil, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	stored, _ := env.triggerRepo.FindByID(context.Background(), uuid.MustParse(trID))
	assert.Nil(t, stored)

	for _, tc := range []struct{ method, path string }{
		{http.MethodPut, "/noparam/trigger"}, {http.MethodDelete, "/noparam/trigger"},
	} {
		w = env.rc1Svc(t, tc.method, tc.path, map[string]any{}, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, tc.path)
	}
}

func TestRC1ServiceAPICredentials(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	env.rc1Provision(t)
	onBehalf := map[string]string{"X-On-Behalf-Of": env.userID.String()}

	w := env.rc1Svc(t, http.MethodPost, "/credentials", map[string]any{"name": "c"}, onBehalf)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPost, "/credentials", map[string]any{
		"name": "c", "credential_type": "nope", "data": map[string]string{"k": "v"},
	}, onBehalf)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_CREDENTIAL_TYPE", rc1JSON(t, w)["code"])

	w = env.rc1Svc(t, http.MethodPost, "/credentials", map[string]any{
		"name": "key", "description": "d", "credential_type": "api_key", "provider": "openai",
		"data": map[string]string{"api_key": "sk-1"},
	}, onBehalf)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	created := rc1JSON(t, w)
	credID := created["ID"].(string)
	assert.NotContains(t, w.Body.String(), "sk-1")

	// List requires user_id; filters by provider.
	w = env.rc1Svc(t, http.MethodGet, "/credentials", nil, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "USER_ID_REQUIRED", rc1JSON(t, w)["code"])
	w = env.rc1Svc(t, http.MethodGet, "/credentials?user_id="+env.userID.String(), nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["credentials"], 1)
	w = env.rc1Svc(t, http.MethodGet, "/credentials?user_id="+env.userID.String()+"&provider=anthropic", nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["credentials"], 0)

	// Update.
	w = env.rc1Svc(t, http.MethodPut, "/credentials/"+credID, map[string]any{"name": "renamed", "description": "new"}, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "renamed", rc1JSON(t, w)["Name"])
	w = env.rc1Svc(t, http.MethodPut, "/credentials/"+credID, "{bad", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.rc1Svc(t, http.MethodPut, "/credentials/"+uuid.NewString(), map[string]any{"name": "x"}, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Delete.
	w = env.rc1Svc(t, http.MethodDelete, "/credentials/"+uuid.NewString(), nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = env.rc1Svc(t, http.MethodDelete, "/credentials/"+credID, nil, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = env.rc1Svc(t, http.MethodGet, "/credentials?user_id="+env.userID.String(), nil, nil)
	assert.Len(t, rc1JSON(t, w)["credentials"], 0)

	for _, tc := range []struct{ method, path string }{
		{http.MethodPut, "/noparam/credential"}, {http.MethodDelete, "/noparam/credential"},
	} {
		w = env.rc1Svc(t, tc.method, tc.path, map[string]any{}, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, tc.path)
	}

	// Without a user in context creation is unauthorized.
	w = rc1Do(t, env.router, http.MethodPost, "/raw/service/credentials", map[string]any{
		"name": "c", "credential_type": "api_key", "data": map[string]string{"k": "v"},
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Without an encryption key credentials are disabled.
	noEnc := rc1NewServiceEnv(t, env.db, false)
	w = rc1Do(t, noEnc.router, http.MethodPost, "/raw/service/credentials", map[string]any{
		"name": "c", "credential_type": "api_key", "data": map[string]string{"k": "v"},
	}, map[string]string{"X-Test-User": env.userID.String()})
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestRC1ServiceAPIAuditLog(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1DB(t), true)
	env.rc1Provision(t)

	w := env.rc1Svc(t, http.MethodPost, "/workflows", map[string]any{"name": "audited"},
		map[string]string{"X-On-Behalf-Of": env.userID.String()})
	require.Equal(t, http.StatusCreated, w.Code)
	env.rc1WaitAudit(t, "workflow.create")

	from := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	to := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	q := "/audit-log?service_name=billing&action=workflow.create&resource_type=workflow&impersonated_user_id=" +
		env.userID.String() + "&date_from=" + from + "&date_to=" + to
	w = env.rc1Svc(t, http.MethodGet, q, nil, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := rc1JSON(t, w)
	assert.EqualValues(t, 1, body["total"])

	// Unparseable optional filters are ignored rather than rejected.
	w = env.rc1Svc(t, http.MethodGet, "/audit-log?action=workflow.create&impersonated_user_id=x&date_from=x&date_to=x", nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, 1, rc1JSON(t, w)["total"])

	// A filter that matches nothing.
	w = env.rc1Svc(t, http.MethodGet, "/audit-log?service_name=other", nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, 0, rc1JSON(t, w)["total"])
}

func TestRC1ServiceAPI_StorageFailures(t *testing.T) {
	t.Parallel()
	env := rc1NewServiceEnv(t, rc1ClosedDB(t), true)
	user := map[string]string{"X-Test-User": uuid.NewString()}
	id := uuid.NewString()

	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/workflows", nil},
		{http.MethodGet, "/workflows/" + id, nil},
		{http.MethodPost, "/workflows", map[string]any{"name": "x"}},
		{http.MethodPut, "/workflows/" + id, map[string]any{
			"nodes":     []map[string]any{{"id": "a", "name": "A", "type": "transform"}},
			"edges":     []map[string]any{},
			"resources": []map[string]any{{"resource_id": uuid.NewString(), "alias": "r"}},
		}},
		{http.MethodDelete, "/workflows/" + id, nil},
		{http.MethodGet, "/executions", nil},
		{http.MethodGet, "/executions/" + id, nil},
		{http.MethodGet, "/triggers", nil},
		{http.MethodDelete, "/triggers/" + id, nil},
		{http.MethodGet, "/credentials?user_id=" + id, nil},
		{http.MethodPut, "/credentials/" + id, map[string]any{"name": "x"}},
		{http.MethodDelete, "/credentials/" + id, nil},
		{http.MethodPost, "/credentials", map[string]any{"name": "c", "credential_type": "api_key", "data": map[string]string{"k": "v"}}},
		{http.MethodGet, "/audit-log", nil},
	} {
		w := rc1Do(t, env.router, tc.method, "/raw/service"+tc.path, tc.body, user)
		assert.GreaterOrEqual(t, w.Code, 400, "%s %s: %s", tc.method, tc.path, w.Body.String())
		assert.NotEqual(t, http.StatusOK, w.Code)
	}

	// The audit list endpoint reports a generic failure.
	w := rc1Do(t, env.router, http.MethodGet, "/raw/service/audit-log", nil, user)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "failed to list audit logs", rc1JSON(t, w)["message"])
}

// Compile-time check that the repository filter type used by the system key
// handlers stays compatible with the storage implementation.
var _ repository.SystemKeyRepository = (*storage.SystemKeyRepoImpl)(nil)

// rc1HookURL returns the URL of a local server that accepts webhook calls.
func rc1HookURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	t.Cleanup(srv.Close)
	return srv.URL
}
