# Cereal - Universal Serialization Service

> **A unified, type-safe serialization service with optional field-level scoping**

## Overview

Cereal is ZBZ's unified serialization service that consolidates all serialization needs across the core ZBZ services (zlog, hodor, flux, cache). It follows the established ZBZ architectural patterns with contracts, providers, and singleton management while providing optional scoped serialization and performance optimizations.

## Architecture Philosophy

**Why "Cereal"?** *Because serialization should be as simple as having breakfast - consistent, reliable, and exactly what you need to start your day.*

### Core Design Principles

1. **Universal Interface**: Single service for all serialization needs across ZBZ core services
2. **Type Safety**: Generic interfaces with compile-time guarantees
3. **Optional Scoping**: Field-level permissions applied only when scope tags exist
4. **Format Agnostic**: Built-in support for common formats + external library providers
5. **Performance First**: Optimized for high-frequency cache and configuration operations
6. **Zero Dependencies**: Core functionality works without external libraries

### Target Integration (Core ZBZ Services Only)

**Primary Targets:**
- ✅ **Cache**: Replace existing type-based serialization system
- ✅ **Flux**: Configuration serialization and hot-reload
- ✅ **Zlog**: Field preprocessing and structured data serialization
- ✅ **Hodor**: Metadata and object serialization

**Not Targeting:**
- ❌ Legacy API layer (`api/` package)
- ❌ Legacy HTTP handlers
- ❌ Deprecated services

## Service Architecture

### Core Interface Design

```go
// CerealProvider defines the unified interface that all serialization providers implement
type CerealProvider interface {
    // Core serialization operations
    Marshal(data any) ([]byte, error)
    Unmarshal(data []byte, target any) error
    
    // Scoped serialization (no-op if no scope tags exist)
    MarshalScoped(data any, userPermissions []string) ([]byte, error)
    UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error
    
    // Format metadata
    ContentType() string
    Format() string
    
    // Performance and capabilities
    SupportsBinaryData() bool
    SupportsStreaming() bool
    
    // Provider lifecycle
    Close() error
}
```

### Universal Configuration

```go
// CerealConfig defines provider-agnostic cereal configuration
type CerealConfig struct {
    // Service configuration
    Name           string `yaml:"name" json:"name"`
    DefaultFormat  string `yaml:"default_format" json:"default_format"`     // "json", "protobuf", "msgpack"
    
    // Performance settings
    EnableCaching  bool   `yaml:"enable_caching" json:"enable_caching"`     // Cache serialization results
    CacheSize      int    `yaml:"cache_size" json:"cache_size"`             // Max cache entries
    CompressAbove  int    `yaml:"compress_above" json:"compress_above"`     // Compress data above N bytes
    
    // Type-based format selection
    TypeFormats    map[string]string `yaml:"type_formats,omitempty" json:"type_formats,omitempty"`
    
    // Scoped serialization settings
    EnableScoping  bool   `yaml:"enable_scoping" json:"enable_scoping"`     // Enable field-level permissions
    DefaultScope   string `yaml:"default_scope" json:"default_scope"`       // Default permission scope
    StrictMode     bool   `yaml:"strict_mode" json:"strict_mode"`           // Fail on permission violations
    
    // Format-specific settings (each provider uses what it needs)
    JSONSettings   *JSONConfig      `yaml:"json,omitempty" json:"json,omitempty"`
    ProtobufPaths  []string         `yaml:"protobuf_paths,omitempty" json:"protobuf_paths,omitempty"`
    Compression    *CompressionConfig `yaml:"compression,omitempty" json:"compression,omitempty"`
    
    // Provider-specific extensions
    Extensions     map[string]interface{} `yaml:"extensions,omitempty" json:"extensions,omitempty"`
}

// JSONConfig provides JSON-specific configuration
type JSONConfig struct {
    Indent         string `yaml:"indent,omitempty" json:"indent,omitempty"`
    HTMLEscape     bool   `yaml:"html_escape" json:"html_escape"`
    ValidateUTF8   bool   `yaml:"validate_utf8" json:"validate_utf8"`
}

// CompressionConfig configures compression providers
type CompressionConfig struct {
    Algorithm      string `yaml:"algorithm" json:"algorithm"`               // "gzip", "lz4", "zstd"
    Level          int    `yaml:"level" json:"level"`                       // Compression level
    Threshold      int    `yaml:"threshold" json:"threshold"`               // Min bytes to compress
}
```

### Contract System

```go
// CerealContract provides type-safe access to native serialization clients
type CerealContract[T any] struct {
    name     string
    provider CerealProvider
    native   T                    // The typed native client (e.g., *json.Encoder)
    config   CerealConfig
}

// Contract creation and registration
func NewContract[T any](name string, provider CerealProvider, native T, config CerealConfig) *CerealContract[T]
func (c *CerealContract[T]) Register() error
func (c *CerealContract[T]) Native() T
```

### Singleton Service

```go
// zCereal is the singleton service that orchestrates serialization operations
type zCereal struct {
    provider       CerealProvider    // Backend provider wrapper
    config         CerealConfig      // Service configuration  
    contractName   string            // Name of the contract that created this singleton
    cache          *SerializationCache // Optional caching layer
    typeRegistry   *TypeFormatRegistry // Type-based format selection
}

// Package-level functions (singleton delegation)
func Marshal(data any) ([]byte, error)
func Unmarshal(data []byte, target any) error
func MarshalScoped(data any, userPermissions []string) ([]byte, error)
func UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error
```

## Provider Implementation Plan

### 🎯 **Built-in Providers (Zero Dependencies)**

#### **1. JSON Provider** (`cereal/json.go`)
- **Purpose**: Replace existing JSON serialization across ZBZ core services
- **Features**:
  - High-performance JSON marshaling using standard library
  - Optional scoped field filtering (only when scope tags exist)
  - Custom JSON tag support (`json:"name,omitempty" scope:"read:admin"`)
  - Pretty-printing for development configurations
- **Native Type**: `*json.Encoder` / `*json.Decoder`
- **Dependencies**: Standard library only

#### **2. Raw Provider** (`cereal/raw.go`)
- **Purpose**: Pass-through for []byte data (cache compatibility)
- **Features**:
  - Zero-copy operations for byte slices
  - Content-type detection for binary data
  - No scoping (raw bytes have no fields)
- **Native Type**: `[]byte`
- **Dependencies**: Standard library only

#### **3. String Provider** (`cereal/string.go`)
- **Purpose**: String serialization (cache compatibility)
- **Features**:
  - UTF-8 validation and encoding
  - No scoping (strings have no fields)
  - Text template integration for configuration
- **Native Type**: `string`
- **Dependencies**: Standard library only

#### **4. GOB Provider** (`cereal/gob.go`)
- **Purpose**: Native Go binary serialization
- **Features**:
  - High-performance binary encoding
  - Type-safe with Go type system
  - Schema evolution with Go versioning
- **Native Type**: `*gob.Encoder` / `*gob.Decoder`
- **Dependencies**: Standard library only

### 🚀 **External Library Providers (Separate Packages)**

#### **5. MessagePack Provider** (`providers/cereal-msgpack/`)
- **Purpose**: Compact binary format for high-frequency cache operations
- **Features**:
  - Faster than JSON for cache operations
  - Smaller payload size
  - Schema evolution support
- **Native Type**: `*msgpack.Encoder`
- **Dependencies**: `github.com/vmihailenco/msgpack/v5`

#### **6. Protobuf Provider** (`providers/cereal-protobuf/`)
- **Purpose**: High-performance binary serialization for inter-service communication
- **Features**:
  - Dynamic protobuf with reflection
  - Type-safe message generation
  - Schema evolution with protobuf compatibility
- **Native Type**: `*protoreflect.Message`
- **Dependencies**: `google.golang.org/protobuf`

#### **7. YAML Provider** (`providers/cereal-yaml/`)
- **Purpose**: Configuration files and human-readable data (flux integration)
- **Features**:
  - Multi-document support for flux configurations
  - Comment preservation
  - Schema validation
- **Native Type**: `*yaml.Encoder`
- **Dependencies**: `gopkg.in/yaml.v3`

### ⚡ **Performance Enhancement Providers**

#### **8. Compression Provider** (`cereal/compression.go`)
- **Purpose**: Wrap other providers with compression (built-in gzip)
- **Features**:
  - Standard library gzip compression
  - Automatic threshold-based compression
  - Transparent decompression
- **Native Type**: `*gzip.Writer`
- **Dependencies**: Standard library only

#### **9. Cached Provider** (`cereal/cached.go`)
- **Purpose**: Memoization for expensive serialization operations
- **Features**:
  - LRU cache for serialization results
  - Hash-based cache keys
  - TTL-based invalidation
- **Native Type**: Internal cache structure
- **Dependencies**: Standard library only

## Example Usage Patterns

### Core ZBZ Service Integration

```go
// Built-in JSON provider (no external dependencies)
func setupCoreServices() {
    config := cereal.CerealConfig{
        Name:          "core-serializer",
        DefaultFormat: "json",
        EnableCaching: true,
    }
    
    // Built-in JSON provider
    jsonContract := cereal.NewJSONProvider(config)
    jsonContract.Register()
    
    // For high-performance cache operations
    msgpackContract, _ := cerealmsgpack.NewMessagePackProvider(config)
    
    // Type-safe native access
    jsonEncoder := jsonContract.Native() // *json.Encoder
}
```

### Optional Scoped Serialization

```go
// Scoping only applied when scope tags exist
type User struct {
    ID       int    `json:"id"`                                    // No scope = always visible
    Name     string `json:"name"`                                  // No scope = always visible  
    Email    string `json:"email" scope:"read:user,write:admin"`   // Scoped field
    Internal string `json:"internal" scope:"read:admin"`           // Admin-only field
}

type Product struct {
    ID    int     `json:"id"`     // No scope tags anywhere
    Name  string  `json:"name"`   // = no scoping applied
    Price float64 `json:"price"`  // = standard serialization
}

// Scoped serialization when scope tags exist
data, _ := cereal.MarshalScoped(user, []string{"user"}) 
// Result: {"id": 1, "name": "John", "email": "john@example.com"} (internal filtered)

// Standard serialization when no scope tags
data, _ := cereal.Marshal(product) 
// Result: {"id": 1, "name": "Widget", "price": 9.99} (no filtering)
```

### Cache Service Integration

```go
// Cache automatically uses cereal for serialization
type CacheData struct {
    Value     string    `json:"value"`
    Timestamp time.Time `json:"timestamp"`
}

// Cache uses cereal under the hood - seamless replacement
cache.Set("key", data, time.Hour)    // Uses cereal JSON provider
var retrieved CacheData
cache.Get("key", &retrieved)         // Uses cereal JSON provider

// High-performance binary serialization for cache
cache.SetProvider("msgpack")
cache.Set("key", data, time.Hour)    // Uses MessagePack provider
```

### Flux Configuration Integration

```go
// YAML provider for flux configurations
fluxConfig := cereal.CerealConfig{
    Name: "flux-serializer", 
    DefaultFormat: "yaml",
}

yamlContract, _ := cerealyaml.NewYAMLProvider(fluxConfig)

// Flux uses cereal for configuration serialization
flux.RegisterConfig("database", func(data []byte) error {
    var dbConfig DatabaseConfig
    return cereal.Unmarshal(data, &dbConfig) // Auto-detects YAML format
})
```

### Zlog Field Processing Integration

```go
// Cereal for structured data in log fields
type LogEvent struct {
    UserID    int    `json:"user_id"`
    Action    string `json:"action"`
    Sensitive string `json:"sensitive" scope:"read:admin"` // Filtered in logs
}

// Zlog can use cereal for field serialization
event := LogEvent{UserID: 123, Action: "login", Sensitive: "secret"}
serialized, _ := cereal.MarshalScoped(event, []string{"public"})

zlog.Info("User action", zlog.Any("event", serialized))
// Logs: {"user_id": 123, "action": "login"} (sensitive filtered)
```

### Hodor Metadata Serialization

```go
// Hodor can use cereal for object metadata
type ObjectMetadata struct {
    ContentType string    `json:"content_type"`
    Owner       string    `json:"owner"`
    Private     string    `json:"private" scope:"read:owner"`
}

// Hodor stores serialized metadata with objects
metadata := ObjectMetadata{ContentType: "image/png", Owner: "user123", Private: "internal"}
serializedMeta, _ := cereal.Marshal(metadata)
hodor.SetWithMetadata("image.png", imageData, serializedMeta)
```

## Migration Strategy (Core Services Only)

### Phase 1: Cache Service Integration
1. **Replace SerializerManager**: Update cache to use cereal providers instead of current serialization system
2. **Maintain API compatibility**: Keep existing cache.Set/Get signatures
3. **Performance validation**: Ensure cereal matches or exceeds current cache serialization performance

### Phase 2: Flux Configuration Integration  
1. **YAML provider**: Add YAML support for flux configuration files
2. **Hot-reload compatibility**: Ensure cereal works with flux's configuration watching
3. **Format auto-detection**: Allow flux to handle multiple config formats seamlessly

### Phase 3: Zlog Integration
1. **Field serialization**: Use cereal for complex log field serialization
2. **Scoped logging**: Optional field filtering for sensitive data in logs
3. **Performance optimization**: Ensure no impact on high-frequency logging

### Phase 4: Hodor Integration
1. **Metadata serialization**: Use cereal for object metadata
2. **Binary format support**: Ensure efficient serialization for storage operations
3. **Cross-provider compatibility**: Ensure metadata works across all hodor providers

## Performance Optimizations

### 1. **Reflection Caching**
- Cache struct field metadata to avoid repeated reflection
- Pre-compute scope mappings for known types
- JIT compilation of field access validators

### 2. **Serialization Memoization**
- LRU cache for expensive serialization operations
- Hash-based cache keys with content and permission fingerprints
- TTL-based invalidation for dynamic data

### 3. **Type-Based Routing**
- Compile-time format selection where possible
- Type registry for O(1) provider lookup
- Interface specialization for known types

### 4. **Streaming & Compression**
- Automatic compression for large payloads
- Streaming support for memory-efficient large data processing
- Async serialization with worker pools

## Migration Strategy

### Phase 1: Cache Service Migration
1. Update cache serialization to use cereal providers
2. Maintain backward compatibility with existing cache API
3. Add performance benchmarks and monitoring

### Phase 2: HTTP Response Migration  
1. Replace manual JSON marshaling with scoped cereal
2. Update error handling middleware
3. Add permission integration with auth service

### Phase 3: Configuration Migration
1. Migrate YAML config handling to cereal
2. Add validation and schema support
3. Integrate with flux for hot-reload

### Phase 4: Performance Optimization
1. Add caching and memoization
2. Implement compression providers
3. Add streaming support for large datasets

## Success Metrics

- **Performance**: 50% reduction in serialization overhead
- **Security**: 100% field-level permission coverage
- **Maintainability**: Single service for all serialization needs
- **Flexibility**: Support for 6+ serialization formats
- **Backward Compatibility**: Zero breaking changes to existing APIs

---

*Cereal: Making serialization a consistent, secure, and performant experience across the ZBZ ecosystem.*