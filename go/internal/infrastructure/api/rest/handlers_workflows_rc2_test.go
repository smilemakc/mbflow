package rest

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgmodels "github.com/smilemakc/mbflow/go/pkg/models"
)

func TestRC2_Workflows_CreateGetListOwnership(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	admin := env.user(t, true)
	ra := env.router(alice, false, true)
	rb := env.router(bob, false, true)
	radmin := env.router(admin, true, true)

	// Validation.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/workflows", map[string]any{}).Code)
	w := rc2Do(t, ra, http.MethodPost, "/api/v1/workflows", map[string]any{
		"name":  "bad",
		"nodes": []map[string]any{{"id": "n1", "name": "x", "type": "no-such-executor"}},
	})
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	wf := env.createWorkflow(t, ra, rc2SimpleWorkflow("alice-wf"))
	wfID := wf["id"].(string)
	assert.Equal(t, alice, wf["created_by"])
	assert.Len(t, wf["nodes"], 2)
	assert.Len(t, wf["edges"], 1)
	env.createWorkflow(t, rb, map[string]any{"name": "bob-wf"})

	// Get: owner, other user (hidden as 404), admin, malformed ID.
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/"+wfID, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "alice-wf", rc2DecodeMap(t, w)["name"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodGet, "/api/v1/workflows/"+wfID, nil).Code)
	assert.Equal(t, http.StatusOK, rc2Do(t, radmin, http.MethodGet, "/api/v1/workflows/"+wfID, nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/not-a-uuid", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, radmin, http.MethodGet, "/api/v1/workflows/"+uuid.NewString(), nil).Code)

	// List: users see only their own workflows; admins see everything.
	assert.ElementsMatch(t, []string{wfID}, rc2ListIDs(t, ra, "/api/v1/workflows"))
	assert.Len(t, rc2ListIDs(t, radmin, "/api/v1/workflows?limit=10&offset=0"), 2)
	assert.ElementsMatch(t, []string{wfID}, rc2ListIDs(t, ra, "/api/v1/workflows?user_id="+alice+"&status=draft"))
	assert.Empty(t, rc2ListIDs(t, ra, "/api/v1/workflows?status=active"))
	assert.ElementsMatch(t, []string{wfID}, rc2ListIDs(t, radmin, "/api/v1/workflows?user_id="+alice))
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/workflows?user_id="+bob, nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/workflows?user_id=bad", nil)
	assert.Equal(t, "INVALID_USER_ID", rc2DecodeMap(t, w)["code"])
}

func rc2ListIDs(t *testing.T, r http.Handler, path string) []string {
	t.Helper()
	w := rc2Do(t, r, http.MethodGet, path, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var ids []string
	for _, it := range rc2DecodeMap(t, w)["data"].([]any) {
		ids = append(ids, it.(map[string]any)["id"].(string))
	}
	return ids
}

func TestRC2_Workflows_UpdatePublishDiagramDelete(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	ra := env.router(alice, false, true)
	raw := env.router(alice, false, false)
	wfID := env.createWorkflow(t, ra, rc2SimpleWorkflow("wf"))["id"].(string)
	base := "/api/v1/workflows/" + wfID

	// Update validation.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPut, "/api/v1/workflows/bad", map[string]any{}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPut, base, "{bad").Code)
	w := rc2Do(t, raw, http.MethodPut, "/api/v1/workflows/"+uuid.NewString(), map[string]any{"name": "x"})
	assert.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	// Update: rename, replace nodes and edges.
	w = rc2Do(t, ra, http.MethodPut, base, map[string]any{
		"name":        "renamed",
		"description": "d",
		"metadata":    map[string]any{"k": "v"},
		"nodes": []map[string]any{
			{"id": "a", "name": "A", "type": "transform", "config": map[string]any{"type": "passthrough"}},
			{"id": "b", "name": "B", "type": "transform", "config": map[string]any{"type": "passthrough"}},
		},
		"edges": []map[string]any{{"id": "ab", "from": "a", "to": "b", "loop": nil}},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	upd := rc2DecodeMap(t, w)
	assert.Equal(t, "renamed", upd["name"])
	assert.Len(t, upd["nodes"], 2)

	// Resources in an update: malformed ID, someone else's resource, own one.
	mine := pkgmodels.NewFileStorageResource(alice, "mine")
	require.NoError(t, env.resourceRepo.Create(context.Background(), mine))
	theirs := pkgmodels.NewFileStorageResource(bob, "theirs")
	require.NoError(t, env.resourceRepo.Create(context.Background(), theirs))
	w = rc2Do(t, ra, http.MethodPut, base, map[string]any{"resources": []map[string]any{{"resource_id": "bad", "alias": "x"}}})
	assert.Equal(t, "INVALID_RESOURCE_ID", rc2DecodeMap(t, w)["code"])
	w = rc2Do(t, ra, http.MethodPut, base, map[string]any{"resources": []map[string]any{{"resource_id": theirs.GetID(), "alias": "x"}}})
	assert.Equal(t, http.StatusNotFound, w.Code, "another user's resource cannot be attached")
	w = rc2Do(t, ra, http.MethodPut, base, map[string]any{"resources": []map[string]any{{"resource_id": mine.GetID(), "alias": "store", "access_type": "write"}}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	// Without the guard the resource list is passed through as-is.
	w = rc2Do(t, raw, http.MethodPut, base, map[string]any{"resources": []map[string]any{{"resource_id": mine.GetID(), "alias": "store2"}}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Publish / unpublish.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/workflows/bad/publish", nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/workflows/bad/unpublish", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodPost, "/api/v1/workflows/"+uuid.NewString()+"/publish", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodPost, "/api/v1/workflows/"+uuid.NewString()+"/unpublish", nil).Code)
	w = rc2Do(t, ra, http.MethodPost, base+"/publish", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "active", rc2DecodeMap(t, w)["status"])
	w = rc2Do(t, ra, http.MethodPost, base+"/unpublish", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "draft", rc2DecodeMap(t, w)["status"])

	// Diagram.
	w = rc2Do(t, ra, http.MethodGet, base+"/diagram", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, w.Body.String(), "flowchart TB")
	assert.Contains(t, w.Body.String(), "a --> b")
	w = rc2Do(t, ra, http.MethodGet, base+"/diagram?format=ascii&direction=LR&show_config=false&show_conditions=false&compact=true", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	w = rc2Do(t, ra, http.MethodGet, base+"/diagram?format=dot", nil)
	require.Equal(t, http.StatusBadRequest, w.Code, "unsupported formats are client errors")
	assert.Equal(t, "INVALID_FORMAT", rc2DecodeMap(t, w)["code"])
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/bad/diagram", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodGet, "/api/v1/workflows/"+uuid.NewString()+"/diagram", nil).Code)

	// Delete.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodDelete, "/api/v1/workflows/bad", nil).Code)
	w = rc2Do(t, ra, http.MethodDelete, base, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, ra, http.MethodGet, base, nil).Code)
}

func TestRC2_Workflows_ResourceEndpoints(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	ra := env.router(alice, false, true)
	raw := env.router(alice, false, false)
	anon := env.router("", false, false)
	wfID := env.createWorkflow(t, ra, map[string]any{"name": "wf"})["id"].(string)
	base := "/api/v1/workflows/" + wfID + "/resources"

	res := pkgmodels.NewFileStorageResource(alice, "mine")
	require.NoError(t, env.resourceRepo.Create(context.Background(), res))
	resID := res.GetID()

	// Unauthenticated callers are rejected by every resource endpoint.
	for _, p := range []struct{ method, path string }{
		{http.MethodPost, base}, {http.MethodGet, base},
		{http.MethodPut, base + "/" + resID}, {http.MethodDelete, base + "/" + resID},
	} {
		assert.Equal(t, http.StatusUnauthorized, rc2Do(t, anon, p.method, p.path, map[string]any{"alias": "a"}).Code, p.path)
	}

	// Attach validation.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/workflows/bad/resources", map[string]any{}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, base, map[string]any{"resource_id": "bad", "alias": "a"}).Code)
	w := rc2Do(t, ra, http.MethodPost, base, map[string]any{"resource_id": resID, "alias": "bad alias!"})
	assert.Equal(t, "VALIDATION_FAILED", rc2DecodeMap(t, w)["code"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodPost, "/api/v1/workflows/"+uuid.NewString()+"/resources", map[string]any{"resource_id": resID, "alias": "a"}).Code)

	w = rc2Do(t, ra, http.MethodPost, base, map[string]any{"resource_id": resID, "alias": "store"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "read", rc2DecodeMap(t, w)["access_type"], "access type defaults to read")

	// List.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/bad/resources", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodGet, "/api/v1/workflows/"+uuid.NewString()+"/resources", nil).Code)
	w = rc2Do(t, ra, http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, w.Code)
	list := rc2DecodeMap(t, w)["resources"].([]any)
	require.Len(t, list, 1)
	assert.Equal(t, "store", list[0].(map[string]any)["alias"])

	// Update alias.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPut, "/api/v1/workflows/bad/resources/"+resID, map[string]any{"alias": "x"}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPut, base+"/bad", map[string]any{"alias": "x"}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPut, base+"/"+resID, map[string]any{}).Code)
	w = rc2Do(t, ra, http.MethodPut, base+"/"+resID, map[string]any{"alias": "bad alias!"})
	assert.Equal(t, "VALIDATION_FAILED", rc2DecodeMap(t, w)["code"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodPut, "/api/v1/workflows/"+uuid.NewString()+"/resources/"+resID, map[string]any{"alias": "x"}).Code)
	w = rc2Do(t, ra, http.MethodPut, base+"/"+resID, map[string]any{"alias": "renamed"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "renamed", rc2DecodeMap(t, w)["alias"])

	// Detach.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodDelete, "/api/v1/workflows/bad/resources/"+resID, nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodDelete, base+"/bad", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodDelete, "/api/v1/workflows/"+uuid.NewString()+"/resources/"+resID, nil).Code)
	w = rc2Do(t, ra, http.MethodDelete, base+"/"+resID, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = rc2Do(t, ra, http.MethodGet, base, nil)
	assert.Empty(t, rc2DecodeMap(t, w)["resources"])

	// Missing path parameters (handlers registered on routes without them).
	for _, p := range []struct{ method, path string }{
		{http.MethodGet, "/np/wf"}, {http.MethodPut, "/np/wf"}, {http.MethodDelete, "/np/wf"},
		{http.MethodPost, "/np/publish"}, {http.MethodPost, "/np/unpublish"}, {http.MethodGet, "/np/diagram"},
		{http.MethodPost, "/np/attach"}, {http.MethodDelete, "/np/detach"}, {http.MethodDelete, "/np/detach/" + wfID},
		{http.MethodGet, "/np/resources"}, {http.MethodPut, "/np/alias"}, {http.MethodPut, "/np/alias/" + wfID},
		{http.MethodGet, "/np/exec"}, {http.MethodGet, "/np/logs"}, {http.MethodGet, "/np/noderesult"},
		{http.MethodGet, "/np/trigger"}, {http.MethodPut, "/np/trigger"}, {http.MethodDelete, "/np/trigger"},
		{http.MethodPost, "/np/enable"}, {http.MethodPost, "/np/disable"}, {http.MethodPost, "/np/manual"},
		{http.MethodGet, "/np/file"}, {http.MethodGet, "/np/filemeta"}, {http.MethodDelete, "/np/file"},
		{http.MethodGet, "/np/export"},
	} {
		w := rc2Do(t, raw, p.method, p.path, map[string]any{"alias": "a"})
		assert.Equal(t, http.StatusBadRequest, w.Code, p.path)
	}
}

func TestRC2_Workflows_DatabaseFailures(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	ra := env.router(alice, false, true)
	raw := env.router(alice, false, false)
	wfID := env.createWorkflow(t, ra, map[string]any{"name": "wf"})["id"].(string)
	resID := uuid.NewString()

	require.NoError(t, env.db.Close())

	// The ownership guard reports lookup failures as 500, never as access.
	w := rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/"+wfID, nil)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "INTERNAL_ERROR", rc2DecodeMap(t, w)["code"])

	for _, p := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/v1/workflows", map[string]any{"name": "x"}},
		{http.MethodGet, "/api/v1/workflows", nil},
		{http.MethodGet, "/api/v1/workflows/" + wfID, nil},
		{http.MethodPut, "/api/v1/workflows/" + wfID, map[string]any{"name": "x"}},
		{http.MethodDelete, "/api/v1/workflows/" + wfID, nil},
		{http.MethodPost, "/api/v1/workflows/" + wfID + "/publish", nil},
		{http.MethodPost, "/api/v1/workflows/" + wfID + "/unpublish", nil},
		{http.MethodGet, "/api/v1/workflows/" + wfID + "/diagram", nil},
		{http.MethodPost, "/api/v1/workflows/" + wfID + "/resources", map[string]any{"resource_id": resID, "alias": "a"}},
		{http.MethodGet, "/api/v1/workflows/" + wfID + "/resources", nil},
		{http.MethodPut, "/api/v1/workflows/" + wfID + "/resources/" + resID, map[string]any{"alias": "a"}},
		{http.MethodDelete, "/api/v1/workflows/" + wfID + "/resources/" + resID, nil},
	} {
		w := rc2Do(t, raw, p.method, p.path, p.body)
		assert.Equal(t, http.StatusInternalServerError, w.Code, "%s %s: %s", p.method, p.path, w.Body.String())
	}
}
