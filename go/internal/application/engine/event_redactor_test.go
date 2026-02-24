package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactMaskLong(t *testing.T) {
	r := NewEventRedactor()
	// String > 8 chars: first 3 + "***" + last 3
	result := r.MaskValue("sk-1234567890abc")
	assert.Equal(t, "sk-***abc", result)
}

func TestRedactMaskShort(t *testing.T) {
	r := NewEventRedactor()
	// String <= 8 chars: just "***"
	result := r.MaskValue("secret")
	assert.Equal(t, "***", result)
}

func TestRedactMaskExactly8(t *testing.T) {
	r := NewEventRedactor()
	result := r.MaskValue("12345678")
	assert.Equal(t, "***", result)
}

func TestRedactMapValues(t *testing.T) {
	r := NewEventRedactor()
	m := map[string]any{
		"api_key": "sk-1234567890abc",
		"name":    "test-workflow",
	}
	result := r.RedactMap(m)
	assert.Equal(t, "sk-***abc", result["api_key"])
	assert.Equal(t, "tes***low", result["name"])
}

func TestRedactNestedMap(t *testing.T) {
	r := NewEventRedactor()
	m := map[string]any{
		"outer": map[string]any{
			"secret": "my-secret-value-here",
		},
	}
	result := r.RedactMap(m)
	inner := result["outer"].(map[string]any)
	assert.Equal(t, "my-***ere", inner["secret"])
}

func TestRedactPreservesNonString(t *testing.T) {
	r := NewEventRedactor()
	m := map[string]any{
		"count":   42,
		"enabled": true,
		"ratio":   3.14,
	}
	result := r.RedactMap(m)
	assert.Equal(t, 42, result["count"])
	assert.Equal(t, true, result["enabled"])
	assert.Equal(t, 3.14, result["ratio"])
}

func TestRedactNilMap(t *testing.T) {
	r := NewEventRedactor()
	result := r.RedactMap(nil)
	assert.Nil(t, result)
}
