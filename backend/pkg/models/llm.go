package models

import "time"

// LLMProvider represents the LLM provider type.
type LLMProvider string

const (
	LLMProviderOpenAI    LLMProvider = "openai"
	LLMProviderAnthropic LLMProvider = "anthropic"
)

// LLMRequest represents a request to an LLM.
type LLMRequest struct {
	Provider           LLMProvider            `json:"provider"`
	Model              string                 `json:"model"`
	Instruction        string                 `json:"instruction,omitempty"` // System message
	Prompt             string                 `json:"prompt"`                // User message
	MaxTokens          int                    `json:"max_tokens,omitempty"`
	Temperature        float64                `json:"temperature,omitempty"`
	TopP               float64                `json:"top_p,omitempty"`
	FrequencyPenalty   float64                `json:"frequency_penalty,omitempty"`
	PresencePenalty    float64                `json:"presence_penalty,omitempty"`
	StopSequences      []string               `json:"stop_sequences,omitempty"`
	VectorStoreID      string                 `json:"vector_store_id,omitempty"`      // OpenAI vector store
	ImageURLs          []string               `json:"image_url,omitempty"`            // Image URLs for vision models
	ImageIDs           []string               `json:"image_id,omitempty"`             // OpenAI file IDs for images
	FileIDs            []string               `json:"file_id,omitempty"`              // OpenAI file IDs for documents
	Tools              []LLMTool              `json:"tools,omitempty"`                // Function definitions
	ResponseFormat     *LLMResponseFormat     `json:"response_format,omitempty"`      // Structured output format
	PreviousResponseID string                 `json:"previous_response_id,omitempty"` // For conversation chaining
	ProviderConfig     map[string]interface{} `json:"provider_config,omitempty"`      // Provider-specific configuration (api_key, base_url, org_id, etc.)
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// LLMTool represents a function tool available to the LLM.
type LLMTool struct {
	Type     string          `json:"type"` // "function"
	Function LLMFunctionTool `json:"function"`
}

// LLMFunctionTool represents a function definition.
type LLMFunctionTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// LLMResponseFormat defines the expected response format.
type LLMResponseFormat struct {
	Type       string         `json:"type"` // "text", "json_object", "json_schema"
	JSONSchema *LLMJSONSchema `json:"json_schema,omitempty"`
}

// LLMJSONSchema defines a JSON schema for structured outputs.
type LLMJSONSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]interface{} `json:"schema"` // JSON Schema definition
	Strict      bool                   `json:"strict,omitempty"`
}

// LLMResponse represents a response from an LLM.
type LLMResponse struct {
	Content      string                 `json:"content"`
	ResponseID   string                 `json:"response_id,omitempty"`
	Model        string                 `json:"model"`
	Usage        LLMUsage               `json:"usage"`
	ToolCalls    []LLMToolCall          `json:"tool_calls,omitempty"`
	FinishReason string                 `json:"finish_reason"` // "stop", "length", "tool_calls", "content_filter"
	CreatedAt    time.Time              `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// LLMUsage represents token usage statistics.
type LLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMToolCall represents a function call made by the LLM.
type LLMToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"` // "function"
	Function LLMFunctionCall `json:"function"`
}

// LLMFunctionCall represents a function call with arguments.
type LLMFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// LLMError represents an error from an LLM API.
type LLMError struct {
	Provider LLMProvider `json:"provider"`
	Code     string      `json:"code"`
	Message  string      `json:"message"`
	Type     string      `json:"type,omitempty"`
	Param    string      `json:"param,omitempty"`
}

func (e *LLMError) Error() string {
	return "LLM error (" + string(e.Provider) + "): " + e.Message
}

// LLMConfig represents the configuration for an LLM executor node.
type LLMConfig struct {
	Provider           string                   `json:"provider"`
	Model              string                   `json:"model"`
	Instruction        string                   `json:"instruction,omitempty"`
	Prompt             string                   `json:"prompt"`
	MaxTokens          int                      `json:"max_tokens,omitempty"`
	Temperature        float64                  `json:"temperature,omitempty"`
	TopP               float64                  `json:"top_p,omitempty"`
	FrequencyPenalty   float64                  `json:"frequency_penalty,omitempty"`
	PresencePenalty    float64                  `json:"presence_penalty,omitempty"`
	StopSequences      []string                 `json:"stop_sequences,omitempty"`
	VectorStoreID      string                   `json:"vector_store_id,omitempty"`
	ImageURLs          []string                 `json:"image_url,omitempty"`
	ImageIDs           []string                 `json:"image_id,omitempty"`
	FileIDs            []string                 `json:"file_id,omitempty"`
	Tools              []map[string]interface{} `json:"tools,omitempty"`
	ResponseFormat     map[string]interface{}   `json:"response_format,omitempty"`
	PreviousResponseID string                   `json:"previous_response_id,omitempty"`
}
