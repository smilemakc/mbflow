package mbflow

import "github.com/smilemakc/mbflow/internal/application/executor"

// Re-export NodeConfig interface for public use
type NodeConfig = executor.NodeConfig

// Re-export all config types for public use
type (
	OpenAICompletionConfig       = executor.OpenAICompletionConfig
	HTTPRequestConfig            = executor.HTTPRequestConfig
	TelegramMessageConfig        = executor.TelegramMessageConfig
	ConditionalRouterConfig      = executor.ConditionalRouterConfig
	DataMergerConfig             = executor.DataMergerConfig
	DataAggregatorConfig         = executor.DataAggregatorConfig
	ScriptExecutorConfig         = executor.ScriptExecutorConfig
	JSONParserConfig             = executor.JSONParserConfig
	OpenAIResponsesConfig        = executor.OpenAIResponsesConfig
	ConditionalEdgeConfig        = executor.ConditionalEdgeConfig
	TransformConfig              = executor.TransformConfig
	FunctionCallConfig           = executor.FunctionCallConfig
	OpenAIFunctionResponseConfig = executor.OpenAIFunctionResponseConfig
	OpenAITool                   = executor.OpenAITool
	OpenAIFunction               = executor.OpenAIFunction
)
