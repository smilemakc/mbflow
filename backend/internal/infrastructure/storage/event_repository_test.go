package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func setupEventRepoTest(t *testing.T) (*EventRepository, *bun.DB, func()) {
	ctx := context.Background()

	// Start PostgreSQL container
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "mbflow_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := postgres.Host(ctx)
	require.NoError(t, err)

	port, err := postgres.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Connect to database
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/mbflow_test?sslmode=disable", host, port.Port())

	// Wait a bit for the database to be fully ready
	time.Sleep(500 * time.Millisecond)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New(), bun.WithDiscardUnknownColumns())

	// Run migrations
	migrator, err := NewMigrator(db, "../../../migrations")
	require.NoError(t, err)
	err = migrator.Init(ctx)
	require.NoError(t, err)
	err = migrator.Up(ctx)
	require.NoError(t, err)

	repo := NewEventRepository(db)

	cleanup := func() {
		db.Close()
		postgres.Terminate(ctx)
	}

	return repo, db, cleanup
}

func createTestExecution(t *testing.T, db *bun.DB) *models.ExecutionModel {
	workflowRepo := NewWorkflowRepository(db)

	// Use unique workflow name to avoid duplicate key violations
	workflow := &models.WorkflowModel{
		Name:      fmt.Sprintf("Test Workflow %s", uuid.New().String()[:8]),
		Status:    "active",
		Version:   1,
		Variables: models.JSONBMap{},
		Metadata:  models.JSONBMap{},
	}

	err := workflowRepo.Create(context.Background(), workflow)
	require.NoError(t, err)

	executionRepo := NewExecutionRepository(db)
	execution := &models.ExecutionModel{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "running",
	}

	err = executionRepo.Create(context.Background(), execution)
	require.NoError(t, err)

	return execution
}

// ========== APPEND TESTS ==========

func TestEventRepo_Append_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	event := &models.EventModel{
		ExecutionID: execution.ID,
		EventType:   models.EventTypeExecutionStarted,
		Payload:     models.JSONBMap{"status": "started"},
	}

	err := repo.Append(context.Background(), event)
	require.NoError(t, err)

	assert.NotEqual(t, uuid.Nil, event.ID)
	assert.Greater(t, event.Sequence, int64(0))
}

func TestEventRepo_Append_GeneratesSequence(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create multiple events
	for i := 0; i < 3; i++ {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{"node": fmt.Sprintf("node%d", i)},
		}

		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
		assert.Equal(t, int64(i+1), event.Sequence)
	}
}

func TestEventRepo_Append_InvalidExecutionID(t *testing.T) {
	repo, _, cleanup := setupEventRepoTest(t)
	defer cleanup()

	event := &models.EventModel{
		ExecutionID: uuid.New(), // Non-existent execution
		EventType:   models.EventTypeExecutionStarted,
		Payload:     models.JSONBMap{},
	}

	err := repo.Append(context.Background(), event)
	assert.Error(t, err) // Should fail due to foreign key constraint
}

// ========== APPEND BATCH TESTS ==========

func TestEventRepo_AppendBatch_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	events := []*models.EventModel{
		{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeWaveStarted,
			Payload:     models.JSONBMap{"wave": 0},
		},
		{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{"node": "n1"},
		},
		{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeCompleted,
			Payload:     models.JSONBMap{"node": "n1"},
		},
	}

	err := repo.AppendBatch(context.Background(), events)
	require.NoError(t, err)

	// Verify all events have IDs and sequences
	for i, event := range events {
		assert.NotEqual(t, uuid.Nil, event.ID)
		assert.Equal(t, int64(i+1), event.Sequence)
	}
}

func TestEventRepo_AppendBatch_EmptySlice(t *testing.T) {
	repo, _, cleanup := setupEventRepoTest(t)
	defer cleanup()

	err := repo.AppendBatch(context.Background(), []*models.EventModel{})
	require.NoError(t, err) // Should not error on empty slice
}

// ========== FIND BY EXECUTION ID TESTS ==========

func TestEventRepo_FindByExecutionID_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create multiple events
	eventTypes := []string{
		models.EventTypeExecutionStarted,
		models.EventTypeNodeStarted,
		models.EventTypeNodeCompleted,
	}

	for _, eventType := range eventTypes {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   eventType,
			Payload:     models.JSONBMap{},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
	}

	events, err := repo.FindByExecutionID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Len(t, events, 3)

	// Verify order (should be sorted by sequence)
	for i := 0; i < len(events)-1; i++ {
		assert.Less(t, events[i].Sequence, events[i+1].Sequence)
	}
}

func TestEventRepo_FindByExecutionID_Empty(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	events, err := repo.FindByExecutionID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

// ========== FIND BY EXECUTION ID SINCE TESTS ==========

func TestEventRepo_FindByExecutionIDSince_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create 5 events
	for i := 0; i < 5; i++ {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{"index": i},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
	}

	// Find events since sequence 3
	events, err := repo.FindByExecutionIDSince(context.Background(), execution.ID, 3)
	require.NoError(t, err)
	assert.Len(t, events, 2) // Should return events with sequence 4 and 5

	assert.Equal(t, int64(4), events[0].Sequence)
	assert.Equal(t, int64(5), events[1].Sequence)
}

func TestEventRepo_FindByExecutionIDSince_FromZero(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create events
	for i := 0; i < 3; i++ {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
	}

	// Find all events (since 0)
	events, err := repo.FindByExecutionIDSince(context.Background(), execution.ID, 0)
	require.NoError(t, err)
	assert.Len(t, events, 3)
}

// ========== FIND BY TYPE TESTS ==========

func TestEventRepo_FindByType_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution1 := createTestExecution(t, db)
	execution2 := createTestExecution(t, db)

	// Create events of different types across executions
	events := []*models.EventModel{
		{ExecutionID: execution1.ID, EventType: models.EventTypeExecutionStarted, Payload: models.JSONBMap{}},
		{ExecutionID: execution1.ID, EventType: models.EventTypeNodeStarted, Payload: models.JSONBMap{}},
		{ExecutionID: execution2.ID, EventType: models.EventTypeExecutionStarted, Payload: models.JSONBMap{}},
		{ExecutionID: execution2.ID, EventType: models.EventTypeNodeCompleted, Payload: models.JSONBMap{}},
	}

	for _, event := range events {
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
	}

	// Find all execution.started events
	startedEvents, err := repo.FindByType(context.Background(), models.EventTypeExecutionStarted, 10, 0)
	require.NoError(t, err)
	assert.Len(t, startedEvents, 2)

	for _, event := range startedEvents {
		assert.Equal(t, models.EventTypeExecutionStarted, event.EventType)
	}
}

func TestEventRepo_FindByType_Pagination(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create 5 events of the same type
	for i := 0; i < 5; i++ {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{"index": i},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
	}

	// Get first page
	page1, err := repo.FindByType(context.Background(), models.EventTypeNodeStarted, 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	// Get second page
	page2, err := repo.FindByType(context.Background(), models.EventTypeNodeStarted, 2, 2)
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Verify different events
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

// ========== FIND BY TIME RANGE TESTS ==========

func TestEventRepo_FindByTimeRange_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create events with different timestamps
	baseTime := time.Now()
	for i := 0; i < 3; i++ {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{"index": i},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// Find events in time range
	from := baseTime.Add(-1 * time.Hour)
	to := baseTime.Add(1 * time.Hour)

	events, err := repo.FindByTimeRange(context.Background(), from, to, 10, 0)
	require.NoError(t, err)
	assert.Len(t, events, 3)
}

func TestEventRepo_FindByTimeRange_NarrowRange(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create an event
	event := &models.EventModel{
		ExecutionID: execution.ID,
		EventType:   models.EventTypeExecutionStarted,
		Payload:     models.JSONBMap{},
	}
	err := repo.Append(context.Background(), event)
	require.NoError(t, err)

	// Search in past time range (should find nothing)
	from := time.Now().Add(-2 * time.Hour)
	to := time.Now().Add(-1 * time.Hour)

	events, err := repo.FindByTimeRange(context.Background(), from, to, 10, 0)
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

// ========== FIND LATEST TESTS ==========

func TestEventRepo_FindLatestByExecutionID_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create multiple events
	var lastEvent *models.EventModel
	for i := 0; i < 3; i++ {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{"index": i},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
		lastEvent = event
	}

	// Find latest event
	latest, err := repo.FindLatestByExecutionID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Equal(t, lastEvent.ID, latest.ID)
	assert.Equal(t, int64(3), latest.Sequence)
}

func TestEventRepo_FindLatestByExecutionID_NotFound(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	latest, err := repo.FindLatestByExecutionID(context.Background(), execution.ID)
	assert.Error(t, err)
	assert.Nil(t, latest)
}

// ========== COUNT TESTS ==========

func TestEventRepo_Count_Total(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution1 := createTestExecution(t, db)
	execution2 := createTestExecution(t, db)

	// Create events across executions
	for i := 0; i < 3; i++ {
		event1 := &models.EventModel{
			ExecutionID: execution1.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{},
		}
		err := repo.Append(context.Background(), event1)
		require.NoError(t, err)

		event2 := &models.EventModel{
			ExecutionID: execution2.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{},
		}
		err = repo.Append(context.Background(), event2)
		require.NoError(t, err)
	}

	count, err := repo.Count(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 6)
}

func TestEventRepo_CountByExecutionID_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create 5 events
	for i := 0; i < 5; i++ {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
	}

	count, err := repo.CountByExecutionID(context.Background(), execution.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestEventRepo_CountByType_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create events of different types
	eventTypes := []string{
		models.EventTypeNodeStarted,
		models.EventTypeNodeStarted,
		models.EventTypeNodeCompleted,
		models.EventTypeNodeStarted,
	}

	for _, eventType := range eventTypes {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   eventType,
			Payload:     models.JSONBMap{},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
	}

	startedCount, err := repo.CountByType(context.Background(), models.EventTypeNodeStarted)
	require.NoError(t, err)
	assert.Equal(t, 3, startedCount)

	completedCount, err := repo.CountByType(context.Background(), models.EventTypeNodeCompleted)
	require.NoError(t, err)
	assert.Equal(t, 1, completedCount)
}

// ========== STREAM TESTS ==========

func TestEventRepo_Stream_Success(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create initial events
	for i := 0; i < 3; i++ {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{"index": i},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
	}

	// Start streaming from sequence 2
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	eventChan, errChan := repo.Stream(ctx, execution.ID, 2)

	// Collect events
	var receivedEvents []*models.EventModel
	timeout := time.After(1 * time.Second)

collectLoop:
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				break collectLoop
			}
			receivedEvents = append(receivedEvents, event)
			if len(receivedEvents) >= 1 { // Expect at least event with sequence 3
				break collectLoop
			}
		case err := <-errChan:
			require.NoError(t, err)
		case <-timeout:
			break collectLoop
		}
	}

	assert.GreaterOrEqual(t, len(receivedEvents), 1)
	assert.GreaterOrEqual(t, receivedEvents[0].Sequence, int64(3))
}

func TestEventRepo_Stream_FromBeginning(t *testing.T) {
	repo, db, cleanup := setupEventRepoTest(t)
	defer cleanup()

	execution := createTestExecution(t, db)

	// Create events
	for i := 0; i < 2; i++ {
		event := &models.EventModel{
			ExecutionID: execution.ID,
			EventType:   models.EventTypeNodeStarted,
			Payload:     models.JSONBMap{},
		}
		err := repo.Append(context.Background(), event)
		require.NoError(t, err)
	}

	// Stream from beginning (sequence 0)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	eventChan, errChan := repo.Stream(ctx, execution.ID, 0)

	// Collect events
	var receivedEvents []*models.EventModel
	timeout := time.After(500 * time.Millisecond)

collectLoop:
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				break collectLoop
			}
			receivedEvents = append(receivedEvents, event)
			if len(receivedEvents) >= 2 {
				break collectLoop
			}
		case err := <-errChan:
			require.NoError(t, err)
		case <-timeout:
			break collectLoop
		}
	}

	assert.GreaterOrEqual(t, len(receivedEvents), 2)
}

// ========== EVENT TYPE HELPERS TESTS ==========

func TestEventModel_IsWorkflowEvent(t *testing.T) {
	workflowEvent := &models.EventModel{
		EventType: models.EventTypeExecutionStarted,
	}
	assert.True(t, workflowEvent.IsWorkflowEvent())

	nodeEvent := &models.EventModel{
		EventType: models.EventTypeNodeStarted,
	}
	assert.False(t, nodeEvent.IsWorkflowEvent())
}

func TestEventModel_IsNodeEvent(t *testing.T) {
	nodeEvent := &models.EventModel{
		EventType: models.EventTypeNodeStarted,
	}
	assert.True(t, nodeEvent.IsNodeEvent())

	workflowEvent := &models.EventModel{
		EventType: models.EventTypeExecutionStarted,
	}
	assert.False(t, workflowEvent.IsNodeEvent())
}
