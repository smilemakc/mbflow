package observer

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
)

// DatabaseObserver persists all events to PostgreSQL via EventRepository
type DatabaseObserver struct {
	name string
	repo repository.EventRepository
}

// NewDatabaseObserver creates a new database observer
func NewDatabaseObserver(repo repository.EventRepository) *DatabaseObserver {
	return &DatabaseObserver{
		name: "database",
		repo: repo,
	}
}

// Name returns the observer's name
func (o *DatabaseObserver) Name() string {
	return o.name
}

// Filter returns nil to receive all events
func (o *DatabaseObserver) Filter() EventFilter {
	return nil // No filter - store all events
}

// OnEvent handles event persistence
func (o *DatabaseObserver) OnEvent(ctx context.Context, event Event) error {
	eventModel := o.convertToEventModel(event)
	return o.repo.Append(ctx, eventModel)
}

// convertToEventModel converts observer.Event to storage.EventModel
func (o *DatabaseObserver) convertToEventModel(event Event) *models.EventModel {
	executionUUID, _ := uuid.Parse(event.ExecutionID)

	payload := models.JSONBMap{
		"workflow_id": event.WorkflowID,
		"status":      event.Status,
		"timestamp":   event.Timestamp.Format(time.RFC3339),
	}

	// Add node-specific fields
	if event.NodeID != nil {
		payload["node_id"] = *event.NodeID
	}
	if event.NodeName != nil {
		payload["node_name"] = *event.NodeName
	}
	if event.NodeType != nil {
		payload["node_type"] = *event.NodeType
	}

	// Add wave-specific fields
	if event.WaveIndex != nil {
		payload["wave_index"] = *event.WaveIndex
	}
	if event.NodeCount != nil {
		payload["node_count"] = *event.NodeCount
	}

	// Add timing information (REQUIRED by user)
	if event.DurationMs != nil {
		payload["duration_ms"] = *event.DurationMs
	}

	// Add error if present
	if event.Error != nil {
		payload["error"] = event.Error.Error()
	}

	// Add input/output data
	if event.Input != nil {
		payload["input"] = event.Input
	}
	if event.Output != nil {
		payload["output"] = event.Output
	}

	// Add variables
	if event.Variables != nil {
		payload["variables"] = event.Variables
	}

	// Add metadata
	if event.Metadata != nil {
		payload["metadata"] = event.Metadata
	}

	return &models.EventModel{
		ExecutionID: executionUUID,
		EventType:   string(event.Type),
		Payload:     payload,
	}
}
