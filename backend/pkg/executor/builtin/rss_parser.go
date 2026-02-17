package builtin

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/smilemakc/mbflow/pkg/executor"
)

// RSSParserExecutor parses RSS/Atom feeds and returns structured data.
// It fetches RSS feed from URL, parses the XML, and returns feed metadata and items.
type RSSParserExecutor struct {
	*executor.BaseExecutor
	client *http.Client
}

// NewRSSParserExecutor creates a new RSS parser executor.
func NewRSSParserExecutor() *RSSParserExecutor {
	return &RSSParserExecutor{
		BaseExecutor: executor.NewBaseExecutor("rss_parser"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RSS 2.0 structures
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	Content     string     `xml:"encoded"` // content:encoded
	PubDate     string     `xml:"pubDate"`
	Author      string     `xml:"author"`
	Categories  []Category `xml:"category"`
	GUID        string     `xml:"guid"`
}

type Category struct {
	Value string `xml:",chardata"`
}

// Atom 1.0 structures
type Atom struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Link    []AtomLink  `xml:"link"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type AtomEntry struct {
	Title      string         `xml:"title"`
	Link       []AtomLink     `xml:"link"`
	Summary    string         `xml:"summary"`
	Content    AtomContent    `xml:"content"`
	Updated    string         `xml:"updated"`
	Author     AtomAuthor     `xml:"author"`
	Categories []AtomCategory `xml:"category"`
	ID         string         `xml:"id"`
}

type AtomContent struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type AtomAuthor struct {
	Name string `xml:"name"`
}

type AtomCategory struct {
	Term string `xml:"term,attr"`
}

// Execute fetches and parses RSS/Atom feed.
func (e *RSSParserExecutor) Execute(ctx context.Context, config map[string]any, input any) (any, error) {
	// Get required URL
	url, err := e.GetString(config, "url")
	if err != nil {
		return nil, err
	}

	// Get optional config
	maxItems := e.GetIntDefault(config, "maxItems", 0)
	includeContent := e.GetBoolDefault(config, "includeContent", false)

	// Fetch RSS feed
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to avoid being blocked
	req.Header.Set("User-Agent", "MBFlow-RSS-Parser/1.0")
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, application/atom+xml")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to parse as RSS 2.0 first
	var rss RSS
	if err := xml.Unmarshal(body, &rss); err == nil && rss.Channel.Title != "" {
		return e.buildRSSOutput(rss, maxItems, includeContent), nil
	}

	// Try to parse as Atom 1.0
	var atom Atom
	if err := xml.Unmarshal(body, &atom); err == nil && atom.Title != "" {
		return e.buildAtomOutput(atom, maxItems, includeContent), nil
	}

	return nil, fmt.Errorf("failed to parse feed: not a valid RSS or Atom feed")
}

// buildRSSOutput converts RSS structure to output format
func (e *RSSParserExecutor) buildRSSOutput(rss RSS, maxItems int, includeContent bool) map[string]any {
	items := make([]map[string]any, 0, len(rss.Channel.Items))

	limit := len(rss.Channel.Items)
	if maxItems > 0 && maxItems < limit {
		limit = maxItems
	}

	for i := 0; i < limit; i++ {
		item := rss.Channel.Items[i]

		// Extract categories
		categories := make([]string, len(item.Categories))
		for j, cat := range item.Categories {
			categories[j] = cat.Value
		}

		itemData := map[string]any{
			"title":       item.Title,
			"link":        item.Link,
			"description": item.Description,
			"pubDate":     item.PubDate,
			"author":      item.Author,
			"categories":  categories,
			"guid":        item.GUID,
		}

		// Add content if requested
		if includeContent {
			content := item.Content
			if content == "" {
				content = item.Description
			}
			itemData["content"] = content
		}

		items = append(items, itemData)
	}

	return map[string]any{
		"title":       rss.Channel.Title,
		"description": rss.Channel.Description,
		"link":        rss.Channel.Link,
		"items":       items,
		"item_count":  len(items),
		"feed_type":   "rss",
	}
}

// buildAtomOutput converts Atom structure to output format
func (e *RSSParserExecutor) buildAtomOutput(atom Atom, maxItems int, includeContent bool) map[string]any {
	items := make([]map[string]any, 0, len(atom.Entries))

	limit := len(atom.Entries)
	if maxItems > 0 && maxItems < limit {
		limit = maxItems
	}

	// Get feed link
	feedLink := ""
	for _, link := range atom.Link {
		if link.Rel == "" || link.Rel == "alternate" {
			feedLink = link.Href
			break
		}
	}

	for i := 0; i < limit; i++ {
		entry := atom.Entries[i]

		// Get entry link
		entryLink := ""
		for _, link := range entry.Link {
			if link.Rel == "" || link.Rel == "alternate" {
				entryLink = link.Href
				break
			}
		}

		// Extract categories
		categories := make([]string, len(entry.Categories))
		for j, cat := range entry.Categories {
			categories[j] = cat.Term
		}

		itemData := map[string]any{
			"title":       entry.Title,
			"link":        entryLink,
			"description": entry.Summary,
			"pubDate":     entry.Updated,
			"author":      entry.Author.Name,
			"categories":  categories,
			"guid":        entry.ID,
		}

		// Add content if requested
		if includeContent {
			content := entry.Content.Value
			if content == "" {
				content = entry.Summary
			}
			itemData["content"] = content
		}

		items = append(items, itemData)
	}

	return map[string]any{
		"title":       atom.Title,
		"description": "",
		"link":        feedLink,
		"items":       items,
		"item_count":  len(items),
		"feed_type":   "atom",
	}
}

// Validate validates the RSS parser executor configuration.
func (e *RSSParserExecutor) Validate(config map[string]any) error {
	// Validate required URL field
	if err := e.ValidateRequired(config, "url"); err != nil {
		return err
	}

	url, err := e.GetString(config, "url")
	if err != nil {
		return err
	}

	if url == "" {
		return fmt.Errorf("url cannot be empty")
	}

	// Validate optional maxItems
	maxItems := e.GetIntDefault(config, "maxItems", 0)
	if maxItems < 0 {
		return fmt.Errorf("maxItems must be non-negative (0 = all items)")
	}

	return nil
}
