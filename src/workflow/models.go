package workflow

import (
	"github.com/uptrace/bun"
	"mbflow/edge"
	"mbflow/internal/db"
	"mbflow/node"
	"mbflow/status"
)

type Workflow struct {
	bun.BaseModel `bun:"table:workflows,alias:workflow"`
	db.Base
	Name        string        `bun:"name,notnull"`
	Version     int           `bun:"version,default:1"`
	Description string        `bun:"description"`
	Status      status.Status `bun:"status,notnull"`
	Nodes       []node.Node   `bun:"rel:has-many,join:id=workflow_id"`
	Edges       []edge.Edge   `bun:"rel:has-many,join:id=workflow_id"`
}

type Draft struct {
	bun.BaseModel `bun:"table:workflow_drafts,alias:draft"`
	db.Base
	Name        string `bun:",notnull"`
	Description string
	Status      string      `bun:",notnull,default:'draft'"` // draft, ready, published
	Nodes       []node.Node `bun:"rel:has-many,join:id=workflow_draft_id"`
	Edges       []edge.Edge `bun:"rel:has-many,join:id=workflow_draft_id"`
}
