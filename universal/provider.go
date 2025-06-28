package universal

import (
	"context"
	"fmt"
	"time"
)

// Provider defines the universal interface that all data providers must implement
// This includes database, cache, storage, search, content, telemetry providers
// Universal package has NO knowledge of native clients - complete abstraction
// Provider methods work with []byte for complete type neutrality
type Provider interface {
	// Byte-level operations - providers implement these for type neutrality
	Get(ctx context.Context, resource ResourceURI) ([]byte, error)
	Set(ctx context.Context, resource ResourceURI, data []byte) error
	Delete(ctx context.Context, resource ResourceURI) error
	List(ctx context.Context, pattern ResourceURI) ([][]byte, error)
	Exists(ctx context.Context, resource ResourceURI) (bool, error)
	Count(ctx context.Context, pattern ResourceURI) (int64, error)
	
	// Complex operations via URI-based routing
	Execute(ctx context.Context, operation OperationURI, params []byte) ([]byte, error)
	ExecuteMany(ctx context.Context, operations []Operation) ([][]byte, error)
	
	// Real-time operations for flux integration (byte-level callbacks)
	Subscribe(ctx context.Context, pattern ResourceURI, callback ProviderChangeCallback) (SubscriptionID, error)
	Unsubscribe(ctx context.Context, id SubscriptionID) error
	
	// Provider metadata and lifecycle
	GetProvider() string           // Provider type: "postgres", "redis", "s3", etc.
	Health(ctx context.Context) (ProviderHealth, error)
	Close() error
	
	// Internal - allows universal service to inject hook emitter
	SetHookEmitter(emitter HookEmitter)
}

// ProviderChangeCallback is called when provider data changes (byte-level)
type ProviderChangeCallback func(event ProviderChangeEvent)

// ProviderChangeEvent represents a change event from a provider (byte-level)
type ProviderChangeEvent struct {
	Type      string            `json:"type"`      // "created", "updated", "deleted"
	URI       ResourceURI       `json:"uri"`       // Resource URI that changed
	OldData   []byte            `json:"old_data"`  // Previous data (nil for create)
	NewData   []byte            `json:"new_data"`  // New data (nil for delete)
	Timestamp time.Time         `json:"timestamp"` // When change occurred
	Source    string            `json:"source"`    // Provider that generated event
	Metadata  map[string]any    `json:"metadata"`  // Provider-specific metadata
}

// ProviderHealth represents the health status of a provider
type ProviderHealth struct {
	Status      string            `json:"status"`       // "healthy", "degraded", "unhealthy"
	Message     string            `json:"message"`      // Human-readable status message
	LastChecked time.Time         `json:"last_checked"` // When health was last checked
	Metrics     map[string]any    `json:"metrics"`      // Provider-specific metrics
	Version     string            `json:"version"`      // Provider version
	Latency     time.Duration     `json:"latency"`      // Average operation latency
}

// ProviderFunction creates a provider instance from configuration
type ProviderFunction func(config DataProviderConfig) (Provider, error)

// DataProviderConfig defines provider-agnostic configuration for data providers
type DataProviderConfig struct {
	// Provider identification
	ProviderKey  string `json:"provider_key,omitempty"`  // "default", "primary", "secondary"
	ProviderType string `json:"provider_type"`           // "postgres", "redis", "s3", etc.
	
	// Connection settings
	ConnectionString string `json:"connection_string,omitempty"` // DSN, URL, or connection string
	Host             string `json:"host,omitempty"`              // Server host
	Port             int    `json:"port,omitempty"`              // Server port
	Username         string `json:"username,omitempty"`          // Authentication username
	Password         string `json:"password,omitempty"`          // Authentication password
	Database         string `json:"database,omitempty"`          // Database/bucket/index name
	
	// Performance settings
	Timeout         time.Duration `json:"timeout,omitempty"`          // Operation timeout
	MaxConnections  int           `json:"max_connections,omitempty"`  // Connection pool size
	MaxRetries      int           `json:"max_retries,omitempty"`      // Retry attempts
	RetryDelay      time.Duration `json:"retry_delay,omitempty"`      // Delay between retries
	
	// Feature flags
	EnableMetrics     bool `json:"enable_metrics,omitempty"`     // Enable metrics collection
	EnableTracing     bool `json:"enable_tracing,omitempty"`     // Enable distributed tracing
	EnableHealthCheck bool `json:"enable_health_check,omitempty"` // Enable health monitoring
	EnableCache       bool `json:"enable_cache,omitempty"`       // Enable local caching
	
	// Provider-specific settings
	Settings map[string]any `json:"settings,omitempty"` // Custom provider configuration
}

// DefaultDataProviderConfig returns sensible defaults for data provider configuration
func DefaultDataProviderConfig() DataProviderConfig {
	return DataProviderConfig{
		ProviderKey:       "default",
		Timeout:           30 * time.Second,
		MaxConnections:    10,
		MaxRetries:        3,
		RetryDelay:        time.Second,
		EnableMetrics:     true,
		EnableTracing:     true,
		EnableHealthCheck: true,
		EnableCache:       false,
		Settings:          make(map[string]any),
	}
}

// HookEmitter allows providers to emit hooks for observability
type HookEmitter interface {
	Emit(ctx context.Context, hookType any, source string, data any, metadata map[string]any)
	EmitBackground(hookType any, source string, data any, metadata map[string]any)
}

// UniversalError represents errors from the universal package
type UniversalError struct {
	Code     string
	Message  string
	Provider string
}

func (e *UniversalError) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("%s: %s (provider: %s)", e.Code, e.Message, e.Provider)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// WithProvider adds provider context to the error
func (e *UniversalError) WithProvider(provider string) error {
	return &UniversalError{
		Code:     e.Code,
		Message:  e.Message,
		Provider: provider,
	}
}

// Common errors
var (
	ErrProviderNotFound = &UniversalError{
		Code:    "PROVIDER_NOT_FOUND",
		Message: "provider not found",
	}
	ErrInvalidURI = &UniversalError{
		Code:    "INVALID_URI",
		Message: "invalid URI format",
	}
	ErrSerializationFailed = &UniversalError{
		Code:    "SERIALIZATION_FAILED",
		Message: "failed to serialize/deserialize data",
	}
)