package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// NodeExecutionModel represents a node execution instance in the database
type NodeExecutionModel struct {
	bun.BaseModel `bun:"table:node_executions,alias:ne"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	ExecutionID uuid.UUID  `bun:"execution_id,notnull,type:uuid" json:"execution_id" validate:"required"`
	NodeID      uuid.UUID  `bun:"node_id,notnull,type:uuid" json:"node_id" validate:"required"`
	Status      string     `bun:"status,notnull,default:'pending'" json:"status" validate:"required,oneof=pending running completed failed skipped retrying"`
	StartedAt   *time.Time `bun:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `bun:"completed_at" json:"completed_at,omitempty"`
	InputData   JSONBMap   `bun:"input_data,type:jsonb,default:'{}'" json:"input_data,omitempty"`
	OutputData  JSONBMap   `bun:"output_data,type:jsonb" json:"output_data,omitempty"`
	Error       string     `bun:"error" json:"error,omitempty"`
	RetryCount  int        `bun:"retry_count,notnull,default:0" json:"retry_count" validate:"gte=0"`
	Wave        int        `bun:"wave,notnull,default:0" json:"wave" validate:"gte=0"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Relationships
	Execution *ExecutionModel `bun:"rel:belongs-to,join:execution_id=id" json:"execution,omitempty"`
	Node      *NodeModel      `bun:"rel:belongs-to,join:node_id=id" json:"node,omitempty"`
}

// TableName returns the table name for NodeExecutionModel
func (NodeExecutionModel) TableName() string {
	return "node_executions"
}

// BeforeInsert hook to set timestamps
func (ne *NodeExecutionModel) BeforeInsert(ctx interface{}) error {
	now := time.Now()
	ne.CreatedAt = now
	ne.UpdatedAt = now
	if ne.ID == uuid.Nil {
		ne.ID = uuid.New()
	}
	if ne.InputData == nil {
		ne.InputData = make(JSONBMap)
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (ne *NodeExecutionModel) BeforeUpdate(ctx interface{}) error {
	ne.UpdatedAt = time.Now()
	return nil
}

// IsPending returns true if node execution is in pending status
func (ne *NodeExecutionModel) IsPending() bool {
	return ne.Status == "pending"
}

// IsRunning returns true if node execution is in running status
func (ne *NodeExecutionModel) IsRunning() bool {
	return ne.Status == "running"
}

// IsCompleted returns true if node execution is in completed status
func (ne *NodeExecutionModel) IsCompleted() bool {
	return ne.Status == "completed"
}

// IsFailed returns true if node execution is in failed status
func (ne *NodeExecutionModel) IsFailed() bool {
	return ne.Status == "failed"
}

// IsSkipped returns true if node execution is in skipped status
func (ne *NodeExecutionModel) IsSkipped() bool {
	return ne.Status == "skipped"
}

// IsRetrying returns true if node execution is in retrying status
func (ne *NodeExecutionModel) IsRetrying() bool {
	return ne.Status == "retrying"
}

// IsTerminal returns true if node execution is in a terminal state
func (ne *NodeExecutionModel) IsTerminal() bool {
	return ne.IsCompleted() || ne.IsFailed() || ne.IsSkipped()
}

// Duration returns the execution duration if completed
func (ne *NodeExecutionModel) Duration() *time.Duration {
	if ne.StartedAt == nil || ne.CompletedAt == nil {
		return nil
	}
	duration := ne.CompletedAt.Sub(*ne.StartedAt)
	return &duration
}

// MarkStarted sets the started timestamp and status
func (ne *NodeExecutionModel) MarkStarted() {
	now := time.Now()
	ne.StartedAt = &now
	ne.Status = "running"
}

// MarkCompleted sets the completed timestamp and status
func (ne *NodeExecutionModel) MarkCompleted() {
	now := time.Now()
	ne.CompletedAt = &now
	ne.Status = "completed"
}

// MarkFailed sets the completed timestamp, status, and error
func (ne *NodeExecutionModel) MarkFailed(err string) {
	now := time.Now()
	ne.CompletedAt = &now
	ne.Status = "failed"
	ne.Error = err
}

// MarkSkipped sets the status to skipped
func (ne *NodeExecutionModel) MarkSkipped() {
	ne.Status = "skipped"
}

// MarkRetrying increments retry count and sets status
func (ne *NodeExecutionModel) MarkRetrying() {
	ne.RetryCount++
	ne.Status = "retrying"
}
