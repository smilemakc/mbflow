package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

// ResourceModel represents a base resource in the database
type ResourceModel struct {
	bun.BaseModel `bun:"table:resources,alias:r"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Type        string     `bun:"type,notnull" json:"type" validate:"required,oneof=file_storage"`
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
}

// TableName returns the table name for ResourceModel
func (ResourceModel) TableName() string {
	return "resources"
}

// BeforeInsert hook to set timestamps and defaults
func (r *ResourceModel) BeforeInsert(ctx interface{}) error {
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
func (r *ResourceModel) BeforeUpdate(ctx interface{}) error {
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
	bun.BaseModel `bun:"table:resource_file_storage,alias:rfs"`

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
	return "resource_file_storage"
}

// BeforeInsert hook to set defaults
func (f *FileStorageModel) BeforeInsert(ctx interface{}) error {
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
	bun.BaseModel `bun:"table:pricing_plans,alias:pp"`

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
	return "pricing_plans"
}

// BeforeInsert hook to set timestamps and defaults
func (p *PricingPlanModel) BeforeInsert(ctx interface{}) error {
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

	var metadata map[string]interface{}
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
