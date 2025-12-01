package template

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Resolver handles variable resolution with support for nested paths.
type Resolver struct {
	context *VariableContext
	options TemplateOptions
}

// NewResolver creates a new variable resolver.
func NewResolver(ctx *VariableContext, opts TemplateOptions) *Resolver {
	return &Resolver{
		context: ctx,
		options: opts,
	}
}

// ResolveVariable resolves a variable reference (e.g., "env.user.name" or "input.data[0].id").
// Returns the resolved value and any error encountered.
func (r *Resolver) ResolveVariable(varType, path string) (interface{}, error) {
	var value interface{}
	var found bool

	switch varType {
	case "env":
		if path == "" {
			return nil, fmt.Errorf("%w: env requires a variable name", ErrInvalidTemplate)
		}
		value, found = r.resolveEnvPath(path)

	case "input":
		if path == "" {
			return nil, fmt.Errorf("%w: input requires a field name", ErrInvalidTemplate)
		}
		value, found = r.resolveInputPath(path)

	default:
		return nil, fmt.Errorf("%w: unknown variable type '%s'", ErrInvalidTemplate, varType)
	}

	if !found {
		// Always return the error, let the engine decide how to handle it
		return nil, fmt.Errorf("%w: {{%s.%s}}", ErrVariableNotFound, varType, path)
	}

	return value, nil
}

// resolveEnvPath resolves an environment variable with nested path support.
func (r *Resolver) resolveEnvPath(path string) (interface{}, bool) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return nil, false
	}

	// Extract root variable name (handle case where first part has array index)
	rootName := parts[0]
	if bracketIdx := strings.Index(rootName, "["); bracketIdx > 0 {
		rootName = rootName[:bracketIdx]
	}

	// Get the root variable
	root, found := r.context.GetEnvVariable(rootName)
	if !found {
		return nil, false
	}

	// If first part has array index, apply it (index only, since root is already resolved)
	if strings.Contains(parts[0], "[") {
		// Extract just the index part (e.g., "items[0]" -> "[0]")
		if bracketIdx := strings.Index(parts[0], "["); bracketIdx >= 0 {
			indexPart := parts[0][bracketIdx:]
			var err error
			root, err = r.resolveArrayIndex(root, indexPart)
			if err != nil {
				return nil, false
			}
		}
		parts = parts[1:] // Consume the first part
	} else {
		parts = parts[1:] // Skip the root variable name
	}

	// If no more nested path, return root
	if len(parts) == 0 {
		return root, true
	}

	// Traverse remaining path
	return r.traversePath(root, parts)
}

// resolveInputPath resolves an input variable with nested path support.
func (r *Resolver) resolveInputPath(path string) (interface{}, bool) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return nil, false
	}

	// Extract root variable name (handle case where first part has array index)
	rootName := parts[0]
	if bracketIdx := strings.Index(rootName, "["); bracketIdx > 0 {
		rootName = rootName[:bracketIdx]
	}

	// Get the root variable
	root, found := r.context.GetInputVariable(rootName)
	if !found {
		return nil, false
	}

	// If first part has array index, apply it (index only, since root is already resolved)
	if strings.Contains(parts[0], "[") {
		// Extract just the index part (e.g., "items[0]" -> "[0]")
		if bracketIdx := strings.Index(parts[0], "["); bracketIdx >= 0 {
			indexPart := parts[0][bracketIdx:]
			var err error
			root, err = r.resolveArrayIndex(root, indexPart)
			if err != nil {
				return nil, false
			}
		}
		parts = parts[1:] // Consume the first part
	} else {
		parts = parts[1:] // Skip the root variable name
	}

	// If no more nested path, return root
	if len(parts) == 0 {
		return root, true
	}

	// Traverse remaining path
	return r.traversePath(root, parts)
}

// traversePath traverses a nested path in a value.
// Supports both object field access (user.name) and array indexing (items[0]).
func (r *Resolver) traversePath(value interface{}, parts []string) (interface{}, bool) {
	current := value

	for _, part := range parts {
		// Check if this is array indexing (e.g., "[0]" or "items[0]")
		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			// Handle array indexing
			var err error
			current, err = r.resolveArrayIndex(current, part)
			if err != nil {
				return nil, false
			}
			continue
		}

		// Handle object field access
		current = r.resolveField(current, part)
		if current == nil {
			return nil, false
		}
	}

	return current, true
}

// resolveField resolves a field in an object.
func (r *Resolver) resolveField(value interface{}, field string) interface{} {
	if value == nil {
		return nil
	}

	// Try map access first
	if m, ok := value.(map[string]interface{}); ok {
		return m[field]
	}

	// Try reflection for structs
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		f := v.FieldByName(field)
		if f.IsValid() {
			return f.Interface()
		}
	}

	// Try JSON unmarshaling for complex types
	if data, err := json.Marshal(value); err == nil {
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err == nil {
			return m[field]
		}
	}

	return nil
}

// resolveArrayIndex resolves array indexing (e.g., "[0]", "items[0]", "[0][1]").
func (r *Resolver) resolveArrayIndex(value interface{}, indexExpr string) (interface{}, error) {
	// Parse field name and indices
	// Examples: "[0]", "items[0]", "[0][1]"
	fieldName := ""
	indexPart := indexExpr

	if bracketIdx := strings.Index(indexExpr, "["); bracketIdx > 0 {
		fieldName = indexExpr[:bracketIdx]
		indexPart = indexExpr[bracketIdx:]
	}

	// If there's a field name, resolve it first
	current := value
	if fieldName != "" {
		current = r.resolveField(current, fieldName)
		if current == nil {
			return nil, fmt.Errorf("%w: field '%s' not found", ErrInvalidPath, fieldName)
		}
	}

	// Parse all indices (support chained indexing like [0][1])
	indices := parseArrayIndices(indexPart)
	if len(indices) == 0 {
		return nil, ErrArrayIndexInvalid
	}

	// Apply each index
	for _, idx := range indices {
		var err error
		current, err = r.indexArray(current, idx)
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

// indexArray applies a single array index to a value.
func (r *Resolver) indexArray(value interface{}, index int) (interface{}, error) {
	if value == nil {
		return nil, ErrTypeNotSupported
	}

	// Try slice/array access
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		if index < 0 || index >= v.Len() {
			return nil, fmt.Errorf("%w: index %d, length %d", ErrArrayOutOfBounds, index, v.Len())
		}
		return v.Index(index).Interface(), nil
	}

	// Try JSON array
	if data, err := json.Marshal(value); err == nil {
		var arr []interface{}
		if err := json.Unmarshal(data, &arr); err == nil {
			if index < 0 || index >= len(arr) {
				return nil, fmt.Errorf("%w: index %d, length %d", ErrArrayOutOfBounds, index, len(arr))
			}
			return arr[index], nil
		}
	}

	return nil, ErrTypeNotSupported
}

// splitPath splits a path into parts, handling dots and brackets.
// Example: "user.profile.items[0].name" -> ["user", "profile", "items[0]", "name"]
func splitPath(path string) []string {
	if path == "" {
		return nil
	}

	var parts []string
	var current strings.Builder
	inBracket := false

	for _, ch := range path {
		switch ch {
		case '.':
			if !inBracket && current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		case '[':
			inBracket = true
			current.WriteRune(ch)
		case ']':
			inBracket = false
			current.WriteRune(ch)
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseArrayIndices parses array indices from a string like "[0]" or "[0][1]".
func parseArrayIndices(expr string) []int {
	var indices []int

	// Find all [n] patterns
	start := 0
	for {
		openIdx := strings.Index(expr[start:], "[")
		if openIdx == -1 {
			break
		}
		openIdx += start

		closeIdx := strings.Index(expr[openIdx:], "]")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx

		// Extract number between brackets
		numStr := expr[openIdx+1 : closeIdx]
		num, err := strconv.Atoi(strings.TrimSpace(numStr))
		if err != nil {
			return nil
		}

		indices = append(indices, num)
		start = closeIdx + 1
	}

	return indices
}
