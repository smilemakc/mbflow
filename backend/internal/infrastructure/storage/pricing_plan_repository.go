package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/pkg/models"
)

var _ repository.PricingPlanRepository = (*PricingPlanRepositoryImpl)(nil)

type PricingPlanRepositoryImpl struct {
	db bun.IDB
}

func NewPricingPlanRepository(db bun.IDB) *PricingPlanRepositoryImpl {
	return &PricingPlanRepositoryImpl{db: db}
}

func (r *PricingPlanRepositoryImpl) GetByID(ctx context.Context, id string) (*pkgmodels.PricingPlan, error) {
	planID, err := uuid.Parse(id)
	if err != nil {
		return nil, pkgmodels.ErrInvalidID
	}

	planModel := new(models.PricingPlanModel)
	err = r.db.NewSelect().
		Model(planModel).
		Where("id = ?", planID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgmodels.ErrPricingPlanNotFound
		}
		return nil, err
	}

	return models.ToPricingPlanDomain(planModel), nil
}

func (r *PricingPlanRepositoryImpl) GetByResourceType(ctx context.Context, resourceType pkgmodels.ResourceType) ([]*pkgmodels.PricingPlan, error) {
	var planModels []*models.PricingPlanModel
	err := r.db.NewSelect().
		Model(&planModels).
		Where("resource_type = ? AND is_active = ?", string(resourceType), true).
		Order("price_per_unit ASC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	plans := make([]*pkgmodels.PricingPlan, len(planModels))
	for i, pm := range planModels {
		plans[i] = models.ToPricingPlanDomain(pm)
	}

	return plans, nil
}

func (r *PricingPlanRepositoryImpl) GetFreePlan(ctx context.Context, resourceType pkgmodels.ResourceType) (*pkgmodels.PricingPlan, error) {
	planModel := new(models.PricingPlanModel)
	err := r.db.NewSelect().
		Model(planModel).
		Where("resource_type = ? AND is_free = ? AND is_active = ?", string(resourceType), true, true).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, pkgmodels.ErrPricingPlanNotFound
		}
		return nil, err
	}

	return models.ToPricingPlanDomain(planModel), nil
}

func (r *PricingPlanRepositoryImpl) GetAll(ctx context.Context) ([]*pkgmodels.PricingPlan, error) {
	var planModels []*models.PricingPlanModel
	err := r.db.NewSelect().
		Model(&planModels).
		Order("resource_type ASC", "price_per_unit ASC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	plans := make([]*pkgmodels.PricingPlan, len(planModels))
	for i, pm := range planModels {
		plans[i] = models.ToPricingPlanDomain(pm)
	}

	return plans, nil
}

func (r *PricingPlanRepositoryImpl) GetActive(ctx context.Context) ([]*pkgmodels.PricingPlan, error) {
	var planModels []*models.PricingPlanModel
	err := r.db.NewSelect().
		Model(&planModels).
		Where("is_active = ?", true).
		Order("resource_type ASC", "price_per_unit ASC").
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	plans := make([]*pkgmodels.PricingPlan, len(planModels))
	for i, pm := range planModels {
		plans[i] = models.ToPricingPlanDomain(pm)
	}

	return plans, nil
}
