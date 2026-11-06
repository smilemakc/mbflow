package builtin

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============== Unit Tests with Mock Server ==============

func TestHTTPExecutor_BinaryResponse_Image(t *testing.T) {
	// Create mock server that returns an image
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(pngHeader)
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method": "GET",
		"url":    server.URL,
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, 200, resultMap["status"])
	assert.Equal(t, "image/png", resultMap["content_type"])
	assert.Nil(t, resultMap["body"])
	assert.NotEmpty(t, resultMap["body_base64"])
	assert.Equal(t, len(pngHeader), resultMap["size"])

	// Verify base64 decoding
	decoded, err := base64.StdEncoding.DecodeString(resultMap["body_base64"].(string))
	require.NoError(t, err)
	assert.Equal(t, pngHeader, decoded)
}

func TestHTTPExecutor_BinaryResponse_PDF(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 test content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write(pdfContent)
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method": "GET",
		"url":    server.URL,
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, "application/pdf", resultMap["content_type"])
	assert.NotEmpty(t, resultMap["body_base64"])

	decoded, err := base64.StdEncoding.DecodeString(resultMap["body_base64"].(string))
	require.NoError(t, err)
	assert.Equal(t, pdfContent, decoded)
}

func TestHTTPExecutor_BinaryResponse_ForceBinary(t *testing.T) {
	// Force binary mode even for text content
	textContent := "This is plain text"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(textContent))
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method":        "GET",
		"url":           server.URL,
		"response_type": "binary",
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.NotEmpty(t, resultMap["body_base64"])

	decoded, err := base64.StdEncoding.DecodeString(resultMap["body_base64"].(string))
	require.NoError(t, err)
	assert.Equal(t, textContent, string(decoded))
}

func TestHTTPExecutor_JSONResponse(t *testing.T) {
	jsonContent := `{"name": "test", "value": 42}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(jsonContent))
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method": "GET",
		"url":    server.URL,
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, 200, resultMap["status"])
	assert.Contains(t, resultMap["content_type"], "application/json")
	assert.NotNil(t, resultMap["body"])
	// body_base64 should not be present for JSON
	_, hasBase64 := resultMap["body_base64"]
	assert.False(t, hasBase64)

	body := resultMap["body"].(map[string]any)
	assert.Equal(t, "test", body["name"])
}

func TestHTTPExecutor_ContentType_Detection(t *testing.T) {
	testCases := []struct {
		contentType string
		isBinary    bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/gif", true},
		{"image/webp", true},
		{"image/svg+xml", true},
		{"audio/mpeg", true},
		{"audio/wav", true},
		{"video/mp4", true},
		{"application/pdf", true},
		{"application/octet-stream", true},
		{"application/zip", true},
		{"application/json", false},
		{"text/plain", false},
		{"text/html", false},
		{"text/xml", false},
	}

	for _, tc := range testCases {
		t.Run(tc.contentType, func(t *testing.T) {
			result := isBinaryContentType(tc.contentType)
			assert.Equal(t, tc.isBinary, result, "content type: %s", tc.contentType)
		})
	}
}

func TestHTTPExecutor_BinaryResponse_JPEG(t *testing.T) {
	// JPEG file signature: FF D8 FF
	jpegContent := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(jpegContent)
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method": "GET",
		"url":    server.URL,
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, "image/jpeg", resultMap["content_type"])
	assert.NotEmpty(t, resultMap["body_base64"])
}

// ============== Complete images over HTTP ==============
// Real, decodable images served locally, so the tests need no network.

// testImage encodes a small gradient image in the given format.
func testImage(t *testing.T, format string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 8))
	for x := 0; x < 16; x++ {
		for y := 0; y < 8; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 32), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	switch format {
	case "png":
		require.NoError(t, png.Encode(&buf, img))
	case "jpeg":
		require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}))
	default:
		t.Fatalf("unknown format %s", format)
	}
	return buf.Bytes()
}

// webp1x1 is a complete 1x1 lossless WebP image.
var webp1x1 = []byte("RIFF\x1a\x00\x00\x00WEBPVP8L\x0d\x00\x00\x00\x2f\x00\x00\x00\x10\x07\x10\x11\x11\x88\x88\xfe\x07\x00")

func serveImage(t *testing.T, contentType string, data []byte) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// fetchImage runs a GET through the HTTP executor and returns the result and
// the decoded body.
func fetchImage(t *testing.T, url string) (map[string]any, []byte) {
	t.Helper()
	result, err := NewHTTPExecutor().Execute(context.Background(), map[string]any{"method": "GET", "url": url}, nil)
	require.NoError(t, err)
	resultMap := result.(map[string]any)
	assert.Equal(t, 200, resultMap["status"])
	require.NotEmpty(t, resultMap["body_base64"])
	decoded, err := base64.StdEncoding.DecodeString(resultMap["body_base64"].(string))
	require.NoError(t, err)
	assert.Equal(t, len(decoded), resultMap["size"])
	return resultMap, decoded
}

func TestHTTPExecutor_Image_WebP(t *testing.T) {
	resultMap, decoded := fetchImage(t, serveImage(t, "image/webp", webp1x1))
	assert.True(t, strings.HasPrefix(resultMap["content_type"].(string), "image/"))
	assert.Equal(t, webp1x1, decoded)
	assert.Equal(t, "RIFF", string(decoded[:4]))
	assert.Equal(t, "WEBP", string(decoded[8:12]))
}

func TestHTTPExecutor_Image_PNG(t *testing.T) {
	want := testImage(t, "png")
	resultMap, decoded := fetchImage(t, serveImage(t, "image/png", want))
	assert.Equal(t, "image/png", resultMap["content_type"])
	assert.Equal(t, want, decoded, "the body arrives byte for byte")
	img, err := png.Decode(bytes.NewReader(decoded))
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 16, 8), img.Bounds())
}

func TestHTTPExecutor_Image_JPEG(t *testing.T) {
	want := testImage(t, "jpeg")
	resultMap, decoded := fetchImage(t, serveImage(t, "image/jpeg", want))
	assert.Equal(t, "image/jpeg", resultMap["content_type"])
	assert.Equal(t, want, decoded, "the body arrives byte for byte")
	img, err := jpeg.Decode(bytes.NewReader(decoded))
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 16, 8), img.Bounds())
}

// ============== Simulated Pipeline Test ==============

func TestHTTPExecutor_PipelineSimulation_ImageToLLM(t *testing.T) {
	// Simulate the HTTP -> LLM pipeline
	// HTTP node fetches image, returns base64, which can be passed to LLM

	// Mock image server
	imageData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(imageData)
	}))
	defer server.Close()

	// Step 1: HTTP node fetches image
	httpExec := NewHTTPExecutor()
	httpConfig := map[string]any{
		"method": "GET",
		"url":    server.URL,
	}

	httpResult, err := httpExec.Execute(context.Background(), httpConfig, nil)
	require.NoError(t, err)

	httpOutput := httpResult.(map[string]any)

	// Verify HTTP output has required fields for LLM
	assert.NotEmpty(t, httpOutput["body_base64"], "HTTP should return base64 encoded data")
	assert.NotEmpty(t, httpOutput["content_type"], "HTTP should return content type")

	// Step 2: Prepare LLM config (simulated - would use template resolution in real workflow)
	llmConfig := map[string]any{
		"provider": "openai",
		"model":    "gpt-4o",
		"prompt":   "What's in this image?",
		"files": []map[string]any{
			{
				"data":      httpOutput["body_base64"],
				"mime_type": httpOutput["content_type"],
				"name":      "image.png",
			},
		},
	}

	// Verify the LLM config structure is correct
	files := llmConfig["files"].([]map[string]any)
	assert.Len(t, files, 1)
	assert.NotEmpty(t, files[0]["data"])
	assert.Equal(t, "image/png", files[0]["mime_type"])
}

// ============== Error Handling Tests ==============

func TestHTTPExecutor_ErrorStatus_Default(t *testing.T) {
	// By default, 4xx/5xx responses should return an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method": "GET",
		"url":    server.URL,
	}

	_, err := exec.Execute(context.Background(), config, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestHTTPExecutor_ErrorStatus_IgnoreStatusErrors(t *testing.T) {
	// With ignore_status_errors: true, 4xx/5xx should not return error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found", "code": "RESOURCE_NOT_FOUND"}`))
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method":               "GET",
		"url":                  server.URL,
		"ignore_status_errors": true,
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, 404, resultMap["status"])
	assert.Equal(t, true, resultMap["is_error"])
	assert.NotNil(t, resultMap["body"])

	body := resultMap["body"].(map[string]any)
	assert.Equal(t, "not found", body["error"])
	assert.Equal(t, "RESOURCE_NOT_FOUND", body["code"])
}

func TestHTTPExecutor_ErrorStatus_IgnoreStatusErrors_500(t *testing.T) {
	// Test with 500 Internal Server Error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "internal error"}`))
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method":               "POST",
		"url":                  server.URL,
		"ignore_status_errors": true,
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, 500, resultMap["status"])
	assert.Equal(t, true, resultMap["is_error"])
}

func TestHTTPExecutor_SuccessStatusCodes_Allowed(t *testing.T) {
	// 404 is allowed when in success_status_codes list
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"exists": false}`))
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method":               "GET",
		"url":                  server.URL,
		"success_status_codes": []any{200, 201, 404},
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, 404, resultMap["status"])
	assert.Equal(t, true, resultMap["is_error"])

	body := resultMap["body"].(map[string]any)
	assert.Equal(t, false, body["exists"])
}

func TestHTTPExecutor_SuccessStatusCodes_NotAllowed(t *testing.T) {
	// 500 is not in success_status_codes, so it should return error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server error"}`))
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method":               "GET",
		"url":                  server.URL,
		"success_status_codes": []any{200, 201, 404}, // 500 not included
	}

	_, err := exec.Execute(context.Background(), config, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestHTTPExecutor_SuccessStatusCodes_PriorityOverIgnore(t *testing.T) {
	// success_status_codes takes priority over ignore_status_errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway) // 502
		w.Write([]byte(`upstream error`))
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method":               "GET",
		"url":                  server.URL,
		"ignore_status_errors": true,                 // Would allow all errors
		"success_status_codes": []any{200, 201, 404}, // But this restricts to specific codes
	}

	// 502 is not in success_status_codes, so it should still error
	_, err := exec.Execute(context.Background(), config, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 502")
}

func TestHTTPExecutor_IsError_FalseOnSuccess(t *testing.T) {
	// is_error should be false for successful responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	exec := NewHTTPExecutor()
	config := map[string]any{
		"method": "GET",
		"url":    server.URL,
	}

	result, err := exec.Execute(context.Background(), config, nil)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, 200, resultMap["status"])
	assert.Equal(t, false, resultMap["is_error"])
}

func TestHTTPExecutor_getIntSlice(t *testing.T) {
	exec := NewHTTPExecutor()

	tests := []struct {
		name     string
		config   map[string]any
		key      string
		expected []int
	}{
		{
			name:     "empty config",
			config:   map[string]any{},
			key:      "codes",
			expected: nil,
		},
		{
			name: "[]int type",
			config: map[string]any{
				"codes": []int{200, 201, 404},
			},
			key:      "codes",
			expected: []int{200, 201, 404},
		},
		{
			name: "[]any with float64 (from JSON)",
			config: map[string]any{
				"codes": []any{float64(200), float64(201), float64(404)},
			},
			key:      "codes",
			expected: []int{200, 201, 404},
		},
		{
			name: "[]any mixed types",
			config: map[string]any{
				"codes": []any{float64(200), 201, float64(404)},
			},
			key:      "codes",
			expected: []int{200, 201, 404},
		},
		{
			name: "wrong type",
			config: map[string]any{
				"codes": "not an array",
			},
			key:      "codes",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exec.getIntSlice(tt.config, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
