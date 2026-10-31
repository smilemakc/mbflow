package rest

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/application/servicekey"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
)

func rc1ServiceKeyRouter(db *bun.DB) *gin.Engine {
	log := rc1Logger()
	svc := servicekey.NewService(storage.NewServiceKeyRepository(db), servicekey.Config{MaxKeysPerUser: 2, BcryptCost: 4})
	user := NewServiceKeyHandlers(svc, log)
	admin := NewServiceKeyAdminHandlers(svc, log)

	r := gin.New()
	g := r.Group("/api/v1", rc1UserFromHeader())
	g.GET("/service-keys", user.ListMyServiceKeys)
	g.GET("/service-keys/:id", user.GetMyServiceKey)
	g.GET("/service-keys-noparam", user.GetMyServiceKey)

	a := g.Group("/admin/service-keys")
	a.POST("", admin.CreateServiceKey)
	a.GET("", admin.ListServiceKeys)
	a.GET("/:id", admin.GetServiceKey)
	a.DELETE("/:id", admin.DeleteServiceKey)
	a.POST("/:id/revoke", admin.RevokeServiceKey)
	g.GET("/admin/service-keys-noparam", admin.GetServiceKey)
	g.DELETE("/admin/service-keys-noparam", admin.DeleteServiceKey)
	g.POST("/admin/service-keys-noparam/revoke", admin.RevokeServiceKey)
	return r
}

func TestRC1ServiceKeys_AdminLifecycleAndOwnerIsolation(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	r := rc1ServiceKeyRouter(db)
	adminID := rc1CreateUser(t, db, "admin")
	alice := rc1CreateUser(t, db, "alice")
	bob := rc1CreateUser(t, db, "bob")
	asAdmin := map[string]string{"X-Test-User": adminID.String(), "X-Test-Admin": "true"}
	asAlice := map[string]string{"X-Test-User": alice.String()}
	asBob := map[string]string{"X-Test-User": bob.String()}
	base := "/api/v1/admin/service-keys"

	// Create: auth + validation.
	w := rc1Do(t, r, http.MethodPost, base, map[string]any{"user_id": alice.String(), "name": "k"}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodPost, base, map[string]any{"user_id": "nope", "name": "k"}, asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, rc1JSON(t, w)["message"], "userid must be a valid UUID")

	days := 30
	w = rc1Do(t, r, http.MethodPost, base, map[string]any{
		"user_id": alice.String(), "name": "alice key", "description": "d", "expires_in_days": days,
	}, asAdmin)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	created := rc1JSON(t, w)
	assert.True(t, strings.HasPrefix(created["key"].(string), "sk_"))
	assert.Equal(t, alice.String(), created["user_id"])
	assert.Equal(t, adminID.String(), created["created_by"])
	assert.Equal(t, "active", created["status"])
	assert.NotEmpty(t, created["expires_at"])
	keyID := created["id"].(string)

	w = rc1Do(t, r, http.MethodPost, base, map[string]any{"user_id": alice.String(), "name": "second"}, asAdmin)
	require.Equal(t, http.StatusCreated, w.Code)
	// MaxKeysPerUser is 2.
	w = rc1Do(t, r, http.MethodPost, base, map[string]any{"user_id": alice.String(), "name": "third"}, asAdmin)
	assert.Equal(t, http.StatusConflict, w.Code)
	w = rc1Do(t, r, http.MethodPost, base, map[string]any{"user_id": bob.String(), "name": "bob key"}, asAdmin)
	require.Equal(t, http.StatusCreated, w.Code)
	bobKeyID := rc1JSON(t, w)["id"].(string)

	// Admin list with filters (unparseable UUID filters are ignored).
	w = rc1Do(t, r, http.MethodGet, base+"?limit=1000&user_id="+alice.String()+"&status=active&created_by="+adminID.String(), nil, asAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	list := rc1JSON(t, w)
	assert.EqualValues(t, 100, list["limit"])
	assert.EqualValues(t, 2, list["total"])
	for _, item := range list["data"].([]any) {
		assert.NotContains(t, item.(map[string]any), "key")
	}
	w = rc1Do(t, r, http.MethodGet, base+"?user_id=bad&created_by=bad", nil, asAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	assert.EqualValues(t, 3, rc1JSON(t, w)["total"])

	// Admin get.
	w = rc1Do(t, r, http.MethodGet, base+"/"+keyID, nil, asAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "alice key", rc1JSON(t, w)["name"])
	w = rc1Do(t, r, http.MethodGet, base+"/bad", nil, asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodGet, base+"/"+uuid.NewString(), nil, asAdmin)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/admin/service-keys-noparam", nil, asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// User endpoints: only own keys are visible.
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys", nil, map[string]string{"X-Test-User": "not-a-uuid"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys", nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["service_keys"], 2)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys", nil, asBob)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, rc1JSON(t, w)["service_keys"], 1)

	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys/"+keyID, nil, asAlice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, keyID, rc1JSON(t, w)["id"])
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys/"+keyID, nil, asBob)
	assert.Equal(t, http.StatusForbidden, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys/"+bobKeyID, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys/bad", nil, asAlice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys/"+uuid.NewString(), nil, asAlice)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys-noparam", nil, asAlice)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Revoke.
	w = rc1Do(t, r, http.MethodPost, base+"/"+keyID+"/revoke", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodPost, base+"/bad/revoke", nil, asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodPost, base+"/"+uuid.NewString()+"/revoke", nil, asAdmin)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, r, http.MethodPost, "/api/v1/admin/service-keys-noparam/revoke", nil, asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodPost, base+"/"+keyID+"/revoke", nil, asAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys/"+keyID, nil, asAlice)
	assert.Equal(t, "revoked", rc1JSON(t, w)["status"])

	// Delete.
	w = rc1Do(t, r, http.MethodDelete, base+"/"+keyID, nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = rc1Do(t, r, http.MethodDelete, base+"/bad", nil, asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodDelete, "/api/v1/admin/service-keys-noparam", nil, asAdmin)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Do(t, r, http.MethodDelete, base+"/"+keyID, nil, asAdmin)
	require.Equal(t, http.StatusOK, w.Code)
	w = rc1Do(t, r, http.MethodDelete, base+"/"+keyID, nil, asAdmin)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Do(t, r, http.MethodGet, "/api/v1/service-keys", nil, asAlice)
	assert.Len(t, rc1JSON(t, w)["service_keys"], 1)
}

func TestRC1ServiceKeys_StorageFailures(t *testing.T) {
	t.Parallel()
	r := rc1ServiceKeyRouter(rc1ClosedDB(t))
	u := map[string]string{"X-Test-User": uuid.NewString()}
	id := uuid.NewString()
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/service-keys"},
		{http.MethodGet, "/api/v1/service-keys/" + id},
		{http.MethodPost, "/api/v1/admin/service-keys"},
		{http.MethodGet, "/api/v1/admin/service-keys"},
		{http.MethodGet, "/api/v1/admin/service-keys/" + id},
		{http.MethodDelete, "/api/v1/admin/service-keys/" + id},
		{http.MethodPost, "/api/v1/admin/service-keys/" + id + "/revoke"},
	} {
		w := rc1Do(t, r, tc.method, tc.path, map[string]any{"user_id": uuid.NewString(), "name": "k"}, u)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "%s %s", tc.method, tc.path)
	}
}
