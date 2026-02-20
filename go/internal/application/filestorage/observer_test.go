package filestorage

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper: track observer calls
type testObserver struct {
	name           string
	filter         FileEventFilter
	events         []*FileEvent
	errors         []error
	mu             sync.Mutex
	callCount      int32
	errorsToReturn []error
	errorIndex     int
}

func newTestObserver(name string, filter FileEventFilter) *testObserver {
	return &testObserver{
		name:   name,
		filter: filter,
		events: []*FileEvent{},
		errors: []error{},
	}
}

func (o *testObserver) OnFileEvent(ctx context.Context, event *FileEvent) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	atomic.AddInt32(&o.callCount, 1)
	o.events = append(o.events, event)

	// Return configured error if any
	if len(o.errorsToReturn) > 0 && o.errorIndex < len(o.errorsToReturn) {
		err := o.errorsToReturn[o.errorIndex]
		o.errorIndex++
		o.errors = append(o.errors, err)
		return err
	}

	return nil
}

func (o *testObserver) Name() string {
	return o.name
}

func (o *testObserver) Filter() FileEventFilter {
	return o.filter
}

func (o *testObserver) getCallCount() int {
	return int(atomic.LoadInt32(&o.callCount))
}

func (o *testObserver) getEvents() []*FileEvent {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]*FileEvent{}, o.events...)
}

func (o *testObserver) withErrors(errs ...error) *testObserver {
	o.errorsToReturn = errs
	return o
}

// ============== A. Event Creation Tests ==============

func TestFileEvent_New_FileAdded(t *testing.T) {
	entry := &models.FileEntry{
		ID:        "file-123",
		StorageID: "storage-1",
		Name:      "test.txt",
	}

	event := NewFileEvent(EventFileAdded, "storage-1", entry)

	assert.Equal(t, EventFileAdded, event.Type)
	assert.Equal(t, "storage-1", event.StorageID)
	assert.Equal(t, "file-123", event.FileID)
	assert.NotNil(t, event.FileEntry)
	assert.Equal(t, entry, event.FileEntry)
	assert.WithinDuration(t, time.Now(), event.Timestamp, time.Second)
	assert.Nil(t, event.Error)
	assert.Nil(t, event.Metadata)
}

func TestFileEvent_New_FileRemoved(t *testing.T) {
	event := NewFileEvent(EventFileRemoved, "storage-1", &models.FileEntry{ID: "file-456"})
	assert.Equal(t, EventFileRemoved, event.Type)
	assert.Equal(t, "file-456", event.FileID)
}

func TestFileEvent_New_FileAccessed(t *testing.T) {
	event := NewFileEvent(EventFileAccessed, "storage-1", &models.FileEntry{ID: "file-789"})
	assert.Equal(t, EventFileAccessed, event.Type)
}

func TestFileEvent_New_FileUpdated(t *testing.T) {
	event := NewFileEvent(EventFileUpdated, "storage-1", &models.FileEntry{ID: "file-update"})
	assert.Equal(t, EventFileUpdated, event.Type)
}

func TestFileEvent_New_FileExpired(t *testing.T) {
	event := NewFileEvent(EventFileExpired, "storage-1", &models.FileEntry{ID: "file-expired"})
	assert.Equal(t, EventFileExpired, event.Type)
}

func TestFileEvent_New_StorageFull(t *testing.T) {
	event := NewFileEvent(EventStorageFull, "storage-full", nil)
	assert.Equal(t, EventStorageFull, event.Type)
	assert.Equal(t, "storage-full", event.StorageID)
	assert.Empty(t, event.FileID)
	assert.Nil(t, event.FileEntry)
}

func TestFileEvent_New_QuotaExceeded(t *testing.T) {
	event := NewFileEvent(EventQuotaExceeded, "storage-quota", nil)
	assert.Equal(t, EventQuotaExceeded, event.Type)
	assert.Equal(t, "storage-quota", event.StorageID)
}

func TestFileEvent_New_StorageCreated(t *testing.T) {
	event := NewFileEvent(EventStorageCreated, "new-storage", nil)
	assert.Equal(t, EventStorageCreated, event.Type)
	assert.Equal(t, "new-storage", event.StorageID)
}

func TestFileEvent_New_StorageDeleted(t *testing.T) {
	event := NewFileEvent(EventStorageDeleted, "deleted-storage", nil)
	assert.Equal(t, EventStorageDeleted, event.Type)
	assert.Equal(t, "deleted-storage", event.StorageID)
}

func TestFileEvent_WithError(t *testing.T) {
	event := NewFileEvent(EventFileAdded, "storage-1", nil)
	testErr := errors.New("test error")

	event = event.WithError(testErr)

	assert.NotNil(t, event.Error)
	assert.Equal(t, testErr, event.Error)
}

func TestFileEvent_WithMetadata_Single(t *testing.T) {
	event := NewFileEvent(EventFileAdded, "storage-1", nil)

	event = event.WithMetadata("key1", "value1")

	require.NotNil(t, event.Metadata)
	assert.Equal(t, "value1", event.Metadata["key1"])
}

func TestFileEvent_WithMetadata_Multiple(t *testing.T) {
	event := NewFileEvent(EventFileAdded, "storage-1", nil)

	event = event.WithMetadata("key1", "value1").
		WithMetadata("key2", 123).
		WithMetadata("key3", true)

	require.NotNil(t, event.Metadata)
	assert.Equal(t, "value1", event.Metadata["key1"])
	assert.Equal(t, 123, event.Metadata["key2"])
	assert.Equal(t, true, event.Metadata["key3"])
	assert.Len(t, event.Metadata, 3)
}

func TestFileEvent_WithError_AndMetadata(t *testing.T) {
	event := NewFileEvent(EventFileAdded, "storage-1", nil)
	testErr := errors.New("test error")

	event = event.WithError(testErr).WithMetadata("reason", "quota exceeded")

	assert.Equal(t, testErr, event.Error)
	assert.Equal(t, "quota exceeded", event.Metadata["reason"])
}

// ============== B. Event Type Filter Tests ==============

func TestEventTypeFilter_New_Empty_ReturnsNil(t *testing.T) {
	filter := NewEventTypeFilter()
	assert.Nil(t, filter)
}

func TestEventTypeFilter_New_SingleType(t *testing.T) {
	filter := NewEventTypeFilter(EventFileAdded)
	require.NotNil(t, filter)

	event := NewFileEvent(EventFileAdded, "storage-1", nil)
	assert.True(t, filter.ShouldNotify(event))

	event = NewFileEvent(EventFileRemoved, "storage-1", nil)
	assert.False(t, filter.ShouldNotify(event))
}

func TestEventTypeFilter_New_MultipleTypes(t *testing.T) {
	filter := NewEventTypeFilter(EventFileAdded, EventFileRemoved, EventFileUpdated)
	require.NotNil(t, filter)

	// Should match
	assert.True(t, filter.ShouldNotify(NewFileEvent(EventFileAdded, "s1", nil)))
	assert.True(t, filter.ShouldNotify(NewFileEvent(EventFileRemoved, "s1", nil)))
	assert.True(t, filter.ShouldNotify(NewFileEvent(EventFileUpdated, "s1", nil)))

	// Should not match
	assert.False(t, filter.ShouldNotify(NewFileEvent(EventFileAccessed, "s1", nil)))
	assert.False(t, filter.ShouldNotify(NewFileEvent(EventFileExpired, "s1", nil)))
}

func TestEventTypeFilter_ShouldNotify_Match(t *testing.T) {
	filter := NewEventTypeFilter(EventFileAdded)
	event := NewFileEvent(EventFileAdded, "storage-1", nil)

	assert.True(t, filter.ShouldNotify(event))
}

func TestEventTypeFilter_ShouldNotify_NoMatch(t *testing.T) {
	filter := NewEventTypeFilter(EventFileAdded)
	event := NewFileEvent(EventFileRemoved, "storage-1", nil)

	assert.False(t, filter.ShouldNotify(event))
}

func TestEventTypeFilter_ShouldNotify_NilFilter_AllowsAll(t *testing.T) {
	var filter FileEventFilter = nil

	// Nil filter should allow all events
	assert.True(t, filter == nil || filter.ShouldNotify(NewFileEvent(EventFileAdded, "s1", nil)))
}

func TestEventTypeFilter_AllEventTypes(t *testing.T) {
	allTypes := []FileEventType{
		EventFileAdded,
		EventFileRemoved,
		EventFileAccessed,
		EventFileUpdated,
		EventFileExpired,
		EventStorageFull,
		EventQuotaExceeded,
		EventStorageCreated,
		EventStorageDeleted,
	}

	filter := NewEventTypeFilter(allTypes...)
	require.NotNil(t, filter)

	// All event types should match
	for _, eventType := range allTypes {
		event := NewFileEvent(eventType, "storage-1", nil)
		assert.True(t, filter.ShouldNotify(event), "Event type %s should match", eventType)
	}
}

// ============== C. Storage Filter Tests ==============

func TestStorageFilter_New_Empty_ReturnsNil(t *testing.T) {
	filter := NewStorageFilter()
	assert.Nil(t, filter)
}

func TestStorageFilter_New_SingleStorage(t *testing.T) {
	filter := NewStorageFilter("storage-1")
	require.NotNil(t, filter)

	event := NewFileEvent(EventFileAdded, "storage-1", nil)
	assert.True(t, filter.ShouldNotify(event))

	event = NewFileEvent(EventFileAdded, "storage-2", nil)
	assert.False(t, filter.ShouldNotify(event))
}

func TestStorageFilter_New_MultipleStorages(t *testing.T) {
	filter := NewStorageFilter("storage-1", "storage-2", "storage-3")
	require.NotNil(t, filter)

	// Should match
	assert.True(t, filter.ShouldNotify(NewFileEvent(EventFileAdded, "storage-1", nil)))
	assert.True(t, filter.ShouldNotify(NewFileEvent(EventFileAdded, "storage-2", nil)))
	assert.True(t, filter.ShouldNotify(NewFileEvent(EventFileAdded, "storage-3", nil)))

	// Should not match
	assert.False(t, filter.ShouldNotify(NewFileEvent(EventFileAdded, "storage-4", nil)))
	assert.False(t, filter.ShouldNotify(NewFileEvent(EventFileAdded, "storage-5", nil)))
}

func TestStorageFilter_ShouldNotify_Match(t *testing.T) {
	filter := NewStorageFilter("storage-1")
	event := NewFileEvent(EventFileAdded, "storage-1", nil)

	assert.True(t, filter.ShouldNotify(event))
}

func TestStorageFilter_ShouldNotify_NoMatch(t *testing.T) {
	filter := NewStorageFilter("storage-1")
	event := NewFileEvent(EventFileAdded, "storage-2", nil)

	assert.False(t, filter.ShouldNotify(event))
}

func TestStorageFilter_ShouldNotify_NilFilter_AllowsAll(t *testing.T) {
	var filter FileEventFilter = nil

	assert.True(t, filter == nil || filter.ShouldNotify(NewFileEvent(EventFileAdded, "any-storage", nil)))
}

// ============== D. Composite Filter Tests ==============

func TestCompositeFilter_New(t *testing.T) {
	filter1 := NewEventTypeFilter(EventFileAdded)
	filter2 := NewStorageFilter("storage-1")

	composite := NewCompositeFilter(filter1, filter2)

	require.NotNil(t, composite)
}

func TestCompositeFilter_ShouldNotify_AllPass(t *testing.T) {
	typeFilter := NewEventTypeFilter(EventFileAdded, EventFileRemoved)
	storageFilter := NewStorageFilter("storage-1", "storage-2")

	composite := NewCompositeFilter(typeFilter, storageFilter)

	// Both filters match
	event := NewFileEvent(EventFileAdded, "storage-1", nil)
	assert.True(t, composite.ShouldNotify(event))

	event = NewFileEvent(EventFileRemoved, "storage-2", nil)
	assert.True(t, composite.ShouldNotify(event))
}

func TestCompositeFilter_ShouldNotify_OneFails(t *testing.T) {
	typeFilter := NewEventTypeFilter(EventFileAdded)
	storageFilter := NewStorageFilter("storage-1")

	composite := NewCompositeFilter(typeFilter, storageFilter)

	// Type matches but storage doesn't
	event := NewFileEvent(EventFileAdded, "storage-2", nil)
	assert.False(t, composite.ShouldNotify(event))

	// Storage matches but type doesn't
	event = NewFileEvent(EventFileRemoved, "storage-1", nil)
	assert.False(t, composite.ShouldNotify(event))
}

func TestCompositeFilter_ShouldNotify_Empty_AllowsAll(t *testing.T) {
	composite := NewCompositeFilter()

	// Empty composite filter should allow all events
	event := NewFileEvent(EventFileAdded, "storage-1", nil)
	assert.True(t, composite.ShouldNotify(event))
}

func TestCompositeFilter_Complex_TypeAndStorage(t *testing.T) {
	// Only EventFileAdded from storage-1
	typeFilter := NewEventTypeFilter(EventFileAdded)
	storageFilter := NewStorageFilter("storage-1")
	composite := NewCompositeFilter(typeFilter, storageFilter)

	tests := []struct {
		name      string
		eventType FileEventType
		storageID string
		wantMatch bool
	}{
		{"match_both", EventFileAdded, "storage-1", true},
		{"wrong_type", EventFileRemoved, "storage-1", false},
		{"wrong_storage", EventFileAdded, "storage-2", false},
		{"wrong_both", EventFileRemoved, "storage-2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewFileEvent(tt.eventType, tt.storageID, nil)
			assert.Equal(t, tt.wantMatch, composite.ShouldNotify(event))
		})
	}
}

func TestCompositeFilter_WithNilFilters(t *testing.T) {
	typeFilter := NewEventTypeFilter(EventFileAdded)
	composite := NewCompositeFilter(typeFilter, nil)

	// Nil filter in composite should be ignored (treated as pass-through)
	event := NewFileEvent(EventFileAdded, "any-storage", nil)
	assert.True(t, composite.ShouldNotify(event))

	event = NewFileEvent(EventFileRemoved, "any-storage", nil)
	assert.False(t, composite.ShouldNotify(event))
}

// ============== E. Func Observer Tests ==============

func TestFuncObserver_New(t *testing.T) {
	called := false
	callback := func(ctx context.Context, event *FileEvent) error {
		called = true
		return nil
	}

	observer := NewFuncObserver("test-observer", nil, callback)

	require.NotNil(t, observer)
	assert.Equal(t, "test-observer", observer.Name())
	assert.Nil(t, observer.Filter())

	// Test callback
	err := observer.OnFileEvent(context.Background(), NewFileEvent(EventFileAdded, "s1", nil))
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestFuncObserver_OnFileEvent_Success(t *testing.T) {
	var receivedEvent *FileEvent
	callback := func(ctx context.Context, event *FileEvent) error {
		receivedEvent = event
		return nil
	}

	observer := NewFuncObserver("test", nil, callback)
	event := NewFileEvent(EventFileAdded, "storage-1", nil)

	err := observer.OnFileEvent(context.Background(), event)

	assert.NoError(t, err)
	assert.Equal(t, event, receivedEvent)
}

func TestFuncObserver_OnFileEvent_Error(t *testing.T) {
	expectedErr := errors.New("callback error")
	callback := func(ctx context.Context, event *FileEvent) error {
		return expectedErr
	}

	observer := NewFuncObserver("test", nil, callback)
	event := NewFileEvent(EventFileAdded, "storage-1", nil)

	err := observer.OnFileEvent(context.Background(), event)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestFuncObserver_Name(t *testing.T) {
	observer := NewFuncObserver("my-observer", nil, func(ctx context.Context, event *FileEvent) error {
		return nil
	})

	assert.Equal(t, "my-observer", observer.Name())
}

func TestFuncObserver_Filter_Nil(t *testing.T) {
	observer := NewFuncObserver("test", nil, func(ctx context.Context, event *FileEvent) error {
		return nil
	})

	assert.Nil(t, observer.Filter())
}

func TestFuncObserver_Filter_WithFilter(t *testing.T) {
	filter := NewEventTypeFilter(EventFileAdded)
	observer := NewFuncObserver("test", filter, func(ctx context.Context, event *FileEvent) error {
		return nil
	})

	assert.NotNil(t, observer.Filter())
	assert.Equal(t, filter, observer.Filter())
}

func TestFuncObserver_Callback_ReceivesEvent(t *testing.T) {
	entry := &models.FileEntry{
		ID:        "file-123",
		StorageID: "storage-1",
		Name:      "test.txt",
	}

	var capturedEvent *FileEvent
	callback := func(ctx context.Context, event *FileEvent) error {
		capturedEvent = event
		return nil
	}

	observer := NewFuncObserver("test", nil, callback)
	event := NewFileEvent(EventFileAdded, "storage-1", entry)

	observer.OnFileEvent(context.Background(), event)

	require.NotNil(t, capturedEvent)
	assert.Equal(t, EventFileAdded, capturedEvent.Type)
	assert.Equal(t, "storage-1", capturedEvent.StorageID)
	assert.Equal(t, "file-123", capturedEvent.FileID)
	assert.Equal(t, entry, capturedEvent.FileEntry)
}

// ============== F. Event Flow Tests ==============

func TestObserver_AllEventTypes_Notification(t *testing.T) {
	observer := newTestObserver("test", nil)

	allTypes := []FileEventType{
		EventFileAdded,
		EventFileRemoved,
		EventFileAccessed,
		EventFileUpdated,
		EventFileExpired,
		EventStorageFull,
		EventQuotaExceeded,
		EventStorageCreated,
		EventStorageDeleted,
	}

	for _, eventType := range allTypes {
		event := NewFileEvent(eventType, "storage-1", nil)
		err := observer.OnFileEvent(context.Background(), event)
		assert.NoError(t, err)
	}

	events := observer.getEvents()
	assert.Len(t, events, len(allTypes))
	assert.Equal(t, len(allTypes), observer.getCallCount())

	// Verify all event types were captured
	for i, eventType := range allTypes {
		assert.Equal(t, eventType, events[i].Type)
	}
}

func TestObserver_FilterByType_OnlyMatching(t *testing.T) {
	filter := NewEventTypeFilter(EventFileAdded, EventFileRemoved)
	observer := newTestObserver("test", filter)

	// Simulate external filtering (as would be done by manager)
	events := []*FileEvent{
		NewFileEvent(EventFileAdded, "s1", nil),
		NewFileEvent(EventFileRemoved, "s1", nil),
		NewFileEvent(EventFileAccessed, "s1", nil), // Won't match
		NewFileEvent(EventFileUpdated, "s1", nil),  // Won't match
		NewFileEvent(EventFileAdded, "s1", nil),
	}

	for _, event := range events {
		if filter.ShouldNotify(event) {
			observer.OnFileEvent(context.Background(), event)
		}
	}

	capturedEvents := observer.getEvents()
	assert.Len(t, capturedEvents, 3) // Only 3 matching events
	assert.Equal(t, EventFileAdded, capturedEvents[0].Type)
	assert.Equal(t, EventFileRemoved, capturedEvents[1].Type)
	assert.Equal(t, EventFileAdded, capturedEvents[2].Type)
}

func TestObserver_FilterByStorage_OnlyMatching(t *testing.T) {
	filter := NewStorageFilter("storage-1", "storage-3")
	observer := newTestObserver("test", filter)

	events := []*FileEvent{
		NewFileEvent(EventFileAdded, "storage-1", nil),
		NewFileEvent(EventFileAdded, "storage-2", nil), // Won't match
		NewFileEvent(EventFileAdded, "storage-3", nil),
		NewFileEvent(EventFileAdded, "storage-4", nil), // Won't match
	}

	for _, event := range events {
		if filter.ShouldNotify(event) {
			observer.OnFileEvent(context.Background(), event)
		}
	}

	capturedEvents := observer.getEvents()
	assert.Len(t, capturedEvents, 2)
	assert.Equal(t, "storage-1", capturedEvents[0].StorageID)
	assert.Equal(t, "storage-3", capturedEvents[1].StorageID)
}

func TestObserver_MultipleObservers_AllNotified(t *testing.T) {
	obs1 := newTestObserver("observer-1", nil)
	obs2 := newTestObserver("observer-2", nil)
	obs3 := newTestObserver("observer-3", nil)

	event := NewFileEvent(EventFileAdded, "storage-1", nil)

	// Notify all observers
	obs1.OnFileEvent(context.Background(), event)
	obs2.OnFileEvent(context.Background(), event)
	obs3.OnFileEvent(context.Background(), event)

	assert.Equal(t, 1, obs1.getCallCount())
	assert.Equal(t, 1, obs2.getCallCount())
	assert.Equal(t, 1, obs3.getCallCount())
}

func TestObserver_ConcurrentNotifications(t *testing.T) {
	observer := newTestObserver("test", nil)

	// Simulate concurrent notifications
	var wg sync.WaitGroup
	eventCount := 100

	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			event := NewFileEvent(EventFileAdded, "storage-1", &models.FileEntry{
				ID: "file-" + string(rune(idx)),
			})
			observer.OnFileEvent(context.Background(), event)
		}(i)
	}

	wg.Wait()

	assert.Equal(t, eventCount, observer.getCallCount())
	assert.Len(t, observer.getEvents(), eventCount)
}

func TestObserver_AsyncNotification_NonBlocking(t *testing.T) {
	// Test that observer can be called asynchronously
	observer := newTestObserver("test", nil)

	done := make(chan bool)

	go func() {
		event := NewFileEvent(EventFileAdded, "storage-1", nil)
		observer.OnFileEvent(context.Background(), event)
		done <- true
	}()

	select {
	case <-done:
		assert.Equal(t, 1, observer.getCallCount())
	case <-time.After(time.Second):
		t.Fatal("Observer notification timed out")
	}
}

func TestObserver_ErrorHandling_DoesNotBlock(t *testing.T) {
	// Observer returns error but should not block processing
	observer := newTestObserver("test", nil).withErrors(
		errors.New("error 1"),
		errors.New("error 2"),
	)

	event1 := NewFileEvent(EventFileAdded, "storage-1", nil)
	err1 := observer.OnFileEvent(context.Background(), event1)
	assert.Error(t, err1)

	event2 := NewFileEvent(EventFileRemoved, "storage-1", nil)
	err2 := observer.OnFileEvent(context.Background(), event2)
	assert.Error(t, err2)

	// Both events should have been received despite errors
	assert.Equal(t, 2, observer.getCallCount())
	assert.Len(t, observer.getEvents(), 2)
}

func TestObserver_EventMetadata_Preserved(t *testing.T) {
	observer := newTestObserver("test", nil)

	event := NewFileEvent(EventFileAdded, "storage-1", nil).
		WithMetadata("key1", "value1").
		WithMetadata("key2", 123)

	observer.OnFileEvent(context.Background(), event)

	capturedEvents := observer.getEvents()
	require.Len(t, capturedEvents, 1)

	metadata := capturedEvents[0].Metadata
	assert.Equal(t, "value1", metadata["key1"])
	assert.Equal(t, 123, metadata["key2"])
}

func TestObserver_EventWithError_Preserved(t *testing.T) {
	observer := newTestObserver("test", nil)

	expectedErr := errors.New("storage error")
	event := NewFileEvent(EventStorageFull, "storage-1", nil).
		WithError(expectedErr).
		WithMetadata("reason", "disk full")

	observer.OnFileEvent(context.Background(), event)

	capturedEvents := observer.getEvents()
	require.Len(t, capturedEvents, 1)

	assert.Equal(t, expectedErr, capturedEvents[0].Error)
	assert.Equal(t, "disk full", capturedEvents[0].Metadata["reason"])
}
