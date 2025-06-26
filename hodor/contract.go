package hodor

import (
	"fmt"
	"time"
)

// HodorContract provides type-safe access to storage with native client type T
// Each contract is an INDEPENDENT instance that can register itself as the singleton
type HodorContract[T any] struct {
	name     string        // Contract name
	provider HodorProvider // The wrapper that implements our interface
	native   T             // The typed native client
	config   HodorConfig   // Configuration for this storage
}

// NewContract creates a typed hodor contract with native client
// This is used by provider packages to create contracts
func NewContract[T any](name string, provider HodorProvider, native T, config HodorConfig) *HodorContract[T] {
	return &HodorContract[T]{
		name:     name,
		provider: provider,
		native:   native,
		config:   config,
	}
}

// Register registers this contract as the global hodor singleton
// If a different contract is already registered, it will be replaced
func (c *HodorContract[T]) Register() error {
	return configureFromContract(c.name, c.provider, c.config)
}

// Native returns the typed native client without any casting
func (c *HodorContract[T]) Native() T {
	return c.native
}

// Provider returns the hodor provider wrapper
func (c *HodorContract[T]) Provider() HodorProvider {
	return c.provider
}

// Name returns the contract name
func (c *HodorContract[T]) Name() string {
	return c.name
}

// Config returns the hodor configuration
func (c *HodorContract[T]) Config() HodorConfig {
	return c.config
}



