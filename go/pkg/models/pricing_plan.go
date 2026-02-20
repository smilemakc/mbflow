package models

import "time"

// PricingModel defines the pricing model type
type PricingModel string

const (
	PricingModelFixed  PricingModel = "fixed"
	PricingModelPayG   PricingModel = "payg"
	PricingModelTiered PricingModel = "tiered"
)

// BillingPeriod defines the billing period
type BillingPeriod string

const (
	BillingPeriodMonthly BillingPeriod = "monthly"
	BillingPeriodAnnual  BillingPeriod = "annual"
)

// PricingPlan represents a pricing plan for a resource type
type PricingPlan struct {
	ID                string        `json:"id"`
	ResourceType      ResourceType  `json:"resource_type"`
	Name              string        `json:"name"`
	Description       string        `json:"description,omitempty"`
	PricePerUnit      float64       `json:"price_per_unit"`
	Unit              string        `json:"unit"`
	StorageLimitBytes *int64        `json:"storage_limit_bytes,omitempty"`
	BillingPeriod     BillingPeriod `json:"billing_period"`
	PricingModel      PricingModel  `json:"pricing_model"`
	IsFree            bool          `json:"is_free"`
	IsActive          bool          `json:"is_active"`
	CreatedAt         time.Time     `json:"created_at"`
}

// Validate validates the pricing plan structure
func (p *PricingPlan) Validate() error {
	if p.Name == "" {
		return &ValidationError{Field: "name", Message: "plan name is required"}
	}
	if p.ResourceType == "" {
		return &ValidationError{Field: "resource_type", Message: "resource type is required"}
	}
	if !p.IsFree && p.PricePerUnit < 0 {
		return &ValidationError{Field: "price_per_unit", Message: "price cannot be negative"}
	}
	return nil
}

// GetMonthlyPrice returns the monthly price of the plan
func (p *PricingPlan) GetMonthlyPrice() float64 {
	if p.IsFree {
		return 0
	}
	if p.BillingPeriod == BillingPeriodAnnual {
		return p.PricePerUnit / 12
	}
	return p.PricePerUnit
}

// GetAnnualPrice returns the annual price of the plan
func (p *PricingPlan) GetAnnualPrice() float64 {
	if p.IsFree {
		return 0
	}
	if p.BillingPeriod == BillingPeriodMonthly {
		return p.PricePerUnit * 12
	}
	return p.PricePerUnit
}

// IsAvailable checks if the plan is available for use
func (p *PricingPlan) IsAvailable() bool {
	return p.IsActive
}
