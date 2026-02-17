package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// EdgeModel represents a workflow edge (connection between nodes) in the database
type EdgeModel struct {
	bun.BaseModel `bun:"table:mbflow_edges,alias:e"`

	ID         uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"-"`
	EdgeID     string    `bun:"edge_id,notnull" json:"id" validate:"required,max=100"`
	WorkflowID uuid.UUID `bun:"workflow_id,notnull,type:uuid" json:"workflow_id" validate:"required"`
	FromNodeID string    `bun:"from_node_id,notnull" json:"from" validate:"required,max=100"`
	ToNodeID   string    `bun:"to_node_id,notnull" json:"to" validate:"required,max=100"`
	Condition  JSONBMap  `bun:"condition,type:jsonb" json:"condition,omitempty"`
	CreatedAt  time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt  time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Relationships
	Workflow   *WorkflowModel `bun:"rel:belongs-to,join:workflow_id=id" json:"workflow,omitempty"`
	SourceNode *NodeModel     `bun:"rel:belongs-to,join:from_node_id=node_id" json:"source_node,omitempty"`
	TargetNode *NodeModel     `bun:"rel:belongs-to,join:to_node_id=node_id" json:"target_node,omitempty"`
}

// TableName returns the table name for EdgeModel
func (EdgeModel) TableName() string {
	return "mbflow_edges"
}

// BeforeInsert hook to set timestamps and validate
func (e *EdgeModel) BeforeInsert(ctx any) error {
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	// Validate no self-reference
	if e.FromNodeID == e.ToNodeID {
		return ErrSelfReferenceEdge
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (e *EdgeModel) BeforeUpdate(ctx any) error {
	e.UpdatedAt = time.Now()
	// Validate no self-reference
	if e.FromNodeID == e.ToNodeID {
		return ErrSelfReferenceEdge
	}
	return nil
}

// IsConditional returns true if edge has a condition
func (e *EdgeModel) IsConditional() bool {
	return e.Condition != nil && len(e.Condition) > 0
}
