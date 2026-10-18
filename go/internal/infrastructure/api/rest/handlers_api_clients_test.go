package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/apiclient"
	"github.com/smilemakc/mbflow/go/internal/config"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/logger"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

const handlerTestSpec = `openapi: 3.0.3
info:
  title: Weather API
  version: "2.1"
servers:
  - url: https://weather.test/api
paths:
  /forecast/{city}:
    get:
      operationId: getForecast
      summary: Forecast for a city
      tags: [forecast]
      parameters:
        - {name: city, in: path, required: true, schema: {type: string}}
        - {name: days, in: query, schema: {type: integer}}
      responses:
        "200": {description: ok}
  /alerts:
    post:
      operationId: createAlert
      requestBody:
        content:
          application/json:
            schema: {$ref: "#/components/schemas/Alert"}
      responses:
        "201": {description: created}
components:
  schemas:
    Alert:
      type: object
      properties:
        city: {type: string}
`

type apiClientTestEnv struct {
	router  *gin.Engine
	userID  string
	userIDs map[string]string
}

func setupAPIClientHandlersTest(t *testing.T) *apiClientTestEnv {
	t.Helper()
	db, _ := testutil.SetupTestTx(t)
	userRepo := storage.NewUserRepository(db)
	createUser := func(name string) string {
		id := uuid.New()
		require.NoError(t, userRepo.Create(context.Background(), &storagemodels.UserModel{
			ID: id, Email: name + "@test.local", Username: name + "-" + id.String()[:8], PasswordHash: "x", IsActive: true,
		}))
		return id.String()
	}
	env := &apiClientTestEnv{userIDs: map[string]string{"alice": createUser("alice"), "bob": createUser("bob")}}

	service := apiclient.NewService(storage.NewAPIClientRepository(db), nil)
	handlers := NewAPIClientHandlers(service, logger.New(config.LoggingConfig{Level: "error", Format: "text"}))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1/api-clients")
	api.Use(func(c *gin.Context) {
		if user := c.GetHeader("X-Test-User"); user != "" {
			c.Set(ContextKeyUserID, env.userIDs[user])
		}
		c.Next()
	})
	api.POST("/preview", handlers.HandlePreview)
	api.POST("", handlers.HandleCreate)
	api.GET("", handlers.HandleList)
	api.GET("/:id", handlers.HandleGet)
	api.PUT("/:id", handlers.HandleUpdate)
	api.DELETE("/:id", handlers.HandleDelete)
	api.GET("/:id/spec", handlers.HandleDownloadSpec)
	api.PUT("/:id/spec", handlers.HandleReplaceSpec)
	env.router = router
	return env
}

func (e *apiClientTestEnv) do(t *testing.T, user, method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if user != "" {
		req.Header.Set("X-Test-User", user)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func multipartSpec(t *testing.T, spec string, fields map[string]string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "weather.yaml")
	require.NoError(t, err)
	_, err = part.Write([]byte(spec))
	require.NoError(t, err)
	for k, v := range fields {
		require.NoError(t, writer.WriteField(k, v))
	}
	require.NoError(t, writer.Close())
	return buf.Bytes(), writer.FormDataContentType()
}

func decodeBody[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v), w.Body.String())
	return v
}

func TestAPIClientHandlers_ImportViaMultipart(t *testing.T) {
	t.Parallel()
	env := setupAPIClientHandlersTest(t)

	body, contentType := multipartSpec(t, handlerTestSpec, map[string]string{
		"name":            "Weather",
		"default_headers": `{"X-Source": "mbflow"}`,
	})
	w := env.do(t, "alice", http.MethodPost, "/api/v1/api-clients", contentType, body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	client := decodeBody[models.APIClient](t, w)
	assert.Equal(t, "Weather", client.Name)
	assert.Equal(t, "Weather API", client.Title)
	assert.Equal(t, "https://weather.test/api", client.BaseURL)
	assert.Equal(t, map[string]string{"X-Source": "mbflow"}, client.DefaultHeaders)
	require.Len(t, client.Operations, 2)
	assert.Contains(t, client.Schemas, "Alert")
	assert.NotContains(t, w.Body.String(), "spec_raw", "the original document is not part of the client JSON")

	got := env.do(t, "alice", http.MethodGet, "/api/v1/api-clients/"+client.ID, "", nil)
	require.Equal(t, http.StatusOK, got.Code)
	assert.Equal(t, client.ID, decodeBody[models.APIClient](t, got).ID)

	list := env.do(t, "alice", http.MethodGet, "/api/v1/api-clients", "", nil)
	require.Equal(t, http.StatusOK, list.Code)
	page := decodeBody[struct {
		Data  []apiClientSummary `json:"data"`
		Total int                `json:"total"`
	}](t, list)
	assert.Equal(t, 1, page.Total)
	require.Len(t, page.Data, 1)
	assert.Equal(t, 2, page.Data[0].OperationCount)
	assert.NotContains(t, list.Body.String(), `"operations"`, "the list does not carry operations")

	download := env.do(t, "alice", http.MethodGet, "/api/v1/api-clients/"+client.ID+"/spec", "", nil)
	require.Equal(t, http.StatusOK, download.Code)
	assert.Equal(t, handlerTestSpec, download.Body.String())
	assert.Contains(t, download.Header().Get("Content-Type"), "application/yaml")
	assert.Equal(t, `attachment; filename=Weather.yaml`, download.Header().Get("Content-Disposition"))
}

func TestAPIClientHandlers_PreviewAndRawBody(t *testing.T) {
	t.Parallel()
	env := setupAPIClientHandlersTest(t)

	w := env.do(t, "alice", http.MethodPost, "/api/v1/api-clients/preview", "application/yaml", []byte(handlerTestSpec))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	preview := decodeBody[specPreviewResponse](t, w)
	assert.Equal(t, "Weather API", preview.Title)
	assert.Equal(t, "https://weather.test/api", preview.BaseURL)
	assert.Equal(t, 2, preview.OperationCount)
	assert.Equal(t, operationSummary{ID: "getForecast", Method: "GET", Path: "/forecast/{city}", Summary: "Forecast for a city", Tags: []string{"forecast"}}, preview.Operations[1])

	list := env.do(t, "alice", http.MethodGet, "/api/v1/api-clients", "", nil)
	assert.Contains(t, list.Body.String(), `"total":0`, "preview does not store anything")

	jsonBody, _ := json.Marshal(map[string]any{"spec": handlerTestSpec, "base_url": "https://staging.weather.test"})
	created := env.do(t, "alice", http.MethodPost, "/api/v1/api-clients", "application/json", jsonBody)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	assert.Equal(t, "https://staging.weather.test", decodeBody[models.APIClient](t, created).BaseURL)
}

func TestAPIClientHandlers_Errors(t *testing.T) {
	t.Parallel()
	env := setupAPIClientHandlersTest(t)

	w := env.do(t, "", http.MethodGet, "/api/v1/api-clients", "", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	w = env.do(t, "alice", http.MethodPost, "/api/v1/api-clients", "application/yaml", []byte("paths: {}\n"))
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_API_SPEC", decodeBody[APIError](t, w).Code)

	w = env.do(t, "alice", http.MethodPost, "/api/v1/api-clients", "application/yaml", []byte("   "))
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "EMPTY_SPEC", decodeBody[APIError](t, w).Code)

	body, contentType := multipartSpec(t, handlerTestSpec, map[string]string{"auth": "not json"})
	w = env.do(t, "alice", http.MethodPost, "/api/v1/api-clients", contentType, body)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_AUTH", decodeBody[APIError](t, w).Code)

	body, contentType = multipartSpec(t, handlerTestSpec, map[string]string{"auth": `{"type": "bearer", "credential_id": "x"}`})
	w = env.do(t, "alice", http.MethodPost, "/api/v1/api-clients", contentType, body)
	require.Equal(t, http.StatusBadRequest, w.Code, "credentials need an encryption key")
	apiErr := decodeBody[APIError](t, w)
	assert.Equal(t, "VALIDATION_ERROR", apiErr.Code)
	assert.Equal(t, "auth.credential_id", apiErr.Details["field"])

	w = env.do(t, "alice", http.MethodGet, "/api/v1/api-clients/not-a-uuid", "", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = env.do(t, "alice", http.MethodGet, "/api/v1/api-clients/"+uuid.NewString(), "", nil)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "API_CLIENT_NOT_FOUND", decodeBody[APIError](t, w).Code)
}

func TestAPIClientHandlers_OwnerIsolationUpdateAndDelete(t *testing.T) {
	t.Parallel()
	env := setupAPIClientHandlersTest(t)

	created := env.do(t, "alice", http.MethodPost, "/api/v1/api-clients", "application/yaml", []byte(handlerTestSpec))
	require.Equal(t, http.StatusCreated, created.Code)
	id := decodeBody[models.APIClient](t, created).ID
	path := "/api/v1/api-clients/" + id

	for _, req := range []struct{ method, path string }{
		{http.MethodGet, path},
		{http.MethodGet, path + "/spec"},
		{http.MethodDelete, path},
	} {
		w := env.do(t, "bob", req.method, req.path, "", nil)
		assert.Equal(t, http.StatusNotFound, w.Code, "%s %s by another user", req.method, req.path)
	}
	w := env.do(t, "bob", http.MethodPut, path, "application/json", []byte(`{"name": "stolen"}`))
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = env.do(t, "alice", http.MethodPut, path, "application/json", []byte(`{"name": "Weather (prod)", "base_url": "https://prod.weather.test/api/"}`))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	updated := decodeBody[models.APIClient](t, w)
	assert.Equal(t, "Weather (prod)", updated.Name)
	assert.Equal(t, "https://prod.weather.test/api", updated.BaseURL)
	assert.Len(t, updated.Operations, 2)

	newSpec := strings.Replace(handlerTestSpec, "operationId: createAlert", "operationId: createWarning", 1)
	w = env.do(t, "alice", http.MethodPut, path+"/spec", "application/yaml", []byte(newSpec))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	replaced := decodeBody[replaceSpecResponse](t, w)
	assert.Equal(t, &apiclient.SpecChanges{Added: []string{"createWarning"}, Removed: []string{"createAlert"}}, replaced.Changes)
	assert.Equal(t, "Weather (prod)", replaced.Client.Name)

	w = env.do(t, "alice", http.MethodDelete, path, "", nil)
	require.Equal(t, http.StatusOK, w.Code)
	w = env.do(t, "alice", http.MethodGet, path, "", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
