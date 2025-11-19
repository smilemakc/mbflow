package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"

	"mbflow/internal/domain/errors"
)

// NodeExecutor defines the interface for executing different types of nodes.
// Each node type has its own executor implementation.
type NodeExecutor interface {
	// Execute executes the node and returns the output
	Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error)

	// Type returns the node type this executor handles
	Type() string
}

// OpenAICompletionExecutor executes OpenAI completion nodes.
// It sends requests to the OpenAI API and returns the generated text.
type OpenAICompletionExecutor struct {
	client *openai.Client
}

// NewOpenAICompletionExecutor creates a new OpenAICompletionExecutor.
func NewOpenAICompletionExecutor(apiKey string) *OpenAICompletionExecutor {
	client := openai.NewClient(apiKey)
	return &OpenAICompletionExecutor{
		client: client,
	}
}

// Type returns the node type.
func (e *OpenAICompletionExecutor) Type() string {
	return "openai-completion"
}

// Execute executes an OpenAI completion node.
func (e *OpenAICompletionExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// Extract configuration
	model, ok := config["model"].(string)
	if !ok {
		model = "gpt-4"
	}

	promptTemplate, ok := config["prompt"].(string)
	if !ok {
		return nil, errors.NewConfigurationError("openai-completion", "missing 'prompt' in config")
	}

	maxTokens := 1000
	if mt, ok := config["max_tokens"].(int); ok {
		maxTokens = mt
	} else if mt, ok := config["max_tokens"].(float64); ok {
		maxTokens = int(mt)
	}

	temperature := 0.7
	if temp, ok := config["temperature"].(float64); ok {
		temperature = temp
	}

	outputKey, ok := config["output_key"].(string)
	if !ok {
		outputKey = "output"
	}

	// Substitute variables in prompt
	prompt := substituteVariables(promptTemplate, execCtx.GetAllVariables())

	// Create OpenAI request
	req := openai.ChatCompletionRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: float32(temperature),
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	// Call OpenAI API
	startTime := time.Now()
	resp, err := e.client.CreateChatCompletion(ctx, req)
	latency := time.Since(startTime)

	if err != nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"openai-completion",
			1,
			fmt.Sprintf("OpenAI API error: %v", err),
			err,
			true, // Retryable
		)
	}

	if len(resp.Choices) == 0 {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"openai-completion",
			1,
			"OpenAI returned no choices",
			nil,
			false,
		)
	}

	// Extract response
	content := resp.Choices[0].Message.Content

	// Store output in execution context
	execCtx.SetVariable(outputKey, content)
	// Return result with metadata
	return map[string]interface{}{
		"content":           content,
		"model":             resp.Model,
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
		"latency_ms":        latency.Milliseconds(),
	}, nil
}

// HTTPRequestExecutor executes HTTP request nodes.
// It sends HTTP requests and returns the response.
type HTTPRequestExecutor struct {
	client *http.Client
}

// NewHTTPRequestExecutor creates a new HTTPRequestExecutor.
func NewHTTPRequestExecutor() *HTTPRequestExecutor {
	return &HTTPRequestExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Type returns the node type.
func (e *HTTPRequestExecutor) Type() string {
	return "http-request"
}

// Execute executes an HTTP request node.
func (e *HTTPRequestExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// Extract configuration
	urlTemplate, ok := config["url"].(string)
	if !ok {
		return nil, errors.NewConfigurationError("http-request", "missing 'url' in config")
	}

	method, ok := config["method"].(string)
	if !ok {
		method = "GET"
	}

	outputKey, ok := config["output_key"].(string)
	if !ok {
		outputKey = "output"
	}

	// Substitute variables
	url := substituteVariables(urlTemplate, execCtx.GetAllVariables())

	// Prepare request body
	var body io.Reader
	if bodyData, ok := config["body"]; ok {
		var bodyBytes []byte
		var err error

		switch v := bodyData.(type) {
		case string:
			// Substitute variables in string body
			bodyStr := substituteVariables(v, execCtx.GetAllVariables())
			bodyBytes = []byte(bodyStr)
		case map[string]interface{}:
			// JSON encode map
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				return nil, errors.NewConfigurationError("http-request", fmt.Sprintf("failed to marshal body: %v", err))
			}
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				return nil, errors.NewConfigurationError("http-request", fmt.Sprintf("failed to marshal body: %v", err))
			}
		}

		body = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"http-request",
			1,
			fmt.Sprintf("failed to create request: %v", err),
			err,
			false,
		)
	}

	// Set headers
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, substituteVariables(strValue, execCtx.GetAllVariables()))
			}
		}
	} else if headers, ok := config["headers"].(map[string]string); ok {
		for key, value := range headers {
			req.Header.Set(key, substituteVariables(value, execCtx.GetAllVariables()))
		}
	}

	// Send request
	startTime := time.Now()
	resp, err := e.client.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"http-request",
			1,
			fmt.Sprintf("HTTP request failed: %v", err),
			err,
			true, // Retryable
		)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"http-request",
			1,
			fmt.Sprintf("failed to read response: %v", err),
			err,
			true,
		)
	}

	// Try to parse as JSON
	var jsonResp interface{}
	if err := json.Unmarshal(respBody, &jsonResp); err == nil {
		// Store JSON response
		execCtx.SetVariable(outputKey, jsonResp)

		return map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        jsonResp,
			"latency_ms":  latency.Milliseconds(),
		}, nil
	}

	// Store as string if not JSON
	respStr := string(respBody)
	execCtx.SetVariable(outputKey, respStr)

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        respStr,
		"latency_ms":  latency.Milliseconds(),
	}, nil
}

// ConditionalRouterExecutor executes conditional routing nodes.
// It evaluates conditions and determines which path to take.
type ConditionalRouterExecutor struct{}

// NewConditionalRouterExecutor creates a new ConditionalRouterExecutor.
func NewConditionalRouterExecutor() *ConditionalRouterExecutor {
	return &ConditionalRouterExecutor{}
}

// Type returns the node type.
func (e *ConditionalRouterExecutor) Type() string {
	return "conditional-router"
}

// Execute executes a conditional router node.
func (e *ConditionalRouterExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// Extract configuration
	inputKey, ok := config["input_key"].(string)
	if !ok {
		return nil, errors.NewConfigurationError("conditional-router", "missing 'input_key' in config")
	}

	// Handle routes as either map[string]interface{} or map[string]string
	var routes map[string]interface{}
	if r, ok := config["routes"].(map[string]interface{}); ok {
		routes = r
	} else if r, ok := config["routes"].(map[string]string); ok {
		// Convert map[string]string to map[string]interface{}
		routes = make(map[string]interface{})
		for k, v := range r {
			routes[k] = v
		}
	} else {
		return nil, errors.NewConfigurationError("conditional-router", "missing or invalid 'routes' in config")
	}

	// Get input value
	inputValue, ok := execCtx.GetVariable(inputKey)
	if !ok {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"conditional-router",
			1,
			fmt.Sprintf("input variable '%s' not found", inputKey),
			nil,
			false,
		)
	}

	// Convert input to string for comparison
	inputStr := fmt.Sprintf("%v", inputValue)

	// Find matching route
	var selectedRoute string
	for condition, route := range routes {
		if condition == inputStr {
			selectedRoute = fmt.Sprintf("%v", route)
			break
		}
	}

	// Check for default route
	if selectedRoute == "" {
		if defaultRoute, ok := routes["default"]; ok {
			selectedRoute = fmt.Sprintf("%v", defaultRoute)
		} else {
			return nil, errors.NewNodeExecutionError(
				execCtx.State().WorkflowID,
				execCtx.State().ExecutionID,
				nodeID,
				"conditional-router",
				1,
				fmt.Sprintf("no route found for value '%s' and no default route", inputStr),
				nil,
				false,
			)
		}
	}

	return map[string]interface{}{
		"input_value":    inputStr,
		"selected_route": selectedRoute,
	}, nil
}

// DataMergerExecutor executes data merger nodes.
// It merges data from multiple sources.
type DataMergerExecutor struct{}

// NewDataMergerExecutor creates a new DataMergerExecutor.
func NewDataMergerExecutor() *DataMergerExecutor {
	return &DataMergerExecutor{}
}

// Type returns the node type.
func (e *DataMergerExecutor) Type() string {
	return "data-merger"
}

// Execute executes a data merger node.
func (e *DataMergerExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// Extract configuration
	strategy, ok := config["strategy"].(string)
	if !ok {
		strategy = "select_first_available"
	}

	sources, ok := config["sources"].([]interface{})
	if !ok {
		if strSources, ok := config["sources"].([]string); ok {
			sources = make([]interface{}, len(strSources))
			for i, s := range strSources {
				sources[i] = s
			}
		} else {
			return nil, errors.NewConfigurationError("data-merger", "missing or invalid 'sources' in config")
		}
	}

	outputKey, ok := config["output_key"].(string)
	if !ok {
		outputKey = "output"
	}

	// Merge based on strategy
	var result interface{}

	switch strategy {
	case "select_first_available":
		// Return the first non-nil source
		for _, source := range sources {
			sourceKey := fmt.Sprintf("%v", source)
			if value, ok := execCtx.GetVariable(sourceKey); ok && value != nil {
				result = value
				break
			}
		}

	case "merge_all":
		// Merge all sources into a map
		merged := make(map[string]interface{})
		for _, source := range sources {
			sourceKey := fmt.Sprintf("%v", source)
			if value, ok := execCtx.GetVariable(sourceKey); ok {
				merged[sourceKey] = value
			}
		}
		result = merged

	default:
		return nil, errors.NewConfigurationError("data-merger", fmt.Sprintf("unknown strategy '%s'", strategy))
	}

	// Store result
	execCtx.SetVariable(outputKey, result)

	return map[string]interface{}{
		"strategy": strategy,
		"result":   result,
	}, nil
}

// DataAggregatorExecutor executes data aggregator nodes.
// It aggregates data from multiple fields into a structured output.
type DataAggregatorExecutor struct{}

// NewDataAggregatorExecutor creates a new DataAggregatorExecutor.
func NewDataAggregatorExecutor() *DataAggregatorExecutor {
	return &DataAggregatorExecutor{}
}

// Type returns the node type.
func (e *DataAggregatorExecutor) Type() string {
	return "data-aggregator"
}

// Execute executes a data aggregator node.
func (e *DataAggregatorExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// Extract configuration
	// Handle fields as either map[string]interface{} or map[string]string
	var fields map[string]interface{}
	if f, ok := config["fields"].(map[string]interface{}); ok {
		fields = f
	} else if f, ok := config["fields"].(map[string]string); ok {
		// Convert map[string]string to map[string]interface{}
		fields = make(map[string]interface{})
		for k, v := range f {
			fields[k] = v
		}
	} else {
		return nil, errors.NewConfigurationError("data-aggregator", "missing or invalid 'fields' in config")
	}

	outputKey, ok := config["output_key"].(string)
	if !ok {
		outputKey = "output"
	}

	// Aggregate data
	aggregated := make(map[string]interface{})
	for outputField, sourceField := range fields {
		sourceKey := fmt.Sprintf("%v", sourceField)
		if value, ok := execCtx.GetVariable(sourceKey); ok {
			aggregated[outputField] = value
		}
	}

	// Store result
	execCtx.SetVariable(outputKey, aggregated)

	return aggregated, nil
}

// ScriptExecutorExecutor executes script executor nodes.
// Note: This is a placeholder - actual script execution would require a JS engine.
type ScriptExecutorExecutor struct{}

// NewScriptExecutorExecutor creates a new ScriptExecutorExecutor.
func NewScriptExecutorExecutor() *ScriptExecutorExecutor {
	return &ScriptExecutorExecutor{}
}

// Type returns the node type.
func (e *ScriptExecutorExecutor) Type() string {
	return "script-executor"
}

// Execute executes a script executor node.
func (e *ScriptExecutorExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// This is a placeholder implementation
	// In a real implementation, you would use a JavaScript engine like goja or otto

	outputKey, ok := config["output_key"].(string)
	if !ok {
		outputKey = "output"
	}

	// For now, just return a placeholder result
	result := map[string]interface{}{
		"status": "script_execution_not_implemented",
		"note":   "Script execution requires a JavaScript engine",
	}

	execCtx.SetVariable(outputKey, result)

	return result, nil
}

// substituteVariables replaces {{variable}} placeholders with actual values.
func substituteVariables(template string, variables map[string]interface{}) string {
	result := template

	// Find all {{variable}} patterns
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		placeholder := match[0]                // {{variable}}
		varPath := strings.TrimSpace(match[1]) // variable

		// Support nested access like {{customer_info.email}}
		value := getNestedValue(variables, varPath)

		// Replace placeholder with value
		if value != nil {
			valueStr := fmt.Sprintf("%v", value)
			result = strings.ReplaceAll(result, placeholder, valueStr)
		}
	}

	return result
}

// getNestedValue retrieves a value from a nested map using dot notation.
func getNestedValue(data map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")

	var current interface{} = data
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}
