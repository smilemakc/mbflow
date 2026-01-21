package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// NodeModel represents a workflow node in the database
type NodeModel struct {
	bun.BaseModel `bun:"table:mbflow_nodes,alias:n"`

	ID         uuid.UUID `bun:"id,pk,type:uuid,default:uuid_generate_v4()" json:"-"`
	NodeID     string    `bun:"node_id,notnull" json:"id" validate:"required,max=100"`
	WorkflowID uuid.UUID `bun:"workflow_id,notnull,type:uuid" json:"workflow_id" validate:"required"`
	Name       string    `bun:"name,notnull" json:"name" validate:"required,max=255"`
	Type       string    `bun:"type,notnull" json:"type" validate:"required,oneof=http transform llm conditional merge split delay webhook"`
	Config     JSONBMap  `bun:"config,type:jsonb,notnull,default:'{}'" json:"config"`
	Position   JSONBMap  `bun:"position,type:jsonb" json:"position,omitempty"`
	CreatedAt  time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt  time.Time `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`

	// Relationships
	Workflow       *WorkflowModel        `bun:"rel:belongs-to,join:workflow_id=id" json:"workflow,omitempty"`
	SourceEdges    []*EdgeModel          `bun:"rel:has-many,join:node_id=from_node_id" json:"source_edges,omitempty"`
	TargetEdges    []*EdgeModel          `bun:"rel:has-many,join:node_id=to_node_id" json:"target_edges,omitempty"`
	NodeExecutions []*NodeExecutionModel `bun:"rel:has-many,join:id=node_id" json:"node_executions,omitempty"`
}

// TableName returns the table name for NodeModel
func (NodeModel) TableName() string {
	return "mbflow_nodes"
}

// BeforeInsert hook to set timestamps
func (n *NodeModel) BeforeInsert(ctx interface{}) error {
	now := time.Now()
	n.CreatedAt = now
	n.UpdatedAt = now
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.Config == nil {
		n.Config = make(JSONBMap)
	}
	if n.Position == nil {
		n.Position = JSONBMap{"x": 0, "y": 0}
	}
	return nil
}

// BeforeUpdate hook to update timestamp
func (n *NodeModel) BeforeUpdate(ctx interface{}) error {
	n.UpdatedAt = time.Now()
	return nil
}

// GetPosition returns x and y coordinates
func (n *NodeModel) GetPosition() (x, y float64) {
	if n.Position == nil {
		return 0, 0
	}
	xVal, _ := n.Position["x"].(float64)
	yVal, _ := n.Position["y"].(float64)
	return xVal, yVal
}

// SetPosition sets x and y coordinates
func (n *NodeModel) SetPosition(x, y float64) {
	if n.Position == nil {
		n.Position = make(JSONBMap)
	}
	n.Position["x"] = x
	n.Position["y"] = y
}
