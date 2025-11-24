package executor

import (
	"encoding/json"

	"github.com/sashabaranov/go-openai"
)

// NodeConfig is an interface that all node configuration structs must implement.
// It allows converting structured configs to map[string]any for internal processing.
type NodeConfig interface {
	ToMap() (map[string]any, error)
}

// TemplateConfigOptions holds configuration for template processing behavior
type TemplateConfigOptions struct {
	// Mode controls error handling: "strict" (fail on missing vars) or "lenient" (leave placeholder)
	Mode string `json:"mode,omitempty"`

	// Fields specifies which fields to template (empty = all string fields)
	Fields []string `json:"fields,omitempty"`
}

// Template mode constants
const (
	TemplateModeStrict  = "strict"
	TemplateModeLenient = "lenient"
)

// configToMap is a helper function that converts any struct to map[string]any using JSON marshaling.
func configToMap(config any) (map[string]any, error) {
	// Marshal to JSON
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	// Unmarshal to map
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// OpenAICompletionConfig represents the configuration for an OpenAI completion node.
type OpenAICompletionConfig struct {
	// Model is the OpenAI model to use (default: "gpt-4o")
	Model string `json:"model"`

	// Prompt is the prompt template with optional variable substitution using {{variable}}
	Prompt string `json:"prompt"`

	// MaxTokens is the maximum number of tokens to generate (optional)
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0-2.0, optional)
	Temperature float64 `json:"temperature,omitempty"`

	// Tools is a list of function definitions available for the model to call
	Tools []OpenAITool `json:"tools,omitempty"`

	// ToolChoice controls which (if any) function is called by the model
	// Options: "none", "auto", or {"type": "function", "function": {"name": "my_function"}}
	ToolChoice interface{} `json:"tool_choice,omitempty"`

	// ResponseFormat specifies the expected structure or format of the response; keys and values can vary based on configuration.
	ResponseFormat map[string]any `json:"response_format,omitempty"`

	History []openai.ChatCompletionMessage `json:"history,omitempty"`

	// TemplateConfig controls template processing behavior for this node
	TemplateConfig *TemplateConfigOptions `json:"template_config,omitempty"`
}

// OpenAITool represents a function that can be called by the model
type OpenAITool struct {
	// Type is always "function"
	Type string `json:"type"`

	// Function contains the function definition
	Function OpenAIFunction `json:"function"`
}

// OpenAIFunction represents a function definition for OpenAI
type OpenAIFunction struct {
	// Name is the name of the function
	Name string `json:"name"`

	// Description describes what the function does
	Description string `json:"description"`

	// Parameters is a JSON Schema object describing the function parameters
	Parameters map[string]interface{} `json:"parameters"`
}

// ToMap converts the config to map[string]any.
func (c *OpenAICompletionConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// HTTPRequestConfig represents the configuration for an HTTP request node.
type HTTPRequestConfig struct {
	// URL is the request URL template with optional variable substitution
	URL string `json:"url"`

	// Method is the HTTP method (default: "GET")
	Method string `json:"method,omitempty"`

	// Body is the request body (string or map, optional)
	Body interface{} `json:"body,omitempty"`

	// Headers is a map of HTTP headers with optional variable substitution
	Headers map[string]string `json:"headers,omitempty"`

	// TemplateConfig controls template processing behavior for this node
	TemplateConfig *TemplateConfigOptions `json:"template_config,omitempty"`
}

// ToMap converts the config to map[string]any.
func (c *HTTPRequestConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// TelegramMessageConfig represents the configuration for sending messages via Telegram Bot API.
type TelegramMessageConfig struct {
	// BotToken is the Telegram bot token (can also be provided via execution context)
	BotToken string `json:"bot_token,omitempty"`

	// ChatID is the target chat ID or username (e.g., "@channelname")
	ChatID string `json:"chat_id"`

	// Text is the message text with optional variable substitution
	Text string `json:"text"`

	// ParseMode is the text parse mode (e.g., "Markdown", "HTML")
	ParseMode string `json:"parse_mode,omitempty"`

	// DisableNotification sends the message silently when true
	DisableNotification bool `json:"disable_notification,omitempty"`

	// TemplateConfig controls template processing behavior for this node
	TemplateConfig *TemplateConfigOptions `json:"template_config,omitempty"`
}

// ToMap converts the config to map[string]any.
func (c *TelegramMessageConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// TransformConfig represents the configuration for a transform node.
type TransformConfig struct {
	// Transformations is a map of output keys to expression strings
	Transformations map[string]string `json:"transformations"`
	AllowUndefined  bool              `json:"allow_undefined,omitempty"`
}

// ToMap converts the config to map[string]any.
func (c *TransformConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// FunctionCallConfig represents the configuration for a function call executor node.
type FunctionCallConfig struct {
	// InputKey is the key containing the AI response with tool_calls
	InputKey string `json:"input_key"`

	// FunctionName is the name of the function to execute (optional, executes first if not specified)
	FunctionName string `json:"function_name,omitempty"`

	// Handler is the type of handler to use: "script", "http", "builtin"
	Handler string `json:"handler"`

	// HandlerConfig contains handler-specific configuration
	HandlerConfig map[string]interface{} `json:"handler_config,omitempty"`

	// AIResponseKey is the key containing the original AI response with tool_calls
	AiResponseKey string `json:"ai_response_key"`
}

// ToMap converts the config to map[string]any.
func (c *FunctionCallConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// OpenAIFunctionResponseConfig represents the configuration for continuing the conversation after function execution.
type OpenAIFunctionResponseConfig struct {
	// Model is the OpenAI model to use (default: "gpt-4o")
	Model string `json:"model,omitempty"`

	// AIResponseKey is the key containing the original AI response with tool_calls
	AIResponseKey string `json:"ai_response_key"`

	// FunctionResultKey is the key containing the function execution result
	FunctionResultKey string `json:"function_result_key"`

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `json:"max_tokens,omitempty"`

	// ResponseFormat specifies the expected structure or format of the response; keys and values can vary based on configuration.
	ResponseFormat map[string]any `json:"response_format,omitempty"`

	// Temperature controls randomness (0.0 to 2.0)
	Temperature float64 `json:"temperature,omitempty"`
}

// ToMap converts the config to map[string]any.
func (c *OpenAIFunctionResponseConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// ConditionalRouterConfig represents the configuration for a conditional router node.
type ConditionalRouterConfig struct {
	// Routes maps condition values to route identifiers
	// Can be map[string]string or map[string]interface{}
	Routes map[string]interface{} `json:"routes"`
}

// ToMap converts the config to map[string]any.
func (c *ConditionalRouterConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// DataMergerConfig represents the configuration for a data merger node.
type DataMergerConfig struct {
	// Strategy is the merging strategy (default: "select_first_available")
	// Options: "select_first_available", "merge_all"
	Strategy string `json:"strategy,omitempty"`

	// Sources is a list of variable keys to merge from
	Sources []string `json:"sources"`
}

// ToMap converts the config to map[string]any.
func (c *DataMergerConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// DataAggregatorConfig represents the configuration for a data aggregator node.
type DataAggregatorConfig struct {
	// Fields maps output field names to source variable keys (for field extraction mode)
	Fields map[string]string `json:"fields,omitempty"`

	// InputKey is the key of the array to aggregate (for array aggregation mode)
	InputKey string `json:"input_key,omitempty"`

	// Function is the aggregation function (sum, count, avg, min, max, collect)
	Function string `json:"function,omitempty"`

	// OutputFormat is the output format (optional, default: "json")
	OutputFormat string `json:"output_format,omitempty"`
}

// ToMap converts the config to map[string]any.
func (c *DataAggregatorConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// ScriptExecutorConfig represents the configuration for a script executor node.
type ScriptExecutorConfig struct {
	// Script is the script code to execute
	Script string `json:"script,omitempty"`

	// Language is the script language (e.g., "javascript", "python")
	Language string `json:"language,omitempty"`
}

// ToMap converts the config to map[string]any.
func (c *ScriptExecutorConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// JSONParserConfig represents the configuration for a JSON parser node.
// This executor parses JSON strings into structured objects for nested field access.
type JSONParserConfig struct {
	// InputKey is the variable key containing the JSON string to parse (default: "raw_data")
	InputKey string `json:"input_key,omitempty"`

	// OutputKey is the variable key to store the parsed JSON object (default: "parsed_data")
	OutputKey string `json:"output_key,omitempty"`

	// FailOnError determines whether to fail the node on parse errors (default: true)
	// If false, the original value will be passed through on parse errors
	FailOnError bool `json:"fail_on_error,omitempty"`
}

// ToMap converts the config to map[string]any.
func (c *JSONParserConfig) ToMap() (map[string]any, error) {
	m, err := configToMap(c)
	if err != nil {
		return nil, err
	}

	// Apply defaults
	if c.InputKey == "" {
		m["input_key"] = "raw_data"
	}
	if c.OutputKey == "" {
		m["output_key"] = "parsed_data"
	}

	return m, nil
}

// OpenAIResponsesConfig represents the configuration for an OpenAI Responses API node.
// This executor supports structured JSON responses using the OpenAI Responses API.
type OpenAIResponsesConfig struct {
	// Model is the OpenAI model to use (default: "gpt-4o")
	Model string `json:"model"`

	// Prompt is the prompt template with optional variable substitution using {{variable}}
	Prompt string `json:"prompt"`

	// MaxTokens is the maximum number of tokens to generate (optional)
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0-2.0, optional)
	Temperature float64 `json:"temperature,omitempty"`

	// ResponseFormat defines the structure of the response (optional)
	// Can be "text" for plain text or a JSON schema object for structured output
	// Example: {"type": "json_schema", "json_schema": {...}}
	ResponseFormat map[string]interface{} `json:"response_format,omitempty"`

	// TopP controls nucleus sampling (0.0-1.0, optional)
	TopP float64 `json:"top_p,omitempty"`

	// FrequencyPenalty controls repetition penalty (-2.0 to 2.0, optional)
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`

	// PresencePenalty controls topic diversity (-2.0 to 2.0, optional)
	PresencePenalty float64 `json:"presence_penalty,omitempty"`

	// Stop sequences where the API will stop generating further tokens (optional)
	Stop []string `json:"stop,omitempty"`

	// TemplateConfig controls template processing behavior for this node
	TemplateConfig *TemplateConfigOptions `json:"template_config,omitempty"`
}

// ToMap converts the config to map[string]any.
func (c *OpenAIResponsesConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}

// ConditionalEdgeConfig represents the configuration for a conditional edge.
type ConditionalEdgeConfig struct {
	// Condition is the expression to evaluate (e.g., "quality_rating == 'high'")
	Condition string `json:"condition"`
}

// ToMap converts the config to map[string]any.
func (c *ConditionalEdgeConfig) ToMap() (map[string]any, error) {
	return configToMap(c)
}
