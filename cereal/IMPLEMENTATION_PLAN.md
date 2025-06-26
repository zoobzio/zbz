# Cereal V2 Implementation Plan

> **Assessment & Roadmap for Production-Ready Universal Serialization Service**

## Executive Summary

Cereal V1 successfully demonstrates unified serialization architecture with innovative scoped field-level permissions. The foundational integration approach is correct - services depend on cereal as core infrastructure rather than connecting as separate services. However, several critical issues prevent production deployment.

**Current Status: B+ (Good foundation, needs refinement)**

**Recommendation: Iterate, don't rebuild** - the core architecture is sound and aligns perfectly with ZBZ's service patterns.

## V1 Assessment

### ✅ **Architectural Strengths**

#### **Perfect ZBZ Ecosystem Integration**
- Contract/provider/singleton pattern matches cache/zlog/hodor exactly
- Type-safe generics eliminate casting anti-patterns  
- Universal configuration approach consistent across services
- Truly foundational - services depend on cereal internally, not as glue code

#### **Innovative Scoped Serialization**
- Unique field-level permission system: `scope:"read:admin,write:admin"`
- Optional by design - only applied when scope tags exist
- Reflection caching for performance optimization
- Solves real security problems for sensitive data

#### **Successful Service Integration**
- **Cache**: Replaced SerializerManager, all table operations use cereal automatically
- **Flux**: All configuration parsing uses cereal providers consistently  
- **Zlog**: Complex field serialization integrated into processing pipeline
- **Hodor**: Metadata operations with scoped access control
- **Zero breaking changes** to existing APIs

### ❌ **Critical Production Issues**

#### **1. Circular Import Hell (Priority: P0)**
```
zbz/cache -> zbz/cereal -> zbz/cache (via examples)
zbz/flux  -> zbz/cereal -> zbz/flux  (via adapters)
```
**Impact**: Won't compile in real Go projects
**Root Cause**: Cereal package has dependencies on services that depend on it

#### **2. Performance Anti-patterns (Priority: P0)**
```go
// Problem: Creating new contracts for every operation
func parseJSON[T any](content []byte) (any, error) {
    config := cereal.DefaultConfig()
    contract := cereal.NewJSONProvider(config)  // NEW CONTRACT EVERY TIME
    return contract.Unmarshal(content, &result)
}
```
**Impact**: Massive performance overhead
**Root Cause**: No singleton contract reuse pattern

#### **3. Configuration Verbosity (Priority: P1)**
```go
// Every service recreates the same setup
cerealConfig := cereal.DefaultConfig()
cerealConfig.Name = "cache-serializer"  
cerealConfig.DefaultFormat = "json"
cerealConfig.EnableCaching = true
cerealConfig.EnableScoping = false
// ... 10+ lines of boilerplate
```
**Impact**: Poor developer experience
**Root Cause**: No auto-configuration or shared singletons

#### **4. Unclear Dependency Injection (Priority: P1)**
```go
// Confusing provider extraction pattern
cerealProvider := cerealContract.Provider() // Manual extraction
cache.serializer = cerealProvider          // Manual assignment
```
**Impact**: Services don't know how to get cereal properly
**Root Cause**: No established dependency injection pattern

## V2 Implementation Plan

### **Phase 1: Foundation Fixes (Week 1-2)**

#### **P0: Resolve Circular Imports**

**Goal**: Make cereal completely dependency-free

**Approach**: Move cereal to standalone module
```bash
# New structure
zbz-cereal/          # Separate Go module
├── go.mod           # module zbz.dev/cereal
├── service.go       # Core cereal service
├── providers/       # Built-in providers only
│   ├── json.go      # Standard library JSON
│   ├── raw.go       # Byte pass-through
│   └── string.go    # String handling
└── README.md

zbz/providers/       # External providers (separate modules)
├── cereal-yaml/     # yaml.v3 dependency
├── cereal-msgpack/  # msgpack dependency
└── cereal-protobuf/ # protobuf dependency
```

**Benefits**:
- Zero circular imports
- Clean dependency management  
- External providers as optional add-ons

#### **P0: Implement Global Singleton Pattern**

**Goal**: One cereal instance per application

**Current (Broken)**:
```go
// Every operation creates new contracts
contract := cereal.NewJSONProvider(config)
contract.Unmarshal(data, target)
```

**Proposed (Fixed)**:
```go
// Global singleton initialization
func init() {
    cereal.Configure(cereal.DefaultConfig())
}

// Services use global singleton
func (cache *zCache) serialize(data any) ([]byte, error) {
    return cereal.Marshal(data) // Uses global singleton
}
```

**Implementation**:
```go
// cereal/service.go
var globalCereal *Cereal

func Configure(config CerealConfig) error {
    globalCereal = NewCereal(config)
    return nil
}

func Marshal(data any) ([]byte, error) {
    return globalCereal.Marshal(data)
}

func Unmarshal(data []byte, target any) error {
    return globalCereal.Unmarshal(data, target)
}
```

#### **P1: Auto-Format Detection**

**Goal**: Eliminate manual provider selection

**Current (Verbose)**:
```go
switch config.Serialization {
case "raw":
    contract := cereal.NewRawProvider(config)
case "json":
    contract := cereal.NewJSONProvider(config)
}
```

**Proposed (Automatic)**:
```go
// Cereal automatically determines format based on data type
cereal.Marshal([]byte("data"))     // Uses raw provider
cereal.Marshal("text")             // Uses string provider
cereal.Marshal(userStruct)         // Uses JSON provider
```

### **Phase 2: Service Integration Cleanup (Week 3)**

#### **Simplify Service Configuration**

**Goal**: Remove boilerplate from service setup

**Current (10+ lines per service)**:
```go
cerealConfig := cereal.DefaultConfig()
cerealConfig.Name = "cache-serializer"
cerealConfig.DefaultFormat = "json"
cerealConfig.EnableCaching = true
cerealConfig.EnableScoping = false
contract := cereal.NewJSONProvider(cerealConfig)
cerealProvider := contract.Provider()
```

**Proposed (Zero lines)**:
```go
// Services automatically use global cereal
// No configuration needed - auto-detected
```

#### **Standardize Service Integration Pattern**

**Goal**: Consistent dependency injection across services

**Pattern**:
```go
// Service struct just references global cereal
type zCache struct {
    provider     CacheProvider
    config       CacheConfig  
    contractName string
    // No cereal field needed - use global functions
}

// Operations use global cereal directly
func (t *TableContract[T]) Set(key string, value T) error {
    data, err := cereal.Marshal(value) // Global function
    return t.cache.provider.Set(key, data, ttl)
}
```

### **Phase 3: Performance & Production Features (Week 4)**

#### **Performance Optimization**

**Benchmarking**:
- Measure cereal vs existing serialization performance
- Identify bottlenecks in reflection caching
- Optimize scoped serialization performance

**Targets**:
- Cereal serialization within 10% of stdlib performance
- Scoped serialization within 25% of regular serialization
- Reflection cache hit rate >95% for repeated operations

#### **Error Handling Strategy**

**Current (Silent failures)**:
```go
if err := cereal.Marshal(data); err != nil {
    return fallbackSerialization(data) // Silent fallback
}
```

**Proposed (Explicit handling)**:
```go
// Configuration-based error handling
type ErrorStrategy int
const (
    ErrorStrategyFail     // Return error immediately
    ErrorStrategyFallback // Try fallback serialization
    ErrorStrategyLog      // Log error but continue
)
```

#### **Missing Production Features**

1. **Metrics Integration**
   - Serialization performance metrics
   - Error rates and fallback usage
   - Cache hit rates for reflection metadata

2. **Configuration Validation**
   - Validate scope tags at startup
   - Detect circular permission dependencies
   - Schema validation for structured data

3. **Developer Tools**
   - CLI tool for testing scope configurations
   - Debug mode for serialization tracing
   - Performance profiling integration

### **Phase 4: Advanced Features (Week 5-6)**

#### **External Provider Integration**

**YAML Provider**:
```go
// zbz/providers/cereal-yaml/
func NewYAMLProvider() cereal.Provider {
    return &yamlProvider{encoder: yaml.NewEncoder()}
}
```

**MessagePack Provider**:
```go  
// zbz/providers/cereal-msgpack/
func NewMessagePackProvider() cereal.Provider {
    return &msgpackProvider{encoder: msgpack.NewEncoder()}
}
```

#### **Advanced Scoped Serialization**

**Conditional Scoping**:
```go
type User struct {
    Email    string `json:"email" scope:"read:user|owner,write:admin"`
    Salary   int    `json:"salary" scope:"read:hr&(manager|admin)"`
    Internal string `json:"internal" scope:"read:admin&internal"`
}
```

**Dynamic Scoping**:
```go
// Runtime scope determination
cereal.MarshalWithDynamicScope(user, func(field string) []string {
    return getUserPermissions(context, field)
})
```

## Success Metrics

### **Phase 1 (Foundation)**
- ✅ Zero circular import errors
- ✅ Global singleton pattern working
- ✅ Auto-format detection for basic types
- ✅ All existing tests pass

### **Phase 2 (Integration)**  
- ✅ Service configuration reduced by >80%
- ✅ Consistent dependency injection pattern
- ✅ All 4 services using cereal seamlessly
- ✅ Zero breaking changes to service APIs

### **Phase 3 (Production)**
- ✅ Performance within 10% of stdlib
- ✅ Comprehensive error handling
- ✅ Production monitoring integrated
- ✅ Load testing at scale

### **Phase 4 (Advanced)**
- ✅ External provider ecosystem
- ✅ Advanced scoping features
- ✅ Developer tooling complete
- ✅ Documentation and examples

## Risk Assessment

### **High Risk**
- **Circular imports**: Could require significant refactoring
- **Performance regression**: May need architecture changes
- **Breaking changes**: Could impact existing service APIs

### **Medium Risk**  
- **Complex scoping logic**: Edge cases in permission evaluation
- **External provider compatibility**: Third-party serialization libraries
- **Migration complexity**: Moving from V1 to V2

### **Low Risk**
- **Configuration changes**: Well-contained modifications
- **Error handling**: Additive changes only
- **Developer tooling**: Independent of core functionality

## Next Steps

1. **Review this plan** and provide feedback on priorities
2. **Approve Phase 1 scope** for immediate implementation
3. **Define success criteria** for each phase milestone
4. **Establish testing strategy** for V1 → V2 migration
5. **Plan external provider roadmap** based on usage priorities

## Questions for Review

1. **Module structure**: Agree with separate zbz-cereal module approach?
2. **Global singleton**: Comfortable with global state vs dependency injection?
3. **Auto-detection**: Should cereal guess formats or require explicit configuration?
4. **Migration strategy**: Big bang upgrade or gradual service-by-service migration?
5. **External providers**: Priority order for YAML, MessagePack, Protobuf?

---

**Prepared by**: Claude Code Assistant  
**Date**: 2025-01-26  
**Status**: Draft for Review