package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rcTelegramCall struct {
	Path   string
	JSON   map[string]any
	Fields map[string]string
	Files  map[string][]byte
}

// rcTelegramServer records Bot API calls and answers with the response
// produced by reply for the method (last path segment).
func rcTelegramServer(t *testing.T, reply func(method string) string) (*TelegramExecutor, func() []rcTelegramCall) {
	t.Helper()
	var mu sync.Mutex
	var calls []rcTelegramCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := rcTelegramCall{Path: r.URL.Path, Fields: map[string]string{}, Files: map[string][]byte{}}
		mt, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if strings.HasPrefix(mt, "multipart/") {
			mr := multipart.NewReader(r.Body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err != nil {
					break
				}
				data, _ := io.ReadAll(p)
				if p.FileName() != "" {
					call.Files[p.FormName()+":"+p.FileName()] = data
				} else {
					call.Fields[p.FormName()] = string(data)
				}
			}
		} else {
			_ = json.NewDecoder(r.Body).Decode(&call.JSON)
		}
		mu.Lock()
		calls = append(calls, call)
		mu.Unlock()
		parts := strings.Split(r.URL.Path, "/")
		_, _ = w.Write([]byte(reply(parts[len(parts)-1])))
	}))
	t.Cleanup(srv.Close)
	e := NewTelegramExecutor()
	e.baseURL = srv.URL
	return e, func() []rcTelegramCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]rcTelegramCall(nil), calls...)
	}
}

func rcTelegramOK(method string) string {
	switch method {
	case "sendPhoto":
		return `{"ok":true,"result":{"message_id":2,"date":5,"chat":{"id":-100},"caption":"cap",
			"photo":[{"file_id":"small","file_unique_id":"s"},{"file_id":"big","file_unique_id":"b","file_size":900}]}}`
	case "sendDocument":
		return `{"ok":true,"result":{"message_id":3,"chat":{"id":1},"document":{"file_id":"doc","file_unique_id":"d","file_size":10}}}`
	case "sendAudio":
		return `{"ok":true,"result":{"message_id":4,"chat":{"id":1},"audio":{"file_id":"aud","file_unique_id":"a"}}}`
	case "sendVideo":
		return `{"ok":true,"result":{"message_id":5,"chat":{"id":1},"video":{"file_id":"vid","file_unique_id":"v","file_size":7}}}`
	default:
		return `{"ok":true,"result":{"message_id":1,"date":5,"chat":{"id":42},"text":"hello"}}`
	}
}

func rcTelegramConfig(extra map[string]any) map[string]any {
	cfg := map[string]any{"bot_token": "123:abc", "chat_id": "42"}
	for k, v := range extra {
		cfg[k] = v
	}
	return cfg
}

func TestRC_Telegram_ExecuteTextWithOptions(t *testing.T) {
	e, calls := rcTelegramServer(t, rcTelegramOK)
	out, err := e.Execute(context.Background(), rcTelegramConfig(map[string]any{
		"message_type":             "text",
		"text":                     "hello",
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
		"disable_notification":     true,
		"protect_content":          true,
		"reply_to_message_id":      7,
		"message_thread_id":        9,
	}), nil)
	require.NoError(t, err)
	res := out.(map[string]any)
	assert.Equal(t, true, res["success"])
	assert.Equal(t, "text", res["message_type"])
	assert.Equal(t, 1, res["message_id"])
	assert.Equal(t, int64(42), res["chat_id"])
	assert.Equal(t, "hello", res["text"])

	c := calls()[0]
	assert.Equal(t, "/bot123:abc/sendMessage", c.Path)
	assert.Equal(t, "HTML", c.JSON["parse_mode"])
	assert.Equal(t, true, c.JSON["disable_web_page_preview"])
	assert.Equal(t, true, c.JSON["disable_notification"])
	assert.Equal(t, true, c.JSON["protect_content"])
	assert.Equal(t, float64(7), c.JSON["reply_to_message_id"])
	assert.Equal(t, float64(9), c.JSON["message_thread_id"])
}

func TestRC_Telegram_ExecuteMediaByFileIDAndURL(t *testing.T) {
	e, calls := rcTelegramServer(t, rcTelegramOK)
	cases := []struct {
		messageType, source, method, fileID string
	}{
		{"photo", "file_id", "sendPhoto", "big"},
		{"photo", "url", "sendPhoto", "big"},
		{"document", "file_id", "sendDocument", "doc"},
		{"document", "url", "sendDocument", "doc"},
		{"audio", "file_id", "sendAudio", "aud"},
		{"audio", "url", "sendAudio", "aud"},
		{"video", "file_id", "sendVideo", "vid"},
		{"video", "url", "sendVideo", "vid"},
	}
	for i, tc := range cases {
		t.Run(tc.messageType+"_"+tc.source, func(t *testing.T) {
			out, err := e.Execute(context.Background(), rcTelegramConfig(map[string]any{
				"message_type": tc.messageType, "file_source": tc.source, "file_data": "DATA",
				"text": "cap", "parse_mode": "Markdown", "disable_notification": true, "protect_content": true,
				"reply_to_message_id": 1, "message_thread_id": 2,
			}), nil)
			require.NoError(t, err)
			res := out.(map[string]any)
			assert.Equal(t, tc.messageType, res["message_type"])
			assert.Equal(t, tc.fileID, res["file_id"])

			c := calls()[i]
			assert.True(t, strings.HasSuffix(c.Path, "/"+tc.method))
			assert.Equal(t, "DATA", c.JSON[tc.messageType])
			assert.Equal(t, "cap", c.JSON["caption"])
			assert.Equal(t, "Markdown", c.JSON["parse_mode"])
			assert.Equal(t, true, c.JSON["disable_notification"])
			assert.Equal(t, true, c.JSON["protect_content"])
			assert.Equal(t, float64(1), c.JSON["reply_to_message_id"])
			assert.Equal(t, float64(2), c.JSON["message_thread_id"])
		})
	}
	// file_size only appears when set
	out, err := e.Execute(context.Background(), rcTelegramConfig(map[string]any{
		"message_type": "photo", "file_source": "file_id", "file_data": "x"}), nil)
	require.NoError(t, err)
	assert.Equal(t, 900, out.(map[string]any)["file_size"])
	assert.Equal(t, "cap", out.(map[string]any)["caption"])
}

func TestRC_Telegram_ExecuteMediaBase64Multipart(t *testing.T) {
	e, calls := rcTelegramServer(t, rcTelegramOK)
	payload := base64.StdEncoding.EncodeToString([]byte("binary-bytes"))
	for _, mt := range []string{"photo", "document", "audio", "video"} {
		_, err := e.Execute(context.Background(), rcTelegramConfig(map[string]any{
			"message_type": mt, "file_source": "base64", "file_data": payload,
			"text": "caption", "parse_mode": "HTML", "disable_notification": true, "protect_content": true,
			"reply_to_message_id": 11, "message_thread_id": 12,
		}), nil)
		require.NoError(t, err, mt)
	}
	got := calls()
	require.Len(t, got, 4)
	defaults := map[string]string{"photo": "photo.jpg", "document": "document.pdf", "audio": "audio.mp3", "video": "video.mp4"}
	for i, mt := range []string{"photo", "document", "audio", "video"} {
		c := got[i]
		assert.Equal(t, []byte("binary-bytes"), c.Files[mt+":"+defaults[mt]], mt)
		assert.Equal(t, "42", c.Fields["chat_id"])
		assert.Equal(t, "caption", c.Fields["caption"])
		assert.Equal(t, "HTML", c.Fields["parse_mode"])
		assert.Equal(t, "true", c.Fields["disable_notification"])
		assert.Equal(t, "true", c.Fields["protect_content"])
		assert.Equal(t, "11", c.Fields["reply_to_message_id"])
		assert.Equal(t, "12", c.Fields["message_thread_id"])
	}

	// Custom file name.
	_, err := e.Execute(context.Background(), rcTelegramConfig(map[string]any{
		"message_type": "document", "file_source": "base64", "file_data": payload, "file_name": "r.txt"}), nil)
	require.NoError(t, err)
	assert.Contains(t, calls()[4].Files, "document:r.txt")
	assert.Equal(t, "file.bin", e.getDefaultFileName("sticker"))
}

func TestRC_Telegram_ExecuteErrors(t *testing.T) {
	e, _ := rcTelegramServer(t, func(method string) string {
		return `{"ok":false,"description":"Bad Request: chat not found","error_code":400}`
	})
	ctx := context.Background()

	out, err := e.Execute(ctx, rcTelegramConfig(map[string]any{"message_type": "text", "text": "x"}), nil)
	require.NoError(t, err, "API-level failures are reported in the output")
	res := out.(map[string]any)
	assert.Equal(t, false, res["success"])
	assert.Equal(t, "Bad Request: chat not found", res["error"])
	assert.Equal(t, 400, res["error_code"])

	_, err = e.Execute(ctx, rcTelegramConfig(map[string]any{"message_type": "sticker"}), nil)
	require.ErrorContains(t, err, "unsupported message_type: sticker")

	_, err = e.Execute(ctx, rcTelegramConfig(map[string]any{"message_type": "text"}), nil)
	require.ErrorContains(t, err, "text cannot be empty")

	for _, mt := range []string{"photo", "document", "audio", "video"} {
		_, err = e.Execute(ctx, rcTelegramConfig(map[string]any{"message_type": mt, "file_source": "ftp"}), nil)
		require.ErrorContains(t, err, "unsupported file_source: ftp", mt)
	}

	_, err = e.Execute(ctx, rcTelegramConfig(map[string]any{"message_type": "photo", "file_source": "base64", "file_data": "!!!"}), nil)
	require.ErrorContains(t, err, "failed to decode base64")
}

func TestRC_Telegram_ParseAPIResponseVariants(t *testing.T) {
	cases := map[string]struct {
		body    string
		wantErr string
		check   func(t *testing.T, res map[string]any)
	}{
		"malformed":         {body: `{`, wantErr: "failed to parse response"},
		"ok without result": {body: `{"ok":true}`, wantErr: "missing result"},
		"error without description": {body: `{"ok":false}`, check: func(t *testing.T, res map[string]any) {
			assert.Equal(t, "unknown error", res["error"])
			assert.NotContains(t, res, "error_code")
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e, _ := rcTelegramServer(t, func(string) string { return tc.body })
			out, err := e.Execute(context.Background(), rcTelegramConfig(map[string]any{"message_type": "text", "text": "x"}), nil)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			tc.check(t, out.(map[string]any))
		})
	}

	// Multipart path with malformed response.
	e, _ := rcTelegramServer(t, func(string) string { return `nope` })
	_, err := e.Execute(context.Background(), rcTelegramConfig(map[string]any{
		"message_type": "photo", "file_source": "base64", "file_data": "aGk="}), nil)
	require.ErrorContains(t, err, "failed to parse response")
}

func TestRC_Telegram_TransportErrors(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	e := NewTelegramExecutor()
	e.baseURL = url
	_, err := e.Execute(context.Background(), rcTelegramConfig(map[string]any{"message_type": "text", "text": "x"}), nil)
	require.ErrorContains(t, err, "request failed")
	_, err = e.Execute(context.Background(), rcTelegramConfig(map[string]any{
		"message_type": "photo", "file_source": "base64", "file_data": "aGk="}), nil)
	require.ErrorContains(t, err, "request failed")

	e.baseURL = "://bad"
	_, err = e.Execute(context.Background(), rcTelegramConfig(map[string]any{"message_type": "text", "text": "x"}), nil)
	require.ErrorContains(t, err, "failed to create request")
	_, err = e.Execute(context.Background(), rcTelegramConfig(map[string]any{
		"message_type": "photo", "file_source": "base64", "file_data": "aGk="}), nil)
	require.ErrorContains(t, err, "failed to create request")

	_, err = e.executeAPIRequest(context.Background(), "http://x", map[string]any{"c": make(chan int)}, 0)
	require.ErrorContains(t, err, "failed to marshal payload")
}

func TestRC_Telegram_Validate(t *testing.T) {
	e := NewTelegramExecutor()
	base := func(extra map[string]any) map[string]any {
		return rcTelegramConfig(extra)
	}
	require.NoError(t, e.Validate(base(map[string]any{"message_type": "text", "text": "hi"})))
	require.NoError(t, e.Validate(base(map[string]any{"message_type": "photo", "file_source": "url", "file_data": "u"})))

	cases := map[string]map[string]any{
		"missing":          {"bot_token": "123:abc"},
		"bad token":        {"bot_token": "abc", "chat_id": "1", "message_type": "text", "text": "x"},
		"token not number": {"bot_token": "x:y", "chat_id": "1", "message_type": "text", "text": "x"},
		"token empty part": {"bot_token": "1:", "chat_id": "1", "message_type": "text", "text": "x"},
		"bad type":         base(map[string]any{"message_type": "gif"}),
		"text missing":     base(map[string]any{"message_type": "text"}),
		"media missing":    base(map[string]any{"message_type": "video"}),
		"bad source":       base(map[string]any{"message_type": "video", "file_source": "ftp", "file_data": "x"}),
		"bad parse mode":   base(map[string]any{"message_type": "text", "text": "x", "parse_mode": "BBCode"}),
		"timeout too big":  base(map[string]any{"message_type": "text", "text": "x", "timeout": 1000}),
		"timeout zero":     base(map[string]any{"message_type": "text", "text": "x", "timeout": 0}),
		"chat id wrong":    {"bot_token": "1:a", "chat_id": 5, "message_type": "text", "text": "x"},
		"type wrong":       {"bot_token": "1:a", "chat_id": "5", "message_type": 5},
		"token wrong type": {"bot_token": 1, "chat_id": "5", "message_type": "text"},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, e.Validate(cfg))
		})
	}
}

func TestRC_TelegramParse_EntitiesUseUTF16Offsets(t *testing.T) {
	e := NewTelegramParseExecutor()
	// "Привет 👋 " is 9 UTF-16 code units long (the emoji takes two units).
	text := "Привет 👋 https://example.com @bob"
	msg := map[string]any{
		"text": text,
		"entities": []any{
			map[string]any{"type": "url", "offset": float64(10), "length": float64(19)},
			map[string]any{"type": "mention", "offset": float64(30), "length": float64(4)},
		},
	}
	got := e.extractEntities(msg)
	assert.Equal(t, []string{"https://example.com"}, got["urls"])
	assert.Equal(t, []string{"@bob"}, got["mentions"])
}
