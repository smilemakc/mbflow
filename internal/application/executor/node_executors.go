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

	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"

	"mbflow/internal/domain/errors"
	"mbflow/internal/infrastructure/monitoring"
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
// API key can be provided in node config, execution context, or as default during construction.
type OpenAICompletionExecutor struct {
	// defaultAPIKey is optional; used as fallback if not provided in config or context
	defaultAPIKey string
	// metrics is optional; when set, AI request usage will be recorded
	metrics *monitoring.MetricsCollector
}

// NewOpenAICompletionExecutor creates a new OpenAICompletionExecutor.
// apiKey is optional and used as fallback if not provided in node config or execution context.
func NewOpenAICompletionExecutor(apiKey string) *OpenAICompletionExecutor {
	return &OpenAICompletionExecutor{
		defaultAPIKey: apiKey,
		metrics:       nil,
	}
}

// NewOpenAICompletionExecutorWithMetrics creates a new OpenAICompletionExecutor with metrics collection enabled.
// apiKey is optional and used as fallback if not provided in node config or execution context.
func NewOpenAICompletionExecutorWithMetrics(apiKey string, metrics *monitoring.MetricsCollector) *OpenAICompletionExecutor {
	return &OpenAICompletionExecutor{
		defaultAPIKey: apiKey,
		metrics:       metrics,
	}
}

// Type returns the node type.
func (e *OpenAICompletionExecutor) Type() string {
	return "openai-completion"
}

// Execute executes an OpenAI completion node.
// API key is resolved in the following order:
// 1. From node config["api_key"]
// 2. From execution context variable "openai_api_key" or "OPENAI_API_KEY"
// 3. From default API key provided during construction
func (e *OpenAICompletionExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// Parse configuration
	cfg, err := parseConfig[OpenAICompletionConfig](config)
	if err != nil {
		return nil, errors.NewConfigurationError("openai-completion", fmt.Sprintf("failed to parse config: %v", err))
	}

	// Validate required fields
	if cfg.Prompt == "" {
		return nil, errors.NewConfigurationError("openai-completion", "missing 'prompt' in config")
	}

	// Set defaults
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}
	if cfg.OutputKey == "" {
		cfg.OutputKey = "output"
	}

	// Resolve API key: priority: config > context variable > default
	// Create a temporary map for resolveAPIKey compatibility
	tempConfig := map[string]any{"api_key": cfg.APIKey}
	apiKey, err := e.resolveAPIKey(tempConfig, execCtx)
	if err != nil {
		return nil, err
	}

	// Create OpenAI client with resolved API key
	client := openai.NewClient(apiKey)

	// Get all variables for substitution
	allVars := execCtx.GetAllVariables()

	// Log available variables for debugging (only if verbose logging is enabled)
	log.Debug().
		Str("node_id", nodeID).
		Interface("available_variables", allVars).
		Msg("Substituting variables in prompt")

	// Substitute variables in prompt
	prompt := substituteVariables(cfg.Prompt, allVars)

	// Log the final prompt (truncated if too long)
	promptPreview := prompt
	if len(promptPreview) > 500 {
		promptPreview = promptPreview[:500] + "..."
	}
	log.Debug().
		Str("node_id", nodeID).
		Str("prompt_preview", promptPreview).
		Msg("Final prompt after variable substitution")

	// Create OpenAI request
	req := openai.ChatCompletionRequest{
		Model:               cfg.Model,
		MaxCompletionTokens: cfg.MaxTokens,
		Temperature:         float32(cfg.Temperature),
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	// Call OpenAI API
	startTime := time.Now()
	resp, err := client.CreateChatCompletion(ctx, req)
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

	// Extract response and trim whitespace
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	// Store output in execution context
	execCtx.SetVariable(cfg.OutputKey, content)
	// Record AI request metrics if enabled
	if e.metrics != nil {
		e.metrics.RecordAIRequest(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, latency)
	}

	log.Debug().Str("node_id", nodeID).Msgf("OpenAI completion: %s", content)
	// Return result with metadata
	return map[string]interface{}{
		"content":           content,
		"model":             resp.Model,
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
		"usage":             resp.Usage,
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
	// Parse configuration
	cfg, err := parseConfig[HTTPRequestConfig](config)
	if err != nil {
		return nil, errors.NewConfigurationError("http-request", fmt.Sprintf("failed to parse config: %v", err))
	}

	// Validate required fields
	if cfg.URL == "" {
		return nil, errors.NewConfigurationError("http-request", "missing 'url' in config")
	}

	// Set defaults
	if cfg.Method == "" {
		cfg.Method = "GET"
	}
	if cfg.OutputKey == "" {
		cfg.OutputKey = "output"
	}

	// Substitute variables
	url := substituteVariables(cfg.URL, execCtx.GetAllVariables())

	// Prepare request body
	var body io.Reader
	if cfg.Body != nil {
		var bodyBytes []byte
		var err error

		switch v := cfg.Body.(type) {
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
	req, err := http.NewRequestWithContext(ctx, cfg.Method, url, body)
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
	if cfg.Headers != nil {
		for key, value := range cfg.Headers {
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
		execCtx.SetVariable(cfg.OutputKey, jsonResp)
		return map[string]interface{}{
			"status_code": resp.StatusCode,
			"body":        jsonResp,
			"latency_ms":  latency.Milliseconds(),
		}, nil
	}

	// Store as string if not JSON
	respStr := string(respBody)
	execCtx.SetVariable(cfg.OutputKey, respStr)

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
	// Parse configuration
	cfg, err := parseConfig[ConditionalRouterConfig](config)
	if err != nil {
		return nil, errors.NewConfigurationError("conditional-router", fmt.Sprintf("failed to parse config: %v", err))
	}

	// Validate required fields
	if cfg.InputKey == "" {
		return nil, errors.NewConfigurationError("conditional-router", "missing 'input_key' in config")
	}
	if len(cfg.Routes) == 0 {
		return nil, errors.NewConfigurationError("conditional-router", "missing or invalid 'routes' in config")
	}

	// Get input value
	inputValue, ok := execCtx.GetVariable(cfg.InputKey)
	if !ok {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"conditional-router",
			1,
			fmt.Sprintf("input variable '%s' not found", cfg.InputKey),
			nil,
			false,
		)
	}

	// Convert input to string for comparison
	inputStr := fmt.Sprintf("%v", inputValue)
	// Normalize to lowercase for case-insensitive comparison
	inputStrLower := strings.ToLower(strings.TrimSpace(inputStr))

	// Find matching route (case-insensitive comparison)
	var selectedRoute string
	for condition, route := range cfg.Routes {
		conditionLower := strings.ToLower(strings.TrimSpace(condition))
		if conditionLower == inputStrLower {
			selectedRoute = fmt.Sprintf("%v", route)
			break
		}
	}

	// Check for default route
	if selectedRoute == "" {
		if defaultRoute, ok := cfg.Routes["default"]; ok {
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
	// Parse configuration
	cfg, err := parseConfig[DataMergerConfig](config)
	if err != nil {
		return nil, errors.NewConfigurationError("data-merger", fmt.Sprintf("failed to parse config: %v", err))
	}

	// Validate required fields
	if len(cfg.Sources) == 0 {
		return nil, errors.NewConfigurationError("data-merger", "missing or invalid 'sources' in config")
	}

	// Set defaults
	if cfg.Strategy == "" {
		cfg.Strategy = "select_first_available"
	}
	if cfg.OutputKey == "" {
		cfg.OutputKey = "output"
	}

	// Merge based on strategy
	var result interface{}

	switch cfg.Strategy {
	case "select_first_available":
		// Return the first non-nil source
		for _, sourceKey := range cfg.Sources {
			if value, ok := execCtx.GetVariable(sourceKey); ok && value != nil {
				result = value
				break
			}
		}

	case "merge_all":
		// Merge all sources into a map
		merged := make(map[string]interface{})
		for _, sourceKey := range cfg.Sources {
			if value, ok := execCtx.GetVariable(sourceKey); ok {
				merged[sourceKey] = value
			}
		}
		result = merged

	default:
		return nil, errors.NewConfigurationError("data-merger", fmt.Sprintf("unknown strategy '%s'", cfg.Strategy))
	}

	// Store result
	execCtx.SetVariable(cfg.OutputKey, result)

	return map[string]interface{}{
		"strategy": cfg.Strategy,
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
	// Parse configuration
	cfg, err := parseConfig[DataAggregatorConfig](config)
	if err != nil {
		return nil, errors.NewConfigurationError("data-aggregator", fmt.Sprintf("failed to parse config: %v", err))
	}

	// Validate required fields
	if len(cfg.Fields) == 0 {
		return nil, errors.NewConfigurationError("data-aggregator", "missing or invalid 'fields' in config")
	}

	// Set defaults
	if cfg.OutputKey == "" {
		cfg.OutputKey = "output"
	}

	// Aggregate data
	aggregated := make(map[string]interface{})
	for outputField, sourceKey := range cfg.Fields {
		if value, ok := execCtx.GetVariable(sourceKey); ok {
			aggregated[outputField] = value
		}
	}

	// Store result
	execCtx.SetVariable(cfg.OutputKey, aggregated)

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

	// Parse configuration
	cfg, err := parseConfig[ScriptExecutorConfig](config)
	if err != nil {
		return nil, errors.NewConfigurationError("script-executor", fmt.Sprintf("failed to parse config: %v", err))
	}

	// Set defaults
	if cfg.OutputKey == "" {
		cfg.OutputKey = "output"
	}

	// For now, just return a placeholder result
	result := map[string]interface{}{
		"status": "script_execution_not_implemented",
		"note":   "Script execution requires a JavaScript engine",
	}

	execCtx.SetVariable(cfg.OutputKey, result)

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
			// Check if value is empty string
			if valueStr == "" {
				log.Warn().
					Str("variable", varPath).
					Msgf("Variable {{%s}} is empty, leaving placeholder", varPath)
			} else {
				result = strings.ReplaceAll(result, placeholder, valueStr)
			}
		} else {
			// Log warning if variable is not found
			log.Warn().
				Str("variable", varPath).
				Interface("available_variables", getVariableKeys(variables)).
				Msgf("Variable {{%s}} not found in execution context, leaving placeholder", varPath)
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

// getVariableKeys returns a list of all top-level variable keys for debugging.
func getVariableKeys(variables map[string]interface{}) []string {
	keys := make([]string, 0, len(variables))
	for k := range variables {
		keys = append(keys, k)
	}
	return keys
}

// resolveAPIKey resolves the API key from config, context, or default.
// Priority: 1) node config["api_key"], 2) context variable, 3) default API key.
// Returns an error if the API key cannot be resolved from any source.
func (e *OpenAICompletionExecutor) resolveAPIKey(config map[string]any, execCtx *ExecutionContext) (string, error) {
	// Priority 1: Check node config
	if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
		return apiKey, nil
	}

	// Priority 2: Check execution context variables
	// Try common variable names
	if apiKey, ok := execCtx.GetVariable("openai_api_key"); ok {
		if keyStr, ok := apiKey.(string); ok && keyStr != "" {
			return keyStr, nil
		}
	}
	if apiKey, ok := execCtx.GetVariable("OPENAI_API_KEY"); ok {
		if keyStr, ok := apiKey.(string); ok && keyStr != "" {
			return keyStr, nil
		}
	}

	// Priority 3: Use default API key from constructor
	if e.defaultAPIKey != "" {
		return e.defaultAPIKey, nil
	}

	// No API key found in any source
	return "", errors.NewConfigurationError("openai-completion", "API key not found in node config, execution context, or default configuration")
}
