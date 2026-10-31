package rest

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rc2WaitExecution polls until the execution reaches a terminal state.
func rc2WaitExecution(t *testing.T, env *rc2Env, id string) string {
	t.Helper()
	execID := uuid.MustParse(id)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		m, err := env.execRepo.FindByID(context.Background(), execID)
		if err == nil && m != nil {
			switch m.Status {
			case "completed", "failed", "cancelled", "timeout":
				return m.Status
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("execution %s did not finish", id)
	return ""
}

func TestRC2_Executions_RunGetListAndNodeResults(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	ra := env.router(alice, false, true)
	rb := env.router(bob, false, true)
	raw := env.router(alice, false, false)
	wfID := env.createWorkflow(t, ra, rc2SimpleWorkflow("exec-wf"))["id"].(string)

	// Validation.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/executions", "{bad").Code)
	w := rc2Do(t, ra, http.MethodPost, "/api/v1/executions", map[string]any{"input": map[string]any{}})
	assert.Equal(t, "WORKFLOW_ID_REQUIRED", rc2DecodeMap(t, w)["code"])
	w = rc2Do(t, ra, http.MethodPost, "/api/v1/workflows/"+wfID+"/execute", map[string]any{
		"webhooks": []map[string]any{{"url": "http://example.invalid/hook", "events": []string{"bogus.event"}}},
	})
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodPost, "/api/v1/executions/run/"+uuid.NewString(), map[string]any{}).Code)
	// Another user cannot run alice's workflow.
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodPost, "/api/v1/executions/run/"+wfID, map[string]any{}).Code)

	// Run with a webhook subscription whose event filter never matches.
	w = rc2Do(t, ra, http.MethodPost, "/api/v1/executions/run/"+wfID, map[string]any{
		"input":     map[string]any{"x": 1},
		"variables": map[string]any{"v": 2},
		"webhooks": []map[string]any{{
			"url": "http://127.0.0.1:1/hook", "events": []string{"execution.timeout"},
			"headers": map[string]string{"X": "y"}, "node_ids": []string{"n1"},
		}},
	})
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	execID := rc2DecodeMap(t, w)["id"].(string)
	assert.Equal(t, "completed", rc2WaitExecution(t, env, execID))

	// Get.
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/executions/"+execID, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, wfID, rc2DecodeMap(t, w)["workflow_id"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodGet, "/api/v1/executions/"+execID, nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/executions/bad", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodGet, "/api/v1/executions/"+uuid.NewString(), nil).Code)

	// Logs.
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/executions/"+execID+"/logs", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, rc2DecodeMap(t, w), "logs")
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/executions/bad/logs", nil).Code)

	// Node results (route parameter is node_id, as in pkg/server).
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/executions/"+execID+"/nodes/n1/result", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "completed", rc2DecodeMap(t, w)["status"])
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/executions/bad/nodes/n1/result", nil).Code)
	assert.GreaterOrEqual(t, rc2Do(t, ra, http.MethodGet, "/api/v1/executions/"+execID+"/nodes/zz/result", nil).Code, 400)

	// List with filters and owner isolation.
	assert.Contains(t, rc2ListIDs(t, ra, "/api/v1/executions?workflow_id="+wfID+"&status=completed"), execID)
	assert.Empty(t, rc2ListIDs(t, rb, "/api/v1/executions"))
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/executions?workflow_id=bad", nil).Code)
	admin := env.router(env.user(t, true), true, true)
	assert.Contains(t, rc2ListIDs(t, admin, "/api/v1/executions"), execID)
	assert.Contains(t, rc2ListIDs(t, admin, "/api/v1/executions?workflow_id="+wfID), execID)
	assert.Contains(t, rc2ListIDs(t, admin, "/api/v1/executions?status=completed"), execID)

	// Query-parameter execution guard.
	assert.Equal(t, http.StatusNoContent, rc2Do(t, ra, http.MethodGet, "/np/exq?execution_id="+execID, nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodGet, "/np/exq?execution_id="+execID, nil).Code)
	assert.Equal(t, http.StatusNoContent, rc2Do(t, rb, http.MethodGet, "/np/exq?execution_id=bad", nil).Code, "malformed IDs are left to the handler")
	assert.Equal(t, http.StatusNoContent, rc2Do(t, rb, http.MethodGet, "/np/exq", nil).Code)

	// Not implemented endpoints.
	for _, p := range []string{"/cancel", "/retry"} {
		assert.Equal(t, http.StatusNotImplemented, rc2Do(t, ra, http.MethodPost, "/api/v1/executions/"+execID+p, nil).Code, p)
	}
	for _, p := range []string{"/watch", "/stream"} {
		assert.Equal(t, http.StatusNotImplemented, rc2Do(t, ra, http.MethodGet, "/api/v1/executions/"+execID+p, nil).Code, p)
	}
}

func rc2InlineWorkflow() map[string]any {
	return map[string]any{
		"name": "inline",
		"nodes": []map[string]any{
			{"id": "n1", "name": "First", "type": "transform", "config": map[string]any{"type": "passthrough"}},
			{"id": "n2", "name": "Second", "type": "transform", "config": map[string]any{"type": "passthrough"}},
		},
		"edges": []map[string]any{{"from": "n1", "to": "n2"}},
	}
}

func TestRC2_Executions_Ephemeral(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	ra := env.router(alice, false, true)
	rb := env.router(bob, false, true)
	const path = "/api/v1/executions/ephemeral"

	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, path, "{bad").Code)
	for body, code := range map[string]string{
		`{"workflow":{}}`:                            "MODE_REQUIRED",
		`{"workflow":{},"mode":"later"}`:             "INVALID_MODE",
		`{"workflow":"not-an-object","mode":"sync"}`: "INVALID_WORKFLOW",
	} {
		w := rc2Do(t, ra, http.MethodPost, path, body)
		require.Equal(t, http.StatusBadRequest, w.Code, body)
		assert.Equal(t, code, rc2DecodeMap(t, w)["code"], body)
	}
	big := `{"mode":"sync","workflow":{"name":"` + strings.Repeat("x", maxWorkflowSnapshotSize) + `"}}`
	w := rc2Do(t, ra, http.MethodPost, path, big)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	// Service-level validation errors are translated.
	w = rc2Do(t, ra, http.MethodPost, path, map[string]any{"mode": "sync", "workflow": rc2InlineWorkflow(), "credential_ids": []string{"c1"}})
	assert.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	// Sync run, persisted: the caller becomes the owner, whatever
	// created_by the client sends.
	wf := rc2InlineWorkflow()
	wf["created_by"] = bob
	w = rc2Do(t, ra, http.MethodPost, path, map[string]any{
		"mode": "sync", "workflow": wf, "input": map[string]any{"a": 1}, "persist_execution": true,
		"webhooks": []map[string]any{{"url": "http://127.0.0.1:1/hook", "events": []string{"execution.timeout"}}},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	exec := rc2DecodeMap(t, w)
	assert.Equal(t, "completed", exec["status"])
	execID := exec["id"].(string)

	// The inline execution is visible to its creator only.
	assert.Equal(t, http.StatusOK, rc2Do(t, ra, http.MethodGet, "/api/v1/executions/"+execID, nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodGet, "/api/v1/executions/"+execID, nil).Code)
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/executions/"+execID+"/nodes/n2/result", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Async run returns 202.
	w = rc2Do(t, ra, http.MethodPost, path, map[string]any{"mode": "async", "workflow": rc2InlineWorkflow(), "persist_execution": true})
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	rc2WaitExecution(t, env, rc2DecodeMap(t, w)["id"].(string))
}

func TestRC2_Executions_DatabaseFailures(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	raw := env.router(alice, false, false)
	id := uuid.NewString()

	require.NoError(t, env.db.Close())
	for _, p := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/v1/executions", nil},
		{http.MethodGet, "/api/v1/executions/" + id, nil},
		{http.MethodGet, "/api/v1/executions/" + id + "/nodes/n1/result", nil},
		{http.MethodPost, "/api/v1/executions/run/" + id, map[string]any{}},
		{http.MethodPost, "/api/v1/executions/ephemeral", map[string]any{"mode": "sync", "workflow": rc2InlineWorkflow(), "persist_execution": true}},
		{http.MethodGet, "/api/v1/triggers", nil},
		{http.MethodPost, "/api/v1/triggers", map[string]any{"workflow_id": id, "name": "t", "type": "manual"}},
		{http.MethodPut, "/api/v1/triggers/" + id, map[string]any{}},
		{http.MethodDelete, "/api/v1/triggers/" + id, nil},
		{http.MethodPost, "/api/v1/triggers/" + id + "/enable", nil},
		{http.MethodPost, "/api/v1/triggers/" + id + "/disable", nil},
	} {
		w := rc2Do(t, raw, p.method, p.path, p.body)
		assert.GreaterOrEqual(t, w.Code, 400, "%s %s", p.method, p.path)
		assert.NotEqual(t, http.StatusBadRequest, w.Code, "%s %s: %s", p.method, p.path, w.Body.String())
	}
	// Logs degrade to an empty list rather than failing.
	w := rc2Do(t, raw, http.MethodGet, "/api/v1/executions/"+id+"/logs", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, float64(0), rc2DecodeMap(t, w)["total"])
}

func TestRC2_Triggers_CRUDAndOwnership(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	ra := env.router(alice, false, true)
	rb := env.router(bob, false, true)
	raw := env.router(alice, false, false)
	wfID := env.createWorkflow(t, ra, map[string]any{"name": "wf"})["id"].(string)

	// Create validation.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/triggers", map[string]any{"name": "t"}).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodPost, "/api/v1/triggers", map[string]any{"workflow_id": wfID, "name": "t", "type": "manual"}).Code,
		"triggers cannot be attached to someone else's workflow")
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/triggers", map[string]any{"workflow_id": wfID, "name": "t", "type": "carrier-pigeon"}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/triggers", map[string]any{"workflow_id": wfID, "name": "t", "type": "cron", "config": map[string]any{"schedule": "not a cron"}}).Code)

	w := rc2Do(t, ra, http.MethodPost, "/api/v1/triggers", map[string]any{"workflow_id": wfID, "name": "manual", "type": "manual", "enabled": true})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	trID := rc2DecodeMap(t, w)["id"].(string)
	base := "/api/v1/triggers/" + trID

	// Get / list.
	w = rc2Do(t, ra, http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, wfID, rc2DecodeMap(t, w)["workflow_id"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodGet, base, nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/triggers/bad", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodGet, "/api/v1/triggers/"+uuid.NewString(), nil).Code)
	assert.Equal(t, []string{trID}, rc2ListIDs(t, ra, "/api/v1/triggers?workflow_id="+wfID+"&type=manual&limit=5"))
	assert.Empty(t, rc2ListIDs(t, rb, "/api/v1/triggers"))
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/triggers?workflow_id=bad", nil).Code)

	// Manual execution of an enabled trigger is not implemented yet.
	w = rc2Do(t, ra, http.MethodPost, base+"/execute", "not json")
	assert.Equal(t, http.StatusNotImplemented, w.Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/triggers/bad/execute", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodPost, "/api/v1/triggers/"+uuid.NewString()+"/execute", nil).Code)

	// Disable / enable.
	w = rc2Do(t, ra, http.MethodPost, base+"/disable", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, false, rc2DecodeMap(t, w)["enabled"])
	w = rc2Do(t, ra, http.MethodPost, base+"/execute", map[string]any{"input": map[string]any{}})
	assert.Equal(t, "TRIGGER_DISABLED", rc2DecodeMap(t, w)["code"])
	w = rc2Do(t, ra, http.MethodPost, base+"/enable", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, true, rc2DecodeMap(t, w)["enabled"])
	for _, p := range []string{"/enable", "/disable"} {
		assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPost, "/api/v1/triggers/bad"+p, nil).Code, p)
		assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodPost, "/api/v1/triggers/"+uuid.NewString()+p, nil).Code, p)
	}

	// Update.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPut, "/api/v1/triggers/bad", map[string]any{}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPut, base, "{bad").Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodPut, "/api/v1/triggers/"+uuid.NewString(), map[string]any{}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodPut, base, map[string]any{"type": "nope"}).Code)
	w = rc2Do(t, ra, http.MethodPut, base, map[string]any{"name": "renamed", "enabled": false, "config": map[string]any{"k": "v"}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	upd := rc2DecodeMap(t, w)
	assert.Equal(t, "renamed", upd["name"])
	assert.Equal(t, false, upd["enabled"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodPut, base, map[string]any{"name": "hijack"}).Code)

	// Delete.
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodDelete, base, nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodDelete, "/api/v1/triggers/bad", nil).Code)
	require.Equal(t, http.StatusOK, rc2Do(t, ra, http.MethodDelete, base, nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, ra, http.MethodGet, base, nil).Code)
}
