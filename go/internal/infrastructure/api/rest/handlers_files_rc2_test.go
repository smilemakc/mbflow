package rest

import (
	"bytes"
	"context"
	"encoding/base64"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/filestorage"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

var rc2PNG = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 'I', 'H', 'D', 'R'}

func rc2Multipart(t *testing.T, filename string, content []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	if filename != "" {
		part, err := mw.CreateFormFile("file", filename)
		require.NoError(t, err)
		_, err = part.Write(content)
		require.NoError(t, err)
	}
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	require.NoError(t, mw.Close())
	return body, mw.FormDataContentType()
}

func rc2PostMultipart(t *testing.T, r http.Handler, path string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRC2_Files_UploadAuthorizationAndValidation(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	ra := env.router(alice, false, true)
	rb := env.router(bob, false, true)
	wfID := env.createWorkflow(t, ra, rc2SimpleWorkflow("files-wf"))["id"].(string)

	// Multipart: missing file, invalid storage ID (path traversal),
	// disallowed MIME type, attachment to someone else's workflow.
	body, ct := rc2Multipart(t, "", nil, map[string]string{"access_scope": "workflow"})
	w := rc2PostMultipart(t, ra, "/api/v1/files", body, ct)
	assert.Equal(t, "file is required", rc2DecodeMap(t, w)["message"])

	body, ct = rc2Multipart(t, "evil.png", rc2PNG, map[string]string{"storage_id": "../../escape"})
	w = rc2PostMultipart(t, ra, "/api/v1/files", body, ct)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid storage_id", rc2DecodeMap(t, w)["message"])
	_, err := os.Stat(filepath.Join(filepath.Dir(env.storageBase), "escape"))
	assert.True(t, os.IsNotExist(err), "nothing is created outside the storage root")

	body, ct = rc2Multipart(t, "a.bin", []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0, 0, 0}, nil)
	w = rc2PostMultipart(t, ra, "/api/v1/files", body, ct)
	assert.Contains(t, rc2DecodeMap(t, w)["message"], "MIME type not allowed")

	body, ct = rc2Multipart(t, "x.png", rc2PNG, map[string]string{"workflow_id": wfID})
	assert.Equal(t, http.StatusNotFound, rc2PostMultipart(t, rb, "/api/v1/files", body, ct).Code)
	body, ct = rc2Multipart(t, "x.png", rc2PNG, map[string]string{"workflow_id": "bad"})
	assert.Equal(t, "invalid workflow_id", rc2DecodeMap(t, rc2PostMultipart(t, ra, "/api/v1/files", body, ct))["message"])
	body, ct = rc2Multipart(t, "x.png", rc2PNG, map[string]string{"execution_id": "bad"})
	assert.Equal(t, "invalid execution_id", rc2DecodeMap(t, rc2PostMultipart(t, ra, "/api/v1/files", body, ct))["message"])
	body, ct = rc2Multipart(t, "x.png", rc2PNG, map[string]string{"execution_id": uuid.NewString()})
	assert.Equal(t, http.StatusNotFound, rc2PostMultipart(t, ra, "/api/v1/files", body, ct).Code)

	// Plain-text uploads are detected as "text/plain; charset=utf-8" and
	// must still pass the MIME allow-list.
	body, ct = rc2Multipart(t, "notes.txt", []byte("hello world"), map[string]string{
		"workflow_id": wfID, "tags": "a, b,,", "storage_id": "team-1", "access_scope": "workflow",
	})
	w = rc2PostMultipart(t, ra, "/api/v1/files", body, ct)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	txt := rc2DecodeMap(t, w)
	assert.Equal(t, "text/plain", txt["mime_type"])
	assert.Equal(t, wfID, txt["workflow_id"])
	assert.Equal(t, []any{"a", "b"}, txt["tags"])
	assert.Equal(t, "team-1", txt["storage_id"])

	// JSON uploads.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/files", "{bad").Code)
	w = rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_name": "x"})
	assert.Equal(t, "either file_data or file_url is required", rc2DecodeMap(t, w)["message"])
	w = rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_data": "!!!", "storage_id": "a/b"})
	assert.Equal(t, "invalid storage_id", rc2DecodeMap(t, w)["message"])
	w = rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_data": "!!!"})
	assert.Equal(t, "invalid base64 data", rc2DecodeMap(t, w)["message"])
	w = rc2Do(t, rb, http.MethodPost, "/api/v1/files", map[string]any{"file_data": "aGk=", "workflow_id": wfID})
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_data": base64.StdEncoding.EncodeToString([]byte("plain text")), "workflow_id": wfID})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "text/plain", rc2DecodeMap(t, w)["mime_type"])

	// Attaching to one's own execution is allowed.
	w = rc2Do(t, ra, http.MethodPost, "/api/v1/executions/run/"+wfID, map[string]any{})
	require.Equal(t, http.StatusAccepted, w.Code)
	execID := rc2DecodeMap(t, w)["id"].(string)
	rc2WaitExecution(t, env, execID)
	w = rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_data": base64.StdEncoding.EncodeToString(rc2PNG), "execution_id": execID, "file_name": "e.png"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, execID, rc2DecodeMap(t, w)["execution_id"])
	w = rc2Do(t, rb, http.MethodPost, "/api/v1/files", map[string]any{"file_data": base64.StdEncoding.EncodeToString(rc2PNG), "execution_id": execID})
	assert.Equal(t, http.StatusNotFound, w.Code, "other users cannot attach files to the execution")
}

func TestRC2_Files_UploadFromURL(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	ra := env.router(alice, false, true)

	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/doc.csv":
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			_, _ = w.Write([]byte("a,b\n1,2\n"))
		case "/raw":
			// No Content-Type header: sniffed from the body.
			w.Header()["Content-Type"] = nil
			_, _ = w.Write(rc2PNG)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(src.Close)

	w := rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_url": src.URL + "/doc.csv", "file_name": "doc.csv"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "text/csv", rc2DecodeMap(t, w)["mime_type"])
	assert.Equal(t, float64(8), rc2DecodeMap(t, w)["size"])

	w = rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_url": src.URL + "/raw"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "image/png", rc2DecodeMap(t, w)["mime_type"])

	w = rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_url": src.URL + "/missing"})
	assert.Equal(t, "failed to download file: HTTP 404", rc2DecodeMap(t, w)["message"])

	w = rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_url": "http://[::1"})
	assert.Equal(t, "invalid file URL", rc2DecodeMap(t, w)["message"])

	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()
	w = rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{"file_url": closedURL + "/x"})
	assert.Equal(t, "failed to download file from URL", rc2DecodeMap(t, w)["message"])
}

func TestRC2_Files_DownloadMetadataDeleteListUsage(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	ra := env.router(alice, false, true)
	rb := env.router(bob, false, true)
	admin := env.router(env.user(t, true), true, true)
	raw := env.router(alice, false, false)
	wfID := env.createWorkflow(t, ra, map[string]any{"name": "wf"})["id"].(string)

	w := rc2Do(t, ra, http.MethodPost, "/api/v1/files", map[string]any{
		"file_data": base64.StdEncoding.EncodeToString(rc2PNG), "file_name": "pic.png",
		"tags": []string{"t1"}, "workflow_id": wfID, "access_scope": "workflow",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	fileID := rc2DecodeMap(t, w)["id"].(string)

	// Download.
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/files/"+fileID, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, rc2PNG, w.Body.Bytes())
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), `filename="pic.png"`)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodGet, "/api/v1/files/"+fileID, nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/files/bad", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodGet, "/api/v1/files/"+uuid.NewString(), nil).Code)

	// Expired files are gone; files whose content vanished are a server error.
	ctx := context.Background()
	past := time.Now().Add(-time.Hour)
	expired := &storagemodels.FileModel{ID: uuid.New(), StorageID: "default", Name: "old", Path: "nope", MimeType: "image/png", AccessScope: "workflow", ExpiresAt: &past}
	require.NoError(t, env.fileRepo.Create(ctx, expired))
	assert.Equal(t, http.StatusGone, rc2Do(t, raw, http.MethodGet, "/api/v1/files/"+expired.ID.String(), nil).Code)
	orphan := &storagemodels.FileModel{ID: uuid.New(), StorageID: "default", Name: "gone", Path: "default/none/gone", MimeType: "image/png", AccessScope: "workflow"}
	require.NoError(t, env.fileRepo.Create(ctx, orphan))
	w = rc2Do(t, raw, http.MethodGet, "/api/v1/files/"+orphan.ID.String(), nil)
	assert.Equal(t, "failed to retrieve file", rc2DecodeMap(t, w)["message"])

	// Metadata.
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/files/"+fileID+"/metadata", nil)
	require.Equal(t, http.StatusOK, w.Code)
	meta := rc2DecodeMap(t, w)
	assert.Equal(t, "pic.png", meta["name"])
	assert.Equal(t, wfID, meta["workflow_id"])
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/files/bad/metadata", nil).Code)
	w = rc2Do(t, raw, http.MethodGet, "/api/v1/files/"+expired.ID.String()+"/metadata", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, rc2DecodeMap(t, w)["expires_at"])

	// List with every filter.
	list := func(r http.Handler, q string) map[string]any {
		w := rc2Do(t, r, http.MethodGet, "/api/v1/files"+q, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		return rc2DecodeMap(t, w)
	}
	got := list(ra, "?workflow_id="+wfID+"&tags=t1,&mime_types=image/png,&storage_id=default&access_scope=workflow&order_by=name&order_dir=ASC&limit=5")
	assert.Equal(t, float64(1), got["total"])
	assert.Equal(t, float64(5), got["limit"])
	assert.Equal(t, float64(0), list(rb, "")["total"], "users only list their own files")
	assert.Equal(t, float64(0), list(ra, "?execution_id="+uuid.NewString())["total"])
	assert.Equal(t, float64(1), list(ra, "?workflow_id=bad&execution_id=bad")["total"], "malformed filters are ignored")
	assert.Equal(t, float64(3), list(admin, "")["total"])

	// Storage usage: filtered for users, aggregate for admins.
	usage := func(r http.Handler, path string) map[string]any {
		w := rc2Do(t, r, http.MethodGet, path, nil)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		return rc2DecodeMap(t, w)
	}
	u := usage(ra, "/api/v1/files/storage/default/usage")
	assert.Equal(t, float64(len(rc2PNG)), u["total_size"])
	assert.Equal(t, float64(1), u["file_count"])
	assert.Equal(t, float64(0), usage(rb, "/api/v1/files/storage/default/usage")["file_count"])
	assert.Equal(t, float64(3), usage(admin, "/api/v1/files/storage/default/usage")["file_count"])
	assert.Equal(t, float64(3), usage(admin, "/api/v1/files/usage")["file_count"], "storage defaults to default")
	assert.Equal(t, "workflow", usage(admin, "/api/v1/files/storage/default/usage?access_scope=workflow")["access_scope"])

	// Delete.
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodDelete, "/api/v1/files/"+fileID, nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodDelete, "/api/v1/files/bad", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodDelete, "/api/v1/files/"+uuid.NewString(), nil).Code)
	stored, err := env.fileRepo.FindByID(ctx, uuid.MustParse(fileID))
	require.NoError(t, err)
	onDisk := filepath.Join(env.storageBase, stored.StorageID, stored.Path)
	_, err = os.Stat(onDisk)
	require.NoError(t, err, "content is stored on disk")
	require.Equal(t, http.StatusOK, rc2Do(t, ra, http.MethodDelete, "/api/v1/files/"+fileID, nil).Code)
	_, err = os.Stat(onDisk)
	assert.True(t, os.IsNotExist(err), "deleting a file removes its content")
	assert.Equal(t, http.StatusNotFound, rc2Do(t, ra, http.MethodGet, "/api/v1/files/"+fileID+"/metadata", nil).Code)

	// Database failures.
	require.NoError(t, env.db.Close())
	for _, p := range []string{"/api/v1/files", "/api/v1/files/usage", "/api/v1/files/storage/default/usage?access_scope=workflow"} {
		assert.Equal(t, http.StatusInternalServerError, rc2Do(t, raw, http.MethodGet, p, nil).Code, p)
		assert.Equal(t, http.StatusInternalServerError, rc2Do(t, env.router(alice, false, false), http.MethodGet, p, nil).Code, p)
	}
	w = rc2Do(t, raw, http.MethodPost, "/api/v1/files", map[string]any{"file_data": base64.StdEncoding.EncodeToString(rc2PNG)})
	assert.Equal(t, "failed to save file metadata", rc2DecodeMap(t, w)["message"])
	body, ct := rc2Multipart(t, "x.png", rc2PNG, nil)
	w = rc2PostMultipart(t, raw, "/api/v1/files", body, ct)
	assert.Equal(t, "failed to save file metadata", rc2DecodeMap(t, w)["message"])
}

func TestRC2_Files_StorageUnavailable(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)

	// A storage root that is a regular file: no storage can be created.
	blocked := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(blocked, []byte("x"), 0o600))
	sm := filestorage.NewStorageManager(&filestorage.ManagerConfig{BasePath: blocked}, env.log)
	t.Cleanup(func() { _ = sm.Close() })
	h := NewFileHandlers(env.fileRepo, sm, env.log)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/files", h.HandleUploadFile)
	r.GET("/files/:id", h.HandleGetFile)

	w := rc2Do(t, r, http.MethodPost, "/files", map[string]any{"file_data": base64.StdEncoding.EncodeToString(rc2PNG), "storage_id": "s1"})
	assert.Equal(t, "failed to get storage", rc2DecodeMap(t, w)["message"])
	body, ct := rc2Multipart(t, "x.png", rc2PNG, map[string]string{"storage_id": "s2"})
	w = rc2PostMultipart(t, r, "/files", body, ct)
	assert.Equal(t, "failed to get storage", rc2DecodeMap(t, w)["message"])

	m := &storagemodels.FileModel{ID: uuid.New(), StorageID: "s3", Name: "f", Path: "p", MimeType: "image/png", AccessScope: "workflow"}
	require.NoError(t, env.fileRepo.Create(context.Background(), m))
	w = rc2Do(t, r, http.MethodGet, "/files/"+m.ID.String(), nil)
	assert.Equal(t, "storage not available", rc2DecodeMap(t, w)["message"])
}

func TestRC2_FileHelpers(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "text/plain", baseMimeType("text/plain; charset=utf-8"))
	assert.Equal(t, "garbage;;", baseMimeType(" garbage;; "))
	assert.True(t, validStorageID("default"))
	assert.True(t, validStorageID(uuid.NewString()))
	for _, bad := range []string{"", "..", "a/b", `a\b`, "a.b", string(make([]byte, 65))} {
		assert.False(t, validStorageID(bad), "%q", bad)
	}
	assert.Equal(t, 1, min(1, 2))
	assert.Equal(t, 1, min(2, 1))

	h := &FileHandlers{}
	ttl := 90 * time.Second
	exp := time.Now().Add(time.Hour)
	wf, ex, node := uuid.NewString(), uuid.NewString(), "node-1"
	model := h.entryToModel(&pkgmodels.FileEntry{
		ID: uuid.NewString(), StorageID: "s", Name: "n", WorkflowID: &wf, ExecutionID: &ex,
		SourceNodeID: &node, TTL: &ttl, ExpiresAt: &exp,
	})
	assert.Equal(t, wf, model.WorkflowID.String())
	assert.Equal(t, ex, model.ExecutionID.String())
	assert.Equal(t, 90, *model.TTLSeconds)
	resp := h.modelToResponse(model)
	assert.Equal(t, "node-1", resp["source_node_id"])
	assert.Equal(t, ex, resp["execution_id"])
	assert.NotNil(t, resp["expires_at"])
}
