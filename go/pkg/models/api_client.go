package models

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrAPIClientNotFound     = errors.New("API client not found")
	ErrAPIOperationNotFound  = errors.New("API operation not found")
	ErrAPIClientAccessDenied = errors.New("API client access denied")
	ErrInvalidAPISpec        = errors.New("invalid API specification")
)

// APIClientAuthType selects how requests of an API client are authenticated.
type APIClientAuthType string

const (
	APIClientAuthNone                    APIClientAuthType = "none"
	APIClientAuthAPIKey                  APIClientAuthType = "api_key"
	APIClientAuthBearer                  APIClientAuthType = "bearer"
	APIClientAuthBasic                   APIClientAuthType = "basic"
	APIClientAuthOAuth2ClientCredentials APIClientAuthType = "oauth2_client_credentials"
)

// API key locations, as in the OpenAPI "in" field of an apiKey security scheme.
const (
	APIKeyInHeader = "header"
	APIKeyInQuery  = "query"
	APIKeyInCookie = "cookie"
)

// APIClientAuth describes how the executor authenticates requests.
// Secret values are never stored here: they come from the credentials
// resource referenced by CredentialID.
type APIClientAuth struct {
	Type         APIClientAuthType `json:"type"`
	CredentialID string            `json:"credential_id,omitempty"`
	// In and Name locate the API key (api_key type only).
	In   string `json:"in,omitempty"`
	Name string `json:"name,omitempty"`
	// SchemeName is the security scheme of the specification the settings were derived from.
	SchemeName string `json:"scheme_name,omitempty"`
	// TokenURL and Scopes configure the OAuth2 client credentials grant.
	TokenURL string   `json:"token_url,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

// IsEnabled reports whether requests need authentication data.
func (a APIClientAuth) IsEnabled() bool {
	return a.Type != "" && a.Type != APIClientAuthNone
}

// Validate checks the auth settings for consistency.
func (a APIClientAuth) Validate() error {
	switch a.Type {
	case "", APIClientAuthNone, APIClientAuthBearer, APIClientAuthBasic:
	case APIClientAuthAPIKey:
		if a.In != APIKeyInHeader && a.In != APIKeyInQuery && a.In != APIKeyInCookie {
			return &ValidationError{Field: "auth.in", Message: "API key location must be header, query or cookie"}
		}
		if strings.TrimSpace(a.Name) == "" {
			return &ValidationError{Field: "auth.name", Message: "API key name is required"}
		}
	case APIClientAuthOAuth2ClientCredentials:
		if a.TokenURL != "" {
			if err := validateAbsoluteHTTPURL(a.TokenURL); err != nil {
				return &ValidationError{Field: "auth.token_url", Message: err.Error()}
			}
		}
	default:
		return &ValidationError{Field: "auth.type", Message: "unsupported auth type: " + string(a.Type)}
	}
	return nil
}

// APIServer is a server of the specification with variables substituted by their defaults.
type APIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// APISecurityScheme is a security scheme declared by the specification.
type APISecurityScheme struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearer_format,omitempty"`
	In           string `json:"in,omitempty"`
	ParamName    string `json:"param_name,omitempty"`
	Description  string `json:"description,omitempty"`
	// TokenURL and Scopes come from the OAuth2 clientCredentials flow, when declared.
	TokenURL string   `json:"token_url,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

// APIParameter is a path, query, header or cookie parameter of an operation.
// Schema is a JSON Schema whose "$ref" values point to APIClient.Schemas.
type APIParameter struct {
	Name        string         `json:"name"`
	In          string         `json:"in"`
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Deprecated  bool           `json:"deprecated,omitempty"`
	Style       string         `json:"style,omitempty"`
	Explode     *bool          `json:"explode,omitempty"`
	Schema      map[string]any `json:"schema,omitempty"`
	Example     any            `json:"example,omitempty"`
	// ContentType is set when the parameter is described by "content" instead of "schema".
	ContentType string `json:"content_type,omitempty"`
}

// APIMediaType is a schema of a request or response body for one content type.
type APIMediaType struct {
	ContentType string         `json:"content_type"`
	Schema      map[string]any `json:"schema,omitempty"`
	Example     any            `json:"example,omitempty"`
}

// APIRequestBody lists the accepted content types in order of preference.
type APIRequestBody struct {
	Description string          `json:"description,omitempty"`
	Required    bool            `json:"required,omitempty"`
	Content     []*APIMediaType `json:"content"`
}

// APIResponse describes a response for a status code, a range ("2XX") or "default".
type APIResponse struct {
	Status      string          `json:"status"`
	Description string          `json:"description,omitempty"`
	Content     []*APIMediaType `json:"content,omitempty"`
}

// APIOperation is a single endpoint of an API client.
type APIOperation struct {
	// ID is unique within the client and is what workflow nodes reference.
	// It equals OperationID when the specification declares one.
	ID          string          `json:"id"`
	OperationID string          `json:"operation_id,omitempty"`
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Summary     string          `json:"summary,omitempty"`
	Description string          `json:"description,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Deprecated  bool            `json:"deprecated,omitempty"`
	Parameters  []*APIParameter `json:"parameters,omitempty"`
	RequestBody *APIRequestBody `json:"request_body,omitempty"`
	Responses   []*APIResponse  `json:"responses,omitempty"`
	// Servers override the client base URL for this operation.
	Servers []APIServer `json:"servers,omitempty"`
	// Security lists alternative requirements, each a set of scheme names.
	Security [][]string `json:"security,omitempty"`
	// NoAuth is set when the operation declares an empty security requirement.
	NoAuth bool `json:"no_auth,omitempty"`
}

// APISpecWarning is a non-fatal problem found while importing a specification.
type APISpecWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// APIClient is an HTTP API imported from an OpenAPI or Swagger specification.
type APIClient struct {
	ID          string `json:"id"`
	OwnerID     string `json:"owner_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`

	Title       string `json:"title"`
	APIVersion  string `json:"api_version,omitempty"`
	SpecVersion string `json:"spec_version"`
	BaseURL     string `json:"base_url,omitempty"`

	Servers         []APIServer          `json:"servers"`
	SecuritySchemes []*APISecurityScheme `json:"security_schemes"`
	Auth            APIClientAuth        `json:"auth"`
	DefaultHeaders  map[string]string    `json:"default_headers"`

	Operations []*APIOperation `json:"operations"`
	// Schemas holds the component schemas referenced as "#/components/schemas/<name>".
	Schemas  map[string]map[string]any `json:"schemas"`
	Warnings []APISpecWarning          `json:"warnings"`

	OperationCount int `json:"operation_count"`
	// SpecRaw is the uploaded document; it is loaded only on explicit request.
	SpecRaw  string `json:"-"`
	SpecSize int    `json:"spec_size"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FindOperation returns the operation with the given ID or nil.
func (c *APIClient) FindOperation(id string) *APIOperation {
	for _, op := range c.Operations {
		if op.ID == id {
			return op
		}
	}
	return nil
}

// Validate checks the client settings.
func (c *APIClient) Validate() error {
	if c.OwnerID == "" {
		return &ValidationError{Field: "owner_id", Message: "owner ID is required"}
	}
	if strings.TrimSpace(c.Name) == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(c.Name) > 255 {
		return &ValidationError{Field: "name", Message: "name must be 255 characters or less"}
	}
	if c.BaseURL != "" {
		if err := validateAbsoluteHTTPURL(c.BaseURL); err != nil {
			return &ValidationError{Field: "base_url", Message: err.Error()}
		}
	}
	for name := range c.DefaultHeaders {
		if strings.TrimSpace(name) == "" {
			return &ValidationError{Field: "default_headers", Message: "header name cannot be empty"}
		}
	}
	return c.Auth.Validate()
}

func validateAbsoluteHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("URL must use http or https")
	}
	if u.Host == "" {
		return errors.New("URL must include a host")
	}
	return nil
}
