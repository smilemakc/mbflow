package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/smilemakc/mbflow/go/pkg/engine"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// WorkflowAPI provides methods for managing workflows.
// It handles CRUD operations, DAG validation, and workflow versioning.
type WorkflowAPI struct {
	client *Client
}

// newWorkflowAPI creates a new WorkflowAPI instance.
func newWorkflowAPI(client *Client) *WorkflowAPI {
	return &WorkflowAPI{
		client: client,
	}
}

// Create creates a new workflow with the given specification.
// The workflow's DAG will be validated before creation.
func (w *WorkflowAPI) Create(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	if err := w.client.checkClosed(); err != nil {
		return nil, err
	}

	// Validate workflow structure
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Validate DAG
	if err := validateDAG(workflow); err != nil {
		return nil, fmt.Errorf("DAG validation failed: %w", err)
	}

	// Mode-specific implementation
	if w.client.config.Mode == ModeRemote {
		return w.createRemote(ctx, workflow)
	}

	return w.createEmbedded(ctx, workflow)
}

// Get retrieves a workflow by ID.
func (w *WorkflowAPI) Get(ctx context.Context, workflowID string) (*models.Workflow, error) {
	if err := w.client.checkClosed(); err != nil {
		return nil, err
	}

	if workflowID == "" {
		return nil, models.ErrInvalidWorkflowID
	}

	if w.client.config.Mode == ModeRemote {
		return w.getRemote(ctx, workflowID)
	}

	return w.getEmbedded(ctx, workflowID)
}

// List retrieves all workflows with optional filtering.
func (w *WorkflowAPI) List(ctx context.Context, opts *ListOptions) ([]*models.Workflow, error) {
	if err := w.client.checkClosed(); err != nil {
		return nil, err
	}

	if w.client.config.Mode == ModeRemote {
		return w.listRemote(ctx, opts)
	}

	return w.listEmbedded(ctx, opts)
}

// Update updates an existing workflow.
// The workflow's DAG will be validated before update.
func (w *WorkflowAPI) Update(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	if err := w.client.checkClosed(); err != nil {
		return nil, err
	}

	if workflow.ID == "" {
		return nil, models.ErrInvalidWorkflowID
	}

	// Validate workflow structure
	if err := workflow.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Validate DAG
	if err := validateDAG(workflow); err != nil {
		return nil, fmt.Errorf("DAG validation failed: %w", err)
	}

	if w.client.config.Mode == ModeRemote {
		return w.updateRemote(ctx, workflow)
	}

	return w.updateEmbedded(ctx, workflow)
}

// Delete deletes a workflow by ID.
func (w *WorkflowAPI) Delete(ctx context.Context, workflowID string) error {
	if err := w.client.checkClosed(); err != nil {
		return err
	}

	if workflowID == "" {
		return models.ErrInvalidWorkflowID
	}

	if w.client.config.Mode == ModeRemote {
		return w.deleteRemote(ctx, workflowID)
	}

	return w.deleteEmbedded(ctx, workflowID)
}

// ValidateDAG validates the workflow's DAG structure without persisting it.
// Returns detailed validation errors if the DAG is invalid.
func (w *WorkflowAPI) ValidateDAG(ctx context.Context, workflow *models.Workflow) (*ValidationResult, error) {
	if err := w.client.checkClosed(); err != nil {
		return nil, err
	}

	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	// Check for cycles
	if hasCycle, path := detectCycle(workflow); hasCycle {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("cycle detected: %v", path))
	}

	// Check for orphaned nodes
	if orphans := findOrphanedNodes(workflow); len(orphans) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("orphaned nodes: %v", orphans))
	}

	// Check for invalid edge connections
	if invalidEdges := findInvalidEdges(workflow); len(invalidEdges) > 0 {
		result.Valid = false
		for _, e := range invalidEdges {
			result.Errors = append(result.Errors, e)
		}
	}

	return result, nil
}

// GetTopology returns the topologically sorted order of nodes for execution.
func (w *WorkflowAPI) GetTopology(ctx context.Context, workflowID string) ([]string, error) {
	if err := w.client.checkClosed(); err != nil {
		return nil, err
	}

	workflow, err := w.Get(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	return topologicalSort(workflow)
}

// ListOptions provides filtering options for listing workflows.
type ListOptions struct {
	// Limit specifies the maximum number of workflows to return
	Limit int

	// Offset specifies the number of workflows to skip
	Offset int

	// Status filters workflows by status
	Status string

	// Tags filters workflows by tags
	Tags []string
}

// ValidationResult contains the results of DAG validation.
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// Embedded mode implementations (standalone mode - in-memory only, no database persistence)
// For full persistence support, use pkg/server.Server directly.

var errWorkflowPersistenceNotAvailable = fmt.Errorf("workflow persistence not available in standalone mode; use remote mode or pkg/server.Server")

// createEmbedded creates a workflow in-memory for standalone execution.
// The workflow is populated with IDs and timestamps but NOT persisted to database.
func (w *WorkflowAPI) createEmbedded(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	// Generate ID if not provided
	if workflow.ID == "" {
		workflow.ID = generateID()
	}

	// Set timestamps
	now := time.Now()
	workflow.CreatedAt = now
	workflow.UpdatedAt = now

	// Set status
	if workflow.Status == "" {
		workflow.Status = models.WorkflowStatusActive
	}

	// Generate node IDs if not provided
	for _, node := range workflow.Nodes {
		if node.ID == "" {
			node.ID = generateID()
		}
	}

	// Generate edge IDs if not provided
	for _, edge := range workflow.Edges {
		if edge.ID == "" {
			edge.ID = generateID()
		}
	}

	return workflow, nil
}

func (w *WorkflowAPI) getEmbedded(ctx context.Context, workflowID string) (*models.Workflow, error) {
	return nil, errWorkflowPersistenceNotAvailable
}

func (w *WorkflowAPI) listEmbedded(ctx context.Context, opts *ListOptions) ([]*models.Workflow, error) {
	return nil, errWorkflowPersistenceNotAvailable
}

func (w *WorkflowAPI) updateEmbedded(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	return nil, errWorkflowPersistenceNotAvailable
}

func (w *WorkflowAPI) deleteEmbedded(ctx context.Context, workflowID string) error {
	return errWorkflowPersistenceNotAvailable
}

// Remote mode implementations.
//
// These call the MBFlow REST API (see pkg/server/routes.go):
//
//	POST   /api/v1/workflows
//	GET    /api/v1/workflows/{id}
//	GET    /api/v1/workflows?limit=&offset=&status=
//	PUT    /api/v1/workflows/{id}
//	DELETE /api/v1/workflows/{id}
//
// The API does not store workflow tags or accept a status on create/update
// (status changes go through publish/unpublish), so ListOptions.Tags is
// applied client-side and Workflow.Status/Tags are not sent.

// remoteNode mirrors rest.NodeRequest.
type remoteNode struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Config   map[string]any `json:"config,omitempty"`
	Position map[string]any `json:"position,omitempty"`
}

// remoteEdge mirrors rest.EdgeRequest.
type remoteEdge struct {
	ID           string             `json:"id"`
	From         string             `json:"from"`
	To           string             `json:"to"`
	SourceHandle string             `json:"source_handle,omitempty"`
	Condition    map[string]any     `json:"condition,omitempty"`
	Loop         *models.LoopConfig `json:"loop,omitempty"`
}

// remoteResource mirrors rest.ResourceRequest.
type remoteResource struct {
	ResourceID string `json:"resource_id"`
	Alias      string `json:"alias"`
	AccessType string `json:"access_type,omitempty"`
}

// remoteWorkflowRequest is the body accepted by the create and update endpoints.
type remoteWorkflowRequest struct {
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	Variables   map[string]any   `json:"variables,omitempty"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
	Nodes       []remoteNode     `json:"nodes,omitempty"`
	Edges       []remoteEdge     `json:"edges,omitempty"`
	Resources   []remoteResource `json:"resources,omitempty"`
}

func toRemoteWorkflowRequest(workflow *models.Workflow, withResources bool) *remoteWorkflowRequest {
	req := &remoteWorkflowRequest{
		Name:        workflow.Name,
		Description: workflow.Description,
		Variables:   workflow.Variables,
		Metadata:    workflow.Metadata,
		Nodes:       make([]remoteNode, 0, len(workflow.Nodes)),
		Edges:       make([]remoteEdge, 0, len(workflow.Edges)),
	}
	for _, n := range workflow.Nodes {
		if n == nil {
			continue
		}
		rn := remoteNode{ID: n.ID, Name: n.Name, Type: n.Type, Config: n.Config}
		if n.Position != nil {
			rn.Position = map[string]any{"x": n.Position.X, "y": n.Position.Y}
		}
		req.Nodes = append(req.Nodes, rn)
	}
	for _, e := range workflow.Edges {
		if e == nil {
			continue
		}
		re := remoteEdge{ID: e.ID, From: e.From, To: e.To, SourceHandle: e.SourceHandle, Loop: e.Loop}
		if e.Condition != "" {
			// The server stores conditions as {"expression": "..."}.
			re.Condition = map[string]any{"expression": e.Condition}
		}
		req.Edges = append(req.Edges, re)
	}
	if withResources && len(workflow.Resources) > 0 {
		req.Resources = make([]remoteResource, 0, len(workflow.Resources))
		for _, r := range workflow.Resources {
			req.Resources = append(req.Resources, remoteResource{ResourceID: r.ResourceID, Alias: r.Alias, AccessType: r.AccessType})
		}
	}
	return req
}

func (w *WorkflowAPI) createRemote(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	var out models.Workflow
	if err := w.client.doRemote(ctx, http.MethodPost, "/api/v1/workflows", nil, toRemoteWorkflowRequest(workflow, false), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (w *WorkflowAPI) getRemote(ctx context.Context, workflowID string) (*models.Workflow, error) {
	var out models.Workflow
	if err := w.client.doRemote(ctx, http.MethodGet, "/api/v1/workflows/"+pathEscape(workflowID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (w *WorkflowAPI) listRemote(ctx context.Context, opts *ListOptions) ([]*models.Workflow, error) {
	query := url.Values{}
	if opts != nil {
		setPagination(query, opts.Limit, opts.Offset)
		if opts.Status != "" {
			query.Set("status", opts.Status)
		}
	}

	var raw map[string]json.RawMessage
	if err := w.client.doRemote(ctx, http.MethodGet, "/api/v1/workflows", query, nil, &raw); err != nil {
		return nil, err
	}
	workflows, err := decodeListEnvelope[*models.Workflow](raw, "workflows")
	if err != nil {
		return nil, err
	}

	if opts == nil || len(opts.Tags) == 0 {
		return workflows, nil
	}
	filtered := make([]*models.Workflow, 0, len(workflows))
	for _, wf := range workflows {
		if wf != nil && hasAllTags(wf.Tags, opts.Tags) {
			filtered = append(filtered, wf)
		}
	}
	return filtered, nil
}

// hasAllTags reports whether have contains every tag in want.
func hasAllTags(have, want []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, t := range have {
		set[t] = struct{}{}
	}
	for _, t := range want {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}

func (w *WorkflowAPI) updateRemote(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	var out models.Workflow
	path := "/api/v1/workflows/" + pathEscape(workflow.ID)
	if err := w.client.doRemote(ctx, http.MethodPut, path, nil, toRemoteWorkflowRequest(workflow, true), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (w *WorkflowAPI) deleteRemote(ctx context.Context, workflowID string) error {
	return w.client.doRemote(ctx, http.MethodDelete, "/api/v1/workflows/"+pathEscape(workflowID), nil, nil, nil)
}

// DAG validation helpers

// validateDAG performs comprehensive DAG validation.
func validateDAG(workflow *models.Workflow) error {
	// Check for cycles
	if hasCycle, path := detectCycle(workflow); hasCycle {
		return fmt.Errorf("cycle detected: %v", path)
	}

	// Check for orphaned nodes (except start nodes)
	if orphans := findOrphanedNodes(workflow); len(orphans) > 0 {
		return fmt.Errorf("orphaned nodes found: %v", orphans)
	}

	return nil
}

// detectCycle uses DFS to detect cycles in the workflow DAG.
func detectCycle(workflow *models.Workflow) (bool, []string) {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, edge := range workflow.Edges {
		graph[edge.From] = append(graph[edge.From], edge.To)
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string

	var dfs func(nodeID string) bool
	dfs = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true
		path = append(path, nodeID)

		for _, neighbor := range graph[nodeID] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[nodeID] = false
		path = path[:len(path)-1]
		return false
	}

	for _, node := range workflow.Nodes {
		if !visited[node.ID] {
			if dfs(node.ID) {
				return true, path
			}
		}
	}

	return false, nil
}

// findOrphanedNodes finds nodes that are not connected to the DAG.
func findOrphanedNodes(workflow *models.Workflow) []string {
	// Build sets of nodes with incoming and outgoing edges
	hasIncoming := make(map[string]bool)
	hasOutgoing := make(map[string]bool)

	for _, edge := range workflow.Edges {
		hasIncoming[edge.To] = true
		hasOutgoing[edge.From] = true
	}

	var orphans []string
	for _, node := range workflow.Nodes {
		// A node is orphaned if it has no incoming and no outgoing edges
		// (except for start nodes which may have no incoming edges)
		if !hasIncoming[node.ID] && !hasOutgoing[node.ID] {
			orphans = append(orphans, node.ID)
		}
	}

	return orphans
}

// findInvalidEdges finds edges with invalid references.
func findInvalidEdges(workflow *models.Workflow) []string {
	nodeMap := make(map[string]bool)
	for _, node := range workflow.Nodes {
		nodeMap[node.ID] = true
	}

	var errors []string
	for _, edge := range workflow.Edges {
		if !nodeMap[edge.From] {
			errors = append(errors, fmt.Sprintf("edge references non-existent source node: %s", edge.From))
		}
		if !nodeMap[edge.To] {
			errors = append(errors, fmt.Sprintf("edge references non-existent target node: %s", edge.To))
		}
	}

	return errors
}

// topologicalSort returns nodes in topological order using shared engine implementation.
func topologicalSort(workflow *models.Workflow) ([]string, error) {
	dag := engine.BuildDAG(workflow)
	waves, err := engine.TopologicalSort(dag)
	if err != nil {
		return nil, err
	}
	return engine.FlattenWaves(waves), nil
}

// generateID generates a new UUID string
func generateID() string {
	return uuid.New().String()
}
