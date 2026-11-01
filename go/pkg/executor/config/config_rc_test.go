package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRC_FileStorageConfig_WriteRequiresPath(t *testing.T) {
	t.Parallel()
	err := (&FileStorageConfig{Operation: "write"}).Validate()
	require.EqualError(t, err, "path is required for write operation")
	require.NoError(t, (&FileStorageConfig{Operation: "write", Path: "a.txt"}).Validate())
	require.NoError(t, (&FileStorageConfig{Operation: "list"}).Validate())
}

func TestRC_ParseConfig_Errors(t *testing.T) {
	t.Parallel()

	_, err := ParseConfig[HTTPConfig](map[string]any{"bad": make(chan int)})
	require.ErrorContains(t, err, "failed to marshal config")

	// A value of the wrong JSON type cannot be decoded into the struct.
	_, err = ParseConfig[FileStorageConfig](map[string]any{"operation": 5})
	require.ErrorContains(t, err, "failed to parse config")

	cfg, err := ParseConfig[FileStorageConfig](map[string]any{"operation": "read", "path": "p"})
	require.NoError(t, err)
	assert.Equal(t, "p", cfg.Path)
}

func TestRC_ToMap_Errors(t *testing.T) {
	t.Parallel()

	_, err := ToMap(map[string]any{"c": make(chan int)})
	require.ErrorContains(t, err, "failed to marshal config")

	// A JSON array is not an object.
	_, err = ToMap([]int{1, 2})
	require.ErrorContains(t, err, "failed to convert to map")

	m, err := ToMap(FileStorageConfig{Operation: "list"})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"operation": "list"}, m)
}
