# Flux - Reactive Configuration Management for Go

> **Type-safe configuration synchronization with cloud storage integration and hot-reload capabilities**

Flux provides reactive configuration management that automatically synchronizes typed configuration structures from cloud storage. Configuration changes trigger customizable handler pipelines, enabling real-time application adaptation without restarts.

## 🚀 Quick Start

```go
import "zbz/flux"

type AppConfig struct {
    DatabaseURL string `json:"database_url"`
    Debug       bool   `json:"debug"`
    MaxConns    int    `json:"max_connections"`
}

// Simple reactive sync - calls function when config changes
contract, err := flux.Sync[AppConfig](hodorContract, "app.json", func(old, new AppConfig) {
    if old.DatabaseURL != new.DatabaseURL {
        reconnectDatabase(new.DatabaseURL)
    }
})
defer contract.Stop()

// One-time config loading without watching
config, err := flux.Get[AppConfig](hodorContract, "app.json")
```

## ✨ Key Features

### 🔄 **Reactive Configuration**
```go
// Edit config.json in AWS S3 console → app immediately responds
flux.Sync[ServerConfig](contract, "config.json", func(old, new ServerConfig) {
    if old.Port != new.Port {
        restartHTTPServer(new.Port)
    }
    if old.LogLevel != new.LogLevel {
        updateLogLevel(new.LogLevel)
    }
})
```

### 🎯 **Type Safety**
```go
// Automatic format detection based on file extension and target type
flux.Sync[DatabaseConfig](contract, "db.yaml", handleDBChange)    // YAML parsing
flux.Sync[APIKeys](contract, "keys.json", handleAPIChange)        // JSON parsing  
flux.Sync[string](contract, "template.html", handleTemplate)     // Raw content
```

### 🔧 **Handler Pipeline**
```go
// Gin-style handler pipeline for processing config changes
flux.SyncWithHandlers[Config](contract, "config.json", []flux.HandlerFunc[Config]{
    flux.Recovery[Config](),                           // Panic recovery
    flux.Validator[Config](validateBusinessRules),     // Custom validation  
    flux.FieldWatcher[Config]([]string{"Debug"}, handleDebugToggle), // Field-specific handlers
    flux.Throttle[Config](1*time.Second, logger),     // Rate limiting
    flux.Backup[Config](5),                           // Version history
    func(event *flux.FluxEvent[Config]) {             // Custom handler
        if event.Changed {
            metrics.ConfigReloads.Inc()
        }
    },
})
```

### ☁️ **Cloud Storage Integration**
```go
// Works with any Hodor-supported cloud provider
s3Contract := hodor.NewS3Contract("my-configs-bucket")
gcsContract := hodor.NewGCSContract("my-configs-bucket")
azureContract := hodor.NewAzureContract("my-configs-container")

// Same API regardless of storage provider
contract, _ := flux.Sync[Config](s3Contract, "app.json", handleChange)
```

## 🏗️ Architecture Overview

Flux uses a **reactive pipeline architecture**:

```
Cloud Storage (S3/GCS/Azure)
      ↓
┌─────────────────┐
│   Hodor Watch   │ ← Real-time change detection
└─────────────────┘
      ↓
┌─────────────────┐
│  Type Parser    │ ← JSON/YAML → Go structs
└─────────────────┘
      ↓
┌─────────────────┐
│ Handler Pipeline│ ← Validate, transform, react
└─────────────────┘
      ↓
Application Code
```

### Watcher State Management

```go
type WatcherState int
const (
    Active      // Normal operation
    Recovering  // Parse error, waiting for valid config
    Paused      // Watching but not executing callbacks
    Dismissed   // Stopped
)

// Runtime control
contract.Pause()   // Temporarily stop callbacks
contract.Resume()  // Resume callbacks
contract.Stop()    // Permanently stop watching
```

## 🔧 Handler System

### Built-in Handlers

```go
import "zbz/flux"

handlers := []flux.HandlerFunc[Config]{
    // Error handling
    flux.Recovery[Config](),                    // Panic recovery
    flux.Validator[Config](customValidator),    // Custom validation
    
    // Flow control  
    flux.Throttle[Config](1*time.Second, logger), // Rate limiting
    flux.OnlyChanges[Config](handler),          // Skip initial load
    flux.ConditionalHandler[Config](condition, handler), // Conditional execution
    
    // Monitoring
    flux.Logger[Config](logger),                // Change logging
    flux.Metrics[Config](metrics),              // Metrics collection
    flux.Diff[Config](logger),                 // Field-by-field diff logging
    
    // Field-specific watching
    flux.FieldWatcher[Config]([]string{"Database.Host"}, reconnectDB),
    flux.FieldWatcher[Config]([]string{"Debug", "LogLevel"}, updateLogging),
    
    // Utilities
    flux.Backup[Config](10),                   // Keep version history
}
```

### Custom Handlers

```go
func validateSecuritySettings(event *flux.FluxEvent[Config]) {
    if event.New.SSLEnabled && event.New.CertPath == "" {
        event.AddError(errors.New("SSL enabled but no certificate path"))
        event.Abort() // Stop pipeline
        return
    }
    
    // Pass data to next handler
    event.Set("security_validated", true)
}

func applySecurityChanges(event *flux.FluxEvent[Config]) {
    if validated := event.Get("security_validated").(bool); !validated {
        return
    }
    
    // Apply security configuration
    updateTLSSettings(event.New.SSLConfig)
}
```

### Event Object

```go
type FluxEvent[T] struct {
    Key     string  // "config.json"
    Old     T       // Previous configuration
    New     T       // New configuration  
    Changed bool    // False on initial load, true on updates
    
    // Handler communication
    Set(key string, value any)       // Pass data between handlers
    Get(key string) any              // Retrieve handler data
    
    // Error handling
    AddError(err error)              // Accumulate errors
    Errors() []error                 // Get all errors
    Abort()                         // Stop pipeline execution
    IsAborted() bool                // Check if aborted
}
```

## 📦 Configuration Patterns

### Environment-Based Configuration
```go
func setupConfig(env string) {
    var contract *hodor.HodorContract
    
    switch env {
    case "production":
        contract = hodor.NewS3Contract("prod-configs")
    case "staging": 
        contract = hodor.NewS3Contract("staging-configs")
    case "development":
        contract = hodor.NewMemoryContract(devConfigs)
    }
    
    flux.Sync[AppConfig](contract, "app.json", handleConfigChange)
}
```

### Multi-Configuration Management
```go
// Watch multiple related configurations
type ConfigManager struct {
    app  flux.FluxContract
    db   flux.FluxContract  
    auth flux.FluxContract
}

func (c *ConfigManager) Start(contract *hodor.HodorContract) error {
    var err error
    
    c.app, err = flux.Sync[AppConfig](contract, "app.json", c.handleAppConfig)
    if err != nil { return err }
    
    c.db, err = flux.Sync[DBConfig](contract, "database.json", c.handleDBConfig)
    if err != nil { return err }
    
    c.auth, err = flux.Sync[AuthConfig](contract, "auth.json", c.handleAuthConfig)
    if err != nil { return err }
    
    return nil
}

func (c *ConfigManager) Stop() {
    c.app.Stop()
    c.db.Stop()
    c.auth.Stop()
}
```

### Hot-Reload Service Configuration
```go
type ServerConfig struct {
    HTTP struct {
        Port    int    `json:"port"`
        Timeout int    `json:"timeout"`
    } `json:"http"`
    
    Database struct {
        Host     string `json:"host"`
        Password string `json:"password"`
    } `json:"database"`
}

func handleServerConfigChange(old, new ServerConfig) {
    // HTTP server changes
    if old.HTTP.Port != new.HTTP.Port {
        log.Info("Restarting HTTP server", "old_port", old.HTTP.Port, "new_port", new.HTTP.Port)
        restartHTTPServer(new.HTTP.Port)
    }
    
    // Database changes
    if old.Database.Host != new.Database.Host || old.Database.Password != new.Database.Password {
        log.Info("Reconnecting to database")
        database.Reconnect(new.Database)
    }
}
```

## 🔐 Security & Validation

### Content Validation
```go
// Built-in security checks
func secureConfigValidator(event *flux.FluxEvent[Config]) {
    // File size limits (automatic)
    // MIME type validation (automatic)  
    // Content structure validation (automatic)
    
    // Custom business rule validation
    if event.New.MaxConnections > 1000 {
        event.AddError(errors.New("max_connections exceeds safety limit"))
    }
    
    if event.New.AllowedIPs == nil {
        event.AddError(errors.New("allowed_ips must be specified"))
    }
}
```

### Error Recovery
```go
// Automatic recovery from parse errors
flux.SyncWithHandlers[Config](contract, "config.json", []flux.HandlerFunc[Config]{
    flux.Recovery[Config](),  // Recovers from panics
    validateConfig,           // Custom validation
    
    func(event *flux.FluxEvent[Config]) {
        if len(event.Errors()) > 0 {
            // Log errors but continue with last known good config
            log.Error("Config validation failed", "errors", event.Errors())
            return
        }
        
        // Apply valid configuration
        applyConfig(event.New)
    },
})
```

## 🎯 Production Patterns

### Service Initialization
```go
func initConfigManagement(hodorContract *hodor.HodorContract) error {
    // Main application config
    appContract, err := flux.SyncWithHandlers[AppConfig](
        hodorContract, "app.json",
        []flux.HandlerFunc[AppConfig]{
            flux.Recovery[AppConfig](),
            flux.Validator[AppConfig](validateAppConfig),
            flux.Logger[AppConfig](logger),
            flux.Metrics[AppConfig](configMetrics),
            applyAppConfig,
        },
    )
    if err != nil {
        return fmt.Errorf("failed to setup app config: %w", err)
    }
    
    // Feature flags  
    flagsContract, err := flux.Sync[FeatureFlags](
        hodorContract, "features.json",
        updateFeatureFlags,
    )
    if err != nil {
        return fmt.Errorf("failed to setup feature flags: %w", err)
    }
    
    // Register shutdown cleanup
    shutdown.Register(func() {
        appContract.Stop()
        flagsContract.Stop()
    })
    
    return nil
}
```

### Monitoring & Observability
```go
func configMetricsHandler(event *flux.FluxEvent[Config]) {
    if event.Changed {
        metrics.ConfigReloads.WithLabels("file", event.Key).Inc()
        
        // Track field-level changes
        diff := compareConfigs(event.Old, event.New)
        for field := range diff {
            metrics.ConfigFieldChanges.WithLabels("field", field).Inc()
        }
    }
    
    // Performance tracking
    start := time.Now()
    defer func() {
        metrics.ConfigProcessingDuration.Observe(time.Since(start).Seconds())
    }()
}
```

### Blue/Green Configuration Deployment
```go
func blueGreenConfigDeploy(event *flux.FluxEvent[Config]) {
    // Validate new config in isolated environment
    if err := validateInSandbox(event.New); err != nil {
        event.AddError(fmt.Errorf("sandbox validation failed: %w", err))
        return
    }
    
    // Health check with new config
    if err := healthCheckWithConfig(event.New); err != nil {
        event.AddError(fmt.Errorf("health check failed: %w", err))
        return
    }
    
    // Gradual rollout
    rolloutConfig(event.New, 0.1) // 10% of traffic
    time.Sleep(30 * time.Second)
    
    if getErrorRate() < 0.01 { // Less than 1% error rate
        rolloutConfig(event.New, 1.0) // Full rollout
    } else {
        event.AddError(errors.New("rollout aborted due to high error rate"))
    }
}
```

## 🔧 Configuration Reference

### FluxContract Interface
```go
type FluxContract interface {
    Stop()                      // Stop watching
    Pause()                     // Pause callbacks
    Resume()                    // Resume callbacks  
    State() WatcherState        // Get current state
    LastUpdate() time.Time      // When last updated
    Errors() []error           // Get accumulated errors
}
```

### FluxOptions
```go
type FluxOptions struct {
    InitialLoad    bool          // Call handlers on startup (default: true)
    ThrottleWindow time.Duration // Minimum time between callbacks
    MaxFileSize    int64         // Maximum config file size
    Timeout        time.Duration // Storage operation timeout
}

contract, _ := flux.Sync[Config](hodorContract, "config.json", handler, 
    flux.FluxOptions{
        InitialLoad:    false,
        ThrottleWindow: 5 * time.Second,
        MaxFileSize:    1024 * 1024, // 1MB
        Timeout:        30 * time.Second,
    },
)
```

## 🚧 Migration Guide

### From Static Configuration
```go
// Before: Static config loading
func loadConfig() Config {
    data, _ := os.ReadFile("config.json")
    var config Config
    json.Unmarshal(data, &config)
    return config
}

config := loadConfig()

// After: Reactive configuration
flux.Sync[Config](contract, "config.json", func(old, new Config) {
    // Automatically called when config changes
    updateApplicationConfig(new)
})
```

### From Environment Variables
```go
// Before: Environment variable configuration
func loadFromEnv() Config {
    return Config{
        Port:        getEnvInt("PORT", 8080),
        DatabaseURL: getEnv("DATABASE_URL", ""),
        Debug:       getEnvBool("DEBUG", false),
    }
}

// After: File-based with hot reload
type Config struct {
    Port        int    `json:"port"`
    DatabaseURL string `json:"database_url"`
    Debug       bool   `json:"debug"`
}

flux.Sync[Config](contract, "config.json", applyConfiguration)
```

## 🎛️ Advanced Usage

### Custom Type Parsing
```go
// Register custom parsers for specific types
flux.RegisterParser(".toml", func(data []byte, v any) error {
    return toml.Unmarshal(data, v)
})

// Now TOML files are supported
flux.Sync[Config](contract, "config.toml", handler)
```

### Conditional Field Watching
```go
// Only react to specific field changes
handlers := []flux.HandlerFunc[Config]{
    flux.FieldWatcher[Config]([]string{"Database"}, func(event *flux.FluxEvent[Config]) {
        log.Info("Database config changed")
        reconnectDatabase(event.New.Database)
    }),
    
    flux.FieldWatcher[Config]([]string{"Logging.Level"}, func(event *flux.FluxEvent[Config]) {
        log.Info("Log level changed")
        updateLogLevel(event.New.Logging.Level)
    }),
}
```

### Configuration Schema Evolution
```go
type ConfigV1 struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

type ConfigV2 struct {
    Server struct {
        Host string `json:"host"`
        Port int    `json:"port"`
    } `json:"server"`
    Version int `json:"version"`
}

func migrateConfig(event *flux.FluxEvent[ConfigV2]) {
    if event.New.Version == 0 {
        // Auto-migrate from V1 format
        log.Info("Migrating config from V1 to V2 format")
        // Migration logic here
    }
}
```

## 🔮 Roadmap

- [ ] **GraphQL Integration**: Query configuration via GraphQL API
- [ ] **Configuration Diffing**: Enhanced diff visualization and rollback
- [ ] **Multi-Tenant Support**: Namespace configurations by tenant/environment
- [ ] **Configuration Templates**: Template-based configuration generation
- [ ] **Audit Logging**: Track all configuration changes with attribution
- [ ] **Configuration Testing**: Built-in configuration testing framework

## 📚 Examples

See the `/examples` directory for complete usage examples:

- [Basic Configuration Sync](./examples/basic)
- [Handler Pipeline Setup](./examples/handlers) 
- [Multi-Environment Deployment](./examples/multi-env)
- [Feature Flag Management](./examples/feature-flags)
- [Microservice Configuration](./examples/microservices)

---

Flux provides reactive configuration management with type safety, cloud storage integration, and real-time hot-reload capabilities.