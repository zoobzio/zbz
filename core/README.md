# ZBZ Core Package

The Core package provides the foundational data access layer for ZBZ applications. It transforms any user type into a ZBZ-native entity with automatic lifecycle management, resource chaining, and reactive hooks.

## Key Features

- **Universal Model Wrapper**: `ZbzModel[T]` wraps user types with standard ZBZ fields
- **Catalog Integration**: Smart ID extraction using existing cereal catalog metadata  
- **Resource Chaining**: Declarative multi-provider fallback patterns (cache → database)
- **Reactive Hooks**: Event-driven business logic with before/after operation hooks
- **Pure Business Logic**: No context dependencies - rocco handles auth/security
- **Provider Agnostic**: Works with any universal provider (database, cache, search, etc.)

## Quick Start

### Basic Usage

```go
// Define your model
type User struct {
    ID    int    `json:"id" zbz:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Wrap in ZbzModel (adds ID, timestamps, version, metadata)
user := User{ID: 123, Name: "John", Email: "john@example.com"}
model := core.Wrap(user)

fmt.Println(model.ID())       // "123" (extracted from struct)
fmt.Println(model.Version())  // 1
fmt.Println(model.Data())     // Original User struct

// Basic CRUD operations
core.Set(universal.NewResourceURI("db://users/123"), model)
retrieved, _ := core.Get[User](universal.NewResourceURI("db://users/123"))
core.Delete[User](universal.NewResourceURI("db://users/123"))
```

### Resource Chains (Cache → Database)

```go
// Define a resource chain
userChain := core.ResourceChain{
    Name:    "user-profile",
    Primary: universal.NewResourceURI("db://users/{id}"),
    Fallbacks: []universal.ResourceURI{
        universal.NewResourceURI("cache://users/{id}"),  // Try cache first
    },
    Strategy: core.ReadThroughCacheFirst,
    TTL:      "15m",
}

// Register the chain
core.RegisterChain(userChain)

// Use the chain (automatically tries cache → database)
user, err := core.GetChain[User]("user-profile", map[string]any{"id": "123"})
```

### Reactive Hooks

```go
// React to user creation
core.OnCreated[User](func(user core.ZbzModel[User]) error {
    // Send welcome email
    return emailService.SendWelcome(user.Data().Email)
})

// React to user updates
core.OnUpdated[User](func(old, new core.ZbzModel[User]) error {
    // Invalidate cache if email changed
    if old.Data().Email != new.Data().Email {
        cache.Delete("user-by-email:" + old.Data().Email)
    }
    return nil
})

// Complex business rules
userCore, _ := core.GetCore[User]()
userCore.OnBeforeUpdate(func(old, new core.ZbzModel[User]) error {
    // Prevent email changes for admin users
    if contains(old.Data().Roles, "admin") && old.Data().Email != new.Data().Email {
        return errors.New("admin email cannot be changed")
    }
    return nil
})
```

## Architecture

### ZbzModel[T] Structure

```go
type ZbzModel[T any] struct {
    data      T              // Your original struct
    createdAt *time.Time     // Managed by providers
    updatedAt *time.Time     // Auto-updated on changes
    deletedAt *time.Time     // Soft delete support
    version   int64          // Optimistic concurrency
    metadata  map[string]any // Custom metadata
    catalog   *cereal.CatalogEntry // Links to catalog metadata
}
```

### Resource Chain Strategies

- **ReadThroughCacheFirst**: Try cache → database, populate cache on miss
- **WriteThroughBoth**: Write to both cache and database
- **WriteAroundCache**: Write to database, invalidate cache
- **ReplicationFailover**: Try replicas until success
- **EventualConsistency**: Best effort across all providers

### Provider Integration

Cores work with any provider implementing the universal `DataAccess[T]` interface:

```go
type DataAccess[T any] interface {
    Get(ctx context.Context, resource ResourceURI) (T, error)
    Set(ctx context.Context, resource ResourceURI, data T) error
    Delete(ctx context.Context, resource ResourceURI) error
    List(ctx context.Context, pattern ResourceURI) ([]T, error)
    // ... other operations
}
```

## Advanced Usage

### Lifecycle Hooks

```go
userCore, _ := core.GetCore[User]()

// Validation before creation
userCore.OnBeforeCreate(func(user core.ZbzModel[User]) error {
    if user.Data().Email == "" {
        return errors.New("email is required")
    }
    return nil
})

// Audit trail after any change
userCore.OnAnyChange(func(eventType string, old, new core.ZbzModel[User]) error {
    audit.Log(eventType, old, new)
    return nil
})

// Field-specific hooks
userCore.OnFieldChanged(
    func(u core.ZbzModel[User]) any { return u.Data().Email },
    func(old, new core.ZbzModel[User], oldEmail, newEmail any) error {
        // React to email changes specifically
        return updateEmailIndex(oldEmail.(string), newEmail.(string))
    },
)
```

### Complex Operations

```go
// Named queries via OperationURI
findByEmail := universal.QueryURI("db", "find-by-email")
result, err := core.Execute[User](findByEmail, map[string]any{
    "email": "john@example.com",
})

// Batch operations
operations := []universal.Operation{
    {Type: "create", Target: "users", Params: user1},
    {Type: "create", Target: "users", Params: user2},
}
results, err := userCore.ExecuteMany(operations)
```

### Global Configuration

```go
// Set up default chains for common patterns
func init() {
    // User data with caching
    core.RegisterChain(core.ResourceChain{
        Name:    "user-data",
        Primary: universal.NewResourceURI("db://users/{id}"),
        Fallbacks: []universal.ResourceURI{
            universal.NewResourceURI("cache://users/{id}"),
        },
        Strategy: core.ReadThroughCacheFirst,
        TTL:      "15m",
    })
    
    // Session data with write-through
    core.RegisterChain(core.ResourceChain{
        Name:    "session-data", 
        Primary: universal.NewResourceURI("db://sessions/{id}"),
        Fallbacks: []universal.ResourceURI{
            universal.NewResourceURI("cache://sessions/{id}"),
        },
        Strategy: core.WriteThroughBoth,
        TTL:      "30m",
    })
}
```

## Integration with Other ZBZ Components

### With Rocco (Authentication)

```go
// HTTP handler protected by rocco
func getUserProfile(w http.ResponseWriter, r *http.Request) {
    // Rocco middleware already validated auth
    identity := rocco.GetIdentity(r.Context())
    
    // Pure business logic - no context needed
    user, err := core.GetChain[User]("user-profile", map[string]any{
        "id": identity.ID,
    })
    
    if err != nil {
        http.Error(w, "User not found", 404)
        return
    }
    
    json.NewEncoder(w).Encode(user)
}
```

### With Capitan (Events)

```go
// Cores automatically emit capitan events
func init() {
    // Listen for user creation events
    capitan.Subscribe("core.User.created", func(event core.CoreEvent) {
        user := event.Data.(core.ZbzModel[User])
        analytics.Track("user_registered", user.Data())
    })
    
    // Listen for any core events
    capitan.Subscribe("core.*", func(event core.CoreEvent) {
        audit.Log(event.Type, event.Resource, event.CoreType)
    })
}
```

### With Cereal (Serialization & Security)

```go
type User struct {
    ID       int    `json:"id" zbz:"id"`
    Name     string `json:"name" cereal:"encrypt"`
    Email    string `json:"email" cereal:"pii"`
    Password string `json:"-" cereal:"secret"`
}

// ZbzModel automatically uses cereal catalog for:
// - ID field detection via zbz:"id" tags
// - Encryption via cereal:"encrypt" tags  
// - PII handling via cereal:"pii" tags
// - Validation rules from catalog
```

## Performance Considerations

- **ZbzModel Creation**: ~100ns per wrap (uses cached catalog metadata)
- **JSON Serialization**: ~1μs per model (preserves user struct format)
- **Hook Execution**: After-hooks run asynchronously to avoid blocking operations
- **Chain Resolution**: Short-circuits on first success, caches provider lookups

## Error Handling

```go
// Chain-aware error handling
user, err := core.GetChain[User]("user-profile", map[string]any{"id": "123"})
if err != nil {
    switch {
    case errors.Is(err, core.ErrNotFound):
        // Handle not found (tried all providers in chain)
    case errors.Is(err, core.ErrProviderError):
        // Handle provider-specific error
    default:
        // Handle other errors
    }
}
```

## Testing

```go
func TestUserCreation(t *testing.T) {
    // Create test core
    userCore := core.NewCore[User]()
    
    // Register test hooks
    var createdUser core.ZbzModel[User]
    userCore.OnAfterCreate(func(user core.ZbzModel[User]) error {
        createdUser = user
        return nil
    })
    
    // Test business logic without infrastructure setup
    user := core.Wrap(User{ID: 123, Name: "Test User"})
    
    assert.Equal(t, "123", user.ID())
    assert.Equal(t, "Test User", user.Data().Name)
    assert.Equal(t, int64(1), user.Version())
}
```

## Migration from Traditional Patterns

### Before (Manual Cache-Aside)

```go
func GetUser(id string) (*User, error) {
    // Try cache first
    if cached, err := cache.Get("user:" + id); err == nil {
        return cached.(*User), nil
    }
    
    // Fallback to database
    user, err := db.GetUser(id)
    if err != nil {
        return nil, err
    }
    
    // Populate cache
    cache.Set("user:"+id, user, 15*time.Minute)
    return user, nil
}
```

### After (ZBZ Core)

```go
func GetUser(id string) (core.ZbzModel[User], error) {
    return core.GetChain[User]("user-profile", map[string]any{"id": id})
}
```

The complexity is moved to declarative configuration, and the business logic becomes pure and simple.