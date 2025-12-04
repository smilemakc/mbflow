package builtin

import "github.com/smilemakc/mbflow/pkg/executor"

// RegisterBuiltins registers all built-in executors with the given manager.
// This function should be called by applications that want to use built-in executors.
func RegisterBuiltins(manager executor.Manager) error {
	executors := map[string]executor.Executor{
		"http":          NewHTTPExecutor(),
		"transform":     NewTransformExecutor(),
		"llm":           NewLLMExecutor(),
		"function_call": NewFunctionCallExecutor(),
		"telegram":      NewTelegramExecutor(),
	}

	for name, exec := range executors {
		if err := manager.Register(name, exec); err != nil {
			return err
		}
	}

	return nil
}

// MustRegisterBuiltins registers all built-in executors and panics on error.
// This is a convenience function for initialization code.
func MustRegisterBuiltins(manager executor.Manager) {
	if err := RegisterBuiltins(manager); err != nil {
		panic("failed to register built-in executors: " + err.Error())
	}
}
