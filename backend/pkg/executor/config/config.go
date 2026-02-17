// Package config provides typed configuration structs for executor types.
// These structs provide type safety when working with executor configurations.
package config

import (
	"encoding/json"
	"fmt"
)

// HTTPConfig represents the configuration for the HTTP executor.
type HTTPConfig struct {
	Method             string            `json:"method"`
	URL                string            `json:"url"`
	Headers            map[string]string `json:"headers,omitempty"`
	Body               any               `json:"body,omitempty"`
	Timeout            int               `json:"timeout,omitempty"`
	IgnoreStatusErrors bool              `json:"ignore_status_errors,omitempty"`
	SuccessStatusCodes []int             `json:"success_status_codes,omitempty"`
	ResponseType       string            `json:"response_type,omitempty"` // "auto", "binary", "json", "text"
	QueryParams        map[string]string `json:"query_params,omitempty"`
	Auth               *HTTPAuthConfig   `json:"auth,omitempty"`
}

// HTTPAuthConfig represents authentication configuration for HTTP requests.
type HTTPAuthConfig struct {
	Type     string `json:"type"`               // "basic", "bearer", "api_key"
	Username string `json:"username,omitempty"` // For basic auth
	Password string `json:"password,omitempty"` // For basic auth
	Token    string `json:"token,omitempty"`    // For bearer auth
	APIKey   string `json:"api_key,omitempty"`  // For API key auth
	Header   string `json:"header,omitempty"`   // Header name for API key
}

// Validate validates the HTTP configuration.
func (c *HTTPConfig) Validate() error {
	if c.Method == "" {
		return fmt.Errorf("method is required")
	}
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[c.Method] {
		return fmt.Errorf("invalid HTTP method: %s", c.Method)
	}

	return nil
}

// TransformConfig represents the configuration for the Transform executor.
type TransformConfig struct {
	Type       string `json:"type"`                 // "passthrough", "template", "expression", "jq"
	Template   string `json:"template,omitempty"`   // For template type
	Expression string `json:"expression,omitempty"` // For expression type
	Filter     string `json:"filter,omitempty"`     // For jq type
}

// Validate validates the Transform configuration.
func (c *TransformConfig) Validate() error {
	validTypes := map[string]bool{
		"passthrough": true, "template": true, "expression": true, "jq": true,
	}

	if c.Type == "" {
		c.Type = "passthrough"
	}

	if !validTypes[c.Type] {
		return fmt.Errorf("invalid transformation type: %s", c.Type)
	}

	switch c.Type {
	case "template":
		if c.Template == "" {
			return fmt.Errorf("template is required for template transformation")
		}
	case "expression":
		if c.Expression == "" {
			return fmt.Errorf("expression is required for expression transformation")
		}
	case "jq":
		if c.Filter == "" {
			return fmt.Errorf("filter is required for jq transformation")
		}
	}

	return nil
}

// LLMConfig represents the configuration for the LLM executor.
type LLMConfig struct {
	Provider         string             `json:"provider"` // "openai", "anthropic", "gemini"
	Model            string             `json:"model"`
	APIKey           string             `json:"api_key,omitempty"`
	Prompt           string             `json:"prompt,omitempty"`
	SystemPrompt     string             `json:"system_prompt,omitempty"`
	Messages         []LLMMessage       `json:"messages,omitempty"`
	Temperature      float64            `json:"temperature,omitempty"`
	MaxTokens        int                `json:"max_tokens,omitempty"`
	TopP             float64            `json:"top_p,omitempty"`
	FrequencyPenalty float64            `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64            `json:"presence_penalty,omitempty"`
	Stop             []string           `json:"stop,omitempty"`
	Tools            []LLMTool          `json:"tools,omitempty"`
	ToolChoice       string             `json:"tool_choice,omitempty"` // "auto", "none", "required"
	ResponseFormat   *LLMResponseFormat `json:"response_format,omitempty"`
	Stream           bool               `json:"stream,omitempty"`
	UseInputDirectly bool               `json:"use_input_directly,omitempty"`
	Metadata         map[string]any     `json:"metadata,omitempty"`
}

// LLMMessage represents a message in the LLM conversation.
type LLMMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// LLMTool represents a tool/function that the LLM can call.
type LLMTool struct {
	Type     string          `json:"type"` // "function"
	Function LLMToolFunction `json:"function"`
}

// LLMToolFunction represents a function definition for tool calling.
type LLMToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// LLMResponseFormat specifies the response format for the LLM.
type LLMResponseFormat struct {
	Type   string         `json:"type"` // "json_object", "json_schema"
	Schema map[string]any `json:"schema,omitempty"`
}

// Validate validates the LLM configuration.
func (c *LLMConfig) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required")
	}

	validProviders := map[string]bool{
		"openai": true, "anthropic": true, "gemini": true, "azure": true,
	}
	if !validProviders[c.Provider] {
		return fmt.Errorf("invalid LLM provider: %s", c.Provider)
	}

	return nil
}

// ConditionalConfig represents the configuration for the Conditional executor.
type ConditionalConfig struct {
	Condition  string              `json:"condition"`
	TrueValue  any                 `json:"true_value,omitempty"`
	FalseValue any                 `json:"false_value,omitempty"`
	Branches   []ConditionalBranch `json:"branches,omitempty"` // For multi-branch conditionals
	Default    any                 `json:"default,omitempty"`
}

// ConditionalBranch represents a branch in a multi-branch conditional.
type ConditionalBranch struct {
	Condition string `json:"condition"`
	Value     any    `json:"value"`
}

// Validate validates the Conditional configuration.
func (c *ConditionalConfig) Validate() error {
	if c.Condition == "" && len(c.Branches) == 0 {
		return fmt.Errorf("condition or branches is required")
	}
	return nil
}

// MergeConfig represents the configuration for the Merge executor.
type MergeConfig struct {
	Strategy string   `json:"strategy,omitempty"` // "concat", "deep_merge", "first", "last"
	Keys     []string `json:"keys,omitempty"`     // Keys to merge (for selective merge)
}

// Validate validates the Merge configuration.
func (c *MergeConfig) Validate() error {
	validStrategies := map[string]bool{
		"concat": true, "deep_merge": true, "first": true, "last": true, "": true,
	}
	if !validStrategies[c.Strategy] {
		return fmt.Errorf("invalid merge strategy: %s", c.Strategy)
	}
	return nil
}

// FileStorageConfig represents the configuration for the FileStorage executor.
type FileStorageConfig struct {
	Operation   string         `json:"operation"` // "read", "write", "delete", "list"
	Path        string         `json:"path,omitempty"`
	Content     any            `json:"content,omitempty"`
	ContentType string         `json:"content_type,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Validate validates the FileStorage configuration.
func (c *FileStorageConfig) Validate() error {
	if c.Operation == "" {
		return fmt.Errorf("operation is required")
	}

	validOperations := map[string]bool{
		"read": true, "write": true, "delete": true, "list": true,
	}
	if !validOperations[c.Operation] {
		return fmt.Errorf("invalid file storage operation: %s", c.Operation)
	}

	switch c.Operation {
	case "read", "delete":
		if c.Path == "" {
			return fmt.Errorf("path is required for %s operation", c.Operation)
		}
	case "write":
		if c.Path == "" {
			return fmt.Errorf("path is required for write operation")
		}
	}

	return nil
}

// ParseConfig parses a map[string]any into a typed config struct.
// The target must be a pointer to a struct.
func ParseConfig[T any](config map[string]any) (*T, error) {
	// Marshal to JSON then unmarshal to struct
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &result, nil
}

// ToMap converts a typed config struct to map[string]any.
func ToMap(config any) (map[string]any, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to convert to map: %w", err)
	}

	return result, nil
}
