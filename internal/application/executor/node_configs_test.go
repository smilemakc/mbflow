package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenAICompletionConfig_ToMap(t *testing.T) {
	config := &OpenAICompletionConfig{
		Model:       "gpt-4o",
		Prompt:      "Test prompt",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	result, err := config.ToMap()
	if err != nil {
		t.Fatalf("ToMap() error = %v", err)
	}

	if result["model"] != "gpt-4o" {
		t.Errorf("Expected model = gpt-4o, got %v", result["model"])
	}
	if result["prompt"] != "Test prompt" {
		t.Errorf("Expected prompt = Test prompt, got %v", result["prompt"])
	}
	if result["max_tokens"] != float64(100) {
		t.Errorf("Expected max_tokens = 100, got %v", result["max_tokens"])
	}
	if result["temperature"] != 0.7 {
		t.Errorf("Expected temperature = 0.7, got %v", result["temperature"])
	}
}

func TestHTTPRequestConfig_ToMap(t *testing.T) {
	config := &HTTPRequestConfig{
		URL:    "https://api.example.com",
		Method: "GET",
		Headers: map[string]string{
			"Authorization": "Bearer token",
			"Accept":        "application/json",
		},
	}

	result, err := config.ToMap()
	assert.NoError(t, err, "ToMap() error")

	if result["url"] != "https://api.example.com" {
		t.Errorf("Expected url = https://api.example.com, got %v", result["url"])
	}
	if result["method"] != "GET" {
		t.Errorf("Expected method = GET, got %v", result["method"])
	}

	headers, ok := result["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected headers to be map[string]interface{}")
	}
	if headers["Accept"] != "application/json" {
		t.Errorf("Expected Accept header = application/json, got %v", headers["Accept"])
	}
}

func TestJSONParserConfig_ToMap(t *testing.T) {
	config := &JSONParserConfig{
		FailOnError: true,
	}

	result, err := config.ToMap()
	if err != nil {
		t.Fatalf("ToMap() error = %v", err)
	}

	if result["fail_on_error"] != true {
		t.Errorf("Expected fail_on_error = true, got %v", result["fail_on_error"])
	}
}

func TestDataAggregatorConfig_ToMap(t *testing.T) {
	config := &DataAggregatorConfig{
		Fields: map[string]string{
			"name":  "user.name",
			"email": "user.email",
		},
		OutputFormat: "json",
	}

	result, err := config.ToMap()
	if err != nil {
		t.Fatalf("ToMap() error = %v", err)
	}

	fields, ok := result["fields"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected fields to be map[string]interface{}")
	}
	if fields["name"] != "user.name" {
		t.Errorf("Expected name field = user.name, got %v", fields["name"])
	}
	if fields["email"] != "user.email" {
		t.Errorf("Expected email field = user.email, got %v", fields["email"])
	}
}

func TestConditionalEdgeConfig_ToMap(t *testing.T) {
	config := &ConditionalEdgeConfig{
		Condition: "status == 'success'",
	}

	result, err := config.ToMap()
	if err != nil {
		t.Fatalf("ToMap() error = %v", err)
	}

	if result["condition"] != "status == 'success'" {
		t.Errorf("Expected condition = status == 'success', got %v", result["condition"])
	}
}

func TestTelegramMessageConfig_ToMap(t *testing.T) {
	config := &TelegramMessageConfig{
		BotToken:            "test_token",
		ChatID:              "@test_channel",
		Text:                "Test message",
		ParseMode:           "Markdown",
		DisableNotification: true,
	}

	result, err := config.ToMap()
	if err != nil {
		t.Fatalf("ToMap() error = %v", err)
	}

	if result["bot_token"] != "test_token" {
		t.Errorf("Expected bot_token = test_token, got %v", result["bot_token"])
	}
	if result["chat_id"] != "@test_channel" {
		t.Errorf("Expected chat_id = @test_channel, got %v", result["chat_id"])
	}
	if result["disable_notification"] != true {
		t.Errorf("Expected disable_notification = true, got %v", result["disable_notification"])
	}
}
