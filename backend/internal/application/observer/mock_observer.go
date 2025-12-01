package observer

import (
	"context"
	"fmt"
	"sync"
)

// MockObserver is a test observer that records all events
type MockObserver struct {
	name       string
	events     []Event
	callCount  int
	mu         sync.Mutex
	filter     EventFilter
	shouldFail bool
	failError  error
}

// NewMockObserver creates a new mock observer
func NewMockObserver(name string) *MockObserver {
	return &MockObserver{
		name:   name,
		events: make([]Event, 0),
	}
}

// Name returns the observer's name
func (m *MockObserver) Name() string {
	return m.name
}

// Filter returns the event filter
func (m *MockObserver) Filter() EventFilter {
	return m.filter
}

// OnEvent records the event
func (m *MockObserver) OnEvent(ctx context.Context, event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	m.events = append(m.events, event)

	if m.shouldFail {
		if m.failError != nil {
			return m.failError
		}
		return fmt.Errorf("mock observer error")
	}

	return nil
}

// GetEvents returns a copy of all recorded events
func (m *MockObserver) GetEvents() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	eventsCopy := make([]Event, len(m.events))
	copy(eventsCopy, m.events)
	return eventsCopy
}

// GetCallCount returns the number of times OnEvent was called
func (m *MockObserver) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// SetFilter sets the event filter
func (m *MockObserver) SetFilter(filter EventFilter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.filter = filter
}

// SetShouldFail configures failure behavior
func (m *MockObserver) SetShouldFail(fail bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
	m.failError = err
}

// Reset clears all recorded events and resets call count
func (m *MockObserver) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = make([]Event, 0)
	m.callCount = 0
}
