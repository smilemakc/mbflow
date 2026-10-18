package openapi

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/smilemakc/mbflow/go/pkg/models"
)

// Argument keys of an operation called as an LLM tool. They match the config
// keys of the openapi node, so tool arguments are passed to the operation as they are.
const (
	ToolArgPathParams  = "path_params"
	ToolArgQueryParams = "query_params"
	ToolArgHeaders     = "headers"
	ToolArgCookies     = "cookies"
	ToolArgBody        = "body"
)

const (
	// maxToolSchemaDepth bounds nesting of inlined schemas; deeper schemas keep only their type.
	maxToolSchemaDepth = 8
	// maxToolSchemaNodes bounds the schema of one tool, which is sent to the
	// model with every request; fields beyond it are omitted.
	maxToolSchemaNodes = 256
	// maxToolEnumValues bounds enum lists copied into tool schemas.
	maxToolEnumValues       = 64
	maxToolFieldDescription = 300
	maxToolDescription      = 1024
)

var toolParamGroups = []struct {
	in, key, description string
}{
	{"path", ToolArgPathParams, "Path parameters."},
	{"query", ToolArgQueryParams, "Query string parameters."},
	{"header", ToolArgHeaders, "HTTP request headers."},
	{"cookie", ToolArgCookies, "Cookies."},
}

// ToolArgKey returns the argument key that groups parameters of the given
// location ("path", "query", "header" or "cookie"), or "" for other values.
func ToolArgKey(in string) string {
	for _, group := range toolParamGroups {
		if group.in == in {
			return group.key
		}
	}
	return ""
}

// ToolParameters returns the JSON Schema of the arguments of op called as an
// LLM tool. Parameters are grouped by location under the ToolArg keys and the
// request body is under ToolArgBody. Component references are inlined, since
// LLM providers accept only self-contained schemas; recursive, too deep and
// oversized parts keep only their type. Parameters filled in by the client
// authentication and read-only body fields are left out.
func ToolParameters(client *models.APIClient, op *models.APIOperation) map[string]any {
	b := &toolSchemaBuilder{schemas: client.Schemas, active: make(map[string]bool), budget: maxToolSchemaNodes}
	props := make(map[string]any)
	var required []string
	for _, group := range toolParamGroups {
		schema, groupRequired := b.paramGroup(client, op, group.in, group.description)
		if schema == nil {
			continue
		}
		props[group.key] = schema
		if groupRequired {
			required = append(required, group.key)
		}
	}
	if rb := op.RequestBody; rb != nil && len(rb.Content) > 0 {
		props[ToolArgBody] = b.body(rb)
		if rb.Required {
			required = append(required, ToolArgBody)
		}
	}

	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// ToolDescription describes op for an LLM: its summary and description,
// followed by the HTTP method, the path and the API client name.
func ToolDescription(client *models.APIClient, op *models.APIOperation) string {
	var parts []string
	if op.Deprecated {
		parts = append(parts, "Deprecated.")
	}
	if op.Summary != "" {
		parts = append(parts, op.Summary)
	}
	if op.Description != "" && op.Description != op.Summary {
		parts = append(parts, op.Description)
	}
	suffix := fmt.Sprintf("%s %s (API %q)", op.Method, op.Path, client.Name)
	if len(parts) == 0 {
		return truncateText(suffix, maxToolDescription)
	}
	text := truncateText(strings.Join(parts, "\n\n"), maxToolDescription-utf8.RuneCountInString(suffix)-2)
	return text + "\n\n" + suffix
}

// isAuthParam reports whether the client authentication sets the parameter,
// so the model is not asked for it.
func isAuthParam(client *models.APIClient, op *models.APIOperation, p *models.APIParameter) bool {
	auth := client.Auth
	if op.NoAuth || !auth.IsEnabled() {
		return false
	}
	if auth.Type != models.APIClientAuthAPIKey {
		return p.In == "header" && strings.EqualFold(p.Name, "Authorization")
	}
	in := auth.In
	if in == "" {
		in = models.APIKeyInHeader
	}
	if p.In != in {
		return false
	}
	if in == models.APIKeyInHeader {
		return strings.EqualFold(p.Name, auth.Name)
	}
	return p.Name == auth.Name
}

type toolSchemaBuilder struct {
	schemas map[string]map[string]any
	// active holds the references being inlined, so recursive schemas are cut.
	active map[string]bool
	budget int
}

func (b *toolSchemaBuilder) paramGroup(client *models.APIClient, op *models.APIOperation, in, description string) (map[string]any, bool) {
	props := make(map[string]any)
	var required []string
	for _, p := range op.Parameters {
		if p.In != in || isAuthParam(client, op, p) {
			continue
		}
		schema := map[string]any{"type": "string"}
		if p.Schema != nil {
			schema = b.inline(p.Schema, 1)
		}
		var notes []string
		if p.Deprecated {
			notes = append(notes, "Deprecated.")
		}
		if p.Description != "" {
			notes = append(notes, p.Description)
		} else if d, _ := schema["description"].(string); d != "" {
			notes = append(notes, d)
		}
		if example := scalarText(p.Example); example != "" {
			notes = append(notes, "Example: "+example)
		}
		setToolDescription(schema, notes...)
		props[p.Name] = schema
		if p.Required {
			required = append(required, p.Name)
		}
	}
	if len(props) == 0 {
		return nil, false
	}
	group := map[string]any{"type": "object", "description": description, "properties": props}
	if len(required) > 0 {
		group["required"] = required
	}
	return group, len(required) > 0
}

// body describes the request body in the content type the operation uses by default.
func (b *toolSchemaBuilder) body(rb *models.APIRequestBody) map[string]any {
	media := rb.Content[0]
	mediaType := baseMediaType(media.ContentType)
	isJSON := isJSONMediaType(mediaType) || strings.Contains(mediaType, "*")
	structured := isJSON || mediaType == "application/x-www-form-urlencoded" || mediaType == "multipart/form-data"

	var schema map[string]any
	switch {
	case media.Schema != nil:
		schema = b.inline(media.Schema, 1)
	case structured:
		schema = map[string]any{}
	default:
		schema = map[string]any{"type": "string"}
	}

	var notes []string
	if rb.Description != "" {
		notes = append(notes, rb.Description)
	} else if d, _ := schema["description"].(string); d != "" {
		notes = append(notes, d)
	} else {
		notes = append(notes, "Request body.")
	}
	if !isJSON {
		notes = append(notes, "Sent as "+mediaType+".")
	}
	setToolDescription(schema, notes...)
	return schema
}

// inline returns a self-contained copy of schema limited to the keywords
// that LLM providers accept in tool parameters.
func (b *toolSchemaBuilder) inline(schema map[string]any, depth int) map[string]any {
	if ref, ok := schema["$ref"].(string); ok {
		resolved := ResolveSchema(b.schemas, schema)
		if resolved == nil {
			return map[string]any{}
		}
		if b.active[ref] {
			return cutToolSchema(resolved)
		}
		b.active[ref] = true
		defer delete(b.active, ref)
		schema = resolved
	}
	if schema == nil {
		return map[string]any{}
	}
	if depth > maxToolSchemaDepth || b.budget <= 0 {
		return cutToolSchema(schema)
	}
	b.budget--

	out := make(map[string]any)
	var notes []string
	if deprecated, _ := schema["deprecated"].(bool); deprecated {
		notes = append(notes, "Deprecated.")
	}
	if d, _ := schema["description"].(string); d != "" {
		notes = append(notes, d)
	}
	if t := toolSchemaType(schema); t != "" {
		out["type"] = t
	}
	if f, ok := schema["format"].(string); ok && f != "" {
		out["format"] = f
	}
	for _, key := range []string{"minimum", "maximum", "minLength", "maxLength", "minItems", "maxItems"} {
		if v, ok := schema[key]; ok && isNumber(v) {
			out[key] = v
		}
	}
	if v := schema["default"]; isScalar(v) {
		out["default"] = v
	}
	notes = append(notes, toolEnum(out, schema)...)

	if props, ok := schema["properties"].(map[string]any); ok && len(props) > 0 {
		outProps := make(map[string]any, len(props))
		for _, name := range sortedKeys(props) {
			propSchema, _ := props[name].(map[string]any)
			if readOnly, _ := ResolveSchema(b.schemas, propSchema)["readOnly"].(bool); readOnly {
				continue
			}
			if b.budget <= 0 {
				notes = append(notes, "Some fields are omitted.")
				break
			}
			outProps[name] = b.inline(propSchema, depth+1)
		}
		out["properties"] = outProps
	} else if additional, ok := schema["additionalProperties"].(map[string]any); ok && additional != nil {
		notes = append(notes, "Object with arbitrary keys.")
	}
	if items, ok := schema["items"].(map[string]any); ok {
		out["items"] = b.inline(items, depth+1)
	}
	if required := stringList(schema["required"]); len(required) > 0 {
		out["required"] = required
	}
	setToolDescription(out, notes...)

	// Alternatives and allOf parts only add keywords the schema itself lacks.
	var alternatives []any
	for _, key := range []string{"oneOf", "anyOf"} {
		list, _ := schema[key].([]any)
		for _, item := range list {
			alt, ok := item.(map[string]any)
			if !ok || b.budget <= 0 {
				continue
			}
			if inlined := b.inline(alt, depth+1); inlined["type"] != "null" {
				alternatives = append(alternatives, inlined)
			}
		}
	}
	if len(alternatives) == 1 {
		single, _ := alternatives[0].(map[string]any)
		mergeToolSchema(out, single)
	} else if len(alternatives) > 1 {
		out["anyOf"] = alternatives
	}
	if list, ok := schema["allOf"].([]any); ok {
		for _, item := range list {
			if part, ok := item.(map[string]any); ok {
				mergeToolSchema(out, b.inline(part, depth+1))
			}
		}
	}

	if _, ok := out["type"]; !ok {
		if out["properties"] != nil {
			out["type"] = "object"
		} else if out["items"] != nil {
			out["type"] = "array"
		}
	}
	if out["type"] == "array" && out["items"] == nil {
		out["items"] = map[string]any{}
	}
	filterRequired(out)
	return out
}

// cutToolSchema keeps only the type and the description of schema.
func cutToolSchema(schema map[string]any) map[string]any {
	out := make(map[string]any)
	t := toolSchemaType(schema)
	if t == "" {
		if schema["properties"] != nil {
			t = "object"
		} else if schema["items"] != nil {
			t = "array"
		}
	}
	description, _ := schema["description"].(string)
	switch t {
	case "":
	case "object":
		out["type"] = t
		description += " Nested fields are omitted."
	case "array":
		out["type"] = t
		out["items"] = map[string]any{}
		description += " Item structure is omitted."
	default:
		out["type"] = t
	}
	setToolDescription(out, description)
	return out
}

// toolSchemaType returns the declared type, or the first non-null type of a
// type list; "null" only when it is the only type.
func toolSchemaType(schema map[string]any) string {
	var types []string
	switch t := schema["type"].(type) {
	case string:
		return t
	case []string:
		types = t
	case []any:
		for _, item := range t {
			if s, ok := item.(string); ok {
				types = append(types, s)
			}
		}
	}
	return firstNonNull(types)
}

// toolEnum copies an enum of strings into out. Other enums, which not every
// provider accepts, and long lists are described in the returned notes instead.
func toolEnum(out, schema map[string]any) []string {
	values, _ := schema["enum"].([]any)
	if c, ok := schema["const"]; ok && c != nil {
		values = []any{c}
	}
	var nonNull []any
	allStrings := true
	for _, v := range values {
		if v == nil {
			continue
		}
		if _, ok := v.(string); !ok {
			allStrings = false
		}
		nonNull = append(nonNull, v)
	}
	if len(nonNull) == 0 {
		return nil
	}
	if allStrings && len(nonNull) <= maxToolEnumValues {
		out["enum"] = nonNull
		return nil
	}
	shown := nonNull
	if len(shown) > maxToolEnumValues {
		shown = shown[:maxToolEnumValues]
	}
	texts := make([]string, len(shown))
	for i, v := range shown {
		texts[i] = jsonString(v)
	}
	note := "Allowed values: " + strings.Join(texts, ", ")
	if len(shown) < len(nonNull) {
		note += " and others"
	}
	return []string{note + "."}
}

// mergeToolSchema adds to dst the keywords of src that dst lacks; properties
// and required lists are combined.
func mergeToolSchema(dst, src map[string]any) {
	for key, value := range src {
		switch key {
		case "properties":
			srcProps, _ := value.(map[string]any)
			dstProps, _ := dst["properties"].(map[string]any)
			if dstProps == nil {
				dstProps = make(map[string]any, len(srcProps))
				dst["properties"] = dstProps
			}
			for name, prop := range srcProps {
				if _, exists := dstProps[name]; !exists {
					dstProps[name] = prop
				}
			}
		case "required":
			dst["required"] = append(stringList(dst["required"]), stringList(value)...)
		default:
			if _, exists := dst[key]; !exists {
				dst[key] = value
			}
		}
	}
}

// filterRequired keeps required names that are among the properties, once each.
func filterRequired(schema map[string]any) {
	props, _ := schema["properties"].(map[string]any)
	seen := make(map[string]bool)
	var kept []string
	for _, name := range stringList(schema["required"]) {
		if _, ok := props[name]; ok && !seen[name] {
			seen[name] = true
			kept = append(kept, name)
		}
	}
	if len(kept) == 0 {
		delete(schema, "required")
		return
	}
	schema["required"] = kept
}

func setToolDescription(schema map[string]any, notes ...string) {
	parts := make([]string, 0, len(notes))
	for _, note := range notes {
		if note = strings.TrimSpace(note); note != "" {
			parts = append(parts, note)
		}
	}
	if len(parts) == 0 {
		delete(schema, "description")
		return
	}
	schema["description"] = truncateText(strings.Join(parts, " "), maxToolFieldDescription)
}

// truncateText shortens s to at most limit runes, marking the cut with an ellipsis.
func truncateText(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:limit-1])) + "…"
}

// scalarText renders a scalar example; objects and arrays give "".
func scalarText(v any) string {
	switch v.(type) {
	case string, bool, float64, float32, int, int64, int32:
		return truncateText(jsonString(v), 60)
	}
	return ""
}

func stringList(v any) []string {
	switch list := v.(type) {
	case []string:
		return append([]string(nil), list...)
	case []any:
		result := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

func isNumber(v any) bool {
	switch v.(type) {
	case float64, float32, int, int64, int32, uint64:
		return true
	}
	return false
}

func isScalar(v any) bool {
	switch v.(type) {
	case string, bool:
		return true
	}
	return isNumber(v)
}
