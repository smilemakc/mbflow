package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/uptrace/bun"
)

// TriggerRepository implements repository.TriggerRepository
type TriggerRepository struct {
	db bun.IDB
}

// NewTriggerRepository creates a new TriggerRepository
func NewTriggerRepository(db bun.IDB) repository.TriggerRepository {
	return &TriggerRepository{db: db}
}

// Create creates a new trigger
func (r *TriggerRepository) Create(ctx context.Context, trigger *models.TriggerModel) error {
	trigger.CreatedAt = time.Now()
	trigger.UpdatedAt = time.Now()

	_, err := r.db.NewInsert().
		Model(trigger).
		Value("enabled", "?", trigger.Enabled).
		Exec(ctx)

	return err
}

// Update updates an existing trigger
func (r *TriggerRepository) Update(ctx context.Context, trigger *models.TriggerModel) error {
	trigger.UpdatedAt = time.Now()

	_, err := r.db.NewUpdate().
		Model(trigger).
		WherePK().
		Exec(ctx)

	return err
}

// Delete deletes a trigger
func (r *TriggerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*models.TriggerModel)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// FindByID retrieves a trigger by ID
func (r *TriggerRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.TriggerModel, error) {
	trigger := &models.TriggerModel{}

	err := r.db.NewSelect().
		Model(trigger).
		Where("id = ?", id).
		Scan(ctx)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return trigger, err
}

// FindByWorkflowID retrieves all triggers for a workflow
func (r *TriggerRepository) FindByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.TriggerModel, error) {
	var triggers []*models.TriggerModel

	err := r.db.NewSelect().
		Model(&triggers).
		Where("workflow_id = ?", workflowID).
		Order("created_at DESC").
		Scan(ctx)

	return triggers, err
}

// FindByType retrieves triggers by type with pagination
func (r *TriggerRepository) FindByType(ctx context.Context, triggerType string, limit, offset int) ([]*models.TriggerModel, error) {
	var triggers []*models.TriggerModel

	err := r.db.NewSelect().
		Model(&triggers).
		Where("type = ?", triggerType).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	return triggers, err
}

// FindEnabled retrieves all enabled triggers
func (r *TriggerRepository) FindEnabled(ctx context.Context) ([]*models.TriggerModel, error) {
	var triggers []*models.TriggerModel

	err := r.db.NewSelect().
		Model(&triggers).
		Where("enabled = ?", true).
		Order("created_at DESC").
		Scan(ctx)

	return triggers, err
}

// FindEnabledByType retrieves enabled triggers by type
func (r *TriggerRepository) FindEnabledByType(ctx context.Context, triggerType string) ([]*models.TriggerModel, error) {
	var triggers []*models.TriggerModel

	err := r.db.NewSelect().
		Model(&triggers).
		Where("enabled = ? AND type = ?", true, triggerType).
		Order("created_at DESC").
		Scan(ctx)

	return triggers, err
}

// FindAll retrieves all triggers with pagination
func (r *TriggerRepository) FindAll(ctx context.Context, limit, offset int) ([]*models.TriggerModel, error) {
	var triggers []*models.TriggerModel

	err := r.db.NewSelect().
		Model(&triggers).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	return triggers, err
}

// Count returns the total count of triggers
func (r *TriggerRepository) Count(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.TriggerModel)(nil)).
		Count(ctx)

	return count, err
}

// CountByWorkflowID returns the count of triggers for a workflow
func (r *TriggerRepository) CountByWorkflowID(ctx context.Context, workflowID uuid.UUID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.TriggerModel)(nil)).
		Where("workflow_id = ?", workflowID).
		Count(ctx)

	return count, err
}

// CountByType returns the count of triggers by type
func (r *TriggerRepository) CountByType(ctx context.Context, triggerType string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.TriggerModel)(nil)).
		Where("type = ?", triggerType).
		Count(ctx)

	return count, err
}

// Enable enables a trigger
func (r *TriggerRepository) Enable(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewUpdate().
		Model((*models.TriggerModel)(nil)).
		Set("enabled = ?", true).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// Disable disables a trigger
func (r *TriggerRepository) Disable(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewUpdate().
		Model((*models.TriggerModel)(nil)).
		Set("enabled = ?", false).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// MarkTriggered updates the last triggered timestamp
func (r *TriggerRepository) MarkTriggered(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewUpdate().
		Model((*models.TriggerModel)(nil)).
		Set("last_triggered_at = ?", time.Now()).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Exec(ctx)

	return err
}
