package docula

import (
	"context"
	"time"

	"zbz/universal"
)

// DocumentContract represents typed content within the docula service
// Embeds universal.DataAccess[T] for compile-time guarantee and universal compatibility
type DocumentContract[T any] interface {
	universal.DataAccess[T] // Embedded interface provides automatic compile-time guarantee
	
	// Content-specific extensions beyond universal interface
	Render(ctx context.Context, document T, format ContentFormat) ([]byte, error)
	Process(ctx context.Context, raw []byte, processor ContentProcessor) (T, error)
	Template(ctx context.Context, document T, templateName string, data any) ([]byte, error)
	Search(ctx context.Context, query ContentQuery) ([]T, error)
	
	// Metadata operations
	GetMetadata(ctx context.Context, resource universal.ResourceURI) (ContentMetadata, error)
	SetMetadata(ctx context.Context, resource universal.ResourceURI, metadata ContentMetadata) error
	
	// Collection operations
	ListCollections(ctx context.Context) ([]string, error)
	CreateCollection(ctx context.Context, name string, config CollectionConfig) error
	
	// Content lifecycle operations
	Publish(ctx context.Context, document T) error
	Unpublish(ctx context.Context, resource universal.ResourceURI) error
	GetVersions(ctx context.Context, resource universal.ResourceURI) ([]ContentVersion, error)
	CreateVersion(ctx context.Context, resource universal.ResourceURI, document T) (string, error)
	
	// Content-specific methods
	CollectionName() string
	ContentType() string
	Provider() string
}

// zDocumentContract is the concrete implementation of DocumentContract[T]
// Uses 'z' prefix following existing ZBZ pattern (z = self)
type zDocumentContract[T any] struct {
	collectionName string           // Collection name: "docs", "blog", "wiki"
	providerKey    string           // Provider key: "default", "github", "s3"
	service        *zDoculaService  // Reference to singleton service
	contentType    ContentType      // Type of content being managed
	processor      ContentProcessor // Content processor for this type
}

// ContentType defines the type of content being managed
type ContentType string

const (
	ContentTypeMarkdown   ContentType = "markdown"
	ContentTypeBlog       ContentType = "blog"
	ContentTypeWiki       ContentType = "wiki"
	ContentTypeKnowledge  ContentType = "knowledge"
	ContentTypeOpenAPI    ContentType = "openapi"
	ContentTypeTemplate   ContentType = "template"
	ContentTypeJSON       ContentType = "json"
	ContentTypeYAML       ContentType = "yaml"
)

// ContentFormat defines output formats for rendering
type ContentFormat string

const (
	FormatHTML     ContentFormat = "html"
	FormatMarkdown ContentFormat = "markdown"
	FormatJSON     ContentFormat = "json"
	FormatYAML     ContentFormat = "yaml"
	FormatXML      ContentFormat = "xml"
	FormatPDF      ContentFormat = "pdf"
	FormatText     ContentFormat = "text"
)

// ContentMetadata represents metadata associated with content
type ContentMetadata struct {
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Author      string            `json:"author,omitempty"`
	Created     time.Time         `json:"created"`
	Modified    time.Time         `json:"modified"`
	Published   *time.Time        `json:"published,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Category    string            `json:"category,omitempty"`
	Status      ContentStatus     `json:"status"`
	Version     string            `json:"version,omitempty"`
	Language    string            `json:"language,omitempty"`
	Format      ContentFormat     `json:"format"`
	Size        int64             `json:"size,omitempty"`
	Checksum    string            `json:"checksum,omitempty"`
	Custom      map[string]any    `json:"custom,omitempty"`
}

// ContentStatus represents the publication status of content
type ContentStatus string

const (
	StatusDraft     ContentStatus = "draft"
	StatusReview    ContentStatus = "review"
	StatusPublished ContentStatus = "published"
	StatusArchived  ContentStatus = "archived"
	StatusDeleted   ContentStatus = "deleted"
)

// ContentVersion represents a version of content
type ContentVersion struct {
	ID          string            `json:"id"`
	Version     string            `json:"version"`
	Created     time.Time         `json:"created"`
	Author      string            `json:"author,omitempty"`
	Message     string            `json:"message,omitempty"`
	Size        int64             `json:"size"`
	Checksum    string            `json:"checksum"`
	Metadata    ContentMetadata   `json:"metadata"`
}

// CollectionConfig represents configuration for a content collection
type CollectionConfig struct {
	Name        string            `json:"name"`
	ContentType ContentType       `json:"content_type"`
	Description string            `json:"description,omitempty"`
	Settings    map[string]any    `json:"settings,omitempty"`
	Processors  []string          `json:"processors,omitempty"`
	Templates   []string          `json:"templates,omitempty"`
	Metadata    ContentMetadata   `json:"metadata,omitempty"`
}

// ContentQuery represents a search query for content
type ContentQuery struct {
	Text        string            `json:"text,omitempty"`
	Collections []string          `json:"collections,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Categories  []string          `json:"categories,omitempty"`
	Authors     []string          `json:"authors,omitempty"`
	Status      []ContentStatus   `json:"status,omitempty"`
	DateFrom    *time.Time        `json:"date_from,omitempty"`
	DateTo      *time.Time        `json:"date_to,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
	SortBy      string            `json:"sort_by,omitempty"`
	SortOrder   string            `json:"sort_order,omitempty"`
	Filters     map[string]any    `json:"filters,omitempty"`
}

// ContentProcessor interface for processing different content types
type ContentProcessor interface {
	Process(ctx context.Context, raw []byte, metadata ContentMetadata) (ProcessedContent, error)
	Render(ctx context.Context, content ProcessedContent, format ContentFormat) ([]byte, error)
	Validate(ctx context.Context, content ProcessedContent) error
	GetSupportedFormats() []ContentFormat
	GetContentType() ContentType
}

// ProcessedContent represents processed content ready for storage/rendering
type ProcessedContent struct {
	Raw         []byte          `json:"raw"`
	Processed   []byte          `json:"processed"`
	Metadata    ContentMetadata `json:"metadata"`
	Frontmatter map[string]any  `json:"frontmatter,omitempty"`
	TOC         []TOCEntry      `json:"toc,omitempty"`
	Links       []ContentLink   `json:"links,omitempty"`
	Images      []ContentImage  `json:"images,omitempty"`
}

// TOCEntry represents a table of contents entry
type TOCEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Level int    `json:"level"`
	Anchor string `json:"anchor,omitempty"`
}

// ContentLink represents a link found in content
type ContentLink struct {
	URL    string `json:"url"`
	Title  string `json:"title,omitempty"`
	Type   string `json:"type"` // "internal", "external", "anchor"
	Valid  bool   `json:"valid"`
}

// ContentImage represents an image found in content
type ContentImage struct {
	URL     string `json:"url"`
	Alt     string `json:"alt,omitempty"`
	Title   string `json:"title,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Size    int64  `json:"size,omitempty"`
	Format  string `json:"format,omitempty"`
}

// Common content types that implement the contract

// MarkdownDocument represents a markdown document
type MarkdownDocument struct {
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Frontmatter map[string]any    `json:"frontmatter,omitempty"`
	Metadata    ContentMetadata   `json:"metadata"`
	TOC         []TOCEntry        `json:"toc,omitempty"`
	Links       []ContentLink     `json:"links,omitempty"`
	Images      []ContentImage    `json:"images,omitempty"`
}

// BlogPost represents a blog post
type BlogPost struct {
	Title       string          `json:"title"`
	Content     string          `json:"content"`
	Excerpt     string          `json:"excerpt,omitempty"`
	Author      string          `json:"author"`
	AuthorEmail string          `json:"author_email,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Category    string          `json:"category,omitempty"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
	FeaturedImage string        `json:"featured_image,omitempty"`
	Slug        string          `json:"slug"`
	Draft       bool            `json:"draft"`
	Metadata    ContentMetadata `json:"metadata"`
}

// WikiPage represents a wiki page
type WikiPage struct {
	Title      string          `json:"title"`
	Content    string          `json:"content"`
	Category   string          `json:"category,omitempty"`
	Namespace  string          `json:"namespace,omitempty"`
	LastEditor string          `json:"last_editor,omitempty"`
	Version    int             `json:"version"`
	Protected  bool            `json:"protected"`
	Template   string          `json:"template,omitempty"`
	Redirects  []string        `json:"redirects,omitempty"`
	Metadata   ContentMetadata `json:"metadata"`
}

// KnowledgeArticle represents a knowledge base article
type KnowledgeArticle struct {
	ID           string          `json:"id"`
	Title        string          `json:"title"`
	Content      string          `json:"content"`
	Category     string          `json:"category"`
	Subcategory  string          `json:"subcategory,omitempty"`
	Status       ContentStatus   `json:"status"`
	HelpfulCount int             `json:"helpful_count"`
	ViewCount    int64           `json:"view_count"`
	Related      []string        `json:"related,omitempty"`
	LastReviewed *time.Time      `json:"last_reviewed,omitempty"`
	Difficulty   string          `json:"difficulty,omitempty"`
	EstimatedTime string         `json:"estimated_time,omitempty"`
	Metadata     ContentMetadata `json:"metadata"`
}

// OpenAPIDocument represents an OpenAPI specification
type OpenAPIDocument struct {
	OpenAPI     string                    `json:"openapi"`
	Info        OpenAPIInfo               `json:"info"`
	Paths       map[string]OpenAPIPath    `json:"paths,omitempty"`
	Components  *OpenAPIComponents        `json:"components,omitempty"`
	Servers     []OpenAPIServer           `json:"servers,omitempty"`
	Security    []map[string][]string     `json:"security,omitempty"`
	Tags        []OpenAPITag              `json:"tags,omitempty"`
	Metadata    ContentMetadata           `json:"metadata"`
}

// OpenAPIInfo contains API metadata
type OpenAPIInfo struct {
	Title          string  `json:"title"`
	Description    string  `json:"description,omitempty"`
	Version        string  `json:"version"`
	TermsOfService string  `json:"termsOfService,omitempty"`
	Contact        *OpenAPIContact `json:"contact,omitempty"`
	License        *OpenAPILicense `json:"license,omitempty"`
}

// OpenAPIContact contains contact information
type OpenAPIContact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// OpenAPILicense contains license information
type OpenAPILicense struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// OpenAPIServer represents a server
type OpenAPIServer struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]OpenAPIVariable `json:"variables,omitempty"`
}

// OpenAPIVariable represents a server variable
type OpenAPIVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
}

// OpenAPITag represents a tag
type OpenAPITag struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ExternalDocs *OpenAPIExternalDocs `json:"externalDocs,omitempty"`
}

// OpenAPIExternalDocs represents external documentation
type OpenAPIExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// OpenAPIPath represents an API path with operations
type OpenAPIPath struct {
	Get     *OpenAPIOperation `json:"get,omitempty"`
	Post    *OpenAPIOperation `json:"post,omitempty"`
	Put     *OpenAPIOperation `json:"put,omitempty"`
	Delete  *OpenAPIOperation `json:"delete,omitempty"`
	Options *OpenAPIOperation `json:"options,omitempty"`
	Head    *OpenAPIOperation `json:"head,omitempty"`
	Patch   *OpenAPIOperation `json:"patch,omitempty"`
	Trace   *OpenAPIOperation `json:"trace,omitempty"`
}

// OpenAPIOperation represents an API operation
type OpenAPIOperation struct {
	OperationID string                    `json:"operationId,omitempty"`
	Summary     string                    `json:"summary,omitempty"`
	Description string                    `json:"description,omitempty"`
	Tags        []string                  `json:"tags,omitempty"`
	Parameters  []OpenAPIParameter        `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody       `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
	Security    []map[string][]string     `json:"security,omitempty"`
	Deprecated  bool                      `json:"deprecated,omitempty"`
}

// OpenAPIParameter represents a parameter
type OpenAPIParameter struct {
	Name        string          `json:"name"`
	In          string          `json:"in"` // "query", "header", "path", "cookie"
	Description string          `json:"description,omitempty"`
	Required    bool            `json:"required,omitempty"`
	Schema      *OpenAPISchema  `json:"schema,omitempty"`
	Example     any             `json:"example,omitempty"`
}

// OpenAPIRequestBody represents a request body
type OpenAPIRequestBody struct {
	Description string                       `json:"description,omitempty"`
	Content     map[string]OpenAPIMediaType  `json:"content"`
	Required    bool                         `json:"required,omitempty"`
}

// OpenAPIResponse represents a response
type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Headers     map[string]OpenAPIHeader    `json:"headers,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

// OpenAPIHeader represents a header
type OpenAPIHeader struct {
	Description string         `json:"description,omitempty"`
	Schema      *OpenAPISchema `json:"schema,omitempty"`
	Example     any            `json:"example,omitempty"`
}

// OpenAPIMediaType represents a media type
type OpenAPIMediaType struct {
	Schema   *OpenAPISchema         `json:"schema,omitempty"`
	Example  any                    `json:"example,omitempty"`
	Examples map[string]OpenAPIExample `json:"examples,omitempty"`
}

// OpenAPIExample represents an example
type OpenAPIExample struct {
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	Value       any    `json:"value,omitempty"`
}

// OpenAPIComponents contains reusable components
type OpenAPIComponents struct {
	Schemas         map[string]OpenAPISchema       `json:"schemas,omitempty"`
	Responses       map[string]OpenAPIResponse     `json:"responses,omitempty"`
	Parameters      map[string]OpenAPIParameter    `json:"parameters,omitempty"`
	Examples        map[string]OpenAPIExample      `json:"examples,omitempty"`
	RequestBodies   map[string]OpenAPIRequestBody  `json:"requestBodies,omitempty"`
	Headers         map[string]OpenAPIHeader       `json:"headers,omitempty"`
	SecuritySchemes map[string]OpenAPISecurityScheme `json:"securitySchemes,omitempty"`
}

// OpenAPISchema represents a data schema
type OpenAPISchema struct {
	Type                 string                    `json:"type,omitempty"`
	Format               string                    `json:"format,omitempty"`
	Description          string                    `json:"description,omitempty"`
	Properties           map[string]OpenAPISchema  `json:"properties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	Items                *OpenAPISchema            `json:"items,omitempty"`
	AdditionalProperties any                       `json:"additionalProperties,omitempty"`
	Example              any                       `json:"example,omitempty"`
	Default              any                       `json:"default,omitempty"`
	Enum                 []any                     `json:"enum,omitempty"`
}

// OpenAPISecurityScheme represents a security scheme
type OpenAPISecurityScheme struct {
	Type         string            `json:"type"`
	Description  string            `json:"description,omitempty"`
	Name         string            `json:"name,omitempty"`
	In           string            `json:"in,omitempty"`
	Scheme       string            `json:"scheme,omitempty"`
	BearerFormat string            `json:"bearerFormat,omitempty"`
	Flows        *OpenAPIOAuthFlows `json:"flows,omitempty"`
	OpenIDConnectURL string        `json:"openIdConnectUrl,omitempty"`
}

// OpenAPIOAuthFlows represents OAuth flows
type OpenAPIOAuthFlows struct {
	Implicit          *OpenAPIOAuthFlow `json:"implicit,omitempty"`
	Password          *OpenAPIOAuthFlow `json:"password,omitempty"`
	ClientCredentials *OpenAPIOAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OpenAPIOAuthFlow `json:"authorizationCode,omitempty"`
}

// OpenAPIOAuthFlow represents an OAuth flow
type OpenAPIOAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}