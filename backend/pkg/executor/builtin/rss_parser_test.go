package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRSSParserExecutor_Execute_RSS(t *testing.T) {
	// Create test RSS feed
	rssXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Test RSS Feed</title>
    <link>https://example.com</link>
    <description>A test RSS feed</description>
    <item>
      <title>First Article</title>
      <link>https://example.com/article1</link>
      <description>Summary of first article</description>
      <content:encoded>Full content of first article</content:encoded>
      <pubDate>Mon, 15 Dec 2025 10:00:00 GMT</pubDate>
      <author>John Doe</author>
      <category>Technology</category>
      <category>News</category>
      <guid>article-1</guid>
    </item>
    <item>
      <title>Second Article</title>
      <link>https://example.com/article2</link>
      <description>Summary of second article</description>
      <pubDate>Mon, 15 Dec 2025 11:00:00 GMT</pubDate>
      <author>Jane Smith</author>
      <category>Business</category>
      <guid>article-2</guid>
    </item>
    <item>
      <title>Third Article</title>
      <link>https://example.com/article3</link>
      <description>Summary of third article</description>
      <pubDate>Mon, 15 Dec 2025 12:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssXML))
	}))
	defer server.Close()

	executor := NewRSSParserExecutor()
	ctx := context.Background()

	tests := []struct {
		name        string
		config      map[string]interface{}
		wantErr     bool
		checkOutput func(t *testing.T, output interface{})
	}{
		{
			name: "Parse all items",
			config: map[string]interface{}{
				"url":            server.URL,
				"maxItems":       0,
				"includeContent": false,
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output interface{}) {
				result, ok := output.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map output, got %T", output)
				}

				if result["title"] != "Test RSS Feed" {
					t.Errorf("expected title 'Test RSS Feed', got '%v'", result["title"])
				}

				if result["link"] != "https://example.com" {
					t.Errorf("expected link 'https://example.com', got '%v'", result["link"])
				}

				items, ok := result["items"].([]map[string]interface{})
				if !ok {
					t.Fatalf("expected items to be []map[string]interface{}, got %T", result["items"])
				}

				if len(items) != 3 {
					t.Errorf("expected 3 items, got %d", len(items))
				}

				// Check first item
				if items[0]["title"] != "First Article" {
					t.Errorf("expected first item title 'First Article', got '%v'", items[0]["title"])
				}

				// Check categories
				categories, ok := items[0]["categories"].([]string)
				if !ok {
					t.Fatalf("expected categories to be []string, got %T", items[0]["categories"])
				}
				if len(categories) != 2 {
					t.Errorf("expected 2 categories, got %d", len(categories))
				}

				// Content should not be included
				if _, exists := items[0]["content"]; exists {
					t.Error("content should not be included when includeContent=false")
				}
			},
		},
		{
			name: "Parse with maxItems limit",
			config: map[string]interface{}{
				"url":            server.URL,
				"maxItems":       2,
				"includeContent": false,
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output interface{}) {
				result, ok := output.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map output, got %T", output)
				}

				items, ok := result["items"].([]map[string]interface{})
				if !ok {
					t.Fatalf("expected items to be []map[string]interface{}, got %T", result["items"])
				}

				if len(items) != 2 {
					t.Errorf("expected 2 items (limited by maxItems), got %d", len(items))
				}

				if result["item_count"] != 2 {
					t.Errorf("expected item_count to be 2, got %v", result["item_count"])
				}
			},
		},
		{
			name: "Parse with includeContent",
			config: map[string]interface{}{
				"url":            server.URL,
				"maxItems":       1,
				"includeContent": true,
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output interface{}) {
				result, ok := output.(map[string]interface{})
				if !ok {
					t.Fatalf("expected map output, got %T", output)
				}

				items, ok := result["items"].([]map[string]interface{})
				if !ok {
					t.Fatalf("expected items to be []map[string]interface{}, got %T", result["items"])
				}

				if len(items) != 1 {
					t.Fatalf("expected 1 item, got %d", len(items))
				}

				// Content should be included
				content, exists := items[0]["content"]
				if !exists {
					t.Error("content should be included when includeContent=true")
				}

				if content != "Full content of first article" {
					t.Errorf("expected content 'Full content of first article', got '%v'", content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executor.Execute(ctx, tt.config, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestRSSParserExecutor_Execute_Atom(t *testing.T) {
	// Create test Atom feed
	atomXML := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <link href="https://example.com" rel="alternate"/>
  <entry>
    <title>Atom Entry 1</title>
    <link href="https://example.com/entry1" rel="alternate"/>
    <id>entry-1</id>
    <updated>2025-12-15T10:00:00Z</updated>
    <summary>Summary of entry 1</summary>
    <content type="html">Full content of entry 1</content>
    <author>
      <name>Alice Johnson</name>
    </author>
    <category term="Science"/>
  </entry>
  <entry>
    <title>Atom Entry 2</title>
    <link href="https://example.com/entry2" rel="alternate"/>
    <id>entry-2</id>
    <updated>2025-12-15T11:00:00Z</updated>
    <summary>Summary of entry 2</summary>
    <author>
      <name>Bob Williams</name>
    </author>
    <category term="Technology"/>
    <category term="AI"/>
  </entry>
</feed>`

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(atomXML))
	}))
	defer server.Close()

	executor := NewRSSParserExecutor()
	ctx := context.Background()

	config := map[string]interface{}{
		"url":            server.URL,
		"maxItems":       0,
		"includeContent": true,
	}

	output, err := executor.Execute(ctx, config, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	result, ok := output.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map output, got %T", output)
	}

	if result["title"] != "Test Atom Feed" {
		t.Errorf("expected title 'Test Atom Feed', got '%v'", result["title"])
	}

	if result["feed_type"] != "atom" {
		t.Errorf("expected feed_type 'atom', got '%v'", result["feed_type"])
	}

	items, ok := result["items"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected items to be []map[string]interface{}, got %T", result["items"])
	}

	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	// Check first entry
	if items[0]["title"] != "Atom Entry 1" {
		t.Errorf("expected first item title 'Atom Entry 1', got '%v'", items[0]["title"])
	}

	if items[0]["author"] != "Alice Johnson" {
		t.Errorf("expected author 'Alice Johnson', got '%v'", items[0]["author"])
	}

	// Check content inclusion
	if items[0]["content"] != "Full content of entry 1" {
		t.Errorf("expected content 'Full content of entry 1', got '%v'", items[0]["content"])
	}

	// Check categories
	categories, ok := items[1]["categories"].([]string)
	if !ok {
		t.Fatalf("expected categories to be []string, got %T", items[1]["categories"])
	}
	if len(categories) != 2 {
		t.Errorf("expected 2 categories for second item, got %d", len(categories))
	}
}

func TestRSSParserExecutor_Execute_HTTPErrors(t *testing.T) {
	executor := NewRSSParserExecutor()
	ctx := context.Background()

	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		config      map[string]interface{}
		wantErr     bool
	}{
		{
			name: "404 Not Found",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			config:  map[string]interface{}{"url": ""},
			wantErr: true,
		},
		{
			name: "Invalid XML",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("Not XML content"))
				}))
			},
			config:  map[string]interface{}{"url": ""},
			wantErr: true,
		},
		{
			name: "Empty feed",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`<?xml version="1.0"?><unknown></unknown>`))
				}))
			},
			config:  map[string]interface{}{"url": ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			tt.config["url"] = server.URL

			_, err := executor.Execute(ctx, tt.config, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRSSParserExecutor_Validate(t *testing.T) {
	executor := NewRSSParserExecutor()

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid config",
			config: map[string]interface{}{
				"url":            "https://example.com/feed.xml",
				"maxItems":       10,
				"includeContent": true,
			},
			wantErr: false,
		},
		{
			name: "Valid config with defaults",
			config: map[string]interface{}{
				"url": "https://example.com/feed.xml",
			},
			wantErr: false,
		},
		{
			name: "Missing URL",
			config: map[string]interface{}{
				"maxItems": 10,
			},
			wantErr: true,
		},
		{
			name: "Empty URL",
			config: map[string]interface{}{
				"url": "",
			},
			wantErr: true,
		},
		{
			name: "Negative maxItems",
			config: map[string]interface{}{
				"url":      "https://example.com/feed.xml",
				"maxItems": -5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
