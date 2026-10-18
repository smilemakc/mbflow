package apiclient

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

const petstoreSpec = `openapi: 3.0.3
info:
  title: Petstore
  description: Pets for everyone
  version: 1.0.0
servers:
  - url: https://pets.test/v1
security:
  - key: []
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200": {description: ok}
  /pets/{id}:
    delete:
      operationId: deletePet
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "204": {description: deleted}
components:
  securitySchemes:
    key: {type: apiKey, in: header, name: X-Key}
`

const petstoreSpecV2 = `openapi: 3.0.3
info: {title: Petstore v2, version: 2.0.0}
servers:
  - url: https://pets.test/v2
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200": {description: ok}
  /pets/{id}:
    patch:
      operationId: updatePet
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200": {description: ok}
`

type fakeRepo struct {
	mu         sync.Mutex
	clients    map[string]*models.APIClient
	specs      map[string]string
	findByIDs  int
	updateCall int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{clients: map[string]*models.APIClient{}, specs: map[string]string{}}
}

func clone(c *models.APIClient) *models.APIClient {
	cp := *c
	cp.SpecRaw = ""
	return &cp
}

func (r *fakeRepo) Create(_ context.Context, c *models.APIClient) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c.ID = uuid.NewString()
	c.CreatedAt = time.Now()
	c.UpdatedAt = c.CreatedAt
	r.clients[c.ID] = clone(c)
	r.specs[c.ID] = c.SpecRaw
	return nil
}

func (r *fakeRepo) Update(_ context.Context, c *models.APIClient) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.clients[c.ID]
	if !ok {
		return models.ErrAPIClientNotFound
	}
	r.updateCall++
	c.UpdatedAt = stored.UpdatedAt.Add(time.Second)
	updated := clone(c)
	if c.SpecRaw == "" {
		updated.Operations = stored.Operations
		updated.Schemas = stored.Schemas
	} else {
		r.specs[c.ID] = c.SpecRaw
	}
	r.clients[c.ID] = updated
	return nil
}

func (r *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.clients[id.String()]; !ok {
		return models.ErrAPIClientNotFound
	}
	delete(r.clients, id.String())
	return nil
}

func (r *fakeRepo) FindByID(_ context.Context, id uuid.UUID) (*models.APIClient, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.findByIDs++
	c, ok := r.clients[id.String()]
	if !ok {
		return nil, models.ErrAPIClientNotFound
	}
	return clone(c), nil
}

func (r *fakeRepo) FindSpec(_ context.Context, id uuid.UUID) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.clients[id.String()]; !ok {
		return "", models.ErrAPIClientNotFound
	}
	return r.specs[id.String()], nil
}

func (r *fakeRepo) FindUpdatedAt(_ context.Context, id uuid.UUID) (time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clients[id.String()]
	if !ok {
		return time.Time{}, models.ErrAPIClientNotFound
	}
	return c.UpdatedAt, nil
}

func (r *fakeRepo) FindByOwner(_ context.Context, ownerID uuid.UUID, limit, offset int) ([]*models.APIClient, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*models.APIClient
	for _, c := range r.clients {
		if c.OwnerID == ownerID.String() {
			result = append(result, clone(c))
		}
	}
	return result, int64(len(result)), nil
}

type fakeCredentials map[string]*models.CredentialsResource

func (f fakeCredentials) GetCredentials(_ context.Context, id string) (*models.CredentialsResource, error) {
	if c, ok := f[id]; ok {
		return c, nil
	}
	return nil, models.ErrResourceNotFound
}

func credential(id, owner string, t models.CredentialType) *models.CredentialsResource {
	return &models.CredentialsResource{
		BaseResource:   models.BaseResource{ID: id, OwnerID: owner, Status: models.ResourceStatusActive},
		CredentialType: t,
	}
}

func TestService_CreateTakesDefaultsFromSpec(t *testing.T) {
	svc := NewService(newFakeRepo(), fakeCredentials{})
	owner := uuid.NewString()

	client, err := svc.Create(context.Background(), owner, CreateInput{Spec: []byte(petstoreSpec)})
	require.NoError(t, err)

	assert.NotEmpty(t, client.ID)
	assert.Equal(t, "Petstore", client.Name)
	assert.Equal(t, "Pets for everyone", client.Description)
	assert.Equal(t, "https://pets.test/v1", client.BaseURL)
	assert.Equal(t, models.APIClientAuth{Type: models.APIClientAuthAPIKey, In: "header", Name: "X-Key", SchemeName: "key"}, client.Auth)
	assert.Equal(t, 2, client.OperationCount)
	assert.Equal(t, len(petstoreSpec), client.SpecSize)
}

func TestService_CreateRejectsInvalidInput(t *testing.T) {
	owner := uuid.NewString()
	creds := fakeCredentials{
		"mine":   credential("mine", owner, models.CredentialTypeAPIKey),
		"theirs": credential("theirs", uuid.NewString(), models.CredentialTypeAPIKey),
		"basic":  credential("basic", owner, models.CredentialTypeBasicAuth),
	}
	svc := NewService(newFakeRepo(), creds)
	ctx := context.Background()

	_, err := svc.Create(ctx, owner, CreateInput{Spec: []byte("not: a spec")})
	assert.ErrorIs(t, err, models.ErrInvalidAPISpec)

	_, err = svc.Create(ctx, owner, CreateInput{Spec: []byte(petstoreSpec), BaseURL: "ftp://x"})
	var vErr *models.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "base_url", vErr.Field)

	for name, auth := range map[string]models.APIClientAuth{
		"foreign credential": {Type: models.APIClientAuthBearer, CredentialID: "theirs"},
		"missing credential": {Type: models.APIClientAuthBearer, CredentialID: "nope"},
		"incompatible type":  {Type: models.APIClientAuthBearer, CredentialID: "basic"},
		"auth without type":  {Type: models.APIClientAuthNone, CredentialID: "mine"},
	} {
		t.Run(name, func(t *testing.T) {
			auth := auth
			_, err := svc.Create(ctx, owner, CreateInput{Spec: []byte(petstoreSpec), Auth: &auth})
			var vErr *models.ValidationError
			require.ErrorAs(t, err, &vErr)
			assert.Equal(t, "auth.credential_id", vErr.Field)
		})
	}

	auth := models.APIClientAuth{Type: models.APIClientAuthBearer, CredentialID: "mine"}
	client, err := svc.Create(ctx, owner, CreateInput{Spec: []byte(petstoreSpec), Auth: &auth, Name: "  My pets "})
	require.NoError(t, err)
	assert.Equal(t, "My pets", client.Name)

	noCreds := NewService(newFakeRepo(), nil)
	_, err = noCreds.Create(ctx, owner, CreateInput{Spec: []byte(petstoreSpec), Auth: &auth})
	require.ErrorAs(t, err, &vErr)
	assert.Contains(t, vErr.Message, "encryption key")
}

func TestService_OwnerIsolation(t *testing.T) {
	svc := NewService(newFakeRepo(), fakeCredentials{})
	ctx := context.Background()
	owner, stranger := uuid.NewString(), uuid.NewString()

	client, err := svc.Create(ctx, owner, CreateInput{Spec: []byte(petstoreSpec)})
	require.NoError(t, err)

	_, err = svc.Get(ctx, stranger, client.ID)
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound)
	_, _, err = svc.GetSpec(ctx, stranger, client.ID)
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound)
	_, err = svc.Update(ctx, stranger, client.ID, UpdateInput{})
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound)
	_, _, err = svc.ReplaceSpec(ctx, stranger, client.ID, []byte(petstoreSpecV2))
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound)
	assert.ErrorIs(t, svc.Delete(ctx, stranger, client.ID), models.ErrAPIClientNotFound)

	list, total, err := svc.List(ctx, stranger, 0, 0)
	require.NoError(t, err)
	assert.Empty(t, list)
	assert.Zero(t, total)

	_, err = svc.Get(ctx, owner, "not-a-uuid")
	assert.ErrorIs(t, err, models.ErrInvalidID)

	_, _, err = svc.GetSpec(ctx, owner, client.ID)
	require.NoError(t, err)
	require.NoError(t, svc.Delete(ctx, owner, client.ID))
	_, err = svc.Get(ctx, owner, client.ID)
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound)
}

func TestService_UpdateSettings(t *testing.T) {
	owner := uuid.NewString()
	svc := NewService(newFakeRepo(), fakeCredentials{"cred": credential("cred", owner, models.CredentialTypeBasicAuth)})
	ctx := context.Background()
	client, err := svc.Create(ctx, owner, CreateInput{Spec: []byte(petstoreSpec)})
	require.NoError(t, err)

	name, baseURL := "Renamed", " https://staging.pets.test/v1/ "
	updated, err := svc.Update(ctx, owner, client.ID, UpdateInput{
		Name:           &name,
		BaseURL:        &baseURL,
		Auth:           &models.APIClientAuth{Type: models.APIClientAuthBasic, CredentialID: "cred"},
		DefaultHeaders: map[string]string{"X-Env": "staging"},
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Name)
	assert.Equal(t, "https://staging.pets.test/v1", updated.BaseURL)
	assert.Equal(t, models.APIClientAuthBasic, updated.Auth.Type)
	assert.Equal(t, map[string]string{"X-Env": "staging"}, updated.DefaultHeaders)
	assert.Len(t, updated.Operations, 2, "a settings update keeps operations")

	empty := " "
	_, err = svc.Update(ctx, owner, client.ID, UpdateInput{Name: &empty})
	var vErr *models.ValidationError
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, "name", vErr.Field)
}

func TestService_ReplaceSpecReportsChanges(t *testing.T) {
	svc := NewService(newFakeRepo(), fakeCredentials{})
	ctx := context.Background()
	owner := uuid.NewString()
	client, err := svc.Create(ctx, owner, CreateInput{Spec: []byte(petstoreSpec), Name: "Pets"})
	require.NoError(t, err)

	updated, changes, err := svc.ReplaceSpec(ctx, owner, client.ID, []byte(petstoreSpecV2))
	require.NoError(t, err)
	assert.Equal(t, &SpecChanges{Added: []string{"updatePet"}, Removed: []string{"deletePet"}}, changes)
	assert.Equal(t, "Pets", updated.Name, "settings survive a specification update")
	assert.Equal(t, "https://pets.test/v1", updated.BaseURL, "an existing base URL is kept")
	assert.Equal(t, "Petstore v2", updated.Title)
	assert.Equal(t, 2, updated.OperationCount)

	_, spec, err := svc.GetSpec(ctx, owner, client.ID)
	require.NoError(t, err)
	assert.Equal(t, petstoreSpecV2, spec)

	_, _, err = svc.ReplaceSpec(ctx, owner, client.ID, []byte("{}"))
	assert.ErrorIs(t, err, models.ErrInvalidAPISpec)
}

func TestService_GetAPIClientCachesUntilUpdated(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fakeCredentials{})
	ctx := context.Background()
	owner := uuid.NewString()
	client, err := svc.Create(ctx, owner, CreateInput{Spec: []byte(petstoreSpec)})
	require.NoError(t, err)

	first, err := svc.GetAPIClient(ctx, client.ID)
	require.NoError(t, err)
	second, err := svc.GetAPIClient(ctx, client.ID)
	require.NoError(t, err)
	assert.Same(t, first, second)
	assert.Equal(t, 1, repo.findByIDs)

	// A change made elsewhere (another server) bumps updated_at in storage.
	repo.mu.Lock()
	repo.clients[client.ID].UpdatedAt = repo.clients[client.ID].UpdatedAt.Add(time.Minute)
	repo.clients[client.ID].Name = "Changed elsewhere"
	repo.mu.Unlock()

	third, err := svc.GetAPIClient(ctx, client.ID)
	require.NoError(t, err)
	assert.Equal(t, "Changed elsewhere", third.Name)
	assert.Equal(t, 2, repo.findByIDs)

	require.NoError(t, svc.Delete(ctx, owner, client.ID))
	_, err = svc.GetAPIClient(ctx, client.ID)
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound)

	_, err = svc.GetAPIClient(ctx, "bad-id")
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound)
}

func TestClientCacheEvictsWhenFull(t *testing.T) {
	cache := newClientCache(2)
	now := time.Now()
	for i := 0; i < 3; i++ {
		cache.put(&models.APIClient{ID: uuid.NewString(), UpdatedAt: now})
	}
	assert.Len(t, cache.entries, 2)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "Прив", truncate("Привет", 4))
	assert.Equal(t, "ok", truncate("  ok  ", 10))
}
