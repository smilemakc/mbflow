package engine

// Source handle constants for conditional nodes
const (
	// SourceHandleTrue represents the "true" branch from a conditional node
	SourceHandleTrue = "true"

	// SourceHandleFalse represents the "false" branch from a conditional node
	SourceHandleFalse = "false"
)

// Node types
const (
	// NodeTypeConditional represents a conditional/branching node
	NodeTypeConditional = "conditional"
)

// Default configuration values
const (
	// DefaultMaxParallelism is the default maximum number of concurrent nodes per wave
	DefaultMaxParallelism = 10

	// DefaultNodePriority is the default priority for nodes without explicit priority
	DefaultNodePriority = 0
)
