# Cereal Document Merge Implementation Plan

## Executive Summary

Extend the cereal serialization library to support intelligent document merging, where default documents can be merged with user-provided data so users only need to override specific fields. This feature pairs perfectly with the flux layered fallback architecture to enable reactive configuration systems with sensible defaults.

## Architecture Overview

### **Core Concept**
```go
// Default configuration (framework provides)
defaultConfig := AppConfig{
    Database: DatabaseConfig{
        MaxConnections: 10,
        Timeout:        30 * time.Second,
        LogQueries:     false,
    },
    Cache: CacheConfig{
        TTL:     1 * time.Hour,
        MaxSize: 100 * 1024 * 1024,
    },
    Features: []string{"auth", "logging"},
}

// User override (user provides partial config)
userOverride := AppConfig{
    Database: DatabaseConfig{
        MaxConnections: 100,  // Override
        // Timeout: inherits 30s from default
        // LogQueries: inherits false from default
    },
    // Cache: inherits entire cache config from default
    Features: []string{"auth", "logging", "analytics"},  // Override array
}

// Merged result (cereal handles automatically)
merged := cereal.Merge(defaultConfig, userOverride)
// Result: MaxConnections=100, Timeout=30s, LogQueries=false, Features=["auth","logging","analytics"]
```

## Core Merge API

### **1. Basic Merge Operations**
```go
// Merge with default strategies
func Merge[T any](defaults T, overrides T) T

// Merge with custom strategies
func MergeWithOptions[T any](defaults T, overrides T, options MergeOptions) T

// Merge multiple sources (left-to-right priority)
func MergeMultiple[T any](sources ...T) T

// Merge with validation
func MergeAndValidate[T any](defaults T, overrides T, validator func(T) error) (T, error)
```

### **2. Flux-Reactive Merge**
```go
// Reactive merge that updates when source documents change
func ReactiveMerge[T any](ctx context.Context, sources ...MergeSource[T]) (<-chan T, error)

type MergeSource[T any] struct {
    URI      string        // "bucket://defaults/config.yaml", "bucket://user/config.yaml"
    Priority int           // Lower number = higher priority
    Watch    bool          // Watch for changes via flux
}

// Usage example
configSources := []MergeSource[AppConfig]{
    {URI: "bucket://user-config/app.yaml", Priority: 1, Watch: true},
    {URI: "bucket://env-config/app.yaml", Priority: 2, Watch: true},
    {URI: "bucket://zbz-defaults/app.yaml", Priority: 3, Watch: true},
}

configChan, err := cereal.ReactiveMerge(ctx, configSources...)
for mergedConfig := range configChan {
    // Application config updated automatically when any source changes
    app.UpdateConfig(mergedConfig)
}
```

### **3. MergeOptions Configuration**
```go
type MergeOptions struct {
    ArrayStrategy  ArrayMergeStrategy  // How to handle array fields
    StructStrategy StructMergeStrategy // How to handle nested structs
    MapStrategy    MapMergeStrategy    // How to handle map fields
    NilStrategy    NilMergeStrategy    // How to handle nil/zero values
    FieldRules     map[string]FieldMergeRule // Per-field merge rules
}

type ArrayMergeStrategy int
const (
    ArrayReplace ArrayMergeStrategy = iota // Replace entire array
    ArrayAppend                            // Append to default array
    ArrayMerge                             // Merge arrays by index
    ArrayUnion                             // Union of both arrays (unique items)
)

type StructMergeStrategy int
const (
    StructDeepMerge StructMergeStrategy = iota // Recursively merge struct fields
    StructReplace                              // Replace entire struct
    StructSkipZero                             // Skip zero-value structs
)

type FieldMergeRule struct {
    Strategy    MergeStrategy
    Transformer func(any, any) any  // Custom merge logic for this field
    Required    bool                 // Field must be present in result
}
```

## Implementation Architecture

### **1. Core Merge Engine**
```go
type MergeEngine struct {
    options MergeOptions
    cache   map[reflect.Type]*TypeInfo  // Cache type reflection data
}

func NewMergeEngine(options MergeOptions) *MergeEngine {
    return &MergeEngine{
        options: options,
        cache:   make(map[reflect.Type]*TypeInfo),
    }
}

func (me *MergeEngine) Merge(defaults, overrides any) any {
    defaultVal := reflect.ValueOf(defaults)
    overrideVal := reflect.ValueOf(overrides)
    
    return me.mergeValues(defaultVal, overrideVal).Interface()
}

func (me *MergeEngine) mergeValues(defaults, overrides reflect.Value) reflect.Value {
    // Handle different types: struct, map, slice, primitive
    switch defaults.Kind() {
    case reflect.Struct:
        return me.mergeStructs(defaults, overrides)
    case reflect.Map:
        return me.mergeMaps(defaults, overrides)
    case reflect.Slice:
        return me.mergeSlices(defaults, overrides)
    case reflect.Ptr:
        return me.mergePointers(defaults, overrides)
    default:
        return me.mergePrimitives(defaults, overrides)
    }
}
```

### **2. Struct Merging with Tag Support**
```go
func (me *MergeEngine) mergeStructs(defaults, overrides reflect.Value) reflect.Value {
    result := reflect.New(defaults.Type()).Elem()
    
    for i := 0; i < defaults.NumField(); i++ {
        field := defaults.Type().Field(i)
        defaultField := defaults.Field(i)
        overrideField := overrides.Field(i)
        
        // Check merge tag for field-specific rules
        mergeTag := field.Tag.Get("merge")
        rule := me.parseFieldRule(mergeTag)
        
        var mergedField reflect.Value
        switch rule.Strategy {
        case MergeReplace:
            if !overrideField.IsZero() {
                mergedField = overrideField
            } else {
                mergedField = defaultField
            }
        case MergeDeep:
            mergedField = me.mergeValues(defaultField, overrideField)
        case MergeSkip:
            mergedField = defaultField
        case MergeCustom:
            if rule.Transformer != nil {
                mergedVal := rule.Transformer(defaultField.Interface(), overrideField.Interface())
                mergedField = reflect.ValueOf(mergedVal)
            }
        }
        
        result.Field(i).Set(mergedField)
    }
    
    return result
}

// Struct tag examples:
type AppConfig struct {
    Database DatabaseConfig `merge:"deep"`                    // Deep merge nested struct
    Features []string       `merge:"union"`                   // Union arrays
    Debug    bool           `merge:"replace"`                 // Replace if not zero
    Secrets  map[string]string `merge:"skip"`                // Never merge secrets
    Version  string          `merge:"custom:versionMerger"`   // Custom merge function
}
```

### **3. Array Merging Strategies**
```go
func (me *MergeEngine) mergeSlices(defaults, overrides reflect.Value) reflect.Value {
    if overrides.IsZero() {
        return defaults
    }
    
    switch me.options.ArrayStrategy {
    case ArrayReplace:
        return overrides
        
    case ArrayAppend:
        result := reflect.MakeSlice(defaults.Type(), 0, defaults.Len()+overrides.Len())
        result = reflect.AppendSlice(result, defaults)
        result = reflect.AppendSlice(result, overrides)
        return result
        
    case ArrayMerge:
        return me.mergeSlicesByIndex(defaults, overrides)
        
    case ArrayUnion:
        return me.mergeSlicesUnion(defaults, overrides)
    }
    
    return overrides
}

func (me *MergeEngine) mergeSlicesUnion(defaults, overrides reflect.Value) reflect.Value {
    seen := make(map[any]bool)
    result := reflect.MakeSlice(defaults.Type(), 0, defaults.Len()+overrides.Len())
    
    // Add defaults
    for i := 0; i < defaults.Len(); i++ {
        item := defaults.Index(i)
        key := item.Interface()
        if !seen[key] {
            result = reflect.Append(result, item)
            seen[key] = true
        }
    }
    
    // Add overrides (skip duplicates)
    for i := 0; i < overrides.Len(); i++ {
        item := overrides.Index(i)
        key := item.Interface()
        if !seen[key] {
            result = reflect.Append(result, item)
            seen[key] = true
        }
    }
    
    return result
}
```

### **4. Map Merging**
```go
func (me *MergeEngine) mergeMaps(defaults, overrides reflect.Value) reflect.Value {
    if overrides.IsZero() {
        return defaults
    }
    
    result := reflect.MakeMap(defaults.Type())
    
    // Copy defaults
    for _, key := range defaults.MapKeys() {
        result.SetMapIndex(key, defaults.MapIndex(key))
    }
    
    // Merge overrides
    for _, key := range overrides.MapKeys() {
        overrideVal := overrides.MapIndex(key)
        
        if defaultVal := defaults.MapIndex(key); defaultVal.IsValid() {
            // Key exists in both - merge values
            if me.options.MapStrategy == MapDeepMerge && !isPrimitive(overrideVal.Type()) {
                mergedVal := me.mergeValues(defaultVal, overrideVal)
                result.SetMapIndex(key, mergedVal)
            } else {
                result.SetMapIndex(key, overrideVal)
            }
        } else {
            // Key only in override - add directly
            result.SetMapIndex(key, overrideVal)
        }
    }
    
    return result
}
```

## Flux Integration

### **1. Reactive Configuration System**
```go
type ReactiveConfig[T any] struct {
    sources []MergeSource[T]
    engine  *MergeEngine
    current T
    mutex   sync.RWMutex
    subscribers []chan T
}

func NewReactiveConfig[T any](sources []MergeSource[T], options MergeOptions) *ReactiveConfig[T] {
    rc := &ReactiveConfig[T]{
        sources: sources,
        engine:  NewMergeEngine(options),
    }
    
    // Start watching sources
    rc.startWatching()
    
    return rc
}

func (rc *ReactiveConfig[T]) startWatching() {
    for _, source := range rc.sources {
        if source.Watch {
            go rc.watchSource(source)
        }
    }
    
    // Initial merge
    rc.refresh()
}

func (rc *ReactiveConfig[T]) watchSource(source MergeSource[T]) {
    // Subscribe to flux changes
    depot.Subscribe(source.URI, func(event ChangeEvent) {
        // Re-merge when source changes
        rc.refresh()
    })
}

func (rc *ReactiveConfig[T]) refresh() {
    var documents []T
    
    // Load all sources in priority order
    for _, source := range rc.sortedSources() {
        if doc, err := rc.loadSource(source); err == nil {
            documents = append(documents, doc)
        }
    }
    
    // Merge all documents
    var merged T
    if len(documents) > 0 {
        merged = documents[0]
        for i := 1; i < len(documents); i++ {
            merged = rc.engine.Merge(merged, documents[i]).(T)
        }
    }
    
    // Update current config and notify subscribers
    rc.mutex.Lock()
    rc.current = merged
    rc.mutex.Unlock()
    
    rc.notifySubscribers(merged)
}
```

### **2. Configuration Management Example**
```go
// Application startup
func InitializeConfiguration() {
    sources := []MergeSource[AppConfig]{
        {URI: "bucket://user-config/app.yaml", Priority: 1, Watch: true},
        {URI: "bucket://env-config/prod.yaml", Priority: 2, Watch: true},
        {URI: "bucket://zbz-defaults/app.yaml", Priority: 3, Watch: true},
    }
    
    options := MergeOptions{
        ArrayStrategy:  ArrayUnion,
        StructStrategy: StructDeepMerge,
        FieldRules: map[string]FieldMergeRule{
            "Features": {Strategy: MergeUnion},
            "Secrets":  {Strategy: MergeSkip},
        },
    }
    
    configReactive := cereal.NewReactiveConfig(sources, options)
    
    // Listen for configuration changes
    configReactive.Subscribe(func(config AppConfig) {
        // Update application configuration
        database.UpdateConfig(config.Database)
        cache.UpdateConfig(config.Cache)
        logger.UpdateConfig(config.Logging)
        
        log.Info("Configuration updated",
            zap.Int("max_connections", config.Database.MaxConnections),
            zap.Duration("cache_ttl", config.Cache.TTL))
    })
}
```

## Advanced Features

### **1. Conditional Merging**
```go
type ConditionalMerge struct {
    Condition func(defaults, overrides any) bool
    Strategy  MergeStrategy
}

// Only merge if condition is met
options := MergeOptions{
    ConditionalRules: map[string]ConditionalMerge{
        "DatabaseConfig": {
            Condition: func(defaults, overrides any) bool {
                override := overrides.(DatabaseConfig)
                return override.MaxConnections > 0  // Only merge if valid
            },
            Strategy: MergeDeep,
        },
    },
}
```

### **2. Merge Validation**
```go
func MergeWithValidation[T any](defaults T, overrides T, validators ...func(T) error) (T, error) {
    merged := Merge(defaults, overrides)
    
    for _, validator := range validators {
        if err := validator(merged); err != nil {
            return defaults, fmt.Errorf("merge validation failed: %w", err)
        }
    }
    
    return merged, nil
}

// Usage
validator := func(config AppConfig) error {
    if config.Database.MaxConnections <= 0 {
        return fmt.Errorf("max_connections must be positive")
    }
    if config.Cache.TTL <= 0 {
        return fmt.Errorf("cache TTL must be positive")
    }
    return nil
}

merged, err := cereal.MergeWithValidation(defaults, overrides, validator)
```

### **3. Merge History and Rollback**
```go
type MergeHistory[T any] struct {
    versions []MergeVersion[T]
    current  int
}

type MergeVersion[T any] struct {
    Data      T
    Sources   []string
    Timestamp time.Time
    Hash      string
}

func (mh *MergeHistory[T]) AddVersion(data T, sources []string) {
    version := MergeVersion[T]{
        Data:      data,
        Sources:   sources,
        Timestamp: time.Now(),
        Hash:      computeHash(data),
    }
    
    mh.versions = append(mh.versions, version)
    mh.current = len(mh.versions) - 1
}

func (mh *MergeHistory[T]) Rollback(steps int) T {
    if mh.current-steps >= 0 {
        mh.current -= steps
        return mh.versions[mh.current].Data
    }
    return mh.versions[0].Data
}
```

## Real-World Use Cases

### **1. Multi-Environment Configuration**
```yaml
# zbz-defaults/app.yaml (framework defaults)
database:
  max_connections: 10
  timeout: 30s
  ssl_mode: prefer

cache:
  ttl: 1h
  max_size: 100MB

features:
  - auth
  - logging

# env-config/production.yaml (environment overrides)
database:
  max_connections: 100
  ssl_mode: require

cache:
  max_size: 1GB

# user-config/app.yaml (user overrides)
database:
  timeout: 60s

features:
  - auth
  - logging
  - analytics
  - monitoring

# Merged result:
# database: {max_connections: 100, timeout: 60s, ssl_mode: require}
# cache: {ttl: 1h, max_size: 1GB}
# features: [auth, logging, analytics, monitoring]
```

### **2. API Response Templating**
```go
// Default API response template
defaultResponse := APIResponse{
    Status: "success",
    Data:   nil,
    Meta: Meta{
        Timestamp: time.Now(),
        Version:   "1.0",
        RequestID: generateRequestID(),
    },
    Links: map[string]string{
        "self": "/api/v1/resource",
    },
}

// User-specific response data
userResponse := APIResponse{
    Data: userData,
    Meta: Meta{
        UserID: "123",
    },
    Links: map[string]string{
        "profile": "/api/v1/users/123",
        "preferences": "/api/v1/users/123/preferences",
    },
}

// Merged response
response := cereal.Merge(defaultResponse, userResponse)
// Result includes all default fields plus user-specific data
```

### **3. Feature Flag Configuration**
```go
// Default feature flags
defaultFlags := FeatureFlags{
    Features: map[string]bool{
        "new_ui":           false,
        "advanced_search":  false,
        "beta_features":    false,
    },
    UserGroups: map[string][]string{
        "beta_users": {},
        "admin_users": {"admin", "super_admin"},
    },
}

// User-specific overrides
userFlags := FeatureFlags{
    Features: map[string]bool{
        "new_ui": true,  // Enable for this user
    },
    UserGroups: map[string][]string{
        "beta_users": {"user123"},  // Add user to beta group
    },
}

// Merged flags with smart array handling
merged := cereal.MergeWithOptions(defaultFlags, userFlags, MergeOptions{
    MapStrategy: MapDeepMerge,
    ArrayStrategy: ArrayUnion,  // Union user groups
})
```

## Implementation Phases

### **Phase 1: Core Merge Engine (Week 1)**
- Basic merge operations for structs, maps, slices
- MergeOptions configuration system
- Struct tag support for field-level rules
- Unit tests for all merge strategies

### **Phase 2: Flux Integration (Week 1)**
- ReactiveConfig implementation
- Multi-source merging with priority
- Change watching and automatic re-merging
- Integration tests with depot/flux

### **Phase 3: Advanced Features (Week 1)**
- Conditional merging
- Merge validation
- Custom field transformers
- Performance optimization

### **Phase 4: Production Features (Week 1)**
- Merge history and rollback
- Configuration versioning
- Monitoring and metrics
- Documentation and examples

## Success Metrics

### **Technical Metrics**
- [ ] Merge operations complete in < 1ms for typical configs
- [ ] Memory usage scales linearly with document size
- [ ] Reactive updates trigger within 100ms of source changes
- [ ] Zero memory leaks in long-running reactive configs

### **Developer Experience Metrics**
- [ ] Zero-config merging works for 90% of use cases
- [ ] Configuration-driven merging handles complex scenarios
- [ ] Clear error messages for merge validation failures
- [ ] Documentation covers all common patterns

### **Production Readiness Metrics**
- [ ] Reactive configurations handle source failures gracefully
- [ ] Merge validation prevents invalid configurations
- [ ] Configuration rollback works reliably
- [ ] Monitoring provides visibility into merge operations

## Future Enhancements

### **1. Schema-Based Merging**
```go
// Use JSON Schema to guide merge operations
schema := loadJSONSchema("config-schema.json")
merged := cereal.MergeWithSchema(defaults, overrides, schema)
```

### **2. Conflict Resolution UI**
```go
// Interactive conflict resolution for complex merges
conflicts := cereal.DetectConflicts(defaults, overrides)
resolver := NewConflictResolver(conflicts)
resolved := resolver.ResolveInteractively()
```

### **3. Merge Optimization**
```go
// Optimize merging for large documents
options := MergeOptions{
    LazyMerging: true,     // Only merge accessed fields
    Caching:     true,     // Cache merge results
    Incremental: true,     // Only re-merge changed sections
}
```

## Conclusion

The Cereal Document Merge feature transforms configuration management and data composition in the ZBZ framework. Combined with the flux layered fallback architecture, it enables sophisticated configuration systems where users get sensible defaults but can override any aspect of the system's behavior.

This implementation provides the foundation for:
- **Zero-config applications** that work immediately
- **Sophisticated configuration** through layered overrides
- **Real-time configuration updates** via flux integration
- **Complex data composition** with intelligent merging strategies

The merge system is designed to be both simple for basic use cases and powerful enough for the most complex configuration scenarios, making it an essential component of the ZBZ framework's configuration and data management capabilities.