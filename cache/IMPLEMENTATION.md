# Cache System V2 - BYOC (Bring Your Own Cache) Implementation Plan

## Overview
Design a **Bring Your Own Cache** system following zbz's modular contract-provider pattern. Users can plug in any cache backend through standardized adapters - Redis, Memcached, DynamoDB, file-based, cloud storage, or custom implementations. The system provides type-safe operations with struct scanning while maintaining complete provider flexibility.

## Current State Analysis

### ✅ What We Have
- Basic cache interface with Redis/Memory implementations
- Simple contract structure (but not following zbz patterns)
- Working Redis and Memory providers
- TTL support and basic operations

### ❌ What's Missing
- **Contract-Provider Pattern**: Current system doesn't follow zlog/hodor patterns
- **Generic Support**: No type-safe struct scanning/marshaling
- **Provider Registration**: No registry system for cache providers
- **Advanced Features**: No batch operations, pipeline support, or pub/sub
- **Service Integration**: No zbz engine integration
- **Configuration Management**: Basic config handling

## Architecture Questions Answered

### 🤔 **Contract Pattern: Singleton vs Multi-Instance?**

**Answer: Multi-Instance (like Hodor), NOT Singleton (like zlog)**

- **✅ Multiple independent cache instances** - each with its own provider, config, and type binding
- **✅ Hot-swappable at instance level** - `userCache.SwapProvider(newProvider)`  
- **❌ NOT a global singleton** - no global `cache.Get()` like `zlog.Info()`

**Why multi-instance?**
- **Multiple cache types**: user cache, session cache, config cache, file cache
- **Different backends per use case**: memory for speed, Redis for distribution, DynamoDB for scale
- **Independent configuration**: different TTLs, serializers, key prefixes per cache type
- **Type safety**: each cache bound to specific struct type at compile time

### 🔧 **Generics Implementation?**

**Answer: Per-contract type binding with compile-time safety**

```go
// Each contract is bound to ONE specific type T
userCache := cache.NewContract[User]("users", redisProvider)     // Only works with User structs
sessionCache := cache.NewContract[Session]("sessions", memProvider) // Only works with Session structs

// Type safety enforced at contract level
userService := userCache.Cache()   // Returns CacheService[User]
sessionService := sessionCache.Cache() // Returns CacheService[Session]

// Compile-time type checking
var user User
userService.GetStruct(ctx, "key", &user) // ✅ Correct type

var session Session
userService.GetStruct(ctx, "key", &session) // ❌ Compile error - type mismatch
```

### 🔄 **Hot-Swapping Support?**

**Answer: Per-instance provider swapping + application-level contract replacement**

```go
// Option 1: Swap provider within existing contract
userCache.SwapProvider(newDynamoProvider) // Same User type, different backend

// Option 2: Replace entire contract reference (application-level)
app.userCache = cache.NewContract[User]("users-v2", newProvider).Cache()

// Option 3: Factory pattern for environment-based selection
userCache := createCacheForEnvironment[User]("users", getCurrentEnv())
```

### 🏗️ **Does Contract Tie Cache to Specific Struct?**

**Answer: YES - each contract is bound to exactly one struct type**

- **One contract = One type**: `CacheContract[User]` only works with `User` structs
- **Compile-time safety**: Type mismatches caught at compile time, not runtime
- **Multiple contracts for multiple types**: Need separate contracts for `User`, `Session`, `Config`, etc.
- **Provider reuse**: Same provider can back multiple type-specific contracts

```go
// Same Redis provider, multiple type-bound contracts
redisProvider := cache.NewProvider("redis", redisConfig)

userCache := cache.NewContract[User]("users", redisProvider)       // User-only cache
sessionCache := cache.NewContract[Session]("sessions", redisProvider) // Session-only cache
configCache := cache.NewContract[Config]("config", redisProvider)     // Config-only cache
```

## BYOC Architecture - Unlimited Cache Backends

### Core BYOC Philosophy
```
     User Application
           ↓
    CacheContract[T] (Type-Safe Interface)
           ↓
    CacheProvider (Standardized API)
           ↓
    ┌─────────────────────────────────────────────────────────────────────┐
    │                    BYOC Adapter Ecosystem                           │
    │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
    │  │   Redis     │ │ Memcached   │ │ DynamoDB    │ │   Memory    │    │
    │  │  Adapter    │ │  Adapter    │ │  Adapter    │ │   Adapter   │    │
    │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘    │
    │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
    │  │ File System │ │    HTTP     │ │  S3/Cloud   │ │   Custom    │    │
    │  │  Adapter    │ │   Adapter   │ │  Adapter    │ │   Adapter   │    │
    │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘    │
    │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
    │  │  SQLite     │ │ Cassandra   │ │   Etcd      │ │ Badger/Bolt │    │
    │  │  Adapter    │ │  Adapter    │ │  Adapter    │ │   Adapter   │    │
    │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘    │
    └─────────────────────────────────────────────────────────────────────┘
```

### BYOC Benefits
1. **No Vendor Lock-In**: Switch cache backends without code changes
2. **Environment Flexibility**: Development (memory) → Staging (Redis) → Production (DynamoDB)
3. **Cost Optimization**: Use cheaper storage for non-critical caching
4. **Performance Tuning**: Match cache backend to specific workload requirements
5. **Compliance Ready**: Use enterprise/air-gapped solutions when needed

### Core Components Architecture
```
CacheContract[T] → CacheProvider → BYOC Adapter → Any Backend
     ↓                 ↓              ↓              ↓
  Type-Safe      Standardized    Adapter Layer   Redis/Dynamo/
  Interface       API Layer      Translation     File/Custom/etc
```

### 1. Provider Interface (Core Abstraction)
```go
// CacheProvider defines the standardized interface all cache backends must implement
type CacheProvider interface {
    // Basic operations
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    
    // Batch operations for performance
    GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)
    SetMulti(ctx context.Context, items map[string]CacheItem, ttl time.Duration) error
    DeleteMulti(ctx context.Context, keys []string) error
    
    // Advanced operations
    Keys(ctx context.Context, pattern string) ([]string, error)
    TTL(ctx context.Context, key string) (time.Duration, error)
    Expire(ctx context.Context, key string, ttl time.Duration) error
    
    // Management operations
    Clear(ctx context.Context) error
    Stats(ctx context.Context) (CacheStats, error)
    Ping(ctx context.Context) error
    Close() error
    
    // Provider metadata
    GetProvider() string
    GetVersion() string
}

type CacheItem struct {
    Key   string
    Value []byte
    TTL   time.Duration
}

type CacheStats struct {
    Provider     string
    Hits         int64
    Misses       int64
    Keys         int64
    Memory       int64 // bytes
    Connections  int
}
```

### 2. Generic Contract System (Multi-Instance Pattern)

**Cache follows Hodor's multi-instance pattern, NOT zlog's singleton pattern**

```go
// CacheContract provides type-safe caching bound to a specific struct type
// Each contract is an INDEPENDENT instance (like hodor), not a competing singleton (like zlog)
type CacheContract[T any] struct {
    name     string
    provider CacheProvider
    
    // Optional configuration
    keyPrefix    string
    defaultTTL   time.Duration
    serializer   Serializer
    
    // Registration state (optional - for service discovery)
    alias      string
    registered bool
}

// Serializer handles struct ↔ []byte conversion
type Serializer interface {
    Marshal(v interface{}) ([]byte, error)
    Unmarshal(data []byte, v interface{}) error
    ContentType() string
}

// NewContract creates a typed cache contract bound to type T
// Unlike zlog contracts, each cache contract is a separate instance
func NewContract[T any](name string, provider CacheProvider) *CacheContract[T] {
    return &CacheContract[T]{
        name:       name,
        provider:   provider,
        serializer: NewJSONSerializer(), // Default to JSON
        defaultTTL: 1 * time.Hour,
        registered: false,
    }
}

// Key insight: The generic type T binds the contract to a specific struct type
// This provides compile-time type safety for all cache operations
```

### 3. Multi-Instance Type-Safe Operations

```go
// Cache returns the service interface bound to type T
// Each contract creates its own independent service instance
func (c *CacheContract[T]) Cache() CacheService[T] {
    return &cacheService[T]{
        contract: c,
        provider: c.provider,
    }
}

// Each CacheService[T] is bound to a specific type at compile time
type CacheService[T any] interface {
    // Type-safe struct operations - T is fixed at contract creation
    GetStruct(ctx context.Context, key string, dest *T) error
    SetStruct(ctx context.Context, key string, value T, ttl ...time.Duration) error
    
    // Batch struct operations - all operations use the same type T
    GetStructMulti(ctx context.Context, keys []string) (map[string]T, error)
    SetStructMulti(ctx context.Context, items map[string]T, ttl ...time.Duration) error
    
    // Raw operations (for backwards compatibility)
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl ...time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    
    // Advanced operations
    Keys(ctx context.Context, pattern string) ([]string, error)
    Clear(ctx context.Context) error
    Stats(ctx context.Context) (CacheStats, error)
    
    // Hot-swapping support
    SwapProvider(newProvider CacheProvider) error
}

// Multi-instance usage pattern
type AppServices struct {
    userCache    CacheService[User]    // Independent instance for Users
    sessionCache CacheService[Session] // Independent instance for Sessions  
    configCache  CacheService[Config]  // Independent instance for Config
}

func NewAppServices() *AppServices {
    // Each cache is independent with its own type binding and provider
    userContract := cache.NewContract[User]("users", redisProvider)
    sessionContract := cache.NewContract[Session]("sessions", memoryProvider)
    configContract := cache.NewContract[Config]("config", etcdProvider)
    
    return &AppServices{
        userCache:    userContract.Cache(),
        sessionCache: sessionContract.Cache(), 
        configCache:  configContract.Cache(),
    }
}
```

### 4. Provider Registration System
```go
// Provider registry for discovery and instantiation
var providerRegistry = make(map[string]ProviderFactory)

type ProviderFactory func(config map[string]interface{}) (CacheProvider, error)

// RegisterProvider registers a cache provider factory
func RegisterProvider(name string, factory ProviderFactory) {
    providerRegistry[name] = factory
}

// NewProvider creates a provider instance by name
func NewProvider(name string, config map[string]interface{}) (CacheProvider, error) {
    factory, exists := providerRegistry[name]
    if !exists {
        return nil, fmt.Errorf("unknown cache provider: %s", name)
    }
    return factory(config)
}

// Auto-registration in provider files
func init() {
    RegisterProvider("redis", NewRedisProvider)
    RegisterProvider("memory", NewMemoryProvider)
    RegisterProvider("memcached", NewMemcachedProvider)
}
```

## BYOC Adapter Ecosystem

The true power of the BYOC system is the unlimited adapter ecosystem. Each adapter translates the standardized `CacheProvider` interface to a specific backend's native API.

### 1. In-Memory Adapters

#### Memory Provider (Development/Testing)
```go
// For single-instance applications and testing
func NewMemoryProvider(config map[string]interface{}) (CacheProvider, error) {
    return &MemoryProvider{
        data:     make(map[string]*cacheItem),
        maxSize:  getConfigInt64(config, "max_size", 100*1024*1024), // 100MB
        eviction: getConfigString(config, "eviction", "lru"),
    }, nil
}
```

#### Ristretto Provider (High-Performance Memory)
```go
// Ultra-fast in-memory cache with advanced features
func NewRistrettoProvider(config map[string]interface{}) (CacheProvider, error) {
    cache, _ := ristretto.NewCache(&ristretto.Config{
        NumCounters: getConfigInt64(config, "num_counters", 1e7),
        MaxCost:     getConfigInt64(config, "max_cost", 1<<30),
        BufferItems: getConfigInt64(config, "buffer_items", 64),
    })
    return &RistrettoProvider{cache: cache}, nil
}
```

### 2. Network Cache Adapters

#### Redis Provider (Enhanced)
```go
// Production-ready distributed caching
type RedisProvider struct {
    client      *redis.Client
    clusterMode bool
    pipeline    bool
    stats       *CacheStats
}

func NewRedisProvider(config map[string]interface{}) (CacheProvider, error) {
    cfg := RedisConfig{
        URL:              getConfigString(config, "url", "redis://localhost:6379"),
        PoolSize:         getConfigInt(config, "pool_size", 10),
        MaxRetries:       getConfigInt(config, "max_retries", 3),
        ReadTimeout:      getConfigDuration(config, "read_timeout", 5*time.Second),
        WriteTimeout:     getConfigDuration(config, "write_timeout", 3*time.Second),
        EnablePipelining: getConfigBool(config, "enable_pipelining", true),
        EnableCluster:    getConfigBool(config, "enable_cluster", false),
    }
    
    var client redis.Cmdable
    if cfg.EnableCluster {
        client = redis.NewClusterClient(&redis.ClusterOptions{
            Addrs:        getConfigStringSlice(config, "cluster_addrs"),
            PoolSize:     cfg.PoolSize,
            ReadTimeout:  cfg.ReadTimeout,
            WriteTimeout: cfg.WriteTimeout,
        })
    } else {
        opt, _ := redis.ParseURL(cfg.URL)
        opt.PoolSize = cfg.PoolSize
        opt.MaxRetries = cfg.MaxRetries
        client = redis.NewClient(opt)
    }
    
    return &RedisProvider{
        client:      client.(*redis.Client),
        clusterMode: cfg.EnableCluster,
        pipeline:    cfg.EnablePipelining,
    }, nil
}
```

#### Memcached Provider
```go
// Simple, fast distributed caching
func NewMemcachedProvider(config map[string]interface{}) (CacheProvider, error) {
    servers := getConfigStringSlice(config, "servers")
    if len(servers) == 0 {
        servers = []string{"localhost:11211"}
    }
    
    client := memcache.New(servers...)
    client.MaxIdleConns = getConfigInt(config, "max_idle_conns", 10)
    client.Timeout = getConfigDuration(config, "timeout", 100*time.Millisecond)
    
    return &MemcachedProvider{client: client}, nil
}
```

#### HTTP Cache Provider
```go
// Cache over HTTP API (for microservices/external systems)
func NewHTTPProvider(config map[string]interface{}) (CacheProvider, error) {
    baseURL := getConfigString(config, "base_url", "")
    if baseURL == "" {
        return nil, fmt.Errorf("base_url required for HTTP cache provider")
    }
    
    client := &http.Client{
        Timeout: getConfigDuration(config, "timeout", 30*time.Second),
    }
    
    return &HTTPProvider{
        baseURL: baseURL,
        client:  client,
        headers: getConfigStringMap(config, "headers"),
        auth:    getConfigString(config, "auth_token", ""),
    }, nil
}
```

### 3. Cloud Storage Adapters

#### DynamoDB Provider
```go
// AWS DynamoDB as cache backend
func NewDynamoDBProvider(config map[string]interface{}) (CacheProvider, error) {
    cfg := aws.LoadDefaultConfig(context.TODO())
    
    if region := getConfigString(config, "region", ""); region != "" {
        cfg.Region = region
    }
    
    client := dynamodb.NewFromConfig(cfg)
    tableName := getConfigString(config, "table_name", "cache")
    
    return &DynamoDBProvider{
        client:    client,
        tableName: tableName,
        ttlField:  getConfigString(config, "ttl_field", "expires_at"),
    }, nil
}

func (d *DynamoDBProvider) Get(ctx context.Context, key string) ([]byte, error) {
    result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String(d.tableName),
        Key: map[string]types.AttributeValue{
            "cache_key": &types.AttributeValueMemberS{Value: key},
        },
    })
    if err != nil {
        return nil, err
    }
    
    if result.Item == nil {
        return nil, ErrCacheKeyNotFound
    }
    
    // Check TTL
    if ttl, exists := result.Item[d.ttlField]; exists {
        if expiry, ok := ttl.(*types.AttributeValueMemberN); ok {
            expiryTime, _ := strconv.ParseInt(expiry.Value, 10, 64)
            if time.Now().Unix() > expiryTime {
                return nil, ErrCacheKeyNotFound
            }
        }
    }
    
    if value, exists := result.Item["value"]; exists {
        if valueBytes, ok := value.(*types.AttributeValueMemberB); ok {
            return valueBytes.Value, nil
        }
    }
    
    return nil, ErrCacheKeyNotFound
}
```

#### S3/Cloud Storage Provider
```go
// Use cloud object storage as cache (for large objects)
func NewS3Provider(config map[string]interface{}) (CacheProvider, error) {
    cfg := aws.LoadDefaultConfig(context.TODO())
    client := s3.NewFromConfig(cfg)
    
    return &S3Provider{
        client: client,
        bucket: getConfigString(config, "bucket", ""),
        prefix: getConfigString(config, "prefix", "cache/"),
    }, nil
}

// Integrates with existing hodor S3 provider
func NewHodorProvider(hodorContract *hodor.HodorContract) CacheProvider {
    return &HodorCacheProvider{
        storage: hodorContract,
        prefix:  "cache:",
    }
}
```

### 4. Database Adapters

#### SQLite Provider
```go
// File-based cache using SQLite
func NewSQLiteProvider(config map[string]interface{}) (CacheProvider, error) {
    dbPath := getConfigString(config, "db_path", "cache.db")
    
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }
    
    // Create cache table
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS cache_entries (
            key TEXT PRIMARY KEY,
            value BLOB,
            expires_at INTEGER
        )
    `)
    if err != nil {
        return nil, err
    }
    
    return &SQLiteProvider{db: db}, nil
}
```

#### PostgreSQL Provider
```go
// Use existing PostgreSQL as cache backend
func NewPostgreSQLProvider(config map[string]interface{}) (CacheProvider, error) {
    dsn := getConfigString(config, "dsn", "")
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }
    
    // Enable unlogged tables for performance
    _, err = db.Exec(`
        CREATE UNLOGGED TABLE IF NOT EXISTS cache_entries (
            key TEXT PRIMARY KEY,
            value BYTEA,
            expires_at TIMESTAMPTZ
        )
    `)
    
    return &PostgreSQLProvider{
        db:        db,
        tableName: getConfigString(config, "table_name", "cache_entries"),
    }, nil
}
```

#### Cassandra Provider
```go
// Distributed database as cache backend
func NewCassandraProvider(config map[string]interface{}) (CacheProvider, error) {
    cluster := gocql.NewCluster(getConfigStringSlice(config, "hosts")...)
    cluster.Keyspace = getConfigString(config, "keyspace", "cache")
    cluster.Consistency = gocql.Quorum
    
    session, err := cluster.CreateSession()
    if err != nil {
        return nil, err
    }
    
    return &CassandraProvider{session: session}, nil
}
```

### 5. Embedded Storage Adapters

#### BadgerDB Provider
```go
// High-performance embedded key-value store
func NewBadgerProvider(config map[string]interface{}) (CacheProvider, error) {
    opts := badger.DefaultOptions(getConfigString(config, "dir", "/tmp/badger"))
    opts.Logger = nil // Disable logging
    
    db, err := badger.Open(opts)
    if err != nil {
        return nil, err
    }
    
    return &BadgerProvider{db: db}, nil
}
```

#### BoltDB Provider
```go
// Simple embedded database
func NewBoltProvider(config map[string]interface{}) (CacheProvider, error) {
    dbPath := getConfigString(config, "db_path", "cache.bolt")
    
    db, err := bolt.Open(dbPath, 0600, &bolt.Options{
        Timeout: 1 * time.Second,
    })
    if err != nil {
        return nil, err
    }
    
    bucketName := getConfigString(config, "bucket", "cache")
    
    return &BoltProvider{
        db:     db,
        bucket: bucketName,
    }, nil
}
```

### 6. Specialized Adapters

#### Etcd Provider
```go
// Distributed configuration store as cache
func NewEtcdProvider(config map[string]interface{}) (CacheProvider, error) {
    client, err := clientv3.New(clientv3.Config{
        Endpoints:   getConfigStringSlice(config, "endpoints"),
        DialTimeout: getConfigDuration(config, "dial_timeout", 5*time.Second),
    })
    if err != nil {
        return nil, err
    }
    
    return &EtcdProvider{
        client: client,
        prefix: getConfigString(config, "prefix", "/cache/"),
    }, nil
}
```

#### File System Provider
```go
// Simple file-based cache
func NewFileSystemProvider(config map[string]interface{}) (CacheProvider, error) {
    baseDir := getConfigString(config, "base_dir", "/tmp/cache")
    
    if err := os.MkdirAll(baseDir, 0755); err != nil {
        return nil, err
    }
    
    return &FileSystemProvider{
        baseDir:     baseDir,
        permissions: os.FileMode(getConfigInt(config, "permissions", 0644)),
    }, nil
}
```

#### Multi-Tier Provider
```go
// L1 (memory) + L2 (Redis) cache hierarchy
func NewMultiTierProvider(config map[string]interface{}) (CacheProvider, error) {
    l1Config := getConfigMap(config, "l1")
    l2Config := getConfigMap(config, "l2")
    
    l1Provider, err := NewProvider(getConfigString(l1Config, "provider", "memory"), l1Config)
    if err != nil {
        return nil, err
    }
    
    l2Provider, err := NewProvider(getConfigString(l2Config, "provider", "redis"), l2Config)
    if err != nil {
        return nil, err
    }
    
    return &MultiTierProvider{
        l1:          l1Provider,
        l2:          l2Provider,
        promoteHits: getConfigBool(config, "promote_hits", true),
    }, nil
}
```

## Provider Implementations

### Registry-Based Provider Discovery
```go
// Auto-registration in provider packages
func init() {
    // In-memory providers
    RegisterProvider("memory", NewMemoryProvider)
    RegisterProvider("ristretto", NewRistrettoProvider)
    
    // Network providers  
    RegisterProvider("redis", NewRedisProvider)
    RegisterProvider("memcached", NewMemcachedProvider)
    RegisterProvider("http", NewHTTPProvider)
    
    // Cloud providers
    RegisterProvider("dynamodb", NewDynamoDBProvider)
    RegisterProvider("s3", NewS3Provider)
    RegisterProvider("hodor", NewHodorProvider)
    
    // Database providers
    RegisterProvider("sqlite", NewSQLiteProvider)
    RegisterProvider("postgresql", NewPostgreSQLProvider)
    RegisterProvider("cassandra", NewCassandraProvider)
    
    // Embedded providers
    RegisterProvider("badger", NewBadgerProvider)
    RegisterProvider("bolt", NewBoltProvider)
    
    // Specialized providers
    RegisterProvider("etcd", NewEtcdProvider)
    RegisterProvider("filesystem", NewFileSystemProvider)
    RegisterProvider("multitier", NewMultiTierProvider)
}

// Dynamic provider loading
func LoadProvider(name string, config map[string]interface{}) (CacheProvider, error) {
    if factory, exists := providerRegistry[name]; exists {
        return factory(config)
    }
    
    // Try loading external provider
    if plugin, err := loadExternalProvider(name); err == nil {
        return plugin.NewProvider(config)
    }
    
    return nil, fmt.Errorf("unknown cache provider: %s", name)
}
```

### Enhanced Redis Provider (Production Example)
```go
type RedisProvider struct {
    client   *redis.Client
    config   RedisConfig
    stats    *CacheStats
}

type RedisConfig struct {
    URL              string        `yaml:"url"`
    MaxRetries       int           `yaml:"max_retries"`
    PoolSize         int           `yaml:"pool_size"`
    MinIdleConns     int           `yaml:"min_idle_conns"`
    ConnMaxLifetime  time.Duration `yaml:"conn_max_lifetime"`
    ReadTimeout      time.Duration `yaml:"read_timeout"`
    WriteTimeout     time.Duration `yaml:"write_timeout"`
    
    // Advanced features
    EnablePipelining bool `yaml:"enable_pipelining"`
    EnableCluster    bool `yaml:"enable_cluster"`
}

func NewRedisProvider(config map[string]interface{}) (CacheProvider, error) {
    var redisConfig RedisConfig
    if err := mapstructure.Decode(config, &redisConfig); err != nil {
        return nil, fmt.Errorf("invalid redis config: %w", err)
    }
    
    // Set defaults
    if redisConfig.URL == "" {
        redisConfig.URL = "redis://localhost:6379"
    }
    if redisConfig.PoolSize == 0 {
        redisConfig.PoolSize = 10
    }
    
    opt, err := redis.ParseURL(redisConfig.URL)
    if err != nil {
        return nil, fmt.Errorf("invalid redis URL: %w", err)
    }
    
    // Apply config to options
    opt.MaxRetries = redisConfig.MaxRetries
    opt.PoolSize = redisConfig.PoolSize
    opt.MinIdleConns = redisConfig.MinIdleConns
    opt.ConnMaxLifetime = redisConfig.ConnMaxLifetime
    opt.ReadTimeout = redisConfig.ReadTimeout
    opt.WriteTimeout = redisConfig.WriteTimeout
    
    client := redis.NewClient(opt)
    
    return &RedisProvider{
        client: client,
        config: redisConfig,
        stats:  &CacheStats{Provider: "redis"},
    }, nil
}

// Enhanced batch operations
func (r *RedisProvider) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
    if len(keys) == 0 {
        return make(map[string][]byte), nil
    }
    
    // Use Redis pipeline for efficiency
    pipe := r.client.Pipeline()
    cmds := make([]*redis.StringCmd, len(keys))
    
    for i, key := range keys {
        cmds[i] = pipe.Get(ctx, key)
    }
    
    _, err := pipe.Exec(ctx)
    if err != nil && err != redis.Nil {
        return nil, err
    }
    
    result := make(map[string][]byte)
    for i, cmd := range cmds {
        if cmd.Err() == nil {
            result[keys[i]] = []byte(cmd.Val())
            r.stats.Hits++
        } else {
            r.stats.Misses++
        }
    }
    
    return result, nil
}

func (r *RedisProvider) SetMulti(ctx context.Context, items map[string]CacheItem, ttl time.Duration) error {
    if len(items) == 0 {
        return nil
    }
    
    pipe := r.client.Pipeline()
    for key, item := range items {
        effectiveTTL := ttl
        if item.TTL > 0 {
            effectiveTTL = item.TTL
        }
        pipe.Set(ctx, key, item.Value, effectiveTTL)
    }
    
    _, err := pipe.Exec(ctx)
    return err
}
```

### 2. Memory Provider (Enhanced)
```go
type MemoryProvider struct {
    data     map[string]*cacheItem
    mutex    sync.RWMutex
    stopChan chan bool
    stats    *CacheStats
    maxSize  int64 // Maximum memory usage in bytes
    currentSize int64
}

type cacheItem struct {
    value     []byte
    expiresAt time.Time
    size      int64
    accessCount int64
    lastAccess  time.Time
}

func NewMemoryProvider(config map[string]interface{}) (CacheProvider, error) {
    cfg := struct {
        MaxSize       int64         `mapstructure:"max_size"`       // bytes
        CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
        EvictionPolicy string        `mapstructure:"eviction_policy"` // "lru", "lfu", "ttl"
    }{
        MaxSize:         100 * 1024 * 1024, // 100MB default
        CleanupInterval: 1 * time.Minute,
        EvictionPolicy:  "lru",
    }
    
    if err := mapstructure.Decode(config, &cfg); err != nil {
        return nil, fmt.Errorf("invalid memory config: %w", err)
    }
    
    provider := &MemoryProvider{
        data:     make(map[string]*cacheItem),
        stopChan: make(chan bool),
        maxSize:  cfg.MaxSize,
        stats:    &CacheStats{Provider: "memory"},
    }
    
    go provider.cleanup(cfg.CleanupInterval)
    go provider.evictionWorker(cfg.EvictionPolicy)
    
    return provider, nil
}

// Enhanced memory management with eviction
func (m *MemoryProvider) evictIfNeeded() {
    if m.currentSize <= m.maxSize {
        return
    }
    
    // LRU eviction
    var oldestKey string
    var oldestTime time.Time = time.Now()
    
    for key, item := range m.data {
        if item.lastAccess.Before(oldestTime) {
            oldestTime = item.lastAccess
            oldestKey = key
        }
    }
    
    if oldestKey != "" {
        m.evictKey(oldestKey)
    }
}

func (m *MemoryProvider) evictKey(key string) {
    if item, exists := m.data[key]; exists {
        m.currentSize -= item.size
        delete(m.data, key)
    }
}
```

## BYOC Usage Examples - Same Code, Any Backend

The beauty of BYOC is **provider agnostic code** - write once, run on any cache backend.

### Seamless Provider Switching
```go
// Define your data structure once
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Your application code NEVER changes - only configuration
func UserService(cache cache.CacheService[User]) *userService {
    return &userService{cache: cache}
}

func (s *userService) GetUser(ctx context.Context, id int) (*User, error) {
    var user User
    key := fmt.Sprintf("user:%d", id)
    
    // This works with ANY cache provider
    if err := s.cache.GetStruct(ctx, key, &user); err == nil {
        return &user, nil
    }
    
    // Load from database
    user, err := s.db.LoadUser(id)
    if err != nil {
        return nil, err
    }
    
    // Cache for 1 hour - works with ANY provider
    s.cache.SetStruct(ctx, key, user, 1*time.Hour)
    return &user, nil
}
```

### Development → Staging → Production
```go
// Development: Fast in-memory caching
devProvider, _ := cache.NewProvider("memory", map[string]interface{}{
    "max_size": 50 * 1024 * 1024, // 50MB
})

// Staging: Redis for distributed testing
stagingProvider, _ := cache.NewProvider("redis", map[string]interface{}{
    "url":       "redis://staging-redis:6379",
    "pool_size": 10,
})

// Production: DynamoDB for enterprise scale
prodProvider, _ := cache.NewProvider("dynamodb", map[string]interface{}{
    "table_name": "app-cache",
    "region":     "us-east-1",
})

// Same contract, different backend
userCache := cache.NewContract[User]("user-cache", getProviderForEnv())
service := userCache.Cache()

// IDENTICAL code across all environments
user := User{ID: 123, Name: "John", Email: "john@example.com"}
service.SetStruct(ctx, "user:123", user, 1*time.Hour)
```

### Multi-Provider Architecture
```go
// Different cache backends for different use cases
type AppCaches struct {
    sessions cache.CacheService[Session]   // Redis: distributed sessions
    users    cache.CacheService[User]      // Memory: fast user lookups  
    files    cache.CacheService[FileInfo]  // S3: large file metadata
    config   cache.CacheService[Config]    // Etcd: distributed config
}

func NewAppCaches() *AppCaches {
    // Each cache optimized for its use case
    return &AppCaches{
        sessions: createCacheService[Session]("redis", redisConfig),
        users:    createCacheService[User]("memory", memoryConfig),
        files:    createCacheService[FileInfo]("s3", s3Config),
        config:   createCacheService[Config]("etcd", etcdConfig),
    }
}

func createCacheService[T any](provider string, config map[string]interface{}) cache.CacheService[T] {
    p, _ := cache.NewProvider(provider, config)
    contract := cache.NewContract[T](provider+"-cache", p)
    return contract.Cache()
}
```

### Dynamic Provider Selection
```go
// Runtime provider selection based on configuration
func CreateCacheFromConfig(config CacheConfig) (cache.CacheService[any], error) {
    provider, err := cache.NewProvider(config.Provider, config.Settings)
    if err != nil {
        return nil, err
    }
    
    contract := cache.NewContract[any](config.Name, provider)
    
    // Apply contract-level configuration
    if config.KeyPrefix != "" {
        contract.SetKeyPrefix(config.KeyPrefix)
    }
    
    if config.DefaultTTL > 0 {
        contract.SetDefaultTTL(config.DefaultTTL)
    }
    
    return contract.Cache(), nil
}

// Configuration-driven cache selection
cacheConfigs := []CacheConfig{
    {
        Name:     "user-cache",
        Provider: "redis",
        Settings: map[string]interface{}{
            "url": "redis://localhost:6379",
        },
        KeyPrefix:  "user:",
        DefaultTTL: 1 * time.Hour,
    },
    {
        Name:     "session-cache", 
        Provider: "memory",
        Settings: map[string]interface{}{
            "max_size": 100 * 1024 * 1024,
        },
        KeyPrefix:  "sess:",
        DefaultTTL: 24 * time.Hour,
    },
}

// Create caches from configuration
for _, config := range cacheConfigs {
    cache, err := CreateCacheFromConfig(config)
    if err != nil {
        log.Fatal(err)
    }
    registerCache(config.Name, cache)
}
```

### Advanced BYOC Patterns
```go
// Fallback provider chain
func NewFallbackCache() cache.CacheService[any] {
    // Try Redis first, fallback to memory
    primaryProvider, _ := cache.NewProvider("redis", redisConfig)
    fallbackProvider, _ := cache.NewProvider("memory", memoryConfig)
    
    provider := &FallbackProvider{
        primary:  primaryProvider,
        fallback: fallbackProvider,
    }
    
    contract := cache.NewContract[any]("fallback-cache", provider)
    return contract.Cache()
}

// Multi-region cache
func NewMultiRegionCache() cache.CacheService[any] {
    providers := []cache.CacheProvider{
        createRedisProvider("us-east-1"),
        createRedisProvider("us-west-2"), 
        createRedisProvider("eu-west-1"),
    }
    
    provider := &MultiRegionProvider{
        providers: providers,
        strategy:  "nearest", // or "broadcast", "primary-backup"
    }
    
    contract := cache.NewContract[any]("global-cache", provider)
    return contract.Cache()
}

// Cache with automatic encryption
func NewEncryptedCache(encryptionKey []byte) cache.CacheService[any] {
    baseProvider, _ := cache.NewProvider("redis", redisConfig)
    
    provider := &EncryptionProvider{
        underlying: baseProvider,
        cipher:     aes.NewCipher(encryptionKey),
    }
    
    contract := cache.NewContract[any]("encrypted-cache", provider)
    return contract.Cache()
}
```

### Legacy Migration Example
```go
// Gradual migration from old cache to new BYOC system
type MigrationCache struct {
    old  OldCacheInterface
    new  cache.CacheService[any]
    mode string // "old", "new", "dual"
}

func (m *MigrationCache) Get(ctx context.Context, key string) ([]byte, error) {
    switch m.mode {
    case "old":
        return m.old.Get(key)
    case "new":
        return m.new.Get(ctx, key)
    case "dual":
        // Try new first, fallback to old
        if data, err := m.new.Get(ctx, key); err == nil {
            return data, nil
        }
        return m.old.Get(key)
    }
    return nil, fmt.Errorf("unknown mode: %s", m.mode)
}

// Gradually shift traffic: old → dual → new
func (m *MigrationCache) SetMode(mode string) {
    m.mode = mode
}
```

### ZBZ Engine Integration
```go
// zbz engine configuration
type CacheContractConfig struct {
    Name     string                 `yaml:"name"`
    Provider string                 `yaml:"provider"`
    Config   map[string]interface{} `yaml:"config"`
    KeyPrefix string                `yaml:"key_prefix"`
    DefaultTTL string               `yaml:"default_ttl"`
}

// Engine integration
func (engine *Engine) RegisterCache(config CacheContractConfig) error {
    provider, err := cache.NewProvider(config.Provider, config.Config)
    if err != nil {
        return err
    }
    
    contract := cache.NewContract[any](config.Name, provider)
    
    // Configure contract
    if config.KeyPrefix != "" {
        contract.SetKeyPrefix(config.KeyPrefix)
    }
    
    if config.DefaultTTL != "" {
        ttl, _ := time.ParseDuration(config.DefaultTTL)
        contract.SetDefaultTTL(ttl)
    }
    
    // Register with engine
    return engine.RegisterService("cache", contract)
}
```

### Advanced Provider Features
```go
// Redis-specific advanced features
type RedisCache interface {
    CacheService[T]
    
    // Redis-specific operations
    Increment(ctx context.Context, key string, delta int64) (int64, error)
    Decrement(ctx context.Context, key string, delta int64) (int64, error)
    
    // List operations
    ListPush(ctx context.Context, key string, values ...interface{}) error
    ListPop(ctx context.Context, key string) ([]byte, error)
    ListRange(ctx context.Context, key string, start, stop int64) ([][]byte, error)
    
    // Set operations
    SetAdd(ctx context.Context, key string, members ...interface{}) error
    SetMembers(ctx context.Context, key string) ([][]byte, error)
    
    // Pub/Sub
    Subscribe(ctx context.Context, channels ...string) *redis.PubSub
    Publish(ctx context.Context, channel string, message interface{}) error
}

// Type assertion for advanced features
if redisCache, ok := service.(RedisCache); ok {
    // Use Redis-specific features
    count, err := redisCache.Increment(ctx, "counter:views", 1)
}
```

## Configuration System

### YAML Configuration
```yaml
# config.yaml
cache:
  providers:
    - name: "primary"
      provider: "redis"
      config:
        url: "redis://localhost:6379"
        pool_size: 20
        max_retries: 3
        read_timeout: "5s"
        write_timeout: "3s"
      key_prefix: "app:"
      default_ttl: "1h"
      
    - name: "session"
      provider: "redis"
      config:
        url: "redis://session-redis:6379/1"
        pool_size: 10
      key_prefix: "session:"
      default_ttl: "24h"
      
    - name: "local"
      provider: "memory"
      config:
        max_size: 52428800  # 50MB
        cleanup_interval: "2m"
        eviction_policy: "lru"
      default_ttl: "10m"
```

### Contract-Based Configuration
```go
// Direct contract creation
redisContract := cache.NewContract[User]("users", redisProvider).
    WithKeyPrefix("user:").
    WithDefaultTTL(2 * time.Hour).
    WithSerializer(cache.NewMsgPackSerializer()) // Custom serializer

memoryContract := cache.NewContract[Session]("sessions", memoryProvider).
    WithKeyPrefix("sess:").
    WithDefaultTTL(24 * time.Hour)

// Register with aliases for discovery
redisContract.Register("primary-cache")
memoryContract.Register("session-cache")
```

## Viability Assessment

### ✅ **High Viability Factors**

#### 1. **Excellent Pattern Consistency**
- **Score: 9/10** - Follows established zlog/hodor patterns perfectly
- Contract-provider separation maintains zbz architectural principles
- Generic contracts provide type safety while preserving flexibility
- Provider registration system enables extensibility

#### 2. **Simple, Clean API Surface**
- **Score: 9/10** - Cache operations are naturally simple
- Basic CRUD operations map cleanly to provider interface
- Batch operations add performance without complexity
- Type-safe struct operations eliminate marshaling boilerplate

#### 3. **Strong Foundation for Growth**
- **Score: 8/10** - Provider interface can accommodate advanced features
- Redis-specific operations can be exposed through type assertions
- Serializer interface allows custom encoding strategies
- Stats/monitoring integration points built in

#### 4. **Immediate ZBZ Integration Benefits**
- **Score: 8/10** - Fits naturally into zbz engine service registration
- Contract-based configuration aligns with existing patterns
- Can replace current basic cache system without breaking changes
- Provides foundation for auth caching, API response caching, etc.

### ⚠️ **Moderate Concerns**

#### 1. **Generic Contract Complexity**
- **Score: 6/10** - Generic contracts may be overkill for simple use cases
- Type assertion for provider-specific features could be confusing
- Serializer abstraction adds another layer to understand

#### 2. **Provider Feature Parity**
- **Score: 7/10** - Memory provider can't match Redis advanced features
- Batch operations on memory provider are just loops (no real batching)
- Stats collection overhead on high-frequency operations

#### 3. **Backwards Compatibility**
- **Score: 6/10** - Current cache.Cache interface would need deprecation
- Migration path needed for existing code
- Provider registration system is new concept

### 🚨 **Low Risk Factors**

#### 1. **Performance Overhead**
- **Score: 8/10** - Serialization overhead for struct operations manageable
- Provider abstraction is thin wrapper
- Generic type erasure at runtime minimizes overhead

#### 2. **Implementation Complexity**
- **Score: 7/10** - Provider implementations are straightforward
- Contract system follows established patterns
- Serializer interface has proven implementations (JSON, MessagePack)

### **Overall BYOC Viability: 9.5/10**

#### **BYOC Strengths Summary:**
1. **Perfect Pattern Fit**: Aligns with zbz architectural principles and BYOB philosophy
2. **Natural API**: Cache operations are inherently simple and well-understood
3. **Type Safety**: Generic contracts eliminate common marshaling errors
4. **Unlimited Extensibility**: Provider ecosystem supports ANY storage backend
5. **Performance Ready**: Batch operations and provider-specific optimizations
6. **Zero Vendor Lock-In**: Switch backends without code changes
7. **Environment Flexibility**: Dev (memory) → Staging (Redis) → Prod (DynamoDB)
8. **Cost Optimization**: Match backend to budget and performance requirements

#### **BYOC Implementation Recommendation:**
**Proceed with VERY high confidence** - this is potentially the **strongest BYOC implementation** in the entire zbz ecosystem.

### **Why BYOC Cache is Exceptionally Strong**

#### 1. **Adapter Ecosystem Viability: 10/10**
- **Natural Abstraction**: All cache backends share the same fundamental operations (get/set/delete/ttl)
- **Simple Interface**: Unlike databases with complex queries, cache operations are universally simple
- **Provider Proliferation**: Easy to add new adapters - each is ~100-200 lines
- **No Impedance Mismatch**: Cache semantics translate cleanly across all backends

#### 2. **Real-World Business Value: 9/10**
- **Cost Control**: Use cheap storage for non-critical caches
- **Compliance**: Air-gapped/on-premise solutions when required
- **Performance Tuning**: Match cache backend to specific workload
- **Operational Flexibility**: No cache vendor lock-in

#### 3. **Developer Experience: 9/10**  
- **Write Once, Run Anywhere**: Same code across all environments
- **Type Safety**: Struct scanning eliminates serialization errors
- **Configuration-Driven**: Change backends via config, not code
- **Backward Compatible**: Gradual migration from existing systems

### **Phase 1 Implementation Priority**
1. ✅ **Core Provider Interface** - Define standardized CacheProvider
2. ✅ **Generic Contract System** - Implement CacheContract[T] with type safety
3. ✅ **Redis Provider** - Enhanced with batch operations and stats
4. ✅ **Memory Provider** - With LRU eviction and size limits
5. ✅ **JSON Serializer** - Default struct marshaling
6. ✅ **Provider Registration** - Factory pattern for provider discovery

### **Phase 2 Extensions**
- **Advanced Redis Features** - Pub/sub, atomic operations, data structures
- **Additional Providers** - Memcached, DynamoDB, file-based
- **Custom Serializers** - MessagePack, Protocol Buffers
- **Monitoring Integration** - OpenTelemetry metrics and tracing
- **Distributed Patterns** - Cache warming, invalidation strategies

The BYOC cache system is perfectly positioned to be **the most successful adapter ecosystem** in the zbz framework due to its:

- **Universal Appeal**: Every application needs caching
- **Simple Abstraction**: Cache operations translate cleanly across ALL backends  
- **Zero Migration Pain**: Switch providers without code changes
- **Unlimited Growth**: Easy to add new adapters for any storage system
- **Business Value**: Real cost savings and vendor lock-in prevention

**This will be the flagship example of zbz's BYOB philosophy in action.**

---

*BYOC Cache: Write once, cache anywhere. The ultimate demonstration of zbz's modular, provider-agnostic architecture.*