package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactQuery(t *testing.T) {
	assert.Equal(t, "", redactQuery(""))
	assert.Equal(t, "execution_id=abc", redactQuery("execution_id=abc"))

	got := redactQuery("execution_id=abc&token=secret-jwt")
	assert.NotContains(t, got, "secret-jwt")
	assert.Contains(t, got, "token=REDACTED")
	assert.Contains(t, got, "execution_id=abc")

	assert.NotContains(t, redactQuery("Access_Token=xyz"), "xyz")
}

func TestValidRequestID(t *testing.T) {
	assert.True(t, validRequestID("0b6f4c1e-7f3a-4a57-9b7e-1c2d3e4f5a6b"))
	assert.False(t, validRequestID(""))
	assert.False(t, validRequestID("bad\nid"))
	assert.False(t, validRequestID(string(make([]byte, 200))))
}
