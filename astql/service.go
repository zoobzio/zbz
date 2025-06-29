package astql

import (
	"context"
	"fmt"
	"sync"
	"time"

	"zbz/capitan"
	"zbz/catalog"
	"zbz/zlog"
)

// ASTQLService provides non-generic access to ASTQL operations
type ASTQLService interface {
	// Provider management
	RegisterProvider(providerType string, config ProviderConfig) error
	GetProvider(providerType string) (QueryProvider, error)
	ListProviders() []string
	
	// Query execution
	ExecuteQuery(ctx context.Context, uri OperationURI, params map[string]any) (any, error)
	GenerateForType(typeName string, metadata catalog.ModelMetadata)
	
	// Health and status
	Health(ctx context.Context) (ServiceHealth, error)
	Close() error
}

// QueryProvider defines the interface all providers must implement
type QueryProvider interface {
	// Core operations
	Render(ast *QueryAST) (query string, params map[string]any, err error)
	Execute(ctx context.Context, query string, params map[string]any) (any, error)
	
	// Provider info
	GetType() string
	GetConfig() ProviderConfig
	
	// Lifecycle
	Health(ctx context.Context) (ProviderHealth, error)
	Close() error
}

// ProviderConfig provides universal configuration for all providers
type ProviderConfig struct {
	// Provider identification
	ProviderKey  string `json:"provider_key"`  // "primary", "secondary", "analytics"
	ProviderType string `json:"provider_type"` // "sql", "mongo", "elastic", "redis"
	
	// Universal connection settings
	Host         string `json:"host,omitempty"`
	Port         int    `json:"port,omitempty"`
	Database     string `json:"database,omitempty"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	
	// Universal performance settings
	MaxConnections  int           `json:"max_connections,omitempty"`
	Timeout         time.Duration `json:"timeout,omitempty"`
	MaxRetries      int           `json:"max_retries,omitempty"`
	RetryDelay      time.Duration `json:"retry_delay,omitempty"`
	
	// Universal feature flags
	EnableMetrics     bool `json:"enable_metrics,omitempty"`
	EnableTracing     bool `json:"enable_tracing,omitempty"`
	EnableHealthCheck bool `json:"enable_health_check,omitempty"`
	EnableCache       bool `json:"enable_cache,omitempty"`
	
	// Provider-specific settings (key-value pairs)
	Settings map[string]any `json:"settings,omitempty"`
}

// OperationURI represents a query operation location
type OperationURI struct {
	Scheme    string            `json:"scheme"`    // "sql", "mongo", "elastic"
	Resource  string            `json:"resource"`  // "users", "orders"
	Operation string            `json:"operation"` // "get", "list", "create", "update", "delete"
	Params    map[string]string `json:"params"`    // Additional URI parameters
	QueryPath string            `json:"query_path,omitempty"` // File path for flux watching
}

// ServiceHealth represents the health of the ASTQL service
type ServiceHealth struct {
	Status    string                      `json:"status"`
	Providers map[string]ProviderHealth   `json:"providers"`
	Timestamp time.Time                   `json:"timestamp"`
}

// ProviderHealth represents the health of a specific provider
type ProviderHealth struct {
	Status       string            `json:"status"`       // "healthy", "degraded", "unhealthy"
	Message      string            `json:"message"`      // Human-readable status
	Latency      time.Duration     `json:"latency"`      // Average query latency
	Connections  int               `json:"connections"`  // Active connections
	LastChecked  time.Time         `json:"last_checked"` // When health was checked
	Metrics      map[string]any    `json:"metrics"`      // Provider-specific metrics
}

// astqlService implements ASTQLService using the singleton pattern
type astqlService struct {
	providers map[string]QueryProvider
	configs   map[string]ProviderConfig
	mu        sync.RWMutex
}

// Singleton instance
var (
	service     ASTQLService
	serviceOnce sync.Once
)

// Service returns the singleton ASTQL service instance
func Service() ASTQLService {
	serviceOnce.Do(func() {
		service = &astqlService{
			providers: make(map[string]QueryProvider),
			configs:   make(map[string]ProviderConfig),
		}
		
		zlog.Info("Initialized ASTQL singleton service")
	})
	return service
}

// Package-level convenience functions (ZBZ pattern!)

// RegisterProvider registers a provider with the singleton service
func RegisterProvider(providerType string, config ProviderConfig) error {
	return Service().RegisterProvider(providerType, config)
}

// Execute executes a query using the singleton service
func Execute[T any](ctx context.Context, uri OperationURI, params map[string]any) (T, error) {
	result, err := Service().ExecuteQuery(ctx, uri, params)
	if err != nil {
		var zero T
		return zero, err
	}
	
	// Type assertion to ensure we return the expected type
	if typed, ok := result.(T); ok {
		return typed, nil
	}
	
	var zero T
	return zero, fmt.Errorf("result type mismatch: expected %T, got %T", zero, result)
}

// Health returns the health of all providers
func Health(ctx context.Context) (ServiceHealth, error) {
	return Service().Health(ctx)
}

// Implementation of astqlService

func (s *astqlService) RegisterProvider(providerType string, config ProviderConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Store config for later provider creation
	s.configs[providerType] = config
	
	zlog.Info("Registered ASTQL provider config",
		zlog.String("provider_type", providerType),
		zlog.String("provider_key", config.ProviderKey))
	
	// Emit event for provider registration
	capitan.Emit(context.Background(), ProviderRegistered, "astql", ProviderRegisteredEvent{
		ProviderType: providerType,
		Config:       config,
		Timestamp:    time.Now(),
	}, nil)
	
	return nil
}

func (s *astqlService) GetProvider(providerType string) (QueryProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if provider, exists := s.providers[providerType]; exists {
		return provider, nil
	}
	
	return nil, fmt.Errorf("provider %s not found", providerType)
}

func (s *astqlService) ListProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	providers := make([]string, 0, len(s.providers))
	for providerType := range s.providers {
		providers = append(providers, providerType)
	}
	
	return providers
}

func (s *astqlService) ExecuteQuery(ctx context.Context, uri OperationURI, params map[string]any) (any, error) {
	_, err := s.GetProvider(uri.Scheme)
	if err != nil {
		return nil, fmt.Errorf("provider not found for scheme %s: %w", uri.Scheme, err)
	}
	
	// TODO: Load AST from file or generate from operation
	// For now, this is a placeholder
	
	// Emit execution event
	capitan.Emit(ctx, QueryExecuting, "astql", QueryExecutingEvent{
		Provider:  uri.Scheme,
		TypeName:  uri.Resource,
		Operation: uri.Operation,
		Query:     "placeholder", // TODO: actual query
		Params:    params,
		Timestamp: time.Now(),
	}, nil)
	
	return nil, fmt.Errorf("query execution not yet implemented")
}

func (s *astqlService) GenerateForType(typeName string, metadata catalog.ModelMetadata) {
	zlog.Info("Generating queries for type via service",
		zlog.String("type_name", typeName))
	
	GenerateFromMetadata(metadata)
}

func (s *astqlService) Health(ctx context.Context) (ServiceHealth, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	health := ServiceHealth{
		Status:    "healthy",
		Providers: make(map[string]ProviderHealth),
		Timestamp: time.Now(),
	}
	
	// Check health of all providers
	for providerType, provider := range s.providers {
		providerHealth, err := provider.Health(ctx)
		if err != nil {
			providerHealth = ProviderHealth{
				Status:      "unhealthy",
				Message:     err.Error(),
				LastChecked: time.Now(),
			}
		}
		
		health.Providers[providerType] = providerHealth
		
		// If any provider is unhealthy, mark service as degraded
		if providerHealth.Status != "healthy" {
			health.Status = "degraded"
		}
	}
	
	return health, nil
}

func (s *astqlService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Close all providers
	for providerType, provider := range s.providers {
		if err := provider.Close(); err != nil {
			zlog.Warn("Error closing provider",
				zlog.String("provider_type", providerType),
				zlog.Err(err))
		}
	}
	
	zlog.Info("Closed ASTQL service")
	return nil
}

// DefaultProviderConfig returns sensible defaults for provider configuration
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		ProviderKey:       "default",
		MaxConnections:    10,
		Timeout:           30 * time.Second,
		MaxRetries:        3,
		RetryDelay:        time.Second,
		EnableMetrics:     true,
		EnableTracing:     true,
		EnableHealthCheck: true,
		EnableCache:       false,
		Settings:          make(map[string]any),
	}
}

// Provider hook types are defined in hooks.go

// ProviderRegisteredEvent is emitted when a provider is registered
type ProviderRegisteredEvent struct {
	ProviderType string         `json:"provider_type"`
	Config       ProviderConfig `json:"config"`
	Timestamp    time.Time      `json:"timestamp"`
}