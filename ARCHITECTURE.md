# ZBZ Framework Architecture

## Provider Philosophy & Universal Contract Pattern

The ZBZ framework implements a revolutionary **Universal Contract Pattern** that enables type-safe, provider-agnostic service architecture with powerful generics. This document outlines the key architectural principles and patterns that make ZBZ services flexible, maintainable, and developer-friendly.

## Core Architecture Principles

### 1. **Bring-Your-Own-X (BYOX) Philosophy**
Every service follows the principle that users should be able to "bring their own" implementation:
- **BYOC**: Bring Your Own Cache (Redis, Memory, Filesystem)
- **BYOS**: Bring Your Own Storage (S3, GCS, Filesystem) 
- **BYOL**: Bring Your Own Logger (Zap, Zerolog, Logrus)

### 2. **Convention Over Configuration**
- Structured configuration with sensible defaults
- YAML/JSON struct tags for easy scanning
- Provider-agnostic configuration fields
- Automatic serializer/adapter selection

### 3. **Type Safety Without Compromise**
- Generic contracts preserve native client types
- Zero casting required for provider-specific operations
- Compile-time type guarantees throughout the stack

### 4. **Independence with Collaboration**
- Each provider is a separate Go module
- No cross-dependencies between providers
- Unified interfaces enable interchangeability

## Universal Contract Pattern

### **Pattern Overview**

Every ZBZ service follows the same architectural pattern:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Provider      │───▶│   Contract[T]    │───▶│   Singleton     │
│   Package       │    │  (Type-Safe)     │    │   Service       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                        ┌──────────────────┐
                        │  Package-Level   │
                        │   Functions      │
                        └──────────────────┘
```

### **Components**

#### **1. Self-Registering Provider Packages**

Each provider is an independent package that creates both a native client and a wrapper:

```go
// providers/cache-redis/redis.go
func NewRedisCache(config cache.CacheConfig) (*cache.CacheContract[redis.Cmdable], error) {
    // Create native Redis client
    client := redis.NewClient(opts)
    
    // Create provider wrapper that implements common interface
    provider := &redisProvider{client: client}
    
    // Return typed contract with both
    return cache.NewContract("redis", provider, client, config), nil
}
```

**Key Benefits:**
- **Unique constructors** prevent naming conflicts
- **Type-safe native access** without casting
- **Self-contained** - no external dependencies
- **Provider-specific optimizations** available through native client

#### **2. Generic Contracts with Native Type Safety**

Contracts are generic on the native client type, providing type-safe access:

```go
type CacheContract[T any] struct {
    name     string
    provider CacheProvider  // Common interface wrapper
    native   T              // Typed native client (redis.Cmdable, *s3.S3, etc.)
    config   CacheConfig    // Provider-agnostic configuration
}

// Type-safe native access - no casting required!
func (c *CacheContract[T]) Native() T {
    return c.native
}
```

**Usage:**
```go
// Create contract
contract := cacheredis.NewRedisCache(config)

// Type-safe native access
redisClient := contract.Native()  // redis.Cmdable - no casting!
result := redisClient.HGetAll(ctx, "myhash")  // Full Redis API available

// Common interface access
provider := contract.Provider()  // CacheProvider interface
data, err := provider.Get(ctx, "key")  // Standardized operations
```

#### **3. Singleton Service Registration**

Contracts can register themselves as the global singleton for convenience:

```go
// Register as singleton
err := contract.Register()

// Now package-level functions work
cache.Set(ctx, "key", value)  // Uses the registered singleton
cache.Get(ctx, "key")         // Uses the registered singleton
```

**Singleton Replacement:**
- If different contract registers, old singleton is replaced
- Old provider is properly closed
- All package functions switch to new provider seamlessly

#### **4. Package-Level Functions with Generics**

Package-level functions provide convenience while maintaining type safety:

```go
// Package functions delegate to singleton
func Set(ctx context.Context, key string, value []byte) error {
    if cache == nil {
        panic("cache not configured - register a contract first")
    }
    return cache.provider.Set(ctx, key, value, cache.config.DefaultTTL)
}

// Advanced functions leverage generics for type safety
func Table[T any](name string) *TableContract[T] {
    return cache.Table[T](name)  // Singleton creates typed table
}
```

## Universal Configuration

### **Provider-Agnostic Configuration**

All providers in a service category use the same configuration structure:

```go
// Universal cache configuration works across all providers
type CacheConfig struct {
    // Service settings
    DefaultTTL    time.Duration `yaml:"default_ttl" json:"default_ttl"`
    KeyPrefix     string        `yaml:"key_prefix" json:"key_prefix"`
    
    // Network providers (Redis, Memcached)
    URL      string `yaml:"url,omitempty" json:"url,omitempty"`
    Host     string `yaml:"host,omitempty" json:"host,omitempty"`
    Port     int    `yaml:"port,omitempty" json:"port,omitempty"`
    
    // Storage providers (Memory, Filesystem)
    MaxSize  int64  `yaml:"max_size,omitempty" json:"max_size,omitempty"`
    BaseDir  string `yaml:"base_dir,omitempty" json:"base_dir,omitempty"`
    
    // Performance (all providers)
    PoolSize    int           `yaml:"pool_size,omitempty" json:"pool_size,omitempty"`
    MaxRetries  int           `yaml:"max_retries,omitempty" json:"max_retries,omitempty"`
    Timeout     time.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}
```

### **Configuration Benefits**

1. **Provider Switching**: Change from Redis to Memory without config changes
2. **YAML/JSON Ready**: Struct tags enable easy scanning from files
3. **Environment Agnostic**: Same config works across dev/staging/prod
4. **Hot-Reload Friendly**: Works seamlessly with flux for live updates

### **Provider Selection in Code**

**❌ Old Way (Provider Type in Config):**
```yaml
provider:
  type: "redis"  # ❌ Config specifies provider
  url: "redis://localhost:6379"
```

**✅ New Way (Provider Selection in Go):**
```yaml
# ✅ Config is provider-agnostic
url: "redis://localhost:6379"
pool_size: 10
max_size: 104857600  # Used by memory provider when needed
```

```go
// ✅ Go code selects provider
import cacheredis "zbz/providers/cache-redis"
contract := cacheredis.NewRedisCache(config)  // Provider choice in code
```

## Service Architecture Examples

### **Cache Service (Completed)**

```go
// 1. Provider packages
cacheredis.NewRedisCache(config)      // → *CacheContract[redis.Cmdable]
cachememory.NewMemoryCache(config)    // → *CacheContract[*MemoryProvider]
cachefilesystem.NewFilesystemCache(config) // → *CacheContract[*FileSystemProvider]

// 2. Usage
contract := cacheredis.NewRedisCache(config)
contract.Register()  // Sets global singleton

// 3. Package functions
cache.Set(ctx, "key", value)           // Uses singleton
users := cache.Table[User]("users")    // Type-safe table abstraction

// 4. Native access
redisClient := contract.Native()       // redis.Cmdable - no casting!
```

### **Depot Service (Storage) - New Implementation**

```go
// 1. Provider packages  
depots3.NewS3Storage(config)          // → *DepotContract[*s3.S3]
depotgcs.NewGCSStorage(config)        // → *DepotContract[*storage.Client]
depotfs.NewFilesystemStorage(config) // → *DepotContract[*os.File]

// 2. Usage
contract := depots3.NewS3Storage(config)
contract.Register()  // Sets global singleton

// 3. Package functions
depot.Set("key", data, ttl)           // Uses singleton
depot.Subscribe("key", callback)      // Reactive operations

// 4. Native access
s3Client := contract.Native()         // *s3.S3 - no casting!
```

### **Zlog Service (Logging) - Future Implementation**

```go
// 1. Provider packages
zlogzap.NewZapLogger(config)          // → *Contract[*zap.Logger]
zlogzerolog.NewZerologLogger(config) // → *Contract[zerolog.Logger]

// 2. Usage  
contract := zlogzap.NewZapLogger(config)
contract.Register()  // Sets global singleton

// 3. Package functions
zlog.Info("message", zlog.String("key", "value"))  // Uses singleton

// 4. Native access
zapLogger := contract.Native()        // *zap.Logger - no casting!
```

## Advanced Patterns

### **Flux Integration (Hot-Reload)**

All services support hot-reloading through flux integration:

```go
// Watch config file for changes
fluxContract, err := cache.ConfigureFromFlux(
    cacheredis.NewRedisCache,  // Provider function
    depotContract,             // Config storage
    "cache.yaml",             // Config file to watch
)

// Any changes to cache.yaml automatically reconfigure the cache
// Provider type stays the same (determined by Go code)
// Provider settings can be updated live
```

### **Multi-Service Orchestration**

Services can be composed for complex scenarios:

```go
// 1. Set up storage
storageContract := depots3.NewS3Storage(depotConfig)
storageContract.Register()

// 2. Set up cache with storage backend
cacheContract := cacheredis.NewRedisCache(cacheConfig)
cacheContract.Register()

// 3. Set up logging
logContract := zlogzap.NewZapLogger(zlogConfig)
logContract.Register()

// 4. All package functions now work together
depot.Set("config.yaml", configData, 0)  // Store in S3
cache.Set(ctx, "user:1", userData)       // Cache in Redis
zlog.Info("Services configured")          // Log with Zap
```

### **Environment-Based Provider Selection**

```go
func SetupCache(config CacheConfig) error {
    env := os.Getenv("APP_ENV")
    
    switch env {
    case "production":
        contract := cacheredis.NewRedisCache(config)
        return contract.Register()
    case "development":
        contract := cachememory.NewMemoryCache(config)
        return contract.Register()
    default:
        contract := cachefilesystem.NewFilesystemCache(config)
        return contract.Register()
    }
}
```

## Implementation Guidelines

### **For Service Authors**

1. **Create universal config** with `yaml`/`json` struct tags
2. **Design common interface** that all providers implement
3. **Implement singleton service** that holds current provider + config
4. **Add package-level functions** that delegate to singleton
5. **Support flux integration** for hot-reloading

### **For Provider Authors**

1. **Create unique constructor** like `NewRedisCache(config ServiceConfig)`
2. **Return generic contract** with native client type: `*ServiceContract[NativeType]`
3. **Implement service interface** with wrapper around native client  
4. **Use universal config fields** - ignore irrelevant ones
5. **Test with other providers** to ensure config compatibility

### **For Application Developers**

1. **Import specific providers** you want to use
2. **Create contracts** with universal config
3. **Register as singleton** for package-level convenience
4. **Use package functions** for common operations
5. **Access native clients** for provider-specific features

## Benefits Summary

### **For Framework**
- **Consistent patterns** across all services
- **Easy to add new providers** without breaking changes
- **Type safety** prevents runtime errors
- **Hot-reload support** built-in

### **For Provider Authors**
- **Independence** - no coordination required
- **Full native API access** preserved
- **Standard patterns** to follow
- **Automatic integration** with service features

### **For Application Developers**
- **No vendor lock-in** - switch providers easily
- **Type safety** - no casting required
- **Powerful abstractions** with escape hatches
- **Consistent APIs** across all services

---

## Migration Path

### **Phase 1: Core Services (Completed)**
- ✅ Cache service with universal contract pattern
- ✅ Depot service updated to match cache pattern

### **Phase 2: Provider Updates (In Progress)**
- 🔄 Update all depot providers to universal config
- ⏳ Update all zlog providers to universal config
- ⏳ Add flux integration to all services

### **Phase 3: Advanced Features**
- ⏳ Multi-service orchestration helpers
- ⏳ Health check integration
- ⏳ Metrics collection across all providers
- ⏳ Distributed tracing integration

---

The Universal Contract Pattern represents a fundamental shift in how Go services handle provider abstraction, offering unprecedented flexibility while maintaining type safety and performance. This architecture enables ZBZ to scale from simple single-provider applications to complex multi-service distributed systems with ease.