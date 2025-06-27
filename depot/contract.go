package depot

// DepotContractTyped provides type-safe access to storage with native client type T
// Each contract is an INDEPENDENT instance that can register itself as the singleton
type DepotContractTyped[T any] struct {
	name     string        // Contract name
	provider DepotProvider // The wrapper that implements our interface
	native   T             // The typed native client
	config   DepotConfig   // Configuration for this storage
}

// DepotContract is the simple contract alias for compatibility
type DepotContract = DepotContractTyped[interface{}]

// NewContract creates a typed depot contract with native client
// This is used by provider packages to create contracts
func NewContract[T any](name string, provider DepotProvider, native T, config DepotConfig) *DepotContractTyped[T] {
	return &DepotContractTyped[T]{
		name:     name,
		provider: provider,
		native:   native,
		config:   config,
	}
}

// NewSimpleContract creates a simple depot contract without generic typing (for compatibility)
func NewSimpleContract(name string, provider DepotProvider) *DepotContract {
	return &DepotContractTyped[interface{}]{
		name:     name,
		provider: provider,
		native:   provider, // Use provider as native client
		config:   DepotConfig{}, // Use default config
	}
}

// Register registers this contract as the global depot singleton
// If a different contract is already registered, it will be replaced
func (c *DepotContractTyped[T]) Register() error {
	return configureFromContract(c.name, c.provider, c.config)
}

// Native returns the typed native client without any casting
func (c *DepotContractTyped[T]) Native() T {
	return c.native
}

// Provider returns the depot provider wrapper
func (c *DepotContractTyped[T]) Provider() DepotProvider {
	return c.provider
}

// Name returns the contract name
func (c *DepotContractTyped[T]) Name() string {
	return c.name
}

// Config returns the depot configuration
func (c *DepotContractTyped[T]) Config() DepotConfig {
	return c.config
}



