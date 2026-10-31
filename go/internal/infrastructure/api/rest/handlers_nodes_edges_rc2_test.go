package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/internal/domain/repository"
	storagemodels "github.com/smilemakc/mbflow/go/internal/infrastructure/storage/models"
)

// rc2MemWorkflowRepo is an in-memory WorkflowRepository with per-method
// error injection. Methods that are not implemented panic via the nil
// embedded interface, so tests notice unexpected calls.
type rc2MemWorkflowRepo struct {
	repository.WorkflowRepository

	mu        sync.Mutex
	workflows map[uuid.UUID]*storagemodels.WorkflowModel
	nodes     map[uuid.UUID][]*storagemodels.NodeModel
	edges     map[uuid.UUID][]*storagemodels.EdgeModel
	fail      map[string]error
}

func rc2NewMemWorkflowRepo() *rc2MemWorkflowRepo {
	return &rc2MemWorkflowRepo{
		workflows: map[uuid.UUID]*storagemodels.WorkflowModel{},
		nodes:     map[uuid.UUID][]*storagemodels.NodeModel{},
		edges:     map[uuid.UUID][]*storagemodels.EdgeModel{},
		fail:      map[string]error{},
	}
}

func (r *rc2MemWorkflowRepo) setFail(method string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err == nil {
		delete(r.fail, method)
		return
	}
	r.fail[method] = err
}

func (r *rc2MemWorkflowRepo) failure(method string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fail[method]
}

func (r *rc2MemWorkflowRepo) addWorkflow(owner *uuid.UUID) uuid.UUID {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New()
	r.workflows[id] = &storagemodels.WorkflowModel{ID: id, Name: "wf", Status: "draft", CreatedBy: owner}
	return id
}

func (r *rc2MemWorkflowRepo) Create(ctx context.Context, wf *storagemodels.WorkflowModel) error {
	if err := r.failure("Create"); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[wf.ID] = wf
	r.nodes[wf.ID] = wf.Nodes
	r.edges[wf.ID] = wf.Edges
	return nil
}

func (r *rc2MemWorkflowRepo) HardDelete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workflows, id)
	return nil
}

func (r *rc2MemWorkflowRepo) FindByID(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	if err := r.failure("FindByID"); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	wf, ok := r.workflows[id]
	if !ok {
		return nil, errors.New("workflow not found")
	}
	return wf, nil
}

func (r *rc2MemWorkflowRepo) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*storagemodels.WorkflowModel, error) {
	wf, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *wf
	cp.Nodes = r.nodes[id]
	cp.Edges = r.edges[id]
	return &cp, nil
}

func (r *rc2MemWorkflowRepo) CreateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	if err := r.failure("CreateNode"); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range r.nodes[node.WorkflowID] {
		if n.NodeID == node.NodeID {
			return errors.New(`duplicate key value violates unique constraint "uq_nodes_workflow_node_id"`)
		}
	}
	r.nodes[node.WorkflowID] = append(r.nodes[node.WorkflowID], node)
	return nil
}

func (r *rc2MemWorkflowRepo) UpdateNode(ctx context.Context, node *storagemodels.NodeModel) error {
	return r.failure("UpdateNode")
}

func (r *rc2MemWorkflowRepo) DeleteNode(ctx context.Context, id uuid.UUID) error {
	if err := r.failure("DeleteNode"); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for wf, nodes := range r.nodes {
		for i, n := range nodes {
			if n.ID == id {
				r.nodes[wf] = append(nodes[:i:i], nodes[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (r *rc2MemWorkflowRepo) FindNodesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.NodeModel, error) {
	if err := r.failure("FindNodesByWorkflowID"); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	// Return copies: handlers mutate the models they load.
	out := make([]*storagemodels.NodeModel, 0, len(r.nodes[workflowID]))
	for _, n := range r.nodes[workflowID] {
		cp := *n
		out = append(out, &cp)
	}
	return out, nil
}

func (r *rc2MemWorkflowRepo) CreateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	if err := r.failure("CreateEdge"); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.edges[edge.WorkflowID] {
		if e.EdgeID == edge.EdgeID {
			return errors.New(`duplicate key value violates unique constraint "uq_edges_workflow_edge_id"`)
		}
	}
	r.edges[edge.WorkflowID] = append(r.edges[edge.WorkflowID], edge)
	return nil
}

func (r *rc2MemWorkflowRepo) UpdateEdge(ctx context.Context, edge *storagemodels.EdgeModel) error {
	return r.failure("UpdateEdge")
}

func (r *rc2MemWorkflowRepo) DeleteEdge(ctx context.Context, id uuid.UUID) error {
	if err := r.failure("DeleteEdge"); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for wf, edges := range r.edges {
		for i, e := range edges {
			if e.ID == id {
				r.edges[wf] = append(edges[:i:i], edges[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (r *rc2MemWorkflowRepo) FindEdgesByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*storagemodels.EdgeModel, error) {
	if err := r.failure("FindEdgesByWorkflowID"); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*storagemodels.EdgeModel, 0, len(r.edges[workflowID]))
	for _, e := range r.edges[workflowID] {
		cp := *e
		out = append(out, &cp)
	}
	return out, nil
}

// rc2GraphRouter registers node and edge routes the way pkg/server does
// (snake_case parameters), plus parameter-less routes for validation paths.
func rc2GraphRouter(repo repository.WorkflowRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := rc2Logger()
	nodes := NewNodeHandlers(repo, log)
	edges := NewEdgeHandlers(repo, log)

	r := gin.New()
	wf := r.Group("/api/v1/workflows")
	wf.POST("/:workflow_id/nodes", nodes.HandleAddNode)
	wf.GET("/:workflow_id/nodes", nodes.HandleListNodes)
	wf.GET("/:workflow_id/nodes/:node_id", nodes.HandleGetNode)
	wf.PUT("/:workflow_id/nodes/:node_id", nodes.HandleUpdateNode)
	wf.DELETE("/:workflow_id/nodes/:node_id", nodes.HandleDeleteNode)
	wf.POST("/:workflow_id/edges", edges.HandleAddEdge)
	wf.GET("/:workflow_id/edges", edges.HandleListEdges)
	wf.GET("/:workflow_id/edges/:edge_id", edges.HandleGetEdge)
	wf.PUT("/:workflow_id/edges/:edge_id", edges.HandleUpdateEdge)
	wf.DELETE("/:workflow_id/edges/:edge_id", edges.HandleDeleteEdge)

	np := r.Group("/noparam")
	for _, h := range []struct {
		method string
		path   string
		fn     gin.HandlerFunc
	}{
		{http.MethodPost, "/nodes", nodes.HandleAddNode},
		{http.MethodGet, "/nodes", nodes.HandleListNodes},
		{http.MethodGet, "/node", nodes.HandleGetNode},
		{http.MethodPut, "/node", nodes.HandleUpdateNode},
		{http.MethodDelete, "/node", nodes.HandleDeleteNode},
		{http.MethodPost, "/edges", edges.HandleAddEdge},
		{http.MethodGet, "/edges", edges.HandleListEdges},
		{http.MethodGet, "/edge", edges.HandleGetEdge},
		{http.MethodPut, "/edge", edges.HandleUpdateEdge},
		{http.MethodDelete, "/edge", edges.HandleDeleteEdge},
	} {
		np.Handle(h.method, h.path, h.fn)
		// Workflow ID present but item ID missing.
		np.Handle(h.method, "/:workflow_id"+h.path, h.fn)
	}
	return r
}

func rc2Do(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var raw string
	switch b := body.(type) {
	case nil:
	case string:
		raw = b
	default:
		raw = rc2JSON(t, b)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRC2_Nodes_CRUDAndErrors(t *testing.T) {
	t.Parallel()
	repo := rc2NewMemWorkflowRepo()
	r := rc2GraphRouter(repo)
	wfID := repo.addWorkflow(nil).String()
	base := "/api/v1/workflows/" + wfID + "/nodes"

	// Validation.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodPost, "/api/v1/workflows/bad/nodes", map[string]any{}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "n1"}).Code)
	w := rc2Do(t, r, http.MethodPost, "/api/v1/workflows/"+uuid.NewString()+"/nodes", map[string]any{"id": "n1", "name": "N", "type": "http"})
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Create with position.
	w = rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "n1", "name": "Node 1", "type": "http", "config": map[string]any{"url": "x"}, "position": map[string]any{"x": 10, "y": 20}})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	created := rc2DecodeMap(t, w)
	assert.Equal(t, "n1", created["id"])
	assert.Equal(t, "Node 1", created["name"])
	assert.Equal(t, map[string]any{"x": float64(10), "y": float64(20)}, created["position"])

	// Duplicate ID and generic storage failure.
	w = rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "n1", "name": "dup", "type": "http"})
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "node with this ID already exists", rc2DecodeMap(t, w)["message"])
	repo.setFail("CreateNode", errors.New("disk full"))
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "n2", "name": "x", "type": "http"}).Code)
	repo.setFail("CreateNode", nil)

	// List.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodGet, "/api/v1/workflows/bad/nodes", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodGet, "/api/v1/workflows/"+uuid.NewString()+"/nodes", nil).Code)
	w = rc2Do(t, r, http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, float64(1), rc2DecodeMap(t, w)["total"])
	repo.setFail("FindNodesByWorkflowID", errors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodGet, base, nil).Code)
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodGet, base+"/n1", nil).Code)
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodPut, base+"/n1", map[string]any{}).Code)
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodDelete, base+"/n1", nil).Code)
	repo.setFail("FindNodesByWorkflowID", nil)

	// Get (snake_case route parameter, as registered in pkg/server).
	w = rc2Do(t, r, http.MethodGet, base+"/n1", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "Node 1", rc2DecodeMap(t, w)["name"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodGet, base+"/missing", nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodGet, "/api/v1/workflows/bad/nodes/n1", nil).Code)

	// Update.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodPut, "/api/v1/workflows/bad/nodes/n1", map[string]any{}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodPut, base+"/n1", "{bad").Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodPut, base+"/missing", map[string]any{"name": "x"}).Code)
	w = rc2Do(t, r, http.MethodPut, base+"/n1", map[string]any{"name": "Renamed", "type": "transform", "config": map[string]any{"k": "v"}, "position": map[string]any{"x": 1, "y": 2}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	updated := rc2DecodeMap(t, w)
	assert.Equal(t, "Renamed", updated["name"])
	assert.Equal(t, "transform", updated["type"])
	assert.Equal(t, map[string]any{"k": "v"}, updated["config"])
	repo.setFail("UpdateNode", errors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodPut, base+"/n1", map[string]any{"name": "x"}).Code)
	repo.setFail("UpdateNode", nil)

	// Delete.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodDelete, "/api/v1/workflows/bad/nodes/n1", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodDelete, base+"/missing", nil).Code)
	repo.setFail("DeleteNode", errors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodDelete, base+"/n1", nil).Code)
	repo.setFail("DeleteNode", nil)
	w = rc2Do(t, r, http.MethodDelete, base+"/n1", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodGet, base+"/n1", nil).Code)
}

func TestRC2_NodesEdges_MissingParameters(t *testing.T) {
	t.Parallel()
	repo := rc2NewMemWorkflowRepo()
	r := rc2GraphRouter(repo)
	wfID := repo.addWorkflow(nil).String()

	for _, p := range []struct{ method, path string }{
		{http.MethodPost, "/nodes"}, {http.MethodGet, "/nodes"},
		{http.MethodGet, "/node"}, {http.MethodPut, "/node"}, {http.MethodDelete, "/node"},
		{http.MethodPost, "/edges"}, {http.MethodGet, "/edges"},
		{http.MethodGet, "/edge"}, {http.MethodPut, "/edge"}, {http.MethodDelete, "/edge"},
	} {
		w := rc2Do(t, r, p.method, "/noparam"+p.path, map[string]any{})
		assert.Equal(t, http.StatusBadRequest, w.Code, p.path)
		assert.Contains(t, rc2DecodeMap(t, w)["message"], "workflow ID is required", p.path)
	}
	for _, p := range []struct{ method, path, msg string }{
		{http.MethodGet, "/node", "node ID is required"},
		{http.MethodPut, "/node", "node ID is required"},
		{http.MethodDelete, "/node", "node ID is required"},
		{http.MethodGet, "/edge", "edge ID is required"},
		{http.MethodPut, "/edge", "edge ID is required"},
		{http.MethodDelete, "/edge", "edge ID is required"},
	} {
		w := rc2Do(t, r, p.method, "/noparam/"+wfID+p.path, map[string]any{})
		assert.Equal(t, http.StatusBadRequest, w.Code, p.path)
		assert.Equal(t, p.msg, rc2DecodeMap(t, w)["message"], p.path)
	}
}

func TestRC2_DetectCycle(t *testing.T) {
	t.Parallel()
	edges := []*storagemodels.EdgeModel{
		{FromNodeID: "a", ToNodeID: "b"},
		{FromNodeID: "b", ToNodeID: "c"},
		{FromNodeID: "a", ToNodeID: "d"},
		{FromNodeID: "d", ToNodeID: "c"},
	}
	assert.True(t, detectCycle(edges, "c", "a"))
	assert.False(t, detectCycle(edges, "a", "c"), "diamonds are not cycles")
	assert.False(t, detectCycle(edges, "c", "e"))

	// Existing loop (back) edges are not treated as part of the DAG.
	withLoop := append(edges, &storagemodels.EdgeModel{FromNodeID: "c", ToNodeID: "a", Loop: storagemodels.JSONBMap{"max_iterations": 3}})
	assert.False(t, detectCycle(withLoop, "b", "d"))
	assert.True(t, detectCycle(withLoop, "c", "b"), "plain back edges are still cycles")
}

func TestRC2_Edges_CRUDAndErrors(t *testing.T) {
	t.Parallel()
	repo := rc2NewMemWorkflowRepo()
	r := rc2GraphRouter(repo)
	wfID := repo.addWorkflow(nil).String()
	nodesURL := "/api/v1/workflows/" + wfID + "/nodes"
	base := "/api/v1/workflows/" + wfID + "/edges"
	for _, id := range []string{"a", "b", "c"} {
		require.Equal(t, http.StatusCreated, rc2Do(t, r, http.MethodPost, nodesURL, map[string]any{"id": id, "name": id, "type": "http"}).Code)
	}

	// Request validation.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodPost, "/api/v1/workflows/bad/edges", map[string]any{}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "e"}).Code)
	for body, msg := range map[string]string{
		`{"id":"e","from":"a","to":"a"}`:                                             "self-loop edges are not allowed",
		`{"id":"e","from":"a","to":"b","condition":"x","loop":{"max_iterations":2}}`: "loop edges must not have conditions",
		`{"id":"e","from":"a","to":"b","loop":{"max_iterations":0}}`:                 "loop max_iterations must be > 0",
		`{"id":"e","from":"zz","to":"b"}`:                                            "source node does not exist",
		`{"id":"e","from":"a","to":"zz"}`:                                            "target node does not exist",
	} {
		w := rc2Do(t, r, http.MethodPost, base, body)
		require.Equal(t, http.StatusBadRequest, w.Code, body)
		assert.Equal(t, msg, rc2DecodeMap(t, w)["message"], body)
	}
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodPost, "/api/v1/workflows/"+uuid.NewString()+"/edges", `{"id":"e","from":"a","to":"b"}`).Code)

	// Create a chain a -> b -> c with a condition.
	w := rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "e1", "from": "a", "to": "b", "condition": "x > 1", "source_handle": "out"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	e1 := rc2DecodeMap(t, w)
	assert.Equal(t, "x > 1", e1["condition"])
	require.Equal(t, http.StatusCreated, rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "e2", "from": "b", "to": "c"}).Code)

	// A back edge creates a cycle unless it is an explicit loop edge.
	w = rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "e3", "from": "c", "to": "a"})
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, rc2DecodeMap(t, w)["message"], "cycle")
	w = rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "loop", "from": "c", "to": "a", "loop": map[string]any{"max_iterations": 3}})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.NotNil(t, rc2DecodeMap(t, w)["loop"])

	// A forward edge inside the loop region is allowed.
	w = rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "fwd", "from": "a", "to": "c"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Equal(t, http.StatusOK, rc2Do(t, r, http.MethodDelete, base+"/fwd", nil).Code)

	// Storage errors on create.
	w = rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "e1", "from": "a", "to": "c"})
	assert.Equal(t, "edge with this ID already exists", rc2DecodeMap(t, w)["message"])
	for errMsg, code := range map[string]int{
		"would create a cycle":   http.StatusBadRequest,
		"node not found: zz":     http.StatusBadRequest,
		"connection reset by db": http.StatusInternalServerError,
	} {
		repo.setFail("CreateEdge", errors.New(errMsg))
		w = rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "e9", "from": "a", "to": "c"})
		assert.Equal(t, code, w.Code, errMsg)
	}
	repo.setFail("CreateEdge", nil)
	repo.setFail("FindNodesByWorkflowID", errors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "e9", "from": "a", "to": "c"}).Code)
	repo.setFail("FindNodesByWorkflowID", nil)
	repo.setFail("FindEdgesByWorkflowID", errors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodPost, base, map[string]any{"id": "e9", "from": "a", "to": "c"}).Code)
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodGet, base, nil).Code)
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodGet, base+"/e1", nil).Code)
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodPut, base+"/e1", map[string]any{}).Code)
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodDelete, base+"/e1", nil).Code)
	repo.setFail("FindEdgesByWorkflowID", nil)

	// List and get.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodGet, "/api/v1/workflows/bad/edges", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodGet, "/api/v1/workflows/"+uuid.NewString()+"/edges", nil).Code)
	w = rc2Do(t, r, http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, float64(3), rc2DecodeMap(t, w)["total"])
	w = rc2Do(t, r, http.MethodGet, base+"/e1", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "a", rc2DecodeMap(t, w)["from"])
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodGet, base+"/nope", nil).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodGet, "/api/v1/workflows/bad/edges/e1", nil).Code)

	// Update.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodPut, "/api/v1/workflows/bad/edges/e1", map[string]any{}).Code)
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodPut, base+"/e1", "{bad").Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodPut, base+"/nope", map[string]any{}).Code)
	for body, msg := range map[string]string{
		`{"from":"zz"}`:                    "source node does not exist",
		`{"to":"zz"}`:                      "target node does not exist",
		`{"from":"b"}`:                     "self-loop edges are not allowed",
		`{"loop":{"max_iterations":-1}}`:   "loop max_iterations must be > 0",
		`{"to":"a","from":"a","cond":"x"}`: "self-loop edges are not allowed",
	} {
		w = rc2Do(t, r, http.MethodPut, base+"/e1", body)
		require.Equal(t, http.StatusBadRequest, w.Code, body)
		assert.Equal(t, msg, rc2DecodeMap(t, w)["message"], body)
	}
	repo.setFail("FindNodesByWorkflowID", errors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodPut, base+"/e2", `{"from":"a"}`).Code)
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodPut, base+"/e2", `{"to":"c"}`).Code)
	repo.setFail("FindNodesByWorkflowID", nil)
	w = rc2Do(t, r, http.MethodPut, base+"/e2", map[string]any{"from": "a", "to": "c", "condition": "y", "source_handle": "h", "loop": map[string]any{"max_iterations": 5}})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	upd := rc2DecodeMap(t, w)
	assert.Equal(t, "a", upd["from"])
	assert.Equal(t, "c", upd["to"])
	assert.Equal(t, "y", upd["condition"])
	assert.Equal(t, "h", upd["source_handle"])
	repo.setFail("UpdateEdge", errors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodPut, base+"/e2", map[string]any{}).Code)
	repo.setFail("UpdateEdge", nil)

	// Delete.
	assert.Equal(t, http.StatusBadRequest, rc2Do(t, r, http.MethodDelete, "/api/v1/workflows/bad/edges/e1", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodDelete, base+"/nope", nil).Code)
	repo.setFail("DeleteEdge", errors.New("db down"))
	assert.Equal(t, http.StatusInternalServerError, rc2Do(t, r, http.MethodDelete, base+"/e1", nil).Code)
	repo.setFail("DeleteEdge", nil)
	require.Equal(t, http.StatusOK, rc2Do(t, r, http.MethodDelete, base+"/e1", nil).Code)
	assert.Equal(t, http.StatusNotFound, rc2Do(t, r, http.MethodGet, base+"/e1", nil).Code)
}

func rc2JSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return string(raw)
}
