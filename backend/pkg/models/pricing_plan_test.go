package models

import (
	"testing"
	"time"
)

func TestPricingPlan_Validate(t *testing.T) {
	tests := []struct {
		name    string
		plan    *PricingPlan
		wantErr bool
	}{
		{
			name: "valid free plan",
			plan: &PricingPlan{
				ResourceType: ResourceTypeFileStorage,
				Name:         "Free",
				IsFree:       true,
				PricePerUnit: 0,
			},
			wantErr: false,
		},
		{
			name: "valid paid plan",
			plan: &PricingPlan{
				ResourceType: ResourceTypeFileStorage,
				Name:         "Premium",
				IsFree:       false,
				PricePerUnit: 9.99,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			plan: &PricingPlan{
				ResourceType: ResourceTypeFileStorage,
				IsFree:       false,
				PricePerUnit: 9.99,
			},
			wantErr: true,
		},
		{
			name: "missing resource type",
			plan: &PricingPlan{
				Name:         "Premium",
				IsFree:       false,
				PricePerUnit: 9.99,
			},
			wantErr: true,
		},
		{
			name: "negative price on paid plan",
			plan: &PricingPlan{
				ResourceType: ResourceTypeFileStorage,
				Name:         "Premium",
				IsFree:       false,
				PricePerUnit: -9.99,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PricingPlan.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPricingPlan_GetMonthlyPrice(t *testing.T) {
	tests := []struct {
		name          string
		plan          *PricingPlan
		expectedPrice float64
	}{
		{
			name: "free plan",
			plan: &PricingPlan{
				IsFree:        true,
				PricePerUnit:  0,
				BillingPeriod: BillingPeriodMonthly,
			},
			expectedPrice: 0,
		},
		{
			name: "monthly plan",
			plan: &PricingPlan{
				IsFree:        false,
				PricePerUnit:  9.99,
				BillingPeriod: BillingPeriodMonthly,
			},
			expectedPrice: 9.99,
		},
		{
			name: "annual plan",
			plan: &PricingPlan{
				IsFree:        false,
				PricePerUnit:  120,
				BillingPeriod: BillingPeriodAnnual,
			},
			expectedPrice: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.plan.GetMonthlyPrice()
			if got != tt.expectedPrice {
				t.Errorf("GetMonthlyPrice() = %v, want %v", got, tt.expectedPrice)
			}
		})
	}
}

func TestPricingPlan_GetAnnualPrice(t *testing.T) {
	tests := []struct {
		name          string
		plan          *PricingPlan
		expectedPrice float64
	}{
		{
			name: "free plan",
			plan: &PricingPlan{
				IsFree:        true,
				PricePerUnit:  0,
				BillingPeriod: BillingPeriodAnnual,
			},
			expectedPrice: 0,
		},
		{
			name: "annual plan",
			plan: &PricingPlan{
				IsFree:        false,
				PricePerUnit:  120,
				BillingPeriod: BillingPeriodAnnual,
			},
			expectedPrice: 120,
		},
		{
			name: "monthly plan",
			plan: &PricingPlan{
				IsFree:        false,
				PricePerUnit:  10,
				BillingPeriod: BillingPeriodMonthly,
			},
			expectedPrice: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.plan.GetAnnualPrice()
			if got != tt.expectedPrice {
				t.Errorf("GetAnnualPrice() = %v, want %v", got, tt.expectedPrice)
			}
		})
	}
}

func TestPricingPlan_IsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		isActive bool
		expected bool
	}{
		{
			name:     "active plan is available",
			isActive: true,
			expected: true,
		},
		{
			name:     "inactive plan is not available",
			isActive: false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &PricingPlan{
				ResourceType: ResourceTypeFileStorage,
				Name:         "Test Plan",
				IsActive:     tt.isActive,
				CreatedAt:    time.Now(),
			}

			got := plan.IsAvailable()
			if got != tt.expected {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPricingModel_Values(t *testing.T) {
	models := []PricingModel{
		PricingModelFixed,
		PricingModelPayG,
		PricingModelTiered,
	}

	for _, model := range models {
		plan := &PricingPlan{
			ResourceType: ResourceTypeFileStorage,
			Name:         "Test Plan",
			PricingModel: model,
		}

		if plan.PricingModel != model {
			t.Errorf("PricingModel = %v, want %v", plan.PricingModel, model)
		}
	}
}

func TestBillingPeriod_Values(t *testing.T) {
	periods := []BillingPeriod{
		BillingPeriodMonthly,
		BillingPeriodAnnual,
	}

	for _, period := range periods {
		plan := &PricingPlan{
			ResourceType:  ResourceTypeFileStorage,
			Name:          "Test Plan",
			BillingPeriod: period,
		}

		if plan.BillingPeriod != period {
			t.Errorf("BillingPeriod = %v, want %v", plan.BillingPeriod, period)
		}
	}
}
