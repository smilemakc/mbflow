package rest

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

// rc1CredRepoFaults wraps a credentials repository and injects write failures.
type rc1CredRepoFaults struct {
	repository.CredentialsRepository
	failUpdate, failDelete, failUsage bool
}

func (r rc1CredRepoFaults) UpdateCredentials(ctx context.Context, c *models.CredentialsResource) error {
	if r.failUpdate {
		return errors.New("update failed")
	}
	return r.CredentialsRepository.UpdateCredentials(ctx, c)
}

func (r rc1CredRepoFaults) DeleteCredentials(ctx context.Context, id string) error {
	if r.failDelete {
		return errors.New("delete failed")
	}
	return r.CredentialsRepository.DeleteCredentials(ctx, id)
}

func (r rc1CredRepoFaults) IncrementUsageCount(ctx context.Context, id string) error {
	if r.failUsage {
		return errors.New("usage failed")
	}
	return r.CredentialsRepository.IncrementUsageCount(ctx, id)
}

func (r rc1CredRepoFaults) LogCredentialAccess(ctx context.Context, id, action, actorID, actorType string, md map[string]any) error {
	if r.failUsage {
		return errors.New("log failed")
	}
	return r.CredentialsRepository.LogCredentialAccess(ctx, id, action, actorID, actorType, md)
}

// rc1WorkflowRepoUnassignFails wraps a workflow repository whose resource
// detach step fails.
type rc1WorkflowRepoUnassignFails struct {
	repository.WorkflowRepository
}

func (rc1WorkflowRepoUnassignFails) UnassignResourceFromAllWorkflows(context.Context, uuid.UUID) (int64, error) {
	return 0, errors.New("detach failed")
}

// rc1ResourceRepoFaults wraps a resource repository and injects failures or
// replaces the resource returned by GetByID.
type rc1ResourceRepoFaults struct {
	repository.FileStorageRepository
	override   models.Resource
	failUpdate bool
	failDelete bool
}

func (r rc1ResourceRepoFaults) GetByID(ctx context.Context, id string) (models.Resource, error) {
	if r.override != nil {
		return r.override, nil
	}
	return r.FileStorageRepository.GetByID(ctx, id)
}

func (r rc1ResourceRepoFaults) Update(ctx context.Context, res models.Resource) error {
	if r.failUpdate {
		return errors.New("update failed")
	}
	return r.FileStorageRepository.Update(ctx, res)
}

func (r rc1ResourceRepoFaults) Delete(ctx context.Context, id string) error {
	if r.failDelete {
		return errors.New("delete failed")
	}
	return r.FileStorageRepository.Delete(ctx, id)
}

func rc1CredentialsRouter(creds repository.CredentialsRepository, wfs repository.WorkflowRepository, enc *crypto.EncryptionService) *gin.Engine {
	h := NewCredentialsHandlers(creds, wfs, enc, rc1Logger())
	r := gin.New()
	g := r.Group("/api/v1/credentials", rc1UserFromHeader())
	g.POST("/api-key", h.CreateAPIKey)
	g.POST("/basic-auth", h.CreateBasicAuth)
	g.POST("/oauth2", h.CreateOAuth2)
	g.POST("/service-account", h.CreateServiceAccount)
	g.POST("/custom", h.CreateCustom)
	g.GET("", h.ListCredentials)
	g.GET("/:id", h.GetCredential)
	g.GET("/:id/secrets", h.GetCredentialSecrets)
	g.PUT("/:id", h.UpdateCredential)
	g.DELETE("/:id", h.DeleteCredential)
	np := r.Group("/np", rc1UserFromHeader())
	np.GET("/get", h.GetCredential)
	np.GET("/secrets", h.GetCredentialSecrets)
	np.PUT("/update", h.UpdateCredential)
	np.DELETE("/delete", h.DeleteCredential)
	return r
}

var rc1CredentialCreates = []struct {
	path      string
	body      map[string]any
	credType  string
	secretKey string
	secret    string
}{
	{"/api-key", map[string]any{"name": "ak", "provider": "openai", "api_key": "sk-123"}, "api_key", "api_key", "sk-123"},
	{"/basic-auth", map[string]any{"name": "ba", "username": "u", "password": "p@ss"}, "basic_auth", "password", "p@ss"},
	{"/oauth2", map[string]any{"name": "oa", "provider": "google", "client_id": "cid", "client_secret": "cs",
		"access_token": "at", "refresh_token": "rt", "token_url": "https://t", "scopes": "a b"}, "oauth2", "refresh_token", "rt"},
	{"/service-account", map[string]any{"name": "sa", "provider": "gcp", "json_key": `{"k":"v"}`}, "service_account", "json_key", `{"k":"v"}`},
	{"/custom", map[string]any{"name": "cu", "data": map[string]string{"token": "tok-s3cret"}}, "custom", "token", "tok-s3cret"},
}

func TestRC1Credentials_CreateReadUpdateDelete(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	enc := rc1Encryption(t)
	credRepo := storage.NewCredentialsRepository(db)
	wfRepo := storage.NewWorkflowRepository(db)
	r := rc1CredentialsRouter(credRepo, wfRepo, enc)
	alice := rc1CreateUser(t, db, "alice")
	bob := rc1CreateUser(t, db, "bob")
	asAlice := map[string]string{"X-Test-User": alice.String()}
	asBob := map[string]string{"X-Test-User": bob.String()}

	ids := map[string]string{}
	for _, tc := range rc1CredentialCreates {
		path := "/api/v1/credentials" + tc.path
		w := rc1Do(t, r, http.MethodPost, path, tc.body, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code, path)
		w = rc1Do(t, r, http.MethodPost, path, map[string]any{"name": "x"}, asAlice)
		assert.Equal(t, http.StatusBadRequest, w.Code, path)

		w = rc1Do(t, r, http.MethodPost, path, tc.body, asAlice)
		require.Equal(t, http.StatusCreated, w.Code, "%s: %s", path, w.Body.String())
		assert.NotContains(t, w.Body.String(), tc.secret, "secrets must not be echoed")
		resp := rc1JSON(t, w)
		assert.Equal(t, tc.credType, resp["credential_type"])
		assert.Contains(t, resp["fields"], tc.secretKey)
		ids[tc.credType] = resp["id"].(string)
	}

	// List and filter.
	w := rc1Do(t, r, http.MethodGet, "/api/v1/credentials", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/credentials", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["credentials"], len(rc1CredentialCreates))
	w = rc1Do(t, r, http.MethodGet, "/api/v1/credentials?provider=openai", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["credentials"], 1)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/credentials", nil, asBob)
	assert.Len(t, rc1JSON(t, w)["credentials"], 0)

	// Secrets round-trip for the owner only.
	for _, tc := range rc1CredentialCreates {
		w = rc1Do(t, r, http.MethodGet, "/api/v1/credentials/"+ids[tc.credType]+"/secrets", nil, asAlice)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		assert.Equal(t, tc.secret, rc1JSON(t, w)["data"].(map[string]any)[tc.secretKey])
	}
	apiKeyID := ids["api_key"]
	w = rc1Do(t, r, http.MethodGet, "/api/v1/credentials/"+apiKeyID, nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, 1, rc1JSON(t, w)["usage_count"], "secret access counts as usage")

	// Common auth/ownership/not-found checks for the :id endpoints.
	for _, tc := range []struct{ method, suffix string }{
		{http.MethodGet, ""}, {http.MethodGet, "/secrets"}, {http.MethodPut, ""}, {http.MethodDelete, ""},
	} {
		body := map[string]any{"name": "n"}
		w = rc1Do(t, r, tc.method, "/api/v1/credentials/"+apiKeyID+tc.suffix, body, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		w = rc1Do(t, r, tc.method, "/api/v1/credentials/"+apiKeyID+tc.suffix, body, asBob)
		assert.Equal(t, http.StatusForbidden, w.Code, "%s %s", tc.method, tc.suffix)
		w = rc1Do(t, r, tc.method, "/api/v1/credentials/"+uuid.NewString()+tc.suffix, body, asAlice)
		assert.Equal(t, http.StatusNotFound, w.Code)
		w = rc1Do(t, r, tc.method, "/api/v1/credentials/not-a-uuid"+tc.suffix, body, asAlice)
		assert.Equal(t, http.StatusNotFound, w.Code)
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/np/get"}, {http.MethodGet, "/np/secrets"}, {http.MethodPut, "/np/update"}, {http.MethodDelete, "/np/delete"},
	} {
		w = rc1Do(t, r, tc.method, tc.path, map[string]any{}, asAlice)
		assert.Equal(t, http.StatusBadRequest, w.Code, tc.path)
	}

	// Update.
	w = rc1Do(t, r, http.MethodPut, "/api/v1/credentials/"+apiKeyID, "{bad", asAlice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodPut, "/api/v1/credentials/"+apiKeyID, map[string]any{"name": "renamed", "description": "d"}, asAlice)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "renamed", rc1JSON(t, w)["name"])

	// Expired credentials do not reveal secrets.
	cred, err := credRepo.GetCredentials(context.Background(), ids["custom"])
	require.NoError(t, err)
	past := time.Now().Add(-time.Hour)
	cred.ExpiresAt = &past
	require.NoError(t, credRepo.UpdateCredentials(context.Background(), cred))
	w = rc1Do(t, r, http.MethodGet, "/api/v1/credentials/"+ids["custom"]+"/secrets", nil, asAlice)
	assert.Equal(t, http.StatusGone, w.Code)

	// Delete detaches the credential from workflows first.
	wf := testutil.WorkflowDomainToModel(testutil.CreateSimpleWorkflow())
	require.NoError(t, wfRepo.Create(context.Background(), wf))
	require.NoError(t, wfRepo.AssignResource(context.Background(), wf.ID, &storagemodels.WorkflowResourceModel{ResourceID: uuid.MustParse(apiKeyID), Alias: "creds", AccessType: "read"}, &alice))
	w = rc1Do(t, r, http.MethodDelete, "/api/v1/credentials/"+apiKeyID, nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	attached, err := wfRepo.GetWorkflowResources(context.Background(), wf.ID)
	require.NoError(t, err)
	assert.Empty(t, attached)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/credentials/"+apiKeyID, nil, asAlice)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRC1Credentials_Failures(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	enc := rc1Encryption(t)
	credRepo := storage.NewCredentialsRepository(db)
	wfRepo := storage.NewWorkflowRepository(db)
	alice := rc1CreateUser(t, db, "alice")
	asAlice := map[string]string{"X-Test-User": alice.String()}

	// Encryption failures (an encryption service without a usable key).
	broken := rc1CredentialsRouter(credRepo, wfRepo, &crypto.EncryptionService{})
	for _, tc := range rc1CredentialCreates {
		w := rc1Do(t, broken, http.MethodPost, "/api/v1/credentials"+tc.path, tc.body, asAlice)
		assert.Equal(t, http.StatusInternalServerError, w.Code, tc.path)
		assert.Equal(t, "encryption failed", rc1JSON(t, w)["message"])
	}

	// Storage failures on create/list/get.
	closed := rc1CredentialsRouter(storage.NewCredentialsRepository(rc1ClosedDB(t)), wfRepo, enc)
	for _, tc := range rc1CredentialCreates {
		w := rc1Do(t, closed, http.MethodPost, "/api/v1/credentials"+tc.path, tc.body, asAlice)
		assert.Equal(t, http.StatusInternalServerError, w.Code, tc.path)
		assert.Equal(t, "failed to create credential", rc1JSON(t, w)["message"])
	}
	id := uuid.NewString()
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/credentials"},
		{http.MethodGet, "/api/v1/credentials?provider=x"},
		{http.MethodGet, "/api/v1/credentials/" + id},
		{http.MethodGet, "/api/v1/credentials/" + id + "/secrets"},
		{http.MethodPut, "/api/v1/credentials/" + id},
		{http.MethodDelete, "/api/v1/credentials/" + id},
	} {
		w := rc1Do(t, closed, tc.method, tc.path, map[string]any{"name": "n"}, asAlice)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "%s %s", tc.method, tc.path)
	}

	// Failures after the credential was found.
	live := rc1CredentialsRouter(credRepo, wfRepo, enc)
	w := rc1Do(t, live, http.MethodPost, "/api/v1/credentials/api-key", map[string]any{"name": "k", "api_key": "v"}, asAlice)
	require.Equal(t, http.StatusCreated, w.Code)
	credID := rc1JSON(t, w)["id"].(string)

	faulty := rc1CredentialsRouter(rc1CredRepoFaults{CredentialsRepository: credRepo, failUpdate: true, failDelete: true, failUsage: true}, wfRepo, enc)
	w = rc1Do(t, faulty, http.MethodPut, "/api/v1/credentials/"+credID, map[string]any{"name": "n"}, asAlice)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	w = rc1Do(t, faulty, http.MethodDelete, "/api/v1/credentials/"+credID, nil, asAlice)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "failed to delete credential", rc1JSON(t, w)["message"])
	// Usage bookkeeping failures do not block reading secrets.
	w = rc1Do(t, faulty, http.MethodGet, "/api/v1/credentials/"+credID+"/secrets", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "v", rc1JSON(t, w)["data"].(map[string]any)["api_key"])

	detach := rc1CredentialsRouter(credRepo, rc1WorkflowRepoUnassignFails{wfRepo}, enc)
	w = rc1Do(t, detach, http.MethodDelete, "/api/v1/credentials/"+credID, nil, asAlice)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "failed to detach credential from workflows", rc1JSON(t, w)["message"])

	// Data encrypted with another key cannot be decrypted.
	otherKey, err := crypto.NewEncryptionService([]byte("fedcba9876543210fedcba9876543210"))
	require.NoError(t, err)
	wrongKey := rc1CredentialsRouter(credRepo, wfRepo, otherKey)
	w = rc1Do(t, wrongKey, http.MethodGet, "/api/v1/credentials/"+credID+"/secrets", nil, asAlice)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "decryption failed", rc1JSON(t, w)["message"])
}

// ---------------------------------------------------------------------------
// Resources
// ---------------------------------------------------------------------------

func rc1ResourceRouter(res repository.FileStorageRepository, plans repository.PricingPlanRepository, wfs repository.WorkflowRepository) *gin.Engine {
	h := NewResourceHandlers(res, plans, wfs, rc1Logger())
	r := gin.New()
	g := r.Group("/api/v1/resources", rc1UserFromHeader())
	g.POST("/file-storage", h.CreateFileStorage)
	g.GET("", h.ListResources)
	g.GET("/pricing-plans", h.ListPricingPlans)
	g.GET("/:id", h.GetResource)
	g.PUT("/:id", h.UpdateResource)
	g.DELETE("/:id", h.DeleteResource)
	np := r.Group("/np", rc1UserFromHeader())
	np.GET("/get", h.GetResource)
	np.PUT("/update", h.UpdateResource)
	np.DELETE("/delete", h.DeleteResource)
	return r
}

func TestRC1Resources_Lifecycle(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	enc := rc1Encryption(t)
	resRepo := storage.NewResourceRepository(db)
	planRepo := storage.NewPricingPlanRepository(db)
	wfRepo := storage.NewWorkflowRepository(db)
	r := rc1ResourceRouter(resRepo, planRepo, wfRepo)
	alice := rc1CreateUser(t, db, "alice")
	bob := rc1CreateUser(t, db, "bob")
	asAlice := map[string]string{"X-Test-User": alice.String()}
	asBob := map[string]string{"X-Test-User": bob.String()}
	ctx := context.Background()

	// Create file storage.
	w := rc1Do(t, r, http.MethodPost, "/api/v1/resources/file-storage", map[string]any{"name": "fs"}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodPost, "/api/v1/resources/file-storage", map[string]any{}, asAlice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodPost, "/api/v1/resources/file-storage", map[string]any{"name": "fs", "description": "files"}, asAlice)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	fs := rc1JSON(t, w)
	fsID := fs["id"].(string)
	assert.Equal(t, "fs", fs["name"])
	assert.Greater(t, fs["storage_limit_bytes"].(float64), 0.0)

	// Other resource types owned by alice.
	cred := models.NewCredentialsResource(alice.String(), "cred", models.CredentialTypeAPIKey)
	cred.Provider = "openai"
	cred.EncryptedData = map[string]string{"api_key": "enc"}
	require.NoError(t, storage.NewCredentialsRepository(db).CreateCredentials(ctx, cred))
	rk := models.NewRentalKeyResource(alice.String(), "rental", models.LLMProviderTypeOpenAI)
	require.NoError(t, storage.NewRentalKeyRepository(db, enc).CreateRentalKey(ctx, rk, "sk"))

	// List shows every type with its specific fields.
	w = rc1Do(t, r, http.MethodGet, "/api/v1/resources", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/resources", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	byType := map[string]map[string]any{}
	for _, item := range rc1JSON(t, w)["resources"].([]any) {
		m := item.(map[string]any)
		byType[m["type"].(string)] = m
	}
	require.Len(t, byType, 3)
	assert.Contains(t, byType["file_storage"], "usage_percent")
	assert.Equal(t, "openai", byType["credentials"]["provider"])
	assert.Contains(t, byType["rental_key"], "total_cost")
	w = rc1Do(t, r, http.MethodGet, "/api/v1/resources", nil, asBob)
	assert.Len(t, rc1JSON(t, w)["resources"], 0)

	// Get with ownership.
	w = rc1Do(t, r, http.MethodGet, "/api/v1/resources/"+fsID, nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, rc1JSON(t, w), "available_space")
	w = rc1Do(t, r, http.MethodGet, "/api/v1/resources/"+cred.ID, nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, rc1JSON(t, w), "available_space")

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		body := map[string]any{"name": "n"}
		w = rc1Do(t, r, method, "/api/v1/resources/"+fsID, body, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code, method)
		w = rc1Do(t, r, method, "/api/v1/resources/"+fsID, body, asBob)
		assert.Equal(t, http.StatusForbidden, w.Code, method)
		w = rc1Do(t, r, method, "/api/v1/resources/"+uuid.NewString(), body, asAlice)
		assert.Equal(t, http.StatusNotFound, w.Code, method)
		w = rc1Do(t, r, method, "/api/v1/resources/not-a-uuid", body, asAlice)
		assert.Equal(t, http.StatusBadRequest, w.Code, method)
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/np/get"}, {http.MethodPut, "/np/update"}, {http.MethodDelete, "/np/delete"},
	} {
		w = rc1Do(t, r, tc.method, tc.path, map[string]any{}, asAlice)
		assert.Equal(t, http.StatusBadRequest, w.Code, tc.path)
	}

	// Update.
	w = rc1Do(t, r, http.MethodPut, "/api/v1/resources/"+fsID, map[string]any{"name": ""}, asAlice)
	require.Equal(t, http.StatusOK, w.Code, "empty name keeps the old one")
	assert.Equal(t, "fs", rc1JSON(t, w)["name"])
	w = rc1Do(t, r, http.MethodPut, "/api/v1/resources/"+fsID, "{bad", asAlice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodPut, "/api/v1/resources/"+fsID, map[string]any{"name": "fs2", "description": "d2"}, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	upd := rc1JSON(t, w)
	assert.Equal(t, "fs2", upd["name"])
	assert.Equal(t, "d2", upd["description"])
	assert.Contains(t, upd, "available_space")
	w = rc1Do(t, r, http.MethodPut, "/api/v1/resources/"+cred.ID, map[string]any{"name": "cred2"}, asAlice)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "cred2", rc1JSON(t, w)["name"])
	w = rc1Do(t, r, http.MethodPut, "/api/v1/resources/"+rk.ID, map[string]any{"name": "x"}, asAlice)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "FORBIDDEN_OPERATION", rc1JSON(t, w)["code"])

	// Delete detaches from workflows.
	wf := testutil.WorkflowDomainToModel(testutil.CreateSimpleWorkflow())
	require.NoError(t, wfRepo.Create(ctx, wf))
	require.NoError(t, wfRepo.AssignResource(ctx, wf.ID, &storagemodels.WorkflowResourceModel{ResourceID: uuid.MustParse(fsID), Alias: "files", AccessType: "read"}, &alice))
	w = rc1Do(t, r, http.MethodDelete, "/api/v1/resources/"+fsID, nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	attached, err := wfRepo.GetWorkflowResources(ctx, wf.ID)
	require.NoError(t, err)
	assert.Empty(t, attached)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/resources/"+fsID, nil, asAlice)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Pricing plans.
	w = rc1Do(t, r, http.MethodGet, "/api/v1/resources/pricing-plans", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	plans := rc1JSON(t, w)["plans"].([]any)
	require.NotEmpty(t, plans)
	for _, p := range plans {
		assert.Equal(t, "file_storage", p.(map[string]any)["resource_type"])
	}
	w = rc1Do(t, r, http.MethodGet, "/api/v1/resources/pricing-plans?resource_type=nothing", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["plans"], 0)
}

func TestRC1Resources_Failures(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	resRepo := storage.NewResourceRepository(db)
	planRepo := storage.NewPricingPlanRepository(db)
	wfRepo := storage.NewWorkflowRepository(db)
	alice := rc1CreateUser(t, db, "alice")
	asAlice := map[string]string{"X-Test-User": alice.String()}

	live := rc1ResourceRouter(resRepo, planRepo, wfRepo)
	w := rc1Do(t, live, http.MethodPost, "/api/v1/resources/file-storage", map[string]any{"name": "fs"}, asAlice)
	require.Equal(t, http.StatusCreated, w.Code)
	fsID := rc1JSON(t, w)["id"].(string)

	// Owner must exist.
	w = rc1Do(t, live, http.MethodPost, "/api/v1/resources/file-storage", map[string]any{"name": "fs"},
		map[string]string{"X-Test-User": uuid.NewString()})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	// Malformed owner IDs cannot list anything.
	w = rc1Do(t, live, http.MethodGet, "/api/v1/resources", nil, map[string]string{"X-Test-User": "nope"})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Broken store.
	closedDB := rc1ClosedDB(t)
	closed := rc1ResourceRouter(storage.NewResourceRepository(closedDB), storage.NewPricingPlanRepository(closedDB), wfRepo)
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/resources/file-storage"},
		{http.MethodGet, "/api/v1/resources"},
		{http.MethodGet, "/api/v1/resources/pricing-plans"},
		{http.MethodGet, "/api/v1/resources/" + fsID},
		{http.MethodPut, "/api/v1/resources/" + fsID},
		{http.MethodDelete, "/api/v1/resources/" + fsID},
	} {
		w = rc1Do(t, closed, tc.method, tc.path, map[string]any{"name": "n"}, asAlice)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "%s %s", tc.method, tc.path)
	}

	// Injected failures after the resource was found.
	faulty := rc1ResourceRouter(rc1ResourceRepoFaults{FileStorageRepository: resRepo, failUpdate: true, failDelete: true}, planRepo, wfRepo)
	w = rc1Do(t, faulty, http.MethodPut, "/api/v1/resources/"+fsID, map[string]any{"name": "n"}, asAlice)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	w = rc1Do(t, faulty, http.MethodDelete, "/api/v1/resources/"+fsID, nil, asAlice)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	detach := rc1ResourceRouter(resRepo, planRepo, rc1WorkflowRepoUnassignFails{wfRepo})
	w = rc1Do(t, detach, http.MethodDelete, "/api/v1/resources/"+fsID, nil, asAlice)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// A resource of an unknown kind cannot be updated; one with a malformed ID
	// cannot be deleted.
	odd := &models.BaseResource{ID: "odd-id", Type: "other", OwnerID: alice.String(), Name: "odd"}
	oddRouter := rc1ResourceRouter(rc1ResourceRepoFaults{FileStorageRepository: resRepo, override: odd}, planRepo, wfRepo)
	w = rc1Do(t, oddRouter, http.MethodPut, "/api/v1/resources/odd-id", map[string]any{"name": "n"}, asAlice)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "UNSUPPORTED_RESOURCE_TYPE", rc1JSON(t, w)["code"])
	w = rc1Do(t, oddRouter, http.MethodDelete, "/api/v1/resources/odd-id", nil, asAlice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, oddRouter, http.MethodGet, "/api/v1/resources", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["resources"], 1)
}
