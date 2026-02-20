package models

// ToolCallMode определяет режим работы tool calling
type ToolCallMode string

const (
	ToolCallModeAuto   ToolCallMode = "auto"   // Автоматический цикл внутри executor
	ToolCallModeManual ToolCallMode = "manual" // Ручное подключение через edges
)

// ToolCallConfig конфигурация для tool calling
type ToolCallConfig struct {
	Mode              ToolCallMode `json:"mode"`                           // auto или manual
	MaxIterations     int          `json:"max_iterations,omitempty"`       // Лимит итераций (default: 10)
	TimeoutPerTool    int          `json:"timeout_per_tool,omitempty"`     // Timeout для каждого tool call (seconds)
	TotalTimeout      int          `json:"total_timeout,omitempty"`        // Общий timeout (seconds, default: 300)
	StopOnToolFailure bool         `json:"stop_on_tool_failure,omitempty"` // Остановить при ошибке tool
}

// DefaultToolCallConfig возвращает конфигурацию по умолчанию
func DefaultToolCallConfig() *ToolCallConfig {
	return &ToolCallConfig{
		Mode:              ToolCallModeManual,
		MaxIterations:     10,
		TimeoutPerTool:    30,
		TotalTimeout:      300, // 5 minutes
		StopOnToolFailure: false,
	}
}

// LLMMessage представляет сообщение в conversation history
type LLMMessage struct {
	Role       string         `json:"role"`                   // "user", "assistant", "tool", "system"
	Content    string         `json:"content,omitempty"`      // Текстовое содержимое
	ToolCalls  []LLMToolCall  `json:"tool_calls,omitempty"`   // Tool calls для role="assistant"
	ToolCallID string         `json:"tool_call_id,omitempty"` // ID tool call для role="tool"
	Name       string         `json:"name,omitempty"`         // Имя функции для role="tool"
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// FunctionType тип функции для tool calling
type FunctionType string

const (
	FunctionTypeBuiltin     FunctionType = "builtin"      // Встроенные функции (get_weather, http_request)
	FunctionTypeSubWorkflow FunctionType = "sub_workflow" // Вызов другого workflow
	FunctionTypeCustomCode  FunctionType = "custom_code"  // Inline JS/Python
	FunctionTypeOpenAPI     FunctionType = "openapi"      // Из OpenAPI спецификации
)

// FunctionDefinition определяет функцию для tool calling
type FunctionDefinition struct {
	Type        FunctionType   `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema

	// Для FunctionTypeBuiltin
	BuiltinName string `json:"builtin_name,omitempty"`

	// Для FunctionTypeSubWorkflow
	WorkflowID      string            `json:"workflow_id,omitempty"`
	InputMapping    map[string]string `json:"input_mapping,omitempty"`    // arg -> workflow variable
	OutputExtractor string            `json:"output_extractor,omitempty"` // jq expression

	// Для FunctionTypeCustomCode
	Language string `json:"language,omitempty"` // "javascript" или "python"
	Code     string `json:"code,omitempty"`

	// Для FunctionTypeOpenAPI
	OpenAPISpec string         `json:"openapi_spec,omitempty"` // URL или inline YAML/JSON
	OperationID string         `json:"operation_id,omitempty"`
	BaseURL     string         `json:"base_url,omitempty"`
	AuthConfig  map[string]any `json:"auth_config,omitempty"` // API keys, OAuth, etc
}

// ToolExecutionResult результат выполнения tool
type ToolExecutionResult struct {
	ToolCallID    string         `json:"tool_call_id"`
	FunctionName  string         `json:"function_name"`
	Result        any            `json:"result,omitempty"`
	Error         string         `json:"error,omitempty"`
	ExecutionTime int64          `json:"execution_time_ms"` // В миллисекундах
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// ConversationHistory представляет полную историю разговора
type ConversationHistory struct {
	Messages        []LLMMessage          `json:"messages"`
	ToolExecutions  []ToolExecutionResult `json:"tool_executions,omitempty"`
	TotalIterations int                   `json:"total_iterations"`
}
