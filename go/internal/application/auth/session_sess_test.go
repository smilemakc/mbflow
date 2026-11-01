package auth

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

// sessRepo counts session lookups and activity updates on top of rcUserRepo.
type sessRepo struct {
	*rcUserRepo
	lookups    atomic.Int32
	activities atomic.Int32
	// withUser attaches the owning user to looked-up sessions, like the
	// storage repository's Relation("User").
	withUser bool
}

func (r *sessRepo) FindSessionByToken(ctx context.Context, token string) (*models.SessionModel, error) {
	r.lookups.Add(1)
	s, err := r.rcUserRepo.FindSessionByToken(ctx, token)
	if err != nil || s == nil || !r.withUser {
		return s, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *s
	cp.User = r.users[s.UserID]
	return &cp, nil
}

func (r *sessRepo) UpdateSessionActivity(ctx context.Context, token string) error {
	r.activities.Add(1)
	return nil
}

// sessClock is a manually advanced clock for the session cache.
type sessClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *sessClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *sessClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func sessNewService(t *testing.T, repo *sessRepo) (*Service, *sessClock) {
	t.Helper()
	svc := rcFast(NewService(repo, nil, rcAuthConfig()))
	clock := &sessClock{now: time.Now()}
	svc.sessions.now = clock.Now
	return svc, clock
}

func sessSetup(t *testing.T) (*Service, *sessRepo, *sessClock, *models.UserModel) {
	t.Helper()
	repo := &sessRepo{rcUserRepo: rcNewUserRepo(), withUser: true}
	svc, clock := sessNewService(t, repo)
	user := rcSeedUser(t, svc, repo.rcUserRepo, "sess@example.com")
	return svc, repo, clock, user
}

func sessLogin(t *testing.T, svc *Service) *AuthResult {
	t.Helper()
	res, err := svc.Login(context.Background(), &LoginRequest{Email: "sess@example.com", Password: rcGoodPassword}, "", "")
	require.NoError(t, err)
	return res
}

func TestSess_LogoutRevokesAccessToken(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, user := sessSetup(t)
	res := sessLogin(t, svc)

	claims, err := svc.ValidateToken(ctx, res.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), claims.UserID)

	// Cached: further validations do not hit the repository.
	for i := 0; i < 5; i++ {
		_, err = svc.ValidateToken(ctx, res.AccessToken)
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), repo.lookups.Load())
	assert.Equal(t, int32(1), repo.activities.Load())

	require.NoError(t, svc.Logout(ctx, res.AccessToken))
	_, err = svc.ValidateToken(ctx, res.AccessToken)
	assert.ErrorIs(t, err, ErrSessionRevoked)

	// Also through the builtin provider used by the middleware.
	_, err = NewBuiltinProvider(svc).ValidateToken(ctx, res.AccessToken)
	assert.ErrorIs(t, err, ErrSessionRevoked)
}

func TestSess_RevocationOnOtherReplicaBoundedByTTL(t *testing.T) {
	ctx := context.Background()
	svcA, repo, _, _ := sessSetup(t)
	svcB, clockB := sessNewService(t, repo)
	res := sessLogin(t, svcA)

	_, err := svcB.ValidateToken(ctx, res.AccessToken)
	require.NoError(t, err)

	require.NoError(t, svcA.Logout(ctx, res.AccessToken))
	_, err = svcA.ValidateToken(ctx, res.AccessToken)
	assert.ErrorIs(t, err, ErrSessionRevoked, "immediate on the replica that revoked")

	_, err = svcB.ValidateToken(ctx, res.AccessToken)
	assert.NoError(t, err, "other replica may serve its cached answer within the TTL")

	clockB.Advance(defaultSessionCacheTTL)
	_, err = svcB.ValidateToken(ctx, res.AccessToken)
	assert.ErrorIs(t, err, ErrSessionRevoked, "after the TTL every replica rejects")
}

func TestSess_ValidateTokenFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("bad signature never reaches the store", func(t *testing.T) {
		svc, repo, _, _ := sessSetup(t)
		_, err := svc.ValidateToken(ctx, "not.a.jwt")
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Zero(t, repo.lookups.Load())
	})

	t.Run("store error fails closed and is not cached", func(t *testing.T) {
		svc, repo, _, _ := sessSetup(t)
		res := sessLogin(t, svc)
		repo.mu.Lock()
		repo.errs["FindSessionByToken"] = errRCRepo
		repo.mu.Unlock()
		_, err := svc.ValidateToken(ctx, res.AccessToken)
		assert.ErrorIs(t, err, ErrSessionUnavailable)
		assert.ErrorContains(t, err, errRCRepo.Error())

		repo.mu.Lock()
		delete(repo.errs, "FindSessionByToken")
		repo.mu.Unlock()
		_, err = svc.ValidateToken(ctx, res.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, int32(2), repo.lookups.Load())
	})

	t.Run("token without a session", func(t *testing.T) {
		// Tokens signed before session checking existed always had a session
		// row; one that has none (logged out, or never issued as a session)
		// is rejected.
		svc, _, _, user := sessSetup(t)
		token, _, err := svc.jwtService.GenerateAccessToken(&pkgmodels.User{ID: user.ID.String(), Email: user.Email})
		require.NoError(t, err)
		_, err = svc.ValidateToken(ctx, token)
		assert.ErrorIs(t, err, ErrSessionRevoked)
	})

	t.Run("session of another user", func(t *testing.T) {
		svc, repo, _, user := sessSetup(t)
		token, _, err := svc.jwtService.GenerateAccessToken(&pkgmodels.User{ID: uuid.NewString()})
		require.NoError(t, err)
		require.NoError(t, repo.CreateSession(ctx, &models.SessionModel{UserID: user.ID, Token: token, ExpiresAt: time.Now().Add(time.Hour)}))
		_, err = svc.ValidateToken(ctx, token)
		assert.ErrorIs(t, err, ErrSessionRevoked)
	})

	t.Run("inactive owner", func(t *testing.T) {
		svc, repo, _, user := sessSetup(t)
		res := sessLogin(t, svc)
		repo.mu.Lock()
		user.IsActive = false
		repo.mu.Unlock()
		_, err := svc.ValidateToken(ctx, res.AccessToken)
		assert.ErrorIs(t, err, ErrAccountInactive)
	})
}

func TestSess_SecurityChangesRevokeSessions(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name   string
		change func(t *testing.T, svc *Service, user *models.UserModel)
	}{
		{"password change", func(t *testing.T, svc *Service, u *models.UserModel) {
			require.NoError(t, svc.ChangePassword(ctx, u.ID, rcGoodPassword, rcGoodPassword+"2"))
		}},
		{"admin password reset", func(t *testing.T, svc *Service, u *models.UserModel) {
			require.NoError(t, svc.ResetUserPassword(ctx, u.ID, rcGoodPassword+"3"))
		}},
		{"deactivation", func(t *testing.T, svc *Service, u *models.UserModel) {
			_, err := svc.UpdateUser(ctx, u.ID, u.Email, u.Username, u.FullName, false, u.IsAdmin)
			require.NoError(t, err)
		}},
		{"admin flag change", func(t *testing.T, svc *Service, u *models.UserModel) {
			_, err := svc.UpdateUser(ctx, u.ID, u.Email, u.Username, u.FullName, true, true)
			require.NoError(t, err)
		}},
		{"role removal", func(t *testing.T, svc *Service, u *models.UserModel) {
			require.NoError(t, svc.RemoveRoleFromUser(ctx, u.ID, uuid.New()))
		}},
		{"user deletion", func(t *testing.T, svc *Service, u *models.UserModel) {
			require.NoError(t, svc.DeleteUser(ctx, u.ID))
		}},
		{"logout all", func(t *testing.T, svc *Service, u *models.UserModel) {
			require.NoError(t, svc.LogoutAll(ctx, u.ID))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _, user := sessSetup(t)
			res := sessLogin(t, svc)
			_, err := svc.ValidateToken(ctx, res.AccessToken) // warm the cache
			require.NoError(t, err)

			tc.change(t, svc, user)

			_, err = svc.ValidateToken(ctx, res.AccessToken)
			assert.Error(t, err)
		})
	}

	t.Run("profile edit keeps sessions", func(t *testing.T) {
		svc, _, _, user := sessSetup(t)
		res := sessLogin(t, svc)
		_, err := svc.UpdateUser(ctx, user.ID, user.Email, user.Username, "New Name", true, false)
		require.NoError(t, err)
		_, err = svc.ValidateToken(ctx, res.AccessToken)
		assert.NoError(t, err)
	})

	t.Run("revocation failures are not fatal", func(t *testing.T) {
		svc, repo, _, user := sessSetup(t)
		repo.errs["DeleteSessionsByUserID"] = errRCRepo
		require.NoError(t, svc.ResetUserPassword(ctx, user.ID, rcGoodPassword+"4"))
		assert.ErrorIs(t, svc.LogoutAll(ctx, user.ID), errRCRepo)
		repo.errs["DeleteSession"] = errRCRepo
		assert.ErrorIs(t, svc.Logout(ctx, "x"), errRCRepo)
	})

	t.Run("role removal error", func(t *testing.T) {
		svc, repo, _, user := sessSetup(t)
		res := sessLogin(t, svc)
		failing := &sessFailingRoleRepo{sessRepo: repo}
		svc.userRepo = failing
		assert.ErrorIs(t, svc.RemoveRoleFromUser(ctx, user.ID, uuid.New()), errRCRepo)
		_, err := svc.ValidateToken(ctx, res.AccessToken)
		assert.NoError(t, err, "sessions kept when the role was not removed")
	})
}

type sessFailingRoleRepo struct{ *sessRepo }

func (r *sessFailingRoleRepo) RemoveRole(context.Context, uuid.UUID, uuid.UUID) error {
	return errRCRepo
}

func TestSess_RefreshRotatesAccessToken(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := sessSetup(t)
	res := sessLogin(t, svc)
	_, err := svc.ValidateToken(ctx, res.AccessToken)
	require.NoError(t, err)

	// JWT timestamps have second resolution: cross a second boundary so the
	// new access token differs from the old one.
	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))

	next, err := svc.RefreshToken(ctx, res.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, res.AccessToken, next.AccessToken)

	_, err = svc.ValidateToken(ctx, res.AccessToken)
	assert.ErrorIs(t, err, ErrSessionRevoked, "the replaced access token is revoked with its session")
	_, err = svc.ValidateToken(ctx, next.AccessToken)
	assert.NoError(t, err)
}

func TestSess_ProviderManagerDoesNotMaskSessionErrors(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := sessSetup(t)
	res := sessLogin(t, svc)
	require.NoError(t, svc.Logout(ctx, res.AccessToken))

	fallback := &MockAuthProvider{providerType: ProviderTypeGateway}
	fallback.On("IsAvailable").Return(true).Maybe()
	fallback.On("ValidateToken", mock.Anything, mock.Anything).Return(nil, ErrInvalidToken).Maybe()
	pm := &ProviderManager{
		providers: map[ProviderType]AuthProvider{
			ProviderTypeBuiltin: NewBuiltinProvider(svc),
			ProviderTypeGateway: fallback,
		},
		primaryType:    ProviderTypeBuiltin,
		fallbackType:   ProviderTypeGateway,
		enableFallback: true,
	}
	_, err := pm.ValidateToken(ctx, res.AccessToken)
	assert.ErrorIs(t, err, ErrSessionRevoked)
	fallback.AssertNotCalled(t, "ValidateToken", mock.Anything, mock.Anything)
}

func TestSess_SessionCache(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	newCache := func(ttl time.Duration, maxEntries int) *sessionCache {
		c := newSessionCache(ttl, maxEntries)
		c.now = func() time.Time { return now }
		return c
	}

	t.Run("ttl and token expiry bound entries", func(t *testing.T) {
		c := newCache(10*time.Second, 10)
		_, gen := c.lookup("a")
		c.store("a", "u1", time.Time{}, gen)
		c.store("b", "u1", now.Add(2*time.Second), gen)
		c.store("expired", "u1", now.Add(-time.Second), gen)
		hit, _ := c.lookup("a")
		assert.True(t, hit)
		assert.Equal(t, 2, c.len())

		now = now.Add(3 * time.Second)
		hit, _ = c.lookup("b")
		assert.False(t, hit, "token expiry caps the entry")
		hit, _ = c.lookup("a")
		assert.True(t, hit)

		now = now.Add(10 * time.Second)
		hit, _ = c.lookup("a")
		assert.False(t, hit)
		assert.Zero(t, c.len(), "expired entries are dropped on lookup")
	})

	t.Run("disabled", func(t *testing.T) {
		c := newCache(0, 10)
		c.store("a", "u", time.Time{}, 0)
		assert.Zero(t, c.len())
	})

	t.Run("revocation racing a lookup wins", func(t *testing.T) {
		c := newCache(time.Minute, 10)
		_, gen := c.lookup("a")
		c.evictToken("a") // logout while the DB lookup was in flight
		c.store("a", "u", time.Time{}, gen)
		hit, _ := c.lookup("a")
		assert.False(t, hit)

		_, gen = c.lookup("b")
		c.evictUser("u")
		c.store("b", "u", time.Time{}, gen)
		hit, _ = c.lookup("b")
		assert.False(t, hit)
	})

	t.Run("evict user keeps other users", func(t *testing.T) {
		c := newCache(time.Minute, 10)
		_, gen := c.lookup("x")
		c.store("a1", "a", time.Time{}, gen)
		c.store("a2", "a", time.Time{}, gen)
		c.store("b1", "b", time.Time{}, gen)
		c.evictUser("a")
		assert.Equal(t, 1, c.len())
		hit, _ := c.lookup("b1")
		assert.True(t, hit)
	})

	t.Run("bounded size", func(t *testing.T) {
		c := newCache(time.Minute, 2)
		_, gen := c.lookup("x")
		c.store("a", "u", now.Add(time.Second), gen)
		c.store("b", "u", time.Time{}, gen)
		now = now.Add(2 * time.Second)
		c.store("c", "u", time.Time{}, gen) // sweeps the expired "a"
		assert.Equal(t, 2, c.len())
		hit, _ := c.lookup("b")
		assert.True(t, hit)

		c.store("d", "u", time.Time{}, gen) // full of live entries: reset
		assert.Equal(t, 1, c.len())
		hit, _ = c.lookup("d")
		assert.True(t, hit)
	})
}

func TestSess_ValidateTokenConcurrentWithLogout(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _ := sessSetup(t)
	res := sessLogin(t, svc)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = svc.ValidateToken(ctx, res.AccessToken)
			}
		}()
	}
	require.NoError(t, svc.Logout(ctx, res.AccessToken))
	wg.Wait()

	_, err := svc.ValidateToken(ctx, res.AccessToken)
	assert.ErrorIs(t, err, ErrSessionRevoked)
}
