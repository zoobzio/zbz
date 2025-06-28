# Universal Interface Architecture (GOD Pattern) Implementation Plan

## Overview
Implement standardized interfaces across all ZBZ services to enable universal composability through "interface tunnels" - where every service communicates through three standardized interface types.

## The Triple Interface Pattern

Every ZBZ service gets three standardized interfaces:

### 1. Contract Interface (User-Facing)
What users call - consistent across all implementations
### 2. Provider Interface (Implementation) 
What backends implement - swappable implementations
### 3. Adapter Interface (Integration)
How services connect to each other via capitan

## Phase 1: Define Universal Interfaces

### Data Access Interfaces
```go
// Universal contract - all data services implement this
type DataAccess[T any] interface {
    Get(ctx context.Context, id any) (T, error)
    Set(ctx context.Context, id any, data T) error
    Delete(ctx context.Context, id any) error
    List(ctx context.Context, filter any) ([]T, error)
    Exists(ctx context.Context, id any) (bool, error)
    Count(ctx context.Context, filter any) (int64, error)
}

// Universal provider - all backends implement this
type DataProvider interface {
    Connect(config Config) error
    Execute(operation Operation) (Result, error)
    GetSchema() (Schema, error)
    Close() error
    GetNative() any
    ProviderName() string
}

// Universal adapter - all integrations implement this
type DataAdapter interface {
    OnRead() capitan.InputHookFunc[ReadEvent]
    OnWrite() capitan.InputHookFunc[WriteEvent]
    OnDelete() capitan.InputHookFunc[DeleteEvent]
    Connect() error
    Name() string
}
```

### HTTP Interfaces
```go
type HTTPContract interface {
    RegisterRoute(route RouteContract) error
    RegisterMiddleware(middleware MiddlewareContract) error
    Start(address string) error
    Stop() error
}

type HTTPProvider interface {
    Initialize(config ProviderConfig) error
    AddRoute(method, path string, handler HandlerFunc) error
    AddMiddleware(middleware MiddlewareFunc) error
    Start(address string) error
    Stop() error
    GetNative() any
    ProviderName() string
}

type HTTPAdapter interface {
    OnRouteRegistered() capitan.InputHookFunc[RouteRegisteredEvent]
    OnRequestStarted() capitan.InputHookFunc[RequestStartedEvent]
    OnRequestCompleted() capitan.InputHookFunc[RequestCompletedEvent]
    Connect() error
    Name() string
}
```

### Auth Interfaces
```go
type AuthContract interface {
    Authenticate(ctx context.Context, token string) (User, error)
    Authorize(ctx context.Context, user User, resource string, action string) (bool, error)
    GetUser(ctx context.Context, id string) (User, error)
}

type AuthProvider interface {
    Initialize(config AuthConfig) error
    ValidateToken(token string) (Claims, error)
    GetUserInfo(claims Claims) (User, error)
    ProviderName() string
}

type AuthAdapter interface {
    OnAuthentication() capitan.InputHookFunc[AuthEvent]
    OnAuthorization() capitan.InputHookFunc[AuthzEvent]
    Connect() error
    Name() string
}
```

## Phase 2: Refactor Capitan Architecture

### Remove Reflection, Add Concrete Hooks
```go
// Service layer only deals with bytes
type HookManager struct {
    handlers map[string][]ByteHandler
}

type ByteHandler interface {
    Handle(eventBytes []byte) error
}

// Concrete hook objects handle serialization
type ConcreteInputHook[T any] struct {
    handler func(T) error
}

func (h *ConcreteInputHook[T]) Handle(eventBytes []byte) error {
    var data T
    if err := cereal.Unmarshal(eventBytes, &data); err != nil {
        return err
    }
    return h.handler(data)
}
```

### Standardized Adapter Function Signatures
```go
type InputHookFunc[T any] func(T) error
type OutputHookFunc[T any] func(T) error
type TransformHookFunc[TIn, TOut any] func(TIn) (TOut, error)

type Adapter interface {
    Name() string
    Connect() error
    Disconnect() error
}
```

## Phase 3: Retrofit Existing Services

### Database Service
- Implement `DataAccess[T]` interface
- Implement `DataProvider` interface  
- Create `DatabaseAdapter` for capitan integration
- Ensure existing `database.Table[T]` works with new interfaces

### Cache Service (Pocket)
- Implement `DataAccess[T]` interface
- Implement `DataProvider` interface
- Create `CacheAdapter` for capitan integration
- Align with database interface patterns

### Depot Service
- Implement `DataAccess[T]` interface  
- Implement `DataProvider` interface
- Create `DepotAdapter` for capitan integration
- File-based data access through standard interface

### HTTP Service
- Implement `HTTPContract` interface
- Implement `HTTPProvider` interface
- Create `HTTPAdapter` for capitan integration
- Retrofit provider + plugin architecture

## Phase 4: Adapter Ecosystem

### Standard Adapter Packages
```
adapters/
├── audit/
│   ├── database.go    # DatabaseAdapter
│   ├── http.go        # HTTPAdapter  
│   └── auth.go        # AuthAdapter
├── metrics/
│   ├── database.go
│   ├── http.go
│   └── performance.go
├── docula/
│   ├── http.go        # HTTP -> OpenAPI
│   └── database.go    # Schema -> Docs
└── analytics/
    ├── database.go
    └── user_behavior.go
```

### Universal Adapter Registry
```go
type AdapterRegistry struct {
    adapters map[string]Adapter
}

func (r *AdapterRegistry) Register(adapter Adapter) error
func (r *AdapterRegistry) ConnectAll() error
func (r *AdapterRegistry) Disconnect(name string) error
```

## Phase 5: Interface Discovery System

### Service Registry
```go
type ServiceRegistry struct {
    contracts map[string]any     // User-facing interfaces
    providers map[string]any     // Implementation interfaces  
    adapters  map[string]Adapter // Integration interfaces
}
```

### Auto-Discovery
- Services register their interfaces on startup
- Adapters can discover available services
- Type-safe interface matching
- Runtime composability validation

## Implementation Priority

### High Priority (Complete First)
1. **Refactor Capitan** - Remove reflection, add concrete hooks
2. **Standardize Database** - Implement universal data interfaces
3. **Create Audit Adapter** - Prove the standardized adapter pattern
4. **Retrofit HTTP** - Provider + plugin architecture with capitan

### Medium Priority  
1. **Align Cache/Depot** - Implement universal data interfaces
2. **Create Metrics Adapters** - Expand adapter ecosystem
3. **HTTP->Docula Adapter** - Auto-documentation integration

### Future Priority
1. **Interface Registry** - Service discovery system
2. **Auth Interfaces** - Standardize authentication layer
3. **Search Interfaces** - Add search to universal data access
4. **Third-Party Ecosystem** - Enable external implementations

## Success Metrics

- **Composability**: Any data provider works with any adapter
- **Consistency**: Same interface across database/cache/depot/search
- **Type Safety**: No reflection in hot paths, full compile-time checking
- **Developer Experience**: One-line adapter connections
- **Ecosystem Growth**: Easy for third parties to implement interfaces

## Breaking Changes

This is a major architectural evolution that will require:
- Interface implementations across all services
- Capitan refactoring for performance  
- Adapter pattern standardization
- Some API changes for consistency

The benefits of universal composability justify the breaking changes.