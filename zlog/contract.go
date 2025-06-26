package zlog

// No imports needed for the simple contract

// ZlogContract provides type-safe access to logging with native client type T
// Each contract is an INDEPENDENT instance that can register itself as the singleton
type ZlogContract[T any] struct {
	name     string
	provider ZlogProvider // The wrapper that implements our interface
	native   T            // The typed native client
	config   ZlogConfig   // Configuration for this logger
}

// NewContract creates a typed zlog contract with native client
// This is used by provider packages to create contracts
func NewContract[T any](name string, provider ZlogProvider, native T, config ZlogConfig) *ZlogContract[T] {
	return &ZlogContract[T]{
		name:     name,
		provider: provider,
		native:   native,
		config:   config,
	}
}

// Register registers this contract as the global zlog singleton
// If a different contract is already registered, it will be replaced
func (c *ZlogContract[T]) Register() error {
	return configureFromContract(c.name, c.provider, c.config)
}

// Native returns the typed native client without any casting
func (c *ZlogContract[T]) Native() T {
	return c.native
}

// Provider returns the zlog provider wrapper
func (c *ZlogContract[T]) Provider() ZlogProvider {
	return c.provider
}

// Name returns the contract name
func (c *ZlogContract[T]) Name() string {
	return c.name
}

// Config returns the zlog configuration
func (c *ZlogContract[T]) Config() ZlogConfig {
	return c.config
}

// Register sets this contract as the global singleton
func (c *ZlogContract[T]) Register() {
	// Check if already registered
	if zlog != nil && zlog.contract == c.name {
		return  // Already registered
	}
	
	// Register this contract
	Configure(c.provider)
	zlog.contract = c.name
}

// Name returns the contract name
func (c *ZlogContract[T]) Name() string {
	return c.name
}

