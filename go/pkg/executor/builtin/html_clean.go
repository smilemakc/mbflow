package builtin

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	readability "github.com/go-shiori/go-readability"
	"github.com/smilemakc/mbflow/go/pkg/executor"
)

// HTMLCleanExecutor extracts readable content from HTML.
// It removes scripts, styles, and boilerplate, keeping only the main content.
// If input is not HTML, it returns the input as-is in a passthrough mode.
type HTMLCleanExecutor struct {
	*executor.BaseExecutor
}

// NewHTMLCleanExecutor creates a new HTML clean executor.
func NewHTMLCleanExecutor() *HTMLCleanExecutor {
	return &HTMLCleanExecutor{
		BaseExecutor: executor.NewBaseExecutor("html_clean"),
	}
}

// buildOutput creates a map[string]any output.
// This is required because execution_manager.go only saves output if it's map[string]any.
func buildOutput(textContent, htmlContent, title, author, excerpt, siteName string, length, wordCount int, isHTML, passthrough bool) map[string]any {
	return map[string]any{
		"text_content": textContent,
		"html_content": htmlContent,
		"title":        title,
		"author":       author,
		"excerpt":      excerpt,
		"site_name":    siteName,
		"length":       length,
		"word_count":   wordCount,
		"is_html":      isHTML,
		"passthrough":  passthrough,
	}
}

// Execute extracts readable content from HTML input.
// If the input is not HTML, it returns the input as-is (passthrough mode).
func (e *HTMLCleanExecutor) Execute(_ context.Context, config map[string]any, input any) (any, error) {
	// Get config options
	inputKey := e.GetStringDefault(config, "input_key", "")
	outputFormat := e.GetStringDefault(config, "output_format", "both")
	extractMetadata := e.GetBoolDefault(config, "extract_metadata", true)
	preserveLinks := e.GetBoolDefault(config, "preserve_links", false)
	maxLength := e.GetIntDefault(config, "max_length", 0)

	// Extract content from input using input_key if specified
	content, err := e.extractContentFromInput(input, inputKey)
	if err != nil {
		return nil, err
	}

	if content == "" {
		return nil, fmt.Errorf("input content is empty")
	}

	// Check if content is HTML
	if !e.isHTML(content) {
		// Passthrough mode: return as-is
		return buildOutput(content, "", "", "", "", "", len(content), e.countWords(content), false, true), nil
	}

	// Content is HTML - process it
	// Use a dummy URL for readability (no longer depends on source_url)
	parsedURL, _ := url.Parse("http://localhost")

	// Phase 1: Pre-process with goquery to remove dangerous content
	preprocessedHTML, err := e.preprocess(content)
	if err != nil {
		return nil, fmt.Errorf("failed to preprocess HTML: %w", err)
	}

	// Phase 2: Apply readability algorithm
	article, err := readability.FromReader(strings.NewReader(preprocessedHTML), parsedURL)
	if err != nil {
		// Fallback to simple extraction if readability fails
		return e.fallbackExtraction(preprocessedHTML, outputFormat, extractMetadata, preserveLinks, maxLength)
	}

	// Phase 3: Post-process the content
	cleanedHTML := e.postprocess(article.Content, preserveLinks)

	// Set text content
	textContent := article.TextContent
	if preserveLinks {
		textContent = e.convertLinksToMarkdown(article.Content)
	}

	// Apply max length if specified
	if maxLength > 0 {
		if len(textContent) > maxLength {
			textContent = e.truncateToWordBoundary(textContent, maxLength)
		}
		if len(cleanedHTML) > maxLength {
			cleanedHTML = e.truncateToWordBoundary(cleanedHTML, maxLength)
		}
	}

	// Build output variables
	var outText, outHTML string
	var title, author, excerpt, siteName string

	// Set output based on format
	switch outputFormat {
	case "text":
		outText = textContent
	case "html":
		outHTML = cleanedHTML
	default: // "both"
		outText = textContent
		outHTML = cleanedHTML
	}

	// Set metadata if requested
	if extractMetadata {
		title = article.Title
		author = article.Byline
		excerpt = article.Excerpt
		siteName = article.SiteName
	}

	return buildOutput(outText, outHTML, title, author, excerpt, siteName, len(outText), e.countWords(outText), true, false), nil
}

// Validate validates the HTML clean executor configuration.
func (e *HTMLCleanExecutor) Validate(config map[string]any) error {
	// Validate output_format if provided
	outputFormat := e.GetStringDefault(config, "output_format", "both")
	validFormats := map[string]bool{
		"text": true,
		"html": true,
		"both": true,
	}
	if !validFormats[outputFormat] {
		return fmt.Errorf("invalid output_format: %s (valid: text, html, both)", outputFormat)
	}

	// Validate max_length if provided
	maxLength := e.GetIntDefault(config, "max_length", 0)
	if maxLength < 0 {
		return fmt.Errorf("max_length must be non-negative")
	}

	return nil
}

// extractContentFromInput extracts content string from input using the specified key.
// If inputKey is empty, it tries to extract from the input directly or common field names.
func (e *HTMLCleanExecutor) extractContentFromInput(input any, inputKey string) (string, error) {
	switch v := input.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case map[string]any:
		// If specific key is provided, use it
		if inputKey != "" {
			if val, ok := v[inputKey]; ok {
				switch contentVal := val.(type) {
				case string:
					return contentVal, nil
				case []byte:
					return string(contentVal), nil
				}
			}
			return "", fmt.Errorf("key '%s' not found in input or has unsupported type", inputKey)
		}
		// Try common field names
		for _, field := range []string{"html", "body", "content", "data", "text", "response"} {
			if val, ok := v[field]; ok {
				switch contentVal := val.(type) {
				case string:
					return contentVal, nil
				case []byte:
					return string(contentVal), nil
				}
			}
		}
		return "", fmt.Errorf("no content found in input map (tried: html, body, content, data, text, response). Specify input_key in config")
	default:
		return "", fmt.Errorf("unsupported input type: %T (expected string, []byte, or map)", input)
	}
}

// isHTML checks if the content looks like HTML.
// Returns true if it contains HTML-like patterns.
func (e *HTMLCleanExecutor) isHTML(content string) bool {
	// Trim and check if empty
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}

	// Check for common HTML indicators
	htmlPatterns := []string{
		"<!DOCTYPE",
		"<!doctype",
		"<html",
		"<HTML",
		"<head",
		"<HEAD",
		"<body",
		"<BODY",
		"<div",
		"<DIV",
		"<p>",
		"<P>",
		"<span",
		"<SPAN",
		"<table",
		"<TABLE",
		"<article",
		"<section",
		"<header",
		"<footer",
		"<nav",
		"<main",
	}

	for _, pattern := range htmlPatterns {
		if strings.Contains(trimmed, pattern) {
			return true
		}
	}

	// Check for HTML tag pattern: <tagname> or <tagname ...>
	htmlTagRegex := regexp.MustCompile(`<[a-zA-Z][a-zA-Z0-9]*(\s[^>]*)?>`)
	return htmlTagRegex.MatchString(trimmed)
}

// preprocess removes dangerous content from HTML using goquery.
func (e *HTMLCleanExecutor) preprocess(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	// Remove dangerous and useless elements
	doc.Find("script, style, noscript, iframe, frame, frameset, object, embed, applet, form").Remove()

	// Remove HTML comments
	doc.Find("*").Contents().FilterFunction(func(i int, s *goquery.Selection) bool {
		return goquery.NodeName(s) == "#comment"
	}).Remove()

	// Remove hidden elements
	doc.Find("[hidden], [style*='display:none'], [style*='display: none'], [aria-hidden='true']").Remove()

	// Remove common ad/tracking elements by class/id patterns
	adPatterns := []string{
		"[class*='ad-']", "[class*='ads-']", "[class*='advertisement']",
		"[id*='ad-']", "[id*='ads-']", "[id*='advertisement']",
		"[class*='social']", "[class*='share']", "[class*='sharing']",
		"[class*='sidebar']", "[class*='widget']",
		"[class*='cookie']", "[class*='gdpr']", "[class*='consent']",
		"[class*='popup']", "[class*='modal']", "[class*='overlay']",
		"[class*='newsletter']", "[class*='subscribe']",
		"[class*='related']", "[class*='recommendation']",
		"[class*='comment']", "[id*='comment']",
	}
	for _, pattern := range adPatterns {
		doc.Find(pattern).Remove()
	}

	// Remove event handler attributes
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		for _, attr := range []string{
			"onclick", "onload", "onerror", "onmouseover", "onmouseout",
			"onfocus", "onblur", "onchange", "onsubmit", "onreset",
			"onkeydown", "onkeypress", "onkeyup",
		} {
			s.RemoveAttr(attr)
		}
		// Remove inline styles
		s.RemoveAttr("style")
	})

	result, err := doc.Html()
	if err != nil {
		return "", err
	}

	return result, nil
}

// postprocess cleans up the readability output.
func (e *HTMLCleanExecutor) postprocess(html string, _ bool) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	// Remove unwanted attributes but keep essential ones
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		tagName := goquery.NodeName(s)

		// Keep href for links, src/alt for images
		for _, node := range s.Nodes {
			var attrsToRemove []string
			for _, attr := range node.Attr {
				keep := false
				switch tagName {
				case "a":
					keep = attr.Key == "href"
				case "img":
					keep = attr.Key == "src" || attr.Key == "alt"
				}
				if !keep {
					attrsToRemove = append(attrsToRemove, attr.Key)
				}
			}
			for _, attr := range attrsToRemove {
				s.RemoveAttr(attr)
			}
		}
	})

	result, err := doc.Html()
	if err != nil {
		return html
	}

	// Clean up excessive whitespace
	result = e.cleanWhitespace(result)

	return result
}

// fallbackExtraction provides simple extraction when readability fails.
func (e *HTMLCleanExecutor) fallbackExtraction(html string, outputFormat string, extractMetadata bool, _ bool, maxLength int) (map[string]any, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	// Try to find main content
	mainContent := doc.Find("main, article, .main-content, #content, .content, .post, .entry").First()
	if mainContent.Length() == 0 {
		mainContent = doc.Find("body")
	}

	// Extract text
	text := mainContent.Text()
	text = e.cleanWhitespace(text)

	// Extract HTML
	htmlContent, _ := mainContent.Html()
	htmlContent = e.cleanWhitespace(htmlContent)

	// Apply max length
	if maxLength > 0 {
		if len(text) > maxLength {
			text = e.truncateToWordBoundary(text, maxLength)
		}
		if len(htmlContent) > maxLength {
			htmlContent = e.truncateToWordBoundary(htmlContent, maxLength)
		}
	}

	var outText, outHTML string
	var title, author, excerpt string

	switch outputFormat {
	case "text":
		outText = text
	case "html":
		outHTML = htmlContent
	default:
		outText = text
		outHTML = htmlContent
	}

	if extractMetadata {
		title = doc.Find("title").First().Text()
		author = doc.Find("meta[name='author']").AttrOr("content", "")
		excerpt = doc.Find("meta[name='description']").AttrOr("content", "")
	}

	return buildOutput(outText, outHTML, title, author, excerpt, "", len(outText), e.countWords(outText), true, false), nil
}

// convertLinksToMarkdown converts HTML links to markdown format [text](url).
func (e *HTMLCleanExecutor) convertLinksToMarkdown(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && href != "" {
			text := s.Text()
			if text == "" {
				text = href
			}
			s.ReplaceWithHtml(fmt.Sprintf("[%s](%s)", text, href))
		}
	})

	return doc.Text()
}

// cleanWhitespace normalizes whitespace in text.
func (e *HTMLCleanExecutor) cleanWhitespace(text string) string {
	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`[ \t]+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	// Replace multiple newlines with double newline
	newlineRegex := regexp.MustCompile(`\n\s*\n+`)
	text = newlineRegex.ReplaceAllString(text, "\n\n")

	// Trim leading/trailing whitespace from each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	// Trim overall
	text = strings.TrimSpace(text)

	return text
}

// truncateToWordBoundary truncates text at word boundary.
func (e *HTMLCleanExecutor) truncateToWordBoundary(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Find last space before maxLen
	truncated := text[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}

	return strings.TrimSpace(truncated) + "..."
}

// countWords counts words in text.
func (e *HTMLCleanExecutor) countWords(text string) int {
	if text == "" {
		return 0
	}
	// Split by whitespace and count non-empty strings
	words := strings.Fields(text)
	count := 0
	for _, word := range words {
		if utf8.RuneCountInString(word) > 0 {
			count++
		}
	}
	return count
}
