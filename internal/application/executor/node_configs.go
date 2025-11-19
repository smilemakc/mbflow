package executor

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

	// OutputKey is the key to store the output in execution context (default: "output")
	OutputKey string `json:"output_key,omitempty"`

	// APIKey is the OpenAI API key (optional, can come from context or default)
	APIKey string `json:"api_key,omitempty"`
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

	// OutputKey is the key to store the response in execution context (default: "output")
	OutputKey string `json:"output_key,omitempty"`
}

// ConditionalRouterConfig represents the configuration for a conditional router node.
type ConditionalRouterConfig struct {
	// InputKey is the variable key to read from execution context
	InputKey string `json:"input_key"`

	// Routes maps condition values to route identifiers
	// Can be map[string]string or map[string]interface{}
	Routes map[string]interface{} `json:"routes"`
}

// DataMergerConfig represents the configuration for a data merger node.
type DataMergerConfig struct {
	// Strategy is the merging strategy (default: "select_first_available")
	// Options: "select_first_available", "merge_all"
	Strategy string `json:"strategy,omitempty"`

	// Sources is a list of variable keys to merge from
	Sources []string `json:"sources"`

	// OutputKey is the key to store the merged result (default: "output")
	OutputKey string `json:"output_key,omitempty"`
}

// DataAggregatorConfig represents the configuration for a data aggregator node.
type DataAggregatorConfig struct {
	// Fields maps output field names to source variable keys
	Fields map[string]string `json:"fields"`

	// OutputFormat is the output format (optional, default: "json")
	OutputFormat string `json:"output_format,omitempty"`

	// OutputKey is the key to store the aggregated result (default: "output")
	OutputKey string `json:"output_key,omitempty"`
}

// ScriptExecutorConfig represents the configuration for a script executor node.
type ScriptExecutorConfig struct {
	// Script is the script code to execute
	Script string `json:"script,omitempty"`

	// Language is the script language (e.g., "javascript", "python")
	Language string `json:"language,omitempty"`

	// OutputKey is the key to store the script output (default: "output")
	OutputKey string `json:"output_key,omitempty"`
}

// ConditionalEdgeConfig represents the configuration for a conditional edge.
type ConditionalEdgeConfig struct {
	// Condition is the expression to evaluate (e.g., "quality_rating == 'high'")
	Condition string `json:"condition"`
}
