package domain

// Trigger is a domain entity that represents an event source that can initiate a workflow execution.
// It defines the conditions and configuration for starting a workflow instance.
// Triggers are immutable entities that are part of a Workflow aggregate.
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
