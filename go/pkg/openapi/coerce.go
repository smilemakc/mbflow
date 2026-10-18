package openapi

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

const maxCoerceDepth = 32

var (
	jsonIntegerPattern = regexp.MustCompile(`^-?(0|[1-9]\d*)$`)
	jsonNumberPattern  = regexp.MustCompile(`^-?(0|[1-9]\d*)(\.\d+)?([eE][+-]?\d+)?$`)
)

// Coerce converts string values to the types declared by the schema.
// Node configs carry strings because templates always resolve to strings,
// so "42" becomes 42 for an integer field and "[1,2]" becomes an array.
// Values that do not fit the declared type are returned unchanged and left
// for the API to reject. In objects, an empty string is dropped for fields
// whose type is not string.
func Coerce(value any, schema map[string]any, schemas map[string]map[string]any) any {
	return coerce(value, schema, schemas, 0)
}

func coerce(value any, schema map[string]any, schemas map[string]map[string]any, depth int) any {
	if depth > maxCoerceDepth {
		return value
	}
	schema = flattenAllOf(ResolveSchema(schemas, schema), schemas, 0)
	if schema == nil {
		return value
	}
	typ := schemaType(schema)

	if typ == "" {
		if alternatives := schemaAlternatives(schema); len(alternatives) > 0 {
			return coerceAlternatives(value, alternatives, schemas, depth)
		}
	}

	switch v := value.(type) {
	case string:
		return coerceString(v, typ, schema, schemas, depth)
	case map[string]any:
		if typ != "" && typ != "object" {
			return v
		}
		return coerceObject(v, schema, schemas, depth)
	case []any:
		if typ != "" && typ != "array" {
			return v
		}
		items, _ := schema["items"].(map[string]any)
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = coerce(item, items, schemas, depth+1)
		}
		return out
	}
	return value
}

func coerceString(s, typ string, schema map[string]any, schemas map[string]map[string]any, depth int) any {
	trimmed := strings.TrimSpace(s)
	switch typ {
	case "integer":
		if jsonIntegerPattern.MatchString(trimmed) {
			return json.Number(trimmed)
		}
	case "number":
		if jsonNumberPattern.MatchString(trimmed) {
			return json.Number(trimmed)
		}
	case "boolean":
		if b, err := strconv.ParseBool(trimmed); err == nil {
			return b
		}
	case "array", "object":
		if parsed, ok := parseJSON(trimmed); ok {
			return coerce(parsed, schema, schemas, depth+1)
		}
	case "null":
		if trimmed == "" || trimmed == "null" {
			return nil
		}
	}
	return s
}

func coerceObject(m, schema map[string]any, schemas map[string]map[string]any, depth int) map[string]any {
	props, _ := schema["properties"].(map[string]any)
	additional, _ := schema["additionalProperties"].(map[string]any)

	out := make(map[string]any, len(m))
	for key, value := range m {
		propSchema, _ := props[key].(map[string]any)
		if propSchema == nil {
			propSchema = additional
		}
		if s, ok := value.(string); ok && strings.TrimSpace(s) == "" && propSchema != nil {
			resolved := flattenAllOf(ResolveSchema(schemas, propSchema), schemas, 0)
			if t := schemaType(resolved); t != "" && t != "string" {
				continue
			}
		}
		out[key] = coerce(value, propSchema, schemas, depth+1)
	}
	return out
}

// coerceAlternatives handles oneOf/anyOf: a string stays a string when any
// alternative accepts strings, otherwise the first conversion that changes
// the value wins.
func coerceAlternatives(value any, alternatives []map[string]any, schemas map[string]map[string]any, depth int) any {
	resolved := make([]map[string]any, 0, len(alternatives))
	for _, alt := range alternatives {
		if r := flattenAllOf(ResolveSchema(schemas, alt), schemas, 0); r != nil {
			resolved = append(resolved, r)
		}
	}
	switch v := value.(type) {
	case string:
		for _, alt := range resolved {
			if schemaType(alt) == "string" {
				return v
			}
		}
		for _, alt := range resolved {
			if converted := coerce(v, alt, schemas, depth+1); !isSameString(converted, v) {
				return converted
			}
		}
	case map[string]any:
		for _, alt := range resolved {
			if t := schemaType(alt); t == "object" || (t == "" && alt["properties"] != nil) {
				return coerce(v, alt, schemas, depth+1)
			}
		}
	case []any:
		for _, alt := range resolved {
			if schemaType(alt) == "array" {
				return coerce(v, alt, schemas, depth+1)
			}
		}
	}
	return value
}

func isSameString(value any, s string) bool {
	str, ok := value.(string)
	return ok && str == s
}

func schemaAlternatives(schema map[string]any) []map[string]any {
	var result []map[string]any
	for _, key := range []string{"oneOf", "anyOf"} {
		list, _ := schema[key].([]any)
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				result = append(result, m)
			}
		}
	}
	return result
}

// flattenAllOf merges allOf parts into one schema so their properties can be
// coerced together.
func flattenAllOf(schema map[string]any, schemas map[string]map[string]any, depth int) map[string]any {
	parts, ok := schema["allOf"].([]any)
	if !ok || len(parts) == 0 || depth > maxCoerceDepth {
		return schema
	}
	merged := make(map[string]any, len(schema))
	for k, v := range schema {
		if k != "allOf" {
			merged[k] = v
		}
	}
	props := make(map[string]any)
	if own, ok := schema["properties"].(map[string]any); ok {
		for k, v := range own {
			props[k] = v
		}
	}
	for _, part := range parts {
		partSchema, _ := part.(map[string]any)
		resolved := flattenAllOf(ResolveSchema(schemas, partSchema), schemas, depth+1)
		if resolved == nil {
			continue
		}
		if partProps, ok := resolved["properties"].(map[string]any); ok {
			for k, v := range partProps {
				if _, exists := props[k]; !exists {
					props[k] = v
				}
			}
		}
		for _, key := range []string{"type", "items", "additionalProperties"} {
			if _, exists := merged[key]; !exists && resolved[key] != nil {
				merged[key] = resolved[key]
			}
		}
	}
	if len(props) > 0 {
		merged["properties"] = props
	}
	return merged
}

// schemaType returns the declared type, skipping "null" in OpenAPI 3.1 type
// arrays, or infers it from object and array keywords.
func schemaType(schema map[string]any) string {
	if schema == nil {
		return ""
	}
	switch t := schema["type"].(type) {
	case string:
		return t
	case []string:
		return firstNonNull(t)
	case []any:
		types := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				types = append(types, s)
			}
		}
		return firstNonNull(types)
	}
	if schema["properties"] != nil {
		return "object"
	}
	if schema["items"] != nil {
		return "array"
	}
	return ""
}

func firstNonNull(types []string) string {
	for _, t := range types {
		if t != "null" {
			return t
		}
	}
	if len(types) > 0 {
		return types[0]
	}
	return ""
}

// parseJSON decodes a JSON object or array, keeping numbers exact.
func parseJSON(s string) (any, bool) {
	if s == "" || (s[0] != '{' && s[0] != '[') {
		return nil, false
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil || dec.More() {
		return nil, false
	}
	return v, true
}
