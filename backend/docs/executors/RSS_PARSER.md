# RSS Parser Executor

## Overview

The RSS Parser executor fetches and parses RSS 2.0 and Atom 1.0 feeds, returning structured JSON data with feed metadata and items.

**Type:** `rss_parser`
**Category:** Actions / Content Processing

## Features

- **Multi-format Support**: Automatically detects and parses both RSS 2.0 and Atom 1.0 feeds
- **Item Limiting**: Optionally limit the number of items returned
- **Content Extraction**: Optional full article content extraction
- **Metadata Extraction**: Feed title, description, link, publication dates, authors, categories
- **Error Handling**: Validates feed format and HTTP responses

## Configuration

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `url` | string | URL of the RSS or Atom feed to parse |

### Optional Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `maxItems` | int | 0 | Maximum number of items to return (0 = unlimited) |
| `includeContent` | bool | false | Include full article content in addition to summary |

## Input

This executor does not use input data - it fetches the feed directly from the configured URL.

## Output

### Feed-level Fields

```json
{
  "title": "Feed Title",
  "description": "Feed Description",
  "link": "https://example.com",
  "items": [...],
  "item_count": 10,
  "feed_type": "rss"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Feed title |
| `description` | string | Feed description |
| `link` | string | Feed website URL |
| `items` | array | Array of feed items (see below) |
| `item_count` | number | Number of items returned |
| `feed_type` | string | Feed format: "rss" or "atom" |

### Item Fields

Each item in the `items` array contains:

```json
{
  "title": "Article Title",
  "link": "https://example.com/article",
  "description": "Article summary",
  "content": "Full article content",
  "pubDate": "Mon, 15 Dec 2025 10:00:00 GMT",
  "author": "John Doe",
  "categories": ["Technology", "News"],
  "guid": "article-unique-id"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Item title |
| `link` | string | Item URL |
| `description` | string | Short summary |
| `content` | string | Full content (only if `includeContent=true`) |
| `pubDate` | string | Publication date |
| `author` | string | Item author |
| `categories` | array | Item categories/tags |
| `guid` | string | Unique identifier |

## Examples

### Basic RSS Parsing

```json
{
  "url": "https://blog.example.com/feed.xml",
  "maxItems": 10,
  "includeContent": false
}
```

**Output:**
```json
{
  "title": "Example Blog",
  "description": "Latest posts from our blog",
  "link": "https://blog.example.com",
  "items": [
    {
      "title": "Getting Started with MBFlow",
      "link": "https://blog.example.com/getting-started",
      "description": "Learn how to build workflows...",
      "pubDate": "Mon, 15 Dec 2025 10:00:00 GMT",
      "author": "Jane Smith",
      "categories": ["Tutorial", "Automation"],
      "guid": "post-123"
    }
  ],
  "item_count": 1,
  "feed_type": "rss"
}
```

### With Full Content

```json
{
  "url": "https://news.example.com/atom.xml",
  "maxItems": 5,
  "includeContent": true
}
```

This will include the `content` field in each item with the full article text.

## Use Cases

### 1. News Aggregation

Collect news from multiple RSS feeds and consolidate into a single dashboard:

```
RSS Parser → Transform (filter by keywords) → Database Storage
```

### 2. Blog Monitoring

Monitor company blog for new posts and send notifications:

```
RSS Parser → Conditional (check for new items) → Telegram Bot
```

### 3. Content Curation

Aggregate content from various sources and generate summaries:

```
RSS Parser → HTML Clean → LLM (summarize) → Email
```

### 4. Research Automation

Track academic journals or research feeds:

```
RSS Parser → Transform (extract specific fields) → File Storage
```

## Error Handling

### HTTP Errors

- Returns error if feed URL is unreachable
- Validates HTTP status code (must be 200 OK)
- Includes status code and message in error

### Parsing Errors

- Returns error if feed is not valid RSS or Atom
- Checks for required feed structure
- Provides descriptive error messages

## Validation

The executor validates:

- URL is not empty
- `maxItems` is non-negative (0 = unlimited)
- Feed format is either RSS 2.0 or Atom 1.0

## Implementation Details

### Supported Feed Formats

**RSS 2.0:**
- Standard RSS elements: `title`, `link`, `description`, `pubDate`, `author`, `category`
- `content:encoded` for full content (if `includeContent=true`)
- GUID for unique identification

**Atom 1.0:**
- Standard Atom elements: `title`, `link`, `summary`, `updated`, `author`, `category`
- `content` element for full content
- `id` for unique identification

### HTTP Client Configuration

- 30-second timeout
- User-Agent: `MBFlow-RSS-Parser/1.0`
- Accept headers for RSS/Atom content types

### Auto-Detection

The parser automatically detects feed format by attempting to parse as RSS first, then Atom. No manual format specification needed.

## Testing

The executor includes comprehensive unit tests:

- RSS 2.0 parsing with multiple items
- Atom 1.0 parsing
- `maxItems` limiting
- `includeContent` flag
- HTTP error handling
- Invalid feed format handling
- Configuration validation

Run tests:
```bash
go test ./pkg/executor/builtin/... -run TestRSSParser -v
```

## Limitations

- Maximum 30-second timeout for HTTP requests
- Does not handle authentication (basic auth, API keys)
- Does not follow pagination for large feeds
- Content is returned as-is (HTML/text mix) - use HTML Clean executor for processing

## Related Executors

- **HTTP**: For feeds requiring authentication or custom headers
- **HTML Clean**: Process feed content to extract readable text
- **Transform**: Filter and reshape feed data
- **Conditional**: Filter items based on criteria

## Dependencies

- Go standard library: `encoding/xml`, `net/http`, `io`
- No external dependencies

## Version

- Added in: MBFlow v1.0
- Type identifier: `rss_parser`
