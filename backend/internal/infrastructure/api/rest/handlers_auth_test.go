package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/application/auth"
	"github.com/smilemakc/mbflow/internal/config"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
)

// MockUserRepository implements repository.UserRepository for testing
type MockUserRepository struct {
	users          map[uuid.UUID]*models.UserModel
	sessions       map[string]*models.SessionModel
	roles          map[uuid.UUID]*models.RoleModel
	emailLookup    map[string]uuid.UUID
	usernameLookup map[string]uuid.UUID
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:          make(map[uuid.UUID]*models.UserModel),
		sessions:       make(map[string]*models.SessionModel),
		roles:          make(map[uuid.UUID]*models.RoleModel),
		emailLookup:    make(map[string]uuid.UUID),
		usernameLookup: make(map[string]uuid.UUID),
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.UserModel) error {
	m.users[user.ID] = user
	m.emailLookup[user.Email] = user.ID
	m.usernameLookup[user.Username] = user.ID
	return nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.UserModel) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if user, ok := m.users[id]; ok {
		delete(m.emailLookup, user.Email)
		delete(m.usernameLookup, user.Username)
		delete(m.users, id)
	}
	return nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.UserModel, error) {
	if user, ok := m.users[id]; ok {
		return user, nil
	}
	return nil, nil
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*models.UserModel, error) {
	if id, ok := m.emailLookup[email]; ok {
		return m.users[id], nil
	}
	return nil, nil
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*models.UserModel, error) {
	if id, ok := m.usernameLookup[username]; ok {
		return m.users[id], nil
	}
	return nil, nil
}

func (m *MockUserRepository) FindAll(ctx context.Context, limit, offset int) ([]*models.UserModel, error) {
	var users []*models.UserModel
	for _, u := range m.users {
		users = append(users, u)
	}
	if offset >= len(users) {
		return []*models.UserModel{}, nil
	}
	end := offset + limit
	if end > len(users) {
		end = len(users)
	}
	return users[offset:end], nil
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, exists := m.emailLookup[email]
	return exists, nil
}

func (m *MockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	_, exists := m.usernameLookup[username]
	return exists, nil
}

func (m *MockUserRepository) CreateSession(ctx context.Context, session *models.SessionModel) error {
	m.sessions[session.Token] = session
	return nil
}

func (m *MockUserRepository) FindSessionByToken(ctx context.Context, token string) (*models.SessionModel, error) {
	if session, ok := m.sessions[token]; ok {
		return session, nil
	}
	return nil, nil
}

func (m *MockUserRepository) FindSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.SessionModel, error) {
	for _, s := range m.sessions {
		if s.RefreshToken == refreshToken {
			return s, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) DeleteSession(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *MockUserRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	for token, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, token)
		}
	}
	return nil
}

func (m *MockUserRepository) UpdateSessionActivity(ctx context.Context, token string) error {
	if session, ok := m.sessions[token]; ok {
		session.LastActivityAt = time.Now()
	}
	return nil
}

func (m *MockUserRepository) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockUserRepository) CreateRole(ctx context.Context, role *models.RoleModel) error {
	m.roles[role.ID] = role
	return nil
}

func (m *MockUserRepository) UpdateRole(ctx context.Context, role *models.RoleModel) error {
	m.roles[role.ID] = role
	return nil
}

func (m *MockUserRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	delete(m.roles, id)
	return nil
}

func (m *MockUserRepository) FindRoleByID(ctx context.Context, id uuid.UUID) (*models.RoleModel, error) {
	if role, ok := m.roles[id]; ok {
		return role, nil
	}
	return nil, nil
}

func (m *MockUserRepository) FindRoleByName(ctx context.Context, name string) (*models.RoleModel, error) {
	for _, r := range m.roles {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepository) FindAllRoles(ctx context.Context) ([]*models.RoleModel, error) {
	var roles []*models.RoleModel
	for _, r := range m.roles {
		roles = append(roles, r)
	}
	return roles, nil
}

func (m *MockUserRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID) error {
	return nil
}

func (m *MockUserRepository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	return nil
}

func (m *MockUserRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*models.RoleModel, error) {
	return []*models.RoleModel{}, nil
}

func (m *MockUserRepository) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	return false, nil
}

func (m *MockUserRepository) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return []string{}, nil
}

func (m *MockUserRepository) LogAuditEvent(ctx context.Context, event *models.AuditLogModel) error {
	return nil
}

func (m *MockUserRepository) GetAuditLogs(ctx context.Context, userID *uuid.UUID, action string, limit, offset int) ([]*models.AuditLogModel, error) {
	return []*models.AuditLogModel{}, nil
}

// Additional methods required by UserRepository interface

func (m *MockUserRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	if user, ok := m.users[id]; ok {
		delete(m.emailLookup, user.Email)
		delete(m.usernameLookup, user.Username)
		delete(m.users, id)
	}
	return nil
}

func (m *MockUserRepository) FindByIDWithRoles(ctx context.Context, id uuid.UUID) (*models.UserModel, error) {
	return m.FindByID(ctx, id)
}

func (m *MockUserRepository) FindAllActive(ctx context.Context, limit, offset int) ([]*models.UserModel, error) {
	var users []*models.UserModel
	for _, u := range m.users {
		if u.IsActive {
			users = append(users, u)
		}
	}
	total := len(users)
	if offset >= len(users) {
		return []*models.UserModel{}, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return users[offset:end], nil
}

func (m *MockUserRepository) Count(ctx context.Context) (int, error) {
	return len(m.users), nil
}

func (m *MockUserRepository) CountActive(ctx context.Context) (int, error) {
	count := 0
	for _, u := range m.users {
		if u.IsActive {
			count++
		}
	}
	return count, nil
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	if user, ok := m.users[id]; ok {
		now := time.Now()
		user.LastLoginAt = &now
	}
	return nil
}

func (m *MockUserRepository) IncrementFailedAttempts(ctx context.Context, id uuid.UUID) error {
	if user, ok := m.users[id]; ok {
		user.FailedLoginAttempts++
	}
	return nil
}

func (m *MockUserRepository) ResetFailedAttempts(ctx context.Context, id uuid.UUID) error {
	if user, ok := m.users[id]; ok {
		user.FailedLoginAttempts = 0
	}
	return nil
}

func (m *MockUserRepository) LockAccount(ctx context.Context, id uuid.UUID, until *string) error {
	return nil
}

func (m *MockUserRepository) UnlockAccount(ctx context.Context, id uuid.UUID) error {
	if user, ok := m.users[id]; ok {
		user.LockedUntil = nil
	}
	return nil
}

func (m *MockUserRepository) FindSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.SessionModel, error) {
	var sessions []*models.SessionModel
	for _, s := range m.sessions {
		if s.UserID == userID {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

func (m *MockUserRepository) DeleteSessionByID(ctx context.Context, id uuid.UUID) error {
	for token, s := range m.sessions {
		if s.ID == id {
			delete(m.sessions, token)
			break
		}
	}
	return nil
}

func (m *MockUserRepository) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	return m.DeleteUserSessions(ctx, userID)
}

func (m *MockUserRepository) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockUserRepository) CreateAuditLog(ctx context.Context, log *models.AuditLogModel) error {
	return nil
}

func (m *MockUserRepository) FindAuditLogs(ctx context.Context, userID *uuid.UUID, action string, limit, offset int) ([]*models.AuditLogModel, error) {
	return []*models.AuditLogModel{}, nil
}

// Verify MockUserRepository implements the interface
var _ repository.UserRepository = (*MockUserRepository)(nil)

// Test helpers
func setupAuthTestRouter(authService *auth.Service, pm *auth.ProviderManager) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	authHandlers := NewAuthHandlers(authService, pm, nil)

	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/register", authHandlers.HandleRegister)
		authGroup.POST("/login", authHandlers.HandleLogin)
		authGroup.POST("/refresh", authHandlers.HandleRefresh)
		authGroup.GET("/info", authHandlers.HandleGetAuthInfo)
	}

	return router
}

func setupAuthTestService() (*auth.Service, *MockUserRepository) {
	mockRepo := NewMockUserRepository()

	// Add default user role
	userRole := &models.RoleModel{
		ID:          uuid.New(),
		Name:        "user",
		Description: "Default user role",
		IsSystem:    true,
	}
	mockRepo.CreateRole(context.Background(), userRole)

	cfg := &config.AuthConfig{
		Mode:               "builtin",
		JWTSecret:          "test-secret-key-at-least-32-characters",
		JWTExpirationHours: 24,
		RefreshExpiryDays:  30,
		MinPasswordLength:  8,
		MaxLoginAttempts:   5,
		LockoutDuration:    15 * time.Minute,
		AllowRegistration:  true,
	}

	service := auth.NewService(mockRepo, cfg)
	return service, mockRepo
}

func performAuthRequest(r http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	req, _ := http.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Tests

func TestHandleRegister_Success(t *testing.T) {
	authService, _ := setupAuthTestService()
	router := setupAuthTestRouter(authService, nil)

	req := RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "Password123!",
		FullName: "Test User",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/register", req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var response AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.AccessToken == "" {
		t.Error("expected access token in response")
	}

	if response.TokenType != "Bearer" {
		t.Errorf("expected token type Bearer, got %s", response.TokenType)
	}
}

func TestHandleRegister_DuplicateEmail(t *testing.T) {
	authService, mockRepo := setupAuthTestService()

	// Pre-create user with same email
	existingUser := &models.UserModel{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Username: "existinguser",
	}
	mockRepo.Create(context.Background(), existingUser)

	router := setupAuthTestRouter(authService, nil)

	req := RegisterRequest{
		Email:    "test@example.com",
		Username: "newuser",
		Password: "Password123!",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/register", req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRegister_DuplicateUsername(t *testing.T) {
	authService, mockRepo := setupAuthTestService()

	// Pre-create user with same username
	existingUser := &models.UserModel{
		ID:       uuid.New(),
		Email:    "existing@example.com",
		Username: "testuser",
	}
	mockRepo.Create(context.Background(), existingUser)

	router := setupAuthTestRouter(authService, nil)

	req := RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "Password123!",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/register", req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRegister_InvalidEmail(t *testing.T) {
	authService, _ := setupAuthTestService()
	router := setupAuthTestRouter(authService, nil)

	req := RegisterRequest{
		Email:    "invalid-email",
		Username: "testuser",
		Password: "Password123!",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/register", req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRegister_ShortPassword(t *testing.T) {
	authService, _ := setupAuthTestService()
	router := setupAuthTestRouter(authService, nil)

	req := RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "short",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/register", req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRegister_ShortUsername(t *testing.T) {
	authService, _ := setupAuthTestService()
	router := setupAuthTestRouter(authService, nil)

	req := RegisterRequest{
		Email:    "test@example.com",
		Username: "ab",
		Password: "Password123!",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/register", req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogin_Success(t *testing.T) {
	authService, mockRepo := setupAuthTestService()

	// First register a user
	ctx := context.Background()
	registerReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "Password123!",
	}
	_, err := authService.Register(ctx, registerReq)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	// Verify user exists
	user, _ := mockRepo.FindByEmail(ctx, "test@example.com")
	if user == nil {
		t.Fatal("user should exist after registration")
	}

	router := setupAuthTestRouter(authService, nil)

	loginReq := LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/login", loginReq)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.AccessToken == "" {
		t.Error("expected access token in response")
	}
}

func TestHandleLogin_InvalidEmail(t *testing.T) {
	authService, _ := setupAuthTestService()
	router := setupAuthTestRouter(authService, nil)

	req := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "Password123!",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/login", req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogin_InvalidPassword(t *testing.T) {
	authService, _ := setupAuthTestService()

	// Register a user first
	ctx := context.Background()
	registerReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "Password123!",
	}
	_, _ = authService.Register(ctx, registerReq)

	router := setupAuthTestRouter(authService, nil)

	req := LoginRequest{
		Email:    "test@example.com",
		Password: "WrongPassword!",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/login", req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleLogin_InactiveAccount(t *testing.T) {
	authService, mockRepo := setupAuthTestService()

	// Register and then deactivate user
	ctx := context.Background()
	registerReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "Password123!",
	}
	_, _ = authService.Register(ctx, registerReq)

	// Deactivate user
	user, _ := mockRepo.FindByEmail(ctx, "test@example.com")
	if user != nil {
		user.IsActive = false
		mockRepo.Update(ctx, user)
	}

	router := setupAuthTestRouter(authService, nil)

	req := LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/login", req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRefresh_Success(t *testing.T) {
	authService, _ := setupAuthTestService()

	// Register and login to get tokens
	ctx := context.Background()
	registerReq := &auth.RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "Password123!",
	}
	_, _ = authService.Register(ctx, registerReq)

	loginResult, err := authService.Login(ctx, &auth.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	}, "127.0.0.1", "test-agent")

	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}

	router := setupAuthTestRouter(authService, nil)

	req := RefreshRequest{
		RefreshToken: loginResult.RefreshToken,
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/refresh", req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.AccessToken == "" {
		t.Error("expected new access token in response")
	}
}

func TestHandleRefresh_InvalidToken(t *testing.T) {
	authService, _ := setupAuthTestService()
	router := setupAuthTestRouter(authService, nil)

	req := RefreshRequest{
		RefreshToken: "invalid-refresh-token",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/refresh", req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetAuthInfo(t *testing.T) {
	authService, _ := setupAuthTestService()

	// Create provider manager for testing
	cfg := &config.AuthConfig{
		Mode:      "builtin",
		JWTSecret: "test-secret-key-at-least-32-characters",
	}
	pm, _ := auth.NewProviderManager(cfg, authService)

	router := setupAuthTestRouter(authService, pm)

	w := performAuthRequest(router, "GET", "/api/v1/auth/info", nil)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["mode"] == nil {
		t.Error("expected mode in response")
	}
}

func TestLoginRateLimiter(t *testing.T) {
	cfg := &config.AuthConfig{
		MaxLoginAttempts: 3,
		LockoutDuration:  15 * time.Minute,
	}

	limiter := NewLoginRateLimiter(cfg.MaxLoginAttempts, 5*time.Minute, cfg.LockoutDuration)

	testIP := "192.168.1.1"

	// Should not be blocked initially
	if limiter.IsBlocked(testIP) {
		t.Error("IP should not be blocked initially")
	}

	// Record failed attempts
	limiter.RecordFailedAttempt(testIP)
	limiter.RecordFailedAttempt(testIP)

	// Should still have remaining attempts
	remaining := limiter.GetRemainingAttempts(testIP)
	if remaining != 1 {
		t.Errorf("expected 1 remaining attempt, got %d", remaining)
	}

	// Record third failed attempt - should be blocked now
	limiter.RecordFailedAttempt(testIP)

	if !limiter.IsBlocked(testIP) {
		t.Error("IP should be blocked after max attempts")
	}

	// Successful login should reset
	limiter.RecordSuccessfulLogin(testIP)

	if limiter.IsBlocked(testIP) {
		t.Error("IP should not be blocked after successful login")
	}
}

func TestHandleRegister_RegistrationDisabled(t *testing.T) {
	mockRepo := NewMockUserRepository()

	cfg := &config.AuthConfig{
		Mode:               "builtin",
		JWTSecret:          "test-secret-key-at-least-32-characters",
		JWTExpirationHours: 24,
		AllowRegistration:  false, // Disabled
	}

	authService := auth.NewService(mockRepo, cfg)
	router := setupAuthTestRouter(authService, nil)

	req := RegisterRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "Password123!",
	}

	w := performAuthRequest(router, "POST", "/api/v1/auth/register", req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

// Test helper to verify user domain model conversion
func TestUserModelConversion(t *testing.T) {
	userModel := &models.UserModel{
		ID:        uuid.New(),
		Email:     "test@example.com",
		Username:  "testuser",
		FullName:  "Test User",
		IsActive:  true,
		IsAdmin:   false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	domainUser := models.ToUserDomain(userModel, []string{"user"})

	if domainUser.ID != userModel.ID.String() {
		t.Errorf("ID mismatch: expected %s, got %s", userModel.ID.String(), domainUser.ID)
	}

	if domainUser.Email != userModel.Email {
		t.Errorf("Email mismatch: expected %s, got %s", userModel.Email, domainUser.Email)
	}

	if domainUser.Username != userModel.Username {
		t.Errorf("Username mismatch: expected %s, got %s", userModel.Username, domainUser.Username)
	}

	if len(domainUser.Roles) != 1 || domainUser.Roles[0] != "user" {
		t.Errorf("Roles mismatch: expected [user], got %v", domainUser.Roles)
	}
}

// Benchmark tests
func BenchmarkHandleLogin(b *testing.B) {
	authService, _ := setupAuthTestService()

	// Register a user
	ctx := context.Background()
	registerReq := &auth.RegisterRequest{
		Email:    "bench@example.com",
		Username: "benchuser",
		Password: "Password123!",
	}
	_, _ = authService.Register(ctx, registerReq)

	router := setupAuthTestRouter(authService, nil)

	req := LoginRequest{
		Email:    "bench@example.com",
		Password: "Password123!",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		performAuthRequest(router, "POST", "/api/v1/auth/login", req)
	}
}

// Test to ensure proper validation messages
func TestHandleRegister_ValidationErrors(t *testing.T) {
	authService, _ := setupAuthTestService()
	router := setupAuthTestRouter(authService, nil)

	tests := []struct {
		name           string
		request        interface{}
		expectedStatus int
		description    string
	}{
		{
			name: "missing email",
			request: map[string]string{
				"username": "testuser",
				"password": "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "should fail when email is missing",
		},
		{
			name: "missing username",
			request: map[string]string{
				"email":    "test@example.com",
				"password": "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "should fail when username is missing",
		},
		{
			name: "missing password",
			request: map[string]string{
				"email":    "test@example.com",
				"username": "testuser",
			},
			expectedStatus: http.StatusBadRequest,
			description:    "should fail when password is missing",
		},
		{
			name:           "empty body",
			request:        map[string]string{},
			expectedStatus: http.StatusBadRequest,
			description:    "should fail with empty body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performAuthRequest(router, "POST", "/api/v1/auth/register", tt.request)

			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// Test auth middleware integration
func TestAuthMiddleware_RequireAuth(t *testing.T) {
	authService, _ := setupAuthTestService()

	cfg := &config.AuthConfig{
		Mode:      "builtin",
		JWTSecret: "test-secret-key-at-least-32-characters",
	}
	pm, _ := auth.NewProviderManager(cfg, authService)

	authMiddleware := NewAuthMiddleware(pm, authService)

	router := gin.New()
	router.Use(gin.Recovery())

	// Protected route
	router.GET("/api/v1/protected", authMiddleware.RequireAuth(), func(c *gin.Context) {
		userID, _ := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// Test without token
	t.Run("without token", func(t *testing.T) {
		w := performAuthRequest(router, "GET", "/api/v1/protected", nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	// Test with valid token
	t.Run("with valid token", func(t *testing.T) {
		// Register and login to get token
		ctx := context.Background()
		registerReq := &auth.RegisterRequest{
			Email:    "test@example.com",
			Username: "testuser",
			Password: "Password123!",
		}
		_, _ = authService.Register(ctx, registerReq)

		loginResult, _ := authService.Login(ctx, &auth.LoginRequest{
			Email:    "test@example.com",
			Password: "Password123!",
		}, "127.0.0.1", "test-agent")

		req, _ := http.NewRequest("GET", "/api/v1/protected", nil)
		req.Header.Set("Authorization", "Bearer "+loginResult.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Test with invalid token
	t.Run("with invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

// Test optional auth middleware
func TestAuthMiddleware_OptionalAuth(t *testing.T) {
	authService, _ := setupAuthTestService()

	cfg := &config.AuthConfig{
		Mode:      "builtin",
		JWTSecret: "test-secret-key-at-least-32-characters",
	}
	pm, _ := auth.NewProviderManager(cfg, authService)

	authMiddleware := NewAuthMiddleware(pm, authService)

	router := gin.New()
	router.Use(gin.Recovery())

	// Route with optional auth
	router.GET("/api/v1/optional", authMiddleware.OptionalAuth(), func(c *gin.Context) {
		userID, ok := GetUserID(c)
		if ok {
			c.JSON(http.StatusOK, gin.H{"user_id": userID, "authenticated": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	// Test without token - should still succeed
	t.Run("without token", func(t *testing.T) {
		w := performAuthRequest(router, "GET", "/api/v1/optional", nil)
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["authenticated"] != false {
			t.Error("expected authenticated to be false")
		}
	})

	// Test with valid token
	t.Run("with valid token", func(t *testing.T) {
		ctx := context.Background()
		registerReq := &auth.RegisterRequest{
			Email:    "optional@example.com",
			Username: "optionaluser",
			Password: "Password123!",
		}
		_, _ = authService.Register(ctx, registerReq)

		loginResult, _ := authService.Login(ctx, &auth.LoginRequest{
			Email:    "optional@example.com",
			Password: "Password123!",
		}, "127.0.0.1", "test-agent")

		req, _ := http.NewRequest("GET", "/api/v1/optional", nil)
		req.Header.Set("Authorization", "Bearer "+loginResult.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["authenticated"] != true {
			t.Error("expected authenticated to be true")
		}
	})
}

// Test admin-only middleware
func TestAuthMiddleware_RequireAdmin(t *testing.T) {
	authService, mockRepo := setupAuthTestService()

	cfg := &config.AuthConfig{
		Mode:      "builtin",
		JWTSecret: "test-secret-key-at-least-32-characters",
	}
	pm, _ := auth.NewProviderManager(cfg, authService)

	authMiddleware := NewAuthMiddleware(pm, authService)

	router := gin.New()
	router.Use(gin.Recovery())

	// Admin route
	router.GET("/api/v1/admin", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"admin": true})
	})

	// Register a regular user
	ctx := context.Background()
	registerReq := &auth.RegisterRequest{
		Email:    "user@example.com",
		Username: "regularuser",
		Password: "Password123!",
	}
	_, _ = authService.Register(ctx, registerReq)

	// Register an admin user
	adminReq := &auth.RegisterRequest{
		Email:    "admin@example.com",
		Username: "adminuser",
		Password: "Password123!",
	}
	_, _ = authService.Register(ctx, adminReq)

	// Make admin user an admin
	adminUser, _ := mockRepo.FindByEmail(ctx, "admin@example.com")
	if adminUser != nil {
		adminUser.IsAdmin = true
		mockRepo.Update(ctx, adminUser)
	}

	// Test with regular user token
	t.Run("regular user denied", func(t *testing.T) {
		loginResult, _ := authService.Login(ctx, &auth.LoginRequest{
			Email:    "user@example.com",
			Password: "Password123!",
		}, "127.0.0.1", "test-agent")

		req, _ := http.NewRequest("GET", "/api/v1/admin", nil)
		req.Header.Set("Authorization", "Bearer "+loginResult.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
	})

	// Test with admin user token
	t.Run("admin user allowed", func(t *testing.T) {
		loginResult, _ := authService.Login(ctx, &auth.LoginRequest{
			Email:    "admin@example.com",
			Password: "Password123!",
		}, "127.0.0.1", "test-agent")

		req, _ := http.NewRequest("GET", "/api/v1/admin", nil)
		req.Header.Set("Authorization", "Bearer "+loginResult.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}
