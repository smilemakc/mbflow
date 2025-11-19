package storage

import (
	"context"
	"database/sql"
	"mbflow/internal/domain"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type BunStore struct {
	db *bun.DB
}

func NewBunStore(dsn string) *BunStore {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())
	return &BunStore{db: db}
}

func (s *BunStore) InitSchema(ctx context.Context) error {
	models := []interface{}{
		(*WorkflowModel)(nil),
		(*ExecutionModel)(nil),
		(*EventModel)(nil),
		(*NodeModel)(nil),
		(*EdgeModel)(nil),
		(*TriggerModel)(nil),
	}
	for _, model := range models {
		if _, err := s.db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Workflow

type WorkflowModel struct {
	bun.BaseModel `bun:"table:workflows,alias:w"`

	ID        string    `bun:"id,pk,type:uuid"`
	Name      string    `bun:"name"`
	Version   string    `bun:"version"`
	Spec      []byte    `bun:"spec,type:bytea"`
	CreatedAt time.Time `bun:"created_at"`
}

func (m *WorkflowModel) ToDomain() *domain.Workflow {
	return domain.ReconstructWorkflow(
		m.ID,
		m.Name,
		m.Version,
		m.Spec,
		m.CreatedAt,
	)
}

func NewWorkflowModel(w *domain.Workflow) *WorkflowModel {
	return &WorkflowModel{
		ID:        w.ID(),
		Name:      w.Name(),
		Version:   w.Version(),
		Spec:      w.Spec(),
		CreatedAt: w.CreatedAt(),
	}
}

func (s *BunStore) SaveWorkflow(ctx context.Context, w *domain.Workflow) error {
	model := NewWorkflowModel(w)
	_, err := s.db.NewInsert().Model(model).On("CONFLICT (id) DO UPDATE").Exec(ctx)
	return err
}

func (s *BunStore) GetWorkflow(ctx context.Context, id string) (*domain.Workflow, error) {
	model := new(WorkflowModel)
	err := s.db.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (s *BunStore) ListWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	var models []WorkflowModel
	err := s.db.NewSelect().Model(&models).Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Workflow, len(models))
	for i, m := range models {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// Execution

type ExecutionModel struct {
	bun.BaseModel `bun:"table:executions,alias:e"`

	ID         string                 `bun:"id,pk,type:uuid"`
	WorkflowID string                 `bun:"workflow_id,type:uuid"`
	Status     domain.ExecutionStatus `bun:"status"`
	StartedAt  time.Time              `bun:"started_at"`
	FinishedAt *time.Time             `bun:"finished_at"`
}

func (m *ExecutionModel) ToDomain() *domain.Execution {
	return domain.ReconstructExecution(
		m.ID,
		m.WorkflowID,
		m.Status,
		m.StartedAt,
		m.FinishedAt,
	)
}

func NewExecutionModel(x *domain.Execution) *ExecutionModel {
	return &ExecutionModel{
		ID:         x.ID(),
		WorkflowID: x.WorkflowID(),
		Status:     x.Status(),
		StartedAt:  x.StartedAt(),
		FinishedAt: x.FinishedAt(),
	}
}

func (s *BunStore) SaveExecution(ctx context.Context, x *domain.Execution) error {
	model := NewExecutionModel(x)
	_, err := s.db.NewInsert().Model(model).On("CONFLICT (id) DO UPDATE").Exec(ctx)
	return err
}

func (s *BunStore) GetExecution(ctx context.Context, id string) (*domain.Execution, error) {
	model := new(ExecutionModel)
	err := s.db.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (s *BunStore) ListExecutions(ctx context.Context) ([]*domain.Execution, error) {
	var models []ExecutionModel
	err := s.db.NewSelect().Model(&models).Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Execution, len(models))
	for i, m := range models {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// Event

type EventModel struct {
	bun.BaseModel `bun:"table:events,alias:ev"`

	EventID      string            `bun:"event_id,pk,type:uuid"`
	EventType    string            `bun:"event_type"`
	WorkflowID   string            `bun:"workflow_id,type:uuid"`
	ExecutionID  string            `bun:"execution_id,type:uuid"`
	WorkflowName string            `bun:"workflow_name"`
	NodeID       string            `bun:"node_id"`
	Timestamp    time.Time         `bun:"timestamp"`
	Payload      []byte            `bun:"payload,type:bytea"`
	Metadata     map[string]string `bun:"metadata,type:jsonb"`
}

func (m *EventModel) ToDomain() *domain.Event {
	return domain.ReconstructEvent(
		m.EventID,
		m.EventType,
		m.WorkflowID,
		m.ExecutionID,
		m.WorkflowName,
		m.NodeID,
		m.Timestamp,
		m.Payload,
		m.Metadata,
	)
}

func NewEventModel(ev *domain.Event) *EventModel {
	return &EventModel{
		EventID:      ev.EventID(),
		EventType:    ev.EventType(),
		WorkflowID:   ev.WorkflowID(),
		ExecutionID:  ev.ExecutionID(),
		WorkflowName: ev.WorkflowName(),
		NodeID:       ev.NodeID(),
		Timestamp:    ev.Timestamp(),
		Payload:      ev.Payload(),
		Metadata:     ev.Metadata(),
	}
}

func (s *BunStore) AppendEvent(ctx context.Context, ev *domain.Event) error {
	model := NewEventModel(ev)
	_, err := s.db.NewInsert().Model(model).Exec(ctx)
	return err
}

func (s *BunStore) ListEventsByExecution(ctx context.Context, executionID string) ([]*domain.Event, error) {
	var models []EventModel
	err := s.db.NewSelect().Model(&models).Where("execution_id = ?", executionID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Event, len(models))
	for i, m := range models {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// Node

type NodeModel struct {
	bun.BaseModel `bun:"table:nodes,alias:n"`

	ID         string         `bun:"id,pk,type:uuid"`
	WorkflowID string         `bun:"workflow_id,type:uuid"`
	Type       string         `bun:"type"`
	Name       string         `bun:"name"`
	Config     map[string]any `bun:"config,type:jsonb"`
}

func (m *NodeModel) ToDomain() *domain.Node {
	return domain.ReconstructNode(
		m.ID,
		m.WorkflowID,
		m.Type,
		m.Name,
		m.Config,
	)
}

func NewNodeModel(n *domain.Node) *NodeModel {
	return &NodeModel{
		ID:         n.ID(),
		WorkflowID: n.WorkflowID(),
		Type:       n.Type(),
		Name:       n.Name(),
		Config:     n.Config(),
	}
}

func (s *BunStore) SaveNode(ctx context.Context, n *domain.Node) error {
	model := NewNodeModel(n)
	_, err := s.db.NewInsert().Model(model).On("CONFLICT (id) DO UPDATE").Exec(ctx)
	return err
}

func (s *BunStore) GetNode(ctx context.Context, id string) (*domain.Node, error) {
	model := new(NodeModel)
	err := s.db.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (s *BunStore) ListNodes(ctx context.Context, workflowID string) ([]*domain.Node, error) {
	var models []NodeModel
	err := s.db.NewSelect().Model(&models).Where("workflow_id = ?", workflowID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Node, len(models))
	for i, m := range models {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// Edge

type EdgeModel struct {
	bun.BaseModel `bun:"table:edges,alias:ed"`

	ID         string         `bun:"id,pk,type:uuid"`
	WorkflowID string         `bun:"workflow_id,type:uuid"`
	FromNodeID string         `bun:"from_node_id,type:uuid"`
	ToNodeID   string         `bun:"to_node_id,type:uuid"`
	Type       string         `bun:"type"`
	Config     map[string]any `bun:"config,type:jsonb"`
}

func (m *EdgeModel) ToDomain() *domain.Edge {
	return domain.ReconstructEdge(
		m.ID,
		m.WorkflowID,
		m.FromNodeID,
		m.ToNodeID,
		m.Type,
		m.Config,
	)
}

func NewEdgeModel(e *domain.Edge) *EdgeModel {
	return &EdgeModel{
		ID:         e.ID(),
		WorkflowID: e.WorkflowID(),
		FromNodeID: e.FromNodeID(),
		ToNodeID:   e.ToNodeID(),
		Type:       e.Type(),
		Config:     e.Config(),
	}
}

func (s *BunStore) SaveEdge(ctx context.Context, e *domain.Edge) error {
	model := NewEdgeModel(e)
	_, err := s.db.NewInsert().Model(model).On("CONFLICT (id) DO UPDATE").Exec(ctx)
	return err
}

func (s *BunStore) GetEdge(ctx context.Context, id string) (*domain.Edge, error) {
	model := new(EdgeModel)
	err := s.db.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (s *BunStore) ListEdges(ctx context.Context, workflowID string) ([]*domain.Edge, error) {
	var models []EdgeModel
	err := s.db.NewSelect().Model(&models).Where("workflow_id = ?", workflowID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Edge, len(models))
	for i, m := range models {
		out[i] = m.ToDomain()
	}
	return out, nil
}

// Trigger

type TriggerModel struct {
	bun.BaseModel `bun:"table:triggers,alias:t"`

	ID         string         `bun:"id,pk,type:uuid"`
	WorkflowID string         `bun:"workflow_id,type:uuid"`
	Type       string         `bun:"type"`
	Config     map[string]any `bun:"config,type:jsonb"`
}

func (m *TriggerModel) ToDomain() *domain.Trigger {
	return domain.ReconstructTrigger(
		m.ID,
		m.WorkflowID,
		m.Type,
		m.Config,
	)
}

func NewTriggerModel(t *domain.Trigger) *TriggerModel {
	return &TriggerModel{
		ID:         t.ID(),
		WorkflowID: t.WorkflowID(),
		Type:       t.Type(),
		Config:     t.Config(),
	}
}

func (s *BunStore) SaveTrigger(ctx context.Context, t *domain.Trigger) error {
	model := NewTriggerModel(t)
	_, err := s.db.NewInsert().Model(model).On("CONFLICT (id) DO UPDATE").Exec(ctx)
	return err
}

func (s *BunStore) GetTrigger(ctx context.Context, id string) (*domain.Trigger, error) {
	model := new(TriggerModel)
	err := s.db.NewSelect().Model(model).Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (s *BunStore) ListTriggers(ctx context.Context, workflowID string) ([]*domain.Trigger, error) {
	var models []TriggerModel
	err := s.db.NewSelect().Model(&models).Where("workflow_id = ?", workflowID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Trigger, len(models))
	for i, m := range models {
		out[i] = m.ToDomain()
	}
	return out, nil
}
