package sdk

import (
	"context"
	"fmt"
	"net/http"
)

// ServiceCredentialsAPI provides access to credential operations via Service API.
type ServiceCredentialsAPI struct {
	client *ServiceClient
}

// ServiceCredential represents a credential response from the Service API.
type ServiceCredential struct {
	ID           string            `json:"id"`
	OwnerID      string            `json:"owner_id"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Provider     string            `json:"provider"`
	AuthType     string            `json:"auth_type"`
	Status       string            `json:"status"`
	UsageCount   int64             `json:"usage_count"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

// List returns all credentials, optionally filtered by user_id.
func (a *ServiceCredentialsAPI) List(ctx context.Context, opts *ServiceCredentialListOptions, callOpts ...CallOption) ([]*ServiceCredential, int, error) {
	path := "/credentials"
	if opts != nil {
		path += fmt.Sprintf("?limit=%d&offset=%d", opts.Limit, opts.Offset)
		if opts.UserID != "" {
			path += "&user_id=" + opts.UserID
		}
		if opts.Provider != "" {
			path += "&provider=" + opts.Provider
		}
	}

	resp, err := a.client.doRequest(ctx, http.MethodGet, path, nil, callOpts...)
	if err != nil {
		return nil, 0, err
	}

	return decodeListResponse[*ServiceCredential](resp, "credentials")
}

// Create creates a new credential.
func (a *ServiceCredentialsAPI) Create(ctx context.Context, req *ServiceCreateCredentialRequest, callOpts ...CallOption) (*ServiceCredential, error) {
	resp, err := a.client.doRequest(ctx, http.MethodPost, "/credentials", req, callOpts...)
	if err != nil {
		return nil, err
	}
	return decodeResponse[ServiceCredential](resp)
}

// Update updates a credential.
func (a *ServiceCredentialsAPI) Update(ctx context.Context, credentialID string, req *ServiceUpdateCredentialRequest, callOpts ...CallOption) (*ServiceCredential, error) {
	resp, err := a.client.doRequest(ctx, http.MethodPut, "/credentials/"+credentialID, req, callOpts...)
	if err != nil {
		return nil, err
	}
	return decodeResponse[ServiceCredential](resp)
}

// Delete deletes a credential.
func (a *ServiceCredentialsAPI) Delete(ctx context.Context, credentialID string, callOpts ...CallOption) error {
	resp, err := a.client.doRequest(ctx, http.MethodDelete, "/credentials/"+credentialID, nil, callOpts...)
	if err != nil {
		return err
	}
	return checkResponse(resp)
}

// ServiceCredentialListOptions defines filtering for listing credentials.
type ServiceCredentialListOptions struct {
	Limit    int
	Offset   int
	UserID   string
	Provider string
}

// ServiceCreateCredentialRequest defines the request for creating a credential.
type ServiceCreateCredentialRequest struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Provider     string            `json:"provider"`
	AuthType     string            `json:"auth_type"`
	Data         map[string]string `json:"data"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ServiceUpdateCredentialRequest defines the request for updating a credential.
type ServiceUpdateCredentialRequest struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
