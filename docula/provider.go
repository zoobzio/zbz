package docula

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ContentProvider defines the universal interface for content backends
// Compatible with filesystem, S3, GitHub, Notion, Confluence, etc.
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
	CreateVersion(ctx context.Context, uri string, content []byte, message string) (string, error)
	RestoreVersion(ctx context.Context, uri string, versionID string) error
	
	// Collection operations
	CreateCollection(ctx context.Context, name string, config CollectionConfig) error
	DeleteCollection(ctx context.Context, name string) error
	ListCollections(ctx context.Context) ([]string, error)
	GetCollectionInfo(ctx context.Context, name string) (CollectionInfo, error)
	
	// Real-time operations (for flux integration)
	Subscribe(ctx context.Context, pattern string, callback ContentChangeCallback) (SubscriptionID, error)
	Unsubscribe(ctx context.Context, id SubscriptionID) error
	
	// Batch operations
	StoreBatch(ctx context.Context, items []ContentBatchItem) error
	DeleteBatch(ctx context.Context, uris []string) error
	
	// Provider metadata and health
	GetProvider() string
	GetNative() any // Provider-specific client
	Health(ctx context.Context) (ProviderHealth, error)
	Close() error
}

// ContentItem represents a content item in a listing
type ContentItem struct {
	URI         string          `json:"uri"`
	Name        string          `json:"name"`
	Path        string          `json:"path"`
	ContentType ContentType     `json:"content_type"`
	Size        int64           `json:"size"`
	Modified    time.Time       `json:"modified"`
	Metadata    ContentMetadata `json:"metadata"`
	IsDirectory bool            `json:"is_directory"`
}

// ContentResults represents search results for content
type ContentResults struct {
	Items       []ContentItem   `json:"items"`
	Total       int64           `json:"total"`
	Query       ContentQuery    `json:"query"`
	Facets      []SearchFacet   `json:"facets,omitempty"`
	Suggestions []string        `json:"suggestions,omitempty"`
	Duration    time.Duration   `json:"duration"`
}

// SearchFacet represents a search facet for filtering
type SearchFacet struct {
	Name   string           `json:"name"`
	Values []SearchFacetValue `json:"values"`
}

// SearchFacetValue represents a facet value with count
type SearchFacetValue struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// CollectionInfo represents information about a content collection
type CollectionInfo struct {
	Name        string            `json:"name"`
	ContentType ContentType       `json:"content_type"`
	Description string            `json:"description,omitempty"`
	ItemCount   int64             `json:"item_count"`
	TotalSize   int64             `json:"total_size"`
	Created     time.Time         `json:"created"`
	Modified    time.Time         `json:"modified"`
	Settings    map[string]any    `json:"settings,omitempty"`
	Metadata    ContentMetadata   `json:"metadata,omitempty"`
}

// ContentChangeCallback is called when content changes
type ContentChangeCallback func(event ContentChangeEvent)

// ContentChangeEvent represents a content change event
type ContentChangeEvent struct {
	Type        ContentChangeType `json:"type"`
	URI         string            `json:"uri"`
	Collection  string            `json:"collection,omitempty"`
	OldContent  []byte            `json:"old_content,omitempty"`
	NewContent  []byte            `json:"new_content,omitempty"`
	OldMetadata ContentMetadata   `json:"old_metadata,omitempty"`
	NewMetadata ContentMetadata   `json:"new_metadata"`
	Timestamp   time.Time         `json:"timestamp"`
	Author      string            `json:"author,omitempty"`
	Message     string            `json:"message,omitempty"`
}

// ContentChangeType represents the type of content change
type ContentChangeType string

const (
	ChangeTypeCreated  ContentChangeType = "created"
	ChangeTypeUpdated  ContentChangeType = "updated"
	ChangeTypeDeleted  ContentChangeType = "deleted"
	ChangeTypeMoved    ContentChangeType = "moved"
	ChangeTypeRenamed  ContentChangeType = "renamed"
)

// SubscriptionID identifies a content subscription
type SubscriptionID string

// ContentBatchItem represents an item in a batch operation
type ContentBatchItem struct {
	URI      string          `json:"uri"`
	Content  []byte          `json:"content"`
	Metadata ContentMetadata `json:"metadata"`
}

// ProviderHealth represents the health status of a content provider
type ProviderHealth struct {
	Status      string         `json:"status"`      // "healthy", "degraded", "unhealthy"
	Message     string         `json:"message,omitempty"`
	LastChecked time.Time      `json:"last_checked"`
	Metrics     map[string]any `json:"metrics,omitempty"`
	Version     string         `json:"version,omitempty"`
}

// ContentProviderFunction creates a content provider instance
type ContentProviderFunction func(config ContentProviderConfig) (ContentProvider, error)

// ContentProviderConfig defines provider-agnostic content configuration
type ContentProviderConfig struct {
	// Provider settings
	ProviderKey  string `json:"provider_key,omitempty"`  // "default", "github", "s3"
	ProviderType string `json:"provider_type"`           // "filesystem", "s3", "github", "notion"
	
	// Connection settings
	BasePath     string `json:"base_path,omitempty"`     // Root path or bucket
	Region       string `json:"region,omitempty"`        // For cloud providers
	Endpoint     string `json:"endpoint,omitempty"`      // Custom endpoint
	
	// Authentication
	AccessKey    string `json:"access_key,omitempty"`    // API key or access key
	SecretKey    string `json:"secret_key,omitempty"`    // Secret key
	Token        string `json:"token,omitempty"`         // OAuth token
	Username     string `json:"username,omitempty"`      // Username for basic auth
	Password     string `json:"password,omitempty"`      // Password for basic auth
	
	// Provider-specific settings
	Repository   string `json:"repository,omitempty"`    // GitHub repository
	Branch       string `json:"branch,omitempty"`        // Git branch
	Database     string `json:"database,omitempty"`      // Notion database ID
	Space        string `json:"space,omitempty"`         // Confluence space
	
	// Performance settings
	Timeout      time.Duration `json:"timeout,omitempty"`      // Request timeout
	RetryCount   int           `json:"retry_count,omitempty"`   // Number of retries
	BatchSize    int           `json:"batch_size,omitempty"`    // Batch operation size
	CacheSize    int           `json:"cache_size,omitempty"`    // Cache size in MB
	
	// Feature flags
	EnableCache      bool `json:"enable_cache,omitempty"`      // Enable local caching
	EnableVersioning bool `json:"enable_versioning,omitempty"` // Enable version tracking
	EnableSearch     bool `json:"enable_search,omitempty"`     // Enable search indexing
	EnableWatch      bool `json:"enable_watch,omitempty"`      // Enable file watching
	
	// Custom settings
	Settings map[string]any `json:"settings,omitempty"`
}

// DefaultProviderConfig returns sensible defaults for content provider configuration
func DefaultProviderConfig() ContentProviderConfig {
	return ContentProviderConfig{
		ProviderKey:      "default",
		Timeout:          30 * time.Second,
		RetryCount:       3,
		BatchSize:        100,
		CacheSize:        100, // 100MB
		EnableCache:      true,
		EnableVersioning: true,
		EnableSearch:     true,
		EnableWatch:      true,
	}
}

// Provider registry for dynamic content providers
var providerRegistry = make(map[string]ContentProviderFunction)

// RegisterProvider registers a content provider factory
func RegisterProvider(name string, factory ContentProviderFunction) {
	providerRegistry[name] = factory
}

// NewProvider creates a provider instance by name
func NewProvider(name string, config ContentProviderConfig) (ContentProvider, error) {
	factory, exists := providerRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown content provider: %s", name)
	}
	return factory(config)
}

// ListProviders returns all registered provider names
func ListProviders() []string {
	providers := make([]string, 0, len(providerRegistry))
	for name := range providerRegistry {
		providers = append(providers, name)
	}
	return providers
}

// Common content errors
var (
	ErrContentNotFound      = fmt.Errorf("content not found")
	ErrCollectionNotFound   = fmt.Errorf("collection not found")
	ErrInvalidURI          = fmt.Errorf("invalid content URI")
	ErrProviderUnavailable = fmt.Errorf("content provider unavailable")
	ErrNotConfigured       = fmt.Errorf("docula not configured")
	ErrInvalidContentType  = fmt.Errorf("invalid content type")
	ErrVersionNotFound     = fmt.Errorf("content version not found")
	ErrCollectionExists    = fmt.Errorf("collection already exists")
	ErrInvalidMetadata     = fmt.Errorf("invalid content metadata")
)

// ContentError represents content-specific errors
type ContentError struct {
	Code     string
	Message  string
	Provider string
	URI      string
	Cause    error
}

func (e *ContentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (provider: %s, uri: %s): %v", 
			e.Code, e.Message, e.Provider, e.URI, e.Cause)
	}
	return fmt.Sprintf("%s: %s (provider: %s, uri: %s)", 
		e.Code, e.Message, e.Provider, e.URI)
}

func (e *ContentError) Unwrap() error {
	return e.Cause
}

// Helper functions for provider implementations

// ParseContentURI parses a content URI into components
func ParseContentURI(uri string) (collection, path string, err error) {
	// Parse URIs like "content://docs/getting-started" or "docs/getting-started"
	if strings.HasPrefix(uri, "content://") {
		uri = strings.TrimPrefix(uri, "content://")
	}
	
	parts := strings.SplitN(uri, "/", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid content URI format: %s", uri)
	}
	
	return parts[0], parts[1], nil
}

// BuildContentURI builds a content URI from components
func BuildContentURI(collection, path string) string {
	return fmt.Sprintf("content://%s/%s", collection, path)
}

// ValidateContentType checks if a content type is supported
func ValidateContentType(contentType ContentType) error {
	switch contentType {
	case ContentTypeMarkdown, ContentTypeBlog, ContentTypeWiki, 
		 ContentTypeKnowledge, ContentTypeOpenAPI, ContentTypeTemplate,
		 ContentTypeJSON, ContentTypeYAML:
		return nil
	default:
		return fmt.Errorf("unsupported content type: %s", contentType)
	}
}

// ValidateContentFormat checks if a content format is supported
func ValidateContentFormat(format ContentFormat) error {
	switch format {
	case FormatHTML, FormatMarkdown, FormatJSON, FormatYAML, 
		 FormatXML, FormatPDF, FormatText:
		return nil
	default:
		return fmt.Errorf("unsupported content format: %s", format)
	}
}

// GenerateContentChecksum generates a checksum for content
func GenerateContentChecksum(content []byte) string {
	// Simple checksum implementation - could use more sophisticated hashing
	hash := 0
	for _, b := range content {
		hash = hash*31 + int(b)
	}
	return fmt.Sprintf("%x", hash)
}

// ExtractContentMetadata extracts metadata from content
func ExtractContentMetadata(content []byte, contentType ContentType) ContentMetadata {
	metadata := ContentMetadata{
		Created:  time.Now(),
		Modified: time.Now(),
		Status:   StatusDraft,
		Format:   FormatFromContentType(contentType),
		Size:     int64(len(content)),
		Checksum: GenerateContentChecksum(content),
	}
	
	// Content-type specific metadata extraction would go here
	switch contentType {
	case ContentTypeMarkdown:
		// Extract frontmatter, title, etc.
	case ContentTypeBlog:
		// Extract blog-specific metadata
	case ContentTypeWiki:
		// Extract wiki-specific metadata
	}
	
	return metadata
}

// FormatFromContentType returns the default format for a content type
func FormatFromContentType(contentType ContentType) ContentFormat {
	switch contentType {
	case ContentTypeMarkdown, ContentTypeBlog, ContentTypeWiki, ContentTypeKnowledge:
		return FormatMarkdown
	case ContentTypeOpenAPI:
		return FormatYAML
	case ContentTypeJSON:
		return FormatJSON
	case ContentTypeYAML, ContentTypeTemplate:
		return FormatYAML
	default:
		return FormatText
	}
}

// ContentTypeFromFormat returns the likely content type for a format
func ContentTypeFromFormat(format ContentFormat) ContentType {
	switch format {
	case FormatMarkdown:
		return ContentTypeMarkdown
	case FormatJSON:
		return ContentTypeJSON
	case FormatYAML:
		return ContentTypeYAML
	case FormatHTML, FormatXML, FormatPDF, FormatText:
		return ContentTypeMarkdown // Default fallback
	default:
		return ContentTypeMarkdown
	}
}

// IsValidContentStatus checks if a content status is valid
func IsValidContentStatus(status ContentStatus) bool {
	switch status {
	case StatusDraft, StatusReview, StatusPublished, StatusArchived, StatusDeleted:
		return true
	default:
		return false
	}
}