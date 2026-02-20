package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  HTTPConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid GET request",
			config: HTTPConfig{
				Method: "GET",
				URL:    "https://api.example.com",
			},
			wantErr: false,
		},
		{
			name: "valid POST with body",
			config: HTTPConfig{
				Method:  "POST",
				URL:     "https://api.example.com",
				Body:    map[string]string{"key": "value"},
				Headers: map[string]string{"Content-Type": "application/json"},
			},
			wantErr: false,
		},
		{
			name: "missing method",
			config: HTTPConfig{
				URL: "https://api.example.com",
			},
			wantErr: true,
			errMsg:  "method is required",
		},
		{
			name: "missing URL",
			config: HTTPConfig{
				Method: "GET",
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "invalid method",
			config: HTTPConfig{
				Method: "INVALID",
				URL:    "https://api.example.com",
			},
			wantErr: true,
			errMsg:  "invalid HTTP method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransformConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  TransformConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "passthrough type (default)",
			config:  TransformConfig{},
			wantErr: false,
		},
		{
			name: "template type with template",
			config: TransformConfig{
				Type:     "template",
				Template: "Hello {{.name}}",
			},
			wantErr: false,
		},
		{
			name: "template type missing template",
			config: TransformConfig{
				Type: "template",
			},
			wantErr: true,
			errMsg:  "template is required",
		},
		{
			name: "expression type with expression",
			config: TransformConfig{
				Type:       "expression",
				Expression: "input.value * 2",
			},
			wantErr: false,
		},
		{
			name: "expression type missing expression",
			config: TransformConfig{
				Type: "expression",
			},
			wantErr: true,
			errMsg:  "expression is required",
		},
		{
			name: "jq type with filter",
			config: TransformConfig{
				Type:   "jq",
				Filter: ".items[]",
			},
			wantErr: false,
		},
		{
			name: "jq type missing filter",
			config: TransformConfig{
				Type: "jq",
			},
			wantErr: true,
			errMsg:  "filter is required",
		},
		{
			name: "invalid type",
			config: TransformConfig{
				Type: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid transformation type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLLMConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  LLMConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid OpenAI config",
			config: LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				Prompt:   "Hello, how are you?",
			},
			wantErr: false,
		},
		{
			name: "valid Anthropic config",
			config: LLMConfig{
				Provider: "anthropic",
				Model:    "claude-3-opus",
				Messages: []LLMMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			config: LLMConfig{
				Model: "gpt-4",
			},
			wantErr: true,
			errMsg:  "provider is required",
		},
		{
			name: "missing model",
			config: LLMConfig{
				Provider: "openai",
			},
			wantErr: true,
			errMsg:  "model is required",
		},
		{
			name: "invalid provider",
			config: LLMConfig{
				Provider: "invalid",
				Model:    "some-model",
			},
			wantErr: true,
			errMsg:  "invalid LLM provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConditionalConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  ConditionalConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid simple condition",
			config: ConditionalConfig{
				Condition:  "input.value > 10",
				TrueValue:  "high",
				FalseValue: "low",
			},
			wantErr: false,
		},
		{
			name: "valid with branches",
			config: ConditionalConfig{
				Branches: []ConditionalBranch{
					{Condition: "input.value > 100", Value: "very_high"},
					{Condition: "input.value > 50", Value: "high"},
				},
				Default: "normal",
			},
			wantErr: false,
		},
		{
			name:    "missing condition and branches",
			config:  ConditionalConfig{},
			wantErr: true,
			errMsg:  "condition or branches is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMergeConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  MergeConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "default strategy (empty)",
			config:  MergeConfig{},
			wantErr: false,
		},
		{
			name: "concat strategy",
			config: MergeConfig{
				Strategy: "concat",
			},
			wantErr: false,
		},
		{
			name: "deep_merge strategy",
			config: MergeConfig{
				Strategy: "deep_merge",
			},
			wantErr: false,
		},
		{
			name: "invalid strategy",
			config: MergeConfig{
				Strategy: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid merge strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFileStorageConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  FileStorageConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid read operation",
			config: FileStorageConfig{
				Operation: "read",
				Path:      "/path/to/file",
			},
			wantErr: false,
		},
		{
			name: "valid write operation",
			config: FileStorageConfig{
				Operation: "write",
				Path:      "/path/to/file",
				Content:   "file content",
			},
			wantErr: false,
		},
		{
			name: "valid list operation",
			config: FileStorageConfig{
				Operation: "list",
			},
			wantErr: false,
		},
		{
			name:    "missing operation",
			config:  FileStorageConfig{},
			wantErr: true,
			errMsg:  "operation is required",
		},
		{
			name: "invalid operation",
			config: FileStorageConfig{
				Operation: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid file storage operation",
		},
		{
			name: "read without path",
			config: FileStorageConfig{
				Operation: "read",
			},
			wantErr: true,
			errMsg:  "path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	t.Parallel()

	t.Run("parse HTTP config", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{
			"method":  "POST",
			"url":     "https://api.example.com",
			"headers": map[string]any{"Content-Type": "application/json"},
			"timeout": 30,
		}

		cfg, err := ParseConfig[HTTPConfig](input)
		require.NoError(t, err)
		assert.Equal(t, "POST", cfg.Method)
		assert.Equal(t, "https://api.example.com", cfg.URL)
		assert.Equal(t, 30, cfg.Timeout)
	})

	t.Run("parse Transform config", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{
			"type":   "jq",
			"filter": ".items[]",
		}

		cfg, err := ParseConfig[TransformConfig](input)
		require.NoError(t, err)
		assert.Equal(t, "jq", cfg.Type)
		assert.Equal(t, ".items[]", cfg.Filter)
	})

	t.Run("parse LLM config", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{
			"provider":    "openai",
			"model":       "gpt-4",
			"temperature": 0.7,
			"max_tokens":  1000,
		}

		cfg, err := ParseConfig[LLMConfig](input)
		require.NoError(t, err)
		assert.Equal(t, "openai", cfg.Provider)
		assert.Equal(t, "gpt-4", cfg.Model)
		assert.Equal(t, 0.7, cfg.Temperature)
		assert.Equal(t, 1000, cfg.MaxTokens)
	})
}

func TestToMap(t *testing.T) {
	t.Parallel()

	t.Run("convert HTTP config to map", func(t *testing.T) {
		t.Parallel()

		cfg := HTTPConfig{
			Method:  "GET",
			URL:     "https://api.example.com",
			Timeout: 30,
		}

		result, err := ToMap(cfg)
		require.NoError(t, err)
		assert.Equal(t, "GET", result["method"])
		assert.Equal(t, "https://api.example.com", result["url"])
		assert.Equal(t, float64(30), result["timeout"])
	})

	t.Run("convert LLM config to map", func(t *testing.T) {
		t.Parallel()

		cfg := LLMConfig{
			Provider:    "anthropic",
			Model:       "claude-3",
			Temperature: 0.5,
		}

		result, err := ToMap(cfg)
		require.NoError(t, err)
		assert.Equal(t, "anthropic", result["provider"])
		assert.Equal(t, "claude-3", result["model"])
		assert.Equal(t, 0.5, result["temperature"])
	})
}

func TestHTTPConfig_WithAuth(t *testing.T) {
	t.Parallel()

	t.Run("bearer auth", func(t *testing.T) {
		t.Parallel()

		cfg := HTTPConfig{
			Method: "GET",
			URL:    "https://api.example.com",
			Auth: &HTTPAuthConfig{
				Type:  "bearer",
				Token: "my-token",
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
		assert.NotNil(t, cfg.Auth)
		assert.Equal(t, "bearer", cfg.Auth.Type)
		assert.Equal(t, "my-token", cfg.Auth.Token)
	})

	t.Run("basic auth", func(t *testing.T) {
		t.Parallel()

		cfg := HTTPConfig{
			Method: "POST",
			URL:    "https://api.example.com",
			Auth: &HTTPAuthConfig{
				Type:     "basic",
				Username: "user",
				Password: "pass",
			},
		}

		err := cfg.Validate()
		assert.NoError(t, err)
		assert.NotNil(t, cfg.Auth)
		assert.Equal(t, "basic", cfg.Auth.Type)
	})
}

func TestLLMConfig_WithTools(t *testing.T) {
	t.Parallel()

	cfg := LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Tools: []LLMTool{
			{
				Type: "function",
				Function: LLMToolFunction{
					Name:        "get_weather",
					Description: "Get current weather",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{
								"type":        "string",
								"description": "City name",
							},
						},
					},
				},
			},
		},
		ToolChoice: "auto",
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Len(t, cfg.Tools, 1)
	assert.Equal(t, "get_weather", cfg.Tools[0].Function.Name)
	assert.Equal(t, "auto", cfg.ToolChoice)
}
