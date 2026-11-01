package rest

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/smilemakc/mbflow/go/internal/application/apiclient"
	"github.com/smilemakc/mbflow/go/internal/infrastructure/storage"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

const rc1JSONSpec = `{
  "openapi": "3.0.3",
  "info": {"title": "Pets", "version": "1"},
  "servers": [{"url": "https://pets.test"}],
  "paths": {"/pets": {"get": {"operationId": "listPets", "responses": {"200": {"description": "ok"}}}}}
}`

// rc1APIClientRouter mounts the API client handlers on db. maxBody > 0 caps
// request bodies the way the body size middleware does.
func rc1APIClientRouter(db *bun.DB, maxBody int64) *gin.Engine {
	h := NewAPIClientHandlers(apiclient.NewService(storage.NewAPIClientRepository(db), nil), rc1Logger())
	r := gin.New()
	g := r.Group("/api/v1/api-clients", rc1UserFromHeader())
	if maxBody > 0 {
		g.Use(func(c *gin.Context) {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)
			c.Next()
		})
	}
	g.POST("/preview", h.HandlePreview)
	g.POST("", h.HandleCreate)
	g.GET("", h.HandleList)
	g.GET("/:id", h.HandleGet)
	g.PUT("/:id", h.HandleUpdate)
	g.DELETE("/:id", h.HandleDelete)
	g.GET("/:id/spec", h.HandleDownloadSpec)
	g.PUT("/:id/spec", h.HandleReplaceSpec)
	return r
}

func rc1Raw(t *testing.T, r http.Handler, method, path, contentType string, body []byte, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if user != "" {
		req.Header.Set("X-Test-User", user)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRC1APIClients_AuthAndUploadValidation(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	r := rc1APIClientRouter(db, 0)
	alice := rc1CreateUser(t, db, "alice").String()
	base := "/api/v1/api-clients"
	id := uuid.NewString()

	// Every owner-scoped endpoint requires a user.
	for _, tc := range []struct{ method, path string }{
		{http.MethodPost, base}, {http.MethodGet, base + "/" + id}, {http.MethodPut, base + "/" + id},
		{http.MethodDelete, base + "/" + id}, {http.MethodGet, base + "/" + id + "/spec"}, {http.MethodPut, base + "/" + id + "/spec"},
	} {
		w := rc1Raw(t, r, tc.method, tc.path, "application/yaml", []byte(rc1JSONSpec), "")
		assert.Equal(t, http.StatusUnauthorized, w.Code, "%s %s", tc.method, tc.path)
	}

	// Preview errors.
	w := rc1Raw(t, r, http.MethodPost, base+"/preview", "application/yaml", []byte("paths: {}\n"), "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_API_SPEC", rc1JSON(t, w)["code"])
	w = rc1Raw(t, r, http.MethodPost, base+"/preview", "application/json", []byte("{bad"), "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Multipart without a file, and with malformed default headers.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("name", "x"))
	require.NoError(t, mw.Close())
	w = rc1Raw(t, r, http.MethodPost, base, mw.FormDataContentType(), buf.Bytes(), alice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "FILE_REQUIRED", rc1JSON(t, w)["code"])

	body, ct := multipartSpec(t, rc1JSONSpec, map[string]string{"default_headers": `{"a": 1}`})
	w = rc1Raw(t, r, http.MethodPost, base, ct, body, alice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_HEADERS", rc1JSON(t, w)["code"])

	// Multipart with every optional field set.
	body, ct = multipartSpec(t, rc1JSONSpec, map[string]string{
		"name": "!!!", "description": "pets api", "base_url": "https://staging.pets.test", "auth": `{"type": "none"}`,
	})
	w = rc1Raw(t, r, http.MethodPost, base, ct, body, alice)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	created := rc1JSON(t, w)
	assert.Equal(t, "pets api", created["description"])
	assert.Equal(t, "https://staging.pets.test", created["base_url"])
	clientID := created["id"].(string)

	// A JSON document downloads as JSON; names without usable characters
	// fall back to a generic file name.
	w = rc1Raw(t, r, http.MethodGet, base+"/"+clientID+"/spec", "", nil, alice)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Equal(t, "attachment; filename=openapi.json", w.Header().Get("Content-Disposition"))
	assert.JSONEq(t, rc1JSONSpec, w.Body.String())

	// Update and spec replacement errors.
	w = rc1Raw(t, r, http.MethodPut, base+"/"+clientID, "application/json", []byte("{bad"), alice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Raw(t, r, http.MethodPut, base+"/"+clientID+"/spec", "application/yaml", []byte("  "), alice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "EMPTY_SPEC", rc1JSON(t, w)["code"])
	w = rc1Raw(t, r, http.MethodPut, base+"/"+clientID+"/spec", "application/yaml", []byte("paths: {}\n"), alice)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = rc1Raw(t, r, http.MethodPut, base+"/"+uuid.NewString()+"/spec", "application/yaml", []byte(rc1JSONSpec), alice)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Raw(t, r, http.MethodGet, base+"/"+uuid.NewString()+"/spec", "", nil, alice)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc1Raw(t, r, http.MethodDelete, base+"/"+uuid.NewString(), "", nil, alice)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Pagination bounds are normalised.
	for _, tc := range []struct {
		query          string
		limit, offset  float64
		expectedClient int
	}{
		{"?limit=0&offset=-3", float64(apiclient.DefaultListLimit), 0, 1},
		{"?limit=100000", float64(apiclient.MaxListLimit), 0, 1},
		{"?limit=5&offset=1", 5, 1, 0},
	} {
		w = rc1Raw(t, r, http.MethodGet, base+tc.query, "", nil, alice)
		require.Equal(t, http.StatusOK, w.Code)
		page := rc1JSON(t, w)
		assert.Equal(t, tc.limit, page["limit"], tc.query)
		assert.Equal(t, tc.offset, page["offset"], tc.query)
		assert.Len(t, page["data"], tc.expectedClient, tc.query)
	}
}

func TestRC1APIClients_BodyLimitsAndReadErrors(t *testing.T) {
	t.Parallel()
	db := rc1DB(t)
	alice := rc1CreateUser(t, db, "alice").String()
	r := rc1APIClientRouter(db, 64)
	base := "/api/v1/api-clients"

	big := bytes.Repeat([]byte("a"), 1024)
	w := rc1Raw(t, r, http.MethodPost, base, "application/yaml", big, alice)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	body, ct := multipartSpec(t, string(big), nil)
	w = rc1Raw(t, r, http.MethodPost, base, ct, body, alice)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	// A failing body stream is reported as a read error.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-clients/preview", rc1FailingReader{})
	req.Header.Set("Content-Type", "application/yaml")
	rec := httptest.NewRecorder()
	rc1APIClientRouter(db, 0).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "READ_ERROR", rc1JSON(t, rec)["code"])

	assert.True(t, isBodyTooLarge(&http.MaxBytesError{Limit: 1}))
	assert.False(t, isBodyTooLarge(assert.AnError))
}

func TestRC1APIClients_StorageFailures(t *testing.T) {
	t.Parallel()
	r := rc1APIClientRouter(rc1ClosedDB(t), 0)
	user := uuid.NewString()
	base := "/api/v1/api-clients"
	id := uuid.NewString()

	w := rc1Raw(t, r, http.MethodPost, base, "application/json", []byte(`{"spec": `+rc1JSONQuote(rc1JSONSpec)+`}`), user)
	assert.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, base}, {http.MethodGet, base + "/" + id}, {http.MethodDelete, base + "/" + id},
		{http.MethodGet, base + "/" + id + "/spec"},
	} {
		w = rc1Raw(t, r, tc.method, tc.path, "", nil, user)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "%s %s", tc.method, tc.path)
	}
	w = rc1Raw(t, r, http.MethodPut, base+"/"+id, "application/json", []byte(`{"name": "n"}`), user)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	w = rc1Raw(t, r, http.MethodPut, base+"/"+id+"/spec", "application/json", []byte(`{"spec": `+rc1JSONQuote(rc1JSONSpec)+`}`), user)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Preview needs no storage at all.
	w = rc1Raw(t, r, http.MethodPost, base+"/preview", "application/json", []byte(`{"spec": `+rc1JSONQuote(rc1JSONSpec)+`}`), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "Pets", rc1JSON(t, w)["title"])

	assert.Equal(t, "My-API_v1.2", fileSlug("My API_v1.2"))
	assert.Equal(t, []models.APIClientAuth{}, nonNil[models.APIClientAuth](nil))
}

func rc1JSONQuote(s string) string {
	var b bytes.Buffer
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
