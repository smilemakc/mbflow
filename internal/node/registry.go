package node

import (
	"errors"
	"sync"
)

type Registry struct {
	mu     sync.RWMutex
	byID   map[string]Node
	byName map[string][]Node
}

func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]Node), byName: make(map[string][]Node)}
}

func (r *Registry) Register(n Node) error {
	if n == nil {
		return errors.New("node is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id := n.ID()
	if id == "" {
		return errors.New("node id cannot be empty")
	}
	if _, exists := r.byID[id]; exists {
		return errors.New("node id already registered")
	}
	r.byID[id] = n
	name := n.Name()
	r.byName[name] = append(r.byName[name], n)
	return nil
}

func (r *Registry) GetByID(id string) (Node, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.byID[id]
	return n, ok
}

func (r *Registry) ListByName(name string) []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := r.byName[name]
	out := make([]Node, len(list))
	copy(out, list)
	return out
}

func (r *Registry) ListAll() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Node, 0, len(r.byID))
	for _, n := range r.byID {
		out = append(out, n)
	}
	return out
}
