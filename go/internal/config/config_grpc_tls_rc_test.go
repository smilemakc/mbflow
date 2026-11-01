package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rcSetProductionEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MBFLOW_ENV", EnvProduction)
	t.Setenv("MBFLOW_JWT_SECRET", "test-secret-key-that-is-at-least-32-chars-long")
	t.Setenv("MBFLOW_DATABASE_URL", "postgres://u:p@db:5432/mbflow?sslmode=require")
	t.Setenv("MBFLOW_CORS_ENABLED", "false")
}

func TestRCAuthGRPCTLS_DefaultsToInsecureOutsideProduction(t *testing.T) {
	clearEnv()
	t.Cleanup(clearEnv)
	t.Setenv("MBFLOW_ENV", EnvDevelopment)
	t.Setenv("MBFLOW_JWT_SECRET", "test-secret-key-that-is-at-least-32-chars-long")
	t.Setenv("MBFLOW_AUTH_GRPC_INSECURE", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.Auth.GRPCInsecure)
	assert.Empty(t, cfg.Auth.GRPCTLSCAFile)
	assert.Empty(t, cfg.Auth.GRPCTLSServerName)
}

func TestRCAuthGRPCTLS_DefaultsToTLSInProduction(t *testing.T) {
	clearEnv()
	t.Cleanup(clearEnv)
	rcSetProductionEnv(t)
	t.Setenv("MBFLOW_AUTH_GRPC_INSECURE", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.False(t, cfg.Auth.GRPCInsecure, "production must default to TLS")
}

func TestRCAuthGRPCTLS_ExplicitSettingsAreHonoured(t *testing.T) {
	clearEnv()
	t.Cleanup(clearEnv)
	rcSetProductionEnv(t)
	t.Setenv("MBFLOW_AUTH_GRPC_INSECURE", "true")
	t.Setenv("MBFLOW_AUTH_GRPC_TLS_CA_FILE", "/etc/ca.pem")
	t.Setenv("MBFLOW_AUTH_GRPC_TLS_SERVER_NAME", "auth.internal")

	cfg, err := Load()
	require.NoError(t, err)
	assert.True(t, cfg.Auth.GRPCInsecure, "explicit opt-out must be kept in production")
	assert.Equal(t, "/etc/ca.pem", cfg.Auth.GRPCTLSCAFile)
	assert.Equal(t, "auth.internal", cfg.Auth.GRPCTLSServerName)
}

func TestRCAuthGRPCTLS_InvalidBoolFails(t *testing.T) {
	clearEnv()
	t.Cleanup(clearEnv)
	t.Setenv("MBFLOW_JWT_SECRET", "test-secret-key-that-is-at-least-32-chars-long")
	t.Setenv("MBFLOW_AUTH_GRPC_INSECURE", "maybe")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MBFLOW_AUTH_GRPC_INSECURE")
}
