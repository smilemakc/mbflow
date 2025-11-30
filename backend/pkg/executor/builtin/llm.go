package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
	providers map[models.LLMProvider]LLMProvider
	mu        sync.RWMutex
}

// NewLLMExecutor creates a new LLM executor.
func NewLLMExecutor() *LLMExecutor {
	return &LLMExecutor{
		BaseExecutor: executor.NewBaseExecutor("llm"),
		providers:    make(map[models.LLMProvider]LLMProvider),
	}
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
// The 'input' parameter contains raw parent node output but is typically not used
// directly. Instead, use templates to extract specific fields from parent output.
//
// See: backend/internal/application/engine/node_executor.go for implementation details.
func (e *LLMExecutor) Execute(ctx context.Context, config map[string]interface{}, input interface{}) (interface{}, error) {
	// Parse config into LLMRequest
	req, err := e.parseConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM config: %w", err)
	}

	// Create provider with config
	provider, err := e.getOrCreateProvider(req)
	if err != nil {
		return nil, err
	}

	// Execute request
	response, err := provider.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM execution failed: %w", err)
	}

	// Convert response to map for output
	return e.responseToMap(response), nil
}

// Validate validates the LLM executor configuration.
func (e *LLMExecutor) Validate(config map[string]interface{}) error {
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
		models.LLMProviderOpenAI:    true,
		models.LLMProviderAnthropic: true,
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
	if responseFormat, ok := config["response_format"].(map[string]interface{}); ok {
		if err := e.validateResponseFormat(responseFormat); err != nil {
			return err
		}
	}

	// Validate tools if present
	if tools, ok := config["tools"].([]interface{}); ok {
		if err := e.validateTools(tools); err != nil {
			return err
		}
	}

	return nil
}

// parseConfig parses the executor config into an LLMRequest.
func (e *LLMExecutor) parseConfig(config map[string]interface{}) (*models.LLMRequest, error) {
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
	if imageURLs, ok := config["image_url"].([]interface{}); ok {
		req.ImageURLs = e.toStringSlice(imageURLs)
	}
	if imageIDs, ok := config["image_id"].([]interface{}); ok {
		req.ImageIDs = e.toStringSlice(imageIDs)
	}
	if fileIDs, ok := config["file_id"].([]interface{}); ok {
		req.FileIDs = e.toStringSlice(fileIDs)
	}
	if stopSeqs, ok := config["stop_sequences"].([]interface{}); ok {
		req.StopSequences = e.toStringSlice(stopSeqs)
	}

	// Tools
	if tools, ok := config["tools"].([]interface{}); ok {
		parsedTools, err := e.parseTools(tools)
		if err != nil {
			return nil, err
		}
		req.Tools = parsedTools
	}

	// Response format
	if responseFormat, ok := config["response_format"].(map[string]interface{}); ok {
		parsedFormat, err := e.parseResponseFormat(responseFormat)
		if err != nil {
			return nil, err
		}
		req.ResponseFormat = parsedFormat
	}

	// Extract provider configuration
	req.ProviderConfig = e.extractProviderConfig(config)

	return req, nil
}

// parseTools parses tools configuration into LLMTool structs.
func (e *LLMExecutor) parseTools(toolsConfig []interface{}) ([]models.LLMTool, error) {
	tools := make([]models.LLMTool, len(toolsConfig))

	for i, toolConfig := range toolsConfig {
		toolMap, ok := toolConfig.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("tool %d is not a valid object", i)
		}

		toolType, _ := toolMap["type"].(string)
		if toolType != "function" {
			return nil, fmt.Errorf("tool %d: only 'function' type is supported", i)
		}

		funcConfig, ok := toolMap["function"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("tool %d: missing function definition", i)
		}

		name, _ := funcConfig["name"].(string)
		description, _ := funcConfig["description"].(string)
		params, _ := funcConfig["parameters"].(map[string]interface{})

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
func (e *LLMExecutor) parseResponseFormat(formatConfig map[string]interface{}) (*models.LLMResponseFormat, error) {
	formatType, _ := formatConfig["type"].(string)
	if formatType == "" {
		return nil, fmt.Errorf("response_format type is required")
	}

	format := &models.LLMResponseFormat{
		Type: formatType,
	}

	// Parse JSON schema if present
	if formatType == "json_schema" {
		schemaConfig, ok := formatConfig["json_schema"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("json_schema is required for json_schema type")
		}

		name, _ := schemaConfig["name"].(string)
		description, _ := schemaConfig["description"].(string)
		schema, _ := schemaConfig["schema"].(map[string]interface{})
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
func (e *LLMExecutor) validateResponseFormat(formatConfig map[string]interface{}) error {
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
		schemaConfig, ok := formatConfig["json_schema"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("json_schema is required for json_schema type")
		}

		name, ok := schemaConfig["name"].(string)
		if !ok || name == "" {
			return fmt.Errorf("json_schema name is required")
		}

		if _, ok := schemaConfig["schema"].(map[string]interface{}); !ok {
			return fmt.Errorf("json_schema schema is required")
		}
	}

	return nil
}

// validateTools validates tools configuration.
func (e *LLMExecutor) validateTools(toolsConfig []interface{}) error {
	for i, toolConfig := range toolsConfig {
		toolMap, ok := toolConfig.(map[string]interface{})
		if !ok {
			return fmt.Errorf("tool %d is not a valid object", i)
		}

		toolType, ok := toolMap["type"].(string)
		if !ok || toolType != "function" {
			return fmt.Errorf("tool %d: only 'function' type is supported", i)
		}

		funcConfig, ok := toolMap["function"].(map[string]interface{})
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
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}
}

// responseToMap converts LLMResponse to a map for output.
func (e *LLMExecutor) responseToMap(response *models.LLMResponse) map[string]interface{} {
	result := map[string]interface{}{
		"content":       response.Content,
		"response_id":   response.ResponseID,
		"model":         response.Model,
		"finish_reason": response.FinishReason,
		"created_at":    response.CreatedAt,
		"usage": map[string]interface{}{
			"prompt_tokens":     response.Usage.PromptTokens,
			"completion_tokens": response.Usage.CompletionTokens,
			"total_tokens":      response.Usage.TotalTokens,
		},
	}

	if len(response.ToolCalls) > 0 {
		toolCalls := make([]map[string]interface{}, len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			toolCalls[i] = map[string]interface{}{
				"id":   tc.ID,
				"type": tc.Type,
				"function": map[string]interface{}{
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

	return result
}

// extractProviderConfig extracts provider-specific configuration from the node config.
func (e *LLMExecutor) extractProviderConfig(config map[string]interface{}) map[string]interface{} {
	providerConfig := make(map[string]interface{})

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

// toStringSlice converts []interface{} to []string.
func (e *LLMExecutor) toStringSlice(items []interface{}) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// Helper function to convert response to JSON for debugging
func (e *LLMExecutor) toJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(data)
}
