package executor

import (
	"encoding/json"
	"fmt"
)

// parseConfig converts a map[string]any configuration to a typed struct.
// It uses JSON marshal/unmarshal to perform the conversion, which handles
// type conversions automatically (e.g., float64 -> int from YAML parsing).
func parseConfig[T any](config map[string]any) (*T, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	// Marshal map to JSON bytes
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Unmarshal JSON bytes to typed struct
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &result, nil
}
