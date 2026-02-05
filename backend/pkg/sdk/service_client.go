package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TransportType defines the transport protocol for the Service API client.
type TransportType string

const (
	// TransportHTTP uses HTTP/JSON transport (default).
	TransportHTTP TransportType = "http"
	// TransportGRPC uses gRPC transport.
	TransportGRPC TransportType = "grpc"
)

// ServiceClientConfig holds configuration for the Service API client.
type ServiceClientConfig struct {
	// Endpoint is the base URL of the MBFlow server (e.g., "http://localhost:8585")
	Endpoint string

	// SystemKey is the system key for authentication (sysk_... prefix)
	SystemKey string

	// OnBehalfOf sets a default user ID for impersonation on all requests
	OnBehalfOf string

	// HTTPClient allows providing a custom HTTP client
	HTTPClient *http.Client

	// Timeout for HTTP requests (default: 30s)
	Timeout time.Duration

	// Transport specifies the transport protocol ("http" or "grpc"). Default: "http".
	Transport TransportType

	// GRPCAddress is the gRPC server address (e.g., "localhost:50051").
	// Required when Transport is "grpc".
	GRPCAddress string

	// GRPCInsecure disables TLS for gRPC connections.
	GRPCInsecure bool
}

// ServiceClient provides access to the MBFlow Service API.
// It authenticates using system keys and supports user impersonation.
type ServiceClient struct {
	config        ServiceClientConfig
	httpClient    *http.Client
	grpcTransport *grpcServiceTransport

	Workflows   *ServiceWorkflowsAPI
	Executions  *ServiceExecutionsAPI
	Triggers    *ServiceTriggersAPI
	Credentials *ServiceCredentialsAPI
}

// NewServiceClient creates a new Service API client.
func NewServiceClient(config ServiceClientConfig) (*ServiceClient, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if config.SystemKey == "" {
		return nil, fmt.Errorf("system key is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: config.Timeout}
	}

	if config.Transport == TransportGRPC {
		transport, err := newGRPCServiceTransport(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC transport: %w", err)
		}
		c := &ServiceClient{
			config:        config,
			httpClient:    httpClient,
			grpcTransport: transport,
		}
		c.Workflows = &ServiceWorkflowsAPI{client: c}
		c.Executions = &ServiceExecutionsAPI{client: c}
		c.Triggers = &ServiceTriggersAPI{client: c}
		c.Credentials = &ServiceCredentialsAPI{client: c}
		return c, nil
	}

	c := &ServiceClient{
		config:     config,
		httpClient: httpClient,
	}

	c.Workflows = &ServiceWorkflowsAPI{client: c}
	c.Executions = &ServiceExecutionsAPI{client: c}
	c.Triggers = &ServiceTriggersAPI{client: c}
	c.Credentials = &ServiceCredentialsAPI{client: c}

	return c, nil
}

// As returns a copy of the client that impersonates the given user for all requests.
func (c *ServiceClient) As(userID string) *ServiceClient {
	clone := *c
	clone.config.OnBehalfOf = userID
	if c.grpcTransport != nil {
		clone.grpcTransport = c.grpcTransport.withOnBehalfOf(userID)
	}
	clone.Workflows = &ServiceWorkflowsAPI{client: &clone}
	clone.Executions = &ServiceExecutionsAPI{client: &clone}
	clone.Triggers = &ServiceTriggersAPI{client: &clone}
	clone.Credentials = &ServiceCredentialsAPI{client: &clone}
	return &clone
}

// Close closes the client and releases resources. Must be called for gRPC clients.
func (c *ServiceClient) Close() error {
	if c.grpcTransport != nil {
		return c.grpcTransport.close()
	}
	return nil
}

// CallOption modifies an HTTP request before it is sent.
type CallOption func(*http.Request)

// OnBehalfOf returns a CallOption that sets the impersonation header for a single request.
func OnBehalfOf(userID string) CallOption {
	return func(req *http.Request) {
		req.Header.Set("X-On-Behalf-Of", userID)
	}
}

// doRequest performs an HTTP request to the Service API.
func (c *ServiceClient) doRequest(ctx context.Context, method, path string, body any, opts ...CallOption) (*http.Response, error) {
	url := c.config.Endpoint + "/api/v1/service" + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-System-Key", c.config.SystemKey)

	if c.config.OnBehalfOf != "" {
		req.Header.Set("X-On-Behalf-Of", c.config.OnBehalfOf)
	}

	for _, opt := range opts {
		opt(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// decodeResponse reads and decodes a JSON response body.
func decodeResponse[T any](resp *http.Response) (*T, error) {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("API error (status %d)", resp.StatusCode)
		}
		return nil, fmt.Errorf("API error %s: %s (status %d)", apiErr.Code, apiErr.Message, resp.StatusCode)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// decodeListResponse reads and decodes a JSON list response.
func decodeListResponse[T any](resp *http.Response, key string) ([]T, int, error) {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, 0, fmt.Errorf("API error (status %d)", resp.StatusCode)
		}
		return nil, 0, fmt.Errorf("API error %s: %s (status %d)", apiErr.Code, apiErr.Message, resp.StatusCode)
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	var items []T
	if data, ok := raw[key]; ok {
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, 0, fmt.Errorf("failed to decode items: %w", err)
		}
	}

	var total int
	if data, ok := raw["total"]; ok {
		json.Unmarshal(data, &total)
	}

	return items, total, nil
}

// checkResponse checks if a response indicates an error and closes the body.
func checkResponse(resp *http.Response) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return fmt.Errorf("API error (status %d)", resp.StatusCode)
		}
		return fmt.Errorf("API error %s: %s (status %d)", apiErr.Code, apiErr.Message, resp.StatusCode)
	}

	return nil
}
