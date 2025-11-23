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

	"github.com/smilemakc/mbflow/internal/domain/errors"
	"github.com/smilemakc/mbflow/internal/infrastructure/monitoring"
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
	return NodeTypeOpenAICompletion
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
		"response_id":       resp.ID,
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
		"usage":             resp.Usage,
		"latency_ms":        latency.Milliseconds(),
	}, nil
}

// OpenAIResponsesExecutor executes OpenAI Responses API nodes.
// It sends requests to the OpenAI API with support for structured outputs.
// API key can be provided in node config, execution context, or as default during construction.
type OpenAIResponsesExecutor struct {
	// defaultAPIKey is optional; used as fallback if not provided in config or context
	defaultAPIKey string
	// metrics is optional; when set, AI request usage will be recorded
	metrics *monitoring.MetricsCollector
}

// NewOpenAIResponsesExecutor creates a new OpenAIResponsesExecutor.
// apiKey is optional and used as fallback if not provided in node config or execution context.
func NewOpenAIResponsesExecutor(apiKey string) *OpenAIResponsesExecutor {
	return &OpenAIResponsesExecutor{
		defaultAPIKey: apiKey,
		metrics:       nil,
	}
}

// NewOpenAIResponsesExecutorWithMetrics creates a new OpenAIResponsesExecutor with metrics collection enabled.
// apiKey is optional and used as fallback if not provided in node config or execution context.
func NewOpenAIResponsesExecutorWithMetrics(apiKey string, metrics *monitoring.MetricsCollector) *OpenAIResponsesExecutor {
	return &OpenAIResponsesExecutor{
		defaultAPIKey: apiKey,
		metrics:       metrics,
	}
}

// Type returns the node type.
func (e *OpenAIResponsesExecutor) Type() string {
	return NodeTypeOpenAIResponses
}

// Execute executes an OpenAI Responses API node.
// API key is resolved in the following order:
// 1. From node config["api_key"]
// 2. From execution context variable "openai_api_key" or "OPENAI_API_KEY"
// 3. From default API key provided during construction
func (e *OpenAIResponsesExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// Parse configuration
	cfg, err := parseConfig[OpenAIResponsesConfig](config)
	if err != nil {
		return nil, errors.NewConfigurationError("openai-responses", fmt.Sprintf("failed to parse config: %v", err))
	}

	// Validate required fields
	if cfg.Prompt == "" {
		return nil, errors.NewConfigurationError("openai-responses", "missing 'prompt' in config")
	}

	// Set defaults
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}
	if cfg.OutputKey == "" {
		cfg.OutputKey = "output"
	}

	// Resolve API key: priority: config > context variable > default
	tempConfig := map[string]any{"api_key": cfg.APIKey}
	apiKey, err := e.resolveAPIKey(tempConfig, execCtx)
	if err != nil {
		return nil, err
	}

	// Create OpenAI client with resolved API key
	client := openai.NewClient(apiKey)

	// Get all variables for substitution
	allVars := execCtx.GetAllVariables()

	// Log available variables for debugging
	log.Debug().
		Str("node_id", nodeID).
		Interface("available_variables", allVars).
		Msg("Substituting variables in prompt")

	// Substitute variables in prompt
	prompt := substituteVariables(cfg.Prompt, allVars)

	// Log the final prompt
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

	// Set optional parameters
	if cfg.TopP > 0 {
		req.TopP = float32(cfg.TopP)
	}
	if cfg.FrequencyPenalty != 0 {
		req.FrequencyPenalty = float32(cfg.FrequencyPenalty)
	}
	if cfg.PresencePenalty != 0 {
		req.PresencePenalty = float32(cfg.PresencePenalty)
	}
	if len(cfg.Stop) > 0 {
		req.Stop = cfg.Stop
	}

	// Set response format if configured
	if cfg.ResponseFormat != nil {
		// Convert response_format to the expected format
		formatBytes, err := json.Marshal(cfg.ResponseFormat)
		if err != nil {
			return nil, errors.NewConfigurationError("openai-responses", fmt.Sprintf("failed to marshal response_format: %v", err))
		}

		var responseFormat openai.ChatCompletionResponseFormat
		if err := json.Unmarshal(formatBytes, &responseFormat); err != nil {
			return nil, errors.NewConfigurationError("openai-responses", fmt.Sprintf("failed to parse response_format: %v", err))
		}

		req.ResponseFormat = &responseFormat
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
			"openai-responses",
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
			"openai-responses",
			1,
			"OpenAI returned no choices",
			nil,
			false,
		)
	}

	// Extract response content
	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Try to parse as JSON if response_format was set
	var outputValue interface{} = content
	if cfg.ResponseFormat != nil {
		var jsonContent interface{}
		if err := json.Unmarshal([]byte(content), &jsonContent); err == nil {
			outputValue = jsonContent
		}
	}

	// Store output in execution context
	execCtx.SetVariable(cfg.OutputKey, outputValue)

	// Record AI request metrics if enabled
	if e.metrics != nil {
		e.metrics.RecordAIRequest(resp.Usage.PromptTokens, resp.Usage.CompletionTokens, latency)
	}

	log.Debug().Str("node_id", nodeID).Msgf("OpenAI responses completion: %s", content)

	// Return result with metadata
	return map[string]interface{}{
		"content":           content,
		"parsed_content":    outputValue,
		"model":             resp.Model,
		"response_id":       resp.ID,
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
		"usage":             resp.Usage,
		"latency_ms":        latency.Milliseconds(),
	}, nil
}

// resolveAPIKey resolves the API key from config, context, or default.
// Priority: 1) node config["api_key"], 2) context variable, 3) default API key.
// Returns an error if the API key cannot be resolved from any source.
func (e *OpenAIResponsesExecutor) resolveAPIKey(config map[string]any, execCtx *ExecutionContext) (string, error) {
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
	return "", errors.NewConfigurationError("openai-responses", "API key not found in node config, execution context, or default configuration")
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
	return NodeTypeHTTPRequest
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

// TelegramMessageExecutor executes Telegram message nodes using the Telegram Bot API.
type TelegramMessageExecutor struct {
	client *http.Client
}

// NewTelegramMessageExecutor creates a new TelegramMessageExecutor.
func NewTelegramMessageExecutor() *TelegramMessageExecutor {
	return &TelegramMessageExecutor{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Type returns the node type.
func (e *TelegramMessageExecutor) Type() string {
	return NodeTypeTelegramMessage
}

// Execute sends a message via the Telegram Bot API.
func (e *TelegramMessageExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// Parse configuration
	cfg, err := parseConfig[TelegramMessageConfig](config)
	if err != nil {
		return nil, errors.NewConfigurationError("telegram-message", fmt.Sprintf("failed to parse config: %v", err))
	}

	// Validate required fields
	if cfg.ChatID == "" {
		return nil, errors.NewConfigurationError("telegram-message", "missing 'chat_id' in config")
	}
	if cfg.Text == "" {
		return nil, errors.NewConfigurationError("telegram-message", "missing 'text' in config")
	}

	if cfg.OutputKey == "" {
		cfg.OutputKey = "telegram_response"
	}

	// Resolve bot token: config value takes priority, then execution context variables
	botToken := strings.TrimSpace(cfg.BotToken)
	if botToken == "" {
		if token, ok := execCtx.GetVariable("telegram_bot_token"); ok {
			if tokenStr, ok := token.(string); ok && strings.TrimSpace(tokenStr) != "" {
				botToken = strings.TrimSpace(tokenStr)
			}
		}
	}
	if botToken == "" {
		if token, ok := execCtx.GetVariable("TELEGRAM_BOT_TOKEN"); ok {
			if tokenStr, ok := token.(string); ok && strings.TrimSpace(tokenStr) != "" {
				botToken = strings.TrimSpace(tokenStr)
			}
		}
	}
	if botToken == "" {
		return nil, errors.NewConfigurationError("telegram-message", "missing bot token in config or execution context")
	}

	variables := execCtx.GetAllVariables()
	payload := map[string]interface{}{
		"chat_id": substituteVariables(cfg.ChatID, variables),
		"text":    substituteVariables(cfg.Text, variables),
	}
	if cfg.ParseMode != "" {
		payload["parse_mode"] = cfg.ParseMode
	}
	if cfg.DisableNotification {
		payload["disable_notification"] = true
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"telegram-message",
			1,
			fmt.Sprintf("failed to marshal Telegram payload: %v", err),
			err,
			false,
		)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken), bytes.NewReader(body))
	if err != nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"telegram-message",
			1,
			fmt.Sprintf("failed to create Telegram request: %v", err),
			err,
			false,
		)
	}
	request.Header.Set("Content-Type", "application/json")

	startTime := time.Now()
	resp, err := e.client.Do(request)
	latency := time.Since(startTime)
	if err != nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"telegram-message",
			1,
			fmt.Sprintf("failed to call Telegram API: %v", err),
			err,
			true,
		)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"telegram-message",
			1,
			fmt.Sprintf("failed to read Telegram response: %v", err),
			err,
			false,
		)
	}

	var apiResp struct {
		OK          bool                   `json:"ok"`
		Description string                 `json:"description,omitempty"`
		Result      map[string]interface{} `json:"result,omitempty"`
	}
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"telegram-message",
			1,
			fmt.Sprintf("failed to parse Telegram response: %v", err),
			err,
			false,
		)
	}

	if resp.StatusCode >= http.StatusMultipleChoices || !apiResp.OK {
		description := apiResp.Description
		if description == "" {
			description = fmt.Sprintf("telegram API returned status %d", resp.StatusCode)
		}

		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"telegram-message",
			1,
			description,
			nil,
			resp.StatusCode >= http.StatusInternalServerError,
		)
	}

	execCtx.SetVariable(cfg.OutputKey, apiResp.Result)

	return map[string]interface{}{
		"telegram_message": apiResp.Result,
		"latency_ms":       latency.Milliseconds(),
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
	return NodeTypeConditionalRouter
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

	// Get input value with support for nested fields (e.g., "quality_score.pass")
	allVariables := execCtx.GetAllVariables()
	inputValue := getNestedValue(allVariables, cfg.InputKey)
	if inputValue == nil {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"conditional-router",
			1,
			fmt.Sprintf("input variable '%s' not found or is nil", cfg.InputKey),
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
	return NodeTypeDataMerger
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
	return NodeTypeDataAggregator
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
	return NodeTypeScriptExecutor
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

// JSONParserExecutor executes JSON parser nodes.
// It parses JSON strings into structured objects for nested field access.
type JSONParserExecutor struct{}

// NewJSONParserExecutor creates a new JSONParserExecutor.
func NewJSONParserExecutor() *JSONParserExecutor {
	return &JSONParserExecutor{}
}

// Type returns the node type.
func (e *JSONParserExecutor) Type() string {
	return NodeTypeJSONParser
}

// Execute executes a JSON parser node.
func (e *JSONParserExecutor) Execute(ctx context.Context, execCtx *ExecutionContext, nodeID string, config map[string]any) (interface{}, error) {
	// Parse configuration
	cfg, err := parseConfig[JSONParserConfig](config)
	if err != nil {
		return nil, errors.NewConfigurationError("json-parser", fmt.Sprintf("failed to parse config: %v", err))
	}

	// Validate required fields
	if cfg.InputKey == "" {
		return nil, errors.NewConfigurationError("json-parser", "missing 'input_key' in config")
	}

	// Set defaults
	if cfg.OutputKey == "" {
		cfg.OutputKey = cfg.InputKey // Default: overwrite the same variable
	}
	// Default to fail on error
	failOnError := true
	if config["fail_on_error"] != nil {
		if val, ok := config["fail_on_error"].(bool); ok {
			failOnError = val
		}
	}

	// Get input value
	inputValue, ok := execCtx.GetVariable(cfg.InputKey)
	if !ok {
		return nil, errors.NewNodeExecutionError(
			execCtx.State().WorkflowID,
			execCtx.State().ExecutionID,
			nodeID,
			"json-parser",
			1,
			fmt.Sprintf("input variable '%s' not found", cfg.InputKey),
			nil,
			false,
		)
	}

	// Convert to string if needed
	var jsonStr string
	switch v := inputValue.(type) {
	case string:
		jsonStr = v
	case []byte:
		jsonStr = string(v)
	default:
		// If it's already a structured object, just pass it through
		execCtx.SetVariable(cfg.OutputKey, inputValue)
		return map[string]interface{}{
			"status":         "passthrough",
			"input_type":     fmt.Sprintf("%T", inputValue),
			"already_parsed": true,
		}, nil
	}

	// Trim whitespace
	jsonStr = strings.TrimSpace(jsonStr)

	// Parse JSON
	var parsedValue interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsedValue); err != nil {
		if failOnError {
			return nil, errors.NewNodeExecutionError(
				execCtx.State().WorkflowID,
				execCtx.State().ExecutionID,
				nodeID,
				"json-parser",
				1,
				fmt.Sprintf("failed to parse JSON: %v", err),
				err,
				false,
			)
		}

		// If fail_on_error is false, pass through the original value
		log.Warn().
			Str("node_id", nodeID).
			Str("input_key", cfg.InputKey).
			Err(err).
			Msg("Failed to parse JSON, passing through original value")

		execCtx.SetVariable(cfg.OutputKey, inputValue)
		return map[string]interface{}{
			"status":       "parse_error",
			"error":        err.Error(),
			"passthrough":  true,
			"input_length": len(jsonStr),
		}, nil
	}

	// Store parsed value
	execCtx.SetVariable(cfg.OutputKey, parsedValue)

	log.Debug().
		Str("node_id", nodeID).
		Str("input_key", cfg.InputKey).
		Str("output_key", cfg.OutputKey).
		Msgf("Successfully parsed JSON: %T", parsedValue)

	return map[string]interface{}{
		"status":       "success",
		"input_key":    cfg.InputKey,
		"output_key":   cfg.OutputKey,
		"parsed_type":  fmt.Sprintf("%T", parsedValue),
		"input_length": len(jsonStr),
	}, nil
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
