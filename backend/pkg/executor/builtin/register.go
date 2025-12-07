package builtin

import (
	"github.com/smilemakc/mbflow/internal/application/filestorage"
	"github.com/smilemakc/mbflow/pkg/executor"
)

// RegisterBuiltins registers all built-in executors with the given manager.
// This function should be called by applications that want to use built-in executors.
// Note: file_storage executor requires RegisterFileStorage to be called separately
// with a filestorage.Manager instance.
func RegisterBuiltins(manager executor.Manager) error {
	executors := map[string]executor.Executor{
		"http":          NewHTTPExecutor(),
		"transform":     NewTransformExecutor(),
		"llm":           NewLLMExecutor(),
		"function_call": NewFunctionCallExecutor(),
		"telegram":      NewTelegramExecutor(),
		"conditional":   NewConditionalExecutor(),
		"merge":         NewMergeExecutor(),
	}

	for name, exec := range executors {
		if err := manager.Register(name, exec); err != nil {
			return err
		}
	}

	return nil
}

// RegisterFileStorage registers the file_storage executor with the given manager.
// This must be called after RegisterBuiltins if file storage functionality is needed.
func RegisterFileStorage(manager executor.Manager, storageManager filestorage.Manager) error {
	return manager.Register("file_storage", NewFileStorageExecutor(storageManager))
}

// MustRegisterBuiltins registers all built-in executors and panics on error.
// This is a convenience function for initialization code.
func MustRegisterBuiltins(manager executor.Manager) {
	if err := RegisterBuiltins(manager); err != nil {
		panic("failed to register built-in executors: " + err.Error())
	}
}

// MustRegisterFileStorage registers file_storage executor and panics on error.
func MustRegisterFileStorage(manager executor.Manager, storageManager filestorage.Manager) {
	if err := RegisterFileStorage(manager, storageManager); err != nil {
		panic("failed to register file_storage executor: " + err.Error())
	}
}
