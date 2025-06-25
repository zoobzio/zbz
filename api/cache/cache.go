package cache

import (
	"context"
	"errors"
	"time"
)

// Common cache errors
var (
	ErrCacheKeyNotFound = errors.New("cache key not found")
	ErrCacheConnection  = errors.New("cache connection error")
)

// Cache defines the interface for caching implementations
// Supports any caching backend (Redis, Memcached, in-memory, etc.)
type Cache interface {
	// Basic operations
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) bool
	
	// Cache management
	Clear(ctx context.Context) error
	
	// Contract metadata for service identification
	ContractName() string
	ContractDescription() string
}

// CacheContract defines configuration for cache implementations
type CacheContract struct {
	Name        string
	Description string
	
	// Implementation-specific config
	Config map[string]any
}