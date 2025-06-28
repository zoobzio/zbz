package core

import (
	"fmt"
	"time"
)

// Chain management implementation

func (c *coreImpl[T]) RegisterChain(chain ResourceChain) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Validate chain configuration
	if err := c.validateChain(chain); err != nil {
		return fmt.Errorf("invalid chain configuration: %w", err)
	}
	
	c.chains[chain.Name] = chain
	return nil
}

func (c *coreImpl[T]) GetRegisteredChain(name string) (ResourceChain, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	chain, exists := c.chains[name]
	if !exists {
		return ResourceChain{}, fmt.Errorf("chain not found: %s", name)
	}
	
	return chain, nil
}

func (c *coreImpl[T]) validateChain(chain ResourceChain) error {
	if chain.Name == "" {
		return fmt.Errorf("chain name cannot be empty")
	}
	
	if chain.Primary.String() == "" {
		return fmt.Errorf("chain must have a primary resource URI")
	}
	
	// Validate that all URIs use registered providers
	if err := c.validateChainURI(chain.Primary); err != nil {
		return fmt.Errorf("invalid primary URI: %w", err)
	}
	
	for i, fallback := range chain.Fallbacks {
		if err := c.validateChainURI(fallback); err != nil {
			return fmt.Errorf("invalid fallback URI %d: %w", i, err)
		}
	}
	
	return nil
}

func (c *coreImpl[T]) validateChainURI(uri ResourceURI) error {
	// Placeholder validation for testing
	return nil
}

// Chain operations implementation

func (c *coreImpl[T]) GetChain(chainName string, params map[string]any) (ZbzModel[T], error) {
	chain, err := c.GetRegisteredChain(chainName)
	if err != nil {
		return ZbzModel[T]{}, err
	}
	
	return c.resolveChainRead(chain, params)
}

func (c *coreImpl[T]) SetChain(chainName string, data ZbzModel[T], params map[string]any) error {
	chain, err := c.GetRegisteredChain(chainName)
	if err != nil {
		return err
	}
	
	return c.executeChainWrite(chain, data, params)
}

// Chain resolution for read operations
func (c *coreImpl[T]) resolveChainRead(chain ResourceChain, params map[string]any) (ZbzModel[T], error) {
	var chainErrors []ChainError
	
	// Try fallbacks first (typically cache), then primary (typically database)
	allURIs := append(chain.Fallbacks, chain.Primary)
	
	for i, uri := range allURIs {
		// Template the URI with parameters
		templatedURI := uri.WithParams(params)
		
		result, err := c.Get(templatedURI)
		
		if err == nil {
			// Success! Handle cache population if needed
			if i < len(chain.Fallbacks) {
				// Found in a fallback (cache), optionally populate upstream caches
				go c.populateUpstreamCaches(chain, i, templatedURI, result, params)
			}
			return result, nil
		}
		
		// Classify and handle error
		chainErr := c.classifyChainError(templatedURI, err)
		chainErrors = append(chainErrors, chainErr)
		
		// Decide whether to continue the chain
		if !c.shouldContinueChain(chainErr) {
			break
		}
	}
	
	// Return the most relevant error from the chain
	return ZbzModel[T]{}, c.selectBestChainError(chainErrors)
}

// Chain execution for write operations
func (c *coreImpl[T]) executeChainWrite(chain ResourceChain, data ZbzModel[T], params map[string]any) error {
	switch chain.Strategy {
	case WriteThroughBoth:
		return c.writeThroughBoth(chain, data, params)
	case WriteAroundCache:
		return c.writeAroundCache(chain, data, params)
	case ReadThroughCacheFirst:
		// For reads, this strategy writes to primary and invalidates caches
		return c.writeAndInvalidate(chain, data, params)
	default:
		// Default: write to primary only
		templatedURI := chain.Primary.WithParams(params)
		return c.Set(templatedURI, data)
	}
}

// Write strategies

func (c *coreImpl[T]) writeThroughBoth(chain ResourceChain, data ZbzModel[T], params map[string]any) error {
	// Write to primary first
	primaryURI := chain.Primary.WithParams(params)
	if err := c.Set(primaryURI, data); err != nil {
		return err
	}
	
	// Write to caches (best effort)
	for _, fallback := range chain.Fallbacks {
		templatedURI := fallback.WithParams(params)
		go func(uri ResourceURI) {
			c.Set(uri, data) // Ignore errors for cache writes
		}(templatedURI)
	}
	
	return nil
}

func (c *coreImpl[T]) writeAroundCache(chain ResourceChain, data ZbzModel[T], params map[string]any) error {
	// Write only to primary
	primaryURI := chain.Primary.WithParams(params)
	if err := c.Set(primaryURI, data); err != nil {
		return err
	}
	
	// Invalidate caches
	for _, fallback := range chain.Fallbacks {
		templatedURI := fallback.WithParams(params)
		go func(uri ResourceURI) {
			c.Delete(uri) // Ignore errors for cache invalidation
		}(templatedURI)
	}
	
	return nil
}

func (c *coreImpl[T]) writeAndInvalidate(chain ResourceChain, data ZbzModel[T], params map[string]any) error {
	// Write to primary
	primaryURI := chain.Primary.WithParams(params)
	if err := c.Set(primaryURI, data); err != nil {
		return err
	}
	
	// Invalidate caches to force refresh on next read
	for _, fallback := range chain.Fallbacks {
		templatedURI := fallback.WithParams(params)
		go func(uri ResourceURI) {
			c.Delete(uri)
		}(templatedURI)
	}
	
	return nil
}

// Cache population

func (c *coreImpl[T]) populateUpstreamCaches(chain ResourceChain, foundAtIndex int, foundURI ResourceURI, data ZbzModel[T], params map[string]any) {
	// Populate caches that come before the one where we found the data
	for i := 0; i < foundAtIndex; i++ {
		upstreamURI := chain.Fallbacks[i].WithParams(params)
		c.Set(upstreamURI, data) // Ignore errors
	}
}

// Error handling

type ChainError struct {
	URI       ResourceURI
	Err       error
	ErrorType ChainErrorType
	Retryable bool
}

type ChainErrorType int

const (
	NotFoundError ChainErrorType = iota
	ProviderError
	AuthError
	ValidationError
	TimeoutError
	CircuitBreakerOpen
)

func (c *coreImpl[T]) classifyChainError(uri ResourceURI, err error) ChainError {
	chainErr := ChainError{
		URI: uri,
		Err: err,
	}
	
	// Simple error classification - in practice, this would be more sophisticated
	switch {
	case err == ErrNotFound:
		chainErr.ErrorType = NotFoundError
		chainErr.Retryable = false
	case isTimeoutError(err):
		chainErr.ErrorType = TimeoutError
		chainErr.Retryable = true
	case isAuthError(err):
		chainErr.ErrorType = AuthError
		chainErr.Retryable = false
	case isValidationError(err):
		chainErr.ErrorType = ValidationError
		chainErr.Retryable = false
	default:
		chainErr.ErrorType = ProviderError
		chainErr.Retryable = true
	}
	
	return chainErr
}

func (c *coreImpl[T]) shouldContinueChain(chainErr ChainError) bool {
	switch chainErr.ErrorType {
	case NotFoundError, ProviderError, TimeoutError, CircuitBreakerOpen:
		return true // Continue to next provider
	case AuthError, ValidationError:
		return false // Fail fast
	default:
		return true
	}
}

func (c *coreImpl[T]) selectBestChainError(errors []ChainError) error {
	if len(errors) == 0 {
		return ErrNotFound
	}
	
	// Return the last error (from primary) if it's not a NotFound
	lastErr := errors[len(errors)-1]
	if lastErr.ErrorType != NotFoundError {
		return lastErr.Err
	}
	
	// Otherwise, return NotFound
	return ErrNotFound
}

// Helper functions for error classification

func isTimeoutError(err error) bool {
	// In practice, this would check for specific timeout error types
	return false
}

func isAuthError(err error) bool {
	// In practice, this would check for authentication/authorization errors
	return false
}

func isValidationError(err error) bool {
	// In practice, this would check for validation errors
	return false
}

// Chain strategy helpers

func (s ChainStrategy) String() string {
	switch s {
	case ReadThroughCacheFirst:
		return "read_through_cache_first"
	case WriteThroughBoth:
		return "write_through_both"
	case WriteAroundCache:
		return "write_around_cache"
	case ReplicationFailover:
		return "replication_failover"
	case EventualConsistency:
		return "eventual_consistency"
	case SearchWithFallback:
		return "search_with_fallback"
	default:
		return "unknown"
	}
}

// TTL parsing helper
func (c *coreImpl[T]) parseTTL(ttlStr string) (time.Duration, error) {
	if ttlStr == "" {
		return 0, nil
	}
	return time.ParseDuration(ttlStr)
}