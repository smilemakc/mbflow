package engine

import (
	"container/list"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// ConditionCache is a thread-safe LRU cache for compiled expression programs
type ConditionCache struct {
	capacity int
	cache    map[string]*list.Element
	lruList  *list.List
	mu       sync.RWMutex
}

// cacheEntry represents a cached compiled expression
type cacheEntry struct {
	key     string
	program *vm.Program
}

// NewConditionCache creates a new condition cache with the specified capacity
func NewConditionCache(capacity int) *ConditionCache {
	if capacity <= 0 {
		capacity = 100 // Default capacity
	}

	return &ConditionCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		lruList:  list.New(),
	}
}

// Get retrieves a compiled program from cache
// Returns (program, true) if found, (nil, false) if not found
func (cc *ConditionCache) Get(condition string) (*vm.Program, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if element, found := cc.cache[condition]; found {
		// Move to front (most recently used)
		cc.lruList.MoveToFront(element)
		entry := element.Value.(*cacheEntry)
		return entry.program, true
	}

	return nil, false
}

// Put stores a compiled program in cache
func (cc *ConditionCache) Put(condition string, program *vm.Program) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Check if already exists
	if element, found := cc.cache[condition]; found {
		cc.lruList.MoveToFront(element)
		element.Value.(*cacheEntry).program = program
		return
	}

	// Add new entry
	entry := &cacheEntry{
		key:     condition,
		program: program,
	}
	element := cc.lruList.PushFront(entry)
	cc.cache[condition] = element

	// Evict if over capacity
	if cc.lruList.Len() > cc.capacity {
		cc.evictOldest()
	}
}

// evictOldest removes the least recently used entry
// Must be called with lock held
func (cc *ConditionCache) evictOldest() {
	oldest := cc.lruList.Back()
	if oldest != nil {
		cc.lruList.Remove(oldest)
		entry := oldest.Value.(*cacheEntry)
		delete(cc.cache, entry.key)
	}
}

// Len returns the current number of cached items
func (cc *ConditionCache) Len() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.lruList.Len()
}

// Clear removes all entries from cache
func (cc *ConditionCache) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.cache = make(map[string]*list.Element)
	cc.lruList = list.New()
}

// CompileAndCache compiles an expression and caches it
// Returns the compiled program or an error
func (cc *ConditionCache) CompileAndCache(condition string, env interface{}) (*vm.Program, error) {
	// Try to get from cache first
	if program, found := cc.Get(condition); found {
		return program, nil
	}

	// Compile the expression
	program, err := expr.Compile(condition, expr.Env(env), expr.AsBool())
	if err != nil {
		return nil, err
	}

	// Cache the compiled program
	cc.Put(condition, program)

	return program, nil
}
