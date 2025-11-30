package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Workflow is an aggregate root that represents a workflow definition.
// A workflow defines the structure and configuration of a business process,
// including its nodes, edges, and triggers.
// The workflow owns and manages its child entities (nodes, edges, triggers).
type Workflow interface {
	// Identity
	ID() uuid.UUID
	Name() string
	Version() string
	Description() string
	Spec() map[string]any
	CreatedAt() time.Time
	UpdatedAt() time.Time

	// State management
	State() WorkflowState
	SetState(state WorkflowState) error
	Publish() error
	Archive() error

	// Node management
	UseNode(node Node) error
	AddNode(nodeType NodeType, name string, config map[string]any) (uuid.UUID, error)
	GetNode(id uuid.UUID) (Node, error)
	GetAllNodes() []Node
	RemoveNode(id uuid.UUID) error

	// Edge management
	UseEdge(edge Edge) error
	AddEdge(fromNodeID, toNodeID uuid.UUID, edgeType EdgeType, config map[string]any) (uuid.UUID, error)
	GetEdge(id uuid.UUID) (Edge, error)
	GetAllEdges() []Edge
	RemoveEdge(id uuid.UUID) error

	// Trigger management
	AddTrigger(triggerType TriggerType, config map[string]any) (uuid.UUID, error)
	GetTrigger(id uuid.UUID) (Trigger, error)
	GetAllTriggers() []Trigger
	RemoveTrigger(id uuid.UUID) error

	// Bulk operations
	ClearNodes()    // Removes all nodes and associated edges
	ClearTriggers() // Removes all triggers

	// Validation
	ValidateStructure() error    // Validates structure without requiring triggers (for drafts)
	ValidateForExecution() error // Validates readiness for execution (requires triggers)
	Validate() error             // Alias for ValidateForExecution (backward compatibility)
}

type workflow struct {
	id          uuid.UUID
	name        string
	version     string
	description string
	spec        map[string]any
	state       WorkflowState
	createdAt   time.Time
	updatedAt   time.Time

	// Child entities owned by this aggregate
	nodes    map[uuid.UUID]*node
	edges    map[uuid.UUID]*edge
	triggers map[uuid.UUID]*trigger
}

func RestoreWorkflow(id uuid.UUID, name, version, description string, spec map[string]any) (Workflow, error) {
	now := time.Now()
	return &workflow{
		id:          id,
		name:        name,
		version:     version,
		description: description,
		spec:        spec,
		state:       WorkflowStateDraft,
		createdAt:   now,
		updatedAt:   now,
		nodes:       make(map[uuid.UUID]*node),
		edges:       make(map[uuid.UUID]*edge),
		triggers:    make(map[uuid.UUID]*trigger),
	}, nil
}

// NewWorkflow creates a new Workflow instance.
// If id is uuid.Nil, a new UUID will be generated automatically.
func NewWorkflow(name, version, description string, spec map[string]any) (Workflow, error) {
	return RestoreWorkflow(uuid.New(), name, version, description, spec)
}

// ReconstructWorkflow reconstructs a Workflow from persistence with all its child entities.
func ReconstructWorkflow(
	id uuid.UUID,
	name, version, description string,
	spec map[string]any,
	state WorkflowState,
	createdAt, updatedAt time.Time,
	nodes []Node,
	edges []Edge,
	triggers []Trigger,
) (Workflow, error) {
	w := &workflow{
		id:          id,
		name:        name,
		version:     version,
		description: description,
		spec:        spec,
		state:       state,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		nodes:       make(map[uuid.UUID]*node),
		edges:       make(map[uuid.UUID]*edge),
		triggers:    make(map[uuid.UUID]*trigger),
	}

	// Reconstruct nodes
	for _, n := range nodes {
		if impl, ok := n.(*node); ok {
			w.nodes[impl.ID()] = impl
		}
	}

	// Reconstruct edges
	for _, e := range edges {
		if impl, ok := e.(*edge); ok {
			w.edges[impl.ID()] = impl
		}
	}

	// Reconstruct triggers
	for _, t := range triggers {
		if impl, ok := t.(*trigger); ok {
			w.triggers[impl.ID()] = impl
		}
	}

	return w, nil
}

// ID returns the workflow ID.
func (w *workflow) ID() uuid.UUID {
	return w.id
}

// Name returns the workflow name.
func (w *workflow) Name() string {
	return w.name
}

// Version returns the workflow version.
func (w *workflow) Version() string {
	return w.version
}

// Description returns the workflow description.
func (w *workflow) Description() string {
	return w.description
}

// Spec returns the workflow specification.
func (w *workflow) Spec() map[string]any {
	return w.spec
}

// CreatedAt returns the creation timestamp.
func (w *workflow) CreatedAt() time.Time {
	return w.createdAt
}

// UpdatedAt returns the last update timestamp.
func (w *workflow) UpdatedAt() time.Time {
	return w.updatedAt
}

func (w *workflow) UseNode(n Node) error {
	nn := &node{
		id:       n.ID(),
		nodeType: n.Type(),
		name:     n.Name(),
		config:   n.Config(),
	}

	// Check if node with this ID already exists
	if _, exists := w.nodes[nn.id]; exists {
		// Node with same ID exists - allow update even if name changes
		// This is for updating existing nodes
		w.nodes[nn.id] = nn
		w.updatedAt = time.Now()
		return nil
	}

	// New node - validate uniqueness
	if err := w.addNode(nn); err != nil {
		return err
	}
	return nil
}

// AddNode adds a new node to the workflow.
func (w *workflow) AddNode(nodeType NodeType, name string, config map[string]any) (uuid.UUID, error) {
	// Create new node (without workflowID since it's part of aggregate)
	n := &node{
		id:       uuid.New(),
		nodeType: nodeType,
		name:     name,
		config:   config,
	}

	if err := w.addNode(n); err != nil {
		return uuid.Nil, err
	}
	return n.id, nil
}

func (w *workflow) addNode(n *node) error {
	if err := w.validateUniqueness(n.name, n.nodeType); err != nil {
		return err
	}
	w.nodes[n.id] = n
	w.updatedAt = time.Now()
	return nil
}

func (w *workflow) validateUniqueness(name string, nodeType NodeType) error {
	// Validate node type
	if !nodeType.IsValid() {
		return NewDomainError(
			ErrCodeInvalidInput,
			fmt.Sprintf("invalid node type: %s", nodeType),
			nil,
		)
	}

	// Validate node name uniqueness
	for _, existing := range w.nodes {
		if existing.name == name {
			return NewDomainError(
				ErrCodeAlreadyExists,
				fmt.Sprintf("node with name '%s' already exists", name),
				nil,
			)
		}
	}
	return nil
}

// GetNode retrieves a node by ID.
func (w *workflow) GetNode(id uuid.UUID) (Node, error) {
	n, exists := w.nodes[id]
	if !exists {
		return nil, NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("node with ID %s not found", id),
			nil,
		)
	}
	return n, nil
}

// GetAllNodes returns all nodes in the workflow.
func (w *workflow) GetAllNodes() []Node {
	nodes := make([]Node, 0, len(w.nodes))
	for _, n := range w.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// RemoveNode removes a node from the workflow.
// It also removes all edges connected to this node.
func (w *workflow) RemoveNode(id uuid.UUID) error {
	if _, exists := w.nodes[id]; !exists {
		return NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("node with ID %s not found", id),
			nil,
		)
	}

	// Remove all edges connected to this node
	for edgeID, e := range w.edges {
		if e.fromNodeID == id || e.toNodeID == id {
			delete(w.edges, edgeID)
		}
	}

	delete(w.nodes, id)
	w.updatedAt = time.Now()

	return nil
}

// ClearNodes removes all nodes and associated edges from the workflow.
// This is useful for atomic workflow structure updates.
func (w *workflow) ClearNodes() {
	w.nodes = make(map[uuid.UUID]*node)
	w.edges = make(map[uuid.UUID]*edge)
	w.updatedAt = time.Now()
}

// ClearTriggers removes all triggers from the workflow.
// This is useful for atomic workflow structure updates.
func (w *workflow) ClearTriggers() {
	w.triggers = make(map[uuid.UUID]*trigger)
	w.updatedAt = time.Now()
}

func (w *workflow) UseEdge(e Edge) error {
	ee := &edge{
		id:         e.ID(),
		fromNodeID: e.FromNodeID(),
		toNodeID:   e.ToNodeID(),
		edgeType:   e.Type(),
		config:     e.Config(),
	}
	if err := w.addEdge(ee); err != nil {
		return err
	}

	return nil
}

// AddEdge adds a new edge to the workflow.
func (w *workflow) AddEdge(fromNodeID, toNodeID uuid.UUID, edgeType EdgeType, config map[string]any) (uuid.UUID, error) {
	// Create new edge
	e := &edge{
		id:         uuid.New(),
		fromNodeID: fromNodeID,
		toNodeID:   toNodeID,
		edgeType:   edgeType,
		config:     config,
	}
	if err := w.addEdge(e); err != nil {
		return uuid.Nil, err
	}
	return e.id, nil
}

func (w *workflow) addEdge(e *edge) error {
	if err := w.validateEdge(e.fromNodeID, e.toNodeID, e.edgeType); err != nil {
		return err
	}
	w.edges[e.id] = e
	w.updatedAt = time.Now()
	return nil
}

func (w *workflow) validateEdge(fromNodeID uuid.UUID, toNodeID uuid.UUID, edgeType EdgeType) error {
	// Validate edge type
	if !edgeType.IsValid() {
		return NewDomainError(
			ErrCodeInvalidInput,
			fmt.Sprintf("invalid edge type: %s", edgeType),
			nil,
		)
	}

	// Validate that both nodes exist
	if _, exists := w.nodes[fromNodeID]; !exists {
		return NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("source node with ID %s not found", fromNodeID),
			nil,
		)
	}
	if _, exists := w.nodes[toNodeID]; !exists {
		return NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("destination node with ID %s not found", toNodeID),
			nil,
		)
	}

	// Check for self-loop
	if fromNodeID == toNodeID {
		return NewDomainError(
			ErrCodeInvalidInput,
			"self-loop edges are not allowed",
			nil,
		)
	}
	return nil
}

// GetEdge retrieves an edge by ID.
func (w *workflow) GetEdge(id uuid.UUID) (Edge, error) {
	e, exists := w.edges[id]
	if !exists {
		return nil, NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("edge with ID %s not found", id),
			nil,
		)
	}
	return e, nil
}

// GetAllEdges returns all edges in the workflow.
func (w *workflow) GetAllEdges() []Edge {
	edges := make([]Edge, 0, len(w.edges))
	for _, e := range w.edges {
		edges = append(edges, e)
	}
	return edges
}

// RemoveEdge removes an edge from the workflow.
func (w *workflow) RemoveEdge(id uuid.UUID) error {
	if _, exists := w.edges[id]; !exists {
		return NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("edge with ID %s not found", id),
			nil,
		)
	}

	delete(w.edges, id)
	w.updatedAt = time.Now()

	return nil
}

// AddTrigger adds a new trigger to the workflow.
func (w *workflow) AddTrigger(triggerType TriggerType, config map[string]any) (uuid.UUID, error) {
	// Validate trigger type
	if !triggerType.IsValid() {
		return uuid.Nil, NewDomainError(
			ErrCodeInvalidInput,
			fmt.Sprintf("invalid trigger type: %s", triggerType),
			nil,
		)
	}

	// Create new trigger
	t := &trigger{
		id:          uuid.New(),
		triggerType: triggerType,
		config:      config,
	}

	w.triggers[t.id] = t
	w.updatedAt = time.Now()

	return t.id, nil
}

// GetTrigger retrieves a trigger by ID.
func (w *workflow) GetTrigger(id uuid.UUID) (Trigger, error) {
	t, exists := w.triggers[id]
	if !exists {
		return nil, NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("trigger with ID %s not found", id),
			nil,
		)
	}
	return t, nil
}

// GetAllTriggers returns all triggers in the workflow.
func (w *workflow) GetAllTriggers() []Trigger {
	triggers := make([]Trigger, 0, len(w.triggers))
	for _, t := range w.triggers {
		triggers = append(triggers, t)
	}
	return triggers
}

// RemoveTrigger removes a trigger from the workflow.
func (w *workflow) RemoveTrigger(id uuid.UUID) error {
	if _, exists := w.triggers[id]; !exists {
		return NewDomainError(
			ErrCodeNotFound,
			fmt.Sprintf("trigger with ID %s not found", id),
			nil,
		)
	}

	delete(w.triggers, id)
	w.updatedAt = time.Now()

	return nil
}

func checkUniqueNames(nodes []Node) error {
	// map for fast uniqueness check
	seen := make(map[string]struct{})

	for _, n := range nodes {
		name := n.Name()

		// check if name already seen
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate name: %s", name)
		}

		// mark as seen
		seen[name] = struct{}{}
	}

	return nil
}

// ValidateStructure validates the basic workflow structure without requiring triggers.
// This is suitable for draft workflows that are not yet ready for execution.
func (w *workflow) ValidateStructure() error {
	// Check that workflow has at least one node
	if len(w.nodes) == 0 {
		return NewDomainError(
			ErrCodeValidationFailed,
			"workflow must have at least one node",
			nil,
		)
	}

	// Check node name uniqueness
	if err := checkUniqueNames(w.GetAllNodes()); err != nil {
		return NewDomainError(
			ErrCodeValidationFailed,
			"node names must be unique",
			err,
		)
	}

	// Check for cycles using DFS
	if err := w.checkForCycles(); err != nil {
		return err
	}

	// Check that all edges reference existing nodes
	for _, e := range w.edges {
		if _, exists := w.nodes[e.fromNodeID]; !exists {
			return NewDomainError(
				ErrCodeInvariantViolated,
				fmt.Sprintf("edge %s references non-existent source node %s", e.id, e.fromNodeID),
				nil,
			)
		}
		if _, exists := w.nodes[e.toNodeID]; !exists {
			return NewDomainError(
				ErrCodeInvariantViolated,
				fmt.Sprintf("edge %s references non-existent destination node %s", e.id, e.toNodeID),
				nil,
			)
		}
	}

	return nil
}

// ValidateForExecution validates that the workflow is ready for execution.
// This includes all structural validations plus execution-specific requirements.
func (w *workflow) ValidateForExecution() error {
	// First, validate structure
	if err := w.ValidateStructure(); err != nil {
		return err
	}

	// Check that workflow has at least one trigger (required for execution)
	if len(w.triggers) == 0 {
		return NewDomainError(
			ErrCodeValidationFailed,
			"workflow must have at least one trigger for execution",
			nil,
		)
	}

	return nil
}

// Validate validates the workflow for execution (backward compatibility).
// For draft workflows, use ValidateStructure() instead.
func (w *workflow) Validate() error {
	return w.ValidateForExecution()
}

// checkForCycles performs a DFS-based cycle detection.
func (w *workflow) checkForCycles() error {
	// Build adjacency list
	adj := make(map[uuid.UUID][]uuid.UUID)
	for _, e := range w.edges {
		adj[e.fromNodeID] = append(adj[e.fromNodeID], e.toNodeID)
	}

	// Track visited nodes and nodes in current recursion stack
	visited := make(map[uuid.UUID]bool)
	recStack := make(map[uuid.UUID]bool)

	var dfs func(nodeID uuid.UUID) error
	dfs = func(nodeID uuid.UUID) error {
		visited[nodeID] = true
		recStack[nodeID] = true

		for _, neighborID := range adj[nodeID] {
			if !visited[neighborID] {
				if err := dfs(neighborID); err != nil {
					return err
				}
			} else if recStack[neighborID] {
				return NewDomainError(
					ErrCodeCyclicDependency,
					fmt.Sprintf("cycle detected involving node %s", neighborID),
					nil,
				)
			}
		}

		recStack[nodeID] = false
		return nil
	}

	// Run DFS from each unvisited node
	for nodeID := range w.nodes {
		if !visited[nodeID] {
			if err := dfs(nodeID); err != nil {
				return err
			}
		}
	}

	return nil
}
