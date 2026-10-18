package openapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

func petstoreClient(t *testing.T) *models.APIClient {
	t.Helper()
	spec := parseFixture(t, "petstore-3.0.yaml")
	return &models.APIClient{
		Name:           "Petstore",
		BaseURL:        spec.DefaultBaseURL(),
		Operations:     spec.Operations,
		Schemas:        spec.Schemas,
		DefaultHeaders: map[string]string{"X-Client": "mbflow"},
	}
}

func boolPtr(b bool) *bool { return &b }

func TestBuildRequest_QueryAndHeaders(t *testing.T) {
	client := petstoreClient(t)
	op := client.FindOperation("get_pets")

	req, err := BuildRequest(client, op, RequestInput{
		QueryParams: map[string]any{
			"limit":  "10",
			"tags":   `["dog","cat"]`,
			"filter": `{"status":"available"}`,
			"extra":  "yes",
			"empty":  "",
		},
		Headers: map[string]any{"X-Request-Id": "abc"},
	})
	require.NoError(t, err)

	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "https", req.URL.Scheme)
	assert.Equal(t, "api.petstore.test", req.URL.Host)
	assert.Equal(t, "/v1/pets", req.URL.Path)
	assert.Equal(t, url.Values{
		"limit":          {"10"},
		"tags":           {"dog", "cat"},
		"filter[status]": {"available"},
		"extra":          {"yes"},
	}, req.URL.Query())
	assert.Equal(t, "mbflow", req.Header.Get("X-Client"))
	assert.Equal(t, "abc", req.Header.Get("X-Request-Id"))
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
	assert.Nil(t, req.Body)
}

func TestBuildRequest_MissingRequiredParameters(t *testing.T) {
	client := petstoreClient(t)

	_, err := BuildRequest(client, client.FindOperation("get_pets"), RequestInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required query parameters: tags")

	_, err = BuildRequest(client, client.FindOperation("getPet"), RequestInput{PathParams: map[string]any{"petId": " "}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required path parameters: petId")
}

func TestBuildRequest_ParametersSetByAuthAreNotRequired(t *testing.T) {
	op := &models.APIOperation{ID: "search", Method: "GET", Path: "/search", Parameters: []*models.APIParameter{
		{Name: "X-API-Key", In: "header", Required: true},
		{Name: "key", In: "query", Required: true},
	}}
	client := &models.APIClient{Name: "Search", BaseURL: "https://search.test", Auth: models.APIClientAuth{
		Type: models.APIClientAuthAPIKey, In: models.APIKeyInHeader, Name: "x-api-key", CredentialID: "cred",
	}}

	_, err := BuildRequest(client, op, RequestInput{QueryParams: map[string]any{"key": "k"}})
	require.NoError(t, err, "the executor sets the header from the credential")

	_, err = BuildRequest(client, op, RequestInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required query parameters: key")

	op.NoAuth = true
	_, err = BuildRequest(client, op, RequestInput{QueryParams: map[string]any{"key": "k"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required header parameters: X-API-Key", "an operation without auth gets no key")
}

func TestBuildRequest_PathParameterIsEscaped(t *testing.T) {
	client := petstoreClient(t)
	req, err := BuildRequest(client, client.FindOperation("getPet"), RequestInput{
		PathParams: map[string]any{"petId": "a/b c"},
	})
	require.NoError(t, err)
	assert.Equal(t, "https://api.petstore.test/v1/pets/a%2Fb%20c", req.URL.String())
}

func TestSerializePathParam_Styles(t *testing.T) {
	list := []any{"3", "4", "5"}
	object := map[string]any{"role": "admin", "first": "Alex"}
	cases := []struct {
		style   string
		explode bool
		value   any
		want    string
	}{
		{"", false, "5", "5"},
		{"simple", false, list, "3,4,5"},
		{"simple", false, object, "first,Alex,role,admin"},
		{"simple", true, object, "first=Alex,role=admin"},
		{"label", false, "5", ".5"},
		{"label", false, list, ".3,4,5"},
		{"label", true, list, ".3.4.5"},
		{"label", true, object, ".first=Alex.role=admin"},
		{"matrix", false, "5", ";id=5"},
		{"matrix", false, list, ";id=3,4,5"},
		{"matrix", true, list, ";id=3;id=4;id=5"},
		{"matrix", true, object, ";first=Alex;role=admin"},
	}
	for _, tc := range cases {
		p := &models.APIParameter{Name: "id", In: "path", Style: tc.style, Explode: boolPtr(tc.explode)}
		assert.Equal(t, tc.want, serializePathParam(p, tc.value), "style=%s explode=%v value=%v", tc.style, tc.explode, tc.value)
	}
}

func TestAddQueryParams_Styles(t *testing.T) {
	op := &models.APIOperation{Parameters: []*models.APIParameter{
		{Name: "csv", In: "query", Explode: boolPtr(false), Schema: map[string]any{"type": "array"}},
		{Name: "space", In: "query", Style: "spaceDelimited", Schema: map[string]any{"type": "array"}},
		{Name: "pipe", In: "query", Style: "pipeDelimited", Schema: map[string]any{"type": "array"}},
		{Name: "obj", In: "query", Schema: map[string]any{"type": "object"}},
		{Name: "flat", In: "query", Explode: boolPtr(false), Schema: map[string]any{"type": "object"}},
		{Name: "json", In: "query", ContentType: "application/json", Schema: map[string]any{"type": "object"}},
	}}
	query := url.Values{}
	err := addQueryParams(query, op, map[string]any{
		"csv":   "a, b ,c",
		"space": []any{"x", "y"},
		"pipe":  []any{"x", "y"},
		"obj":   map[string]any{"k": "v"},
		"flat":  `{"a":1,"b":2}`,
		"json":  `{"q":"z"}`,
	}, nil, func(*models.APIParameter) bool { return false })
	require.NoError(t, err)
	assert.Equal(t, url.Values{
		"csv":   {"a,b,c"},
		"space": {"x y"},
		"pipe":  {"x|y"},
		"k":     {"v"},
		"flat":  {"a,1,b,2"},
		"json":  {`{"q":"z"}`},
	}, query)
}

func TestBuildRequest_JSONBodyIsCoercedBySchema(t *testing.T) {
	client := petstoreClient(t)
	req, err := BuildRequest(client, client.FindOperation("post_pets"), RequestInput{
		Body: map[string]any{
			"name":       "Rex",
			"age":        "3",
			"vaccinated": "true",
			"tags":       `["good","boy"]`,
			"category":   map[string]any{"name": "dogs", "parent": map[string]any{"name": "animals"}},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.JSONEq(t,
		`{"name":"Rex","age":3,"vaccinated":true,"tags":["good","boy"],"category":{"name":"dogs","parent":{"name":"animals"}}}`,
		string(req.Body))
}

func TestBuildRequest_RawJSONBodyIsSentAsIs(t *testing.T) {
	client := petstoreClient(t)
	req, err := BuildRequest(client, client.FindOperation("post_pets"), RequestInput{
		Body: ` {"name": "Rex", "age": "3"} `,
	})
	require.NoError(t, err)
	assert.Equal(t, `{"name": "Rex", "age": "3"}`, string(req.Body))
}

func TestBuildRequest_FormBody(t *testing.T) {
	client := petstoreClient(t)
	req, err := BuildRequest(client, client.FindOperation("post_pets"), RequestInput{
		ContentType: "application/x-www-form-urlencoded",
		Body:        map[string]any{"name": "Rex", "tags": `["a","b"]`},
	})
	require.NoError(t, err)
	assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
	values, err := url.ParseQuery(string(req.Body))
	require.NoError(t, err)
	assert.Equal(t, url.Values{"name": {"Rex"}, "tags": {"a", "b"}}, values)
}

func TestBuildRequest_RequiredBody(t *testing.T) {
	client := petstoreClient(t)
	op := client.FindOperation("post_pets")

	_, err := BuildRequest(client, op, RequestInput{Body: "  "})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request body is required")

	req, err := BuildRequest(client, op, RequestInput{Body: map[string]any{}})
	require.NoError(t, err)
	assert.Equal(t, "{}", string(req.Body), "an empty object still satisfies a required body")
}

func TestBuildRequest_MultipartWithFile(t *testing.T) {
	spec := parseFixture(t, "legacy-swagger-2.0.json")
	client := &models.APIClient{Name: "Legacy", BaseURL: spec.DefaultBaseURL(), Operations: spec.Operations, Schemas: spec.Schemas}
	fileJSON, err := json.Marshal(map[string]any{
		"body_base64":  base64.StdEncoding.EncodeToString([]byte("PDF-DATA")),
		"content_type": "application/pdf",
		"filename":     "invoice.pdf",
	})
	require.NoError(t, err)

	req, err := BuildRequest(client, client.FindOperation("uploadAttachment"), RequestInput{
		PathParams: map[string]any{"id": "42"},
		Body:       map[string]any{"file": string(fileJSON), "comment": "see attached"},
	})
	require.NoError(t, err)
	assert.Equal(t, "https://orders.legacy.test/api/orders/42/attachments", req.URL.String())

	mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	require.NoError(t, err)
	assert.Equal(t, "multipart/form-data", mediaType)

	reader := multipart.NewReader(bytes.NewReader(req.Body), params["boundary"])
	parts := map[string]*multipart.Part{}
	contents := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		data, err := io.ReadAll(part)
		require.NoError(t, err)
		parts[part.FormName()] = part
		contents[part.FormName()] = string(data)
	}
	assert.Equal(t, "see attached", contents["comment"])
	assert.Equal(t, "PDF-DATA", contents["file"])
	assert.Equal(t, "invoice.pdf", parts["file"].FileName())
	assert.Equal(t, "application/pdf", parts["file"].Header.Get("Content-Type"))
}

func TestBuildRequest_BaseURLPrecedence(t *testing.T) {
	client := &models.APIClient{Name: "API", BaseURL: "https://client.test/v1/"}
	op := &models.APIOperation{Method: "GET", Path: "/items"}

	req, err := BuildRequest(client, op, RequestInput{})
	require.NoError(t, err)
	assert.Equal(t, "https://client.test/v1/items", req.URL.String())

	op.Servers = []models.APIServer{{URL: "/relative"}, {URL: "https://uploads.test"}}
	req, err = BuildRequest(client, op, RequestInput{})
	require.NoError(t, err)
	assert.Equal(t, "https://uploads.test/items", req.URL.String(), "an absolute operation server wins over the client")

	req, err = BuildRequest(client, op, RequestInput{BaseURL: "http://staging.test"})
	require.NoError(t, err)
	assert.Equal(t, "http://staging.test/items", req.URL.String(), "the node override wins over everything")

	_, err = BuildRequest(&models.APIClient{Name: "API"}, &models.APIOperation{Method: "GET", Path: "/x"}, RequestInput{})
	assert.ErrorIs(t, err, ErrBaseURLRequired)

	_, err = BuildRequest(client, op, RequestInput{BaseURL: "ftp://files.test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "absolute http or https URL")
}

func TestBuildRequest_CookiesAndHeaderParams(t *testing.T) {
	op := &models.APIOperation{
		Method: "GET",
		Path:   "/session",
		Parameters: []*models.APIParameter{
			{Name: "session", In: "cookie", Required: true},
			{Name: "X-Ids", In: "header", Schema: map[string]any{"type": "array"}},
		},
	}
	client := &models.APIClient{Name: "API", BaseURL: "https://api.test"}

	_, err := BuildRequest(client, op, RequestInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required cookie parameters: session")

	req, err := BuildRequest(client, op, RequestInput{
		Cookies: map[string]any{"session": "s 1", "theme": "dark"},
		Headers: map[string]any{"x-ids": "1,2"},
	})
	require.NoError(t, err)
	assert.Equal(t, "session=s+1; theme=dark", req.Header.Get("Cookie"))
	assert.Equal(t, "1,2", req.Header.Get("X-Ids"), "declared headers match case-insensitively")
}

func TestParseAndBuild_EndToEnd(t *testing.T) {
	spec, err := Parse(context.Background(), loadFixture(t, "modern-3.1.json"))
	require.NoError(t, err)
	client := &models.APIClient{Name: "Modern", BaseURL: spec.DefaultBaseURL(), Operations: spec.Operations, Schemas: spec.Schemas}

	req, err := BuildRequest(client, client.FindOperation("patch_users_userId"), RequestInput{
		PathParams: map[string]any{"userId": "u-1"},
		Body:       map[string]any{"nickname": "neo", "score": "9.5"},
	})
	require.NoError(t, err)
	assert.Equal(t, "PATCH", req.Method)
	assert.Equal(t, "https://modern.test/api/users/u-1", req.URL.String())
	assert.Equal(t, "application/merge-patch+json", req.Header.Get("Content-Type"))
	assert.JSONEq(t, `{"nickname":"neo","score":9.5}`, string(req.Body))
}
