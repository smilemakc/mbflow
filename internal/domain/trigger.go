package domain

// Trigger represents an event source that starts a workflow.
type Trigger struct {
	id          string
	workflowID  string
	triggerType string
	config      map[string]any
}

// NewTrigger creates a new Trigger instance.
func NewTrigger(id, workflowID, triggerType string, config map[string]any) *Trigger {
	return &Trigger{
		id:          id,
		workflowID:  workflowID,
		triggerType: triggerType,
		config:      config,
	}
}

// ReconstructTrigger reconstructs a Trigger from persistence.
func ReconstructTrigger(id, workflowID, triggerType string, config map[string]any) *Trigger {
	return &Trigger{
		id:          id,
		workflowID:  workflowID,
		triggerType: triggerType,
		config:      config,
	}
}

// ID returns the trigger ID.
func (t *Trigger) ID() string {
	return t.id
}

// WorkflowID returns the workflow ID this trigger belongs to.
func (t *Trigger) WorkflowID() string {
	return t.workflowID
}

// Type returns the type of the trigger.
func (t *Trigger) Type() string {
	return t.triggerType
}

// Config returns the configuration of the trigger.
func (t *Trigger) Config() map[string]any {
	return t.config
}
