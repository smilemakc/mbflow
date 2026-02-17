package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
	"github.com/smilemakc/mbflow/pkg/models"
)

// LLMProvider interface for different LLM providers.
type LLMProvider interface {
	Execute(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error)
}

// LLMExecutor executes LLM requests with support for multiple providers.
type LLMExecutor struct {
	*executor.BaseExecutor
	providers           map[models.LLMProvider]LLMProvider
	toolCallingRegistry *ToolCallingRegistry
	mu                  sync.RWMutex
}

// NewLLMExecutor creates a new LLM executor.
func NewLLMExecutor() *LLMExecutor {
	return &LLMExecutor{
		BaseExecutor: executor.NewBaseExecutor("llm"),
		providers:    make(map[models.LLMProvider]LLMProvider),
	}
}

// SetToolCallingRegistry sets the tool calling registry for auto mode support.
func (e *LLMExecutor) SetToolCallingRegistry(registry *ToolCallingRegistry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.toolCallingRegistry = registry
}

// RegisterProvider registers a custom LLM provider.
func (e *LLMExecutor) RegisterProvider(providerType models.LLMProvider, provider LLMProvider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.providers[providerType] = provider
}

// Execute executes an LLM request.
//
// Template Resolution (Automatic):
// The workflow engine AUTOMATICALLY wraps this executor with TemplateExecutorWrapper
// during node execution. Templates in the config are resolved BEFORE this method is called.
//
// How it works:
//  1. NodeExecutor gets the base LLM executor from registry
//  2. Creates ExecutionContextData with:
//     - ParentNodeOutput (mapped to {{input.field}})
//     - WorkflowVariables (mapped to {{env.var}})
//     - ExecutionVariables (runtime overrides for {{env.var}})
//  3. Creates template engine from ExecutionContextData
//  4. Wraps this executor: NewTemplateExecutorWrapper(llmExec, engine)
//  5. Calls wrapped Execute - templates are auto-resolved
//
// Example workflow configuration:
//
//	config: {
//	  "provider": "openai",
//	  "model": "{{env.model}}",
//	  "api_key": "{{env.openai_api_key}}",
//	  "prompt": "Analyze this code: {{input.code}}"
//	}
//
// After automatic template resolution:
//
//	config: {
//	  "provider": "openai",
//	  "model": "gpt-4",
//	  "api_key": "sk-abc123...",
//	  "prompt": "Analyze this code: func main() {...}"
//	}
//
// Input Parameter Usage:
// The 'input' parameter contains the complete output from parent nodes and can be used in several ways:
//  1. Enriching the prompt with structured data (when templates aren't sufficient)
//  2. Providing input for Responses API (OpenAI Responses requires structured input)
//  3. Dynamic prompt augmentation based on parent node results
//
// The input parameter is available both through template resolution ({{input.field}}) and
// directly in the request when use_input_directly is enabled in config.
//
// See: executor.Executor for implementation details.
func (e *LLMExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	// Parse config into LLMRequest
	req, err := e.parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM config: %w", err)
	}

	// If config doesn't explicitly set Input field and input parameter is provided,
	// check if we should use it directly (useful for Responses API or structured inputs)
	if req.Input == nil && input != nil {
		// Check if config specifies to use input directly
		if useInputDirectly, ok := config["use_input_directly"].(bool); ok && useInputDirectly {
			req.Input = input
		}
	}

	// Create provider with config
	provider, err := e.getOrCreateProvider(req)
	if err != nil {
		return nil, err
	}

	// Check if auto mode tool calling is enabled
	if req.ToolCallConfig != nil && req.ToolCallConfig.Mode == models.ToolCallModeAuto {
		// Use automatic tool calling mode
		response, err := e.executeWithToolCalling(ctx, req, provider)
		if err != nil {
			return nil, fmt.Errorf("auto mode tool calling failed: %w", err)
		}
		return e.responseToMap(response), nil
	}

	// Execute request (manual mode or no tool calling)
	response, err := provider.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM execution failed: %w", err)
	}

	// Convert response to map for output
	return e.responseToMap(response), nil
}

// Validate validates the LLM executor configuration.
func (e *LLMExecutor) Validate(config map[string]any) error {
	// Validate required fields
	if err := e.ValidateRequired(config, "provider", "model", "prompt", "api_key"); err != nil {
		return err
	}

	// Validate provider
	providerStr, err := e.GetString(config, "provider")
	if err != nil {
		return err
	}

	provider := models.LLMProvider(providerStr)
	validProviders := map[models.LLMProvider]bool{
		models.LLMProviderOpenAI:          true,
		models.LLMProviderOpenAIResponses: true,
		models.LLMProviderAnthropic:       true,
		models.LLMProviderGemini:          true,
	}
	if !validProviders[provider] {
		return fmt.Errorf("unsupported LLM provider: %s", providerStr)
	}

	// Validate model
	model, err := e.GetString(config, "model")
	if err != nil {
		return err
	}
	if model == "" {
		return fmt.Errorf("model cannot be empty")
	}

	// Validate optional numeric fields
	if maxTokens := e.GetIntDefault(config, "max_tokens", 0); maxTokens < 0 {
		return fmt.Errorf("max_tokens must be >= 0")
	}

	if temp, ok := config["temperature"].(float64); ok && (temp < 0 || temp > 2) {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if topP, ok := config["top_p"].(float64); ok && (topP < 0 || topP > 1) {
		return fmt.Errorf("top_p must be between 0 and 1")
	}

	// Validate response_format if present
	if responseFormat, ok := config["response_format"].(map[string]any); ok {
		if err := e.validateResponseFormat(responseFormat); err != nil {
			return err
		}
	}

	// Validate tools if present
	if tools, ok := config["tools"].([]any); ok {
		if err := e.validateTools(tools); err != nil {
			return err
		}
	}

	return nil
}

// parseConfig parses the executor config into an LLMRequest.
func (e *LLMExecutor) parseConfig(config map[string]any) (*models.LLMRequest, error) {
	req := &models.LLMRequest{}

	// Required fields
	providerStr, _ := e.GetString(config, "provider")
	req.Provider = models.LLMProvider(providerStr)

	req.Model, _ = e.GetString(config, "model")
	req.Prompt, _ = e.GetString(config, "prompt")

	// Optional fields
	req.Instruction = e.GetStringDefault(config, "instruction", "")
	req.MaxTokens = e.GetIntDefault(config, "max_tokens", 0)
	req.VectorStoreID = e.GetStringDefault(config, "vector_store_id", "")
	req.PreviousResponseID = e.GetStringDefault(config, "previous_response_id", "")

	// Numeric parameters
	if temp, ok := config["temperature"].(float64); ok {
		req.Temperature = temp
	}
	if topP, ok := config["top_p"].(float64); ok {
		req.TopP = topP
	}
	if freqPenalty, ok := config["frequency_penalty"].(float64); ok {
		req.FrequencyPenalty = freqPenalty
	}
	if presPenalty, ok := config["presence_penalty"].(float64); ok {
		req.PresencePenalty = presPenalty
	}

	// Arrays
	if imageURLs, ok := config["image_url"].([]any); ok {
		req.ImageURLs = e.toStringSlice(imageURLs)
	}
	if imageIDs, ok := config["image_id"].([]any); ok {
		req.ImageIDs = e.toStringSlice(imageIDs)
	}
	if fileIDs, ok := config["file_id"].([]any); ok {
		req.FileIDs = e.toStringSlice(fileIDs)
	}
	if stopSeqs, ok := config["stop_sequences"].([]any); ok {
		req.StopSequences = e.toStringSlice(stopSeqs)
	}

	// Parse file attachments
	if files, ok := config["files"].([]any); ok {
		parsedFiles, err := e.parseFiles(files)
		if err != nil {
			return nil, err
		}
		req.Files = parsedFiles
	}

	// Tools
	if tools, ok := config["tools"].([]any); ok {
		parsedTools, err := e.parseTools(tools)
		if err != nil {
			return nil, err
		}
		req.Tools = parsedTools
	}

	// Response format
	if responseFormat, ok := config["response_format"].(map[string]any); ok {
		parsedFormat, err := e.parseResponseFormat(responseFormat)
		if err != nil {
			return nil, err
		}
		req.ResponseFormat = parsedFormat
	}

	// Extract provider configuration
	req.ProviderConfig = e.extractProviderConfig(config)

	// Responses API specific fields
	if input, ok := config["input"]; ok {
		req.Input = input
	}
	if instructions, ok := config["instructions"].(string); ok {
		req.Instructions = instructions
	}
	if background, ok := config["background"].(bool); ok {
		req.Background = background
	}
	if maxToolCalls, ok := config["max_tool_calls"].(int); ok {
		req.MaxToolCalls = maxToolCalls
	} else if maxToolCallsFloat, ok := config["max_tool_calls"].(float64); ok {
		req.MaxToolCalls = int(maxToolCallsFloat)
	}
	if store, ok := config["store"].(bool); ok {
		req.Store = &store
	}

	// Parse reasoning
	if reasoning, ok := config["reasoning"].(map[string]any); ok {
		req.Reasoning = &models.LLMReasoningInfo{}
		if effort, ok := reasoning["effort"].(string); ok {
			req.Reasoning.Effort = effort
		}
	}

	// Parse hosted tools
	if hostedTools, ok := config["hosted_tools"].([]any); ok {
		parsedHostedTools, err := e.parseHostedTools(hostedTools)
		if err != nil {
			return nil, err
		}
		req.HostedTools = parsedHostedTools
	}

	// Parse tool calling configuration
	if toolCallConfig, ok := config["tool_call_config"].(map[string]any); ok {
		parsedConfig, err := e.parseToolCallConfig(toolCallConfig)
		if err != nil {
			return nil, err
		}
		req.ToolCallConfig = parsedConfig
	}

	// Parse messages (conversation history)
	if messages, ok := config["messages"].([]any); ok {
		parsedMessages, err := e.parseMessages(messages)
		if err != nil {
			return nil, err
		}
		req.Messages = parsedMessages
	}

	// Parse functions (extended function definitions)
	if functions, ok := config["functions"].([]any); ok {
		parsedFunctions, err := e.parseFunctions(functions)
		if err != nil {
			return nil, err
		}
		req.Functions = parsedFunctions
	}

	return req, nil
}

// parseTools parses tools configuration into LLMTool structs.
func (e *LLMExecutor) parseTools(toolsConfig []any) ([]models.LLMTool, error) {
	tools := make([]models.LLMTool, len(toolsConfig))

	for i, toolConfig := range toolsConfig {
		toolMap, ok := toolConfig.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("tool %d is not a valid object", i)
		}

		toolType, _ := toolMap["type"].(string)
		if toolType != "function" {
			return nil, fmt.Errorf("tool %d: only 'function' type is supported", i)
		}

		funcConfig, ok := toolMap["function"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("tool %d: missing function definition", i)
		}

		name, _ := funcConfig["name"].(string)
		description, _ := funcConfig["description"].(string)
		params, _ := funcConfig["parameters"].(map[string]any)

		if name == "" {
			return nil, fmt.Errorf("tool %d: function name is required", i)
		}

		tools[i] = models.LLMTool{
			Type: "function",
			Function: models.LLMFunctionTool{
				Name:        name,
				Description: description,
				Parameters:  params,
			},
		}
	}

	return tools, nil
}

// parseResponseFormat parses response format configuration.
func (e *LLMExecutor) parseResponseFormat(formatConfig map[string]any) (*models.LLMResponseFormat, error) {
	formatType, _ := formatConfig["type"].(string)
	if formatType == "" {
		return nil, fmt.Errorf("response_format type is required")
	}

	format := &models.LLMResponseFormat{
		Type: formatType,
	}

	// Parse JSON schema if present
	if formatType == "json_schema" {
		schemaConfig, ok := formatConfig["json_schema"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("json_schema is required for json_schema type")
		}

		name, _ := schemaConfig["name"].(string)
		description, _ := schemaConfig["description"].(string)
		schema, _ := schemaConfig["schema"].(map[string]any)
		strict, _ := schemaConfig["strict"].(bool)

		if name == "" {
			return nil, fmt.Errorf("json_schema name is required")
		}

		format.JSONSchema = &models.LLMJSONSchema{
			Name:        name,
			Description: description,
			Schema:      schema,
			Strict:      strict,
		}
	}

	return format, nil
}

// validateResponseFormat validates response format configuration.
func (e *LLMExecutor) validateResponseFormat(formatConfig map[string]any) error {
	formatType, ok := formatConfig["type"].(string)
	if !ok || formatType == "" {
		return fmt.Errorf("response_format type is required")
	}

	validTypes := map[string]bool{
		"text":        true,
		"json_object": true,
		"json_schema": true,
	}

	if !validTypes[formatType] {
		return fmt.Errorf("invalid response_format type: %s", formatType)
	}

	if formatType == "json_schema" {
		schemaConfig, ok := formatConfig["json_schema"].(map[string]any)
		if !ok {
			return fmt.Errorf("json_schema is required for json_schema type")
		}

		name, ok := schemaConfig["name"].(string)
		if !ok || name == "" {
			return fmt.Errorf("json_schema name is required")
		}

		if _, ok := schemaConfig["schema"].(map[string]any); !ok {
			return fmt.Errorf("json_schema schema is required")
		}
	}

	return nil
}

// validateTools validates tools configuration.
func (e *LLMExecutor) validateTools(toolsConfig []any) error {
	for i, toolConfig := range toolsConfig {
		toolMap, ok := toolConfig.(map[string]any)
		if !ok {
			return fmt.Errorf("tool %d is not a valid object", i)
		}

		toolType, ok := toolMap["type"].(string)
		if !ok || toolType != "function" {
			return fmt.Errorf("tool %d: only 'function' type is supported", i)
		}

		funcConfig, ok := toolMap["function"].(map[string]any)
		if !ok {
			return fmt.Errorf("tool %d: missing function definition", i)
		}

		name, ok := funcConfig["name"].(string)
		if !ok || name == "" {
			return fmt.Errorf("tool %d: function name is required", i)
		}
	}

	return nil
}

// getProvider gets a provider instance.
func (e *LLMExecutor) getProvider(providerType models.LLMProvider) (LLMProvider, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	provider, ok := e.providers[providerType]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerType)
	}

	return provider, nil
}

// hasProvider checks if a provider is registered.
func (e *LLMExecutor) hasProvider(providerType models.LLMProvider) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	_, ok := e.providers[providerType]
	return ok
}

// getOrCreateProvider creates a provider instance from the request configuration.
// It first checks if a provider is already registered (for testing), then creates a new one from config.
func (e *LLMExecutor) getOrCreateProvider(req *models.LLMRequest) (LLMProvider, error) {
	// Check if provider is already registered (for testing/custom providers)
	if provider, err := e.getProvider(req.Provider); err == nil {
		return provider, nil
	}

	// Create provider from configuration
	switch req.Provider {
	case models.LLMProviderOpenAI:
		apiKey, _ := req.ProviderConfig["api_key"].(string)
		baseURL, _ := req.ProviderConfig["base_url"].(string)
		orgID, _ := req.ProviderConfig["org_id"].(string)
		return NewOpenAIProvider(apiKey, baseURL, orgID)
	case models.LLMProviderOpenAIResponses:
		apiKey, _ := req.ProviderConfig["api_key"].(string)
		baseURL, _ := req.ProviderConfig["base_url"].(string)
		orgID, _ := req.ProviderConfig["org_id"].(string)
		return NewOpenAIResponsesProvider(apiKey, baseURL, orgID)
	case models.LLMProviderGemini:
		apiKey, _ := req.ProviderConfig["api_key"].(string)
		baseURL, _ := req.ProviderConfig["base_url"].(string)
		return NewGeminiProvider(apiKey, baseURL)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}
}

// responseToMap converts LLMResponse to a map for output.
func (e *LLMExecutor) responseToMap(response *models.LLMResponse) map[string]any {
	result := map[string]any{
		"content":       response.Content,
		"response_id":   response.ResponseID,
		"model":         response.Model,
		"finish_reason": response.FinishReason,
		"created_at":    response.CreatedAt,
		"usage": map[string]any{
			"prompt_tokens":     response.Usage.PromptTokens,
			"completion_tokens": response.Usage.CompletionTokens,
			"total_tokens":      response.Usage.TotalTokens,
		},
	}

	if len(response.ToolCalls) > 0 {
		toolCalls := make([]map[string]any, len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			toolCalls[i] = map[string]any{
				"id":   tc.ID,
				"type": tc.Type,
				"function": map[string]any{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			}
		}
		result["tool_calls"] = toolCalls
	}

	if response.Metadata != nil {
		result["metadata"] = response.Metadata
	}

	// Responses API specific fields
	if response.Status != "" {
		result["status"] = response.Status
	}
	if len(response.OutputItems) > 0 {
		outputItems := make([]map[string]any, len(response.OutputItems))
		for i, item := range response.OutputItems {
			itemMap := map[string]any{
				"id":     item.ID,
				"type":   item.Type,
				"status": item.Status,
			}
			if item.Role != "" {
				itemMap["role"] = item.Role
			}
			if len(item.Content) > 0 {
				content := make([]map[string]any, len(item.Content))
				for j, c := range item.Content {
					content[j] = map[string]any{
						"type": c.Type,
						"text": c.Text,
					}
					if len(c.Annotations) > 0 {
						annotations := make([]map[string]any, len(c.Annotations))
						for k, ann := range c.Annotations {
							annotations[k] = map[string]any{
								"type":        ann.Type,
								"start_index": ann.StartIndex,
								"end_index":   ann.EndIndex,
								"url":         ann.URL,
								"title":       ann.Title,
								"index":       ann.Index,
								"file_id":     ann.FileID,
								"filename":    ann.Filename,
							}
						}
						content[j]["annotations"] = annotations
					}
				}
				itemMap["content"] = content
			}
			if item.CallID != "" {
				itemMap["call_id"] = item.CallID
			}
			if item.Name != "" {
				itemMap["name"] = item.Name
			}
			if item.Arguments != "" {
				itemMap["arguments"] = item.Arguments
			}
			if len(item.Queries) > 0 {
				itemMap["queries"] = item.Queries
			}
			if item.Results != nil {
				itemMap["results"] = item.Results
			}
			outputItems[i] = itemMap
		}
		result["output_items"] = outputItems
	}
	if response.Error != nil {
		result["error"] = map[string]any{
			"provider": response.Error.Provider,
			"code":     response.Error.Code,
			"message":  response.Error.Message,
			"type":     response.Error.Type,
		}
	}
	if response.IncompleteDetails != nil {
		result["incomplete_details"] = response.IncompleteDetails
	}
	if response.Reasoning != nil {
		result["reasoning"] = map[string]any{
			"effort":  response.Reasoning.Effort,
			"summary": response.Reasoning.Summary,
		}
	}

	// Tool calling auto mode fields
	if len(response.Messages) > 0 {
		messages := make([]any, len(response.Messages))
		for i, msg := range response.Messages {
			msgMap := map[string]any{
				"role":    msg.Role,
				"content": msg.Content,
			}
			if len(msg.ToolCalls) > 0 {
				toolCalls := make([]any, len(msg.ToolCalls))
				for j, tc := range msg.ToolCalls {
					toolCalls[j] = map[string]any{
						"id":   tc.ID,
						"type": tc.Type,
						"function": map[string]any{
							"name":      tc.Function.Name,
							"arguments": tc.Function.Arguments,
						},
					}
				}
				msgMap["tool_calls"] = toolCalls
			}
			if msg.ToolCallID != "" {
				msgMap["tool_call_id"] = msg.ToolCallID
			}
			if msg.Name != "" {
				msgMap["name"] = msg.Name
			}
			if msg.Metadata != nil {
				msgMap["metadata"] = msg.Metadata
			}
			messages[i] = msgMap
		}
		result["messages"] = messages
	}

	if len(response.ToolExecutions) > 0 {
		toolExecutions := make([]any, len(response.ToolExecutions))
		for i, exec := range response.ToolExecutions {
			execMap := map[string]any{
				"tool_call_id":   exec.ToolCallID,
				"function_name":  exec.FunctionName,
				"execution_time": exec.ExecutionTime,
			}
			if exec.Result != nil {
				execMap["result"] = exec.Result
			}
			if exec.Error != "" {
				execMap["error"] = exec.Error
			}
			if exec.Metadata != nil {
				execMap["metadata"] = exec.Metadata
			}
			toolExecutions[i] = execMap
		}
		result["tool_executions"] = toolExecutions
	}

	if response.TotalIterations > 0 {
		result["total_iterations"] = response.TotalIterations
	}

	if response.StoppedReason != "" {
		result["stopped_reason"] = response.StoppedReason
	}

	return result
}

// extractProviderConfig extracts provider-specific configuration from the node config.
func (e *LLMExecutor) extractProviderConfig(config map[string]any) map[string]any {
	providerConfig := make(map[string]any)

	// OpenAI-specific fields
	if apiKey := e.GetStringDefault(config, "api_key", ""); apiKey != "" {
		providerConfig["api_key"] = apiKey
	}
	if baseURL := e.GetStringDefault(config, "base_url", ""); baseURL != "" {
		providerConfig["base_url"] = baseURL
	}
	if orgID := e.GetStringDefault(config, "org_id", ""); orgID != "" {
		providerConfig["org_id"] = orgID
	}

	return providerConfig
}

// toStringSlice converts []any to []string.
func (e *LLMExecutor) toStringSlice(items []any) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// parseFiles parses file attachments configuration.
func (e *LLMExecutor) parseFiles(filesConfig []any) ([]models.LLMFileAttachment, error) {
	files := make([]models.LLMFileAttachment, 0, len(filesConfig))

	for i, fileConfig := range filesConfig {
		fileMap, ok := fileConfig.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("file %d is not a valid object", i)
		}

		data, _ := fileMap["data"].(string)
		mimeType, _ := fileMap["mime_type"].(string)
		name, _ := fileMap["name"].(string)
		detail, _ := fileMap["detail"].(string)

		if data == "" {
			return nil, fmt.Errorf("file %d: data is required", i)
		}
		if mimeType == "" {
			return nil, fmt.Errorf("file %d: mime_type is required", i)
		}

		file := models.LLMFileAttachment{
			Data:     data,
			MimeType: mimeType,
			Name:     name,
			Detail:   detail,
		}

		if !file.IsSupported() {
			return nil, fmt.Errorf("file %d: unsupported mime_type %s (supported: image/jpeg, image/png, image/gif, image/webp, application/pdf)", i, mimeType)
		}

		files = append(files, file)
	}

	return files, nil
}

// Helper function to convert response to JSON for debugging
func (e *LLMExecutor) toJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(data)
}

// parseHostedTools parses hosted tools configuration (Responses API).
func (e *LLMExecutor) parseHostedTools(toolsConfig []any) ([]models.LLMHostedTool, error) {
	tools := make([]models.LLMHostedTool, 0, len(toolsConfig))

	for i, toolConfig := range toolsConfig {
		toolMap, ok := toolConfig.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("hosted tool %d is not a valid object", i)
		}

		toolType, _ := toolMap["type"].(string)
		if toolType == "" {
			return nil, fmt.Errorf("hosted tool %d: missing type", i)
		}

		tool := models.LLMHostedTool{
			Type: toolType,
		}

		switch toolType {
		case "web_search_preview":
			if domains, ok := toolMap["domains"].([]any); ok {
				tool.Domains = e.toStringSlice(domains)
			}
			if contextSize, ok := toolMap["search_context_size"].(string); ok {
				tool.SearchContextSize = contextSize
			}
		case "file_search":
			if vectorStoreIDs, ok := toolMap["vector_store_ids"].([]any); ok {
				tool.VectorStoreIDs = e.toStringSlice(vectorStoreIDs)
			}
			if maxResults, ok := toolMap["max_num_results"].(int); ok {
				tool.MaxNumResults = maxResults
			} else if maxResultsFloat, ok := toolMap["max_num_results"].(float64); ok {
				tool.MaxNumResults = int(maxResultsFloat)
			}
			if rankingOptions, ok := toolMap["ranking_options"].(map[string]any); ok {
				tool.RankingOptions = rankingOptions
			}
		case "code_interpreter":
			// No additional config needed
		default:
			return nil, fmt.Errorf("hosted tool %d: unsupported type %s", i, toolType)
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

// parseToolCallConfig parses tool calling configuration.
func (e *LLMExecutor) parseToolCallConfig(config map[string]any) (*models.ToolCallConfig, error) {
	tc := models.DefaultToolCallConfig()

	if mode, ok := config["mode"].(string); ok {
		tc.Mode = models.ToolCallMode(mode)
	}
	// Backward compatibility: auto_execute_tools maps to mode
	if autoExecute, ok := config["auto_execute_tools"].(bool); ok && autoExecute {
		tc.Mode = models.ToolCallModeAuto
	}

	if maxIter, ok := config["max_iterations"].(float64); ok {
		tc.MaxIterations = int(maxIter)
	} else if maxIter, ok := config["max_iterations"].(int); ok {
		tc.MaxIterations = maxIter
	}

	if timeout, ok := config["timeout_per_tool"].(float64); ok {
		tc.TimeoutPerTool = int(timeout)
	} else if timeout, ok := config["timeout_per_tool"].(int); ok {
		tc.TimeoutPerTool = timeout
	}

	if totalTimeout, ok := config["total_timeout"].(float64); ok {
		tc.TotalTimeout = int(totalTimeout)
	} else if totalTimeout, ok := config["total_timeout"].(int); ok {
		tc.TotalTimeout = totalTimeout
	}

	if stopOnFailure, ok := config["stop_on_tool_failure"].(bool); ok {
		tc.StopOnToolFailure = stopOnFailure
	}

	return tc, nil
}

// parseMessages parses conversation messages.
func (e *LLMExecutor) parseMessages(messagesConfig []any) ([]models.LLMMessage, error) {
	messages := make([]models.LLMMessage, 0, len(messagesConfig))

	for i, msgConfig := range messagesConfig {
		msgMap, ok := msgConfig.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("message %d is not a valid object", i)
		}

		msg := models.LLMMessage{}

		if role, ok := msgMap["role"].(string); ok {
			msg.Role = role
		}
		if content, ok := msgMap["content"].(string); ok {
			msg.Content = content
		}
		if toolCallID, ok := msgMap["tool_call_id"].(string); ok {
			msg.ToolCallID = toolCallID
		}
		if name, ok := msgMap["name"].(string); ok {
			msg.Name = name
		}

		// Parse tool calls if present
		if toolCalls, ok := msgMap["tool_calls"].([]any); ok {
			for _, tc := range toolCalls {
				tcMap, ok := tc.(map[string]any)
				if !ok {
					continue
				}
				toolCall := models.LLMToolCall{}
				if id, ok := tcMap["id"].(string); ok {
					toolCall.ID = id
				}
				if typ, ok := tcMap["type"].(string); ok {
					toolCall.Type = typ
				}
				if funcMap, ok := tcMap["function"].(map[string]any); ok {
					if name, ok := funcMap["name"].(string); ok {
						toolCall.Function.Name = name
					}
					if args, ok := funcMap["arguments"].(string); ok {
						toolCall.Function.Arguments = args
					}
				}
				msg.ToolCalls = append(msg.ToolCalls, toolCall)
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// parseFunctions parses extended function definitions.
func (e *LLMExecutor) parseFunctions(functionsConfig []any) ([]models.FunctionDefinition, error) {
	functions := make([]models.FunctionDefinition, 0, len(functionsConfig))

	for i, funcConfig := range functionsConfig {
		funcMap, ok := funcConfig.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("function %d is not a valid object", i)
		}

		funcDef := models.FunctionDefinition{}

		if typ, ok := funcMap["type"].(string); ok {
			funcDef.Type = models.FunctionType(typ)
		}
		if name, ok := funcMap["name"].(string); ok {
			funcDef.Name = name
		}
		if desc, ok := funcMap["description"].(string); ok {
			funcDef.Description = desc
		}
		if params, ok := funcMap["parameters"].(map[string]any); ok {
			funcDef.Parameters = params
		}

		// Type-specific fields
		if builtinName, ok := funcMap["builtin_name"].(string); ok {
			funcDef.BuiltinName = builtinName
		}
		if workflowID, ok := funcMap["workflow_id"].(string); ok {
			funcDef.WorkflowID = workflowID
		}
		if inputMapping, ok := funcMap["input_mapping"].(map[string]any); ok {
			funcDef.InputMapping = make(map[string]string)
			for k, v := range inputMapping {
				if str, ok := v.(string); ok {
					funcDef.InputMapping[k] = str
				}
			}
		}
		if outputExtractor, ok := funcMap["output_extractor"].(string); ok {
			funcDef.OutputExtractor = outputExtractor
		}
		if language, ok := funcMap["language"].(string); ok {
			funcDef.Language = language
		}
		if code, ok := funcMap["code"].(string); ok {
			funcDef.Code = code
		}
		if openAPISpec, ok := funcMap["openapi_spec"].(string); ok {
			funcDef.OpenAPISpec = openAPISpec
		}
		if operationID, ok := funcMap["operation_id"].(string); ok {
			funcDef.OperationID = operationID
		}
		if baseURL, ok := funcMap["base_url"].(string); ok {
			funcDef.BaseURL = baseURL
		}
		if authConfig, ok := funcMap["auth_config"].(map[string]any); ok {
			funcDef.AuthConfig = authConfig
		}

		functions = append(functions, funcDef)
	}

	return functions, nil
}

// executeWithToolCalling executes LLM with automatic tool calling cycle.
func (e *LLMExecutor) executeWithToolCalling(
	ctx context.Context,
	req *models.LLMRequest,
	provider LLMProvider,
) (*models.LLMResponse, error) {
	// Check if tool calling registry is configured
	e.mu.RLock()
	registry := e.toolCallingRegistry
	e.mu.RUnlock()

	if registry == nil {
		return nil, fmt.Errorf("tool calling registry not configured")
	}

	// Initialize configuration
	config := req.ToolCallConfig
	if config == nil {
		config = models.DefaultToolCallConfig()
	}

	maxIterations := config.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 10
	}

	// Initialize messages
	messages := make([]models.LLMMessage, 0)
	if len(req.Messages) > 0 {
		messages = append(messages, req.Messages...)
	} else {
		// Create initial messages from prompt
		if req.Instruction != "" {
			messages = append(messages, models.LLMMessage{
				Role:    "system",
				Content: req.Instruction,
			})
		}
		messages = append(messages, models.LLMMessage{
			Role:    "user",
			Content: req.Prompt,
		})
	}

	// Convert functions to tools for LLM
	if len(req.Functions) > 0 {
		tools, err := e.convertFunctionsToTools(req.Functions)
		if err != nil {
			return nil, fmt.Errorf("failed to convert functions to tools: %w", err)
		}
		req.Tools = tools
	}

	allToolExecutions := make([]models.ToolExecutionResult, 0)
	var lastResponse *models.LLMResponse

	// Main tool calling loop
	for iteration := 0; iteration < maxIterations; iteration++ {
		// Update request with current messages
		reqCopy := *req
		reqCopy.Messages = messages

		// Call LLM
		response, err := provider.Execute(ctx, &reqCopy)
		if err != nil {
			return nil, fmt.Errorf("LLM call failed at iteration %d: %w", iteration, err)
		}

		lastResponse = response

		// Add assistant message to history
		assistantMsg := models.LLMMessage{
			Role:      "assistant",
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// Check finish reason
		if response.FinishReason == "stop" || len(response.ToolCalls) == 0 {
			// LLM finished - return result
			return &models.LLMResponse{
				Content:         response.Content,
				ResponseID:      response.ResponseID,
				Model:           response.Model,
				Usage:           response.Usage,
				FinishReason:    response.FinishReason,
				ToolCalls:       response.ToolCalls,
				CreatedAt:       response.CreatedAt,
				Metadata:        response.Metadata,
				Messages:        messages,
				ToolExecutions:  allToolExecutions,
				TotalIterations: iteration + 1,
				StoppedReason:   "finish",
			}, nil
		}

		// Execute tool calls
		if len(response.ToolCalls) > 0 {
			toolResults, err := e.executeToolCallsWithRegistry(ctx, response.ToolCalls, req.Functions, registry)
			if err != nil {
				if config.StopOnToolFailure {
					return nil, fmt.Errorf("tool execution failed at iteration %d: %w", iteration, err)
				}
				// Continue with error - add error message to conversation
			}

			// Check for tool execution errors
			if config.StopOnToolFailure {
				for _, result := range toolResults {
					if result.Error != "" {
						return nil, fmt.Errorf("tool execution failed at iteration %d: %s (function: %s)",
							iteration, result.Error, result.FunctionName)
					}
				}
			}

			// Add tool results to messages
			for _, result := range toolResults {
				toolMsg := models.LLMMessage{
					Role:       "tool",
					ToolCallID: result.ToolCallID,
					Name:       result.FunctionName,
					Content:    e.formatToolResult(result),
				}
				messages = append(messages, toolMsg)
			}

			allToolExecutions = append(allToolExecutions, toolResults...)
		}
	}

	// Max iterations reached
	if lastResponse == nil {
		return nil, fmt.Errorf("no response from LLM")
	}

	return &models.LLMResponse{
		Content:         lastResponse.Content,
		ResponseID:      lastResponse.ResponseID,
		Model:           lastResponse.Model,
		Usage:           lastResponse.Usage,
		FinishReason:    lastResponse.FinishReason,
		Messages:        messages,
		ToolExecutions:  allToolExecutions,
		TotalIterations: maxIterations,
		StoppedReason:   "max_iterations",
		Metadata:        lastResponse.Metadata,
	}, nil
}

// executeToolCallsWithRegistry executes tool calls using the registry.
func (e *LLMExecutor) executeToolCallsWithRegistry(
	ctx context.Context,
	toolCalls []models.LLMToolCall,
	functions []models.FunctionDefinition,
	registry *ToolCallingRegistry,
) ([]models.ToolExecutionResult, error) {
	results := make([]models.ToolExecutionResult, len(toolCalls))

	for i, toolCall := range toolCalls {
		startTime := time.Now()

		// Find function definition
		funcDef, err := e.findFunctionByName(toolCall.Function.Name, functions)
		if err != nil {
			results[i] = models.ToolExecutionResult{
				ToolCallID:    toolCall.ID,
				FunctionName:  toolCall.Function.Name,
				Error:         err.Error(),
				ExecutionTime: time.Since(startTime).Milliseconds(),
			}
			continue
		}

		// Execute function through registry
		result, err := registry.ExecuteFunction(ctx, funcDef, toolCall.Function.Arguments)

		executionTime := time.Since(startTime).Milliseconds()

		if err != nil {
			results[i] = models.ToolExecutionResult{
				ToolCallID:    toolCall.ID,
				FunctionName:  toolCall.Function.Name,
				Error:         err.Error(),
				ExecutionTime: executionTime,
			}
		} else {
			results[i] = models.ToolExecutionResult{
				ToolCallID:    toolCall.ID,
				FunctionName:  toolCall.Function.Name,
				Result:        result,
				ExecutionTime: executionTime,
			}
		}
	}

	return results, nil
}

// convertFunctionsToTools converts FunctionDefinitions to LLMTools.
func (e *LLMExecutor) convertFunctionsToTools(functions []models.FunctionDefinition) ([]models.LLMTool, error) {
	tools := make([]models.LLMTool, len(functions))

	for i, funcDef := range functions {
		tools[i] = models.LLMTool{
			Type: "function",
			Function: models.LLMFunctionTool{
				Name:        funcDef.Name,
				Description: funcDef.Description,
				Parameters:  funcDef.Parameters,
			},
		}
	}

	return tools, nil
}

// findFunctionByName finds a function definition by name.
func (e *LLMExecutor) findFunctionByName(name string, functions []models.FunctionDefinition) (*models.FunctionDefinition, error) {
	for i := range functions {
		if functions[i].Name == name {
			return &functions[i], nil
		}
	}
	return nil, fmt.Errorf("function not found: %s", name)
}

// formatToolResult formats a tool execution result for LLM consumption.
func (e *LLMExecutor) formatToolResult(result models.ToolExecutionResult) string {
	if result.Error != "" {
		return fmt.Sprintf("Error: %s", result.Error)
	}

	// Convert result to JSON string
	resultJSON, err := json.Marshal(result.Result)
	if err != nil {
		return fmt.Sprintf("Error formatting result: %v", err)
	}

	return string(resultJSON)
}
