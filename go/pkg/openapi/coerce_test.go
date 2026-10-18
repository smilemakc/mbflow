package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoerce_Scalars(t *testing.T) {
	assert.Equal(t, json.Number("42"), Coerce("42", map[string]any{"type": "integer"}, nil))
	assert.Equal(t, json.Number("-1.5e3"), Coerce(" -1.5e3 ", map[string]any{"type": "number"}, nil))
	assert.Equal(t, true, Coerce("TRUE", map[string]any{"type": "boolean"}, nil))
	assert.Equal(t, "abc", Coerce("abc", map[string]any{"type": "integer"}, nil), "mismatched values are left for the API to reject")
	assert.Equal(t, "007", Coerce("007", map[string]any{"type": "integer"}, nil), "leading zeros are not a JSON number")
	assert.Equal(t, "0x1F", Coerce("0x1F", map[string]any{"type": "number"}, nil))
	assert.Equal(t, "42", Coerce("42", map[string]any{"type": "string"}, nil))
	assert.Equal(t, "42", Coerce("42", nil, nil), "values without a schema are unchanged")
	assert.Nil(t, Coerce("null", map[string]any{"type": "null"}, nil))
	assert.Equal(t, json.Number("7"), Coerce("7", map[string]any{"type": []any{"null", "integer"}}, nil),
		"OpenAPI 3.1 type arrays skip null")
}

func TestCoerce_NestedStructures(t *testing.T) {
	schemas := map[string]map[string]any{
		"Item": {
			"type": "object",
			"properties": map[string]any{
				"qty":   map[string]any{"type": "integer"},
				"price": map[string]any{"type": "number"},
			},
		},
	}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items":    map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Item"}},
			"meta":     map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "boolean"}},
			"count":    map[string]any{"type": "integer"},
			"comment":  map[string]any{"type": "string"},
			"optional": map[string]any{"type": "integer"},
		},
	}
	got := Coerce(map[string]any{
		"items":    `[{"qty":"2","price":"9.99"}]`,
		"meta":     map[string]any{"gift": "false"},
		"count":    "3",
		"comment":  "",
		"optional": "",
		"unknown":  "kept",
	}, schema, schemas)

	assert.Equal(t, map[string]any{
		"items":   []any{map[string]any{"qty": json.Number("2"), "price": json.Number("9.99")}},
		"meta":    map[string]any{"gift": false},
		"count":   json.Number("3"),
		"comment": "",
		"unknown": "kept",
	}, got, "empty strings are dropped only for non-string fields")
}

func TestCoerce_AllOfAndAlternatives(t *testing.T) {
	schemas := map[string]map[string]any{
		"Base": {"type": "object", "properties": map[string]any{"id": map[string]any{"type": "integer"}}},
	}
	allOf := map[string]any{"allOf": []any{
		map[string]any{"$ref": "#/components/schemas/Base"},
		map[string]any{"properties": map[string]any{"active": map[string]any{"type": "boolean"}}},
	}}
	assert.Equal(t,
		map[string]any{"id": json.Number("5"), "active": true},
		Coerce(map[string]any{"id": "5", "active": "true"}, allOf, schemas))

	stringOrNumber := map[string]any{"oneOf": []any{
		map[string]any{"type": "integer"},
		map[string]any{"type": "string"},
	}}
	assert.Equal(t, "15", Coerce("15", stringOrNumber, nil), "a string alternative keeps strings intact")

	numberOrBool := map[string]any{"anyOf": []any{
		map[string]any{"type": "boolean"},
		map[string]any{"type": "integer"},
	}}
	assert.Equal(t, json.Number("15"), Coerce("15", numberOrBool, nil))
	assert.Equal(t, false, Coerce("false", numberOrBool, nil))
}

func TestCoerce_DepthIsBounded(t *testing.T) {
	schemas := map[string]map[string]any{
		"Node": {"type": "object", "properties": map[string]any{"child": map[string]any{"$ref": "#/components/schemas/Node"}}},
	}
	value := map[string]any{}
	cursor := value
	for i := 0; i < 100; i++ {
		next := map[string]any{}
		cursor["child"] = next
		cursor = next
	}
	assert.NotPanics(t, func() {
		Coerce(value, map[string]any{"$ref": "#/components/schemas/Node"}, schemas)
	})
}
