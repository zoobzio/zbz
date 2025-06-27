package cache

import (
	"context"
	"fmt"
	"time"
)

// CacheProvider defines the simplified interface for cache backends (V3 architecture)
type CacheProvider interface {
	// Basic operations
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	
	// Batch operations for performance
	GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)
	SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error
	DeleteMulti(ctx context.Context, keys []string) error
	
	// Advanced operations
	Keys(ctx context.Context, pattern string) ([]string, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	
	// Management operations
	Clear(ctx context.Context) error
	Stats(ctx context.Context) (CacheStats, error)
	Ping(ctx context.Context) error
	Close() error
	
	// Provider metadata
	GetProvider() string
}

// Common cache errors
var (
	ErrCacheKeyNotFound   = &CacheError{Code: "KEY_NOT_FOUND", Message: "cache key not found"}
	ErrCacheNotConfigured = &CacheError{Code: "NOT_CONFIGURED", Message: "cache not configured"}
)

// CacheError represents cache-specific errors
type CacheError struct {
	Code    string
	Message string
	Cause   error
}

func (e *CacheError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *CacheError) Unwrap() error {
	return e.Cause
}

// CacheItem represents an item to be stored in cache
type CacheItem struct {
	Key   string
	Value []byte
	TTL   time.Duration
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	Provider     string    `json:"provider"`
	Hits         int64     `json:"hits"`
	Misses       int64     `json:"misses"`
	Keys         int64     `json:"keys"`
	Memory       int64     `json:"memory_bytes"`
	Connections  int       `json:"connections"`
	Uptime       time.Duration `json:"uptime"`
	LastAccess   time.Time `json:"last_access"`
}

// ProviderFactory creates cache provider instances
type ProviderFactory func(config map[string]interface{}) (CacheProvider, error)

// Global provider registry
var providerRegistry = make(map[string]ProviderFactory)

// RegisterProvider registers a cache provider factory
func RegisterProvider(name string, factory ProviderFactory) {
	providerRegistry[name] = factory
}

// NewProvider creates a provider instance by name
func NewProvider(name string, config map[string]interface{}) (CacheProvider, error) {
	factory, exists := providerRegistry[name]
	if !exists {
		return nil, fmt.Errorf("unknown cache provider: %s", name)
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

// Config helpers for providers
func getConfigString(config map[string]interface{}, key, defaultValue string) string {
	if val, exists := config[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getConfigInt(config map[string]interface{}, key string, defaultValue int) int {
	if val, exists := config[key]; exists {
		if num, ok := val.(int); ok {
			return num
		}
		if num, ok := val.(float64); ok {
			return int(num)
		}
	}
	return defaultValue
}

func getConfigInt64(config map[string]interface{}, key string, defaultValue int64) int64 {
	if val, exists := config[key]; exists {
		if num, ok := val.(int64); ok {
			return num
		}
		if num, ok := val.(int); ok {
			return int64(num)
		}
		if num, ok := val.(float64); ok {
			return int64(num)
		}
	}
	return defaultValue
}

func getConfigBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, exists := config[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

func getConfigDuration(config map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	if val, exists := config[key]; exists {
		if str, ok := val.(string); ok {
			if duration, err := time.ParseDuration(str); err == nil {
				return duration
			}
		}
		if num, ok := val.(int); ok {
			return time.Duration(num) * time.Second
		}
		if num, ok := val.(float64); ok {
			return time.Duration(num) * time.Second
		}
	}
	return defaultValue
}

func getConfigStringSlice(config map[string]interface{}, key string) []string {
	if val, exists := config[key]; exists {
		if slice, ok := val.([]string); ok {
			return slice
		}
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, v := range slice {
				if str, ok := v.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}
	return []string{}
}

func getConfigMap(config map[string]interface{}, key string) map[string]interface{} {
	if val, exists := config[key]; exists {
		if m, ok := val.(map[string]interface{}); ok {
			return m
		}
	}
	return make(map[string]interface{})
}