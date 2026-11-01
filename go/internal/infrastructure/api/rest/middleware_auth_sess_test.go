package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/auth"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
)

// sessFailingSessionRepo makes the session lookup fail.
type sessFailingSessionRepo struct{ *rc2UserRepo }

func (r *sessFailingSessionRepo) FindSessionByToken(context.Context, string) (*models.SessionModel, error) {
	return nil, assert.AnError
}

func sessMiddlewareRouter(env *rc2AuthEnv) *gin.Engine {
	r := gin.New()
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/auth", env.mw.RequireAuth(), ok)
	r.GET("/admin", env.mw.RequireAdmin(), ok)
	r.GET("/role", env.mw.RequireRole("user"), ok)
	r.GET("/perm", env.mw.RequirePermission("x"), ok)
	r.GET("/opt", env.mw.OptionalAuth(), func(c *gin.Context) {
		uid, _ := GetUserID(c)
		c.String(http.StatusOK, uid)
	})
	return r
}

func sessGet(r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func sessErrorMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	msg, _ := body["message"].(string)
	return msg
}

func TestSess_AuthMiddleware_RejectsRevokedSessions(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	r := sessMiddlewareRouter(env)
	adminToken, _ := env.login(t, "root@example.com", true)

	for _, path := range []string{"/auth", "/admin", "/role", "/perm"} {
		require.Equal(t, http.StatusOK, sessGet(r, path, adminToken).Code, path)
	}
	require.NoError(t, env.svc.Logout(context.Background(), adminToken))

	for _, path := range []string{"/auth", "/admin", "/role", "/perm"} {
		w := sessGet(r, path, adminToken)
		assert.Equal(t, http.StatusUnauthorized, w.Code, path)
		assert.Equal(t, "session revoked", sessErrorMessage(t, w), path)
	}
	w := sessGet(r, "/opt", adminToken)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Body.String(), "optional auth ignores a revoked token")
}

func TestSess_AuthMiddleware_SessionStoreOutageIs503(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	token, _ := env.login(t, "outage@example.com", true)

	// Same configuration and users, but the session lookup fails.
	svc := auth.NewService(&sessFailingSessionRepo{rc2UserRepo: env.repo}, nil, env.cfg)
	pm, err := auth.NewProviderManager(env.cfg, svc)
	require.NoError(t, err)
	r := sessMiddlewareRouter(&rc2AuthEnv{mw: NewAuthMiddleware(pm, svc, nil)})

	for _, path := range []string{"/auth", "/admin", "/role", "/perm"} {
		w := sessGet(r, path, token)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code, path)
	}
}

func TestSess_AdminDeactivationRevokesUserTokens(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	r := sessMiddlewareRouter(env)
	adminToken, _ := env.login(t, "boss@example.com", true)
	userToken, userID := env.login(t, "worker@example.com", false)
	require.Equal(t, http.StatusOK, sessGet(r, "/auth", userToken).Code)

	w := env.do(t, http.MethodPut, "/api/v1/admin/users/"+userID, adminToken, map[string]any{
		"email": "worker@example.com", "full_name": "Worker", "is_active": false, "is_admin": false,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = sessGet(r, "/auth", userToken)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "session revoked", sessErrorMessage(t, w))
	assert.Equal(t, http.StatusOK, sessGet(r, "/auth", adminToken).Code, "other users are unaffected")
}
