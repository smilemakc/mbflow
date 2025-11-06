package node

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"mbflow/internal/db"
)

type Template struct {
	bun.BaseModel `bun:"table:node_templates,alias:node_template"`
	db.Base
	Type        Type           `bun:"type,notnull"`
	Name        string         `bun:"name,notnull"`
	Description string         `bun:"description"`
	Parameters  map[string]any `bun:"parameters,type:jsonb"`
}

type Node struct {
	Template   `bun:",extend"`
	WorkflowID uuid.UUID `bun:"workflow_id,type:uuid,notnull"`
}
