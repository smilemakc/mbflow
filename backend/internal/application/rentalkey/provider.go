// Package rentalkey provides services for managing rental API keys for LLM providers.
package rentalkey

import (
	"context"
	"fmt"
	"time"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/pkg/crypto"
	"github.com/smilemakc/mbflow/pkg/models"
)

// Provider manages rental key operations including key retrieval for execution,
// usage tracking, and lifecycle management.
type Provider struct {
	repo       repository.RentalKeyRepository
	encryption *crypto.EncryptionService
}

// NewProvider creates a new rental key provider.
func NewProvider(repo repository.RentalKeyRepository, encryption *crypto.EncryptionService) *Provider {
	return &Provider{
		repo:       repo,
		encryption: encryption,
	}
}

// ExecutionCredentials contains the credentials needed for LLM execution.
type ExecutionCredentials struct {
	APIKey   string
	Provider models.LLMProviderType
}

// GetAPIKeyForExecution retrieves and decrypts the API key for workflow execution.
// It validates ownership, checks if the key is active, and verifies usage limits.
func (p *Provider) GetAPIKeyForExecution(ctx context.Context, rentalKeyID, userID string) (*ExecutionCredentials, error) {
	// Get rental key
	key, err := p.repo.GetRentalKey(ctx, rentalKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rental key: %w", err)
	}

	// Validate ownership
	if key.OwnerID != userID {
		return nil, models.ErrRentalKeyAccessDenied
	}

	// Check status
	if key.Status != models.ResourceStatusActive {
		return nil, models.ErrRentalKeySuspended
	}

	// Check daily request limit
	if key.DailyRequestLimit != nil && key.RequestsToday >= *key.DailyRequestLimit {
		return nil, models.ErrDailyLimitExceeded
	}

	// Check monthly token limit
	if key.MonthlyTokenLimit != nil && key.TokensThisMonth >= *key.MonthlyTokenLimit {
		return nil, models.ErrMonthlyTokenLimitExceeded
	}

	// Decrypt API key
	apiKey, err := p.repo.GetDecryptedAPIKey(ctx, rentalKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return &ExecutionCredentials{
		APIKey:   apiKey,
		Provider: key.Provider,
	}, nil
}

// RecordUsage records the usage of a rental key after an LLM execution.
func (p *Provider) RecordUsage(ctx context.Context, rentalKeyID string, usage *models.RentalKeyUsageRecord) error {
	if usage == nil {
		return fmt.Errorf("usage record is required")
	}

	// Set timestamp if not set
	if usage.CreatedAt.IsZero() {
		usage.CreatedAt = time.Now()
	}

	return p.repo.RecordUsage(ctx, rentalKeyID, usage)
}

// GetKey retrieves a rental key by ID (without API key value).
func (p *Provider) GetKey(ctx context.Context, rentalKeyID string) (*models.RentalKeyResource, error) {
	return p.repo.GetRentalKey(ctx, rentalKeyID)
}

// GetKeysByOwner retrieves all rental keys for a specific user.
func (p *Provider) GetKeysByOwner(ctx context.Context, ownerID string) ([]*models.RentalKeyResource, error) {
	return p.repo.GetRentalKeysByOwner(ctx, ownerID)
}

// GetKeysByProvider retrieves rental keys filtered by provider for a user.
func (p *Provider) GetKeysByProvider(ctx context.Context, ownerID string, provider models.LLMProviderType) ([]*models.RentalKeyResource, error) {
	return p.repo.GetRentalKeysByProvider(ctx, ownerID, provider)
}

// GetUsageHistory retrieves usage history for a rental key.
func (p *Provider) GetUsageHistory(ctx context.Context, rentalKeyID string, limit, offset int) ([]*models.RentalKeyUsageRecord, error) {
	return p.repo.GetUsageHistory(ctx, rentalKeyID, limit, offset)
}

// GetUsageHistoryByTimeRange retrieves usage history for a rental key within a time range.
func (p *Provider) GetUsageHistoryByTimeRange(ctx context.Context, rentalKeyID string, from, to time.Time) ([]*models.RentalKeyUsageRecord, error) {
	return p.repo.GetUsageHistoryByTimeRange(ctx, rentalKeyID, from.Format(time.RFC3339), to.Format(time.RFC3339))
}

// GetUsageSummary retrieves aggregated usage statistics for a rental key.
func (p *Provider) GetUsageSummary(ctx context.Context, rentalKeyID string) (*UsageSummary, error) {
	usage, totalRequests, totalCost, err := p.repo.GetUsageSummary(ctx, rentalKeyID)
	if err != nil {
		return nil, err
	}

	return &UsageSummary{
		TotalUsage:    usage,
		TotalRequests: totalRequests,
		TotalCost:     totalCost,
	}, nil
}

// UsageSummary contains aggregated usage statistics.
type UsageSummary struct {
	TotalUsage    *models.MultimodalUsage `json:"total_usage"`
	TotalRequests int64                   `json:"total_requests"`
	TotalCost     float64                 `json:"total_cost"`
}

// AdminService provides administrative operations for rental keys.
type AdminService struct {
	repo       repository.RentalKeyRepository
	encryption *crypto.EncryptionService
}

// NewAdminService creates a new rental key admin service.
func NewAdminService(repo repository.RentalKeyRepository, encryption *crypto.EncryptionService) *AdminService {
	return &AdminService{
		repo:       repo,
		encryption: encryption,
	}
}

// CreateKeyRequest contains data for creating a new rental key.
type CreateKeyRequest struct {
	OwnerID           string
	Name              string
	Description       string
	Provider          models.LLMProviderType
	PlainAPIKey       string
	ProviderConfig    map[string]any
	DailyRequestLimit *int
	MonthlyTokenLimit *int64
	PricingPlanID     string
	CreatedBy         string
	ProvisionerType   models.ProvisionerType
}

// CreateKey creates a new rental key with the given parameters.
func (s *AdminService) CreateKey(ctx context.Context, req *CreateKeyRequest) (*models.RentalKeyResource, error) {
	if req == nil {
		return nil, fmt.Errorf("create request is required")
	}

	if req.PlainAPIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Create the resource
	key := models.NewRentalKeyResource(
		req.OwnerID,
		req.Name,
		req.Provider,
	)
	key.Description = req.Description
	key.ProviderConfig = req.ProviderConfig
	key.DailyRequestLimit = req.DailyRequestLimit
	key.MonthlyTokenLimit = req.MonthlyTokenLimit
	key.PricingPlanID = req.PricingPlanID
	key.CreatedBy = req.CreatedBy
	if req.ProvisionerType != "" {
		key.ProvisionerType = req.ProvisionerType
	}

	// Validate
	if err := key.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create in repository (encryption happens there)
	if err := s.repo.CreateRentalKey(ctx, key, req.PlainAPIKey); err != nil {
		return nil, fmt.Errorf("failed to create rental key: %w", err)
	}

	return key, nil
}

// UpdateKeyRequest contains data for updating a rental key.
type UpdateKeyRequest struct {
	Name              *string
	Description       *string
	Status            *models.ResourceStatus
	DailyRequestLimit *int
	MonthlyTokenLimit *int64
	ProviderConfig    map[string]any
}

// UpdateKey updates an existing rental key.
func (s *AdminService) UpdateKey(ctx context.Context, rentalKeyID string, req *UpdateKeyRequest) (*models.RentalKeyResource, error) {
	if req == nil {
		return nil, fmt.Errorf("update request is required")
	}

	// Get existing key
	key, err := s.repo.GetRentalKey(ctx, rentalKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rental key: %w", err)
	}

	// Apply updates
	if req.Name != nil {
		key.Name = *req.Name
	}
	if req.Description != nil {
		key.Description = *req.Description
	}
	if req.Status != nil {
		key.Status = *req.Status
	}
	if req.DailyRequestLimit != nil {
		key.DailyRequestLimit = req.DailyRequestLimit
	}
	if req.MonthlyTokenLimit != nil {
		key.MonthlyTokenLimit = req.MonthlyTokenLimit
	}
	if req.ProviderConfig != nil {
		key.ProviderConfig = req.ProviderConfig
	}

	// Update
	if err := s.repo.UpdateRentalKey(ctx, key); err != nil {
		return nil, fmt.Errorf("failed to update rental key: %w", err)
	}

	return key, nil
}

// RotateKey replaces the API key for a rental key.
func (s *AdminService) RotateKey(ctx context.Context, rentalKeyID, newPlainAPIKey string) error {
	if newPlainAPIKey == "" {
		return fmt.Errorf("new API key is required")
	}

	// Verify key exists
	if _, err := s.repo.GetRentalKey(ctx, rentalKeyID); err != nil {
		return fmt.Errorf("failed to get rental key: %w", err)
	}

	return s.repo.RotateAPIKey(ctx, rentalKeyID, newPlainAPIKey)
}

// DeleteKey soft-deletes a rental key.
func (s *AdminService) DeleteKey(ctx context.Context, rentalKeyID string) error {
	return s.repo.DeleteRentalKey(ctx, rentalKeyID)
}

// ListAllKeys retrieves all rental keys with optional filtering.
func (s *AdminService) ListAllKeys(ctx context.Context, filter repository.RentalKeyFilter) ([]*models.RentalKeyResource, int64, error) {
	return s.repo.GetAllRentalKeys(ctx, filter)
}

// GetKey retrieves a rental key by ID.
func (s *AdminService) GetKey(ctx context.Context, rentalKeyID string) (*models.RentalKeyResource, error) {
	return s.repo.GetRentalKey(ctx, rentalKeyID)
}

// ResetDailyUsage resets daily request counters for all rental keys.
// This should be called by a scheduled job at midnight.
func (s *AdminService) ResetDailyUsage(ctx context.Context) error {
	return s.repo.ResetDailyUsage(ctx)
}

// ResetMonthlyUsage resets monthly token counters for all rental keys.
// This should be called by a scheduled job at the start of each month.
func (s *AdminService) ResetMonthlyUsage(ctx context.Context) error {
	return s.repo.ResetMonthlyUsage(ctx)
}
