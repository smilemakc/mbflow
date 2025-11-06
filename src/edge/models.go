package edge

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"mbflow/internal/db"
	"mbflow/node"
)

type Template struct {
	bun.BaseModel `bun:"table:edge_templates,alias:edge_template"`
	db.Base
	Name        string         `bun:"name,notnull"`
	Description string         `bun:"description"`
	SourceType  node.Type      `bun:"source_type,notnull"`
	TargetType  node.Type      `bun:"target_type,notnull"`
	Condition   map[string]any `bun:"type:jsonb"`
}

type Edge struct {
	Template   `bun:",extend"`
	WorkflowID uuid.UUID  `bun:"workflow_id,type:uuid,notnull"`
	SourceID   uuid.UUID  `bun:"source_id,type:uuid,notnull"`
	Source     *node.Node `bun:"rel:belongs-to,join:source_id=id,on_delete:CASCADE"`
	TargetID   uuid.UUID  `bun:"target_id,type:uuid,notnull"`
	Target     *node.Node `bun:"rel:belongs-to,join:target_id=id,on_delete:CASCADE"`
}
