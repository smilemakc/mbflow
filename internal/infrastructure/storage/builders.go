package storage

import (
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/internal/domain"
)

type WorkflowBuilder struct {
	id          uuid.UUID
	name        string
	version     string
	description string
	spec        map[string]any
	createdAt   time.Time
	updatedAt   time.Time
	nodes       []domain.Node
	edges       []domain.Edge
	triggers    []domain.Trigger
}

func NewWorkflowBuilder() *WorkflowBuilder {
	return &WorkflowBuilder{createdAt: time.Now(), id: uuid.New()}
}
func (b *WorkflowBuilder) ID(id uuid.UUID) *WorkflowBuilder          { b.id = id; return b }
func (b *WorkflowBuilder) Name(name string) *WorkflowBuilder         { b.name = name; return b }
func (b *WorkflowBuilder) Version(v string) *WorkflowBuilder         { b.version = v; return b }
func (b *WorkflowBuilder) Description(desc string) *WorkflowBuilder  { b.description = desc; return b }
func (b *WorkflowBuilder) Spec(spec map[string]any) *WorkflowBuilder { b.spec = spec; return b }
func (b *WorkflowBuilder) CreatedAt(t time.Time) *WorkflowBuilder    { b.createdAt = t; return b }
func (b *WorkflowBuilder) UpdatedAt(t time.Time) *WorkflowBuilder    { b.updatedAt = t; return b }
func (b *WorkflowBuilder) Nodes(nodes []domain.Node) *WorkflowBuilder {
	b.nodes = nodes
	return b
}
func (b *WorkflowBuilder) Edges(edges []domain.Edge) *WorkflowBuilder {
	b.edges = edges
	return b
}
func (b *WorkflowBuilder) Triggers(triggers []domain.Trigger) *WorkflowBuilder {
	b.triggers = triggers
	return b
}
func (b *WorkflowBuilder) Build() (domain.Workflow, error) {
	if b.id == uuid.Nil {
		b.id = uuid.New()
	}
	if b.createdAt.IsZero() {
		b.createdAt = time.Now()
	}
	if b.updatedAt.IsZero() {
		b.updatedAt = b.createdAt
	}
	return domain.ReconstructWorkflow(b.id, b.name, b.version, b.description, b.spec, b.createdAt, b.updatedAt, b.nodes, b.edges, b.triggers)
}

type ExecutionBuilder struct {
	id         uuid.UUID
	workflowID uuid.UUID
}

func NewExecutionBuilder() *ExecutionBuilder {
	return &ExecutionBuilder{
		id: uuid.New(),
	}
}
func (b *ExecutionBuilder) ID(id uuid.UUID) *ExecutionBuilder         { b.id = id; return b }
func (b *ExecutionBuilder) WorkflowID(id uuid.UUID) *ExecutionBuilder { b.workflowID = id; return b }
func (b *ExecutionBuilder) Build() (domain.Execution, error) {
	return domain.NewExecution(b.id, b.workflowID)
}

type EventBuilder struct {
	eventID     uuid.UUID
	eventType   domain.EventType
	workflowID  uuid.UUID
	executionID uuid.UUID
	nodeID      uuid.UUID
	timestamp   time.Time
	sequenceNum int64
	data        map[string]any
	metadata    map[string]string
}

func NewEventBuilder() *EventBuilder {
	return &EventBuilder{
		eventID:     uuid.New(),
		workflowID:  uuid.New(),
		executionID: uuid.New(),
		nodeID:      uuid.New(),
		timestamp:   time.Now(),
		metadata:    map[string]string{},
	}
}
func (b *EventBuilder) EventID(id uuid.UUID) *EventBuilder { b.eventID = id; return b }
func (b *EventBuilder) EventType(t domain.EventType) *EventBuilder {
	b.eventType = t
	return b
}
func (b *EventBuilder) WorkflowID(id uuid.UUID) *EventBuilder  { b.workflowID = id; return b }
func (b *EventBuilder) ExecutionID(id uuid.UUID) *EventBuilder { b.executionID = id; return b }
func (b *EventBuilder) NodeID(id uuid.UUID) *EventBuilder      { b.nodeID = id; return b }
func (b *EventBuilder) Timestamp(t time.Time) *EventBuilder    { b.timestamp = t; return b }
func (b *EventBuilder) SequenceNumber(seq int64) *EventBuilder {
	b.sequenceNum = seq
	return b
}
func (b *EventBuilder) Data(data map[string]any) *EventBuilder {
	b.data = data
	return b
}
func (b *EventBuilder) MetadataKV(k, v string) *EventBuilder {
	if b.metadata == nil {
		b.metadata = map[string]string{}
	}
	b.metadata[k] = v
	return b
}
func (b *EventBuilder) Build() domain.Event {
	return domain.ReconstructEvent(
		b.eventID,
		b.eventType,
		b.executionID,
		b.timestamp,
		b.sequenceNum,
		b.workflowID,
		b.nodeID,
		b.data,
		b.metadata,
	)
}

type ExecutionStateBuilder struct {
	executionID uuid.UUID
	workflowID  uuid.UUID
	status      domain.ExecutionStateStatus
	variables   map[string]interface{}
	nodeStates  map[uuid.UUID]*domain.NodeState
	startedAt   time.Time
	finishedAt  *time.Time
	errorMsg    string
}

func NewExecutionStateBuilder() *ExecutionStateBuilder {
	return &ExecutionStateBuilder{
		executionID: uuid.New(),
		workflowID:  uuid.New(),
		startedAt:   time.Now(),
		status:      domain.ExecutionStateStatusPending,
		variables:   make(map[string]interface{}),
		nodeStates:  make(map[uuid.UUID]*domain.NodeState),
	}
}

func (b *ExecutionStateBuilder) ExecutionID(id uuid.UUID) *ExecutionStateBuilder {
	b.executionID = id
	return b
}

func (b *ExecutionStateBuilder) WorkflowID(id uuid.UUID) *ExecutionStateBuilder {
	b.workflowID = id
	return b
}

func (b *ExecutionStateBuilder) Status(s domain.ExecutionStateStatus) *ExecutionStateBuilder {
	b.status = s
	return b
}

func (b *ExecutionStateBuilder) Variable(key string, value interface{}) *ExecutionStateBuilder {
	if b.variables == nil {
		b.variables = make(map[string]interface{})
	}
	b.variables[key] = value
	return b
}

func (b *ExecutionStateBuilder) Variables(vars map[string]interface{}) *ExecutionStateBuilder {
	b.variables = vars
	return b
}

func (b *ExecutionStateBuilder) NodeState(nodeID uuid.UUID, state *domain.NodeState) *ExecutionStateBuilder {
	if b.nodeStates == nil {
		b.nodeStates = make(map[uuid.UUID]*domain.NodeState)
	}
	b.nodeStates[nodeID] = state
	return b
}

func (b *ExecutionStateBuilder) NodeStates(states map[uuid.UUID]*domain.NodeState) *ExecutionStateBuilder {
	b.nodeStates = states
	return b
}

func (b *ExecutionStateBuilder) StartedAt(t time.Time) *ExecutionStateBuilder {
	b.startedAt = t
	return b
}

func (b *ExecutionStateBuilder) FinishedAt(t time.Time) *ExecutionStateBuilder {
	b.finishedAt = &t
	return b
}

func (b *ExecutionStateBuilder) ErrorMsg(msg string) *ExecutionStateBuilder {
	b.errorMsg = msg
	return b
}

func (b *ExecutionStateBuilder) Build() *domain.ExecutionState {
	return domain.ReconstructExecutionState(
		b.executionID,
		b.workflowID,
		b.status,
		b.variables,
		b.nodeStates,
		b.startedAt,
		b.finishedAt,
		b.errorMsg,
	)
}
