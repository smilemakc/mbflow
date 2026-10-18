package openapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func objectAt(t *testing.T, schema map[string]any, path ...string) map[string]any {
	t.Helper()
	current := schema
	for _, key := range path {
		next, ok := current[key].(map[string]any)
		require.Truef(t, ok, "no object at %q in %v", key, current)
		current = next
	}
	return current
}

// assertSelfContained fails when the schema still references components:
// LLM providers reject tool schemas with unresolved "$ref".
func assertSelfContained(t *testing.T, schema map[string]any) {
	t.Helper()
	data, err := json.Marshal(schema)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "$ref")
}

func TestToolParameters_GroupsParametersByLocation(t *testing.T) {
	client := petstoreClient(t)
	params := ToolParameters(client, client.FindOperation("getPet"))
	assertSelfContained(t, params)

	assert.Equal(t, "object", params["type"])
	assert.Equal(t, []string{ToolArgPathParams}, params["required"])

	path := objectAt(t, params, "properties", ToolArgPathParams)
	assert.Equal(t, []string{"petId"}, path["required"])
	petID := objectAt(t, path, "properties", "petId")
	assert.Equal(t, "integer", petID["type"])
	assert.Equal(t, "int64", petID["format"])
	assert.Equal(t, "Overridden at operation level", petID["description"])

	headers := objectAt(t, params, "properties", ToolArgHeaders)
	assert.Contains(t, objectAt(t, headers, "properties"), "X-Trace-Id")
	assert.NotContains(t, headers, "required")
	assert.NotContains(t, objectAt(t, params, "properties"), ToolArgBody)
}

func TestToolParameters_QueryParameters(t *testing.T) {
	client := petstoreClient(t)
	params := ToolParameters(client, client.FindOperation("get_pets"))

	query := objectAt(t, params, "properties", ToolArgQueryParams)
	assert.Equal(t, []string{"tags"}, query["required"])
	assert.Equal(t, []string{ToolArgQueryParams}, params["required"])

	limit := objectAt(t, query, "properties", "limit")
	assert.Equal(t, "integer", limit["type"])
	assert.EqualValues(t, 100, limit["maximum"])

	tags := objectAt(t, query, "properties", "tags")
	assert.Equal(t, "array", tags["type"])
	assert.Equal(t, map[string]any{"type": "string"}, tags["items"])

	filter := objectAt(t, query, "properties", "filter", "properties", "status")
	assert.Equal(t, "string", filter["type"])
}

func TestToolParameters_InlinesBodyAndCutsRecursion(t *testing.T) {
	client := petstoreClient(t)
	op := client.FindOperation("post_pets")
	require.NotNil(t, op)
	params := ToolParameters(client, op)
	assertSelfContained(t, params)

	assert.Equal(t, []string{ToolArgBody}, params["required"])
	body := objectAt(t, params, "properties", ToolArgBody)
	assert.Equal(t, "object", body["type"])
	assert.Equal(t, []string{"name"}, body["required"])
	assert.Equal(t, "Request body.", body["description"])

	props := objectAt(t, body, "properties")
	assert.Equal(t, "string", objectAt(t, props, "name")["type"])
	assert.Equal(t, "boolean", objectAt(t, props, "vaccinated")["type"])

	category := objectAt(t, props, "category")
	assert.Equal(t, "string", objectAt(t, category, "properties", "name")["type"])
	parent := objectAt(t, category, "properties", "parent")
	assert.Equal(t, "object", parent["type"])
	assert.NotContains(t, parent, "properties", "a recursive reference keeps only its type")
	assert.Contains(t, parent["description"], "omitted")
}

func TestToolParameters_MergesAllOfAndDropsReadOnlyFields(t *testing.T) {
	client := &models.APIClient{
		Name: "Pets",
		Schemas: map[string]map[string]any{
			"NewPet": {"type": "object", "required": []any{"name"}, "properties": map[string]any{
				"name": map[string]any{"type": "string", "description": "Pet name"},
			}},
			"Pet": {"allOf": []any{
				map[string]any{"$ref": "#/components/schemas/NewPet"},
				map[string]any{"type": "object", "required": []any{"id", "kind"}, "properties": map[string]any{
					"id":   map[string]any{"type": "integer", "readOnly": true},
					"kind": map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "integer"}}},
					"note": map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "null"}}},
				}},
			}},
		},
	}
	op := &models.APIOperation{ID: "updatePet", Method: "PUT", Path: "/pets", RequestBody: &models.APIRequestBody{
		Content: []*models.APIMediaType{{ContentType: "application/json", Schema: map[string]any{"$ref": "#/components/schemas/Pet"}}},
	}}

	body := objectAt(t, ToolParameters(client, op), "properties", ToolArgBody)
	assert.Equal(t, "object", body["type"])
	assert.ElementsMatch(t, []string{"name", "kind"}, body["required"])

	props := objectAt(t, body, "properties")
	assert.NotContains(t, props, "id")
	assert.Equal(t, "Pet name", objectAt(t, props, "name")["description"])
	assert.Equal(t, []any{map[string]any{"type": "string"}, map[string]any{"type": "integer"}}, objectAt(t, props, "kind")["anyOf"])
	assert.Equal(t, map[string]any{"type": "string"}, objectAt(t, props, "note"), "a null alternative is dropped")
}

func TestToolParameters_TypeListsAndEnums(t *testing.T) {
	client := parseFixtureClient(t, "modern-3.1.json")
	op := client.FindOperation("patch_users_userId")
	require.NotNil(t, op)

	params := ToolParameters(client, op)
	body := objectAt(t, params, "properties", ToolArgBody)
	assert.Equal(t, "Request body.", body["description"], "a +json content type needs no note")
	assert.Equal(t, "string", objectAt(t, body, "properties", "nickname")["type"])
	assert.Equal(t, "uuid", objectAt(t, params, "properties", ToolArgPathParams, "properties", "userId")["format"])

	schemas := map[string]map[string]any{}
	tb := &toolSchemaBuilder{schemas: schemas, active: map[string]bool{}, budget: maxToolSchemaNodes}
	strEnum := tb.inline(map[string]any{"type": "string", "enum": []any{"asc", "desc", nil}}, 1)
	assert.Equal(t, []any{"asc", "desc"}, strEnum["enum"])

	intEnum := tb.inline(map[string]any{"type": "integer", "enum": []any{1.0, 2.0}}, 1)
	assert.NotContains(t, intEnum, "enum", "providers such as Gemini accept only string enums")
	assert.Equal(t, "Allowed values: 1, 2.", intEnum["description"])
}

func TestToolParameters_TextBody(t *testing.T) {
	op := &models.APIOperation{ID: "note", Method: "POST", Path: "/notes", RequestBody: &models.APIRequestBody{
		Description: "Note text",
		Content:     []*models.APIMediaType{{ContentType: "text/plain; charset=utf-8"}},
	}}
	body := objectAt(t, ToolParameters(&models.APIClient{}, op), "properties", ToolArgBody)
	assert.Equal(t, map[string]any{"type": "string", "description": "Note text Sent as text/plain."}, body)
}

func TestToolParameters_SkipsParametersSetByAuth(t *testing.T) {
	op := &models.APIOperation{ID: "search", Method: "GET", Path: "/search", Parameters: []*models.APIParameter{
		{Name: "X-API-Key", In: "header", Required: true, Schema: map[string]any{"type": "string"}},
		{Name: "q", In: "query", Schema: map[string]any{"type": "string"}},
	}}
	client := &models.APIClient{Name: "Search", Auth: models.APIClientAuth{
		Type: models.APIClientAuthAPIKey, In: models.APIKeyInHeader, Name: "x-api-key", CredentialID: "cred",
	}}

	params := ToolParameters(client, op)
	assert.NotContains(t, objectAt(t, params, "properties"), ToolArgHeaders)
	assert.NotContains(t, params, "required")

	client.Auth = models.APIClientAuth{Type: models.APIClientAuthAPIKey, In: models.APIKeyInQuery, Name: "q"}
	params = ToolParameters(client, op)
	assert.NotContains(t, objectAt(t, params, "properties"), ToolArgQueryParams)
	assert.Contains(t, objectAt(t, params, "properties"), ToolArgHeaders)

	op.NoAuth = true
	params = ToolParameters(client, op)
	assert.Contains(t, objectAt(t, params, "properties"), ToolArgQueryParams, "an operation without auth needs the parameter")
}

func TestToolParameters_LimitsSchemaSize(t *testing.T) {
	props := make(map[string]any)
	for i := 0; i < 1000; i++ {
		props[fmt.Sprintf("field%04d", i)] = map[string]any{"$ref": "#/components/schemas/Item"}
	}
	client := &models.APIClient{Name: "Big", Schemas: map[string]map[string]any{
		"Big":  {"type": "object", "properties": props},
		"Item": {"type": "object", "properties": map[string]any{"value": map[string]any{"type": "string"}}},
	}}
	op := &models.APIOperation{ID: "big", Method: "POST", Path: "/big", RequestBody: &models.APIRequestBody{
		Content: []*models.APIMediaType{{ContentType: "application/json", Schema: map[string]any{"$ref": "#/components/schemas/Big"}}},
	}}

	body := objectAt(t, ToolParameters(client, op), "properties", ToolArgBody)
	fields := objectAt(t, body, "properties")
	assert.Less(t, len(fields), maxToolSchemaNodes)
	assert.Contains(t, fields, "field0000", "fields are kept in name order")
	assert.Contains(t, body["description"], "Some fields are omitted.")
}

func TestToolParameters_CutsDeepNesting(t *testing.T) {
	schema := map[string]any{"type": "string"}
	for i := 0; i < 20; i++ {
		schema = map[string]any{"type": "object", "properties": map[string]any{"child": schema}}
	}
	op := &models.APIOperation{ID: "deep", Method: "POST", Path: "/deep", RequestBody: &models.APIRequestBody{
		Content: []*models.APIMediaType{{ContentType: "application/json", Schema: schema}},
	}}

	current := objectAt(t, ToolParameters(&models.APIClient{}, op), "properties", ToolArgBody)
	depth := 1
	for current["properties"] != nil {
		current = objectAt(t, current, "properties", "child")
		depth++
	}
	assert.Equal(t, maxToolSchemaDepth+1, depth)
	assert.Equal(t, "object", current["type"])
}

func TestToolDescription(t *testing.T) {
	client := &models.APIClient{Name: "Pets"}
	op := &models.APIOperation{Method: "GET", Path: "/pets/{petId}", Summary: "Get a pet", Description: "Returns one pet.", Deprecated: true}
	assert.Equal(t, "Deprecated.\n\nGet a pet\n\nReturns one pet.\n\nGET /pets/{petId} (API \"Pets\")", ToolDescription(client, op))

	op = &models.APIOperation{Method: "GET", Path: "/health"}
	assert.Equal(t, `GET /health (API "Pets")`, ToolDescription(client, op))

	op = &models.APIOperation{Method: "GET", Path: "/long", Description: strings.Repeat("word ", 1000)}
	description := ToolDescription(client, op)
	assert.LessOrEqual(t, len([]rune(description)), maxToolDescription)
	assert.True(t, strings.HasSuffix(description, `GET /long (API "Pets")`))
	assert.Contains(t, description, "…")
}

func parseFixtureClient(t *testing.T, name string) *models.APIClient {
	t.Helper()
	spec := parseFixture(t, name)
	return &models.APIClient{Name: spec.Title, Operations: spec.Operations, Schemas: spec.Schemas}
}
