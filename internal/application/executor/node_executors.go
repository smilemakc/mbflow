package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/sashabaranov/go-openai"
	"github.com/smilemakc/mbflow/internal/domain"
)

// parseNodeConfig parses node config map into a typed struct
func parseNodeConfig[T any](config map[string]any) (*T, error) {
	// Marshal config to JSON
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Unmarshal into typed struct
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &result, nil
}

// StartNodeExecutor executes start nodes (entry points)
type StartNodeExecutor struct{}

func NewStartNodeExecutor() *StartNodeExecutor {
	return &StartNodeExecutor{}
}

func (e *StartNodeExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Start nodes just pass through their input
	return make(map[string]any), nil
}

// EndNodeExecutor executes end nodes (exit points)
type EndNodeExecutor struct{}

func NewEndNodeExecutor() *EndNodeExecutor {
	return &EndNodeExecutor{}
}

func (e *EndNodeExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// End nodes collect final output from variables
	output := make(map[string]any)
	allVars := inputs.Variables.All()
	// Get output keys from config
	if outputKeys, ok := node.Config()["output_keys"].([]interface{}); ok {
		for _, key := range outputKeys {
			keyStr, ok := key.(string)
			if !ok {
				continue
			}
			if value := getNestedValue(allVars, keyStr); value != nil {
				output[keyStr] = value
			}
		}
	} else {
		// If no output keys specified, return all variables
		output = inputs.Variables.All()
	}

	return output, nil
}

// TransformNodeExecutor executes transform nodes using expressions
type TransformNodeExecutor struct {
	evaluator *ConditionEvaluator
}

func NewTransformNodeExecutor() *TransformNodeExecutor {
	return &TransformNodeExecutor{
		evaluator: NewConditionEvaluator(true),
	}
}

func (e *TransformNodeExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Parse config into structured type
	cfg, err := parseNodeConfig[TransformConfig](node.Config())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if len(cfg.Transformations) == 0 {
		return nil, fmt.Errorf("transformations not specified")
	}

	// Merge scoped + global for expression evaluation
	mergedVars := inputs.Variables.Clone()
	_ = mergedVars.Merge(inputs.GlobalContext)
	vars := mergedVars.All()

	output := make(map[string]any)

	// Sort keys to ensure deterministic execution order
	// This allows transformations to reference earlier transformations
	keys := make([]string, 0, len(cfg.Transformations))
	for k := range cfg.Transformations {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, outputKey := range keys {
		exprStr := cfg.Transformations[outputKey]

		// Compile with current vars (includes previous transformation results)
		program, err := expr.Compile(exprStr, expr.Env(vars), expr.AllowUndefinedVariables())
		if err != nil {
			return nil, fmt.Errorf("failed to compile expression for %s: %w", outputKey, err)
		}

		// Execute with current vars
		result, err := expr.Run(program, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to execute expression for %s: %w", outputKey, err)
		}

		output[outputKey] = result
		// Add the result to vars so subsequent transformations can use it
		vars[outputKey] = result
	}

	return output, nil
}

// HTTPNodeExecutor executes HTTP request nodes
type HTTPNodeExecutor struct {
	client *http.Client
}

func NewHTTPNodeExecutor() *HTTPNodeExecutor {
	return &HTTPNodeExecutor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (e *HTTPNodeExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	config := node.Config()

	// Get URL
	url, ok := config["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url not specified in config")
	}

	// URL already processed by template processor in engine

	// Get method (default GET)
	method := "GET"
	if m, ok := config["method"].(string); ok {
		method = m
	}

	// Prepare request body
	var body io.Reader
	if bodyData, ok := config["body"]; ok {
		bodyJSON, err := json.Marshal(bodyData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(bodyJSON)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (already processed by template processor in engine)
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if valueStr, ok := value.(string); ok {
				req.Header.Set(key, valueStr)
			}
		}
	}

	// Execute request
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If not JSON, return as string
		result = map[string]any{
			"body":        string(respBody),
			"status_code": resp.StatusCode,
			"headers":     resp.Header,
		}
	} else {
		result["status_code"] = resp.StatusCode
		result["headers"] = resp.Header
	}

	// Check if response is successful
	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	return result, nil
}

// JSONParserExecutor parses JSON data
type JSONParserExecutor struct{}

func NewJSONParserExecutor() *JSONParserExecutor {
	return &JSONParserExecutor{}
}

func (e *JSONParserExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	config := node.Config()

	// Get input key
	inputKey, ok := config["input_key"].(string)
	if !ok {
		return nil, fmt.Errorf("input_key not specified")
	}

	// Get input data
	inputValue, exists := inputs.Variables.Get(inputKey)
	if !exists {
		return nil, fmt.Errorf("input key %s not found in variables", inputKey)
	}

	var data map[string]any

	// Parse based on input type
	switch v := inputValue.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON string: %w", err)
		}

	case []byte:
		if err := json.Unmarshal(v, &data); err != nil {
			return nil, fmt.Errorf("failed to parse JSON bytes: %w", err)
		}

	case map[string]any:
		data = v

	default:
		return nil, fmt.Errorf("unsupported input type: %T", inputValue)
	}

	return data, nil
}

// DataMergerExecutor merges data from multiple sources
type DataMergerExecutor struct{}

func NewDataMergerExecutor() *DataMergerExecutor {
	return &DataMergerExecutor{}
}

func (e *DataMergerExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Parse config into structured type
	cfg, err := parseNodeConfig[DataMergerConfig](node.Config())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if len(cfg.Sources) == 0 {
		return nil, fmt.Errorf("sources not specified")
	}

	// Set default merge strategy
	strategy := cfg.Strategy
	if strategy == "" {
		strategy = "overwrite"
	}

	output := make(map[string]any)

	// Merge data from all sources
	for _, keyStr := range cfg.Sources {
		value, exists := inputs.Variables.Get(keyStr)
		if !exists {
			continue
		}

		if dataMap, ok := value.(map[string]any); ok {
			for k, v := range dataMap {
				switch strategy {
				case "overwrite":
					output[k] = v

				case "keep_first":
					if _, exists := output[k]; !exists {
						output[k] = v
					}

				case "collect":
					if existing, exists := output[k]; exists {
						// Convert to array
						if arr, ok := existing.([]any); ok {
							output[k] = append(arr, v)
						} else {
							output[k] = []any{existing, v}
						}
					} else {
						output[k] = v
					}
				}
			}
		}
	}

	return output, nil
}

// DataAggregatorExecutor aggregates data from multiple fields or arrays
type DataAggregatorExecutor struct{}

func NewDataAggregatorExecutor() *DataAggregatorExecutor {
	return &DataAggregatorExecutor{}
}

func (e *DataAggregatorExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Parse config into structured type
	cfg, err := parseNodeConfig[DataAggregatorConfig](node.Config())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Check if we have fields configuration (field extraction mode)
	if len(cfg.Fields) > 0 {
		return e.executeFieldExtraction(cfg.Fields, inputs)
	}

	// Otherwise, use array aggregation mode
	if cfg.InputKey == "" {
		return nil, fmt.Errorf("either fields or input_key must be specified")
	}
	return e.executeArrayAggregation(cfg, inputs)
}

// executeFieldExtraction extracts fields from variables based on field mapping
func (e *DataAggregatorExecutor) executeFieldExtraction(
	fields map[string]string,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	aggregated := make(map[string]any)

	for outputField, sourceKey := range fields {
		// Support nested field access like "parsed_data.name"
		value := getNestedValue(inputs.Variables.All(), sourceKey)
		if value != nil {
			aggregated[outputField] = value
		}
	}

	return aggregated, nil
}

// executeArrayAggregation performs aggregation functions on array data
func (e *DataAggregatorExecutor) executeArrayAggregation(
	cfg *DataAggregatorConfig,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Get input data
	inputValue, exists := inputs.Variables.Get(cfg.InputKey)
	if !exists {
		return nil, fmt.Errorf("input key %s not found", cfg.InputKey)
	}

	// Convert to array
	var items []any
	switch v := inputValue.(type) {
	case []any:
		items = v
	default:
		return nil, fmt.Errorf("input is not an array: %T", inputValue)
	}

	// Set default aggregation function
	aggFunc := cfg.Function
	if aggFunc == "" {
		aggFunc = "sum"
	}

	output := make(map[string]any)

	switch aggFunc {
	case "sum":
		sum := 0.0
		for _, item := range items {
			if num, ok := item.(float64); ok {
				sum += num
			}
		}
		output["result"] = sum

	case "count":
		output["result"] = len(items)

	case "avg", "average":
		sum := 0.0
		count := 0
		for _, item := range items {
			if num, ok := item.(float64); ok {
				sum += num
				count++
			}
		}
		if count > 0 {
			output["result"] = sum / float64(count)
		} else {
			output["result"] = 0.0
		}

	case "min":
		if len(items) == 0 {
			output["result"] = nil
		} else {
			m := float64(0)
			first := true
			for _, item := range items {
				if num, ok := item.(float64); ok {
					if first || num < m {
						m = num
						first = false
					}
				}
			}
			output["result"] = m
		}

	case "max":
		if len(items) == 0 {
			output["result"] = nil
		} else {
			m := float64(0)
			first := true
			for _, item := range items {
				if num, ok := item.(float64); ok {
					if first || num > m {
						m = num
						first = false
					}
				}
			}
			output["result"] = m
		}

	case "collect":
		output["result"] = items

	default:
		return nil, fmt.Errorf("unknown aggregation function: %s", aggFunc)
	}

	output["count"] = len(items)

	return output, nil
}

// ConditionalRouteExecutor routes execution based on conditions
type ConditionalRouteExecutor struct {
	evaluator *ConditionEvaluator
}

func NewConditionalRouteExecutor() *ConditionalRouteExecutor {
	return &ConditionalRouteExecutor{
		evaluator: NewConditionEvaluator(true),
	}
}

func (e *ConditionalRouteExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	config := node.Config()

	// Get routes - handle both []interface{} and []map[string]any
	var routes []map[string]any
	if routesInterface, ok := config["routes"].([]interface{}); ok {
		for _, r := range routesInterface {
			if routeMap, ok := r.(map[string]interface{}); ok {
				// Convert to map[string]any
				converted := make(map[string]any)
				for k, v := range routeMap {
					converted[k] = v
				}
				routes = append(routes, converted)
			}
		}
	} else if routesMaps, ok := config["routes"].([]map[string]any); ok {
		routes = routesMaps
	} else {
		return nil, fmt.Errorf("routes not specified or invalid type")
	}

	if len(routes) == 0 {
		return nil, fmt.Errorf("routes is empty")
	}

	vars := inputs.Variables.All()
	selectedRoute := ""

	// Evaluate routes in order
	for i, routeMap := range routes {
		// Get condition
		condition, ok := routeMap["condition"].(string)
		if !ok {
			continue
		}

		// Evaluate condition
		result, err := e.evaluator.Evaluate(condition, vars)
		if err != nil {
			continue
		}

		if result {
			if name, ok := routeMap["name"].(string); ok {
				selectedRoute = name
			} else {
				selectedRoute = fmt.Sprintf("route_%d", i)
			}
			break
		}
	}

	// If no route matched, use default
	if selectedRoute == "" {
		if defaultRoute, ok := config["default_route"].(string); ok {
			selectedRoute = defaultRoute
		} else {
			selectedRoute = "default"
		}
	}

	return map[string]any{
		"selected_route": selectedRoute,
	}, nil
}

// ScriptExecutorNode executes scripts (simplified, would need sandboxing in production)
type ScriptExecutorNode struct{}

func NewScriptExecutorNode() *ScriptExecutorNode {
	return &ScriptExecutorNode{}
}

func (e *ScriptExecutorNode) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	config := node.Config()

	// Get script
	script, ok := config["script"].(string)
	if !ok {
		return nil, fmt.Errorf("script not specified")
	}

	// Get language
	language := "expr"
	if lang, ok := config["language"].(string); ok {
		language = lang
	}

	switch language {
	case "expr":
		// Execute using expr
		vars := inputs.Variables.All()
		program, err := expr.Compile(script, expr.Env(vars), expr.AsAny())
		if err != nil {
			return nil, fmt.Errorf("failed to compile script: %w", err)
		}

		result, err := expr.Run(program, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to execute script: %w", err)
		}

		return map[string]any{
			"result": result,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported script language: %s", language)
	}
}

// ParallelNodeExecutor is handled by the execution engine via graph structure
// This is just a placeholder
type ParallelNodeExecutor struct{}

func NewParallelNodeExecutor() *ParallelNodeExecutor {
	return &ParallelNodeExecutor{}
}

func (e *ParallelNodeExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Parallel execution is handled by the engine
	// This node just passes through
	return make(map[string]any), nil
}

// OpenAICompletionExecutor executes OpenAI completion nodes
type OpenAICompletionExecutor struct {
	defaultAPIKey string
}

// AIMetricsRecorder defines interface for recording AI API metrics
type AIMetricsRecorder interface {
	RecordAIRequest(promptTokens, completionTokens int, latency time.Duration)
}

func NewOpenAICompletionExecutor(apiKey string) *OpenAICompletionExecutor {
	return &OpenAICompletionExecutor{
		defaultAPIKey: apiKey,
	}
}

func (e *OpenAICompletionExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Parse config into structured type
	cfg, err := parseNodeConfig[OpenAICompletionConfig](node.Config())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if cfg.Prompt == "" {
		return nil, fmt.Errorf("prompt not specified in config")
	}

	// Set defaults
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}

	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}

	// Resolve API key (config.APIKey takes precedence, then variables, then default)
	apiKey, err := resolveAPIKey(node.Config(), inputs.GlobalContext, e.defaultAPIKey)
	if err != nil {
		return nil, err
	}

	// Get prompt (already processed by template processor in engine)
	prompt := cfg.Prompt

	// Create OpenAI client
	client := openai.NewClient(apiKey)

	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}
	var messages []openai.ChatCompletionMessage
	messages = append(messages, cfg.History...)
	messages = append(messages, message)
	// Create OpenAI request
	req := openai.ChatCompletionRequest{
		Model:       cfg.Model,
		Temperature: float32(cfg.Temperature),
		Messages:    messages,
	}

	// Set max tokens if specified
	if cfg.MaxTokens > 0 {
		req.MaxCompletionTokens = cfg.MaxTokens
	}

	// Handle tools/function calling if configured
	if len(cfg.Tools) > 0 {
		req.Tools = e.convertTools(cfg.Tools)

		// Set tool_choice if specified
		if cfg.ToolChoice != nil {
			req.ToolChoice = cfg.ToolChoice
		}
	}

	// Structured output
	// Set response format if configured
	if cfg.ResponseFormat != nil {
		// Convert response_format to the expected format
		formatBytes, err := json.Marshal(cfg.ResponseFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response_format: %w", err)
		}

		var openaiResponseFormat openai.ChatCompletionResponseFormat
		if err := json.Unmarshal(formatBytes, &openaiResponseFormat); err != nil {
			return nil, fmt.Errorf("failed to parse response_format: %w", err)
		}

		req.ResponseFormat = &openaiResponseFormat
	}
	// Call OpenAI API
	startTime := time.Now()
	resp, err := client.CreateChatCompletion(ctx, req)
	latency := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	choice := resp.Choices[0]

	var content any
	// Extract response and trim whitespace
	if cfg.ResponseFormat != nil {
		var jsonContent map[string]any
		if err := json.Unmarshal([]byte(choice.Message.Content), &jsonContent); err != nil {
			return nil, fmt.Errorf("failed to parse structured response: %w", err)
		}
		content = jsonContent
	} else {
		content = strings.TrimSpace(choice.Message.Content)
	}

	// Return result with metadata
	result := map[string]any{
		"content":            content,
		"model":              resp.Model,
		"response_id":        resp.ID,
		"prompt_tokens":      resp.Usage.PromptTokens,
		"completion_tokens":  resp.Usage.CompletionTokens,
		"total_tokens":       resp.Usage.TotalTokens,
		"latency_ms":         latency.Milliseconds(),
		"finish_reason":      string(choice.FinishReason),
		"usage":              resp.Usage,
		"completion_request": req,
	}

	// Include tool_calls if present
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]map[string]any, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			toolCalls = append(toolCalls, map[string]any{
				"id":   tc.ID,
				"type": tc.Type,
				"function": map[string]any{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			})
		}
		result["tool_calls"] = toolCalls
	}

	return result, nil
}

// convertTools converts OpenAITool configs to OpenAI SDK format
func (e *OpenAICompletionExecutor) convertTools(tools []OpenAITool) []openai.Tool {
	openaiTools := make([]openai.Tool, 0, len(tools))
	for _, t := range tools {
		openaiTools = append(openaiTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}
	return openaiTools
}

// OpenAIResponsesExecutor executes OpenAI Responses API nodes with structured outputs
type OpenAIResponsesExecutor struct {
	defaultAPIKey string
}

func NewOpenAIResponsesExecutor(apiKey string) *OpenAIResponsesExecutor {
	return &OpenAIResponsesExecutor{
		defaultAPIKey: apiKey,
	}
}

func (e *OpenAIResponsesExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	config := node.Config()

	// Get prompt
	prompt, ok := config["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("prompt not specified in config")
	}

	// Get model (default: gpt-4o)
	model := "gpt-4o"
	if m, ok := config["model"].(string); ok && m != "" {
		model = m
	}

	// Get optional parameters
	maxTokens := 0
	if mt, ok := config["max_tokens"].(float64); ok {
		maxTokens = int(mt)
	} else if mt, ok := config["max_tokens"].(int); ok {
		maxTokens = mt
	}

	temperature := 0.7
	if temp, ok := config["temperature"].(float64); ok {
		temperature = temp
	}

	topP := 0.0
	if tp, ok := config["top_p"].(float64); ok {
		topP = tp
	}

	frequencyPenalty := 0.0
	if fp, ok := config["frequency_penalty"].(float64); ok {
		frequencyPenalty = fp
	}

	presencePenalty := 0.0
	if pp, ok := config["presence_penalty"].(float64); ok {
		presencePenalty = pp
	}

	// Get stop sequences
	var stopSequences []string
	if stop, ok := config["stop"].([]interface{}); ok {
		for _, s := range stop {
			if str, ok := s.(string); ok {
				stopSequences = append(stopSequences, str)
			}
		}
	}

	// Get response format (for structured outputs)
	var responseFormat map[string]any
	if rf, ok := config["response_format"].(map[string]interface{}); ok {
		responseFormat = rf
	}

	// Resolve API key
	apiKey, err := resolveAPIKey(config, inputs.GlobalContext, e.defaultAPIKey)
	if err != nil {
		return nil, err
	}

	// Prompt already processed by template processor in engine

	// Create OpenAI client
	client := openai.NewClient(apiKey)

	// Create OpenAI request
	req := openai.ChatCompletionRequest{
		Model:       model,
		Temperature: float32(temperature),
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	// Set max tokens if specified
	if maxTokens > 0 {
		req.MaxCompletionTokens = maxTokens
	}

	// Set optional parameters
	if topP > 0 {
		req.TopP = float32(topP)
	}
	if frequencyPenalty != 0 {
		req.FrequencyPenalty = float32(frequencyPenalty)
	}
	if presencePenalty != 0 {
		req.PresencePenalty = float32(presencePenalty)
	}
	if len(stopSequences) > 0 {
		req.Stop = stopSequences
	}

	// Set response format if configured
	if responseFormat != nil {
		// Convert response_format to the expected format
		formatBytes, err := json.Marshal(responseFormat)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response_format: %w", err)
		}

		var openaiResponseFormat openai.ChatCompletionResponseFormat
		if err := json.Unmarshal(formatBytes, &openaiResponseFormat); err != nil {
			return nil, fmt.Errorf("failed to parse response_format: %w", err)
		}

		req.ResponseFormat = &openaiResponseFormat
	}

	// Call OpenAI API
	startTime := time.Now()
	resp, err := client.CreateChatCompletion(ctx, req)
	latency := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	// Extract response content
	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Try to parse as JSON if response_format was set
	var outputValue interface{} = content
	if responseFormat != nil {
		var jsonContent interface{}
		if err := json.Unmarshal([]byte(content), &jsonContent); err == nil {
			outputValue = jsonContent
		}
	}

	// Return result with metadata
	result := map[string]any{
		"content":           content,
		"parsed_content":    outputValue,
		"model":             resp.Model,
		"response_id":       resp.ID,
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
		"latency_ms":        latency.Milliseconds(),
	}

	return result, nil
}

func resolveAPIKey(config map[string]any, globalContext *domain.VariableSet, defaultAPIKey string) (string, error) {
	// Priority 1: Check node config
	if apiKey, ok := config["api_key"].(string); ok && apiKey != "" {
		return apiKey, nil
	}

	// Priority 2: Check global context variables
	if apiKey, exists := globalContext.Get("openai_api_key"); exists {
		if keyStr, ok := apiKey.(string); ok && keyStr != "" {
			return keyStr, nil
		}
	}
	if apiKey, exists := globalContext.Get("OPENAI_API_KEY"); exists {
		if keyStr, ok := apiKey.(string); ok && keyStr != "" {
			return keyStr, nil
		}
	}

	// Priority 3: Use default API key from constructor
	if defaultAPIKey != "" {
		return defaultAPIKey, nil
	}

	return "", fmt.Errorf("API key not found in node config, execution context, or default configuration")
}

// Helper functions

// getNestedValue retrieves a value from a nested map using dot notation
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

// FunctionCallExecutor executes function calls from OpenAI tool_calls
type FunctionCallExecutor struct{}

func NewFunctionCallExecutor() *FunctionCallExecutor {
	return &FunctionCallExecutor{}
}

func (e *FunctionCallExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Parse config into structured type
	cfg, err := parseNodeConfig[FunctionCallConfig](node.Config())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if cfg.InputKey == "" {
		return nil, fmt.Errorf("input_key not specified")
	}
	if cfg.Handler == "" {
		return nil, fmt.Errorf("handler not specified")
	}

	// Get the AI response
	inputValue, exists := inputs.Variables.Get(cfg.InputKey)
	if !exists {
		return nil, fmt.Errorf("input key %s not found", cfg.InputKey)
	}

	// Extract tool_calls from the response
	responseMap, ok := inputValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("input is not a map: %T", inputValue)
	}

	toolCallsInterface, ok := responseMap["tool_calls"]
	if !ok {
		return nil, fmt.Errorf("no tool_calls found in response")
	}

	toolCalls, ok := toolCallsInterface.([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("tool_calls is not a list: %T", toolCallsInterface)
	}

	if len(toolCalls) == 0 {
		return nil, fmt.Errorf("tool_calls list is empty")
	}

	// Find the target function call
	var targetCall map[string]any
	for _, tc := range toolCalls {
		funcData, ok := tc["function"].(map[string]any)
		if !ok {
			continue
		}

		name, _ := funcData["name"].(string)
		if cfg.FunctionName == "" || name == cfg.FunctionName {
			targetCall = tc
			break
		}
	}

	if targetCall == nil {
		if cfg.FunctionName != "" {
			return nil, fmt.Errorf("function %s not found in tool_calls", cfg.FunctionName)
		}
		return nil, fmt.Errorf("no function calls found")
	}

	// Extract function details
	funcData := targetCall["function"].(map[string]any)
	name, _ := funcData["name"].(string)
	arguments, _ := funcData["arguments"].(string)

	// Parse arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse function arguments: %w", err)
	}

	var result map[string]any

	switch cfg.Handler {
	case "builtin":
		// Execute built-in function handler
		result = e.executeBuiltin(name, args)

	case "script":
		// Execute via script (expr-lang)
		scriptTemplate, _ := cfg.HandlerConfig["script"].(string)
		if scriptTemplate == "" {
			return nil, fmt.Errorf("script not specified in handler_config")
		}

		// Merge scoped + global + function arguments
		mergedVars := inputs.Variables.Clone()
		_ = mergedVars.Merge(inputs.GlobalContext)

		// Make function arguments available
		for k, v := range args {
			_ = mergedVars.Set(k, v)
		}

		// Get all variables including function arguments
		vars := mergedVars.All()

		// Execute script
		program, err := expr.Compile(scriptTemplate, expr.Env(vars), expr.AsAny())
		if err != nil {
			return nil, fmt.Errorf("failed to compile script: %w", err)
		}

		scriptResult, err := expr.Run(program, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to execute script: %w", err)
		}

		result = map[string]any{
			"result": scriptResult,
		}

	case "http":
		// Execute via HTTP call
		return nil, fmt.Errorf("http handler not yet implemented")

	default:
		return nil, fmt.Errorf("unknown handler type: %s", cfg.Handler)
	}

	// Add metadata
	result["function_name"] = name
	result["arguments"] = args

	return result, nil
}

// executeBuiltin executes built-in function handlers
func (e *FunctionCallExecutor) executeBuiltin(name string, args map[string]interface{}) map[string]any {
	// This is a placeholder for built-in functions
	// Users can extend this by registering custom handlers
	return map[string]any{
		"status": "not_implemented",
		"error":  fmt.Sprintf("builtin function %s not implemented", name),
	}
}

// OpenAIFunctionResultExecutor continues the conversation after function execution
type OpenAIFunctionResultExecutor struct {
	defaultAPIKey string
	metrics       AIMetricsRecorder
}

func NewOpenAIFunctionResultExecutor(apiKey string) *OpenAIFunctionResultExecutor {
	return &OpenAIFunctionResultExecutor{
		defaultAPIKey: apiKey,
		metrics:       nil,
	}
}

func NewOpenAIFunctionResultExecutorWithMetrics(apiKey string, metrics AIMetricsRecorder) *OpenAIFunctionResultExecutor {
	return &OpenAIFunctionResultExecutor{
		defaultAPIKey: apiKey,
		metrics:       metrics,
	}
}

func (e *OpenAIFunctionResultExecutor) Execute(
	ctx context.Context,
	node domain.Node,
	inputs *NodeExecutionInputs,
) (map[string]any, error) {
	// Parse config into structured type
	cfg, err := parseNodeConfig[OpenAIFunctionResponseConfig](node.Config())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate required fields
	if cfg.AIResponseKey == "" {
		return nil, fmt.Errorf("ai_response_key not specified")
	}
	if cfg.FunctionResultKey == "" {
		return nil, fmt.Errorf("function_result_key not specified")
	}

	// Set defaults
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}

	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}

	// Resolve API key
	apiKey, err := resolveAPIKey(node.Config(), inputs.GlobalContext, e.defaultAPIKey)
	if err != nil {
		return nil, err
	}

	// Get the original AI response
	aiResponseValue, exists := inputs.Variables.Get(cfg.AIResponseKey)
	if !exists {
		return nil, fmt.Errorf("ai_response_key %s not found", cfg.AIResponseKey)
	}

	aiResponse, ok := aiResponseValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("ai_response is not a map: %T", aiResponseValue)
	}

	// Get the function result
	functionResultValue, exists := inputs.Variables.Get(cfg.FunctionResultKey)
	if !exists {
		return nil, fmt.Errorf("function_result_key %s not found", cfg.FunctionResultKey)
	}

	functionResult, ok := functionResultValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("function_result is not a map: %T", functionResultValue)
	}

	// Extract the original request data
	completionRequest, ok := aiResponse["completion_request"].(openai.ChatCompletionRequest)
	if !ok {
		return nil, fmt.Errorf("completion_request not found in ai_response")
	}

	// Extract tool_calls from AI response
	toolCallsInterface, ok := aiResponse["tool_calls"]
	if !ok {
		return nil, fmt.Errorf("no tool_calls found in ai_response")
	}

	toolCallsSlice, ok := toolCallsInterface.([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("tool_calls is not a list: %T", toolCallsInterface)
	}

	if len(toolCallsSlice) == 0 {
		return nil, fmt.Errorf("tool_calls list is empty")
	}

	// Get the first tool call (we'll use it to continue the conversation)
	toolCall := toolCallsSlice[0]
	toolCallID, _ := toolCall["id"].(string)
	functionData, _ := toolCall["function"].(map[string]any)
	functionName, _ := functionData["name"].(string)

	// Serialize function result to JSON
	resultJSON, err := json.Marshal(functionResult["result"])
	if err != nil {
		return nil, fmt.Errorf("failed to serialize function result: %w", err)
	}

	// Build messages history for the continuation
	messages := completionRequest.Messages

	// Add the assistant's message with tool_calls
	messages = append(messages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleAssistant,
		ToolCalls: []openai.ToolCall{
			{
				ID:   toolCallID,
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      functionName,
					Arguments: functionData["arguments"].(string),
				},
			},
		},
	})

	// Add the function result as a tool message
	messages = append(messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    string(resultJSON),
		ToolCallID: toolCallID,
	})

	// Create OpenAI client
	client := openai.NewClient(apiKey)

	// Create the continuation request
	req := openai.ChatCompletionRequest{
		Model:       cfg.Model,
		Temperature: float32(cfg.Temperature),
		Messages:    messages,
	}

	if cfg.MaxTokens > 0 {
		req.MaxCompletionTokens = cfg.MaxTokens
	}

	// Call OpenAI API
	startTime := time.Now()
	resp, err := client.CreateChatCompletion(ctx, req)
	latency := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	choice := resp.Choices[0]
	content := strings.TrimSpace(choice.Message.Content)

	// Record metrics if enabled
	if e.metrics != nil {
		e.metrics.RecordAIRequest(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, latency)
	}

	// Return result with metadata
	result := map[string]any{
		"content":           content,
		"model":             resp.Model,
		"response_id":       resp.ID,
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
		"latency_ms":        latency.Milliseconds(),
		"finish_reason":     string(choice.FinishReason),
		"messages_count":    len(messages),
	}

	return result, nil
}

// RegisterDefaultExecutors registers all default node executors with the engine
func RegisterDefaultExecutors(engine *WorkflowEngine) {
	engine.RegisterNodeExecutor(domain.NodeTypeStart, NewStartNodeExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeEnd, NewEndNodeExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeTransform, NewTransformNodeExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeHTTP, NewHTTPNodeExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeHTTPRequest, NewHTTPNodeExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeJSONParser, NewJSONParserExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeDataMerger, NewDataMergerExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeDataAggregator, NewDataAggregatorExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeConditionalRoute, NewConditionalRouteExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeScriptExecutor, NewScriptExecutorNode())
	engine.RegisterNodeExecutor(domain.NodeTypeParallel, NewParallelNodeExecutor())

	// Register OpenAI executors
	engine.RegisterNodeExecutor(domain.NodeTypeOpenAICompletion, NewOpenAICompletionExecutor(""))
	engine.RegisterNodeExecutor(domain.NodeTypeOpenAIResponses, NewOpenAIResponsesExecutor(""))

	// Register function call executors
	engine.RegisterNodeExecutor(domain.NodeTypeFunctionCall, NewFunctionCallExecutor())
	engine.RegisterNodeExecutor(domain.NodeTypeOpenAIFunctionResult, NewOpenAIFunctionResultExecutor(""))
}
