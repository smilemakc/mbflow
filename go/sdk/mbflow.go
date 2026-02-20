package mbflow

import (
	"errors"

	"github.com/smilemakc/mbflow/go/sdk/internal/grpcclient"
	"github.com/smilemakc/mbflow/go/sdk/internal/httpclient"
)

// Client is the main entry point for the MBFlow SDK.
type Client struct {
	opts clientOptions

	transport interface{ Close() error }

	workflows   WorkflowService
	executions  ExecutionService
	triggers    TriggerService
	credentials CredentialService
}

// NewClient creates a new MBFlow client.
// At least one transport (WithHTTP or WithGRPC) must be specified.
func NewClient(opts ...Option) (*Client, error) {
	options := defaultOptions()
	for _, o := range opts {
		if err := o(&options); err != nil {
			return nil, err
		}
	}

	c := &Client{opts: options}

	switch {
	case options.grpcAddress != "":
		grpcTr, err := grpcclient.New(options.grpcAddress, &grpcclient.Config{
			SystemKey:  options.systemKey,
			OnBehalfOf: options.onBehalfOf,
			Insecure:   options.grpcInsecure,
		})
		if err != nil {
			return nil, err
		}
		c.transport = grpcTr
		c.workflows = newGRPCWorkflowClient(grpcTr)
		c.executions = newGRPCExecutionClient(grpcTr)
		c.triggers = newGRPCTriggerClient(grpcTr)
		c.credentials = newGRPCCredentialClient(grpcTr)
	case options.httpEndpoint != "":
		tr := httpclient.New(options.httpEndpoint, &httpclient.Config{
			APIKey:     options.apiKey,
			SystemKey:  options.systemKey,
			OnBehalfOf: options.onBehalfOf,
			Timeout:    options.timeout,
		})
		c.transport = tr
		c.workflows = newWorkflowClient(tr)
		c.executions = newExecutionClient(tr)
		c.triggers = newTriggerClient(tr)
		c.credentials = newCredentialClient(tr)
	default:
		return nil, errors.New("mbflow: at least one transport (WithHTTP or WithGRPC) must be specified")
	}

	return c, nil
}

func (c *Client) Workflows() WorkflowService     { return c.workflows }
func (c *Client) Executions() ExecutionService   { return c.executions }
func (c *Client) Triggers() TriggerService       { return c.triggers }
func (c *Client) Credentials() CredentialService { return c.credentials }

func (c *Client) Close() error {
	if c.transport != nil {
		return c.transport.Close()
	}
	return nil
}
