package cache

import (
	"errors"
)

// Common cache errors
var (
	ErrCacheKeyNotFound = errors.New("cache key not found")
	ErrCacheConnection  = errors.New("cache connection error")
)

// Re-export key types for convenience
type (
	Provider = CacheProvider
	Contract[T any] = CacheContract[T]
	Service[T any] = CacheService[T]
)

// Convenience functions

// New creates a new cache contract (alias for NewContract)
func New[T any](name string, provider CacheProvider) *CacheContract[T] {
	return NewContract[T](name, provider)
}

// Providers returns a list of all registered provider names
func Providers() []string {
	return ListProviders()
}