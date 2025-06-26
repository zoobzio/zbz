# BYOC Cache - Implementation V3

## Architecture: Hybrid Provider System

Based on analysis of zlog/flux/hodor patterns, cache combines the best of all three:

- **Singleton Pattern** (from zlog): One cache service per application
- **Table Abstraction** (from hodor): Logical data organization within single cache
- **Type Safety** (from flux): Compile-time guarantees with generics
- **Service Layer** (new): Orchestrates serialization, prefixing, and provider management

## Core Components

### 1. **zCache Service Layer**
```go
// Singleton service instance (like zlog)
var cache *zCache

// Service orchestrates provider + serialization + table management
type zCache struct {
    provider   CacheProvider      // Backend: redis.Cmdable, *memcache.Client, etc.
    serializer SerializerManager  // Type-based serializer selection
    config     CacheConfig        // Default TTL, key prefixes, etc.
}
```

### 2. **Provider Interface** (Simplified from V1)
```go
type CacheProvider interface {
    // Basic operations
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    
    // Batch operations (for performance)
    GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)
    SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error
    DeleteMulti(ctx context.Context, keys []string) error
    
    // Advanced operations
    Keys(ctx context.Context, pattern string) ([]string, error)
    Clear(ctx context.Context) error
    Close() error
}
```

### 3. **Table Contracts** (Type-Safe Data Organization)
```go
// Each table handles one data type with automatic serialization
type TableContract[T any] struct {
    name       string           // "users", "sessions", "products"
    serializer Serializer[T]    // Type-specific JSON/msgpack/bytes
    cache      *zCache          // Reference to singleton service
}

// Type-safe operations
type TableOperations[T any] interface {
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

## API Design

### **Configuration** (zlog-style singleton setup)
```go
// Type-safe provider configuration
func Configure[P CacheProvider](provider P, config CacheConfig) {
    cache = newZCache(provider, config)
}

// Environment-based auto-configuration
func ConfigureFromEnv() error {
    // Auto-detect Redis/Memory based on env vars
}

// Configuration struct
type CacheConfig struct {
    DefaultTTL    time.Duration          // 1*time.Hour
    KeyPrefix     string                 // "myapp:"
    Serialization SerializationFormat    // JSON, MessagePack, etc.
}
```

### **Table Operations** (Simple, type-safe API)
```go
// Spawn typed table from singleton cache
func Table[T any](name string) *TableContract[T] {
    return cache.Table[T](name)
}

// Usage examples
users := cache.Table[User]("users")
sessions := cache.Table[Session]("sessions")
products := cache.Table[Product]("products")

// All operations are type-safe
user := User{ID: 123, Name: "Alice", Email: "alice@example.com"}
users.Set(ctx, "123", user, 30*time.Minute)
users.Get(ctx, "123", &user)
```

### **Native Provider Access** (Type-safe, no casting)
```go
// Type guaranteed by Configure[P]() call
func GetNative[P CacheProvider]() P {
    return cache.provider.(P)  // P is guaranteed at compile time
}

// Usage with full type safety
func doRedisSpecificOperation() {
    // No type casting needed - compile-time guarantee
    redisClient := cache.GetNative[redis.Cmdable]()
    
    // Full Redis API available
    redisClient.HSet(ctx, "user:123:profile", map[string]interface{}{
        "login_count": 42,
        "last_ip":     "192.168.1.1",
    })
    
    redisClient.SAdd(ctx, "active_users", "123", "456", "789")
    redisClient.Expire(ctx, "user:123", 30*time.Minute)
}
```

## Service Layer Responsibilities

### **1. Serialization Management** (flux-style type adapters)
```go
// Automatic serializer selection based on type
func (s *SerializerManager) ForType[T any]() Serializer[T] {
    switch any(*new(T)).(type) {
    case []byte:
        return &ByteSerializer[T]{}      // Pass-through for raw data
    case string:
        return &StringSerializer[T]{}    // UTF-8 string handling
    default:
        return &JSONSerializer[T]{}      // Struct types use JSON
    }
}

// Advanced: Custom serializer registration
func RegisterSerializer[T any](serializer Serializer[T]) {
    // Allow custom serializers for specific types
}
```

### **2. Key Management** (Automatic prefixing)
```go
// Table operations automatically prefix keys
func (t *TableContract[T]) Set(ctx context.Context, key string, value T, ttl ...time.Duration) error {
    // 1. Build full key: "myapp:users:123"
    fullKey := t.cache.config.KeyPrefix + t.name + ":" + key
    
    // 2. Serialize value using type-specific serializer
    data, err := t.serializer.Marshal(value)
    if err != nil {
        return fmt.Errorf("serialization failed: %w", err)
    }
    
    // 3. Apply TTL (table default or provided)
    effectiveTTL := t.getEffectiveTTL(ttl...)
    
    // 4. Delegate to provider
    return t.cache.provider.Set(ctx, fullKey, data, effectiveTTL)
}
```

### **3. Provider Abstraction** (Hot-swappable backends)
```go
// Switch providers at runtime (for testing, failover, etc.)
func SwapProvider[P CacheProvider](newProvider P) error {
    if cache == nil {
        return fmt.Errorf("cache not configured")
    }
    
    // Close old provider
    if err := cache.provider.Close(); err != nil {
        return fmt.Errorf("failed to close old provider: %w", err)
    }
    
    // Swap to new provider
    cache.provider = newProvider
    return nil
}
```

## Implementation Structure

```
cache/
├── api.go               # Public API: Configure(), Table(), GetNative()
├── service.go           # zCache service layer implementation
├── table.go             # TableContract[T] operations
├── serializer.go        # Type-based serializer selection
├── config.go            # Configuration management
├── provider.go          # CacheProvider interface definition
└── IMPLEMENTATION_V3.md # This document

providers/
├── cache-redis/
│   ├── redis.go         # redis.Cmdable implementation
│   ├── go.mod           # Module with Redis dependency
│   └── README.md        # Redis-specific documentation
├── cache-memory/
│   ├── memory.go        # In-memory implementation
│   ├── go.mod           # Module without external dependencies
│   └── README.md        # Memory cache documentation
└── cache-memcached/
    ├── memcached.go     # *memcache.Client implementation
    ├── go.mod           # Module with memcache dependency
    └── README.md        # Memcached documentation
```

## Usage Examples

### **Basic Setup & Operations**
```go
import (
    "zbz/cache"
    _ "zbz/providers/cache-redis"  // Auto-registers redis provider
)

func main() {
    // Configure cache with Redis
    redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    cache.Configure(redisClient, cache.Config{
        DefaultTTL: 1 * time.Hour,
        KeyPrefix:  "myapp:",
    })
    
    // Create typed tables
    users := cache.Table[User]("users")
    sessions := cache.Table[Session]("sessions")
    
    // Type-safe operations
    user := User{ID: 123, Name: "Alice", Email: "alice@example.com"}
    users.Set(ctx, "123", user, 30*time.Minute)
    
    var retrievedUser User
    users.Get(ctx, "123", &retrievedUser)
}
```

### **Advanced Operations**
```go
// Batch operations
userMap := map[string]User{
    "user1": {ID: 1, Name: "Alice"},
    "user2": {ID: 2, Name: "Bob"},
    "user3": {ID: 3, Name: "Carol"},
}
users.SetMulti(ctx, userMap, 2*time.Hour)

retrievedUsers, err := users.GetMulti(ctx, []string{"user1", "user2", "user3"})

// Native Redis operations (type-safe)
redisClient := cache.GetNative[redis.Cmdable]()
redisClient.SAdd(ctx, "active_users", "123", "456", "789")

// Table management
userKeys, err := users.Keys(ctx, "*")  // Find all user keys
users.Clear(ctx)                      // Clear all users
```

### **Testing & Development**
```go
// Easy provider swapping for tests
func TestUserOperations(t *testing.T) {
    // Use memory provider for tests
    memProvider := memory.NewProvider()
    cache.Configure(memProvider, cache.Config{})
    
    users := cache.Table[User]("users")
    // ... test operations
}

// Environment-based configuration
func setupCacheFromEnv() {
    switch os.Getenv("CACHE_PROVIDER") {
    case "redis":
        redisClient := redis.NewClient(&redis.Options{
            Addr: os.Getenv("REDIS_URL"),
        })
        cache.Configure(redisClient, loadConfigFromEnv())
    case "memory":
        cache.Configure(memory.NewProvider(), cache.Config{})
    }
}
```

## Key Benefits

### **1. Familiar zbz Pattern**
```go
// Like zlog singleton pattern users know
zlog.Configure(zapProvider)
zlog.Info("message")

// Cache follows same pattern
cache.Configure(redisProvider, config)
cache.Table[User]("users").Set(ctx, "123", user)
```

### **2. Type Safety Without Casting**
```go
// V1 (bad): Runtime type casting
native := service.NativeClient()
redisClient, ok := native.(redis.Cmdable)  // Can fail at runtime

// V3 (good): Compile-time type guarantee
redisClient := cache.GetNative[redis.Cmdable]()  // Type guaranteed
```

### **3. Natural Data Organization**
```go
// Like database tables - logical separation in one service
users := cache.Table[User]("users")           // User data management
sessions := cache.Table[Session]("sessions")   // Session management  
products := cache.Table[Product]("products")   // Product catalog
// All using same Redis cluster with automatic key prefixing
```

### **4. Provider Flexibility**
```go
// Development: Fast in-memory cache
cache.Configure(memory.NewProvider(), config)

// Staging: Single Redis instance
cache.Configure(redis.NewClient(opts), config)

// Production: Redis cluster
cache.Configure(redis.NewClusterClient(clusterOpts), config)

// Same application code works across all environments
```

## Implementation Priority

### **Phase 1: Core Implementation**
1. `service.go` - zCache singleton service layer
2. `table.go` - TableContract[T] operations
3. `serializer.go` - JSON serializer for structs
4. `api.go` - Public API functions

### **Phase 2: Provider Implementation** 
1. `providers/cache-memory/` - In-memory provider
2. `providers/cache-redis/` - Redis provider
3. Provider auto-registration system

### **Phase 3: Advanced Features**
1. Batch operations optimization
2. Custom serializer registration
3. Configuration management
4. Comprehensive testing

**This V3 architecture provides the perfect balance of simplicity (zlog), flexibility (hodor), and type safety (flux) while solving all the issues identified in the current implementation.**