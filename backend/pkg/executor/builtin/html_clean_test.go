package builtin

import (
	"context"
	"testing"
)

func TestHTMLCleanExecutor_Execute(t *testing.T) {
	executor := NewHTMLCleanExecutor()
	ctx := context.Background()

	tests := []struct {
		name        string
		config      map[string]interface{}
		input       interface{}
		wantTextLen int  // minimum expected text length
		wantHTMLLen int  // minimum expected HTML length
		wantTitle   bool // expect title to be extracted
		wantErr     bool
		errContains string
	}{
		{
			name:   "basic HTML cleaning",
			config: map[string]interface{}{},
			input: `<!DOCTYPE html>
<html>
<head>
	<title>Test Article</title>
	<script>alert('evil');</script>
	<style>.hidden { display: none; }</style>
</head>
<body>
	<nav>Navigation menu</nav>
	<main>
		<article>
			<h1>Main Article Title</h1>
			<p>This is the main content of the article. It has enough text to be recognized as the primary content by the readability algorithm. The article discusses important topics that are relevant to the reader.</p>
			<p>Additional paragraph with more content to ensure the readability algorithm has enough material to work with.</p>
		</article>
	</main>
	<footer>Footer content</footer>
</body>
</html>`,
			wantTextLen: 50,
			wantHTMLLen: 50,
			wantTitle:   true,
			wantErr:     false,
		},
		{
			name: "text only output",
			config: map[string]interface{}{
				"output_format": "text",
			},
			input:       `<html><body><p>Simple text content for testing.</p></body></html>`,
			wantTextLen: 10,
			wantHTMLLen: 0,
			wantErr:     false,
		},
		{
			name: "html only output",
			config: map[string]interface{}{
				"output_format": "html",
			},
			input:       `<html><body><p>Simple HTML content for testing.</p></body></html>`,
			wantTextLen: 0,
			wantHTMLLen: 10,
			wantErr:     false,
		},
		{
			name: "with max length",
			config: map[string]interface{}{
				"max_length": 50,
			},
			input:       `<html><body><p>This is a very long paragraph that should be truncated because we set a max length limit on the output.</p></body></html>`,
			wantTextLen: 1,
			wantHTMLLen: 1,
			wantErr:     false,
		},
		{
			name: "no metadata extraction",
			config: map[string]interface{}{
				"extract_metadata": false,
			},
			input:       `<html><head><title>Should Not Be Extracted</title></head><body><p>Content here.</p></body></html>`,
			wantTextLen: 1,
			wantTitle:   false,
			wantErr:     false,
		},
		{
			name:   "input as map with html field",
			config: map[string]interface{}{},
			input: map[string]interface{}{
				"html": `<html><body><p>Content from map input.</p></body></html>`,
			},
			wantTextLen: 1,
			wantErr:     false,
		},
		{
			name:   "input as map with body field",
			config: map[string]interface{}{},
			input: map[string]interface{}{
				"body": `<html><body><p>Content from body field.</p></body></html>`,
			},
			wantTextLen: 1,
			wantErr:     false,
		},
		{
			name:        "input as bytes",
			config:      map[string]interface{}{},
			input:       []byte(`<html><body><p>Content as bytes.</p></body></html>`),
			wantTextLen: 1,
			wantErr:     false,
		},
		{
			name:        "empty input",
			config:      map[string]interface{}{},
			input:       "",
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "unsupported input type",
			config:      map[string]interface{}{},
			input:       12345,
			wantErr:     true,
			errContains: "unsupported input type",
		},
		{
			name:   "map without html fields",
			config: map[string]interface{}{},
			input: map[string]interface{}{
				"unknown": "value",
			},
			wantErr:     true,
			errContains: "no content found",
		},
		{
			name:   "removes script tags",
			config: map[string]interface{}{},
			input: `<html><body>
				<script>alert('xss');</script>
				<p>Safe content here.</p>
				<script src="evil.js"></script>
			</body></html>`,
			wantTextLen: 1,
			wantErr:     false,
		},
		{
			name:   "removes style tags",
			config: map[string]interface{}{},
			input: `<html><head><style>body { color: red; }</style></head><body>
				<p>Content without styles.</p>
				<style>.hidden { display: none; }</style>
			</body></html>`,
			wantTextLen: 1,
			wantErr:     false,
		},
		{
			name:   "removes iframes",
			config: map[string]interface{}{},
			input: `<html><body>
				<iframe src="https://evil.com"></iframe>
				<p>Safe content.</p>
			</body></html>`,
			wantTextLen: 1,
			wantErr:     false,
		},
		{
			name: "input_key extracts from specific field",
			config: map[string]interface{}{
				"input_key": "custom_field",
			},
			input: map[string]interface{}{
				"custom_field": `<html><body><p>Content from custom field.</p></body></html>`,
				"html":         `<html><body><p>Wrong content.</p></body></html>`,
			},
			wantTextLen: 1,
			wantErr:     false,
		},
		{
			name: "input_key not found returns error",
			config: map[string]interface{}{
				"input_key": "missing_key",
			},
			input: map[string]interface{}{
				"html": `<html><body><p>Content here.</p></body></html>`,
			},
			wantErr:     true,
			errContains: "key 'missing_key' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, tt.config, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Execute() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Execute() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Execute() unexpected error: %v", err)
				return
			}

			output, ok := result.(map[string]interface{})
			if !ok {
				t.Errorf("Execute() result type = %T, want map[string]interface{}", result)
				return
			}

			textContent, _ := output["text_content"].(string)
			htmlContent, _ := output["html_content"].(string)
			title, _ := output["title"].(string)

			if tt.wantTextLen > 0 && len(textContent) < tt.wantTextLen {
				t.Errorf("Execute() text_content length = %d, want >= %d", len(textContent), tt.wantTextLen)
			}

			if tt.wantHTMLLen > 0 && len(htmlContent) < tt.wantHTMLLen {
				t.Errorf("Execute() html_content length = %d, want >= %d", len(htmlContent), tt.wantHTMLLen)
			}

			if tt.wantTitle && title == "" {
				t.Errorf("Execute() title is empty, expected non-empty")
			}

			if !tt.wantTitle && title != "" {
				// Check if metadata extraction was disabled
				if extractMeta, ok := tt.config["extract_metadata"].(bool); ok && !extractMeta {
					if title != "" {
						t.Errorf("Execute() title = %q, expected empty (metadata extraction disabled)", title)
					}
				}
			}

			// Verify no script content in output
			if contains(textContent, "alert") || contains(textContent, "<script") {
				t.Errorf("Execute() text_content contains script content, should be removed")
			}

			if contains(htmlContent, "<script") {
				t.Errorf("Execute() html_content contains script tags, should be removed")
			}
		})
	}
}

func TestHTMLCleanExecutor_Validate(t *testing.T) {
	executor := NewHTMLCleanExecutor()

	tests := []struct {
		name        string
		config      map[string]interface{}
		wantErr     bool
		errContains string
	}{
		{
			name:    "empty config is valid",
			config:  map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "valid text output format",
			config: map[string]interface{}{
				"output_format": "text",
			},
			wantErr: false,
		},
		{
			name: "valid html output format",
			config: map[string]interface{}{
				"output_format": "html",
			},
			wantErr: false,
		},
		{
			name: "valid both output format",
			config: map[string]interface{}{
				"output_format": "both",
			},
			wantErr: false,
		},
		{
			name: "invalid output format",
			config: map[string]interface{}{
				"output_format": "invalid",
			},
			wantErr:     true,
			errContains: "invalid output_format",
		},
		{
			name: "negative max length",
			config: map[string]interface{}{
				"max_length": -1,
			},
			wantErr:     true,
			errContains: "max_length must be non-negative",
		},
		{
			name: "zero max length is valid",
			config: map[string]interface{}{
				"max_length": 0,
			},
			wantErr: false,
		},
		{
			name: "positive max length is valid",
			config: map[string]interface{}{
				"max_length": 1000,
			},
			wantErr: false,
		},
		{
			name: "all valid options",
			config: map[string]interface{}{
				"output_format":    "both",
				"extract_metadata": true,
				"preserve_links":   false,
				"max_length":       5000,
				"source_url":       "https://example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestHTMLCleanExecutor_WordCount(t *testing.T) {
	executor := NewHTMLCleanExecutor()
	ctx := context.Background()

	input := `<html><body><p>This is a sentence with seven words.</p></body></html>`
	result, err := executor.Execute(ctx, map[string]interface{}{}, input)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	output := result.(map[string]interface{})
	wordCount, _ := output["word_count"].(int)
	if wordCount < 5 {
		t.Errorf("word_count = %d, expected at least 5", wordCount)
	}
}

func TestHTMLCleanExecutor_PreserveLinks(t *testing.T) {
	executor := NewHTMLCleanExecutor()
	ctx := context.Background()

	input := `<html><body><p>Visit <a href="https://example.com">our website</a> for more info.</p></body></html>`

	// Test with preserve_links = true
	result, err := executor.Execute(ctx, map[string]interface{}{
		"preserve_links": true,
		"output_format":  "text",
	}, input)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	output := result.(map[string]interface{})
	textContent, _ := output["text_content"].(string)
	if !contains(textContent, "example.com") {
		t.Errorf("text_content with preserve_links=true should contain URL, got: %s", textContent)
	}
}

func TestHTMLCleanExecutor_Passthrough(t *testing.T) {
	executor := NewHTMLCleanExecutor()
	ctx := context.Background()

	tests := []struct {
		name            string
		input           string
		wantPassthrough bool
		wantIsHTML      bool
	}{
		{
			name:            "plain text passes through",
			input:           "This is just plain text without any HTML tags.",
			wantPassthrough: true,
			wantIsHTML:      false,
		},
		{
			name:            "JSON passes through",
			input:           `{"key": "value", "nested": {"foo": "bar"}}`,
			wantPassthrough: true,
			wantIsHTML:      false,
		},
		{
			name:            "markdown passes through",
			input:           "# Heading\n\nThis is **bold** and *italic* text.\n\n- Item 1\n- Item 2",
			wantPassthrough: true,
			wantIsHTML:      false,
		},
		{
			name:            "HTML is processed",
			input:           `<html><body><p>This is HTML content.</p></body></html>`,
			wantPassthrough: false,
			wantIsHTML:      true,
		},
		{
			name:            "partial HTML is processed",
			input:           `<div><p>Just a div with paragraph.</p></div>`,
			wantPassthrough: false,
			wantIsHTML:      true,
		},
		{
			name:            "DOCTYPE triggers HTML processing",
			input:           `<!DOCTYPE html><html><body>Content</body></html>`,
			wantPassthrough: false,
			wantIsHTML:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(ctx, map[string]interface{}{}, tt.input)
			if err != nil {
				t.Fatalf("Execute() error: %v", err)
			}

			output := result.(map[string]interface{})
			passthrough, _ := output["passthrough"].(bool)
			isHTML, _ := output["is_html"].(bool)
			textContent, _ := output["text_content"].(string)

			if passthrough != tt.wantPassthrough {
				t.Errorf("passthrough = %v, want %v", passthrough, tt.wantPassthrough)
			}

			if isHTML != tt.wantIsHTML {
				t.Errorf("is_html = %v, want %v", isHTML, tt.wantIsHTML)
			}

			// For passthrough, text content should be the original input
			if tt.wantPassthrough && textContent != tt.input {
				t.Errorf("text_content = %q, want original input %q", textContent, tt.input)
			}
		})
	}
}

func TestHTMLCleanExecutor_MaxLength(t *testing.T) {
	executor := NewHTMLCleanExecutor()
	ctx := context.Background()

	longContent := `<html><body><p>` + string(make([]byte, 1000)) + `</p></body></html>`

	result, err := executor.Execute(ctx, map[string]interface{}{
		"max_length":    100,
		"output_format": "text",
	}, longContent)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	output := result.(map[string]interface{})
	textContent, _ := output["text_content"].(string)
	if len(textContent) > 110 { // Allow some margin for "..."
		t.Errorf("text_content length = %d, expected <= 110", len(textContent))
	}
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsCI(s, substr)))
}

func containsCI(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
