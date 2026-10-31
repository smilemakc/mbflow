package rest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerRun_StartsExecution(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	ra := env.router(alice, false, true)
	rb := env.router(bob, false, true)
	wfID := env.createWorkflow(t, ra, rc2SimpleWorkflow("run-now"))["id"].(string)

	w := rc2Do(t, ra, http.MethodPost, "/api/v1/triggers", map[string]any{
		"workflow_id": wfID, "name": "Run me", "description": "manual run", "type": "manual", "enabled": true,
		"config": map[string]any{"input": map[string]any{"default": "d", "override": "old"}},
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	trID := rc2DecodeMap(t, w)["id"].(string)
	base := "/api/v1/triggers/" + trID

	// Another user cannot run alice's trigger.
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodPost, base+"/execute", nil).Code)

	w = rc2Do(t, ra, http.MethodPost, base+"/execute", map[string]any{"input": map[string]any{"override": "new"}})
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	exec := rc2DecodeMap(t, w)
	execID, _ := exec["id"].(string)
	require.NotEmpty(t, execID)
	assert.NotEmpty(t, exec["status"])
	assert.Equal(t, wfID, exec["workflow_id"])
	assert.Equal(t, "completed", rc2WaitExecution(t, env, execID))

	stored, err := env.execRepo.FindByID(context.Background(), uuid.MustParse(execID))
	require.NoError(t, err)
	require.NotNil(t, stored.TriggerID)
	assert.Equal(t, trID, stored.TriggerID.String())
	assert.Equal(t, "d", stored.InputData["default"], "default config.input is applied")
	assert.Equal(t, "new", stored.InputData["override"], "request input wins")

	// last_run is recorded and the name survives a read.
	got := rc2DecodeMap(t, rc2Do(t, ra, http.MethodGet, base, nil))
	assert.Equal(t, "Run me", got["name"])
	assert.Equal(t, "manual run", got["description"])
	lastRun, ok := got["last_run"].(string)
	require.True(t, ok, "last_run missing: %v", got)
	ts, err := time.Parse(time.RFC3339Nano, lastRun)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), ts, time.Minute)
	assert.Nil(t, got["next_run"], "manual triggers have no next run")

	// An empty body runs with the default input.
	w = rc2Do(t, ra, http.MethodPost, base+"/execute", nil)
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	rc2WaitExecution(t, env, rc2DecodeMap(t, w)["id"].(string))
}

func TestTriggerUpdate_NameDescriptionAndNextRun(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	ra := env.router(alice, false, true)
	wfID := env.createWorkflow(t, ra, rc2SimpleWorkflow("names"))["id"].(string)

	w := rc2Do(t, ra, http.MethodPost, "/api/v1/triggers", map[string]any{
		"workflow_id": wfID, "name": "Every 5m", "description": "poll", "type": "interval", "enabled": true,
		"config": map[string]any{"interval": "5m"},
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	created := rc2DecodeMap(t, w)
	trID := created["id"].(string)
	base := "/api/v1/triggers/" + trID

	nextRun, ok := created["next_run"].(string)
	require.True(t, ok, "next_run missing on create: %v", created)
	ts, err := time.Parse(time.RFC3339Nano, nextRun)
	require.NoError(t, err)
	assert.True(t, ts.After(time.Now().Add(-time.Second)))
	assert.LessOrEqual(t, time.Until(ts), 5*time.Minute)
	assert.Zero(t, ts.Unix()%300, "interval ticks are aligned to multiples of the interval")

	// Updating other fields keeps name and description.
	w = rc2Do(t, ra, http.MethodPut, base, map[string]any{"config": map[string]any{"interval": "1h"}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	upd := rc2DecodeMap(t, w)
	assert.Equal(t, "Every 5m", upd["name"])
	assert.Equal(t, "poll", upd["description"])

	// An explicit empty description clears it; a new name replaces it.
	w = rc2Do(t, ra, http.MethodPut, base, map[string]any{"name": "Hourly", "description": ""})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	upd = rc2DecodeMap(t, w)
	assert.Equal(t, "Hourly", upd["name"])
	assert.Nil(t, upd["description"])

	// Listing returns the stored name; disabled triggers have no next run.
	require.Equal(t, http.StatusOK, rc2Do(t, ra, http.MethodPost, base+"/disable", nil).Code)
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/triggers?workflow_id="+wfID, nil)
	require.Equal(t, http.StatusOK, w.Code)
	list := rc2DecodeMap(t, w)["data"].([]any)
	require.Len(t, list, 1)
	item := list[0].(map[string]any)
	assert.Equal(t, "Hourly", item["name"])
	assert.Nil(t, item["next_run"])
}
