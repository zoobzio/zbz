# HTMX Template Engine - Implementation Plan

## Overview
Design a template engine that combines Go html/template preprocessing with HTMX-ready output, using Depot contracts for storage and Flux for hot-reloading. This creates a fully customizable, strongly-typed documentation system with live template updates.

## Core Architecture

### Template Storage & Hot-Reloading
- **Depot Contracts**: Templates stored in any provider (filesystem, S3, GCS, memory)
- **Flux Integration**: Hot-reload both `.md` content AND `.tmpl` templates
- **Convention-Based**: Specific file names power different site types
- **Cloud Template Swapping**: Change templates via cloud buckets in real-time

### Template Processing Pipeline
```
.tmpl files (Go html/template) + Structured Data → HTMX-ready HTML fragments
```

1. **Templates**: `.tmpl` files with Go template syntax
2. **Data Models**: Strongly-typed Go structs (BlogPost, DocPage, etc.)
3. **Processing**: Go html/template engine with custom functions
4. **Output**: HTMX-ready HTML (full pages + partial fragments)

### Provider System
Each site type becomes a specialized provider:
- **docula-blog**: Blog-specific templates and logic
- **docula-docs**: Documentation site templates
- **docula-kb**: Knowledge base templates

## Template Convention System

### File Naming Convention
Similar to SQL macros, templates follow naming patterns:

```
templates/
├── blog/
│   ├── index.tmpl           # Blog homepage
│   ├── post.tmpl            # Individual blog post
│   ├── search-results.tmpl  # HTMX search partial
│   └── post-card.tmpl       # Reusable post component
├── docs/
│   ├── index.tmpl           # Docs homepage
│   ├── section.tmpl         # Documentation section
│   └── sidebar.tmpl         # Navigation sidebar
└── kb/
    ├── index.tmpl           # Knowledge base home
    ├── article.tmpl         # KB article
    └── category.tmpl        # Article category
```

### Template Data Interface
Each template type receives strongly-typed data:

```go
// Blog templates
type BlogTemplateData struct {
    Site  SiteConfig
    Posts []BlogPost
    Meta  PageMeta
}

// Docs templates  
type DocsTemplateData struct {
    Site     SiteConfig
    Pages    []DocPage
    Nav      NavigationTree
    Breadcrumbs []NavItem
}

// KB templates
type KBTemplateData struct {
    Site       SiteConfig
    Articles   []KBArticle
    Categories []Category
    Search     SearchConfig
}
```

## Template Replacement System

### Macro-Style Replacements
Templates use tagged replacements similar to SQL macros:

```html
<!-- blog/index.tmpl -->
<!DOCTYPE html>
<html>
<head>
    <title>{{.Site.Title}}</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body>
    <div class="search-container">
        <input type="search" 
               hx-get="/search" 
               hx-trigger="keyup changed delay:300ms"
               hx-target="#search-results">
        <div id="search-results"></div>
    </div>
    
    <main>
        {{range .Posts}}
        <!-- TEMPLATE_INCLUDE: post-card.tmpl -->
        {{end}}
    </main>
</body>
</html>
```

### Component Inclusion
- `<!-- TEMPLATE_INCLUDE: post-card.tmpl -->` - Include sub-templates
- `<!-- HTMX_PARTIAL: search-results.tmpl -->` - Mark HTMX partials
- `<!-- DATA_BINDING: BlogPost -->` - Type hints for IDE support

## Flux Hot-Reloading Integration

### Dual Watching
Flux monitors both content and templates:
- **Content**: `*.md` files trigger content regeneration
- **Templates**: `*.tmpl` files trigger template recompilation
- **Cascading Updates**: Template changes regenerate all affected pages

### Real-Time Template Swapping
```go
// User uploads new blog template to S3
contract := &depot.DepotContract{
    Provider: "s3",
    Config: map[string]interface{}{
        "bucket": "my-blog-templates",
        "region": "us-east-1",
    },
}

// Flux detects template change
flux.SyncCollection(contract, "blog/*.tmpl", func(old, new map[string][]byte) {
    // Blog automatically uses new templates
    blog.ReloadTemplates(new)
})
```

## Provider Architecture

### Site Type Providers
Each documentation type becomes a pluggable provider:

```go
// Blog Provider
type BlogProvider struct {
    templates *TemplateEngine
    content   *ContentStore
}

func (bp *BlogProvider) Render(data BlogTemplateData) ([]byte, error)
func (bp *BlogProvider) GetPartial(name string, data interface{}) ([]byte, error)
func (bp *BlogProvider) ListTemplates() []string

// Docs Provider  
type DocsProvider struct {
    templates *TemplateEngine
    content   *ContentStore
    nav       *NavigationBuilder
}

// KB Provider
type KBProvider struct {
    templates *TemplateEngine
    content   *ContentStore
    search    *SearchEngine
}
```

### Provider Registration
```go
// Register site type providers
docula.RegisterProvider("blog", NewBlogProvider)
docula.RegisterProvider("docs", NewDocsProvider) 
docula.RegisterProvider("kb", NewKBProvider)

// Use in contract
contract := DoculaContract{
    Sites: []SiteConfig{
        {
            Type: "blog",           // References BlogProvider
            Templates: blogDepot,   // Depot contract for templates
            Content: contentDepot,  // Depot contract for content
        },
    },
}
```

## Template Engine Core

### Template Processing
```go
type TemplateEngine struct {
    templates map[string]*template.Template
    functions template.FuncMap
    cache     *TemplateCache
    flux      *FluxWatcher
}

func (te *TemplateEngine) Render(name string, data interface{}) ([]byte, error)
func (te *TemplateEngine) RenderPartial(name string, data interface{}) ([]byte, error)
func (te *TemplateEngine) Reload() error
```

### Custom Template Functions
Extend Go templates with documentation-specific functions:
```go
funcMap := template.FuncMap{
    "formatDate":    formatDate,
    "excerpt":       generateExcerpt,
    "markdown":      renderMarkdown,
    "searchIndex":   buildSearchIndex,
    "htmxAttrs":     generateHTMXAttributes,
}
```

## HTMX Integration Points

### Full Pages vs Partials
- **Full Pages**: Complete HTML documents with HTMX attributes
- **Partials**: HTML fragments returned by HTMX endpoints
- **Components**: Reusable template pieces included in multiple contexts

### HTMX-Ready Output
Templates generate HTML with proper HTMX attributes:
```html
<!-- Search that works with future HTTP server -->
<input hx-get="/api/search" 
       hx-trigger="keyup changed delay:300ms"
       hx-target="#results"
       hx-swap="innerHTML">

<!-- Navigation with HTMX page loading -->
<nav hx-boost="true">
    <a href="/docs/getting-started">Getting Started</a>
</nav>
```

## Benefits of This Architecture

### For Developers
- **Strongly Typed**: Full Go type safety for template data
- **Hot Reloading**: Live updates for both content AND templates
- **Familiar Syntax**: Standard Go html/template syntax
- **Modular**: Pluggable providers for different site types

### For Users
- **Customizable**: Replace any template via Depot contracts
- **Cloud Native**: Templates can live in S3, GCS, etc.
- **Live Updates**: Change templates without deployments
- **HTMX Ready**: Interactive features work when HTTP server exists

### For Operations
- **Provider Agnostic**: Templates stored anywhere (file, cloud, memory)
- **Hot Swappable**: Update site appearance in real-time
- **Strongly Typed**: Catch template errors at compile time
- **Cacheable**: Template compilation results cached for performance

## Implementation Phases

### Phase 1: Template Engine Core
- Basic template loading from Depot contracts
- Go html/template processing with custom functions
- Template caching and reloading

### Phase 2: Flux Integration
- Hot-reloading for `.tmpl` files
- Cascading updates when templates change
- Integration with existing markdown watching

### Phase 3: Provider System
- BlogProvider, DocsProvider, KBProvider implementations
- Provider registration and factory pattern
- Site type routing in main service

### Phase 4: HTMX Enhancement
- Partial template rendering
- HTMX attribute generation helpers
- Component-based template architecture

## Technical Questions to Resolve

1. **Template Inheritance**: How do we handle layouts and partial inclusion?
2. **Data Binding**: How do we ensure type safety between Go structs and templates?
3. **Performance**: How do we cache compiled templates efficiently?
4. **Error Handling**: How do we handle template compilation errors gracefully?
5. **Versioning**: How do we handle template version compatibility?

## Next Steps

1. **Refine Architecture**: Iterate on this design until confident
2. **Prototype Core**: Build minimal template engine with Depot integration
3. **Add Flux**: Integrate hot-reloading for templates
4. **Build Providers**: Implement blog/docs/kb providers
5. **HTMX Integration**: Add partial rendering and HTMX helpers

---

*This creates a powerful, flexible documentation system where everything is hot-reloadable and customizable through cloud storage contracts.*