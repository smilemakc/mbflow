package engine

import (
	"sync"
	"testing"

	"github.com/expr-lang/expr"
)

func TestConditionCache_GetPut(t *testing.T) {
	t.Parallel()
	cache := NewConditionCache(3)

	// Compile a test expression
	program, err := expr.Compile("x > 5", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	// Test Put and Get
	cache.Put("x > 5", program)

	retrieved, found := cache.Get("x > 5")
	if !found {
		t.Error("expected to find cached program")
	}
	if retrieved != program {
		t.Error("retrieved program doesn't match stored program")
	}

	// Test Get non-existent
	_, found = cache.Get("y > 10")
	if found {
		t.Error("should not find non-existent program")
	}
}

func TestConditionCache_Eviction(t *testing.T) {
	t.Parallel()
	cache := NewConditionCache(2) // Capacity of 2

	// Compile test expressions
	prog1, _ := expr.Compile("x > 1", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())
	prog2, _ := expr.Compile("x > 2", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())
	prog3, _ := expr.Compile("x > 3", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())

	// Add first two
	cache.Put("x > 1", prog1)
	cache.Put("x > 2", prog2)

	if cache.Len() != 2 {
		t.Errorf("expected cache length 2, got %d", cache.Len())
	}

	// Add third, should evict oldest (x > 1)
	cache.Put("x > 3", prog3)

	if cache.Len() != 2 {
		t.Errorf("expected cache length 2 after eviction, got %d", cache.Len())
	}

	// Check that x > 1 was evicted
	_, found := cache.Get("x > 1")
	if found {
		t.Error("oldest entry should have been evicted")
	}

	// Check that x > 2 and x > 3 are still there
	_, found = cache.Get("x > 2")
	if !found {
		t.Error("x > 2 should still be in cache")
	}

	_, found = cache.Get("x > 3")
	if !found {
		t.Error("x > 3 should be in cache")
	}
}

func TestConditionCache_LRUBehavior(t *testing.T) {
	t.Parallel()
	cache := NewConditionCache(2)

	prog1, _ := expr.Compile("x > 1", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())
	prog2, _ := expr.Compile("x > 2", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())
	prog3, _ := expr.Compile("x > 3", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())

	// Add two items
	cache.Put("x > 1", prog1)
	cache.Put("x > 2", prog2)

	// Access x > 1 to make it recently used
	cache.Get("x > 1")

	// Add x > 3, should evict x > 2 (least recently used)
	cache.Put("x > 3", prog3)

	// x > 1 should still be there
	_, found := cache.Get("x > 1")
	if !found {
		t.Error("x > 1 should still be in cache (was accessed recently)")
	}

	// x > 2 should be evicted
	_, found = cache.Get("x > 2")
	if found {
		t.Error("x > 2 should have been evicted (least recently used)")
	}

	// x > 3 should be there
	_, found = cache.Get("x > 3")
	if !found {
		t.Error("x > 3 should be in cache")
	}
}

func TestConditionCache_UpdateExisting(t *testing.T) {
	t.Parallel()
	cache := NewConditionCache(3)

	prog1, _ := expr.Compile("x > 1", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())
	prog2, _ := expr.Compile("x > 2", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())

	// Add initial program
	cache.Put("test", prog1)

	// Update with different program
	cache.Put("test", prog2)

	// Should still have length 1
	if cache.Len() != 1 {
		t.Errorf("expected length 1 after update, got %d", cache.Len())
	}

	// Should get the updated program
	retrieved, found := cache.Get("test")
	if !found {
		t.Error("program should be found")
	}
	if retrieved != prog2 {
		t.Error("should get updated program")
	}
}

func TestConditionCache_Clear(t *testing.T) {
	t.Parallel()
	cache := NewConditionCache(10)

	prog1, _ := expr.Compile("x > 1", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())
	prog2, _ := expr.Compile("x > 2", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())

	cache.Put("x > 1", prog1)
	cache.Put("x > 2", prog2)

	if cache.Len() != 2 {
		t.Errorf("expected length 2, got %d", cache.Len())
	}

	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("expected length 0 after clear, got %d", cache.Len())
	}

	_, found := cache.Get("x > 1")
	if found {
		t.Error("cache should be empty after clear")
	}
}

func TestConditionCache_CompileAndCache(t *testing.T) {
	t.Parallel()
	cache := NewConditionCache(10)

	env := map[string]interface{}{"x": 10}

	// First call should compile and cache
	prog1, err := cache.CompileAndCache("x > 5", env)
	if err != nil {
		t.Fatalf("failed to compile and cache: %v", err)
	}

	// Second call should retrieve from cache
	prog2, err := cache.CompileAndCache("x > 5", env)
	if err != nil {
		t.Fatalf("failed to get from cache: %v", err)
	}

	// Should be the same program
	if prog1 != prog2 {
		t.Error("should retrieve same program from cache")
	}

	// Test with invalid expression
	_, err = cache.CompileAndCache("invalid expression >>>", env)
	if err == nil {
		t.Error("expected error for invalid expression")
	}
}

func TestConditionCache_ThreadSafety(t *testing.T) {
	t.Parallel()
	cache := NewConditionCache(100)
	var wg sync.WaitGroup

	// Spawn multiple goroutines doing concurrent reads and writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				condition := "x > 5"
				prog, _ := expr.Compile(condition, expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())

				// Put
				cache.Put(condition, prog)

				// Get
				cache.Get(condition)

				// CompileAndCache
				cache.CompileAndCache(condition, map[string]interface{}{"x": 0})
			}
		}(i)
	}

	wg.Wait()

	// If we get here without deadlock or race, test passes
}

func TestConditionCache_ZeroCapacity(t *testing.T) {
	t.Parallel()
	cache := NewConditionCache(0) // Should default to 100

	prog, _ := expr.Compile("x > 5", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())
	cache.Put("x > 5", prog)

	_, found := cache.Get("x > 5")
	if !found {
		t.Error("cache with zero capacity should default to non-zero")
	}
}

func TestConditionCache_NegativeCapacity(t *testing.T) {
	t.Parallel()
	cache := NewConditionCache(-5) // Should default to 100

	prog, _ := expr.Compile("x > 5", expr.Env(map[string]interface{}{"x": 0}), expr.AsBool())
	cache.Put("x > 5", prog)

	_, found := cache.Get("x > 5")
	if !found {
		t.Error("cache with negative capacity should default to non-zero")
	}
}
