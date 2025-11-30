package models

import (
	"encoding/json"
	"fmt"
)

// FunctionHandler is a function that executes a function call.
type FunctionHandler func(args map[string]interface{}) (interface{}, error)

// FunctionCallInput represents the input to a function call executor.
type FunctionCallInput struct {
	FunctionName string                 `json:"function_name"`
	Arguments    string                 `json:"arguments"` // JSON string
	ToolCallID   string                 `json:"tool_call_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// FunctionCallOutput represents the output from a function call executor.
type FunctionCallOutput struct {
	Result       interface{}            `json:"result"`
	FunctionName string                 `json:"function_name"`
	ToolCallID   string                 `json:"tool_call_id,omitempty"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// FunctionCallConfig represents the configuration for a function call executor.
type FunctionCallConfig struct {
	FunctionName string `json:"function_name"`
	Arguments    string `json:"arguments"`
	ToolCallID   string `json:"tool_call_id,omitempty"`
}

// ParseArguments parses the arguments JSON string into a map.
func (f *FunctionCallInput) ParseArguments() (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(f.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse function arguments: %w", err)
	}
	return args, nil
}

// FunctionRegistry is a registry of function handlers.
type FunctionRegistry struct {
	handlers map[string]FunctionHandler
}

// NewFunctionRegistry creates a new function registry.
func NewFunctionRegistry() *FunctionRegistry {
	return &FunctionRegistry{
		handlers: make(map[string]FunctionHandler),
	}
}

// Register registers a function handler.
func (r *FunctionRegistry) Register(name string, handler FunctionHandler) {
	r.handlers[name] = handler
}

// Get retrieves a function handler by name.
func (r *FunctionRegistry) Get(name string) (FunctionHandler, bool) {
	handler, ok := r.handlers[name]
	return handler, ok
}

// Has checks if a function handler exists.
func (r *FunctionRegistry) Has(name string) bool {
	_, ok := r.handlers[name]
	return ok
}

// List returns all registered function names.
func (r *FunctionRegistry) List() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

// Unregister removes a function handler.
func (r *FunctionRegistry) Unregister(name string) {
	delete(r.handlers, name)
}
