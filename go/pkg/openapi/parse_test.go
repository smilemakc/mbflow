package openapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func parseFixture(t *testing.T, name string) *Spec {
	t.Helper()
	spec, err := Parse(context.Background(), loadFixture(t, name))
	require.NoError(t, err)
	return spec
}

func operationByID(t *testing.T, spec *Spec, id string) *models.APIOperation {
	t.Helper()
	for _, op := range spec.Operations {
		if op.ID == id {
			return op
		}
	}
	t.Fatalf("operation %q not found", id)
	return nil
}

func paramByName(t *testing.T, op *models.APIOperation, name, in string) *models.APIParameter {
	t.Helper()
	for _, p := range op.Parameters {
		if p.Name == name && p.In == in {
			return p
		}
	}
	t.Fatalf("parameter %s in %s not found in %s", name, in, op.ID)
	return nil
}

func warningCodes(spec *Spec) []string {
	codes := make([]string, 0, len(spec.Warnings))
	for _, w := range spec.Warnings {
		codes = append(codes, w.Code)
	}
	return codes
}

func TestParse_OpenAPI30(t *testing.T) {
	spec := parseFixture(t, "petstore-3.0.yaml")

	assert.Equal(t, "Petstore", spec.Title)
	assert.Equal(t, "1.2.0", spec.APIVersion)
	assert.Equal(t, "3.0.3", spec.SpecVersion)
	assert.Equal(t, []models.APIServer{
		{URL: "https://api.petstore.test/v1", Description: "Production"},
		{URL: "/relative"},
	}, spec.Servers)
	assert.Equal(t, "https://api.petstore.test/v1", spec.DefaultBaseURL())

	ids := make([]string, 0, len(spec.Operations))
	for _, op := range spec.Operations {
		ids = append(ids, op.ID)
	}
	assert.Equal(t, []string{"get_health", "get_pets", "post_pets", "getPet", "deletePet"}, ids,
		"operations are sorted by path and method; shared operationIds fall back to method and path")

	list := operationByID(t, spec, "get_pets")
	assert.Equal(t, "listPets", list.OperationID)
	assert.Equal(t, "GET", list.Method)
	assert.Equal(t, "/pets", list.Path)
	assert.Equal(t, []string{"pets"}, list.Tags)
	assert.Equal(t, [][]string{{"api_key"}}, list.Security)
	assert.False(t, list.NoAuth)
	assert.Len(t, list.Parameters, 3, "the Accept header parameter is ignored")
	tags := paramByName(t, list, "tags", "query")
	assert.True(t, tags.Required)
	assert.Equal(t, "array", tags.Schema["type"])
	filter := paramByName(t, list, "filter", "query")
	assert.Equal(t, "deepObject", filter.Style)
	assert.Equal(t, "application/json", acceptHeader(list))

	require.Len(t, list.Responses, 2)
	assert.Equal(t, "200", list.Responses[0].Status)
	assert.Equal(t, "default", list.Responses[1].Status)
	assert.Equal(t, map[string]any{"$ref": "#/components/schemas/Error"}, list.Responses[1].Content[0].Schema)

	create := operationByID(t, spec, "post_pets")
	assert.Equal(t, "Create a pet", create.Summary)
	require.NotNil(t, create.RequestBody)
	assert.True(t, create.RequestBody.Required)
	require.Len(t, create.RequestBody.Content, 2)
	assert.Equal(t, "application/json", create.RequestBody.Content[0].ContentType, "JSON is preferred over forms")
	assert.Equal(t, map[string]any{"name": "Rex"}, create.RequestBody.Content[0].Example)
	assert.Equal(t, "application/x-www-form-urlencoded", create.RequestBody.Content[1].ContentType)

	get := operationByID(t, spec, "getPet")
	petID := paramByName(t, get, "petId", "path")
	assert.Equal(t, "integer", petID.Schema["type"], "operation parameters override path-level ones")
	assert.Equal(t, "Overridden at operation level", petID.Description)
	paramByName(t, get, "X-Trace-Id", "header")

	assert.True(t, operationByID(t, spec, "deletePet").NoAuth)

	assert.Contains(t, spec.Schemas, "Pet")
	assert.Contains(t, spec.Schemas, "NewPet")
	assert.Contains(t, spec.Schemas, "Category")
	assert.Contains(t, spec.Schemas, "Error")
	assert.NotContains(t, spec.Schemas, "Unused", "only schemas reachable from operations are kept")
	category := spec.Schemas["Category"]["properties"].(map[string]any)
	assert.Equal(t, map[string]any{"$ref": "#/components/schemas/Category"}, category["parent"],
		"recursive schemas stay references")

	assert.Equal(t, models.APIClientAuth{
		Type:       models.APIClientAuthAPIKey,
		In:         "header",
		Name:       "X-API-Key",
		SchemeName: "api_key",
	}, spec.SuggestedAuth)
	require.Len(t, spec.SecuritySchemes, 2)
	assert.Equal(t, "api_key", spec.SecuritySchemes[0].Name)
	assert.Equal(t, "X-API-Key", spec.SecuritySchemes[0].ParamName)

	assert.Contains(t, warningCodes(spec), WarningGeneratedIDs)
	assert.Contains(t, warningCodes(spec), WarningDuplicateIDs)
	assert.NotContains(t, warningCodes(spec), WarningNoServers)
}

func TestParse_Swagger20(t *testing.T) {
	spec := parseFixture(t, "legacy-swagger-2.0.json")

	assert.Equal(t, "Legacy Orders", spec.Title)
	assert.Equal(t, "2019-01", spec.APIVersion)
	assert.Equal(t, "2.0", spec.SpecVersion)
	assert.Equal(t, "https://orders.legacy.test/api", spec.DefaultBaseURL())

	create := operationByID(t, spec, "createOrder")
	require.NotNil(t, create.RequestBody)
	assert.True(t, create.RequestBody.Required)
	assert.Equal(t, "application/json", create.RequestBody.Content[0].ContentType)
	assert.Equal(t, map[string]any{"$ref": "#/components/schemas/Order"}, create.RequestBody.Content[0].Schema)
	assert.Equal(t, "boolean", paramByName(t, create, "dryRun", "query").Schema["type"])
	assert.Contains(t, spec.Schemas, "Item", "schemas referenced from other schemas are kept")

	upload := operationByID(t, spec, "uploadAttachment")
	require.NotNil(t, upload.RequestBody)
	assert.Equal(t, "multipart/form-data", upload.RequestBody.Content[0].ContentType)
	assert.True(t, paramByName(t, upload, "id", "path").Required)

	assert.Equal(t, models.APIClientAuthBasic, spec.SuggestedAuth.Type)
	assert.Equal(t, "basicAuth", spec.SuggestedAuth.SchemeName)
}

func TestParse_OpenAPI31(t *testing.T) {
	spec := parseFixture(t, "modern-3.1.json")

	assert.Equal(t, "3.1.0", spec.SpecVersion)
	assert.Equal(t, "https://modern.test/api", spec.DefaultBaseURL())

	patch := operationByID(t, spec, "patch_users_userId")
	assert.Equal(t, "Update a user", patch.Summary)
	assert.Equal(t, "application/merge-patch+json", patch.RequestBody.Content[0].ContentType)
	nickname := spec.Schemas["UserPatch"]["properties"].(map[string]any)["nickname"].(map[string]any)
	assert.Equal(t, []string{"string", "null"}, nickname["type"])

	assert.True(t, operationByID(t, spec, "status").NoAuth)

	assert.Equal(t, models.APIClientAuth{
		Type:       models.APIClientAuthOAuth2ClientCredentials,
		SchemeName: "oauth",
		TokenURL:   "https://auth.modern.test/token",
		Scopes:     []string{"users:read", "users:write"},
	}, spec.SuggestedAuth)
}

func TestParse_RejectsExternalReferences(t *testing.T) {
	for name, ref := range map[string]string{
		"local file": "/etc/passwd",
		"remote url": "http://169.254.169.254/latest/meta-data",
		"relative":   "other.yaml#/components/schemas/Pet",
	} {
		t.Run(name, func(t *testing.T) {
			doc := `openapi: 3.0.0
info: {title: T, version: "1"}
paths:
  /x:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "` + ref + `"
`
			_, err := Parse(context.Background(), []byte(doc))
			require.Error(t, err)
			assert.ErrorIs(t, err, models.ErrInvalidAPISpec)
		})
	}
}

func TestParse_InvalidDocuments(t *testing.T) {
	cases := map[string]string{
		"empty":           "   \n",
		"not a spec":      "just: yaml\nwith: keys\n",
		"unsupported":     `{"openapi": "4.0.0", "info": {"title": "x", "version": "1"}, "paths": {}}`,
		"broken json":     `{"openapi": "3.0.0",`,
		"no operations":   "openapi: 3.0.0\ninfo: {title: T, version: '1'}\npaths: {}\n",
		"swagger 1.2":     `{"swaggerVersion": "1.2", "apis": []}`,
		"invalid openapi": "openapi: 3.0.0\ninfo: [not, an, object]\npaths: {}\n",
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(context.Background(), []byte(doc))
			require.Error(t, err)
			assert.ErrorIs(t, err, models.ErrInvalidAPISpec)
		})
	}
}

func TestParse_ServerWarnings(t *testing.T) {
	noServers := "\xef\xbb\xbfopenapi: 3.0.0\ninfo: {title: T, version: '1'}\npaths:\n  /x:\n    get:\n      responses:\n        '200': {description: ok}\n"
	spec, err := Parse(context.Background(), []byte(noServers))
	require.NoError(t, err, "a UTF-8 byte order mark is accepted")
	assert.Contains(t, warningCodes(spec), WarningNoServers)
	assert.Equal(t, "", spec.DefaultBaseURL())
	assert.Equal(t, "get_x", spec.Operations[0].ID)
	assert.Equal(t, models.APIClientAuthNone, spec.SuggestedAuth.Type)

	relative := "openapi: 3.0.0\ninfo: {title: T, version: '1'}\nservers: [{url: /api/v2}]\npaths:\n  /x:\n    get:\n      responses:\n        '200': {description: ok}\n"
	spec, err = Parse(context.Background(), []byte(relative))
	require.NoError(t, err)
	assert.Contains(t, warningCodes(spec), WarningRelativeServer)
}

func TestParse_UnquotedSwaggerVersion(t *testing.T) {
	doc := "swagger: 2.0\ninfo: {title: T, version: '1'}\nhost: example.test\npaths:\n  /ping:\n    get:\n      responses:\n        200: {description: ok}\n"
	spec, err := Parse(context.Background(), []byte(doc))
	require.NoError(t, err)
	assert.Equal(t, "2.0", spec.SpecVersion)
	assert.Equal(t, "https://example.test", spec.DefaultBaseURL())
	require.Len(t, spec.Operations[0].Responses, 1)
	assert.Equal(t, "200", spec.Operations[0].Responses[0].Status)
}

func TestParse_UnsupportedSecuritySchemeWarns(t *testing.T) {
	doc := `openapi: 3.1.0
info: {title: T, version: "1"}
security: [{mtls: []}]
paths:
  /x:
    get:
      responses:
        "200": {description: ok}
components:
  securitySchemes:
    mtls:
      type: mutualTLS
`
	spec, err := Parse(context.Background(), []byte(doc))
	require.NoError(t, err)
	assert.Equal(t, models.APIClientAuthNone, spec.SuggestedAuth.Type)
	assert.Contains(t, warningCodes(spec), WarningUnsupportedAuth)
}

func TestParse_GeneratedKeysDoNotTakeExplicitOperationIDs(t *testing.T) {
	doc := `openapi: 3.0.0
info: {title: T, version: "1"}
paths:
  /x:
    get:
      responses:
        "200": {description: ok}
  /y:
    get:
      operationId: get_x
      responses:
        "200": {description: ok}
`
	spec, err := Parse(context.Background(), []byte(doc))
	require.NoError(t, err)
	assert.Equal(t, "get_x_2", spec.Operations[0].ID)
	assert.Equal(t, "get_x", spec.Operations[1].ID)
}

func TestGenerateOperationKey(t *testing.T) {
	assert.Equal(t, "get_pets_petId_photos", generateOperationKey("GET", "/pets/{petId}/photos"))
	assert.Equal(t, "delete_root", generateOperationKey("DELETE", "/"))
	assert.Equal(t, "post_v1_users_me_avatar_png", generateOperationKey("POST", "/v1/users/@me/avatar.png"))
}

func TestResolveSchema(t *testing.T) {
	schemas := map[string]map[string]any{
		"A":    {"$ref": "#/components/schemas/B"},
		"B":    {"type": "string"},
		"a/b":  {"type": "integer"},
		"Loop": {"$ref": "#/components/schemas/Loop"},
	}
	assert.Equal(t, map[string]any{"type": "string"}, ResolveSchema(schemas, map[string]any{"$ref": "#/components/schemas/A"}))
	assert.Equal(t, map[string]any{"type": "integer"}, ResolveSchema(schemas, map[string]any{"$ref": "#/components/schemas/a~1b"}))
	assert.Nil(t, ResolveSchema(schemas, map[string]any{"$ref": "#/components/schemas/Missing"}))
	assert.Nil(t, ResolveSchema(schemas, map[string]any{"$ref": "#/components/schemas/Loop"}))
	assert.Nil(t, ResolveSchema(schemas, nil))
}
