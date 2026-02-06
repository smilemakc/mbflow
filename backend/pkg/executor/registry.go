package executor

import (
	"fmt"
	"sync"

	"github.com/smilemakc/mbflow/pkg/models"
)

// Registry implements the Manager interface with thread-safe executor registration.
type Registry struct {
	mu        sync.RWMutex
	executors map[string]Executor
}

// NewRegistry creates a new executor registry.
func NewRegistry() *Registry {
	return &Registry{
		executors: make(map[string]Executor),
	}
}

// NewManager creates a new executor manager.
// Built-in executors should be registered separately using RegisterBuiltins function
// from pkg/executor/builtin package to avoid import cycles.
func NewManager() Manager {
	return NewRegistry()
}

// Register registers an executor for a specific node type.
func (r *Registry) Register(nodeType string, executor Executor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if nodeType == "" {
		return fmt.Errorf("node type cannot be empty")
	}

	if executor == nil {
		return fmt.Errorf("executor cannot be nil")
	}

	r.executors[nodeType] = executor
	return nil
}

// Get retrieves an executor by node type.
func (r *Registry) Get(nodeType string) (Executor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executor, ok := r.executors[nodeType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", models.ErrExecutorNotFound, nodeType)
	}

	return executor, nil
}

// Has checks if an executor is registered for the given node type.
func (r *Registry) Has(nodeType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.executors[nodeType]
	return ok
}

// List returns a list of all registered executor types.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.executors))
	for nodeType := range r.executors {
		types = append(types, nodeType)
	}

	return types
}

// Unregister removes an executor for a specific node type.
func (r *Registry) Unregister(nodeType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.executors[nodeType]; !ok {
		return fmt.Errorf("%w: %s", models.ErrExecutorNotFound, nodeType)
	}

	delete(r.executors, nodeType)
	return nil
}
