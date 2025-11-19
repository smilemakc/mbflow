package domain

// Node represents a step in a workflow.
type Node struct {
	id         string
	workflowID string
	nodeType   string
	name       string
	config     map[string]any
}

// NewNode creates a new Node instance.
func NewNode(id, workflowID, nodeType, name string, config map[string]any) *Node {
	return &Node{
		id:         id,
		workflowID: workflowID,
		nodeType:   nodeType,
		name:       name,
		config:     config,
	}
}

// ReconstructNode reconstructs a Node from persistence.
func ReconstructNode(id, workflowID, nodeType, name string, config map[string]any) *Node {
	return &Node{
		id:         id,
		workflowID: workflowID,
		nodeType:   nodeType,
		name:       name,
		config:     config,
	}
}

// ID returns the node ID.
func (n *Node) ID() string {
	return n.id
}

// WorkflowID returns the workflow ID this node belongs to.
func (n *Node) WorkflowID() string {
	return n.workflowID
}

// Type returns the type of the node.
func (n *Node) Type() string {
	return n.nodeType
}

// Name returns the name of the node.
func (n *Node) Name() string {
	return n.name
}

// Config returns the configuration of the node.
func (n *Node) Config() map[string]any {
	return n.config
}
