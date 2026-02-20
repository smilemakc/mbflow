package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== ValidateHTTPConfig Tests ====================

func TestValidateHTTPConfig_Success(t *testing.T) {
	config := map[string]any{
		"method": "GET",
		"url":    "https://api.example.com",
	}

	err := ValidateHTTPConfig(config)
	assert.NoError(t, err)
}

func TestValidateHTTPConfig_WithAdditionalFields(t *testing.T) {
	config := map[string]any{
		"method":  "POST",
		"url":     "https://api.example.com",
		"headers": map[string]string{"Content-Type": "application/json"},
		"body":    `{"key": "value"}`,
	}

	err := ValidateHTTPConfig(config)
	assert.NoError(t, err)
}

func TestValidateHTTPConfig_MissingMethod(t *testing.T) {
	config := map[string]any{
		"url": "https://api.example.com",
	}

	err := ValidateHTTPConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 'method' field")
}

func TestValidateHTTPConfig_MissingURL(t *testing.T) {
	config := map[string]any{
		"method": "GET",
	}

	err := ValidateHTTPConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 'url' field")
}

func TestValidateHTTPConfig_EmptyConfig(t *testing.T) {
	config := map[string]any{}

	err := ValidateHTTPConfig(config)
	assert.Error(t, err)
}

// ==================== ValidateLLMConfig Tests ====================

func TestValidateLLMConfig_Success(t *testing.T) {
	config := map[string]any{
		"provider": "openai",
		"model":    "gpt-4",
		"prompt":   "Hello",
		"api_key":  "sk-test",
	}

	err := ValidateLLMConfig(config)
	assert.NoError(t, err)
}

func TestValidateLLMConfig_WithOptionalFields(t *testing.T) {
	config := map[string]any{
		"provider":    "anthropic",
		"model":       "claude-3",
		"prompt":      "Hello",
		"api_key":     "sk-test",
		"temperature": 0.7,
		"top_p":       0.9,
		"max_tokens":  1000,
	}

	err := ValidateLLMConfig(config)
	assert.NoError(t, err)
}

func TestValidateLLMConfig_MissingProvider(t *testing.T) {
	config := map[string]any{
		"model":   "gpt-4",
		"prompt":  "Hello",
		"api_key": "sk-test",
	}

	err := ValidateLLMConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 'provider' field")
}

func TestValidateLLMConfig_MissingModel(t *testing.T) {
	config := map[string]any{
		"provider": "openai",
		"prompt":   "Hello",
		"api_key":  "sk-test",
	}

	err := ValidateLLMConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 'model' field")
}

func TestValidateLLMConfig_MissingPrompt(t *testing.T) {
	config := map[string]any{
		"provider": "openai",
		"model":    "gpt-4",
		"api_key":  "sk-test",
	}

	err := ValidateLLMConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 'prompt' field")
}

func TestValidateLLMConfig_MissingAPIKey(t *testing.T) {
	config := map[string]any{
		"provider": "openai",
		"model":    "gpt-4",
		"prompt":   "Hello",
	}

	err := ValidateLLMConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 'api_key' field")
}

func TestValidateLLMConfig_InvalidTemperatureTooLow(t *testing.T) {
	config := map[string]any{
		"provider":    "openai",
		"model":       "gpt-4",
		"prompt":      "Hello",
		"api_key":     "sk-test",
		"temperature": -0.5,
	}

	err := ValidateLLMConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "temperature must be between 0 and 2")
}

func TestValidateLLMConfig_InvalidTemperatureTooHigh(t *testing.T) {
	config := map[string]any{
		"provider":    "openai",
		"model":       "gpt-4",
		"prompt":      "Hello",
		"api_key":     "sk-test",
		"temperature": 2.5,
	}

	err := ValidateLLMConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "temperature must be between 0 and 2")
}

func TestValidateLLMConfig_InvalidTopPTooLow(t *testing.T) {
	config := map[string]any{
		"provider": "openai",
		"model":    "gpt-4",
		"prompt":   "Hello",
		"api_key":  "sk-test",
		"top_p":    -0.1,
	}

	err := ValidateLLMConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "top_p must be between 0 and 1")
}

func TestValidateLLMConfig_InvalidTopPTooHigh(t *testing.T) {
	config := map[string]any{
		"provider": "openai",
		"model":    "gpt-4",
		"prompt":   "Hello",
		"api_key":  "sk-test",
		"top_p":    1.5,
	}

	err := ValidateLLMConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "top_p must be between 0 and 1")
}

func TestValidateLLMConfig_InvalidMaxTokensNegative(t *testing.T) {
	config := map[string]any{
		"provider":   "openai",
		"model":      "gpt-4",
		"prompt":     "Hello",
		"api_key":    "sk-test",
		"max_tokens": -100,
	}

	err := ValidateLLMConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_tokens must be >= 0")
}

func TestValidateLLMConfig_BoundaryTemperature(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		shouldPass  bool
	}{
		{"zero temperature", 0.0, true},
		{"max temperature", 2.0, true},
		{"mid temperature", 1.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]any{
				"provider":    "openai",
				"model":       "gpt-4",
				"prompt":      "Hello",
				"api_key":     "sk-test",
				"temperature": tt.temperature,
			}

			err := ValidateLLMConfig(config)
			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// ==================== ValidateTransformConfig Tests ====================

func TestValidateTransformConfig_Passthrough(t *testing.T) {
	config := map[string]any{
		"type": "passthrough",
	}

	err := ValidateTransformConfig(config)
	assert.NoError(t, err)
}

func TestValidateTransformConfig_Expression(t *testing.T) {
	config := map[string]any{
		"type":       "expression",
		"expression": "input.value * 2",
	}

	err := ValidateTransformConfig(config)
	assert.NoError(t, err)
}

func TestValidateTransformConfig_JQ(t *testing.T) {
	config := map[string]any{
		"type":   "jq",
		"filter": ".data | select(.active)",
	}

	err := ValidateTransformConfig(config)
	assert.NoError(t, err)
}

func TestValidateTransformConfig_Template(t *testing.T) {
	config := map[string]any{
		"type":     "template",
		"template": "Hello {{.name}}",
	}

	err := ValidateTransformConfig(config)
	assert.NoError(t, err)
}

func TestValidateTransformConfig_MissingType(t *testing.T) {
	config := map[string]any{}

	err := ValidateTransformConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 'type' field")
}

func TestValidateTransformConfig_TypeNotString(t *testing.T) {
	config := map[string]any{
		"type": 123,
	}

	err := ValidateTransformConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'type' must be a string")
}

func TestValidateTransformConfig_InvalidType(t *testing.T) {
	config := map[string]any{
		"type": "unknown_type",
	}

	err := ValidateTransformConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transform type")
}

func TestValidateTransformConfig_ExpressionMissingExpression(t *testing.T) {
	config := map[string]any{
		"type": "expression",
	}

	err := ValidateTransformConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Expression transform requires 'expression' field")
}

func TestValidateTransformConfig_JQMissingFilter(t *testing.T) {
	config := map[string]any{
		"type": "jq",
	}

	err := ValidateTransformConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JQ transform requires 'filter' field")
}

func TestValidateTransformConfig_TemplateMissingTemplate(t *testing.T) {
	config := map[string]any{
		"type": "template",
	}

	err := ValidateTransformConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Template transform requires 'template' field")
}

// ==================== ValidateNodeConfig Tests ====================

func TestValidateNodeConfig_HTTP(t *testing.T) {
	config := map[string]any{
		"method": "GET",
		"url":    "https://api.example.com",
	}

	err := ValidateNodeConfig("http", config)
	assert.NoError(t, err)
}

func TestValidateNodeConfig_LLM(t *testing.T) {
	config := map[string]any{
		"provider": "openai",
		"model":    "gpt-4",
		"prompt":   "Hello",
		"api_key":  "sk-test",
	}

	err := ValidateNodeConfig("llm", config)
	assert.NoError(t, err)
}

func TestValidateNodeConfig_Transform(t *testing.T) {
	config := map[string]any{
		"type": "passthrough",
	}

	err := ValidateNodeConfig("transform", config)
	assert.NoError(t, err)
}

func TestValidateNodeConfig_UnknownType(t *testing.T) {
	config := map[string]any{
		"custom_field": "value",
	}

	// Unknown types should not error (may be custom executors)
	err := ValidateNodeConfig("custom_executor", config)
	assert.NoError(t, err)
}

func TestValidateNodeConfig_HTTPInvalid(t *testing.T) {
	config := map[string]any{
		"method": "GET",
		// Missing URL
	}

	err := ValidateNodeConfig("http", config)
	assert.Error(t, err)
}

func TestValidateNodeConfig_LLMInvalid(t *testing.T) {
	config := map[string]any{
		"provider": "openai",
		// Missing model, prompt, api_key
	}

	err := ValidateNodeConfig("llm", config)
	assert.Error(t, err)
}

func TestValidateNodeConfig_TransformInvalid(t *testing.T) {
	config := map[string]any{
		"type": "expression",
		// Missing expression
	}

	err := ValidateNodeConfig("transform", config)
	assert.Error(t, err)
}
