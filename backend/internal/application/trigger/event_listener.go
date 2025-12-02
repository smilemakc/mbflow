package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/smilemakc/mbflow/internal/application/engine"
	"github.com/smilemakc/mbflow/internal/domain/repository"
	"github.com/smilemakc/mbflow/internal/infrastructure/cache"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
)

// EventListener manages event-based triggers via Redis pub/sub
type EventListener struct {
	triggerRepo  repository.TriggerRepository
	workflowRepo repository.WorkflowRepository
	executionMgr *engine.ExecutionManager
	cache        *cache.RedisCache

	pubsub      *redis.PubSub
	triggers    map[string][]*models.Trigger // eventType -> triggers
	mu          sync.RWMutex
	stopChan    chan struct{}
	stoppedChan chan struct{}
	isRunning   bool
}

// EventListenerConfig holds configuration for event listener
type EventListenerConfig struct {
	TriggerRepo  repository.TriggerRepository
	WorkflowRepo repository.WorkflowRepository
	ExecutionMgr *engine.ExecutionManager
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

	// Subscribe to event channels
	if len(el.triggers) > 0 {
		channels := el.getChannels()
		el.pubsub = el.cache.Client().Subscribe(ctx, channels...)

		// Start listening in background
		el.isRunning = true
		go el.listen(ctx)
	} else {
		// No triggers, close stoppedChan immediately so Stop() doesn't hang
		close(el.stoppedChan)
	}

	return nil
}

// Stop stops the event listener
func (el *EventListener) Stop() error {
	el.mu.Lock()
	isRunning := el.isRunning
	el.mu.Unlock()

	// Only close stopChan if listener is running
	if isRunning {
		close(el.stopChan)
	}

	// Close pub/sub connection
	if el.pubsub != nil {
		if err := el.pubsub.Close(); err != nil {
			return fmt.Errorf("failed to close pub/sub: %w", err)
		}
	}

	// Wait for listener to stop (only if it was started)
	if isRunning {
		<-el.stoppedChan
	}

	return nil
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

	// If listener is already running, subscribe to new channel
	if el.pubsub != nil {
		channel := el.getEventChannel(eventType)
		if err := el.pubsub.Subscribe(ctx, channel); err != nil {
			return fmt.Errorf("failed to subscribe to channel %s: %w", channel, err)
		}
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
func (el *EventListener) listen(ctx context.Context) {
	defer close(el.stoppedChan)

	ch := el.pubsub.Channel()

	for {
		select {
		case <-el.stopChan:
			return
		case msg := <-ch:
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
			el.handleEvent(ctx, event)
		}
	}
}

// handleEvent handles an incoming event
func (el *EventListener) handleEvent(ctx context.Context, event Event) {
	el.mu.RLock()
	triggers := el.triggers[event.Type]
	el.mu.RUnlock()

	for _, trigger := range triggers {
		// Check if event matches trigger filter
		if !el.matchesFilter(event, trigger) {
			continue
		}

		// Execute workflow in background
		go func(t *models.Trigger) {
			execCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			if err := el.executeTrigger(execCtx, t, event.Data); err != nil {
				fmt.Printf("trigger %s execution failed: %v\n", t.ID, err)
			}
		}(trigger)
	}
}

// matchesFilter checks if event matches trigger filter
func (el *EventListener) matchesFilter(event Event, trigger *models.Trigger) bool {
	filter, ok := trigger.Config["filter"].(map[string]interface{})
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
func (el *EventListener) executeTrigger(ctx context.Context, trigger *models.Trigger, eventData map[string]interface{}) error {
	// Merge trigger input with event data
	input := make(map[string]interface{})

	// First add trigger's default input
	if defaultInput, ok := trigger.Config["input"].(map[string]interface{}); ok {
		for k, v := range defaultInput {
			input[k] = v
		}
	}

	// Then add event data (overrides trigger input)
	for k, v := range eventData {
		input[k] = v
	}

	// Execute workflow
	_, err := el.executionMgr.Execute(ctx, trigger.WorkflowID, input, nil)
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
		Config:     make(map[string]interface{}),
		Enabled:    tm.Enabled,
		CreatedAt:  tm.CreatedAt,
		UpdatedAt:  tm.UpdatedAt,
	}

	if tm.Config != nil {
		trigger.Config = map[string]interface{}(tm.Config)
	}

	if tm.LastTriggeredAt != nil {
		trigger.LastRun = tm.LastTriggeredAt
	}

	return trigger
}

// Event represents an event published to Redis
type Event struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// PublishEvent publishes an event to Redis
func PublishEvent(ctx context.Context, cache *cache.RedisCache, event Event) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	channel := fmt.Sprintf("mbflow:events:%s", event.Type)
	if err := cache.Client().Publish(ctx, channel, string(data)).Err(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
