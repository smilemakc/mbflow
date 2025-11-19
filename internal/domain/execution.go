package domain

import (
	"time"
)

// ExecutionStatus represents the status of a workflow execution.
type ExecutionStatus string

const (
	ExecutionStatusPending  ExecutionStatus = "pending"
	ExecutionStatusRunning  ExecutionStatus = "running"
	ExecutionStatusFinished ExecutionStatus = "finished"
	ExecutionStatusFailed   ExecutionStatus = "failed"
)

// Execution is a domain entity that represents a workflow execution instance.
// It tracks the high-level execution metadata such as ID, workflow ID, status, and timestamps.
// This entity is separate from ExecutionState, which contains detailed execution context,
// variables, and node states. Execution serves as a lightweight record of execution lifecycle.
type Execution struct {
	id         string
	workflowID string
	status     ExecutionStatus
	startedAt  time.Time
	finishedAt *time.Time
}

// NewExecution creates a new Execution instance.
func NewExecution(id, workflowID string) *Execution {
	return &Execution{
		id:         id,
		workflowID: workflowID,
		status:     ExecutionStatusPending,
		startedAt:  time.Now(),
	}
}

// ReconstructExecution reconstructs an Execution from persistence.
func ReconstructExecution(id, workflowID string, status ExecutionStatus, startedAt time.Time, finishedAt *time.Time) *Execution {
	return &Execution{
		id:         id,
		workflowID: workflowID,
		status:     status,
		startedAt:  startedAt,
		finishedAt: finishedAt,
	}
}

// ID returns the execution ID.
func (e *Execution) ID() string {
	return e.id
}

// WorkflowID returns the ID of the workflow being executed.
func (e *Execution) WorkflowID() string {
	return e.workflowID
}

// Status returns the current status of the execution.
func (e *Execution) Status() ExecutionStatus {
	return e.status
}

// StartedAt returns the start timestamp.
func (e *Execution) StartedAt() time.Time {
	return e.startedAt
}

// FinishedAt returns the finish timestamp, if any.
func (e *Execution) FinishedAt() *time.Time {
	return e.finishedAt
}

// Start marks the execution as running.
func (e *Execution) Start() {
	e.status = ExecutionStatusRunning
}

// Finish marks the execution as finished.
func (e *Execution) Finish() {
	e.status = ExecutionStatusFinished
	now := time.Now()
	e.finishedAt = &now
}

// Fail marks the execution as failed.
func (e *Execution) Fail() {
	e.status = ExecutionStatusFailed
	now := time.Now()
	e.finishedAt = &now
}
