# ZBZ Zlog - Apex Driver

Elegant structured logging driver using TJ Holowaychuk's Apex Log. Built for beautiful interfaces, handler-based architecture, and developer experience.

## Features

- **🎭 Handler Architecture**: Pluggable handlers for different output formats and destinations
- **🎨 Beautiful CLI Output**: Gorgeous colored console formatting perfect for development
- **🔧 Flexible Formatting**: JSON, text, and CLI handlers with consistent interfaces
- **⚡ Performant**: Well-optimized with clean separation of concerns
- **👨‍💻 Developer First**: Created by the author of Commander.js, Express.js focusing on DX
- **🎯 Clean API**: Simple, intuitive interface with powerful capabilities

## Quick Start

```yaml
# zlog.yaml
name: "elegant-logger"
provider: "apex"
config:
  level: "info"
  format: "cli"
  outputs:
    - type: "console"
      format: "cli"          # Beautiful colored output
    - type: "file"
      format: "json"         # Structured logs for analysis
      target: "/var/log/app.json"
```

```go
package main

import (
    "zbz/zlog"
    _ "zbz/providers/zlog-apex"  // Auto-registers apex driver
)

func main() {
    contract, _ := zlog.New("zlog.yaml")
    logger := contract.Zlog()
    
    zlog.Info("Application starting",
        zlog.String("version", "2.1.0"),
        zlog.Int("port", 3000),
        zlog.Secret("auth_token", "secret123"),   // → ***REDACTED***
        zlog.Correlation(ctx))                    // → trace_id, span_id
}
```

## Why Choose Apex?

### 🎭 **Handler-Based Architecture**
```go
// Apex excels at pluggable output handling
// Different handlers for different purposes:
// - CLI handler: Beautiful development output
// - JSON handler: Structured production logs  
// - Text handler: Simple text formatting
// - Multi handler: Send to multiple destinations
```

### 🎨 **Stunning CLI Output**
```yaml
outputs:
  - type: "console"
    format: "cli"
```
```
  INFO application starting version=2.1.0 port=3000
 ERROR database connection failed host=localhost error="connection refused"
  WARN rate limit exceeded client=192.168.1.100 limit=1000
```

### 🔧 **Flexible Configuration**
```yaml
outputs:
  # Development: Beautiful CLI
  - type: "console"
    format: "cli"
    
  # Production: Structured JSON
  - type: "file"
    format: "json"
    target: "/var/log/app.json"
    
  # Debug: Simple text
  - type: "file"
    format: "text" 
    target: "/tmp/debug.log"
```

## Configuration Options

### Handler Types
```yaml
config:
  level: "debug|info|warn|error|fatal"
  format: "cli|json|text"      # Default format for outputs
  
  outputs:
    - type: "console"
      format: "cli"             # Beautiful colored console
      
    - type: "file"
      format: "json"            # Structured JSON logs
      target: "/var/log/app.json"
      options:
        max_size: 100
        max_backups: 10
        compress: true
```

### Multiple Handler Example
```yaml
outputs:
  # CLI for development
  - type: "console"
    format: "cli"
    
  # JSON for log aggregation  
  - type: "file"
    format: "json"
    target: "/var/log/structured.json"
    
  # Text for traditional syslog
  - type: "file"
    format: "text"
    target: "/var/log/application.log"
```

## Handler Showcase

### CLI Handler (Development)
```go
zlog.Info("Request processed",
    zlog.String("method", "POST"),
    zlog.String("path", "/api/users"),
    zlog.Int("status", 201),
    zlog.Duration("duration", 42*time.Millisecond))
```
```
  INFO request processed method=POST path=/api/users status=201 duration=42ms
```

### JSON Handler (Production)
```json
{
  "level": "info",
  "timestamp": "2023-12-01T15:04:05Z", 
  "message": "Request processed",
  "fields": {
    "method": "POST",
    "path": "/api/users", 
    "status": 201,
    "duration": "42ms",
    "duration_ms": 42
  }
}
```

### Text Handler (Traditional)
```
2023-12-01T15:04:05Z level=info msg="Request processed" method=POST path=/api/users status=201 duration=42ms
```

## Field Type Mapping

Apex driver provides comprehensive field type support:

```go
zlog.Info("Field demonstration",
    zlog.String("name", "alex"),              // → fields["name"] = "alex"
    zlog.Int("age", 28),                      // → fields["age"] = 28
    zlog.Float64("score", 98.7),              // → fields["score"] = 98.7
    zlog.Bool("verified", true),              // → fields["verified"] = true
    zlog.Err(err),                            // → fields["error"] = err.Error()
    zlog.Duration("latency", time.Second),    // → fields["latency"] = "1s", fields["latency_ms"] = 1000
    zlog.Time("created", time.Now()),         // → fields["created"] = time.Time
    zlog.Strings("roles", []string{"admin"})) // → fields["roles"] = []string{"admin"}
```

## Production Examples

### Web Service
```yaml
name: "web-service"
provider: "apex"
config:
  level: "info"
  outputs:
    # Beautiful development logs
    - type: "console"
      format: "cli"
    # Structured production logs
    - type: "file"
      format: "json"
      target: "/var/log/web-service.json"
      options:
        max_size: 200
        max_backups: 15
        compress: true
```

### Microservice
```yaml
name: "payment-service"
provider: "apex"
config:
  level: "warn"              # Only important events
  format: "json"
  outputs:
    # Container stdout for log collection
    - type: "console"
      format: "json"
```

### Development Environment
```yaml
name: "dev-logger"
provider: "apex"
config:
  level: "debug"
  outputs:
    # Gorgeous CLI output for development
    - type: "console"
      format: "cli"
    # JSON for debugging complex issues
    - type: "file"
      format: "json"
      target: "debug.json"
```

## Advanced Features

### Pipeline Integration
```go
zlog.Error("Payment failed",
    zlog.String("payment_id", "pay_abc123"),
    zlog.PII("customer_email", "user@example.com"),   // → customer_email_hash: sha256:abc123
    zlog.Secret("api_key", "sk_test_123"),            // → api_key: ***REDACTED***
    zlog.Metric("failure_rate", 0.02),                // → failure_rate_value: 0.02
    zlog.Correlation(ctx),                            // → trace_id, span_id, request_id
    zlog.Err(paymentErr))
```

**CLI Output:**
```
 ERROR payment failed payment_id=pay_abc123 customer_email_hash=sha256:abc123 api_key=***REDACTED*** failure_rate_value=0.02 trace_id=abc123 error="insufficient funds"
```

### Request Middleware
```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        zlog.Info("Request started",
            zlog.String("method", r.Method),
            zlog.String("path", r.URL.Path),
            zlog.String("remote_addr", r.RemoteAddr))
        
        next.ServeHTTP(w, r)
        
        zlog.Info("Request completed",
            zlog.String("method", r.Method),
            zlog.String("path", r.URL.Path),
            zlog.Duration("duration", time.Since(start)),
            zlog.Metric("request_duration_ms", time.Since(start).Milliseconds()))
    })
}
```

### Error Tracking
```go
func processOrder(order Order) error {
    if err := validateOrder(order); err != nil {
        zlog.Error("Order validation failed",
            zlog.String("order_id", order.ID),
            zlog.String("customer_id", order.CustomerID),
            zlog.Float64("amount", order.Amount),
            zlog.Err(err))
        return err
    }
    
    zlog.Info("Order processed successfully",
        zlog.String("order_id", order.ID),
        zlog.Float64("amount", order.Amount),
        zlog.Metric("order_value", order.Amount))
    
    return nil
}
```

## Driver Comparison

| Feature | Apex | Zap | Zerolog | Logrus | Simple |
|---------|------|-----|---------|--------|--------|
| **Philosophy** | Beautiful DX | Ultra Performance | Zero Allocation | Simple + Hooks | Minimal |
| **CLI Output** | Gorgeous | Basic | Basic | Good | Basic |
| **Handlers** | Built-in | Plugins | Minimal | Hooks | None |
| **Performance** | Good | Excellent | Excellent | Good | Fast |
| **Configuration** | Clean | Complex | Simple | Flexible | Minimal |
| **Dev Experience** | Excellent | Good | Good | Good | Simple |

## When to Choose Apex

✅ **Perfect for:**
- Teams prioritizing developer experience
- Applications needing beautiful CLI output
- Services requiring multiple output formats
- Projects valuing clean, readable configuration
- Development environments (stunning console output)
- Applications needing handler-based architecture

❓ **Consider alternatives for:**
- Ultra-high performance requirements (→ zap/zerolog)
- Zero-allocation needs (→ zerolog)
- Complex hook systems (→ logrus)
- Minimal dependencies (→ simple)

## Migration Benefits

### From Standard Logging
```go
// Before: Basic logging
import "log"
log.Printf("User %s performed action %s", userID, action)

// After: ZBZ with Apex driver  
import "zbz/zlog"
zlog.Info("User action",
    zlog.String("user_id", userID),
    zlog.String("action", action))
```

### Easy Driver Switching
```yaml
# Development: Beautiful CLI
provider: "apex"
config:
  format: "cli"

# Production: High-performance  
provider: "zerolog"
```

Same code, different output experience!

## Future Enhancements

Coming to apex driver:
- **Custom Handlers**: Integration with ZBZ pipeline for advanced preprocessing
- **Sampling Handlers**: Built-in rate limiting for high-volume applications
- **Metric Handlers**: Automatic metric collection through apex handlers
- **Alert Handlers**: Error notification through handler system

The apex driver brings beautiful developer experience and flexible handler architecture to the ZBZ ecosystem.