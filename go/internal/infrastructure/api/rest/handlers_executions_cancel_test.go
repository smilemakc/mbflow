package rest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/serviceapi"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/testutil"
)

const cancelRESTNodeType = "rest-cancel-block"

type cancelRESTEnv struct {
	router   *gin.Engine
	em       *engine.ExecutionManager
	execRepo *storage.ExecutionRepository
	wfID     uuid.UUID
	started  chan struct{}
}

// setupCancelRESTEnv serves the user and service API cancel/retry endpoints
// over a real database; its workflow's node blocks until cancelled.
func setupCancelRESTEnv(t *testing.T) *cancelRESTEnv {
	t.Helper()
	db, cleanup := testutil.SetupTestTx(t)

	env := &cancelRESTEnv{started: make(chan struct{}, 8)}
	registry := executor.NewManager()
	require.NoError(t, registry.Register(cancelRESTNodeType, &executor.ExecutorFunc{
		ExecuteFn: func(ctx context.Context, _ map[string]any, _ any) (any, error) {
			env.started <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}))

	workflowRepo := storage.NewWorkflowRepository(db)
	env.execRepo = storage.NewExecutionRepository(db)
	env.em = engine.NewExecutionManager(registry, workflowRepo, env.execRepo, nil, nil, nil)
	// Shut down before the test transaction is rolled back.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = env.em.Shutdown(ctx)
		cleanup()
	})

	wf := &storagemodels.WorkflowModel{
		Name: "cancel-retry", Status: "active", Version: 1,
		Variables: storagemodels.JSONBMap{}, Metadata: storagemodels.JSONBMap{},
		Nodes: []*storagemodels.NodeModel{
			{NodeID: "n1", Name: "N1", Type: cancelRESTNodeType, Config: storagemodels.JSONBMap{}, Position: storagemodels.JSONBMap{"x": 0, "y": 0}},
		},
	}
	require.NoError(t, workflowRepo.Create(context.Background(), wf))
	env.wfID = wf.ID

	log := logger.New(config.LoggingConfig{Level: "error", Format: "text"})
	ops := &serviceapi.Operations{
		WorkflowRepo:    workflowRepo,
		ExecutionRepo:   env.execRepo,
		ExecutionMgr:    env.em,
		ExecutorManager: registry,
		Logger:          log,
	}

	gin.SetMode(gin.TestMode)
	env.router = gin.New()
	user := NewExecutionHandlers(ops, log)
	env.router.POST("/api/v1/executions/:id/cancel", user.HandleCancelExecution)
	env.router.POST("/api/v1/executions/:id/retry", user.HandleRetryExecution)
	svc := NewServiceAPIExecutionHandlers(ops)
	env.router.POST("/api/v1/service/executions/:id/cancel", svc.CancelExecution)
	env.router.POST("/api/v1/service/executions/:id/retry", svc.RetryExecution)
	return env
}

func (e *cancelRESTEnv) insert(t *testing.T, status, source string, withWorkflow bool) uuid.UUID {
	t.Helper()
	m := &storagemodels.ExecutionModel{
		ID: uuid.New(), Status: status, WorkflowSource: source,
		InputData: storagemodels.JSONBMap{"in": "orig"},
		Variables: storagemodels.JSONBMap{"v": "orig"},
	}
	if withWorkflow {
		m.WorkflowID = &e.wfID
	} else {
		m.WorkflowSnapshot = storagemodels.JSONBMap{"name": "inline"}
	}
	require.NoError(t, e.execRepo.Create(context.Background(), m))
	return m.ID
}

func TestHandlers_CancelAndRetryExecution(t *testing.T) {
	for _, prefix := range []string{"/api/v1/executions/", "/api/v1/service/executions/"} {
		t.Run(prefix, func(t *testing.T) {
			env := setupCancelRESTEnv(t)
			post := func(id, action string) (int, map[string]any) {
				w := testutil.MakeRequest(t, env.router, http.MethodPost, prefix+id+"/"+action, nil)
				var body map[string]any
				testutil.ParseResponse(t, w, &body)
				return w.Code, body
			}

			// A running execution owned by this process: 200 once stopped.
			exec, err := env.em.ExecuteAsync(context.Background(), env.wfID.String(), map[string]any{"in": "x"}, nil)
			require.NoError(t, err)
			select {
			case <-env.started:
			case <-time.After(5 * time.Second):
				t.Fatal("node did not start")
			}
			code, body := post(exec.ID, "cancel")
			require.Equal(t, http.StatusOK, code, body)
			assert.Equal(t, "cancelled", body["status"])
			assert.Equal(t, engine.CancelledExecutionError, body["error"])

			// Running on another replica: recorded as cancelled, 202.
			remote := env.insert(t, "running", "stored", true)
			code, body = post(remote.String(), "cancel")
			require.Equal(t, http.StatusAccepted, code, body)
			assert.Equal(t, "cancelled", body["status"])

			// Already finished, unknown, malformed.
			code, body = post(remote.String(), "cancel")
			assert.Equal(t, http.StatusConflict, code)
			assert.Equal(t, serviceapi.CodeExecutionNotCancellable, body["code"])
			code, body = post(uuid.NewString(), "cancel")
			assert.Equal(t, http.StatusNotFound, code)
			assert.Equal(t, serviceapi.CodeExecutionNotFound, body["code"])
			code, _ = post("bad", "cancel")
			assert.Equal(t, http.StatusBadRequest, code)

			// Retry the cancelled execution: a new execution, 202.
			code, body = post(remote.String(), "retry")
			require.Equal(t, http.StatusAccepted, code, body)
			assert.NotEqual(t, remote.String(), body["id"])
			assert.Equal(t, env.wfID.String(), body["workflow_id"])
			assert.Equal(t, map[string]any{"in": "orig"}, body["input"])
			assert.Equal(t, map[string]any{serviceapi.MetadataKeyRetryOf: remote.String()}, body["metadata"])
			select {
			case <-env.started:
			case <-time.After(5 * time.Second):
				t.Fatal("the retry did not start")
			}
			code, _ = post(body["id"].(string), "cancel")
			require.Equal(t, http.StatusOK, code, "the retry can be cancelled in turn")

			// Not retryable: still running, inline, unknown, malformed.
			running := env.insert(t, "running", "stored", true)
			code, body = post(running.String(), "retry")
			assert.Equal(t, http.StatusConflict, code)
			assert.Equal(t, serviceapi.CodeExecutionNotRetryable, body["code"])
			inline := env.insert(t, "failed", "inline", false)
			code, body = post(inline.String(), "retry")
			assert.Equal(t, http.StatusConflict, code)
			assert.Contains(t, body["message"], "inline executions")
			code, _ = post(uuid.NewString(), "retry")
			assert.Equal(t, http.StatusNotFound, code)
			code, _ = post("bad", "retry")
			assert.Equal(t, http.StatusBadRequest, code)
		})
	}
}
