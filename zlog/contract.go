package zlog

// No imports needed for the simple contract

// ZlogContract defines the contract that provides both interfaces
type ZlogContract[T any] struct {
	name     string
	service  ZlogService
	provider ZlogProvider
	logger   T
}

// NewContract creates a new contract with service and typed logger
func NewContract[T any](name string, provider ZlogProvider, logger T) *ZlogContract[T] {
	return &ZlogContract[T]{
		name:     name,
		service:  NewZlogService(provider),
		provider: provider,
		logger:   logger,
	}
}

// Zlog returns the service interface for internal ZBZ services
// Self-registers as global singleton if this contract isn't already active
func (c *ZlogContract[T]) Zlog() ZlogService {
	// Check if global singleton came from this contract
	if zlog != nil && zlog.contract == c.name {
		return zlog  // Return existing singleton
	}
	
	// Different contract - self-register
	Configure(c.provider)    // Configure with this contract's provider
	zlog.contract = c.name   // Track which contract created this
	return zlog              // Return updated singleton
}

// Logger returns the typed logger for direct user access
func (c *ZlogContract[T]) Logger() T {
	return c.logger
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

