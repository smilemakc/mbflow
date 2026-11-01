package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/trigger"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

func TestRC2_Webhooks(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	wfID := env.createWorkflow(t, env.router(alice, false, true), rc2SimpleWorkflow("hook-wf"))["id"].(string)

	mr, _ := setupMiniRedis(t)
	rc, err := cache.NewRedisCache(config.RedisConfig{URL: "redis://" + mr.Addr()})
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })

	registry := trigger.NewWebhookRegistry(trigger.WebhookRegistryConfig{
		TriggerRepo:  env.triggerRepo,
		WorkflowRepo: env.workflowRepo,
		ExecutionMgr: env.execMgr,
		Cache:        rc,
	})
	ctx := context.Background()
	lastRun := time.Now().Add(-time.Hour)
	register := func(enabled bool, cfg map[string]any) string {
		id := uuid.NewString()
		// Executions reference their trigger, so it must exist in the DB.
		require.NoError(t, env.triggerRepo.Create(ctx, &storagemodels.TriggerModel{
			ID: uuid.MustParse(id), WorkflowID: uuid.MustParse(wfID), Type: "webhook", Enabled: enabled,
			Config: storagemodels.JSONBMap(cfg), CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
		require.NoError(t, registry.RegisterWebhook(ctx, &pkgmodels.Trigger{
			ID: id, WorkflowID: wfID, Type: pkgmodels.TriggerTypeWebhook, Enabled: enabled,
			Config: cfg, CreatedAt: time.Now(), UpdatedAt: time.Now(), LastRun: &lastRun,
		}))
		return id
	}
	open := register(true, map[string]any{"input": map[string]any{"default": "yes"}})
	disabled := register(false, map[string]any{})
	signed := register(true, map[string]any{"secret": "s3cret"})
	whitelisted := register(true, map[string]any{"ip_whitelist": []any{"10.0.0.0/8"}})
	missingWorkflow := uuid.NewString()
	require.NoError(t, registry.RegisterWebhook(ctx, &pkgmodels.Trigger{
		ID: missingWorkflow, WorkflowID: uuid.NewString(), Type: pkgmodels.TriggerTypeWebhook, Enabled: true, Config: map[string]any{},
	}))

	h := NewWebhookHandlers(registry, env.log)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/webhooks/:trigger_id", h.HandleWebhook)
	r.GET("/webhooks/:trigger_id", h.HandleWebhookGet)
	r.POST("/webhooks", h.HandleWebhook)
	r.GET("/webhooks", h.HandleWebhookGet)

	post := func(id, body string, headers map[string]string, remote string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/"+id, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		req.RemoteAddr = remote
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	const client = "192.0.2.10:5555"

	// Missing ID / invalid body.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/webhooks", strings.NewReader("{}")))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/webhooks", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, http.StatusBadRequest, post(open, "{bad", nil, client).Code)

	// Error mapping.
	assert.Equal(t, http.StatusNotFound, post(uuid.NewString(), "{}", nil, client).Code)
	assert.Equal(t, http.StatusForbidden, post(disabled, "{}", nil, client).Code)
	assert.Equal(t, http.StatusUnauthorized, post(signed, "{}", nil, client).Code)
	assert.Equal(t, http.StatusUnauthorized, post(signed, "{}", map[string]string{"X-Webhook-Signature": "deadbeef"}, client).Code)
	assert.Equal(t, http.StatusForbidden, post(whitelisted, "{}", nil, client).Code)
	// Forwarding headers from an untrusted client must not bypass the whitelist.
	spoof := map[string]string{"X-Forwarded-For": "10.1.2.3", "X-Real-IP": "10.1.2.3"}
	r.ForwardedByClientIP = true
	require.NoError(t, r.SetTrustedProxies(nil))
	assert.Equal(t, http.StatusForbidden, post(whitelisted, "{}", spoof, client).Code, "spoofed X-Forwarded-For is ignored")
	assert.Equal(t, http.StatusInternalServerError, post(missingWorkflow, "{}", nil, client).Code, "execution failures are server errors")

	// Allowed source address executes the workflow.
	w = post(whitelisted, `{"k":"v"}`, nil, "10.9.8.7:1")
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	assert.NotEmpty(t, rc2DecodeMap(t, w)["execution_id"])

	w = post(open, `{"k":"v"}`, map[string]string{"X-Custom": "1"}, client)
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	execID := rc2DecodeMap(t, w)["execution_id"].(string)
	m, err := env.execRepo.FindByID(ctx, uuid.MustParse(execID))
	require.NoError(t, err)
	assert.Equal(t, "yes", m.InputData["default"], "trigger default input is merged")
	assert.Equal(t, "v", m.InputData["k"])

	// Rate limit.
	require.NoError(t, mr.Set("trigger:"+open+":ratelimit", "100"))
	assert.Equal(t, http.StatusTooManyRequests, post(open, "{}", nil, client).Code)

	// Generic failures map to 500.
	execFails := register(true, map[string]any{})
	require.NoError(t, env.db.Close())
	assert.Equal(t, http.StatusInternalServerError, post(execFails, "{}", nil, client).Code)

	// GET returns non-sensitive configuration only.
	get := func(id string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/webhooks/"+id, nil))
		return w
	}
	assert.Equal(t, http.StatusNotFound, get(uuid.NewString()).Code)
	w = get(signed)
	require.Equal(t, http.StatusOK, w.Code)
	info := rc2DecodeMap(t, w)
	assert.Equal(t, wfID, info["workflow_id"])
	assert.NotNil(t, info["last_run"])
	assert.Equal(t, true, info["config"].(map[string]any)["signature_validation_enabled"])
	assert.NotContains(t, w.Body.String(), "s3cret", "secrets are never returned")
	w = get(whitelisted)
	cfg := rc2DecodeMap(t, w)["config"].(map[string]any)
	assert.Equal(t, true, cfg["ip_whitelist_enabled"])
	assert.Equal(t, []any{"10.0.0.0/8"}, cfg["ip_whitelist"])
}
