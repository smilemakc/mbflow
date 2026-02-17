package models

import (
	"time"
)

// Trigger represents a workflow trigger configuration.
type Trigger struct {
	ID          string         `json:"id"`
	WorkflowID  string         `json:"workflow_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Type        TriggerType    `json:"type"`
	Config      map[string]any `json:"config"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	LastRun     *time.Time     `json:"last_run,omitempty"`
	NextRun     *time.Time     `json:"next_run,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// TriggerType represents the type of trigger.
type TriggerType string

const (
	// TriggerTypeManual represents a manual trigger (user-initiated)
	TriggerTypeManual TriggerType = "manual"

	// TriggerTypeCron represents a time-based trigger using cron expressions
	TriggerTypeCron TriggerType = "cron"

	// TriggerTypeWebhook represents an HTTP webhook trigger
	TriggerTypeWebhook TriggerType = "webhook"

	// TriggerTypeEvent represents an event-driven trigger
	TriggerTypeEvent TriggerType = "event"

	// TriggerTypeInterval represents an interval-based trigger
	TriggerTypeInterval TriggerType = "interval"
)

// Validate validates the trigger structure.
func (t *Trigger) Validate() error {
	if t.WorkflowID == "" {
		return &ValidationError{Field: "workflow_id", Message: "workflow ID is required"}
	}

	if t.Name == "" {
		return &ValidationError{Field: "name", Message: "trigger name is required"}
	}

	if t.Type == "" {
		return &ValidationError{Field: "type", Message: "trigger type is required"}
	}

	// Validate type-specific configuration
	switch t.Type {
	case TriggerTypeCron:
		if err := t.validateCronConfig(); err != nil {
			return err
		}
	case TriggerTypeWebhook:
		if err := t.validateWebhookConfig(); err != nil {
			return err
		}
	case TriggerTypeEvent:
		if err := t.validateEventConfig(); err != nil {
			return err
		}
	case TriggerTypeInterval:
		if err := t.validateIntervalConfig(); err != nil {
			return err
		}
	case TriggerTypeManual:
		// Manual triggers don't require specific configuration
	default:
		return &ValidationError{Field: "type", Message: "invalid trigger type"}
	}

	return nil
}

// validateCronConfig validates cron trigger configuration.
func (t *Trigger) validateCronConfig() error {
	schedule, ok := t.Config["schedule"].(string)
	if !ok || schedule == "" {
		return &ValidationError{Field: "config.schedule", Message: "cron schedule is required"}
	}

	// TODO: Validate cron expression format
	return nil
}

// validateWebhookConfig validates webhook trigger configuration.
func (t *Trigger) validateWebhookConfig() error {
	// Webhook config is optional - the system will generate a webhook URL
	return nil
}

// validateEventConfig validates event trigger configuration.
func (t *Trigger) validateEventConfig() error {
	eventType, ok := t.Config["event_type"].(string)
	if !ok || eventType == "" {
		return &ValidationError{Field: "config.event_type", Message: "event type is required"}
	}

	return nil
}

// validateIntervalConfig validates interval trigger configuration.
func (t *Trigger) validateIntervalConfig() error {
	interval, ok := t.Config["interval"]
	if !ok {
		return &ValidationError{Field: "config.interval", Message: "interval is required"}
	}

	// interval can be a number (seconds) or a duration string
	switch v := interval.(type) {
	case float64:
		if v <= 0 {
			return &ValidationError{Field: "config.interval", Message: "interval must be positive"}
		}
	case string:
		if _, err := time.ParseDuration(v); err != nil {
			return &ValidationError{Field: "config.interval", Message: "invalid duration format"}
		}
	default:
		return &ValidationError{Field: "config.interval", Message: "interval must be a number or duration string"}
	}

	return nil
}

// CronConfig represents the configuration for a cron trigger.
type CronConfig struct {
	Schedule string `json:"schedule"`
	Timezone string `json:"timezone,omitempty"`
}

// WebhookConfig represents the configuration for a webhook trigger.
type WebhookConfig struct {
	Secret      string            `json:"secret,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
}

// EventConfig represents the configuration for an event trigger.
type EventConfig struct {
	EventType string         `json:"event_type"`
	Filter    map[string]any `json:"filter,omitempty"`
	Source    string         `json:"source,omitempty"`
}

// IntervalConfig represents the configuration for an interval trigger.
type IntervalConfig struct {
	Interval string `json:"interval"` // Duration string like "30s", "5m", "1h"
}
