package rest

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/filestorage"
)

func TestRC2_Auth_RefreshValidation(t *testing.T) {
	env := rc2NewAuthEnv(t, nil)
	w := env.do(t, http.MethodPost, "/api/v1/auth/refresh", "", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = env.do(t, http.MethodPost, "/api/v1/auth/refresh", "", map[string]string{"refresh_token": "unknown"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRC2_Workflows_LoopEdgesAndNonUUIDCaller(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	ra := env.router(alice, false, true)

	body := rc2SimpleWorkflow("loop-wf")
	body["edges"] = []map[string]any{
		{"id": "e1", "from": "n1", "to": "n2"},
		{"id": "back", "from": "n2", "to": "n1", "loop": map[string]any{"max_iterations": 2}},
	}
	wf := env.createWorkflow(t, ra, body)
	var loop map[string]any
	for _, e := range wf["edges"].([]any) {
		if e.(map[string]any)["id"] == "back" {
			loop = e.(map[string]any)
		}
	}
	require.NotNil(t, loop, "loop edge is stored")
	assert.Equal(t, float64(2), loop["loop"].(map[string]any)["max_iterations"])

	// A caller whose ID is not a UUID (e.g. an external identity) can still
	// attach resources when no ownership guard is configured.
	raw := env.router("external-user", false, false)
	w := rc2Do(t, raw, http.MethodPost, "/api/v1/workflows/"+wf["id"].(string)+"/resources",
		map[string]any{"resource_id": uuid.NewString(), "alias": "a", "access_type": "write"})
	assert.GreaterOrEqual(t, w.Code, 400, "unknown resource cannot be attached: %s", w.Body.String())
}

func TestRC2_Files_StoreFailuresAndMIME(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)

	sm := filestorage.NewStorageManager(&filestorage.ManagerConfig{BasePath: t.TempDir(), MaxFileSize: 4}, env.log)
	t.Cleanup(func() { _ = sm.Close() })
	h := NewFileHandlers(env.fileRepo, sm, env.log)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/files", h.HandleUploadFile)

	// The storage rejects files over its size limit.
	w := rc2Do(t, r, http.MethodPost, "/files", map[string]any{"file_data": base64.StdEncoding.EncodeToString(rc2PNG)})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, rc2DecodeMap(t, w)["message"], "exceeds maximum")
	body, ct := rc2Multipart(t, "x.png", rc2PNG, nil)
	w = rc2PostMultipart(t, r, "/files", body, ct)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Explicit MIME types are checked against the allow-list.
	w = rc2Do(t, r, http.MethodPost, "/files", map[string]any{"file_data": "aGk=", "mime_type": "application/x-msdownload"})
	assert.Equal(t, "MIME type not allowed: application/x-msdownload", rc2DecodeMap(t, w)["message"])

	// A truncated download is reported as a read failure.
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte("short"))
	}))
	t.Cleanup(src.Close)
	w = rc2Do(t, r, http.MethodPost, "/files", map[string]any{"file_url": src.URL})
	assert.Equal(t, "failed to read file from URL", rc2DecodeMap(t, w)["message"])
}

func TestRC2_Files_AdminUsageDatabaseFailure(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	admin := env.router(uuid.NewString(), true, false)
	require.NoError(t, env.db.Close())
	w := rc2Do(t, admin, http.MethodGet, "/api/v1/files/storage/default/usage", nil)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "failed to get storage usage", rc2DecodeMap(t, w)["message"])
}
