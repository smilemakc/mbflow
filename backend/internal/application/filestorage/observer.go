package filestorage

import (
	"context"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
)

// FileEventType represents the type of file storage event
type FileEventType string

const (
	// EventFileAdded is emitted when a file is added to storage
	EventFileAdded FileEventType = "file.added"
	// EventFileRemoved is emitted when a file is removed from storage
	EventFileRemoved FileEventType = "file.removed"
	// EventFileAccessed is emitted when a file is accessed (read)
	EventFileAccessed FileEventType = "file.accessed"
	// EventFileUpdated is emitted when file metadata is updated
	EventFileUpdated FileEventType = "file.updated"
	// EventFileExpired is emitted when a file expires (TTL)
	EventFileExpired FileEventType = "file.expired"
	// EventStorageFull is emitted when storage quota is reached
	EventStorageFull FileEventType = "storage.full"
	// EventQuotaExceeded is emitted when trying to store beyond quota
	EventQuotaExceeded FileEventType = "storage.quota_exceeded"
	// EventStorageCreated is emitted when a new storage is created
	EventStorageCreated FileEventType = "storage.created"
	// EventStorageDeleted is emitted when a storage is deleted
	EventStorageDeleted FileEventType = "storage.deleted"
)

// FileEvent represents an event in the file storage system
type FileEvent struct {
	Type      FileEventType     `json:"type"`
	FileID    string            `json:"file_id,omitempty"`
	StorageID string            `json:"storage_id"`
	FileEntry *models.FileEntry `json:"file_entry,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Error     error             `json:"error,omitempty"`
	Metadata  map[string]any    `json:"metadata,omitempty"`
}

// NewFileEvent creates a new file event
func NewFileEvent(eventType FileEventType, storageID string, entry *models.FileEntry) *FileEvent {
	event := &FileEvent{
		Type:      eventType,
		StorageID: storageID,
		Timestamp: time.Now(),
	}
	if entry != nil {
		event.FileID = entry.ID
		event.FileEntry = entry
	}
	return event
}

// WithError adds an error to the event
func (e *FileEvent) WithError(err error) *FileEvent {
	e.Error = err
	return e
}

// WithMetadata adds metadata to the event
func (e *FileEvent) WithMetadata(key string, value any) *FileEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// FileObserver observes file storage events
type FileObserver interface {
	// OnFileEvent is called when a file event occurs
	OnFileEvent(ctx context.Context, event *FileEvent) error

	// Name returns the observer's unique identifier
	Name() string

	// Filter returns the event filter for this observer (nil = all events)
	Filter() FileEventFilter
}

// FileEventFilter filters file events
type FileEventFilter interface {
	// ShouldNotify returns true if the event should be sent to the observer
	ShouldNotify(event *FileEvent) bool
}

// EventTypeFilter filters events by type
type EventTypeFilter struct {
	allowedTypes map[FileEventType]bool
}

// NewEventTypeFilter creates a filter for specific event types
func NewEventTypeFilter(types ...FileEventType) FileEventFilter {
	if len(types) == 0 {
		return nil // nil filter = all events
	}

	filter := &EventTypeFilter{
		allowedTypes: make(map[FileEventType]bool),
	}
	for _, t := range types {
		filter.allowedTypes[t] = true
	}
	return filter
}

// ShouldNotify checks if the event should trigger notification
func (f *EventTypeFilter) ShouldNotify(event *FileEvent) bool {
	if f == nil || len(f.allowedTypes) == 0 {
		return true // No filter = all events
	}
	return f.allowedTypes[event.Type]
}

// StorageFilter filters events by storage ID
type StorageFilter struct {
	storageIDs map[string]bool
}

// NewStorageFilter creates a filter for specific storage IDs
func NewStorageFilter(storageIDs ...string) FileEventFilter {
	if len(storageIDs) == 0 {
		return nil
	}

	filter := &StorageFilter{
		storageIDs: make(map[string]bool),
	}
	for _, id := range storageIDs {
		filter.storageIDs[id] = true
	}
	return filter
}

// ShouldNotify checks if the event should trigger notification
func (f *StorageFilter) ShouldNotify(event *FileEvent) bool {
	if f == nil || len(f.storageIDs) == 0 {
		return true
	}
	return f.storageIDs[event.StorageID]
}

// CompositeFilter combines multiple filters (AND logic)
type CompositeFilter struct {
	filters []FileEventFilter
}

// NewCompositeFilter creates a composite filter
func NewCompositeFilter(filters ...FileEventFilter) FileEventFilter {
	return &CompositeFilter{filters: filters}
}

// ShouldNotify returns true only if all filters pass
func (f *CompositeFilter) ShouldNotify(event *FileEvent) bool {
	for _, filter := range f.filters {
		if filter != nil && !filter.ShouldNotify(event) {
			return false
		}
	}
	return true
}

// FuncObserver is a function-based observer implementation
type FuncObserver struct {
	name     string
	filter   FileEventFilter
	callback func(ctx context.Context, event *FileEvent) error
}

// NewFuncObserver creates a new function-based observer
func NewFuncObserver(name string, filter FileEventFilter, callback func(ctx context.Context, event *FileEvent) error) FileObserver {
	return &FuncObserver{
		name:     name,
		filter:   filter,
		callback: callback,
	}
}

// OnFileEvent calls the callback function
func (o *FuncObserver) OnFileEvent(ctx context.Context, event *FileEvent) error {
	return o.callback(ctx, event)
}

// Name returns the observer name
func (o *FuncObserver) Name() string {
	return o.name
}

// Filter returns the event filter
func (o *FuncObserver) Filter() FileEventFilter {
	return o.filter
}
