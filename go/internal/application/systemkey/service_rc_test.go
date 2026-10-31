package systemkey

import (
	"context"
	"encoding/json"
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

var errRCRepo = errors.New("repo failure")

// rcKeyRepo is an in-memory repository.SystemKeyRepository with error injection.
type rcKeyRepo struct {
	mu   sync.Mutex
	keys map[string]*models.SystemKey
	errs map[string]error
	// count overrides the number of keys reported by Count when >= 0.
	count int64
}

func rcNewKeyRepo() *rcKeyRepo {
	return &rcKeyRepo{keys: map[string]*models.SystemKey{}, errs: map[string]error{}, count: -1}
}

func (r *rcKeyRepo) Create(ctx context.Context, key *models.SystemKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Create"]; err != nil {
		return err
	}
	r.keys[key.ID] = key
	return nil
}

func (r *rcKeyRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.SystemKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["FindByID"]; err != nil {
		return nil, err
	}
	return r.keys[id.String()], nil
}

func (r *rcKeyRepo) FindByPrefix(ctx context.Context, prefix string) ([]*models.SystemKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["FindByPrefix"]; err != nil {
		return nil, err
	}
	var out []*models.SystemKey
	for _, k := range r.keys {
		if k.KeyPrefix == prefix {
			out = append(out, k)
		}
	}
	return out, nil
}

func (r *rcKeyRepo) FindAll(ctx context.Context, filter repository.SystemKeyFilter) ([]*models.SystemKey, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["FindAll"]; err != nil {
		return nil, 0, err
	}
	var out []*models.SystemKey
	for _, k := range r.keys {
		if filter.ServiceName != nil && k.ServiceName != *filter.ServiceName {
			continue
		}
		out = append(out, k)
	}
	return out, int64(len(out)), nil
}

func (r *rcKeyRepo) Update(ctx context.Context, key *models.SystemKey) error { return nil }

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
	r.keys[id.String()].Status = models.SystemKeyStatusRevoked
	return nil
}

func (r *rcKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.errs["UpdateLastUsed"]
}

func (r *rcKeyRepo) Count(ctx context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Count"]; err != nil {
		return 0, err
	}
	if r.count >= 0 {
		return r.count, nil
	}
	return int64(len(r.keys)), nil
}

func rcService(repo *rcKeyRepo) *Service {
	return NewService(repo, Config{MaxKeys: 5, BcryptCost: 4})
}

func TestRCSystemKey_CreateValidateRevokeLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := rcNewKeyRepo()
	svc := rcService(repo)
	admin := uuid.New()

	res, err := svc.CreateKey(ctx, "billing", "billing service", "billing-svc", admin, nil)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(res.PlainKey, models.SystemKeyPrefix))
	assert.Equal(t, res.PlainKey[:models.SystemKeyPrefixLength], res.Key.KeyPrefix)
	assert.NotContains(t, res.Key.KeyHash, res.PlainKey, "only the hash is stored")
	assert.Nil(t, res.Key.ExpiresAt)

	got, err := svc.ValidateKey(ctx, res.PlainKey)
	require.NoError(t, err)
	assert.Equal(t, res.Key.ID, got.ID)
	assert.Equal(t, int64(1), got.UsageCount)
	assert.NotNil(t, got.LastUsedAt)

	// Same prefix, different secret → invalid
	tampered := res.PlainKey[:len(res.PlainKey)-1] + "X"
	if tampered == res.PlainKey {
		tampered = res.PlainKey[:len(res.PlainKey)-1] + "Y"
	}
	_, err = svc.ValidateKey(ctx, tampered)
	assert.ErrorIs(t, err, models.ErrInvalidSystemKey)

	byID, err := svc.GetByID(ctx, uuid.MustParse(res.Key.ID))
	require.NoError(t, err)
	assert.Same(t, res.Key, byID)

	svcName := "billing-svc"
	list, total, err := svc.ListAll(ctx, repository.SystemKeyFilter{ServiceName: &svcName})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, list, 1)

	require.NoError(t, svc.RevokeKey(ctx, uuid.MustParse(res.Key.ID)))
	_, err = svc.ValidateKey(ctx, res.PlainKey)
	assert.ErrorIs(t, err, models.ErrSystemKeyRevoked)

	require.NoError(t, svc.DeleteKey(ctx, uuid.MustParse(res.Key.ID)))
	_, err = svc.GetByID(ctx, uuid.MustParse(res.Key.ID))
	assert.ErrorIs(t, err, models.ErrSystemKeyNotFound)
}

func TestRCSystemKey_CreateExpiryAndErrors(t *testing.T) {
	ctx := context.Background()
	admin := uuid.New()

	t.Run("explicit expiry", func(t *testing.T) {
		svc := rcService(rcNewKeyRepo())
		days := 3
		res, err := svc.CreateKey(ctx, "n", "", "svc", admin, &days)
		require.NoError(t, err)
		require.NotNil(t, res.Key.ExpiresAt)
		assert.WithinDuration(t, time.Now().AddDate(0, 0, 3), *res.Key.ExpiresAt, time.Minute)
	})
	t.Run("default expiry", func(t *testing.T) {
		svc := NewService(rcNewKeyRepo(), Config{BcryptCost: 4, DefaultExpiryDays: 7})
		zero := 0
		res, err := svc.CreateKey(ctx, "n", "", "svc", admin, &zero)
		require.NoError(t, err)
		require.NotNil(t, res.Key.ExpiresAt)
		assert.WithinDuration(t, time.Now().AddDate(0, 0, 7), *res.Key.ExpiresAt, time.Minute)
	})
	t.Run("count error", func(t *testing.T) {
		repo := rcNewKeyRepo()
		repo.errs["Count"] = errRCRepo
		_, err := rcService(repo).CreateKey(ctx, "n", "", "svc", admin, nil)
		assert.ErrorIs(t, err, errRCRepo)
	})
	t.Run("limit reached", func(t *testing.T) {
		repo := rcNewKeyRepo()
		repo.count = 5
		_, err := rcService(repo).CreateKey(ctx, "n", "", "svc", admin, nil)
		assert.ErrorIs(t, err, models.ErrSystemKeyLimitReached)
	})
	t.Run("hash failure", func(t *testing.T) {
		svc := NewService(rcNewKeyRepo(), Config{BcryptCost: 99})
		_, err := svc.CreateKey(ctx, "n", "", "svc", admin, nil)
		assert.ErrorContains(t, err, "failed to hash key")
	})
	t.Run("validation", func(t *testing.T) {
		_, err := rcService(rcNewKeyRepo()).CreateKey(ctx, "n", "", "  ", admin, nil)
		assert.ErrorContains(t, err, "validation failed")
	})
	t.Run("repo create error", func(t *testing.T) {
		repo := rcNewKeyRepo()
		repo.errs["Create"] = errRCRepo
		_, err := rcService(repo).CreateKey(ctx, "n", "", "svc", admin, nil)
		assert.ErrorIs(t, err, errRCRepo)
	})
}

func TestRCSystemKey_RepoErrors(t *testing.T) {
	ctx := context.Background()
	repo := rcNewKeyRepo()
	svc := rcService(repo)
	res, err := svc.CreateKey(ctx, "n", "", "svc", uuid.New(), nil)
	require.NoError(t, err)
	id := uuid.MustParse(res.Key.ID)

	repo.errs["UpdateLastUsed"] = errRCRepo
	_, err = svc.ValidateKey(ctx, res.PlainKey)
	assert.ErrorIs(t, err, errRCRepo)

	repo.errs["FindByPrefix"] = errRCRepo
	_, err = svc.ValidateKey(ctx, res.PlainKey)
	assert.ErrorIs(t, err, errRCRepo)

	repo.errs["Revoke"] = errRCRepo
	assert.ErrorIs(t, svc.RevokeKey(ctx, id), errRCRepo)
	repo.errs["Delete"] = errRCRepo
	assert.ErrorIs(t, svc.DeleteKey(ctx, id), errRCRepo)

	assert.ErrorIs(t, svc.RevokeKey(ctx, uuid.New()), models.ErrSystemKeyNotFound)
	assert.ErrorIs(t, svc.DeleteKey(ctx, uuid.New()), models.ErrSystemKeyNotFound)

	repo.errs["FindByID"] = errRCRepo
	_, err = svc.GetByID(ctx, id)
	assert.ErrorIs(t, err, errRCRepo)
	assert.ErrorIs(t, svc.RevokeKey(ctx, id), errRCRepo)
	assert.ErrorIs(t, svc.DeleteKey(ctx, id), errRCRepo)

	repo.errs["FindAll"] = errRCRepo
	_, _, err = svc.ListAll(ctx, repository.SystemKeyFilter{})
	assert.ErrorIs(t, err, errRCRepo)
}

// ---------------------------------------------------------------------------
// Audit service
// ---------------------------------------------------------------------------

type rcAuditRepo struct {
	mu      sync.Mutex
	entries []*models.ServiceAuditLog
	before  time.Time
	err     error
}

func (r *rcAuditRepo) Create(ctx context.Context, log *models.ServiceAuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.entries = append(r.entries, log)
	return nil
}

func (r *rcAuditRepo) FindAll(ctx context.Context, filter repository.ServiceAuditLogFilter) ([]*models.ServiceAuditLog, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return nil, 0, r.err
	}
	return r.entries, int64(len(r.entries)), nil
}

func (r *rcAuditRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return 0, r.err
	}
	r.before = before
	return 7, nil
}

func TestRCAuditService_LogAction(t *testing.T) {
	ctx := context.Background()
	repo := &rcAuditRepo{}
	svc := NewAuditService(repo, 30)

	resID, userID := "wf-1", "user-1"
	body := `{"name":"wf","api_key":"sk-1","nested":{"Password":"p","ok":1},"items":[{"token":"t","v":2},"plain"],"refreshToken":"r"}`
	require.NoError(t, svc.LogAction(ctx, "sys-1", "billing", "workflow.create", "workflow", &resID, &userID, "POST", "/api/v1/workflows", &body, "10.0.0.1", 201))

	require.Len(t, repo.entries, 1)
	e := repo.entries[0]
	assert.Equal(t, "sys-1", e.SystemKeyID)
	assert.Equal(t, "billing", e.ServiceName)
	assert.Equal(t, &resID, e.ResourceID)
	assert.Equal(t, &userID, e.ImpersonatedUserID)
	assert.Equal(t, 201, e.ResponseStatus)
	require.NotNil(t, e.RequestBody)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(*e.RequestBody), &got))
	assert.Equal(t, "wf", got["name"])
	assert.Equal(t, "[REDACTED]", got["api_key"])
	assert.Equal(t, "[REDACTED]", got["refreshToken"])
	nested := got["nested"].(map[string]any)
	assert.Equal(t, "[REDACTED]", nested["Password"], "matching is case-insensitive")
	assert.EqualValues(t, 1, nested["ok"])
	items := got["items"].([]any)
	assert.Equal(t, "[REDACTED]", items[0].(map[string]any)["token"], "objects inside arrays are sanitized")
	assert.EqualValues(t, 2, items[0].(map[string]any)["v"])
	assert.Equal(t, "plain", items[1])
	assert.NotContains(t, *e.RequestBody, "sk-1")

	// Without body
	require.NoError(t, svc.LogAction(ctx, "sys-1", "billing", "a", "r", nil, nil, "GET", "/", nil, "ip", 200))
	assert.Nil(t, repo.entries[1].RequestBody)

	// Non-JSON body is kept as is
	raw := "not json"
	require.NoError(t, svc.LogAction(ctx, "sys-1", "billing", "a", "r", nil, nil, "POST", "/", &raw, "ip", 400))
	assert.Equal(t, "not json", *repo.entries[2].RequestBody)

	repo.err = errRCRepo
	err := svc.LogAction(ctx, "sys-1", "billing", "a", "r", nil, nil, "GET", "/", nil, "ip", 200)
	assert.ErrorIs(t, err, errRCRepo)
}

func TestRCAuditService_ListAndCleanup(t *testing.T) {
	ctx := context.Background()
	repo := &rcAuditRepo{}

	assert.Equal(t, DefaultRetentionDays, NewAuditService(repo, 0).retentionDays)

	svc := NewAuditService(repo, 10)
	require.NoError(t, svc.LogAction(ctx, "k", "s", "a", "r", nil, nil, "GET", "/", nil, "ip", 200))
	logs, total, err := svc.ListLogs(ctx, repository.ServiceAuditLogFilter{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, int64(1), total)

	deleted, err := svc.Cleanup(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(7), deleted)
	assert.WithinDuration(t, time.Now().AddDate(0, 0, -10), repo.before, time.Minute)

	repo.err = errRCRepo
	_, _, err = svc.ListLogs(ctx, repository.ServiceAuditLogFilter{})
	assert.ErrorIs(t, err, errRCRepo)
	_, err = svc.Cleanup(ctx)
	assert.ErrorIs(t, err, errRCRepo)
}
