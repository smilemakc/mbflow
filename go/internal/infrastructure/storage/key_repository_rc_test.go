package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func rcPrefix(base string) string {
	return base + uuid.NewString()[:8]
}

func TestRC_ServiceKeyRepository_Lifecycle(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewServiceKeyRepository(db)

	user := createCredentialsTestUser(t, db)
	admin := createCredentialsTestUser(t, db)
	userID := uuid.MustParse(user)
	adminID := uuid.MustParse(admin)

	newKey := func(owner, creator string, name string) *pkgmodels.ServiceKey {
		return &pkgmodels.ServiceKey{
			UserID:      owner,
			Name:        name,
			Description: "d-" + name,
			KeyPrefix:   rcPrefix("sk_abc"),
			KeyHash:     "hash-" + name,
			Status:      pkgmodels.ServiceKeyStatusActive,
			CreatedBy:   creator,
		}
	}

	k1 := newKey(user, admin, "one")
	k2 := newKey(user, user, "two")
	k3 := newKey(admin, admin, "three")
	for _, k := range []*pkgmodels.ServiceKey{k1, k2, k3} {
		require.NoError(t, repo.Create(ctx, k))
		require.NotEmpty(t, k.ID)
		require.NotEqual(t, uuid.Nil.String(), k.ID)
	}

	// prefixes are not unique (lookup returns all candidates), but must match the CHECK pattern
	dup := newKey(user, admin, "dup")
	dup.KeyPrefix = k1.KeyPrefix
	require.NoError(t, repo.Create(ctx, dup))
	candidates, err := repo.FindByPrefix(ctx, k1.KeyPrefix)
	require.NoError(t, err)
	assert.Len(t, candidates, 2)
	require.NoError(t, repo.Delete(ctx, uuid.MustParse(dup.ID)))
	badPrefix := newKey(user, admin, "bad")
	badPrefix.KeyPrefix = "nope"
	assert.Error(t, repo.Create(ctx, badPrefix))

	got, err := repo.FindByID(ctx, uuid.MustParse(k1.ID))
	require.NoError(t, err)
	assert.Equal(t, "one", got.Name)
	assert.Equal(t, user, got.UserID)
	assert.Equal(t, admin, got.CreatedBy)
	assert.Equal(t, pkgmodels.ServiceKeyStatusActive, got.Status)
	assert.Zero(t, got.UsageCount)

	_, err = repo.FindByID(ctx, uuid.New())
	assert.ErrorIs(t, err, pkgmodels.ErrServiceKeyNotFound)

	byPrefix, err := repo.FindByPrefix(ctx, k2.KeyPrefix)
	require.NoError(t, err)
	require.Len(t, byPrefix, 1)
	assert.Equal(t, k2.ID, byPrefix[0].ID)
	none, err := repo.FindByPrefix(ctx, "sk_missing")
	require.NoError(t, err)
	assert.Empty(t, none)

	byUser, err := repo.FindByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, byUser, 2)

	cnt, err := repo.CountByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), cnt)

	// FindAll with filters
	all, total, err := repo.FindAll(ctx, repository.ServiceKeyFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, all, 3)

	filtered, total, err := repo.FindAll(ctx, repository.ServiceKeyFilter{UserID: &userID, CreatedBy: &adminID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, filtered, 1)
	assert.Equal(t, k1.ID, filtered[0].ID)

	page, total, err := repo.FindAll(ctx, repository.ServiceKeyFilter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, page, 1)

	// update
	exp := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	got.Name = "one-renamed"
	got.Description = "new"
	got.ExpiresAt = &exp
	require.NoError(t, repo.Update(ctx, got))
	got, err = repo.FindByID(ctx, uuid.MustParse(k1.ID))
	require.NoError(t, err)
	assert.Equal(t, "one-renamed", got.Name)
	assert.Equal(t, "new", got.Description)
	require.NotNil(t, got.ExpiresAt)
	assert.True(t, exp.Equal(*got.ExpiresAt))
	assert.ErrorIs(t, repo.Update(ctx, &pkgmodels.ServiceKey{ID: "bad"}), pkgmodels.ErrInvalidID)

	// usage
	require.NoError(t, repo.UpdateLastUsed(ctx, uuid.MustParse(k1.ID)))
	require.NoError(t, repo.UpdateLastUsed(ctx, uuid.MustParse(k1.ID)))
	got, _ = repo.FindByID(ctx, uuid.MustParse(k1.ID))
	assert.Equal(t, int64(2), got.UsageCount)
	assert.NotNil(t, got.LastUsedAt)

	// revoke
	require.NoError(t, repo.Revoke(ctx, uuid.MustParse(k1.ID)))
	got, _ = repo.FindByID(ctx, uuid.MustParse(k1.ID))
	assert.Equal(t, pkgmodels.ServiceKeyStatusRevoked, got.Status)
	assert.NotNil(t, got.RevokedAt)

	revoked := pkgmodels.ServiceKeyStatusRevoked
	filtered, total, err = repo.FindAll(ctx, repository.ServiceKeyFilter{Status: &revoked})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, filtered, 1)

	// delete
	require.NoError(t, repo.Delete(ctx, uuid.MustParse(k1.ID)))
	_, err = repo.FindByID(ctx, uuid.MustParse(k1.ID))
	assert.ErrorIs(t, err, pkgmodels.ErrServiceKeyNotFound)
}

func TestRC_SystemKeyRepository_Lifecycle(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewSystemKeyRepo(db)

	admin := createCredentialsTestUser(t, db)
	other := createCredentialsTestUser(t, db)
	adminID := uuid.MustParse(admin)

	newKey := func(service, creator, name string) *pkgmodels.SystemKey {
		return &pkgmodels.SystemKey{
			Name:        name,
			Description: "d-" + name,
			KeyPrefix:   rcPrefix("sysk_abc"),
			KeyHash:     "hash",
			ServiceName: service,
			Status:      pkgmodels.SystemKeyStatusActive,
			CreatedBy:   creator,
		}
	}
	k1 := newKey("billing", admin, "one")
	k2 := newKey("billing", other, "two")
	k3 := newKey("search", admin, "three")
	for _, k := range []*pkgmodels.SystemKey{k1, k2, k3} {
		require.NoError(t, repo.Create(ctx, k))
		require.NotEmpty(t, k.ID)
	}
	dup := newKey("billing", admin, "dup")
	dup.KeyPrefix = k1.KeyPrefix
	assert.Error(t, repo.Create(ctx, dup))

	got, err := repo.FindByID(ctx, uuid.MustParse(k1.ID))
	require.NoError(t, err)
	assert.Equal(t, "billing", got.ServiceName)
	assert.Equal(t, admin, got.CreatedBy)
	_, err = repo.FindByID(ctx, uuid.New())
	assert.ErrorIs(t, err, pkgmodels.ErrSystemKeyNotFound)

	byPrefix, err := repo.FindByPrefix(ctx, k3.KeyPrefix)
	require.NoError(t, err)
	require.Len(t, byPrefix, 1)
	assert.Equal(t, "search", byPrefix[0].ServiceName)

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	billing := "billing"
	list, total, err := repo.FindAll(ctx, repository.SystemKeyFilter{ServiceName: &billing})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, list, 2)

	list, total, err = repo.FindAll(ctx, repository.SystemKeyFilter{ServiceName: &billing, CreatedBy: &adminID})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, k1.ID, list[0].ID)

	list, total, err = repo.FindAll(ctx, repository.SystemKeyFilter{Limit: 1, Offset: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, list, 1)

	got.Name = "renamed"
	require.NoError(t, repo.Update(ctx, got))
	got, _ = repo.FindByID(ctx, uuid.MustParse(k1.ID))
	assert.Equal(t, "renamed", got.Name)
	assert.ErrorIs(t, repo.Update(ctx, &pkgmodels.SystemKey{ID: "bad"}), pkgmodels.ErrInvalidID)

	require.NoError(t, repo.UpdateLastUsed(ctx, uuid.MustParse(k1.ID)))
	got, _ = repo.FindByID(ctx, uuid.MustParse(k1.ID))
	assert.Equal(t, int64(1), got.UsageCount)
	assert.NotNil(t, got.LastUsedAt)

	require.NoError(t, repo.Revoke(ctx, uuid.MustParse(k1.ID)))
	revoked := pkgmodels.SystemKeyStatusRevoked
	list, total, err = repo.FindAll(ctx, repository.SystemKeyFilter{Status: &revoked})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.NotNil(t, list[0].RevokedAt)

	require.NoError(t, repo.Delete(ctx, uuid.MustParse(k1.ID)))
	count, _ = repo.Count(ctx)
	assert.Equal(t, int64(2), count)
}

func TestRC_ServiceAuditLogRepository(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	keyRepo := NewSystemKeyRepo(db)
	repo := NewServiceAuditLogRepo(db)

	admin := createCredentialsTestUser(t, db)
	key := &pkgmodels.SystemKey{
		Name: "k", KeyPrefix: rcPrefix("sysk_abc"), KeyHash: "h",
		ServiceName: "svc", Status: pkgmodels.SystemKeyStatusActive, CreatedBy: admin,
	}
	require.NoError(t, keyRepo.Create(ctx, key))
	keyID := uuid.MustParse(key.ID)

	impersonated := admin
	resID := uuid.NewString()
	body := `{"a":1}`
	mk := func(action, resourceType string) *pkgmodels.ServiceAuditLog {
		return &pkgmodels.ServiceAuditLog{
			SystemKeyID:    key.ID,
			ServiceName:    "svc",
			Action:         action,
			ResourceType:   resourceType,
			RequestMethod:  "POST",
			RequestPath:    "/api/v1/" + resourceType,
			ResponseStatus: 200,
			IPAddress:      "10.0.0.1",
		}
	}
	l1 := mk("create", "workflow")
	l1.ImpersonatedUserID = &impersonated
	l1.ResourceID = &resID
	l1.RequestBody = &body
	l2 := mk("delete", "workflow")
	l3 := mk("create", "trigger")
	for _, l := range []*pkgmodels.ServiceAuditLog{l1, l2, l3} {
		require.NoError(t, repo.Create(ctx, l))
		require.NotEmpty(t, l.ID)
	}
	// FK violation
	bad := mk("x", "y")
	bad.SystemKeyID = uuid.NewString()
	assert.Error(t, repo.Create(ctx, bad))

	all, total, err := repo.FindAll(ctx, repository.ServiceAuditLogFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, all, 3)

	svc := "svc"
	create := "create"
	wf := "workflow"
	impUUID := uuid.MustParse(admin)
	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)
	list, total, err := repo.FindAll(ctx, repository.ServiceAuditLogFilter{
		SystemKeyID:        &keyID,
		ServiceName:        &svc,
		Action:             &create,
		ResourceType:       &wf,
		ImpersonatedUserID: &impUUID,
		DateFrom:           &from,
		DateTo:             &to,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, list, 1)
	assert.Equal(t, l1.ID, list[0].ID)
	require.NotNil(t, list[0].RequestBody)
	assert.JSONEq(t, body, *list[0].RequestBody)
	require.NotNil(t, list[0].ResourceID)
	assert.Equal(t, resID, *list[0].ResourceID)

	list, total, err = repo.FindAll(ctx, repository.ServiceAuditLogFilter{Limit: 2, Offset: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, list, 2)

	// nothing older than an hour ago
	n, err := repo.DeleteOlderThan(ctx, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Zero(t, n)
	n, err = repo.DeleteOlderThan(ctx, time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}
