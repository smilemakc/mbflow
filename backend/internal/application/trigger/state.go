package trigger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/smilemakc/mbflow/internal/infrastructure/cache"
)

// TriggerState represents the runtime state of a trigger
type TriggerState struct {
	TriggerID      string    `json:"trigger_id"`
	LastExecuted   time.Time `json:"last_executed"`
	NextExecution  time.Time `json:"next_execution,omitempty"`
	ExecutionCount int64     `json:"execution_count"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewTriggerState creates a new trigger state
func NewTriggerState(triggerID string) *TriggerState {
	return &TriggerState{
		TriggerID:      triggerID,
		ExecutionCount: 0,
		UpdatedAt:      time.Now(),
	}
}

// MarkExecuted marks the trigger as executed
func (ts *TriggerState) MarkExecuted() {
	ts.LastExecuted = time.Now()
	ts.ExecutionCount++
	ts.UpdatedAt = time.Now()
}

// SetNextExecution sets the next execution time
func (ts *TriggerState) SetNextExecution(t time.Time) {
	ts.NextExecution = t
	ts.UpdatedAt = time.Now()
}

// Save persists the trigger state to Redis
func (ts *TriggerState) Save(ctx context.Context, cache *cache.RedisCache) error {
	key := getTriggerStateKey(ts.TriggerID)

	data, err := json.Marshal(ts)
	if err != nil {
		return fmt.Errorf("failed to marshal trigger state: %w", err)
	}

	// Store with no expiration - state persists until trigger is deleted
	if err := cache.Set(ctx, key, string(data), 0); err != nil {
		return fmt.Errorf("failed to save trigger state: %w", err)
	}

	return nil
}

// LoadTriggerState loads trigger state from Redis
func LoadTriggerState(ctx context.Context, cache *cache.RedisCache, triggerID string) (*TriggerState, error) {
	key := getTriggerStateKey(triggerID)

	data, err := cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to load trigger state: %w", err)
	}

	var state TriggerState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trigger state: %w", err)
	}

	return &state, nil
}

// DeleteTriggerState deletes trigger state from Redis
func DeleteTriggerState(ctx context.Context, cache *cache.RedisCache, triggerID string) error {
	key := getTriggerStateKey(triggerID)
	return cache.Delete(ctx, key)
}

// getTriggerStateKey returns the Redis key for trigger state
func getTriggerStateKey(triggerID string) string {
	return fmt.Sprintf("trigger:%s:state", triggerID)
}
