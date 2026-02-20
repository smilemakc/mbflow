package builder

import (
	"fmt"
)

// ValidateHTTPConfig validates HTTP node configuration.
func ValidateHTTPConfig(config map[string]any) error {
	// Check required fields
	if _, ok := config["method"]; !ok {
		return fmt.Errorf("HTTP node requires 'method' field")
	}

	if _, ok := config["url"]; !ok {
		return fmt.Errorf("HTTP node requires 'url' field")
	}

	return nil
}

// ValidateLLMConfig validates LLM node configuration.
func ValidateLLMConfig(config map[string]any) error {
	// Check required fields
	requiredFields := []string{"provider", "model", "prompt", "api_key"}
	for _, field := range requiredFields {
		if _, ok := config[field]; !ok {
			return fmt.Errorf("LLM node requires '%s' field", field)
		}
	}

	// Validate temperature if present
	if temp, ok := config["temperature"].(float64); ok {
		if temp < 0 || temp > 2 {
			return fmt.Errorf("temperature must be between 0 and 2, got %f", temp)
		}
	}

	// Validate top_p if present
	if topP, ok := config["top_p"].(float64); ok {
		if topP < 0 || topP > 1 {
			return fmt.Errorf("top_p must be between 0 and 1, got %f", topP)
		}
	}

	// Validate max_tokens if present
	if maxTokens, ok := config["max_tokens"].(int); ok {
		if maxTokens < 0 {
			return fmt.Errorf("max_tokens must be >= 0, got %d", maxTokens)
		}
	}

	return nil
}

// ValidateTransformConfig validates Transform node configuration.
func ValidateTransformConfig(config map[string]any) error {
	// Check required field
	transformType, ok := config["type"]
	if !ok {
		return fmt.Errorf("Transform node requires 'type' field")
	}

	typeStr, ok := transformType.(string)
	if !ok {
		return fmt.Errorf("Transform 'type' must be a string")
	}

	// Validate type-specific requirements
	switch typeStr {
	case "passthrough":
		// No additional fields required
	case "expression":
		if _, ok := config["expression"]; !ok {
			return fmt.Errorf("Expression transform requires 'expression' field")
		}
	case "jq":
		if _, ok := config["filter"]; !ok {
			return fmt.Errorf("JQ transform requires 'filter' field")
		}
	case "template":
		if _, ok := config["template"]; !ok {
			return fmt.Errorf("Template transform requires 'template' field")
		}
	default:
		return fmt.Errorf("invalid transform type: %s", typeStr)
	}

	return nil
}

// ValidateNodeConfig validates node configuration based on node type.
// This is optional and only used in strict validation mode.
func ValidateNodeConfig(nodeType string, config map[string]any) error {
	switch nodeType {
	case "http":
		return ValidateHTTPConfig(config)
	case "llm":
		return ValidateLLMConfig(config)
	case "transform":
		return ValidateTransformConfig(config)
	default:
		// For unknown types, skip validation
		// They may be custom executors
		return nil
	}
}
