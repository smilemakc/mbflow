package engine

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/observer"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	pkgengine "github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

const credSecret = "sk-ant-secret-value-0001"

func testEncryption(t *testing.T) *crypto.EncryptionService {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	enc, err := crypto.NewEncryptionService(key)
	require.NoError(t, err)
	return enc
}

func encryptedCredential(t *testing.T, enc *crypto.EncryptionService, owner string, credType models.CredentialType, values map[string]string) *models.CredentialsResource {
	t.Helper()
	cred := models.NewCredentialsResource(owner, "cred", credType)
	cred.ID = uuid.NewString()
	cred.Provider = "anthropic"
	encrypted, err := enc.EncryptMap(values)
	require.NoError(t, err)
	cred.EncryptedData = encrypted
	return cred
}

// fakeCredUsage records IncrementUsageCount calls.
type fakeCredUsage struct {
	repository.CredentialsRepository
	mu    sync.Mutex
	calls []string
	err   error
}

func (f *fakeCredUsage) IncrementUsageCount(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, id)
	return f.err
}

// probeExecutor records the config each node receives; a node configured
// with fail=true returns an error that quotes its api_key.
type probeExecutor struct {
	mu      sync.Mutex
	configs map[string]map[string]any
	inputs  map[string]any
}

func newProbeExecutor() *probeExecutor {
	return &probeExecutor{configs: map[string]map[string]any{}, inputs: map[string]any{}}
}

func (p *probeExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	execCtx, _ := executor.GetExecutionContext(ctx)
	p.mu.Lock()
	p.configs[execCtx.NodeID] = config
	p.inputs[execCtx.NodeID] = input
	p.mu.Unlock()
	key, _ := config["api_key"].(string)
	if config["fail"] == true {
		return nil, errors.New("provider rejected key " + key)
	}
	return map[string]any{"echo": key}, nil
}

func (p *probeExecutor) Validate(map[string]any) error { return nil }

func (p *probeExecutor) config(nodeID string) map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.configs[nodeID]
}

// credFixture is a stored workflow n1 -> n2 whose nodes use the credential
// attached as alias "anthropic".
func credFixture(t *testing.T, probe *probeExecutor, failSecond bool) (*storedFixture, *models.CredentialsResource, *fakeCredUsage) {
	t.Helper()
	f := newStoredFixture(t, probe, newFakeExecutionRepo())
	enc := testEncryption(t)
	owner := uuid.New()
	cred := encryptedCredential(t, enc, owner.String(), models.CredentialTypeAPIKey, map[string]string{"api_key": credSecret})
	f.resources.resources[cred.ID] = cred

	f.wf.CreatedBy = &owner
	f.wf.Resources = []*storagemodels.WorkflowResourceModel{{ResourceID: uuid.MustParse(cred.ID), Alias: "anthropic", AccessType: "read"}}
	f.wf.Nodes[0].Config = storagemodels.JSONBMap{"api_key": "{{resource.anthropic.api_key}}", "provider": "{{resource.anthropic.provider}}"}
	f.wf.Nodes[1].Config = storagemodels.JSONBMap{"api_key": "{{input.echo}}", "fail": failSecond}

	usage := &fakeCredUsage{}
	f.em.SetCredentialDecryption(enc, usage)
	return f, cred, usage
}

func assertNoSecret(t *testing.T, v any, what string) {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	assert.NotContains(t, string(data), credSecret, "%s leaks the secret", what)
}

func TestCredentials_StoredExecution_ResolvesAndRedacts(t *testing.T) {
	probe := newProbeExecutor()
	f, cred, usage := credFixture(t, probe, false)

	exec, err := f.em.Execute(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusCompleted, exec.Status)

	// Executors get the real values, also downstream.
	assert.Equal(t, credSecret, probe.config("n1")["api_key"])
	assert.Equal(t, "anthropic", probe.config("n1")["provider"])
	assert.Equal(t, credSecret, probe.config("n2")["api_key"])

	// Persisted / returned data is redacted.
	assert.Equal(t, map[string]any{"echo": "***"}, exec.Output)
	require.Len(t, exec.NodeExecutions, 2)
	assert.Equal(t, "***", exec.NodeExecutions[0].ResolvedConfig["api_key"])
	assert.Equal(t, "{{resource.anthropic.api_key}}", exec.NodeExecutions[0].Config["api_key"])
	assert.Equal(t, "***", exec.NodeExecutions[1].Input["echo"])
	assertNoSecret(t, exec, "returned execution")

	row, ok := f.execRepo.get(exec.ID)
	require.True(t, ok)
	assertNoSecret(t, row, "stored execution")

	eventually(t, func() bool {
		return contains(f.recorder.types(exec.ID), observer.EventTypeExecutionCompleted)
	}, "completion event")
	for _, e := range f.recorder.snapshot() {
		assertNoSecret(t, map[string]any{"output": e.Output, "input": e.Input, "vars": e.Variables}, string(e.Type))
	}

	usage.mu.Lock()
	assert.Equal(t, []string{cred.ID}, usage.calls, "usage is recorded once per execution")
	usage.mu.Unlock()
}

func TestCredentials_StoredExecution_RedactsErrors(t *testing.T) {
	probe := newProbeExecutor()
	f, _, usage := credFixture(t, probe, true)
	usage.err = errors.New("db down") // must not fail the execution

	exec, err := f.em.Execute(context.Background(), f.wf.ID.String(), nil, nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), credSecret)
	assert.Contains(t, err.Error(), "provider rejected key ***")
	assert.Equal(t, models.ExecutionStatusFailed, exec.Status)
	assert.NotContains(t, exec.Error, credSecret)
	assert.Contains(t, exec.NodeExecutions[1].Error, "provider rejected key ***")
	assertNoSecret(t, exec, "failed execution")

	eventually(t, func() bool {
		return contains(f.recorder.types(exec.ID), observer.EventTypeExecutionFailed)
	}, "failure event")
	for _, e := range f.recorder.snapshot() {
		if e.Error != nil {
			assert.NotContains(t, e.Error.Error(), credSecret, "event %s", e.Type)
		}
	}
}

func TestCredentials_NoEncryptionService(t *testing.T) {
	probe := newProbeExecutor()
	f, _, _ := credFixture(t, probe, false)
	f.em.SetCredentialDecryption(nil, nil)

	exec, err := f.em.Execute(context.Background(), f.wf.ID.String(), nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrResourceUnavailable)
	assert.Contains(t, err.Error(), crypto.KeyEnvVar, "the error says how to fix it")
	assert.Equal(t, models.ExecutionStatusFailed, exec.Status)
	assert.Nil(t, probe.config("n1"), "no node ran")

	// A credential that no node references does not block the execution.
	f.wf.Nodes[0].Config = storagemodels.JSONBMap{"api_key": "static"}
	exec, err = f.em.Execute(context.Background(), f.wf.ID.String(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusCompleted, exec.Status)
}

func TestLoadResources_Credentials(t *testing.T) {
	enc := testEncryption(t)
	owner := uuid.NewString()
	ctx := context.Background()

	oauth := encryptedCredential(t, enc, owner, models.CredentialTypeOAuth2, map[string]string{
		"client_id": "client-id-1", "client_secret": "client-secret-1",
		"token_url": "https://auth.example.com/token", "scopes": "read",
		"name": "shadowed-name",
	})
	basic := encryptedCredential(t, enc, owner, models.CredentialTypeBasicAuth, map[string]string{"username": "alice", "password": "hunter22"})
	repo := &fakeResourceRepo{resources: map[string]models.Resource{oauth.ID: oauth, basic.ID: basic}}
	usage := &fakeCredUsage{}
	em := &ExecutionManager{resourceRepo: repo}
	em.SetCredentialDecryption(enc, usage)

	secrets := pkgengine.NewSecretRedactor()
	wf := &models.Workflow{CreatedBy: owner}
	res, err := em.loadResources(ctx, []models.WorkflowResource{
		{ResourceID: oauth.ID, Alias: "gh"},
		{ResourceID: basic.ID, Alias: "db", AccessType: "write"},
	}, owner, wf, secrets)
	require.NoError(t, err)

	gh := res["gh"].(map[string]any)
	assert.Equal(t, "client-secret-1", gh["client_secret"])
	assert.Equal(t, "cred", gh["name"], "reserved keys are not overridden")
	assert.Equal(t, "shadowed-name", gh["fields"].(map[string]any)["name"], "but stay reachable under fields")
	assert.Equal(t, "oauth2", gh["credential_type"])
	assert.Equal(t, "anthropic", gh["provider"])
	assert.Equal(t, "read", gh["access_type"], "default access type")
	db := res["db"].(map[string]any)
	assert.Equal(t, "hunter22", db["password"])
	assert.Equal(t, "write", db["access_type"])

	assert.Equal(t, "*** *** *** ***", secrets.String("client-id-1 client-secret-1 alice hunter22"))
	assert.Equal(t, "https://auth.example.com/token read", secrets.String("https://auth.example.com/token read"),
		"configuration fields are not treated as secrets")
	assert.Len(t, usage.calls, 2)
}

func TestLoadResources_Errors(t *testing.T) {
	enc := testEncryption(t)
	owner := uuid.NewString()
	ctx := context.Background()
	refsFor := func(id string) []models.WorkflowResource {
		return []models.WorkflowResource{{ResourceID: id, Alias: "k"}}
	}
	referencing := &models.Workflow{Nodes: []*models.Node{{ID: "n", Config: map[string]any{"h": "Bearer {{resource.k.api_key}}"}}}}

	t.Run("no repository", func(t *testing.T) {
		em := &ExecutionManager{}
		res, err := em.loadResources(ctx, nil, owner, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, res)
		_, err = em.loadResources(ctx, refsFor(uuid.NewString()), owner, nil, nil)
		assert.ErrorIs(t, err, ErrResourceUnavailable)
	})

	t.Run("not found / nil resource", func(t *testing.T) {
		em := &ExecutionManager{resourceRepo: &fakeResourceRepo{resources: map[string]models.Resource{"nil": nil}}}
		_, err := em.loadResources(ctx, refsFor(uuid.NewString()), owner, nil, nil)
		assert.ErrorIs(t, err, ErrResourceUnavailable)
		_, err = em.loadResources(ctx, refsFor("nil"), owner, nil, nil)
		assert.ErrorIs(t, err, models.ErrResourceNotFound)
	})

	t.Run("owner mismatch and inactive", func(t *testing.T) {
		cred := encryptedCredential(t, enc, owner, models.CredentialTypeAPIKey, map[string]string{"api_key": credSecret})
		em := &ExecutionManager{resourceRepo: &fakeResourceRepo{resources: map[string]models.Resource{cred.ID: cred}}}
		_, err := em.loadResources(ctx, refsFor(cred.ID), uuid.NewString(), nil, nil)
		require.ErrorIs(t, err, ErrResourceUnavailable)
		assert.Contains(t, err.Error(), "access denied")

		cred.Status = models.ResourceStatusSuspended
		_, err = em.loadResources(ctx, refsFor(cred.ID), owner, nil, nil)
		require.ErrorIs(t, err, ErrResourceUnavailable)
		assert.Contains(t, err.Error(), "not active")
	})

	t.Run("expired or undecryptable", func(t *testing.T) {
		expired := encryptedCredential(t, enc, owner, models.CredentialTypeAPIKey, map[string]string{"api_key": credSecret})
		past := time.Now().Add(-time.Hour)
		expired.ExpiresAt = &past
		garbled := encryptedCredential(t, enc, owner, models.CredentialTypeAPIKey, map[string]string{"api_key": credSecret})
		garbled.EncryptedData["api_key"] = "not-ciphertext"
		em := &ExecutionManager{resourceRepo: &fakeResourceRepo{resources: map[string]models.Resource{expired.ID: expired, garbled.ID: garbled}}}
		em.SetCredentialDecryption(enc, nil)

		_, err := em.loadResources(ctx, refsFor(expired.ID), owner, referencing, nil)
		require.ErrorIs(t, err, ErrResourceUnavailable)
		assert.Contains(t, err.Error(), "expired")

		_, err = em.loadResources(ctx, refsFor(garbled.ID), owner, referencing, nil)
		require.ErrorIs(t, err, ErrResourceUnavailable)
		assert.Contains(t, err.Error(), "decrypt")

		// Unreferenced: metadata only.
		res, err := em.loadResources(ctx, refsFor(garbled.ID), owner, &models.Workflow{}, nil)
		require.NoError(t, err)
		entry := res["k"].(map[string]any)
		assert.NotContains(t, entry, "api_key")
		assert.Equal(t, "api_key", entry["credential_type"])
	})
}

func TestWorkflowReferencesResource(t *testing.T) {
	wf := func(cfg map[string]any) *models.Workflow {
		return &models.Workflow{Nodes: []*models.Node{nil, {ID: "empty"}, {ID: "n", Config: cfg}}}
	}
	assert.False(t, workflowReferencesResource(nil, "a"))
	assert.False(t, workflowReferencesResource(wf(map[string]any{"x": "{{resource.a.k}}"}), ""))
	assert.True(t, workflowReferencesResource(wf(map[string]any{"x": "{{resource.a.k}}"}), "a"))
	assert.True(t, workflowReferencesResource(wf(map[string]any{"x": "{{resource.a}}"}), "a"))
	assert.True(t, workflowReferencesResource(wf(map[string]any{"x": "resource.a"}), "a"))
	assert.False(t, workflowReferencesResource(wf(map[string]any{"x": "{{resource.ab.k}}"}), "a"))
	assert.False(t, workflowReferencesResource(wf(map[string]any{"bad": make(chan int)}), "a"))
}

func TestEphemeralResourceRefs(t *testing.T) {
	id1, id2 := uuid.NewString(), uuid.NewString()
	wf := &models.Workflow{Resources: []models.WorkflowResource{{ResourceID: id1, Alias: "main", AccessType: "read"}}}
	refs := ephemeralResourceRefs(wf, []string{id1, "", id2, id2})
	assert.Equal(t, []models.WorkflowResource{
		{ResourceID: id1, Alias: "main", AccessType: "read"},
		{ResourceID: id2, Alias: id2, AccessType: "read"},
	}, refs)
}

func TestRedactExecution_NoSecrets(t *testing.T) {
	exec := &models.Execution{Error: credSecret}
	redactExecution(exec, nil)
	redactExecution(exec, pkgengine.NewExecutionState("e", "w", &models.Workflow{}, nil, nil))
	assert.Equal(t, credSecret, exec.Error, "nothing to redact without secrets")

	state := pkgengine.NewExecutionState("e", "w", &models.Workflow{}, nil, nil)
	state.Secrets = pkgengine.NewSecretRedactor(credSecret)
	exec.NodeExecutions = []*models.NodeExecution{nil, {Error: credSecret}}
	redactExecution(exec, state)
	assert.Equal(t, "***", exec.Error)
	assert.Equal(t, "***", exec.NodeExecutions[1].Error)
}

func ephemeralCredWorkflow(owner, alias string) *models.Workflow {
	wf := &models.Workflow{
		Name:      "inline",
		CreatedBy: owner,
		Nodes: []*models.Node{
			{ID: "n1", Name: "N1", Type: "probe", Config: map[string]any{"api_key": "{{resource." + alias + ".api_key}}"}},
		},
	}
	return wf
}

func TestCredentials_EphemeralExecution(t *testing.T) {
	enc := testEncryption(t)
	owner := uuid.New()
	cred := encryptedCredential(t, enc, owner.String(), models.CredentialTypeAPIKey, map[string]string{"api_key": credSecret})

	newEM := func(t *testing.T, probe *probeExecutor) (*ExecutionManager, *recordingObserver) {
		registry := executor.NewManager()
		require.NoError(t, registry.Register("probe", probe))
		recorder := &recordingObserver{}
		om := observer.NewObserverManager()
		require.NoError(t, om.Register(recorder))
		em := NewExecutionManager(registry, nil, newFakeExecutionRepo(), nil,
			&fakeResourceRepo{resources: map[string]models.Resource{cred.ID: cred}}, om, NewEphemeralStreamRegistry(time.Minute))
		em.SetCredentialDecryption(enc, &fakeCredUsage{})
		t.Cleanup(func() { _ = em.Shutdown(context.Background()) })
		return em, recorder
	}

	t.Run("credential_ids alias is the credential ID", func(t *testing.T) {
		probe := newProbeExecutor()
		em, recorder := newEM(t, probe)
		exec, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{
			Mode: "sync", Workflow: ephemeralCredWorkflow("", cred.ID),
			CredentialIDs: []string{cred.ID}, CreatedBy: &owner, PersistExecution: true,
		})
		require.NoError(t, err)
		assert.Equal(t, credSecret, probe.config("n1")["api_key"])
		assert.Equal(t, map[string]any{"echo": "***"}, exec.Output)
		assertNoSecret(t, exec, "inline execution")
		eventually(t, func() bool {
			return contains(recorder.types(exec.ID), observer.EventTypeExecutionCompleted)
		}, "completion event")
		for _, e := range recorder.snapshot() {
			assertNoSecret(t, map[string]any{"output": e.Output}, string(e.Type))
		}
	})

	t.Run("workflow resources give the alias (async)", func(t *testing.T) {
		probe := newProbeExecutor()
		em, recorder := newEM(t, probe)
		wf := ephemeralCredWorkflow(owner.String(), "anthropic")
		wf.Nodes[0].Config["fail"] = true
		wf.Resources = []models.WorkflowResource{{ResourceID: cred.ID, Alias: "anthropic", AccessType: "read"}}
		exec, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{
			Mode: "async", Workflow: wf, CredentialIDs: []string{cred.ID},
		})
		require.NoError(t, err)
		eventually(t, func() bool {
			return contains(recorder.types(exec.ID), observer.EventTypeExecutionFailed)
		}, "failure event")
		assert.Equal(t, credSecret, probe.config("n1")["api_key"])
		for _, e := range recorder.snapshot() {
			if e.Error != nil {
				assert.NotContains(t, e.Error.Error(), credSecret, "event %s", e.Type)
			}
		}
	})

	t.Run("rejected before start", func(t *testing.T) {
		em, _ := newEM(t, newProbeExecutor())
		_, err := em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{
			Mode: "sync", Workflow: ephemeralCredWorkflow("", cred.ID), CredentialIDs: []string{cred.ID},
		})
		require.ErrorIs(t, err, ErrResourceUnavailable)
		assert.Contains(t, err.Error(), "require an owner")

		other := uuid.New()
		_, err = em.ExecuteEphemeral(context.Background(), &EphemeralExecutionOptions{
			Mode: "sync", Workflow: ephemeralCredWorkflow("", cred.ID), CredentialIDs: []string{cred.ID}, CreatedBy: &other,
		})
		require.ErrorIs(t, err, ErrResourceUnavailable)
		assert.Contains(t, err.Error(), "access denied")
	})
}
