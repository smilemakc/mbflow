package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

const testOwner = "owner-1"

type fakeAPIClients map[string]*models.APIClient

func (f fakeAPIClients) GetAPIClient(_ context.Context, id string) (*models.APIClient, error) {
	if c, ok := f[id]; ok {
		return c, nil
	}
	return nil, models.ErrAPIClientNotFound
}

type fakeCredentialProvider map[string]*models.CredentialsResource

func (f fakeCredentialProvider) GetDecrypted(_ context.Context, id string) (*models.CredentialsResource, error) {
	if c, ok := f[id]; ok {
		return c, nil
	}
	return nil, models.ErrResourceNotFound
}

func decryptedCredential(owner string, t models.CredentialType, data map[string]string) *models.CredentialsResource {
	return &models.CredentialsResource{
		BaseResource:   models.BaseResource{ID: "cred-1", OwnerID: owner, Status: models.ResourceStatusActive},
		CredentialType: t,
		DecryptedData:  data,
	}
}

func ownerContext(owner string) context.Context {
	return executor.WithExecutionContext(context.Background(), &executor.ExecutionContextData{OwnerID: owner})
}

type capturedRequest struct {
	method string
	path   string
	query  map[string][]string
	header http.Header
	body   string
}

func captureServer(t *testing.T, status int, response string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*captured = capturedRequest{
			method: r.Method,
			path:   r.URL.EscapedPath(),
			query:  r.URL.Query(),
			header: r.Header.Clone(),
			body:   string(body),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(server.Close)
	return server, captured
}

func petClient(baseURL string) *models.APIClient {
	return &models.APIClient{
		ID:      "client-1",
		OwnerID: testOwner,
		Name:    "Pets",
		BaseURL: baseURL,
		Operations: []*models.APIOperation{
			{
				ID:     "createPet",
				Method: "POST",
				Path:   "/stores/{storeId}/pets",
				Parameters: []*models.APIParameter{
					{Name: "storeId", In: "path", Required: true, Schema: map[string]any{"type": "string"}},
					{Name: "notify", In: "query", Schema: map[string]any{"type": "boolean"}},
				},
				RequestBody: &models.APIRequestBody{Required: true, Content: []*models.APIMediaType{{
					ContentType: "application/json",
					Schema:      map[string]any{"$ref": "#/components/schemas/Pet"},
				}}},
			},
			{ID: "health", Method: "GET", Path: "/health", NoAuth: true},
		},
		Schemas: map[string]map[string]any{
			"Pet": {"type": "object", "properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"age":  map[string]any{"type": "integer"},
			}},
		},
	}
}

func TestAPIOperationExecutor_CallsOperation(t *testing.T) {
	server, captured := captureServer(t, http.StatusCreated, `{"id": 7}`)
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": petClient(server.URL + "/v1")}, nil)

	out, err := exec.Execute(ownerContext(testOwner), map[string]any{
		"client_id":    "client-1",
		"operation_id": "createPet",
		"path_params":  map[string]any{"storeId": "main store"},
		"query_params": `{"notify": "true"}`,
		"headers":      map[string]any{"X-Request-Id": "r-1"},
		"body":         map[string]any{"name": "Rex", "age": "3"},
	}, nil)
	require.NoError(t, err)

	assert.Equal(t, "POST", captured.method)
	assert.Equal(t, "/v1/stores/main%20store/pets", captured.path)
	assert.Equal(t, []string{"true"}, captured.query["notify"])
	assert.Equal(t, "r-1", captured.header.Get("X-Request-Id"))
	assert.Equal(t, "application/json", captured.header.Get("Content-Type"))
	assert.JSONEq(t, `{"name":"Rex","age":3}`, captured.body)

	result := out.(map[string]any)
	assert.Equal(t, http.StatusCreated, result["status"])
	assert.Equal(t, map[string]any{"id": float64(7)}, result["body"])
	assert.Equal(t, "createPet", result["operation_id"])
	assert.Equal(t, map[string]any{"method": "POST", "url": server.URL + "/v1/stores/main%20store/pets?notify=true"}, result["request"])
}

func TestAPIOperationExecutor_Auth(t *testing.T) {
	cases := []struct {
		name   string
		auth   models.APIClientAuth
		cred   *models.CredentialsResource
		verify func(t *testing.T, r *capturedRequest)
	}{
		{
			name: "api key in header",
			auth: models.APIClientAuth{Type: models.APIClientAuthAPIKey, In: "header", Name: "X-API-Key", CredentialID: "cred-1"},
			cred: decryptedCredential(testOwner, models.CredentialTypeAPIKey, map[string]string{"api_key": "secret-key"}),
			verify: func(t *testing.T, r *capturedRequest) {
				assert.Equal(t, "secret-key", r.header.Get("X-API-Key"))
			},
		},
		{
			name: "api key in query",
			auth: models.APIClientAuth{Type: models.APIClientAuthAPIKey, In: "query", Name: "key", CredentialID: "cred-1"},
			cred: decryptedCredential(testOwner, models.CredentialTypeCustom, map[string]string{"token": "secret-key"}),
			verify: func(t *testing.T, r *capturedRequest) {
				assert.Equal(t, []string{"secret-key"}, r.query["key"])
			},
		},
		{
			name: "api key in cookie",
			auth: models.APIClientAuth{Type: models.APIClientAuthAPIKey, In: "cookie", Name: "sid", CredentialID: "cred-1"},
			cred: decryptedCredential(testOwner, models.CredentialTypeAPIKey, map[string]string{"api_key": "secret-key"}),
			verify: func(t *testing.T, r *capturedRequest) {
				assert.Equal(t, "sid=secret-key", r.header.Get("Cookie"))
			},
		},
		{
			name: "bearer from oauth2 access token",
			auth: models.APIClientAuth{Type: models.APIClientAuthBearer, CredentialID: "cred-1"},
			cred: decryptedCredential(testOwner, models.CredentialTypeOAuth2, map[string]string{"client_id": "c", "client_secret": "s", "access_token": "tok-1"}),
			verify: func(t *testing.T, r *capturedRequest) {
				assert.Equal(t, "Bearer tok-1", r.header.Get("Authorization"))
			},
		},
		{
			name: "basic",
			auth: models.APIClientAuth{Type: models.APIClientAuthBasic, CredentialID: "cred-1"},
			cred: decryptedCredential(testOwner, models.CredentialTypeBasicAuth, map[string]string{"username": "u", "password": "p"}),
			verify: func(t *testing.T, r *capturedRequest) {
				user, pass, ok := (&http.Request{Header: r.header}).BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "u", user)
				assert.Equal(t, "p", pass)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, captured := captureServer(t, http.StatusOK, `{}`)
			client := petClient(server.URL)
			client.Auth = tc.auth
			exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": client}, fakeCredentialProvider{"cred-1": tc.cred})

			out, err := exec.Execute(ownerContext(testOwner), map[string]any{
				"client_id":    "client-1",
				"operation_id": "createPet",
				"path_params":  map[string]any{"storeId": "s1"},
				"body":         map[string]any{"name": "Rex"},
			}, nil)
			require.NoError(t, err)
			tc.verify(t, captured)
			request := out.(map[string]any)["request"].(map[string]any)
			assert.NotContains(t, request["url"], "secret-key", "auth values are not exposed in the output")
		})
	}
}

func TestAPIOperationExecutor_NoAuthOperationSkipsCredentials(t *testing.T) {
	server, captured := captureServer(t, http.StatusOK, `ok`)
	client := petClient(server.URL)
	client.Auth = models.APIClientAuth{Type: models.APIClientAuthBearer, CredentialID: "cred-1"}
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": client}, nil)

	_, err := exec.Execute(ownerContext(testOwner), map[string]any{"client_id": "client-1", "operation_id": "health"}, nil)
	require.NoError(t, err)
	assert.Empty(t, captured.header.Get("Authorization"))
}

func TestAPIOperationExecutor_ErrorsMaskSecrets(t *testing.T) {
	server, _ := captureServer(t, http.StatusUnauthorized, `{"error": "bad key secret-key"}`)
	client := petClient(server.URL)
	client.Auth = models.APIClientAuth{Type: models.APIClientAuthAPIKey, In: "query", Name: "key", CredentialID: "cred-1"}
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": client}, fakeCredentialProvider{
		"cred-1": decryptedCredential(testOwner, models.CredentialTypeAPIKey, map[string]string{"api_key": "secret-key"}),
	})

	_, err := exec.Execute(ownerContext(testOwner), map[string]any{
		"client_id":    "client-1",
		"operation_id": "createPet",
		"path_params":  map[string]any{"storeId": "s1"},
		"body":         map[string]any{"name": "Rex"},
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
	assert.NotContains(t, err.Error(), "secret-key")
	assert.Contains(t, err.Error(), "***")
}

func TestAPIOperationExecutor_OAuth2ClientCredentialsTokenIsCached(t *testing.T) {
	var tokenRequests atomic.Int32
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequests.Add(1)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
		assert.Equal(t, "pets:write", r.Form.Get("scope"))
		user, pass, _ := r.BasicAuth()
		assert.Equal(t, "app", user)
		assert.Equal(t, "app-secret", pass)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "issued-token", "token_type": "bearer", "expires_in": 3600})
	}))
	defer tokenServer.Close()

	server, captured := captureServer(t, http.StatusOK, `{}`)
	client := petClient(server.URL)
	client.Auth = models.APIClientAuth{
		Type:         models.APIClientAuthOAuth2ClientCredentials,
		CredentialID: "cred-1",
		TokenURL:     tokenServer.URL,
		Scopes:       []string{"pets:write"},
	}
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": client}, fakeCredentialProvider{
		"cred-1": decryptedCredential(testOwner, models.CredentialTypeOAuth2, map[string]string{"client_id": "app", "client_secret": "app-secret"}),
	})
	config := map[string]any{
		"client_id":    "client-1",
		"operation_id": "createPet",
		"path_params":  map[string]any{"storeId": "s1"},
		"body":         map[string]any{"name": "Rex"},
	}

	for i := 0; i < 2; i++ {
		_, err := exec.Execute(ownerContext(testOwner), config, nil)
		require.NoError(t, err)
		assert.Equal(t, "Bearer issued-token", captured.header.Get("Authorization"))
	}
	assert.Equal(t, int32(1), tokenRequests.Load())
}

func TestAPIOperationExecutor_AccessChecks(t *testing.T) {
	client := petClient("http://127.0.0.1:1")
	clients := fakeAPIClients{"client-1": client}
	base := map[string]any{"client_id": "client-1", "operation_id": "createPet"}

	_, err := NewAPIOperationExecutor(clients, nil).Execute(ownerContext("someone-else"), base, nil)
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound, "another user's client is reported as missing")

	_, err = NewAPIOperationExecutor(clients, nil).Execute(context.Background(), base, nil)
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound, "a workflow without owner cannot use owned clients")

	_, err = NewAPIOperationExecutor(clients, nil).Execute(ownerContext(testOwner),
		map[string]any{"client_id": "client-1", "operation_id": "removed"}, nil)
	assert.ErrorIs(t, err, models.ErrAPIOperationNotFound)

	_, err = NewAPIOperationExecutor(clients, nil).Execute(ownerContext(testOwner),
		map[string]any{"client_id": "missing", "operation_id": "createPet"}, nil)
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound)

	withAuth := petClient("http://127.0.0.1:1")
	withAuth.Auth = models.APIClientAuth{Type: models.APIClientAuthBearer, CredentialID: "cred-1"}
	authConfig := map[string]any{
		"client_id": "client-1", "operation_id": "createPet",
		"path_params": map[string]any{"storeId": "s"}, "body": map[string]any{"name": "x"},
	}

	_, err = NewAPIOperationExecutor(fakeAPIClients{"client-1": withAuth}, nil).Execute(ownerContext(testOwner), authConfig, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encryption key is not configured")

	foreign := fakeCredentialProvider{"cred-1": decryptedCredential("intruder", models.CredentialTypeAPIKey, map[string]string{"api_key": "k"})}
	_, err = NewAPIOperationExecutor(fakeAPIClients{"client-1": withAuth}, foreign).Execute(ownerContext(testOwner), authConfig, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not available")

	withAuth.Auth.CredentialID = ""
	_, err = NewAPIOperationExecutor(fakeAPIClients{"client-1": withAuth}, foreign).Execute(ownerContext(testOwner), authConfig, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no credential is selected")
}

func TestAPIOperationExecutor_Validate(t *testing.T) {
	exec := NewAPIOperationExecutor(fakeAPIClients{}, nil)
	assert.NoError(t, exec.Validate(map[string]any{"client_id": "c", "operation_id": "o"}))
	assert.NoError(t, exec.Validate(map[string]any{"client_id": "c", "operation_id": "o", "query_params": `{"a": 1}`, "headers": ""}))

	err := exec.Validate(map[string]any{"operation_id": "o"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_id")
	assert.Error(t, exec.Validate(map[string]any{"client_id": "c", "operation_id": " "}))
	assert.Error(t, exec.Validate(map[string]any{"client_id": "c", "operation_id": "o", "headers": []any{"x"}}))
	assert.Error(t, exec.Validate(map[string]any{"client_id": "c", "operation_id": "o", "path_params": "not json"}))
}

func TestRedactSecrets(t *testing.T) {
	err := redactSecrets(assert.AnError, []string{"abc"})
	assert.Same(t, assert.AnError, err, "short secrets and unchanged messages keep the original error")

	wrapped := redactSecrets(io.ErrUnexpectedEOF, []string{"unexpected"})
	assert.Equal(t, "*** EOF", wrapped.Error())
	assert.ErrorIs(t, wrapped, io.ErrUnexpectedEOF)
	assert.False(t, strings.Contains(wrapped.Error(), "unexpected"))
}
