package models

import "time"

// LLMProvider represents the LLM provider type.
type LLMProvider string

const (
	LLMProviderOpenAI          LLMProvider = "openai"           // Chat Completions API
	LLMProviderOpenAIResponses LLMProvider = "openai-responses" // Responses API (GPT-5, o3-mini, gpt-4.1+)
	LLMProviderAnthropic       LLMProvider = "anthropic"
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

	// Responses API specific fields
	Input        interface{}       `json:"input,omitempty"`          // string or []LLMInputItem for Responses API
	HostedTools  []LLMHostedTool   `json:"hosted_tools,omitempty"`   // Built-in OpenAI tools (web_search, file_search, code_interpreter)
	Instructions string            `json:"instructions,omitempty"`   // Alternative to Instruction for Responses API
	Background   bool              `json:"background,omitempty"`     // Background processing for long-running tasks
	Reasoning    *LLMReasoningInfo `json:"reasoning,omitempty"`      // Reasoning configuration for o3-mini, etc.
	Store        *bool             `json:"store,omitempty"`          // Whether to store response in OpenAI (defaults to true)
	MaxToolCalls int               `json:"max_tool_calls,omitempty"` // Limit number of tool iterations
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

	// Responses API specific fields
	Status            string                 `json:"status,omitempty"`             // "completed", "in_progress", "incomplete", "failed"
	OutputItems       []LLMOutputItem        `json:"output_items,omitempty"`       // Polymorphic output array from Responses API
	Error             *LLMError              `json:"error,omitempty"`              // Error object if status is "failed"
	IncompleteDetails map[string]interface{} `json:"incomplete_details,omitempty"` // Reason for incomplete status
	Reasoning         *LLMReasoningInfo      `json:"reasoning,omitempty"`          // Reasoning info for o3-mini, etc.
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

// --- Responses API Specific Types ---

// LLMHostedTool represents built-in OpenAI tools for Responses API.
type LLMHostedTool struct {
	Type string `json:"type"` // "web_search_preview", "file_search", "code_interpreter"

	// For web_search_preview
	Domains           []string `json:"domains,omitempty"`
	SearchContextSize string   `json:"search_context_size,omitempty"` // "small", "medium", "large"

	// For file_search
	VectorStoreIDs []string               `json:"vector_store_ids,omitempty"`
	MaxNumResults  int                    `json:"max_num_results,omitempty"`
	RankingOptions map[string]interface{} `json:"ranking_options,omitempty"`

	// For code_interpreter - no specific config needed
}

// LLMOutputItem represents polymorphic output items from Responses API.
type LLMOutputItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`   // "message", "function_call", "web_search_call", "file_search_call"
	Status string `json:"status"` // "completed", "in_progress", "incomplete", "failed"

	// For type="message"
	Role    string             `json:"role,omitempty"` // "assistant"
	Content []LLMOutputContent `json:"content,omitempty"`

	// For type="function_call"
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"` // JSON string

	// For type="web_search_call" or "file_search_call"
	Queries []string    `json:"queries,omitempty"`
	Results interface{} `json:"results,omitempty"`
}

// LLMOutputContent represents content parts within a message output item.
type LLMOutputContent struct {
	Type        string                `json:"type"` // "output_text"
	Text        string                `json:"text,omitempty"`
	Annotations []LLMOutputAnnotation `json:"annotations,omitempty"`
}

// LLMOutputAnnotation represents citations and references in output.
type LLMOutputAnnotation struct {
	Type       string `json:"type"` // "url_citation", "file_citation"
	StartIndex int    `json:"start_index,omitempty"`
	EndIndex   int    `json:"end_index,omitempty"`

	// For url_citation
	URL   string `json:"url,omitempty"`
	Title string `json:"title,omitempty"`

	// For file_citation
	Index    int    `json:"index,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// LLMReasoningInfo represents reasoning effort and summary for reasoning models.
type LLMReasoningInfo struct {
	Effort  string `json:"effort,omitempty"`  // "low", "medium", "high" (for o3-mini)
	Summary string `json:"summary,omitempty"` // Summary of reasoning process
}

// LLMInputItem represents input items for multimodal requests (Responses API).
type LLMInputItem struct {
	Role    string            `json:"role"` // "user", "system"
	Content []LLMInputContent `json:"content"`
}

// LLMInputContent represents input content parts (Responses API).
type LLMInputContent struct {
	Type     string `json:"type"` // "input_text", "input_image", "input_file"
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	FileURL  string `json:"file_url,omitempty"`
}
