package executor

import (
	"context"

	"github.com/smilemakc/mbflow/internal/application/template"
)

// TemplateExecutorWrapper wraps an executor to automatically resolve templates in its configuration.
// This allows all executors to benefit from template substitution without modifying their code.
type TemplateExecutorWrapper struct {
	executor Executor
	engine   *template.Engine
}

// NewTemplateExecutorWrapper creates a new template-aware executor wrapper.
func NewTemplateExecutorWrapper(executor Executor, engine *template.Engine) Executor {
	if engine == nil {
		// If no engine provided, return the executor as-is
		return executor
	}

	return &TemplateExecutorWrapper{
		executor: executor,
		engine:   engine,
	}
}

// Execute resolves templates in the config before executing.
func (w *TemplateExecutorWrapper) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	// Resolve templates in the config
	resolvedConfig, err := w.engine.ResolveConfig(config)
	if err != nil {
		return nil, err
	}

	// Execute with resolved config
	return w.executor.Execute(ctx, resolvedConfig, input)
}

// Validate validates the config without resolving templates.
// Template validation happens at execution time.
func (w *TemplateExecutorWrapper) Validate(config map[string]interface{}) error {
	return w.executor.Validate(config)
}

// ExecutionContextKey is used to store execution context in context.Context
type ExecutionContextKey struct{}

// ExecutionContextData holds data needed for template resolution during execution.
type ExecutionContextData struct {
	WorkflowVariables  map[string]interface{}
	ExecutionVariables map[string]interface{}
	ParentNodeOutput   map[string]interface{}
	Resources          map[string]interface{} // alias -> resource data
	StrictMode         bool
}

// GetExecutionContext retrieves execution context from context.Context.
func GetExecutionContext(ctx context.Context) (*ExecutionContextData, bool) {
	data, ok := ctx.Value(ExecutionContextKey{}).(*ExecutionContextData)
	return data, ok
}

// WithExecutionContext adds execution context to context.Context.
func WithExecutionContext(ctx context.Context, data *ExecutionContextData) context.Context {
	return context.WithValue(ctx, ExecutionContextKey{}, data)
}

// NewTemplateEngine creates a template engine from execution context.
func NewTemplateEngine(execCtx *ExecutionContextData) *template.Engine {
	varCtx := template.NewVariableContext()
	varCtx.WorkflowVars = execCtx.WorkflowVariables
	varCtx.ExecutionVars = execCtx.ExecutionVariables
	varCtx.InputVars = execCtx.ParentNodeOutput
	varCtx.ResourceVars = execCtx.Resources

	opts := template.TemplateOptions{
		StrictMode:           execCtx.StrictMode,
		PlaceholderOnMissing: false,
	}

	return template.NewEngine(varCtx, opts)
}
