package openapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

const rcRichSpec = `
openapi: 3.0.3
info:
  title: Rich
  version: "1"
servers:
  - url: https://{env}.example.com/api/
    variables:
      env:
        default: prod
components:
  securitySchemes:
    oidc:
      type: openIdConnect
      openIdConnectUrl: https://id.example.com/.well-known/openid-configuration
  schemas:
    Alias:
      $ref: '#/components/schemas/Target'
    Target:
      type: object
      properties:
        id:
          type: integer
          minimum: 1
          maximum: 10
          multipleOf: 1
        tags:
          type: array
          minItems: 1
          maxItems: 3
          uniqueItems: true
          items:
            type: string
            minLength: 1
            maxLength: 5
            pattern: "^[a-z]+$"
        extra:
          type: object
          additionalProperties:
            type: string
        open:
          type: object
          additionalProperties: true
        kind:
          type: string
          enum: [a, b]
          default: a
          deprecated: true
          readOnly: true
        secret:
          type: string
          writeOnly: true
          nullable: true
          format: password
          example: hunter2
      discriminator:
        propertyName: kind
paths:
  /things:
    get:
      operationId: listThings
      security:
        - oidc: []
      parameters:
        - name: filter
          in: query
          content:
            application/json:
              schema:
                type: object
                properties:
                  q:
                    type: string
        - name: sort
          in: query
          schema:
            type: string
          examples:
            byName:
              value: name
      responses:
        "2XX":
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Alias'
        default:
          description: error
        "200":
          description: ok
          content:
            application/vnd.api+json:
              schema:
                type: object
    post:
      operationId: createThing
      requestBody:
        content: {}
      responses:
        "201":
          description: created
`

func TestRCParse_RichSpec(t *testing.T) {
	spec, err := Parse(context.Background(), []byte(rcRichSpec))
	require.NoError(t, err)

	assert.Equal(t, "https://prod.example.com/api", spec.DefaultBaseURL())
	assert.Equal(t, models.APIClientAuthBearer, spec.SuggestedAuth.Type)

	list := operationByID(t, spec, "listThings")
	filter := paramByName(t, list, "filter", "query")
	assert.Equal(t, "application/json", filter.ContentType)
	assert.Equal(t, "object", filter.Schema["type"])
	sort := paramByName(t, list, "sort", "query")
	assert.Equal(t, "name", sort.Example)

	require.Len(t, list.Responses, 3)
	assert.Equal(t, []string{"200", "2XX", "default"}, []string{list.Responses[0].Status, list.Responses[1].Status, list.Responses[2].Status})

	create := operationByID(t, spec, "createThing")
	require.NotNil(t, create.RequestBody)
	assert.Equal(t, "application/json", create.RequestBody.Content[0].ContentType)

	// The alias component points at Target, which is converted fully.
	assert.Equal(t, map[string]any{"$ref": "#/components/schemas/Target"}, spec.Schemas["Alias"])
	target := spec.Schemas["Target"]
	require.NotNil(t, target)
	props := target["properties"].(map[string]any)
	id := props["id"].(map[string]any)
	assert.Equal(t, 1.0, id["minimum"])
	assert.Equal(t, 10.0, id["maximum"])
	assert.Equal(t, 1.0, id["multipleOf"])
	tags := props["tags"].(map[string]any)
	assert.Equal(t, true, tags["uniqueItems"])
	assert.EqualValues(t, 1, tags["minItems"])
	assert.EqualValues(t, 3, tags["maxItems"])
	items := tags["items"].(map[string]any)
	assert.EqualValues(t, 1, items["minLength"])
	assert.EqualValues(t, 5, items["maxLength"])
	assert.Equal(t, "^[a-z]+$", items["pattern"])
	assert.Equal(t, map[string]any{"type": "string"}, props["extra"].(map[string]any)["additionalProperties"])
	assert.Equal(t, true, props["open"].(map[string]any)["additionalProperties"])
	kind := props["kind"].(map[string]any)
	assert.Equal(t, true, kind["deprecated"])
	assert.Equal(t, true, kind["readOnly"])
	assert.Equal(t, "a", kind["default"])
	secret := props["secret"].(map[string]any)
	assert.Equal(t, true, secret["writeOnly"])
	assert.Equal(t, true, secret["nullable"])
	assert.Equal(t, "hunter2", secret["example"])
	assert.Equal(t, map[string]any{"propertyName": "kind"}, target["discriminator"])
}

func TestRCAuthFromScheme(t *testing.T) {
	tests := []struct {
		s    models.APISecurityScheme
		want models.APIClientAuthType
		ok   bool
	}{
		{models.APISecurityScheme{Type: "apiKey", In: "header", ParamName: "X-Key"}, models.APIClientAuthAPIKey, true},
		{models.APISecurityScheme{Type: "apiKey", In: "header"}, models.APIClientAuthAPIKey, false},
		{models.APISecurityScheme{Type: "apiKey", In: "body", ParamName: "k"}, "", false},
		{models.APISecurityScheme{Type: "http", Scheme: "basic"}, models.APIClientAuthBasic, true},
		{models.APISecurityScheme{Type: "http", Scheme: "bearer"}, models.APIClientAuthBearer, true},
		{models.APISecurityScheme{Type: "http", Scheme: "digest"}, "", false},
		{models.APISecurityScheme{Type: "oauth2", TokenURL: "https://t"}, models.APIClientAuthOAuth2ClientCredentials, true},
		{models.APISecurityScheme{Type: "oauth2"}, models.APIClientAuthBearer, true},
		{models.APISecurityScheme{Type: "openIdConnect"}, models.APIClientAuthBearer, true},
		{models.APISecurityScheme{Type: "mutualTLS"}, "", false},
	}
	for _, tt := range tests {
		got, ok := authFromScheme(&tt.s)
		assert.Equal(t, tt.ok, ok, tt.s)
		assert.Equal(t, tt.want, got.Type, tt.s)
	}
}

func TestRCSmallHelpers(t *testing.T) {
	assert.Equal(t, ToolArgPathParams, ToolArgKey("path"))
	assert.Equal(t, ToolArgQueryParams, ToolArgKey("query"))
	assert.Equal(t, ToolArgHeaders, ToolArgKey("header"))
	assert.Equal(t, ToolArgCookies, ToolArgKey("cookie"))
	assert.Equal(t, "", ToolArgKey("body"))

	assert.Equal(t, 0, contentTypeRank("application/json; charset=utf-8"))
	assert.Equal(t, 1, contentTypeRank("application/problem+json"))
	assert.Equal(t, 2, contentTypeRank("application/x-www-form-urlencoded"))
	assert.Equal(t, 3, contentTypeRank("multipart/form-data"))
	assert.Equal(t, 4, contentTypeRank("text/plain"))
	assert.Equal(t, 5, contentTypeRank("application/octet-stream"))

	assert.Equal(t, 404, statusRank("404"))
	assert.Equal(t, 1200, statusRank("2xx"))
	assert.Equal(t, 10000, statusRank("default"))
	assert.Equal(t, 10000, statusRank("9XX"))

	assert.Equal(t, "text/plain", baseMediaType("Text/Plain; charset=utf-8"))
	assert.Equal(t, "bad", baseMediaType(" BAD ;;=x"))
	assert.Equal(t, "", baseMediaType(""))

	assert.Equal(t, "", truncateText("abc", 0))
	assert.Equal(t, "abc", truncateText("abc", 3))
	assert.Equal(t, "ab…", truncateText("abcdef", 3))

	assert.Equal(t, "x", scalarText("x"))
	assert.Equal(t, "true", scalarText(true))
	assert.Equal(t, "3", scalarText(3))
	assert.Equal(t, "", scalarText(map[string]any{}))
	assert.True(t, isScalar("s"))
	assert.True(t, isScalar(false))
	assert.True(t, isScalar(uint64(2)))
	assert.False(t, isScalar([]any{}))

	assert.Equal(t, "string", toolSchemaType(map[string]any{"type": "string"}))
	assert.Equal(t, "integer", toolSchemaType(map[string]any{"type": []string{"null", "integer"}}))
	assert.Equal(t, "boolean", toolSchemaType(map[string]any{"type": []any{"null", "boolean", 3}}))
	assert.Equal(t, "null", toolSchemaType(map[string]any{"type": []any{"null"}}))
	assert.Equal(t, "", toolSchemaType(map[string]any{}))
	assert.Equal(t, "", firstNonNull(nil))

	assert.Equal(t, "integer", schemaType(map[string]any{"type": []string{"null", "integer"}}))
	assert.Equal(t, "object", schemaType(map[string]any{"properties": map[string]any{}}))
	assert.Equal(t, "array", schemaType(map[string]any{"items": map[string]any{}}))
	assert.Equal(t, "", schemaType(nil))

	assert.Equal(t, "", stringValue(nil))
	assert.Equal(t, "12", stringValue(json.Number("12")))
	assert.Equal(t, `{"a":1}`, stringValue(map[string]any{"a": 1}))
	assert.Equal(t, "s", jsonString("s"))
	assert.Contains(t, jsonString(make(chan int)), "0x")

	assert.True(t, isEmptyValue(nil))
	assert.True(t, isEmptyValue(" "))
	assert.True(t, isEmptyValue([]any{}))
	assert.False(t, isEmptyValue(0))

	v, ok := lookupFold(map[string]any{"X-Token": "t"}, "x-token")
	assert.True(t, ok)
	assert.Equal(t, "t", v)
	_, ok = lookupFold(map[string]any{}, "x")
	assert.False(t, ok)

	assert.Nil(t, findParam(&models.APIOperation{Parameters: []*models.APIParameter{{Name: "a", In: "query"}}}, "a", "header"))

	out := map[string]any{}
	setBool(out, "x", false)
	assert.Empty(t, out)
}

func TestRCCutToolSchema(t *testing.T) {
	assert.Equal(t, map[string]any{"type": "object", "description": "Obj. Nested fields are omitted."},
		cutToolSchema(map[string]any{"properties": map[string]any{}, "description": "Obj."}))
	assert.Equal(t, map[string]any{"type": "array", "items": map[string]any{}, "description": "Item structure is omitted."},
		cutToolSchema(map[string]any{"items": map[string]any{}}))
	assert.Equal(t, map[string]any{"type": "string"}, cutToolSchema(map[string]any{"type": "string"}))
	assert.Equal(t, map[string]any{}, cutToolSchema(map[string]any{}))
}

func TestRCSerializeSimple(t *testing.T) {
	assert.Equal(t, "a,1", serializeSimple([]any{"a", 1}, false))
	assert.Equal(t, "k1,v1,k2,2", serializeSimple(map[string]any{"k2": 2, "k1": "v1"}, false))
	assert.Equal(t, "k1=v1,k2=2", serializeSimple(map[string]any{"k2": 2, "k1": "v1"}, true))
	assert.Equal(t, "x", serializeSimple("x", true))
}

func TestRCCoerceAlternatives(t *testing.T) {
	schemas := map[string]map[string]any{
		"Num": {"type": "integer"},
		"Obj": {"properties": map[string]any{"n": map[string]any{"type": "integer"}}},
	}
	// A string alternative keeps strings untouched.
	assert.Equal(t, "5", Coerce("5", map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "integer"}}}, schemas))
	// Otherwise the first converting alternative wins.
	assert.Equal(t, json.Number("5"), Coerce("5", map[string]any{"anyOf": []any{map[string]any{"$ref": "#/components/schemas/Num"}}}, schemas))
	// Objects use an object (or property-bearing) alternative.
	got := Coerce(map[string]any{"n": "7"}, map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"$ref": "#/components/schemas/Obj"}}}, schemas)
	assert.Equal(t, json.Number("7"), got.(map[string]any)["n"])
	// Arrays use an array alternative.
	arr := Coerce([]any{"1"}, map[string]any{"anyOf": []any{map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}}}, schemas)
	assert.Equal(t, json.Number("1"), arr.([]any)[0])
	// No matching alternative leaves values alone; dangling refs are dropped.
	assert.Equal(t, true, Coerce(true, map[string]any{"oneOf": []any{map[string]any{"$ref": "#/components/schemas/Missing"}}}, schemas))
	assert.Equal(t, "zz", Coerce("zz", map[string]any{"oneOf": []any{map[string]any{"type": "integer"}}}, schemas))
	assert.Equal(t, map[string]any{"a": 1}, Coerce(map[string]any{"a": 1}, map[string]any{"oneOf": []any{map[string]any{"type": "integer"}}}, schemas))
	assert.Equal(t, []any{1}, Coerce([]any{1}, map[string]any{"oneOf": []any{map[string]any{"type": "integer"}}}, schemas))
}

func TestRCFlattenAllOf(t *testing.T) {
	schemas := map[string]map[string]any{
		"Base": {"type": "object", "properties": map[string]any{"id": map[string]any{"type": "integer"}, "name": map[string]any{"type": "integer"}}},
	}
	merged := flattenAllOf(map[string]any{
		"properties": map[string]any{"name": map[string]any{"type": "string"}},
		"allOf": []any{
			map[string]any{"$ref": "#/components/schemas/Base"},
			map[string]any{"$ref": "#/components/schemas/Missing"},
			"not-a-map",
		},
	}, schemas, 0)
	assert.Equal(t, "object", merged["type"])
	props := merged["properties"].(map[string]any)
	assert.Equal(t, "string", props["name"].(map[string]any)["type"], "own properties win")
	assert.Contains(t, props, "id")
	assert.NotContains(t, merged, "allOf")

	deep := map[string]any{"allOf": []any{map[string]any{}}}
	assert.Equal(t, deep, flattenAllOf(deep, schemas, maxCoerceDepth+1))
}

func rcOp(contentType string, schema map[string]any, required bool) *models.APIOperation {
	return &models.APIOperation{Method: "POST", Path: "/x", RequestBody: &models.APIRequestBody{
		Required: required,
		Content:  []*models.APIMediaType{{ContentType: contentType, Schema: schema}},
	}}
}

func TestRCEncodeBody(t *testing.T) {
	// Required body missing.
	_, _, err := encodeBody(rcOp("application/json", nil, true), nil, "", nil)
	assert.ErrorContains(t, err, "request body is required")

	// Raw bodies.
	data, ct, err := encodeBody(rcOp("text/plain", nil, false), "hello", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
	assert.Equal(t, "text/plain", ct)

	data, _, err = encodeBody(rcOp("application/octet-stream", nil, false), []byte{1, 2}, "", nil)
	require.NoError(t, err)
	assert.Equal(t, []byte{1, 2}, data)

	file := map[string]any{"body_base64": base64.StdEncoding.EncodeToString([]byte("bin"))}
	data, _, err = encodeBody(rcOp("application/octet-stream", nil, false), file, "", nil)
	require.NoError(t, err)
	assert.Equal(t, "bin", string(data))

	data, _, err = encodeBody(rcOp("text/csv", nil, false), map[string]any{"a": 1}, "", nil)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, string(data))

	_, _, err = encodeBody(rcOp("application/xml", nil, false), []any{make(chan int)}, "", nil)
	assert.ErrorContains(t, err, "failed to encode body")

	// Wildcard content types are sent as JSON.
	data, ct, err = encodeBody(rcOp("*/*", nil, false), map[string]any{"a": 1}, "", nil)
	require.NoError(t, err)
	assert.Equal(t, "application/json", ct)
	assert.JSONEq(t, `{"a":1}`, string(data))

	// JSON: raw bytes, valid JSON strings, plain strings, unencodable values.
	data, _, _ = encodeBody(rcOp("application/json", nil, false), []byte(`{"x":1}`), "", nil)
	assert.Equal(t, `{"x":1}`, string(data))
	data, _, _ = encodeBody(rcOp("application/json", nil, false), ` {"x":1} `, "", nil)
	assert.Equal(t, `{"x":1}`, string(data))
	data, _, _ = encodeBody(rcOp("application/json", nil, false), "plain", "", nil)
	assert.Equal(t, `"plain"`, string(data))
	_, _, err = encodeBody(rcOp("application/json", nil, false), []any{make(chan int)}, "", nil)
	assert.ErrorContains(t, err, "failed to encode JSON body")

	// Form bodies.
	data, _, err = encodeBody(rcOp("application/x-www-form-urlencoded", nil, false), "a=1", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "a=1", string(data))
	data, _, err = encodeBody(rcOp("application/x-www-form-urlencoded", nil, false), map[string]any{"a": []any{"x", "y"}, "b": nil, "c": 2}, "", nil)
	require.NoError(t, err)
	assert.Equal(t, "a=x&a=y&c=2", string(data))
	_, _, err = encodeBody(rcOp("application/x-www-form-urlencoded", nil, false), []any{1}, "", nil)
	assert.ErrorContains(t, err, "form body must be an object")

	// Override not declared by the operation keeps no schema.
	_, ct, err = encodeBody(rcOp("application/json", nil, false), "x", "text/plain", nil)
	require.NoError(t, err)
	assert.Equal(t, "text/plain", ct)
	// No request body declared: override or JSON default.
	_, ct, _ = encodeBody(&models.APIOperation{}, "x", "text/plain", nil)
	assert.Equal(t, "text/plain", ct)
	_, ct, _ = encodeBody(&models.APIOperation{}, map[string]any{"a": 1}, "", nil)
	assert.Equal(t, "application/json", ct)
}

func rcReadMultipart(t *testing.T, data []byte, ct string) map[string][]*multipart.Part {
	t.Helper()
	_, params, err := mime.ParseMediaType(ct)
	require.NoError(t, err)
	r := multipart.NewReader(bytes.NewReader(data), params["boundary"])
	parts := map[string][]*multipart.Part{}
	bodies := map[*multipart.Part][]byte{}
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		b, _ := io.ReadAll(p)
		bodies[p] = b
		parts[p.FormName()] = append(parts[p.FormName()], p)
	}
	rcPartBodies = bodies
	return parts
}

var rcPartBodies map[*multipart.Part][]byte

func TestRCEncodeMultipart(t *testing.T) {
	schemas := map[string]map[string]any{
		"Upload": {"type": "object", "properties": map[string]any{
			"doc":   map[string]any{"type": "string", "format": "binary"},
			"files": map[string]any{"type": "array", "items": map[string]any{"type": "string", "format": "binary"}},
			"count": map[string]any{"type": "integer"},
		}},
	}
	op := rcOp("multipart/form-data", map[string]any{"$ref": "#/components/schemas/Upload"}, false)
	encoded := base64.StdEncoding.EncodeToString([]byte("PDF"))
	fileJSON, _ := json.Marshal(map[string]any{"content_base64": encoded, "file_name": "a.pdf", "mime_type": "application/pdf"})

	body := map[string]any{
		"doc":   "raw-binary",
		"files": []any{string(fileJSON), "inline", map[string]any{"base64": "!!notbase64"}},
		"count": "3",
		"skip":  nil,
		"meta":  map[string]any{"data_base64": encoded, "name": "m.bin", "content_type": "x/y"},
	}
	data, ct, err := encodeBody(op, body, "", schemas)
	require.NoError(t, err)
	parts := rcReadMultipart(t, data, ct)

	require.Len(t, parts["doc"], 1)
	assert.Equal(t, "doc", parts["doc"][0].FileName())
	assert.Equal(t, "raw-binary", string(rcPartBodies[parts["doc"][0]]))

	require.Len(t, parts["files"], 3)
	assert.Equal(t, "a.pdf", parts["files"][0].FileName())
	assert.Equal(t, "application/pdf", parts["files"][0].Header.Get("Content-Type"))
	assert.Equal(t, "PDF", string(rcPartBodies[parts["files"][0]]))
	assert.Equal(t, "files", parts["files"][1].FileName(), "binary item strings become files")
	assert.Equal(t, "", parts["files"][2].FileName(), "invalid base64 falls back to a text field")

	assert.Equal(t, "3", string(rcPartBodies[parts["count"][0]]))
	assert.NotContains(t, parts, "skip")
	assert.Equal(t, "m.bin", parts["meta"][0].FileName())
	assert.Equal(t, "x/y", parts["meta"][0].Header.Get("Content-Type"))

	// JSON text bodies are decoded; other types are rejected.
	data, ct, err = encodeBody(op, `{"count": 1}`, "", schemas)
	require.NoError(t, err)
	assert.Equal(t, "1", string(rcPartBodies[rcReadMultipart(t, data, ct)["count"][0]]))
	_, _, err = encodeBody(op, "not json", "", schemas)
	assert.ErrorContains(t, err, "multipart body must be an object")
	_, _, err = encodeBody(op, 42, "", schemas)
	assert.ErrorContains(t, err, "multipart body must be an object")
}

func TestRCBuildRequest_HeadersAndAccept(t *testing.T) {
	client := &models.APIClient{Name: "c", BaseURL: "https://api.example.com/", DefaultHeaders: map[string]string{"X-Default": "d"}}
	op := &models.APIOperation{
		Method: "GET", Path: "/items",
		Parameters: []*models.APIParameter{
			{Name: "X-Ids", In: "header", Schema: map[string]any{"type": "array"}},
			{Name: "X-Obj", In: "header", Schema: map[string]any{"type": "object"}},
		},
		Responses: []*models.APIResponse{
			{Status: "200", Content: []*models.APIMediaType{{ContentType: "application/json"}, {ContentType: "application/json"}}},
			{Status: "201", Content: []*models.APIMediaType{{ContentType: "text/csv"}}},
			{Status: "404", Content: []*models.APIMediaType{{ContentType: "application/problem+json"}}},
		},
	}
	req, err := BuildRequest(client, op, RequestInput{Headers: map[string]any{
		"X-Ids": "1, 2,3", "X-Obj": `{"b":2,"a":1}`,
	}})
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/items", req.URL.String())
	assert.Equal(t, "d", req.Header.Get("X-Default"))
	assert.Equal(t, "1,2,3", req.Header.Get("X-Ids"))
	assert.Equal(t, "a,1,b,2", req.Header.Get("X-Obj"))
	assert.Equal(t, "application/json, text/csv", req.Header.Get("Accept"))

	_, err = BuildRequest(&models.APIClient{Name: "c"}, op, RequestInput{})
	assert.ErrorIs(t, err, ErrBaseURLRequired)
	_, err = BuildRequest(client, op, RequestInput{BaseURL: "/relative"})
	assert.ErrorContains(t, err, "absolute http or https URL")
}

func TestRCWithStringVersionAndWarn(t *testing.T) {
	out, err := withStringVersion([]byte("swagger: 2.0\ninfo: {title: t}\n"), specVersionInfo{swagger: "2.0"})
	require.NoError(t, err)
	assert.Contains(t, string(out), `"swagger":"2.0"`)
	_, err = withStringVersion([]byte(":\t- bad: ["), specVersionInfo{})
	assert.Error(t, err)

	b := &specBuilder{spec: &Spec{}}
	b.warn("CODE", strings.Repeat("x", maxWarningLength+10))
	require.Len(t, b.spec.Warnings, 1)
	assert.True(t, strings.HasSuffix(b.spec.Warnings[0].Message, "…"))
	assert.Len(t, []rune(b.spec.Warnings[0].Message), maxWarningLength+1)
}
