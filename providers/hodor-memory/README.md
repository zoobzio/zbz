# Hodor Memory Provider

> **In-memory storage provider for development and testing**

The memory provider implements Hodor's storage interface using in-memory data structures. This provider is ideal for development environments, testing, and scenarios where persistent storage is not required.

## 🚀 Quick Start

```go
import _ "zbz/providers/hodor-memory" // Auto-registers the driver

// Create memory storage contract
contract := hodor.NewContract("memory", "my-memory-storage", nil)

// Use with any Hodor-compatible service
err := contract.Set("config.json", []byte(`{"env": "development"}`), 0)
data, err := contract.Get("config.json")
```

## ✨ Features

### **Thread-Safe Operations**
- Full concurrency protection with read-write mutex
- Safe for concurrent access from multiple goroutines
- Atomic operations for all CRUD functionality

### **TTL Support**
```go
// Set data with expiration
contract.Set("session:user123", sessionData, 30*time.Minute)

// Automatic cleanup of expired items
exists, _ := contract.Exists("session:user123") // false after 30 minutes
```

### **Prefix-Based Listing**
```go
// Store multiple items
contract.Set("logs/app.log", logData, 0)
contract.Set("logs/error.log", errorData, 0)
contract.Set("configs/app.json", configData, 0)

// List by prefix
logFiles, _ := contract.List("logs/")     // ["logs/app.log", "logs/error.log"]
configs, _ := contract.List("configs/")   // ["configs/app.json"]
```

### **Zero Configuration**
```go
// No setup required - works out of the box
import _ "zbz/providers/hodor-memory"

contract := hodor.NewContract("memory", "test-storage", nil)
// Ready to use immediately
```

## 🔧 Configuration

The memory provider accepts minimal configuration:

```go
config := map[string]any{
    // No configuration options needed
    // Provider operates entirely in memory
}

contract := hodor.NewContract("memory", "my-storage", config)
```

## 🎯 Use Cases

### **Development Environment**
```go
func setupDevelopmentStorage() hodor.HodorContract {
    // Fast, zero-setup storage for local development
    return hodor.NewContract("memory", "dev-storage", nil)
}

// Use with zlog for development logging
zapLogger := zap.NewWithHodor(config, &setupDevelopmentStorage())
```

### **Testing**
```go
func TestConfigurationSync(t *testing.T) {
    // Clean, isolated storage for each test
    storage := hodor.NewContract("memory", "test-storage", nil)
    defer storage.Close()
    
    // Test configuration synchronization
    config := AppConfig{Debug: true, Port: 8080}
    data, _ := json.Marshal(config)
    
    storage.Set("app.json", data, 0)
    
    // Test flux integration
    watcher, err := flux.Sync[AppConfig](storage, "app.json", handleConfigChange)
    assert.NoError(t, err)
}
```

### **Caching Layer**
```go
func setupCache() hodor.HodorContract {
    return hodor.NewContract("memory", "cache", nil)
}

func cacheUserSession(userID string, session UserSession) {
    cache := setupCache()
    data, _ := json.Marshal(session)
    
    // Cache for 1 hour
    cache.Set(fmt.Sprintf("session:%s", userID), data, time.Hour)
}
```

### **Feature Flags**
```go
func setupFeatureFlags() {
    storage := hodor.NewContract("memory", "features", nil)
    
    // Set feature flags
    flags := map[string]bool{
        "new_ui":         true,
        "beta_features":  false,
        "advanced_auth":  true,
    }
    
    for flag, enabled := range flags {
        value := "false"
        if enabled {
            value = "true"
        }
        storage.Set(fmt.Sprintf("flag:%s", flag), []byte(value), 0)
    }
}
```

## 🔧 Integration Examples

### **With Flux Configuration Management**
```go
func setupDevelopmentConfig() error {
    // Memory storage for development configs
    storage := hodor.NewContract("memory", "dev-configs", nil)
    
    // Seed with default configuration
    defaultConfig := AppConfig{
        Database: DatabaseConfig{
            Host: "localhost",
            Port: 5432,
            Name: "app_dev",
        },
        Debug:    true,
        LogLevel: "debug",
    }
    
    configData, _ := json.Marshal(defaultConfig)
    storage.Set("app.json", configData, 0)
    
    // Set up reactive configuration
    _, err := flux.Sync[AppConfig](storage, "app.json", func(old, new AppConfig) {
        log.Printf("Config updated: %+v", new)
        applyConfiguration(new)
    })
    
    return err
}
```

### **With ZLog for Development Logging**
```go
func initDevelopmentLogging() {
    // Memory storage for development logs
    storage := hodor.NewContract("memory", "dev-logs", nil)
    
    zapConfig := zap.Config{
        Name:   "development",
        Level:  "debug",
        Format: "console",
    }
    
    logger := zap.NewWithHodor(zapConfig, &storage)
    zlog.Configure(logger.Zlog())
    
    zlog.Info("Development logging initialized")
}
```

### **Multi-Environment Testing**
```go
func TestMultiEnvironmentConfig(t *testing.T) {
    environments := []string{"development", "staging", "production"}
    
    for _, env := range environments {
        t.Run(env, func(t *testing.T) {
            // Isolated storage per environment
            storage := hodor.NewContract("memory", fmt.Sprintf("%s-storage", env), nil)
            defer storage.Close()
            
            // Environment-specific configuration
            config := getConfigForEnvironment(env)
            configData, _ := json.Marshal(config)
            storage.Set("app.json", configData, 0)
            
            // Test environment setup
            testEnvironmentSetup(t, storage, config)
        })
    }
}
```

## ⚠️ Limitations

### **Data Persistence**
- **Memory-only**: Data is lost when the application restarts
- **No durability**: Not suitable for production data storage
- **Process-local**: Data is not shared between application instances

### **Memory Usage**
- **Unbounded growth**: No built-in memory limits or LRU eviction
- **Memory leaks**: Long-running applications should monitor memory usage
- **Large datasets**: Not suitable for storing large amounts of data

### **Concurrency**
- **Single process**: No sharing between different application processes
- **No clustering**: Cannot be used in distributed environments

## 🎛️ Best Practices

### **Development Setup**
```go
func setupDevelopment() {
    if os.Getenv("ENV") != "development" {
        log.Fatal("Memory provider should only be used in development")
    }
    
    storage := hodor.NewContract("memory", "dev-storage", nil)
    // Use for configuration, logging, caching in development
}
```

### **Testing Isolation**
```go
func TestWithIsolatedStorage(t *testing.T) {
    // Create fresh storage for each test
    storage := hodor.NewContract("memory", fmt.Sprintf("test-%d", time.Now().UnixNano()), nil)
    defer storage.Close()
    
    // Test logic here - guaranteed clean state
}
```

### **TTL Management**
```go
func setupSessionCache() {
    storage := hodor.NewContract("memory", "sessions", nil)
    
    // Short TTL for sensitive data
    storage.Set("session:sensitive", sessionData, 15*time.Minute)
    
    // Longer TTL for less sensitive data
    storage.Set("session:preferences", prefData, 24*time.Hour)
}
```

## 🔮 Roadmap

- [ ] **Memory Limits**: Configurable memory usage limits with LRU eviction
- [ ] **Metrics**: Memory usage and operation metrics
- [ ] **Persistence**: Optional disk persistence for development environments
- [ ] **Compression**: Optional compression for large values
- [ ] **Bulk Operations**: Batch set/get/delete operations

---

The memory provider offers fast, zero-configuration storage ideal for development and testing environments where persistence is not required.