package observer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewObserverManager(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		mgr := NewObserverManager()

		assert.NotNil(t, mgr)
		assert.Equal(t, 0, mgr.Count())
		assert.Equal(t, 100, mgr.bufferSize)
		assert.Nil(t, mgr.logger)
	})

	t.Run("with logger option", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})
		mgr := NewObserverManager(WithLogger(log))

		assert.NotNil(t, mgr)
		assert.NotNil(t, mgr.logger)
	})

	t.Run("with buffer size option", func(t *testing.T) {
		mgr := NewObserverManager(WithBufferSize(500))

		assert.NotNil(t, mgr)
		assert.Equal(t, 500, mgr.bufferSize)
	})

	t.Run("with multiple options", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})
		mgr := NewObserverManager(
			WithLogger(log),
			WithBufferSize(250),
		)

		assert.NotNil(t, mgr)
		assert.NotNil(t, mgr.logger)
		assert.Equal(t, 250, mgr.bufferSize)
	})
}

func TestObserverManager_Register(t *testing.T) {
	t.Run("register single observer", func(t *testing.T) {
		mgr := NewObserverManager()
		obs := NewMockObserver("test-observer")

		err := mgr.Register(obs)
		assert.NoError(t, err)
		assert.Equal(t, 1, mgr.Count())
	})

	t.Run("register multiple observers", func(t *testing.T) {
		mgr := NewObserverManager()
		obs1 := NewMockObserver("observer-1")
		obs2 := NewMockObserver("observer-2")
		obs3 := NewMockObserver("observer-3")

		err := mgr.Register(obs1)
		assert.NoError(t, err)
		err = mgr.Register(obs2)
		assert.NoError(t, err)
		err = mgr.Register(obs3)
		assert.NoError(t, err)

		assert.Equal(t, 3, mgr.Count())
	})

	t.Run("register duplicate name fails", func(t *testing.T) {
		mgr := NewObserverManager()
		obs1 := NewMockObserver("duplicate")
		obs2 := NewMockObserver("duplicate")

		err := mgr.Register(obs1)
		assert.NoError(t, err)

		err = mgr.Register(obs2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
		assert.Equal(t, 1, mgr.Count())
	})

	t.Run("thread-safe registration", func(t *testing.T) {
		mgr := NewObserverManager()
		var wg sync.WaitGroup

		// Concurrently register 10 observers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				obs := NewMockObserver(string(rune('a' + id)))
				mgr.Register(obs)
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 10, mgr.Count())
	})
}

func TestObserverManager_Unregister(t *testing.T) {
	t.Run("unregister existing observer", func(t *testing.T) {
		mgr := NewObserverManager()
		obs := NewMockObserver("test-observer")

		err := mgr.Register(obs)
		require.NoError(t, err)

		err = mgr.Unregister("test-observer")
		assert.NoError(t, err)
		assert.Equal(t, 0, mgr.Count())
	})

	t.Run("unregister non-existent observer", func(t *testing.T) {
		mgr := NewObserverManager()

		err := mgr.Unregister("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("unregister from multiple observers", func(t *testing.T) {
		mgr := NewObserverManager()
		obs1 := NewMockObserver("observer-1")
		obs2 := NewMockObserver("observer-2")
		obs3 := NewMockObserver("observer-3")

		mgr.Register(obs1)
		mgr.Register(obs2)
		mgr.Register(obs3)

		err := mgr.Unregister("observer-2")
		assert.NoError(t, err)
		assert.Equal(t, 2, mgr.Count())
	})

	t.Run("thread-safe unregistration", func(t *testing.T) {
		mgr := NewObserverManager()

		// Register 10 observers
		for i := 0; i < 10; i++ {
			obs := NewMockObserver(string(rune('a' + i)))
			mgr.Register(obs)
		}

		var wg sync.WaitGroup

		// Concurrently unregister 5 observers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				mgr.Unregister(string(rune('a' + id)))
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 5, mgr.Count())
	})
}

func TestObserverManager_Notify(t *testing.T) {
	t.Run("notify single observer", func(t *testing.T) {
		mgr := NewObserverManager()
		obs := NewMockObserver("test-observer")
		mgr.Register(obs)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "running",
		}

		mgr.Notify(context.Background(), event)

		// Give goroutine time to process
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 1, obs.GetCallCount())
		events := obs.GetEvents()
		require.Len(t, events, 1)
		assert.Equal(t, EventTypeExecutionStarted, events[0].Type)
	})

	t.Run("notify multiple observers", func(t *testing.T) {
		mgr := NewObserverManager()
		obs1 := NewMockObserver("observer-1")
		obs2 := NewMockObserver("observer-2")
		obs3 := NewMockObserver("observer-3")

		mgr.Register(obs1)
		mgr.Register(obs2)
		mgr.Register(obs3)

		event := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
			Status:      "completed",
		}

		mgr.Notify(context.Background(), event)

		// Give goroutines time to process
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 1, obs1.GetCallCount())
		assert.Equal(t, 1, obs2.GetCallCount())
		assert.Equal(t, 1, obs3.GetCallCount())
	})

	t.Run("non-blocking notification", func(t *testing.T) {
		mgr := NewObserverManager()

		// Create a slow observer that blocks
		slowObs := &SlowObserver{
			name:  "slow-observer",
			delay: 100 * time.Millisecond,
		}
		mgr.Register(slowObs)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		// Notify should return immediately, not wait for slow observer
		start := time.Now()
		mgr.Notify(context.Background(), event)
		duration := time.Since(start)

		// Notify should return in < 10ms (not wait for 100ms observer)
		assert.Less(t, duration, 10*time.Millisecond, "Notify should be non-blocking")
	})

	t.Run("observer error does not propagate", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})
		mgr := NewObserverManager(WithLogger(log))

		failingObs := NewMockObserver("failing-observer")
		failingObs.SetShouldFail(true, errors.New("observer error"))

		successObs := NewMockObserver("success-observer")

		mgr.Register(failingObs)
		mgr.Register(successObs)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		// Should not panic
		mgr.Notify(context.Background(), event)

		time.Sleep(10 * time.Millisecond)

		// Both observers should have been called
		assert.Equal(t, 1, failingObs.GetCallCount())
		assert.Equal(t, 1, successObs.GetCallCount())
	})

	t.Run("panic recovery", func(t *testing.T) {
		log := logger.New(config.LoggingConfig{Level: "debug", Format: "text"})
		mgr := NewObserverManager(WithLogger(log))

		panicObs := &PanicObserver{name: "panic-observer"}
		successObs := NewMockObserver("success-observer")

		mgr.Register(panicObs)
		mgr.Register(successObs)

		event := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		// Should not panic
		assert.NotPanics(t, func() {
			mgr.Notify(context.Background(), event)
			time.Sleep(10 * time.Millisecond)
		})

		// Success observer should still have been called
		assert.Equal(t, 1, successObs.GetCallCount())
	})

	t.Run("event filtering", func(t *testing.T) {
		mgr := NewObserverManager()

		// Observer that only wants execution events
		execObs := NewMockObserver("exec-observer")
		execObs.SetFilter(NewEventTypeFilter(
			EventTypeExecutionStarted,
			EventTypeExecutionCompleted,
			EventTypeExecutionFailed,
		))

		// Observer that wants all events
		allObs := NewMockObserver("all-observer")

		mgr.Register(execObs)
		mgr.Register(allObs)

		// Send node event (should be filtered for execObs)
		nodeEvent := Event{
			Type:        EventTypeNodeCompleted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		mgr.Notify(context.Background(), nodeEvent)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 0, execObs.GetCallCount(), "Filtered observer should not receive node events")
		assert.Equal(t, 1, allObs.GetCallCount(), "Unfiltered observer should receive all events")

		// Send execution event (should reach both)
		execEvent := Event{
			Type:        EventTypeExecutionStarted,
			ExecutionID: "exec-123",
			WorkflowID:  "wf-456",
			Timestamp:   time.Now(),
		}

		mgr.Notify(context.Background(), execEvent)
		time.Sleep(10 * time.Millisecond)

		assert.Equal(t, 1, execObs.GetCallCount(), "Filtered observer should receive execution events")
		assert.Equal(t, 2, allObs.GetCallCount(), "Unfiltered observer should receive all events")
	})

	t.Run("concurrent notifications", func(t *testing.T) {
		mgr := NewObserverManager()
		obs := NewMockObserver("test-observer")
		mgr.Register(obs)

		var wg sync.WaitGroup
		numNotifications := 100

		// Send 100 notifications concurrently
		for i := 0; i < numNotifications; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				event := Event{
					Type:        EventTypeExecutionStarted,
					ExecutionID: "exec-123",
					WorkflowID:  "wf-456",
					Timestamp:   time.Now(),
				}
				mgr.Notify(context.Background(), event)
			}(i)
		}

		wg.Wait()
		time.Sleep(50 * time.Millisecond)

		// All notifications should have been received
		assert.Equal(t, numNotifications, obs.GetCallCount())
	})
}

func TestObserverManager_Count(t *testing.T) {
	mgr := NewObserverManager()

	assert.Equal(t, 0, mgr.Count())

	obs1 := NewMockObserver("observer-1")
	mgr.Register(obs1)
	assert.Equal(t, 1, mgr.Count())

	obs2 := NewMockObserver("observer-2")
	mgr.Register(obs2)
	assert.Equal(t, 2, mgr.Count())

	mgr.Unregister("observer-1")
	assert.Equal(t, 1, mgr.Count())

	mgr.Unregister("observer-2")
	assert.Equal(t, 0, mgr.Count())
}

// Test helper: SlowObserver simulates a slow observer
type SlowObserver struct {
	name  string
	delay time.Duration
	calls int32
}

func (s *SlowObserver) Name() string {
	return s.name
}

func (s *SlowObserver) Filter() EventFilter {
	return nil
}

func (s *SlowObserver) OnEvent(ctx context.Context, event Event) error {
	atomic.AddInt32(&s.calls, 1)
	time.Sleep(s.delay)
	return nil
}

// Test helper: PanicObserver simulates an observer that panics
type PanicObserver struct {
	name string
}

func (p *PanicObserver) Name() string {
	return p.name
}

func (p *PanicObserver) Filter() EventFilter {
	return nil
}

func (p *PanicObserver) OnEvent(ctx context.Context, event Event) error {
	panic("intentional panic for testing")
}
