# BYOC Cache - Architecture V2

## Problem with Current Implementation

The current multi-instance architecture has several flaws:

1. **Type Casting Anti-Pattern**: `native.(redis.Cmdable)` breaks type safety with runtime assertions
2. **Over-Engineering**: Most apps have ONE cache service (Redis cluster, etc.) not multiple cache instances
3. **Unnatural API**: `cache.NewContract[User]("users", provider)` is verbose and complex
4. **Missing Guarantees**: No compile-time guarantees about provider types

## Proposed Architecture V2

### Core Concept: Two-Layer Generic System

**Layer 1: Cache Contract[ProviderType]** - Handles 3rd party cache system type
**Layer 2: Table Contract[DataType]** - Handles individual struct serialization

### Architecture Overview

```go
// Layer 1: Cache provider contract with type guarantee
type CacheContract[T CacheProvider] struct {
    provider T  // T is redis.Cmdable, *memcache.Client, etc.
    config   CacheConfig
}

// Layer 2: Table contract for data type serialization  
type TableContract[T any] struct {
    name     string  // "users", "sessions", etc.
    serializer Serializer[T]
    // References singleton cache internally
}

// Core interfaces
type CacheProvider interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    // ... other operations
}
```

### Key Changes

#### 1. Singleton Pattern (Like zlog)
```go
// Global singleton cache instance
var activeCacheContract CacheContractInterface

// Initialize once per service
func Configure[T CacheProvider](provider T, config CacheConfig) {
    activeCacheContract = NewCacheContract(provider, config)
}

// OR environment-based auto-configuration
func ConfigureFromEnv() error {
    // Auto-detect and configure cache based on environment
}
```

#### 2. Table Abstraction (Natural Cache Pattern)
```go
// Clean API - spawn typed tables from singleton cache
func Table[T any](name string) *TableContract[T] {
    return NewTableContract[T](name, activeCacheContract)
}

// Usage becomes natural
userTable := cache.Table[User]("users")
sessionTable := cache.Table[Session]("sessions")
```

#### 3. Type-Safe Provider Access
```go
// Layer 1: Cache contract with provider type guarantee
func Configure[T CacheProvider](provider T) *CacheContract[T] {
    return &CacheContract[T]{provider: provider}
}

// No type casting needed - T is guaranteed at compile time
func (c *CacheContract[T]) Native() T {
    return c.provider  // Type T guaranteed by contract
}
```

### API Design V2

#### Simple Import & Usage
```go
import "zbz/cache"

// One-time service configuration
cache.Configure(redisClient, cache.Config{DefaultTTL: 1*time.Hour})

// Natural table operations
users := cache.Table[User]("users")
users.Set(ctx, "123", user, 30*time.Minute)
users.Get(ctx, "123", &user)

sessions := cache.Table[Session]("sessions")  
sessions.Set(ctx, "abc123", session)
```

#### Provider Types as Compile-Time Guarantees
```go
// Redis configuration with type guarantee
func ConfigureRedis(client redis.Cmdable) {
    cache.Configure[redis.Cmdable](client, cacheConfig)
}

// Access native Redis with zero type casting
func getRedisStats() {
    nativeRedis := cache.GetNative()  // Returns redis.Cmdable, not interface{}
    info := nativeRedis.Info(ctx, "stats")
    // Full Redis API available with compile-time type safety
}
```

#### Table Operations API
```go
// Clean, type-safe table operations
type TableContract[T any] interface {
    // Basic operations
    Set(ctx context.Context, key string, value T, ttl ...time.Duration) error
    Get(ctx context.Context, key string, dest *T) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    
    // Batch operations
    SetMulti(ctx context.Context, items map[string]T, ttl ...time.Duration) error
    GetMulti(ctx context.Context, keys []string) (map[string]T, error)
    DeleteMulti(ctx context.Context, keys []string) error
    
    // Advanced operations
    Keys(ctx context.Context, pattern string) ([]string, error)
    TTL(ctx context.Context, key string) (time.Duration, error)
    Clear(ctx context.Context) error
}
```

### Implementation Structure

```
cache/
├── api.go           # Public API functions (Table[T], Configure[T], etc.)
├── contract.go      # CacheContract[T] implementation
├── table.go         # TableContract[T] implementation  
├── serializer.go    # JSON/msgpack serialization
├── config.go        # Configuration and defaults
└── singleton.go     # Global cache management

providers/
├── cache-redis/     # redis.Cmdable provider
├── cache-memory/    # In-memory provider
└── cache-memcached/ # *memcache.Client provider
```

### Benefits of V2 Architecture

#### 1. Type Safety Without Casting
```go
// V1 (Bad): Runtime type casting
native := service.NativeClient()
redisClient, ok := native.(redis.Cmdable)  // Runtime risk

// V2 (Good): Compile-time type guarantee  
redisClient := cache.GetNative()  // Type guaranteed by Configure[redis.Cmdable]()
```

#### 2. Natural Cache Patterns
```go
// V1 (Verbose): Multiple cache instances
userCache := cache.NewContract[User]("users", memoryProvider)
sessionCache := cache.NewContract[Session]("sessions", redisProvider)

// V2 (Natural): One cache, multiple tables
cache.Configure(redisCluster, config)
users := cache.Table[User]("users")
sessions := cache.Table[Session]("sessions")
```

#### 3. Simplified API
```go
// V1 (Complex): Multi-step setup
provider, err := cache.NewProvider("redis", config)
contract := cache.NewContract[User]("users", provider).WithTTL(1*time.Hour)
service := contract.Cache()

// V2 (Simple): One-step usage
cache.Configure(redisClient, cache.Config{DefaultTTL: 1*time.Hour})
users := cache.Table[User]("users")
```

#### 4. Familiar zbz Pattern
```go
// Like zlog singleton pattern
zlog.Configure(zlog.NewZap(config))
zlog.Info("message", zlog.String("key", "value"))

// New cache singleton pattern  
cache.Configure(redisClient, cacheConfig)
cache.Table[User]("users").Set(ctx, "123", user)
```

### Migration Strategy

1. **Keep V1 API temporarily** for backward compatibility
2. **Implement V2 alongside V1** in the same package
3. **Deprecate V1** once V2 is proven
4. **Remove V1** in next major version

### Open Questions

1. **Provider Interface**: Should providers implement a common interface or rely on duck typing?
2. **Serialization**: JSON default with msgpack/protobuf options?
3. **Table Naming**: Automatic pluralization? Configurable prefixes?
4. **Configuration**: YAML/ENV support for cache setup?

---

**This V2 architecture solves the type safety issue while providing a much more natural and familiar API pattern that aligns with zbz's singleton-based service architecture.**