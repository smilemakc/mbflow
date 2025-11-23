package executor

// NodeExecutorType represents the type of a node executor.
// This is a type alias for string, allowing seamless use of string literals
// while providing convenient predefined constants.
type NodeExecutorType = string

// Node executor type constants.
// These define all available node types in the system.
const (
	// NodeTypeOpenAICompletion represents an OpenAI completion node.
	// Uses the OpenAI API to generate text completions.
	NodeTypeOpenAICompletion NodeExecutorType = "openai-completion"

	// NodeTypeOpenAIResponses represents an OpenAI Responses API node.
	// Supports structured JSON responses with schema validation.
	NodeTypeOpenAIResponses NodeExecutorType = "openai-responses"

	// NodeTypeHTTPRequest represents an HTTP request node.
	// Sends HTTP requests and processes responses.
	NodeTypeHTTPRequest NodeExecutorType = "http-request"

	// NodeTypeTelegramMessage represents a Telegram message node.
	// Sends messages via the Telegram Bot API.
	NodeTypeTelegramMessage NodeExecutorType = "telegram-message"

	// NodeTypeConditionalRouter represents a conditional routing node.
	// Routes execution based on condition evaluation.
	NodeTypeConditionalRouter NodeExecutorType = "conditional-router"

	// NodeTypeDataMerger represents a data merger node.
	// Merges data from multiple sources using various strategies.
	NodeTypeDataMerger NodeExecutorType = "data-merger"

	// NodeTypeDataAggregator represents a data aggregator node.
	// Aggregates data from multiple fields into a structured output.
	NodeTypeDataAggregator NodeExecutorType = "data-aggregator"

	// NodeTypeScriptExecutor represents a script executor node.
	// Executes scripts (placeholder - requires JS engine).
	NodeTypeScriptExecutor NodeExecutorType = "script-executor"

	// NodeTypeJSONParser represents a JSON parser node.
	// Parses JSON strings into structured objects for nested field access.
	NodeTypeJSONParser NodeExecutorType = "json-parser"
)
