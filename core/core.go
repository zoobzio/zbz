package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"zbz/catalog"
	"zbz/universal"
	"zbz/zlog"
)

// ResourceURI represents a resource identifier (simplified for standalone testing)
type ResourceURI struct {
	URI string
}

// NewResourceURI creates a new ResourceURI
func NewResourceURI(uri string) ResourceURI {
	return ResourceURI{URI: uri}
}

func (r ResourceURI) String() string {
	return r.URI
}

func (r ResourceURI) Service() string {
	// Simplified - extract service from URI
	return "test"
}

func (r ResourceURI) WithParams(params map[string]any) ResourceURI {
	// Simplified implementation for testing
	return r
}

// OperationURI represents an operation identifier (simplified for standalone testing)
type OperationURI struct {
	URI string
}

// Operation represents a data operation (simplified)
type Operation struct {
	Type   string
	Target string
	Params any
}

// Core represents a ZBZ-native wrapped type that provides universal API access
type Core[T any] interface {
	// Pure business operations
	Get(resource ResourceURI) (ZbzModel[T], error)
	Set(resource ResourceURI, data ZbzModel[T]) error
	Delete(resource ResourceURI) error
	List(pattern ResourceURI) ([]ZbzModel[T], error)
	Exists(resource ResourceURI) (bool, error)
	Count(pattern ResourceURI) (int64, error)
	
	// Complex operations via OperationURI
	Execute(operation OperationURI, params any) (any, error)
	ExecuteMany(operations []Operation) ([]any, error)
	
	// Chain operations
	GetChain(chainName string, params map[string]any) (ZbzModel[T], error)
	SetChain(chainName string, data ZbzModel[T], params map[string]any) error
	
	// Hook management
	OnBeforeCreate(hook BeforeCreateHook[T]) HookID
	OnAfterCreate(hook AfterCreateHook[T]) HookID
	OnBeforeUpdate(hook BeforeUpdateHook[T]) HookID
	OnAfterUpdate(hook AfterUpdateHook[T]) HookID
	OnBeforeDelete(hook BeforeDeleteHook[T]) HookID
	OnAfterDelete(hook AfterDeleteHook[T]) HookID
	
	// Hook removal
	RemoveHook(hookID HookID) error
	
	// Chain management
	RegisterChain(chain ResourceChain) error
	GetRegisteredChain(name string) (ResourceChain, error)
	
	// Model metadata
	Type() reflect.Type
	TypeName() string
}

// Hook function types
type BeforeCreateHook[T any] func(data ZbzModel[T]) error
type AfterCreateHook[T any] func(data ZbzModel[T]) error
type BeforeUpdateHook[T any] func(old, new ZbzModel[T]) error
type AfterUpdateHook[T any] func(old, new ZbzModel[T]) error
type BeforeDeleteHook[T any] func(data ZbzModel[T]) error
type AfterDeleteHook[T any] func(data ZbzModel[T]) error

type HookID string

// ResourceChain defines an ordered sequence of providers for the same logical resource
type ResourceChain struct {
	Name      string        `json:"name"`
	Primary   ResourceURI   `json:"primary"`
	Fallbacks []ResourceURI `json:"fallbacks"`
	Strategy  ChainStrategy `json:"strategy"`
	TTL       string        `json:"ttl"` // Duration string like "15m"
}

type ChainStrategy int

const (
	ReadThroughCacheFirst ChainStrategy = iota
	WriteThroughBoth
	WriteAroundCache
	ReplicationFailover
	EventualConsistency
	SearchWithFallback
)

// LifecycleProvider interface for providers that support lifecycle management
type LifecycleProvider interface {
	OnBeforeCreate(model any) error
	OnBeforeUpdate(model any) error
	OnBeforeDelete(model any) error
	
	SupportsTimestamps() bool
	SupportsSoftDelete() bool
	SupportsVersioning() bool
}

// coreImpl implements the Core interface
type coreImpl[T any] struct {
	typeName string
	
	// Hook storage
	beforeCreateHooks map[HookID]BeforeCreateHook[T]
	afterCreateHooks  map[HookID]AfterCreateHook[T]
	beforeUpdateHooks map[HookID]BeforeUpdateHook[T]
	afterUpdateHooks  map[HookID]AfterUpdateHook[T]
	beforeDeleteHooks map[HookID]BeforeDeleteHook[T]
	afterDeleteHooks  map[HookID]AfterDeleteHook[T]
	
	// Chain registry
	chains map[string]ResourceChain
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// Hook ID counter
	hookCounter int64
}

// NewCore creates a new Core instance for type T
func NewCore[T any]() Core[T] {
	typeName := catalog.GetTypeName[T]()
	
	// Log core creation
	zlog.Info("Creating new Core instance",
		zlog.String("type", typeName),
	)
	
	// Trigger metadata extraction in catalog (lazy)
	catalog.Select[T]()
	
	return &coreImpl[T]{
		typeName:          typeName,
		beforeCreateHooks: make(map[HookID]BeforeCreateHook[T]),
		afterCreateHooks:  make(map[HookID]AfterCreateHook[T]),
		beforeUpdateHooks: make(map[HookID]BeforeUpdateHook[T]),
		afterUpdateHooks:  make(map[HookID]AfterUpdateHook[T]),
		beforeDeleteHooks: make(map[HookID]BeforeDeleteHook[T]),
		afterDeleteHooks:  make(map[HookID]AfterDeleteHook[T]),
		chains:            make(map[string]ResourceChain),
	}
}

// Basic CRUD operations

func (c *coreImpl[T]) Get(resource ResourceURI) (ZbzModel[T], error) {
	// Placeholder implementation for testing
	return ZbzModel[T]{}, ErrNotFound
}

func (c *coreImpl[T]) Set(resource ResourceURI, data ZbzModel[T]) error {
	isCreate := data.IsNew()
	operation := "update"
	if isCreate {
		operation = "create"
	}
	
	zlog.Debug("Core Set operation started",
		zlog.String("type", c.typeName),
		zlog.String("operation", operation),
		zlog.String("resource", resource.String()),
		zlog.String("id", data.ID()),
	)
	
	// Validate data before processing
	if err := data.Validate(); err != nil {
		zlog.Error("Core Set validation failed",
			zlog.String("type", c.typeName),
			zlog.String("resource", resource.String()),
			zlog.String("error", err.Error()),
		)
		return err
	}
	
	var old ZbzModel[T]
	
	// Execute before hooks
	c.mu.RLock()
	hookCount := 0
	if isCreate {
		hookCount = len(c.beforeCreateHooks)
		for _, hook := range c.beforeCreateHooks {
			if err := hook(data); err != nil {
				c.mu.RUnlock()
				zlog.Error("Before create hook failed",
					zlog.String("type", c.typeName),
					zlog.String("resource", resource.String()),
					zlog.String("error", err.Error()),
				)
				return err
			}
		}
	} else {
		hookCount = len(c.beforeUpdateHooks)
		for _, hook := range c.beforeUpdateHooks {
			if err := hook(old, data); err != nil {
				c.mu.RUnlock()
				zlog.Error("Before update hook failed",
					zlog.String("type", c.typeName),
					zlog.String("resource", resource.String()),
					zlog.String("error", err.Error()),
				)
				return err
			}
		}
	}
	c.mu.RUnlock()
	
	zlog.Debug("Before hooks executed",
		zlog.String("type", c.typeName),
		zlog.String("operation", operation),
		zlog.Int("hook_count", hookCount),
	)
	
	// Placeholder - would perform the actual operation
	
	// Emit events and execute after hooks
	if isCreate {
		data.EmitEvent("created", resource.String(), c.typeName, nil)
		
		c.mu.RLock()
		afterHookCount := len(c.afterCreateHooks)
		for _, hook := range c.afterCreateHooks {
			go hook(data) // Execute after hooks asynchronously
		}
		c.mu.RUnlock()
		
		zlog.Info("Core create operation completed",
			zlog.String("type", c.typeName),
			zlog.String("resource", resource.String()),
			zlog.String("id", data.ID()),
			zlog.Int("after_hooks", afterHookCount),
		)
	} else {
		data.EmitEvent("updated", resource.String(), c.typeName, &old)
		
		c.mu.RLock()
		afterHookCount := len(c.afterUpdateHooks)
		for _, hook := range c.afterUpdateHooks {
			go hook(old, data)
		}
		c.mu.RUnlock()
		
		zlog.Info("Core update operation completed",
			zlog.String("type", c.typeName),
			zlog.String("resource", resource.String()),
			zlog.String("id", data.ID()),
			zlog.Int("after_hooks", afterHookCount),
		)
	}
	
	return nil
}

func (c *coreImpl[T]) Delete(resource ResourceURI) error {
	// Placeholder implementation for testing
	return nil
}

func (c *coreImpl[T]) List(pattern ResourceURI) ([]ZbzModel[T], error) {
	// Placeholder implementation for testing
	return []ZbzModel[T]{}, nil
}

func (c *coreImpl[T]) Exists(resource ResourceURI) (bool, error) {
	// Placeholder implementation for testing
	return false, nil
}

func (c *coreImpl[T]) Count(pattern ResourceURI) (int64, error) {
	// Placeholder implementation for testing
	return 0, nil
}

// Complex operations

func (c *coreImpl[T]) Execute(operation OperationURI, params any) (any, error) {
	// Placeholder implementation for testing
	return nil, nil
}

func (c *coreImpl[T]) ExecuteMany(operations []Operation) ([]any, error) {
	// Placeholder implementation for testing
	return []any{}, nil
}

// Type metadata

func (c *coreImpl[T]) Type() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func (c *coreImpl[T]) TypeName() string {
	return c.typeName
}

// CoreService interface implementation (non-generic methods)

func (c *coreImpl[T]) GetByURI(ctx context.Context, uri ResourceURI) (ZbzModelInterface, error) {
	result, err := c.Get(uri)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *coreImpl[T]) SetByURI(ctx context.Context, uri ResourceURI, data ZbzModelInterface) error {
	// Convert ZbzModelInterface back to ZbzModel[T]
	if zbzModel, ok := data.(*ZbzModel[T]); ok {
		return c.Set(uri, *zbzModel)
	}
	return fmt.Errorf("invalid data type for SetByURI")
}

func (c *coreImpl[T]) DeleteByURI(ctx context.Context, uri ResourceURI) error {
	return c.Delete(uri)
}

func (c *coreImpl[T]) ExistsByURI(ctx context.Context, uri ResourceURI) (bool, error) {
	return c.Exists(uri)
}

func (c *coreImpl[T]) CountByURI(ctx context.Context, pattern ResourceURI) (int64, error) {
	return c.Count(pattern)
}

func (c *coreImpl[T]) GetChainByName(ctx context.Context, chainName string, params map[string]any) (ZbzModelInterface, error) {
	result, err := c.GetChain(chainName, params)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *coreImpl[T]) SetChainByName(ctx context.Context, chainName string, data ZbzModelInterface, params map[string]any) error {
	if zbzModel, ok := data.(*ZbzModel[T]); ok {
		return c.SetChain(chainName, *zbzModel, params)
	}
	return fmt.Errorf("invalid data type for SetChainByName")
}

func (c *coreImpl[T]) ConfigureProviders(providers map[string]universal.Provider) error {
	// Implementation will be added when we work on provider configuration
	return fmt.Errorf("provider configuration not yet implemented")
}

func (c *coreImpl[T]) GetProviderHealth() map[string]ProviderHealth {
	// Implementation will be added when we work on provider health
	return make(map[string]ProviderHealth)
}

// Metadata delegation to catalog (NO reflection in core!)
func (c *coreImpl[T]) GetMetadata() catalog.ModelMetadata {
	return catalog.Select[T]()
}

func (c *coreImpl[T]) GetFieldMetadata() []catalog.FieldMetadata {
	return c.GetMetadata().Fields
}

func (c *coreImpl[T]) GetPermissionScopes() []string {
	metadata := c.GetMetadata()
	var scopes []string
	for _, field := range metadata.Fields {
		scopes = append(scopes, field.Scopes...)
	}
	
	// Add model-level scopes if type implements conventions
	for _, function := range metadata.Functions {
		if function.Convention == "ScopeProvider" {
			// Type implements ScopeProvider - could call the method here
			break
		}
	}
	
	return uniqueStrings(scopes)
}

func (c *coreImpl[T]) GetValidationRules() []catalog.ValidationInfo {
	metadata := c.GetMetadata()
	var rules []catalog.ValidationInfo
	for _, field := range metadata.Fields {
		if field.Validation.Required || len(field.Validation.CustomRules) > 0 {
			rules = append(rules, field.Validation)
		}
	}
	return rules
}

func (c *coreImpl[T]) PublishAPIContracts() []APIContract {
	// Generate default CRUD contracts
	typeName := c.GetMetadata().TypeName
	collection := strings.ToLower(typeName + "s")
	
	var contracts []APIContract
	
	// GET /api/{collection}/{id} - get single resource
	contracts = append(contracts, APIContract{
		Endpoint:   fmt.Sprintf("/api/%s/{id}", collection),
		Method:     "GET",
		Operation:  "get",
		Scopes:     []string{fmt.Sprintf("%s:read", strings.ToLower(typeName))},
		OutputType: typeName,
		ResourceURI: func(params map[string]string) ResourceURI {
			return ResourceURI{URI: fmt.Sprintf("db://%s/%s", collection, params["id"])}
		},
	})
	
	// GET /api/{collection} - list resources
	contracts = append(contracts, APIContract{
		Endpoint:   fmt.Sprintf("/api/%s", collection),
		Method:     "GET",
		Operation:  "list",
		Scopes:     []string{fmt.Sprintf("%s:read", strings.ToLower(typeName))},
		OutputType: fmt.Sprintf("[]%s", typeName),
		ResourceURI: func(params map[string]string) ResourceURI {
			return ResourceURI{URI: fmt.Sprintf("db://%s/*", collection)}
		},
	})
	
	// POST /api/{collection} - create resource
	contracts = append(contracts, APIContract{
		Endpoint:   fmt.Sprintf("/api/%s", collection),
		Method:     "POST",
		Operation:  "create",
		Scopes:     []string{fmt.Sprintf("%s:write", strings.ToLower(typeName))},
		InputType:  typeName,
		OutputType: typeName,
		ResourceURI: func(params map[string]string) ResourceURI {
			return ResourceURI{URI: fmt.Sprintf("db://%s", collection)}
		},
	})
	
	// PUT /api/{collection}/{id} - update resource
	contracts = append(contracts, APIContract{
		Endpoint:   fmt.Sprintf("/api/%s/{id}", collection),
		Method:     "PUT",
		Operation:  "update",
		Scopes:     []string{fmt.Sprintf("%s:write", strings.ToLower(typeName))},
		InputType:  typeName,
		OutputType: typeName,
		ResourceURI: func(params map[string]string) ResourceURI {
			return ResourceURI{URI: fmt.Sprintf("db://%s/%s", collection, params["id"])}
		},
	})
	
	// DELETE /api/{collection}/{id} - delete resource
	contracts = append(contracts, APIContract{
		Endpoint:   fmt.Sprintf("/api/%s/{id}", collection),
		Method:     "DELETE",
		Operation:  "delete",
		Scopes:     []string{fmt.Sprintf("%s:delete", strings.ToLower(typeName))},
		ResourceURI: func(params map[string]string) ResourceURI {
			return ResourceURI{URI: fmt.Sprintf("db://%s/%s", collection, params["id"])}
		},
	})
	
	return contracts
}

// Helper function for unique strings
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, str := range slice {
		if !seen[str] && str != "" {
			seen[str] = true
			result = append(result, str)
		}
	}
	
	return result
}

// Common errors
var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidData   = errors.New("invalid data")
	ErrProviderError = errors.New("provider error")
)