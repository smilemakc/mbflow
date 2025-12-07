package builtin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramParseExecutor_Execute_TextMessage(t *testing.T) {
	executor := NewTelegramParseExecutor()

	input := map[string]interface{}{
		"update_type": "message",
		"message": map[string]interface{}{
			"message_id": float64(42),
			"text":       "Hello, bot!",
			"from": map[string]interface{}{
				"id":            float64(123456),
				"username":      "john_doe",
				"first_name":    "John",
				"language_code": "en",
			},
			"chat": map[string]interface{}{
				"id":   float64(123456),
				"type": "private",
			},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "message", resultMap["update_type"])
	assert.Equal(t, "text", resultMap["message_type"])
	assert.Equal(t, "Hello, bot!", resultMap["text"])
	assert.Equal(t, 42, resultMap["message_id"])

	user := resultMap["user"].(map[string]interface{})
	assert.Equal(t, int64(123456), user["id"])
	assert.Equal(t, "john_doe", user["username"])

	chat := resultMap["chat"].(map[string]interface{})
	assert.Equal(t, int64(123456), chat["id"])
	assert.Equal(t, "private", chat["type"])
}

func TestTelegramParseExecutor_Execute_Command(t *testing.T) {
	executor := NewTelegramParseExecutor()

	input := map[string]interface{}{
		"update_type": "message",
		"message": map[string]interface{}{
			"message_id": float64(43),
			"text":       "/start arg1 arg2",
			"from": map[string]interface{}{
				"id": float64(123456),
			},
			"chat": map[string]interface{}{
				"id":   float64(123456),
				"type": "private",
			},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{
		"extract_commands": true,
	}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "/start", resultMap["command"])
	assert.Equal(t, []string{"arg1", "arg2"}, resultMap["command_args"])
}

func TestTelegramParseExecutor_Execute_CommandWithBotName(t *testing.T) {
	executor := NewTelegramParseExecutor()

	input := map[string]interface{}{
		"update_type": "message",
		"message": map[string]interface{}{
			"message_id": float64(44),
			"text":       "/help@mybot",
			"from": map[string]interface{}{
				"id": float64(123456),
			},
			"chat": map[string]interface{}{
				"id":   float64(-100123456),
				"type": "group",
			},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "/help", resultMap["command"])
}

func TestTelegramParseExecutor_Execute_PhotoMessage(t *testing.T) {
	executor := NewTelegramParseExecutor()

	input := map[string]interface{}{
		"update_type": "message",
		"message": map[string]interface{}{
			"message_id": float64(50),
			"caption":    "Check this out!",
			"photo": []interface{}{
				map[string]interface{}{
					"file_id":        "small-photo-id",
					"file_unique_id": "small-unique",
					"width":          float64(320),
					"height":         float64(240),
					"file_size":      float64(1024),
				},
				map[string]interface{}{
					"file_id":        "large-photo-id",
					"file_unique_id": "large-unique",
					"width":          float64(1280),
					"height":         float64(720),
					"file_size":      float64(12345),
				},
			},
			"from": map[string]interface{}{
				"id": float64(123456),
			},
			"chat": map[string]interface{}{
				"id":   float64(123456),
				"type": "private",
			},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "photo", resultMap["message_type"])
	assert.Equal(t, "Check this out!", resultMap["text"])

	files := resultMap["files"].([]map[string]interface{})
	require.Len(t, files, 1) // Only largest photo

	assert.Equal(t, "photo", files[0]["type"])
	assert.Equal(t, "large-photo-id", files[0]["file_id"])
	assert.Equal(t, 1280, files[0]["width"])
	assert.Equal(t, 720, files[0]["height"])
}

func TestTelegramParseExecutor_Execute_DocumentMessage(t *testing.T) {
	executor := NewTelegramParseExecutor()

	input := map[string]interface{}{
		"update_type": "message",
		"message": map[string]interface{}{
			"message_id": float64(51),
			"document": map[string]interface{}{
				"file_id":        "doc-file-id",
				"file_unique_id": "doc-unique",
				"file_name":      "report.pdf",
				"mime_type":      "application/pdf",
				"file_size":      float64(654321),
			},
			"from": map[string]interface{}{
				"id": float64(123456),
			},
			"chat": map[string]interface{}{
				"id":   float64(123456),
				"type": "private",
			},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "document", resultMap["message_type"])

	files := resultMap["files"].([]map[string]interface{})
	require.Len(t, files, 1)

	assert.Equal(t, "document", files[0]["type"])
	assert.Equal(t, "doc-file-id", files[0]["file_id"])
	assert.Equal(t, "report.pdf", files[0]["file_name"])
	assert.Equal(t, "application/pdf", files[0]["mime_type"])
}

func TestTelegramParseExecutor_Execute_CallbackQuery(t *testing.T) {
	executor := NewTelegramParseExecutor()

	input := map[string]interface{}{
		"update_type": "callback_query",
		"callback_query": map[string]interface{}{
			"id":   "callback-id-123",
			"data": "action:like:item_456",
			"from": map[string]interface{}{
				"id":         float64(123456),
				"username":   "john_doe",
				"first_name": "John",
			},
			"message": map[string]interface{}{
				"message_id": float64(100),
				"chat": map[string]interface{}{
					"id":   float64(123456),
					"type": "private",
				},
			},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "callback_query", resultMap["update_type"])
	assert.Equal(t, "action:like:item_456", resultMap["callback_data"])
	assert.Equal(t, "callback-id-123", resultMap["callback_query_id"])

	user := resultMap["user"].(map[string]interface{})
	assert.Equal(t, int64(123456), user["id"])

	chat := resultMap["chat"].(map[string]interface{})
	assert.Equal(t, int64(123456), chat["id"])
}

func TestTelegramParseExecutor_Execute_MultipleFiles(t *testing.T) {
	executor := NewTelegramParseExecutor()

	// Message with photo and document (hypothetical, but tests multiple file extraction)
	input := map[string]interface{}{
		"update_type": "message",
		"message": map[string]interface{}{
			"message_id": float64(60),
			"photo": []interface{}{
				map[string]interface{}{
					"file_id":        "photo-id",
					"file_unique_id": "photo-unique",
					"width":          float64(800),
					"height":         float64(600),
				},
			},
			"video": map[string]interface{}{
				"file_id":        "video-id",
				"file_unique_id": "video-unique",
				"duration":       float64(30),
				"width":          float64(1920),
				"height":         float64(1080),
			},
			"from": map[string]interface{}{"id": float64(123456)},
			"chat": map[string]interface{}{"id": float64(123456), "type": "private"},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	files := resultMap["files"].([]map[string]interface{})

	// Should extract both photo and video
	assert.Len(t, files, 2)

	fileTypes := make(map[string]bool)
	for _, f := range files {
		fileTypes[f["type"].(string)] = true
	}
	assert.True(t, fileTypes["photo"])
	assert.True(t, fileTypes["video"])
}

func TestTelegramParseExecutor_Execute_ExtractFilesDisabled(t *testing.T) {
	executor := NewTelegramParseExecutor()

	input := map[string]interface{}{
		"update_type": "message",
		"message": map[string]interface{}{
			"message_id": float64(70),
			"document": map[string]interface{}{
				"file_id": "doc-id",
			},
			"from": map[string]interface{}{"id": float64(123456)},
			"chat": map[string]interface{}{"id": float64(123456), "type": "private"},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{
		"extract_files": false,
	}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.NotContains(t, resultMap, "files")
}

func TestTelegramParseExecutor_Execute_ReplyToMessage(t *testing.T) {
	executor := NewTelegramParseExecutor()

	input := map[string]interface{}{
		"update_type": "message",
		"message": map[string]interface{}{
			"message_id": float64(80),
			"text":       "This is a reply",
			"reply_to_message": map[string]interface{}{
				"message_id": float64(75),
				"text":       "Original message",
			},
			"from": map[string]interface{}{"id": float64(123456)},
			"chat": map[string]interface{}{"id": float64(123456), "type": "private"},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 75, resultMap["reply_to_message_id"])
}

func TestTelegramParseExecutor_Execute_ExtractEntities(t *testing.T) {
	executor := NewTelegramParseExecutor()

	input := map[string]interface{}{
		"update_type": "message",
		"message": map[string]interface{}{
			"message_id": float64(90),
			"text":       "Check https://example.com and email@test.com",
			"entities": []interface{}{
				map[string]interface{}{
					"type":   "url",
					"offset": float64(6),
					"length": float64(19),
				},
				map[string]interface{}{
					"type":   "email",
					"offset": float64(30),
					"length": float64(14),
				},
			},
			"from": map[string]interface{}{"id": float64(123456)},
			"chat": map[string]interface{}{"id": float64(123456), "type": "private"},
		},
	}

	result, err := executor.Execute(context.Background(), map[string]interface{}{
		"extract_entities": true,
	}, input)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	entities := resultMap["entities"].(map[string]interface{})

	urls := entities["urls"].([]string)
	assert.Contains(t, urls, "https://example.com")

	emails := entities["emails"].([]string)
	assert.Contains(t, emails, "email@test.com")
}

func TestTelegramParseExecutor_Execute_NilInput(t *testing.T) {
	executor := NewTelegramParseExecutor()

	result, err := executor.Execute(context.Background(), map[string]interface{}{}, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "unknown", resultMap["update_type"])
}

func TestTelegramParseExecutor_Validate(t *testing.T) {
	executor := NewTelegramParseExecutor()

	// All configs should be valid (all options are optional)
	err := executor.Validate(map[string]interface{}{})
	assert.NoError(t, err)

	err = executor.Validate(map[string]interface{}{
		"extract_files":    true,
		"extract_commands": false,
		"extract_entities": true,
	})
	assert.NoError(t, err)
}
