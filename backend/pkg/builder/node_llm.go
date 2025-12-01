package builder

import (
	"fmt"

	"github.com/smilemakc/mbflow/pkg/models"
)

// LLMProvider sets the LLM provider.
func LLMProvider(provider models.LLMProvider) NodeOption {
	return func(nb *NodeBuilder) error {
		validProviders := map[models.LLMProvider]bool{
			models.LLMProviderOpenAI:    true,
			models.LLMProviderAnthropic: true,
		}
		if !validProviders[provider] {
			return fmt.Errorf("unsupported LLM provider: %s", provider)
		}
		nb.config["provider"] = string(provider)
		return nil
	}
}

// LLMModel sets the model name.
func LLMModel(model string) NodeOption {
	return func(nb *NodeBuilder) error {
		if model == "" {
			return fmt.Errorf("model cannot be empty")
		}
		nb.config["model"] = model
		return nil
	}
}

// LLMPrompt sets the prompt.
func LLMPrompt(prompt string) NodeOption {
	return func(nb *NodeBuilder) error {
		if prompt == "" {
			return fmt.Errorf("prompt cannot be empty")
		}
		nb.config["prompt"] = prompt
		return nil
	}
}

// LLMAPIKey sets the API key.
func LLMAPIKey(apiKey string) NodeOption {
	return func(nb *NodeBuilder) error {
		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}
		nb.config["api_key"] = apiKey
		return nil
	}
}

// LLMTemperature sets the temperature (0-2).
func LLMTemperature(temp float64) NodeOption {
	return func(nb *NodeBuilder) error {
		if temp < 0 || temp > 2 {
			return fmt.Errorf("temperature must be between 0 and 2, got %f", temp)
		}
		nb.config["temperature"] = temp
		return nil
	}
}

// LLMMaxTokens sets the maximum tokens.
func LLMMaxTokens(tokens int) NodeOption {
	return func(nb *NodeBuilder) error {
		if tokens < 0 {
			return fmt.Errorf("max_tokens must be >= 0, got %d", tokens)
		}
		nb.config["max_tokens"] = tokens
		return nil
	}
}

// LLMTopP sets the top-p sampling parameter (0-1).
func LLMTopP(topP float64) NodeOption {
	return func(nb *NodeBuilder) error {
		if topP < 0 || topP > 1 {
			return fmt.Errorf("top_p must be between 0 and 1, got %f", topP)
		}
		nb.config["top_p"] = topP
		return nil
	}
}

// LLMSystemPrompt sets the system prompt.
func LLMSystemPrompt(systemPrompt string) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["system_prompt"] = systemPrompt
		return nil
	}
}

// LLMTools sets the tools/functions available to the LLM.
func LLMTools(tools []map[string]interface{}) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["tools"] = tools
		return nil
	}
}

// LLMResponseFormat sets the response format.
// For JSON mode: map[string]interface{}{"type": "json_object"}
func LLMResponseFormat(format map[string]interface{}) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["response_format"] = format
		return nil
	}
}

// LLMJSONMode enables JSON response mode.
// This is a convenience wrapper for LLMResponseFormat.
func LLMJSONMode() NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["response_format"] = map[string]interface{}{
			"type": "json_object",
		}
		return nil
	}
}

// LLMStop sets stop sequences.
func LLMStop(stop []string) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["stop"] = stop
		return nil
	}
}

// NewOpenAINode creates a new OpenAI LLM node builder.
func NewOpenAINode(id, name, model, prompt string, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		LLMProvider(models.LLMProviderOpenAI),
		LLMModel(model),
		LLMPrompt(prompt),
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "llm", name, allOpts...)
}

// NewAnthropicNode creates a new Anthropic LLM node builder.
func NewAnthropicNode(id, name, model, prompt string, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		LLMProvider(models.LLMProviderAnthropic),
		LLMModel(model),
		LLMPrompt(prompt),
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "llm", name, allOpts...)
}

// NewLLMNode creates a new generic LLM node builder.
// You must specify the provider using LLMProvider option.
func NewLLMNode(id, name string, opts ...NodeOption) *NodeBuilder {
	return NewNode(id, "llm", name, opts...)
}
