package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TriggerModel represents a workflow trigger configuration in the database
type TriggerModel struct {
	bun.BaseModel `bun:"table:mbflow_triggers,alias:t"`

	ID              uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	WorkflowID      uuid.UUID  `bun:"workflow_id,notnull,type:uuid" json:"workflow_id" validate:"required"`
	Type            string     `bun:"type,notnull" json:"type" validate:"required,oneof=manual cron webhook event interval"`
	Config          JSONBMap   `bun:"config,type:jsonb,notnull,default:'{}'" json:"config"`
	Enabled         bool       `bun:"enabled,notnull,default:true" json:"enabled"`
	LastTriggeredAt *time.Time `bun:"last_triggered_at" json:"last_triggered_at,omitempty"`
	CreatedAt       time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Relationships
	Workflow *WorkflowModel `bun:"rel:belongs-to,join:workflow_id=id" json:"workflow,omitempty"`
}

// TableName returns the table name for TriggerModel
func (TriggerModel) TableName() string {
	return "mbflow_triggers"
}

// BeforeInsert hook to set timestamps
func (t *TriggerModel) BeforeInsert(ctx interface{}) error {
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Config == nil {
		t.Config = make(JSONBMap)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (t *TriggerModel) BeforeUpdate(ctx interface{}) error {
	t.UpdatedAt = time.Now()
	return nil
}

// IsManual returns true if trigger is manual type
func (t *TriggerModel) IsManual() bool {
	return t.Type == "manual"
}

// IsCron returns true if trigger is cron type
func (t *TriggerModel) IsCron() bool {
	return t.Type == "cron"
}

// IsWebhook returns true if trigger is webhook type
func (t *TriggerModel) IsWebhook() bool {
	return t.Type == "webhook"
}

// IsEvent returns true if trigger is event type
func (t *TriggerModel) IsEvent() bool {
	return t.Type == "event"
}

// IsInterval returns true if trigger is interval type
func (t *TriggerModel) IsInterval() bool {
	return t.Type == "interval"
}

// MarkTriggered updates the last triggered timestamp
func (t *TriggerModel) MarkTriggered() {
	now := time.Now()
	t.LastTriggeredAt = &now
}

// GetCronExpression returns the cron expression if trigger is cron type
func (t *TriggerModel) GetCronExpression() string {
	if !t.IsCron() {
		return ""
	}
	if expr, ok := t.Config["expression"].(string); ok {
		return expr
	}
	return ""
}

// GetWebhookURL returns the webhook URL if trigger is webhook type
func (t *TriggerModel) GetWebhookURL() string {
	if !t.IsWebhook() {
		return ""
	}
	if url, ok := t.Config["url"].(string); ok {
		return url
	}
	return ""
}

// GetIntervalDuration returns the interval duration if trigger is interval type
func (t *TriggerModel) GetIntervalDuration() time.Duration {
	if !t.IsInterval() {
		return 0
	}
	if seconds, ok := t.Config["seconds"].(float64); ok {
		return time.Duration(seconds) * time.Second
	}
	return 0
}
