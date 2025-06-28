# Docula Universal Transformation Design

## Vision: From Documentation Tool to Universal Content System

Docula transforms from a documentation-specific service to a **universal content management system** using the proven ZBZ patterns.

## Core Transformation

### Current Architecture (Limited)
```
Service → MarkdownProcessor → OpenAPI Generator → Flux Watcher
```

### Universal Architecture (Powerful)
```
DocumentContract[T] → universal.DataAccess[T] → ContentProvider → CapitanHooks
```

## Universal Interface Implementation

### DocumentContract[T] - The Core Interface

```go
// DocumentContract represents typed content within the docula service
// Embeds universal.DataAccess[T] for compile-time guarantee and universal compatibility
type DocumentContract[T any] interface {
    universal.DataAccess[T] // Embedded interface provides automatic compile-time guarantee
    
    // Content-specific extensions beyond universal interface
    Render(ctx context.Context, document T, format ContentFormat) ([]byte, error)
    Process(ctx context.Context, raw []byte, processor ContentProcessor) (T, error)
    Template(ctx context.Context, document T, template string, data any) ([]byte, error)
    Search(ctx context.Context, query ContentQuery) ([]T, error)
    
    // Metadata operations
    GetMetadata(ctx context.Context, resource universal.ResourceURI) (ContentMetadata, error)
    SetMetadata(ctx context.Context, resource universal.ResourceURI, metadata ContentMetadata) error
    
    // Collection operations
    ListCollections(ctx context.Context) ([]string, error)
    CreateCollection(ctx context.Context, name string, config CollectionConfig) error
    
    // Content-specific methods
    Publish(ctx context.Context, document T) error
    Unpublish(ctx context.Context, resource universal.ResourceURI) error
    GetVersions(ctx context.Context, resource universal.ResourceURI) ([]ContentVersion, error)
}
```

### Content Types (T) - Any Content Structure

```go
// Markdown documents
type MarkdownDocument struct {
    Title      string            `json:"title"`
    Content    string            `json:"content"`
    Frontmatter map[string]any   `json:"frontmatter"`
    Metadata   ContentMetadata   `json:"metadata"`
}

// OpenAPI specifications
type OpenAPIDocument struct {
    OpenAPI    string                    `json:"openapi"`
    Info       OpenAPIInfo              `json:"info"`
    Paths      map[string]OpenAPIPath   `json:"paths"`
    Metadata   ContentMetadata          `json:"metadata"`
}

// Blog posts
type BlogPost struct {
    Title      string          `json:"title"`
    Content    string          `json:"content"`
    Author     string          `json:"author"`
    Tags       []string        `json:"tags"`
    PublishedAt time.Time      `json:"published_at"`
    Metadata   ContentMetadata `json:"metadata"`
}

// Wiki pages
type WikiPage struct {
    Title      string          `json:"title"`
    Content    string          `json:"content"`
    Category   string          `json:"category"`
    LastEditor string          `json:"last_editor"`
    Version    int             `json:"version"`
    Metadata   ContentMetadata `json:"metadata"`
}

// Knowledge base articles
type KnowledgeArticle struct {
    ID         string          `json:"id"`
    Title      string          `json:"title"`
    Content    string          `json:"content"`
    Category   string          `json:"category"`
    Status     string          `json:"status"`
    HelpfulCount int           `json:"helpful_count"`
    Metadata   ContentMetadata `json:"metadata"`
}
```

### Universal URI Operations

```go
// ResourceURI examples for different content types
"content://docs/getting-started"           // Markdown document
"content://api/user-endpoints"             // OpenAPI specification  
"content://blog/2024/01/15/release-notes" // Blog post
"content://wiki/troubleshooting/database" // Wiki page
"content://kb/KB-001"                      // Knowledge base article

// OperationURI examples for content operations
"content://operations/render?format=html"  // Render content as HTML
"content://operations/search"              // Search across content
"content://operations/publish"             // Publish draft content
"content://operations/template?name=blog"  // Apply template to content
```

## Provider Abstraction

### ContentProvider Interface

```go
type ContentProvider interface {
    // Basic content operations
    Store(ctx context.Context, uri string, content []byte, metadata ContentMetadata) error
    Retrieve(ctx context.Context, uri string) ([]byte, ContentMetadata, error)
    Delete(ctx context.Context, uri string) error
    List(ctx context.Context, pattern string) ([]ContentItem, error)
    Exists(ctx context.Context, uri string) (bool, error)
    
    // Content-specific operations
    Search(ctx context.Context, query ContentQuery) (ContentResults, error)
    GetVersions(ctx context.Context, uri string) ([]ContentVersion, error)
    CreateVersion(ctx context.Context, uri string, content []byte) (string, error)
    
    // Collection operations
    CreateCollection(ctx context.Context, name string, config CollectionConfig) error
    ListCollections(ctx context.Context) ([]string, error)
    
    // Real-time operations (for flux integration)
    Subscribe(ctx context.Context, pattern string, callback ContentChangeCallback) (SubscriptionID, error)
    Unsubscribe(ctx context.Context, id SubscriptionID) error
    
    // Provider metadata
    GetProvider() string
    GetNative() any // Provider-specific client
    Close() error
}

// Multi-provider support
var providerRegistry = map[string]ContentProviderFunction{
    "filesystem": filesystem.NewProvider,
    "s3":         s3.NewProvider,
    "github":     github.NewProvider,
    "notion":     notion.NewProvider,
    "confluence": confluence.NewProvider,
    "contentful": contentful.NewProvider,
}
```

## API Transformation

### Current API (Limited)
```go
// Current docula API - documentation specific
service := docula.NewService(contract)
service.LoadContent()
page, exists := service.GetPage("path")
html, err := service.RenderPageHTML("path")
spec, err := service.GetSpecYAML()
```

### Universal API (Powerful)
```go
// Universal docula API - any content type
docula.Register(s3.NewProvider, config)

// Different content collections with same interface
docs := docula.Documents[MarkdownDocument]("documentation")
specs := docula.Documents[OpenAPIDocument]("api-specs")
blog := docula.Documents[BlogPost]("blog")
wiki := docula.Documents[WikiPage]("wiki")
kb := docula.Documents[KnowledgeArticle]("knowledge-base")

// All implement universal.DataAccess[T] automatically
doc, err := docs.Get(ctx, universal.ParseResourceURI("content://docs/getting-started"))
posts, err := blog.List(ctx, universal.ParseResourceURI("content://blog/*"))
articles, err := kb.Execute(ctx, universal.ParseOperationURI("content://operations/search"), searchParams)
```

## Processor Architecture

### ContentProcessor Interface
```go
type ContentProcessor interface {
    Process(ctx context.Context, raw []byte, metadata ContentMetadata) (ProcessedContent, error)
    Validate(ctx context.Context, content ProcessedContent) error
    GetSupportedFormats() []ContentFormat
}

// Processor implementations
type MarkdownProcessor struct {
    goldmark goldmark.Markdown
    sanitizer *bluemonday.Policy
}

type OpenAPIProcessor struct {
    validator openapi3.T
    generator *SpecGenerator
}

type TemplateProcessor struct {
    engine template.Engine
    helpers map[string]func
}

// Processor registration
var processorRegistry = map[ContentFormat]ContentProcessor{
    FormatMarkdown: &MarkdownProcessor{},
    FormatOpenAPI:  &OpenAPIProcessor{},
    FormatTemplate: &TemplateProcessor{},
    FormatJSON:     &JSONProcessor{},
    FormatYAML:     &YAMLProcessor{},
}
```

## Capitan Hook Integration

### Content Hook Types
```go
type ContentHookType int

const (
    DocumentCreated ContentHookType = iota
    DocumentUpdated
    DocumentDeleted
    DocumentPublished
    DocumentUnpublished
    DocumentRendered
    CollectionCreated
    TemplateApplied
    SearchExecuted
)

// Hook data types
type DocumentCreatedData struct {
    Collection string          `json:"collection"`
    URI        string          `json:"uri"`
    ContentType string         `json:"content_type"`
    Author     string          `json:"author"`
    Metadata   ContentMetadata `json:"metadata"`
}

type DocumentRenderedData struct {
    URI        string        `json:"uri"`
    Format     ContentFormat `json:"format"`
    Size       int64         `json:"size"`
    Duration   time.Duration `json:"duration"`
}
```

### Automatic Hook Emission
```go
// Automatic hook emission on all content operations
func (z *zDocumentContract[T]) Set(ctx context.Context, resource universal.ResourceURI, data T) error {
    // Store the document
    err := z.provider.Store(ctx, resource.String(), serializedData, metadata)
    if err != nil {
        return err
    }
    
    // Emit capitan hook automatically
    hookData := DocumentCreatedData{
        Collection:  z.collectionName,
        URI:         resource.String(),
        ContentType: z.contentType,
        Author:      extractAuthor(ctx),
        Metadata:    metadata,
    }
    
    capitan.Emit(ctx, DocumentCreated, "docula-service", hookData, nil)
    return nil
}
```

## Multi-Site Architecture

### Site Contract with Universal Interface
```go
type SiteContract[T any] interface {
    universal.DataAccess[T] // Sites are also data accessible
    
    // Site-specific operations
    Render(ctx context.Context, path string) ([]byte, error)
    Build(ctx context.Context, outputDir string) error
    Deploy(ctx context.Context, target DeployTarget) error
    GetTheme() Theme
    SetTheme(theme Theme) error
}

type zSiteContract[T any] struct {
    siteName   string
    template   SiteTemplate
    documents  DocumentContract[T]
    theme      Theme
    config     SiteConfig
}

// Multiple sites with different content types
docs := docula.Site[MarkdownDocument]("documentation", TemplateDocsSite)
blog := docula.Site[BlogPost]("blog", TemplateBlogSite)
kb := docula.Site[KnowledgeArticle]("support", TemplateKnowledgeBaseSite)
```

## Real-Time Features via Universal Subscriptions

### Flux-Powered Live Updates
```go
// Subscribe to content changes via universal interface
subscription, err := docs.Subscribe(ctx, 
    universal.ParseResourceURI("content://docs/*"),
    func(event universal.ChangeEvent[MarkdownDocument]) {
        // Document changed - regenerate site
        site.RegenerateFromDocument(event.New)
        
        // Notify connected clients via WebSocket
        websocket.BroadcastUpdate(event)
        
        // Update search index
        search.UpdateIndex(event.New)
    },
)

// Real-time collaboration
wiki.Subscribe(ctx,
    universal.ParseResourceURI("content://wiki/*"),
    func(event universal.ChangeEvent[WikiPage]) {
        // Live wiki updates
        collaboration.NotifyEditors(event)
    },
)
```

## Provider Ecosystem

### Multiple Content Backends
```go
// Register multiple providers simultaneously
docula.Register("filesystem", filesystem.NewProvider, localConfig)
docula.Register("s3", s3.NewProvider, s3Config)
docula.Register("github", github.NewProvider, githubConfig)
docula.Register("notion", notion.NewProvider, notionConfig)

// Different collections use different providers
docs := docula.DocumentsWithProvider[MarkdownDocument]("docs", "github")      // GitHub Pages
blog := docula.DocumentsWithProvider[BlogPost]("blog", "s3")                 // S3 bucket
wiki := docula.DocumentsWithProvider[WikiPage]("wiki", "notion")             // Notion pages
specs := docula.DocumentsWithProvider[OpenAPIDocument]("specs", "filesystem") // Local files
```

### Provider-Specific Features
```go
// GitHub provider with git operations
type GitHubProvider struct {
    client *github.Client
    repo   string
    branch string
}

func (g *GitHubProvider) CreatePullRequest(ctx context.Context, content []byte, message string) (*github.PullRequest, error) {
    // Provider-specific functionality
}

// Access native provider features
githubProvider, err := docula.Provider("github")
if gitProvider, ok := githubProvider.GetNative().(*GitHubProvider); ok {
    pr, err := gitProvider.CreatePullRequest(ctx, content, "Update documentation")
}
```

## Benefits of Universal Transformation

### 1. Content Type Flexibility
```go
// Same API for any content type
type CustomContent struct {
    Data string `json:"data"`
    Meta map[string]any `json:"meta"`
}

custom := docula.Documents[CustomContent]("custom")
// All universal methods available automatically
```

### 2. Provider Independence
```go
// Switch providers without code changes
// From filesystem to S3 to GitHub - same API
docs.Set(ctx, uri, document) // Works with any provider
```

### 3. Cross-Service Integration
```go
// Content automatically available to other ZBZ services
http.RegisterCRUD(docs)           // Auto-generate REST API
database.SyncFromContent(docs)    // Sync to database
telemetry.MonitorContent(docs)    // Monitor content operations
```

### 4. Zero-Config Features
```go
// Register once, get automatic features
docula.Register(provider, config)

// Automatic features via capitan hooks:
// - Search indexing
// - Analytics tracking  
// - Version control
// - Backup and sync
// - Real-time collaboration
```

## Implementation Strategy

### Phase 1: Universal Interface Foundation
1. Create `DocumentContract[T]` interface with `universal.DataAccess[T]` embedding
2. Implement `zDocumentContract[T]` concrete type
3. Add URI-based operations for content access
4. Create `ContentProvider` interface abstraction

### Phase 2: Provider Implementation
1. Migrate existing depot integration to `ContentProvider`
2. Add filesystem, S3, GitHub providers
3. Implement provider registration system
4. Add provider-specific feature access

### Phase 3: Capitan Integration
1. Define content hook types and data structures
2. Add automatic hook emission to all operations
3. Create content adapters for common integrations
4. Enable zero-config content monitoring

### Phase 4: Multi-Format Support
1. Implement processor architecture for different content types
2. Add template system for site generation
3. Create site contracts with universal interface
4. Enable multi-site deployments

This transformation makes docula into a **universal content management platform** that can handle any content type with any provider while maintaining the simplicity and power of the ZBZ universal interface pattern.