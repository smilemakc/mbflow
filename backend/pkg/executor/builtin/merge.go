package builtin

import (
	"context"
	"fmt"

	"github.com/smilemakc/mbflow/pkg/executor"
)

// MergeExecutor combines outputs from multiple nodes.
type MergeExecutor struct {
	*executor.BaseExecutor
}

// NewMergeExecutor creates a new merge executor.
func NewMergeExecutor() *MergeExecutor {
	return &MergeExecutor{
		BaseExecutor: executor.NewBaseExecutor("merge"),
	}
}

// Execute executes the merge logic.
func (e *MergeExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	mergeStrategy := e.GetStringDefault(config, "merge_strategy", "all")

	switch mergeStrategy {
	case "all":
		// For 'all' strategy, the engine should have already collected all inputs.
		// We simply pass through the collected input.
		return input, nil

	case "any":
		// For 'any' strategy, the engine might trigger this for the first arriving input.
		// We also pass through the input.
		return input, nil

	default:
		return nil, fmt.Errorf("unknown merge strategy: %s", mergeStrategy)
	}
}

// Validate validates the merge executor configuration.
func (e *MergeExecutor) Validate(config map[string]interface{}) error {
	mergeStrategy := e.GetStringDefault(config, "merge_strategy", "all")

	validStrategies := map[string]bool{
		"all": true,
		"any": true,
	}

	if !validStrategies[mergeStrategy] {
		return fmt.Errorf("invalid merge strategy: %s", mergeStrategy)
	}

	return nil
}
