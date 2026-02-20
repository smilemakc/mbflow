package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
)

// TriggerRepository defines the interface for trigger persistence
type TriggerRepository interface {
	// Create creates a new trigger
	Create(ctx context.Context, trigger *models.TriggerModel) error

	// Update updates an existing trigger
	Update(ctx context.Context, trigger *models.TriggerModel) error

	// Delete deletes a trigger
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a trigger by ID
	FindByID(ctx context.Context, id uuid.UUID) (*models.TriggerModel, error)

	// FindByWorkflowID retrieves all triggers for a workflow
	FindByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*models.TriggerModel, error)

	// FindByType retrieves triggers by type with pagination
	FindByType(ctx context.Context, triggerType string, limit, offset int) ([]*models.TriggerModel, error)

	// FindEnabled retrieves all enabled triggers
	FindEnabled(ctx context.Context) ([]*models.TriggerModel, error)

	// FindEnabledByType retrieves enabled triggers by type
	FindEnabledByType(ctx context.Context, triggerType string) ([]*models.TriggerModel, error)

	// FindAll retrieves all triggers with pagination
	FindAll(ctx context.Context, limit, offset int) ([]*models.TriggerModel, error)

	// Count returns the total count of triggers
	Count(ctx context.Context) (int, error)

	// CountByWorkflowID returns the count of triggers for a workflow
	CountByWorkflowID(ctx context.Context, workflowID uuid.UUID) (int, error)

	// CountByType returns the count of triggers by type
	CountByType(ctx context.Context, triggerType string) (int, error)

	// Enable enables a trigger
	Enable(ctx context.Context, id uuid.UUID) error

	// Disable disables a trigger
	Disable(ctx context.Context, id uuid.UUID) error

	// MarkTriggered updates the last triggered timestamp
	MarkTriggered(ctx context.Context, id uuid.UUID) error
}
