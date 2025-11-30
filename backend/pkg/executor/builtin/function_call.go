package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// FunctionCallExecutor executes function calls from LLM responses.
// It maintains a registry of function handlers that can be registered at runtime.
type FunctionCallExecutor struct {
	*executor.BaseExecutor
	registry *models.FunctionRegistry
	mu       sync.RWMutex
	client   *http.Client
}

// NewFunctionCallExecutor creates a new function call executor.
func NewFunctionCallExecutor() *FunctionCallExecutor {
	exec := &FunctionCallExecutor{
		BaseExecutor: executor.NewBaseExecutor("function_call"),
		registry:     models.NewFunctionRegistry(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Register built-in functions
	exec.registerBuiltInFunctions()

	return exec
}

// RegisterFunction registers a custom function handler.
func (e *FunctionCallExecutor) RegisterFunction(name string, handler models.FunctionHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.registry.Register(name, handler)
}

// UnregisterFunction removes a function handler.
func (e *FunctionCallExecutor) UnregisterFunction(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.registry.Unregister(name)
}

// ListFunctions returns all registered function names.
func (e *FunctionCallExecutor) ListFunctions() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.registry.List()
}

// Execute executes a function call.
// Expected input formats:
// 1. Map with function_name and arguments fields
// 2. FunctionCallInput struct
// 3. LLM tool_calls array (will execute first tool call)
func (e *FunctionCallExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	// Parse input into FunctionCallInput
	funcInput, err := e.parseInput(config, input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse function call input: %w", err)
	}

	// Parse arguments
	args, err := funcInput.ParseArguments()
	if err != nil {
		return nil, fmt.Errorf("failed to parse function arguments: %w", err)
	}

	// Get function handler
	handler, ok := e.registry.Get(funcInput.FunctionName)
	if !ok {
		return e.buildErrorOutput(funcInput, fmt.Errorf("function not found: %s", funcInput.FunctionName)), nil
	}

	// Execute function
	result, err := handler(args)
	if err != nil {
		return e.buildErrorOutput(funcInput, err), nil
	}

	// Build output
	return e.buildSuccessOutput(funcInput, result), nil
}

// Validate validates the function call executor configuration.
func (e *FunctionCallExecutor) Validate(config map[string]interface{}) error {
	// Check if function_name is provided in config
	if functionName, ok := config["function_name"].(string); ok {
		if functionName == "" {
			return fmt.Errorf("function_name cannot be empty")
		}
	}

	// Arguments can be optional or come from input
	return nil
}

// parseInput parses various input formats into FunctionCallInput.
func (e *FunctionCallExecutor) parseInput(config map[string]interface{}, input interface{}) (*models.FunctionCallInput, error) {
	funcInput := &models.FunctionCallInput{}

	// Try to get function_name from config first (template-resolved)
	if functionName, ok := config["function_name"].(string); ok && functionName != "" {
		funcInput.FunctionName = functionName
	}

	// Try to get arguments from config (template-resolved)
	if arguments, ok := config["arguments"].(string); ok && arguments != "" {
		funcInput.Arguments = arguments
	}

	// Try to get tool_call_id from config
	if toolCallID, ok := config["tool_call_id"].(string); ok {
		funcInput.ToolCallID = toolCallID
	}

	// If function_name not in config, try to parse from input
	if funcInput.FunctionName == "" {
		if err := e.parseInputData(funcInput, input); err != nil {
			return nil, err
		}
	}

	// Validate that we have required fields
	if funcInput.FunctionName == "" {
		return nil, fmt.Errorf("function_name is required")
	}

	if funcInput.Arguments == "" {
		funcInput.Arguments = "{}" // Default to empty object
	}

	return funcInput, nil
}

// parseInputData parses input data into FunctionCallInput.
func (e *FunctionCallExecutor) parseInputData(funcInput *models.FunctionCallInput, input interface{}) error {
	switch v := input.(type) {
	case map[string]interface{}:
		// Check if input contains tool_calls (from LLM output)
		if toolCalls, ok := v["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
			return e.parseToolCall(funcInput, toolCalls[0])
		}

		// Check if input is a direct function call
		if functionName, ok := v["function_name"].(string); ok {
			funcInput.FunctionName = functionName
			if args, ok := v["arguments"].(string); ok {
				funcInput.Arguments = args
			}
			if toolCallID, ok := v["tool_call_id"].(string); ok {
				funcInput.ToolCallID = toolCallID
			}
			return nil
		}

		// Check if input has function field (tool call format)
		if function, ok := v["function"].(map[string]interface{}); ok {
			return e.parseFunctionField(funcInput, v, function)
		}

	case *models.FunctionCallInput:
		*funcInput = *v
		return nil

	case string:
		// Try to parse as JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(v), &data); err != nil {
			return fmt.Errorf("failed to parse input JSON: %w", err)
		}
		return e.parseInputData(funcInput, data)
	}

	return fmt.Errorf("unsupported input format")
}

// parseToolCall parses a tool call from LLM output.
func (e *FunctionCallExecutor) parseToolCall(funcInput *models.FunctionCallInput, toolCall interface{}) error {
	toolCallMap, ok := toolCall.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid tool call format")
	}

	if id, ok := toolCallMap["id"].(string); ok {
		funcInput.ToolCallID = id
	}

	function, ok := toolCallMap["function"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("tool call missing function field")
	}

	return e.parseFunctionField(funcInput, toolCallMap, function)
}

// parseFunctionField parses the function field.
func (e *FunctionCallExecutor) parseFunctionField(funcInput *models.FunctionCallInput, parent map[string]interface{}, function map[string]interface{}) error {
	name, ok := function["name"].(string)
	if !ok {
		return fmt.Errorf("function name is required")
	}
	funcInput.FunctionName = name

	args, ok := function["arguments"].(string)
	if !ok {
		return fmt.Errorf("function arguments must be a string")
	}
	funcInput.Arguments = args

	// Try to get ID from parent (tool call format)
	if id, ok := parent["id"].(string); ok {
		funcInput.ToolCallID = id
	}

	return nil
}

// buildSuccessOutput builds a success output.
func (e *FunctionCallExecutor) buildSuccessOutput(input *models.FunctionCallInput, result interface{}) *models.FunctionCallOutput {
	return &models.FunctionCallOutput{
		Result:       result,
		FunctionName: input.FunctionName,
		ToolCallID:   input.ToolCallID,
		Success:      true,
	}
}

// buildErrorOutput builds an error output.
func (e *FunctionCallExecutor) buildErrorOutput(input *models.FunctionCallInput, err error) *models.FunctionCallOutput {
	return &models.FunctionCallOutput{
		Result:       nil,
		FunctionName: input.FunctionName,
		ToolCallID:   input.ToolCallID,
		Success:      false,
		Error:        err.Error(),
	}
}

// registerBuiltInFunctions registers built-in function handlers.
func (e *FunctionCallExecutor) registerBuiltInFunctions() {
	// get_current_time - Returns current time in various formats
	e.registry.Register("get_current_time", func(args map[string]interface{}) (interface{}, error) {
		format := "RFC3339"
		if f, ok := args["format"].(string); ok {
			format = f
		}

		now := time.Now()

		switch format {
		case "RFC3339":
			return now.Format(time.RFC3339), nil
		case "unix":
			return now.Unix(), nil
		case "iso8601":
			return now.Format("2006-01-02T15:04:05Z07:00"), nil
		default:
			return now.Format(format), nil
		}
	})

	// get_current_date - Returns current date
	e.registry.Register("get_current_date", func(args map[string]interface{}) (interface{}, error) {
		return time.Now().Format("2006-01-02"), nil
	})

	// http_request - Makes an HTTP request
	e.registry.Register("http_request", func(args map[string]interface{}) (interface{}, error) {
		method, ok := args["method"].(string)
		if !ok {
			method = "GET"
		}

		url, ok := args["url"].(string)
		if !ok {
			return nil, fmt.Errorf("url is required")
		}

		var body io.Reader
		if bodyData, ok := args["body"]; ok {
			bodyJSON, err := json.Marshal(bodyData)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			body = http.NoBody
			if len(bodyJSON) > 0 {
				body = io.NopCloser(http.NoBody)
			}
		}

		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		if headers, ok := args["headers"].(map[string]interface{}); ok {
			for key, value := range headers {
				if strVal, ok := value.(string); ok {
					req.Header.Set(key, strVal)
				}
			}
		}

		resp, err := e.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var result interface{}
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &result); err != nil {
				result = string(respBody)
			}
		}

		return map[string]interface{}{
			"status": resp.StatusCode,
			"body":   result,
		}, nil
	})

	// json_parse - Parses a JSON string
	e.registry.Register("json_parse", func(args map[string]interface{}) (interface{}, error) {
		jsonStr, ok := args["json"].(string)
		if !ok {
			return nil, fmt.Errorf("json parameter is required")
		}

		var result interface{}
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		return result, nil
	})

	// json_stringify - Converts a value to JSON string
	e.registry.Register("json_stringify", func(args map[string]interface{}) (interface{}, error) {
		value, ok := args["value"]
		if !ok {
			return nil, fmt.Errorf("value parameter is required")
		}

		pretty := false
		if p, ok := args["pretty"].(bool); ok {
			pretty = p
		}

		var data []byte
		var err error

		if pretty {
			data, err = json.MarshalIndent(value, "", "  ")
		} else {
			data, err = json.Marshal(value)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to stringify: %w", err)
		}

		return string(data), nil
	})

	// Example weather function (mock implementation)
	e.registry.Register("get_weather", func(args map[string]interface{}) (interface{}, error) {
		location, ok := args["location"].(string)
		if !ok {
			return nil, fmt.Errorf("location is required")
		}

		unit := "celsius"
		if u, ok := args["unit"].(string); ok {
			unit = u
		}

		// Mock weather data
		return map[string]interface{}{
			"location":    location,
			"temperature": 22,
			"unit":        unit,
			"condition":   "sunny",
			"humidity":    65,
		}, nil
	})
}
