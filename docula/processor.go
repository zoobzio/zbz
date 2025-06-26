package docula

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v2"
)

// MarkdownProcessor handles markdown parsing and sanitization
type MarkdownProcessor struct {
	parser    goldmark.Markdown
	sanitizer *bluemonday.Policy
}

// NewMarkdownProcessor creates a new processor with security-first defaults
func NewMarkdownProcessor() *MarkdownProcessor {
	// Configure goldmark with security-first defaults
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,           // GitHub Flavored Markdown
			extension.Table,         // Table support
			extension.Strikethrough, // Strikethrough support
			meta.Meta,               // Frontmatter support
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Auto-generate heading IDs
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // We'll sanitize with bluemonday
		),
	)

	// Configure bluemonday for HTML sanitization
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Matching(regexp.MustCompile(`^language-[\w-]+$`)).OnElements("code")
	policy.AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")

	return &MarkdownProcessor{
		parser:    md,
		sanitizer: policy,
	}
}

// DocPage represents a processed documentation page
type DocPage struct {
	// Frontmatter fields
	Title      string   `yaml:"title"`
	NavTitle   string   `yaml:"nav_title"`
	NavOrder   int      `yaml:"nav_order"`
	Category   string   `yaml:"category"`
	Tags       []string `yaml:"tags"`

	// Generated fields
	Slug    string
	Content template.HTML
	TOC     []TOCEntry
}

// TOCEntry represents a table of contents entry
type TOCEntry struct {
	ID    string
	Title string
	Level int
}

// ProcessForAPI returns sanitized markdown for OpenAPI descriptions
func (mp *MarkdownProcessor) ProcessForAPI(raw []byte) string {
	// OpenAPI supports markdown natively, just sanitize
	return mp.sanitizer.Sanitize(string(raw))
}

// ProcessForHTML converts markdown to safe HTML for templates
func (mp *MarkdownProcessor) ProcessForHTML(raw []byte) (template.HTML, error) {
	var buf bytes.Buffer
	if err := mp.parser.Convert(raw, &buf); err != nil {
		return "", err
	}

	sanitized := mp.sanitizer.Sanitize(buf.String())
	return template.HTML(sanitized), nil
}

// ParseDocPage extracts frontmatter and processes markdown content
func (mp *MarkdownProcessor) ParseDocPage(path string, raw []byte) (*DocPage, error) {
	var page DocPage

	// Parse frontmatter and content
	ctx := parser.NewContext()
	_ = mp.parser.Parser().Parse(text.NewReader(raw), parser.WithContext(ctx))

	// Extract frontmatter
	if metaData := meta.Get(ctx); metaData != nil {
		// Convert map[string]interface{} to YAML and back to struct
		yamlData, err := yaml.Marshal(metaData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
		}

		if err := yaml.Unmarshal(yamlData, &page); err != nil {
			return nil, fmt.Errorf("failed to unmarshal frontmatter: %w", err)
		}
	}

	// Process markdown content
	content, err := mp.ProcessForHTML(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to process markdown: %w", err)
	}

	page.Content = content
	page.Slug = pathToSlug(path)

	// Extract TOC from content
	page.TOC = extractTOC(string(content))

	// Set defaults
	if page.NavTitle == "" {
		page.NavTitle = page.Title
	}

	return &page, nil
}

// pathToSlug converts a file path to a URL slug
func pathToSlug(path string) string {
	// Remove extension and clean up
	slug := strings.TrimSuffix(path, ".md")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ToLower(slug)
	return slug
}

// extractTOC extracts table of contents from HTML content
func extractTOC(html string) []TOCEntry {
	// Simple regex-based extraction (could be improved)
	headingRegex := regexp.MustCompile(`<h([1-6])[^>]*id="([^"]*)"[^>]*>([^<]*)</h[1-6]>`)
	matches := headingRegex.FindAllStringSubmatch(html, -1)

	var toc []TOCEntry
	for _, match := range matches {
		if len(match) >= 4 {
			level := int(match[1][0] - '0') // Convert '1' to 1, etc.
			toc = append(toc, TOCEntry{
				ID:    match[2],
				Title: match[3],
				Level: level,
			})
		}
	}

	return toc
}