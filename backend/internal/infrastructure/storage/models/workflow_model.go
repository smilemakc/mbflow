package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// WorkflowModel represents a workflow definition in the database
type WorkflowModel struct {
	bun.BaseModel `bun:"table:workflows,alias:w"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name        string     `bun:"name,notnull" json:"name" validate:"required,max=255"`
	Description string     `bun:"description" json:"description,omitempty"`
	Status      string     `bun:"status,notnull,default:'draft'" json:"status" validate:"required,oneof=draft active archived"`
	Version     int        `bun:"version,notnull,default:1" json:"version" validate:"gte=1"`
	Variables   JSONBMap   `bun:"variables,type:jsonb,default:'{}'" json:"variables,omitempty"`
	Metadata    JSONBMap   `bun:"metadata,type:jsonb,default:'{}'" json:"metadata,omitempty"`
	CreatedBy   *uuid.UUID `bun:"created_by,type:uuid" json:"created_by,omitempty"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deleted_at,omitempty"`

	// Relationships
	Nodes    []*NodeModel    `bun:"rel:has-many,join:id=workflow_id" json:"nodes,omitempty"`
	Edges    []*EdgeModel    `bun:"rel:has-many,join:id=workflow_id" json:"edges,omitempty"`
	Triggers []*TriggerModel `bun:"rel:has-many,join:id=workflow_id" json:"triggers,omitempty"`
}

// TableName returns the table name for WorkflowModel
func (WorkflowModel) TableName() string {
	return "workflows"
}

// BeforeInsert hook to set timestamps
func (w *WorkflowModel) BeforeInsert(ctx interface{}) error {
	now := time.Now()
	w.CreatedAt = now
	w.UpdatedAt = now
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	if w.Metadata == nil {
		w.Metadata = make(JSONBMap)
	}
	if w.Variables == nil {
		w.Variables = make(JSONBMap)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (w *WorkflowModel) BeforeUpdate(ctx interface{}) error {
	w.UpdatedAt = time.Now()
	return nil
}

// IsActive returns true if workflow is in active status
func (w *WorkflowModel) IsActive() bool {
	return w.Status == "active"
}

// IsDraft returns true if workflow is in draft status
func (w *WorkflowModel) IsDraft() bool {
	return w.Status == "draft"
}

// IsDeleted returns true if workflow is soft-deleted
func (w *WorkflowModel) IsDeleted() bool {
	return w.DeletedAt != nil
}
