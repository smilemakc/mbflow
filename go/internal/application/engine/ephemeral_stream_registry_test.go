package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewEphemeralStreamRegistry(5 * time.Minute)
	notifier := &EphemeralNotifier{}

	reg.Register("exec-1", notifier)

	got, ok := reg.Get("exec-1")
	require.True(t, ok)
	assert.Equal(t, notifier, got)
}

func TestRegistryGetUnknown(t *testing.T) {
	reg := NewEphemeralStreamRegistry(5 * time.Minute)

	got, ok := reg.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistryMarkTerminal(t *testing.T) {
	reg := NewEphemeralStreamRegistry(5 * time.Minute)
	notifier := &EphemeralNotifier{}

	reg.Register("exec-1", notifier)
	reg.MarkTerminal("exec-1")

	// Still accessible after marking terminal (within TTL)
	got, ok := reg.Get("exec-1")
	assert.True(t, ok)
	assert.Equal(t, notifier, got)
}

func TestRegistryCleanup(t *testing.T) {
	// Use very short TTL for testing
	reg := NewEphemeralStreamRegistry(1 * time.Millisecond)
	notifier := &EphemeralNotifier{}

	reg.Register("exec-1", notifier)
	reg.MarkTerminal("exec-1")

	// Wait for TTL to expire
	time.Sleep(5 * time.Millisecond)

	reg.Cleanup()

	_, ok := reg.Get("exec-1")
	assert.False(t, ok, "entry should be cleaned up after TTL")
}

func TestRegistryCleanupKeepsActive(t *testing.T) {
	reg := NewEphemeralStreamRegistry(5 * time.Minute)
	notifier := &EphemeralNotifier{}

	reg.Register("exec-1", notifier)
	// NOT marked terminal

	reg.Cleanup()

	// Should still be there (not terminal, not expired)
	got, ok := reg.Get("exec-1")
	assert.True(t, ok)
	assert.Equal(t, notifier, got)
}

func TestRegistryMarkTerminalUnknown(t *testing.T) {
	reg := NewEphemeralStreamRegistry(5 * time.Minute)
	// Should not panic
	reg.MarkTerminal("nonexistent")
}
