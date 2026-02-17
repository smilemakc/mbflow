package builder

import (
	"testing"
	"time"

	"github.com/smilemakc/mbflow/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Generic NodeBuilder Tests ====================

func TestNewNode_Success(t *testing.T) {
	node, err := NewNode("test-node", "http", "Test Node").Build()

	require.NoError(t, err)
	assert.Equal(t, "test-node", node.ID)
	assert.Equal(t, "http", node.Type)
	assert.Equal(t, "Test Node", node.Name)
	assert.NotNil(t, node.Config)
	assert.NotNil(t, node.Metadata)
}

func TestNodeBuilder_WithNodeDescription(t *testing.T) {
	node, err := NewNode("test-node", "http", "Test Node",
		WithNodeDescription("This is a test node"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "This is a test node", node.Description)
}

func TestNodeBuilder_WithPosition(t *testing.T) {
	node, err := NewNode("test-node", "http", "Test Node",
		WithPosition(100.5, 200.7),
	).Build()

	require.NoError(t, err)
	require.NotNil(t, node.Position)
	assert.Equal(t, 100.5, node.Position.X)
	assert.Equal(t, 200.7, node.Position.Y)
}

func TestNodeBuilder_GridPosition_Success(t *testing.T) {
	tests := []struct {
		name      string
		row       int
		col       int
		expectedX float64
		expectedY float64
	}{
		{"origin", 0, 0, 0, 0},
		{"row 1 col 1", 1, 1, 200, 200},
		{"row 2 col 3", 2, 3, 600, 400},
		{"large grid", 10, 5, 1000, 2000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode("test-node", "http", "Test Node",
				GridPosition(tt.row, tt.col),
			).Build()

			require.NoError(t, err)
			require.NotNil(t, node.Position)
			assert.Equal(t, tt.expectedX, node.Position.X)
			assert.Equal(t, tt.expectedY, node.Position.Y)
		})
	}
}

func TestNodeBuilder_GridPosition_NegativeValues(t *testing.T) {
	tests := []struct {
		name string
		row  int
		col  int
	}{
		{"negative row", -1, 0},
		{"negative col", 0, -1},
		{"both negative", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode("test-node", "http", "Test Node",
				GridPosition(tt.row, tt.col),
			).Build()

			assert.Error(t, err)
			assert.Nil(t, node)
			assert.Contains(t, err.Error(), "grid position row and col must be non-negative")
		})
	}
}

func TestNodeBuilder_WithNodeMetadata_Success(t *testing.T) {
	node, err := NewNode("test-node", "http", "Test Node",
		WithNodeMetadata("key1", "value1"),
		WithNodeMetadata("key2", 42),
		WithNodeMetadata("key3", map[string]string{"nested": "value"}),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "value1", node.Metadata["key1"])
	assert.Equal(t, 42, node.Metadata["key2"])
	assert.NotNil(t, node.Metadata["key3"])
}

func TestNodeBuilder_WithNodeMetadata_EmptyKey(t *testing.T) {
	node, err := NewNode("test-node", "http", "Test Node",
		WithNodeMetadata("", "value"),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "metadata key cannot be empty")
}

func TestNodeBuilder_WithConfig(t *testing.T) {
	config := map[string]any{
		"method": "GET",
		"url":    "https://api.example.com",
	}

	node, err := NewNode("test-node", "http", "Test Node",
		WithConfig(config),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "GET", node.Config["method"])
	assert.Equal(t, "https://api.example.com", node.Config["url"])
}

func TestNodeBuilder_WithConfigValue_Success(t *testing.T) {
	node, err := NewNode("test-node", "http", "Test Node",
		WithConfigValue("custom_field", "custom_value"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "custom_value", node.Config["custom_field"])
}

func TestNodeBuilder_WithConfigValue_EmptyKey(t *testing.T) {
	node, err := NewNode("test-node", "http", "Test Node",
		WithConfigValue("", "value"),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "config key cannot be empty")
}

// ==================== HTTP Node Tests ====================

func TestHTTPMethod_AllValidMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			node, err := NewNode("http-node", "http", "HTTP Node",
				HTTPMethod(method),
				HTTPURL("https://api.example.com"),
			).Build()

			require.NoError(t, err)
			assert.Equal(t, method, node.Config["method"])
		})
	}
}

func TestHTTPMethod_LowercaseConversion(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("get"),
		HTTPURL("https://api.example.com"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "GET", node.Config["method"])
}

func TestHTTPMethod_InvalidMethod(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("INVALID"),
		HTTPURL("https://api.example.com"),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "invalid HTTP method")
}

func TestHTTPURL_Success(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL("https://api.example.com/v1/users"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/v1/users", node.Config["url"])
}

func TestHTTPURL_EmptyURL(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL(""),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "HTTP URL cannot be empty")
}

func TestHTTPBody_Success(t *testing.T) {
	body := map[string]any{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("POST"),
		HTTPURL("https://api.example.com/users"),
		HTTPBody(body),
	).Build()

	require.NoError(t, err)
	bodyConfig, ok := node.Config["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "John Doe", bodyConfig["name"])
	assert.Equal(t, "john@example.com", bodyConfig["email"])
}

func TestHTTPHeaders_Success(t *testing.T) {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer token123",
	}

	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL("https://api.example.com"),
		HTTPHeaders(headers),
	).Build()

	require.NoError(t, err)
	headerConfig, ok := node.Config["headers"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "application/json", headerConfig["Content-Type"])
	assert.Equal(t, "Bearer token123", headerConfig["Authorization"])
}

func TestHTTPHeader_Single(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL("https://api.example.com"),
		HTTPHeader("Content-Type", "application/json"),
	).Build()

	require.NoError(t, err)
	headerConfig, ok := node.Config["headers"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "application/json", headerConfig["Content-Type"])
}

func TestHTTPHeader_Multiple(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL("https://api.example.com"),
		HTTPHeader("Content-Type", "application/json"),
		HTTPHeader("Authorization", "Bearer token123"),
		HTTPHeader("X-Custom-Header", "custom-value"),
	).Build()

	require.NoError(t, err)
	headerConfig, ok := node.Config["headers"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "application/json", headerConfig["Content-Type"])
	assert.Equal(t, "Bearer token123", headerConfig["Authorization"])
	assert.Equal(t, "custom-value", headerConfig["X-Custom-Header"])
}

func TestHTTPHeader_EmptyKey(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL("https://api.example.com"),
		HTTPHeader("", "value"),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "header key cannot be empty")
}

func TestHTTPTimeout_Success(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL("https://api.example.com"),
		HTTPTimeout(30*time.Second),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "30s", node.Config["timeout"])
}

func TestHTTPTimeout_ZeroOrNegative(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"zero timeout", 0},
		{"negative timeout", -5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode("http-node", "http", "HTTP Node",
				HTTPMethod("GET"),
				HTTPURL("https://api.example.com"),
				HTTPTimeout(tt.timeout),
			).Build()

			assert.Error(t, err)
			assert.Nil(t, node)
			assert.Contains(t, err.Error(), "timeout must be positive")
		})
	}
}

func TestHTTPQueryParam_Single(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL("https://api.example.com"),
		HTTPQueryParam("page", "1"),
	).Build()

	require.NoError(t, err)
	params, ok := node.Config["query_params"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "1", params["page"])
}

func TestHTTPQueryParam_Multiple(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL("https://api.example.com"),
		HTTPQueryParam("page", "1"),
		HTTPQueryParam("limit", "10"),
		HTTPQueryParam("sort", "name"),
	).Build()

	require.NoError(t, err)
	params, ok := node.Config["query_params"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "1", params["page"])
	assert.Equal(t, "10", params["limit"])
	assert.Equal(t, "name", params["sort"])
}

func TestHTTPQueryParam_EmptyKey(t *testing.T) {
	node, err := NewNode("http-node", "http", "HTTP Node",
		HTTPMethod("GET"),
		HTTPURL("https://api.example.com"),
		HTTPQueryParam("", "value"),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "query param key cannot be empty")
}

func TestNewHTTPGetNode_Success(t *testing.T) {
	node, err := NewHTTPGetNode("get-node", "GET Request", "https://api.example.com").Build()

	require.NoError(t, err)
	assert.Equal(t, "get-node", node.ID)
	assert.Equal(t, "GET Request", node.Name)
	assert.Equal(t, "http", node.Type)
	assert.Equal(t, "GET", node.Config["method"])
	assert.Equal(t, "https://api.example.com", node.Config["url"])
}

func TestNewHTTPPostNode_Success(t *testing.T) {
	body := map[string]any{"name": "test"}
	node, err := NewHTTPPostNode("post-node", "POST Request", "https://api.example.com", body).Build()

	require.NoError(t, err)
	assert.Equal(t, "POST", node.Config["method"])
	assert.NotNil(t, node.Config["body"])
}

func TestNewHTTPPutNode_Success(t *testing.T) {
	body := map[string]any{"id": 1, "name": "updated"}
	node, err := NewHTTPPutNode("put-node", "PUT Request", "https://api.example.com/1", body).Build()

	require.NoError(t, err)
	assert.Equal(t, "PUT", node.Config["method"])
	assert.NotNil(t, node.Config["body"])
}

func TestNewHTTPDeleteNode_Success(t *testing.T) {
	node, err := NewHTTPDeleteNode("delete-node", "DELETE Request", "https://api.example.com/1").Build()

	require.NoError(t, err)
	assert.Equal(t, "DELETE", node.Config["method"])
}

func TestNewHTTPPatchNode_Success(t *testing.T) {
	body := map[string]any{"status": "active"}
	node, err := NewHTTPPatchNode("patch-node", "PATCH Request", "https://api.example.com/1", body).Build()

	require.NoError(t, err)
	assert.Equal(t, "PATCH", node.Config["method"])
	assert.NotNil(t, node.Config["body"])
}

// ==================== Transform Node Tests ====================

func TestTransformType_AllValidTypes(t *testing.T) {
	types := []string{"passthrough", "expression", "jq", "template"}

	for _, ttype := range types {
		t.Run(ttype, func(t *testing.T) {
			node, err := NewNode("transform-node", "transform", "Transform Node",
				TransformType(ttype),
			).Build()

			require.NoError(t, err)
			assert.Equal(t, ttype, node.Config["type"])
		})
	}
}

func TestTransformType_InvalidType(t *testing.T) {
	node, err := NewNode("transform-node", "transform", "Transform Node",
		TransformType("invalid"),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "invalid transform type")
}

func TestTransformExpression_Success(t *testing.T) {
	node, err := NewNode("transform-node", "transform", "Transform Node",
		TransformType("expression"),
		TransformExpression("input.value * 2"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "input.value * 2", node.Config["expression"])
}

func TestTransformExpression_Empty(t *testing.T) {
	node, err := NewNode("transform-node", "transform", "Transform Node",
		TransformType("expression"),
		TransformExpression(""),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "expression cannot be empty")
}

func TestTransformJQ_Success(t *testing.T) {
	node, err := NewNode("transform-node", "transform", "Transform Node",
		TransformType("jq"),
		TransformJQ(".data | select(.active)"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, ".data | select(.active)", node.Config["filter"])
}

func TestTransformJQ_Empty(t *testing.T) {
	node, err := NewNode("transform-node", "transform", "Transform Node",
		TransformType("jq"),
		TransformJQ(""),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "JQ filter cannot be empty")
}

func TestTransformTemplate_Success(t *testing.T) {
	node, err := NewNode("transform-node", "transform", "Transform Node",
		TransformType("template"),
		TransformTemplate("Hello {{input.name}}"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "Hello {{input.name}}", node.Config["template"])
}

func TestTransformTemplate_Empty(t *testing.T) {
	node, err := NewNode("transform-node", "transform", "Transform Node",
		TransformType("template"),
		TransformTemplate(""),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "template cannot be empty")
}

func TestTransformMapping_Success(t *testing.T) {
	mapping := map[string]string{
		"outputField1": "input.field1",
		"outputField2": "input.field2",
	}

	node, err := NewNode("transform-node", "transform", "Transform Node",
		TransformType("passthrough"),
		TransformMapping(mapping),
	).Build()

	require.NoError(t, err)
	mappingConfig, ok := node.Config["mapping"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "input.field1", mappingConfig["outputField1"])
	assert.Equal(t, "input.field2", mappingConfig["outputField2"])
}

func TestNewPassthroughNode_Success(t *testing.T) {
	node, err := NewPassthroughNode("passthrough-node", "Passthrough").Build()

	require.NoError(t, err)
	assert.Equal(t, "passthrough-node", node.ID)
	assert.Equal(t, "Passthrough", node.Name)
	assert.Equal(t, "transform", node.Type)
	assert.Equal(t, "passthrough", node.Config["type"])
}

func TestNewExpressionNode_Success(t *testing.T) {
	node, err := NewExpressionNode("expr-node", "Expression", "input.value * 2").Build()

	require.NoError(t, err)
	assert.Equal(t, "expression", node.Config["type"])
	assert.Equal(t, "input.value * 2", node.Config["expression"])
}

func TestNewJQNode_Success(t *testing.T) {
	node, err := NewJQNode("jq-node", "JQ Filter", ".data").Build()

	require.NoError(t, err)
	assert.Equal(t, "jq", node.Config["type"])
	assert.Equal(t, ".data", node.Config["filter"])
}

func TestNewTemplateNode_Success(t *testing.T) {
	node, err := NewTemplateNode("template-node", "Template", "Hello {{input.name}}").Build()

	require.NoError(t, err)
	assert.Equal(t, "template", node.Config["type"])
	assert.Equal(t, "Hello {{input.name}}", node.Config["template"])
}

func TestNewTransformNode_Generic(t *testing.T) {
	node, err := NewTransformNode("transform-node", "Transform",
		TransformType("passthrough"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "transform", node.Type)
	assert.Equal(t, "passthrough", node.Config["type"])
}

// ==================== LLM Node Tests ====================

func TestLLMProvider_OpenAI(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("Test prompt"),
		LLMAPIKey("sk-test"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "openai", node.Config["provider"])
}

func TestLLMProvider_Anthropic(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderAnthropic),
		LLMModel("claude-3"),
		LLMPrompt("Test prompt"),
		LLMAPIKey("sk-test"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "anthropic", node.Config["provider"])
}

func TestLLMProvider_Invalid(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider("invalid"),
		LLMModel("model"),
		LLMPrompt("prompt"),
		LLMAPIKey("key"),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "unsupported LLM provider")
}

func TestLLMModel_Success(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4-turbo"),
		LLMPrompt("Test"),
		LLMAPIKey("sk-test"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "gpt-4-turbo", node.Config["model"])
}

func TestLLMModel_Empty(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel(""),
		LLMPrompt("Test"),
		LLMAPIKey("sk-test"),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "model cannot be empty")
}

func TestLLMPrompt_Success(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("Explain quantum computing"),
		LLMAPIKey("sk-test"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "Explain quantum computing", node.Config["prompt"])
}

func TestLLMPrompt_Empty(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt(""),
		LLMAPIKey("sk-test"),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "prompt cannot be empty")
}

func TestLLMAPIKey_Success(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("Test"),
		LLMAPIKey("sk-test123"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "sk-test123", node.Config["api_key"])
}

func TestLLMAPIKey_Empty(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("Test"),
		LLMAPIKey(""),
	).Build()

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "API key cannot be empty")
}

func TestLLMTemperature_BoundaryValues(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		shouldPass  bool
	}{
		{"minimum (0)", 0.0, true},
		{"low valid", 0.5, true},
		{"medium", 1.0, true},
		{"high valid", 1.8, true},
		{"maximum (2)", 2.0, true},
		{"below minimum", -0.1, false},
		{"above maximum", 2.1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode("llm-node", "llm", "LLM Node",
				LLMProvider(models.LLMProviderOpenAI),
				LLMModel("gpt-4"),
				LLMPrompt("Test"),
				LLMAPIKey("sk-test"),
				LLMTemperature(tt.temperature),
			).Build()

			if tt.shouldPass {
				require.NoError(t, err)
				assert.Equal(t, tt.temperature, node.Config["temperature"])
			} else {
				assert.Error(t, err)
				assert.Nil(t, node)
				assert.Contains(t, err.Error(), "temperature must be between 0 and 2")
			}
		})
	}
}

func TestLLMMaxTokens_Success(t *testing.T) {
	tests := []struct {
		name       string
		maxTokens  int
		shouldPass bool
	}{
		{"zero", 0, true},
		{"small", 100, true},
		{"medium", 1000, true},
		{"large", 4096, true},
		{"negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode("llm-node", "llm", "LLM Node",
				LLMProvider(models.LLMProviderOpenAI),
				LLMModel("gpt-4"),
				LLMPrompt("Test"),
				LLMAPIKey("sk-test"),
				LLMMaxTokens(tt.maxTokens),
			).Build()

			if tt.shouldPass {
				require.NoError(t, err)
				assert.Equal(t, tt.maxTokens, node.Config["max_tokens"])
			} else {
				assert.Error(t, err)
				assert.Nil(t, node)
				assert.Contains(t, err.Error(), "max_tokens must be >= 0")
			}
		})
	}
}

func TestLLMTopP_BoundaryValues(t *testing.T) {
	tests := []struct {
		name       string
		topP       float64
		shouldPass bool
	}{
		{"minimum (0)", 0.0, true},
		{"low valid", 0.1, true},
		{"medium", 0.5, true},
		{"high valid", 0.9, true},
		{"maximum (1)", 1.0, true},
		{"below minimum", -0.1, false},
		{"above maximum", 1.1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode("llm-node", "llm", "LLM Node",
				LLMProvider(models.LLMProviderOpenAI),
				LLMModel("gpt-4"),
				LLMPrompt("Test"),
				LLMAPIKey("sk-test"),
				LLMTopP(tt.topP),
			).Build()

			if tt.shouldPass {
				require.NoError(t, err)
				assert.Equal(t, tt.topP, node.Config["top_p"])
			} else {
				assert.Error(t, err)
				assert.Nil(t, node)
				assert.Contains(t, err.Error(), "top_p must be between 0 and 1")
			}
		})
	}
}

func TestLLMSystemPrompt_Success(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("User prompt"),
		LLMAPIKey("sk-test"),
		LLMSystemPrompt("You are a helpful assistant"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "You are a helpful assistant", node.Config["system_prompt"])
}

func TestLLMTools_Success(t *testing.T) {
	tools := []map[string]any{
		{
			"name":        "get_weather",
			"description": "Get weather information",
		},
		{
			"name":        "search_db",
			"description": "Search database",
		},
	}

	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("Test"),
		LLMAPIKey("sk-test"),
		LLMTools(tools),
	).Build()

	require.NoError(t, err)
	toolsConfig, ok := node.Config["tools"].([]map[string]any)
	require.True(t, ok)
	assert.Len(t, toolsConfig, 2)
	assert.Equal(t, "get_weather", toolsConfig[0]["name"])
	assert.Equal(t, "search_db", toolsConfig[1]["name"])
}

func TestLLMResponseFormat_Success(t *testing.T) {
	format := map[string]any{
		"type": "json_object",
	}

	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("Test"),
		LLMAPIKey("sk-test"),
		LLMResponseFormat(format),
	).Build()

	require.NoError(t, err)
	formatConfig, ok := node.Config["response_format"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "json_object", formatConfig["type"])
}

func TestLLMJSONMode_Success(t *testing.T) {
	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("Test"),
		LLMAPIKey("sk-test"),
		LLMJSONMode(),
	).Build()

	require.NoError(t, err)
	formatConfig, ok := node.Config["response_format"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "json_object", formatConfig["type"])
}

func TestLLMStop_Success(t *testing.T) {
	stop := []string{"\n", "END"}

	node, err := NewNode("llm-node", "llm", "LLM Node",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("Test"),
		LLMAPIKey("sk-test"),
		LLMStop(stop),
	).Build()

	require.NoError(t, err)
	stopConfig, ok := node.Config["stop"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"\n", "END"}, stopConfig)
}

func TestNewOpenAINode_Success(t *testing.T) {
	node, err := NewOpenAINode("openai-node", "OpenAI LLM", "gpt-4", "Test prompt",
		LLMAPIKey("sk-test"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "openai-node", node.ID)
	assert.Equal(t, "OpenAI LLM", node.Name)
	assert.Equal(t, "llm", node.Type)
	assert.Equal(t, "openai", node.Config["provider"])
	assert.Equal(t, "gpt-4", node.Config["model"])
	assert.Equal(t, "Test prompt", node.Config["prompt"])
}

func TestNewAnthropicNode_Success(t *testing.T) {
	node, err := NewAnthropicNode("anthropic-node", "Anthropic LLM", "claude-3", "Test prompt",
		LLMAPIKey("sk-test"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "anthropic-node", node.ID)
	assert.Equal(t, "Anthropic LLM", node.Name)
	assert.Equal(t, "llm", node.Type)
	assert.Equal(t, "anthropic", node.Config["provider"])
	assert.Equal(t, "claude-3", node.Config["model"])
	assert.Equal(t, "Test prompt", node.Config["prompt"])
}

func TestNewLLMNode_Generic(t *testing.T) {
	node, err := NewLLMNode("llm-node", "Generic LLM",
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel("gpt-4"),
		LLMPrompt("Test"),
		LLMAPIKey("sk-test"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "llm", node.Type)
	assert.Equal(t, "openai", node.Config["provider"])
}

// ==================== Complex Integration Tests ====================

func TestNodeBuilder_HTTPWithAllOptions(t *testing.T) {
	body := map[string]any{"name": "test"}

	node, err := NewHTTPPostNode("complex-http", "Complex HTTP", "https://api.example.com", body,
		HTTPHeader("Content-Type", "application/json"),
		HTTPQueryParam("v", "1"),
		HTTPTimeout(30*time.Second),
		WithNodeDescription("Complex HTTP request"),
		WithPosition(100, 200),
		WithNodeMetadata("category", "api"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "POST", node.Config["method"])
	assert.NotNil(t, node.Config["body"])
	assert.NotNil(t, node.Config["headers"])
	assert.NotNil(t, node.Config["query_params"])
	assert.Equal(t, "30s", node.Config["timeout"])
	assert.Equal(t, "Complex HTTP request", node.Description)
	assert.NotNil(t, node.Position)
	assert.Equal(t, "api", node.Metadata["category"])
}

func TestNodeBuilder_LLMWithAllOptions(t *testing.T) {
	tools := []map[string]any{
		{"name": "get_weather", "description": "Get weather"},
	}

	node, err := NewOpenAINode("complex-llm", "Complex LLM", "gpt-4", "Complex prompt",
		LLMAPIKey("sk-test"),
		LLMSystemPrompt("You are helpful"),
		LLMTemperature(0.7),
		LLMMaxTokens(1000),
		LLMTopP(0.9),
		LLMTools(tools),
		LLMJSONMode(),
		LLMStop([]string{"\n"}),
		WithNodeDescription("Complex LLM node"),
		GridPosition(1, 2),
		WithNodeMetadata("model_type", "gpt"),
	).Build()

	require.NoError(t, err)
	assert.Equal(t, "openai", node.Config["provider"])
	assert.Equal(t, 0.7, node.Config["temperature"])
	assert.Equal(t, 1000, node.Config["max_tokens"])
	assert.Equal(t, 0.9, node.Config["top_p"])
	assert.NotNil(t, node.Config["tools"])
	assert.NotNil(t, node.Config["response_format"])
	assert.NotNil(t, node.Config["stop"])
	assert.NotNil(t, node.Position)
	assert.Equal(t, "gpt", node.Metadata["model_type"])
}
