package storage

import (
	"mbflow/internal/domain"
	"time"
)

type WorkflowBuilder struct {
	id        string
	name      string
	version   string
	spec      []byte
	createdAt time.Time
}

func NewWorkflowBuilder() *WorkflowBuilder {
	return &WorkflowBuilder{createdAt: time.Now()}
}
func (b *WorkflowBuilder) ID(id string) *WorkflowBuilder          { b.id = id; return b }
func (b *WorkflowBuilder) Name(name string) *WorkflowBuilder      { b.name = name; return b }
func (b *WorkflowBuilder) Version(v string) *WorkflowBuilder      { b.version = v; return b }
func (b *WorkflowBuilder) SpecBytes(spec []byte) *WorkflowBuilder { b.spec = spec; return b }
func (b *WorkflowBuilder) CreatedAt(t time.Time) *WorkflowBuilder { b.createdAt = t; return b }
func (b *WorkflowBuilder) Build() *domain.Workflow {
	return domain.ReconstructWorkflow(b.id, b.name, b.version, b.spec, b.createdAt)
}

type ExecutionBuilder struct {
	id         string
	workflowID string
	status     domain.ExecutionStatus
	startedAt  time.Time
	finishedAt *time.Time
}

func NewExecutionBuilder() *ExecutionBuilder {
	return &ExecutionBuilder{startedAt: time.Now(), status: domain.ExecutionStatusPending}
}
func (b *ExecutionBuilder) ID(id string) *ExecutionBuilder                    { b.id = id; return b }
func (b *ExecutionBuilder) WorkflowID(id string) *ExecutionBuilder            { b.workflowID = id; return b }
func (b *ExecutionBuilder) Status(s domain.ExecutionStatus) *ExecutionBuilder { b.status = s; return b }
func (b *ExecutionBuilder) StartedAt(t time.Time) *ExecutionBuilder           { b.startedAt = t; return b }
func (b *ExecutionBuilder) FinishedAt(t time.Time) *ExecutionBuilder          { b.finishedAt = &t; return b }
func (b *ExecutionBuilder) Build() *domain.Execution {
	return domain.ReconstructExecution(b.id, b.workflowID, b.status, b.startedAt, b.finishedAt)
}

type EventBuilder struct {
	eventID      string
	eventType    string
	workflowID   string
	executionID  string
	workflowName string
	nodeID       string
	timestamp    time.Time
	payload      []byte
	metadata     map[string]string
}

func NewEventBuilder() *EventBuilder {
	return &EventBuilder{timestamp: time.Now(), metadata: map[string]string{}}
}
func (b *EventBuilder) EventID(id string) *EventBuilder        { b.eventID = id; return b }
func (b *EventBuilder) EventType(t string) *EventBuilder       { b.eventType = t; return b }
func (b *EventBuilder) WorkflowID(id string) *EventBuilder     { b.workflowID = id; return b }
func (b *EventBuilder) ExecutionID(id string) *EventBuilder    { b.executionID = id; return b }
func (b *EventBuilder) WorkflowName(name string) *EventBuilder { b.workflowName = name; return b }
func (b *EventBuilder) NodeID(id string) *EventBuilder         { b.nodeID = id; return b }
func (b *EventBuilder) Timestamp(t time.Time) *EventBuilder    { b.timestamp = t; return b }
func (b *EventBuilder) PayloadBytes(p []byte) *EventBuilder    { b.payload = p; return b }
func (b *EventBuilder) MetadataKV(k, v string) *EventBuilder {
	if b.metadata == nil {
		b.metadata = map[string]string{}
	}
	b.metadata[k] = v
	return b
}
func (b *EventBuilder) Build() *domain.Event {
	return domain.ReconstructEvent(b.eventID, b.eventType, b.workflowID, b.executionID, b.workflowName, b.nodeID, b.timestamp, b.payload, b.metadata)
}
