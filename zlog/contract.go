package zlog

// Provider defines the interface that providers implement
type Provider interface {
	// Core logging methods
	Info(msg string, fields []Field)
	Error(msg string, fields []Field)
	Debug(msg string, fields []Field)
	Warn(msg string, fields []Field)
	Fatal(msg string, fields []Field)
	Close() error
}

// Processor is a function that transforms a field into one or more fields
type Processor func(Field) []Field

// Contract provides type-safe access to logging with native client type T
type Contract[T any] struct {
	config   Config   // Configuration for this logger
	logger   T        // The typed native logger (e.g. zap.Logger, logrus.Entry)
	provider Provider // The wrapper that implements our interface
}

// NewContract creates a typed zlog contract with native client and auto-registers
// This is used by provider packages to create contracts
func NewContract[T any](config Config, logger T, provider Provider) *Contract[T] {
	contract := &Contract[T]{
		config:   config,
		logger:   logger,
		provider: provider,
	}

	zlog.Register(config, provider)

	return contract
}

// Config returns the zlog configuration
func (c *Contract[T]) Config() Config {
	return c.config
}

// Logger returns the typed native logger (semantic naming)
func (c *Contract[T]) Logger() T {
	return c.logger
}

// Provider returns the zlog provider wrapper
func (c *Contract[T]) Provider() Provider {
	return c.provider
}
