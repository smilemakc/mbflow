package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

// ResourceModel represents a base resource in the database
type ResourceModel struct {
	bun.BaseModel `bun:"table:mbflow_resources,alias:r"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Type        string     `bun:"type,notnull" json:"type" validate:"required,oneof=file_storage credentials rental_key"`
	OwnerID     uuid.UUID  `bun:"owner_id,notnull,type:uuid" json:"owner_id" validate:"required"`
	Name        string     `bun:"name,notnull" json:"name" validate:"required,max=255"`
	Description string     `bun:"description" json:"description,omitempty" validate:"max=1000"`
	Status      string     `bun:"status,notnull,default:'active'" json:"status" validate:"required,oneof=active suspended deleted"`
	Metadata    JSONBMap   `bun:"metadata,type:jsonb,default:'{}'" json:"metadata,omitempty"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deleted_at,omitempty"`

	// Relations
	Owner       *UserModel        `bun:"rel:belongs-to,join:owner_id=id" json:"owner,omitempty"`
	FileStorage *FileStorageModel `bun:"rel:has-one,join:id=resource_id" json:"file_storage,omitempty"`
	Credentials *CredentialsModel `bun:"rel:has-one,join:id=resource_id" json:"credentials,omitempty"`
	RentalKey   *RentalKeyModel   `bun:"rel:has-one,join:id=resource_id" json:"rental_key,omitempty"`
}

// TableName returns the table name for ResourceModel
func (ResourceModel) TableName() string {
	return "mbflow_resources"
}

// BeforeInsert hook to set timestamps and defaults
func (r *ResourceModel) BeforeInsert(ctx any) error {
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.Metadata == nil {
		r.Metadata = make(JSONBMap)
	}
	if r.Status == "" {
		r.Status = string(pkgmodels.ResourceStatusActive)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (r *ResourceModel) BeforeUpdate(ctx any) error {
	r.UpdatedAt = time.Now()
	return nil
}

// IsActive returns true if resource is active
func (r *ResourceModel) IsActive() bool {
	return r.Status == string(pkgmodels.ResourceStatusActive)
}

// IsDeleted returns true if resource is soft-deleted
func (r *ResourceModel) IsDeleted() bool {
	return r.DeletedAt != nil
}

// FileStorageModel represents file storage specific data in the database
type FileStorageModel struct {
	bun.BaseModel `bun:"table:mbflow_resource_file_storage,alias:rfs"`

	ResourceID        uuid.UUID  `bun:"resource_id,pk,type:uuid" json:"resource_id" validate:"required"`
	StorageLimitBytes int64      `bun:"storage_limit_bytes,notnull,default:5242880" json:"storage_limit_bytes" validate:"required,min=0"`
	UsedStorageBytes  int64      `bun:"used_storage_bytes,notnull,default:0" json:"used_storage_bytes" validate:"min=0"`
	FileCount         int        `bun:"file_count,notnull,default:0" json:"file_count" validate:"min=0"`
	PricingPlanID     *uuid.UUID `bun:"pricing_plan_id,type:uuid" json:"pricing_plan_id,omitempty"`

	// Relations
	Resource    *ResourceModel    `bun:"rel:belongs-to,join:resource_id=id" json:"resource,omitempty"`
	PricingPlan *PricingPlanModel `bun:"rel:belongs-to,join:pricing_plan_id=id" json:"pricing_plan,omitempty"`
}

// TableName returns the table name for FileStorageModel
func (FileStorageModel) TableName() string {
	return "mbflow_resource_file_storage"
}

// BeforeInsert hook to set defaults
func (f *FileStorageModel) BeforeInsert(ctx any) error {
	if f.StorageLimitBytes == 0 {
		f.StorageLimitBytes = 5 * 1024 * 1024
	}
	return nil
}

// GetUsagePercent returns the storage usage percentage
func (f *FileStorageModel) GetUsagePercent() float64 {
	if f.StorageLimitBytes == 0 {
		return 0
	}
	return float64(f.UsedStorageBytes) / float64(f.StorageLimitBytes) * 100
}

// CanAddFile checks if a file of the given size can be added
func (f *FileStorageModel) CanAddFile(fileSize int64) bool {
	return f.UsedStorageBytes+fileSize <= f.StorageLimitBytes
}

// GetAvailableSpace returns the available storage space in bytes
func (f *FileStorageModel) GetAvailableSpace() int64 {
	available := f.StorageLimitBytes - f.UsedStorageBytes
	if available < 0 {
		return 0
	}
	return available
}

// PricingPlanModel represents a pricing plan in the database
type PricingPlanModel struct {
	bun.BaseModel `bun:"table:mbflow_pricing_plans,alias:pp"`

	ID                uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ResourceType      string    `bun:"resource_type,notnull" json:"resource_type" validate:"required,oneof=file_storage"`
	Name              string    `bun:"name,notnull" json:"name" validate:"required,max=255"`
	Description       string    `bun:"description" json:"description,omitempty" validate:"max=1000"`
	PricePerUnit      float64   `bun:"price_per_unit,notnull,default:0" json:"price_per_unit" validate:"min=0"`
	Unit              string    `bun:"unit,notnull" json:"unit" validate:"required,max=50"`
	StorageLimitBytes *int64    `bun:"storage_limit_bytes" json:"storage_limit_bytes,omitempty" validate:"omitempty,min=0"`
	BillingPeriod     string    `bun:"billing_period,notnull,default:'monthly'" json:"billing_period" validate:"required,oneof=monthly annual"`
	PricingModel      string    `bun:"pricing_model,notnull,default:'fixed'" json:"pricing_model" validate:"required,oneof=fixed payg tiered"`
	IsFree            bool      `bun:"is_free,notnull,default:false" json:"is_free"`
	IsActive          bool      `bun:"is_active,notnull,default:true" json:"is_active"`
	CreatedAt         time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}

// TableName returns the table name for PricingPlanModel
func (PricingPlanModel) TableName() string {
	return "mbflow_pricing_plans"
}

// BeforeInsert hook to set timestamps and defaults
func (p *PricingPlanModel) BeforeInsert(ctx any) error {
	p.CreatedAt = time.Now()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.BillingPeriod == "" {
		p.BillingPeriod = string(pkgmodels.BillingPeriodMonthly)
	}
	if p.PricingModel == "" {
		p.PricingModel = string(pkgmodels.PricingModelFixed)
	}
	return nil
}

// GetMonthlyPrice returns the monthly price of the plan
func (p *PricingPlanModel) GetMonthlyPrice() float64 {
	if p.IsFree {
		return 0
	}
	if p.BillingPeriod == string(pkgmodels.BillingPeriodAnnual) {
		return p.PricePerUnit / 12
	}
	return p.PricePerUnit
}

// GetAnnualPrice returns the annual price of the plan
func (p *PricingPlanModel) GetAnnualPrice() float64 {
	if p.IsFree {
		return 0
	}
	if p.BillingPeriod == string(pkgmodels.BillingPeriodMonthly) {
		return p.PricePerUnit * 12
	}
	return p.PricePerUnit
}

// IsAvailable checks if the plan is available for use
func (p *PricingPlanModel) IsAvailable() bool {
	return p.IsActive
}

// ============================================================================
// Domain Model Conversions
// ============================================================================

// ToResourceDomain converts ResourceModel and FileStorageModel to domain FileStorageResource
func ToResourceDomain(r *ResourceModel, fs *FileStorageModel) *pkgmodels.FileStorageResource {
	if r == nil || fs == nil {
		return nil
	}

	var metadata map[string]any
	if r.Metadata != nil {
		metadata = r.Metadata
	}

	var pricingPlanID string
	if fs.PricingPlanID != nil {
		pricingPlanID = fs.PricingPlanID.String()
	}

	return &pkgmodels.FileStorageResource{
		BaseResource: pkgmodels.BaseResource{
			ID:          r.ID.String(),
			Type:        pkgmodels.ResourceType(r.Type),
			OwnerID:     r.OwnerID.String(),
			Name:        r.Name,
			Description: r.Description,
			Status:      pkgmodels.ResourceStatus(r.Status),
			Metadata:    metadata,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		},
		StorageLimitBytes: fs.StorageLimitBytes,
		UsedStorageBytes:  fs.UsedStorageBytes,
		FileCount:         fs.FileCount,
		PricingPlanID:     pricingPlanID,
	}
}

// ToPricingPlanDomain converts PricingPlanModel to domain PricingPlan
func ToPricingPlanDomain(p *PricingPlanModel) *pkgmodels.PricingPlan {
	if p == nil {
		return nil
	}

	return &pkgmodels.PricingPlan{
		ID:                p.ID.String(),
		ResourceType:      pkgmodels.ResourceType(p.ResourceType),
		Name:              p.Name,
		Description:       p.Description,
		PricePerUnit:      p.PricePerUnit,
		Unit:              p.Unit,
		StorageLimitBytes: p.StorageLimitBytes,
		BillingPeriod:     pkgmodels.BillingPeriod(p.BillingPeriod),
		PricingModel:      pkgmodels.PricingModel(p.PricingModel),
		IsFree:            p.IsFree,
		IsActive:          p.IsActive,
		CreatedAt:         p.CreatedAt,
	}
}

// FromResourceDomain converts domain FileStorageResource to ResourceModel and FileStorageModel
func FromResourceDomain(resource *pkgmodels.FileStorageResource) (*ResourceModel, *FileStorageModel) {
	if resource == nil {
		return nil, nil
	}

	var resourceID uuid.UUID
	if resource.ID != "" {
		resourceID = uuid.MustParse(resource.ID)
	}

	var ownerID uuid.UUID
	if resource.OwnerID != "" {
		ownerID = uuid.MustParse(resource.OwnerID)
	}

	var metadata JSONBMap
	if resource.Metadata != nil {
		metadata = JSONBMap(resource.Metadata)
	}

	resourceModel := &ResourceModel{
		ID:          resourceID,
		Type:        string(resource.Type),
		OwnerID:     ownerID,
		Name:        resource.Name,
		Description: resource.Description,
		Status:      string(resource.Status),
		Metadata:    metadata,
		CreatedAt:   resource.CreatedAt,
		UpdatedAt:   resource.UpdatedAt,
	}

	var pricingPlanID *uuid.UUID
	if resource.PricingPlanID != "" {
		planID := uuid.MustParse(resource.PricingPlanID)
		pricingPlanID = &planID
	}

	fileStorageModel := &FileStorageModel{
		ResourceID:        resourceID,
		StorageLimitBytes: resource.StorageLimitBytes,
		UsedStorageBytes:  resource.UsedStorageBytes,
		FileCount:         resource.FileCount,
		PricingPlanID:     pricingPlanID,
	}

	return resourceModel, fileStorageModel
}

// FromPricingPlanDomain converts domain PricingPlan to PricingPlanModel
func FromPricingPlanDomain(plan *pkgmodels.PricingPlan) *PricingPlanModel {
	if plan == nil {
		return nil
	}

	var planID uuid.UUID
	if plan.ID != "" {
		planID = uuid.MustParse(plan.ID)
	}

	return &PricingPlanModel{
		ID:                planID,
		ResourceType:      string(plan.ResourceType),
		Name:              plan.Name,
		Description:       plan.Description,
		PricePerUnit:      plan.PricePerUnit,
		Unit:              plan.Unit,
		StorageLimitBytes: plan.StorageLimitBytes,
		BillingPeriod:     string(plan.BillingPeriod),
		PricingModel:      string(plan.PricingModel),
		IsFree:            plan.IsFree,
		IsActive:          plan.IsActive,
		CreatedAt:         plan.CreatedAt,
	}
}

// ToFileStorageResourceDomain converts ResourceModel and FileStorageModel to domain FileStorageResource
func ToFileStorageResourceDomain(r *ResourceModel, fs *FileStorageModel) pkgmodels.Resource {
	return ToResourceDomain(r, fs)
}

// ============================================================================
// Credentials Model
// ============================================================================

// CredentialsModel represents credentials-specific data in the database
type CredentialsModel struct {
	bun.BaseModel `bun:"table:mbflow_resource_credentials,alias:rc"`

	ResourceID     uuid.UUID  `bun:"resource_id,pk,type:uuid" json:"resource_id" validate:"required"`
	CredentialType string     `bun:"credential_type,notnull" json:"credential_type" validate:"required,oneof=api_key basic_auth oauth2 service_account custom"`
	EncryptedData  JSONBMap   `bun:"encrypted_data,type:jsonb,notnull,default:'{}'" json:"encrypted_data"`
	Provider       *string    `bun:"provider" json:"provider,omitempty"`
	ExpiresAt      *time.Time `bun:"expires_at" json:"expires_at,omitempty"`
	LastUsedAt     *time.Time `bun:"last_used_at" json:"last_used_at,omitempty"`
	UsageCount     int64      `bun:"usage_count,notnull,default:0" json:"usage_count"`
	PricingPlanID  *uuid.UUID `bun:"pricing_plan_id,type:uuid" json:"pricing_plan_id,omitempty"`

	// Relations
	Resource    *ResourceModel    `bun:"rel:belongs-to,join:resource_id=id" json:"resource,omitempty"`
	PricingPlan *PricingPlanModel `bun:"rel:belongs-to,join:pricing_plan_id=id" json:"pricing_plan,omitempty"`
}

// TableName returns the table name for CredentialsModel
func (CredentialsModel) TableName() string {
	return "mbflow_resource_credentials"
}

// ToCredentialsResourceDomain converts ResourceModel and CredentialsModel to domain CredentialsResource
func ToCredentialsResourceDomain(r *ResourceModel, c *CredentialsModel) *pkgmodels.CredentialsResource {
	if r == nil || c == nil {
		return nil
	}

	var metadata map[string]any
	if r.Metadata != nil {
		metadata = r.Metadata
	}

	var pricingPlanID string
	if c.PricingPlanID != nil {
		pricingPlanID = c.PricingPlanID.String()
	}

	var provider string
	if c.Provider != nil {
		provider = *c.Provider
	}

	// Convert encrypted data from JSONBMap to map[string]string
	encryptedData := make(map[string]string)
	for k, v := range c.EncryptedData {
		if str, ok := v.(string); ok {
			encryptedData[k] = str
		}
	}

	return &pkgmodels.CredentialsResource{
		BaseResource: pkgmodels.BaseResource{
			ID:          r.ID.String(),
			Type:        pkgmodels.ResourceType(r.Type),
			OwnerID:     r.OwnerID.String(),
			Name:        r.Name,
			Description: r.Description,
			Status:      pkgmodels.ResourceStatus(r.Status),
			Metadata:    metadata,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		},
		CredentialType: pkgmodels.CredentialType(c.CredentialType),
		EncryptedData:  encryptedData,
		Provider:       provider,
		ExpiresAt:      c.ExpiresAt,
		LastUsedAt:     c.LastUsedAt,
		UsageCount:     c.UsageCount,
		PricingPlanID:  pricingPlanID,
	}
}

// FromCredentialsResourceDomain converts domain CredentialsResource to ResourceModel and CredentialsModel
func FromCredentialsResourceDomain(cred *pkgmodels.CredentialsResource) (*ResourceModel, *CredentialsModel) {
	if cred == nil {
		return nil, nil
	}

	var resourceID uuid.UUID
	if cred.ID != "" {
		resourceID = uuid.MustParse(cred.ID)
	}

	var ownerID uuid.UUID
	if cred.OwnerID != "" {
		ownerID = uuid.MustParse(cred.OwnerID)
	}

	var metadata JSONBMap
	if cred.Metadata != nil {
		metadata = JSONBMap(cred.Metadata)
	}

	resourceModel := &ResourceModel{
		ID:          resourceID,
		Type:        string(cred.Type),
		OwnerID:     ownerID,
		Name:        cred.Name,
		Description: cred.Description,
		Status:      string(cred.Status),
		Metadata:    metadata,
		CreatedAt:   cred.CreatedAt,
		UpdatedAt:   cred.UpdatedAt,
	}

	var pricingPlanID *uuid.UUID
	if cred.PricingPlanID != "" {
		planID := uuid.MustParse(cred.PricingPlanID)
		pricingPlanID = &planID
	}

	var provider *string
	if cred.Provider != "" {
		provider = &cred.Provider
	}

	// Convert encrypted data from map[string]string to JSONBMap
	encryptedData := make(JSONBMap)
	for k, v := range cred.EncryptedData {
		encryptedData[k] = v
	}

	credentialsModel := &CredentialsModel{
		ResourceID:     resourceID,
		CredentialType: string(cred.CredentialType),
		EncryptedData:  encryptedData,
		Provider:       provider,
		ExpiresAt:      cred.ExpiresAt,
		LastUsedAt:     cred.LastUsedAt,
		UsageCount:     cred.UsageCount,
		PricingPlanID:  pricingPlanID,
	}

	return resourceModel, credentialsModel
}

// ============================================================================
// Rental Key Model
// ============================================================================

// RentalKeyModel represents rental key specific data in the database
type RentalKeyModel struct {
	bun.BaseModel `bun:"table:mbflow_resource_rental_key,alias:rrk"`

	ResourceID      uuid.UUID `bun:"resource_id,pk,type:uuid" json:"resource_id" validate:"required"`
	Provider        string    `bun:"provider,notnull" json:"provider" validate:"required,oneof=openai anthropic google_ai"`
	EncryptedAPIKey string    `bun:"encrypted_api_key,notnull" json:"-"` // Never exposed in JSON
	ProviderConfig  JSONBMap  `bun:"provider_config,type:jsonb,default:'{}'" json:"provider_config"`

	// Limits
	DailyRequestLimit *int   `bun:"daily_request_limit" json:"daily_request_limit"`
	MonthlyTokenLimit *int64 `bun:"monthly_token_limit" json:"monthly_token_limit"`

	// Current usage
	RequestsToday    int       `bun:"requests_today,notnull,default:0" json:"requests_today"`
	TokensThisMonth  int64     `bun:"tokens_this_month,notnull,default:0" json:"tokens_this_month"`
	LastUsageResetAt time.Time `bun:"last_usage_reset_at,notnull,default:current_timestamp" json:"last_usage_reset_at"`

	// Total statistics - Text
	TotalRequests         int64 `bun:"total_requests,notnull,default:0" json:"total_requests"`
	TotalPromptTokens     int64 `bun:"total_prompt_tokens,notnull,default:0" json:"total_prompt_tokens"`
	TotalCompletionTokens int64 `bun:"total_completion_tokens,notnull,default:0" json:"total_completion_tokens"`

	// Total statistics - Image
	TotalImageInputTokens  int64 `bun:"total_image_input_tokens,notnull,default:0" json:"total_image_input_tokens"`
	TotalImageOutputTokens int64 `bun:"total_image_output_tokens,notnull,default:0" json:"total_image_output_tokens"`

	// Total statistics - Audio
	TotalAudioInputTokens  int64 `bun:"total_audio_input_tokens,notnull,default:0" json:"total_audio_input_tokens"`
	TotalAudioOutputTokens int64 `bun:"total_audio_output_tokens,notnull,default:0" json:"total_audio_output_tokens"`

	// Total statistics - Video
	TotalVideoInputTokens  int64 `bun:"total_video_input_tokens,notnull,default:0" json:"total_video_input_tokens"`
	TotalVideoOutputTokens int64 `bun:"total_video_output_tokens,notnull,default:0" json:"total_video_output_tokens"`

	// Cost
	TotalCost float64 `bun:"total_cost,notnull,default:0" json:"total_cost"`

	// Timestamps and relations
	LastUsedAt      *time.Time `bun:"last_used_at" json:"last_used_at"`
	PricingPlanID   *uuid.UUID `bun:"pricing_plan_id,type:uuid" json:"pricing_plan_id"`
	CreatedBy       *uuid.UUID `bun:"created_by,type:uuid" json:"created_by"`
	ProvisionerType string     `bun:"provisioner_type,notnull,default:'manual'" json:"provisioner_type" validate:"required,oneof=manual auto_openai auto_anthropic auto_google"`

	// Relations
	Resource    *ResourceModel    `bun:"rel:belongs-to,join:resource_id=id" json:"resource,omitempty"`
	PricingPlan *PricingPlanModel `bun:"rel:belongs-to,join:pricing_plan_id=id" json:"pricing_plan,omitempty"`
	Creator     *UserModel        `bun:"rel:belongs-to,join:created_by=id" json:"creator,omitempty"`
}

// TableName returns the table name for RentalKeyModel
func (RentalKeyModel) TableName() string {
	return "mbflow_resource_rental_key"
}

// RentalKeyUsageModel represents a usage log entry for rental keys
type RentalKeyUsageModel struct {
	bun.BaseModel `bun:"table:mbflow_rental_key_usage,alias:rku"`

	ID          uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	RentalKeyID uuid.UUID `bun:"rental_key_id,notnull,type:uuid" json:"rental_key_id" validate:"required"`
	Model       string    `bun:"model,notnull" json:"model" validate:"required,max=100"`

	// Text tokens
	PromptTokens     int `bun:"prompt_tokens,notnull,default:0" json:"prompt_tokens"`
	CompletionTokens int `bun:"completion_tokens,notnull,default:0" json:"completion_tokens"`

	// Image tokens
	ImageInputTokens  int `bun:"image_input_tokens,notnull,default:0" json:"image_input_tokens"`
	ImageOutputTokens int `bun:"image_output_tokens,notnull,default:0" json:"image_output_tokens"`

	// Audio tokens
	AudioInputTokens  int `bun:"audio_input_tokens,notnull,default:0" json:"audio_input_tokens"`
	AudioOutputTokens int `bun:"audio_output_tokens,notnull,default:0" json:"audio_output_tokens"`

	// Video tokens
	VideoInputTokens  int `bun:"video_input_tokens,notnull,default:0" json:"video_input_tokens"`
	VideoOutputTokens int `bun:"video_output_tokens,notnull,default:0" json:"video_output_tokens"`

	// Cost and context
	EstimatedCost float64    `bun:"estimated_cost,notnull,default:0" json:"estimated_cost"`
	ExecutionID   *uuid.UUID `bun:"execution_id,type:uuid" json:"execution_id"`
	WorkflowID    *uuid.UUID `bun:"workflow_id,type:uuid" json:"workflow_id"`
	NodeID        *string    `bun:"node_id" json:"node_id"`

	// Status
	Status         string  `bun:"status,notnull,default:'success'" json:"status" validate:"required,oneof=success failed rate_limited"`
	ErrorMessage   *string `bun:"error_message" json:"error_message"`
	ResponseTimeMs *int    `bun:"response_time_ms" json:"response_time_ms"`

	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`

	// Relations
	RentalKey *RentalKeyModel `bun:"rel:belongs-to,join:rental_key_id=resource_id" json:"rental_key,omitempty"`
}

// TableName returns the table name for RentalKeyUsageModel
func (RentalKeyUsageModel) TableName() string {
	return "mbflow_rental_key_usage"
}

// GetTotalTokens returns the sum of all token types
func (r *RentalKeyUsageModel) GetTotalTokens() int {
	return r.PromptTokens + r.CompletionTokens +
		r.ImageInputTokens + r.ImageOutputTokens +
		r.AudioInputTokens + r.AudioOutputTokens +
		r.VideoInputTokens + r.VideoOutputTokens
}

// ToRentalKeyResourceDomain converts ResourceModel and RentalKeyModel to domain RentalKeyResource
func ToRentalKeyResourceDomain(r *ResourceModel, rk *RentalKeyModel) *pkgmodels.RentalKeyResource {
	if r == nil || rk == nil {
		return nil
	}

	var metadata map[string]any
	if r.Metadata != nil {
		metadata = r.Metadata
	}

	var providerConfig map[string]any
	if rk.ProviderConfig != nil {
		providerConfig = rk.ProviderConfig
	}

	var pricingPlanID string
	if rk.PricingPlanID != nil {
		pricingPlanID = rk.PricingPlanID.String()
	}

	var createdBy string
	if rk.CreatedBy != nil {
		createdBy = rk.CreatedBy.String()
	}

	return &pkgmodels.RentalKeyResource{
		BaseResource: pkgmodels.BaseResource{
			ID:          r.ID.String(),
			Type:        pkgmodels.ResourceType(r.Type),
			OwnerID:     r.OwnerID.String(),
			Name:        r.Name,
			Description: r.Description,
			Status:      pkgmodels.ResourceStatus(r.Status),
			Metadata:    metadata,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		},
		Provider:          pkgmodels.LLMProviderType(rk.Provider),
		ProviderConfig:    providerConfig,
		EncryptedAPIKey:   rk.EncryptedAPIKey,
		DailyRequestLimit: rk.DailyRequestLimit,
		MonthlyTokenLimit: rk.MonthlyTokenLimit,
		RequestsToday:     rk.RequestsToday,
		TokensThisMonth:   rk.TokensThisMonth,
		LastUsageResetAt:  rk.LastUsageResetAt,
		TotalRequests:     rk.TotalRequests,
		TotalUsage: pkgmodels.MultimodalUsage{
			PromptTokens:      rk.TotalPromptTokens,
			CompletionTokens:  rk.TotalCompletionTokens,
			ImageInputTokens:  rk.TotalImageInputTokens,
			ImageOutputTokens: rk.TotalImageOutputTokens,
			AudioInputTokens:  rk.TotalAudioInputTokens,
			AudioOutputTokens: rk.TotalAudioOutputTokens,
			VideoInputTokens:  rk.TotalVideoInputTokens,
			VideoOutputTokens: rk.TotalVideoOutputTokens,
		},
		TotalCost:       rk.TotalCost,
		LastUsedAt:      rk.LastUsedAt,
		PricingPlanID:   pricingPlanID,
		CreatedBy:       createdBy,
		ProvisionerType: pkgmodels.ProvisionerType(rk.ProvisionerType),
	}
}

// FromRentalKeyResourceDomain converts domain RentalKeyResource to ResourceModel and RentalKeyModel
func FromRentalKeyResourceDomain(rental *pkgmodels.RentalKeyResource) (*ResourceModel, *RentalKeyModel) {
	if rental == nil {
		return nil, nil
	}

	var resourceID uuid.UUID
	if rental.ID != "" {
		resourceID = uuid.MustParse(rental.ID)
	}

	var ownerID uuid.UUID
	if rental.OwnerID != "" {
		ownerID = uuid.MustParse(rental.OwnerID)
	}

	var metadata JSONBMap
	if rental.Metadata != nil {
		metadata = JSONBMap(rental.Metadata)
	}

	resourceModel := &ResourceModel{
		ID:          resourceID,
		Type:        string(rental.Type),
		OwnerID:     ownerID,
		Name:        rental.Name,
		Description: rental.Description,
		Status:      string(rental.Status),
		Metadata:    metadata,
		CreatedAt:   rental.CreatedAt,
		UpdatedAt:   rental.UpdatedAt,
	}

	var providerConfig JSONBMap
	if rental.ProviderConfig != nil {
		providerConfig = JSONBMap(rental.ProviderConfig)
	}

	var pricingPlanID *uuid.UUID
	if rental.PricingPlanID != "" {
		planID := uuid.MustParse(rental.PricingPlanID)
		pricingPlanID = &planID
	}

	var createdBy *uuid.UUID
	if rental.CreatedBy != "" {
		creatorID := uuid.MustParse(rental.CreatedBy)
		createdBy = &creatorID
	}

	rentalKeyModel := &RentalKeyModel{
		ResourceID:             resourceID,
		Provider:               string(rental.Provider),
		EncryptedAPIKey:        rental.EncryptedAPIKey,
		ProviderConfig:         providerConfig,
		DailyRequestLimit:      rental.DailyRequestLimit,
		MonthlyTokenLimit:      rental.MonthlyTokenLimit,
		RequestsToday:          rental.RequestsToday,
		TokensThisMonth:        rental.TokensThisMonth,
		LastUsageResetAt:       rental.LastUsageResetAt,
		TotalRequests:          rental.TotalRequests,
		TotalPromptTokens:      rental.TotalUsage.PromptTokens,
		TotalCompletionTokens:  rental.TotalUsage.CompletionTokens,
		TotalImageInputTokens:  rental.TotalUsage.ImageInputTokens,
		TotalImageOutputTokens: rental.TotalUsage.ImageOutputTokens,
		TotalAudioInputTokens:  rental.TotalUsage.AudioInputTokens,
		TotalAudioOutputTokens: rental.TotalUsage.AudioOutputTokens,
		TotalVideoInputTokens:  rental.TotalUsage.VideoInputTokens,
		TotalVideoOutputTokens: rental.TotalUsage.VideoOutputTokens,
		TotalCost:              rental.TotalCost,
		LastUsedAt:             rental.LastUsedAt,
		PricingPlanID:          pricingPlanID,
		CreatedBy:              createdBy,
		ProvisionerType:        string(rental.ProvisionerType),
	}

	return resourceModel, rentalKeyModel
}

// ToRentalKeyUsageRecordDomain converts RentalKeyUsageModel to domain RentalKeyUsageRecord
func ToRentalKeyUsageRecordDomain(m *RentalKeyUsageModel) *pkgmodels.RentalKeyUsageRecord {
	if m == nil {
		return nil
	}

	var executionID, workflowID, nodeID string
	if m.ExecutionID != nil {
		executionID = m.ExecutionID.String()
	}
	if m.WorkflowID != nil {
		workflowID = m.WorkflowID.String()
	}
	if m.NodeID != nil {
		nodeID = *m.NodeID
	}

	var errorMessage string
	if m.ErrorMessage != nil {
		errorMessage = *m.ErrorMessage
	}

	var responseTimeMs int
	if m.ResponseTimeMs != nil {
		responseTimeMs = *m.ResponseTimeMs
	}

	return &pkgmodels.RentalKeyUsageRecord{
		ID:          m.ID.String(),
		RentalKeyID: m.RentalKeyID.String(),
		Model:       m.Model,
		Usage: pkgmodels.MultimodalUsage{
			PromptTokens:      int64(m.PromptTokens),
			CompletionTokens:  int64(m.CompletionTokens),
			ImageInputTokens:  int64(m.ImageInputTokens),
			ImageOutputTokens: int64(m.ImageOutputTokens),
			AudioInputTokens:  int64(m.AudioInputTokens),
			AudioOutputTokens: int64(m.AudioOutputTokens),
			VideoInputTokens:  int64(m.VideoInputTokens),
			VideoOutputTokens: int64(m.VideoOutputTokens),
		},
		EstimatedCost:  m.EstimatedCost,
		ExecutionID:    executionID,
		WorkflowID:     workflowID,
		NodeID:         nodeID,
		Status:         m.Status,
		ErrorMessage:   errorMessage,
		ResponseTimeMs: responseTimeMs,
		CreatedAt:      m.CreatedAt,
	}
}

// FromRentalKeyUsageRecordDomain converts domain RentalKeyUsageRecord to RentalKeyUsageModel
func FromRentalKeyUsageRecordDomain(r *pkgmodels.RentalKeyUsageRecord) *RentalKeyUsageModel {
	if r == nil {
		return nil
	}

	var id uuid.UUID
	if r.ID != "" {
		id = uuid.MustParse(r.ID)
	}

	var rentalKeyID uuid.UUID
	if r.RentalKeyID != "" {
		rentalKeyID = uuid.MustParse(r.RentalKeyID)
	}

	var executionID, workflowID *uuid.UUID
	if r.ExecutionID != "" {
		execID := uuid.MustParse(r.ExecutionID)
		executionID = &execID
	}
	if r.WorkflowID != "" {
		wfID := uuid.MustParse(r.WorkflowID)
		workflowID = &wfID
	}

	var nodeID *string
	if r.NodeID != "" {
		nodeID = &r.NodeID
	}

	var errorMessage *string
	if r.ErrorMessage != "" {
		errorMessage = &r.ErrorMessage
	}

	var responseTimeMs *int
	if r.ResponseTimeMs > 0 {
		responseTimeMs = &r.ResponseTimeMs
	}

	return &RentalKeyUsageModel{
		ID:                id,
		RentalKeyID:       rentalKeyID,
		Model:             r.Model,
		PromptTokens:      int(r.Usage.PromptTokens),
		CompletionTokens:  int(r.Usage.CompletionTokens),
		ImageInputTokens:  int(r.Usage.ImageInputTokens),
		ImageOutputTokens: int(r.Usage.ImageOutputTokens),
		AudioInputTokens:  int(r.Usage.AudioInputTokens),
		AudioOutputTokens: int(r.Usage.AudioOutputTokens),
		VideoInputTokens:  int(r.Usage.VideoInputTokens),
		VideoOutputTokens: int(r.Usage.VideoOutputTokens),
		EstimatedCost:     r.EstimatedCost,
		ExecutionID:       executionID,
		WorkflowID:        workflowID,
		NodeID:            nodeID,
		Status:            r.Status,
		ErrorMessage:      errorMessage,
		ResponseTimeMs:    responseTimeMs,
		CreatedAt:         r.CreatedAt,
	}
}
