package core

import (
	"context"
	"fmt"
	"sync"

	"zbz/catalog"
	"zbz/universal"
)

// CoreService defines the non-generic interface that all Core[T] instances implement
// This enables type-safe cross-service communication without casting
type CoreService interface {
	// Type identification
	TypeName() string
	
	// Non-generic CRUD operations using ResourceURI
	GetByURI(ctx context.Context, uri ResourceURI) (ZbzModelInterface, error)
	SetByURI(ctx context.Context, uri ResourceURI, data ZbzModelInterface) error
	DeleteByURI(ctx context.Context, uri ResourceURI) error
	ExistsByURI(ctx context.Context, uri ResourceURI) (bool, error)
	CountByURI(ctx context.Context, pattern ResourceURI) (int64, error)
	
	// Chain operations (non-generic)
	GetChainByName(ctx context.Context, chainName string, params map[string]any) (ZbzModelInterface, error)
	SetChainByName(ctx context.Context, chainName string, data ZbzModelInterface, params map[string]any) error
	
	// Provider management
	ConfigureProviders(providers map[string]universal.Provider) error
	GetProviderHealth() map[string]ProviderHealth
	
	// Metadata delegation to catalog (NO reflection in core!)
	GetMetadata() catalog.ModelMetadata
	GetFieldMetadata() []catalog.FieldMetadata
	GetPermissionScopes() []string
	GetValidationRules() []catalog.ValidationInfo
	
	// API contract publishing
	PublishAPIContracts() []APIContract
}

// APIContract defines what endpoints a core handles
type APIContract struct {
	Endpoint    string                                      // "/api/users/{id}"
	Method      string                                      // "GET", "POST", etc.
	Operation   string                                      // "get", "create", "reset-password"
	Scopes      []string                                    // ["user:read", "admin:users"]
	InputType   string                                      // "User", "ResetPasswordRequest"
	OutputType  string                                      // "User", "StatusResponse"
	ResourceURI func(params map[string]string) ResourceURI  // Custom URI mapping
}

// ProviderHealth represents the health status of a provider
type ProviderHealth struct {
	Status    string                 `json:"status"`    // "healthy", "degraded", "unhealthy"
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details"`
	Timestamp string                 `json:"timestamp"`
}

// Private singleton service layer
var core *zCore
var once sync.Once

// zCore is the singleton service that manages all cores and API contracts
type zCore struct {
	registry     map[string]CoreService   // Core[T] instances by type name
	apiRegistry  map[string]APIContract   // "GET:/api/users/{id}" -> contract
	contractMap  map[string]string        // contract key -> core type mapping
	chainRegistry map[string]ResourceChain // Global chain definitions
	providers    map[string]universal.Provider // Global provider instances
	mu           sync.RWMutex
}

// Service returns the singleton core service instance
func Service() *zCore {
	once.Do(func() {
		core = &zCore{
			registry:     make(map[string]CoreService),
			apiRegistry:  make(map[string]APIContract),
			contractMap:  make(map[string]string),
			chainRegistry: make(map[string]ResourceChain),
			providers:    make(map[string]universal.Provider),
		}
	})
	return core
}

// Core service methods for managing cores and API contracts

// RegisterCore registers a core instance and its API contracts
func (c *zCore) RegisterCore(typeName string, coreService CoreService) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Register the core instance
	c.registry[typeName] = coreService
	
	// Get API contracts from the core and register them
	contracts := coreService.PublishAPIContracts()
	for _, contract := range contracts {
		contractKey := fmt.Sprintf("%s:%s", contract.Method, contract.Endpoint)
		c.apiRegistry[contractKey] = contract
		c.contractMap[contractKey] = typeName
	}
	
	return nil
}

// GetCoreServiceByTypeName retrieves a core service by type name (for cross-service access)
func (c *zCore) GetCoreServiceByTypeName(typeName string) (CoreService, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	service, exists := c.registry[typeName]
	if !exists {
		return nil, fmt.Errorf("core service not found for type: %s", typeName)
	}
	return service, nil
}

// GetAPIContract looks up an API contract by HTTP method and path
func (c *zCore) GetAPIContract(method, path string) (APIContract, CoreService, error) {
	contractKey := fmt.Sprintf("%s:%s", method, path)
	
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	contract, exists := c.apiRegistry[contractKey]
	if !exists {
		return APIContract{}, nil, fmt.Errorf("no contract found for %s", contractKey)
	}
	
	typeName := c.contractMap[contractKey]
	coreService := c.registry[typeName]
	
	return contract, coreService, nil
}

// ListCoreTypes returns all registered core type names
func (c *zCore) ListCoreTypes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	types := make([]string, 0, len(c.registry))
	for typeName := range c.registry {
		types = append(types, typeName)
	}
	return types
}

// ListAPIContracts returns all registered API contracts
func (c *zCore) ListAPIContracts() []APIContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	contracts := make([]APIContract, 0, len(c.apiRegistry))
	for _, contract := range c.apiRegistry {
		contracts = append(contracts, contract)
	}
	return contracts
}

// Public API functions that delegate to the service

// RegisterCore registers a core instance for type T
func RegisterCore[T any](core Core[T]) error {
	typeName := catalog.GetTypeName[T]()
	
	// Core[T] must also implement CoreService
	coreService, ok := core.(CoreService)
	if !ok {
		return fmt.Errorf("core for type %s does not implement CoreService interface", typeName)
	}
	
	return Service().RegisterCore(typeName, coreService)
}

// GetAPIContract looks up an API contract (for rocco)
func GetAPIContract(method, path string) (APIContract, CoreService, error) {
	return Service().GetAPIContract(method, path)
}

// GetCoreServiceByTypeName gets a core service by string type name (for cross-service access)
func GetCoreServiceByTypeName(typeName string) (CoreService, error) {
	return Service().GetCoreServiceByTypeName(typeName)
}

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

// Legacy core management (now handled by service layer)

// GetCore retrieves the registered core for type T, creating one if it doesn't exist
func GetCore[T any]() (Core[T], error) {
	typeName := catalog.GetTypeName[T]()
	
	service := Service()
	service.mu.RLock()
	if existing, exists := service.registry[typeName]; exists {
		service.mu.RUnlock()
		if core, ok := existing.(Core[T]); ok {
			return core, nil
		}
		return nil, fmt.Errorf("type assertion failed for core: %s", typeName)
	}
	service.mu.RUnlock()
	
	// Create new core if it doesn't exist
	service.mu.Lock()
	defer service.mu.Unlock()
	
	// Double-check after acquiring write lock
	if existing, exists := service.registry[typeName]; exists {
		if core, ok := existing.(Core[T]); ok {
			return core, nil
		}
		return nil, fmt.Errorf("type assertion failed for core: %s", typeName)
	}
	
	// Create and register new core
	newCore := NewCore[T]()
	
	// Register the core with the service
	if err := RegisterCore(newCore); err != nil {
		return nil, fmt.Errorf("failed to register core: %w", err)
	}
	
	return newCore, nil
}

// GetOrCreateCore is an alias for GetCore for clarity
func GetOrCreateCore[T any]() (Core[T], error) {
	return GetCore[T]()
}

// ListCores returns all registered core type names
func ListCores() []string {
	return Service().ListCoreTypes()
}

// Chain management at package level

// RegisterChain registers a resource chain globally
func RegisterChain(chain ResourceChain) error {
	service := Service()
	service.mu.Lock()
	defer service.mu.Unlock()
	
	service.chainRegistry[chain.Name] = chain
	return nil
}

// GetChainDefinition retrieves a chain definition by name
func GetChainDefinition(name string) (ResourceChain, error) {
	service := Service()
	service.mu.RLock()
	defer service.mu.RUnlock()
	
	chain, exists := service.chainRegistry[name]
	if !exists {
		return ResourceChain{}, fmt.Errorf("chain not found: %s", name)
	}
	
	return chain, nil
}

// ListChains returns all registered chain names
func ListChains() []string {
	service := Service()
	service.mu.RLock()
	defer service.mu.RUnlock()
	
	names := make([]string, 0, len(service.chainRegistry))
	for name := range service.chainRegistry {
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
	service := Service()
	service.mu.RLock()
	defer service.mu.RUnlock()
	
	return map[string]any{
		"registered_cores":  len(service.registry),
		"registered_chains": len(service.chainRegistry),
		"core_types":        ListCores(),
		"chain_names":       ListChains(),
	}
}