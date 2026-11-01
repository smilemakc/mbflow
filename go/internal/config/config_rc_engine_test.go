package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rcEngineDevEnv(t *testing.T) {
	t.Helper()
	clearEnv()
	t.Cleanup(clearEnv)
	t.Setenv("MBFLOW_ENV", EnvDevelopment)
	t.Setenv("MBFLOW_JWT_SECRET", "test-secret-key-that-is-at-least-32-chars-long")
	for _, k := range []string{"MBFLOW_MAX_CONCURRENT_EXECUTIONS", "MBFLOW_EXECUTION_HEARTBEAT_INTERVAL", "MBFLOW_EXECUTION_STALE_AFTER"} {
		t.Setenv(k, "")
	}
}

func validRCEngineConfig() *Config {
	return &Config{
		Server:   ServerConfig{Port: 8080},
		Database: DatabaseConfig{URL: "postgres://localhost/x", MaxConnections: 2, MinConnections: 1},
		Logging:  LoggingConfig{Level: "info", Format: "json"},
		Auth:     validAuthConfig(),
	}
}

func TestLoad_ExecutionDefaults(t *testing.T) {
	rcEngineDevEnv(t)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 100, cfg.Execution.MaxConcurrentExecutions)
	assert.Equal(t, 30*time.Second, cfg.Execution.HeartbeatInterval)
	assert.Zero(t, cfg.Execution.StaleAfter, "0 means three heartbeat intervals")
}

func TestLoad_ExecutionCustom(t *testing.T) {
	rcEngineDevEnv(t)
	t.Setenv("MBFLOW_MAX_CONCURRENT_EXECUTIONS", "0")
	t.Setenv("MBFLOW_EXECUTION_HEARTBEAT_INTERVAL", "10s")
	t.Setenv("MBFLOW_EXECUTION_STALE_AFTER", "1m")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Zero(t, cfg.Execution.MaxConcurrentExecutions, "0 = unlimited")
	assert.Equal(t, 10*time.Second, cfg.Execution.HeartbeatInterval)
	assert.Equal(t, time.Minute, cfg.Execution.StaleAfter)
}

func TestLoad_ExecutionInvalid(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"negative limit":       {"MBFLOW_MAX_CONCURRENT_EXECUTIONS": "-1"},
		"unparsable limit":     {"MBFLOW_MAX_CONCURRENT_EXECUTIONS": "many"},
		"stale below interval": {"MBFLOW_EXECUTION_HEARTBEAT_INTERVAL": "1m", "MBFLOW_EXECUTION_STALE_AFTER": "30s"},
		"negative interval":    {"MBFLOW_EXECUTION_HEARTBEAT_INTERVAL": "-1s"},
	} {
		t.Run(name, func(t *testing.T) {
			rcEngineDevEnv(t)
			for k, v := range env {
				t.Setenv(k, v)
			}
			_, err := Load()
			require.Error(t, err)
		})
	}
}

func TestValidate_ExecutionZeroValueIsValid(t *testing.T) {
	cfg := validRCEngineConfig()
	require.NoError(t, cfg.Validate(), "configs built in code may leave Execution empty")

	cfg.Execution = ExecutionConfig{HeartbeatInterval: time.Second}
	require.NoError(t, cfg.Validate(), "stale-after may be left to its default")

	cfg.Execution = ExecutionConfig{StaleAfter: -time.Second}
	require.Error(t, cfg.Validate())
}

func TestValidateAuth_Modes(t *testing.T) {
	secret := "test-secret-key-that-is-at-least-32-chars-long"
	cases := []struct {
		name    string
		auth    AuthConfig
		wantErr string
	}{
		{name: "unknown mode", auth: AuthConfig{Mode: "magic", MinPasswordLength: 8}, wantErr: "invalid MBFLOW_AUTH_MODE"},
		{name: "builtin without secret", auth: AuthConfig{Mode: "builtin", MinPasswordLength: 8}, wantErr: "MBFLOW_JWT_SECRET is required"},
		{name: "short secret", auth: AuthConfig{Mode: "hybrid", JWTSecret: "short", MinPasswordLength: 8}, wantErr: "at least 32"},
		{name: "gateway without url", auth: AuthConfig{Mode: "gateway", MinPasswordLength: 8}, wantErr: "MBFLOW_AUTH_GATEWAY_URL"},
		{name: "gateway ok", auth: AuthConfig{Mode: "gateway", GatewayURL: "https://gw", ClientID: "c", MinPasswordLength: 8}},
		{name: "grpc without address", auth: AuthConfig{Mode: "grpc", MinPasswordLength: 8}, wantErr: "MBFLOW_AUTH_GRPC_ADDRESS"},
		{name: "grpc ok", auth: AuthConfig{Mode: "grpc", GRPCAddress: "auth:9000", MinPasswordLength: 8}},
		{name: "admin email without password", auth: AuthConfig{Mode: "grpc_hybrid", JWTSecret: secret, BootstrapAdminEmail: "a@b.c", MinPasswordLength: 8}, wantErr: "must be set together"},
		{name: "weak password policy", auth: AuthConfig{Mode: "builtin", JWTSecret: secret, MinPasswordLength: 4}, wantErr: "MBFLOW_MIN_PASSWORD_LENGTH"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validRCEngineConfig()
			cfg.Auth = tc.auth
			err := cfg.Validate()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestGetEnvAsInt64AndFloat(t *testing.T) {
	parseErrors = nil
	t.Cleanup(func() { parseErrors = nil })

	t.Setenv("RC_ENGINE_INT64", "1234567890123")
	t.Setenv("RC_ENGINE_FLOAT", "0.25")
	assert.Equal(t, int64(1234567890123), getEnvAsInt64("RC_ENGINE_INT64", 1))
	assert.InDelta(t, 0.25, getEnvAsFloat("RC_ENGINE_FLOAT", 1), 1e-9)
	assert.Equal(t, int64(7), getEnvAsInt64("RC_ENGINE_UNSET", 7))
	assert.InDelta(t, 0.5, getEnvAsFloat("RC_ENGINE_UNSET", 0.5), 1e-9)
	assert.Empty(t, parseErrors)

	t.Setenv("RC_ENGINE_INT64", "big")
	t.Setenv("RC_ENGINE_FLOAT", "half")
	assert.Equal(t, int64(3), getEnvAsInt64("RC_ENGINE_INT64", 3))
	assert.InDelta(t, 1.5, getEnvAsFloat("RC_ENGINE_FLOAT", 1.5), 1e-9)
	assert.Len(t, parseErrors, 2, "malformed values are reported")
}
