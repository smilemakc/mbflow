package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func rcNewFileStorage(ownerID, name string) *pkgmodels.FileStorageResource {
	return &pkgmodels.FileStorageResource{
		BaseResource: pkgmodels.BaseResource{
			Type:        pkgmodels.ResourceTypeFileStorage,
			OwnerID:     ownerID,
			Name:        name,
			Description: "desc " + name,
			Status:      pkgmodels.ResourceStatusActive,
			Metadata:    map[string]any{"name": name},
		},
		StorageLimitBytes: 1000,
	}
}

func rcNewCredentials(ownerID, name, provider string) *pkgmodels.CredentialsResource {
	return &pkgmodels.CredentialsResource{
		BaseResource: pkgmodels.BaseResource{
			Type:    pkgmodels.ResourceTypeCredentials,
			OwnerID: ownerID,
			Name:    name,
			Status:  pkgmodels.ResourceStatusActive,
		},
		CredentialType: pkgmodels.CredentialTypeAPIKey,
		EncryptedData:  map[string]string{"api_key": "enc-" + name},
		Provider:       provider,
	}
}

func TestRC_ResourceRepository_FileStorageLifecycle(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewResourceRepository(db)
	owner := createCredentialsTestUser(t, db)

	plan, err := NewPricingPlanRepository(db).GetFreePlan(ctx, pkgmodels.ResourceTypeFileStorage)
	require.NoError(t, err)

	fs := rcNewFileStorage(owner, "store-1")
	fs.PricingPlanID = plan.ID
	require.NoError(t, repo.Create(ctx, fs))
	require.NotEmpty(t, fs.ID)

	got, err := repo.GetFileStorage(ctx, fs.ID)
	require.NoError(t, err)
	assert.Equal(t, "store-1", got.Name)
	assert.Equal(t, owner, got.OwnerID)
	assert.Equal(t, int64(1000), got.StorageLimitBytes)
	assert.Equal(t, plan.ID, got.PricingPlanID)
	assert.Equal(t, "store-1", got.Metadata["name"])

	// usage accounting
	require.NoError(t, repo.IncrementUsage(ctx, fs.ID, 300))
	require.NoError(t, repo.IncrementUsage(ctx, fs.ID, 200))
	got, _ = repo.GetFileStorage(ctx, fs.ID)
	assert.Equal(t, int64(500), got.UsedStorageBytes)
	assert.Equal(t, 2, got.FileCount)

	require.NoError(t, repo.DecrementUsage(ctx, fs.ID, 1000))
	require.NoError(t, repo.DecrementUsage(ctx, fs.ID, 1))
	require.NoError(t, repo.DecrementUsage(ctx, fs.ID, 1))
	got, _ = repo.GetFileStorage(ctx, fs.ID)
	assert.Equal(t, int64(0), got.UsedStorageBytes, "usage never drops below zero")
	assert.Equal(t, 0, got.FileCount)

	require.NoError(t, repo.UpdateUsage(ctx, fs.ID, 77, 7))
	got, _ = repo.GetFileStorage(ctx, fs.ID)
	assert.Equal(t, int64(77), got.UsedStorageBytes)
	assert.Equal(t, 7, got.FileCount)

	// full update
	got.Name = "renamed"
	got.Description = "new desc"
	got.Status = pkgmodels.ResourceStatusSuspended
	got.Metadata = map[string]any{"v": 2.0}
	got.StorageLimitBytes = 2000
	got.UsedStorageBytes = 10
	got.FileCount = 1
	require.NoError(t, repo.Update(ctx, got))
	upd, err := repo.GetFileStorage(ctx, fs.ID)
	require.NoError(t, err)
	assert.Equal(t, "renamed", upd.Name)
	assert.Equal(t, "new desc", upd.Description)
	assert.Equal(t, pkgmodels.ResourceStatusSuspended, upd.Status)
	assert.Equal(t, 2.0, upd.Metadata["v"])
	assert.Equal(t, int64(2000), upd.StorageLimitBytes)
	assert.Equal(t, int64(10), upd.UsedStorageBytes)
	assert.Equal(t, 1, upd.FileCount)

	// a CHECK violation in the type-specific table aborts the update
	upd.StorageLimitBytes = 0
	assert.Error(t, repo.Update(ctx, upd))

	// soft delete hides the resource
	require.NoError(t, repo.Delete(ctx, fs.ID))
	_, err = repo.GetByID(ctx, fs.ID)
	assert.ErrorIs(t, err, pkgmodels.ErrResourceNotFound)
	_, err = repo.GetFileStorage(ctx, fs.ID)
	assert.ErrorIs(t, err, pkgmodels.ErrResourceNotFound)

	var status string
	require.NoError(t, db.NewRaw("SELECT status FROM mbflow_resources WHERE id = ?", fs.ID).Scan(ctx, &status))
	assert.Equal(t, string(pkgmodels.ResourceStatusDeleted), status)

	// hard delete removes the row and cascades
	require.NoError(t, repo.HardDelete(ctx, fs.ID))
	var n int
	require.NoError(t, db.NewRaw("SELECT count(*) FROM mbflow_resource_file_storage WHERE resource_id = ?", fs.ID).Scan(ctx, &n))
	assert.Zero(t, n)
}

func TestRC_ResourceRepository_MixedTypesAndQueries(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewResourceRepository(db)
	credRepo := NewCredentialsRepository(db)
	owner := createCredentialsTestUser(t, db)
	other := createCredentialsTestUser(t, db)

	fs := rcNewFileStorage(owner, "files")
	require.NoError(t, repo.Create(ctx, fs))
	require.NoError(t, repo.Create(ctx, rcNewFileStorage(other, "other-files")))

	cred := rcNewCredentials(owner, "openai", "openai")
	require.NoError(t, credRepo.CreateCredentials(ctx, cred))

	enc := rcEncryption(t)
	rk := rcNewRentalKey(owner, "rk", "openai")
	require.NoError(t, NewRentalKeyRepository(db, enc).CreateRentalKey(ctx, rk, "sk-plain"))

	// non file-storage resources cannot be created through this repo
	assert.ErrorIs(t, repo.Create(ctx, cred), pkgmodels.ErrInvalidResourceType)

	// GetByID returns the concrete type for each resource kind
	r, err := repo.GetByID(ctx, fs.ID)
	require.NoError(t, err)
	assert.IsType(t, &pkgmodels.FileStorageResource{}, r)
	r, err = repo.GetByID(ctx, cred.ID)
	require.NoError(t, err)
	require.IsType(t, &pkgmodels.CredentialsResource{}, r)
	assert.Equal(t, "openai", r.(*pkgmodels.CredentialsResource).Provider)
	r, err = repo.GetByID(ctx, rk.ID)
	require.NoError(t, err)
	assert.IsType(t, &pkgmodels.RentalKeyResource{}, r)

	_, err = repo.GetFileStorage(ctx, cred.ID)
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidResourceType)

	all, err := repo.GetByOwner(ctx, owner)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	onlyFS, err := repo.GetByOwnerAndType(ctx, owner, pkgmodels.ResourceTypeFileStorage)
	require.NoError(t, err)
	require.Len(t, onlyFS, 1)
	assert.Equal(t, fs.ID, onlyFS[0].GetID())

	onlyCred, err := repo.GetByOwnerAndType(ctx, owner, pkgmodels.ResourceTypeCredentials)
	require.NoError(t, err)
	require.Len(t, onlyCred, 1)

	// Update through the generic repository also updates credential fields
	c := onlyCred[0].(*pkgmodels.CredentialsResource)
	c.Name = "openai-2"
	c.EncryptedData = map[string]string{"api_key": "rotated"}
	c.Provider = ""
	c.CredentialType = pkgmodels.CredentialTypeBasicAuth
	require.NoError(t, repo.Update(ctx, c))
	updated, err := credRepo.GetCredentials(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "openai-2", updated.Name)
	assert.Equal(t, "rotated", updated.EncryptedData["api_key"])
	assert.Empty(t, updated.Provider)
	assert.Equal(t, pkgmodels.CredentialTypeBasicAuth, updated.CredentialType)

	// A resource row without its type-specific row is skipped
	orphanID := uuid.New()
	_, err = db.NewRaw(
		"INSERT INTO mbflow_resources (id, type, owner_id, name, status) VALUES (?, 'file_storage', ?, 'orphan', 'active')",
		orphanID, owner,
	).Exec(ctx)
	require.NoError(t, err)
	all, err = repo.GetByOwner(ctx, owner)
	require.NoError(t, err)
	assert.Len(t, all, 3, "orphan resource without file storage row is skipped")
	got, err := repo.GetByID(ctx, orphanID.String())
	require.NoError(t, err)
	assert.Nil(t, got)

	// invalid ids
	_, err = repo.GetByID(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, err = repo.GetByOwner(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, err = repo.GetByOwnerAndType(ctx, "bad", pkgmodels.ResourceTypeFileStorage)
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.Update(ctx, &pkgmodels.FileStorageResource{}), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.Delete(ctx, "bad"), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.HardDelete(ctx, "bad"), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.UpdateUsage(ctx, "bad", 0, 0), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.IncrementUsage(ctx, "bad", 0), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.DecrementUsage(ctx, "bad", 0), pkgmodels.ErrInvalidID)
	_, err = repo.GetFileStorage(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)

	// owner FK violation
	assert.Error(t, repo.Create(ctx, rcNewFileStorage(uuid.NewString(), "x")))
}

// ---------------------------------------------------------------------------
// Rental keys
// ---------------------------------------------------------------------------

func rcEncryption(t *testing.T) *crypto.EncryptionService {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	enc, err := crypto.NewEncryptionService(key)
	require.NoError(t, err)
	return enc
}

func rcNewRentalKey(ownerID, name string, provider pkgmodels.LLMProviderType) *pkgmodels.RentalKeyResource {
	daily := 100
	monthly := int64(10000)
	return &pkgmodels.RentalKeyResource{
		BaseResource: pkgmodels.BaseResource{
			Type:    pkgmodels.ResourceTypeRentalKey,
			OwnerID: ownerID,
			Name:    name,
			Status:  pkgmodels.ResourceStatusActive,
		},
		Provider:          provider,
		ProviderConfig:    map[string]any{"model": "x"},
		DailyRequestLimit: &daily,
		MonthlyTokenLimit: &monthly,
		CreatedBy:         ownerID,
		ProvisionerType:   pkgmodels.ProvisionerType("manual"),
	}
}

func TestRC_RentalKeyRepository_Lifecycle(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	enc := rcEncryption(t)
	repo := NewRentalKeyRepository(db, enc)
	owner := createCredentialsTestUser(t, db)

	plans, err := NewPricingPlanRepository(db).GetByResourceType(ctx, pkgmodels.ResourceTypeRentalKey)
	require.NoError(t, err)
	require.NotEmpty(t, plans)

	key := rcNewRentalKey(owner, "main", "openai")
	key.PricingPlanID = plans[0].ID
	require.NoError(t, repo.CreateRentalKey(ctx, key, "sk-secret-1"))
	require.NotEmpty(t, key.ID)
	assert.NotEqual(t, "sk-secret-1", key.EncryptedAPIKey, "API key must be stored encrypted")

	got, err := repo.GetRentalKey(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, "main", got.Name)
	assert.Equal(t, pkgmodels.LLMProviderType("openai"), got.Provider)
	assert.Equal(t, "x", got.ProviderConfig["model"])
	assert.Equal(t, 100, *got.DailyRequestLimit)
	assert.Equal(t, int64(10000), *got.MonthlyTokenLimit)
	assert.Equal(t, owner, got.CreatedBy)
	assert.Equal(t, plans[0].ID, got.PricingPlanID)

	plain, err := repo.GetDecryptedAPIKey(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, "sk-secret-1", plain)

	require.NoError(t, repo.RotateAPIKey(ctx, key.ID, "sk-secret-2"))
	plain, err = repo.GetDecryptedAPIKey(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, "sk-secret-2", plain)

	// update
	newDaily := 5
	got.Name = "renamed"
	got.Provider = "anthropic"
	got.ProviderConfig = nil
	got.DailyRequestLimit = &newDaily
	got.PricingPlanID = ""
	require.NoError(t, repo.UpdateRentalKey(ctx, got))
	upd, err := repo.GetRentalKey(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, "renamed", upd.Name)
	assert.Equal(t, pkgmodels.LLMProviderType("anthropic"), upd.Provider)
	assert.Equal(t, 5, *upd.DailyRequestLimit)
	assert.Empty(t, upd.PricingPlanID)

	// invalid provider violates CHECK constraint
	upd.Provider = "nope"
	assert.Error(t, repo.UpdateRentalKey(ctx, upd))
	upd.Provider = "anthropic"
	upd.PricingPlanID = plans[1].ID
	require.NoError(t, repo.UpdateRentalKey(ctx, upd))

	// delete
	require.NoError(t, repo.DeleteRentalKey(ctx, key.ID))
	_, err = repo.GetRentalKey(ctx, key.ID)
	assert.ErrorIs(t, err, pkgmodels.ErrRentalKeyNotFound)

	// not found / invalid
	_, err = repo.GetRentalKey(ctx, uuid.NewString())
	assert.ErrorIs(t, err, pkgmodels.ErrRentalKeyNotFound)
	_, err = repo.GetDecryptedAPIKey(ctx, uuid.NewString())
	assert.ErrorIs(t, err, pkgmodels.ErrRentalKeyNotFound)
	_, err = repo.GetRentalKey(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, err = repo.GetDecryptedAPIKey(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.RotateAPIKey(ctx, "bad", "x"), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.UpdateRentalKey(ctx, &pkgmodels.RentalKeyResource{}), pkgmodels.ErrInvalidID)
	assert.ErrorIs(t, repo.DeleteRentalKey(ctx, "bad"), pkgmodels.ErrInvalidID)
	_, err = repo.GetRentalKeysByOwner(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, err = repo.GetRentalKeysByProvider(ctx, "bad", "openai")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)

	// invalid provider on create violates CHECK and rolls back the resource row
	bad := rcNewRentalKey(owner, "bad", "nope")
	assert.Error(t, repo.CreateRentalKey(ctx, bad, "x"))
	keys, err := repo.GetRentalKeysByOwner(ctx, owner)
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestRC_RentalKeyRepository_DecryptWithWrongKeyFails(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	owner := createCredentialsTestUser(t, db)

	key := rcNewRentalKey(owner, "k", "openai")
	require.NoError(t, NewRentalKeyRepository(db, rcEncryption(t)).CreateRentalKey(ctx, key, "secret"))

	_, err := NewRentalKeyRepository(db, rcEncryption(t)).GetDecryptedAPIKey(ctx, key.ID)
	assert.Error(t, err)
}

func TestRC_RentalKeyRepository_ListingAndFilters(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewRentalKeyRepository(db, rcEncryption(t))
	owner := createCredentialsTestUser(t, db)
	other := createCredentialsTestUser(t, db)

	k1 := rcNewRentalKey(owner, "openai-1", "openai")
	k2 := rcNewRentalKey(owner, "anthropic-1", "anthropic")
	k3 := rcNewRentalKey(other, "openai-other", "openai")
	k3.Status = pkgmodels.ResourceStatusSuspended
	k3.CreatedBy = ""
	for _, k := range []*pkgmodels.RentalKeyResource{k1, k2, k3} {
		require.NoError(t, repo.CreateRentalKey(ctx, k, "plain-"+k.Name))
	}

	byOwner, err := repo.GetRentalKeysByOwner(ctx, owner)
	require.NoError(t, err)
	assert.Len(t, byOwner, 2)

	byProvider, err := repo.GetRentalKeysByProvider(ctx, owner, "anthropic")
	require.NoError(t, err)
	require.Len(t, byProvider, 1)
	assert.Equal(t, k2.ID, byProvider[0].ID)

	all, total, err := repo.GetAllRentalKeys(ctx, repository.RentalKeyFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, all, 3)

	openai := pkgmodels.LLMProviderType("openai")
	suspended := pkgmodels.ResourceStatusSuspended
	ownerStr := owner
	badStr := "not-a-uuid"

	filtered, total, err := repo.GetAllRentalKeys(ctx, repository.RentalKeyFilter{Provider: &openai})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, filtered, 2)

	filtered, total, err = repo.GetAllRentalKeys(ctx, repository.RentalKeyFilter{Status: &suspended})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, filtered, 1)
	assert.Equal(t, k3.ID, filtered[0].ID)

	filtered, total, err = repo.GetAllRentalKeys(ctx, repository.RentalKeyFilter{OwnerID: &ownerStr, CreatedBy: &ownerStr})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, filtered, 2)

	// unparsable ids are ignored rather than failing
	_, total, err = repo.GetAllRentalKeys(ctx, repository.RentalKeyFilter{OwnerID: &badStr, CreatedBy: &badStr})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)

	page, total, err := repo.GetAllRentalKeys(ctx, repository.RentalKeyFilter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total, "total ignores pagination")
	assert.Len(t, page, 1)

	count, err := repo.GetAllRentalKeysCount(ctx, repository.RentalKeyFilter{Provider: &openai, Status: &suspended})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestRC_RentalKeyRepository_UsageTracking(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewRentalKeyRepository(db, rcEncryption(t))
	owner := createCredentialsTestUser(t, db)

	key := rcNewRentalKey(owner, "k", "openai")
	require.NoError(t, repo.CreateRentalKey(ctx, key, "secret"))

	u1 := &pkgmodels.RentalKeyUsageRecord{
		Model:          "gpt-4",
		Usage:          pkgmodels.MultimodalUsage{PromptTokens: 10, CompletionTokens: 5, ImageInputTokens: 1},
		EstimatedCost:  0.5,
		Status:         "success",
		NodeID:         "n1",
		ResponseTimeMs: 12,
	}
	u2 := &pkgmodels.RentalKeyUsageRecord{
		Model:         "gpt-4o",
		Usage:         pkgmodels.MultimodalUsage{PromptTokens: 1, AudioInputTokens: 2, AudioOutputTokens: 3, VideoInputTokens: 4, VideoOutputTokens: 5, ImageOutputTokens: 6},
		EstimatedCost: 0.25,
		Status:        "failed",
		ErrorMessage:  "boom",
	}
	require.NoError(t, repo.RecordUsage(ctx, key.ID, u1))
	require.NoError(t, repo.RecordUsage(ctx, key.ID, u2))
	assert.NotEmpty(t, u1.ID)
	assert.Equal(t, key.ID, u1.RentalKeyID)

	got, err := repo.GetRentalKey(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, got.RequestsToday)
	assert.Equal(t, int64(2), got.TotalRequests)
	assert.Equal(t, int64(16+21), got.TokensThisMonth)
	assert.Equal(t, int64(11), got.TotalUsage.PromptTokens)
	assert.Equal(t, int64(5), got.TotalUsage.CompletionTokens)
	assert.Equal(t, int64(1), got.TotalUsage.ImageInputTokens)
	assert.Equal(t, int64(6), got.TotalUsage.ImageOutputTokens)
	assert.Equal(t, int64(2), got.TotalUsage.AudioInputTokens)
	assert.Equal(t, int64(3), got.TotalUsage.AudioOutputTokens)
	assert.Equal(t, int64(4), got.TotalUsage.VideoInputTokens)
	assert.Equal(t, int64(5), got.TotalUsage.VideoOutputTokens)
	assert.InDelta(t, 0.75, got.TotalCost, 1e-9)
	assert.NotNil(t, got.LastUsedAt)

	history, err := repo.GetUsageHistory(ctx, key.ID, 0, 0)
	require.NoError(t, err)
	require.Len(t, history, 2)
	page, err := repo.GetUsageHistory(ctx, key.ID, 1, 1)
	require.NoError(t, err)
	require.Len(t, page, 1)

	var failed *pkgmodels.RentalKeyUsageRecord
	for _, h := range history {
		if h.Status == "failed" {
			failed = h
		}
	}
	require.NotNil(t, failed)
	assert.Equal(t, "boom", failed.ErrorMessage)
	assert.Equal(t, "gpt-4o", failed.Model)

	from := time.Now().Add(-time.Hour).Format(time.RFC3339)
	to := time.Now().Add(time.Hour).Format(time.RFC3339)
	ranged, err := repo.GetUsageHistoryByTimeRange(ctx, key.ID, from, to)
	require.NoError(t, err)
	assert.Len(t, ranged, 2)
	ranged, err = repo.GetUsageHistoryByTimeRange(ctx, key.ID, "", "")
	require.NoError(t, err)
	assert.Len(t, ranged, 2)
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	ranged, err = repo.GetUsageHistoryByTimeRange(ctx, key.ID, future, "")
	require.NoError(t, err)
	assert.Empty(t, ranged)

	usage, requests, cost, err := repo.GetUsageSummary(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), requests)
	assert.InDelta(t, 0.75, cost, 1e-9)
	assert.Equal(t, int64(11), usage.PromptTokens)
	assert.Equal(t, int64(6), usage.ImageOutputTokens)
	assert.Equal(t, int64(5), usage.VideoOutputTokens)

	usage, requests, _, err = repo.GetUsageSummary(ctx, uuid.NewString())
	require.NoError(t, err)
	assert.Zero(t, requests)
	assert.Zero(t, usage.PromptTokens)

	require.NoError(t, repo.ResetDailyUsage(ctx))
	require.NoError(t, repo.ResetMonthlyUsage(ctx))
	got, _ = repo.GetRentalKey(ctx, key.ID)
	assert.Zero(t, got.RequestsToday)
	assert.Zero(t, got.TokensThisMonth)
	assert.Equal(t, int64(2), got.TotalRequests, "lifetime counters are kept")

	// invalid usage status violates CHECK and the counters stay untouched
	assert.Error(t, repo.RecordUsage(ctx, key.ID, &pkgmodels.RentalKeyUsageRecord{Model: "m", Status: "weird"}))
	got, _ = repo.GetRentalKey(ctx, key.ID)
	assert.Equal(t, int64(2), got.TotalRequests)

	assert.ErrorIs(t, repo.RecordUsage(ctx, "bad", u1), pkgmodels.ErrInvalidID)
	_, err = repo.GetUsageHistory(ctx, "bad", 0, 0)
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, err = repo.GetUsageHistoryByTimeRange(ctx, "bad", "", "")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, _, _, err = repo.GetUsageSummary(ctx, "bad")
	assert.ErrorIs(t, err, pkgmodels.ErrInvalidID)
	_, err = repo.GetUsageHistoryByTimeRange(ctx, key.ID, "not-a-time", "")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Credentials: paths not covered by credentials_repository_test.go
// ---------------------------------------------------------------------------

func TestRC_CredentialsRepository_ExtraOperations(t *testing.T) {
	t.Parallel()
	db, _ := testutil.SetupTestTx(t)
	ctx := context.Background()
	repo := NewCredentialsRepository(db)
	owner := createCredentialsTestUser(t, db)

	c1 := rcNewCredentials(owner, "a", "openai")
	c2 := rcNewCredentials(owner, "b", "github")
	require.NoError(t, repo.CreateCredentials(ctx, c1))
	require.NoError(t, repo.CreateCredentials(ctx, c2))

	byProvider, err := repo.GetCredentialsByProvider(ctx, owner, "github")
	require.NoError(t, err)
	require.Len(t, byProvider, 1)
	assert.Equal(t, c2.ID, byProvider[0].ID)

	require.NoError(t, repo.UpdateEncryptedData(ctx, c1.ID, map[string]string{"api_key": "new", "extra": "v"}))
	got, err := repo.GetCredentials(ctx, c1.ID)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"api_key": "new", "extra": "v"}, got.EncryptedData)
	assert.ErrorIs(t, repo.UpdateEncryptedData(ctx, "bad", nil), pkgmodels.ErrInvalidID)

	require.NoError(t, repo.LogCredentialAccess(ctx, c1.ID, "read", owner, "user", map[string]any{"ip": "1.2.3.4"}))
	require.NoError(t, repo.LogCredentialAccess(ctx, c1.ID, "read", "not-a-uuid", "system", nil))
	assert.Error(t, repo.LogCredentialAccess(ctx, "bad", "read", owner, "user", nil))

	var actions []string
	require.NoError(t, db.NewRaw(
		"SELECT action FROM mbflow_credential_audit_log WHERE credential_id = ? ORDER BY created_at", c1.ID,
	).Scan(ctx, &actions))
	assert.Equal(t, []string{"created", "read", "read"}, actions)

	var nullActors int
	require.NoError(t, db.NewRaw(
		"SELECT count(*) FROM mbflow_credential_audit_log WHERE credential_id = ? AND actor_id IS NULL", c1.ID,
	).Scan(ctx, &nullActors))
	assert.Equal(t, 1, nullActors, "unparsable actor id is stored as NULL")

	// Deleting an unknown credential fails inside the transaction
	assert.Error(t, repo.DeleteCredentials(ctx, uuid.NewString()))
	// Updating credentials with an unknown owner/actor still succeeds (actor is nullable)
	got.OwnerID = ""
	require.NoError(t, repo.UpdateCredentials(ctx, got))
}

func rcCountRows(t *testing.T, db bun.IDB, table string) int {
	t.Helper()
	var n int
	require.NoError(t, db.NewRaw("SELECT count(*) FROM "+table).Scan(context.Background(), &n))
	return n
}
