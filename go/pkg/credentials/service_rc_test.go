package credentials

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

var errRCRepo = errors.New("repo failure")

type rcAccessLog struct {
	resourceID, action, actorID, actorType string
	metadata                               map[string]any
}

// rcCredRepo is an in-memory repository.CredentialsRepository.
type rcCredRepo struct {
	mu        sync.Mutex
	creds     map[string]*models.CredentialsResource
	usage     map[string]int
	usageCh   chan string
	accessLog []rcAccessLog
	logErr    error
}

func rcNewRepo() *rcCredRepo {
	return &rcCredRepo{
		creds:   map[string]*models.CredentialsResource{},
		usage:   map[string]int{},
		usageCh: make(chan string, 16),
	}
}

func (r *rcCredRepo) CreateCredentials(ctx context.Context, cred *models.CredentialsResource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creds[cred.ID] = cred
	return nil
}

func (r *rcCredRepo) GetCredentials(ctx context.Context, id string) (*models.CredentialsResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.creds[id]
	if !ok {
		return nil, errRCRepo
	}
	cp := *c
	return &cp, nil
}

func (r *rcCredRepo) GetCredentialsByOwner(ctx context.Context, ownerID string) ([]*models.CredentialsResource, error) {
	return nil, nil
}

func (r *rcCredRepo) GetCredentialsByProvider(ctx context.Context, ownerID, provider string) ([]*models.CredentialsResource, error) {
	return nil, nil
}

func (r *rcCredRepo) UpdateCredentials(ctx context.Context, cred *models.CredentialsResource) error {
	return nil
}

func (r *rcCredRepo) UpdateEncryptedData(ctx context.Context, id string, data map[string]string) error {
	return nil
}

func (r *rcCredRepo) DeleteCredentials(ctx context.Context, id string) error { return nil }

func (r *rcCredRepo) IncrementUsageCount(ctx context.Context, id string) error {
	r.mu.Lock()
	r.usage[id]++
	r.mu.Unlock()
	r.usageCh <- id
	return nil
}

func (r *rcCredRepo) LogCredentialAccess(ctx context.Context, id, action, actorID, actorType string, metadata map[string]any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.logErr != nil {
		return r.logErr
	}
	r.accessLog = append(r.accessLog, rcAccessLog{id, action, actorID, actorType, metadata})
	return nil
}

type rcEnv struct {
	svc  *Service
	repo *rcCredRepo
	enc  *crypto.EncryptionService
}

func rcSetup(t *testing.T) *rcEnv {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	enc, err := crypto.NewEncryptionService(key)
	require.NoError(t, err)
	repo := rcNewRepo()
	return &rcEnv{svc: NewService(repo, enc), repo: repo, enc: enc}
}

func (e *rcEnv) add(t *testing.T, id string, typ models.CredentialType, plain map[string]string) *models.CredentialsResource {
	t.Helper()
	encrypted, err := e.enc.EncryptMap(plain)
	require.NoError(t, err)
	c := models.NewCredentialsResource("owner", id, typ)
	c.ID = id
	c.EncryptedData = encrypted
	require.NoError(t, e.repo.CreateCredentials(context.Background(), c))
	return c
}

// rcAwaitUsage waits for the asynchronous usage increment of id.
func (e *rcEnv) rcAwaitUsage(t *testing.T, id string) {
	t.Helper()
	select {
	case got := <-e.repo.usageCh:
		assert.Equal(t, id, got)
	case <-time.After(5 * time.Second):
		t.Fatalf("usage counter for %s was not incremented", id)
	}
}

func TestRCGetDecrypted(t *testing.T) {
	ctx := context.Background()
	e := rcSetup(t)
	e.add(t, "c1", models.CredentialTypeCustom, map[string]string{"a": "1", "b": "2"})

	cred, err := e.svc.GetDecrypted(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, cred.DecryptedData)
	assert.NotEqual(t, "1", cred.EncryptedData["a"], "stored data stays encrypted")
	e.rcAwaitUsage(t, "c1")

	_, err = e.svc.GetDecrypted(ctx, "missing")
	assert.ErrorIs(t, err, errRCRepo)

	expired := e.add(t, "old", models.CredentialTypeCustom, map[string]string{"a": "1"})
	past := time.Now().Add(-time.Minute)
	expired.ExpiresAt = &past
	_, err = e.svc.GetDecrypted(ctx, "old")
	assert.ErrorContains(t, err, "has expired")

	broken := models.NewCredentialsResource("owner", "broken", models.CredentialTypeCustom)
	broken.ID = "broken"
	broken.EncryptedData = map[string]string{"a": "not-encrypted"}
	require.NoError(t, e.repo.CreateCredentials(ctx, broken))
	_, err = e.svc.GetDecrypted(ctx, "broken")
	assert.ErrorContains(t, err, "failed to decrypt credential")

	all, err := e.svc.GetAllDecryptedValues(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, "2", all["b"])
	e.rcAwaitUsage(t, "c1")
	_, err = e.svc.GetAllDecryptedValues(ctx, "missing")
	assert.Error(t, err)
}

func TestRCTypedAccessors(t *testing.T) {
	ctx := context.Background()
	e := rcSetup(t)
	e.add(t, "api", models.CredentialTypeAPIKey, map[string]string{"api_key": "sk-123"})
	e.add(t, "basic", models.CredentialTypeBasicAuth, map[string]string{"username": "u", "password": "p"})
	e.add(t, "oauth", models.CredentialTypeOAuth2, map[string]string{"client_id": "cid", "client_secret": "cs", "token_url": "https://t"})
	e.add(t, "sa", models.CredentialTypeServiceAccount, map[string]string{"json_key": `{"type":"service_account"}`})

	key, err := e.svc.GetAPIKey(ctx, "api")
	require.NoError(t, err)
	assert.Equal(t, "sk-123", key)

	u, p, err := e.svc.GetBasicAuth(ctx, "basic")
	require.NoError(t, err)
	assert.Equal(t, "u", u)
	assert.Equal(t, "p", p)

	o, err := e.svc.GetOAuth2(ctx, "oauth")
	require.NoError(t, err)
	assert.Equal(t, "cid", o.ClientID)
	assert.Equal(t, "cs", o.ClientSecret)
	assert.Equal(t, "https://t", o.TokenURL)

	js, err := e.svc.GetServiceAccountJSON(ctx, "sa")
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"service_account"}`, js)

	// Type mismatches
	_, err = e.svc.GetAPIKey(ctx, "basic")
	assert.ErrorContains(t, err, "is not an API key")
	_, _, err = e.svc.GetBasicAuth(ctx, "api")
	assert.ErrorContains(t, err, "is not basic auth")
	_, err = e.svc.GetOAuth2(ctx, "api")
	assert.ErrorContains(t, err, "is not OAuth2")
	_, err = e.svc.GetServiceAccountJSON(ctx, "api")
	assert.ErrorContains(t, err, "is not a service account")

	// Missing credentials
	_, err = e.svc.GetAPIKey(ctx, "missing")
	assert.Error(t, err)
	_, _, err = e.svc.GetBasicAuth(ctx, "missing")
	assert.Error(t, err)
	_, err = e.svc.GetOAuth2(ctx, "missing")
	assert.Error(t, err)
	_, err = e.svc.GetServiceAccountJSON(ctx, "missing")
	assert.Error(t, err)

	// Custom values
	v, err := e.svc.GetCustomValue(ctx, "basic", "username")
	require.NoError(t, err)
	assert.Equal(t, "u", v)
	_, err = e.svc.GetCustomValue(ctx, "basic", "nope")
	assert.ErrorContains(t, err, `field "nope" not found`)
	_, err = e.svc.GetCustomValue(ctx, "missing", "x")
	assert.Error(t, err)

	// Only the 5 successful accesses above increment usage (asynchronously);
	// type mismatches and missing fields are rejected and not counted.
	for i := 0; i < 5; i++ {
		select {
		case <-e.repo.usageCh:
		case <-time.After(5 * time.Second):
			t.Fatal("missing usage increment")
		}
	}
	select {
	case id := <-e.repo.usageCh:
		t.Fatalf("unexpected usage increment for %s", id)
	case <-time.After(50 * time.Millisecond):
	}
	e.repo.mu.Lock()
	defer e.repo.mu.Unlock()
	assert.Equal(t, 1, e.repo.usage["api"])
	assert.Equal(t, 2, e.repo.usage["basic"])
	assert.Equal(t, 1, e.repo.usage["oauth"])
	assert.Equal(t, 1, e.repo.usage["sa"])
	assert.Zero(t, e.repo.usage["missing"])
}

func TestRCLogWorkflowUsage(t *testing.T) {
	ctx := context.Background()
	e := rcSetup(t)
	require.NoError(t, e.svc.LogWorkflowUsage(ctx, "c1", "wf-1", "ex-1"))
	require.Len(t, e.repo.accessLog, 1)
	entry := e.repo.accessLog[0]
	assert.Equal(t, "c1", entry.resourceID)
	assert.Equal(t, "used_in_workflow", entry.action)
	assert.Equal(t, "workflow", entry.actorType)
	assert.Equal(t, "wf-1", entry.metadata["workflow_id"])
	assert.Equal(t, "ex-1", entry.metadata["execution_id"])

	e.repo.logErr = errRCRepo
	assert.ErrorIs(t, e.svc.LogWorkflowUsage(ctx, "c1", "wf", "ex"), errRCRepo)
}
