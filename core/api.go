package core

import (
	"fmt"
	"reflect"
	"sync"
)

// Global registry for Core instances
var (
	registry     = make(map[string]any)
	registryMu   sync.RWMutex
	chainRegistry = make(map[string]ResourceChain)
	chainMu      sync.RWMutex
)

// Package-level API functions for clean usage

// Get retrieves a resource using the registered core for type T
func Get[T any](resource ResourceURI) (ZbzModel[T], error) {
	core, err := GetCore[T]()
	if err != nil {
		return ZbzModel[T]{}, err
	}
	return core.Get(resource)
}

// Set stores a resource using the registered core for type T
func Set[T any](resource ResourceURI, data ZbzModel[T]) error {
	core, err := GetCore[T]()
	if err != nil {
		return err
	}
	return core.Set(resource, data)
}

// Delete removes a resource using the registered core for type T
func Delete[T any](resource ResourceURI) error {
	core, err := GetCore[T]()
	if err != nil {
		return err
	}
	return core.Delete(resource)
}

// List retrieves multiple resources using a pattern
func List[T any](pattern ResourceURI) ([]ZbzModel[T], error) {
	core, err := GetCore[T]()
	if err != nil {
		return nil, err
	}
	return core.List(pattern)
}

// Exists checks if a resource exists
func Exists[T any](resource ResourceURI) (bool, error) {
	core, err := GetCore[T]()
	if err != nil {
		return false, err
	}
	return core.Exists(resource)
}

// Count returns the number of resources matching a pattern
func Count[T any](pattern ResourceURI) (int64, error) {
	core, err := GetCore[T]()
	if err != nil {
		return 0, err
	}
	return core.Count(pattern)
}

// Execute runs a complex operation
func Execute[T any](operation OperationURI, params any) (any, error) {
	core, err := GetCore[T]()
	if err != nil {
		return nil, err
	}
	return core.Execute(operation, params)
}

// Chain operations

// GetChain retrieves a resource using a registered chain
func GetChain[T any](chainName string, params map[string]any) (ZbzModel[T], error) {
	core, err := GetCore[T]()
	if err != nil {
		return ZbzModel[T]{}, err
	}
	return core.GetChain(chainName, params)
}

// SetChain stores a resource using a registered chain
func SetChain[T any](chainName string, data ZbzModel[T], params map[string]any) error {
	core, err := GetCore[T]()
	if err != nil {
		return err
	}
	return core.SetChain(chainName, data, params)
}

// Core management

// RegisterCore registers a core instance for type T
func RegisterCore[T any](core Core[T]) error {
	var zero T
	typeName := reflect.TypeOf(zero).String()
	
	registryMu.Lock()
	defer registryMu.Unlock()
	
	registry[typeName] = core
	return nil
}

// GetCore retrieves the registered core for type T, creating one if it doesn't exist
func GetCore[T any]() (Core[T], error) {
	var zero T
	typeName := reflect.TypeOf(zero).String()
	
	registryMu.RLock()
	if existing, exists := registry[typeName]; exists {
		registryMu.RUnlock()
		if core, ok := existing.(Core[T]); ok {
			return core, nil
		}
		return nil, fmt.Errorf("type assertion failed for core: %s", typeName)
	}
	registryMu.RUnlock()
	
	// Create new core if it doesn't exist
	registryMu.Lock()
	defer registryMu.Unlock()
	
	// Double-check after acquiring write lock
	if existing, exists := registry[typeName]; exists {
		if core, ok := existing.(Core[T]); ok {
			return core, nil
		}
		return nil, fmt.Errorf("type assertion failed for core: %s", typeName)
	}
	
	// Create and register new core
	newCore := NewCore[T]()
	registry[typeName] = newCore
	return newCore, nil
}

// GetOrCreateCore is an alias for GetCore for clarity
func GetOrCreateCore[T any]() (Core[T], error) {
	return GetCore[T]()
}

// ListCores returns all registered core type names
func ListCores() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	
	types := make([]string, 0, len(registry))
	for typeName := range registry {
		types = append(types, typeName)
	}
	return types
}

// Chain management at package level

// RegisterChain registers a resource chain globally
func RegisterChain(chain ResourceChain) error {
	chainMu.Lock()
	defer chainMu.Unlock()
	
	chainRegistry[chain.Name] = chain
	return nil
}

// GetChainDefinition retrieves a chain definition by name
func GetChainDefinition(name string) (ResourceChain, error) {
	chainMu.RLock()
	defer chainMu.RUnlock()
	
	chain, exists := chainRegistry[name]
	if !exists {
		return ResourceChain{}, fmt.Errorf("chain not found: %s", name)
	}
	
	return chain, nil
}

// ListChains returns all registered chain names
func ListChains() []string {
	chainMu.RLock()
	defer chainMu.RUnlock()
	
	names := make([]string, 0, len(chainRegistry))
	for name := range chainRegistry {
		names = append(names, name)
	}
	return names
}

// ApplyChainToCore applies a global chain to a specific core type
func ApplyChainToCore[T any](chainName string) error {
	chain, err := GetChainDefinition(chainName)
	if err != nil {
		return err
	}
	
	core, err := GetCore[T]()
	if err != nil {
		return err
	}
	
	return core.RegisterChain(chain)
}

// Hook management at package level

// OnCreated registers a hook for when any model of type T is created
func OnCreated[T any](hook func(ZbzModel[T]) error) (HookID, error) {
	core, err := GetCore[T]()
	if err != nil {
		return "", err
	}
	return core.OnAfterCreate(hook), nil
}

// OnUpdated registers a hook for when any model of type T is updated
func OnUpdated[T any](hook func(old, new ZbzModel[T]) error) (HookID, error) {
	core, err := GetCore[T]()
	if err != nil {
		return "", err
	}
	return core.OnAfterUpdate(hook), nil
}

// OnDeleted registers a hook for when any model of type T is deleted
func OnDeleted[T any](hook func(ZbzModel[T]) error) (HookID, error) {
	core, err := GetCore[T]()
	if err != nil {
		return "", err
	}
	return core.OnAfterDelete(hook), nil
}

// OnChanged registers a hook for any change to models of type T
func OnChanged[T any](hook func(eventType string, old, new ZbzModel[T]) error) ([]HookID, error) {
	core, err := GetCore[T]()
	if err != nil {
		return nil, err
	}
	
	coreImpl, ok := core.(*coreImpl[T])
	if !ok {
		return nil, fmt.Errorf("core does not support OnAnyChange")
	}
	
	return coreImpl.OnAnyChange(hook), nil
}

// Convenience functions for common patterns

// CreateUser example - convenience function for user creation
func CreateUser(user any) error {
	// This would be implemented for specific types
	return fmt.Errorf("not implemented - use type-specific creation")
}

// Configuration helpers

// ConfigureDefaults sets up default chains and hooks for common patterns
func ConfigureDefaults() error {
	// User data chain with cache-first strategy
	userChain := ResourceChain{
		Name:    "user-data",
		Primary: NewResourceURI("db://users/{id}"),
		Fallbacks: []ResourceURI{
			NewResourceURI("cache://users/{id}"),
		},
		Strategy: ReadThroughCacheFirst,
		TTL:      "15m",
	}
	
	if err := RegisterChain(userChain); err != nil {
		return fmt.Errorf("failed to register user chain: %w", err)
	}
	
	// Session data chain with write-through strategy
	sessionChain := ResourceChain{
		Name:    "session-data",
		Primary: NewResourceURI("db://sessions/{id}"),
		Fallbacks: []ResourceURI{
			NewResourceURI("cache://sessions/{id}"),
		},
		Strategy: WriteThroughBoth,
		TTL:      "30m",
	}
	
	if err := RegisterChain(sessionChain); err != nil {
		return fmt.Errorf("failed to register session chain: %w", err)
	}
	
	return nil
}

// Health check function
func HealthCheck() map[string]any {
	registryMu.RLock()
	defer registryMu.RUnlock()
	
	return map[string]any{
		"registered_cores":  len(registry),
		"registered_chains": len(chainRegistry),
		"core_types":        ListCores(),
		"chain_names":       ListChains(),
	}
}