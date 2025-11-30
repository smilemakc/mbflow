package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ExecutionModel represents a workflow execution instance in the database
type ExecutionModel struct {
	bun.BaseModel `bun:"table:executions,alias:ex"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	WorkflowID  uuid.UUID  `bun:"workflow_id,notnull,type:uuid" json:"workflow_id" validate:"required"`
	TriggerID   *uuid.UUID `bun:"trigger_id,type:uuid" json:"trigger_id,omitempty"`
	Status      string     `bun:"status,notnull,default:'pending'" json:"status" validate:"required,oneof=pending running completed failed cancelled paused"`
	StartedAt   *time.Time `bun:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `bun:"completed_at" json:"completed_at,omitempty"`
	InputData   JSONBMap   `bun:"input_data,type:jsonb,default:'{}'" json:"input_data,omitempty"`
	OutputData  JSONBMap   `bun:"output_data,type:jsonb" json:"output_data,omitempty"`
	Variables   JSONBMap   `bun:"variables,type:jsonb,default:'{}'" json:"variables,omitempty"`
	StrictMode  bool       `bun:"strict_mode,default:false" json:"strict_mode"`
	Error       string     `bun:"error" json:"error,omitempty"`
	Metadata    JSONBMap   `bun:"metadata,type:jsonb,default:'{}'" json:"metadata,omitempty"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Relationships
	Workflow       *WorkflowModel        `bun:"rel:belongs-to,join:workflow_id=id" json:"workflow,omitempty"`
	Trigger        *TriggerModel         `bun:"rel:belongs-to,join:trigger_id=id" json:"trigger,omitempty"`
	NodeExecutions []*NodeExecutionModel `bun:"rel:has-many,join:id=execution_id" json:"node_executions,omitempty"`
	Events         []*EventModel         `bun:"rel:has-many,join:id=execution_id" json:"events,omitempty"`
}

// TableName returns the table name for ExecutionModel
func (ExecutionModel) TableName() string {
	return "executions"
}

// BeforeInsert hook to set timestamps
func (e *ExecutionModel) BeforeInsert(ctx interface{}) error {
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.InputData == nil {
		e.InputData = make(JSONBMap)
	}
	if e.Variables == nil {
		e.Variables = make(JSONBMap)
	}
	if e.Metadata == nil {
		e.Metadata = make(JSONBMap)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (e *ExecutionModel) BeforeUpdate(ctx interface{}) error {
	e.UpdatedAt = time.Now()
	return nil
}

// IsPending returns true if execution is in pending status
func (e *ExecutionModel) IsPending() bool {
	return e.Status == "pending"
}

// IsRunning returns true if execution is in running status
func (e *ExecutionModel) IsRunning() bool {
	return e.Status == "running"
}

// IsCompleted returns true if execution is in completed status
func (e *ExecutionModel) IsCompleted() bool {
	return e.Status == "completed"
}

// IsFailed returns true if execution is in failed status
func (e *ExecutionModel) IsFailed() bool {
	return e.Status == "failed"
}

// IsCancelled returns true if execution is in cancelled status
func (e *ExecutionModel) IsCancelled() bool {
	return e.Status == "cancelled"
}

// IsPaused returns true if execution is in paused status
func (e *ExecutionModel) IsPaused() bool {
	return e.Status == "paused"
}

// IsTerminal returns true if execution is in a terminal state
func (e *ExecutionModel) IsTerminal() bool {
	return e.IsCompleted() || e.IsFailed() || e.IsCancelled()
}

// Duration returns the execution duration if completed
func (e *ExecutionModel) Duration() *time.Duration {
	if e.StartedAt == nil || e.CompletedAt == nil {
		return nil
	}
	duration := e.CompletedAt.Sub(*e.StartedAt)
	return &duration
}

// MarkStarted sets the started timestamp and status
func (e *ExecutionModel) MarkStarted() {
	now := time.Now()
	e.StartedAt = &now
	e.Status = "running"
}

// MarkCompleted sets the completed timestamp and status
func (e *ExecutionModel) MarkCompleted() {
	now := time.Now()
	e.CompletedAt = &now
	e.Status = "completed"
}

// MarkFailed sets the completed timestamp, status, and error
func (e *ExecutionModel) MarkFailed(err string) {
	now := time.Now()
	e.CompletedAt = &now
	e.Status = "failed"
	e.Error = err
}

// MarkCancelled sets the completed timestamp and status
func (e *ExecutionModel) MarkCancelled() {
	now := time.Now()
	e.CompletedAt = &now
	e.Status = "cancelled"
}
