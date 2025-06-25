# ZLog - Structured Logging for Go

> **Structured logging with automatic security, cloud storage, and provider abstraction**

ZLog provides structured logging with built-in security features, cloud storage integration, and the ability to swap logging providers without code changes. Field preprocessing handles secrets and PII automatically, while the universal output system writes to both console and cloud storage simultaneously.

## 🚀 Quick Start

```go
import "zbz/zlog"

// Simple console logging (development)
zlog.Info("Application started",
    zlog.String("version", "1.0.0"),
    zlog.Int("port", 8080))

// Production with cloud storage
hodorContract := setupHodorStorage() // Your cloud storage contract
zapLogger := zap.NewWithHodor(zap.Config{
    Name:   "my-app",
    Level:  "info",
    Format: "json",
}, &hodorContract)

zlog.Configure(zapLogger.Zlog())
zlog.Info("Now logging to console + cloud storage!")
```

## ✨ Key Features

### 🔐 **Automatic Security**

```go
// Secrets automatically redacted before ANY logger sees them
zlog.Info("User login attempt",
    zlog.String("username", "john@example.com"),
    zlog.Secret("password", "secret123"),        // → ***REDACTED***
    zlog.PII("ssn", "123-45-6789"),             // → sha256:8f7b2c1a
)
```

### ☁️ **Cloud Storage Integration**

```go
// Automatic dual output: console + cloud storage
// Works with S3, GCS, Azure, Minio, any hodor provider
logger := zap.NewWithHodor(config, &hodorContract)
// Console output for development
// Cloud storage for production
// Automatic rotation, compression, buffering
```

### 🔄 **Provider Independence**

```go
// Same application code, different logging backends
zapLogger := zap.NewWithHodor(config, hodor)     // Performance-focused
zerologLogger := zerolog.NewWithHodor(config, hodor) // Memory efficient
logrusLogger := logrus.NewWithHodor(config, hodor)   // Feature-rich

// Switch with configuration only
zlog.Configure(zapLogger.Zlog()) // or any other provider
```

### 🎯 **Dual Interface Access**

```go
contract := zap.NewWithHodor(config, hodor)

// Universal interface for application code
logger := contract.Zlog()
logger.Info("Universal interface")

// Native interface for library-specific features
zapLogger := contract.Logger() // *zap.Logger
zapLogger.Sugar().Infof("Native zap features")
```

## 🏗️ Architecture Overview

ZLog uses a **three-layer architecture**:

```
Application Code
      ↓
┌─────────────────┐
│  Service Layer  │ ← Field preprocessing (secrets, PII, metrics)
└─────────────────┘
      ↓
┌─────────────────┐
│ Universal APIs  │ ← Level/format conversion, cloud storage
└─────────────────┘
      ↓
┌─────────────────┐
│    Providers    │ ← Zap, Zerolog, Logrus, slog, Apex
└─────────────────┘
```

### Field Processing Pipeline

Fields are processed through a universal pipeline before reaching any logging provider:

```go
func processFields(fields []Field) []Field {
    for field := range fields {
        switch field.Type {
        case SecretType:      // Encrypt/redact sensitive data
        case PIIType:         // Hash PII for compliance
        case MetricType:      // Convert to structured metrics
        case CorrelationType: // Extract trace/request context
        case CallDepthType:   // Adjust call stack accuracy
        }
    }
}
```

## 🔧 Advanced Features

### Secret & PII Management

```go
zlog.Info("Payment processed",
    zlog.String("user_id", "user123"),
    zlog.Secret("api_key", key),           // Automatic redaction
    zlog.PII("email", "user@example.com"), // SHA256 hash for compliance
    zlog.String("amount", "$50.00"),
)
// Output: {"user_id":"user123","api_key":"***REDACTED***","email_hash":"sha256:8f7b2c1a","amount":"$50.00"}
```

### Observability Integration

```go
zlog.Info("Request completed",
    zlog.Metric("latency_ms", 142),           // Structured for metrics systems
    zlog.Metric("response_size", 1024),       // Automatic metric formatting
    zlog.Correlation(ginContext),             // Extract trace/span IDs
    zlog.String("endpoint", "/api/users"),
)
```

### Call Depth Correction

```go
// Wrapper functions report correct file/line automatically
func LogWrapper(msg string) {
    zlog.Info(msg, zlog.CallDepth(1)) // Adjusts call stack
}

LogWrapper("Test") // Reports caller's file:line, not wrapper's
```

### Performance Optimization

```go
// Zero allocation field helpers
zlog.Info("High performance logging",
    zlog.String("key", value),     // No reflect, no interface{}
    zlog.Int("count", 42),         // Direct type conversion
    zlog.Duration("elapsed", dur), // Optimized for each provider
)
```

## 📦 Available Providers

All providers support the **same features** through universal services:

| Provider    | Strengths                            | Use Case                        |
| ----------- | ------------------------------------ | ------------------------------- |
| **Zap**     | Performance-focused, zero-allocation | High-throughput services        |
| **Zerolog** | Memory efficient, fast JSON          | Memory-constrained environments |
| **Logrus**  | Feature-rich, extensive ecosystem    | Complex logging requirements    |
| **slog**    | Go standard library, stable          | Standard library preference     |
| **Apex**    | Beautiful output, developer-friendly | Development and debugging       |
| **Simple**  | Minimal, console-only                | Testing and minimal deployments |

### Provider Configuration

```go
// All providers use the same configuration pattern
config := Config{
    Name:   "my-service",
    Level:  "info",        // debug, info, warn, error, fatal
    Format: "json",        // json, console, text
}

// Only difference is the constructor
zapLogger := zap.NewWithHodor(config, hodor)
zerologLogger := zerolog.NewWithHodor(config, hodor)
// etc.
```

## ☁️ Cloud Storage Integration

### Hodor Contract Setup

```go
// Works with any cloud provider through hodor contracts
s3Contract := hodor.NewS3Contract("my-logs-bucket")
gcsContract := hodor.NewGCSContract("my-logs-bucket")
minioContract := hodor.NewMinioContract("localhost:9000")

// Same API regardless of provider
logger := zap.NewWithHodor(config, &s3Contract)
```

### Automatic Features

- **Rotation**: Size-based, time-based, or hybrid strategies
- **Compression**: Gzip compression for storage efficiency
- **Buffering**: Buffering to prevent log loss and improve performance
- **Async**: Storage operations don't block your application
- **Retry**: Retry with exponential backoff for storage failures

### Storage Configuration

```go
hodorConfig := zlog.DefaultHodorConfig(contract, "my-app/logs")
hodorConfig.Rotation = zlog.RotationStrategy{
    Method:   "hybrid",           // size + time based
    MaxSize:  50 * 1024 * 1024,  // 50MB files
    MaxAge:   24 * time.Hour,     // 1 day rotation
    MaxFiles: 30,                 // 30 days retention
}
hodorConfig.Compression = true
hodorConfig.BufferSize = 8192    // 8KB buffer
```

## 🎯 Production Patterns

### Service Initialization

```go
func initLogging(hodorContract *hodor.HodorContract) {
    if os.Getenv("ENV") == "development" {
        // Console-only for development
        zlog.Configure(zlog.NewSimpleProvider())
        return
    }

    // Production: zap + cloud storage
    config := zap.Config{
        Name:   "my-service",
        Level:  os.Getenv("LOG_LEVEL"),
        Format: "json",
    }

    logger := zap.NewWithHodor(config, hodorContract)
    zlog.Configure(logger.Zlog())

    zlog.Info("Logging initialized",
        zlog.String("provider", "zap"),
        zlog.String("storage", hodorContract.GetProvider()),
    )
}
```

### Error Handling with Context

```go
func handleRequest(ctx *gin.Context) {
    start := time.Now()

    defer func() {
        zlog.Info("Request completed",
            zlog.Correlation(ctx),              // Automatic trace extraction
            zlog.String("method", ctx.Request.Method),
            zlog.String("path", ctx.Request.URL.Path),
            zlog.Metric("latency_ms", time.Since(start).Milliseconds()),
            zlog.Int("status", ctx.Writer.Status()),
        )
    }()

    // Request handling...
    if err != nil {
        zlog.Error("Request failed",
            zlog.Correlation(ctx),
            zlog.Err(err),
            zlog.String("user_id", getUserID(ctx)),
        )
        return
    }
}
```

### Structured Application Events

```go
// User events
zlog.Info("User registered",
    zlog.String("user_id", user.ID),
    zlog.PII("email", user.Email),          // Automatic hashing
    zlog.String("plan", user.Plan),
    zlog.Metric("registration_count", 1),   // For metrics aggregation
)

// System events
zlog.Info("Database migration completed",
    zlog.String("migration", "add_user_preferences"),
    zlog.Metric("migration_duration_ms", elapsed.Milliseconds()),
    zlog.Int("affected_rows", rows),
)

// Security events
zlog.Warn("Suspicious login attempt",
    zlog.String("ip", request.RemoteAddr),
    zlog.String("user_agent", request.UserAgent()),
    zlog.Secret("attempted_password", password), // Automatic redaction
    zlog.Int("attempt_count", attempts),
)
```

## 🔧 Configuration Reference

### Universal Configuration

```go
type Config struct {
    Name   string // Service name (used in log keys)
    Level  string // debug, info, warn, error, fatal
    Format string // json, console, text

    // Optional: Provider-specific settings
    Sampling *SamplingConfig // Rate limiting for high-volume logs
}

type SamplingConfig struct {
    Initial    int // Sample first N per second
    Thereafter int // Then 1 in N thereafter
}
```

### Storage Configuration

```go
type HodorConfig struct {
    Contract      HodorContract     // Cloud storage contract
    KeyPrefix     string           // "my-app/logs" → my-app/logs-2023-12-25.log
    Rotation      RotationStrategy // Size/time/hybrid rotation
    Compression   bool             // Gzip compression
    BufferSize    int              // Buffer size in bytes
    FlushInterval time.Duration    // Max time before flush
}
```

## 🚧 Migration Guide

### From Zap

```go
// Before: Traditional zap
logger, _ := zap.NewProduction()
defer logger.Sync()
logger.Info("message", zap.String("key", "value"))

// After: ZBZ zap adapter
contract := zap.NewWithHodor(config, hodor)
zlog.Configure(contract.Zlog())
zlog.Info("message", zlog.String("key", "value"))

// Bonus: Still access native zap when needed
zapLogger := contract.Logger()
zapLogger.Sugar().Infof("Native zap: %s", value)
```

### From Logrus

```go
// Before: Traditional logrus
logger := logrus.New()
logger.WithField("key", "value").Info("message")

// After: ZBZ logrus adapter
contract := logrus.NewWithHodor(config, hodor)
zlog.Configure(contract.Zlog())
zlog.Info("message", zlog.String("key", "value"))
```

## 🎛️ Advanced Usage

### Custom Field Types

```go
// Add your own preprocessing logic
func DatabaseQuery(query string, duration time.Duration) zlog.Field {
    return zlog.Any("db_query", map[string]interface{}{
        "query":    sanitizeSQL(query),     // Custom sanitization
        "duration": duration.Milliseconds(),
        "type":     "database",
    })
}

zlog.Info("Database operation", DatabaseQuery(sql, elapsed))
```

### Provider Performance Testing

```go
func BenchmarkProviders() {
    providers := []struct{
        name string
        logger zlog.ZlogService
    }{
        {"zap", zap.NewWithHodor(config, hodor).Zlog()},
        {"zerolog", zerolog.NewWithHodor(config, hodor).Zlog()},
        {"logrus", logrus.NewWithHodor(config, hodor).Zlog()},
    }

    for _, p := range providers {
        zlog.Configure(p.logger)
        // Benchmark your workload with each provider
        benchmarkLogging(p.name)
    }
}
```

### Multi-Region Logging

```go
// Different storage per region
func setupRegionalLogging(region string) {
    var contract hodor.HodorContract

    switch region {
    case "us-east-1":
        contract = hodor.NewS3Contract("logs-us-east-1")
    case "eu-west-1":
        contract = hodor.NewS3Contract("logs-eu-west-1")
    case "ap-southeast-1":
        contract = hodor.NewS3Contract("logs-ap-southeast-1")
    }

    logger := zap.NewWithHodor(config, &contract)
    zlog.Configure(logger.Zlog())
}
```

## 🔮 Roadmap

- [ ] **Enhanced Encryption**: Customer-controlled encryption keys for secrets
- [ ] **Metrics Integration**: Direct Prometheus/StatsD output
- [ ] **Trace Integration**: Native OpenTelemetry integration
- [ ] **Log Streaming**: Real-time log streaming for development
- [ ] **Schema Validation**: Optional log schema enforcement
- [ ] **Cost Optimization**: Intelligent log sampling based on storage costs

## 📚 Examples

See the `/examples` directory for complete usage examples:

- [Basic Usage](./examples/basic)
- [Web Service Integration](./examples/web-service)
- [Multi-Provider Setup](./examples/multi-provider)
- [Enterprise Security](./examples/enterprise)
- [Performance Optimization](./examples/performance)

---

ZLog provides structured logging with built-in security, cloud storage, and provider flexibility.
