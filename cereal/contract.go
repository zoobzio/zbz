package cereal

import (
	"fmt"
)

// CerealContract provides type-safe access to native serialization clients
type CerealContract[T any] struct {
	name     string
	provider CerealProvider
	native   T              // The typed native client
	config   CerealConfig
}

// NewContract creates a new cereal contract with type-safe native client access
func NewContract[T any](name string, provider CerealProvider, native T, config CerealConfig) *CerealContract[T] {
	return &CerealContract[T]{
		name:     name,
		provider: provider,
		native:   native,
		config:   config,
	}
}

// Register sets this contract as the global cereal singleton
func (c *CerealContract[T]) Register() error {
	return configureFromContract(c.name, c.provider, c.config)
}

// Native returns the typed native client without casting
func (c *CerealContract[T]) Native() T {
	return c.native
}

// Name returns the contract name
func (c *CerealContract[T]) Name() string {
	return c.name
}

// Provider returns the cereal provider
func (c *CerealContract[T]) Provider() CerealProvider {
	return c.provider
}

// Config returns the configuration
func (c *CerealContract[T]) Config() CerealConfig {
	return c.config
}

// Marshal delegates to the provider
func (c *CerealContract[T]) Marshal(data any) ([]byte, error) {
	return c.provider.Marshal(data)
}

// Unmarshal delegates to the provider  
func (c *CerealContract[T]) Unmarshal(data []byte, target any) error {
	return c.provider.Unmarshal(data, target)
}

// MarshalScoped delegates to the provider
func (c *CerealContract[T]) MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	return c.provider.MarshalScoped(data, userPermissions)
}

// UnmarshalScoped delegates to the provider
func (c *CerealContract[T]) UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	return c.provider.UnmarshalScoped(data, target, userPermissions, operation)
}

// Close cleans up the contract's provider
func (c *CerealContract[T]) Close() error {
	if c.provider == nil {
		return nil
	}
	return c.provider.Close()
}

// String returns a string representation of the contract
func (c *CerealContract[T]) String() string {
	return fmt.Sprintf("CerealContract[%s](%s)", c.name, c.provider.Format())
}