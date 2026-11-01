package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smilemakc/mbflow/go/pkg/executor"
	"github.com/smilemakc/mbflow/go/pkg/models"
)

// --- function_call ---

func TestRC_FunctionCall_HTTPRequestSendsBody(t *testing.T) {
	type seen struct {
		method, contentType, auth string
		body                      string
	}
	got := make(chan seen, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- seen{r.Method, r.Header.Get("Content-Type"), r.Header.Get("Authorization"), string(b)}
		switch r.URL.Path {
		case "/text":
			_, _ = w.Write([]byte("plain text"))
		case "/empty":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
		}
	}))
	defer srv.Close()

	e := NewFunctionCallExecutor()
	call := func(args map[string]any) *models.FunctionCallOutput {
		t.Helper()
		raw, _ := json.Marshal(args)
		out, err := e.Execute(context.Background(), map[string]any{"function_name": "http_request", "arguments": string(raw)}, nil)
		require.NoError(t, err)
		return out.(*models.FunctionCallOutput)
	}

	out := call(map[string]any{"method": "POST", "url": srv.URL + "/items", "body": map[string]any{"name": "x"},
		"headers": map[string]any{"Authorization": "Bearer t", "X-Num": 1}})
	require.True(t, out.Success, out.Error)
	s := <-got
	assert.Equal(t, "POST", s.method)
	assert.Equal(t, `{"name":"x"}`, s.body, "the JSON body reaches the server")
	assert.Equal(t, "application/json", s.contentType)
	assert.Equal(t, "Bearer t", s.auth)
	assert.Equal(t, map[string]any{"status": http.StatusCreated, "body": map[string]any{"id": float64(1)}}, out.Result)

	out = call(map[string]any{"method": "PUT", "url": srv.URL + "/text", "body": "raw-string",
		"headers": map[string]any{"Content-Type": "text/plain"}})
	require.True(t, out.Success)
	s = <-got
	assert.Equal(t, "raw-string", s.body)
	assert.Equal(t, "text/plain", s.contentType, "explicit headers win")
	assert.Equal(t, "plain text", out.Result.(map[string]any)["body"])

	out = call(map[string]any{"url": srv.URL + "/empty"})
	require.True(t, out.Success)
	s = <-got
	assert.Equal(t, "GET", s.method)
	assert.Empty(t, s.contentType)
	assert.Nil(t, out.Result.(map[string]any)["body"])

	assert.False(t, call(map[string]any{}).Success)
	assert.Contains(t, call(map[string]any{"url": "http://[::1"}).Error, "failed to create request")
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()
	assert.Contains(t, call(map[string]any{"url": closedURL}).Error, "request failed")
}

func TestRC_FunctionCall_BuiltinFunctions(t *testing.T) {
	e := NewFunctionCallExecutor()
	run := func(name, args string) *models.FunctionCallOutput {
		out, err := e.Execute(context.Background(), map[string]any{"function_name": name, "arguments": args, "tool_call_id": "tc"}, nil)
		require.NoError(t, err)
		return out.(*models.FunctionCallOutput)
	}

	for _, format := range []string{"RFC3339", "unix", "iso8601", "2006"} {
		out := run("get_current_time", `{"format":"`+format+`"}`)
		assert.True(t, out.Success, format)
		assert.NotNil(t, out.Result)
	}
	assert.Len(t, run("get_current_date", `{}`).Result, 10)

	assert.Equal(t, map[string]any{"a": float64(1)}, run("json_parse", `{"json":"{\"a\":1}"}`).Result)
	assert.Contains(t, run("json_parse", `{}`).Error, "json parameter is required")
	assert.Contains(t, run("json_parse", `{"json":"{"}`).Error, "failed to parse JSON")

	assert.Equal(t, `{"a":1}`, run("json_stringify", `{"value":{"a":1}}`).Result)
	assert.Equal(t, "{\n  \"a\": 1\n}", run("json_stringify", `{"value":{"a":1},"pretty":true}`).Result)
	assert.Contains(t, run("json_stringify", `{}`).Error, "value parameter is required")

	weather := run("get_weather", `{"location":"Paris","unit":"fahrenheit"}`)
	require.True(t, weather.Success)
	assert.Equal(t, "fahrenheit", weather.Result.(map[string]any)["unit"])
	assert.Equal(t, "tc", weather.ToolCallID)
	assert.Contains(t, run("get_weather", `{}`).Error, "location is required")

	assert.Contains(t, run("no_such_fn", `{}`).Error, "function not found")
}

func TestRC_FunctionCall_InputFormats(t *testing.T) {
	e := NewFunctionCallExecutor()
	e.RegisterFunction("echo", func(args map[string]any) (any, error) { return args, nil })
	assert.Contains(t, e.ListFunctions(), "echo")

	cases := map[string]struct {
		input   any
		want    any
		id      string
		wantErr string
	}{
		"llm tool calls": {input: map[string]any{"tool_calls": []any{map[string]any{"id": "t1",
			"function": map[string]any{"name": "echo", "arguments": `{"x":1}`}}}}, want: map[string]any{"x": float64(1)}, id: "t1"},
		"direct": {input: map[string]any{"function_name": "echo", "arguments": `{"y":2}`, "tool_call_id": "t2"},
			want: map[string]any{"y": float64(2)}, id: "t2"},
		"function field": {input: map[string]any{"id": "t3", "function": map[string]any{"name": "echo", "arguments": `{}`}},
			want: map[string]any{}, id: "t3"},
		"struct pointer": {input: &models.FunctionCallInput{FunctionName: "echo", Arguments: `{"z":3}`},
			want: map[string]any{"z": float64(3)}},
		"json string":           {input: `{"function_name":"echo"}`, want: map[string]any{}},
		"bad json string":       {input: `{`, wantErr: "failed to parse input JSON"},
		"unsupported":           {input: 42, wantErr: "unsupported input format"},
		"empty map":             {input: map[string]any{}, wantErr: "unsupported input format"},
		"tool call not object":  {input: map[string]any{"tool_calls": []any{"x"}}, wantErr: "invalid tool call format"},
		"tool call no function": {input: map[string]any{"tool_calls": []any{map[string]any{"id": "1"}}}, wantErr: "missing function field"},
		"function no name":      {input: map[string]any{"function": map[string]any{}}, wantErr: "function name is required"},
		"function bad args": {input: map[string]any{"function": map[string]any{"name": "echo", "arguments": 1}},
			wantErr: "function arguments must be a string"},
		"no function name": {input: &models.FunctionCallInput{}, wantErr: "function_name is required"},
		"invalid arguments": {input: map[string]any{"function_name": "echo", "arguments": "{bad"},
			wantErr: "failed to parse function arguments"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			out, err := e.Execute(context.Background(), map[string]any{}, tc.input)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			res := out.(*models.FunctionCallOutput)
			require.True(t, res.Success, res.Error)
			assert.Equal(t, tc.want, res.Result)
			assert.Equal(t, tc.id, res.ToolCallID)
		})
	}

	e.UnregisterFunction("echo")
	assert.NotContains(t, e.ListFunctions(), "echo")
	require.Error(t, e.Validate(map[string]any{"function_name": ""}))
	require.NoError(t, e.Validate(map[string]any{}))
}

// --- telegram_parse ---

func TestRC_TelegramParse_UpdateTypes(t *testing.T) {
	e := NewTelegramParseExecutor()
	ctx := context.Background()
	msg := func(extra map[string]any) map[string]any {
		m := map[string]any{"message_id": float64(10), "chat": map[string]any{"id": float64(-5), "type": "channel", "title": "T", "username": "chan"},
			"from": map[string]any{"id": float64(7), "username": "u", "first_name": "F", "last_name": "L", "language_code": "en", "is_bot": false}}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}

	for _, kind := range []string{"message", "edited_message", "channel_post", "edited_channel_post"} {
		t.Run("preparsed "+kind, func(t *testing.T) {
			out, err := e.Execute(ctx, nil, map[string]any{"update_type": kind, kind: msg(map[string]any{"text": "hi"})})
			require.NoError(t, err)
			res := out.(map[string]any)
			assert.Equal(t, kind, res["update_type"])
			assert.Equal(t, "text", res["message_type"])
			assert.Equal(t, 10, res["message_id"])
			assert.Equal(t, map[string]any{"id": int64(-5), "type": "channel", "title": "T", "username": "chan"}, res["chat"])
			user := res["user"].(map[string]any)
			assert.Equal(t, "L", user["last_name"])
			assert.Equal(t, "en", user["language_code"])
		})
		t.Run("detected "+kind, func(t *testing.T) {
			out, err := e.Execute(ctx, nil, map[string]any{kind: msg(map[string]any{"caption": "c", "poll": map[string]any{}})})
			require.NoError(t, err)
			res := out.(map[string]any)
			assert.Equal(t, kind, res["update_type"])
			assert.Equal(t, "poll", res["message_type"])
			assert.Equal(t, "c", res["text"])
		})
	}

	out, err := e.Execute(ctx, nil, map[string]any{"update_type": "callback_query",
		"callback_query": map[string]any{"id": "cb1", "data": "yes", "from": map[string]any{"id": float64(1)},
			"message": map[string]any{"chat": map[string]any{"id": float64(2)}}}})
	require.NoError(t, err)
	res := out.(map[string]any)
	assert.Equal(t, "yes", res["callback_data"])
	assert.Equal(t, "cb1", res["callback_query_id"])
	assert.Equal(t, int64(2), res["chat"].(map[string]any)["id"])

	for _, input := range []map[string]any{
		{"update_type": "inline_query", "inline_query": map[string]any{"id": "iq", "query": "cats", "from": map[string]any{"id": float64(3)}}},
		{"inline_query": map[string]any{"id": "iq", "query": "cats", "from": map[string]any{"id": float64(3)}}},
	} {
		out, err = e.Execute(ctx, nil, input)
		require.NoError(t, err)
		res = out.(map[string]any)
		assert.Equal(t, "inline_query", res["update_type"])
		assert.Equal(t, "cats", res["query"])
		assert.Equal(t, "iq", res["inline_query_id"])
		assert.Equal(t, int64(3), res["user"].(map[string]any)["id"])
	}

	out, err = e.Execute(ctx, nil, map[string]any{"callback_query": map[string]any{"data": "x"}})
	require.NoError(t, err)
	assert.Equal(t, "callback_query", out.(map[string]any)["update_type"])

	// Pre-parsed type without matching payload.
	out, err = e.Execute(ctx, nil, map[string]any{"update_type": "message"})
	require.NoError(t, err)
	assert.NotContains(t, out.(map[string]any), "message_type")
	out, err = e.Execute(ctx, nil, map[string]any{"update_type": "my_chat_member"})
	require.NoError(t, err)
	assert.Equal(t, "my_chat_member", out.(map[string]any)["update_type"])

	out, err = e.Execute(ctx, nil, map[string]any{"poll_answer": map[string]any{}})
	require.NoError(t, err)
	assert.Equal(t, "unknown", out.(map[string]any)["update_type"])
}

func TestRC_TelegramParse_MessageTypesAndFiles(t *testing.T) {
	e := NewTelegramParseExecutor()
	file := map[string]any{"file_id": "f", "file_unique_id": "u", "file_size": float64(9), "file_name": "n",
		"mime_type": "m", "width": float64(1), "height": float64(2), "duration": float64(3)}
	for _, kind := range []string{"document", "audio", "video", "voice", "video_note", "sticker", "animation", "location", "contact"} {
		out, err := e.Execute(context.Background(), nil, map[string]any{"message": map[string]any{kind: file}})
		require.NoError(t, err, kind)
		res := out.(map[string]any)
		assert.Equal(t, kind, res["message_type"], kind)
		if kind == "location" || kind == "contact" {
			continue
		}
		files := res["files"].([]map[string]any)
		require.Len(t, files, 1, kind)
		assert.Equal(t, map[string]any{"type": kind, "file_id": "f", "file_unique_id": "u", "file_size": 9, "file_name": "n",
			"mime_type": "m", "width": 1, "height": 2, "duration": 3}, files[0])
	}

	out, err := e.Execute(context.Background(), nil, map[string]any{"message": map[string]any{}})
	require.NoError(t, err)
	assert.Equal(t, "unknown", out.(map[string]any)["message_type"])

	// Photo with non-object last size yields no files.
	out, err = e.Execute(context.Background(), nil, map[string]any{"message": map[string]any{"photo": []any{"x"}}})
	require.NoError(t, err)
	assert.NotContains(t, out.(map[string]any), "files")
}

func TestRC_TelegramParse_EntitiesAndCommands(t *testing.T) {
	e := NewTelegramParseExecutor()
	text := "/start@bot arg1 hi @alice mail me@x.io http://a.b"
	out, err := e.Execute(context.Background(), map[string]any{"extract_entities": true}, map[string]any{
		"message": map[string]any{
			"text": text,
			"entities": []any{
				"skip",
				map[string]any{"type": "mention", "offset": float64(19), "length": float64(6)},
				map[string]any{"type": "email", "offset": float64(31), "length": float64(7)},
				map[string]any{"type": "url", "offset": float64(39), "length": float64(10)},
				map[string]any{"type": "text_mention", "offset": float64(16), "length": float64(2)},
				map[string]any{"type": "bold", "offset": float64(0), "length": float64(1)},
				map[string]any{"type": "url", "offset": float64(45), "length": float64(100)},
				map[string]any{"type": "url"}, // missing offset/length must not panic
				map[string]any{"type": "url", "offset": 1, "length": 2},
				map[string]any{"type": "url", "offset": float64(-1), "length": float64(2)},
			},
			"reply_to_message": map[string]any{"message_id": float64(99)},
		},
	})
	require.NoError(t, err)
	res := out.(map[string]any)
	assert.Equal(t, "/start", res["command"])
	assert.Equal(t, []string{"arg1", "hi", "@alice", "mail", "me@x.io", "http://a.b"}, res["command_args"])
	assert.Equal(t, 99, res["reply_to_message_id"])
	ents := res["entities"].(map[string]any)
	assert.Equal(t, []string{"@alice", "hi"}, ents["mentions"])
	assert.Equal(t, []string{"me@x.io"}, ents["emails"])
	assert.Equal(t, []string{"http://a.b"}, ents["urls"])

	// caption_entities and regex URL fallback.
	out, err = e.Execute(context.Background(), map[string]any{"extract_entities": true, "extract_commands": false},
		map[string]any{"message": map[string]any{"caption": "/cmd see https://go.dev now", "photo": []any{map[string]any{"file_id": "p"}},
			"caption_entities": []any{}}})
	require.NoError(t, err)
	res = out.(map[string]any)
	assert.NotContains(t, res, "command")
	assert.Equal(t, []string{"https://go.dev"}, res["entities"].(map[string]any)["urls"])

	// No text: entities are empty lists.
	assert.Equal(t, []string{}, e.extractEntities(map[string]any{})["urls"])
	cmd, args := e.extractCommand("/")
	assert.Equal(t, "/", cmd)
	assert.Empty(t, args)
}

// --- html_clean fallback ---

func TestRC_HTMLClean_FallbackExtraction(t *testing.T) {
	e := NewHTMLCleanExecutor()
	html := `<html><head><title>Doc</title><meta name="author" content="Ann"><meta name="description" content="About"></head>
<body><nav>menu</nav><article><p>First   words here</p><p>More text</p></article></body></html>`

	out, err := e.fallbackExtraction(html, "both", true, false, 0)
	require.NoError(t, err)
	assert.Equal(t, "Doc", out["title"])
	assert.Equal(t, "Ann", out["author"])
	assert.Equal(t, "About", out["excerpt"])
	assert.Contains(t, out["text_content"], "First words here")
	assert.NotContains(t, out["text_content"], "menu")
	assert.Contains(t, out["html_content"], "<p>")
	assert.Equal(t, true, out["is_html"])

	out, err = e.fallbackExtraction(html, "text", false, false, 10)
	require.NoError(t, err)
	assert.Empty(t, out["html_content"])
	assert.Empty(t, out["title"])
	assert.LessOrEqual(t, len(out["text_content"].(string)), 13)

	out, err = e.fallbackExtraction(`<body><div>only body text</div></body>`, "html", false, false, 5)
	require.NoError(t, err)
	assert.Empty(t, out["text_content"])
	assert.NotEmpty(t, out["html_content"])

	assert.Equal(t, "[https://x](https://x)", strings.TrimSpace(e.convertLinksToMarkdown(`<a href="https://x"></a>`)))
}

// --- http ---

func TestRC_HTTPExecutor_ExecuteAndValidate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		w.Header().Set("X-CT", r.Header.Get("Content-Type"))
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	e := NewHTTPExecutor()

	out, err := e.Execute(context.Background(), map[string]any{"method": "POST", "url": srv.URL, "body": []byte(`{"a":1}`)}, nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"a": float64(1)}, out.(map[string]any)["body"])

	_, err = e.Execute(context.Background(), map[string]any{"method": "POST", "url": srv.URL, "body": map[string]any{"c": make(chan int)}}, nil)
	require.ErrorContains(t, err, "failed to marshal request body")
	_, err = e.Execute(context.Background(), map[string]any{"url": srv.URL}, nil)
	require.Error(t, err)
	_, err = e.Execute(context.Background(), map[string]any{"method": "GET"}, nil)
	require.Error(t, err)
	_, err = e.Execute(context.Background(), map[string]any{"method": "BAD METHOD", "url": srv.URL}, nil)
	require.ErrorContains(t, err, "failed to create request")
	closed := httptest.NewServer(http.NotFoundHandler())
	u := closed.URL
	closed.Close()
	_, err = e.Execute(context.Background(), map[string]any{"method": "GET", "url": u}, nil)
	require.ErrorContains(t, err, "request failed")

	require.NoError(t, e.Validate(map[string]any{"method": "GET", "url": "http://x", "ignore_status_errors": true,
		"success_status_codes": []any{float64(200), 404}}))
	for name, cfg := range map[string]map[string]any{
		"missing":          {"method": "GET"},
		"method type":      {"method": 1, "url": "u"},
		"bad method":       {"method": "FETCH", "url": "u"},
		"url type":         {"method": "GET", "url": 1},
		"empty url":        {"method": "GET", "url": ""},
		"ignore not bool":  {"method": "GET", "url": "u", "ignore_status_errors": "yes"},
		"codes wrong type": {"method": "GET", "url": "u", "success_status_codes": "200"},
		"codes range":      {"method": "GET", "url": "u", "success_status_codes": []int{99}},
	} {
		assert.Error(t, e.Validate(cfg), name)
	}
}

// --- transform ---

func TestRC_TransformExecutor_Errors(t *testing.T) {
	e := NewTransformExecutor()
	ctx := context.Background()
	for name, cfg := range map[string]map[string]any{
		"template missing":   {"type": "template"},
		"expression missing": {"type": "expression"},
		"expression compile": {"type": "expression", "expression": "input +"},
		"expression runtime": {"type": "expression", "expression": "input.a.b"},
		"jq missing":         {"type": "jq"},
		"jq parse":           {"type": "jq", "filter": ".["},
		"jq compile":         {"type": "jq", "filter": "nofn(1)"},
		"jq no output":       {"type": "jq", "filter": "empty"},
		"jq runtime":         {"type": "jq", "filter": "error(\"x\")"},
	} {
		_, err := e.Execute(ctx, cfg, map[string]any{"a": nil})
		assert.Error(t, err, name)
	}

	out, err := e.Execute(ctx, map[string]any{"type": "jq", "filter": ".a"}, []byte(`{"a":5}`))
	require.NoError(t, err)
	assert.Equal(t, float64(5), out)
	out, err = e.Execute(ctx, map[string]any{"type": "jq", "filter": "."}, []byte(`not json`))
	require.NoError(t, err)
	assert.Equal(t, "not json", out)
	out, err = e.Execute(ctx, map[string]any{"type": "jq", "filter": "."}, "plain")
	require.NoError(t, err)
	assert.Equal(t, "plain", out)
}

// --- base64 adapters ---

func TestRC_Base64Adapters(t *testing.T) {
	dec := NewBase64ToBytesExecutor()
	ctx := context.Background()
	raw := []byte{0xfb, 0xff, 0x01}
	for enc, s := range map[string]string{
		"url":          base64.URLEncoding.EncodeToString(raw),
		"raw_standard": base64.RawStdEncoding.EncodeToString(raw),
		"raw_url":      base64.RawURLEncoding.EncodeToString(raw),
		"auto":         base64.RawURLEncoding.EncodeToString(raw),
	} {
		out, err := dec.Execute(ctx, map[string]any{"encoding": enc, "output_format": "hex"}, s)
		require.NoError(t, err, enc)
		assert.Equal(t, "fbff01", out.(map[string]any)["result"], enc)
	}
	_, err := dec.Execute(ctx, map[string]any{"encoding": "auto"}, "!!!")
	require.ErrorContains(t, err, "failed to auto-detect encoding")
	_, err = dec.Execute(ctx, map[string]any{}, "!!!")
	require.ErrorContains(t, err, "decoding failed")
	_, err = dec.Execute(ctx, map[string]any{"output_format": "bin"}, "aGk=")
	require.ErrorContains(t, err, "invalid output_format")
	out, err := dec.Execute(ctx, map[string]any{}, map[string]any{"base64": " aGk= "})
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), out.(map[string]any)["result"])
	_, err = dec.Execute(ctx, map[string]any{}, map[string]any{})
	require.ErrorContains(t, err, "expected 'data' or 'base64'")
	_, err = dec.Execute(ctx, map[string]any{}, 5)
	require.ErrorContains(t, err, "unsupported input type")

	enc := NewBytesToBase64Executor()
	for _, name := range []string{"url", "raw_standard", "raw_url"} {
		out, err := enc.Execute(ctx, map[string]any{"encoding": name}, raw)
		require.NoError(t, err)
		assert.NotEmpty(t, out.(map[string]any)["result"])
	}
	_, err = enc.Execute(ctx, map[string]any{"encoding": "b32"}, raw)
	require.ErrorContains(t, err, "invalid encoding")
	_, err = enc.Execute(ctx, map[string]any{}, map[string]any{})
	require.ErrorContains(t, err, "expected 'data' field")
	_, err = enc.Execute(ctx, map[string]any{}, 1.5)
	require.ErrorContains(t, err, "unsupported input type")
	out, err = enc.Execute(ctx, map[string]any{}, map[string]any{"data": "not base64!"})
	require.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("not base64!")), out.(map[string]any)["result"])
}

// --- registration ---

// rcFailingManager rejects every registration.
type rcFailingManager struct{ executor.Manager }

func (rcFailingManager) Register(string, executor.Executor) error {
	return errors.New("registry closed")
}

func TestRC_RegisterHelpers(t *testing.T) {
	m := executor.NewManager()
	storage := newMockManager()
	require.NotPanics(t, func() { MustRegisterBuiltins(m) })
	require.NotPanics(t, func() { MustRegisterAdapters(m) })
	require.NotPanics(t, func() { MustRegisterFileStorage(m, storage) })
	require.NotPanics(t, func() { MustRegisterFileAdapters(m, storage) })
	for _, name := range []string{"http", "llm", "google_drive", "csv_to_json", "file_storage", "file_to_bytes", "bytes_to_file"} {
		assert.True(t, m.Has(name), name)
	}
	require.NoError(t, RegisterOpenAPI(m, fakeAPIClients{}, nil))
	llm, err := m.Get("llm")
	require.NoError(t, err)
	assert.NotNil(t, llm.(*LLMExecutor).toolRegistry().openAPI(), "llm gets the API operations as tools")

	bad := rcFailingManager{Manager: executor.NewManager()}
	assert.PanicsWithValue(t, "failed to register built-in executors: registry closed", func() { MustRegisterBuiltins(bad) })
	assert.PanicsWithValue(t, "failed to register adapter executors: registry closed", func() { MustRegisterAdapters(bad) })
	assert.PanicsWithValue(t, "failed to register file_storage executor: registry closed", func() { MustRegisterFileStorage(bad, storage) })
	assert.PanicsWithValue(t, "failed to register file adapter executors: registry closed", func() { MustRegisterFileAdapters(bad, storage) })
	require.ErrorContains(t, RegisterOpenAPI(bad, fakeAPIClients{}, nil), "registry closed")
}
