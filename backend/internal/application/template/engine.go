package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Engine is the main template resolution engine.
// It resolves templates in strings and complex data structures.
type Engine struct {
	resolver *Resolver
	options  TemplateOptions
}

// NewEngine creates a new template engine with the given context and options.
func NewEngine(ctx *VariableContext, opts TemplateOptions) *Engine {
	return &Engine{
		resolver: NewResolver(ctx, opts),
		options:  opts,
	}
}

// NewEngineWithDefaults creates a new template engine with default options.
func NewEngineWithDefaults(ctx *VariableContext) *Engine {
	return NewEngine(ctx, DefaultOptions())
}

// templatePattern matches template placeholders like {{env.varName}} or {{input.field.path}}
var templatePattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// Resolve resolves all templates in the input data.
// Supports strings, maps, slices, and nested structures.
func (e *Engine) Resolve(data interface{}) (interface{}, error) {
	if data == nil {
		return nil, nil
	}

	switch v := data.(type) {
	case string:
		return e.ResolveString(v)
	case map[string]interface{}:
		return e.resolveMap(v)
	case []interface{}:
		return e.resolveSlice(v)
	default:
		// For other types, try to convert to map and resolve
		return e.resolveComplex(v)
	}
}

// ResolveString resolves templates in a single string.
// Example: "Hello {{env.name}}" -> "Hello World"
func (e *Engine) ResolveString(template string) (string, error) {
	if template == "" {
		return template, nil
	}

	var resolveErr error
	result := templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		// Extract the variable reference (remove {{ and }})
		varRef := strings.TrimSpace(match[2 : len(match)-2])

		// Parse variable type and path
		varType, path := e.parseVariableRef(varRef)
		if varType == "" {
			// Only set error in strict mode
			if e.options.StrictMode {
				resolveErr = fmt.Errorf("%w: invalid variable reference '%s'", ErrInvalidTemplate, varRef)
			}
			if e.options.PlaceholderOnMissing {
				return match
			}
			return ""
		}

		// Resolve the variable
		value, err := e.resolver.ResolveVariable(varType, path)
		if err != nil {
			// Only set error in strict mode
			if e.options.StrictMode {
				resolveErr = &TemplateError{
					Template: template,
					Variable: varType,
					Path:     path,
					Err:      err,
				}
				return ""
			}

			// Non-strict mode: return placeholder or empty
			if e.options.PlaceholderOnMissing {
				return match
			}
			return ""
		}

		// Convert value to string
		return e.valueToString(value)
	})

	if resolveErr != nil {
		return "", resolveErr
	}

	return result, nil
}

// resolveMap resolves templates in all values of a map.
func (e *Engine) resolveMap(m map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{}, len(m))

	for key, value := range m {
		resolved, err := e.Resolve(value)
		if err != nil {
			return nil, fmt.Errorf("error resolving key '%s': %w", key, err)
		}
		result[key] = resolved
	}

	return result, nil
}

// resolveSlice resolves templates in all elements of a slice.
func (e *Engine) resolveSlice(s []interface{}) ([]interface{}, error) {
	result := make([]interface{}, len(s))

	for i, value := range s {
		resolved, err := e.Resolve(value)
		if err != nil {
			return nil, fmt.Errorf("error resolving index %d: %w", i, err)
		}
		result[i] = resolved
	}

	return result, nil
}

// resolveComplex handles complex types by converting to JSON and back.
func (e *Engine) resolveComplex(data interface{}) (interface{}, error) {
	// For primitive types that don't need template resolution, return as-is
	switch data.(type) {
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, complex64, complex128:
		return data, nil
	}

	// Convert to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		// If can't marshal, return as-is
		return data, nil
	}

	// Try to unmarshal to generic structure
	var generic interface{}
	if err := json.Unmarshal(jsonData, &generic); err != nil {
		return data, nil
	}

	// Check if the unmarshaled type is one of the supported types
	// to avoid infinite recursion
	switch v := generic.(type) {
	case map[string]interface{}:
		return e.resolveMap(v)
	case []interface{}:
		return e.resolveSlice(v)
	case string:
		return e.ResolveString(v)
	default:
		// If still not a supported type, return as-is
		return generic, nil
	}
}

// parseVariableRef parses a variable reference into type and path.
// Examples:
//   - "env.userName" -> ("env", "userName")
//   - "input.data.user.email" -> ("input", "data.user.email")
//   - "env.items[0].name" -> ("env", "items[0].name")
func (e *Engine) parseVariableRef(ref string) (string, string) {
	parts := strings.SplitN(ref, ".", 2)
	if len(parts) < 2 {
		return "", ""
	}

	varType := strings.TrimSpace(parts[0])
	path := strings.TrimSpace(parts[1])

	return varType, path
}

// valueToString converts a value to its string representation.
func (e *Engine) valueToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		// For complex types, marshal to JSON
		if data, err := json.Marshal(v); err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	}
}

// ResolveConfig is a convenience method for resolving templates in node configurations.
// It resolves all template strings in the config map.
func (e *Engine) ResolveConfig(config map[string]interface{}) (map[string]interface{}, error) {
	resolved, err := e.resolveMap(config)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config: %w", err)
	}

	return resolved, nil
}

// HasTemplates checks if a string contains any template placeholders.
func HasTemplates(s string) bool {
	return templatePattern.MatchString(s)
}

// ExtractVariables extracts all variable references from a template string.
// Returns a slice of variable references in the format "type.path".
func ExtractVariables(template string) []string {
	matches := templatePattern.FindAllStringSubmatch(template, -1)
	vars := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) > 1 {
			vars = append(vars, strings.TrimSpace(match[1]))
		}
	}

	return vars
}

// ValidateTemplate validates that a template string has valid syntax.
func ValidateTemplate(template string) error {
	vars := ExtractVariables(template)

	for _, varRef := range vars {
		parts := strings.SplitN(varRef, ".", 2)
		if len(parts) < 2 {
			return fmt.Errorf("%w: invalid variable reference '{{%s}}' (expected format: {{type.path}})", ErrInvalidTemplate, varRef)
		}

		varType := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[1])

		if varType != "env" && varType != "input" {
			return fmt.Errorf("%w: unknown variable type '%s' (supported: env, input)", ErrInvalidTemplate, varType)
		}

		if path == "" {
			return fmt.Errorf("%w: empty path for variable type '%s'", ErrInvalidTemplate, varType)
		}
	}

	return nil
}
