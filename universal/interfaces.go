package universal

import (
	"context"
	"time"
)

// URI Format: scheme://resource/identifier
// Examples:
//   - sql://users/123           (database: table users, id 123)
//   - bucket://assets/logo.png  (depot: bucket assets, file logo.png)
//   - cache://sessions/abc123   (cache: namespace sessions, key abc123)
//   - search://products/sku-789 (search: index products, doc sku-789)
//
// Pattern Format: scheme://resource/pattern
// Examples:
//   - sql://users/*             (all users)
//   - bucket://content/*.md     (all markdown files)
//   - cache://temp/*            (all temp cache entries)
//   - sql://posts/author:john   (posts filtered by author)
//   - bucket://**/*.json        (all JSON files recursively)

// DataAccess defines the universal interface for all data access layers
// This is implemented by database, cache, depot, search, etc.
type DataAccess[T any] interface {
	// Basic CRUD operations (Resource URI-based)
	Get(ctx context.Context, resource ResourceURI) (T, error)
	Set(ctx context.Context, resource ResourceURI, data T) error
	Delete(ctx context.Context, resource ResourceURI) error
	List(ctx context.Context, pattern ResourceURI) ([]T, error)
	Exists(ctx context.Context, resource ResourceURI) (bool, error)
	Count(ctx context.Context, pattern ResourceURI) (int64, error)

	// Complex operations (Operation URI-based)
	Execute(ctx context.Context, operation OperationURI, params any) (any, error)
	ExecuteMany(ctx context.Context, operations []Operation) ([]any, error)

	// Synchronization operations (powers flux!)
	Subscribe(ctx context.Context, pattern ResourceURI, callback ChangeCallback[T]) (SubscriptionID, error)
	Unsubscribe(ctx context.Context, id SubscriptionID) error

	// Metadata
	Name() string
	Type() string
}

// Operation represents a single data operation
type Operation struct {
	Type   string `json:"type"`   // "get", "set", "delete", "query"
	Target string `json:"target"` // table/collection/key name
	Params any    `json:"params"` // operation parameters
}

// SubscriptionID uniquely identifies a subscription
type SubscriptionID string

// ChangeCallback is called when data changes matching a subscription pattern
type ChangeCallback[T any] func(event ChangeEvent[T])

// ChangeEvent represents a data change notification
type ChangeEvent[T any] struct {
	Operation string      `json:"operation"` // "create", "update", "delete"
	Resource  ResourceURI `json:"resource"`  // Resource URI that changed
	Pattern   ResourceURI `json:"pattern"`   // Pattern that matched this change
	Old       *T          `json:"old"`       // Previous value (nil for create)
	New       *T          `json:"new"`       // New value (nil for delete)
	Source    string      `json:"source"`    // Which provider generated event
	Timestamp time.Time   `json:"timestamp"` // When the change occurred
	Metadata  map[string]any `json:"metadata,omitempty"` // Provider-specific metadata
}

// HTTPProvider defines the universal interface for HTTP providers
// This is implemented by gin, echo, chi, fiber providers
type HTTPProvider interface {
	// Provider lifecycle
	Initialize(config ProviderConfig) error
	Start(address string) error
	Stop() error
	Name() string

	// Route management
	AddRoute(method, path string, handler HandlerFunc) error
	RemoveRoute(method, path string) error
	
	// Middleware management
	AddMiddleware(middleware MiddlewareFunc) error
	
	// Request handling
	ServeHTTP(w ResponseWriter, r Request) error
	
	// Provider-specific access
	GetNative() any
}

// ProviderConfig contains universal provider configuration
type ProviderConfig struct {
	Name         string            `yaml:"name" json:"name"`
	Debug        bool              `yaml:"debug" json:"debug"`
	ReadTimeout  string            `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout string            `yaml:"write_timeout" json:"write_timeout"`
	MaxBodySize  int64             `yaml:"max_body_size" json:"max_body_size"`
	TLS          *TLSConfig        `yaml:"tls,omitempty" json:"tls,omitempty"`
	Middleware   []string          `yaml:"middleware" json:"middleware"`
	Headers      map[string]string `yaml:"headers" json:"headers"`
	CORS         *CORSConfig       `yaml:"cors,omitempty" json:"cors,omitempty"`
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
	AutoTLS  bool   `yaml:"auto_tls" json:"auto_tls"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	AllowOrigins []string `yaml:"allow_origins" json:"allow_origins"`
	AllowMethods []string `yaml:"allow_methods" json:"allow_methods"`
	AllowHeaders []string `yaml:"allow_headers" json:"allow_headers"`
}

// HandlerFunc is the universal handler function signature
type HandlerFunc func(RequestContext)

// MiddlewareFunc is the universal middleware function signature
type MiddlewareFunc func(RequestContext, NextFunc)

// NextFunc represents the next middleware in the chain
type NextFunc func()

// RequestContext provides the universal request/response interface
type RequestContext interface {
	// Request information
	Method() string
	Path() string
	PathParam(name string) string
	QueryParam(name string) string
	Header(name string) string
	BodyBytes() ([]byte, error)
	Cookie(name string) (string, error)

	// Response operations
	Status(code int)
	SetHeader(name, value string)
	SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)
	JSON(data any) error
	Data(contentType string, data []byte) error
	HTML(name string, data any) error
	Redirect(code int, url string)

	// Context storage
	Set(key string, value any)
	Get(key string) (any, bool)

	// Framework integration
	Context() context.Context
	Unwrap() any
}

// ResponseWriter is the universal response writer interface
type ResponseWriter interface {
	Header() map[string][]string
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
}

// Request is the universal request interface
type Request interface {
	Method() string
	URL() string
	Header() map[string][]string
	Body() []byte
	Context() context.Context
}