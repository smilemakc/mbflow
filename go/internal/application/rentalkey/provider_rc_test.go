package rentalkey

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rcErrRepo = errors.New("repo failure")

// rcRentalRepo is an in-memory repository.RentalKeyRepository.
type rcRentalRepo struct {
	mu      sync.Mutex
	keys    map[string]*models.RentalKeyResource
	secrets map[string]string
	usage   map[string][]*models.RentalKeyUsageRecord

	errs map[string]error

	lastRangeFrom, lastRangeTo string
	lastFilter                 repository.RentalKeyFilter
	dailyResets, monthlyResets int
}

func rcNewRepo() *rcRentalRepo {
	return &rcRentalRepo{
		keys:    map[string]*models.RentalKeyResource{},
		secrets: map[string]string{},
		usage:   map[string][]*models.RentalKeyUsageRecord{},
		errs:    map[string]error{},
	}
}

func (r *rcRentalRepo) CreateRentalKey(ctx context.Context, key *models.RentalKeyResource, plainAPIKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Create"]; err != nil {
		return err
	}
	if key.ID == "" {
		key.ID = "rk-" + key.Name
	}
	r.keys[key.ID] = key
	r.secrets[key.ID] = plainAPIKey
	return nil
}

func (r *rcRentalRepo) GetRentalKey(ctx context.Context, id string) (*models.RentalKeyResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Get"]; err != nil {
		return nil, err
	}
	k, ok := r.keys[id]
	if !ok {
		return nil, models.ErrResourceNotFound
	}
	return k, nil
}

func (r *rcRentalRepo) GetRentalKeysByOwner(ctx context.Context, ownerID string) ([]*models.RentalKeyResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.RentalKeyResource
	for _, k := range r.keys {
		if k.OwnerID == ownerID {
			out = append(out, k)
		}
	}
	return out, nil
}

func (r *rcRentalRepo) GetRentalKeysByProvider(ctx context.Context, ownerID string, provider models.LLMProviderType) ([]*models.RentalKeyResource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.RentalKeyResource
	for _, k := range r.keys {
		if k.OwnerID == ownerID && k.Provider == provider {
			out = append(out, k)
		}
	}
	return out, nil
}

func (r *rcRentalRepo) UpdateRentalKey(ctx context.Context, key *models.RentalKeyResource) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Update"]; err != nil {
		return err
	}
	r.keys[key.ID] = key
	return nil
}

func (r *rcRentalRepo) DeleteRentalKey(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.keys[id]; !ok {
		return models.ErrResourceNotFound
	}
	delete(r.keys, id)
	return nil
}

func (r *rcRentalRepo) GetDecryptedAPIKey(ctx context.Context, id string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Decrypt"]; err != nil {
		return "", err
	}
	return r.secrets[id], nil
}

func (r *rcRentalRepo) RotateAPIKey(ctx context.Context, id string, newKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.secrets[id] = newKey
	return nil
}

func (r *rcRentalRepo) RecordUsage(ctx context.Context, id string, usage *models.RentalKeyUsageRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usage[id] = append(r.usage[id], usage)
	return nil
}

func (r *rcRentalRepo) GetUsageHistory(ctx context.Context, id string, limit int, offset int) ([]*models.RentalKeyUsageRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	recs := r.usage[id]
	if offset >= len(recs) {
		return nil, nil
	}
	end := offset + limit
	if end > len(recs) {
		end = len(recs)
	}
	return recs[offset:end], nil
}

func (r *rcRentalRepo) GetUsageHistoryByTimeRange(ctx context.Context, id string, from, to string) ([]*models.RentalKeyUsageRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastRangeFrom, r.lastRangeTo = from, to
	return r.usage[id], nil
}

func (r *rcRentalRepo) GetUsageSummary(ctx context.Context, id string) (*models.MultimodalUsage, int64, float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.errs["Summary"]; err != nil {
		return nil, 0, 0, err
	}
	total := &models.MultimodalUsage{}
	var cost float64
	for _, u := range r.usage[id] {
		total.PromptTokens += u.Usage.PromptTokens
		total.CompletionTokens += u.Usage.CompletionTokens
		cost += u.EstimatedCost
	}
	return total, int64(len(r.usage[id])), cost, nil
}

func (r *rcRentalRepo) ResetDailyUsage(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dailyResets++
	for _, k := range r.keys {
		k.RequestsToday = 0
	}
	return nil
}

func (r *rcRentalRepo) ResetMonthlyUsage(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.monthlyResets++
	for _, k := range r.keys {
		k.TokensThisMonth = 0
	}
	return nil
}

func (r *rcRentalRepo) GetAllRentalKeys(ctx context.Context, filter repository.RentalKeyFilter) ([]*models.RentalKeyResource, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastFilter = filter
	var out []*models.RentalKeyResource
	for _, k := range r.keys {
		if filter.OwnerID != nil && k.OwnerID != *filter.OwnerID {
			continue
		}
		out = append(out, k)
	}
	return out, int64(len(out)), nil
}

func (r *rcRentalRepo) GetAllRentalKeysCount(ctx context.Context, filter repository.RentalKeyFilter) (int64, error) {
	return int64(len(r.keys)), nil
}

func rcSetup(t *testing.T) (*Provider, *AdminService, *rcRentalRepo) {
	t.Helper()
	repo := rcNewRepo()
	return NewProvider(repo, nil), NewAdminService(repo, nil), repo
}

func rcCreate(t *testing.T, admin *AdminService, owner, name string) *models.RentalKeyResource {
	t.Helper()
	key, err := admin.CreateKey(context.Background(), &CreateKeyRequest{
		OwnerID: owner, Name: name, Provider: models.LLMProviderTypeOpenAI, PlainAPIKey: "sk-" + name,
	})
	require.NoError(t, err)
	return key
}

func TestRCAdminService_CreateKey(t *testing.T) {
	ctx := context.Background()
	_, admin, repo := rcSetup(t)

	daily := 10
	monthly := int64(1000)
	key, err := admin.CreateKey(ctx, &CreateKeyRequest{
		OwnerID:           "owner-1",
		Name:              "primary",
		Description:       "desc",
		Provider:          models.LLMProviderTypeAnthropic,
		PlainAPIKey:       "sk-secret",
		ProviderConfig:    map[string]any{"base_url": "https://x"},
		DailyRequestLimit: &daily,
		MonthlyTokenLimit: &monthly,
		PricingPlanID:     "plan",
		CreatedBy:         "admin",
		ProvisionerType:   models.ProvisionerTypeAutoAnthropic,
	})
	require.NoError(t, err)
	assert.Equal(t, models.ResourceStatusActive, key.Status)
	assert.Equal(t, "desc", key.Description)
	assert.Equal(t, models.ProvisionerTypeAutoAnthropic, key.ProvisionerType)
	assert.Equal(t, &daily, key.DailyRequestLimit)
	assert.Equal(t, "sk-secret", repo.secrets[key.ID], "plain key is handed to the repository for encryption")

	// Default provisioner type
	k2 := rcCreate(t, admin, "owner-1", "second")
	assert.Equal(t, models.ProvisionerTypeManual, k2.ProvisionerType)
}

func TestRCAdminService_CreateKeyErrors(t *testing.T) {
	ctx := context.Background()
	_, admin, repo := rcSetup(t)

	_, err := admin.CreateKey(ctx, nil)
	assert.EqualError(t, err, "create request is required")
	_, err = admin.CreateKey(ctx, &CreateKeyRequest{OwnerID: "o", Name: "n", Provider: models.LLMProviderTypeOpenAI})
	assert.EqualError(t, err, "API key is required")
	_, err = admin.CreateKey(ctx, &CreateKeyRequest{OwnerID: "o", Name: "", Provider: models.LLMProviderTypeOpenAI, PlainAPIKey: "k"})
	assert.ErrorContains(t, err, "validation failed")
	_, err = admin.CreateKey(ctx, &CreateKeyRequest{OwnerID: "o", Name: "n", Provider: "bogus", PlainAPIKey: "k"})
	assert.ErrorContains(t, err, "invalid provider type")
	_, err = admin.CreateKey(ctx, &CreateKeyRequest{OwnerID: "o", Name: "n", Provider: models.LLMProviderTypeOpenAI, PlainAPIKey: "k", ProvisionerType: "weird"})
	assert.ErrorContains(t, err, "invalid provisioner type")

	repo.errs["Create"] = rcErrRepo
	_, err = admin.CreateKey(ctx, &CreateKeyRequest{OwnerID: "o", Name: "n", Provider: models.LLMProviderTypeOpenAI, PlainAPIKey: "k"})
	assert.ErrorIs(t, err, rcErrRepo)
}

func TestRCAdminService_UpdateRotateDeleteList(t *testing.T) {
	ctx := context.Background()
	_, admin, repo := rcSetup(t)
	key := rcCreate(t, admin, "owner-1", "k")

	_, err := admin.UpdateKey(ctx, key.ID, nil)
	assert.EqualError(t, err, "update request is required")
	_, err = admin.UpdateKey(ctx, "missing", &UpdateKeyRequest{})
	assert.ErrorIs(t, err, models.ErrResourceNotFound)

	name, desc := "renamed", "new desc"
	status := models.ResourceStatusSuspended
	daily := 5
	monthly := int64(50)
	updated, err := admin.UpdateKey(ctx, key.ID, &UpdateKeyRequest{
		Name: &name, Description: &desc, Status: &status,
		DailyRequestLimit: &daily, MonthlyTokenLimit: &monthly,
		ProviderConfig: map[string]any{"org": "x"},
	})
	require.NoError(t, err)
	assert.Equal(t, "renamed", updated.Name)
	assert.Equal(t, "new desc", updated.Description)
	assert.Equal(t, models.ResourceStatusSuspended, updated.Status)
	assert.Equal(t, 5, *updated.DailyRequestLimit)
	assert.Equal(t, int64(50), *updated.MonthlyTokenLimit)
	assert.Equal(t, "x", updated.ProviderConfig["org"])

	// Empty update keeps fields
	same, err := admin.UpdateKey(ctx, key.ID, &UpdateKeyRequest{})
	require.NoError(t, err)
	assert.Equal(t, "renamed", same.Name)

	repo.errs["Update"] = rcErrRepo
	_, err = admin.UpdateKey(ctx, key.ID, &UpdateKeyRequest{})
	assert.ErrorIs(t, err, rcErrRepo)
	delete(repo.errs, "Update")

	// Rotate
	assert.EqualError(t, admin.RotateKey(ctx, key.ID, ""), "new API key is required")
	assert.ErrorIs(t, admin.RotateKey(ctx, "missing", "x"), models.ErrResourceNotFound)
	require.NoError(t, admin.RotateKey(ctx, key.ID, "sk-rotated"))
	assert.Equal(t, "sk-rotated", repo.secrets[key.ID])

	// Get / list
	got, err := admin.GetKey(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, key.ID, got.ID)
	owner := "owner-1"
	list, total, err := admin.ListAllKeys(ctx, repository.RentalKeyFilter{OwnerID: &owner, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, 10, repo.lastFilter.Limit)

	// Resets
	require.NoError(t, admin.ResetDailyUsage(ctx))
	require.NoError(t, admin.ResetMonthlyUsage(ctx))
	assert.Equal(t, 1, repo.dailyResets)
	assert.Equal(t, 1, repo.monthlyResets)

	// Delete
	require.NoError(t, admin.DeleteKey(ctx, key.ID))
	assert.ErrorIs(t, admin.DeleteKey(ctx, key.ID), models.ErrResourceNotFound)
}

func TestRCProvider_GetAPIKeyForExecution(t *testing.T) {
	ctx := context.Background()
	p, admin, repo := rcSetup(t)
	key := rcCreate(t, admin, "owner-1", "exec")

	creds, err := p.GetAPIKeyForExecution(ctx, key.ID, "owner-1")
	require.NoError(t, err)
	assert.Equal(t, "sk-exec", creds.APIKey)
	assert.Equal(t, models.LLMProviderTypeOpenAI, creds.Provider)

	_, err = p.GetAPIKeyForExecution(ctx, "missing", "owner-1")
	assert.ErrorIs(t, err, models.ErrResourceNotFound)

	_, err = p.GetAPIKeyForExecution(ctx, key.ID, "intruder")
	assert.ErrorIs(t, err, models.ErrRentalKeyAccessDenied)

	key.Status = models.ResourceStatusSuspended
	_, err = p.GetAPIKeyForExecution(ctx, key.ID, "owner-1")
	assert.ErrorIs(t, err, models.ErrRentalKeySuspended)
	key.Status = models.ResourceStatusActive

	daily := 3
	key.DailyRequestLimit = &daily
	key.RequestsToday = 3
	_, err = p.GetAPIKeyForExecution(ctx, key.ID, "owner-1")
	assert.ErrorIs(t, err, models.ErrDailyLimitExceeded)
	key.RequestsToday = 2
	_, err = p.GetAPIKeyForExecution(ctx, key.ID, "owner-1")
	assert.NoError(t, err, "below the daily limit is allowed")

	monthly := int64(100)
	key.MonthlyTokenLimit = &monthly
	key.TokensThisMonth = 100
	_, err = p.GetAPIKeyForExecution(ctx, key.ID, "owner-1")
	assert.ErrorIs(t, err, models.ErrMonthlyTokenLimitExceeded)
	key.TokensThisMonth = 0

	repo.errs["Decrypt"] = rcErrRepo
	_, err = p.GetAPIKeyForExecution(ctx, key.ID, "owner-1")
	assert.ErrorIs(t, err, rcErrRepo)
	assert.ErrorContains(t, err, "failed to decrypt API key")
}

func TestRCProvider_UsageAndQueries(t *testing.T) {
	ctx := context.Background()
	p, admin, repo := rcSetup(t)
	k1 := rcCreate(t, admin, "owner-1", "a")
	_, err := admin.CreateKey(ctx, &CreateKeyRequest{OwnerID: "owner-1", Name: "b", Provider: models.LLMProviderTypeAnthropic, PlainAPIKey: "x"})
	require.NoError(t, err)
	rcCreate(t, admin, "owner-2", "c")

	assert.EqualError(t, p.RecordUsage(ctx, k1.ID, nil), "usage record is required")

	rec := &models.RentalKeyUsageRecord{Model: "gpt", Usage: models.MultimodalUsage{PromptTokens: 10, CompletionTokens: 5}, EstimatedCost: 0.5}
	require.NoError(t, p.RecordUsage(ctx, k1.ID, rec))
	assert.False(t, rec.CreatedAt.IsZero(), "timestamp is filled in")

	fixed := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	rec2 := &models.RentalKeyUsageRecord{Model: "gpt", Usage: models.MultimodalUsage{PromptTokens: 1}, EstimatedCost: 0.25, CreatedAt: fixed}
	require.NoError(t, p.RecordUsage(ctx, k1.ID, rec2))
	assert.Equal(t, fixed, rec2.CreatedAt, "explicit timestamp is preserved")

	hist, err := p.GetUsageHistory(ctx, k1.ID, 1, 1)
	require.NoError(t, err)
	require.Len(t, hist, 1)
	assert.Same(t, rec2, hist[0])

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	_, err = p.GetUsageHistoryByTimeRange(ctx, k1.ID, from, to)
	require.NoError(t, err)
	assert.Equal(t, "2025-01-01T00:00:00Z", repo.lastRangeFrom)
	assert.Equal(t, "2025-02-01T00:00:00Z", repo.lastRangeTo)

	sum, err := p.GetUsageSummary(ctx, k1.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), sum.TotalRequests)
	assert.InDelta(t, 0.75, sum.TotalCost, 1e-9)
	assert.Equal(t, int64(11), sum.TotalUsage.PromptTokens)

	repo.errs["Summary"] = rcErrRepo
	_, err = p.GetUsageSummary(ctx, k1.ID)
	assert.ErrorIs(t, err, rcErrRepo)

	got, err := p.GetKey(ctx, k1.ID)
	require.NoError(t, err)
	assert.Equal(t, "a", got.Name)

	byOwner, err := p.GetKeysByOwner(ctx, "owner-1")
	require.NoError(t, err)
	assert.Len(t, byOwner, 2)

	byProvider, err := p.GetKeysByProvider(ctx, "owner-1", models.LLMProviderTypeAnthropic)
	require.NoError(t, err)
	require.Len(t, byProvider, 1)
	assert.Equal(t, "b", byProvider[0].Name)
}
