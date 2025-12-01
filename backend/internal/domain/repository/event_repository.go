package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
)

// EventRepository defines the interface for event persistence (Event Sourcing)
type EventRepository interface {
	// Append appends a new event to the event log (immutable)
	Append(ctx context.Context, event *models.EventModel) error

	// AppendBatch appends multiple events atomically
	AppendBatch(ctx context.Context, events []*models.EventModel) error

	// FindByExecutionID retrieves all events for an execution ordered by sequence
	FindByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*models.EventModel, error)

	// FindByExecutionIDSince retrieves events since a specific sequence number
	FindByExecutionIDSince(ctx context.Context, executionID uuid.UUID, sinceSequence int64) ([]*models.EventModel, error)

	// FindByType retrieves events by type with pagination
	FindByType(ctx context.Context, eventType string, limit, offset int) ([]*models.EventModel, error)

	// FindByTimeRange retrieves events within a time range
	FindByTimeRange(ctx context.Context, from, to time.Time, limit, offset int) ([]*models.EventModel, error)

	// FindLatestByExecutionID retrieves the latest event for an execution
	FindLatestByExecutionID(ctx context.Context, executionID uuid.UUID) (*models.EventModel, error)

	// Count returns the total count of events
	Count(ctx context.Context) (int, error)

	// CountByExecutionID returns the count of events for an execution
	CountByExecutionID(ctx context.Context, executionID uuid.UUID) (int, error)

	// CountByType returns the count of events by type
	CountByType(ctx context.Context, eventType string) (int, error)

	// Stream streams events for an execution in real-time (for WebSocket observers)
	Stream(ctx context.Context, executionID uuid.UUID, fromSequence int64) (<-chan *models.EventModel, <-chan error)
}
