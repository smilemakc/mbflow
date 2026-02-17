package builder

import (
	"github.com/smilemakc/mbflow/pkg/models"
)

// NewOpenAIResponsesNode creates a new LLM node using OpenAI Responses API.
// This supports GPT-5, o3-mini, gpt-4.1+ and other modern reasoning models.
//
// Example:
//
//	node := builder.NewOpenAIResponsesNode(
//	    "research",
//	    "llm",
//	    "Web Research",
//	    "gpt-4.1",
//	    "Research quantum computing advances",
//	    builder.WithInstructions("You are a research assistant."),
//	    builder.WithWebSearch(nil, "large"),
//	    builder.WithReasoningEffort("high"),
//	)
func NewOpenAIResponsesNode(id, nodeType, name, model, prompt string, opts ...NodeOption) *NodeBuilder {
	nb := NewNode(id, nodeType, name,
		WithConfigValue("provider", string(models.LLMProviderOpenAIResponses)),
		WithConfigValue("model", model),
		WithConfigValue("prompt", prompt),
	)

	for _, opt := range opts {
		_ = opt(nb) // Errors are captured in nb.err
	}

	return nb
}

// --- Responses API Specific Options ---

// WithInstructions sets the system instructions (alternative to WithInstruction).
// For Responses API, this is the recommended way to set system prompts.
func WithInstructions(instructions string) NodeOption {
	return WithConfigValue("instructions", instructions)
}

// WithStructuredInput sets structured multimodal input for Responses API.
// This allows combining text, images, and files in a single request.
//
// Example:
//
//	input := []map[string]any{
//	    {
//	        "role": "user",
//	        "content": []map[string]any{
//	            {"type": "input_text", "text": "What is in this image?"},
//	            {"type": "input_image", "image_url": "https://..."},
//	        },
//	    },
//	}
//	WithStructuredInput(input)
func WithStructuredInput(items []map[string]any) NodeOption {
	return WithConfigValue("input", items)
}

// WithWebSearch enables web search tool for Responses API.
// Allows the model to search the web and cite sources.
//
// Parameters:
//   - domains: Optional list of domains to restrict search (nil for all domains)
//   - contextSize: "small", "medium", or "large" (controls search depth)
//
// Example:
//
//	WithWebSearch(nil, "large")  // Search all domains with large context
//	WithWebSearch([]string{"wikipedia.org", "arxiv.org"}, "medium")
func WithWebSearch(domains []string, contextSize string) NodeOption {
	return func(nb *NodeBuilder) error {
		tools := getHostedTools(nb)
		tool := map[string]any{
			"type": "web_search_preview",
		}
		if len(domains) > 0 {
			tool["domains"] = domains
		}
		if contextSize != "" {
			tool["search_context_size"] = contextSize
		}
		tools = append(tools, tool)
		nb.config["hosted_tools"] = tools
		return nil
	}
}

// WithFileSearch enables file search tool for Responses API.
// Searches through documents in a vector store.
//
// Parameters:
//   - vectorStoreIDs: List of vector store IDs to search
//   - maxResults: Maximum number of search results (0 for default)
//
// Example:
//
//	WithFileSearch([]string{"vs_abc123"}, 20)
func WithFileSearch(vectorStoreIDs []string, maxResults int) NodeOption {
	return func(nb *NodeBuilder) error {
		tools := getHostedTools(nb)
		tool := map[string]any{
			"type":             "file_search",
			"vector_store_ids": vectorStoreIDs,
		}
		if maxResults > 0 {
			tool["max_num_results"] = maxResults
		}
		tools = append(tools, tool)
		nb.config["hosted_tools"] = tools
		return nil
	}
}

// WithCodeInterpreter enables code interpreter tool for Responses API.
// Allows the model to write and execute Python code.
//
// Example:
//
//	WithCodeInterpreter()
func WithCodeInterpreter() NodeOption {
	return func(nb *NodeBuilder) error {
		tools := getHostedTools(nb)
		tools = append(tools, map[string]any{
			"type": "code_interpreter",
		})
		nb.config["hosted_tools"] = tools
		return nil
	}
}

// WithReasoningEffort sets reasoning effort level for reasoning models (o3-mini, etc.).
//
// Parameters:
//   - effort: "low", "medium", or "high"
//
// Higher effort means more thorough reasoning but slower responses and higher cost.
//
// Example:
//
//	WithReasoningEffort("high")  // For complex problem solving
//	WithReasoningEffort("low")   // For simpler queries
func WithReasoningEffort(effort string) NodeOption {
	return WithConfigValue("reasoning", map[string]any{
		"effort": effort,
	})
}

// WithBackground enables background processing for long-running tasks.
// The request returns immediately and can be polled for completion.
//
// Example:
//
//	WithBackground(true)
func WithBackground(background bool) NodeOption {
	return WithConfigValue("background", background)
}

// WithConversationState sets previous response ID for multi-turn conversations.
// This preserves reasoning state across multiple workflow executions.
//
// Example:
//
//	WithConversationState("{{env.last_response_id}}")
func WithConversationState(previousResponseID string) NodeOption {
	return WithConfigValue("previous_response_id", previousResponseID)
}

// WithMaxToolCalls limits the number of tool iterations.
// Useful for preventing infinite loops in complex workflows.
//
// Example:
//
//	WithMaxToolCalls(5)  // Maximum 5 tool calls per request
func WithMaxToolCalls(maxCalls int) NodeOption {
	return WithConfigValue("max_tool_calls", maxCalls)
}

// WithStore controls whether to store the response in OpenAI's storage.
// Defaults to true. Set to false to avoid storing sensitive data.
//
// Example:
//
//	WithStore(false)  // Don't store response
func WithStore(store bool) NodeOption {
	return WithConfigValue("store", store)
}

// Helper function to get or initialize hosted_tools array
func getHostedTools(nb *NodeBuilder) []map[string]any {
	if tools, ok := nb.config["hosted_tools"].([]map[string]any); ok {
		return tools
	}
	return []map[string]any{}
}
