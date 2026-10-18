package builtin

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/pkg/netguard"
	"github.com/smilemakc/mbflow/go/pkg/openapi"
)

// OpenAPINodeType is the node type of operations of imported API clients.
const OpenAPINodeType = "openapi"

const (
	oauthTokenTimeout = 30 * time.Second
	// oauthTokenTTL is used for tokens issued without expires_in.
	oauthTokenTTL = 5 * time.Minute
	// oauthExpiryMargin renews tokens shortly before they expire.
	oauthExpiryMargin = 30 * time.Second
	// minRedactedSecretLength avoids masking short values that occur in ordinary text.
	minRedactedSecretLength = 4
)

// passThroughHTTPKeys are node config keys handled by the http executor as is.
var passThroughHTTPKeys = []string{"response_type", "ignore_status_errors", "success_status_codes"}

// APIClientProvider loads imported API clients. Returned clients may be
// shared between executions and must not be modified.
type APIClientProvider interface {
	GetAPIClient(ctx context.Context, id string) (*models.APIClient, error)
}

// CredentialProvider returns credentials with decrypted values.
type CredentialProvider interface {
	GetDecrypted(ctx context.Context, id string) (*models.CredentialsResource, error)
}

// APIOperationExecutor calls an operation of an API client imported from an
// OpenAPI or Swagger specification. Node config:
//
//	client_id, operation_id      operation to call (required)
//	path_params, query_params,
//	headers, cookies             parameter values by name; undeclared names are sent as is
//	body                         object coerced by the operation schema, or a raw string
//	content_type                 one of the content types the operation accepts
//	base_url                     overrides the base URL of the client
//	response_type, ignore_status_errors, success_status_codes
//	                             same as in the http node
//
// Requests go through the http executor, so they share its network policy,
// response handling and output format. The output also contains
// "operation_id" and "request" (method and URL without auth parameters).
type APIOperationExecutor struct {
	*executor.BaseExecutor
	clients     APIClientProvider
	credentials CredentialProvider
	http        *HTTPExecutor
	tokens      *oauthTokenCache
}

// NewAPIOperationExecutor creates an APIOperationExecutor. credentials may be nil
// when credentials are unavailable; clients with auth settings then fail to run.
func NewAPIOperationExecutor(clients APIClientProvider, credentials CredentialProvider) *APIOperationExecutor {
	return &APIOperationExecutor{
		BaseExecutor: executor.NewBaseExecutor(OpenAPINodeType),
		clients:      clients,
		credentials:  credentials,
		http:         NewHTTPExecutor(),
		tokens:       &oauthTokenCache{entries: make(map[string]cachedToken)},
	}
}

// Execute calls the configured operation.
func (e *APIOperationExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	if err := e.Validate(config); err != nil {
		return nil, err
	}
	clientID := strings.TrimSpace(config["client_id"].(string))
	operationID := strings.TrimSpace(config["operation_id"].(string))

	client, err := e.clients.GetAPIClient(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("API client %s: %w", clientID, err)
	}
	// Only the owner of the workflow may call the client: it can carry the
	// owner's credentials. Other users get the same error as for a missing client.
	owner := ""
	if execCtx, ok := executor.GetExecutionContext(ctx); ok {
		owner = execCtx.OwnerID
	}
	if client.OwnerID != owner {
		return nil, fmt.Errorf("API client %s: %w", clientID, models.ErrAPIClientNotFound)
	}

	op := client.FindOperation(operationID)
	if op == nil {
		return nil, fmt.Errorf("%w: %q in API client %q; the specification may have changed",
			models.ErrAPIOperationNotFound, operationID, client.Name)
	}

	req, err := openapi.BuildRequest(client, op, openapi.RequestInput{
		BaseURL:     e.GetStringDefault(config, "base_url", ""),
		PathParams:  objectConfig(config, "path_params"),
		QueryParams: objectConfig(config, "query_params"),
		Headers:     objectConfig(config, "headers"),
		Cookies:     objectConfig(config, "cookies"),
		Body:        config["body"],
		ContentType: e.GetStringDefault(config, "content_type", ""),
	})
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", op.Method, op.Path, err)
	}
	requestURL := req.URL.Redacted()

	var secrets []string
	if !op.NoAuth && client.Auth.IsEnabled() {
		if secrets, err = e.applyAuth(ctx, client, req); err != nil {
			return nil, err
		}
	}

	httpConfig := map[string]any{
		"method":  req.Method,
		"url":     req.URL.String(),
		"headers": headerConfig(req.Header),
	}
	if req.Body != nil {
		httpConfig["body"] = req.Body
	}
	for _, key := range passThroughHTTPKeys {
		if value, ok := config[key]; ok {
			httpConfig[key] = value
		}
	}

	output, err := e.http.Execute(ctx, httpConfig, input)
	if err != nil {
		return nil, redactSecrets(fmt.Errorf("%s %s: %w", op.Method, op.Path, err), secrets)
	}
	result, ok := output.(map[string]any)
	if !ok {
		return output, nil
	}
	result["operation_id"] = op.ID
	result["request"] = map[string]any{"method": req.Method, "url": requestURL}
	return result, nil
}

// Validate checks that the node references an operation.
func (e *APIOperationExecutor) Validate(config map[string]any) error {
	for _, key := range []string{"client_id", "operation_id"} {
		value, ok := config[key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("required field missing: %s", key)
		}
	}
	for _, key := range []string{"path_params", "query_params", "headers", "cookies"} {
		value, ok := config[key]
		if !ok || value == nil {
			continue
		}
		if _, isMap := value.(map[string]any); isMap {
			continue
		}
		if s, isString := value.(string); isString && (strings.TrimSpace(s) == "" || objectConfig(config, key) != nil) {
			continue
		}
		return fmt.Errorf("%s must be an object", key)
	}
	return nil
}

// applyAuth adds authentication to the request and returns the secret values
// it used, so they can be masked in error messages.
func (e *APIOperationExecutor) applyAuth(ctx context.Context, client *models.APIClient, req *openapi.Request) ([]string, error) {
	auth := client.Auth
	if auth.CredentialID == "" {
		return nil, fmt.Errorf("API client %q uses %s auth but no credential is selected", client.Name, auth.Type)
	}
	if e.credentials == nil {
		return nil, errors.New("credentials are not available: encryption key is not configured")
	}
	cred, err := e.credentials.GetDecrypted(ctx, auth.CredentialID)
	if err != nil {
		return nil, fmt.Errorf("credential of API client %q: %w", client.Name, err)
	}
	if cred.OwnerID != client.OwnerID || cred.Status != models.ResourceStatusActive {
		return nil, fmt.Errorf("credential of API client %q is not available", client.Name)
	}
	data := cred.DecryptedData

	switch auth.Type {
	case models.APIClientAuthAPIKey:
		key := secretValue(data, "api_key", "access_token", "token", "key")
		if key == "" {
			return nil, fmt.Errorf("credential of API client %q has no API key", client.Name)
		}
		switch auth.In {
		case models.APIKeyInQuery:
			query := req.URL.Query()
			query.Set(auth.Name, key)
			req.URL.RawQuery = query.Encode()
		case models.APIKeyInCookie:
			appendCookie(req.Header, auth.Name, key)
		default:
			req.Header.Set(auth.Name, key)
		}
		return []string{key}, nil

	case models.APIClientAuthBearer:
		token := secretValue(data, "access_token", "api_key", "token", "bearer_token")
		if token == "" {
			return nil, fmt.Errorf("credential of API client %q has no token", client.Name)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return []string{token}, nil

	case models.APIClientAuthBasic:
		username, password := data["username"], data["password"]
		if username == "" && password == "" {
			return nil, fmt.Errorf("credential of API client %q has no username or password", client.Name)
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req.Header.Set("Authorization", "Basic "+encoded)
		return []string{password, encoded}, nil

	case models.APIClientAuthOAuth2ClientCredentials:
		token, err := e.clientCredentialsToken(ctx, client, cred)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return []string{token}, nil
	}
	return nil, fmt.Errorf("API client %q: unsupported auth type %q", client.Name, auth.Type)
}

func (e *APIOperationExecutor) clientCredentialsToken(ctx context.Context, client *models.APIClient, cred *models.CredentialsResource) (string, error) {
	data := cred.DecryptedData
	tokenURL := strings.TrimSpace(client.Auth.TokenURL)
	if tokenURL == "" {
		tokenURL = strings.TrimSpace(data["token_url"])
	}
	if tokenURL == "" {
		if token := data["access_token"]; token != "" {
			return token, nil
		}
		return "", fmt.Errorf("API client %q: OAuth2 token URL is not configured", client.Name)
	}

	scopes := client.Auth.Scopes
	if len(scopes) == 0 && data["scopes"] != "" {
		scopes = strings.FieldsFunc(data["scopes"], func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	}
	cfg := clientcredentials.Config{
		ClientID:     data["client_id"],
		ClientSecret: data["client_secret"],
		TokenURL:     tokenURL,
		Scopes:       scopes,
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return "", fmt.Errorf("credential of API client %q has no client_id or client_secret", client.Name)
	}

	key := tokenCacheKey(cred.ID, cfg)
	if token := e.tokens.get(key); token != "" {
		return token, nil
	}
	tokenCtx := context.WithValue(ctx, oauth2.HTTPClient, netguard.DefaultClient(oauthTokenTimeout))
	token, err := cfg.Token(tokenCtx)
	if err != nil {
		return "", redactSecrets(fmt.Errorf("API client %q: obtain OAuth2 token: %w", client.Name, err), []string{cfg.ClientSecret})
	}
	e.tokens.put(key, token)
	return token.AccessToken, nil
}

// secretValue returns the first non-empty value among keys, or the only
// value of a credential that has exactly one field.
func secretValue(data map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := data[key]; v != "" {
			return v
		}
	}
	if len(data) == 1 {
		for _, v := range data {
			return v
		}
	}
	return ""
}

func appendCookie(header http.Header, name, value string) {
	cookie := name + "=" + url.QueryEscape(value)
	if existing := header.Get("Cookie"); existing != "" {
		cookie = existing + "; " + cookie
	}
	header.Set("Cookie", cookie)
}

// objectConfig returns a map config value. A JSON object string is accepted
// too, since a template such as "{{input.params}}" resolves to JSON text.
func objectConfig(config map[string]any, key string) map[string]any {
	switch v := config[key].(type) {
	case map[string]any:
		return v
	case string:
		trimmed := strings.TrimSpace(v)
		if strings.HasPrefix(trimmed, "{") {
			var m map[string]any
			if err := json.Unmarshal([]byte(trimmed), &m); err == nil {
				return m
			}
		}
	}
	return nil
}

func headerConfig(header http.Header) map[string]any {
	result := make(map[string]any, len(header))
	for name, values := range header {
		result[name] = strings.Join(values, ", ")
	}
	return result
}

// redactedError masks secret values in the message of err.
type redactedError struct {
	msg string
	err error
}

func (e *redactedError) Error() string { return e.msg }
func (e *redactedError) Unwrap() error { return e.err }

func redactSecrets(err error, secrets []string) error {
	msg := err.Error()
	redacted := msg
	for _, secret := range secrets {
		if len(secret) < minRedactedSecretLength {
			continue
		}
		for _, form := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
			redacted = strings.ReplaceAll(redacted, form, "***")
		}
	}
	if redacted == msg {
		return err
	}
	return &redactedError{msg: redacted, err: err}
}

func tokenCacheKey(credentialID string, cfg clientcredentials.Config) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		credentialID, cfg.TokenURL, cfg.ClientID, cfg.ClientSecret, strings.Join(cfg.Scopes, " "),
	}, "\x00")))
	return hex.EncodeToString(sum[:])
}

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

// oauthTokenCache keeps client credentials tokens per credential, token URL,
// client and scopes, so a workflow does not request a token on every call.
type oauthTokenCache struct {
	mu      sync.Mutex
	entries map[string]cachedToken
}

func (c *oauthTokenCache) get(key string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		delete(c.entries, key)
		return ""
	}
	return entry.accessToken
}

func (c *oauthTokenCache) put(key string, token *oauth2.Token) {
	now := time.Now()
	expiresAt := now.Add(oauthTokenTTL)
	if !token.Expiry.IsZero() {
		expiresAt = token.Expiry.Add(-oauthExpiryMargin)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, k)
		}
	}
	c.entries[key] = cachedToken{accessToken: token.AccessToken, expiresAt: expiresAt}
}
