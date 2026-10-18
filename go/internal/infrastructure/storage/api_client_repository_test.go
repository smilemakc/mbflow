package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func newTestAPIClient(ownerID string) *pkgmodels.APIClient {
	return &pkgmodels.APIClient{
		OwnerID:     ownerID,
		Name:        "Petstore",
		Description: "Pets",
		Title:       "Petstore API",
		APIVersion:  "1.0.0",
		SpecVersion: "3.0.3",
		BaseURL:     "https://pets.test/v1",
		Servers:     []pkgmodels.APIServer{{URL: "https://pets.test/v1"}},
		SecuritySchemes: []*pkgmodels.APISecurityScheme{
			{Name: "key", Type: "apiKey", In: "header", ParamName: "X-Key"},
		},
		Auth: pkgmodels.APIClientAuth{Type: pkgmodels.APIClientAuthAPIKey, In: "header", Name: "X-Key"},
		Operations: []*pkgmodels.APIOperation{
			{
				ID:     "getPet",
				Method: "GET",
				Path:   "/pets/{id}",
				Parameters: []*pkgmodels.APIParameter{
					{Name: "id", In: "path", Required: true, Schema: map[string]any{"type": "integer"}},
				},
				Responses: []*pkgmodels.APIResponse{{
					Status:  "200",
					Content: []*pkgmodels.APIMediaType{{ContentType: "application/json", Schema: map[string]any{"$ref": "#/components/schemas/Pet"}}},
				}},
			},
		},
		Schemas:  map[string]map[string]any{"Pet": {"type": "object"}},
		Warnings: []pkgmodels.APISpecWarning{{Code: "no_servers", Message: "x"}},
		SpecRaw:  "openapi: 3.0.3",
		SpecSize: 14,
	}
}

func TestAPIClientRepository_CRUD(t *testing.T) {
	t.Parallel()
	db, cleanup := testutil.SetupTestTx(t)
	defer cleanup()
	repo := NewAPIClientRepository(db)
	ctx := context.Background()
	ownerID := createCredentialsTestUser(t, db)

	client := newTestAPIClient(ownerID)
	require.NoError(t, repo.Create(ctx, client))
	require.NotEmpty(t, client.ID)
	assert.Equal(t, 1, client.OperationCount)

	id := uuid.MustParse(client.ID)
	loaded, err := repo.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "Petstore", loaded.Name)
	assert.Equal(t, ownerID, loaded.OwnerID)
	assert.Equal(t, client.Auth, loaded.Auth)
	assert.Equal(t, client.Servers, loaded.Servers)
	assert.Empty(t, loaded.SpecRaw, "the original document is not loaded with the client")
	require.Len(t, loaded.Operations, 1)
	assert.Equal(t, "/pets/{id}", loaded.Operations[0].Path)
	assert.Equal(t, "integer", loaded.Operations[0].Parameters[0].Schema["type"])
	assert.Equal(t, map[string]any{"type": "object"}, loaded.Schemas["Pet"])
	assert.Equal(t, map[string]string{}, loaded.DefaultHeaders)

	spec, err := repo.FindSpec(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "openapi: 3.0.3", spec)

	// A settings update keeps the specification intact.
	loaded.Name = "Renamed"
	loaded.DefaultHeaders = map[string]string{"X-Env": "test"}
	loaded.Operations = nil
	require.NoError(t, repo.Update(ctx, loaded))
	reloaded, err := repo.FindByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", reloaded.Name)
	assert.Equal(t, map[string]string{"X-Env": "test"}, reloaded.DefaultHeaders)
	assert.Len(t, reloaded.Operations, 1)
	assert.Equal(t, 1, reloaded.OperationCount)

	// A new document replaces the specification columns.
	reloaded.SpecRaw = "openapi: 3.1.0"
	reloaded.SpecVersion = "3.1.0"
	reloaded.Operations = append(reloaded.Operations, &pkgmodels.APIOperation{ID: "listPets", Method: "GET", Path: "/pets"})
	require.NoError(t, repo.Update(ctx, reloaded))
	assert.Equal(t, 2, reloaded.OperationCount)
	spec, err = repo.FindSpec(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "openapi: 3.1.0", spec)

	updatedAt, err := repo.FindUpdatedAt(ctx, id)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), updatedAt, time.Minute)

	list, total, err := repo.FindByOwner(ctx, uuid.MustParse(ownerID), 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, 2, list[0].OperationCount)
	assert.Empty(t, list[0].Operations, "listing skips operations")

	require.NoError(t, repo.Delete(ctx, id))
	_, err = repo.FindByID(ctx, id)
	assert.ErrorIs(t, err, pkgmodels.ErrAPIClientNotFound)
	_, err = repo.FindSpec(ctx, id)
	assert.ErrorIs(t, err, pkgmodels.ErrAPIClientNotFound)
	_, err = repo.FindUpdatedAt(ctx, id)
	assert.ErrorIs(t, err, pkgmodels.ErrAPIClientNotFound)
	assert.ErrorIs(t, repo.Delete(ctx, id), pkgmodels.ErrAPIClientNotFound)
	assert.ErrorIs(t, repo.Update(ctx, reloaded), pkgmodels.ErrAPIClientNotFound)

	_, total, err = repo.FindByOwner(ctx, uuid.MustParse(ownerID), 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
}

func TestAPIClientRepository_FindByOwnerIsScopedAndPaginated(t *testing.T) {
	t.Parallel()
	db, cleanup := testutil.SetupTestTx(t)
	defer cleanup()
	repo := NewAPIClientRepository(db)
	ctx := context.Background()
	alice := createCredentialsTestUser(t, db)
	bob := createCredentialsTestUser(t, db)

	for i := 0; i < 3; i++ {
		require.NoError(t, repo.Create(ctx, newTestAPIClient(alice)))
	}
	require.NoError(t, repo.Create(ctx, newTestAPIClient(bob)))

	page, total, err := repo.FindByOwner(ctx, uuid.MustParse(alice), 2, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, page, 2)

	page, _, err = repo.FindByOwner(ctx, uuid.MustParse(alice), 2, 2)
	require.NoError(t, err)
	assert.Len(t, page, 1)
	assert.Equal(t, alice, page[0].OwnerID)

	_, err = repo.FindByID(ctx, uuid.New())
	assert.ErrorIs(t, err, pkgmodels.ErrAPIClientNotFound)
}
