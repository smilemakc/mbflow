package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/engine"
	"github.com/smilemakc/mbflow/go/internal/application/observer"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

const rcCredSecret = "sk-ant-integration-secret-42"

// rcKeyProbe records the api_key its nodes receive and echoes it.
type rcKeyProbe struct {
	mu   sync.Mutex
	keys []string
}

func (p *rcKeyProbe) Execute(_ context.Context, config map[string]any, _ any) (any, error) {
	key, _ := config["api_key"].(string)
	p.mu.Lock()
	p.keys = append(p.keys, key)
	p.mu.Unlock()
	return map[string]any{"echo": key, "header": "Bearer " + key}, nil
}

func (p *rcKeyProbe) Validate(map[string]any) error { return nil }

func (p *rcKeyProbe) seen() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.keys...)
}

// TestRC_CredentialsInNodes runs workflows that use an encrypted credential
// against the real storage: the executor gets the decrypted key, while the
// stored execution, node executions, events and API responses never contain it.
func TestRC_CredentialsInNodes(t *testing.T) {
	env := rc2NewEnv(t)
	ctx := context.Background()

	probe := &rcKeyProbe{}
	require.NoError(t, env.executors.Register("key_probe", probe))

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	enc, err := crypto.NewEncryptionService(key)
	require.NoError(t, err)
	credRepo := storage.NewCredentialsRepository(env.db)
	eventRepo := storage.NewEventRepository(env.db)

	om := observer.NewObserverManager()
	require.NoError(t, om.Register(observer.NewDatabaseObserver(eventRepo)))
	em := engine.NewExecutionManager(env.executors, env.workflowRepo, env.execRepo, eventRepo, env.resourceRepo, om)
	em.SetCredentialDecryption(enc, credRepo)
	t.Cleanup(func() {
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = em.Shutdown(sctx)
	})
	env.ops.ExecutionMgr = em

	owner := env.user(t, false)
	cred := models.NewCredentialsResource(owner, "anthropic-key", models.CredentialTypeAPIKey)
	cred.Provider = "anthropic"
	encrypted, err := enc.EncryptMap(map[string]string{"api_key": rcCredSecret})
	require.NoError(t, err)
	cred.EncryptedData = encrypted
	require.NoError(t, credRepo.CreateCredentials(ctx, cred))
	require.NotEmpty(t, cred.ID)

	r := env.router(owner, false, true)

	assertClean := func(t *testing.T, what string, v any) {
		t.Helper()
		data, err := json.Marshal(v)
		require.NoError(t, err)
		assert.NotContains(t, string(data), rcCredSecret, "%s leaks the secret", what)
	}
	assertStoredClean := func(t *testing.T, execID string) {
		t.Helper()
		id := uuid.MustParse(execID)
		row, err := env.execRepo.FindByIDWithRelations(ctx, id)
		require.NoError(t, err)
		assertClean(t, "stored execution", row)
		nodes, err := env.execRepo.FindNodeExecutionsByExecutionID(ctx, id)
		require.NoError(t, err)
		require.NotEmpty(t, nodes)
		for _, n := range nodes {
			assertClean(t, "stored node execution", n)
		}
		require.Eventually(t, func() bool {
			events, err := eventRepo.FindByExecutionID(ctx, id)
			return err == nil && len(events) >= 4
		}, 5*time.Second, 20*time.Millisecond, "events are stored")
		events, err := eventRepo.FindByExecutionID(ctx, id)
		require.NoError(t, err)
		assertClean(t, "stored events", events)
	}

	t.Run("stored workflow with attached credential", func(t *testing.T) {
		body := map[string]any{
			"name": "llm-with-credential",
			"nodes": []map[string]any{
				{"id": "call", "name": "Call", "type": "key_probe", "config": map[string]any{"api_key": "{{resource.anthropic.api_key}}"}},
				{"id": "next", "name": "Next", "type": "key_probe", "config": map[string]any{"api_key": "{{input.echo}}"}},
			},
			"edges": []map[string]any{{"id": "e1", "from": "call", "to": "next"}},
		}
		wf := env.createWorkflow(t, r, body)
		wfID := wf["id"].(string)

		w := rc2Do(t, r, http.MethodPost, "/api/v1/workflows/"+wfID+"/resources",
			map[string]any{"resource_id": cred.ID, "alias": "anthropic"})
		require.Less(t, w.Code, 300, w.Body.String())

		w = rc2Do(t, r, http.MethodPost, "/api/v1/workflows/"+wfID+"/execute", map[string]any{"input": map[string]any{}})
		require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
		execID := rc2DecodeMap(t, w)["id"].(string)

		require.Eventually(t, func() bool {
			w = rc2Do(t, r, http.MethodGet, "/api/v1/executions/"+execID, nil)
			return w.Code == http.StatusOK && rc2DecodeMap(t, w)["status"] == "completed"
		}, 10*time.Second, 20*time.Millisecond, "execution completes")
		assert.NotContains(t, w.Body.String(), rcCredSecret, "API response")
		assert.Equal(t, []string{rcCredSecret, rcCredSecret}, probe.seen(), "both nodes got the real key")

		assertStoredClean(t, execID)

		w = rc2Do(t, r, http.MethodGet, "/api/v1/executions/"+execID+"/nodes/call/result", nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.NotContains(t, w.Body.String(), rcCredSecret)
		assert.Contains(t, w.Body.String(), `"***"`)

		stored, err := credRepo.GetCredentials(ctx, cred.ID)
		require.NoError(t, err)
		assert.EqualValues(t, 1, stored.UsageCount, "usage is recorded")
		assert.NotNil(t, stored.LastUsedAt)
	})

	t.Run("inline execution with credential_ids", func(t *testing.T) {
		before := len(probe.seen())
		w := rc2Do(t, r, http.MethodPost, "/api/v1/executions/ephemeral", map[string]any{
			"mode":              "sync",
			"persist_execution": true,
			"credential_ids":    []string{cred.ID},
			"workflow": map[string]any{
				"name":      "inline",
				"resources": []map[string]any{{"resource_id": cred.ID, "alias": "anthropic"}},
				"nodes": []map[string]any{
					{"id": "call", "name": "Call", "type": "key_probe", "config": map[string]any{"api_key": "{{resource.anthropic.api_key}}"}},
				},
			},
		})
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.NotContains(t, w.Body.String(), rcCredSecret, "API response")
		resp := rc2DecodeMap(t, w)
		assert.Equal(t, "completed", resp["status"])
		assert.Equal(t, rcCredSecret, probe.seen()[before])
		assertStoredClean(t, resp["id"].(string))
	})

	t.Run("inline execution cannot use another user's credential", func(t *testing.T) {
		other := env.router(env.user(t, false), false, false)
		w := rc2Do(t, other, http.MethodPost, "/api/v1/executions/ephemeral", map[string]any{
			"mode":           "sync",
			"credential_ids": []string{cred.ID},
			"workflow": map[string]any{
				"name": "inline",
				"nodes": []map[string]any{
					{"id": "call", "name": "Call", "type": "key_probe", "config": map[string]any{"api_key": "{{resource." + cred.ID + ".api_key}}"}},
				},
			},
		})
		assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		assert.Contains(t, w.Body.String(), "RESOURCE_UNAVAILABLE")
		assert.NotContains(t, w.Body.String(), rcCredSecret)
	})

	t.Run("invalid credential id", func(t *testing.T) {
		w := rc2Do(t, r, http.MethodPost, "/api/v1/executions/ephemeral", map[string]any{
			"mode": "sync", "credential_ids": []string{"nope"},
			"workflow": map[string]any{"name": "inline", "nodes": []map[string]any{{"id": "a", "name": "A", "type": "key_probe"}}},
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "INVALID_CREDENTIAL_ID")
	})
}

var _ executor.Executor = (*rcKeyProbe)(nil)
