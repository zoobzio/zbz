# Provider System Analysis: Learning from zlog/flux/hodor

## Current State of zbz Provider Systems

### 1. **zlog**: Singleton with Contract Competition
```go
// Pattern: Global singleton with contract-based provider swapping
var zlog *zZlog  // Private singleton

// Contract can self-register as singleton
func (c *ZlogContract[T]) Zlog() ZlogService {
    if zlog != nil && zlog.contract == c.name {
        return zlog  // Return existing singleton
    }
    Configure(c.provider)    // Replace global singleton
    zlog.contract = c.name   // Track which contract owns it
    return zlog
}

// Simple API functions proxy to singleton
func Info(msg string, fields ...Field) {
    zlog.Info(msg, fields...)
}
```

**zlog Characteristics:**
- ✅ **Singleton Pattern**: One active logger per service
- ✅ **Contract Competition**: Contracts can take over the singleton
- ✅ **Service Layer**: `zZlog` processes fields before passing to provider
- ✅ **Simple API**: `zlog.Info()` without complex setup
- ✅ **Provider Interface**: Clean provider abstraction
- ❌ **Type Safety**: Uses `interface{}` for native access

### 2. **hodor**: Multi-Instance with Service Registry
```go
// Pattern: Multiple independent contracts with service registry
var hodor *zHodor  // Registry service, not data service

// Each contract is independent
func NewContract(name string, provider HodorProvider) *HodorContract {
    return &HodorContract{name: name, provider: provider}
}

// Optional registration for discovery
func (c *HodorContract) Register(alias string) error {
    return hodor.RegisterContract(alias, c.provider)
}
```

**hodor Characteristics:**
- ✅ **Multi-Instance**: Multiple independent storage contracts
- ✅ **Service Registry**: Central discovery service
- ✅ **Direct Operations**: Contracts proxy directly to providers
- ✅ **Optional Registration**: Contracts work standalone or registered
- ❌ **No Service Layer**: No preprocessing/middleware
- ❌ **No Global API**: Must create contracts explicitly

### 3. **flux**: Adapter-Based with Hodor Integration
```go
// Pattern: Pure functions with type-based adapter selection
func Sync[T any](contract *hodor.HodorContract, key string, callback func(old, new T)) {
    parseFunc, err := selectParserForKey[T](key)  // Adapter selection
    // Uses hodor contract for storage, adds reactive capabilities
}

// Type-based adapter selection
func selectParserForKey[T any](key string) (func([]byte) (any, error), error) {
    switch any(*new(T)).(type) {
    case []byte: return parseBytes, nil
    case string: return parseText, nil  
    default: return parseJSON[T], nil  // Struct types
    }
}
```

**flux Characteristics:**
- ✅ **Pure Function API**: No state management
- ✅ **Type-Based Adapters**: Automatic parser selection
- ✅ **Hodor Integration**: Builds on existing storage contracts
- ✅ **Generic Type Safety**: Compile-time type guarantees
- ❌ **No Standalone Service**: Requires hodor contracts
- ❌ **No Provider Management**: Delegates to hodor

## Cache System Requirements Analysis

Based on your optimization feedback, cache needs to combine the best of all three:

### **From zlog**: Singleton Pattern + Service Layer
- ✅ **One cache service per application** (not multiple instances)
- ✅ **Simple API**: `cache.Table[User]("users")` without complex setup
- ✅ **Service layer** for serialization/deserialization preprocessing

### **From hodor**: Table Abstraction  
- ✅ **"Table" concept** (like hodor's keys) for data organization
- ✅ **Provider interface** for backend abstraction
- ✅ **Independent table contracts** within single cache service

### **From flux**: Type Safety + Adapters
- ✅ **Compile-time type guarantees** (no type casting)
- ✅ **Type-based serializer selection** (JSON/msgpack based on type)
- ✅ **Generic type parameters** for full type safety

## Proposed Cache Architecture V3

### **Core Design: Singleton + Table + Type Safety**

```go
// Singleton cache service (like zlog)
var cache *zCache

// Service layer with provider abstraction
type zCache struct {
    provider CacheProvider      // Backend (redis.Cmdable, etc.)
    serializer SerializerManager // Type-based serialization
    config   CacheConfig
}

// Table contract with type safety (like flux generics)
type TableContract[T any] struct {
    name       string
    serializer Serializer[T]  // Type-specific serializer
    // Uses singleton cache internally
}
```

### **API Design: Best of All Worlds**

```go
// Configuration (like zlog singleton setup)
func Configure[P CacheProvider](provider P, config CacheConfig) {
    cache = newZCache(provider, config)
}

// Simple table API (like zlog global functions)
func Table[T any](name string) *TableContract[T] {
    return cache.Table[T](name)  // Delegate to singleton
}

// Type-safe operations (like flux generics)
users := cache.Table[User]("users")
users.Set(ctx, "123", user, 30*time.Minute)
users.Get(ctx, "123", &user)

// Native access with type safety (no casting!)
func GetNative[P CacheProvider]() P {
    return cache.provider.(P)  // Type guaranteed by Configure[P]()
}
```

### **Provider Interface: Enhanced from hodor**

```go
// Simpler than current over-engineered interface
type CacheProvider interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    
    // Batch operations (like Redis pipelining)
    GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)
    SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error
    
    // Management
    Keys(ctx context.Context, pattern string) ([]string, error)
    Clear(ctx context.Context) error
    Close() error
}
```

### **Service Layer: Inspired by zlog preprocessing**

```go
func (z *zCache) Table[T any](name string) *TableContract[T] {
    // Select serializer based on type T (like flux adapters)
    serializer := z.serializer.ForType[T]()
    
    return &TableContract[T]{
        name:       name,
        serializer: serializer,
        cache:      z,  // Reference to singleton
    }
}

// Table operations with preprocessing (like zlog service layer)
func (t *TableContract[T]) Set(ctx context.Context, key string, value T, ttl ...time.Duration) error {
    // 1. Add table prefix to key
    fullKey := t.name + ":" + key
    
    // 2. Serialize value using type-specific serializer
    data, err := t.serializer.Marshal(value)
    if err != nil {
        return err
    }
    
    // 3. Apply TTL (default or provided)
    effectiveTTL := t.getEffectiveTTL(ttl...)
    
    // 4. Delegate to cache provider
    return t.cache.provider.Set(ctx, fullKey, data, effectiveTTL)
}
```

## Key Architectural Decisions

### **1. Do we need a system layer like zCache?**
**YES** - Essential for:
- **Serialization management** (JSON/msgpack selection by type)
- **Table prefix management** (`users:123` vs raw keys)
- **TTL default handling** (table-level defaults)
- **Provider abstraction** (Redis/Memory/Filesystem switching)

### **2. Provider Type Safety Strategy**
```go
// Configure with type parameter guarantees provider type
cache.Configure[redis.Cmdable](redisClient, config)

// Native access is type-safe (no casting!)
func doRedisSpecificOperation() {
    client := cache.GetNative[redis.Cmdable]()  // Guaranteed redis.Cmdable
    client.HSet(ctx, "key", "field", "value")   // Full Redis API
}
```

### **3. Serializer Selection (like flux adapters)**
```go
// Type-based automatic selection
func (s *SerializerManager) ForType[T any]() Serializer[T] {
    switch any(*new(T)).(type) {
    case []byte:   return &ByteSerializer[T]{}
    case string:   return &StringSerializer[T]{}
    default:       return &JSONSerializer[T]{}  // Struct types
    }
}
```

### **4. Table vs Multi-Instance**
**Table Pattern** (like database tables):
- ✅ **Natural**: `users`, `sessions`, `products` tables in one cache
- ✅ **Efficient**: One Redis cluster serves all data types
- ✅ **Simple**: One configuration, multiple use cases

## Implementation Structure

```
cache/
├── api.go           # Public functions: Configure(), Table(), GetNative()
├── service.go       # zCache service layer with provider management
├── table.go         # TableContract[T] implementation
├── serializer.go    # Type-based serializer selection
├── provider.go      # CacheProvider interface
└── config.go        # Configuration management

providers/
├── cache-redis/     # redis.Cmdable implementation
├── cache-memory/    # In-memory implementation  
└── cache-memcached/ # memcache.Client implementation
```

## Benefits of This Hybrid Approach

### **From zlog**: Singleton Simplicity
```go
cache.Configure(redisClient, config)     // One-time setup
cache.Table[User]("users").Set(ctx, "123", user)  // Simple usage
```

### **From hodor**: Table Organization
```go
users := cache.Table[User]("users")          // Logical data separation
sessions := cache.Table[Session]("sessions") // Different serialization
products := cache.Table[Product]("products") // All in same Redis cluster
```

### **From flux**: Type Safety
```go
users := cache.Table[User]("users")    // Compile-time type binding
users.Set(ctx, "123", user)            // Type-safe operations
users.Get(ctx, "123", &user)           // No interface{} casting
```

**This architecture solves all the current issues while learning from the proven patterns in zbz's existing services!** 🚀