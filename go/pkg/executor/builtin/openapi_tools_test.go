package builtin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

func keyClient(baseURL string) (*models.APIClient, fakeCredentialProvider) {
	client := petClient(baseURL)
	client.Auth = models.APIClientAuth{Type: models.APIClientAuthAPIKey, In: "query", Name: "key", CredentialID: "cred-1"}
	return client, fakeCredentialProvider{
		"cred-1": decryptedCredential(testOwner, models.CredentialTypeAPIKey, map[string]string{"api_key": "secret-key"}),
	}
}

func createPetArgs() map[string]any {
	return map[string]any{"path_params": map[string]any{"storeId": "s1"}, "body": map[string]any{"name": "Rex"}}
}

func TestAPIOperationExecutor_OperationTools(t *testing.T) {
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": petClient("https://pets.test")}, nil)

	defs, err := exec.OperationTools(ownerContext(testOwner), []models.APIToolRef{
		{ClientID: "client-1", OperationIDs: []string{"createPet", "health", "createPet"}},
	})
	require.NoError(t, err)
	require.Len(t, defs, 2, "a repeated operation becomes one tool")

	create := defs[0]
	assert.Equal(t, models.FunctionTypeOpenAPI, create.Type)
	assert.Equal(t, "createPet", create.Name)
	assert.Equal(t, "client-1", create.ClientID)
	assert.Equal(t, "createPet", create.OperationID)
	assert.Equal(t, `POST /stores/{storeId}/pets (API "Pets")`, create.Description)
	assert.Equal(t, []string{"path_params", "body"}, create.Parameters["required"])
	body := create.Parameters["properties"].(map[string]any)["body"].(map[string]any)
	assert.Contains(t, body["properties"], "name")
	assert.Equal(t, "health", defs[1].Name)

	_, err = exec.OperationTools(ownerContext("someone-else"), []models.APIToolRef{{ClientID: "client-1", OperationIDs: []string{"health"}}})
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound, "another user's client is reported as missing")

	_, err = exec.OperationTools(ownerContext(testOwner), []models.APIToolRef{{ClientID: "client-1", OperationIDs: []string{"removed"}}})
	assert.ErrorIs(t, err, models.ErrAPIOperationNotFound)
}

func TestAPIOperationExecutor_ExecuteOperation(t *testing.T) {
	server, captured := captureServer(t, http.StatusCreated, `{"id": 7, "name": "Rex"}`)
	client := petClient(server.URL + "/v1")
	client.Auth = models.APIClientAuth{Type: models.APIClientAuthBearer, CredentialID: "cred-1"}
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": client}, fakeCredentialProvider{
		"cred-1": decryptedCredential(testOwner, models.CredentialTypeAPIKey, map[string]string{"access_token": "tok-1"}),
	})

	args, err := parseToolArguments(`{"path_params": {"storeId": "main"}, "query_params": {"notify": true},
		"body": {"name": "Rex", "age": 9007199254740993}}`, true)
	require.NoError(t, err)
	out, err := exec.ExecuteOperation(ownerContext(testOwner), "client-1", "createPet", args)
	require.NoError(t, err)

	assert.Equal(t, "POST", captured.method)
	assert.Equal(t, "/v1/stores/main/pets", captured.path)
	assert.Equal(t, []string{"true"}, captured.query["notify"])
	assert.Equal(t, "Bearer tok-1", captured.header.Get("Authorization"))
	assert.Equal(t, `{"age":9007199254740993,"name":"Rex"}`, captured.body, "integers keep their exact value")
	assert.Equal(t, map[string]any{"status": http.StatusCreated, "body": map[string]any{"id": float64(7), "name": "Rex"}}, out)
}

func TestAPIOperationExecutor_ExecuteOperationFlattenedArguments(t *testing.T) {
	server, captured := captureServer(t, http.StatusOK, `{}`)
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": petClient(server.URL)}, nil)
	ctx := ownerContext(testOwner)

	_, err := exec.ExecuteOperation(ctx, "client-1", "createPet", map[string]any{"storeId": "s1", "notify": "false", "name": "Rex"})
	require.NoError(t, err)
	assert.Equal(t, "/stores/s1/pets", captured.path)
	assert.Equal(t, []string{"false"}, captured.query["notify"])
	assert.JSONEq(t, `{"name":"Rex"}`, captured.body)

	_, err = exec.ExecuteOperation(ctx, "client-1", "createPet", map[string]any{
		"path_params": map[string]any{"storeId": "s2"}, "storeId": "ignored", "body": `{"name": "Max"}`,
	})
	require.NoError(t, err)
	assert.Equal(t, "/stores/s2/pets", captured.path, "the grouped value wins")
	assert.JSONEq(t, `{"name":"Max"}`, captured.body)

	_, err = exec.ExecuteOperation(ctx, "client-1", "health", map[string]any{"verbose": true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown arguments: verbose")

	_, err = exec.ExecuteOperation(ctx, "client-1", "createPet", map[string]any{"path_params": []any{"s1"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `argument "path_params" must be an object`)

	_, err = exec.ExecuteOperation(ctx, "client-1", "createPet", map[string]any{"body": map[string]any{"name": "Rex"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required path parameters: storeId", "the model learns what to fix")
}

func TestAPIOperationExecutor_ExecuteOperationErrorStatus(t *testing.T) {
	server, _ := captureServer(t, http.StatusNotFound, `{"error": "store not found for key secret-key"}`)
	client, credentials := keyClient(server.URL)
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": client}, credentials)

	_, err := exec.ExecuteOperation(ownerContext(testOwner), "client-1", "createPet", createPetArgs())
	require.Error(t, err)
	assert.Equal(t, `HTTP 404: {"error":"store not found for key ***"}`, err.Error())
}

func TestAPIOperationExecutor_ExecuteOperationMasksSecretsInBody(t *testing.T) {
	server, captured := captureServer(t, http.StatusOK, `{"echo": "secret-key", "count": 2}`)
	client, credentials := keyClient(server.URL)
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": client}, credentials)

	out, err := exec.ExecuteOperation(ownerContext(testOwner), "client-1", "createPet", createPetArgs())
	require.NoError(t, err)
	assert.Equal(t, []string{"secret-key"}, captured.query["key"])
	assert.Equal(t, map[string]any{"echo": "***", "count": float64(2)}, out.(map[string]any)["body"])
}

func TestAPIOperationExecutor_ExecuteOperationTruncatesLargeBodies(t *testing.T) {
	server, _ := captureServer(t, http.StatusOK, `{"items": "`+strings.Repeat("я", maxAPIToolResultBytes)+`"}`)
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": petClient(server.URL)}, nil)

	out, err := exec.ExecuteOperation(ownerContext(testOwner), "client-1", "health", nil)
	require.NoError(t, err)
	result := out.(map[string]any)
	assert.Equal(t, true, result["truncated"])
	assert.Greater(t, result["body_bytes"], maxAPIToolResultBytes)
	body := result["body"].(string)
	assert.True(t, strings.HasPrefix(body, `{"items":"яяя`))
	assert.True(t, strings.HasSuffix(body, "я…[truncated]"), "the cut does not split a character")
	assert.LessOrEqual(t, len(body), maxAPIToolResultBytes+len("…[truncated]"))
}

func TestAPIOperationExecutor_ExecuteOperationBinaryResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
	}))
	defer server.Close()
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": petClient(server.URL)}, nil)

	out, err := exec.ExecuteOperation(ownerContext(testOwner), "client-1", "health", nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"status": http.StatusOK, "binary": true, "content_type": "image/png", "size": 4}, out)
}

func TestAPIOperationExecutor_ExecuteOperationChecksOwner(t *testing.T) {
	exec := NewAPIOperationExecutor(fakeAPIClients{"client-1": petClient("http://127.0.0.1:1")}, nil)
	_, err := exec.ExecuteOperation(ownerContext("someone-else"), "client-1", "health", nil)
	assert.ErrorIs(t, err, models.ErrAPIClientNotFound)
}

func TestToolName(t *testing.T) {
	assert.Equal(t, "listPets", toolName("listPets"))
	assert.Equal(t, "pets_list", toolName("pets.list"))
	assert.Equal(t, "get_users_id_", toolName("get /users/{id}"))
	assert.Equal(t, "op_1st", toolName("1st"))
	assert.Equal(t, "op_-x", toolName("-x"))
	assert.Len(t, toolName(strings.Repeat("a", 100)), maxToolNameLength)
}

func TestUniqueToolName(t *testing.T) {
	assert.Equal(t, "get", uniqueToolName("get", map[string]bool{"other": true}))
	assert.Equal(t, "get_3", uniqueToolName("get", map[string]bool{"get": true, "get_2": true}))

	long := strings.Repeat("a", maxToolNameLength)
	renamed := uniqueToolName(long, map[string]bool{long: true})
	assert.Len(t, renamed, maxToolNameLength)
	assert.True(t, strings.HasSuffix(renamed, "_2"))
}

func TestRegisterOpenAPI_EnablesAPIToolsInLLMNodes(t *testing.T) {
	manager := executor.NewManager()
	require.NoError(t, RegisterBuiltins(manager))
	require.NoError(t, RegisterOpenAPI(manager, fakeAPIClients{}, nil))

	exec, err := manager.Get("llm")
	require.NoError(t, err)
	registry := exec.(*LLMExecutor).toolRegistry()
	assert.NotNil(t, registry.openAPI())
	_, ok := registry.builtinRegistry.Get("get_current_time")
	assert.True(t, ok, "llm nodes call the built-in functions of function_call")
}
