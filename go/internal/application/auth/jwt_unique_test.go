package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTGenerateAccessToken_IsUniqueWithinTheSameSecond(t *testing.T) {
	svc := NewJWTService(newTestConfig())
	user := newTestUser()

	first, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)
	second, _, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)
	assert.NotEqual(t, first, second)

	claims, err := svc.ValidateAccessToken(first)
	require.NoError(t, err)
	assert.NotEmpty(t, claims.ID)
}
