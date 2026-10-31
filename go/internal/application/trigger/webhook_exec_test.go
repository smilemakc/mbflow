package trigger

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func newTestWebhookRegistry(t *testing.T, rc *cache.RedisCache, exec WorkflowExecutor) (*WebhookRegistry, *mockTriggerRepo) {
	t.Helper()
	repo := &mockTriggerRepo{}
	repo.On("MarkTriggered", mock.Anything, mock.Anything).Return(nil).Maybe()
	return NewWebhookRegistry(WebhookRegistryConfig{
		TriggerRepo:  repo,
		WorkflowRepo: &mockWorkflowRepo{},
		ExecutionMgr: exec,
		Cache:        rc,
	}), repo
}

func webhookTrigger(config map[string]any) *models.Trigger {
	return &models.Trigger{
		ID: uuid.NewString(), WorkflowID: "wf-hook", Type: models.TriggerTypeWebhook,
		Enabled: true, Config: config,
	}
}

func TestWebhookRegistry_ExecuteWebhook_Success(t *testing.T) {
	rc := setupTestRedis(t)
	exec := &fakeExecutor{}
	wr, repo := newTestWebhookRegistry(t, rc, exec)
	ctx := context.Background()

	trig := webhookTrigger(map[string]any{
		"secret":       "s3cret",
		"ip_whitelist": []any{"10.0.0.0/8"},
		"input":        map[string]any{"env": "prod", "x": "default"},
	})
	require.NoError(t, wr.RegisterWebhook(ctx, trig))

	payload := map[string]any{"x": "payload"}
	headers := map[string]string{"X-Webhook-Signature": wr.computeSignature("s3cret", payload)}

	execID, err := wr.ExecuteWebhook(ctx, trig.ID, payload, headers, "10.1.2.3")
	require.NoError(t, err)
	assert.Equal(t, "exec-wf-hook", execID)

	call := exec.Calls()[0]
	assert.Equal(t, trig.ID, call.TriggerID)
	assert.Equal(t, "prod", call.Input["env"])
	assert.Equal(t, "payload", call.Input["x"], "payload overrides default input")
	meta, ok := call.Input["_webhook"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, trig.ID, meta["trigger_id"])
	assert.Equal(t, "10.1.2.3", meta["source_ip"])

	state, err := LoadTriggerState(ctx, rc, trig.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), state.ExecutionCount)
	repo.AssertNumberOfCalls(t, "MarkTriggered", 1)
}

func TestWebhookRegistry_ExecuteWebhook_Rejections(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	exec := &fakeExecutor{}
	wr, _ := newTestWebhookRegistry(t, rc, exec)
	ctx := context.Background()

	_, err := wr.ExecuteWebhook(ctx, "missing", nil, nil, "")
	assert.ErrorContains(t, err, "webhook trigger not found")

	disabled := webhookTrigger(map[string]any{})
	disabled.Enabled = false
	require.NoError(t, wr.RegisterWebhook(ctx, disabled))
	_, err = wr.ExecuteWebhook(ctx, disabled.ID, nil, nil, "")
	assert.ErrorContains(t, err, "disabled")

	signed := webhookTrigger(map[string]any{"secret": "s"})
	require.NoError(t, wr.RegisterWebhook(ctx, signed))
	_, err = wr.ExecuteWebhook(ctx, signed.ID, map[string]any{}, map[string]string{"X-Webhook-Signature": "bad"}, "")
	assert.ErrorContains(t, err, "signature validation failed")

	whitelisted := webhookTrigger(map[string]any{"ip_whitelist": []any{"192.168.1.1"}})
	require.NoError(t, wr.RegisterWebhook(ctx, whitelisted))
	_, err = wr.ExecuteWebhook(ctx, whitelisted.ID, nil, nil, "10.0.0.1")
	assert.ErrorContains(t, err, "IP not whitelisted")

	limited := webhookTrigger(map[string]any{})
	require.NoError(t, wr.RegisterWebhook(ctx, limited))
	require.NoError(t, mr.Set("trigger:"+limited.ID+":ratelimit", "100"))
	_, err = wr.ExecuteWebhook(ctx, limited.ID, nil, nil, "")
	assert.ErrorContains(t, err, "rate limit exceeded")

	failing := webhookTrigger(map[string]any{})
	require.NoError(t, wr.RegisterWebhook(ctx, failing))
	exec.err = errExec
	_, err = wr.ExecuteWebhook(ctx, failing.ID, nil, nil, "")
	assert.ErrorIs(t, err, errExec)
}

func TestWebhookRegistry_CheckRateLimit(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	wr, _ := newTestWebhookRegistry(t, rc, &fakeExecutor{})
	ctx := context.Background()

	require.NoError(t, wr.checkRateLimit(ctx, "t"))
	assert.Greater(t, mr.TTL("trigger:t:ratelimit"), 0*mr.TTL("none"), "first hit starts the window")
	for i := 0; i < 99; i++ {
		require.NoError(t, wr.checkRateLimit(ctx, "t"))
	}
	assert.Error(t, wr.checkRateLimit(ctx, "t"), "101st request in the window is rejected")

	// Redis errors fail open.
	mr.SetError("ERR redis unavailable")
	assert.NoError(t, wr.checkRateLimit(ctx, "t"))
}

func TestWebhookRegistry_ExecuteWebhook_BookkeepingFailuresAreLogged(t *testing.T) {
	rc, mr := setupTestRedisServer(t)
	exec := &fakeExecutor{}
	wr, repo := newTestWebhookRegistry(t, rc, exec)
	ctx := context.Background()
	trig := webhookTrigger(map[string]any{})
	require.NoError(t, wr.RegisterWebhook(ctx, trig))

	repo.ExpectedCalls = nil
	repo.On("MarkTriggered", mock.Anything, mock.Anything).Return(errExec).Once()
	mr.SetError("ERR redis unavailable") // rate limit fails open, state save fails

	execID, err := wr.ExecuteWebhook(ctx, trig.ID, nil, nil, "")
	require.NoError(t, err)
	assert.NotEmpty(t, execID)
	repo.AssertExpectations(t)
}
