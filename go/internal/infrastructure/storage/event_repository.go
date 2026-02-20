package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/uptrace/bun"
)

// Ensure EventRepository implements the interface
var _ repository.EventRepository = (*EventRepository)(nil)

// EventRepository implements repository.EventRepository using Bun ORM
type EventRepository struct {
	db bun.IDB
}

// NewEventRepository creates a new EventRepository
func NewEventRepository(db bun.IDB) *EventRepository {
	return &EventRepository{db: db}
}

// Append appends a new event to the event log (immutable)
func (r *EventRepository) Append(ctx context.Context, event *models.EventModel) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	_, err := r.db.NewInsert().Model(event).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to append event: %w", err)
	}
	return nil
}

// AppendBatch appends multiple events atomically
func (r *EventRepository) AppendBatch(ctx context.Context, events []*models.EventModel) error {
	if len(events) == 0 {
		return nil
	}

	// Set IDs for events that don't have one
	for _, event := range events {
		if event.ID == uuid.Nil {
			event.ID = uuid.New()
		}
	}

	_, err := r.db.NewInsert().Model(&events).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to append events batch: %w", err)
	}
	return nil
}

// FindByExecutionID retrieves all events for an execution ordered by sequence
func (r *EventRepository) FindByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*models.EventModel, error) {
	var events []*models.EventModel
	err := r.db.NewSelect().
		Model(&events).
		Where("execution_id = ?", executionID).
		Order("sequence ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find events by execution ID: %w", err)
	}
	return events, nil
}

// FindByExecutionIDSince retrieves events since a specific sequence number
func (r *EventRepository) FindByExecutionIDSince(ctx context.Context, executionID uuid.UUID, sinceSequence int64) ([]*models.EventModel, error) {
	var events []*models.EventModel
	err := r.db.NewSelect().
		Model(&events).
		Where("execution_id = ?", executionID).
		Where("sequence > ?", sinceSequence).
		Order("sequence ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find events since sequence: %w", err)
	}
	return events, nil
}

// FindByType retrieves events by type with pagination
func (r *EventRepository) FindByType(ctx context.Context, eventType string, limit, offset int) ([]*models.EventModel, error) {
	var events []*models.EventModel
	err := r.db.NewSelect().
		Model(&events).
		Where("event_type = ?", eventType).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find events by type: %w", err)
	}
	return events, nil
}

// FindByTimeRange retrieves events within a time range
func (r *EventRepository) FindByTimeRange(ctx context.Context, from, to time.Time, limit, offset int) ([]*models.EventModel, error) {
	var events []*models.EventModel
	err := r.db.NewSelect().
		Model(&events).
		Where("created_at >= ?", from).
		Where("created_at <= ?", to).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find events by time range: %w", err)
	}
	return events, nil
}

// FindLatestByExecutionID retrieves the latest event for an execution
func (r *EventRepository) FindLatestByExecutionID(ctx context.Context, executionID uuid.UUID) (*models.EventModel, error) {
	var event models.EventModel
	err := r.db.NewSelect().
		Model(&event).
		Where("execution_id = ?", executionID).
		Order("sequence DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find latest event: %w", err)
	}
	return &event, nil
}

// Count returns the total count of events
func (r *EventRepository) Count(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().Model((*models.EventModel)(nil)).Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}

// CountByExecutionID returns the count of events for an execution
func (r *EventRepository) CountByExecutionID(ctx context.Context, executionID uuid.UUID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.EventModel)(nil)).
		Where("execution_id = ?", executionID).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count events by execution ID: %w", err)
	}
	return count, nil
}

// CountByType returns the count of events by type
func (r *EventRepository) CountByType(ctx context.Context, eventType string) (int, error) {
	count, err := r.db.NewSelect().
		Model((*models.EventModel)(nil)).
		Where("event_type = ?", eventType).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count events by type: %w", err)
	}
	return count, nil
}

// Stream streams events for an execution in real-time (for WebSocket observers)
// Note: This is a simplified implementation. For production, consider using PostgreSQL LISTEN/NOTIFY
func (r *EventRepository) Stream(ctx context.Context, executionID uuid.UUID, fromSequence int64) (<-chan *models.EventModel, <-chan error) {
	eventChan := make(chan *models.EventModel, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(eventChan)
		defer close(errChan)

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		lastSequence := fromSequence

		// Initial fetch of existing events before entering the poll loop
		initialEvents, err := r.FindByExecutionIDSince(ctx, executionID, lastSequence)
		if err != nil {
			errChan <- err
			return
		}
		for _, event := range initialEvents {
			select {
			case eventChan <- event:
				if event.Sequence > lastSequence {
					lastSequence = event.Sequence
				}
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case <-ticker.C:
				events, err := r.FindByExecutionIDSince(ctx, executionID, lastSequence)
				if err != nil {
					errChan <- err
					return
				}

				for _, event := range events {
					select {
					case eventChan <- event:
						if event.Sequence > lastSequence {
							lastSequence = event.Sequence
						}
					case <-ctx.Done():
						errChan <- ctx.Err()
						return
					}
				}
			}
		}
	}()

	return eventChan, errChan
}
