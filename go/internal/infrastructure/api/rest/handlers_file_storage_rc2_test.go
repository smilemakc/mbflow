package rest

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/application/filestorage"
	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/smilemakc/mbflow/go/testutil"
)

func rc2FileStorageRouter(env *rc2Env, userID string, maxFileSize int64) *gin.Engine {
	svc := filestorage.NewResourceFileService(env.db, env.resourceRepo, env.fileRepo, env.storageMgr, maxFileSize)
	h := NewFileStorageHandlers(env.resourceRepo, svc, env.log)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	if userID != "" {
		r.Use(testutil.MockAuthMiddleware(userID, false))
	}
	res := r.Group("/api/v1/resources")
	res.POST("/:id/files", h.UploadFile)
	res.GET("/:id/files", h.ListFiles)
	res.GET("/:id/files/:file_id", h.GetFileMetadata)
	res.GET("/:id/files/:file_id/download", h.DownloadFile)
	res.DELETE("/:id/files/:file_id", h.DeleteFile)

	np := r.Group("/np")
	np.POST("/upload", h.UploadFile)
	np.GET("/list", h.ListFiles)
	np.GET("/meta", h.GetFileMetadata)
	np.GET("/meta/:id", h.GetFileMetadata)
	np.GET("/download", h.DownloadFile)
	np.GET("/download/:id", h.DownloadFile)
	np.DELETE("/delete", h.DeleteFile)
	np.DELETE("/delete/:id", h.DeleteFile)
	return r
}

func rc2CreateFileStorage(t *testing.T, env *rc2Env, owner string, mutate func(*pkgmodels.FileStorageResource)) string {
	t.Helper()
	res := pkgmodels.NewFileStorageResource(owner, "rc2 storage")
	if mutate != nil {
		mutate(res)
	}
	require.NoError(t, env.resourceRepo.Create(context.Background(), res))
	return res.GetID()
}

func TestRC2_FileStorage_UploadDownloadDelete(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	ra := rc2FileStorageRouter(env, alice, 1024)
	rb := rc2FileStorageRouter(env, bob, 1024)
	anon := rc2FileStorageRouter(env, "", 1024)

	resID := rc2CreateFileStorage(t, env, alice, nil)
	otherRes := rc2CreateFileStorage(t, env, alice, nil)
	base := "/api/v1/resources/" + resID + "/files"

	upload := func(r http.Handler, path, name string, content []byte) *bytesResponse {
		body, ct := rc2Multipart(t, name, content, nil)
		w := rc2PostMultipart(t, r, path, body, ct)
		return &bytesResponse{code: w.Code, body: w.Body.Bytes(), m: func() map[string]any { return rc2DecodeMap(t, w) }}
	}

	// Authentication and ownership.
	for _, p := range []struct{ method, path string }{
		{http.MethodPost, base}, {http.MethodGet, base},
		{http.MethodGet, base + "/" + uuid.NewString()},
		{http.MethodGet, base + "/" + uuid.NewString() + "/download"},
		{http.MethodDelete, base + "/" + uuid.NewString()},
	} {
		assert.Equal(t, http.StatusUnauthorized, rc2Do(t, anon, p.method, p.path, nil).Code, p.path)
		assert.Equal(t, http.StatusForbidden, rc2Do(t, rb, p.method, p.path, nil).Code, p.path)
		assert.Equal(t, http.StatusNotFound, rc2Do(t, ra, p.method, "/api/v1/resources/"+uuid.NewString()+p.path[len("/api/v1/resources/")+len(resID):], nil).Code, p.path)
		assert.Equal(t, http.StatusNotFound, rc2Do(t, ra, p.method, "/api/v1/resources/not-a-uuid"+p.path[len("/api/v1/resources/")+len(resID):], nil).Code,
			"malformed resource IDs are not server errors: %s", p.path)
	}

	// Upload validation.
	w := rc2Do(t, ra, http.MethodPost, base, nil)
	assert.Equal(t, "file is required", rc2DecodeMap(t, w)["message"])
	resp := upload(ra, base, "a.bin", []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0})
	assert.Equal(t, http.StatusBadRequest, resp.code)
	resp = upload(ra, base, "big.txt", bytes.Repeat([]byte("a"), 2048))
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.code, string(resp.body))

	// Upload a text file (charset parameter is stripped).
	resp = upload(ra, base, "notes.txt", []byte("hello resource"))
	require.Equal(t, http.StatusCreated, resp.code, string(resp.body))
	created := resp.m()
	assert.Equal(t, "text/plain", created["mime_type"])
	fileID := created["id"].(string)
	resp = upload(ra, "/api/v1/resources/"+otherRes+"/files", "other.txt", []byte("other"))
	require.Equal(t, http.StatusCreated, resp.code, string(resp.body))
	otherFileID := resp.m()["id"].(string)

	// List.
	w = rc2Do(t, ra, http.MethodGet, base+"?limit=10", nil)
	require.Equal(t, http.StatusOK, w.Code)
	list := rc2DecodeMap(t, w)
	assert.Equal(t, float64(1), list["total"])
	assert.Equal(t, float64(10), list["limit"])

	// Metadata.
	w = rc2Do(t, ra, http.MethodGet, base+"/"+fileID, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "notes.txt", rc2DecodeMap(t, w)["name"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, ra, http.MethodGet, base+"/"+uuid.NewString(), nil).Code)
	assert.Equal(t, http.StatusForbidden, rc2Do(t, ra, http.MethodGet, base+"/"+otherFileID, nil).Code, "file of another resource")

	// Download.
	w = rc2Do(t, ra, http.MethodGet, base+"/"+fileID+"/download", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "hello resource", w.Body.String())
	assert.Contains(t, w.Header().Get("Content-Disposition"), "notes.txt")
	assert.Equal(t, http.StatusNotFound, rc2Do(t, ra, http.MethodGet, base+"/"+uuid.NewString()+"/download", nil).Code)
	assert.Equal(t, http.StatusForbidden, rc2Do(t, ra, http.MethodGet, base+"/"+otherFileID+"/download", nil).Code)

	// Expired files.
	past := time.Now().Add(-time.Minute)
	_, err := env.db.NewUpdate().Table("mbflow_files").Set("expires_at = ?", past).Where("id = ?", uuid.MustParse(fileID)).Exec(context.Background())
	require.NoError(t, err)
	w = rc2Do(t, ra, http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, w.Code)
	files := rc2DecodeMap(t, w)["files"].([]any)
	require.Len(t, files, 1)
	assert.NotNil(t, files[0].(map[string]any)["expires_at"])
	w = rc2Do(t, ra, http.MethodGet, base+"/"+fileID, nil)
	assert.NotNil(t, rc2DecodeMap(t, w)["expires_at"])
	assert.Equal(t, http.StatusGone, rc2Do(t, ra, http.MethodGet, base+"/"+fileID+"/download", nil).Code)

	// Delete.
	assert.Equal(t, http.StatusNotFound, rc2Do(t, ra, http.MethodDelete, base+"/"+uuid.NewString(), nil).Code)
	assert.Equal(t, http.StatusForbidden, rc2Do(t, ra, http.MethodDelete, base+"/"+otherFileID, nil).Code)
	w = rc2Do(t, ra, http.MethodDelete, base+"/"+fileID, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, http.StatusNotFound, rc2Do(t, ra, http.MethodGet, base+"/"+fileID, nil).Code)

	// Malformed file IDs are client errors.
	for _, p := range []struct{ method, path string }{
		{http.MethodGet, base + "/bad"}, {http.MethodGet, base + "/bad/download"}, {http.MethodDelete, base + "/bad"},
	} {
		w = rc2Do(t, ra, p.method, p.path, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code, p.path)
		assert.Equal(t, "invalid file ID", rc2DecodeMap(t, w)["message"], p.path)
	}

	// Missing path parameters.
	for _, p := range []struct{ method, path string }{
		{http.MethodPost, "/np/upload"}, {http.MethodGet, "/np/list"},
		{http.MethodGet, "/np/meta"}, {http.MethodGet, "/np/meta/" + resID},
		{http.MethodGet, "/np/download"}, {http.MethodGet, "/np/download/" + resID},
		{http.MethodDelete, "/np/delete"}, {http.MethodDelete, "/np/delete/" + resID},
	} {
		w = rc2Do(t, ra, p.method, p.path, nil)
		assert.Equal(t, "MISSING_PARAMETER", rc2DecodeMap(t, w)["code"], p.path)
	}
}

type bytesResponse struct {
	code int
	body []byte
	m    func() map[string]any
}

func TestRC2_FileStorage_QuotaInactiveAndDatabaseFailures(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	ra := rc2FileStorageRouter(env, alice, 1<<20)

	tiny := rc2CreateFileStorage(t, env, alice, func(r *pkgmodels.FileStorageResource) { r.StorageLimitBytes = 4 })
	body, ct := rc2Multipart(t, "q.txt", []byte("more than four bytes"), nil)
	w := rc2PostMultipart(t, ra, "/api/v1/resources/"+tiny+"/files", body, ct)
	require.Equal(t, http.StatusInsufficientStorage, w.Code, w.Body.String())
	assert.Contains(t, rc2DecodeMap(t, w)["message"], "quota exceeded")

	inactive := rc2CreateFileStorage(t, env, alice, func(r *pkgmodels.FileStorageResource) { r.Status = pkgmodels.ResourceStatusSuspended })
	body, ct = rc2Multipart(t, "i.txt", []byte("x"), nil)
	w = rc2PostMultipart(t, ra, "/api/v1/resources/"+inactive+"/files", body, ct)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "failed to upload file", rc2DecodeMap(t, w)["message"])

	require.NoError(t, env.db.Close())
	for _, p := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/resources/" + tiny + "/files"},
		{http.MethodGet, "/api/v1/resources/" + tiny + "/files"},
		{http.MethodGet, "/api/v1/resources/" + tiny + "/files/" + uuid.NewString()},
		{http.MethodGet, "/api/v1/resources/" + tiny + "/files/" + uuid.NewString() + "/download"},
		{http.MethodDelete, "/api/v1/resources/" + tiny + "/files/" + uuid.NewString()},
	} {
		w := rc2Do(t, ra, p.method, p.path, nil)
		require.Equal(t, http.StatusInternalServerError, w.Code, p.path)
		assert.Equal(t, "failed to get resource", rc2DecodeMap(t, w)["message"])
	}
}
