package pipz

import (
	"reflect"
	"sync"
)

// Processor is the universal function signature for processing pipeline stages
type Processor[Input, Output any] func(Input) Output

// ServiceContract provides type-safe processing using typed keys (no magic strings)
type ServiceContract[KeyType comparable, Input, Output any] struct {
	processors map[KeyType]Processor[Input, Output]
	mu         sync.RWMutex
}

// Global contract registry using type signatures as keys
var contractRegistry = struct {
	contracts map[reflect.Type]any
	mu        sync.RWMutex
}{
	contracts: make(map[reflect.Type]any),
}

// GetContract returns a type-safe contract for specific Key/Input/Output combination
// Type signature becomes the registry key - 100% type safe, zero magic strings
func GetContract[KeyType comparable, Input, Output any]() *ServiceContract[KeyType, Input, Output] {
	// Use type signature as contract key
	signature := reflect.TypeOf((*ServiceContract[KeyType, Input, Output])(nil))
	
	contractRegistry.mu.RLock()
	if existing, exists := contractRegistry.contracts[signature]; exists {
		contractRegistry.mu.RUnlock()
		return existing.(*ServiceContract[KeyType, Input, Output])
	}
	contractRegistry.mu.RUnlock()
	
	// Create new contract for this exact type combination
	contract := &ServiceContract[KeyType, Input, Output]{
		processors: make(map[KeyType]Processor[Input, Output]),
	}
	
	contractRegistry.mu.Lock()
	contractRegistry.contracts[signature] = contract
	contractRegistry.mu.Unlock()
	
	return contract
}

// Type-safe functions using typed keys (no magic strings)

// Register adds a processor with type-safe key
func (c *ServiceContract[KeyType, Input, Output]) Register(key KeyType, processor Processor[Input, Output]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.processors[key] = processor
}

// Unregister removes a processor by typed key
func (c *ServiceContract[KeyType, Input, Output]) Unregister(key KeyType) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.processors, key)
}

// Process runs input through specific processor with 100% type safety
func (c *ServiceContract[KeyType, Input, Output]) Process(key KeyType, input Input) (Output, bool) {
	c.mu.RLock()
	processor, exists := c.processors[key]
	c.mu.RUnlock()
	
	if !exists {
		var zero Output
		return zero, false
	}
	
	return processor(input), true
}

// HasProcessor checks if processor exists for key
func (c *ServiceContract[KeyType, Input, Output]) HasProcessor(key KeyType) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	_, exists := c.processors[key]
	return exists
}

// ListKeys returns all registered processor keys
func (c *ServiceContract[KeyType, Input, Output]) ListKeys() []KeyType {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	keys := make([]KeyType, 0, len(c.processors))
	for key := range c.processors {
		keys = append(keys, key)
	}
	
	return keys
}

// That's it! Pure type-safe contract system with zero magic strings or non-generic interfaces