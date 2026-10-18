package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// ToolCallingRegistry управляет выполнением различных типов функций
type ToolCallingRegistry struct {
	builtinRegistry  *models.FunctionRegistry
	workflowExecutor WorkflowExecutor
	codeExecutor     CodeExecutor
	openAPIExecutor  OpenAPIExecutor
	mu               sync.RWMutex
}

// WorkflowExecutor интерфейс для выполнения workflow
type WorkflowExecutor interface {
	ExecuteWorkflow(ctx context.Context, workflowID string, input map[string]any) (any, error)
}

// CodeExecutor интерфейс для выполнения кода
type CodeExecutor interface {
	ExecuteCode(ctx context.Context, language, code string, args map[string]any) (any, error)
}

// OpenAPIExecutor предоставляет операции импортированных API-клиентов как
// инструменты; его реализует APIOperationExecutor.
type OpenAPIExecutor interface {
	// OperationTools строит определения функций для выбранных операций.
	OperationTools(ctx context.Context, refs []models.APIToolRef) ([]models.FunctionDefinition, error)
	// ExecuteOperation вызывает операцию с аргументами tool call.
	ExecuteOperation(ctx context.Context, clientID, operationID string, args map[string]any) (any, error)
}

// NewToolCallingRegistry создает новый registry
func NewToolCallingRegistry(builtinRegistry *models.FunctionRegistry) *ToolCallingRegistry {
	return &ToolCallingRegistry{
		builtinRegistry: builtinRegistry,
	}
}

// SetWorkflowExecutor устанавливает executor для sub-workflows
func (r *ToolCallingRegistry) SetWorkflowExecutor(exec WorkflowExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflowExecutor = exec
}

// SetCodeExecutor устанавливает executor для custom code
func (r *ToolCallingRegistry) SetCodeExecutor(exec CodeExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.codeExecutor = exec
}

// SetOpenAPIExecutor устанавливает executor для OpenAPI
func (r *ToolCallingRegistry) SetOpenAPIExecutor(exec OpenAPIExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.openAPIExecutor = exec
}

// ExecuteFunction выполняет функцию по определению
func (r *ToolCallingRegistry) ExecuteFunction(
	ctx context.Context,
	funcDef *models.FunctionDefinition,
	argumentsJSON string,
) (any, error) {
	// Аргументы операций API попадают в URL и тело запроса, поэтому числа
	// в них не округляются до float64.
	args, err := parseToolArguments(argumentsJSON, funcDef.Type == models.FunctionTypeOpenAPI)
	if err != nil {
		return nil, err
	}

	switch funcDef.Type {
	case models.FunctionTypeBuiltin:
		return r.executeBuiltin(ctx, funcDef, args)
	case models.FunctionTypeSubWorkflow:
		return r.executeSubWorkflow(ctx, funcDef, args)
	case models.FunctionTypeCustomCode:
		return r.executeCustomCode(ctx, funcDef, args)
	case models.FunctionTypeOpenAPI:
		return r.executeOpenAPI(ctx, funcDef, args)
	default:
		return nil, fmt.Errorf("unknown function type: %s", funcDef.Type)
	}
}

func (r *ToolCallingRegistry) executeBuiltin(
	ctx context.Context,
	funcDef *models.FunctionDefinition,
	args map[string]any,
) (any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.builtinRegistry == nil {
		return nil, fmt.Errorf("builtin registry not configured")
	}

	handler, ok := r.builtinRegistry.Get(funcDef.BuiltinName)
	if !ok {
		return nil, fmt.Errorf("builtin function not found: %s", funcDef.BuiltinName)
	}

	return handler(args)
}

func (r *ToolCallingRegistry) executeSubWorkflow(
	ctx context.Context,
	funcDef *models.FunctionDefinition,
	args map[string]any,
) (any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.workflowExecutor == nil {
		return nil, fmt.Errorf("workflow executor not configured")
	}

	// Мапинг аргументов на workflow variables
	workflowInput := make(map[string]any)
	for argName, workflowVar := range funcDef.InputMapping {
		if val, ok := args[argName]; ok {
			workflowInput[workflowVar] = val
		}
	}

	// Выполнить workflow
	result, err := r.workflowExecutor.ExecuteWorkflow(ctx, funcDef.WorkflowID, workflowInput)
	if err != nil {
		return nil, fmt.Errorf("sub-workflow execution failed: %w", err)
	}

	// TODO: Применить output extractor если задан (jq)
	// if funcDef.OutputExtractor != "" {
	//     result = applyJQExtractor(result, funcDef.OutputExtractor)
	// }

	return result, nil
}

func (r *ToolCallingRegistry) executeCustomCode(
	ctx context.Context,
	funcDef *models.FunctionDefinition,
	args map[string]any,
) (any, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.codeExecutor == nil {
		return nil, fmt.Errorf("code executor not configured")
	}

	return r.codeExecutor.ExecuteCode(ctx, funcDef.Language, funcDef.Code, args)
}

func (r *ToolCallingRegistry) executeOpenAPI(
	ctx context.Context,
	funcDef *models.FunctionDefinition,
	args map[string]any,
) (any, error) {
	exec := r.openAPI()
	if exec == nil {
		return nil, fmt.Errorf("OpenAPI executor not configured")
	}
	return exec.ExecuteOperation(ctx, funcDef.ClientID, funcDef.OperationID, args)
}

// APITools возвращает определения функций для операций API-клиентов из
// config "api_tools".
func (r *ToolCallingRegistry) APITools(ctx context.Context, refs []models.APIToolRef) ([]models.FunctionDefinition, error) {
	exec := r.openAPI()
	if exec == nil {
		return nil, fmt.Errorf("API tools are not available: OpenAPI executor not configured")
	}
	return exec.OperationTools(ctx, refs)
}

func (r *ToolCallingRegistry) openAPI() OpenAPIExecutor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.openAPIExecutor
}

func parseToolArguments(argumentsJSON string, exactNumbers bool) (map[string]any, error) {
	var args map[string]any
	if strings.TrimSpace(argumentsJSON) == "" {
		return args, nil
	}
	decoder := json.NewDecoder(strings.NewReader(argumentsJSON))
	if exactNumbers {
		decoder.UseNumber()
	}
	if err := decoder.Decode(&args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("failed to parse arguments: unexpected data after the JSON object")
	}
	return args, nil
}

// ValidateFunctionDefinition валидирует определение функции
func (r *ToolCallingRegistry) ValidateFunctionDefinition(funcDef *models.FunctionDefinition) error {
	if funcDef.Name == "" {
		return fmt.Errorf("function name is required")
	}

	switch funcDef.Type {
	case models.FunctionTypeBuiltin:
		if funcDef.BuiltinName == "" {
			return fmt.Errorf("builtin_name is required for builtin functions")
		}
		// Проверить что функция существует
		r.mu.RLock()
		defer r.mu.RUnlock()
		if r.builtinRegistry != nil {
			if _, ok := r.builtinRegistry.Get(funcDef.BuiltinName); !ok {
				return fmt.Errorf("builtin function not found: %s", funcDef.BuiltinName)
			}
		}

	case models.FunctionTypeSubWorkflow:
		if funcDef.WorkflowID == "" {
			return fmt.Errorf("workflow_id is required for sub-workflow functions")
		}

	case models.FunctionTypeCustomCode:
		if funcDef.Language == "" || funcDef.Code == "" {
			return fmt.Errorf("language and code are required for custom code functions")
		}
		if funcDef.Language != "javascript" && funcDef.Language != "python" {
			return fmt.Errorf("unsupported language: %s (supported: javascript, python)", funcDef.Language)
		}

	case models.FunctionTypeOpenAPI:
		if funcDef.ClientID == "" || funcDef.OperationID == "" {
			return fmt.Errorf("client_id and operation_id are required for OpenAPI functions")
		}

	default:
		return fmt.Errorf("unknown function type: %s", funcDef.Type)
	}

	// Validate JSON Schema (базовая проверка)
	if funcDef.Parameters != nil {
		if _, err := json.Marshal(funcDef.Parameters); err != nil {
			return fmt.Errorf("invalid parameters schema: %w", err)
		}
	}

	return nil
}
