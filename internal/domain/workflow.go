package domain

import (
	"time"
)

// Workflow represents a flow of tasks.
type Workflow struct {
	id        string
	name      string
	version   string
	spec      []byte
	createdAt time.Time
}

// NewWorkflow creates a new Workflow instance.
func NewWorkflow(id, name, version string, spec []byte) *Workflow {
	return &Workflow{
		id:        id,
		name:      name,
		version:   version,
		spec:      spec,
		createdAt: time.Now(),
	}
}

// ReconstructWorkflow reconstructs a Workflow from persistence.
func ReconstructWorkflow(id, name, version string, spec []byte, createdAt time.Time) *Workflow {
	return &Workflow{
		id:        id,
		name:      name,
		version:   version,
		spec:      spec,
		createdAt: createdAt,
	}
}

// ID returns the workflow ID.
func (w *Workflow) ID() string {
	return w.id
}

// Name returns the workflow name.
func (w *Workflow) Name() string {
	return w.name
}

// Version returns the workflow version.
func (w *Workflow) Version() string {
	return w.version
}

// Spec returns the workflow specification.
func (w *Workflow) Spec() []byte {
	return w.spec
}

// CreatedAt returns the creation timestamp.
func (w *Workflow) CreatedAt() time.Time {
	return w.createdAt
}
