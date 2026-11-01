package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

var errRCRepo = errors.New("repo failure")

// rcUserRepo is an in-memory repository.UserRepository with error injection.
type rcUserRepo struct {
	mu        sync.Mutex
	users     map[uuid.UUID]*models.UserModel
	sessions  map[uuid.UUID]*models.SessionModel
	roles     map[uuid.UUID]*models.RoleModel
	userRoles map[uuid.UUID][]uuid.UUID
	audit     []*models.AuditLogModel
	perms     map[string]bool

	failed  map[uuid.UUID]int
	locked  map[uuid.UUID]string
	deleted map[uuid.UUID]bool

	errs map[string]error
	// existsOverride forces ExistsByEmail result when set.
	existsOverride *bool
}

func rcNewUserRepo() *rcUserRepo {
	return &rcUserRepo{
		users:     map[uuid.UUID]*models.UserModel{},
		sessions:  map[uuid.UUID]*models.SessionModel{},
		roles:     map[uuid.UUID]*models.RoleModel{},
		userRoles: map[uuid.UUID][]uuid.UUID{},
		perms:     map[string]bool{},
		failed:    map[uuid.UUID]int{},
		locked:    map[uuid.UUID]string{},
		deleted:   map[uuid.UUID]bool{},
		errs:      map[string]error{},
	}
}

func (r *rcUserRepo) err(op string) error { return r.errs[op] }

func (r *rcUserRepo) Create(ctx context.Context, user *models.UserModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Create"); err != nil {
		return err
	}
	r.users[user.ID] = user
	return nil
}

func (r *rcUserRepo) Update(ctx context.Context, user *models.UserModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Update"); err != nil {
		return err
	}
	r.users[user.ID] = user
	return nil
}

func (r *rcUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Delete"); err != nil {
		return err
	}
	r.deleted[id] = true
	return nil
}

func (r *rcUserRepo) HardDelete(ctx context.Context, id uuid.UUID) error {
	delete(r.users, id)
	return nil
}

func (r *rcUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.UserModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("FindByID"); err != nil {
		return nil, err
	}
	return r.users[id], nil
}

func (r *rcUserRepo) FindByEmail(ctx context.Context, email string) (*models.UserModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("FindByEmail"); err != nil {
		return nil, err
	}
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (r *rcUserRepo) FindByUsername(ctx context.Context, username string) (*models.UserModel, error) {
	for _, u := range r.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, nil
}

func (r *rcUserRepo) FindByIDWithRoles(ctx context.Context, id uuid.UUID) (*models.UserModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("FindByIDWithRoles"); err != nil {
		return nil, err
	}
	u := r.users[id]
	if u == nil {
		return nil, nil
	}
	cp := *u
	cp.Roles = nil
	for _, rid := range r.userRoles[id] {
		cp.Roles = append(cp.Roles, r.roles[rid])
	}
	return &cp, nil
}

func (r *rcUserRepo) FindAll(ctx context.Context, limit, offset int) ([]*models.UserModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("FindAll"); err != nil {
		return nil, err
	}
	out := make([]*models.UserModel, 0, len(r.users))
	for _, u := range r.users {
		out = append(out, u)
	}
	return out, nil
}

func (r *rcUserRepo) FindAllActive(ctx context.Context, limit, offset int) ([]*models.UserModel, error) {
	return r.FindAll(ctx, limit, offset)
}

func (r *rcUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("ExistsByEmail"); err != nil {
		return false, err
	}
	if r.existsOverride != nil {
		return *r.existsOverride, nil
	}
	for _, u := range r.users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (r *rcUserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("ExistsByUsername"); err != nil {
		return false, err
	}
	for _, u := range r.users {
		if u.Username == username {
			return true, nil
		}
	}
	return false, nil
}

func (r *rcUserRepo) Count(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Count"); err != nil {
		return 0, err
	}
	return len(r.users), nil
}

func (r *rcUserRepo) CountActive(ctx context.Context) (int, error) { return r.Count(ctx) }

func (r *rcUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error { return nil }

func (r *rcUserRepo) IncrementFailedAttempts(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failed[id]++
	return nil
}

func (r *rcUserRepo) ResetFailedAttempts(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failed[id] = 0
	return nil
}

func (r *rcUserRepo) LockAccount(ctx context.Context, id uuid.UUID, until *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.locked[id] = *until
	return nil
}

func (r *rcUserRepo) UnlockAccount(ctx context.Context, id uuid.UUID) error { return nil }

func (r *rcUserRepo) CreateSession(ctx context.Context, session *models.SessionModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("CreateSession"); err != nil {
		return err
	}
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	r.sessions[session.ID] = session
	return nil
}

func (r *rcUserRepo) FindSessionByToken(ctx context.Context, token string) (*models.SessionModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("FindSessionByToken"); err != nil {
		return nil, err
	}
	for _, s := range r.sessions {
		if s.Token == token {
			return s, nil
		}
	}
	return nil, nil
}

func (r *rcUserRepo) FindSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.SessionModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("FindSessionByRefreshToken"); err != nil {
		return nil, err
	}
	for _, s := range r.sessions {
		if s.RefreshToken == refreshToken {
			return s, nil
		}
	}
	return nil, nil
}

func (r *rcUserRepo) FindSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.SessionModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.SessionModel
	for _, s := range r.sessions {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (r *rcUserRepo) DeleteSession(ctx context.Context, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("DeleteSession"); err != nil {
		return err
	}
	for id, s := range r.sessions {
		if s.Token == token {
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *rcUserRepo) DeleteSessionByID(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("DeleteSessionByID"); err != nil {
		return err
	}
	delete(r.sessions, id)
	return nil
}

func (r *rcUserRepo) DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("DeleteSessionsByUserID"); err != nil {
		return err
	}
	for id, s := range r.sessions {
		if s.UserID == userID {
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *rcUserRepo) DeleteExpiredSessions(ctx context.Context) (int64, error) { return 0, nil }

func (r *rcUserRepo) UpdateSessionActivity(ctx context.Context, token string) error { return nil }

func (r *rcUserRepo) FindRoleByID(ctx context.Context, id uuid.UUID) (*models.RoleModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("FindRoleByID"); err != nil {
		return nil, err
	}
	return r.roles[id], nil
}

func (r *rcUserRepo) FindRoleByName(ctx context.Context, name string) (*models.RoleModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, role := range r.roles {
		if role.Name == name {
			return role, nil
		}
	}
	return nil, nil
}

func (r *rcUserRepo) FindAllRoles(ctx context.Context) ([]*models.RoleModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("FindAllRoles"); err != nil {
		return nil, err
	}
	var out []*models.RoleModel
	for _, role := range r.roles {
		out = append(out, role)
	}
	return out, nil
}

func (r *rcUserRepo) CreateRole(ctx context.Context, role *models.RoleModel) error {
	r.roles[role.ID] = role
	return nil
}

func (r *rcUserRepo) UpdateRole(ctx context.Context, role *models.RoleModel) error { return nil }

func (r *rcUserRepo) DeleteRole(ctx context.Context, id uuid.UUID) error { return nil }

func (r *rcUserRepo) AssignRole(ctx context.Context, userID, roleID uuid.UUID, assignedBy *uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("AssignRole"); err != nil {
		return err
	}
	r.userRoles[userID] = append(r.userRoles[userID], roleID)
	return nil
}

func (r *rcUserRepo) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := r.userRoles[userID]
	for i, id := range ids {
		if id == roleID {
			r.userRoles[userID] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
	return nil
}

func (r *rcUserRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*models.RoleModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("GetUserRoles"); err != nil {
		return nil, err
	}
	var out []*models.RoleModel
	for _, rid := range r.userRoles[userID] {
		out = append(out, r.roles[rid])
	}
	return out, nil
}

func (r *rcUserRepo) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return nil, nil
}

func (r *rcUserRepo) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.perms[userID.String()+":"+permission], nil
}

func (r *rcUserRepo) CreateAuditLog(ctx context.Context, log *models.AuditLogModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.audit = append(r.audit, log)
	return nil
}

func (r *rcUserRepo) FindAuditLogs(ctx context.Context, userID *uuid.UUID, action string, limit, offset int) ([]*models.AuditLogModel, error) {
	return r.audit, nil
}

// rcAccountRepo records billing account creation.
type rcAccountRepo struct {
	mu      sync.Mutex
	created []*pkgmodels.Account
	err     error
}

func (a *rcAccountRepo) Create(ctx context.Context, account *pkgmodels.Account) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return a.err
	}
	a.created = append(a.created, account)
	return nil
}
func (a *rcAccountRepo) GetByID(ctx context.Context, id string) (*pkgmodels.Account, error) {
	return nil, nil
}
func (a *rcAccountRepo) GetByUserID(ctx context.Context, userID string) (*pkgmodels.Account, error) {
	return nil, nil
}
func (a *rcAccountRepo) Update(ctx context.Context, account *pkgmodels.Account) error { return nil }
func (a *rcAccountRepo) UpdateBalance(ctx context.Context, id string, newBalance float64) error {
	return nil
}
func (a *rcAccountRepo) Suspend(ctx context.Context, id string) error  { return nil }
func (a *rcAccountRepo) Activate(ctx context.Context, id string) error { return nil }
func (a *rcAccountRepo) Close(ctx context.Context, id string) error    { return nil }

func rcAuthConfig() *config.AuthConfig {
	return &config.AuthConfig{
		JWTSecret:          "rc-test-secret-that-is-at-least-32-characters",
		JWTExpirationHours: 1,
		RefreshExpiryDays:  1,
		MinPasswordLength:  8,
		MaxLoginAttempts:   3,
		LockoutDuration:    time.Minute,
		AllowRegistration:  true,
	}
}

const rcGoodPassword = "Str0ngPassw0rd"

func rcNewService(t *testing.T) (*Service, *rcUserRepo, *rcAccountRepo) {
	t.Helper()
	repo := rcNewUserRepo()
	acct := &rcAccountRepo{}
	return rcFast(NewService(repo, acct, rcAuthConfig())), repo, acct
}

// rcFast lowers the bcrypt cost so tests stay fast under -race.
func rcFast(svc *Service) *Service {
	svc.passwordService.bcryptCost = bcrypt.MinCost
	return svc
}

func rcAddRole(repo *rcUserRepo, name string, perms ...string) *models.RoleModel {
	role := &models.RoleModel{ID: uuid.New(), Name: name, Description: name + " role", Permissions: perms}
	repo.roles[role.ID] = role
	return role
}

func TestRCService_RegisterLoginRefreshLogout(t *testing.T) {
	svc, repo, acct := rcNewService(t)
	rcAddRole(repo, "user", "workflows:read")
	ctx := context.Background()

	res, err := svc.Register(ctx, &RegisterRequest{Email: "  Alice@Example.COM ", Password: rcGoodPassword, FullName: "Alice"})
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", res.User.Email)
	assert.Equal(t, "alice@example.com", res.User.Username)
	assert.Equal(t, []string{"user"}, res.User.Roles)
	assert.Equal(t, "Bearer", res.TokenType)
	assert.Equal(t, 3600, res.ExpiresIn)
	require.Len(t, acct.created, 1)
	assert.Equal(t, res.User.ID, acct.created[0].UserID)

	// Duplicate registration
	_, err = svc.Register(ctx, &RegisterRequest{Email: "alice@example.com", Password: rcGoodPassword})
	assert.ErrorIs(t, err, ErrEmailAlreadyTaken)

	// Token validates
	claims, err := svc.ValidateToken(ctx, res.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, res.User.ID, claims.UserID)
	_, err = svc.ValidateToken(ctx, "garbage")
	assert.Error(t, err)

	// Login with mixed-case email
	login, err := svc.Login(ctx, &LoginRequest{Email: "ALICE@example.com", Password: rcGoodPassword}, "1.2.3.4", "ua")
	require.NoError(t, err)
	assert.NotEmpty(t, login.RefreshToken)
	require.NotEmpty(t, repo.audit)
	last := repo.audit[len(repo.audit)-1]
	assert.Equal(t, "login_success", last.Action)
	assert.Equal(t, "success", last.Metadata["status"])

	// Refresh rotates the session
	refreshed, err := svc.RefreshToken(ctx, login.RefreshToken)
	require.NoError(t, err)
	assert.NotEqual(t, login.RefreshToken, refreshed.RefreshToken)
	_, err = svc.RefreshToken(ctx, login.RefreshToken)
	assert.ErrorIs(t, err, ErrInvalidRefreshToken, "old refresh token must be single-use")

	// Current user
	u, err := svc.GetCurrentUser(ctx, res.User.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"user"}, u.Roles)
	_, err = svc.GetCurrentUser(ctx, "not-a-uuid")
	assert.Error(t, err)
	_, err = svc.GetCurrentUser(ctx, uuid.NewString())
	assert.ErrorIs(t, err, ErrUserNotFound)

	// Logout
	require.NoError(t, svc.Logout(ctx, refreshed.AccessToken))
	uid := uuid.MustParse(res.User.ID)
	require.NoError(t, svc.LogoutAll(ctx, uid))
	sessions, _ := repo.FindSessionsByUserID(ctx, uid)
	assert.Empty(t, sessions)
}

func TestRCService_RegisterValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("disabled", func(t *testing.T) {
		svc, _, _ := rcNewService(t)
		svc.config.AllowRegistration = false
		_, err := svc.Register(ctx, &RegisterRequest{Email: "a@b.c", Password: rcGoodPassword})
		assert.ErrorIs(t, err, ErrRegistrationDisabled)
	})
	t.Run("weak password", func(t *testing.T) {
		svc, _, _ := rcNewService(t)
		_, err := svc.Register(ctx, &RegisterRequest{Email: "a@b.c", Password: "short"})
		assert.ErrorIs(t, err, ErrPasswordTooShort)
	})
	t.Run("exists check fails", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		repo.errs["ExistsByEmail"] = errRCRepo
		_, err := svc.Register(ctx, &RegisterRequest{Email: "a@b.c", Password: rcGoodPassword})
		assert.ErrorIs(t, err, errRCRepo)
	})
	t.Run("create fails", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		repo.errs["Create"] = errRCRepo
		_, err := svc.Register(ctx, &RegisterRequest{Email: "a@b.c", Password: rcGoodPassword})
		assert.ErrorIs(t, err, errRCRepo)
	})
	t.Run("roles lookup fails", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		repo.errs["GetUserRoles"] = errRCRepo
		_, err := svc.Register(ctx, &RegisterRequest{Email: "a@b.c", Password: rcGoodPassword})
		assert.ErrorIs(t, err, errRCRepo)
	})
	t.Run("session create fails", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		repo.errs["CreateSession"] = errRCRepo
		_, err := svc.Register(ctx, &RegisterRequest{Email: "a@b.c", Password: rcGoodPassword})
		assert.ErrorIs(t, err, errRCRepo)
	})
	t.Run("billing account failure is not fatal", func(t *testing.T) {
		svc, _, acct := rcNewService(t)
		acct.err = errRCRepo
		_, err := svc.Register(ctx, &RegisterRequest{Email: "a@b.c", Password: rcGoodPassword})
		assert.NoError(t, err)
	})
	t.Run("nil account repo", func(t *testing.T) {
		repo := rcNewUserRepo()
		svc := rcFast(NewService(repo, nil, rcAuthConfig()))
		_, err := svc.Register(ctx, &RegisterRequest{Email: "a@b.c", Password: rcGoodPassword})
		assert.NoError(t, err)
	})
}

func rcSeedUser(t *testing.T, svc *Service, repo *rcUserRepo, email string) *models.UserModel {
	t.Helper()
	u, err := svc.CreateUser(context.Background(), &RegisterRequest{Email: email, Password: rcGoodPassword, FullName: "Seed"}, false)
	require.NoError(t, err)
	return repo.users[uuid.MustParse(u.ID)]
}

func TestRCService_LoginFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("lookup error", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		repo.errs["FindByEmail"] = errRCRepo
		_, err := svc.Login(ctx, &LoginRequest{Email: "x@y.z"}, "", "")
		assert.ErrorIs(t, err, errRCRepo)
	})
	t.Run("unknown user", func(t *testing.T) {
		svc, _, _ := rcNewService(t)
		_, err := svc.Login(ctx, &LoginRequest{Email: "x@y.z", Password: "p"}, "", "")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})
	t.Run("inactive", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		u := rcSeedUser(t, svc, repo, "in@x.y")
		u.IsActive = false
		_, err := svc.Login(ctx, &LoginRequest{Email: "in@x.y", Password: rcGoodPassword}, "ip", "ua")
		assert.ErrorIs(t, err, ErrAccountInactive)
		assert.Equal(t, "account_inactive", repo.audit[len(repo.audit)-1].Metadata["error"])
	})
	t.Run("locked", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		u := rcSeedUser(t, svc, repo, "lk@x.y")
		until := time.Now().Add(time.Hour)
		u.LockedUntil = &until
		_, err := svc.Login(ctx, &LoginRequest{Email: "lk@x.y", Password: rcGoodPassword}, "ip", "ua")
		assert.ErrorIs(t, err, ErrAccountLocked)
	})
	t.Run("wrong password increments and locks", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		u := rcSeedUser(t, svc, repo, "wp@x.y")
		_, err := svc.Login(ctx, &LoginRequest{Email: "wp@x.y", Password: "nope-nope"}, "", "")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.Equal(t, 1, repo.failed[u.ID])
		assert.Empty(t, repo.locked[u.ID])

		u.FailedLoginAttempts = 2 // next failure reaches MaxLoginAttempts=3
		_, err = svc.Login(ctx, &LoginRequest{Email: "wp@x.y", Password: "nope-nope"}, "", "")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
		assert.NotEmpty(t, repo.locked[u.ID])
		_, perr := time.Parse(time.RFC3339, repo.locked[u.ID])
		assert.NoError(t, perr)
	})
	t.Run("token generation fails", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		rcSeedUser(t, svc, repo, "tg@x.y")
		repo.errs["CreateSession"] = errRCRepo
		_, err := svc.Login(ctx, &LoginRequest{Email: "tg@x.y", Password: rcGoodPassword}, "", "")
		assert.ErrorIs(t, err, errRCRepo)
	})
}

func TestRCService_LogoutErrors(t *testing.T) {
	svc, repo, _ := rcNewService(t)
	repo.errs["DeleteSession"] = errRCRepo
	repo.errs["DeleteSessionsByUserID"] = errRCRepo
	assert.ErrorIs(t, svc.Logout(context.Background(), "t"), errRCRepo)
	assert.ErrorIs(t, svc.LogoutAll(context.Background(), uuid.New()), errRCRepo)
}

func TestRCService_RefreshTokenFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("lookup error", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		repo.errs["FindSessionByRefreshToken"] = errRCRepo
		_, err := svc.RefreshToken(ctx, "r")
		assert.ErrorIs(t, err, errRCRepo)
	})
	t.Run("user lookup error", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		repo.sessions[uuid.New()] = &models.SessionModel{UserID: uuid.New(), RefreshToken: "r"}
		repo.errs["FindByID"] = errRCRepo
		_, err := svc.RefreshToken(ctx, "r")
		assert.ErrorIs(t, err, errRCRepo)
	})
	t.Run("user gone removes session", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		sid := uuid.New()
		repo.sessions[sid] = &models.SessionModel{ID: sid, UserID: uuid.New(), RefreshToken: "r"}
		_, err := svc.RefreshToken(ctx, "r")
		assert.ErrorIs(t, err, ErrInvalidRefreshToken)
		assert.NotContains(t, repo.sessions, sid)
	})
	t.Run("delete old session fails", func(t *testing.T) {
		svc, repo, _ := rcNewService(t)
		u := rcSeedUser(t, svc, repo, "r@x.y")
		sid := uuid.New()
		repo.sessions[sid] = &models.SessionModel{ID: sid, UserID: u.ID, RefreshToken: "r"}
		repo.errs["DeleteSessionByID"] = errRCRepo
		_, err := svc.RefreshToken(ctx, "r")
		assert.ErrorIs(t, err, errRCRepo)
	})
}

func TestRCService_GetCurrentUserRepoError(t *testing.T) {
	svc, repo, _ := rcNewService(t)
	repo.errs["FindByIDWithRoles"] = errRCRepo
	_, err := svc.GetCurrentUser(context.Background(), uuid.NewString())
	assert.ErrorIs(t, err, errRCRepo)
	_, err = svc.GetUser(context.Background(), uuid.New())
	assert.ErrorIs(t, err, errRCRepo)
}

func TestRCService_ChangePassword(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := rcNewService(t)
	u := rcSeedUser(t, svc, repo, "cp@x.y")
	_, err := svc.Login(ctx, &LoginRequest{Email: "cp@x.y", Password: rcGoodPassword}, "", "")
	require.NoError(t, err)

	assert.ErrorIs(t, svc.ChangePassword(ctx, uuid.New(), "a", "b"), ErrUserNotFound)
	assert.ErrorIs(t, svc.ChangePassword(ctx, u.ID, "wrong-old", "N3wPassword"), ErrInvalidCredentials)
	assert.ErrorIs(t, svc.ChangePassword(ctx, u.ID, rcGoodPassword, "short"), ErrPasswordTooShort)
	assert.Error(t, svc.ChangePassword(ctx, u.ID, rcGoodPassword, "Aa1"+string(rcRepeat('x', 80))), "bcrypt rejects >72 bytes")

	require.NoError(t, svc.ChangePassword(ctx, u.ID, rcGoodPassword, "N3wPassword!"))
	sessions, _ := repo.FindSessionsByUserID(ctx, u.ID)
	assert.Empty(t, sessions, "all sessions are revoked")
	_, err = svc.Login(ctx, &LoginRequest{Email: "cp@x.y", Password: "N3wPassword!"}, "", "")
	assert.NoError(t, err)

	repo.errs["Update"] = errRCRepo
	assert.ErrorIs(t, svc.ChangePassword(ctx, u.ID, "N3wPassword!", "An0therPass"), errRCRepo)
	repo.errs["FindByID"] = errRCRepo
	assert.ErrorIs(t, svc.ChangePassword(ctx, u.ID, "x", "y"), errRCRepo)
}

func rcRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func TestRCService_CreateUserAndEnsureAdmin(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := rcNewService(t)

	created, err := svc.EnsureAdmin(ctx, "Admin@X.y", rcGoodPassword)
	require.NoError(t, err)
	assert.True(t, created)
	u, _ := repo.FindByEmail(ctx, "admin@x.y")
	require.NotNil(t, u)
	assert.True(t, u.IsAdmin)
	assert.Equal(t, "Administrator", u.FullName)

	created, err = svc.EnsureAdmin(ctx, "admin@x.y", rcGoodPassword)
	require.NoError(t, err)
	assert.False(t, created, "existing admin is not recreated")

	_, err = svc.CreateUser(ctx, &RegisterRequest{Email: "admin@x.y", Password: rcGoodPassword}, false)
	assert.ErrorIs(t, err, ErrEmailAlreadyTaken)
	_, err = svc.CreateUser(ctx, &RegisterRequest{Email: "w@x.y", Password: "weak"}, false)
	assert.Error(t, err)
	_, err = svc.CreateUser(ctx, &RegisterRequest{Email: "w@x.y", Password: "Aa1" + string(rcRepeat('x', 80))}, false)
	assert.Error(t, err)

	repo.errs["Create"] = errRCRepo
	_, err = svc.CreateUser(ctx, &RegisterRequest{Email: "c@x.y", Password: rcGoodPassword}, false)
	assert.ErrorIs(t, err, errRCRepo)
	_, err = svc.EnsureAdmin(ctx, "c@x.y", rcGoodPassword)
	assert.ErrorIs(t, err, errRCRepo)
	delete(repo.errs, "Create")

	repo.errs["ExistsByEmail"] = errRCRepo
	_, err = svc.CreateUser(ctx, &RegisterRequest{Email: "c@x.y", Password: rcGoodPassword}, false)
	assert.ErrorIs(t, err, errRCRepo)
	_, err = svc.EnsureAdmin(ctx, "c@x.y", rcGoodPassword)
	assert.ErrorIs(t, err, errRCRepo)
}

// rcRaceRepo reports that the email does not exist on the first check but
// exists on the second, simulating a concurrent replica creating the admin.
type rcRaceRepo struct {
	*rcUserRepo
	calls int
}

func (r *rcRaceRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	r.calls++
	return r.calls > 1, nil
}

func TestRCService_EnsureAdminConcurrentCreation(t *testing.T) {
	repo := &rcRaceRepo{rcUserRepo: rcNewUserRepo()}
	svc := rcFast(NewService(repo, nil, rcAuthConfig()))
	created, err := svc.EnsureAdmin(context.Background(), "a@x.y", rcGoodPassword)
	require.NoError(t, err)
	assert.False(t, created)
}

func TestRCService_UserAdministration(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := rcNewService(t)
	u := rcSeedUser(t, svc, repo, "u@x.y")
	other := rcSeedUser(t, svc, repo, "o@x.y")
	u.Metadata = models.JSONBMap{"k": "v"}

	got, err := svc.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "v", got.Metadata["k"])
	_, err = svc.GetUser(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrUserNotFound)

	list, total, err := svc.ListUsers(ctx, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, list, 2)

	// Update: email collision, username collision, success
	_, err = svc.UpdateUser(ctx, u.ID, other.Email, u.Username, "", true, false)
	assert.ErrorIs(t, err, ErrEmailAlreadyTaken)
	_, err = svc.UpdateUser(ctx, u.ID, u.Email, other.Username, "", true, false)
	assert.ErrorIs(t, err, ErrUsernameAlreadyTaken)
	updated, err := svc.UpdateUser(ctx, u.ID, "new@x.y", "newname", "New Name", false, true)
	require.NoError(t, err)
	assert.Equal(t, "new@x.y", updated.Email)
	assert.Equal(t, "newname", updated.Username)
	assert.False(t, updated.IsActive)
	assert.True(t, updated.IsAdmin)
	_, err = svc.UpdateUser(ctx, uuid.New(), "a", "b", "", true, false)
	assert.ErrorIs(t, err, ErrUserNotFound)

	// Reset password
	assert.ErrorIs(t, svc.ResetUserPassword(ctx, uuid.New(), rcGoodPassword), ErrUserNotFound)
	assert.Error(t, svc.ResetUserPassword(ctx, u.ID, "weak"))
	assert.Error(t, svc.ResetUserPassword(ctx, u.ID, "Aa1"+string(rcRepeat('x', 80))))
	require.NoError(t, svc.ResetUserPassword(ctx, u.ID, "Res3tPassword"))
	assert.NoError(t, svc.passwordService.VerifyPassword("Res3tPassword", repo.users[u.ID].PasswordHash))

	// Delete
	assert.ErrorIs(t, svc.DeleteUser(ctx, uuid.New()), ErrUserNotFound)
	require.NoError(t, svc.DeleteUser(ctx, other.ID))
	assert.True(t, repo.deleted[other.ID])
}

func TestRCService_UserAdministrationErrors(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := rcNewService(t)
	u := rcSeedUser(t, svc, repo, "u@x.y")

	repo.errs["Count"] = errRCRepo
	_, _, err := svc.ListUsers(ctx, 10, 0)
	assert.ErrorIs(t, err, errRCRepo)
	repo.errs["FindAll"] = errRCRepo
	_, _, err = svc.ListUsers(ctx, 10, 0)
	assert.ErrorIs(t, err, errRCRepo)

	repo.errs["ExistsByUsername"] = errRCRepo
	_, err = svc.UpdateUser(ctx, u.ID, u.Email, "changed", "", true, false)
	assert.ErrorIs(t, err, errRCRepo)
	repo.errs["ExistsByEmail"] = errRCRepo
	_, err = svc.UpdateUser(ctx, u.ID, "changed@x.y", u.Username, "", true, false)
	assert.ErrorIs(t, err, errRCRepo)
	repo.errs["Update"] = errRCRepo
	_, err = svc.UpdateUser(ctx, u.ID, u.Email, u.Username, "", true, false)
	assert.ErrorIs(t, err, errRCRepo)
	assert.ErrorIs(t, svc.ResetUserPassword(ctx, u.ID, "Res3tPassword"), errRCRepo)

	repo.errs["Delete"] = errRCRepo
	assert.ErrorIs(t, svc.DeleteUser(ctx, u.ID), errRCRepo)

	repo.errs["FindByID"] = errRCRepo
	_, err = svc.UpdateUser(ctx, u.ID, "", "", "", true, false)
	assert.ErrorIs(t, err, errRCRepo)
	assert.ErrorIs(t, svc.DeleteUser(ctx, u.ID), errRCRepo)
	assert.ErrorIs(t, svc.ResetUserPassword(ctx, u.ID, "x"), errRCRepo)
}

func TestRCService_Roles(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := rcNewService(t)
	u := rcSeedUser(t, svc, repo, "u@x.y")
	editor := rcAddRole(repo, "editor", "workflows:write")

	roles, err := svc.ListRoles(ctx)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "editor", roles[0].Name)
	assert.Equal(t, []string{"workflows:write"}, roles[0].Permissions)

	assert.ErrorIs(t, svc.AssignRoleToUser(ctx, uuid.New(), editor.ID, nil), ErrUserNotFound)
	assert.ErrorIs(t, svc.AssignRoleToUser(ctx, u.ID, uuid.New(), nil), ErrRoleNotFound)
	by := uuid.New()
	require.NoError(t, svc.AssignRoleToUser(ctx, u.ID, editor.ID, &by))

	userRoles, err := svc.GetUserRoles(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, userRoles, 1)
	assert.Equal(t, editor.ID.String(), userRoles[0].ID)

	require.NoError(t, svc.RemoveRoleFromUser(ctx, u.ID, editor.ID))
	userRoles, err = svc.GetUserRoles(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, userRoles)

	repo.perms[u.ID.String()+":workflows:write"] = true
	ok, err := svc.HasPermission(ctx, u.ID, "workflows:write")
	require.NoError(t, err)
	assert.True(t, ok)
	ok, _ = svc.HasPermission(ctx, u.ID, "workflows:delete")
	assert.False(t, ok)

	repo.errs["FindAllRoles"] = errRCRepo
	repo.errs["GetUserRoles"] = errRCRepo
	_, err = svc.ListRoles(ctx)
	assert.ErrorIs(t, err, errRCRepo)
	_, err = svc.GetUserRoles(ctx, u.ID)
	assert.ErrorIs(t, err, errRCRepo)
}

func TestRCBuiltinProvider(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := rcNewService(t)
	rcSeedUser(t, svc, repo, "b@x.y")
	p := NewBuiltinProvider(svc)

	assert.Equal(t, ProviderTypeBuiltin, p.GetType())
	assert.True(t, p.IsAvailable())
	assert.Empty(t, p.GetAuthorizationURL("s", "n"))
	_, err := p.HandleCallback(ctx, "c", "s")
	assert.Error(t, err)

	_, err = p.Authenticate(ctx, &Credentials{Email: "b@x.y", Password: "wrong-password"})
	assert.ErrorIs(t, err, ErrInvalidCredentials)

	res, err := p.Authenticate(ctx, &Credentials{Email: "b@x.y", Password: rcGoodPassword, IPAddress: "ip", UserAgent: "ua"})
	require.NoError(t, err)
	assert.Equal(t, "Bearer", res.TokenType)

	claims, err := p.ValidateToken(ctx, res.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "b@x.y", claims.Email)

	user, err := p.GetUserInfo(ctx, res.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, "b@x.y", user.Email)
	_, err = p.GetUserInfo(ctx, "bad")
	assert.Error(t, err)

	refreshed, err := p.RefreshToken(ctx, res.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.AccessToken)
	_, err = p.RefreshToken(ctx, "unknown")
	assert.ErrorIs(t, err, ErrInvalidRefreshToken)
}
