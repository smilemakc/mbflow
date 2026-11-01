package rest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
)

const rc2ImportYAML = `
# exported by hand
metadata:
  name: "Imported RC2"
  description: "round trip"
  version: 3
variables:
  region: eu
nodes:
  - id: start
    name: "Start"
    type: transform
    config:
      type: passthrough
    position: {x: 10, y: 20}
  - id: finish
    name: "Finish"
    type: transform
    config:
      type: passthrough
edges:
  - id: e1
    from: start
    to: finish
    source_handle: out
    condition: "input.ok == true"
trigger:
  name: "Nightly"
  description: "runs at night"
  type: cron
  config:
    schedule: "0 3 * * *"
`

// rc2TriggerRepo is a minimal trigger repository with error injection.
type rc2TriggerRepo struct {
	repository.TriggerRepository
	createErr error
	findErr   error
	triggers  []*storagemodels.TriggerModel
}

func (r *rc2TriggerRepo) Create(ctx context.Context, tm *storagemodels.TriggerModel) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.triggers = append(r.triggers, tm)
	return nil
}

func (r *rc2TriggerRepo) FindByWorkflowID(ctx context.Context, id uuid.UUID) ([]*storagemodels.TriggerModel, error) {
	return r.triggers, r.findErr
}

func TestRC2_Import_RequestValidation(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	r := env.router(env.user(t, false), false, true)
	const path = "/api/v1/workflows/import"

	for _, tc := range []struct {
		name, ct, body, code string
	}{
		{"json content type", "application/json", "{}", "INVALID_CONTENT_TYPE"},
		{"empty body", "application/x-yaml", "", "EMPTY_CONTENT"},
		{"whitespace only", "text/yaml", "  \n\t ", "PARSE_ERROR"},
		{"invalid yaml", "text/plain", "metadata: [unclosed", "IMPORT_ERROR"},
		{"unknown node type", "application/x-yaml", "metadata: {name: x}\nnodes:\n  - {id: a, name: A, type: teleport}\n", "IMPORT_ERROR"},
	} {
		w := rc2DoHeaders(t, r, http.MethodPost, path, tc.body, map[string]string{"Content-Type": tc.ct})
		require.Equal(t, http.StatusBadRequest, w.Code, tc.name)
		assert.Equal(t, tc.code, rc2DecodeMap(t, w)["code"], tc.name)
	}

	body, ct := rc2Multipart(t, "", nil, map[string]string{"x": "y"})
	w := rc2PostMultipart(t, r, path, body, ct)
	assert.Equal(t, "FILE_REQUIRED", rc2DecodeMap(t, w)["code"])
	body, ct = rc2Multipart(t, "workflow.json", []byte(rc2ImportYAML), nil)
	w = rc2PostMultipart(t, r, path, body, ct)
	assert.Equal(t, "INVALID_FILE_TYPE", rc2DecodeMap(t, w)["code"])
	body, ct = rc2Multipart(t, "WORKFLOW.YML", []byte(rc2ImportYAML), nil)
	w = rc2PostMultipart(t, r, path, body, ct)
	assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	w = rc2Do(t, r, http.MethodGet, "/api/v1/workflows/import/types", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, rc2DecodeMap(t, w)["node_types"], "transform")
}

func TestRC2_Import_ExportRoundTrip(t *testing.T) {
	t.Parallel()
	env := rc2NewEnv(t)
	alice := env.user(t, false)
	bob := env.user(t, false)
	ra := env.router(alice, false, true)
	rb := env.router(bob, false, true)
	raw := env.router(alice, false, false)

	w := rc2DoHeaders(t, ra, http.MethodPost, "/api/v1/workflows/import", rc2ImportYAML, map[string]string{"Content-Type": "application/x-yaml"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	imported := rc2DecodeMap(t, w)
	wfID := imported["workflow_id"].(string)
	assert.Equal(t, "Imported RC2", imported["name"])
	assert.Equal(t, "draft", imported["status"])
	assert.Equal(t, float64(2), imported["nodes_count"])
	assert.Equal(t, float64(1), imported["edges_count"])
	assert.NotEmpty(t, imported["trigger_id"])

	// The importing user owns the workflow; the edge keeps its handle.
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/"+wfID, nil)
	require.Equal(t, http.StatusOK, w.Code)
	wf := rc2DecodeMap(t, w)
	assert.Equal(t, alice, wf["created_by"])
	edge := wf["edges"].([]any)[0].(map[string]any)
	assert.Equal(t, "out", edge["source_handle"], "source handles survive the import")
	assert.Equal(t, "input.ok == true", edge["condition"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, rb, http.MethodGet, "/api/v1/workflows/"+wfID+"/export", nil).Code)

	// YAML export (default format) contains the full workflow definition.
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/"+wfID+"/export", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "application/x-yaml", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "workflow.yaml")
	var exported struct {
		Metadata struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
		Nodes []struct {
			ID   string `yaml:"id"`
			Type string `yaml:"type"`
		} `yaml:"nodes"`
		Edges []struct {
			From         string `yaml:"from"`
			SourceHandle string `yaml:"source_handle"`
		} `yaml:"edges"`
		Trigger *struct {
			Name        string         `yaml:"name"`
			Description string         `yaml:"description"`
			Type        string         `yaml:"type"`
			Config      map[string]any `yaml:"config"`
		} `yaml:"trigger"`
	}
	require.NoError(t, yaml.Unmarshal(w.Body.Bytes(), &exported), w.Body.String())
	assert.Equal(t, "Imported RC2", exported.Metadata.Name)
	require.Len(t, exported.Nodes, 2)
	require.Len(t, exported.Edges, 1)
	assert.Equal(t, "out", exported.Edges[0].SourceHandle)
	require.NotNil(t, exported.Trigger)
	assert.Equal(t, "Nightly", exported.Trigger.Name)
	assert.Equal(t, "runs at night", exported.Trigger.Description)
	assert.Equal(t, "cron", exported.Trigger.Type)
	assert.Equal(t, map[string]any{"schedule": "0 3 * * *"}, exported.Trigger.Config, "name/description are not duplicated in the config")

	// The export can be imported again.
	w2 := rc2DoHeaders(t, ra, http.MethodPost, "/api/v1/workflows/import", w.Body.String(), map[string]string{"Content-Type": "application/x-yaml"})
	require.Equal(t, http.StatusCreated, w2.Code, w2.Body.String())
	assert.Equal(t, float64(2), rc2DecodeMap(t, w2)["nodes_count"])

	// JSON export.
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/"+wfID+"/export?format=JSON", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Disposition"), "workflow.json")
	jsonExport := rc2DecodeMap(t, w)
	assert.Equal(t, "Imported RC2", jsonExport["workflow"].(map[string]any)["name"])
	assert.Equal(t, "Nightly", jsonExport["trigger"].(map[string]any)["Name"])

	// Errors.
	w = rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/"+wfID+"/export?format=xml", nil)
	assert.Equal(t, "INVALID_FORMAT", rc2DecodeMap(t, w)["code"])
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, ra, http.MethodGet, "/api/v1/workflows/bad/export", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, raw, http.MethodGet, "/api/v1/workflows/"+uuid.NewString()+"/export", nil).Code)

	// Workflow persistence failure.
	require.NoError(t, env.db.Close())
	w = rc2DoHeaders(t, raw, http.MethodPost, "/api/v1/workflows/import", rc2ImportYAML, map[string]string{"Content-Type": "application/x-yaml"})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRC2_Import_TriggerFailureRollsBackWorkflow(t *testing.T) {
	t.Parallel()
	repo := rc2NewMemWorkflowRepo()
	triggers := &rc2TriggerRepo{createErr: errors.New("trigger insert failed")}
	env := rc2NewEnv(t)
	h := NewImportHandlers(repo, triggers, env.log, env.executors)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/import", h.HandleImportWorkflow)
	r.GET("/export/:workflow_id", h.HandleExportWorkflow)

	w := rc2DoHeaders(t, r, http.MethodPost, "/import", rc2ImportYAML, map[string]string{"Content-Type": "application/x-yaml"})
	require.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	repo.mu.Lock()
	remaining := len(repo.workflows)
	repo.mu.Unlock()
	assert.Equal(t, 0, remaining, "the workflow is removed when its trigger cannot be saved")

	// Without a trigger section the import succeeds; a failing trigger
	// lookup does not prevent the export.
	triggers.createErr = nil
	triggers.findErr = errors.New("lookup failed")
	noTrigger := "metadata: {name: plain}\nnodes:\n  - {id: a, name: A, type: transform, config: {type: passthrough}}\n"
	w = rc2DoHeaders(t, r, http.MethodPost, "/import", noTrigger, map[string]string{"Content-Type": "text/yaml"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	wfID := rc2DecodeMap(t, w)["workflow_id"].(string)
	assert.Nil(t, rc2DecodeMap(t, w)["trigger_id"])
	w = rc2Do(t, r, http.MethodGet, "/export/"+wfID+"?format=json", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, rc2DecodeMap(t, w), "trigger")
}

func TestRC2_ExportTrigger_ToDomain(t *testing.T) {
	t.Parallel()
	var none *exportTrigger
	assert.Nil(t, none.toDomain())

	et := newExportTrigger(&storagemodels.TriggerModel{
		Type: "webhook", Enabled: true,
		Config: storagemodels.JSONBMap{"name": "n", "description": "d", "secret": "x"},
	})
	assert.Equal(t, "n", et.Name)
	assert.Equal(t, "d", et.Description)
	d := et.toDomain()
	assert.Equal(t, "n", d.Name)
	assert.Equal(t, map[string]any{"secret": "x"}, d.Config)
	assert.True(t, d.Enabled)
}
