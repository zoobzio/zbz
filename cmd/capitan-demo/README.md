# Capitan Event System Demo

Interactive demonstrations of Capitan's event coordination capabilities.

## Quick Start

```bash
cd cmd/capitan-demo

# Run interactive demo
go run .

# Run performance benchmark  
go run . --benchmark
```

## What You'll See

### 1. 🔄 Auto-Hydration Demo
Shows how Capitan automatically receives zlog events when imported:
```go
import _ "zbz/capitan"  // Auto-connects to zlog
zlog.Info("message")   // Automatically triggers capitan handlers
```

### 2. 📡 Direct Event Emission
Demonstrates custom event emission and handling:
```go
capitan.EmitEvent("user.created", userData)
// Multiple handlers can process the same event
```

### 3. 🔗 Adapter Ecosystem 
Shows multiple adapters processing the same events simultaneously:
- **Metrics Adapter**: Counts events, tracks KPIs
- **Analytics Adapter**: Records events for analysis
- **Audit Adapter**: Logs events for compliance

### 4. 🔄 Event Transformation Pipeline
Demonstrates event chains and data transformation:
```
Raw Signup → Data Enrichment → Welcome Workflow → Business Metrics → Dashboard
```

### 5. 🎯 Typed Event Handling
Shows compile-time type safety with generic events:
```go
capitan.RegisterInput[UserEvent](UserCreated, handler)
capitan.Emit(ctx, UserCreated, source, userData, metadata)
```

### 6. ⚡ Performance Demo
High-throughput testing with:
- 10,000 events
- 5 handlers per event  
- 10 concurrent emitters
- Performance metrics and throughput analysis

### 7. 🧠 Memory Efficiency
Demonstrates zero-allocation event processing patterns.

## Key Concepts Demonstrated

### Adapter Patterns
```go
// Simple function-based adapter
capitan.RegisterByteHandler("event.type", func(data []byte) error {
    // Process event data
    return nil
})

// Type-safe adapter
capitan.RegisterInput[EventType](EventEnum, func(event EventType) error {
    // Strongly typed event processing
    return nil
})
```

### Event Composition
Events can trigger other events, creating powerful data pipelines:
```go
userSignup → dataEnrichment → welcomeWorkflow → metrics → dashboard
```

### Zero-Configuration Observability
Just import capitan and all zlog events automatically become available to adapters.

## Performance Characteristics

Actual benchmark results on modern hardware:
- **Emit Rate**: ~800,000+ events/sec (single-threaded)
- **Process Rate**: ~11,000,000+ handlers/sec (concurrent)  
- **Latency**: ~1.2µs per event (sub-millisecond)
- **Memory**: ~5KB handler overhead, near-zero per-event overhead
- **Throughput**: ~1GB/sec estimated data processing

Run `go run . --benchmark` to test on your hardware!

## Real-World Applications

This demo shows patterns for:
- **Observability**: Metrics, analytics, monitoring
- **Compliance**: Audit trails, GDPR, SOX
- **Business Intelligence**: Real-time dashboards, KPIs
- **Automation**: Workflow triggers, notifications
- **Performance**: High-throughput event processing

## Architecture Benefits

1. **Composability**: Add new adapters without changing core code
2. **Type Safety**: Compile-time guarantees with runtime flexibility  
3. **Performance**: Byte-based service layer, minimal overhead
4. **Simplicity**: One-line adapter connections
5. **Zero Coupling**: Services don't know about adapters

Run the demo to see these concepts in action! 🚀