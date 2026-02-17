package observer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockEventRepository is a mock implementation of EventRepository
type MockEventRepository struct {
	mock.Mock
}

func (m *MockEventRepository) Append(ctx context.Context, event *models.EventModel) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventRepository) AppendBatch(ctx context.Context, events []*models.EventModel) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockEventRepository) FindByExecutionID(ctx context.Context, executionID uuid.UUID) ([]*models.EventModel, error) {
	args := m.Called(ctx, executionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EventModel), args.Error(1)
}

func (m *MockEventRepository) FindByExecutionIDSince(ctx context.Context, executionID uuid.UUID, sinceSequence int64) ([]*models.EventModel, error) {
	args := m.Called(ctx, executionID, sinceSequence)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EventModel), args.Error(1)
}

func (m *MockEventRepository) FindByType(ctx context.Context, eventType string, limit, offset int) ([]*models.EventModel, error) {
	args := m.Called(ctx, eventType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EventModel), args.Error(1)
}

func (m *MockEventRepository) FindByTimeRange(ctx context.Context, from, to time.Time, limit, offset int) ([]*models.EventModel, error) {
	args := m.Called(ctx, from, to, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EventModel), args.Error(1)
}

func (m *MockEventRepository) FindLatestByExecutionID(ctx context.Context, executionID uuid.UUID) (*models.EventModel, error) {
	args := m.Called(ctx, executionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EventModel), args.Error(1)
}

func (m *MockEventRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockEventRepository) CountByExecutionID(ctx context.Context, executionID uuid.UUID) (int, error) {
	args := m.Called(ctx, executionID)
	return args.Int(0), args.Error(1)
}

func (m *MockEventRepository) CountByType(ctx context.Context, eventType string) (int, error) {
	args := m.Called(ctx, eventType)
	return args.Int(0), args.Error(1)
}

func (m *MockEventRepository) Stream(ctx context.Context, executionID uuid.UUID, fromSequence int64) (<-chan *models.EventModel, <-chan error) {
	args := m.Called(ctx, executionID, fromSequence)
	return args.Get(0).(<-chan *models.EventModel), args.Get(1).(<-chan error)
}

func TestNewDatabaseObserver(t *testing.T) {
	mockRepo := new(MockEventRepository)
	obs := NewDatabaseObserver(mockRepo)

	assert.NotNil(t, obs)
	assert.Equal(t, "database", obs.Name())
	assert.Nil(t, obs.Filter(), "DatabaseObserver should not have a filter (receives all events)")
}

func TestDatabaseObserver_Name(t *testing.T) {
	mockRepo := new(MockEventRepository)
	obs := NewDatabaseObserver(mockRepo)

	assert.Equal(t, "database", obs.Name())
}

func TestDatabaseObserver_Filter(t *testing.T) {
	mockRepo := new(MockEventRepository)
	obs := NewDatabaseObserver(mockRepo)

	assert.Nil(t, obs.Filter(), "DatabaseObserver should receive all events")
}

func TestDatabaseObserver_OnEvent(t *testing.T) {
	t.Run("execution started event", func(t *testing.T) {
		mockRepo := new(MockEventRepository)
		obs := NewDatabaseObserver(mockRepo)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			Status:      "running",
		}

		mockRepo.On("Append", mock.Anything, mock.MatchedBy(func(e *models.EventModel) bool {
			return e.EventType == "execution.started" &&
				e.Payload["workflow_id"] == event.WorkflowID &&
				e.Payload["status"] == "running"
		})).Return(nil)

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("node completed event with all fields", func(t *testing.T) {
		mockRepo := new(MockEventRepository)
		obs := NewDatabaseObserver(mockRepo)

		nodeID := "node-123"
		nodeName := "HTTP Request"
		nodeType := "http"
		durationMs := int64(1500)

		event := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			NodeID:      &nodeID,
			NodeName:    &nodeName,
			NodeType:    &nodeType,
			Status:      "completed",
			Input: map[string]any{
				"url": "https://api.example.com",
			},
			Output: map[string]any{
				"status": 200,
				"data":   "response",
			},
			Variables: map[string]any{
				"user_id": "123",
			},
			DurationMs: &durationMs,
		}

		mockRepo.On("Append", mock.Anything, mock.MatchedBy(func(e *models.EventModel) bool {
			return e.EventType == "node.completed" &&
				e.Payload["node_id"] == "node-123" &&
				e.Payload["node_name"] == "HTTP Request" &&
				e.Payload["node_type"] == "http" &&
				e.Payload["duration_ms"] == int64(1500) &&
				e.Payload["status"] == "completed"
		})).Return(nil)

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("wave started event", func(t *testing.T) {
		mockRepo := new(MockEventRepository)
		obs := NewDatabaseObserver(mockRepo)

		waveIndex := 2
		nodeCount := 5

		event := Event{
			Type:        EventTypeWaveStarted,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			WaveIndex:   &waveIndex,
			NodeCount:   &nodeCount,
			Status:      "running",
		}

		mockRepo.On("Append", mock.Anything, mock.MatchedBy(func(e *models.EventModel) bool {
			return e.EventType == "wave.started" &&
				e.Payload["wave_index"] == 2 &&
				e.Payload["node_count"] == 5
		})).Return(nil)

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("event with error", func(t *testing.T) {
		mockRepo := new(MockEventRepository)
		obs := NewDatabaseObserver(mockRepo)

		testErr := errors.New("execution failed")
		event := Event{
			Type:        EventTypeExecutionFailed,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			Status:      "failed",
			Error:       testErr,
		}

		mockRepo.On("Append", mock.Anything, mock.MatchedBy(func(e *models.EventModel) bool {
			return e.EventType == "execution.failed" &&
				e.Payload["error"] == "execution failed"
		})).Return(nil)

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("event with metadata", func(t *testing.T) {
		mockRepo := new(MockEventRepository)
		obs := NewDatabaseObserver(mockRepo)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			Status:      "running",
			Metadata: map[string]any{
				"trigger_type": "manual",
				"user_id":      "user-123",
			},
		}

		mockRepo.On("Append", mock.Anything, mock.MatchedBy(func(e *models.EventModel) bool {
			metadata, ok := e.Payload["metadata"].(map[string]any)
			return ok &&
				metadata["trigger_type"] == "manual" &&
				metadata["user_id"] == "user-123"
		})).Return(nil)

		err := obs.OnEvent(context.Background(), event)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository append error", func(t *testing.T) {
		mockRepo := new(MockEventRepository)
		obs := NewDatabaseObserver(mockRepo)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			Status:      "running",
		}

		expectedErr := errors.New("database connection error")
		mockRepo.On("Append", mock.Anything, mock.Anything).Return(expectedErr)

		err := obs.OnEvent(context.Background(), event)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestDatabaseObserver_convertToEventModel(t *testing.T) {
	mockRepo := new(MockEventRepository)
	obs := NewDatabaseObserver(mockRepo)

	t.Run("converts execution started event", func(t *testing.T) {
		executionID := uuid.New().String()
		workflowID := uuid.New().String()
		timestamp := time.Now()

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: executionID,
			WorkflowID:  workflowID,
			Timestamp:   timestamp,
			Status:      "running",
		}

		model := obs.convertToEventModel(event)

		assert.NotNil(t, model)
		assert.Equal(t, "execution.started", model.EventType)
		assert.Equal(t, workflowID, model.Payload["workflow_id"])
		assert.Equal(t, "running", model.Payload["status"])
		assert.Equal(t, timestamp.Format(time.RFC3339), model.Payload["timestamp"])
	})

	t.Run("converts node event with all fields", func(t *testing.T) {
		executionID := uuid.New().String()
		nodeID := "node-123"
		nodeName := "Transform"
		nodeType := "transform"
		durationMs := int64(250)

		event := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: executionID,
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			NodeID:      &nodeID,
			NodeName:    &nodeName,
			NodeType:    &nodeType,
			Status:      "completed",
			DurationMs:  &durationMs,
			Input: map[string]any{
				"data": "input",
			},
			Output: map[string]any{
				"result": "output",
			},
		}

		model := obs.convertToEventModel(event)

		assert.Equal(t, "node.completed", model.EventType)
		assert.Equal(t, "node-123", model.Payload["node_id"])
		assert.Equal(t, "Transform", model.Payload["node_name"])
		assert.Equal(t, "transform", model.Payload["node_type"])
		assert.Equal(t, int64(250), model.Payload["duration_ms"])

		input, ok := model.Payload["input"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "input", input["data"])

		output, ok := model.Payload["output"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "output", output["result"])
	})

	t.Run("converts wave event", func(t *testing.T) {
		waveIndex := 3
		nodeCount := 7

		event := Event{
			Type:        EventTypeWaveStarted,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			WaveIndex:   &waveIndex,
			NodeCount:   &nodeCount,
			Status:      "running",
		}

		model := obs.convertToEventModel(event)

		assert.Equal(t, "wave.started", model.EventType)
		assert.Equal(t, 3, model.Payload["wave_index"])
		assert.Equal(t, 7, model.Payload["node_count"])
	})

	t.Run("converts event with error", func(t *testing.T) {
		testErr := errors.New("node execution failed")

		event := Event{
			Type:        EventTypeNodeFailed,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			Status:      "failed",
			Error:       testErr,
		}

		model := obs.convertToEventModel(event)

		assert.Equal(t, "node.failed", model.EventType)
		assert.Equal(t, "node execution failed", model.Payload["error"])
	})

	t.Run("converts event with variables and metadata", func(t *testing.T) {
		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			Status:      "running",
			Variables: map[string]any{
				"env": map[string]any{
					"api_key": "secret",
				},
			},
			Metadata: map[string]any{
				"trigger": "webhook",
			},
		}

		model := obs.convertToEventModel(event)

		variables, ok := model.Payload["variables"].(map[string]any)
		require.True(t, ok)
		env, ok := variables["env"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "secret", env["api_key"])

		metadata, ok := model.Payload["metadata"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "webhook", metadata["trigger"])
	})

	t.Run("handles nil optional fields", func(t *testing.T) {
		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: uuid.New().String(),
			WorkflowID:  uuid.New().String(),
			Timestamp:   time.Now(),
			Status:      "running",
			// All optional fields nil
			NodeID:     nil,
			NodeName:   nil,
			NodeType:   nil,
			WaveIndex:  nil,
			NodeCount:  nil,
			DurationMs: nil,
			Error:      nil,
			Input:      nil,
			Output:     nil,
			Variables:  nil,
			Metadata:   nil,
		}

		model := obs.convertToEventModel(event)

		// Should only have required fields
		assert.Equal(t, "execution.started", model.EventType)
		assert.Contains(t, model.Payload, "workflow_id")
		assert.Contains(t, model.Payload, "status")
		assert.Contains(t, model.Payload, "timestamp")

		// Optional fields should not be present
		assert.NotContains(t, model.Payload, "node_id")
		assert.NotContains(t, model.Payload, "duration_ms")
		assert.NotContains(t, model.Payload, "error")
	})
}
