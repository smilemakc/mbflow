package rest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/rentalkey"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// rc1RentalUsageFails wraps a rental key repository whose usage queries fail.
type rc1RentalUsageFails struct {
	repository.RentalKeyRepository
}

func (rc1RentalUsageFails) GetUsageHistory(context.Context, string, int, int) ([]*models.RentalKeyUsageRecord, error) {
	return nil, errors.New("usage failed")
}

func (rc1RentalUsageFails) GetUsageSummary(context.Context, string) (*models.MultimodalUsage, int64, float64, error) {
	return nil, 0, 0, errors.New("summary failed")
}

func rc1RentalRouter(repo repository.RentalKeyRepository) *gin.Engine {
	log := rc1Logger()
	user := NewRentalKeyHandlers(rentalkey.NewProvider(repo, nil), log)
	admin := NewRentalKeyAdminHandlers(rentalkey.NewAdminService(repo, nil), log)

	r := gin.New()
	g := r.Group("/api/v1", rc1UserFromHeader())
	g.GET("/rental-keys", user.ListRentalKeys)
	g.GET("/rental-keys/:id", user.GetRentalKey)
	g.GET("/rental-keys/:id/usage", user.GetRentalKeyUsage)
	g.GET("/rental-keys/:id/summary", user.GetRentalKeyUsageSummary)
	g.GET("/np/rk", user.GetRentalKey)
	g.GET("/np/rk/usage", user.GetRentalKeyUsage)
	g.GET("/np/rk/summary", user.GetRentalKeyUsageSummary)

	a := g.Group("/admin/rental-keys")
	a.POST("", admin.CreateRentalKey)
	a.GET("", admin.ListAllRentalKeys)
	a.GET("/:id", admin.GetRentalKey)
	a.PUT("/:id", admin.UpdateRentalKey)
	a.DELETE("/:id", admin.DeleteRentalKey)
	a.POST("/:id/rotate-key", admin.RotateRentalKeyAPIKey)
	a.POST("/reset-daily", admin.ResetDailyUsage)
	a.POST("/reset-monthly", admin.ResetMonthlyUsage)
	g.GET("/np/admin", admin.GetRentalKey)
	g.PUT("/np/admin", admin.UpdateRentalKey)
	g.DELETE("/np/admin", admin.DeleteRentalKey)
	g.POST("/np/admin/rotate", admin.RotateRentalKeyAPIKey)
	return r
}

func TestRC1RentalKeys_AdminAndUserFlows(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	repo := storage.NewRentalKeyRepository(db, rc1Encryption(t))
	r := rc1RentalRouter(repo)
	adminID := rc1CreateUser(t, db, "admin")
	alice := rc1CreateUser(t, db, "alice")
	bob := rc1CreateUser(t, db, "bob")
	asAdmin := map[string]string{"X-Test-User": adminID.String()}
	asAlice := map[string]string{"X-Test-User": alice.String()}
	asBob := map[string]string{"X-Test-User": bob.String()}
	base := "/api/v1/admin/rental-keys"

	// Create.
	w := rc1Do(t, r, http.MethodPost, base, map[string]any{}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodPost, base, map[string]any{"owner_id": alice.String(), "name": "k", "provider": "nope", "api_key": "x"}, asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodPost, base, map[string]any{
		"owner_id": uuid.NewString(), "name": "orphan", "provider": "openai", "api_key": "sk-x",
	}, asAdmin)
	assert.Equal(t, http.StatusInternalServerError, w.Code, "owner must exist")

	limit := 100
	w = rc1Do(t, r, http.MethodPost, base, map[string]any{
		"owner_id": alice.String(), "name": "alice openai", "description": "d", "provider": "openai",
		"api_key": "sk-secret-value", "daily_request_limit": limit, "provisioner_type": "manual",
	}, asAdmin)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	created := rc1JSON(t, w)
	keyID := created["id"].(string)
	assert.Equal(t, alice.String(), created["owner_id"])
	assert.Equal(t, adminID.String(), created["created_by"])
	assert.Equal(t, "manual", created["provisioner_type"])
	assert.NotContains(t, w.Body.String(), "sk-secret-value")

	w = rc1Do(t, r, http.MethodPost, base, map[string]any{
		"owner_id": alice.String(), "name": "alice anthropic", "provider": "anthropic", "api_key": "sk-ant",
	}, asAdmin)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// Record usage so the usage endpoints have data.
	require.NoError(t, repo.RecordUsage(context.Background(), keyID, &models.RentalKeyUsageRecord{
		Model: "gpt-4o", Usage: models.MultimodalUsage{PromptTokens: 10, CompletionTokens: 5},
		EstimatedCost: 0.25, Status: "success", ResponseTimeMs: 12,
	}))

	// User list (with provider filter).
	w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["rental_keys"], 2)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys?provider=anthropic", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["rental_keys"], 1)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys", nil, asBob)
	assert.Len(t, rc1JSON(t, w)["rental_keys"], 0)

	// User get/usage/summary with ownership checks.
	for _, suffix := range []string{"", "/usage?limit=1000", "/summary"} {
		path := "/api/v1/rental-keys/" + keyID + suffix
		w = rc1Do(t, r, http.MethodGet, path, nil, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code, path)
		w = rc1Do(t, r, http.MethodGet, path, nil, asBob)
		assert.Equal(t, http.StatusForbidden, w.Code, path)
		w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys/"+uuid.NewString()+suffix, nil, asAlice)
		assert.Equal(t, http.StatusNotFound, w.Code, path)
		w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys/not-a-uuid"+suffix, nil, asAlice)
		assert.Equal(t, http.StatusNotFound, w.Code, path)
	}
	w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys/"+keyID, nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	got := rc1JSON(t, w)
	assert.EqualValues(t, 1, got["total_requests"])
	assert.EqualValues(t, 15, got["total_usage"].(map[string]any)["total"])
	assert.NotContains(t, got, "owner_id", "user view omits admin fields")

	w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys/"+keyID+"/usage?limit=1000", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	usage := rc1JSON(t, w)
	assert.EqualValues(t, 100, usage["limit"])
	require.Len(t, usage["usage"], 1)
	rec := usage["usage"].([]any)[0].(map[string]any)
	assert.Equal(t, "gpt-4o", rec["model"])
	assert.EqualValues(t, 15, rec["usage"].(map[string]any)["total"])

	w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys/"+keyID+"/summary", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	summary := rc1JSON(t, w)["summary"].(map[string]any)
	assert.EqualValues(t, 1, summary["total_requests"])
	assert.EqualValues(t, 0.25, summary["total_cost"])

	for _, p := range []string{"/api/v1/np/rk", "/api/v1/np/rk/usage", "/api/v1/np/rk/summary"} {
		w = rc1Do(t, r, http.MethodGet, p, nil, asAlice)
		assert.Equal(t, http.StatusBadRequest, w.Code, p)
	}

	// Admin list with filters.
	w = rc1Do(t, r, http.MethodGet, base+"?limit=500&provider=openai&status=active&owner_id="+alice.String()+"&created_by="+adminID.String(), nil, asAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	list := rc1JSON(t, w)
	assert.EqualValues(t, 100, list["limit"])
	assert.EqualValues(t, 1, list["total"])

	// Admin get.
	w = rc1Do(t, r, http.MethodGet, base+"/"+keyID, nil, asAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, alice.String(), rc1JSON(t, w)["owner_id"])
	w = rc1Do(t, r, http.MethodGet, base+"/"+uuid.NewString(), nil, asAdmin)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Admin update.
	w = rc1Do(t, r, http.MethodPut, base+"/"+keyID, map[string]any{"name": "x"}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodPut, base+"/"+keyID, "{bad", asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodPut, base+"/"+uuid.NewString(), map[string]any{"name": "x"}, asAdmin)
	assert.Equal(t, http.StatusNotFound, w.Code)
	newLimit := int64(5000)
	w = rc1Do(t, r, http.MethodPut, base+"/"+keyID, map[string]any{
		"name": "renamed", "description": "nd", "status": "suspended", "monthly_token_limit": newLimit,
		"daily_request_limit": 7, "provider_config": map[string]any{"org": "o"},
	}, asAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	upd := rc1JSON(t, w)
	assert.Equal(t, "renamed", upd["name"])
	assert.Equal(t, "suspended", upd["status"])
	assert.EqualValues(t, newLimit, upd["monthly_token_limit"])

	// Rotate.
	w = rc1Do(t, r, http.MethodPost, base+"/"+keyID+"/rotate-key", map[string]any{"new_api_key": "k2"}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodPost, base+"/"+keyID+"/rotate-key", map[string]any{}, asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodPost, base+"/"+uuid.NewString()+"/rotate-key", map[string]any{"new_api_key": "k2"}, asAdmin)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, r, http.MethodPost, base+"/"+keyID+"/rotate-key", map[string]any{"new_api_key": "sk-rotated"}, asAdmin)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	plain, err := repo.GetDecryptedAPIKey(context.Background(), keyID)
	require.NoError(t, err)
	assert.Equal(t, "sk-rotated", plain)

	// Reset counters.
	for _, p := range []string{"/reset-daily", "/reset-monthly"} {
		w = rc1Do(t, r, http.MethodPost, base+p, nil, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code, p)
		w = rc1Do(t, r, http.MethodPost, base+p, nil, asAdmin)
		assert.Equal(t, http.StatusOK, w.Code, p)
	}
	w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys/"+keyID, nil, asAlice)
	assert.EqualValues(t, 0, rc1JSON(t, w)["requests_today"])
	assert.EqualValues(t, 0, rc1JSON(t, w)["tokens_this_month"])

	// Delete.
	w = rc1Do(t, r, http.MethodDelete, base+"/"+keyID, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodDelete, base+"/not-a-uuid", nil, asAdmin)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, r, http.MethodDelete, base+"/"+keyID, nil, asAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys/"+keyID, nil, asAlice)
	assert.Equal(t, http.StatusNotFound, w.Code)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/np/admin"}, {http.MethodPut, "/api/v1/np/admin"},
		{http.MethodDelete, "/api/v1/np/admin"}, {http.MethodPost, "/api/v1/np/admin/rotate"},
	} {
		w = rc1Do(t, r, tc.method, tc.path, map[string]any{"new_api_key": "x"}, asAdmin)
		assert.Equal(t, http.StatusBadRequest, w.Code, tc.path)
	}
}

func TestRC1RentalKeys_StorageFailures(t *testing.T) {
	t.Parallel()
	enc := rc1Encryption(t)
	r := rc1RentalRouter(storage.NewRentalKeyRepository(rc1ClosedDB(t), enc))
	u := map[string]string{"X-Test-User": uuid.NewString()}
	id := uuid.NewString()
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/v1/rental-keys", nil},
		{http.MethodGet, "/api/v1/rental-keys?provider=openai", nil},
		{http.MethodGet, "/api/v1/rental-keys/" + id, nil},
		{http.MethodGet, "/api/v1/rental-keys/" + id + "/usage", nil},
		{http.MethodGet, "/api/v1/rental-keys/" + id + "/summary", nil},
		{http.MethodPost, "/api/v1/admin/rental-keys", map[string]any{"owner_id": id, "name": "k", "provider": "openai", "api_key": "x"}},
		{http.MethodGet, "/api/v1/admin/rental-keys", nil},
		{http.MethodGet, "/api/v1/admin/rental-keys/" + id, nil},
		{http.MethodPut, "/api/v1/admin/rental-keys/" + id, map[string]any{"name": "n"}},
		{http.MethodDelete, "/api/v1/admin/rental-keys/" + id, nil},
		{http.MethodPost, "/api/v1/admin/rental-keys/" + id + "/rotate-key", map[string]any{"new_api_key": "x"}},
		{http.MethodPost, "/api/v1/admin/rental-keys/reset-daily", nil},
		{http.MethodPost, "/api/v1/admin/rental-keys/reset-monthly", nil},
	} {
		w := rc1Do(t, r, tc.method, tc.path, tc.body, u)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "%s %s", tc.method, tc.path)
	}

	// Usage queries failing on an otherwise healthy store.
	db := rc1DB(t)
	owner := rc1CreateUser(t, db, "owner")
	repo := storage.NewRentalKeyRepository(db, enc)
	key := models.NewRentalKeyResource(owner.String(), "k", models.LLMProviderTypeOpenAI)
	require.NoError(t, repo.CreateRentalKey(context.Background(), key, "sk"))
	r = rc1RentalRouter(rc1RentalUsageFails{repo})
	asOwner := map[string]string{"X-Test-User": owner.String()}
	for _, suffix := range []string{"/usage", "/summary"} {
		w := rc1Do(t, r, http.MethodGet, "/api/v1/rental-keys/"+key.ID+suffix, nil, asOwner)
		assert.Equal(t, http.StatusInternalServerError, w.Code, suffix)
	}
}

func TestRC1RentalKeys_UsageConvertersHandleNil(t *testing.T) {
	t.Parallel()
	assert.Equal(t, MultimodalUsageResponse{}, toMultimodalUsageResponse(nil))
	assert.Equal(t, MultimodalUsageResponse{}, (&RentalKeyHandlers{}).toMultimodalUsageResponse(nil))
	full := &models.MultimodalUsage{PromptTokens: 1, CompletionTokens: 2, ImageInputTokens: 3, ImageOutputTokens: 4,
		AudioInputTokens: 5, AudioOutputTokens: 6, VideoInputTokens: 7, VideoOutputTokens: 8}
	assert.EqualValues(t, 36, toMultimodalUsageResponse(full).Total)
}
