# Docula V2 - Living Documentation System

## Vision

Transform docula from static OpenAPI generation to a **living documentation system** powered by:

- **Hodor cloud storage** for markdown content
- **Flux reactive updates** for real-time doc changes
- **Self-contained UI** for docs site hosting
- **Markdown-driven content** with convention-based naming

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Hodor Cloud   │◄──►│  Flux Watcher   │◄──►│ Docula Service  │
│   Storage       │    │  (Reactive)     │    │  (Generator)    │
│  - api.md       │    │                 │    │                 │
│  - create_user  │    │  Auto-updates   │    │  OpenAPI +      │
│  - get_users    │    │  on changes     │    │  Docs UI        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Core Components

### 1. **Hodor Integration**

- Store markdown docs in cloud bucket (S3, GCS, MinIO, etc.)
- Support any hodor provider for maximum flexibility
- Bucket structure follows naming conventions

### 2. **Flux Reactive Updates**

- Watch for changes to markdown files in bucket
- Auto-regenerate OpenAPI spec when content changes
- Hot-reload documentation without restarting service

### 3. **Self-Contained Docs UI**

- Embedded docs site (Scalar, Swagger UI, or custom)
- Serve docs at `/docs` endpoint
- No external dependencies for viewing docs

### 4. **Convention-Based Content**

- `api.md` → Sets OpenAPI description field
- `{operationId}.md` → Documents specific endpoints
- `{model}.md` → Documents data models
- `auth.md` → Documents authentication

## Implementation Plan

### Phase 1: Foundation ✅ (Current State Analysis)

**Current Implementation:**

- ✅ Basic OpenAPI spec generation from Go structs
- ✅ Automatic schema generation with validation constraints
- ✅ Endpoint registration system
- ✅ YAML/JSON export capabilities
- ✅ Contract-based configuration
- ✅ Thread-safe operations

**Strengths to Preserve:**

- Solid OpenAPI 3.1.0 foundation
- Good struct → schema generation
- Clean contract interface
- Thread-safe singleton pattern

### Phase 2: Hodor + Flux Integration 🎯 (Next Sprint)

#### 2.1 Add Dependencies

```go
// go.mod additions needed
require (
    zbz/hodor v0.0.0-00010101000000-000000000000
    zbz/flux v0.0.0-00010101000000-000000000000
)

// Add to replacements
replace zbz/hodor => ../hodor
replace zbz/flux => ../flux
```

#### 2.2 Enhanced Contract System

```go
type DoculaContract struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description,omitempty"`
    Info        *OpenAPIInfo `yaml:"info,omitempty"`

    // NEW: Hodor storage for markdown content
    Storage     *hodor.HodorContract `yaml:"storage,omitempty"`

    // NEW: Flux configuration for reactive updates
    ReactiveUpdates bool `yaml:"reactive_updates,omitempty"`

    // NEW: UI configuration
    DocsUI      *DocsUIConfig `yaml:"docs_ui,omitempty"`
}

type DocsUIConfig struct {
    Enabled  bool   `yaml:"enabled"`
    Path     string `yaml:"path"`     // Default: "/docs"
    Engine   string `yaml:"engine"`   // "scalar", "swagger", "redoc"
    Title    string `yaml:"title,omitempty"`
}
```

#### 2.3 Markdown Content Loader

```go
type MarkdownLoader struct {
    hodor     hodor.HodorContract
    processor *MarkdownProcessor
}

func (m *MarkdownLoader) LoadAPIDescription() (string, error) {
    content, err := m.hodor.Get("api.md")
    // Process markdown → HTML/text for OpenAPI description
}

func (m *MarkdownLoader) LoadOperationDoc(operationId string) (*OperationDoc, error) {
    content, err := m.hodor.Get(operationId + ".md")
    // Parse markdown for operation-specific docs
}
```

#### 2.4 Flux Reactive System

```go
func (d *doculaService) setupReactiveUpdates() error {
    if d.contract.ReactiveUpdates && d.contract.Storage != nil {
        // Watch for changes to markdown files
        watcher, err := flux.Sync[map[string][]byte](
            *d.contract.Storage,
            "*.md",  // Watch all markdown files
            d.handleMarkdownChange,
        )
        return err
    }
    return nil
}

func (d *doculaService) handleMarkdownChange(old, new map[string][]byte) {
    zlog.Info("Markdown content changed, regenerating docs")
    d.regenerateSpecFromMarkdown(new)
}
```

### Phase 3: Self-Contained UI 🚀 (Sprint 2)

#### 3.1 Embedded Docs Engines

```go
//go:embed ui/scalar/*
var scalarAssets embed.FS

//go:embed ui/swagger/*
var swaggerAssets embed.FS

func (d *doculaService) ServeDocsUI(w http.ResponseWriter, r *http.Request) {
    switch d.contract.DocsUI.Engine {
    case "scalar":
        d.serveScalarUI(w, r)
    case "swagger":
        d.serveSwaggerUI(w, r)
    default:
        d.serveScalarUI(w, r) // Default to Scalar
    }
}
```

#### 3.2 Convention-Based Content Processing

```go
type ConventionProcessor struct {
    loader *MarkdownLoader
}

// Convention: create_user.md documents POST /users with operationId: CreateUser
func (c *ConventionProcessor) EnrichOperation(operation *OpenAPIOperation) {
    if doc, err := c.loader.LoadOperationDoc(operation.OperationId); err == nil {
        operation.Description = doc.Description
        operation.Summary = doc.Summary
        // Merge any additional markdown content
    }
}

// Convention: User.md documents User model schema
func (c *ConventionProcessor) EnrichSchema(name string, schema *OpenAPISchema) {
    if doc, err := c.loader.LoadModelDoc(name); err == nil {
        schema.Description = doc.Description
        // Add examples, field descriptions, etc.
    }
}
```

### Phase 4: Advanced Features 🌟 (Future)

#### 4.1 Live Editing Interface

- Web-based markdown editor at `/docs/edit`
- Direct editing of markdown in hodor storage
- Real-time preview of changes

#### 4.2 Multi-Language Support

- Support for multiple markdown languages
- `api.en.md`, `api.es.md`, etc.
- Language-aware docs UI

#### 4.3 Advanced Content Features

- Mermaid diagram support in markdown
- Code example generation
- Interactive API explorer
- SDK generation documentation

## File Naming Conventions

### Core Documentation

- `api.md` → OpenAPI `info.description`
- `auth.md` → Authentication documentation
- `errors.md` → Error response documentation

### Operation Documentation

- `{operationId}.md` → Specific endpoint docs
- Examples:
  - `CreateUser.md` → POST /users operation
  - `GetUser.md` → GET /users/{id} operation
  - `ListUsers.md` → GET /users operation
  - `UpdateUser.md` → PUT /users/{id} operation
  - `DeleteUser.md` → DELETE /users/{id} operation

### Model Documentation

- `{ModelName}.md` → Schema documentation
- Examples:
  - `User.md` → User model schema
  - `Product.md` → Product model schema
  - `Order.md` → Order model schema

### Tag Documentation

- `tags/{tagName}.md` → Tag descriptions
- Examples:
  - `tags/users.md` → Users tag documentation
  - `tags/products.md` → Products tag documentation

## Example Usage

```yaml
# docula.yaml
name: "api-docs"
description: "Living API documentation"

info:
  title: "My API"
  version: "1.0.0"

storage:
  provider: "s3"
  config:
    bucket: "myapi-docs"
    region: "us-west-2"

reactive_updates: true

docs_ui:
  enabled: true
  path: "/docs"
  engine: "scalar"
  title: "My API Documentation"
```

```go
// Usage in application
contract := docula.DoculaContract{
    Name:            "api-docs",
    ReactiveUpdates: true,
    Storage: &hodor.HodorContract{
        Provider: "s3",
        Config: map[string]any{
            "bucket": "myapi-docs",
            "region": "us-west-2",
        },
    },
    DocsUI: &docula.DocsUIConfig{
        Enabled: true,
        Path:    "/docs",
        Engine:  "scalar",
    },
}

docs := contract.Docula()

// Existing programmatic API still works
docs.RegisterModel("User", User{})
docs.RegisterEndpoint("POST", "/users", "users", "Create User", "Create a new user", ...)

// NEW: Markdown content enhances the generated docs automatically
// Content from hodor storage bucket enriches the OpenAPI spec
```

## Benefits

### For Developers

- **Hot Updates**: Change docs without redeploying
- **Rich Markdown**: Use full markdown features for documentation
- **Version Control**: Store docs in git alongside code
- **Collaborative**: Non-technical team members can edit docs

### For Operations

- **Self-Contained**: No external doc hosting needed
- **Scalable Storage**: Use any cloud provider via hodor
- **Reactive**: Automatic updates without restarts
- **Flexible UI**: Choose your preferred docs engine

### For Users

- **Always Current**: Docs update in real-time
- **Rich Content**: Markdown enables rich documentation
- **Interactive**: Built-in API explorer
- **Accessible**: Standard web interface

## Migration Path

### Existing Docula Users

1. **No Breaking Changes**: Current programmatic API unchanged
2. **Opt-In Features**: Hodor/Flux integration is optional
3. **Enhanced Output**: Markdown content enriches existing docs
4. **Backward Compatible**: Existing contracts continue working

### Implementation Strategy

1. **Phase 2**: Core hodor/flux integration (1-2 weeks)
2. **Phase 3**: Self-contained UI (1 week)
3. **Phase 4**: Advanced features (ongoing)

## Next Steps

1. **Update go.mod** with hodor/flux dependencies
2. **Implement MarkdownLoader** for hodor integration
3. **Add Flux reactive system** for live updates
4. **Embed Scalar UI** for self-contained docs
5. **Test with example application**

## Implementation Details

### Markdown Processing Pipeline

#### Library Selection
**Recommendation: [goldmark](https://github.com/yuin/goldmark)**
- Pure Go implementation with excellent performance
- Highly extensible with plugins
- CommonMark compliant
- Built-in security features for XSS prevention
- Supports custom renderers for template injection

#### Flux Integration for Live Documentation

The real power comes from Flux's reactive file watching, not complex field processing. Here's a cleaner approach:

```go
// DoculaContent represents all markdown content for the service
type DoculaContent struct {
    OpenAPI  map[string]string      // operationId -> markdown
    Models   map[string]string      // modelName -> markdown  
    DevDocs  map[string]string      // path -> markdown
    Metadata map[string]interface{} // additional metadata
}

// Flux watches the hodor bucket and syncs content to docula
func (d *doculaService) setupFluxSync() error {
    // Simple sync - Flux handles the watching, we handle the processing
    watcher, err := flux.Sync[map[string][]byte](
        *d.contract.Storage,
        "*.md",
        d.processMarkdownChanges,
    )
    if err != nil {
        return fmt.Errorf("failed to setup flux sync: %w", err)
    }
    
    d.contentWatcher = watcher
    return nil
}

// Process changes and update our internal state
func (d *doculaService) processMarkdownChanges(old, new map[string][]byte) {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    // Clear cache for changed files
    for path := range new {
        d.cache.Invalidate(path)
    }
    
    // Process markdown files by convention
    for path, content := range new {
        switch {
        case strings.HasSuffix(path, ".md"):
            d.processDocumentationFile(path, content)
        case strings.HasPrefix(path, "operations/"):
            d.processOperationDoc(path, content)
        case strings.HasPrefix(path, "models/"):
            d.processModelDoc(path, content)
        }
    }
    
    // Regenerate OpenAPI spec with new content
    d.regenerateOpenAPISpec()
}
```

#### Why This Integration Works

1. **Flux handles the hard parts**: File watching, change detection, bucket syncing
2. **Docula focuses on its domain**: Documentation generation and serving
3. **Clean separation**: Flux provides raw content, docula processes it
4. **Convention-driven**: File paths determine documentation type
5. **Reactive by default**: Changes immediately trigger regeneration

#### Markdown Processing as a Utility

```go
// MarkdownProcessor is a simple utility, not a flux field processor
type MarkdownProcessor struct {
    parser    goldmark.Markdown
    sanitizer *bluemonday.Policy
}

// ProcessForAPI returns sanitized markdown for OpenAPI descriptions
func (mp *MarkdownProcessor) ProcessForAPI(raw []byte) string {
    // OpenAPI supports markdown natively, just sanitize
    return mp.sanitizer.SanitizeBytes(raw)
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
```

### Template-Ready Output

```go
// TemplateData represents processed markdown ready for injection
type TemplateData struct {
    Title       string
    Content     template.HTML // Safe HTML for direct template use
    TOC         []TOCEntry    // Table of contents from headers
    LastUpdated time.Time
}

// ProcessForTemplate converts markdown to template-ready format
func (mp *MarkdownProcessor) ProcessForTemplate(markdown string) (*TemplateData, error) {
    field := &MarkdownField{Raw: markdown}
    processed, err := mp.Process(field)
    if err != nil {
        return nil, err
    }
    
    mf := processed.(*MarkdownField)
    
    // Extract title from first H1
    title := extractTitle(mf.Raw)
    
    // Generate TOC from headers
    toc := generateTOC(mf.HTML)
    
    return &TemplateData{
        Title:       title,
        Content:     template.HTML(mf.Sanitized),
        TOC:         toc,
        LastUpdated: time.Now(),
    }, nil
}
```

### Simple Cache Strategy

Until we have a proper cache service with contracts, keep it dead simple:

```go
// doculaService with simple in-memory cache
type doculaService struct {
    contract    DoculaContract
    spec        *OpenAPISpec
    
    // Simple cache - just store the bytes
    cachedYAML  []byte
    cachedJSON  []byte
    
    processor   *MarkdownProcessor
    mu          sync.RWMutex
}

// Flux callback triggers regeneration and cache update
func (d *doculaService) processMarkdownChanges(old, new map[string][]byte) {
    d.mu.Lock()
    defer d.mu.Unlock()
    
    // Process markdown updates
    for path, content := range new {
        d.processDocumentationFile(path, content)
    }
    
    // Regenerate spec and update cache
    d.regenerateOpenAPISpec()
    d.updateCache()
}

// Pre-compute formats after changes
func (d *doculaService) updateCache() {
    // Compute once, serve many times
    if yamlData, err := yaml.Marshal(d.spec); err == nil {
        d.cachedYAML = yamlData
    }
    
    if jsonData, err := json.MarshalIndent(d.spec, "", "  "); err == nil {
        d.cachedJSON = jsonData
    }
}

// Handlers just return cached bytes
func (d *doculaService) HandleOpenAPISpec(format string) ([]byte, error) {
    d.mu.RLock()
    defer d.mu.RUnlock()
    
    switch format {
    case "yaml":
        return d.cachedYAML, nil
    case "json":
        return d.cachedJSON, nil
    default:
        return nil, fmt.Errorf("unsupported format: %s", format)
    }
}
```

This will integrate cleanly with the future cache contract system.

### Error Handling Strategy

Simple, graceful degradation with sensible defaults:

```go
// Process documentation with fallbacks
func (d *doculaService) processDocumentationFile(path string, content []byte) {
    operationId := strings.TrimSuffix(filepath.Base(path), ".md")
    
    if len(content) == 0 {
        // Provide placeholder for missing content
        d.setOperationDescription(operationId, "Documentation pending...")
        return
    }
    
    // Process markdown
    description := d.processor.ProcessForAPI(content)
    d.setOperationDescription(operationId, description)
}

// Handle missing required documentation
func (d *doculaService) loadAPIDescription() string {
    content, err := d.storage.Get("api.md")
    if err != nil {
        // Sensible default instead of error
        return "API documentation available at /docs"
    }
    
    return d.processor.ProcessForAPI(content)
}

// Flux error handling
func (d *doculaService) setupFluxSync() error {
    watcher, err := flux.Sync[map[string][]byte](
        *d.contract.Storage,
        "*.md",
        d.processMarkdownChanges,
    )
    if err != nil {
        // Log but don't fail - docs work without live updates
        zlog.Warn("Live documentation updates disabled", 
            zap.Error(err))
        return nil
    }
    
    d.contentWatcher = watcher
    return nil
}
```

#### Error Handling Principles

1. **Never fail the service** - Documentation errors shouldn't break the API
2. **Provide placeholders** - "Documentation pending..." is better than empty
3. **Log but continue** - Record issues for debugging without disrupting service
4. **Graceful degradation** - Work without Flux/Hodor if needed
5. **User-friendly defaults** - Help users find the docs even when content is missing

### Developer Documentation Site (Go + HTMX)

For simple, fast documentation sites, Go templates + HTMX is the perfect fit - especially for markdown-driven content. No build step, no node_modules, just Go.

#### Why Go/HTMX for Docs

1. **Perfect for SSR**: Documentation is mostly static content
2. **Minimal JavaScript**: HTMX handles interactivity without a framework
3. **Fast**: Sub-millisecond render times with cached markdown
4. **Simple**: No build process, just Go templates
5. **Consistent**: Stays within the zbz Go ecosystem

#### Markdown Frontmatter Convention

```markdown
---
title: Getting Started with ZBZ
nav_title: Getting Started
nav_order: 1
category: Guides
tags: [quickstart, installation]
---

# Getting Started with ZBZ

Your actual markdown content here...
```

#### Frontmatter Processing

```go
type DocPage struct {
    // Frontmatter fields
    Title      string   `yaml:"title"`
    NavTitle   string   `yaml:"nav_title"`
    NavOrder   int      `yaml:"nav_order"`
    Category   string   `yaml:"category"`
    Tags       []string `yaml:"tags"`
    
    // Generated fields
    Slug       string
    Content    template.HTML
    TOC        []TOCEntry
    LastUpdate time.Time
}

// Extract frontmatter using goldmark + go-yaml
func (d *doculaService) parseDocPage(path string, raw []byte) (*DocPage, error) {
    var page DocPage
    
    // goldmark has frontmatter extension
    md := goldmark.New(
        goldmark.WithExtensions(
            meta.Meta,  // Frontmatter support
        ),
    )
    
    // Parse and extract metadata
    ctx := parser.NewContext()
    doc := md.Parser().Parse(text.NewReader(raw), parser.WithContext(ctx))
    
    // Get frontmatter
    if data := meta.Get(ctx); data != nil {
        if err := mapstructure.Decode(data, &page); err != nil {
            return nil, err
        }
    }
    
    // Process remaining markdown
    page.Content = d.processor.ProcessForHTML(raw)
    page.Slug = pathToSlug(path)
    
    return &page, nil
}
```

#### Go Template Structure

```go
//go:embed templates/*.tmpl
var templates embed.FS

// Base layout template
const baseTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}} - {{.SiteName}}</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        /* Minimal CSS for proof of concept */
        body { max-width: 800px; margin: 0 auto; padding: 2rem; }
        nav { border-top: 1px solid #ccc; margin-top: 3rem; padding-top: 1rem; }
        .search-results { border: 1px solid #ddd; padding: 1rem; }
    </style>
</head>
<body>
    {{template "content" .}}
    {{template "navigation" .}}
</body>
</html>`

// Doc page template
const docTemplate = `{{define "content"}}
<article>
    <h1>{{.Page.Title}}</h1>
    
    <div hx-get="/docs/search" 
         hx-trigger="keyup changed delay:500ms" 
         hx-target="#search-results"
         hx-include="[name='q']">
        <input type="search" name="q" placeholder="Search docs...">
        <div id="search-results"></div>
    </div>
    
    {{if .Page.TOC}}
    <nav class="toc">
        <h3>On this page</h3>
        {{range .Page.TOC}}
        <a href="#{{.ID}}">{{.Title}}</a>
        {{end}}
    </nav>
    {{end}}
    
    <div class="content">
        {{.Page.Content}}
    </div>
</article>
{{end}}

{{define "navigation"}}
<nav>
    <div hx-get="/docs/nav" hx-trigger="load" hx-swap="outerHTML">
        Loading navigation...
    </div>
</nav>
{{end}}`
```

#### HTMX-Powered Features

```go
// Search endpoint for HTMX
func (d *doculaService) handleSearch(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    if query == "" {
        return
    }
    
    // Simple search through cached pages
    results := d.searchDocs(query)
    
    // Return HTML fragment for HTMX
    fmt.Fprintf(w, `<div class="search-results">`)
    for _, result := range results {
        fmt.Fprintf(w, `<a href="/docs/%s">%s</a>`, 
            result.Slug, result.Title)
    }
    fmt.Fprintf(w, `</div>`)
}

// Dynamic navigation loaded via HTMX
func (d *doculaService) handleNav(w http.ResponseWriter, r *http.Request) {
    currentPath := r.URL.Query().Get("current")
    
    // Build navigation from pages
    nav := d.buildNavigation()
    
    // Return HTML fragment
    fmt.Fprintf(w, `<nav>`)
    for _, cat := range nav.Categories {
        fmt.Fprintf(w, `<div class="category">%s</div>`, cat.Name)
        for _, page := range cat.Pages {
            active := ""
            if page.Slug == currentPath {
                active = "active"
            }
            fmt.Fprintf(w, `<a href="/docs/%s" class="%s">%s</a>`,
                page.Slug, active, page.NavTitle)
        }
    }
    fmt.Fprintf(w, `</nav>`)
}
```

#### Benefits of This Approach

1. **No Build Step**: Just run your Go binary
2. **Live Updates**: Flux updates trigger immediate changes
3. **Fast Search**: In-memory search with HTMX updates
4. **Progressive Enhancement**: Works without JavaScript
5. **Simple Deployment**: Single binary with embedded templates

#### Future Enhancements

- **Alpine.js**: For more complex interactions (dropdowns, modals)
- **Tailwind CSS**: Via CDN for better styling
- **Full-text search**: Using bleve or similar
- **Syntax highlighting**: With chroma for code blocks
- **Dark mode**: CSS variables + localStorage

This approach keeps everything in Go while providing a modern, reactive documentation experience.

### Content Site Templates

Docula provides pre-built site templates that leverage the same markdown + frontmatter + HTMX system. Each template is a starting point that can be customized.

#### 1. Documentation Site Template

**Purpose**: Technical documentation, API guides, tutorials

**Frontmatter Convention**:
```yaml
---
title: API Authentication
nav_title: Authentication
nav_order: 2
category: API Reference
subcategory: Security
tags: [auth, oauth, jwt]
toc: true
---
```

**Features**:
- Hierarchical navigation (categories → subcategories → pages)
- Auto-generated table of contents
- Code syntax highlighting
- Search across all docs
- Version selector (future)

**URL Structure**:
- `/docs` - Documentation home
- `/docs/guides/getting-started` - Category/page structure
- `/docs/api/authentication` - Nested categories
- `/docs/search?q=oauth` - Search results

**Special Pages**:
- `index.md` - Documentation home page
- `{category}/index.md` - Category landing pages
- `404.md` - Custom 404 page

#### 2. Blog Template

**Purpose**: Developer blogs, changelogs, company news

**Frontmatter Convention**:
```yaml
---
title: Introducing Docula V2
slug: introducing-docula-v2
date: 2024-01-15
author: John Doe
author_email: john@example.com
tags: [announcement, docula, release]
excerpt: Living documentation powered by cloud storage and reactive updates
featured_image: /images/docula-v2-hero.png
draft: false
---
```

**Features**:
- Chronological post listing
- Tag-based filtering
- Author pages
- RSS/Atom feeds (via template)
- Draft posts (hidden unless ?draft=true)
- Related posts

**URL Structure**:
- `/blog` - Blog home with paginated posts
- `/blog/2024/01/introducing-docula-v2` - Individual posts
- `/blog/tags/announcement` - Posts by tag
- `/blog/authors/john-doe` - Posts by author
- `/blog/feed.xml` - RSS feed

**Special Features**:
```go
// Auto-generate RSS feed
func (b *BlogSite) GenerateRSS() []byte {
    feed := &feeds.Feed{
        Title: b.Config.Title,
        Link:  &feeds.Link{Href: b.Config.URL},
        // ... populate from posts
    }
    rss, _ := feed.ToRss()
    return []byte(rss)
}

// Recent posts widget via HTMX
<div hx-get="/blog/recent" hx-trigger="load">
    Loading recent posts...
</div>
```

#### 3. Knowledge Base Template

**Purpose**: Support docs, FAQs, internal wikis

**Frontmatter Convention**:
```yaml
---
title: How to Reset Your Password
kb_id: KB-001
category: Account Management
status: published
helpful_count: 0
related: [KB-002, KB-003]
last_reviewed: 2024-01-15
---
```

**Features**:
- Article search with relevance scoring
- Helpful/not helpful voting (HTMX)
- Related articles
- Category browsing
- Popular articles
- Recently updated feed

**URL Structure**:
- `/kb` - Knowledge base home
- `/kb/account-management` - Category view
- `/kb/articles/KB-001` - Individual article
- `/kb/search` - Advanced search page

**Interactive Features**:
```html
<!-- Helpful voting with HTMX -->
<div class="article-feedback">
    Was this helpful?
    <button hx-post="/kb/articles/{{.Article.ID}}/helpful" 
            hx-swap="outerHTML">
        Yes ({{.Article.HelpfulCount}})
    </button>
    <button hx-post="/kb/articles/{{.Article.ID}}/not-helpful" 
            hx-swap="outerHTML">
        No
    </button>
</div>

<!-- Live search suggestions -->
<input type="search" 
       hx-get="/kb/suggestions" 
       hx-trigger="keyup changed delay:300ms"
       hx-target="#suggestions">
<div id="suggestions"></div>
```

#### Site Configuration

Each template is configured via the DoculaContract:

```go
type SiteConfig struct {
    Template    string            `yaml:"template"`    // "docs", "blog", "kb"
    BasePath    string            `yaml:"base_path"`   // URL prefix
    Title       string            `yaml:"title"`
    Description string            `yaml:"description"`
    Features    map[string]bool   `yaml:"features"`    // Toggle features
    Theme       map[string]string `yaml:"theme"`       // Colors, fonts
}

// Example configuration
contract := DoculaContract{
    Sites: []SiteConfig{
        {
            Template:    "docs",
            BasePath:    "/docs",
            Title:       "ZBZ Documentation",
            Features: map[string]bool{
                "search": true,
                "toc":    true,
            },
        },
        {
            Template:    "blog",
            BasePath:    "/blog",
            Title:       "ZBZ Developer Blog",
            Features: map[string]bool{
                "rss":      true,
                "comments": false,
            },
        },
        {
            Template:    "kb",
            BasePath:    "/support",
            Title:       "ZBZ Support Center",
            Features: map[string]bool{
                "voting":     true,
                "suggestions": true,
            },
        },
    },
}
```

#### Shared Components

All templates share common building blocks:

```go
// Base page structure all templates extend
type BasePage struct {
    Title       string
    Content     template.HTML
    Metadata    map[string]interface{}
    Site        SiteConfig
}

// Common navigation builder
type Navigation struct {
    Items []NavItem
    Active string
}

// Shared search interface
type SearchProvider interface {
    Search(query string, filters map[string]string) []SearchResult
    Suggest(query string) []string
}
```

#### Customization Points

1. **Templates**: Override any template by providing custom .tmpl files
2. **Styles**: Inject custom CSS or override CSS variables
3. **Components**: Add custom HTMX components
4. **Processors**: Add custom markdown processors
5. **Metadata**: Extend frontmatter with custom fields

This template system turns Docula into a complete content management solution while maintaining the simplicity of markdown files in cloud storage.

### Database Schema Specification for NLQ

A structured, living document that describes database schemas comprehensively for both human understanding and LLM-powered natural language queries.

#### Schema Document Structure

```yaml
# schema.yaml - Main database schema specification
version: "1.0"
name: "E-Commerce Database"
description: "Multi-tenant e-commerce platform with user management, products, and orders"
dialect: "postgresql"

# Business context for better NLQ understanding
business_context:
  domain: "e-commerce"
  terminology:
    customer: ["user", "buyer", "shopper"]
    product: ["item", "merchandise", "sku"]
    order: ["purchase", "transaction", "sale"]
  common_queries:
    - "Show me all orders from last month"
    - "Which products are low in stock?"
    - "Find customers who haven't ordered recently"

# Tables with rich metadata
tables:
  users:
    description: "System users who can place orders"
    business_purpose: "Stores customer information for authentication and order history"
    
    columns:
      id:
        type: "uuid"
        primary_key: true
        description: "Unique user identifier"
        
      email:
        type: "varchar(255)"
        unique: true
        nullable: false
        description: "User's email address for login"
        business_rules:
          - "Must be unique across all tenants"
          - "Used as primary login credential"
        examples: ["john@example.com", "jane.doe@company.org"]
        
      created_at:
        type: "timestamp"
        description: "When the user account was created"
        default: "CURRENT_TIMESTAMP"
        nlq_hints:
          - "Use for 'new customers' queries"
          - "Combine with orders for retention analysis"
    
    relationships:
      orders:
        type: "one_to_many"
        target: "orders"
        foreign_key: "user_id"
        description: "All orders placed by this user"
        
    nlq_examples:
      - query: "Find users who signed up last week"
        hint: "Filter by created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)"
      - query: "Show me users without any orders"
        hint: "LEFT JOIN orders and check for NULL"

  products:
    description: "Items available for purchase"
    business_purpose: "Product catalog with pricing and inventory"
    
    columns:
      sku:
        type: "varchar(50)"
        primary_key: true
        description: "Stock keeping unit - unique product identifier"
        pattern: "^[A-Z]{3}-[0-9]{6}$"
        examples: ["ABC-123456", "XYZ-789012"]
        
      name:
        type: "varchar(200)"
        description: "Product display name"
        searchable: true
        nlq_hints:
          - "Users often search by partial name"
          - "Consider using ILIKE for case-insensitive search"
          
      quantity_in_stock:
        type: "integer"
        description: "Current inventory level"
        business_rules:
          - "Low stock threshold is 10 units"
          - "Zero means out of stock"
        nlq_hints:
          - "For 'low stock' use: quantity_in_stock < 10"
          - "For 'available' use: quantity_in_stock > 0"

# Views for common access patterns
views:
  customer_lifetime_value:
    description: "Aggregated customer spending and order history"
    source_sql: |
      SELECT 
        u.id,
        u.email,
        COUNT(o.id) as order_count,
        SUM(o.total_amount) as lifetime_value,
        MAX(o.created_at) as last_order_date
      FROM users u
      LEFT JOIN orders o ON u.id = o.user_id
      GROUP BY u.id, u.email
    nlq_hints:
      - "Use for 'valuable customers' queries"
      - "Sort by lifetime_value DESC for 'top customers'"

# Common query patterns for NLQ training
query_patterns:
  - name: "Date range filtering"
    example: "orders from January"
    sql_pattern: "WHERE created_at >= '2024-01-01' AND created_at < '2024-02-01'"
    
  - name: "Aggregation with grouping"
    example: "total sales by product category"
    sql_pattern: |
      SELECT category, SUM(total_amount) as total_sales
      FROM orders o
      JOIN order_items oi ON o.id = oi.order_id
      JOIN products p ON oi.product_sku = p.sku
      GROUP BY p.category
```

#### Metadata Enrichment via Flux

```yaml
# metadata/users.yaml - Additional context for users table
table: users
domain_knowledge:
  - "Users can have multiple addresses but one primary"
  - "Email verification required before first purchase"
  - "GDPR requires data deletion after 7 years of inactivity"
  
common_filters:
  active_users: "last_login_at > DATE_SUB(NOW(), INTERVAL 30 DAY)"
  verified_users: "email_verified = true"
  
business_metrics:
  - name: "Customer Acquisition Cost"
    involves: ["marketing_spend", "new_user_count"]
  - name: "User Retention Rate"
    calculation: "users_with_repeat_orders / total_users"
```

#### Integration with Docula

```go
type DatabaseSchemaContract struct {
    Name        string
    Description string
    Storage     *hodor.HodorContract // For schema.yaml and metadata files
    AutoGenerate bool                 // Generate from zbz models
}

// Generate schema from zbz models
func (d *doculaService) generateDatabaseSchema() {
    schema := &DatabaseSchema{
        Version: "1.0",
        Dialect: d.database.Dialect(),
        Tables:  make(map[string]*TableSchema),
    }
    
    // Extract from registered models
    for _, model := range d.registeredModels {
        table := d.extractTableSchema(model)
        schema.Tables[table.Name] = table
    }
    
    // Merge with user-provided metadata
    d.mergeMetadata(schema)
    
    // Publish to hodor
    d.publishSchema(schema)
}

// NLQ-ready endpoint
func (d *doculaService) HandleNLQSchema(w http.ResponseWriter, r *http.Request) {
    // Return enriched schema for LLM consumption
    schema := d.getCachedSchema()
    
    // Add request-specific context
    schema.Context = map[string]interface{}{
        "current_time": time.Now(),
        "timezone":     r.Header.Get("X-Timezone"),
        "user_role":    r.Header.Get("X-User-Role"),
    }
    
    json.NewEncoder(w).Encode(schema)
}
```

#### Benefits of This Approach

1. **Human Readable**: YAML format with descriptions and examples
2. **LLM Optimized**: Rich context, examples, and query patterns
3. **Living Document**: Updates automatically from code changes
4. **Extensible**: Add domain knowledge without touching code
5. **Version Controlled**: Schema evolution tracked over time

#### NLQ Implementation Example

```go
// LLM receives schema + natural language query
query := "Show me customers who bought electronics last month but haven't ordered this month"

// LLM generates SQL using schema context:
/*
SELECT DISTINCT u.id, u.email, u.name
FROM users u
JOIN orders o1 ON u.id = o1.user_id
JOIN order_items oi ON o1.id = oi.order_id
JOIN products p ON oi.product_sku = p.sku
WHERE p.category = 'Electronics'
  AND o1.created_at >= DATE_SUB(NOW(), INTERVAL 2 MONTH)
  AND o1.created_at < DATE_SUB(NOW(), INTERVAL 1 MONTH)
  AND NOT EXISTS (
    SELECT 1 FROM orders o2
    WHERE o2.user_id = u.id
    AND o2.created_at >= DATE_SUB(NOW(), INTERVAL 1 MONTH)
  )
*/
```

This schema specification becomes the bridge between your database and natural language, enabling powerful NLQ features while maintaining human readability.

---

_This living documentation system transforms API docs from static generation to dynamic, collaborative, cloud-powered documentation that grows with your API._
