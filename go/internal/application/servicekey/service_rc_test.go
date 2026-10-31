package servicekey

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

var rcErrRepo = errors.New("repo failure")

// rcKeyRepo is an in-memory repository.ServiceKeyRepository with error injection.
type rcKeyRepo struct {
	mu   sync.Mutex
	keys map[string]*models.ServiceKey
	errs map[string]error
}

func rcNewKeyRepo() *rcKeyRepo {
	return &rcKeyRepo{keys: map[string]*models.ServiceKey{}, errs: map[string]error{}}
}

func (r *rcKeyRepo) Create(ctx context.Context, key *models.ServiceKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Create"]; err != nil {
		return err
	}
	r.keys[key.ID] = key
	return nil
}

func (r *rcKeyRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.ServiceKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["FindByID"]; err != nil {
		return nil, err
	}
	return r.keys[id.String()], nil
}

func (r *rcKeyRepo) FindByPrefix(ctx context.Context, prefix string) ([]*models.ServiceKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["FindByPrefix"]; err != nil {
		return nil, err
	}
	var out []*models.ServiceKey
	for _, k := range r.keys {
		if k.KeyPrefix == prefix {
			out = append(out, k)
		}
	}
	return out, nil
}

func (r *rcKeyRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*models.ServiceKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["FindByUserID"]; err != nil {
		return nil, err
	}
	var out []*models.ServiceKey
	for _, k := range r.keys {
		if k.UserID == userID.String() {
			out = append(out, k)
		}
	}
	return out, nil
}

func (r *rcKeyRepo) FindAll(ctx context.Context, filter repository.ServiceKeyFilter) ([]*models.ServiceKey, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["FindAll"]; err != nil {
		return nil, 0, err
	}
	var out []*models.ServiceKey
	for _, k := range r.keys {
		if filter.UserID != nil && k.UserID != filter.UserID.String() {
			continue
		}
		out = append(out, k)
	}
	return out, int64(len(out)), nil
}

func (r *rcKeyRepo) Update(ctx context.Context, key *models.ServiceKey) error { return nil }

func (r *rcKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Delete"]; err != nil {
		return err
	}
	delete(r.keys, id.String())
	return nil
}

func (r *rcKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Revoke"]; err != nil {
		return err
	}
	r.keys[id.String()].Status = models.ServiceKeyStatusRevoked
	return nil
}

func (r *rcKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.errs["UpdateLastUsed"]
}

func (r *rcKeyRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["CountByUserID"]; err != nil {
		return 0, err
	}
	var n int64
	for _, k := range r.keys {
		if k.UserID == userID.String() {
			n++
		}
	}
	return n, nil
}

func rcService(repo *rcKeyRepo) *Service {
	return NewService(repo, Config{MaxKeysPerUser: 2, BcryptCost: 4})
}

func TestRCServiceKey_Lifecycle(t *testing.T) {
	ctx := context.Background()
	repo := rcNewKeyRepo()
	svc := rcService(repo)
	user, other, admin := uuid.New(), uuid.New(), uuid.New()

	res, err := svc.CreateKey(ctx, user, "ci", "ci key", admin, nil)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(res.PlainKey, models.ServiceKeyPrefix))
	assert.Equal(t, res.PlainKey[:models.ServiceKeyPrefixLength], res.Key.KeyPrefix)
	assert.Equal(t, user.String(), res.Key.UserID)
	assert.Equal(t, admin.String(), res.Key.CreatedBy)

	_, err = svc.CreateKey(ctx, user, "ci-2", "", admin, nil)
	require.NoError(t, err)
	_, err = svc.CreateKey(ctx, user, "ci-3", "", admin, nil)
	assert.ErrorIs(t, err, models.ErrServiceKeyLimitReached, "per-user limit applies")
	_, err = svc.CreateKey(ctx, other, "other", "", admin, nil)
	require.NoError(t, err, "other users are not affected by the limit")

	got, err := svc.ValidateKey(ctx, res.PlainKey)
	require.NoError(t, err)
	assert.Equal(t, res.Key.ID, got.ID)
	assert.Equal(t, int64(1), got.UsageCount)

	_, err = svc.ValidateKey(ctx, res.Key.KeyPrefix+"definitely-wrong-secret")
	assert.ErrorIs(t, err, models.ErrInvalidServiceKey)

	byID, err := svc.GetByID(ctx, uuid.MustParse(res.Key.ID))
	require.NoError(t, err)
	assert.Same(t, res.Key, byID)
	_, err = svc.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, models.ErrServiceKeyNotFound)

	mine, err := svc.GetByUserID(ctx, user)
	require.NoError(t, err)
	assert.Len(t, mine, 2)

	all, total, err := svc.ListAll(ctx, repository.ServiceKeyFilter{})
	require.NoError(t, err)
	assert.Len(t, all, 3)
	assert.Equal(t, int64(3), total)
	filtered, _, err := svc.ListAll(ctx, repository.ServiceKeyFilter{UserID: &other})
	require.NoError(t, err)
	assert.Len(t, filtered, 1)

	require.NoError(t, svc.RevokeKey(ctx, uuid.MustParse(res.Key.ID)))
	_, err = svc.ValidateKey(ctx, res.PlainKey)
	assert.ErrorIs(t, err, models.ErrServiceKeyRevoked)

	require.NoError(t, svc.DeleteKey(ctx, uuid.MustParse(res.Key.ID)))
	assert.ErrorIs(t, svc.DeleteKey(ctx, uuid.MustParse(res.Key.ID)), models.ErrServiceKeyNotFound)
	assert.ErrorIs(t, svc.RevokeKey(ctx, uuid.MustParse(res.Key.ID)), models.ErrServiceKeyNotFound)
}

func TestRCServiceKey_CreateExpiryAndErrors(t *testing.T) {
	ctx := context.Background()
	user, admin := uuid.New(), uuid.New()

	t.Run("explicit expiry", func(t *testing.T) {
		days := 2
		res, err := rcService(rcNewKeyRepo()).CreateKey(ctx, user, "n", "", admin, &days)
		require.NoError(t, err)
		require.NotNil(t, res.Key.ExpiresAt)
		assert.WithinDuration(t, time.Now().AddDate(0, 0, 2), *res.Key.ExpiresAt, time.Minute)
	})
	t.Run("default expiry", func(t *testing.T) {
		svc := NewService(rcNewKeyRepo(), Config{BcryptCost: 4, DefaultExpiryDays: 30})
		res, err := svc.CreateKey(ctx, user, "n", "", admin, nil)
		require.NoError(t, err)
		require.NotNil(t, res.Key.ExpiresAt)
		assert.WithinDuration(t, time.Now().AddDate(0, 0, 30), *res.Key.ExpiresAt, time.Minute)
	})
	t.Run("no expiry", func(t *testing.T) {
		res, err := rcService(rcNewKeyRepo()).CreateKey(ctx, user, "n", "", admin, nil)
		require.NoError(t, err)
		assert.Nil(t, res.Key.ExpiresAt)
	})
	t.Run("count error", func(t *testing.T) {
		repo := rcNewKeyRepo()
		repo.errs["CountByUserID"] = rcErrRepo
		_, err := rcService(repo).CreateKey(ctx, user, "n", "", admin, nil)
		assert.ErrorIs(t, err, rcErrRepo)
	})
	t.Run("hash failure", func(t *testing.T) {
		svc := NewService(rcNewKeyRepo(), Config{BcryptCost: 99})
		_, err := svc.CreateKey(ctx, user, "n", "", admin, nil)
		assert.ErrorContains(t, err, "failed to hash key")
	})
	t.Run("validation", func(t *testing.T) {
		_, err := rcService(rcNewKeyRepo()).CreateKey(ctx, user, "", "", admin, nil)
		assert.ErrorContains(t, err, "validation failed")
	})
	t.Run("repo create error", func(t *testing.T) {
		repo := rcNewKeyRepo()
		repo.errs["Create"] = rcErrRepo
		_, err := rcService(repo).CreateKey(ctx, user, "n", "", admin, nil)
		assert.ErrorIs(t, err, rcErrRepo)
	})
}

func TestRCServiceKey_RepoErrors(t *testing.T) {
	ctx := context.Background()
	repo := rcNewKeyRepo()
	svc := rcService(repo)
	user := uuid.New()
	res, err := svc.CreateKey(ctx, user, "n", "", uuid.New(), nil)
	require.NoError(t, err)
	id := uuid.MustParse(res.Key.ID)

	repo.errs["UpdateLastUsed"] = rcErrRepo
	_, err = svc.ValidateKey(ctx, res.PlainKey)
	assert.ErrorIs(t, err, rcErrRepo)
	repo.errs["FindByPrefix"] = rcErrRepo
	_, err = svc.ValidateKey(ctx, res.PlainKey)
	assert.ErrorIs(t, err, rcErrRepo)

	repo.errs["Revoke"] = rcErrRepo
	assert.ErrorIs(t, svc.RevokeKey(ctx, id), rcErrRepo)
	repo.errs["Delete"] = rcErrRepo
	assert.ErrorIs(t, svc.DeleteKey(ctx, id), rcErrRepo)

	repo.errs["FindByID"] = rcErrRepo
	_, err = svc.GetByID(ctx, id)
	assert.ErrorIs(t, err, rcErrRepo)
	assert.ErrorIs(t, svc.RevokeKey(ctx, id), rcErrRepo)
	assert.ErrorIs(t, svc.DeleteKey(ctx, id), rcErrRepo)

	repo.errs["FindByUserID"] = rcErrRepo
	_, err = svc.GetByUserID(ctx, user)
	assert.ErrorIs(t, err, rcErrRepo)
	repo.errs["FindAll"] = rcErrRepo
	_, _, err = svc.ListAll(ctx, repository.ServiceKeyFilter{})
	assert.ErrorIs(t, err, rcErrRepo)
}
