package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/crypto"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

var _ repository.RentalKeyRepository = (*RentalKeyRepositoryImpl)(nil)

// RentalKeyRepositoryImpl implements the RentalKeyRepository interface
type RentalKeyRepositoryImpl struct {
	db         bun.IDB
	encryption *crypto.EncryptionService
}

// NewRentalKeyRepository creates a new RentalKeyRepositoryImpl
func NewRentalKeyRepository(db bun.IDB, encryption *crypto.EncryptionService) *RentalKeyRepositoryImpl {
	return &RentalKeyRepositoryImpl{
		db:         db,
		encryption: encryption,
	}
}

// CreateRentalKey creates a new rental key resource with encrypted API key
func (r *RentalKeyRepositoryImpl) CreateRentalKey(ctx context.Context, key *pkgmodels.RentalKeyResource, plainAPIKey string) error {
	// Encrypt the API key
	encryptedKey, err := r.encryption.EncryptString(plainAPIKey)
	if err != nil {
		return err
	}

	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		resourceModel := &models.ResourceModel{
			ID:          uuid.New(),
			Type:        string(pkgmodels.ResourceTypeRentalKey),
			OwnerID:     uuid.MustParse(key.OwnerID),
			Name:        key.Name,
			Description: key.Description,
			Status:      string(key.Status),
			Metadata:    key.Metadata,
		}

		if _, err := tx.NewInsert().Model(resourceModel).Exec(ctx); err != nil {
			return err
		}

		// Convert provider config
		var providerConfig models.JSONBMap
		if key.ProviderConfig != nil {
			providerConfig = models.JSONBMap(key.ProviderConfig)
		}

		var pricingPlanID *uuid.UUID
		if key.PricingPlanID != "" {
			planID := uuid.MustParse(key.PricingPlanID)
			pricingPlanID = &planID
		}

		var createdBy *uuid.UUID
		if key.CreatedBy != "" {
			creatorID := uuid.MustParse(key.CreatedBy)
			createdBy = &creatorID
		}

		rentalKeyModel := &models.RentalKeyModel{
			ResourceID:        resourceModel.ID,
			Provider:          string(key.Provider),
			EncryptedAPIKey:   encryptedKey,
			ProviderConfig:    providerConfig,
			DailyRequestLimit: key.DailyRequestLimit,
			MonthlyTokenLimit: key.MonthlyTokenLimit,
			RequestsToday:     0,
			TokensThisMonth:   0,
			LastUsageResetAt:  time.Now(),
			TotalRequests:     0,
			PricingPlanID:     pricingPlanID,
			CreatedBy:         createdBy,
			ProvisionerType:   string(key.ProvisionerType),
		}

		if _, err := tx.NewInsert().Model(rentalKeyModel).Exec(ctx); err != nil {
			return err
		}

		// Update the domain model with generated values
		key.ID = resourceModel.ID.String()
		key.CreatedAt = resourceModel.CreatedAt
		key.UpdatedAt = resourceModel.UpdatedAt
		key.EncryptedAPIKey = encryptedKey

		return nil
	})
}

// GetRentalKey retrieves a rental key by resource ID
func (r *RentalKeyRepositoryImpl) GetRentalKey(ctx context.Context, resourceID string) (*pkgmodels.RentalKeyResource, error) {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	resourceModel := new(models.ResourceModel)
	err = r.db.NewSelect().
		Model(resourceModel).
		Relation("RentalKey").
		Where("r.id = ? AND r.deleted_at IS NULL", resID).
		Where("r.type = ?", string(pkgmodels.ResourceTypeRentalKey)).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgmodels.ErrRentalKeyNotFound
		}
		return nil, err
	}

	if resourceModel.RentalKey == nil {
		return nil, pkgmodels.ErrRentalKeyNotFound
	}

	return models.ToRentalKeyResourceDomain(resourceModel, resourceModel.RentalKey), nil
}

// GetRentalKeysByOwner retrieves all rental keys for an owner
func (r *RentalKeyRepositoryImpl) GetRentalKeysByOwner(ctx context.Context, ownerID string) ([]*pkgmodels.RentalKeyResource, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var resourceModels []*models.ResourceModel
	err = r.db.NewSelect().
		Model(&resourceModels).
		Relation("RentalKey").
		Where("r.owner_id = ? AND r.deleted_at IS NULL", ownerUUID).
		Where("r.type = ?", string(pkgmodels.ResourceTypeRentalKey)).
		Order("r.created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	rentalKeys := make([]*pkgmodels.RentalKeyResource, 0, len(resourceModels))
	for _, rm := range resourceModels {
		if rm.RentalKey != nil {
			rentalKeys = append(rentalKeys, models.ToRentalKeyResourceDomain(rm, rm.RentalKey))
		}
	}

	return rentalKeys, nil
}

// GetRentalKeysByProvider retrieves rental keys by provider for an owner
func (r *RentalKeyRepositoryImpl) GetRentalKeysByProvider(ctx context.Context, ownerID string, provider pkgmodels.LLMProviderType) ([]*pkgmodels.RentalKeyResource, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var resourceModels []*models.ResourceModel
	err = r.db.NewSelect().
		Model(&resourceModels).
		Relation("RentalKey", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("rrk.provider = ?", string(provider))
		}).
		Where("r.owner_id = ? AND r.deleted_at IS NULL", ownerUUID).
		Where("r.type = ?", string(pkgmodels.ResourceTypeRentalKey)).
		Order("r.created_at DESC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	rentalKeys := make([]*pkgmodels.RentalKeyResource, 0, len(resourceModels))
	for _, rm := range resourceModels {
		if rm.RentalKey != nil {
			rentalKeys = append(rentalKeys, models.ToRentalKeyResourceDomain(rm, rm.RentalKey))
		}
	}

	return rentalKeys, nil
}

// UpdateRentalKey updates a rental key resource (does not update encrypted API key)
func (r *RentalKeyRepositoryImpl) UpdateRentalKey(ctx context.Context, key *pkgmodels.RentalKeyResource) error {
	resourceID, err := uuid.Parse(key.ID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		// Update base resource
		_, err := tx.NewUpdate().
			Model((*models.ResourceModel)(nil)).
			Set("name = ?", key.Name).
			Set("description = ?", key.Description).
			Set("status = ?", string(key.Status)).
			Set("metadata = ?", key.Metadata).
			Set("updated_at = ?", time.Now()).
			Where("id = ? AND deleted_at IS NULL", resourceID).
			Exec(ctx)

		if err != nil {
			return err
		}

		// Convert provider config
		var providerConfig models.JSONBMap
		if key.ProviderConfig != nil {
			providerConfig = models.JSONBMap(key.ProviderConfig)
		}

		var pricingPlanID *uuid.UUID
		if key.PricingPlanID != "" {
			planID := uuid.MustParse(key.PricingPlanID)
			pricingPlanID = &planID
		}

		// Update rental key specific data
		_, err = tx.NewUpdate().
			Model((*models.RentalKeyModel)(nil)).
			Set("provider = ?", string(key.Provider)).
			Set("provider_config = ?", providerConfig).
			Set("daily_request_limit = ?", key.DailyRequestLimit).
			Set("monthly_token_limit = ?", key.MonthlyTokenLimit).
			Set("pricing_plan_id = ?", pricingPlanID).
			Where("resource_id = ?", resourceID).
			Exec(ctx)

		return err
	})
}

// DeleteRentalKey soft-deletes a rental key resource
func (r *RentalKeyRepositoryImpl) DeleteRentalKey(ctx context.Context, resourceID string) error {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	_, err = r.db.NewUpdate().
		Model((*models.ResourceModel)(nil)).
		Set("deleted_at = ?", time.Now()).
		Set("status = ?", string(pkgmodels.ResourceStatusDeleted)).
		Where("id = ? AND deleted_at IS NULL", resID).
		Exec(ctx)

	return err
}

// GetDecryptedAPIKey retrieves and decrypts the API key (internal use only)
func (r *RentalKeyRepositoryImpl) GetDecryptedAPIKey(ctx context.Context, resourceID string) (string, error) {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return "", pkgmodels.ErrInvalidID
	}

	var rentalKeyModel models.RentalKeyModel
	err = r.db.NewSelect().
		Model(&rentalKeyModel).
		Column("encrypted_api_key").
		Where("resource_id = ?", resID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", pkgmodels.ErrRentalKeyNotFound
		}
		return "", err
	}

	return r.encryption.DecryptString(rentalKeyModel.EncryptedAPIKey)
}

// RotateAPIKey replaces the API key with a new one
func (r *RentalKeyRepositoryImpl) RotateAPIKey(ctx context.Context, resourceID string, newPlainAPIKey string) error {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	encryptedKey, err := r.encryption.EncryptString(newPlainAPIKey)
	if err != nil {
		return err
	}

	_, err = r.db.NewUpdate().
		Model((*models.RentalKeyModel)(nil)).
		Set("encrypted_api_key = ?", encryptedKey).
		Where("resource_id = ?", resID).
		Exec(ctx)

	return err
}

// RecordUsage records a usage event and updates counters
func (r *RentalKeyRepositoryImpl) RecordUsage(ctx context.Context, resourceID string, usage *pkgmodels.RentalKeyUsageRecord) error {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return pkgmodels.ErrInvalidID
	}

	return r.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		// Create usage log entry
		usageModel := models.FromRentalKeyUsageRecordDomain(usage)
		usageModel.ID = uuid.New()
		usageModel.RentalKeyID = resID
		usageModel.CreatedAt = time.Now()

		if _, err := tx.NewInsert().Model(usageModel).Exec(ctx); err != nil {
			return err
		}

		// Update rental key counters
		now := time.Now()
		totalTokens := usage.Usage.TotalTokens()

		_, err := tx.NewUpdate().
			Model((*models.RentalKeyModel)(nil)).
			Set("requests_today = requests_today + 1").
			Set("tokens_this_month = tokens_this_month + ?", totalTokens).
			Set("total_requests = total_requests + 1").
			Set("total_prompt_tokens = total_prompt_tokens + ?", usage.Usage.PromptTokens).
			Set("total_completion_tokens = total_completion_tokens + ?", usage.Usage.CompletionTokens).
			Set("total_image_input_tokens = total_image_input_tokens + ?", usage.Usage.ImageInputTokens).
			Set("total_image_output_tokens = total_image_output_tokens + ?", usage.Usage.ImageOutputTokens).
			Set("total_audio_input_tokens = total_audio_input_tokens + ?", usage.Usage.AudioInputTokens).
			Set("total_audio_output_tokens = total_audio_output_tokens + ?", usage.Usage.AudioOutputTokens).
			Set("total_video_input_tokens = total_video_input_tokens + ?", usage.Usage.VideoInputTokens).
			Set("total_video_output_tokens = total_video_output_tokens + ?", usage.Usage.VideoOutputTokens).
			Set("total_cost = total_cost + ?", usage.EstimatedCost).
			Set("last_used_at = ?", now).
			Where("resource_id = ?", resID).
			Exec(ctx)

		if err != nil {
			return err
		}

		// Update domain object
		usage.ID = usageModel.ID.String()
		usage.RentalKeyID = resID.String()
		usage.CreatedAt = usageModel.CreatedAt

		return nil
	})
}

// GetUsageHistory retrieves usage history with pagination
func (r *RentalKeyRepositoryImpl) GetUsageHistory(ctx context.Context, resourceID string, limit int, offset int) ([]*pkgmodels.RentalKeyUsageRecord, error) {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var usageModels []*models.RentalKeyUsageModel
	query := r.db.NewSelect().
		Model(&usageModels).
		Where("rental_key_id = ?", resID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	records := make([]*pkgmodels.RentalKeyUsageRecord, 0, len(usageModels))
	for _, m := range usageModels {
		records = append(records, models.ToRentalKeyUsageRecordDomain(m))
	}

	return records, nil
}

// GetUsageHistoryByTimeRange retrieves usage history for a time range
func (r *RentalKeyRepositoryImpl) GetUsageHistoryByTimeRange(ctx context.Context, resourceID string, from, to string) ([]*pkgmodels.RentalKeyUsageRecord, error) {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	var usageModels []*models.RentalKeyUsageModel
	query := r.db.NewSelect().
		Model(&usageModels).
		Where("rental_key_id = ?", resID).
		Order("created_at DESC")

	if from != "" {
		query = query.Where("created_at >= ?", from)
	}
	if to != "" {
		query = query.Where("created_at <= ?", to)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	records := make([]*pkgmodels.RentalKeyUsageRecord, 0, len(usageModels))
	for _, m := range usageModels {
		records = append(records, models.ToRentalKeyUsageRecordDomain(m))
	}

	return records, nil
}

// GetUsageSummary retrieves aggregated usage statistics
func (r *RentalKeyRepositoryImpl) GetUsageSummary(ctx context.Context, resourceID string) (*pkgmodels.MultimodalUsage, int64, float64, error) {
	resID, err := uuid.Parse(resourceID)
	if err != nil {
		return nil, 0, 0, pkgmodels.ErrInvalidID
	}

	var result struct {
		TotalRequests         int64   `bun:"total_requests"`
		TotalPromptTokens     int64   `bun:"total_prompt_tokens"`
		TotalCompletionTokens int64   `bun:"total_completion_tokens"`
		TotalImageInput       int64   `bun:"total_image_input"`
		TotalImageOutput      int64   `bun:"total_image_output"`
		TotalAudioInput       int64   `bun:"total_audio_input"`
		TotalAudioOutput      int64   `bun:"total_audio_output"`
		TotalVideoInput       int64   `bun:"total_video_input"`
		TotalVideoOutput      int64   `bun:"total_video_output"`
		TotalCost             float64 `bun:"total_cost"`
	}

	err = r.db.NewSelect().
		Model((*models.RentalKeyUsageModel)(nil)).
		ColumnExpr("COUNT(*) AS total_requests").
		ColumnExpr("COALESCE(SUM(prompt_tokens), 0) AS total_prompt_tokens").
		ColumnExpr("COALESCE(SUM(completion_tokens), 0) AS total_completion_tokens").
		ColumnExpr("COALESCE(SUM(image_input_tokens), 0) AS total_image_input").
		ColumnExpr("COALESCE(SUM(image_output_tokens), 0) AS total_image_output").
		ColumnExpr("COALESCE(SUM(audio_input_tokens), 0) AS total_audio_input").
		ColumnExpr("COALESCE(SUM(audio_output_tokens), 0) AS total_audio_output").
		ColumnExpr("COALESCE(SUM(video_input_tokens), 0) AS total_video_input").
		ColumnExpr("COALESCE(SUM(video_output_tokens), 0) AS total_video_output").
		ColumnExpr("COALESCE(SUM(estimated_cost), 0) AS total_cost").
		Where("rental_key_id = ?", resID).
		Scan(ctx, &result)

	if err != nil {
		return nil, 0, 0, err
	}

	usage := &pkgmodels.MultimodalUsage{
		PromptTokens:      result.TotalPromptTokens,
		CompletionTokens:  result.TotalCompletionTokens,
		ImageInputTokens:  result.TotalImageInput,
		ImageOutputTokens: result.TotalImageOutput,
		AudioInputTokens:  result.TotalAudioInput,
		AudioOutputTokens: result.TotalAudioOutput,
		VideoInputTokens:  result.TotalVideoInput,
		VideoOutputTokens: result.TotalVideoOutput,
	}

	return usage, result.TotalRequests, result.TotalCost, nil
}

// ResetDailyUsage resets the daily request counter for all rental keys
func (r *RentalKeyRepositoryImpl) ResetDailyUsage(ctx context.Context) error {
	_, err := r.db.NewUpdate().
		Model((*models.RentalKeyModel)(nil)).
		Set("requests_today = 0").
		Where("1 = 1").
		Exec(ctx)

	return err
}

// ResetMonthlyUsage resets the monthly token counter for all rental keys
func (r *RentalKeyRepositoryImpl) ResetMonthlyUsage(ctx context.Context) error {
	_, err := r.db.NewUpdate().
		Model((*models.RentalKeyModel)(nil)).
		Set("tokens_this_month = 0").
		Set("last_usage_reset_at = ?", time.Now()).
		Where("1 = 1").
		Exec(ctx)

	return err
}

// GetAllRentalKeys retrieves all rental keys with filters (admin)
func (r *RentalKeyRepositoryImpl) GetAllRentalKeys(ctx context.Context, filter repository.RentalKeyFilter) ([]*pkgmodels.RentalKeyResource, int64, error) {
	var resourceModels []*models.ResourceModel

	query := r.db.NewSelect().
		Model(&resourceModels).
		Relation("RentalKey").
		Where("r.type = ?", string(pkgmodels.ResourceTypeRentalKey)).
		Where("r.deleted_at IS NULL")

	// Apply filters
	if filter.Provider != nil {
		query = query.Where("rrk.provider = ?", string(*filter.Provider))
	}
	if filter.Status != nil {
		query = query.Where("r.status = ?", string(*filter.Status))
	}
	if filter.OwnerID != nil {
		ownerUUID, err := uuid.Parse(*filter.OwnerID)
		if err == nil {
			query = query.Where("r.owner_id = ?", ownerUUID)
		}
	}
	if filter.CreatedBy != nil {
		creatorUUID, err := uuid.Parse(*filter.CreatedBy)
		if err == nil {
			query = query.Where("rrk.created_by = ?", creatorUUID)
		}
	}

	// Get total count first
	count, err := r.GetAllRentalKeysCount(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	query = query.Order("r.created_at DESC")
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, 0, err
	}

	rentalKeys := make([]*pkgmodels.RentalKeyResource, 0, len(resourceModels))
	for _, rm := range resourceModels {
		if rm.RentalKey != nil {
			rentalKeys = append(rentalKeys, models.ToRentalKeyResourceDomain(rm, rm.RentalKey))
		}
	}

	return rentalKeys, count, nil
}

// GetAllRentalKeysCount returns the total count of rental keys matching filters
func (r *RentalKeyRepositoryImpl) GetAllRentalKeysCount(ctx context.Context, filter repository.RentalKeyFilter) (int64, error) {
	query := r.db.NewSelect().
		Model((*models.ResourceModel)(nil)).
		Join("JOIN mbflow_resource_rental_key AS rrk ON rrk.resource_id = r.id").
		Where("r.type = ?", string(pkgmodels.ResourceTypeRentalKey)).
		Where("r.deleted_at IS NULL")

	// Apply filters
	if filter.Provider != nil {
		query = query.Where("rrk.provider = ?", string(*filter.Provider))
	}
	if filter.Status != nil {
		query = query.Where("r.status = ?", string(*filter.Status))
	}
	if filter.OwnerID != nil {
		ownerUUID, err := uuid.Parse(*filter.OwnerID)
		if err == nil {
			query = query.Where("r.owner_id = ?", ownerUUID)
		}
	}
	if filter.CreatedBy != nil {
		creatorUUID, err := uuid.Parse(*filter.CreatedBy)
		if err == nil {
			query = query.Where("rrk.created_by = ?", creatorUUID)
		}
	}

	count, err := query.Count(ctx)
	return int64(count), err
}
