package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// EventListener manages event-based triggers via Redis pub/sub
type EventListener struct {
	triggerRepo  repository.TriggerRepository
	workflowRepo repository.WorkflowRepository
	executionMgr WorkflowExecutor
	cache        *cache.RedisCache

	pubsub      *redis.PubSub
	triggers    map[string][]*models.Trigger // eventType -> triggers
	mu          sync.RWMutex
	ctx         context.Context // context passed to Start
	started     bool            // Start was called
	stopped     bool            // Stop was called
	isRunning   bool            // listen goroutine is running
	stopChan    chan struct{}
	stoppedChan chan struct{}
}

// EventListenerConfig holds configuration for event listener
type EventListenerConfig struct {
	TriggerRepo  repository.TriggerRepository
	WorkflowRepo repository.WorkflowRepository
	ExecutionMgr WorkflowExecutor
	Cache        *cache.RedisCache
}

// NewEventListener creates a new event listener
func NewEventListener(cfg EventListenerConfig) (*EventListener, error) {
	return &EventListener{
		triggerRepo:  cfg.TriggerRepo,
		workflowRepo: cfg.WorkflowRepo,
		executionMgr: cfg.ExecutionMgr,
		cache:        cfg.Cache,
		triggers:     make(map[string][]*models.Trigger),
		stopChan:     make(chan struct{}),
		stoppedChan:  make(chan struct{}),
	}, nil
}

// Start starts the event listener with initial triggers
func (el *EventListener) Start(ctx context.Context, triggers []*storagemodels.TriggerModel) error {
	el.mu.Lock()
	defer el.mu.Unlock()

	// Group triggers by event type
	for _, trigger := range triggers {
		if trigger.Type == string(models.TriggerTypeEvent) {
			domainTrigger := el.modelToDomain(trigger)
			if err := el.addTriggerLocked(ctx, domainTrigger); err != nil {
				fmt.Printf("failed to add event trigger %s: %v\n", trigger.ID, err)
				continue
			}
		}
	}

	el.ctx = ctx
	el.started = true

	// Subscribe to event channels. Without event triggers the subscription
	// is created lazily by the first AddTrigger.
	if len(el.triggers) > 0 {
		el.subscribeLocked(ctx, el.getChannels()...)
	}

	return nil
}

// subscribeLocked opens the pub/sub connection on the given channels and
// starts the listen goroutine (must hold lock).
func (el *EventListener) subscribeLocked(ctx context.Context, channels ...string) {
	el.pubsub = el.cache.Client().Subscribe(ctx, channels...)
	el.isRunning = true
	go el.listen(ctx, el.pubsub.Channel())
}

// Stop stops the event listener
func (el *EventListener) Stop() error {
	el.mu.Lock()
	if el.stopped {
		el.mu.Unlock()
		return nil
	}
	el.stopped = true
	isRunning := el.isRunning
	pubsub := el.pubsub
	el.mu.Unlock()

	// Only close stopChan if listener is running
	if isRunning {
		close(el.stopChan)
	}

	// Close pub/sub connection
	var closeErr error
	if pubsub != nil {
		if err := pubsub.Close(); err != nil {
			closeErr = fmt.Errorf("failed to close pub/sub: %w", err)
		}
	}

	// Wait for listener to stop (only if it was started)
	if isRunning {
		<-el.stoppedChan
	}

	return closeErr
}

// AddTrigger adds a new event trigger
func (el *EventListener) AddTrigger(ctx context.Context, trigger *models.Trigger) error {
	el.mu.Lock()
	defer el.mu.Unlock()

	return el.addTriggerLocked(ctx, trigger)
}

// addTriggerLocked adds a trigger (must hold lock)
func (el *EventListener) addTriggerLocked(ctx context.Context, trigger *models.Trigger) error {
	if trigger.Type != models.TriggerTypeEvent {
		return nil // Not an event trigger
	}

	eventType, ok := trigger.Config["event_type"].(string)
	if !ok || eventType == "" {
		return fmt.Errorf("event_type not found in trigger config")
	}

	el.triggers[eventType] = append(el.triggers[eventType], trigger)

	channel := el.getEventChannel(eventType)
	switch {
	case el.pubsub != nil:
		// Listener is already running: subscribe to the new channel.
		if err := el.pubsub.Subscribe(ctx, channel); err != nil {
			return fmt.Errorf("failed to subscribe to channel %s: %w", channel, err)
		}
	case el.started && !el.stopped:
		// Started without event triggers: this is the first one, so open
		// the subscription now. Use the Start context so the listener lives
		// as long as the manager, not as long as this (request) context.
		el.subscribeLocked(el.ctx, channel)
	}

	return nil
}

// RemoveTrigger removes an event trigger
func (el *EventListener) RemoveTrigger(ctx context.Context, triggerID string) error {
	el.mu.Lock()
	defer el.mu.Unlock()

	// Find and remove trigger
	for eventType, triggers := range el.triggers {
		for i, trigger := range triggers {
			if trigger.ID == triggerID {
				// Remove trigger from list
				el.triggers[eventType] = append(triggers[:i], triggers[i+1:]...)

				// If no more triggers for this event type, unsubscribe
				if len(el.triggers[eventType]) == 0 {
					delete(el.triggers, eventType)
					if el.pubsub != nil {
						channel := el.getEventChannel(eventType)
						if err := el.pubsub.Unsubscribe(ctx, channel); err != nil {
							fmt.Printf("failed to unsubscribe from channel %s: %v\n", channel, err)
						}
					}
				}

				return nil
			}
		}
	}

	return nil
}

// listen listens for events from Redis pub/sub
func (el *EventListener) listen(ctx context.Context, ch <-chan *redis.Message) {
	defer close(el.stoppedChan)

	for {
		select {
		case <-el.stopChan:
			return
		case msg, ok := <-ch:
			if !ok {
				// Pub/sub connection closed.
				return
			}
			if msg == nil {
				continue
			}

			// Parse event
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				fmt.Printf("failed to parse event: %v\n", err)
				continue
			}

			// Handle event
			el.handleEvent(ctx, event, eventDedupID(event, msg.Payload))
		}
	}
}

// handleEvent handles an incoming event. dedupID identifies the event across
// replicas (see eventDedupID); every replica subscribed to the channel
// receives the event, and only the one that claims (trigger, event) fires.
func (el *EventListener) handleEvent(ctx context.Context, event Event, dedupID string) {
	el.mu.RLock()
	triggers := append([]*models.Trigger(nil), el.triggers[event.Type]...)
	el.mu.RUnlock()

	for _, trigger := range triggers {
		// Check if event matches trigger filter
		if !el.matchesFilter(event, trigger) {
			continue
		}

		// Execute workflow in background
		go el.fire(trigger, event, dedupID)
	}
}

// fire executes the trigger for an event if this replica wins the claim for
// the (trigger, event) pair. It reports whether the trigger was fired.
func (el *EventListener) fire(t *models.Trigger, event Event, dedupID string) bool {
	execCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if !claimOccurrence(execCtx, el.cache, eventClaimKey(t.ID, dedupID)) {
		// Another replica handles this event for this trigger.
		return false
	}

	if err := el.executeTrigger(execCtx, t, event.Data); err != nil {
		fmt.Printf("trigger %s execution failed: %v\n", t.ID, err)
	}
	return true
}

// matchesFilter checks if event matches trigger filter
func (el *EventListener) matchesFilter(event Event, trigger *models.Trigger) bool {
	filter, ok := trigger.Config["filter"].(map[string]any)
	if !ok || len(filter) == 0 {
		return true // No filter means match all
	}

	// Check source filter
	if source, ok := filter["source"].(string); ok && source != "" {
		if event.Source != source {
			return false
		}
	}

	// Check custom filters
	for key, expectedValue := range filter {
		if key == "source" {
			continue // Already checked
		}

		actualValue, exists := event.Data[key]
		if !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// executeTrigger executes a workflow triggered by an event
func (el *EventListener) executeTrigger(ctx context.Context, trigger *models.Trigger, eventData map[string]any) error {
	// Merge trigger input with event data
	input := make(map[string]any)

	// First add trigger's default input
	if defaultInput, ok := trigger.Config["input"].(map[string]any); ok {
		for k, v := range defaultInput {
			input[k] = v
		}
	}

	// Then add event data (overrides trigger input)
	for k, v := range eventData {
		input[k] = v
	}

	// Execute workflow
	_, err := el.executionMgr.Execute(ctx, trigger.WorkflowID, input, executionOptionsFor(trigger.ID))
	if err != nil {
		return fmt.Errorf("failed to execute workflow: %w", err)
	}

	// Update trigger state
	state, err := LoadTriggerState(ctx, el.cache, trigger.ID)
	if err != nil {
		state = NewTriggerState(trigger.ID)
	}
	state.MarkExecuted()

	if err := state.Save(ctx, el.cache); err != nil {
		fmt.Printf("failed to save trigger state: %v\n", err)
	}

	// Update last triggered timestamp in database
	triggerUUID, _ := uuid.Parse(trigger.ID)
	if err := el.triggerRepo.MarkTriggered(ctx, triggerUUID); err != nil {
		fmt.Printf("failed to mark trigger as triggered: %v\n", err)
	}

	return nil
}

// getChannels returns all subscribed Redis channels
func (el *EventListener) getChannels() []string {
	channels := make([]string, 0, len(el.triggers))
	for eventType := range el.triggers {
		channels = append(channels, el.getEventChannel(eventType))
	}
	return channels
}

// getEventChannel returns the Redis channel for an event type
func (el *EventListener) getEventChannel(eventType string) string {
	return fmt.Sprintf("mbflow:events:%s", eventType)
}

// modelToDomain converts storage model to domain model
func (el *EventListener) modelToDomain(tm *storagemodels.TriggerModel) *models.Trigger {
	trigger := &models.Trigger{
		ID:         tm.ID.String(),
		WorkflowID: tm.WorkflowID.String(),
		Type:       models.TriggerType(tm.Type),
		Config:     make(map[string]any),
		Enabled:    tm.Enabled,
		CreatedAt:  tm.CreatedAt,
		UpdatedAt:  tm.UpdatedAt,
	}

	if tm.Config != nil {
		trigger.Config = map[string]any(tm.Config)
	}

	if tm.LastTriggeredAt != nil {
		trigger.LastRun = tm.LastTriggeredAt
	}

	return trigger
}

// Event represents an event published to Redis
type Event struct {
	// ID uniquely identifies the event; replicas use it to make sure each
	// trigger fires once per event. PublishEvent assigns one when empty.
	ID        string         `json:"id,omitempty"`
	Type      string         `json:"type"`
	Source    string         `json:"source"`
	Data      map[string]any `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
}

// PublishEvent publishes an event to Redis
func PublishEvent(ctx context.Context, c *cache.RedisCache, event Event) error {
	event.Timestamp = time.Now()
	if event.ID == "" {
		event.ID = uuid.NewString()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	channel := fmt.Sprintf("mbflow:events:%s", event.Type)
	if err := c.Client().Publish(ctx, channel, string(data)).Err(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
