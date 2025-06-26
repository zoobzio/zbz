package cache

import (
	"context"
	"fmt"
	"time"
)

// CacheContract provides type-safe access to cache with native client type T
// Each contract is an INDEPENDENT instance that can register itself as the singleton
type CacheContract[T any] struct {
	name     string
	provider CacheProvider  // The wrapper that implements our interface
	native   T              // The typed native client
	config   CacheConfig    // Configuration for this cache
}

// NewContract creates a typed cache contract with native client
// This is used by provider packages to create contracts
func NewContract[T any](name string, provider CacheProvider, native T, config CacheConfig) *CacheContract[T] {
	return &CacheContract[T]{
		name:     name,
		provider: provider,
		native:   native,
		config:   config,
	}
}



// Register registers this contract as the global cache singleton
// If a different contract is already registered, it will be replaced
func (c *CacheContract[T]) Register() error {
	return configureFromContract(c.name, c.provider, c.config)
}

// Native returns the typed native client without any casting
func (c *CacheContract[T]) Native() T {
	return c.native
}

// Provider returns the cache provider wrapper
func (c *CacheContract[T]) Provider() CacheProvider {
	return c.provider
}


// Name returns the contract name
func (c *CacheContract[T]) Name() string {
	return c.name
}

// Config returns the cache configuration
func (c *CacheContract[T]) Config() CacheConfig {
	return c.config
}