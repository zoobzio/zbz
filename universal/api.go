package universal

import (
	"context"
)

// Public API functions for universal data access

// Register registers a data provider with the universal registry for a specific URI scheme
// Example: universal.Register("db", postgres.NewProvider, config)
// This allows URIs like "db://users/123" to route to the postgres provider
func Register(scheme string, providerFunc ProviderFunction, config DataProviderConfig) error {
	registry := Registry()
	return registry.register(scheme, providerFunc, config)
}

// Provider management functions

// GetProvider returns a registered provider by scheme
// Example: provider, err := universal.GetProvider("db")
func GetProvider(scheme string) (Provider, error) {
	registry := Registry()
	return registry.routeToProvider(scheme)
}

// ListProviders returns all registered provider schemes
// Example: schemes := universal.ListProviders() // ["db", "cache", "search"]
func ListProviders() []string {
	registry := Registry()
	return registry.listProviders()
}

// Health returns the health status of a specific provider by scheme
// Example: health, err := universal.Health(ctx, "db")
func Health(ctx context.Context, scheme string) (ProviderHealth, error) {
	registry := Registry()
	return registry.getProviderHealth(ctx, scheme)
}

// RegistryHealth returns the overall health of the universal registry
// Example: health, err := universal.RegistryHealth(ctx)
func RegistryHealth(ctx context.Context) (UniversalRegistryHealth, error) {
	registry := Registry()
	return registry.getRegistryHealth(ctx)
}

// Close closes all providers and shuts down the universal registry
// Example: universal.Close()
func Close() error {
	registry := Registry()
	return registry.close()
}

// Quick setup functions for common provider configurations

// SetupDatabase registers a database provider for "db" scheme
// Example: universal.SetupDatabase(postgres.NewProvider, config)
// Enables URIs like "db://users/123"
func SetupDatabase(providerFunc ProviderFunction, config DataProviderConfig) error {
	return Register("db", providerFunc, config)
}

// SetupCache registers a cache provider for "cache" scheme  
// Example: universal.SetupCache(redis.NewProvider, config)
// Enables URIs like "cache://sessions/abc123"
func SetupCache(providerFunc ProviderFunction, config DataProviderConfig) error {
	return Register("cache", providerFunc, config)
}

// SetupStorage registers a storage provider for "storage" scheme
// Example: universal.SetupStorage(s3.NewProvider, config)
// Enables URIs like "storage://documents/file.pdf"
func SetupStorage(providerFunc ProviderFunction, config DataProviderConfig) error {
	return Register("storage", providerFunc, config)
}

// SetupSearch registers a search provider for "search" scheme
// Example: universal.SetupSearch(elasticsearch.NewProvider, config)
// Enables URIs like "search://products/widget-123"
func SetupSearch(providerFunc ProviderFunction, config DataProviderConfig) error {
	return Register("search", providerFunc, config)
}

// SetupContent registers a content provider for "content" scheme
// Example: universal.SetupContent(github.NewProvider, config)
// Enables URIs like "content://docs/readme.md"
func SetupContent(providerFunc ProviderFunction, config DataProviderConfig) error {
	return Register("content", providerFunc, config)
}

// SetupMetrics registers a metrics provider for "metrics" scheme
// Example: universal.SetupMetrics(prometheus.NewProvider, config)
// Enables URIs like "metrics://api-requests/count"
func SetupMetrics(providerFunc ProviderFunction, config DataProviderConfig) error {
	return Register("metrics", providerFunc, config)
}