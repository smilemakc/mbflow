package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/trigger"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// rc1TelegramExecutor records webhook-started executions.
type rc1TelegramExecutor struct {
	mu     sync.Mutex
	inputs []map[string]any
	err    error
}

func (e *rc1TelegramExecutor) Execute(_ context.Context, workflowID string, input map[string]any, _ *engine.ExecutionOptions) (*models.Execution, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.err != nil {
		return nil, e.err
	}
	e.inputs = append(e.inputs, input)
	return &models.Execution{ID: "exec-" + workflowID}, nil
}

func (e *rc1TelegramExecutor) last() map[string]any {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.inputs[len(e.inputs)-1]
}

type rc1TelegramEnv struct {
	router   *gin.Engine
	registry *trigger.WebhookRegistry
	exec     *rc1TelegramExecutor
	redis    *miniredis.Miniredis
}

func rc1NewTelegramEnv(t *testing.T) *rc1TelegramEnv {
	t.Helper()
	mr := miniredis.RunT(t)
	rc, err := cache.NewRedisCache(config.RedisConfig{URL: "redis://" + mr.Addr(), PoolSize: 2})
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })

	db := rc1DB(t)
	exec := &rc1TelegramExecutor{}
	registry := trigger.NewWebhookRegistry(trigger.WebhookRegistryConfig{
		TriggerRepo:  storage.NewTriggerRepository(db),
		WorkflowRepo: storage.NewWorkflowRepository(db),
		ExecutionMgr: exec,
		Cache:        rc,
	})
	h := NewTelegramWebhookHandlers(registry, rc1Logger())
	r := gin.New()
	r.POST("/api/v1/webhooks/telegram/:trigger_id", h.HandleTelegramWebhook)
	r.POST("/telegram-noparam", h.HandleTelegramWebhook)
	return &rc1TelegramEnv{router: r, registry: registry, exec: exec, redis: mr}
}

func (e *rc1TelegramEnv) rc1Register(t *testing.T, enabled bool, cfg map[string]any) string {
	t.Helper()
	trig := &models.Trigger{
		ID: uuid.NewString(), WorkflowID: "wf-tg", Type: models.TriggerTypeWebhook, Enabled: enabled, Config: cfg,
	}
	require.NoError(t, e.registry.RegisterWebhook(context.Background(), trig))
	return trig.ID
}

func (e *rc1TelegramEnv) rc1Post(t *testing.T, triggerID string, update any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	return rc1Do(t, e.router, http.MethodPost, "/api/v1/webhooks/telegram/"+triggerID, update, headers)
}

type rc1FailingReader struct{}

func (rc1FailingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestRC1TelegramWebhook_Rejections(t *testing.T) {
	t.Parallel()
	env := rc1NewTelegramEnv(t)

	w := rc1Do(t, env.router, http.MethodPost, "/telegram-noparam", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = env.rc1Post(t, uuid.NewString(), map[string]any{"update_id": 1}, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	disabled := env.rc1Register(t, false, nil)
	w = env.rc1Post(t, disabled, map[string]any{"update_id": 1}, nil)
	assert.Equal(t, http.StatusForbidden, w.Code)

	secured := env.rc1Register(t, true, map[string]any{"secret_token": "tg-secret"})
	w = env.rc1Post(t, secured, map[string]any{"update_id": 1}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = env.rc1Post(t, secured, map[string]any{"update_id": 1}, map[string]string{"X-Telegram-Bot-Api-Secret-Token": "wrong"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = env.rc1Post(t, secured, map[string]any{"update_id": 1}, map[string]string{"X-Telegram-Bot-Api-Secret-Token": "tg-secret"})
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	open := env.rc1Register(t, true, nil)
	w = env.rc1Post(t, open, "{not json", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid Telegram update format", rc1JSON(t, w)["message"])

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/telegram/"+open, rc1FailingReader{})
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "failed to read request body", rc1JSON(t, rec)["message"])
}

func TestRC1TelegramWebhook_ExecutionErrorsMapToStatus(t *testing.T) {
	t.Parallel()
	env := rc1NewTelegramEnv(t)
	id := env.rc1Register(t, true, nil)

	for _, tc := range []struct {
		err  error
		code int
	}{
		{errors.New("workflow not found"), http.StatusNotFound},
		{errors.New("workflow is disabled"), http.StatusForbidden},
		{errors.New("boom"), http.StatusInternalServerError},
	} {
		env.exec.mu.Lock()
		env.exec.err = tc.err
		env.exec.mu.Unlock()
		w := env.rc1Post(t, id, map[string]any{"update_id": 1}, nil)
		assert.Equal(t, tc.code, w.Code, tc.err.Error())
	}

	env.exec.mu.Lock()
	env.exec.err = nil
	env.exec.mu.Unlock()
	env.redis.Set("trigger:"+id+":ratelimit", "100")
	w := env.rc1Post(t, id, map[string]any{"update_id": 1}, nil)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRC1TelegramWebhook_UpdateConversion(t *testing.T) {
	t.Parallel()
	env := rc1NewTelegramEnv(t)
	id := env.rc1Register(t, true, map[string]any{"input": map[string]any{"bot": "support", "text": "default"}})

	full := map[string]any{
		"message_id": 7, "date": 1700000000,
		"from": map[string]any{"id": 42, "is_bot": false, "first_name": "Ann", "last_name": "Lee", "username": "ann"},
		"chat": map[string]any{"id": -100, "type": "group", "title": "Team"},
		"text": "/start", "caption": "cap",
		"photo":    []map[string]any{{"file_id": "p1", "file_unique_id": "u1", "width": 10, "height": 20, "file_size": 30}},
		"document": map[string]any{"file_id": "d1", "file_unique_id": "du", "file_name": "a.pdf", "mime_type": "application/pdf", "file_size": 1},
		"audio":    map[string]any{"file_id": "a1", "file_unique_id": "au", "duration": 3, "performer": "p", "title": "t", "mime_type": "audio/mpeg", "file_size": 2},
		"video":    map[string]any{"file_id": "v1", "file_unique_id": "vu", "width": 1, "height": 2, "duration": 3, "mime_type": "video/mp4", "file_size": 4},
		"voice":    map[string]any{"file_id": "o1", "file_unique_id": "ou", "duration": 5, "mime_type": "audio/ogg", "file_size": 6},
		"location": map[string]any{"longitude": 1.5, "latitude": 2.5},
		"contact":  map[string]any{"phone_number": "+1", "first_name": "C", "last_name": "D", "user_id": 9},
	}
	w := env.rc1Post(t, id, map[string]any{"update_id": 100, "message": full}, map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	resp := rc1JSON(t, w)
	assert.Equal(t, true, resp["ok"])
	assert.Equal(t, "exec-wf-tg", resp["execution_id"])

	in := env.exec.last()
	assert.Equal(t, "support", in["bot"], "default trigger input is merged")
	assert.Equal(t, "message", in["update_type"])
	assert.Equal(t, "/start", in["text"], "message text overrides default input")
	assert.EqualValues(t, -100, in["chat_id"])
	assert.EqualValues(t, 42, in["user_id"])
	assert.Equal(t, "ann", in["username"])
	assert.Equal(t, "Ann", in["first_name"])
	msg := in["message"].(map[string]any)
	for _, k := range []string{"from", "text", "caption", "photo", "document", "audio", "video", "voice", "location", "contact"} {
		assert.Contains(t, msg, k)
	}
	meta := in["_webhook"].(map[string]any)
	assert.Equal(t, "1.2.3.4", meta["source_ip"])
	assert.Contains(t, meta["headers"], "Content-Type")

	minimal := map[string]any{"message_id": 1, "chat": map[string]any{"id": 5, "type": "channel"}}
	for _, kind := range []string{"edited_message", "channel_post", "edited_channel_post"} {
		w = env.rc1Post(t, id, map[string]any{"update_id": 1, kind: minimal}, nil)
		require.Equal(t, http.StatusOK, w.Code)
		in = env.exec.last()
		assert.Equal(t, kind, in["update_type"])
		m := in[kind].(map[string]any)
		assert.NotContains(t, m, "from")
		assert.NotContains(t, m, "text")
		assert.NotContains(t, in, "chat_id", "convenience fields only for plain messages")
	}

	w = env.rc1Post(t, id, map[string]any{"update_id": 2, "callback_query": map[string]any{
		"id": "cb", "data": "btn", "inline_message_id": "im",
		"from":    map[string]any{"id": 1, "first_name": "F"},
		"message": minimal,
	}}, nil)
	require.Equal(t, http.StatusOK, w.Code)
	cb := env.exec.last()["callback_query"].(map[string]any)
	assert.Equal(t, "btn", cb["data"])
	assert.Equal(t, "im", cb["inline_message_id"])
	assert.Contains(t, cb, "message")

	w = env.rc1Post(t, id, map[string]any{"update_id": 3, "callback_query": map[string]any{"id": "cb2", "from": map[string]any{"id": 1}}}, nil)
	require.Equal(t, http.StatusOK, w.Code)
	cb = env.exec.last()["callback_query"].(map[string]any)
	assert.NotContains(t, cb, "message")
	assert.NotContains(t, cb, "inline_message_id")

	w = env.rc1Post(t, id, map[string]any{"update_id": 4, "inline_query": map[string]any{
		"id": "iq", "query": "cats", "offset": "0", "chat_type": "private",
		"from": map[string]any{"id": 3}, "location": map[string]any{"longitude": 1, "latitude": 2},
	}}, nil)
	require.Equal(t, http.StatusOK, w.Code)
	iq := env.exec.last()["inline_query"].(map[string]any)
	assert.Equal(t, "cats", iq["query"])
	assert.Equal(t, "private", iq["chat_type"])
	assert.Contains(t, iq, "location")

	w = env.rc1Post(t, id, map[string]any{"update_id": 5, "inline_query": map[string]any{"id": "iq2", "from": map[string]any{"id": 3}}}, nil)
	require.Equal(t, http.StatusOK, w.Code)
	iq = env.exec.last()["inline_query"].(map[string]any)
	assert.NotContains(t, iq, "chat_type")
	assert.NotContains(t, iq, "location")

	// An update of an unknown kind still reaches the workflow.
	raw, err := json.Marshal(map[string]any{"update_id": 6, "poll": map[string]any{}})
	require.NoError(t, err)
	w = env.rc1Post(t, id, bytes.NewBuffer(raw).Bytes(), nil)
	require.Equal(t, http.StatusOK, w.Code)
	in = env.exec.last()
	assert.Equal(t, "", in["update_type"])
	assert.EqualValues(t, 6, in["update_id"])
}
