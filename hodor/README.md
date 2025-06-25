# Hodor - Cloud Storage Abstraction for Go

> **Multi-provider storage abstraction with reactive capabilities and contract-based management**

Hodor provides a unified interface for cloud storage operations across multiple providers (S3, GCS, Azure, MinIO, Memory). With contract-based management and reactive change notifications, Hodor enables true "bring your own bucket" architecture for cloud-native applications.

## 🚀 Quick Start

```go
import "zbz/hodor"

// Create storage contract with any provider
contract := hodor.NewMemory(hodor.MemoryConfig{Name: "development"})

// Universal storage operations
err := contract.Set("config.json", []byte(`{"debug": true}`), 0)
data, err := contract.Get("config.json")
exists, err := contract.Exists("config.json")

// Register for service discovery
contract.Register("app-config")

// List all registered contracts
contracts := hodor.List()
```

## ✨ Key Features

### 🏗️ **Multi-Provider Abstraction**
```go
// Same interface, different storage backends
s3Contract := hodor.NewS3(s3Config)
gcsContract := hodor.NewGCS(gcsConfig)
azureContract := hodor.NewAzure(azureConfig)
memoryContract := hodor.NewMemory(memConfig)

// Identical operations across all providers
for _, contract := range []hodor.HodorContract{s3Contract, gcsContract, azureContract, memoryContract} {
    contract.Set("file.txt", data, 0)
    content, _ := contract.Get("file.txt")
}
```

### 🔄 **Reactive Storage**
```go
// Subscribe to storage changes
subscriptionID, err := contract.Subscribe("config/*", func(event hodor.ChangeEvent) {
    switch event.Type {
    case hodor.EventCreate:
        log.Printf("File created: %s", event.Key)
    case hodor.EventUpdate:
        log.Printf("File updated: %s", event.Key)
    case hodor.EventDelete:
        log.Printf("File deleted: %s", event.Key)
    }
})

// Cleanup subscription
defer contract.Unsubscribe(subscriptionID)
```

### 📋 **Contract Registration & Discovery**
```go
// Register contracts for service discovery
configStorage := hodor.NewS3(s3Config)
configStorage.Register("config-storage")

logStorage := hodor.NewGCS(gcsConfig)
logStorage.Register("log-storage")

// Discover registered contracts
contracts := hodor.List()                    // Get all contracts
info, _ := hodor.GetContract("config-storage") // Get specific contract
status, _ := hodor.Status("log-storage")     // Check health
```

### ⏰ **Universal TTL Support**
```go
// TTL works across all providers
contract.Set("session:user123", sessionData, 30*time.Minute)  // Memory TTL
contract.Set("cache:data", cacheData, 1*time.Hour)            // S3 metadata TTL
contract.Set("temp:upload", uploadData, 5*time.Minute)        // GCS TTL

// Automatic expiration handling
exists, _ := contract.Exists("session:user123") // false after 30 minutes
```

## 🏗️ Architecture Overview

Hodor uses a **contract-based storage architecture**:

```
Application Code
      ↓
┌─────────────────┐
│ Hodor Contracts │ ← Storage abstraction layer
└─────────────────┘
      ↓
┌─────────────────┐
│ Provider Router │ ← Dynamic provider selection
└─────────────────┘
      ↓
┌─────────────────┐
│  Storage Drivers│ ← S3, GCS, Azure, Memory, MinIO
└─────────────────┘
```

### Contract System

```go
type HodorContract interface {
    // Core CRUD operations
    Get(key string) ([]byte, error)
    Set(key string, data []byte, ttl time.Duration) error
    Delete(key string) error
    Exists(key string) (bool, error)
    List(prefix string) ([]string, error)
    
    // Metadata and stats
    Stat(key string) (FileInfo, error)
    
    // Reactive capabilities
    Subscribe(keyPattern string, callback ChangeCallback) (SubscriptionID, error)
    Unsubscribe(id SubscriptionID) error
    
    // Service management
    Register(alias string) error
    Close() error
    Provider() string
    Name() string
}
```

## 📦 Available Providers

| Provider | Use Case | Features |
|----------|----------|----------|
| **Memory** | Development, Testing, Caching | Fast, Zero-config, TTL support |
| **MinIO** | Self-hosted, S3-compatible | S3 API, Self-hosted, Multi-region |
| **S3** | AWS Production | Managed service, Global scale |
| **GCS** | Google Cloud | Managed service, ML integration |
| **Azure** | Microsoft Cloud | Managed service, Enterprise integration |

### Provider Setup

```go
// Memory (development/testing)
memContract := hodor.NewMemory(hodor.MemoryConfig{
    Name: "dev-storage",
})

// MinIO/S3 (production)
s3Contract := hodor.NewS3(hodor.S3Config{
    Region: "us-west-2",
    Bucket: "my-app-storage",
    // Credentials from environment
})

// Google Cloud Storage
gcsContract := hodor.NewGCS(hodor.GCSConfig{
    ProjectID: "my-project",
    Bucket:    "my-gcs-bucket",
    // Service account from environment
})
```

## 🎯 Integration Patterns

### **With ZLog (Logging)**
```go
func setupCloudLogging() error {
    // S3 storage for production logs
    logStorage := hodor.NewS3(hodor.S3Config{
        Region: "us-west-2", 
        Bucket: "myapp-logs-prod",
    })
    logStorage.Register("log-storage")
    
    // ZLog automatically uses hodor for log persistence
    zapConfig := zap.Config{
        Name:   "production",
        Level:  "info",
        Format: "json",
    }
    
    logger := zap.NewWithHodor(zapConfig, &logStorage)
    zlog.Configure(logger.Zlog())
    
    return nil
}
```

### **With Flux (Configuration)**
```go
func setupReactiveConfig() error {
    // GCS storage for configuration files
    configStorage := hodor.NewGCS(hodor.GCSConfig{
        ProjectID: "my-project",
        Bucket:    "myapp-configs",
    })
    configStorage.Register("config-storage")
    
    // Flux automatically syncs config changes from cloud storage
    _, err := flux.Sync[AppConfig](configStorage, "app.json", func(old, new AppConfig) {
        log.Printf("Configuration updated from cloud storage")
        applyConfiguration(new)
    })
    
    return err
}
```

### **Multi-Environment Management**
```go
func setupEnvironmentStorage(env string) error {
    var contract hodor.HodorContract
    
    switch env {
    case "production":
        contract = hodor.NewS3(hodor.S3Config{
            Region: "us-west-2",
            Bucket: "myapp-prod",
        })
    case "staging":
        contract = hodor.NewGCS(hodor.GCSConfig{
            ProjectID: "myproject-staging",
            Bucket:    "myapp-staging",
        })
    case "development":
        contract = hodor.NewMemory(hodor.MemoryConfig{
            Name: "dev-storage",
        })
    }
    
    contract.Register("primary-storage")
    return nil
}
```

## 🔧 Advanced Features

### **Change Notifications**
```go
func watchConfigurationChanges(contract hodor.HodorContract) {
    // Subscribe to all configuration file changes
    subID, err := contract.Subscribe("config/*", func(event hodor.ChangeEvent) {
        switch event.Key {
        case "config/app.json":
            reloadApplicationConfig()
        case "config/database.json": 
            reconnectDatabase()
        case "config/features.json":
            updateFeatureFlags()
        }
        
        log.Printf("Configuration change: %s %s", event.Type, event.Key)
    })
    
    if err != nil {
        log.Printf("Failed to subscribe to config changes: %v", err)
        return
    }
    
    // Cleanup on shutdown
    defer contract.Unsubscribe(subID)
}
```

### **Multi-Contract Orchestration**
```go
type StorageManager struct {
    configs hodor.HodorContract
    logs    hodor.HodorContract
    cache   hodor.HodorContract
}

func NewStorageManager(env string) *StorageManager {
    sm := &StorageManager{}
    
    if env == "production" {
        // Production: Different providers for different use cases
        sm.configs = hodor.NewS3(s3ConfigsConfig)
        sm.logs = hodor.NewGCS(gcsLogsConfig)
        sm.cache = hodor.NewAzure(azureCacheConfig)
    } else {
        // Development: Memory for everything
        sm.configs = hodor.NewMemory(hodor.MemoryConfig{Name: "configs"})
        sm.logs = hodor.NewMemory(hodor.MemoryConfig{Name: "logs"})
        sm.cache = hodor.NewMemory(hodor.MemoryConfig{Name: "cache"})
    }
    
    // Register all contracts
    sm.configs.Register("configs")
    sm.logs.Register("logs")
    sm.cache.Register("cache")
    
    return sm
}
```

### **Backup and Replication**
```go
func setupBackupReplication() {
    // Primary storage
    primary := hodor.NewS3(hodor.S3Config{
        Region: "us-west-2",
        Bucket: "myapp-primary",
    })
    
    // Backup storage (different region)
    backup := hodor.NewS3(hodor.S3Config{
        Region: "eu-west-1", 
        Bucket: "myapp-backup",
    })
    
    // Cross-region replication
    primary.Subscribe("*", func(event hodor.ChangeEvent) {
        switch event.Type {
        case hodor.EventCreate, hodor.EventUpdate:
            data, err := primary.Get(event.Key)
            if err == nil {
                backup.Set(event.Key, data, 0)
            }
        case hodor.EventDelete:
            backup.Delete(event.Key)
        }
    })
}
```

## 🎯 Production Patterns

### **Service Discovery Pattern**
```go
func setupServiceDiscovery() {
    // Register multiple storage contracts
    configStorage := hodor.NewS3(s3Config)
    configStorage.Register("config")
    
    logStorage := hodor.NewGCS(gcsConfig)
    logStorage.Register("logs")
    
    cacheStorage := hodor.NewMemory(memConfig)
    cacheStorage.Register("cache")
    
    // Services can discover storage by purpose
    configContract, _ := hodor.GetContract("config")
    logContract, _ := hodor.GetContract("logs") 
    cacheContract, _ := hodor.GetContract("cache")
}
```

### **Health Monitoring**
```go
func monitorStorageHealth() {
    contracts := hodor.List()
    
    for _, contractInfo := range contracts {
        status, err := hodor.Status(contractInfo.Alias)
        if err != nil {
            log.Printf("Health check failed for %s: %v", contractInfo.Alias, err)
            metrics.StorageHealth.WithLabels("contract", contractInfo.Alias).Set(0)
        } else {
            log.Printf("Storage %s is healthy: %+v", contractInfo.Alias, status)
            metrics.StorageHealth.WithLabels("contract", contractInfo.Alias).Set(1)
        }
    }
}
```

### **Graceful Shutdown**
```go
func gracefulShutdown() {
    // Close all registered contracts
    contracts := hodor.List()
    
    for _, contract := range contracts {
        log.Printf("Closing storage contract: %s", contract.Alias)
        if storageContract, err := hodor.GetContract(contract.Alias); err == nil {
            storageContract.Close()
        }
    }
    
    // Close global hodor service
    hodor.Close()
}
```

## 🔧 Configuration Reference

### **Memory Provider**
```go
config := hodor.MemoryConfig{
    Name: "memory-storage",
    // TTL cleanup interval, max size, etc.
}
contract := hodor.NewMemory(config)
```

### **S3 Provider**
```go
config := hodor.S3Config{
    Region:    "us-west-2",
    Bucket:    "my-s3-bucket",
    AccessKey: "", // From AWS_ACCESS_KEY_ID env var
    SecretKey: "", // From AWS_SECRET_ACCESS_KEY env var
    UseSSL:    true,
}
contract := hodor.NewS3(config)
```

### **GCS Provider**
```go
config := hodor.GCSConfig{
    ProjectID:   "my-gcp-project",
    Bucket:      "my-gcs-bucket",
    Credentials: "", // From GOOGLE_APPLICATION_CREDENTIALS env var
}
contract := hodor.NewGCS(config)
```

## 📊 Performance Considerations

### **Concurrent Operations**
```go
// Hodor contracts are thread-safe
var wg sync.WaitGroup

for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(index int) {
        defer wg.Done()
        key := fmt.Sprintf("item-%d", index)
        data := []byte(fmt.Sprintf("data-%d", index))
        contract.Set(key, data, 0)
    }(i)
}

wg.Wait()
```

### **Batch Operations**
```go
func batchOperations(contract hodor.HodorContract, items map[string][]byte) error {
    // Concurrent uploads for better throughput
    semaphore := make(chan struct{}, 10) // Limit concurrency
    var wg sync.WaitGroup
    errors := make(chan error, len(items))
    
    for key, data := range items {
        wg.Add(1)
        go func(k string, d []byte) {
            defer wg.Done()
            semaphore <- struct{}{}        // Acquire
            defer func() { <-semaphore }() // Release
            
            if err := contract.Set(k, d, 0); err != nil {
                errors <- err
            }
        }(key, data)
    }
    
    wg.Wait()
    close(errors)
    
    for err := range errors {
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

## 🚧 Migration Guide

### **From Direct Cloud SDKs**
```go
// Before: Direct S3 usage
s3Client := s3.New(session.New())
result, err := s3Client.GetObject(&s3.GetObjectInput{
    Bucket: aws.String("my-bucket"),
    Key:    aws.String("config.json"),
})

// After: Hodor abstraction
contract := hodor.NewS3(hodor.S3Config{Bucket: "my-bucket"})
data, err := contract.Get("config.json")
```

### **From File System**
```go
// Before: Local file operations
data, err := os.ReadFile("config.json")
err = os.WriteFile("config.json", data, 0644)

// After: Cloud storage with same semantics
contract := hodor.NewS3(s3Config) // or any provider
data, err := contract.Get("config.json")
err = contract.Set("config.json", data, 0)
```

## 🎛️ Advanced Usage

### **Custom Provider Implementation**
```go
type customProvider struct {
    // Your custom storage implementation
}

func (c *customProvider) Get(key string) ([]byte, error) {
    // Implement custom storage logic
}

// Register as provider
hodor.RegisterProvider("custom", func(config map[string]any) hodor.HodorProvider {
    return &customProvider{}
})

// Use like any other provider
contract := hodor.NewContract("custom", "my-custom", customConfig)
```

### **Middleware Pattern**
```go
func withLogging(contract hodor.HodorContract) hodor.HodorContract {
    return &loggingWrapper{
        underlying: contract,
        logger:     logger,
    }
}

func withMetrics(contract hodor.HodorContract) hodor.HodorContract {
    return &metricsWrapper{
        underlying: contract,
        metrics:    metrics,
    }
}

// Chain middleware
contract := withMetrics(withLogging(hodor.NewS3(config)))
```

## 🔮 Roadmap

- [ ] **Object Versioning**: Version control for stored objects
- [ ] **Encryption**: Client-side and server-side encryption support
- [ ] **Compression**: Automatic compression for large objects
- [ ] **Caching**: Local caching layer with invalidation
- [ ] **Metrics**: Detailed operation metrics and monitoring
- [ ] **Lifecycle Policies**: Automatic object lifecycle management

---

Hodor provides cloud storage abstraction with multi-provider support, reactive capabilities, and contract-based management for building cloud-native applications.