package domain

import (
	"time"

	"github.com/google/uuid"
)

// Trigger is a domain entity that represents an event source that can initiate a workflow execution.
// It defines the conditions and configuration for starting a workflow instance.
// Trigger is an entity that is part of the Workflow aggregate.
type Trigger interface {
	ID() uuid.UUID
	Type() TriggerType
	Config() map[string]any
	IsActive() bool
	ShouldTrigger(input map[string]any) bool

	// Extended functionality
	Name() string
	Description() string
	Priority() int
	MaxConcurrentExecutions() int
	Cooldown() time.Duration
	ValidateInput(input map[string]any) error
}

// trigger is the internal implementation of Trigger entity.
// It is managed by the Workflow aggregate and has no independent lifecycle.
type trigger struct {
	id          uuid.UUID
	name        string
	description string
	triggerType TriggerType
	config      map[string]any
	priority    int
}

// RestoreTrigger creates a Trigger instance for reconstruction from persistence.
// This function is used internally for rebuilding the aggregate from storage.
func RestoreTrigger(id uuid.UUID, triggerType TriggerType, config map[string]any) Trigger {
	t := &trigger{
		id:          id,
		triggerType: triggerType,
		config:      config,
		priority:    0,
	}

	// Extract extended fields from config
	if name, ok := config["name"].(string); ok {
		t.name = name
	}
	if desc, ok := config["description"].(string); ok {
		t.description = desc
	}
	if priority, ok := config["priority"].(int); ok {
		t.priority = priority
	} else if priority, ok := config["priority"].(float64); ok {
		t.priority = int(priority)
	}

	return t
}

func NewTrigger(triggerType TriggerType, config map[string]any) Trigger {
	return RestoreTrigger(uuid.New(), triggerType, config)
}

// ID returns the trigger ID.
func (t *trigger) ID() uuid.UUID {
	return t.id
}

// Type returns the type of the trigger.
func (t *trigger) Type() TriggerType {
	return t.triggerType
}

// Config returns the configuration of the trigger.
func (t *trigger) Config() map[string]any {
	return t.config
}

// IsActive checks if the trigger is currently active.
func (t *trigger) IsActive() bool {
	// Check if trigger is enabled in config
	if enabled, ok := t.config["enabled"].(bool); ok {
		return enabled
	}
	// Default to active if not specified
	return true
}

// ShouldTrigger evaluates if the trigger condition is met based on input.
func (t *trigger) ShouldTrigger(input map[string]any) bool {
	switch t.triggerType {
	case TriggerTypeManual:
		// Manual triggers always fire when invoked
		return true

	case TriggerTypeAuto:
		// Auto triggers always fire
		return true

	case TriggerTypeHTTP:
		// HTTP triggers check for required parameters
		if requiredParams, ok := t.config["required_params"].([]string); ok {
			for _, param := range requiredParams {
				if _, exists := input[param]; !exists {
					return false
				}
			}
		}
		return true

	case TriggerTypeSchedule:
		// Schedule triggers would check time-based conditions
		// For now, simplified implementation
		return true

	case TriggerTypeEvent:
		// Event triggers check for specific event types
		if requiredEventType, ok := t.config["event_type"].(string); ok {
			if eventType, exists := input["event_type"]; exists {
				return eventType == requiredEventType
			}
			return false
		}
		return true

	default:
		return false
	}
}

// Name returns the trigger name.
func (t *trigger) Name() string {
	if t.name != "" {
		return t.name
	}
	return string(t.triggerType)
}

// Description returns the trigger description.
func (t *trigger) Description() string {
	return t.description
}

// Priority returns the trigger priority (higher = more important).
func (t *trigger) Priority() int {
	return t.priority
}

// MaxConcurrentExecutions returns the maximum number of concurrent executions allowed.
// Returns 0 for unlimited.
func (t *trigger) MaxConcurrentExecutions() int {
	if maxConcurrent, ok := t.config["max_concurrent"].(int); ok {
		return maxConcurrent
	}
	if maxConcurrent, ok := t.config["max_concurrent"].(float64); ok {
		return int(maxConcurrent)
	}
	return 0 // Unlimited
}

// Cooldown returns the minimum time between trigger activations.
func (t *trigger) Cooldown() time.Duration {
	if cooldown, ok := t.config["cooldown"].(string); ok {
		if d, err := time.ParseDuration(cooldown); err == nil {
			return d
		}
	}
	if cooldownMs, ok := t.config["cooldown_ms"].(float64); ok {
		return time.Duration(cooldownMs) * time.Millisecond
	}
	return 0
}

// ValidateInput validates the trigger input parameters.
func (t *trigger) ValidateInput(input map[string]any) error {
	// Validate required parameters
	if requiredParams, ok := t.config["required_params"].([]interface{}); ok {
		for _, param := range requiredParams {
			paramName, ok := param.(string)
			if !ok {
				continue
			}
			if _, exists := input[paramName]; !exists {
				return NewDomainError(
					ErrCodeValidationFailed,
					"missing required parameter: "+paramName,
					nil,
				)
			}
		}
	}

	// Validate parameter types if schema is defined
	if schema, ok := t.config["input_schema"].(map[string]interface{}); ok {
		for paramName, expectedType := range schema {
			value, exists := input[paramName]
			if !exists {
				continue // Already checked in required_params
			}

			typeStr, ok := expectedType.(string)
			if !ok {
				continue
			}

			if !validateType(value, typeStr) {
				return NewDomainError(
					ErrCodeValidationFailed,
					"parameter '"+paramName+"' has invalid type, expected: "+typeStr,
					nil,
				)
			}
		}
	}

	return nil
}

// validateType checks if a value matches the expected type string.
func validateType(value any, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "int", "integer":
		switch value.(type) {
		case int, int64, int32, float64:
			return true
		}
		return false
	case "float", "number":
		switch value.(type) {
		case float64, float32, int, int64:
			return true
		}
		return false
	case "bool", "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		switch value.(type) {
		case []interface{}, []string, []int, []float64:
			return true
		}
		return false
	case "object", "map":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true // Unknown type, skip validation
	}
}
