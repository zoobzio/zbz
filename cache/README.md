# Cache Module

The cache module provides caching drivers for ZBZ. It follows the driver pattern where you can plug in different caching backends (Redis, Memory, Memcached, etc.) while maintaining a consistent interface.

## Architecture

```
lib/cache.go (ZBZ Cache Alias) → Cache Interface → Concrete Implementations
                                      ↓
                        ┌─────────────────────────────────┐
                        │  redis.go (Redis Driver)        │
                        │  memory.go (Memory Driver)      │
                        │  memcached.go (Custom Driver)   │
                        └─────────────────────────────────┘
```

## Cache Interface

All cache drivers must implement the `Cache` interface defined in `cache.go`:

```go
type Cache interface {
    // Basic operations
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl int) error
    Delete(ctx context.Context, key string) error
    
    // Batch operations
    GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)
    SetMulti(ctx context.Context, items map[string][]byte, ttl int) error
    DeleteMulti(ctx context.Context, keys []string) error
    
    // Utility operations
    Exists(ctx context.Context, key string) (bool, error)
    Clear(ctx context.Context) error
    Keys(ctx context.Context, pattern string) ([]string, error)
    
    // Connection management
    Ping(ctx context.Context) error
    Close() error
    
    // Driver metadata
    DriverName() string
    DriverVersion() string
}
```

## Built-in Drivers

### Redis Driver (`redis.go`)

High-performance distributed caching with Redis:

```go
cache := cache.NewRedisCache("redis://localhost:6379")

// With password and database
cache := cache.NewRedisCache("redis://:password@localhost:6379/1")
```

**Features**:
- Connection pooling
- Automatic failover
- Pub/Sub support (future)
- Cluster support (future)

### Memory Driver (`memory.go`)

In-memory caching for development and testing:

```go
cache := cache.NewMemoryCache()
```

**Features**:
- Thread-safe operations
- Automatic cleanup of expired keys
- No external dependencies
- TTL support

## Creating a Custom Cache Driver

### 1. Implement the Cache Interface

```go
package mycache

import (
    "context"
    "time"
    "zbz/lib/cache"
)

type MyCacheDriver struct {
    client   MyClientType
    config   MyConfig
}

func NewMyCacheDriver(config MyConfig) cache.Cache {
    client := initializeMyClient(config)
    return &MyCacheDriver{
        client: client,
        config: config,
    }
}

func (m *MyCacheDriver) Get(ctx context.Context, key string) ([]byte, error) {
    // Implement get operation for your cache backend
    value, err := m.client.Get(ctx, key)
    if err != nil {
        if isNotFoundError(err) {
            return nil, cache.ErrKeyNotFound
        }
        return nil, err
    }
    
    return value, nil
}

func (m *MyCacheDriver) Set(ctx context.Context, key string, value []byte, ttl int) error {
    // Implement set operation with TTL
    expiration := time.Duration(ttl) * time.Second
    return m.client.Set(ctx, key, value, expiration)
}

func (m *MyCacheDriver) Delete(ctx context.Context, key string) error {
    return m.client.Delete(ctx, key)
}

func (m *MyCacheDriver) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
    // Implement batch get operation
    result := make(map[string][]byte)
    
    for _, key := range keys {
        if value, err := m.Get(ctx, key); err == nil {
            result[key] = value
        }
    }
    
    return result, nil
}

func (m *MyCacheDriver) SetMulti(ctx context.Context, items map[string][]byte, ttl int) error {
    // Implement batch set operation
    for key, value := range items {
        if err := m.Set(ctx, key, value, ttl); err != nil {
            return err
        }
    }
    return nil
}

func (m *MyCacheDriver) DeleteMulti(ctx context.Context, keys []string) error {
    // Implement batch delete operation
    for _, key := range keys {
        if err := m.Delete(ctx, key); err != nil {
            return err
        }
    }
    return nil
}

func (m *MyCacheDriver) Exists(ctx context.Context, key string) (bool, error) {
    _, err := m.Get(ctx, key)
    if err == cache.ErrKeyNotFound {
        return false, nil
    }
    return err == nil, err
}

func (m *MyCacheDriver) Clear(ctx context.Context) error {
    // Implement cache clear operation
    return m.client.FlushAll(ctx)
}

func (m *MyCacheDriver) Keys(ctx context.Context, pattern string) ([]string, error) {
    // Implement key pattern matching
    return m.client.Keys(ctx, pattern)
}

func (m *MyCacheDriver) Ping(ctx context.Context) error {
    return m.client.Ping(ctx)
}

func (m *MyCacheDriver) Close() error {
    return m.client.Close()
}

func (m *MyCacheDriver) DriverName() string {
    return "mycache"
}

func (m *MyCacheDriver) DriverVersion() string {
    return "1.0.0"
}
```

### 2. Integration with ZBZ

Cache drivers are typically used through contracts:

```go
package main

import (
    "zbz/lib"
    "mycache"
)

func main() {
    engine := zbz.NewEngine()
    
    // Create cache contract
    cacheContract := zbz.CacheContract{
        BaseContract: zbz.BaseContract{
            Name:        "primary-cache",
            Description: "Primary application cache",
        },
        Service: "mycache",
        URL:     "mycache://localhost:1234",
        Config: map[string]any{
            "pool_size": 10,
            "timeout":   30,
        },
    }
    
    // Cache is resolved automatically by the contract
    cache := cacheContract.Cache()
    
    // Use cache in other services (auth, custom handlers, etc.)
}
```

## Common Use Cases

### Authentication Caching

Store user session data and tokens:

```go
// Cache user data after authentication
userData := &auth.AuthUser{
    Sub:   "user123",
    Email: "user@example.com",
    Name:  "John Doe",
}

data, _ := json.Marshal(userData)
cache.Set(ctx, "user:user123", data, 3600) // 1 hour TTL

// Retrieve user data
data, err := cache.Get(ctx, "user:user123")
if err == nil {
    var user auth.AuthUser
    json.Unmarshal(data, &user)
}
```

### API Response Caching

Cache expensive API responses:

```go
func ExpensiveHandler(ctx zbz.RequestContext) {
    cacheKey := "api:expensive:" + ctx.PathParam("id")
    
    // Try cache first
    if cached, err := cache.Get(ctx.Context(), cacheKey); err == nil {
        ctx.JSON(json.RawMessage(cached))
        return
    }
    
    // Compute expensive result
    result := performExpensiveOperation()
    data, _ := json.Marshal(result)
    
    // Cache for 5 minutes
    cache.Set(ctx.Context(), cacheKey, data, 300)
    
    ctx.JSON(result)
}
```

### Configuration Caching

Cache frequently accessed configuration:

```go
func GetConfig(key string) (string, error) {
    cacheKey := "config:" + key
    
    // Check cache first
    if value, err := cache.Get(context.Background(), cacheKey); err == nil {
        return string(value), nil
    }
    
    // Load from database/file
    value, err := loadConfigFromSource(key)
    if err != nil {
        return "", err
    }
    
    // Cache for 1 hour
    cache.Set(context.Background(), cacheKey, []byte(value), 3600)
    
    return value, nil
}
```

## Performance Considerations

### Key Design

- **Use consistent prefixes**: `user:123`, `api:endpoint:param`
- **Avoid hot keys**: Distribute load across multiple keys
- **Use meaningful TTLs**: Balance freshness vs performance

### Memory Management

- **Monitor memory usage**: Especially with Memory driver
- **Set appropriate TTLs**: Prevent unbounded growth
- **Use batch operations**: More efficient than individual calls

### Network Optimization

- **Connection pooling**: Reuse connections (Redis driver does this)
- **Batch operations**: Reduce round trips with GetMulti/SetMulti
- **Compression**: For large values (implement in your driver)

## Error Handling

```go
// Standard error handling pattern
value, err := cache.Get(ctx, "mykey")
if err != nil {
    if err == cache.ErrKeyNotFound {
        // Handle cache miss
        value = computeValue()
        cache.Set(ctx, "mykey", value, 300)
    } else {
        // Handle cache error (network, etc.)
        // Log error and continue without cache
        logger.Log.Warn("Cache error", logger.Err(err))
        value = computeValue()
    }
}
```

## Testing

Test your cache driver with:

```go
func TestMyCacheDriver(t *testing.T) {
    cache := mycache.NewMyCacheDriver(config)
    ctx := context.Background()
    
    // Test basic operations
    err := cache.Set(ctx, "test-key", []byte("test-value"), 60)
    assert.NoError(t, err)
    
    value, err := cache.Get(ctx, "test-key")
    assert.NoError(t, err)
    assert.Equal(t, []byte("test-value"), value)
    
    // Test existence
    exists, err := cache.Exists(ctx, "test-key")
    assert.NoError(t, err)
    assert.True(t, exists)
    
    // Test deletion
    err = cache.Delete(ctx, "test-key")
    assert.NoError(t, err)
    
    _, err = cache.Get(ctx, "test-key")
    assert.Equal(t, cache.ErrKeyNotFound, err)
    
    // Test batch operations
    items := map[string][]byte{
        "key1": []byte("value1"),
        "key2": []byte("value2"),
    }
    
    err = cache.SetMulti(ctx, items, 60)
    assert.NoError(t, err)
    
    result, err := cache.GetMulti(ctx, []string{"key1", "key2"})
    assert.NoError(t, err)
    assert.Equal(t, items, result)
}
```

## Driver-Specific Notes

### Redis
- Supports complex data types (lists, sets, hashes)
- Atomic operations available
- Pub/Sub for real-time features
- Persistence options available

### Memory
- Fastest for single-instance applications
- Data lost on restart
- Good for development/testing
- Memory usage grows with cache size

### Memcached
- Simple key-value store
- Good for distributed caching
- No persistence
- LRU eviction policy