package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	storagemodels "github.com/smilemakc/mbflow/internal/infrastructure/storage/models"
	"github.com/smilemakc/mbflow/pkg/models"
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

// Embedded mode implementations
func (w *WorkflowAPI) createEmbedded(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	// If no repository available, fallback to in-memory mode
	if w.client.workflowRepo == nil {
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

	// Use repository for persistence
	storageWorkflow, err := workflowToStorageForCreate(workflow)
	if err != nil {
		return nil, err
	}

	// Set default status
	if storageWorkflow.Status == "" {
		storageWorkflow.Status = string(models.WorkflowStatusActive)
	}

	// Create in database
	if err := w.client.workflowRepo.Create(ctx, storageWorkflow); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// Convert back to domain model
	return workflowFromStorage(storageWorkflow), nil
}

func (w *WorkflowAPI) getEmbedded(ctx context.Context, workflowID string) (*models.Workflow, error) {
	if w.client.workflowRepo == nil {
		return nil, fmt.Errorf("embedded mode get not available: no repository configured")
	}

	id, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, models.ErrInvalidWorkflowID
	}

	storageWorkflow, err := w.client.workflowRepo.FindByIDWithRelations(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	return workflowFromStorage(storageWorkflow), nil
}

func (w *WorkflowAPI) listEmbedded(ctx context.Context, opts *ListOptions) ([]*models.Workflow, error) {
	if w.client.workflowRepo == nil {
		return nil, fmt.Errorf("embedded mode list not available: no repository configured")
	}

	if opts == nil {
		opts = &ListOptions{Limit: 100, Offset: 0}
	}

	var storageWorkflows []*storagemodels.WorkflowModel
	var err error

	if opts.Status != "" {
		storageWorkflows, err = w.client.workflowRepo.FindByStatus(ctx, opts.Status, opts.Limit, opts.Offset)
	} else {
		storageWorkflows, err = w.client.workflowRepo.FindAll(ctx, opts.Limit, opts.Offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	workflows := make([]*models.Workflow, len(storageWorkflows))
	for i, sw := range storageWorkflows {
		workflows[i] = workflowFromStorage(sw)
	}

	return workflows, nil
}

func (w *WorkflowAPI) updateEmbedded(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	if w.client.workflowRepo == nil {
		return nil, fmt.Errorf("embedded mode update not available: no repository configured")
	}

	storageWorkflow, err := workflowToStorageForUpdate(workflow)
	if err != nil {
		return nil, err
	}

	// Update in database (smart merge with UUID preservation)
	if err := w.client.workflowRepo.Update(ctx, storageWorkflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// Fetch updated workflow to get current state
	updated, err := w.client.workflowRepo.FindByIDWithRelations(ctx, storageWorkflow.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated workflow: %w", err)
	}

	return workflowFromStorage(updated), nil
}

func (w *WorkflowAPI) deleteEmbedded(ctx context.Context, workflowID string) error {
	if w.client.workflowRepo == nil {
		return fmt.Errorf("embedded mode delete not available: no repository configured")
	}

	id, err := uuid.Parse(workflowID)
	if err != nil {
		return models.ErrInvalidWorkflowID
	}

	if err := w.client.workflowRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	return nil
}

// Remote mode implementations
func (w *WorkflowAPI) createRemote(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}

func (w *WorkflowAPI) getRemote(ctx context.Context, workflowID string) (*models.Workflow, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}

func (w *WorkflowAPI) listRemote(ctx context.Context, opts *ListOptions) ([]*models.Workflow, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}

func (w *WorkflowAPI) updateRemote(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	// TODO: Implement HTTP API call
	return nil, fmt.Errorf("remote mode not implemented yet")
}

func (w *WorkflowAPI) deleteRemote(ctx context.Context, workflowID string) error {
	// TODO: Implement HTTP API call
	return fmt.Errorf("remote mode not implemented yet")
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

// topologicalSort returns nodes in topological order using Kahn's algorithm.
func topologicalSort(workflow *models.Workflow) ([]string, error) {
	// Build adjacency list and in-degree map
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize all nodes with 0 in-degree
	for _, node := range workflow.Nodes {
		inDegree[node.ID] = 0
	}

	// Build graph and calculate in-degrees
	for _, edge := range workflow.Edges {
		graph[edge.From] = append(graph[edge.From], edge.To)
		inDegree[edge.To]++
	}

	// Find all nodes with in-degree 0 (start nodes)
	var queue []string
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	var result []string
	for len(queue) > 0 {
		// Dequeue
		nodeID := queue[0]
		queue = queue[1:]
		result = append(result, nodeID)

		// Reduce in-degree for neighbors
		for _, neighbor := range graph[nodeID] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// If result doesn't contain all nodes, there's a cycle
	if len(result) != len(workflow.Nodes) {
		return nil, fmt.Errorf("workflow contains a cycle")
	}

	return result, nil
}

// generateID generates a new UUID string
func generateID() string {
	return uuid.New().String()
}
