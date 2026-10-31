package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/auth"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

const rc2JWTSecret = "rc2-test-secret-key-at-least-32-characters"

// rc2UserRepo wraps the in-memory user repository with error injection and
// configurable permissions.
type rc2UserRepo struct {
	*MockUserRepository
	fail  map[string]error
	perms map[string]bool
}

func (r *rc2UserRepo) FindAll(ctx context.Context, limit, offset int) ([]*models.UserModel, error) {
	if err := r.fail["FindAll"]; err != nil {
		return nil, err
	}
	return r.MockUserRepository.FindAll(ctx, limit, offset)
}

func (r *rc2UserRepo) FindAllRoles(ctx context.Context) ([]*models.RoleModel, error) {
	if err := r.fail["FindAllRoles"]; err != nil {
		return nil, err
	}
	return r.MockUserRepository.FindAllRoles(ctx)
}

func (r *rc2UserRepo) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if err := r.fail["RemoveRole"]; err != nil {
		return err
	}
	return r.MockUserRepository.RemoveRole(ctx, userID, roleID)
}

func (r *rc2UserRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*models.RoleModel, error) {
	if err := r.fail["GetUserRoles"]; err != nil {
		return nil, err
	}
	return r.MockUserRepository.GetUserRoles(ctx, userID)
}

func (r *rc2UserRepo) DeleteSession(ctx context.Context, token string) error {
	if err := r.fail["DeleteSession"]; err != nil {
		return err
	}
	return r.MockUserRepository.DeleteSession(ctx, token)
}

func (r *rc2UserRepo) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	if err := r.fail["HasPermission"]; err != nil {
		return false, err
	}
	return r.perms[userID.String()+":"+permission], nil
}

type rc2AuthEnv struct {
	cfg    *config.AuthConfig
	repo   *rc2UserRepo
	svc    *auth.Service
	pm     *auth.ProviderManager
	mw     *AuthMiddleware
	h      *AuthHandlers
	rl     *LoginRateLimiter
	router *gin.Engine
}

func rc2AuthConfig() *config.AuthConfig {
	return &config.AuthConfig{
		Mode:               "builtin",
		JWTSecret:          rc2JWTSecret,
		JWTExpirationHours: 1,
		RefreshExpiryDays:  1,
		MinPasswordLength:  8,
		MaxLoginAttempts:   100,
		LockoutDuration:    time.Minute,
		AllowRegistration:  true,
	}
}

func rc2NewAuthEnv(t *testing.T, mutate func(*config.AuthConfig)) *rc2AuthEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := &rc2UserRepo{MockUserRepository: NewMockUserRepository(), fail: map[string]error{}, perms: map[string]bool{}}
	_ = repo.CreateRole(context.Background(), &models.RoleModel{ID: uuid.New(), Name: "user", IsSystem: true})

	cfg := rc2AuthConfig()
	if mutate != nil {
		mutate(cfg)
	}
	svc := auth.NewService(repo, nil, cfg)
	pm, err := auth.NewProviderManager(cfg, svc)
	require.NoError(t, err)

	env := &rc2AuthEnv{
		cfg:  cfg,
		repo: repo,
		svc:  svc,
		pm:   pm,
		mw:   NewAuthMiddleware(pm, svc, nil),
		rl:   NewLoginRateLimiter(2, time.Hour, time.Hour),
	}
	env.h = NewAuthHandlers(svc, pm, env.rl)

	r := gin.New()
	a := r.Group("/api/v1/auth")
	a.POST("/register", env.h.HandleRegister)
	a.POST("/login", env.h.HandleLogin)
	a.POST("/refresh", env.h.HandleRefresh)
	a.GET("/info", env.h.HandleGetAuthInfo)
	a.GET("/oauth/authorize", env.h.HandleOAuthAuthorize)
	a.GET("/oauth/callback", env.h.HandleOAuthCallback)
	a.POST("/logout", env.mw.RequireAuth(), env.h.HandleLogout)
	a.GET("/me", env.mw.RequireAuth(), env.h.HandleGetMe)
	a.POST("/password", env.mw.RequireAuth(), env.h.HandleChangePassword)
	// Without authentication middleware (handlers must reject).
	a.POST("/logout-noauth", env.h.HandleLogout)
	a.GET("/me-noauth", env.h.HandleGetMe)
	a.POST("/password-noauth", env.h.HandleChangePassword)

	admin := r.Group("/api/v1/admin", env.mw.RequireAdmin())
	admin.GET("/users", env.h.HandleAdminListUsers)
	admin.POST("/users", env.h.HandleAdminCreateUser)
	admin.GET("/users/:id", env.h.HandleAdminGetUser)
	admin.PUT("/users/:id", env.h.HandleAdminUpdateUser)
	admin.DELETE("/users/:id", env.h.HandleAdminDeleteUser)
	admin.POST("/users/:id/reset-password", env.h.HandleAdminResetPassword)
	admin.GET("/roles", env.h.HandleListRoles)
	admin.GET("/users/:id/roles", env.h.HandleGetUserRoles)
	admin.POST("/users/:id/roles", env.h.HandleAssignRole)
	admin.DELETE("/users/:id/roles/:role_id", env.h.HandleRemoveRole)
	// Routes lacking the path parameters the handlers expect.
	np := admin.Group("/noparam")
	np.GET("/get", env.h.HandleAdminGetUser)
	np.PUT("/update", env.h.HandleAdminUpdateUser)
	np.DELETE("/delete", env.h.HandleAdminDeleteUser)
	np.POST("/reset", env.h.HandleAdminResetPassword)
	np.GET("/roles", env.h.HandleGetUserRoles)
	np.POST("/roles", env.h.HandleAssignRole)
	np.DELETE("/roles", env.h.HandleRemoveRole)
	np.DELETE("/users/:id/roles", env.h.HandleRemoveRole)

	env.router = r
	return env
}

func (e *rc2AuthEnv) do(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	switch b := body.(type) {
	case nil:
		reader = strings.NewReader("")
	case string:
		reader = strings.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		require.NoError(t, err)
		reader = strings.NewReader(string(raw))
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.1:1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

// login creates (if needed) and logs in a user, returning the access token.
func (e *rc2AuthEnv) login(t *testing.T, email string, admin bool) (string, string) {
	t.Helper()
	ctx := context.Background()
	if exists, _ := e.repo.ExistsByEmail(ctx, email); !exists {
		_, err := e.svc.CreateUser(ctx, &auth.RegisterRequest{Email: email, Password: "Password123!"}, admin)
		require.NoError(t, err)
	}
	res, err := e.svc.Login(ctx, &auth.LoginRequest{Email: email, Password: "Password123!"}, "127.0.0.1", "ua")
	require.NoError(t, err)
	return res.AccessToken, res.User.ID
}

func (e *rc2AuthEnv) token(t *testing.T, user *pkgmodels.User, mutate func(*config.AuthConfig)) string {
	t.Helper()
	cfg := *e.cfg
	if mutate != nil {
		mutate(&cfg)
	}
	tok, _, err := auth.NewJWTService(&cfg).GenerateAccessToken(user)
	require.NoError(t, err)
	return tok
}

func TestRC2_Auth_LoginRateLimitingAndProviderPaths(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	_, _ = env.login(t, "alice@example.com", false)

	// Invalid JSON / validation.
	w := env.do(t, http.MethodPost, "/api/v1/auth/login", "", `{"email":"x"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Wrong password: remaining attempts are reported.
	w = env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "alice@example.com", "password": "wrong-password"})
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	details := rc2DecodeMap(t, w)["details"].(map[string]any)
	assert.Equal(t, float64(1), details["remaining_attempts"])

	// Second failure locks this (IP, account) pair.
	w = env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "alice@example.com", "password": "wrong-password"})
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, float64(0), rc2DecodeMap(t, w)["details"].(map[string]any)["remaining_attempts"])

	w = env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "alice@example.com", "password": "Password123!"})
	require.Equal(t, http.StatusTooManyRequests, w.Code, "locked out even with the right password")
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", rc2DecodeMap(t, w)["code"])

	// Another account from the same IP is unaffected; success resets it.
	_, _ = env.login(t, "bob@example.com", false)
	key := LoginAttemptKey("192.0.2.1", "bob@example.com")
	env.rl.RecordFailedAttempt(key)
	w = env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "bob@example.com", "password": "Password123!"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.NotEmpty(t, rc2DecodeMap(t, w)["access_token"])
	assert.Equal(t, 2, env.rl.GetRemainingAttempts(key))

	// Non-credential errors (here: token generation fails while loading
	// roles) are reported without remaining attempts.
	env.repo.fail["GetUserRoles"] = assert.AnError
	w = env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "bob@example.com", "password": "Password123!"})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "remaining_attempts")
	delete(env.repo.fail, "GetUserRoles")
}

func TestRC2_Auth_LoginWithoutRateLimiter(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	env.h.rateLimiter = nil
	_, _ = env.login(t, "carol@example.com", false)

	w := env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "carol@example.com", "password": "bad-password"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NotContains(t, w.Body.String(), "remaining_attempts")
	w = env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "carol@example.com", "password": "Password123!"})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRC2_Auth_SessionEndpoints(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	token, userID := env.login(t, "dave@example.com", false)

	// Missing auth context.
	for _, p := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/auth/logout-noauth"},
		{http.MethodGet, "/api/v1/auth/me-noauth"},
		{http.MethodPost, "/api/v1/auth/password-noauth"},
	} {
		w := env.do(t, p.method, p.path, "", `{}`)
		assert.Equal(t, http.StatusUnauthorized, w.Code, p.path)
	}

	// /me returns the current user.
	w := env.do(t, http.MethodGet, "/api/v1/auth/me", token, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	user := rc2DecodeMap(t, w)["user"].(map[string]any)
	assert.Equal(t, userID, user["id"])
	assert.Equal(t, "dave@example.com", user["email"])

	// Change password: validation, wrong old password, success.
	w = env.do(t, http.MethodPost, "/api/v1/auth/password", token, map[string]string{"old_password": "x"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodPost, "/api/v1/auth/password", token, map[string]string{"old_password": "not-the-password", "new_password": "NewPassword123!"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = env.do(t, http.MethodPost, "/api/v1/auth/password", token, map[string]string{"old_password": "Password123!", "new_password": "NewPassword123!"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	_, err := env.svc.Login(context.Background(), &auth.LoginRequest{Email: "dave@example.com", Password: "NewPassword123!"}, "", "")
	assert.NoError(t, err, "new password works")

	// Logout failure and success.
	env.repo.fail["DeleteSession"] = assert.AnError
	w = env.do(t, http.MethodPost, "/api/v1/auth/logout", token, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	delete(env.repo.fail, "DeleteSession")
	w = env.do(t, http.MethodPost, "/api/v1/auth/logout", token, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "logged out successfully", rc2DecodeMap(t, w)["message"])

	// A valid token for a user that no longer exists.
	ghost := env.token(t, &pkgmodels.User{ID: uuid.NewString(), Email: "ghost@example.com"}, nil)
	w = env.do(t, http.MethodGet, "/api/v1/auth/me", ghost, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRC2_Auth_AdminUserManagement(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	adminToken, adminID := env.login(t, "admin@example.com", true)
	userToken, userID := env.login(t, "user@example.com", false)

	// Non-admins are rejected by RequireAdmin.
	w := env.do(t, http.MethodGet, "/api/v1/admin/users", userToken, nil)
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = env.do(t, http.MethodGet, "/api/v1/admin/users?limit=10&offset=0", adminToken, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	list := rc2DecodeMap(t, w)
	assert.Equal(t, float64(2), list["total"])
	assert.Len(t, list["data"], 2)

	env.repo.fail["FindAll"] = assert.AnError
	w = env.do(t, http.MethodGet, "/api/v1/admin/users", adminToken, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	delete(env.repo.fail, "FindAll")

	// Create.
	w = env.do(t, http.MethodPost, "/api/v1/admin/users", adminToken, map[string]any{"email": "new@example.com", "password": "short"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodPost, "/api/v1/admin/users", adminToken, map[string]any{"email": "new@example.com", "password": "Password123!", "full_name": "New", "is_admin": true})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	created := rc2DecodeMap(t, w)["user"].(map[string]any)
	assert.Equal(t, true, created["is_admin"])
	newID := created["id"].(string)
	w = env.do(t, http.MethodPost, "/api/v1/admin/users", adminToken, map[string]any{"email": "new@example.com", "password": "Password123!"})
	assert.Equal(t, http.StatusConflict, w.Code)

	// Get.
	w = env.do(t, http.MethodGet, "/api/v1/admin/users/"+newID, adminToken, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "new@example.com", rc2DecodeMap(t, w)["user"].(map[string]any)["email"])
	w = env.do(t, http.MethodGet, "/api/v1/admin/users/not-a-uuid", adminToken, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodGet, "/api/v1/admin/users/"+uuid.NewString(), adminToken, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Update.
	w = env.do(t, http.MethodPut, "/api/v1/admin/users/bad", adminToken, map[string]any{"email": "x@example.com"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodPut, "/api/v1/admin/users/"+newID, adminToken, map[string]any{"email": "nope"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodPut, "/api/v1/admin/users/"+newID, adminToken, map[string]any{"email": "user@example.com"})
	assert.Equal(t, http.StatusConflict, w.Code, "email already taken")
	w = env.do(t, http.MethodPut, "/api/v1/admin/users/"+newID, adminToken, map[string]any{"email": "renamed@example.com", "full_name": "Renamed", "is_active": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	updated := rc2DecodeMap(t, w)["user"].(map[string]any)
	assert.Equal(t, "renamed@example.com", updated["email"])
	assert.Equal(t, false, updated["is_admin"])

	// Reset password.
	w = env.do(t, http.MethodPost, "/api/v1/admin/users/bad/reset-password", adminToken, map[string]any{"new_password": "Password999!"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodPost, "/api/v1/admin/users/"+newID+"/reset-password", adminToken, map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodPost, "/api/v1/admin/users/"+uuid.NewString()+"/reset-password", adminToken, map[string]any{"new_password": "Password999!"})
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = env.do(t, http.MethodPost, "/api/v1/admin/users/"+newID+"/reset-password", adminToken, map[string]any{"new_password": "Password999!"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Delete: self-deletion is refused.
	w = env.do(t, http.MethodDelete, "/api/v1/admin/users/"+adminID, adminToken, nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "SELF_DELETION_FORBIDDEN", rc2DecodeMap(t, w)["code"])
	w = env.do(t, http.MethodDelete, "/api/v1/admin/users/bad", adminToken, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodDelete, "/api/v1/admin/users/"+uuid.NewString(), adminToken, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = env.do(t, http.MethodDelete, "/api/v1/admin/users/"+userID, adminToken, nil)
	require.Equal(t, http.StatusOK, w.Code)

	// Missing path parameters.
	for _, p := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/admin/noparam/get"},
		{http.MethodPut, "/api/v1/admin/noparam/update"},
		{http.MethodDelete, "/api/v1/admin/noparam/delete"},
		{http.MethodPost, "/api/v1/admin/noparam/reset"},
		{http.MethodGet, "/api/v1/admin/noparam/roles"},
		{http.MethodPost, "/api/v1/admin/noparam/roles"},
		{http.MethodDelete, "/api/v1/admin/noparam/roles"},
		{http.MethodDelete, "/api/v1/admin/noparam/users/" + newID + "/roles"},
	} {
		w = env.do(t, p.method, p.path, adminToken, `{}`)
		require.Equal(t, http.StatusBadRequest, w.Code, p.path)
		assert.Equal(t, "MISSING_PARAMETER", rc2DecodeMap(t, w)["code"], p.path)
	}
}

func TestRC2_Auth_AdminRoleManagement(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	adminToken, _ := env.login(t, "admin@example.com", true)
	_, userID := env.login(t, "user@example.com", false)

	w := env.do(t, http.MethodGet, "/api/v1/admin/roles", adminToken, nil)
	require.Equal(t, http.StatusOK, w.Code)
	roles := rc2DecodeMap(t, w)["roles"].([]any)
	require.Len(t, roles, 1)
	roleID := roles[0].(map[string]any)["id"].(string)

	env.repo.fail["FindAllRoles"] = assert.AnError
	w = env.do(t, http.MethodGet, "/api/v1/admin/roles", adminToken, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	delete(env.repo.fail, "FindAllRoles")

	base := "/api/v1/admin/users/" + userID + "/roles"

	// Assign.
	w = env.do(t, http.MethodPost, "/api/v1/admin/users/bad/roles", adminToken, map[string]any{"role_id": roleID})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodPost, base, adminToken, map[string]any{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodPost, base, adminToken, map[string]any{"role_id": "nope"})
	assert.Equal(t, "INVALID_ROLE_ID", rc2DecodeMap(t, w)["code"])
	w = env.do(t, http.MethodPost, base, adminToken, map[string]any{"role_id": uuid.NewString()})
	assert.Equal(t, http.StatusNotFound, w.Code, "unknown role")
	w = env.do(t, http.MethodPost, base, adminToken, map[string]any{"role_id": roleID})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// List user roles.
	w = env.do(t, http.MethodGet, base, adminToken, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, rc2DecodeMap(t, w), "roles")
	w = env.do(t, http.MethodGet, "/api/v1/admin/users/bad/roles", adminToken, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	env.repo.fail["GetUserRoles"] = assert.AnError
	w = env.do(t, http.MethodGet, base, adminToken, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	delete(env.repo.fail, "GetUserRoles")

	// Remove.
	w = env.do(t, http.MethodDelete, "/api/v1/admin/users/bad/roles/"+roleID, adminToken, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodDelete, base+"/bad", adminToken, nil)
	assert.Equal(t, "INVALID_ROLE_ID", rc2DecodeMap(t, w)["code"])
	env.repo.fail["RemoveRole"] = assert.AnError
	w = env.do(t, http.MethodDelete, base+"/"+roleID, adminToken, nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	delete(env.repo.fail, "RemoveRole")
	w = env.do(t, http.MethodDelete, base+"/"+roleID, adminToken, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "role removed successfully", rc2DecodeMap(t, w)["message"])
}

func TestRC2_AuthMiddleware_RoleAndPermission(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	gin.SetMode(gin.TestMode)

	r := gin.New()
	ok := func(c *gin.Context) {
		uid, _ := GetUserID(c)
		claims, hasClaims := GetClaims(c)
		tok, hasTok := GetToken(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":    uid,
			"admin":      IsAdmin(c),
			"has_claims": hasClaims && claims.UserID == uid,
			"has_token":  hasTok && tok != "",
			"method":     GetAuthMethod(c),
		})
	}
	r.GET("/role", env.mw.RequireRole("editor", "publisher"), ok)
	r.GET("/perm", env.mw.RequirePermission("workflows:write"), ok)

	editorID := uuid.NewString()
	editor := env.token(t, &pkgmodels.User{ID: editorID, Roles: []string{"Viewer", "EDITOR"}}, nil)
	viewer := env.token(t, &pkgmodels.User{ID: uuid.NewString(), Roles: []string{"viewer"}}, nil)
	admin := env.token(t, &pkgmodels.User{ID: uuid.NewString(), IsAdmin: true}, nil)
	badUser := env.token(t, &pkgmodels.User{ID: "not-a-uuid"}, nil)
	expired := env.token(t, &pkgmodels.User{ID: editorID}, func(c *config.AuthConfig) { c.JWTExpirationHours = -1 })

	get := func(path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	for _, path := range []string{"/role", "/perm"} {
		assert.Equal(t, http.StatusUnauthorized, get(path, "").Code, path)
		assert.Equal(t, http.StatusUnauthorized, get(path, "garbage").Code, path)
		assert.Equal(t, http.StatusUnauthorized, get(path, expired).Code, path)
		w := get(path, admin)
		require.Equal(t, http.StatusOK, w.Code, path)
		assert.Equal(t, true, rc2DecodeMap(t, w)["admin"], "admins bypass %s", path)
	}

	w := get("/role", editor)
	require.Equal(t, http.StatusOK, w.Code, "role match is case-insensitive")
	body := rc2DecodeMap(t, w)
	assert.Equal(t, editorID, body["user_id"])
	assert.Equal(t, true, body["has_claims"])
	assert.Equal(t, true, body["has_token"])
	assert.Equal(t, "jwt", body["method"])
	assert.Equal(t, http.StatusForbidden, get("/role", viewer).Code)

	// Permissions come from the auth service.
	assert.Equal(t, http.StatusUnauthorized, get("/perm", badUser).Code)
	assert.Equal(t, http.StatusForbidden, get("/perm", editor).Code)
	env.repo.perms[editorID+":workflows:write"] = true
	assert.Equal(t, http.StatusOK, get("/perm", editor).Code)
	env.repo.fail["HasPermission"] = assert.AnError
	assert.Equal(t, http.StatusForbidden, get("/perm", editor).Code, "permission lookup errors deny access")
}

func TestRC2_AuthMiddleware_TokenSources(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	r := gin.New()
	r.GET("/p", env.mw.RequireAuth(), func(c *gin.Context) {
		uid, _ := GetUserID(c)
		c.String(http.StatusOK, uid)
	})
	r.GET("/admin", env.mw.RequireAdmin(), func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/opt", env.mw.OptionalAuth(), func(c *gin.Context) {
		uid, _ := GetUserID(c)
		c.String(http.StatusOK, uid)
	})

	uid := uuid.NewString()
	tok := env.token(t, &pkgmodels.User{ID: uid}, nil)
	expired := env.token(t, &pkgmodels.User{ID: uid}, func(c *config.AuthConfig) { c.JWTExpirationHours = -1 })

	// Cookie.
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: tok})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, uid, w.Body.String())

	// Query parameter (WebSocket).
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/p?token="+url.QueryEscape(tok), nil))
	assert.Equal(t, uid, w.Body.String())

	// Non-bearer Authorization header is ignored.
	req = httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Basic abc")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Expired JWT.
	req = httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+expired)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "token expired", rc2DecodeMap(t, w)["message"])

	// An sk_ token without a service key service is validated as a JWT.
	req = httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("X-Service-Key", "sk_whatever")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, "invalid token", rc2DecodeMap(t, w)["message"])

	// RequireAdmin without token / with invalid token.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin?token=bad", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// OptionalAuth with an invalid token continues anonymously.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/opt?token=bad", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", w.Body.String())
}

func TestRC2_AuthContextGetters_Defaults(t *testing.T) {
	t.Parallel()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	_, ok := GetClaims(c)
	assert.False(t, ok)
	_, ok = GetToken(c)
	assert.False(t, ok)
	_, ok = GetUser(c)
	assert.False(t, ok)
	_, ok = GetServiceKeyID(c)
	assert.False(t, ok)
	assert.Equal(t, "jwt", GetAuthMethod(c))
	assert.False(t, IsServiceKeyAuth(c))
	_, ok = GetUserIDAsUUID(c)
	assert.False(t, ok)

	u := &pkgmodels.User{ID: "u"}
	c.Set(ContextKeyUser, u)
	c.Set(ContextKeyAuthMethod, "service_key")
	c.Set(ContextKeyServiceKeyID, "sk-1")
	c.Set(ContextKeyUserID, "not-a-uuid")
	got, ok := GetUser(c)
	assert.True(t, ok)
	assert.Same(t, u, got)
	assert.True(t, IsServiceKeyAuth(c))
	id, ok := GetServiceKeyID(c)
	assert.True(t, ok)
	assert.Equal(t, "sk-1", id)
	_, ok = GetUserIDAsUUID(c)
	assert.False(t, ok, "malformed user IDs are rejected")
}

// rc2OIDCServer serves a minimal OIDC discovery document whose token
// endpoint always rejects the request.
func rc2OIDCServer(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"jwks_uri":%q,"userinfo_endpoint":%q,"id_token_signing_alg_values_supported":["RS256"]}`,
				srv.URL, srv.URL+"/authorize", srv.URL+"/token", srv.URL+"/jwks", srv.URL+"/userinfo")
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRC2_Auth_OAuthFlow(t *testing.T) {
	// Without a gateway OAuth is unavailable.
	plain := rc2NewAuthEnv(t, nil)
	w := plain.do(t, http.MethodGet, "/api/v1/auth/oauth/authorize", "", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "OAUTH_NOT_AVAILABLE", rc2DecodeMap(t, w)["code"])

	srv := rc2OIDCServer(t)
	env := rc2NewAuthEnv(t, func(c *config.AuthConfig) {
		c.Mode = "gateway"
		c.IssuerURL = srv.URL
		c.ClientID = "rc2-client"
		c.ClientSecret = "rc2-secret"
		c.RedirectURL = "http://localhost/callback"
	})
	require.True(t, env.pm.IsGatewayAvailable())

	w = env.do(t, http.MethodGet, "/api/v1/auth/oauth/authorize", "", nil)
	require.Equal(t, http.StatusTemporaryRedirect, w.Code, w.Body.String())
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/authorize", loc.Scheme+"://"+loc.Host+loc.Path)
	state := loc.Query().Get("state")
	require.NotEmpty(t, state)
	assert.NotEmpty(t, loc.Query().Get("nonce"))
	var stateCookie *http.Cookie
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "oauth_state" {
			stateCookie = ck
		}
	}
	require.NotNil(t, stateCookie)
	cookieState, err := url.QueryUnescape(stateCookie.Value)
	require.NoError(t, err)
	assert.Equal(t, state, cookieState)
	assert.True(t, stateCookie.HttpOnly)

	callback := func(query string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/callback?"+query, nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		return rec
	}

	w = callback("error=access_denied&error_description=nope", nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "OAUTH_ERROR", rc2DecodeMap(t, w)["code"])

	w = callback("code=abc", nil)
	assert.Equal(t, "MISSING_OAUTH_PARAMS", rc2DecodeMap(t, w)["code"])

	w = callback("code=abc&state="+state, nil)
	assert.Equal(t, "INVALID_STATE", rc2DecodeMap(t, w)["code"], "state cookie is required")
	w = callback("code=abc&state=other", stateCookie)
	assert.Equal(t, "INVALID_STATE", rc2DecodeMap(t, w)["code"], "state must match the cookie")

	// Valid state: the token exchange with the provider fails.
	w = callback("code=abc&state="+state, stateCookie)
	assert.GreaterOrEqual(t, w.Code, 400)
	assert.NotEqual(t, http.StatusOK, w.Code)

	// In gateway mode logins go through the provider manager; the gateway
	// rejects the password grant.
	w = env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "x@example.com", "password": "Password123!"})
	assert.GreaterOrEqual(t, w.Code, 400)

	w = env.do(t, http.MethodGet, "/api/v1/auth/info", "", nil)
	require.Equal(t, http.StatusOK, w.Code)
	info := rc2DecodeMap(t, w)
	assert.Equal(t, true, info["gateway_available"])
	assert.Equal(t, "gateway", info["mode"])
}

func TestRC2_Auth_HybridLoginFallsBackToBuiltin(t *testing.T) {
	srv := rc2OIDCServer(t)
	env := rc2NewAuthEnv(t, func(c *config.AuthConfig) {
		c.Mode = "hybrid"
		c.IssuerURL = srv.URL
		c.ClientID = "rc2-client"
	})
	_, _ = env.login(t, "erin@example.com", false)

	w := env.do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "erin@example.com", "password": "Password123!"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := rc2DecodeMap(t, w)
	assert.Equal(t, "Bearer", body["token_type"])
	assert.NotEmpty(t, body["access_token"])
}
