package openapi

import (
	"mime"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ComponentSchemaRefPrefix starts every "$ref" left in converted schemas.
const ComponentSchemaRefPrefix = "#/components/schemas/"

const (
	// maxInlineDepth bounds nesting of schemas that are inlined rather than referenced.
	maxInlineDepth = 64
	// maxRefHops bounds chains of component schemas that are aliases of each other.
	maxRefHops = 32
	// maxConvertedSchemas bounds the total number of converted schema nodes:
	// inlined schemas shared through non-component references are copied at
	// every use, which a crafted document can make exponential.
	maxConvertedSchemas = 200_000
)

// schemaConverter turns kin-openapi schemas into plain JSON Schema maps.
// Component schemas stay references, so recursive and shared schemas are
// stored once; only components reachable from operations are emitted.
type schemaConverter struct {
	source    openapi3.Schemas
	converted map[string]map[string]any
	queue     []string
	queued    map[string]bool
	inlining  map[*openapi3.Schema]bool
	budget    int
}

func newSchemaConverter(doc *openapi3.T) *schemaConverter {
	c := &schemaConverter{
		converted: make(map[string]map[string]any),
		queued:    make(map[string]bool),
		inlining:  make(map[*openapi3.Schema]bool),
		budget:    maxConvertedSchemas,
	}
	if doc.Components != nil {
		c.source = doc.Components.Schemas
	}
	return c
}

func (c *schemaConverter) convert(ref *openapi3.SchemaRef) map[string]any {
	return c.convertRef(ref, 0)
}

// components converts every component schema referenced so far, including
// the ones referenced from other components.
func (c *schemaConverter) components() map[string]map[string]any {
	for len(c.queue) > 0 {
		name := c.queue[0]
		c.queue = c.queue[1:]
		ref := c.source[name]
		if ref == nil {
			continue
		}
		if target, ok := componentName(ref.Ref); ok && target != name && c.source[target] != nil {
			c.enqueue(target)
			c.converted[name] = map[string]any{"$ref": componentRef(target)}
			continue
		}
		c.converted[name] = c.convertValue(ref.Value, 0)
	}
	return c.converted
}

func (c *schemaConverter) enqueue(name string) {
	if !c.queued[name] {
		c.queued[name] = true
		c.queue = append(c.queue, name)
	}
}

func (c *schemaConverter) convertRef(ref *openapi3.SchemaRef, depth int) map[string]any {
	if ref == nil {
		return nil
	}
	if name, ok := componentName(ref.Ref); ok && c.source[name] != nil {
		c.enqueue(name)
		return map[string]any{"$ref": componentRef(name)}
	}
	return c.convertValue(ref.Value, depth)
}

func (c *schemaConverter) convertValue(s *openapi3.Schema, depth int) map[string]any {
	if s == nil {
		return map[string]any{}
	}
	if depth >= maxInlineDepth || c.inlining[s] {
		return map[string]any{"description": "recursive schema"}
	}
	if c.budget <= 0 {
		return map[string]any{"description": "schema omitted: specification is too complex"}
	}
	c.budget--
	c.inlining[s] = true
	defer delete(c.inlining, s)

	if s.Always != nil {
		if *s.Always {
			return map[string]any{}
		}
		return map[string]any{"not": map[string]any{}}
	}

	out := make(map[string]any)
	if types := s.Type.Slice(); len(types) == 1 {
		out["type"] = types[0]
	} else if len(types) > 1 {
		out["type"] = append([]string(nil), types...)
	}
	setString(out, "title", s.Title)
	setString(out, "format", s.Format)
	setString(out, "description", s.Description)
	setString(out, "pattern", s.Pattern)
	if len(s.Enum) > 0 {
		out["enum"] = s.Enum
	}
	if s.Const != nil {
		out["const"] = s.Const
	}
	if s.Default != nil {
		out["default"] = s.Default
	}
	if s.Example != nil {
		out["example"] = s.Example
	} else if len(s.Examples) > 0 {
		out["example"] = s.Examples[0]
	}
	setBool(out, "nullable", s.Nullable)
	setBool(out, "readOnly", s.ReadOnly)
	setBool(out, "writeOnly", s.WriteOnly)
	setBool(out, "deprecated", s.Deprecated)
	setBool(out, "uniqueItems", s.UniqueItems)
	if s.Min != nil {
		out["minimum"] = *s.Min
	}
	if s.Max != nil {
		out["maximum"] = *s.Max
	}
	if s.MultipleOf != nil {
		out["multipleOf"] = *s.MultipleOf
	}
	if s.MinLength > 0 {
		out["minLength"] = s.MinLength
	}
	if s.MaxLength != nil {
		out["maxLength"] = *s.MaxLength
	}
	if s.MinItems > 0 {
		out["minItems"] = s.MinItems
	}
	if s.MaxItems != nil {
		out["maxItems"] = *s.MaxItems
	}

	next := depth + 1
	if s.Items != nil {
		out["items"] = c.convertRef(s.Items, next)
	}
	if len(s.Properties) > 0 {
		props := make(map[string]any, len(s.Properties))
		for name, prop := range s.Properties {
			props[name] = c.convertRef(prop, next)
		}
		out["properties"] = props
	}
	if len(s.Required) > 0 {
		out["required"] = append([]string(nil), s.Required...)
	}
	if has := s.AdditionalProperties.Has; has != nil {
		out["additionalProperties"] = *has
	} else if ap := s.AdditionalProperties.Schema; ap != nil {
		out["additionalProperties"] = c.convertRef(ap, next)
	}
	if list := c.convertList(s.OneOf, next); list != nil {
		out["oneOf"] = list
	}
	if list := c.convertList(s.AnyOf, next); list != nil {
		out["anyOf"] = list
	}
	if list := c.convertList(s.AllOf, next); list != nil {
		out["allOf"] = list
	}
	if s.Discriminator != nil && s.Discriminator.PropertyName != "" {
		out["discriminator"] = map[string]any{"propertyName": s.Discriminator.PropertyName}
	}
	return out
}

func (c *schemaConverter) convertList(refs openapi3.SchemaRefs, depth int) []any {
	if len(refs) == 0 {
		return nil
	}
	list := make([]any, 0, len(refs))
	for _, ref := range refs {
		list = append(list, c.convertRef(ref, depth))
	}
	return list
}

func setString(out map[string]any, key, value string) {
	if value != "" {
		out[key] = value
	}
}

func setBool(out map[string]any, key string, value bool) {
	if value {
		out[key] = true
	}
}

// componentName extracts the schema name from "#/components/schemas/<name>",
// decoding JSON Pointer escapes.
func componentName(ref string) (string, bool) {
	if !strings.HasPrefix(ref, ComponentSchemaRefPrefix) {
		return "", false
	}
	name := ref[len(ComponentSchemaRefPrefix):]
	if name == "" || strings.Contains(name, "/") {
		return "", false
	}
	return strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~"), true
}

func componentRef(name string) string {
	return ComponentSchemaRefPrefix + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
}

// ResolveSchema follows "$ref" values into schemas. It returns nil for a
// dangling reference or a reference cycle.
func ResolveSchema(schemas map[string]map[string]any, schema map[string]any) map[string]any {
	for i := 0; i < maxRefHops; i++ {
		if schema == nil {
			return nil
		}
		ref, ok := schema["$ref"].(string)
		if !ok {
			return schema
		}
		name, ok := componentName(ref)
		if !ok {
			return nil
		}
		schema = schemas[name]
	}
	return nil
}

// baseMediaType lowercases a content type and strips its parameters.
func baseMediaType(contentType string) string {
	if mediaType, _, err := mime.ParseMediaType(contentType); err == nil {
		return mediaType
	}
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = contentType[:i]
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}

func isJSONMediaType(mediaType string) bool {
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") || mediaType == "*/*"
}
