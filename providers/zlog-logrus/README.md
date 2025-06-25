# ZBZ Zlog - Logrus Driver

Popular structured logging driver using Sirupsen Logrus. Built for simplicity, extensibility, and production reliability with powerful hooks system.

## Features

- **🔗 Hooks System**: Extensible logging with custom hooks for metrics, alerts, sampling
- **📊 Structured Logging**: Clean field-based logging with JSON/text formatters  
- **🎛️ Flexible Configuration**: Simple setup with powerful customization options
- **🌍 Battle-Tested**: Used by thousands of Go projects in production
- **🔧 Error Handling**: Special error field handling with stack traces
- **🎨 Beautiful Text**: Colored console output perfect for development

## Quick Start

```yaml
# zlog.yaml
name: "production-logger"
provider: "logrus"
config:
  level: "info"
  format: "json"
  outputs:
    - type: "console"
      format: "text"         # Beautiful colored text
    - type: "file"
      target: "/var/log/app.json"
```

```go
package main

import (
    "zbz/zlog"
    _ "zbz/providers/zlog-logrus"  // Auto-registers logrus driver
)

func main() {
    contract, _ := zlog.New("zlog.yaml")
    logger := contract.Zlog()
    
    zlog.Info("Server starting",
        zlog.String("version", "1.2.3"),
        zlog.Int("workers", 8),
        zlog.Secret("api_key", "secret123"),      // → ***REDACTED***
        zlog.Correlation(ctx))                    // → trace_id, span_id
}
```

## Why Choose Logrus?

### 🔗 **Hooks System**
```go
// Logrus excels at extensibility through hooks
// (Future: ZBZ will provide built-in hooks for common patterns)

// Example hook concepts:
// - Sampling hooks for high-volume logging
// - Metric hooks for Prometheus integration  
// - Alert hooks for Slack notifications
// - Correlation hooks for distributed tracing
```

### 📊 **Structured Fields**
```go
zlog.Info("User action completed",
    zlog.String("user_id", "12345"),
    zlog.String("action", "purchase"),
    zlog.Float64("amount", 99.99),
    zlog.Duration("processing_time", elapsed))
```
```json
{
  "time": "2023-12-01T10:30:00Z",
  "level": "info", 
  "message": "User action completed",
  "user_id": "12345",
  "action": "purchase",
  "amount": 99.99,
  "processing_time": "150ms",
  "processing_time_ms": 150
}
```

### 🎨 **Beautiful Development Output**
```yaml
outputs:
  - type: "console"
    format: "text"
```
```
INFO[10:30:00] User action completed  action=purchase amount=99.99 processing_time=150ms user_id=12345
```

## Configuration Options

### Simple Setup
```yaml
config:
  level: "debug|info|warn|error|fatal|panic|trace"
  format: "json|text|console"
```

### Multiple Outputs
```yaml
outputs:
  # Development: Colored text to console
  - type: "console"
    format: "text"
    
  # Production: JSON to rotating files
  - type: "file"
    format: "json"
    target: "/var/log/application.json"
    options:
      max_size: 100      # MB
      max_backups: 10
      max_age: 30        # days
      compress: true
```

### Error Handling
```yaml
# Logrus has special error field handling
outputs:
  - type: "file"
    format: "json"
    target: "/var/log/errors.json"
    level: "error"       # Only errors and above
```

## Field Types & Mapping

Logrus driver provides rich field type mapping:

```go
zlog.Info("Field showcase",
    zlog.String("name", "john"),              // → fields["name"] = "john"
    zlog.Int("age", 30),                      // → fields["age"] = 30  
    zlog.Float64("score", 95.5),              // → fields["score"] = 95.5
    zlog.Bool("active", true),                // → fields["active"] = true
    zlog.Err(err),                            // → fields["error"] = err (special handling)
    zlog.Duration("took", time.Second),       // → fields["took"] = "1s", fields["took_ms"] = 1000
    zlog.Time("created", time.Now()),         // → fields["created"] = time.Time
    zlog.Strings("tags", []string{"api"}))    // → fields["tags"] = []string{"api"}
```

## Production Examples

### Microservice API
```yaml
name: "user-service"
provider: "logrus"
config:
  level: "info"
  format: "json"
  outputs:
    # Kubernetes stdout - collected by fluent-bit
    - type: "console"
      format: "json"
```

### Development Environment
```yaml
name: "dev-logger"
provider: "logrus"
config:
  level: "debug"
  outputs:
    # Beautiful console for development
    - type: "console"
      format: "text"
    # Structured logs for debugging
    - type: "file" 
      format: "json"
      target: "debug.log"
```

### High-Availability Service
```yaml
name: "critical-service"
provider: "logrus"
config:
  level: "warn"            # Only important events
  format: "json"
  outputs:
    # Application logs
    - type: "file"
      format: "json"
      target: "/var/log/app.json"
      options:
        max_size: 500
        max_backups: 20
        compress: true
    # Error-only logs for alerts
    - type: "file"
      format: "json"
      target: "/var/log/errors.json"
      level: "error"
      options:
        max_size: 100
        max_backups: 50
```

## Advanced Features

### Pipeline Integration
```go
zlog.Error("Payment processing failed",
    zlog.String("payment_id", "pay_123"),
    zlog.PII("card_number", "4111-1111-1111-1111"),  // → card_number_hash: sha256:abc123
    zlog.Secret("merchant_key", "sk_live_abc"),       // → ***REDACTED***
    zlog.Metric("failure_count", failureCount),      // → failure_count_value: 3
    zlog.Correlation(ctx),                           // → trace_id, span_id, request_id
    zlog.Err(err))                                   // → Special error handling
```

### Request Logging
```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    
    zlog.Info("Request started",
        zlog.String("method", r.Method),
        zlog.String("path", r.URL.Path),
        zlog.String("remote_addr", r.RemoteAddr))
    
    defer func() {
        zlog.Info("Request completed",
            zlog.String("method", r.Method),
            zlog.String("path", r.URL.Path),
            zlog.Duration("duration", time.Since(start)),
            zlog.Metric("request_duration_ms", time.Since(start).Milliseconds()))
    }()
}
```

### Error Tracking
```go
if err := processPayment(payment); err != nil {
    zlog.Error("Payment processing failed",
        zlog.String("payment_id", payment.ID),
        zlog.String("user_id", payment.UserID), 
        zlog.Float64("amount", payment.Amount),
        zlog.Err(err))                           // Logrus handles error specially
    return err
}
```

## Driver Comparison

| Feature | Logrus | Zap | Zerolog | Slog |
|---------|--------|-----|---------|------|
| **Philosophy** | Simple + Hooks | Ultra Performance | Zero Allocation | Go Standard |
| **Extensibility** | Excellent (hooks) | Good | Minimal | Good |
| **Performance** | Good | Excellent | Excellent | Good |
| **Configuration** | Flexible | Complex | Simple | Standard |
| **Ecosystem** | Large | Large | Growing | New |
| **Error Handling** | Special support | Standard | Standard | Standard |

## When to Choose Logrus

✅ **Perfect for:**
- Teams familiar with logrus ecosystem
- Applications needing hooks/extensibility
- Services requiring special error handling
- Projects prioritizing simplicity over max performance
- Gradual migration from existing logrus usage
- Development environments (great text output)

❓ **Consider alternatives for:**
- Ultra-high performance requirements (→ zap/zerolog)
- Zero-allocation needs (→ zerolog)
- Bleeding-edge Go features (→ slog)

## Migration from Direct Logrus

Easy migration path for existing logrus users:

```go
// Before: Direct logrus usage
import "github.com/sirupsen/logrus"
logrus.WithFields(logrus.Fields{
    "user_id": userID,
    "action": "login",
}).Info("User logged in")

// After: ZBZ zlog with logrus driver
import "zbz/zlog"
zlog.Info("User logged in",
    zlog.String("user_id", userID),
    zlog.String("action", "login"))
```

Benefits of migration:
- ✅ Keep familiar logrus behavior
- ✅ Gain zlog preprocessing pipeline (Secret, PII, Correlation)  
- ✅ Easy driver switching (logrus → zap → zerolog)
- ✅ Consistent logging across all services

## Future Enhancements

Coming soon to logrus driver:
- **Sampling Hooks**: Built-in rate limiting for high-volume logs
- **Metric Hooks**: Automatic Prometheus counter integration
- **Alert Hooks**: Slack/Discord notifications for errors
- **Correlation Hooks**: Automatic trace ID injection
- **Health Hooks**: Service health monitoring through log patterns

The logrus driver combines familiarity with modern ZBZ preprocessing capabilities.