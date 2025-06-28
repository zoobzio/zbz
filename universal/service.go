package universal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"zbz/capitan"
	"zbz/cereal"
)

// Singleton registry instance (z = self pattern)
var registry *zRegistry
var registryOnce sync.Once

// zRegistry is the singleton universal provider registry
// It routes URIs to the appropriate providers without being generic itself
type zRegistry struct {
	providers   map[string]Provider  // Registered providers by scheme (db, cache, search, etc.)
	hookEmitter *capitanHookEmitter  // Hook emitter for provider integration
	mutex       sync.RWMutex         // Thread-safe access
}

// Registry returns the singleton universal registry instance
func Registry() *zRegistry {
	registryOnce.Do(func() {
		registry = &zRegistry{
			providers:   make(map[string]Provider),
			hookEmitter: &capitanHookEmitter{},
			mutex:       sync.RWMutex{},
		}
	})
	return registry
}

// Provider registration and management

// register registers a provider with the universal registry
func (r *zRegistry) register(scheme string, providerFunc ProviderFunction, config DataProviderConfig) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	// Create provider instance
	provider, err := providerFunc(config)
	if err != nil {
		return fmt.Errorf("failed to create provider for scheme '%s': %w", scheme, err)
	}
	
	// Inject hook emitter
	provider.SetHookEmitter(r.hookEmitter)
	
	// Store provider by scheme
	r.providers[scheme] = provider
	
	// Emit registration hook
	capitan.Emit(context.Background(), ProviderRegistered, "universal-registry", ProviderRegisteredData{
		Scheme:       scheme,
		ProviderType: provider.GetProvider(),
		Timestamp:    time.Now(),
	}, nil)
	
	return nil
}

// routeToProvider gets the provider for a URI scheme
func (r *zRegistry) routeToProvider(scheme string) (Provider, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	provider, exists := r.providers[scheme]
	if !exists {
		return nil, ErrProviderNotFound.WithProvider(scheme)
	}
	
	return provider, nil
}

// Package-level DataAccess[T] implementation
// These functions route URIs to the appropriate providers

// Get retrieves a resource by URI - routes to provider based on URI scheme
func Get[T any](ctx context.Context, resource ResourceURI) (T, error) {
	var zero T
	
	// Route to provider
	provider, err := Registry().routeToProvider(resource.Service())
	if err != nil {
		return zero, err
	}
	
	// Get raw bytes from provider
	data, err := provider.Get(ctx, resource)
	if err != nil {
		return zero, err
	}
	
	// Deserialize to target type using cereal
	var result T
	if err := cereal.JSON.Unmarshal(data, &result); err != nil {
		return zero, fmt.Errorf("failed to deserialize response: %w", err)
	}
	
	return result, nil
}

// Set stores a resource by URI - routes to provider based on URI scheme
func Set[T any](ctx context.Context, resource ResourceURI, data T) error {
	// Route to provider
	provider, err := Registry().routeToProvider(resource.Service())
	if err != nil {
		return err
	}
	
	// Serialize data to bytes using cereal
	bytes, err := cereal.JSON.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}
	
	// Delegate to provider
	return provider.Set(ctx, resource, bytes)
}

// Delete removes a resource by URI - routes to provider based on URI scheme
func Delete[T any](ctx context.Context, resource ResourceURI) error {
	// Route to provider
	provider, err := Registry().routeToProvider(resource.Service())
	if err != nil {
		return err
	}
	
	// Delegate to provider (no serialization needed for delete)
	return provider.Delete(ctx, resource)
}

// List retrieves resources matching a pattern - routes to provider based on URI scheme
func List[T any](ctx context.Context, pattern ResourceURI) ([]T, error) {
	// Route to provider
	provider, err := Registry().routeToProvider(pattern.Service())
	if err != nil {
		return nil, err
	}
	
	// Get raw bytes list from provider
	dataList, err := provider.List(ctx, pattern)
	if err != nil {
		return nil, err
	}
	
	// Deserialize each item
	results := make([]T, len(dataList))
	for i, data := range dataList {
		if err := cereal.JSON.Unmarshal(data, &results[i]); err != nil {
			return nil, fmt.Errorf("failed to deserialize item %d: %w", i, err)
		}
	}
	
	return results, nil
}

// Exists checks if a resource exists - routes to provider based on URI scheme
func Exists[T any](ctx context.Context, resource ResourceURI) (bool, error) {
	// Route to provider
	provider, err := Registry().routeToProvider(resource.Service())
	if err != nil {
		return false, err
	}
	
	// Delegate to provider (no serialization needed)
	return provider.Exists(ctx, resource)
}

// Count counts resources matching a pattern - routes to provider based on URI scheme
func Count[T any](ctx context.Context, pattern ResourceURI) (int64, error) {
	// Route to provider
	provider, err := Registry().routeToProvider(pattern.Service())
	if err != nil {
		return 0, err
	}
	
	// Delegate to provider (no serialization needed)
	return provider.Count(ctx, pattern)
}

// Execute performs complex operations - routes to provider based on URI scheme
func Execute[T any](ctx context.Context, operation OperationURI, params any) (any, error) {
	// Route to provider
	provider, err := Registry().routeToProvider(operation.Service())
	if err != nil {
		return nil, err
	}
	
	// Serialize params to bytes
	paramBytes, err := cereal.JSON.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize params: %w", err)
	}
	
	// Execute operation
	resultBytes, err := provider.Execute(ctx, operation, paramBytes)
	if err != nil {
		return nil, err
	}
	
	// Deserialize result
	var result any
	if err := cereal.JSON.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to deserialize result: %w", err)
	}
	
	return result, nil
}

// ExecuteMany performs multiple operations - routes each to appropriate provider
func ExecuteMany[T any](ctx context.Context, operations []Operation) ([]any, error) {
	results := make([]any, len(operations))
	
	for i, op := range operations {
		// Parse operation URI to route to provider
		opURI, err := ParseOperationURI(fmt.Sprintf("%s://%s/%s", "unknown", op.Type, op.Target))
		if err != nil {
			return nil, fmt.Errorf("invalid operation %d: %w", i, err)
		}
		
		// Route to provider
		provider, err := Registry().routeToProvider(opURI.Service())
		if err != nil {
			return nil, fmt.Errorf("operation %d: %w", i, err)
		}
		
		// Serialize params
		paramBytes, err := cereal.JSON.Marshal(op.Params)
		if err != nil {
			return nil, fmt.Errorf("operation %d: failed to serialize params: %w", i, err)
		}
		
		// Execute operation
		resultBytes, err := provider.Execute(ctx, opURI, paramBytes)
		if err != nil {
			return nil, fmt.Errorf("operation %d failed: %w", i, err)
		}
		
		// Deserialize result
		var result any
		if err := cereal.JSON.Unmarshal(resultBytes, &result); err != nil {
			return nil, fmt.Errorf("operation %d: failed to deserialize result: %w", i, err)
		}
		
		results[i] = result
	}
	
	return results, nil
}

// Subscribe creates subscriptions - routes to provider based on URI scheme
func Subscribe[T any](ctx context.Context, pattern ResourceURI, callback ChangeCallback[T]) (SubscriptionID, error) {
	// Route to provider
	provider, err := Registry().routeToProvider(pattern.Service())
	if err != nil {
		return "", err
	}
	
	// Create a wrapper callback that handles deserialization
	wrapperCallback := func(event ProviderChangeEvent) {
		// Create typed change event
		typedEvent := ChangeEvent[T]{
			Operation: event.Type,
			Resource:  event.URI,
			Pattern:   pattern,
			Source:    event.Source,
			Timestamp: event.Timestamp,
			Metadata:  event.Metadata,
		}
		
		// Deserialize old data if present
		if event.OldData != nil {
			var oldValue T
			if err := cereal.JSON.Unmarshal(event.OldData, &oldValue); err == nil {
				typedEvent.Old = &oldValue
			}
		}
		
		// Deserialize new data if present
		if event.NewData != nil {
			var newValue T
			if err := cereal.JSON.Unmarshal(event.NewData, &newValue); err == nil {
				typedEvent.New = &newValue
			}
		}
		
		// Call user callback with typed event
		callback(typedEvent)
	}
	
	// Subscribe with wrapper callback
	return provider.Subscribe(ctx, pattern, wrapperCallback)
}

// Unsubscribe removes subscriptions - implementation depends on subscription ID format
func Unsubscribe[T any](ctx context.Context, id SubscriptionID) error {
	// For now, try all providers until one handles it
	// TODO: Better subscription ID format that includes provider scheme
	registry := Registry()
	registry.mutex.RLock()
	providers := make([]Provider, 0, len(registry.providers))
	for _, provider := range registry.providers {
		providers = append(providers, provider)
	}
	registry.mutex.RUnlock()
	
	for _, provider := range providers {
		if err := provider.Unsubscribe(ctx, id); err == nil {
			return nil // Successfully unsubscribed
		}
	}
	
	return fmt.Errorf("subscription %s not found in any provider", id)
}

// Registry management functions

// listProviders returns all registered provider schemes
func (r *zRegistry) listProviders() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	schemes := make([]string, 0, len(r.providers))
	for scheme := range r.providers {
		schemes = append(schemes, scheme)
	}
	
	return schemes
}

// getProviderHealth checks the health of a provider by scheme
func (r *zRegistry) getProviderHealth(ctx context.Context, scheme string) (ProviderHealth, error) {
	provider, err := r.routeToProvider(scheme)
	if err != nil {
		return ProviderHealth{}, err
	}
	
	return provider.Health(ctx)
}

// close closes all providers and cleans up resources
func (r *zRegistry) close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	var lastErr error
	
	// Close all providers
	for scheme, provider := range r.providers {
		if err := provider.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close provider for scheme '%s': %w", scheme, err)
		}
	}
	
	// Clear state
	r.providers = make(map[string]Provider)
	
	// Emit service shutdown hook
	capitan.Emit(context.Background(), ServiceShutdown, "universal-registry", ServiceShutdownData{
		Timestamp: time.Now(),
	}, nil)
	
	return lastErr
}

// Hook emitter implementation using capitan

// capitanHookEmitter implements HookEmitter using capitan
type capitanHookEmitter struct{}

// Emit emits a hook with context
func (h *capitanHookEmitter) Emit(ctx context.Context, hookType any, source string, data any, metadata map[string]any) {
	// Type assert to UniversalHookType which implements capitan.HookType
	if ht, ok := hookType.(UniversalHookType); ok {
		_ = capitan.Emit(ctx, ht, source, data, metadata)
	}
	// Silently ignore if not a proper hook type
}

// EmitBackground emits a hook in background context
func (h *capitanHookEmitter) EmitBackground(hookType any, source string, data any, metadata map[string]any) {
	// Type assert to UniversalHookType which implements capitan.HookType
	if ht, ok := hookType.(UniversalHookType); ok {
		_ = capitan.Emit(context.Background(), ht, source, data, metadata)
	}
	// Silently ignore if not a proper hook type
}

// Hook types for universal service events
type UniversalHookType int

const (
	// Provider lifecycle hooks
	ProviderRegistered UniversalHookType = iota + 3000 // Start at 3000 to avoid conflicts
	ProviderUnregistered
	ProviderHealthCheck
	
	// Data operation hooks (emitted by providers)
	DataGet
	DataSet
	DataDelete
	DataList
	DataExists
	DataExecute
	DataSubscribe
	
	// Service lifecycle hooks
	ServiceStartup
	ServiceShutdown
)

// String implements capitan.HookType interface
func (h UniversalHookType) String() string {
	switch h {
	case ProviderRegistered:
		return "provider-registered"
	case ProviderUnregistered:
		return "provider-unregistered"
	case ProviderHealthCheck:
		return "provider-health-check"
	case DataGet:
		return "data-get"
	case DataSet:
		return "data-set"
	case DataDelete:
		return "data-delete"
	case DataList:
		return "data-list"
	case DataExists:
		return "data-exists"
	case DataExecute:
		return "data-execute"
	case DataSubscribe:
		return "data-subscribe"
	case ServiceStartup:
		return "service-startup"
	case ServiceShutdown:
		return "service-shutdown"
	default:
		return "unknown-hook"
	}
}

// Hook data structures

// ProviderRegisteredData contains data for provider registration events
type ProviderRegisteredData struct {
	Scheme       string    `json:"scheme"`        // URI scheme this provider handles
	ProviderType string    `json:"provider_type"` // Type of provider (postgres, redis, etc.)
	Config       any       `json:"config,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ProviderUnregisteredData contains data for provider unregistration events
type ProviderUnregisteredData struct {
	Scheme    string    `json:"scheme"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ServiceShutdownData contains data for service shutdown events
type ServiceShutdownData struct {
	ProvidersCount int       `json:"providers_count"`
	Timestamp      time.Time `json:"timestamp"`
}

// DataOperationData contains data for data operation events
type DataOperationData struct {
	Operation string        `json:"operation"`   // "get", "set", "delete", etc.
	URI       string        `json:"uri"`         // Resource URI
	Provider  string        `json:"provider"`    // Provider that handled operation
	Duration  time.Duration `json:"duration"`    // Operation duration
	Error     string        `json:"error,omitempty"` // Error message if operation failed
	Timestamp time.Time     `json:"timestamp"`   // When operation occurred
}


// Registry health and monitoring

// GetRegistryHealth returns the overall health of the universal registry
func (r *zRegistry) getRegistryHealth(ctx context.Context) (UniversalRegistryHealth, error) {
	r.mutex.RLock()
	schemes := make([]string, 0, len(r.providers))
	for scheme := range r.providers {
		schemes = append(schemes, scheme)
	}
	r.mutex.RUnlock()
	
	// Check health of all providers
	providerHealth := make(map[string]ProviderHealth)
	overallStatus := "healthy"
	
	for _, scheme := range schemes {
		health, err := r.getProviderHealth(ctx, scheme)
		if err != nil {
			health = ProviderHealth{
				Status:  "unhealthy",
				Message: err.Error(),
			}
		}
		
		providerHealth[scheme] = health
		
		// Update overall status
		if health.Status == "unhealthy" {
			overallStatus = "unhealthy"
		} else if health.Status == "degraded" && overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}
	
	return UniversalRegistryHealth{
		Status:         overallStatus,
		ProvidersCount: len(schemes),
		ProviderHealth: providerHealth,
		LastChecked:    time.Now(),
	}, nil
}

// UniversalRegistryHealth represents the health of the universal registry
type UniversalRegistryHealth struct {
	Status         string                    `json:"status"`          // "healthy", "degraded", "unhealthy"
	ProvidersCount int                       `json:"providers_count"` // Number of registered providers
	ProviderHealth map[string]ProviderHealth `json:"provider_health"` // Health of each provider
	LastChecked    time.Time                 `json:"last_checked"`    // When health was checked
}