package repository

import (
	"context"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// PricingPlanRepository defines the interface for pricing plan operations
type PricingPlanRepository interface {
	// GetByID retrieves a pricing plan by ID
	GetByID(ctx context.Context, id string) (*models.PricingPlan, error)

	// GetByResourceType retrieves all active plans for a resource type
	GetByResourceType(ctx context.Context, resourceType models.ResourceType) ([]*models.PricingPlan, error)

	// GetFreePlan retrieves the free plan for a resource type
	GetFreePlan(ctx context.Context, resourceType models.ResourceType) (*models.PricingPlan, error)

	// GetAll retrieves all pricing plans
	GetAll(ctx context.Context) ([]*models.PricingPlan, error)

	// GetActive retrieves all active pricing plans
	GetActive(ctx context.Context) ([]*models.PricingPlan, error)
}
