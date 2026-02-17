package mbflow

import (
	"context"

	"github.com/smilemakc/mbflow/sdk/go/models"
)

// WorkflowService provides operations on workflows.
type WorkflowService interface {
	Create(ctx context.Context, workflow *models.Workflow, opts ...RequestOption) (*models.Workflow, error)
	Get(ctx context.Context, id string, opts ...RequestOption) (*models.Workflow, error)
	Update(ctx context.Context, id string, workflow *models.Workflow, opts ...RequestOption) (*models.Workflow, error)
	Delete(ctx context.Context, id string, opts ...RequestOption) error
	List(ctx context.Context, listOpts *models.ListOptions, opts ...RequestOption) (*models.Page[models.Workflow], error)
}

// ExecutionService provides operations on workflow executions.
type ExecutionService interface {
	Run(ctx context.Context, workflowID string, input map[string]any, opts ...RequestOption) (*models.Execution, error)
	Get(ctx context.Context, id string, opts ...RequestOption) (*models.Execution, error)
	List(ctx context.Context, listOpts *models.ListOptions, opts ...RequestOption) (*models.Page[models.Execution], error)
	Cancel(ctx context.Context, id string, opts ...RequestOption) (*models.Execution, error)
	Retry(ctx context.Context, id string, opts ...RequestOption) (*models.Execution, error)
}

// TriggerService provides operations on workflow triggers.
type TriggerService interface {
	Create(ctx context.Context, trigger *models.Trigger, opts ...RequestOption) (*models.Trigger, error)
	Get(ctx context.Context, id string, opts ...RequestOption) (*models.Trigger, error)
	Update(ctx context.Context, id string, trigger *models.Trigger, opts ...RequestOption) (*models.Trigger, error)
	Delete(ctx context.Context, id string, opts ...RequestOption) error
	List(ctx context.Context, listOpts *models.ListOptions, opts ...RequestOption) (*models.Page[models.Trigger], error)
}

// CredentialService provides operations on credentials.
type CredentialService interface {
	Create(ctx context.Context, cred *models.Credential, opts ...RequestOption) (*models.Credential, error)
	Get(ctx context.Context, id string, opts ...RequestOption) (*models.Credential, error)
	Update(ctx context.Context, id string, cred *models.Credential, opts ...RequestOption) (*models.Credential, error)
	Delete(ctx context.Context, id string, opts ...RequestOption) error
	List(ctx context.Context, listOpts *models.ListOptions, opts ...RequestOption) (*models.Page[models.Credential], error)
}
