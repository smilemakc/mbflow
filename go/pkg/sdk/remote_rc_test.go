package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/smilemakc/mbflow/go/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rcAPIKey = "rc-test-key"

// rcFakeAPI emulates the subset of the MBFlow REST API (pkg/server/routes.go)
// used by the SDK's remote mode. Request bodies are decoded into structs that
// mirror the REST handlers' binding structs, so shape mismatches (e.g. an edge
// condition sent as a string) fail the same way they would on the server.
type rcFakeAPI struct {
	t *testing.T

	mu        sync.Mutex
	workflows map[string]*models.Workflow
	triggers  map[string]*models.Trigger
	execs     map[string]*models.Execution
	// execPolls counts GETs per execution; the execution becomes completed
	// after completeAfter polls.
	execPolls     map[string]int
	completeAfter int
	seq           int
	requests      []string
	lastQuery     map[string]string
	lastBody      []byte
}

func rcNewFakeAPI(t *testing.T) (*rcFakeAPI, *httptest.Server) {
	t.Helper()
	f := &rcFakeAPI{
		t:             t,
		workflows:     map[string]*models.Workflow{},
		triggers:      map[string]*models.Trigger{},
		execs:         map[string]*models.Execution{},
		execPolls:     map[string]int{},
		completeAfter: 2,
	}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *rcFakeAPI) nextID(prefix string) string {
	f.seq++
	return fmt.Sprintf("%s-%d", prefix, f.seq)
}

func rcWriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func rcWriteAPIError(w http.ResponseWriter, status int, code, msg string) {
	rcWriteJSON(w, status, map[string]any{"code": code, "message": msg})
}

type rcNodeReq struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Config   map[string]any `json:"config,omitempty"`
	Position map[string]any `json:"position,omitempty"`
}

type rcEdgeReq struct {
	ID           string         `json:"id"`
	From         string         `json:"from"`
	To           string         `json:"to"`
	SourceHandle string         `json:"source_handle,omitempty"`
	Condition    map[string]any `json:"condition,omitempty"`
	Loop         *struct {
		MaxIterations int `json:"max_iterations"`
	} `json:"loop,omitempty"`
}

type rcWorkflowReq struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Nodes       []rcNodeReq    `json:"nodes,omitempty"`
	Edges       []rcEdgeReq    `json:"edges,omitempty"`
	Resources   []struct {
		ResourceID string `json:"resource_id"`
		Alias      string `json:"alias"`
		AccessType string `json:"access_type"`
	} `json:"resources,omitempty"`
}

func (r *rcWorkflowReq) apply(wf *models.Workflow) {
	if r.Name != "" {
		wf.Name = r.Name
	}
	wf.Description = r.Description
	wf.Variables = r.Variables
	wf.Metadata = r.Metadata
	if r.Nodes != nil {
		wf.Nodes = nil
		for _, n := range r.Nodes {
			node := &models.Node{ID: n.ID, Name: n.Name, Type: n.Type, Config: n.Config}
			if n.Position != nil {
				x, _ := n.Position["x"].(float64)
				y, _ := n.Position["y"].(float64)
				node.Position = &models.Position{X: x, Y: y}
			}
			wf.Nodes = append(wf.Nodes, node)
		}
	}
	if r.Edges != nil {
		wf.Edges = nil
		for _, e := range r.Edges {
			edge := &models.Edge{ID: e.ID, From: e.From, To: e.To, SourceHandle: e.SourceHandle}
			if e.Condition != nil {
				edge.Condition, _ = e.Condition["expression"].(string)
			}
			if e.Loop != nil {
				edge.Loop = &models.LoopConfig{MaxIterations: e.Loop.MaxIterations}
			}
			wf.Edges = append(wf.Edges, edge)
		}
	}
	if r.Resources != nil {
		wf.Resources = nil
		for _, res := range r.Resources {
			wf.Resources = append(wf.Resources, models.WorkflowResource{ResourceID: res.ResourceID, Alias: res.Alias, AccessType: res.AccessType})
		}
	}
}

type rcTriggerReq struct {
	WorkflowID  string         `json:"workflow_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	Enabled     *bool          `json:"enabled,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (f *rcFakeAPI) decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		rcWriteAPIError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return false
	}
	raw, _ := json.Marshal(v)
	f.lastBody = raw
	return true
}

func rcPage[T any](items []T, r *http.Request) map[string]any {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	total := len(items)
	if offset > len(items) {
		offset = len(items)
	}
	items = items[offset:]
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return map[string]any{"data": items, "total": total, "limit": limit, "offset": offset}
}

func (f *rcFakeAPI) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.lastQuery = map[string]string{}
	for k := range r.URL.Query() {
		f.lastQuery[k] = r.URL.Query().Get(k)
	}

	if r.URL.Path == "/health" {
		rcWriteJSON(w, http.StatusOK, map[string]any{"status": "ok", "version": "9.9.9"})
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+rcAPIKey {
		rcWriteAPIError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid token")
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/")
	switch parts[0] {
	case "workflows":
		f.serveWorkflows(w, r, parts[1:])
	case "triggers":
		f.serveTriggers(w, r, parts[1:])
	case "executions":
		f.serveExecutions(w, r, parts[1:])
	default:
		rcWriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
	}
}

func (f *rcFakeAPI) serveWorkflows(w http.ResponseWriter, r *http.Request, parts []string) {
	switch {
	case len(parts) == 0 && r.Method == http.MethodPost:
		var req rcWorkflowReq
		if !f.decode(w, r, &req) {
			return
		}
		if req.Name == "" {
			rcWriteAPIError(w, http.StatusBadRequest, "ERROR", "name is required")
			return
		}
		wf := &models.Workflow{ID: f.nextID("wf"), Status: models.WorkflowStatusDraft, Version: 1, CreatedAt: time.Now().UTC()}
		req.apply(wf)
		f.workflows[wf.ID] = wf
		rcWriteJSON(w, http.StatusCreated, wf)
	case len(parts) == 0 && r.Method == http.MethodGet:
		var list []*models.Workflow
		status := r.URL.Query().Get("status")
		for _, id := range rcSortedKeys(f.workflows) {
			wf := f.workflows[id]
			if status == "" || string(wf.Status) == status {
				list = append(list, wf)
			}
		}
		rcWriteJSON(w, http.StatusOK, rcPage(list, r))
	case len(parts) == 1:
		wf, ok := f.workflows[parts[0]]
		if !ok {
			rcWriteAPIError(w, http.StatusNotFound, "WORKFLOW_NOT_FOUND", "workflow not found")
			return
		}
		switch r.Method {
		case http.MethodGet:
			rcWriteJSON(w, http.StatusOK, wf)
		case http.MethodPut:
			var req rcWorkflowReq
			if !f.decode(w, r, &req) {
				return
			}
			req.apply(wf)
			wf.Version++
			rcWriteJSON(w, http.StatusOK, wf)
		case http.MethodDelete:
			delete(f.workflows, wf.ID)
			rcWriteJSON(w, http.StatusOK, map[string]any{"message": "workflow deleted successfully"})
		default:
			rcWriteAPIError(w, http.StatusMethodNotAllowed, "METHOD", "not allowed")
		}
	default:
		rcWriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
	}
}

func (f *rcFakeAPI) serveTriggers(w http.ResponseWriter, r *http.Request, parts []string) {
	switch {
	case len(parts) == 0 && r.Method == http.MethodPost:
		var req rcTriggerReq
		if !f.decode(w, r, &req) {
			return
		}
		tr := &models.Trigger{
			ID: f.nextID("tr"), WorkflowID: req.WorkflowID, Name: req.Name, Description: req.Description,
			Type: models.TriggerType(req.Type), Config: req.Config, Metadata: req.Metadata,
		}
		if req.Enabled != nil {
			tr.Enabled = *req.Enabled
		}
		f.triggers[tr.ID] = tr
		rcWriteJSON(w, http.StatusCreated, tr)
	case len(parts) == 0 && r.Method == http.MethodGet:
		var list []*models.Trigger
		q := r.URL.Query()
		for _, id := range rcSortedKeys(f.triggers) {
			tr := f.triggers[id]
			if wfID := q.Get("workflow_id"); wfID != "" && tr.WorkflowID != wfID {
				continue
			}
			if typ := q.Get("type"); typ != "" && string(tr.Type) != typ {
				continue
			}
			list = append(list, tr)
		}
		rcWriteJSON(w, http.StatusOK, rcPage(list, r))
	case len(parts) >= 1:
		tr, ok := f.triggers[parts[0]]
		if !ok {
			rcWriteAPIError(w, http.StatusNotFound, "TRIGGER_NOT_FOUND", "trigger not found")
			return
		}
		action := ""
		if len(parts) == 2 {
			action = parts[1]
		}
		switch {
		case action == "" && r.Method == http.MethodGet:
			rcWriteJSON(w, http.StatusOK, tr)
		case action == "" && r.Method == http.MethodPut:
			var req rcTriggerReq
			if !f.decode(w, r, &req) {
				return
			}
			if req.Name != "" {
				tr.Name = req.Name
			}
			tr.Description = req.Description
			if req.Config != nil {
				tr.Config = req.Config
			}
			if req.Enabled != nil {
				tr.Enabled = *req.Enabled
			}
			rcWriteJSON(w, http.StatusOK, tr)
		case action == "" && r.Method == http.MethodDelete:
			delete(f.triggers, tr.ID)
			rcWriteJSON(w, http.StatusOK, map[string]any{"message": "trigger deleted successfully"})
		case action == "enable" && r.Method == http.MethodPost:
			tr.Enabled = true
			rcWriteJSON(w, http.StatusOK, tr)
		case action == "disable" && r.Method == http.MethodPost:
			tr.Enabled = false
			rcWriteJSON(w, http.StatusOK, tr)
		case action == "execute" && r.Method == http.MethodPost:
			var req struct {
				Input map[string]any `json:"input"`
			}
			if !f.decode(w, r, &req) {
				return
			}
			if !tr.Enabled {
				rcWriteAPIError(w, http.StatusBadRequest, "TRIGGER_DISABLED", "trigger is disabled")
				return
			}
			// Mirrors the current server behaviour.
			rcWriteAPIError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "trigger execution requires trigger manager integration")
		default:
			rcWriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		}
	default:
		rcWriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
	}
}

func (f *rcFakeAPI) serveExecutions(w http.ResponseWriter, r *http.Request, parts []string) {
	switch {
	case len(parts) == 2 && parts[0] == "run" && r.Method == http.MethodPost:
		var req ExecutionRequest
		if !f.decode(w, r, &req) {
			return
		}
		if _, ok := f.workflows[parts[1]]; !ok {
			rcWriteAPIError(w, http.StatusNotFound, "WORKFLOW_NOT_FOUND", "workflow not found")
			return
		}
		ex := &models.Execution{ID: f.nextID("ex"), WorkflowID: parts[1], Status: models.ExecutionStatusPending, Input: req.Input}
		f.execs[ex.ID] = ex
		rcWriteJSON(w, http.StatusAccepted, ex)
	case len(parts) == 0 && r.Method == http.MethodGet:
		var list []*models.Execution
		for _, id := range rcSortedKeys(f.execs) {
			list = append(list, f.execs[id])
		}
		rcWriteJSON(w, http.StatusOK, rcPage(list, r))
	case len(parts) >= 1:
		ex, ok := f.execs[parts[0]]
		if !ok {
			rcWriteAPIError(w, http.StatusNotFound, "EXECUTION_NOT_FOUND", "execution not found")
			return
		}
		rest := strings.Join(parts[1:], "/")
		switch {
		case rest == "" && r.Method == http.MethodGet:
			f.execPolls[ex.ID]++
			switch n := f.execPolls[ex.ID]; {
			case n >= f.completeAfter:
				ex.Status = models.ExecutionStatusCompleted
				ex.Output = map[string]any{"ok": true}
			default:
				ex.Status = models.ExecutionStatusRunning
			}
			rcWriteJSON(w, http.StatusOK, ex)
		case rest == "logs" && r.Method == http.MethodGet:
			rcWriteJSON(w, http.StatusOK, map[string]any{
				"logs": []map[string]any{
					{"timestamp": "2026-01-02T03:04:05Z", "event_type": "execution.started", "level": "info", "message": "started", "data": map[string]any{"k": "v"}},
				},
				"total": 1,
			})
		case strings.HasPrefix(rest, "nodes/") && strings.HasSuffix(rest, "/result") && r.Method == http.MethodGet:
			nodeID := strings.TrimSuffix(strings.TrimPrefix(rest, "nodes/"), "/result")
			rcWriteJSON(w, http.StatusOK, &models.NodeExecution{ExecutionID: ex.ID, NodeID: nodeID, Status: models.NodeExecutionStatusCompleted})
		case (rest == "cancel" || rest == "retry") && r.Method == http.MethodPost:
			rcWriteAPIError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "execution "+rest+" not yet implemented")
		default:
			rcWriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		}
	default:
		rcWriteAPIError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
	}
}

func rcSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// simple insertion sort keeps the helper dependency-free
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

func rcRemoteClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := NewClient(WithHTTPEndpoint(baseURL), WithAPIKey(rcAPIKey))
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func rcSampleWorkflow() *models.Workflow {
	return &models.Workflow{
		Name:        "rc-flow",
		Description: "desc",
		Tags:        []string{"a", "b"},
		Variables:   map[string]any{"v": "1"},
		Nodes: []*models.Node{
			{ID: "n1", Name: "Start", Type: "transform", Config: map[string]any{"x": 1.0}, Position: &models.Position{X: 10, Y: 20}},
			{ID: "n2", Name: "End", Type: "transform"},
		},
		Edges: []*models.Edge{
			{ID: "e1", From: "n1", To: "n2", Condition: "output.ok == true", SourceHandle: "out"},
		},
	}
}

func TestRCWorkflowRemote_CRUD(t *testing.T) {
	api, srv := rcNewFakeAPI(t)
	c := rcRemoteClient(t, srv.URL)
	ctx := context.Background()

	created, err := c.Workflows().Create(ctx, rcSampleWorkflow())
	require.NoError(t, err)
	assert.Equal(t, "wf-1", created.ID)
	assert.Equal(t, "rc-flow", created.Name)
	require.Len(t, created.Nodes, 2)
	require.NotNil(t, created.Nodes[0].Position)
	assert.Equal(t, 10.0, created.Nodes[0].Position.X)
	require.Len(t, created.Edges, 1)
	// The condition must have been sent as {"expression": ...} and round-tripped.
	assert.Equal(t, "output.ok == true", created.Edges[0].Condition)
	assert.Equal(t, "out", created.Edges[0].SourceHandle)
	assert.Contains(t, string(api.lastBody), `"condition":{"expression":"output.ok == true"}`)

	got, err := c.Workflows().Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)

	topo, err := c.Workflows().GetTopology(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"n1", "n2"}, topo)

	// Update, including resources and a loop edge.
	upd := rcSampleWorkflow()
	upd.ID = created.ID
	upd.Name = "renamed"
	upd.Edges[0].Condition = ""
	upd.Edges[0].Loop = &models.LoopConfig{MaxIterations: 3}
	upd.Resources = []models.WorkflowResource{{ResourceID: "res-1", Alias: "creds", AccessType: "read"}}
	updated, err := c.Workflows().Update(ctx, upd)
	require.NoError(t, err)
	assert.Equal(t, "renamed", updated.Name)
	assert.Equal(t, 2, updated.Version)
	require.Len(t, updated.Resources, 1)
	assert.Equal(t, "creds", updated.Resources[0].Alias)
	require.NotNil(t, updated.Edges[0].Loop)
	assert.Equal(t, 3, updated.Edges[0].Loop.MaxIterations)

	require.NoError(t, c.Workflows().Delete(ctx, created.ID))

	_, err = c.Workflows().Get(ctx, created.ID)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
	assert.Equal(t, "WORKFLOW_NOT_FOUND", apiErr.Code)
	assert.False(t, errors.Is(err, ErrNotSupportedByAPI))

	err = c.Workflows().Delete(ctx, created.ID)
	require.ErrorAs(t, err, &apiErr)

	_, err = c.Workflows().Update(ctx, upd)
	require.ErrorAs(t, err, &apiErr)

	assert.Contains(t, api.requests, "POST /api/v1/workflows")
	assert.Contains(t, api.requests, "PUT /api/v1/workflows/wf-1")
	assert.Contains(t, api.requests, "DELETE /api/v1/workflows/wf-1")
}

func TestRCWorkflowRemote_List(t *testing.T) {
	api, srv := rcNewFakeAPI(t)
	c := rcRemoteClient(t, srv.URL)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := c.Workflows().Create(ctx, rcSampleWorkflow())
		require.NoError(t, err)
	}
	// Server does not persist tags; give one workflow tags to test the filter.
	api.mu.Lock()
	api.workflows["wf-2"].Tags = []string{"a", "b", "c"}
	api.mu.Unlock()

	all, err := c.Workflows().List(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	page, err := c.Workflows().List(ctx, &ListOptions{Limit: 1, Offset: 1, Status: "draft"})
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, "wf-2", page[0].ID)
	assert.Equal(t, map[string]string{"limit": "1", "offset": "1", "status": "draft"}, api.lastQuery)

	tagged, err := c.Workflows().List(ctx, &ListOptions{Tags: []string{"a", "c"}})
	require.NoError(t, err)
	require.Len(t, tagged, 1)
	assert.Equal(t, "wf-2", tagged[0].ID)

	none, err := c.Workflows().List(ctx, &ListOptions{Status: "archived"})
	require.NoError(t, err)
	assert.Empty(t, none)
	assert.NotNil(t, none)
}

func TestRCWorkflowRemote_Unauthorized(t *testing.T) {
	_, srv := rcNewFakeAPI(t)
	c, err := NewClient(WithHTTPEndpoint(srv.URL), WithAPIKey("wrong"))
	require.NoError(t, err)
	defer c.Close()

	_, err = c.Workflows().List(context.Background(), nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Contains(t, err.Error(), "UNAUTHORIZED")
}

func TestRCTriggerRemote_Lifecycle(t *testing.T) {
	api, srv := rcNewFakeAPI(t)
	c := rcRemoteClient(t, srv.URL)
	ctx := context.Background()

	cron, err := c.Triggers().Create(ctx, &models.Trigger{
		WorkflowID: "wf-x", Name: "nightly", Type: models.TriggerTypeCron,
		Config: map[string]any{"schedule": "0 0 * * *"}, Enabled: false,
		Metadata: map[string]any{"m": "1"},
	})
	require.NoError(t, err)
	assert.Equal(t, "tr-1", cron.ID)
	assert.False(t, cron.Enabled)
	// enabled=false must be sent explicitly.
	assert.Contains(t, string(api.lastBody), `"enabled":false`)

	hook, err := c.Triggers().Create(ctx, &models.Trigger{
		WorkflowID: "wf-y", Name: "hook", Type: models.TriggerTypeWebhook, Enabled: true,
	})
	require.NoError(t, err)

	// Invalid cron is rejected client-side before any request.
	before := len(api.requests)
	_, err = c.Triggers().Create(ctx, &models.Trigger{
		WorkflowID: "wf-x", Name: "bad", Type: models.TriggerTypeCron,
		Config: map[string]any{"schedule": "every minute"},
	})
	require.Error(t, err)
	assert.Len(t, api.requests, before)

	got, err := c.Triggers().Get(ctx, cron.ID)
	require.NoError(t, err)
	assert.Equal(t, "nightly", got.Name)

	all, err := c.Triggers().List(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	byWF, err := c.Triggers().List(ctx, &TriggerListOptions{WorkflowID: "wf-y", Type: "webhook", Limit: 5, Offset: 0})
	require.NoError(t, err)
	require.Len(t, byWF, 1)
	assert.Equal(t, hook.ID, byWF[0].ID)
	assert.Equal(t, map[string]string{"workflow_id": "wf-y", "type": "webhook", "limit": "5"}, api.lastQuery)

	enabled := true
	onlyEnabled, err := c.Triggers().List(ctx, &TriggerListOptions{Enabled: &enabled})
	require.NoError(t, err)
	require.Len(t, onlyEnabled, 1)
	assert.Equal(t, hook.ID, onlyEnabled[0].ID)

	// Manual execution of a disabled trigger fails with the server's error.
	_, err = c.Triggers().Trigger(ctx, cron.ID, nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "TRIGGER_DISABLED", apiErr.Code)
	assert.Contains(t, string(api.lastBody), `"input":{}`)

	require.NoError(t, c.Triggers().Enable(ctx, cron.ID))
	got, err = c.Triggers().Get(ctx, cron.ID)
	require.NoError(t, err)
	assert.True(t, got.Enabled)

	// The server currently answers 501 for manual execution.
	_, err = c.Triggers().Trigger(ctx, cron.ID, map[string]any{"a": 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotSupportedByAPI)

	require.NoError(t, c.Triggers().Disable(ctx, cron.ID))
	got, err = c.Triggers().Get(ctx, cron.ID)
	require.NoError(t, err)
	assert.False(t, got.Enabled)

	got.Name = "renamed"
	got.Enabled = true
	got.Config = map[string]any{"schedule": "*/10 * * * * *"}
	updated, err := c.Triggers().Update(ctx, got)
	require.NoError(t, err)
	assert.Equal(t, "renamed", updated.Name)
	assert.True(t, updated.Enabled)
	assert.Equal(t, "*/10 * * * * *", updated.Config["schedule"])

	url, err := c.Triggers().GetWebhookURL(ctx, hook.ID)
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/api/v1/webhooks/"+hook.ID, url)

	_, err = c.Triggers().GetWebhookURL(ctx, cron.ID)
	assert.EqualError(t, err, "trigger is not a webhook trigger")

	_, err = c.Triggers().GetHistory(ctx, cron.ID, &TriggerHistoryOptions{Limit: 1})
	assert.ErrorIs(t, err, ErrNotSupportedByAPI)
	var unsupported *UnsupportedOperationError
	require.ErrorAs(t, err, &unsupported)
	assert.Equal(t, "TriggerAPI.GetHistory", unsupported.Operation)

	require.NoError(t, c.Triggers().Delete(ctx, cron.ID))
	_, err = c.Triggers().Get(ctx, cron.ID)
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)

	for _, call := range []func() error{
		func() error { return c.Triggers().Delete(ctx, cron.ID) },
		func() error { return c.Triggers().Enable(ctx, cron.ID) },
		func() error { return c.Triggers().Disable(ctx, cron.ID) },
		func() error { _, err := c.Triggers().Update(ctx, got); return err },
		func() error { _, err := c.Triggers().GetWebhookURL(ctx, cron.ID); return err },
	} {
		assert.ErrorAs(t, call(), &apiErr)
	}

	assert.Contains(t, api.requests, "POST /api/v1/triggers/tr-1/enable")
	assert.Contains(t, api.requests, "POST /api/v1/triggers/tr-1/disable")
	assert.Contains(t, api.requests, "POST /api/v1/triggers/tr-1/execute")
}

func TestRCTriggerRemote_ListError(t *testing.T) {
	srv := httptest.NewServer(withHealthCheck(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data": "not-a-list"}`))
	}))
	defer srv.Close()
	c := rcRemoteClient(t, srv.URL)

	_, err := c.Triggers().List(context.Background(), nil)
	assert.ErrorContains(t, err, "failed to decode list items")
	_, err = c.Workflows().List(context.Background(), nil)
	assert.ErrorContains(t, err, "failed to decode list items")
	_, err = c.Executions().List(context.Background(), nil)
	assert.ErrorContains(t, err, "failed to decode list items")
}

func TestRCTriggerRemote_TriggerSuccess(t *testing.T) {
	srv := httptest.NewServer(withHealthCheck(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/triggers/t%2F1/execute", r.URL.EscapedPath())
		rcWriteJSON(w, http.StatusAccepted, &models.Execution{ID: "ex-9", Status: models.ExecutionStatusPending})
	}))
	defer srv.Close()
	c := rcRemoteClient(t, srv.URL)

	ex, err := c.Triggers().Trigger(context.Background(), "t/1", map[string]any{"k": "v"})
	require.NoError(t, err)
	assert.Equal(t, "ex-9", ex.ID)
}

func TestRCExecutionRemote_RunSyncPolls(t *testing.T) {
	old := syncPollInterval
	syncPollInterval = time.Millisecond
	t.Cleanup(func() { syncPollInterval = old })

	api, srv := rcNewFakeAPI(t)
	api.completeAfter = 3
	c := rcRemoteClient(t, srv.URL)
	ctx := context.Background()

	wf, err := c.Workflows().Create(ctx, rcSampleWorkflow())
	require.NoError(t, err)

	async, err := c.Executions().Run(ctx, wf.ID, map[string]any{"in": 1.0}, nil)
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusPending, async.Status)
	assert.Equal(t, 1.0, async.Input["in"])

	done, err := c.Executions().RunSync(ctx, wf.ID, nil, map[string]any{"v": 1})
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusCompleted, done.Status)
	assert.Equal(t, true, done.Output["ok"])
	assert.Equal(t, 3, api.execPolls[done.ID])
	assert.Contains(t, api.requests, "POST /api/v1/executions/run/"+wf.ID)

	_, err = c.Executions().Run(ctx, "missing", nil, nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "WORKFLOW_NOT_FOUND", apiErr.Code)

	list, err := c.Executions().List(ctx, &ExecutionListOptions{WorkflowID: wf.ID, Status: "completed", Limit: 10, Offset: 1})
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, map[string]string{"workflow_id": wf.ID, "status": "completed", "limit": "10", "offset": "1"}, api.lastQuery)
}

func TestRCExecutionRemote_RunSyncContextCancelled(t *testing.T) {
	old := syncPollInterval
	syncPollInterval = time.Millisecond
	t.Cleanup(func() { syncPollInterval = old })

	api, srv := rcNewFakeAPI(t)
	api.completeAfter = 1 << 30
	c := rcRemoteClient(t, srv.URL)
	wf, err := c.Workflows().Create(context.Background(), rcSampleWorkflow())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	ex, err := c.Executions().RunSync(ctx, wf.ID, nil, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NotNil(t, ex)
	assert.NotEmpty(t, ex.ID)
}

func TestRCExecutionRemote_RunSyncPollError(t *testing.T) {
	old := syncPollInterval
	syncPollInterval = time.Millisecond
	t.Cleanup(func() { syncPollInterval = old })

	srv := httptest.NewServer(withHealthCheck(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			rcWriteJSON(w, http.StatusAccepted, &models.Execution{ID: "ex-1", Status: models.ExecutionStatusRunning})
			return
		}
		rcWriteAPIError(w, http.StatusInternalServerError, "BOOM", "db down")
	}))
	defer srv.Close()
	c := rcRemoteClient(t, srv.URL)

	_, err := c.Executions().RunSync(context.Background(), "wf", nil, nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "BOOM", apiErr.Code)
}

func TestRCExecutionRemote_OtherEndpoints(t *testing.T) {
	api, srv := rcNewFakeAPI(t)
	c := rcRemoteClient(t, srv.URL)
	ctx := context.Background()

	wf, err := c.Workflows().Create(ctx, rcSampleWorkflow())
	require.NoError(t, err)
	ex, err := c.Executions().Run(ctx, wf.ID, nil, nil)
	require.NoError(t, err)

	logs, err := c.Executions().GetLogs(ctx, ex.ID, &LogOptions{NodeID: "n1", Level: "info", Limit: 5, Offset: 2})
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Unix(), logs[0].Timestamp)
	assert.Equal(t, "execution.started", logs[0].EventType)
	assert.Equal(t, "v", logs[0].Fields["k"])
	assert.Equal(t, map[string]string{"node_id": "n1", "level": "info", "limit": "5", "offset": "2"}, api.lastQuery)

	nr, err := c.Executions().GetNodeResult(ctx, ex.ID, "n1")
	require.NoError(t, err)
	assert.Equal(t, "n1", nr.NodeID)
	assert.Contains(t, api.requests, "GET /api/v1/executions/"+ex.ID+"/nodes/n1/result")

	err = c.Executions().Cancel(ctx, ex.ID)
	assert.ErrorIs(t, err, ErrNotSupportedByAPI)
	_, err = c.Executions().Retry(ctx, ex.ID)
	assert.ErrorIs(t, err, ErrNotSupportedByAPI)

	_, err = c.Executions().Watch(ctx, ex.ID)
	assert.ErrorIs(t, err, ErrNotSupportedByAPI)
	_, err = c.Executions().StreamLogs(ctx, ex.ID, nil)
	assert.ErrorIs(t, err, ErrNotSupportedByAPI)

	_, err = c.Executions().GetLogs(ctx, "missing", nil)
	assert.Error(t, err)
}

func TestRCExecutionRemote_CancelRetrySuccess(t *testing.T) {
	srv := httptest.NewServer(withHealthCheck(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/executions/e1/cancel":
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/executions/e1/retry":
			rcWriteJSON(w, http.StatusOK, &models.Execution{ID: "e2"})
		default:
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	defer srv.Close()
	c := rcRemoteClient(t, srv.URL)

	require.NoError(t, c.Executions().Cancel(context.Background(), "e1"))
	ex, err := c.Executions().Retry(context.Background(), "e1")
	require.NoError(t, err)
	assert.Equal(t, "e2", ex.ID)
}

func TestRCLogEntry_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    int64
		fields  map[string]any
		wantErr bool
	}{
		{name: "unix seconds", in: `{"timestamp": 1700000000, "message": "m"}`, want: 1700000000},
		{name: "float", in: `{"timestamp": 1700000000.9}`, want: 1700000000},
		{name: "rfc3339", in: `{"timestamp": "2023-11-14T22:13:20Z"}`, want: 1700000000},
		{name: "rfc3339 nano", in: `{"timestamp": "2023-11-14T22:13:20.123456789Z"}`, want: 1700000000},
		{name: "null", in: `{"timestamp": null}`, want: 0},
		{name: "missing", in: `{"message": "x"}`, want: 0},
		{name: "empty string", in: `{"timestamp": ""}`, want: 0},
		{name: "fields win over data", in: `{"fields": {"a": 1}, "data": {"b": 2}}`, fields: map[string]any{"a": 1.0}},
		{name: "data maps to fields", in: `{"data": {"b": 2}}`, fields: map[string]any{"b": 2.0}},
		{name: "bad string", in: `{"timestamp": "yesterday"}`, wantErr: true},
		{name: "bad type", in: `{"timestamp": true}`, wantErr: true},
		{name: "bad json", in: `{"timestamp": `, wantErr: true},
		{name: "bad level type", in: `{"level": 5}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var l LogEntry
			err := json.Unmarshal([]byte(tt.in), &l)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, l.Timestamp)
			if tt.fields != nil {
				assert.Equal(t, tt.fields, l.Fields)
			}
		})
	}

	// Round trip through Marshal keeps numeric timestamps.
	orig := LogEntry{Timestamp: 42, Level: "warn", Message: "hi", NodeID: "n", ExecutionID: "e"}
	b, err := json.Marshal(orig)
	require.NoError(t, err)
	var back LogEntry
	require.NoError(t, json.Unmarshal(b, &back))
	assert.Equal(t, orig, back)
}

func TestRCAPIError_Formatting(t *testing.T) {
	assert.Equal(t, "API error (404) NOT_FOUND: gone", (&APIError{StatusCode: 404, Code: "NOT_FOUND", Message: "gone"}).Error())
	assert.Equal(t, "API error (500): oops", (&APIError{StatusCode: 500, Body: "oops"}).Error())
	assert.Equal(t, "API error (502)", (&APIError{StatusCode: 502}).Error())
	assert.True(t, errors.Is(&APIError{StatusCode: 501}, ErrNotSupportedByAPI))
	assert.False(t, errors.Is(&APIError{StatusCode: 500}, ErrNotSupportedByAPI))

	e := &UnsupportedOperationError{Operation: "Op"}
	assert.Equal(t, "Op: operation not supported by the MBFlow REST API", e.Error())
	e.Reason = "why"
	assert.Equal(t, "Op: operation not supported by the MBFlow REST API (why)", e.Error())
	assert.ErrorIs(t, e, ErrNotSupportedByAPI)
}

func TestRCDoRemote_ErrorPaths(t *testing.T) {
	var mode string
	srv := httptest.NewServer(withHealthCheck(func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case "plain":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("upstream exploded\n"))
		case "jsonNoCode":
			rcWriteJSON(w, http.StatusConflict, map[string]any{"foo": "bar"})
		case "empty":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "badJSON":
			_, _ = w.Write([]byte("{not json"))
		}
	}))
	defer srv.Close()
	c := rcRemoteClient(t, srv.URL)
	ctx := context.Background()

	var apiErr *APIError

	mode = "plain"
	err := c.doRemote(ctx, http.MethodGet, "/x", nil, nil, nil)
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "upstream exploded", apiErr.Body)
	assert.Equal(t, http.StatusBadGateway, apiErr.StatusCode)

	mode = "jsonNoCode"
	err = c.doRemote(ctx, http.MethodGet, "/x", nil, nil, nil)
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, `{"foo":"bar"}`, apiErr.Body)
	assert.Empty(t, apiErr.Code)

	mode = "empty"
	err = c.doRemote(ctx, http.MethodGet, "/x", nil, nil, nil)
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "API error (503)", err.Error())

	mode = "badJSON"
	var out map[string]any
	err = c.doRemote(ctx, http.MethodGet, "/x", nil, nil, &out)
	assert.ErrorContains(t, err, "failed to decode response")

	err = c.doRemote(ctx, http.MethodPost, "/x", nil, map[string]any{"c": make(chan int)}, nil)
	assert.ErrorContains(t, err, "failed to marshal request")

	err = c.doRemote(ctx, "BAD METHOD", "/x", nil, nil, nil)
	assert.ErrorContains(t, err, "failed to create request")

	// Connection failure after the server goes away.
	srv.Close()
	err = c.doRemote(ctx, http.MethodGet, "/x", nil, nil, nil)
	assert.ErrorContains(t, err, "request failed")
}

func TestRCDecodeListEnvelope(t *testing.T) {
	raw := map[string]json.RawMessage{"executions": json.RawMessage(`[{"id":"a"}]`)}
	items, err := decodeListEnvelope[*models.Execution](raw, "executions")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "a", items[0].ID)

	items, err = decodeListEnvelope[*models.Execution](map[string]json.RawMessage{"data": json.RawMessage("null")}, "")
	require.NoError(t, err)
	assert.Empty(t, items)

	items, err = decodeListEnvelope[*models.Execution](map[string]json.RawMessage{}, "")
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestRCHasAllTags(t *testing.T) {
	assert.True(t, hasAllTags([]string{"a", "b"}, nil))
	assert.True(t, hasAllTags([]string{"a", "b"}, []string{"b"}))
	assert.False(t, hasAllTags(nil, []string{"b"}))
}

func TestRCToRemoteWorkflowRequest_SkipsNil(t *testing.T) {
	wf := &models.Workflow{
		Name:      "x",
		Nodes:     []*models.Node{nil, {ID: "a", Name: "A", Type: "t"}},
		Edges:     []*models.Edge{nil},
		Resources: []models.WorkflowResource{{ResourceID: "r", Alias: "al"}},
	}
	req := toRemoteWorkflowRequest(wf, false)
	assert.Len(t, req.Nodes, 1)
	assert.Empty(t, req.Edges)
	assert.Nil(t, req.Resources)
	assert.Nil(t, req.Nodes[0].Position)

	req = toRemoteWorkflowRequest(wf, true)
	assert.Equal(t, []remoteResource{{ResourceID: "r", Alias: "al"}}, req.Resources)
}
