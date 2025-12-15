package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenAIResponsesNode_WithWebSearch(t *testing.T) {
	tests := []struct {
		name          string
		domains       []string
		contextSize   string
		expectDomains bool
		expectContext bool
	}{
		{
			name:          "all domains with large context",
			domains:       nil,
			contextSize:   "large",
			expectDomains: false,
			expectContext: true,
		},
		{
			name:          "specific domains with medium context",
			domains:       []string{"wikipedia.org", "arxiv.org"},
			contextSize:   "medium",
			expectDomains: true,
			expectContext: true,
		},
		{
			name:          "no context size specified",
			domains:       []string{"example.com"},
			contextSize:   "",
			expectDomains: true,
			expectContext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewOpenAIResponsesNode(
				"research",
				"llm",
				"Test Node",
				"gpt-4.1",
				"Test prompt",
				WithWebSearch(tt.domains, tt.contextSize),
			)

			require.NoError(t, node.err)

			hostedTools, ok := node.config["hosted_tools"].([]map[string]interface{})
			require.True(t, ok, "hosted_tools should be present")
			require.Len(t, hostedTools, 1)

			tool := hostedTools[0]
			assert.Equal(t, "web_search_preview", tool["type"])

			if tt.expectDomains {
				assert.Equal(t, tt.domains, tool["domains"])
			} else {
				assert.Nil(t, tool["domains"])
			}

			if tt.expectContext {
				assert.Equal(t, tt.contextSize, tool["search_context_size"])
			} else {
				assert.Nil(t, tool["search_context_size"])
			}
		})
	}
}

func TestWithFileSearch(t *testing.T) {
	tests := []struct {
		name             string
		vectorStoreIDs   []string
		maxResults       int
		expectMaxResults bool
	}{
		{
			name:             "with max results",
			vectorStoreIDs:   []string{"vs_123", "vs_456"},
			maxResults:       10,
			expectMaxResults: true,
		},
		{
			name:             "default max results",
			vectorStoreIDs:   []string{"vs_789"},
			maxResults:       0,
			expectMaxResults: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewOpenAIResponsesNode(
				"search",
				"llm",
				"File Search Node",
				"gpt-4.1",
				"Search files",
				WithFileSearch(tt.vectorStoreIDs, tt.maxResults),
			)

			require.NoError(t, node.err)

			hostedTools, ok := node.config["hosted_tools"].([]map[string]interface{})
			require.True(t, ok)
			require.Len(t, hostedTools, 1)

			tool := hostedTools[0]
			assert.Equal(t, "file_search", tool["type"])
			assert.Equal(t, tt.vectorStoreIDs, tool["vector_store_ids"])

			if tt.expectMaxResults {
				assert.Equal(t, tt.maxResults, tool["max_num_results"])
			} else {
				assert.Nil(t, tool["max_num_results"])
			}
		})
	}
}

func TestWithCodeInterpreter(t *testing.T) {
	node := NewOpenAIResponsesNode(
		"code",
		"llm",
		"Code Interpreter Node",
		"gpt-4.1",
		"Execute code",
		WithCodeInterpreter(),
	)

	require.NoError(t, node.err)

	hostedTools, ok := node.config["hosted_tools"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, hostedTools, 1)

	tool := hostedTools[0]
	assert.Equal(t, "code_interpreter", tool["type"])
}

func TestWithReasoningEffort(t *testing.T) {
	tests := []struct {
		name   string
		effort string
	}{
		{"low effort", "low"},
		{"medium effort", "medium"},
		{"high effort", "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewOpenAIResponsesNode(
				"reasoning",
				"llm",
				"Reasoning Node",
				"o3-mini",
				"Think deeply",
				WithReasoningEffort(tt.effort),
			)

			require.NoError(t, node.err)

			reasoning, ok := node.config["reasoning"].(map[string]interface{})
			require.True(t, ok, "reasoning should be a map")
			assert.Equal(t, tt.effort, reasoning["effort"])
		})
	}
}

func TestWithBackground(t *testing.T) {
	node := NewOpenAIResponsesNode(
		"background",
		"llm",
		"Background Node",
		"gpt-4.1",
		"Process in background",
		WithBackground(true),
	)

	require.NoError(t, node.err)
	assert.Equal(t, true, node.config["background"])
}

func TestWithConversationState(t *testing.T) {
	state := "response_abc123"

	node := NewOpenAIResponsesNode(
		"chat",
		"llm",
		"Chat Node",
		"gpt-4.1",
		"Continue conversation",
		WithConversationState(state),
	)

	require.NoError(t, node.err)
	assert.Equal(t, state, node.config["previous_response_id"])
}

func TestWithMaxToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		maxCalls int
	}{
		{"single call", 1},
		{"multiple calls", 5},
		{"many calls", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewOpenAIResponsesNode(
				"tools",
				"llm",
				"Tool Calling Node",
				"gpt-4.1",
				"Use tools",
				WithMaxToolCalls(tt.maxCalls),
			)

			require.NoError(t, node.err)
			assert.Equal(t, tt.maxCalls, node.config["max_tool_calls"])
		})
	}
}

func TestWithStore(t *testing.T) {
	tests := []struct {
		name  string
		store bool
	}{
		{"enable store", true},
		{"disable store", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewOpenAIResponsesNode(
				"store",
				"llm",
				"Store Node",
				"gpt-4.1",
				"Use store",
				WithStore(tt.store),
			)

			require.NoError(t, node.err)
			assert.Equal(t, tt.store, node.config["store"])
		})
	}
}

func TestWithStructuredInput(t *testing.T) {
	input := []map[string]interface{}{
		{
			"role": "user",
			"content": []map[string]interface{}{
				{"type": "input_text", "text": "What is in this image?"},
				{"type": "input_image", "image_url": "https://example.com/image.jpg"},
			},
		},
	}

	node := NewOpenAIResponsesNode(
		"multimodal",
		"llm",
		"Multimodal Node",
		"gpt-4.1",
		"",
		WithStructuredInput(input),
	)

	require.NoError(t, node.err)
	assert.Equal(t, input, node.config["input"])
}

func TestMultipleHostedTools(t *testing.T) {
	node := NewOpenAIResponsesNode(
		"multi",
		"llm",
		"Multi Tool Node",
		"gpt-4.1",
		"Use multiple tools",
		WithWebSearch([]string{"example.com"}, "large"),
		WithFileSearch([]string{"vs_123"}, 5),
		WithCodeInterpreter(),
	)

	require.NoError(t, node.err)

	hostedTools, ok := node.config["hosted_tools"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, hostedTools, 3, "should have 3 hosted tools")

	// Verify each tool type
	toolTypes := make([]string, len(hostedTools))
	for i, tool := range hostedTools {
		toolTypes[i] = tool["type"].(string)
	}

	assert.Contains(t, toolTypes, "web_search_preview")
	assert.Contains(t, toolTypes, "file_search")
	assert.Contains(t, toolTypes, "code_interpreter")
}

func TestWithInstructions(t *testing.T) {
	instructions := "You are a helpful research assistant specializing in quantum physics."

	node := NewOpenAIResponsesNode(
		"assistant",
		"llm",
		"Assistant Node",
		"gpt-4.1",
		"Answer questions",
		WithInstructions(instructions),
	)

	require.NoError(t, node.err)
	assert.Equal(t, instructions, node.config["instructions"])
}

func TestComplexResponsesNode(t *testing.T) {
	// Test a complex node with multiple options
	node := NewOpenAIResponsesNode(
		"research",
		"llm",
		"Advanced Research Node",
		"o3-mini",
		"Research quantum computing advances",
		WithInstructions("You are a PhD-level research assistant."),
		WithWebSearch([]string{"arxiv.org", "scholar.google.com"}, "large"),
		WithReasoningEffort("high"),
		WithMaxToolCalls(5),
		WithBackground(false),
	)

	require.NoError(t, node.err)

	// Verify basic config
	assert.Equal(t, "openai-responses", node.config["provider"])
	assert.Equal(t, "o3-mini", node.config["model"])
	assert.Equal(t, "Research quantum computing advances", node.config["prompt"])

	// Verify additional options
	assert.Equal(t, "You are a PhD-level research assistant.", node.config["instructions"])

	reasoning, ok := node.config["reasoning"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "high", reasoning["effort"])

	assert.Equal(t, 5, node.config["max_tool_calls"])
	assert.Equal(t, false, node.config["background"])

	// Verify hosted tools
	hostedTools, ok := node.config["hosted_tools"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, hostedTools, 1)

	tool := hostedTools[0]
	assert.Equal(t, "web_search_preview", tool["type"])
	assert.Equal(t, []string{"arxiv.org", "scholar.google.com"}, tool["domains"])
	assert.Equal(t, "large", tool["search_context_size"])
}
