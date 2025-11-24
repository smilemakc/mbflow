package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
	"github.com/uptrace/bun"
)

// MemoryEventStore is an in-memory implementation of EventStore for development/testing
type MemoryEventStore struct {
	mu     sync.RWMutex
	events map[uuid.UUID][]domain.Event // executionID -> events
}

// NewMemoryEventStore creates a new in-memory event store
func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{
		events: make(map[uuid.UUID][]domain.Event),
	}
}

// AppendEvent appends a single event to the event stream
func (es *MemoryEventStore) AppendEvent(ctx context.Context, event domain.Event) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	executionID := event.ExecutionID()

	// Initialize slice if needed
	if es.events[executionID] == nil {
		es.events[executionID] = make([]domain.Event, 0)
	}

	// Append event
	es.events[executionID] = append(es.events[executionID], event)

	return nil
}

// AppendEvents appends multiple events atomically
func (es *MemoryEventStore) AppendEvents(ctx context.Context, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	// Group events by execution ID
	eventsByExecution := make(map[uuid.UUID][]domain.Event)
	for _, event := range events {
		executionID := event.ExecutionID()
		eventsByExecution[executionID] = append(eventsByExecution[executionID], event)
	}

	// Append all events
	for executionID, execEvents := range eventsByExecution {
		if es.events[executionID] == nil {
			es.events[executionID] = make([]domain.Event, 0)
		}
		es.events[executionID] = append(es.events[executionID], execEvents...)
	}

	return nil
}

// GetEvents retrieves all events for an execution
func (es *MemoryEventStore) GetEvents(ctx context.Context, executionID uuid.UUID) ([]domain.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events := es.events[executionID]
	if events == nil {
		return []domain.Event{}, nil
	}

	// Return a copy to prevent external modification
	result := make([]domain.Event, len(events))
	copy(result, events)

	return result, nil
}

// GetEventsSince retrieves events after a specific sequence number
func (es *MemoryEventStore) GetEventsSince(ctx context.Context, executionID uuid.UUID, sequenceNumber int64) ([]domain.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	allEvents := es.events[executionID]
	if allEvents == nil {
		return []domain.Event{}, nil
	}

	// Filter events with sequence number > sequenceNumber
	result := make([]domain.Event, 0)
	for _, event := range allEvents {
		if event.SequenceNumber() > sequenceNumber {
			result = append(result, event)
		}
	}

	return result, nil
}

// GetEventsByType retrieves events of a specific type
func (es *MemoryEventStore) GetEventsByType(ctx context.Context, executionID uuid.UUID, eventType domain.EventType) ([]domain.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	allEvents := es.events[executionID]
	if allEvents == nil {
		return []domain.Event{}, nil
	}

	// Filter events by type
	result := make([]domain.Event, 0)
	for _, event := range allEvents {
		if event.EventType() == eventType {
			result = append(result, event)
		}
	}

	return result, nil
}

// GetEventsByWorkflow retrieves all events for a workflow (all executions)
func (es *MemoryEventStore) GetEventsByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]domain.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	result := make([]domain.Event, 0)

	// Iterate through all executions
	for _, events := range es.events {
		for _, event := range events {
			if event.WorkflowID() == workflowID {
				result = append(result, event)
			}
		}
	}

	return result, nil
}

// GetEventCount returns the number of events for an execution
func (es *MemoryEventStore) GetEventCount(ctx context.Context, executionID uuid.UUID) (int64, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	events := es.events[executionID]
	if events == nil {
		return 0, nil
	}

	return int64(len(events)), nil
}

// Clear clears all events (useful for testing)
func (es *MemoryEventStore) Clear() {
	es.mu.Lock()
	defer es.mu.Unlock()

	es.events = make(map[uuid.UUID][]domain.Event)
}

// PostgresEventStore is a PostgreSQL-based event store implementation using Bun ORM
type PostgresEventStore struct {
	db *bun.DB
	mu sync.RWMutex
}

// EventRecord represents the database schema for events
type EventRecord struct {
	bun.BaseModel `bun:"table:events,alias:ev"`

	ID             int64             `bun:"id,pk,autoincrement"`
	EventID        uuid.UUID         `bun:"event_id,notnull,unique"`
	ExecutionID    uuid.UUID         `bun:"execution_id,notnull"`
	WorkflowID     uuid.UUID         `bun:"workflow_id,notnull"`
	NodeID         uuid.UUID         `bun:"node_id"`
	SequenceNumber int64             `bun:"sequence_number,notnull"`
	EventType      domain.EventType  `bun:"event_type,notnull"`
	Payload        map[string]any    `bun:"payload,type:jsonb"`
	Metadata       map[string]string `bun:"metadata,type:jsonb"`
	CreatedAt      time.Time         `bun:"created_at,notnull,default:current_timestamp"`
}

// NewPostgresEventStore creates a new PostgreSQL event store
func NewPostgresEventStore(db *bun.DB) *PostgresEventStore {
	return &PostgresEventStore{
		db: db,
	}
}

// InitSchema creates the events table if it doesn't exist
func (es *PostgresEventStore) InitSchema(ctx context.Context) error {
	_, err := es.db.NewCreateTable().
		Model((*EventRecord)(nil)).
		IfNotExists().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to create events table: %w", err)
	}

	// Create indexes for better query performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_events_execution_id ON events(execution_id)",
		"CREATE INDEX IF NOT EXISTS idx_events_workflow_id ON events(workflow_id)",
		"CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type)",
		"CREATE INDEX IF NOT EXISTS idx_events_sequence_number ON events(execution_id, sequence_number)",
	}

	for _, indexSQL := range indexes {
		if _, err := es.db.ExecContext(ctx, indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// eventToRecord converts domain.Event to EventRecord
func eventToRecord(event domain.Event) (*EventRecord, error) {
	return &EventRecord{
		EventID:        event.EventID(),
		ExecutionID:    event.ExecutionID(),
		WorkflowID:     event.WorkflowID(),
		SequenceNumber: event.SequenceNumber(),
		EventType:      event.EventType(),
		Payload:        event.Data(),
		Metadata:       event.Metadata(),
		CreatedAt:      event.Timestamp(),
	}, nil
}

// recordToEvent converts EventRecord to domain.Event
func recordToEvent(record *EventRecord) (domain.Event, error) {
	return domain.ReconstructEvent(
		record.EventID,
		record.EventType,
		record.ExecutionID, // aggregateID = executionID
		record.CreatedAt,
		record.SequenceNumber,
		record.WorkflowID,
		record.NodeID,
		record.Payload,
		record.Metadata,
	), nil
}

// AppendEvent appends a single event
func (es *PostgresEventStore) AppendEvent(ctx context.Context, event domain.Event) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	record, err := eventToRecord(event)
	if err != nil {
		return fmt.Errorf("failed to convert event to record: %w", err)
	}

	_, err = es.db.NewInsert().
		Model(record).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to append event: %w", err)
	}

	return nil
}

// AppendEvents appends multiple events atomically using a transaction
func (es *PostgresEventStore) AppendEvents(ctx context.Context, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	es.mu.Lock()
	defer es.mu.Unlock()

	// Start transaction
	tx, err := es.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Convert events to records
	records := make([]*EventRecord, len(events))
	for i, event := range events {
		record, err := eventToRecord(event)
		if err != nil {
			return fmt.Errorf("failed to convert event %d to record: %w", i, err)
		}
		records[i] = record
	}

	// Insert all events in one batch
	_, err = tx.NewInsert().
		Model(&records).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to insert events: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetEvents retrieves all events for an execution
func (es *PostgresEventStore) GetEvents(ctx context.Context, executionID uuid.UUID) ([]domain.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var records []EventRecord
	err := es.db.NewSelect().
		Model(&records).
		Where("execution_id = ?", executionID).
		Order("sequence_number ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Convert records to domain events
	events := make([]domain.Event, 0, len(records))
	for _, record := range records {
		event, err := recordToEvent(&record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// GetEventsSince retrieves events after a specific sequence number
func (es *PostgresEventStore) GetEventsSince(ctx context.Context, executionID uuid.UUID, sequenceNumber int64) ([]domain.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var records []EventRecord
	err := es.db.NewSelect().
		Model(&records).
		Where("execution_id = ? AND sequence_number > ?", executionID, sequenceNumber).
		Order("sequence_number ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get events since: %w", err)
	}

	// Convert records to domain events
	events := make([]domain.Event, 0, len(records))
	for _, record := range records {
		event, err := recordToEvent(&record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// GetEventsByType retrieves events of a specific type
func (es *PostgresEventStore) GetEventsByType(ctx context.Context, executionID uuid.UUID, eventType domain.EventType) ([]domain.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var records []EventRecord
	err := es.db.NewSelect().
		Model(&records).
		Where("execution_id = ? AND event_type = ?", executionID, eventType).
		Order("sequence_number ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get events by type: %w", err)
	}

	// Convert records to domain events
	events := make([]domain.Event, 0, len(records))
	for _, record := range records {
		event, err := recordToEvent(&record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// GetEventsByWorkflow retrieves all events for a workflow (across all executions)
func (es *PostgresEventStore) GetEventsByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]domain.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var records []EventRecord
	err := es.db.NewSelect().
		Model(&records).
		Where("workflow_id = ?", workflowID).
		Order("created_at ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get events by workflow: %w", err)
	}

	// Convert records to domain events
	events := make([]domain.Event, 0, len(records))
	for _, record := range records {
		event, err := recordToEvent(&record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

// GetEventCount returns the number of events for an execution
func (es *PostgresEventStore) GetEventCount(ctx context.Context, executionID uuid.UUID) (int64, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	count, err := es.db.NewSelect().
		Model((*EventRecord)(nil)).
		Where("execution_id = ?", executionID).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to get event count: %w", err)
	}

	return int64(count), nil
}

// EventStoreWithSnapshots wraps an event store with snapshot support
type EventStoreWithSnapshots struct {
	eventStore    domain.EventStore
	snapshotStore SnapshotStore
	mu            sync.RWMutex

	// Configuration
	snapshotInterval int64 // Take snapshot every N events
}

// SnapshotStore defines interface for storing execution snapshots
type SnapshotStore interface {
	SaveSnapshot(ctx context.Context, executionID uuid.UUID, sequenceNumber int64, state map[string]any) error
	GetLatestSnapshot(ctx context.Context, executionID uuid.UUID) (sequenceNumber int64, state map[string]any, err error)
}

// NewEventStoreWithSnapshots creates an event store with snapshot support
func NewEventStoreWithSnapshots(eventStore domain.EventStore, snapshotStore SnapshotStore, snapshotInterval int64) *EventStoreWithSnapshots {
	return &EventStoreWithSnapshots{
		eventStore:       eventStore,
		snapshotStore:    snapshotStore,
		snapshotInterval: snapshotInterval,
	}
}

// AppendEvent appends an event and creates snapshot if needed
func (es *EventStoreWithSnapshots) AppendEvent(ctx context.Context, event domain.Event) error {
	if err := es.eventStore.AppendEvent(ctx, event); err != nil {
		return err
	}

	// Check if we should create a snapshot
	count, err := es.eventStore.GetEventCount(ctx, event.ExecutionID())
	if err != nil {
		return err
	}

	if count%es.snapshotInterval == 0 {
		// Create snapshot (would need execution state here)
		// This is simplified - real implementation would rebuild state from events
		_ = es.snapshotStore.SaveSnapshot(ctx, event.ExecutionID(), event.SequenceNumber(), nil)
	}

	return nil
}

// AppendEvents appends multiple events
func (es *EventStoreWithSnapshots) AppendEvents(ctx context.Context, events []domain.Event) error {
	return es.eventStore.AppendEvents(ctx, events)
}

// GetEvents retrieves events, potentially from snapshot
func (es *EventStoreWithSnapshots) GetEvents(ctx context.Context, executionID uuid.UUID) ([]domain.Event, error) {
	// Try to get latest snapshot
	snapshotSeq, _, err := es.snapshotStore.GetLatestSnapshot(ctx, executionID)
	if err == nil && snapshotSeq > 0 {
		// Get events since snapshot
		return es.eventStore.GetEventsSince(ctx, executionID, snapshotSeq)
	}

	// No snapshot, get all events
	return es.eventStore.GetEvents(ctx, executionID)
}

// GetEventsSince delegates to underlying event store
func (es *EventStoreWithSnapshots) GetEventsSince(ctx context.Context, executionID uuid.UUID, sequenceNumber int64) ([]domain.Event, error) {
	return es.eventStore.GetEventsSince(ctx, executionID, sequenceNumber)
}

// GetEventsByType delegates to underlying event store
func (es *EventStoreWithSnapshots) GetEventsByType(ctx context.Context, executionID uuid.UUID, eventType domain.EventType) ([]domain.Event, error) {
	return es.eventStore.GetEventsByType(ctx, executionID, eventType)
}

// GetEventsByWorkflow delegates to underlying event store
func (es *EventStoreWithSnapshots) GetEventsByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]domain.Event, error) {
	return es.eventStore.GetEventsByWorkflow(ctx, workflowID)
}

// GetEventCount delegates to underlying event store
func (es *EventStoreWithSnapshots) GetEventCount(ctx context.Context, executionID uuid.UUID) (int64, error) {
	return es.eventStore.GetEventCount(ctx, executionID)
}

// MemorySnapshotStore is an in-memory snapshot store
type MemorySnapshotStore struct {
	mu        sync.RWMutex
	snapshots map[uuid.UUID]struct {
		sequenceNumber int64
		state          map[string]any
	}
}

// NewMemorySnapshotStore creates a new in-memory snapshot store
func NewMemorySnapshotStore() *MemorySnapshotStore {
	return &MemorySnapshotStore{
		snapshots: make(map[uuid.UUID]struct {
			sequenceNumber int64
			state          map[string]any
		}),
	}
}

// SaveSnapshot saves an execution snapshot
func (ss *MemorySnapshotStore) SaveSnapshot(ctx context.Context, executionID uuid.UUID, sequenceNumber int64, state map[string]any) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.snapshots[executionID] = struct {
		sequenceNumber int64
		state          map[string]any
	}{
		sequenceNumber: sequenceNumber,
		state:          state,
	}

	return nil
}

// GetLatestSnapshot retrieves the latest snapshot for an execution
func (ss *MemorySnapshotStore) GetLatestSnapshot(ctx context.Context, executionID uuid.UUID) (int64, map[string]any, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	snapshot, exists := ss.snapshots[executionID]
	if !exists {
		return 0, nil, fmt.Errorf("snapshot not found")
	}

	return snapshot.sequenceNumber, snapshot.state, nil
}
