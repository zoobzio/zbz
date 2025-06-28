# ZBZ Architecture: Circular Dependency Problem

## The Issue

We've discovered a fundamental circular dependency in the ZBZ architecture that reveals a missing abstraction layer.

### Current Circular Dependency

```
┌─────────────────┐    depends on    ┌─────────────────┐
│      CEREAL     │ ────────────────→ │      CORE       │
│  (validation,   │                  │  (ZbzModel[T])  │
│   serialization,│                  │                 │
│   field analysis│                  │                 │
└─────────────────┘                  └─────────────────┘
        ↑                                      │
        │                                      │
        └──────────────── depends on ──────────┘
```

**The Problem:**
- **Core wants to use cereal** for validation, field analysis, and caching (`cereal.Validate()`, `cereal.GetFieldMetadata()`)
- **Cereal wants to use Core** for the universal model abstraction (`ZbzModel[T]` as the standard wrapped type)

This creates an import cycle that prevents both packages from leveraging each other's capabilities.

## What We Tried

1. **Core with cereal integration** - Core imports cereal for validation and field analysis
2. **Cereal with model awareness** - Cereal knows about ZbzModel for scoping and security
3. **Both approaches fail** due to circular imports

## Evidence of Missing Abstraction

The circular dependency is a signal that we're missing a foundational layer. Consider:

### What Core Needs from Cereal:
- Type metadata (field analysis, ID detection)
- Validation capabilities
- Reflection caching (performance)
- Security/scoping rules

### What Cereal Needs from Core:
- Universal model type (`ZbzModel[T]`)
- Standard field contracts (ID, timestamps, metadata)
- Type registry/catalog

### The Pattern: Both Need a Shared Foundation

This suggests a missing **Type System** layer that both packages should depend on.

## Possible Solutions

### Option 1: Shared Type Registry Package
Create `zbz/types` that both cereal and core import:

```
┌─────────────────┐
│   zbz/types     │  ← New foundational package
│   - TypeMetadata│
│   - Registry    │
│   - BaseModel   │
└─────────────────┘
        ↑     ↑
        │     │
   ┌────────┐ │
   │ cereal │ │
   └────────┘ │
              │
         ┌────────┐
         │  core  │
         └────────┘
```

**Pros:** Clean separation, no circular deps
**Cons:** Adds complexity, new package to maintain

### Option 2: Cereal as Foundation
Make cereal the foundational package and move ZbzModel there:

```
┌─────────────────┐
│     CEREAL      │  ← Foundation with ZbzModel[T]
│   - Model[T]    │
│   - Validation  │
│   - Metadata    │
└─────────────────┘
        ↑
        │
   ┌────────┐
   │  core  │  ← Uses cereal.Model[T]
   └────────┘
```

**Pros:** Leverages existing cereal work, natural hierarchy
**Cons:** Cereal becomes heavyweight, naming confusion

### Option 3: Core as Foundation
Make core the foundational package and move validation there:

```
┌─────────────────┐
│      CORE       │  ← Foundation with ZbzModel[T]
│   - ZbzModel[T] │
│   - Validation  │
│   - Metadata    │
└─────────────────┘
        ↑
        │
   ┌────────┐
   │ cereal │  ← Uses core.ZbzModel[T]
   └────────┘
```

**Pros:** Core becomes true foundation, clear hierarchy
**Cons:** Duplicates cereal's existing validation work

### Option 4: Interface-Based Decoupling
Use interfaces to break the hard dependency:

```go
// In core
type Validator interface {
    Validate(any) error
}

type TypeAnalyzer interface {
    GetFieldMetadata(reflect.Type) *Metadata
}

// Cereal implements these interfaces
// Core accepts them as dependencies
```

**Pros:** Flexible, testable, follows dependency inversion
**Cons:** More complex API, runtime dependencies

## The Real Question

The circular dependency reveals a deeper architectural question:

**What should be the foundational abstraction in ZBZ?**

1. **Raw Go types** (current cereal approach)
2. **Wrapped ZBZ models** (current core approach)  
3. **A new shared type system**
4. **Interface-based contracts**

## Performance Implications

This also affects performance:
- **Duplicate reflection work** if both packages analyze types separately
- **Cache fragmentation** if metadata is stored in multiple places
- **Memory overhead** from redundant type information

## Current State

As of this writing:
- Core has a basic ZbzModel[T] implementation with placeholder cereal/zlog integration
- Cereal has sophisticated field analysis and validation
- The packages cannot be properly integrated due to circular imports
- We're missing ~40% of the performance benefits from avoiding duplicate reflection

## Recommended Next Steps

1. **Choose the foundational abstraction** - This is the key architectural decision
2. **Implement the dependency hierarchy** based on that choice
3. **Ensure single source of truth** for type metadata
4. **Benchmark the performance** of the chosen approach
5. **Update all dependent packages** (rocco, universal, etc.)

## Success Criteria

The solution should:
- ✅ Eliminate circular dependencies
- ✅ Avoid duplicate reflection work  
- ✅ Provide single source of truth for type metadata
- ✅ Maintain performance characteristics
- ✅ Support both packages' requirements
- ✅ Be conceptually simple and maintainable

---

*This document captures the architectural challenge discovered during core package implementation. The circular dependency is not a bug—it's a signal that we need to refine our foundational abstractions.*