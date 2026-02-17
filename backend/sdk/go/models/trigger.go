package models

import "time"

// TriggerType represents the mechanism that initiates a workflow execution.
type TriggerType string

const (
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeCron     TriggerType = "cron"
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypeEvent    TriggerType = "event"
	TriggerTypeInterval TriggerType = "interval"
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
