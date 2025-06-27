# Depot - Cloud Storage Abstraction for Go

> **Multi-provider storage abstraction with reactive capabilities and contract-based management**

Depot provides a unified interface for cloud storage operations across multiple providers (S3, GCS, Azure, MinIO, Memory). With contract-based management and reactive change notifications, Depot enables true "bring your own bucket" architecture for cloud-native applications.

## 🚀 Quick Start

```go
import "zbz/depot"

// Create storage contract with any provider
contract := depot.NewMemory(depot.MemoryConfig{Name: "development"})

// Universal storage operations
err := contract.Set("config.json", []byte(`{"debug": true}`), 0)
data, err := contract.Get("config.json")
exists, err := contract.Exists("config.json")

// Register for service discovery
contract.Register("app-config")

// List all registered contracts
contracts := depot.List()
```

## ✨ Key Features

### 🏗️ **Multi-Provider Abstraction**
```go
// Same interface, different storage backends
s3Contract := depot.NewS3(s3Config)
gcsContract := depot.NewGCS(gcsConfig)
azureContract := depot.NewAzure(azureConfig)
memoryContract := depot.NewMemory(memConfig)

// Identical operations across all providers
for _, contract := range []depot.DepotContract{s3Contract, gcsContract, azureContract, memoryContract} {
    contract.Set("file.txt", data, 0)
    content, _ := contract.Get("file.txt")
}
```

### 🔄 **Reactive Storage**
```go
// Subscribe to storage changes
subscriptionID, err := contract.Subscribe("config/*", func(event depot.ChangeEvent) {
    switch event.Type {
    case depot.EventCreate:
        log.Printf("File created: %s", event.Key)
    case depot.EventUpdate:
        log.Printf("File updated: %s", event.Key)
    case depot.EventDelete:
        log.Printf("File deleted: %s", event.Key)
    }
})

// Cleanup subscription
defer contract.Unsubscribe(subscriptionID)
```

### 📋 **Contract Registration & Discovery**
```go
// Register contracts for service discovery
configStorage := depot.NewS3(s3Config)
configStorage.Register("config-storage")

logStorage := depot.NewGCS(gcsConfig)
logStorage.Register("log-storage")

// Discover registered contracts
contracts := depot.List()                    // Get all contracts
info, _ := depot.GetContract("config-storage") // Get specific contract
status, _ := depot.Status("log-storage")     // Check health
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

Depot uses a **contract-based storage architecture**:

```
Application Code
      ↓
┌─────────────────┐
│ Depot Contracts │ ← Storage abstraction layer
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
type DepotContract interface {
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
memContract := depot.NewMemory(depot.MemoryConfig{
    Name: "dev-storage",
})

// MinIO/S3 (production)
s3Contract := depot.NewS3(depot.S3Config{
    Region: "us-west-2",
    Bucket: "my-app-storage",
    // Credentials from environment
})

// Google Cloud Storage
gcsContract := depot.NewGCS(depot.GCSConfig{
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
    logStorage := depot.NewS3(depot.S3Config{
        Region: "us-west-2", 
        Bucket: "myapp-logs-prod",
    })
    logStorage.Register("log-storage")
    
    // ZLog automatically uses depot for log persistence
    zapConfig := zap.Config{
        Name:   "production",
        Level:  "info",
        Format: "json",
    }
    
    logger := zap.NewWithDepot(zapConfig, &logStorage)
    zlog.Configure(logger.Zlog())
    
    return nil
}
```

### **With Flux (Configuration)**
```go
func setupReactiveConfig() error {
    // GCS storage for configuration files
    configStorage := depot.NewGCS(depot.GCSConfig{
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
    var contract depot.DepotContract
    
    switch env {
    case "production":
        contract = depot.NewS3(depot.S3Config{
            Region: "us-west-2",
            Bucket: "myapp-prod",
        })
    case "staging":
        contract = depot.NewGCS(depot.GCSConfig{
            ProjectID: "myproject-staging",
            Bucket:    "myapp-staging",
        })
    case "development":
        contract = depot.NewMemory(depot.MemoryConfig{
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
func watchConfigurationChanges(contract depot.DepotContract) {
    // Subscribe to all configuration file changes
    subID, err := contract.Subscribe("config/*", func(event depot.ChangeEvent) {
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
    configs depot.DepotContract
    logs    depot.DepotContract
    cache   depot.DepotContract
}

func NewStorageManager(env string) *StorageManager {
    sm := &StorageManager{}
    
    if env == "production" {
        // Production: Different providers for different use cases
        sm.configs = depot.NewS3(s3ConfigsConfig)
        sm.logs = depot.NewGCS(gcsLogsConfig)
        sm.cache = depot.NewAzure(azureCacheConfig)
    } else {
        // Development: Memory for everything
        sm.configs = depot.NewMemory(depot.MemoryConfig{Name: "configs"})
        sm.logs = depot.NewMemory(depot.MemoryConfig{Name: "logs"})
        sm.cache = depot.NewMemory(depot.MemoryConfig{Name: "cache"})
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
    primary := depot.NewS3(depot.S3Config{
        Region: "us-west-2",
        Bucket: "myapp-primary",
    })
    
    // Backup storage (different region)
    backup := depot.NewS3(depot.S3Config{
        Region: "eu-west-1", 
        Bucket: "myapp-backup",
    })
    
    // Cross-region replication
    primary.Subscribe("*", func(event depot.ChangeEvent) {
        switch event.Type {
        case depot.EventCreate, depot.EventUpdate:
            data, err := primary.Get(event.Key)
            if err == nil {
                backup.Set(event.Key, data, 0)
            }
        case depot.EventDelete:
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
    configStorage := depot.NewS3(s3Config)
    configStorage.Register("config")
    
    logStorage := depot.NewGCS(gcsConfig)
    logStorage.Register("logs")
    
    cacheStorage := depot.NewMemory(memConfig)
    cacheStorage.Register("cache")
    
    // Services can discover storage by purpose
    configContract, _ := depot.GetContract("config")
    logContract, _ := depot.GetContract("logs") 
    cacheContract, _ := depot.GetContract("cache")
}
```

### **Health Monitoring**
```go
func monitorStorageHealth() {
    contracts := depot.List()
    
    for _, contractInfo := range contracts {
        status, err := depot.Status(contractInfo.Alias)
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
    contracts := depot.List()
    
    for _, contract := range contracts {
        log.Printf("Closing storage contract: %s", contract.Alias)
        if storageContract, err := depot.GetContract(contract.Alias); err == nil {
            storageContract.Close()
        }
    }
    
    // Close global depot service
    depot.Close()
}
```

## 🔧 Configuration Reference

### **Memory Provider**
```go
config := depot.MemoryConfig{
    Name: "memory-storage",
    // TTL cleanup interval, max size, etc.
}
contract := depot.NewMemory(config)
```

### **S3 Provider**
```go
config := depot.S3Config{
    Region:    "us-west-2",
    Bucket:    "my-s3-bucket",
    AccessKey: "", // From AWS_ACCESS_KEY_ID env var
    SecretKey: "", // From AWS_SECRET_ACCESS_KEY env var
    UseSSL:    true,
}
contract := depot.NewS3(config)
```

### **GCS Provider**
```go
config := depot.GCSConfig{
    ProjectID:   "my-gcp-project",
    Bucket:      "my-gcs-bucket",
    Credentials: "", // From GOOGLE_APPLICATION_CREDENTIALS env var
}
contract := depot.NewGCS(config)
```

## 📊 Performance Considerations

### **Concurrent Operations**
```go
// Depot contracts are thread-safe
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
func batchOperations(contract depot.DepotContract, items map[string][]byte) error {
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

// After: Depot abstraction
contract := depot.NewS3(depot.S3Config{Bucket: "my-bucket"})
data, err := contract.Get("config.json")
```

### **From File System**
```go
// Before: Local file operations
data, err := os.ReadFile("config.json")
err = os.WriteFile("config.json", data, 0644)

// After: Cloud storage with same semantics
contract := depot.NewS3(s3Config) // or any provider
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
depot.RegisterProvider("custom", func(config map[string]any) depot.DepotProvider {
    return &customProvider{}
})

// Use like any other provider
contract := depot.NewContract("custom", "my-custom", customConfig)
```

### **Middleware Pattern**
```go
func withLogging(contract depot.DepotContract) depot.DepotContract {
    return &loggingWrapper{
        underlying: contract,
        logger:     logger,
    }
}

func withMetrics(contract depot.DepotContract) depot.DepotContract {
    return &metricsWrapper{
        underlying: contract,
        metrics:    metrics,
    }
}

// Chain middleware
contract := withMetrics(withLogging(depot.NewS3(config)))
```

## 🔮 Roadmap

- [ ] **Object Versioning**: Version control for stored objects
- [ ] **Encryption**: Client-side and server-side encryption support
- [ ] **Compression**: Automatic compression for large objects
- [ ] **Caching**: Local caching layer with invalidation
- [ ] **Metrics**: Detailed operation metrics and monitoring
- [ ] **Lifecycle Policies**: Automatic object lifecycle management

---

Depot provides cloud storage abstraction with multi-provider support, reactive capabilities, and contract-based management for building cloud-native applications.