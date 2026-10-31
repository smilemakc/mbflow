package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func rcNewUser(name string) *models.UserModel {
	suffix := uuid.NewString()[:8]
	return &models.UserModel{
		Email:        name + "_" + suffix + "@example.com",
		Username:     name + "_" + suffix,
		PasswordHash: "hash",
		FullName:     "Full " + name,
		IsActive:     true,
		Metadata:     models.JSONBMap{"src": "rc"},
	}
}

func TestRC_UserRepository_CRUD(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewUserRepository(db)

	u := rcNewUser("alice")
	require.NoError(t, repo.Create(ctx, u))
	require.NotEqual(t, uuid.Nil, u.ID)

	// unique email
	dup := rcNewUser("dup")
	dup.Email = u.Email
	assert.Error(t, repo.Create(ctx, dup))

	got, err := repo.FindByID(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.Email, got.Email)
	assert.Equal(t, "rc", got.Metadata["src"])

	got, err = repo.FindByEmail(ctx, u.Email)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.ID, got.ID)

	got, err = repo.FindByUsername(ctx, u.Username)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, u.ID, got.ID)

	missing, err := repo.FindByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missing)
	missing, err = repo.FindByEmail(ctx, "nobody@example.com")
	require.NoError(t, err)
	assert.Nil(t, missing)
	missing, err = repo.FindByUsername(ctx, "nobody")
	require.NoError(t, err)
	assert.Nil(t, missing)

	exists, err := repo.ExistsByEmail(ctx, u.Email)
	require.NoError(t, err)
	assert.True(t, exists)
	exists, err = repo.ExistsByUsername(ctx, u.Username)
	require.NoError(t, err)
	assert.True(t, exists)
	exists, err = repo.ExistsByEmail(ctx, "nobody@example.com")
	require.NoError(t, err)
	assert.False(t, exists)
	exists, err = repo.ExistsByUsername(ctx, "nobody")
	require.NoError(t, err)
	assert.False(t, exists)

	// update
	got.FullName = "Alice Updated"
	got.IsAdmin = true
	got.EmailVerified = true
	require.NoError(t, repo.Update(ctx, got))
	got, _ = repo.FindByID(ctx, u.ID)
	assert.Equal(t, "Alice Updated", got.FullName)
	assert.True(t, got.IsAdmin)
	assert.True(t, got.EmailVerified)

	// list / count with an inactive user
	inactive := rcNewUser("bob")
	inactive.IsActive = false
	require.NoError(t, repo.Create(ctx, inactive))

	all, err := repo.FindAll(ctx, 10, 0)
	require.NoError(t, err)
	assert.Len(t, all, 2)
	active, err := repo.FindAllActive(ctx, 10, 0)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, u.ID, active[0].ID)
	page, err := repo.FindAll(ctx, 1, 1)
	require.NoError(t, err)
	assert.Len(t, page, 1)

	n, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	n, err = repo.CountActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	// soft delete hides the user
	require.NoError(t, repo.Delete(ctx, inactive.ID))
	missing, err = repo.FindByID(ctx, inactive.ID)
	require.NoError(t, err)
	assert.Nil(t, missing)
	n, _ = repo.Count(ctx)
	assert.Equal(t, 1, n)

	// hard delete removes the row
	require.NoError(t, repo.HardDelete(ctx, inactive.ID))
	var rows int
	require.NoError(t, db.NewRaw("SELECT count(*) FROM mbflow_users WHERE id = ?", inactive.ID).Scan(ctx, &rows))
	assert.Zero(t, rows)
}

func TestRC_UserRepository_LoginTracking(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewUserRepository(db)

	u := rcNewUser("carol")
	require.NoError(t, repo.Create(ctx, u))

	require.NoError(t, repo.UpdateLastLogin(ctx, u.ID))
	require.NoError(t, repo.IncrementFailedAttempts(ctx, u.ID))
	require.NoError(t, repo.IncrementFailedAttempts(ctx, u.ID))
	got, _ := repo.FindByID(ctx, u.ID)
	assert.NotNil(t, got.LastLoginAt)
	assert.Equal(t, 2, got.FailedLoginAttempts)

	require.NoError(t, repo.ResetFailedAttempts(ctx, u.ID))
	got, _ = repo.FindByID(ctx, u.ID)
	assert.Zero(t, got.FailedLoginAttempts)

	until := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	require.NoError(t, repo.LockAccount(ctx, u.ID, &until))
	got, _ = repo.FindByID(ctx, u.ID)
	require.NotNil(t, got.LockedUntil)
	assert.True(t, got.IsLocked())
	assert.False(t, got.CanLogin())

	bad := "tomorrow"
	assert.Error(t, repo.LockAccount(ctx, u.ID, &bad))

	require.NoError(t, repo.IncrementFailedAttempts(ctx, u.ID))
	require.NoError(t, repo.UnlockAccount(ctx, u.ID))
	got, _ = repo.FindByID(ctx, u.ID)
	assert.Nil(t, got.LockedUntil)
	assert.Zero(t, got.FailedLoginAttempts)

	require.NoError(t, repo.LockAccount(ctx, u.ID, &until))
	require.NoError(t, repo.LockAccount(ctx, u.ID, nil))
	got, _ = repo.FindByID(ctx, u.ID)
	assert.Nil(t, got.LockedUntil, "nil lock time clears the lock")
}

func TestRC_UserRepository_Sessions(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewUserRepository(db)

	u := rcNewUser("dave")
	require.NoError(t, repo.Create(ctx, u))

	s1 := &models.SessionModel{UserID: u.ID, Token: "tok-" + uuid.NewString(), RefreshToken: "ref-1", ExpiresAt: time.Now().Add(time.Hour), IPAddress: "127.0.0.1", UserAgent: "ua"}
	s2 := &models.SessionModel{UserID: u.ID, Token: "tok-" + uuid.NewString(), RefreshToken: "ref-2", ExpiresAt: time.Now().Add(time.Hour)}
	expired := &models.SessionModel{UserID: u.ID, Token: "tok-" + uuid.NewString(), RefreshToken: "ref-3", ExpiresAt: time.Now().Add(-time.Hour)}
	for _, s := range []*models.SessionModel{s1, s2, expired} {
		require.NoError(t, repo.CreateSession(ctx, s))
		require.NotEqual(t, uuid.Nil, s.ID)
	}
	// duplicate token
	assert.Error(t, repo.CreateSession(ctx, &models.SessionModel{UserID: u.ID, Token: s1.Token, ExpiresAt: time.Now().Add(time.Hour)}))

	got, err := repo.FindSessionByToken(ctx, s1.Token)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, s1.ID, got.ID)
	require.NotNil(t, got.User)
	assert.Equal(t, u.Email, got.User.Email)

	none, err := repo.FindSessionByToken(ctx, expired.Token)
	require.NoError(t, err)
	assert.Nil(t, none, "expired sessions are not returned by token")

	got, err = repo.FindSessionByRefreshToken(ctx, "ref-2")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, s2.ID, got.ID)
	none, err = repo.FindSessionByRefreshToken(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, none)

	list, err := repo.FindSessionsByUserID(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	before := got.LastActivityAt
	require.NoError(t, repo.UpdateSessionActivity(ctx, s2.Token))
	got, _ = repo.FindSessionByToken(ctx, s2.Token)
	assert.False(t, got.LastActivityAt.Before(before))

	n, err := repo.DeleteExpiredSessions(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)

	require.NoError(t, repo.DeleteSession(ctx, s1.Token))
	none, _ = repo.FindSessionByToken(ctx, s1.Token)
	assert.Nil(t, none)

	require.NoError(t, repo.DeleteSessionByID(ctx, s2.ID))
	list, _ = repo.FindSessionsByUserID(ctx, u.ID)
	assert.Empty(t, list)

	s3 := &models.SessionModel{UserID: u.ID, Token: "tok-" + uuid.NewString(), ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, repo.CreateSession(ctx, s3))
	require.NoError(t, repo.DeleteSessionsByUserID(ctx, u.ID))
	list, _ = repo.FindSessionsByUserID(ctx, u.ID)
	assert.Empty(t, list)
}

func TestRC_UserRepository_RolesAndPermissions(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewUserRepository(db)

	u := rcNewUser("erin")
	admin := rcNewUser("root")
	admin.IsAdmin = true
	require.NoError(t, repo.Create(ctx, u))
	require.NoError(t, repo.Create(ctx, admin))

	seeded, err := repo.FindAllRoles(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, seeded, "migrations seed system roles")

	sysRole, err := repo.FindRoleByName(ctx, "ADMIN")
	require.NoError(t, err)
	require.NotNil(t, sysRole, "role lookup by name is case-insensitive")
	assert.True(t, sysRole.IsSystem)

	role := &models.RoleModel{Name: "rc-editor-" + uuid.NewString()[:6], Description: "edit", Permissions: models.StringArray{"workflow:read", "workflow:write"}}
	require.NoError(t, repo.CreateRole(ctx, role))
	require.NotEqual(t, uuid.Nil, role.ID)
	assert.Error(t, repo.CreateRole(ctx, &models.RoleModel{Name: role.Name}), "duplicate role name")

	got, err := repo.FindRoleByID(ctx, role.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, models.StringArray{"workflow:read", "workflow:write"}, got.Permissions)
	none, err := repo.FindRoleByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, none)
	none, err = repo.FindRoleByName(ctx, "no-such-role")
	require.NoError(t, err)
	assert.Nil(t, none)

	got.Description = "edited"
	got.Permissions = models.StringArray{"workflow:read", "workflow:write", "workflow:delete"}
	require.NoError(t, repo.UpdateRole(ctx, got))
	got, _ = repo.FindRoleByID(ctx, role.ID)
	assert.Equal(t, "edited", got.Description)
	assert.Len(t, got.Permissions, 3)

	// system roles are protected from update / delete
	sysRole.Description = "hacked"
	require.NoError(t, repo.UpdateRole(ctx, sysRole))
	require.NoError(t, repo.DeleteRole(ctx, sysRole.ID))
	sysAfter, _ := repo.FindRoleByID(ctx, sysRole.ID)
	require.NotNil(t, sysAfter)
	assert.NotEqual(t, "hacked", sysAfter.Description)

	// assignment (idempotent)
	require.NoError(t, repo.AssignRole(ctx, u.ID, role.ID, &admin.ID))
	require.NoError(t, repo.AssignRole(ctx, u.ID, role.ID, nil))
	roles, err := repo.GetUserRoles(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, role.ID, roles[0].ID)

	withRoles, err := repo.FindByIDWithRoles(ctx, u.ID)
	require.NoError(t, err)
	require.NotNil(t, withRoles)
	require.Len(t, withRoles.Roles, 1)
	assert.Equal(t, role.Name, withRoles.Roles[0].Name)
	missingUser, err := repo.FindByIDWithRoles(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missingUser)

	perms, err := repo.GetUserPermissions(ctx, u.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"workflow:read", "workflow:write", "workflow:delete"}, perms)

	ok, err := repo.HasPermission(ctx, u.ID, "workflow:delete")
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = repo.HasPermission(ctx, u.ID, "admin:all")
	require.NoError(t, err)
	assert.False(t, ok)

	// admins implicitly have everything
	perms, err = repo.GetUserPermissions(ctx, admin.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"*"}, perms)
	ok, err = repo.HasPermission(ctx, admin.ID, "anything")
	require.NoError(t, err)
	assert.True(t, ok)

	// unknown users
	perms, err = repo.GetUserPermissions(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, perms)
	ok, err = repo.HasPermission(ctx, uuid.New(), "x")
	require.NoError(t, err)
	assert.False(t, ok)

	// FK violation on assignment
	assert.Error(t, repo.AssignRole(ctx, uuid.New(), role.ID, nil))

	require.NoError(t, repo.RemoveRole(ctx, u.ID, role.ID))
	roles, _ = repo.GetUserRoles(ctx, u.ID)
	assert.Empty(t, roles)

	require.NoError(t, repo.DeleteRole(ctx, role.ID))
	none, _ = repo.FindRoleByID(ctx, role.ID)
	assert.Nil(t, none)
}

func TestRC_UserRepository_AuditLogs(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewUserRepository(db)

	u := rcNewUser("frank")
	require.NoError(t, repo.Create(ctx, u))

	resID := uuid.New()
	logs := []*models.AuditLogModel{
		{UserID: &u.ID, Action: "login", IPAddress: "127.0.0.1", UserAgent: "ua"},
		{UserID: &u.ID, Action: "logout"},
		{Action: "login", ResourceType: "workflow", ResourceID: &resID, Metadata: models.JSONBMap{"k": "v"}},
	}
	for _, l := range logs {
		require.NoError(t, repo.CreateAuditLog(ctx, l))
		require.NotEqual(t, uuid.Nil, l.ID)
	}

	all, err := repo.FindAuditLogs(ctx, nil, "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	byUser, err := repo.FindAuditLogs(ctx, &u.ID, "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, byUser, 2)

	logins, err := repo.FindAuditLogs(ctx, nil, "login", 10, 0)
	require.NoError(t, err)
	assert.Len(t, logins, 2)

	both, err := repo.FindAuditLogs(ctx, &u.ID, "login", 10, 0)
	require.NoError(t, err)
	require.Len(t, both, 1)
	assert.Equal(t, "login", both[0].Action)

	page, err := repo.FindAuditLogs(ctx, nil, "", 2, 2)
	require.NoError(t, err)
	assert.Len(t, page, 1)
}
