package models

// EphemeralExecutionRequest represents a request to execute a workflow
// without persisting the workflow definition.
type EphemeralExecutionRequest struct {
	Workflow         *Workflow       `json:"workflow"`
	Input            map[string]any  `json:"input,omitempty"`
	Mode             ExecutionMode   `json:"mode"`
	CredentialIDs    []string        `json:"credential_ids,omitempty"`
	Variables        map[string]any  `json:"variables,omitempty"`
	PersistExecution bool            `json:"persist_execution,omitempty"`
	Webhooks         []WebhookConfig `json:"webhooks,omitempty"`
}

// ExecutionMode defines the execution mode.
type ExecutionMode string

const (
	ExecutionModeSync  ExecutionMode = "sync"
	ExecutionModeAsync ExecutionMode = "async"
)

// WebhookConfig defines a webhook callback for execution events.
type WebhookConfig struct {
	URL     string            `json:"url"`
	Events  []string          `json:"events,omitempty"`
	NodeIDs []string          `json:"node_ids,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}
