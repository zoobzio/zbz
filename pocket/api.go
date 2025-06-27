package cache

import (
	"context"
	"time"
)

// Public API functions that delegate to the singleton service (like zlog pattern)

// Configuration API - structured configuration with optional flux integration

// ConfigureWithProvider creates cache from provider function and config (main API)
// Example: cache.ConfigureWithProvider(redis.NewProvider, config)
func ConfigureWithProvider(providerFunc ProviderFunction, config CacheConfig) error {
	return configureWithProvider(providerFunc, config)
}

// ConfigureFromYAML loads configuration from YAML and uses provider function
// Example: cache.ConfigureFromYAML(redis.NewProvider, "cache.yaml")
func ConfigureFromYAML(providerFunc ProviderFunction, filename string) error {
	config, err := LoadConfigFromYAML(filename)
	if err != nil {
		return err
	}
	return configureWithProvider(providerFunc, config)
}

// ConfigureFromFlux sets up cache with flux hot-reloading (reactive configuration)
// Example: cache.ConfigureFromFlux(redis.NewProvider, depotContract, "cache.yaml")
func ConfigureFromFlux(providerFunc ProviderFunction, depotContract *depot.DepotContract, configKey string) (flux.FluxContract, error) {
	return configureFromFluxWithProvider(providerFunc, depotContract, configKey)
}


// Table creates a typed table contract from the singleton cache service
// This is the main usage pattern: cache.Table[User]("users")
func Table[T any](name string) *TableContract[T] {
	if cache == nil {
		panic("cache not configured - call cache.Configure*() first")
	}
	return cache.Table[T](name)
}

// GetNative returns the original typed provider (type-safe, no casting)
// Example: redisClient := cache.GetNative[redis.Cmdable]()
func GetNative[T any]() T {
	if cache == nil {
		panic("cache not configured - call cache.Configure() first")
	}
	return cache.GetNative[T]()
}

// Provider returns the standardized provider interface
func Provider() CacheProvider {
	if cache == nil {
		return nil
	}
	return cache.Provider()
}

// Config returns the current cache configuration
func Config() CacheConfig {
	if cache == nil {
		return CacheConfig{}
	}
	return cache.Config()
}

// Health check operations

// Ping checks if the cache service is available
func Ping(ctx context.Context) error {
	if cache == nil {
		return ErrCacheNotConfigured
	}
	return cache.Ping(ctx)
}

// Stats returns cache performance metrics
func Stats(ctx context.Context) (CacheStats, error) {
	if cache == nil {
		return CacheStats{}, ErrCacheNotConfigured
	}
	return cache.Stats(ctx)
}

// Close shuts down the cache service
func Close() error {
	if cache == nil {
		return nil
	}
	return cache.Close()
}

// Convenience functions for direct singleton operations (bypass tables)

// Set stores raw bytes directly in the cache with optional TTL
func Set(ctx context.Context, key string, value []byte, ttl ...time.Duration) error {
	if cache == nil {
		return ErrCacheNotConfigured
	}
	
	effectiveTTL := cache.config.DefaultTTL
	if len(ttl) > 0 {
		effectiveTTL = ttl[0]
	}
	
	fullKey := cache.config.KeyPrefix + key
	return cache.provider.Set(ctx, fullKey, value, effectiveTTL)
}

// Get retrieves raw bytes directly from the cache
func Get(ctx context.Context, key string) ([]byte, error) {
	if cache == nil {
		return nil, ErrCacheNotConfigured
	}
	
	fullKey := cache.config.KeyPrefix + key
	return cache.provider.Get(ctx, fullKey)
}

// Delete removes a key directly from the cache
func Delete(ctx context.Context, key string) error {
	if cache == nil {
		return ErrCacheNotConfigured
	}
	
	fullKey := cache.config.KeyPrefix + key
	return cache.provider.Delete(ctx, fullKey)
}

// Exists checks if a key exists directly in the cache
func Exists(ctx context.Context, key string) (bool, error) {
	if cache == nil {
		return false, ErrCacheNotConfigured
	}
	
	fullKey := cache.config.KeyPrefix + key
	return cache.provider.Exists(ctx, fullKey)
}

// Configuration helpers

// IsConfigured returns true if the cache service has been configured
func IsConfigured() bool {
	return cache != nil
}

// DefaultTTL returns the configured default TTL
func DefaultTTL() time.Duration {
	if cache == nil {
		return 0
	}
	return cache.config.DefaultTTL
}

// KeyPrefix returns the configured key prefix
func KeyPrefix() string {
	if cache == nil {
		return ""
	}
	return cache.config.KeyPrefix
}