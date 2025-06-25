# ZBZ Zlog - Zap Driver

Ultra-high performance structured logging driver using Uber's Zap library. Perfect for production workloads requiring maximum throughput and configurability.

## Features

- **🚀 Blazing Fast**: Uber's zap - one of the fastest Go loggers
- **⚙️ Highly Configurable**: Multiple outputs, formats, sampling, rotation
- **📊 Production Ready**: Built-in file rotation, sampling, level filtering
- **🎯 Zero-Copy Performance**: Efficient field encoding and memory usage
- **🔧 Advanced Features**: Caller info, stack traces, structured fields

## Quick Start

```yaml
# zlog.yaml
name: "production-logger"
provider: "zap"
config:
  level: "info"
  format: "json"
  outputs:
    - type: "console"
      format: "console"
    - type: "file"
      target: "/var/log/app.log"
      options:
        max_size: 100
        max_backups: 10
```

```go
package main

import (
    "zbz/zlog"
    _ "zbz/providers/zlog-zap"  // Auto-registers zap driver
)

func main() {
    contract, _ := zlog.New("zlog.yaml")
    logger := contract.Zlog()
    
    zlog.Info("Server starting",
        zlog.String("host", "localhost"),
        zlog.Int("port", 8080),
        zlog.Secret("api_key", "secret123"),      // → ***REDACTED***
        zlog.Metric("startup_time", 42))          // → startup_time_value: 42
}
```

## Configuration Options

### Global Settings
```yaml
config:
  level: "debug|info|warn|error|fatal"    # Minimum log level
  format: "json|console|logfmt"            # Default format
```

### Multiple Outputs
```yaml
outputs:
  # Pretty console for development
  - type: "console"
    level: "debug"           # Show debug on console
    format: "console"        # Human-readable format
    
  # Production JSON logs
  - type: "file"
    level: "info"            # File gets info+
    format: "json"           # Structured for parsing
    target: "/var/log/app.log"
    options:
      max_size: 100          # 100MB per file
      max_backups: 10        # Keep 10 old files
      max_age: 30            # 30 days retention
      compress: true         # Compress rotated files
      
  # Error alerting
  - type: "file"
    level: "error"           # Errors only
    target: "/var/log/errors.log"
```

### High-Volume Sampling
```yaml
sampling:
  initial: 100              # Log first 100/sec
  thereafter: 1000          # Then 1 in 1000
```

## Performance Features

### Field Types
Zap driver supports all zlog field types with zero-copy performance:

```go
zlog.Info("Performance test",
    zlog.String("str", "value"),           // → zap.String()
    zlog.Int("num", 42),                   // → zap.Int()
    zlog.Float64("ratio", 0.95),           // → zap.Float64()
    zlog.Duration("latency", time.Ms*100), // → zap.Duration()
    zlog.Time("timestamp", time.Now()),    // → zap.Time()
    zlog.Bool("success", true),            // → zap.Bool()
    zlog.Err(err),                         // → zap.Error()
    zlog.Any("complex", complexObj))       // → zap.Any()
```

### Automatic Optimizations
- **Caller detection**: Automatic file/line reporting with correct stack depth
- **Level checking**: Zero-cost logging when level is disabled
- **Field pooling**: Reusable field slices to reduce garbage collection
- **Encoder selection**: Optimal encoders per output format

## Advanced Features

### Stack Traces
```yaml
# Automatic stack traces on errors
config:
  level: "info"
# Stack traces automatically added on ERROR and FATAL levels
```

### Custom Encoders
```yaml
outputs:
  - type: "console"
    format: "console"    # Human-readable with colors
  - type: "file"  
    format: "json"       # Machine-parseable
  - type: "file"
    format: "logfmt"     # Key=value format
```

### Per-Output Levels
```yaml
outputs:
  - type: "console"
    level: "debug"       # Verbose console for dev
  - type: "file"
    level: "warn"        # Only important stuff to file
```

## Production Examples

### High-Throughput API
```yaml
name: "api-server"
provider: "zap" 
config:
  level: "info"
  format: "json"
  sampling:
    initial: 100
    thereafter: 10000    # Very aggressive sampling
  outputs:
    - type: "console"
      format: "json"     # Container stdout
    - type: "file"
      target: "/var/log/api.log"
      options:
        max_size: 500    # Large files for high volume
        max_backups: 50
        compress: true
```

### Microservice with Tracing
```go
zlog.Info("Request processed",
    zlog.String("endpoint", "/api/users"),
    zlog.Int("status", 200),
    zlog.Duration("duration", time.Since(start)),
    zlog.Correlation(ctx),               // → trace_id, span_id
    zlog.Metric("response_time", 42))    // → response_time_value: 42
```

## Benchmarks

Zap driver performance (approximate):
- **10M+ logs/sec** on modern hardware
- **Zero allocations** for basic field types
- **< 50ns/log** for disabled levels
- **Minimal GC pressure** with field pooling

Perfect for high-performance applications requiring structured logging with maximum configurability.