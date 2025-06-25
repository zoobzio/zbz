# Hodor MinIO Provider

> **S3-compatible object storage provider for scalable cloud storage**

The MinIO provider implements Hodor's storage interface using MinIO, an S3-compatible object storage service. This provider enables cloud-native storage capabilities while supporting both self-hosted MinIO instances and S3-compatible services.

## 🚀 Quick Start

```go
import _ "zbz/providers/hodor-minio" // Auto-registers the driver

// Environment-based configuration
os.Setenv("ZBZ_MINIO_ENDPOINT", "localhost:9000")
os.Setenv("ZBZ_MINIO_ACCESS_KEY", "minioadmin")
os.Setenv("ZBZ_MINIO_SECRET_KEY", "minioadmin")
os.Setenv("ZBZ_MINIO_BUCKET", "my-app-storage")

// Create MinIO storage contract
contract := hodor.NewContract("minio", "production-storage", nil)

// Use with any Hodor-compatible service
err := contract.Set("config.json", []byte(`{"env": "production"}`), 0)
data, err := contract.Get("config.json")
```

## ✨ Features

### **S3 Compatibility**
- Compatible with Amazon S3, MinIO, DigitalOcean Spaces, Wasabi, and other S3-compatible services
- Standard S3 operations with familiar semantics
- Multi-region support for global deployments

### **TTL Support via Metadata**
```go
// Set data with expiration using S3 metadata
contract.Set("session:user123", sessionData, 30*time.Minute)

// Automatic TTL checking and cleanup
exists, _ := contract.Exists("session:user123") // false after 30 minutes
```

### **Automatic Bucket Management**
```go
// Buckets are created automatically if they don't exist
contract := hodor.NewContract("minio", "auto-bucket", map[string]any{
    "endpoint":   "s3.amazonaws.com",
    "bucket":     "my-new-bucket", // Created automatically
    "access_key": "AKIAIOSFODNN7EXAMPLE",
    "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "ssl":        true,
})
```

### **Environment-Driven Configuration**
```go
// Configuration via environment variables (12-factor app pattern)
export ZBZ_MINIO_ENDPOINT="minio.example.com:9000"
export ZBZ_MINIO_ACCESS_KEY="your-access-key"
export ZBZ_MINIO_SECRET_KEY="your-secret-key"
export ZBZ_MINIO_BUCKET="production-storage"
export ZBZ_MINIO_SSL="true"

// No configuration needed in code
contract := hodor.NewContract("minio", "prod", nil)
```

## 🔧 Configuration

### **Environment Variables**
| Variable | Default | Description |
|----------|---------|-------------|
| `ZBZ_MINIO_ENDPOINT` | `localhost:9000` | MinIO/S3 endpoint |
| `ZBZ_MINIO_ACCESS_KEY` | `minioadmin` | Access key ID |
| `ZBZ_MINIO_SECRET_KEY` | `minioadmin` | Secret access key |
| `ZBZ_MINIO_BUCKET` | `zbz-storage` | Bucket name |
| `ZBZ_MINIO_SSL` | `false` | Enable HTTPS |

### **Programmatic Configuration**
```go
config := map[string]any{
    "endpoint":   "s3.amazonaws.com",
    "access_key": "AKIAIOSFODNN7EXAMPLE", 
    "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "bucket":     "my-production-bucket",
    "ssl":        true,
}

contract := hodor.NewContract("minio", "production", config)
```

## ☁️ Cloud Provider Setup

### **Amazon S3**
```go
config := map[string]any{
    "endpoint":   "s3.amazonaws.com",
    "access_key": os.Getenv("AWS_ACCESS_KEY_ID"),
    "secret_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
    "bucket":     "my-s3-bucket",
    "ssl":        true,
}

s3Contract := hodor.NewContract("minio", "s3-storage", config)
```

### **DigitalOcean Spaces**
```go
config := map[string]any{
    "endpoint":   "nyc3.digitaloceanspaces.com",
    "access_key": os.Getenv("DO_SPACES_KEY"),
    "secret_key": os.Getenv("DO_SPACES_SECRET"),
    "bucket":     "my-space-name",
    "ssl":        true,
}

spacesContract := hodor.NewContract("minio", "do-spaces", config)
```

### **Self-Hosted MinIO**
```bash
# Start MinIO server
docker run -p 9000:9000 -p 9001:9001 \
  -e "MINIO_ROOT_USER=minioadmin" \
  -e "MINIO_ROOT_PASSWORD=minioadmin" \
  minio/minio server /data --console-address ":9001"
```

```go
config := map[string]any{
    "endpoint":   "localhost:9000",
    "access_key": "minioadmin",
    "secret_key": "minioadmin", 
    "bucket":     "development-storage",
    "ssl":        false,
}

minioContract := hodor.NewContract("minio", "local-minio", config)
```

## 🎯 Production Use Cases

### **Configuration Management**
```go
func setupProductionConfig() error {
    // S3-backed configuration storage
    config := map[string]any{
        "endpoint":   "s3.us-west-2.amazonaws.com",
        "access_key": os.Getenv("AWS_ACCESS_KEY_ID"),
        "secret_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
        "bucket":     "myapp-configs-prod",
        "ssl":        true,
    }
    
    storage := hodor.NewContract("minio", "config-storage", config)
    
    // Reactive configuration with S3 backend
    _, err := flux.Sync[AppConfig](storage, "app.json", func(old, new AppConfig) {
        log.Printf("Configuration updated from S3")
        applyConfiguration(new)
    })
    
    return err
}
```

### **Distributed Logging**
```go
func setupCloudLogging() {
    // Multi-region log storage
    regions := []string{"us-west-2", "eu-west-1", "ap-southeast-1"}
    
    for _, region := range regions {
        config := map[string]any{
            "endpoint":   fmt.Sprintf("s3.%s.amazonaws.com", region),
            "access_key": os.Getenv("AWS_ACCESS_KEY_ID"),
            "secret_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
            "bucket":     fmt.Sprintf("myapp-logs-%s", region),
            "ssl":        true,
        }
        
        storage := hodor.NewContract("minio", fmt.Sprintf("logs-%s", region), config)
        
        zapConfig := zap.Config{
            Name:   fmt.Sprintf("app-%s", region),
            Level:  "info",
            Format: "json",
        }
        
        logger := zap.NewWithHodor(zapConfig, &storage)
        // Configure region-specific logging
    }
}
```

### **Multi-Environment Deployment**
```go
func setupEnvironmentStorage(env string) hodor.HodorContract {
    var bucketName string
    var endpoint string
    
    switch env {
    case "production":
        bucketName = "myapp-prod"
        endpoint = "s3.amazonaws.com"
    case "staging":
        bucketName = "myapp-staging"  
        endpoint = "s3.amazonaws.com"
    case "development":
        bucketName = "myapp-dev"
        endpoint = "localhost:9000"
    }
    
    config := map[string]any{
        "endpoint":   endpoint,
        "access_key": os.Getenv("MINIO_ACCESS_KEY"),
        "secret_key": os.Getenv("MINIO_SECRET_KEY"),
        "bucket":     bucketName,
        "ssl":        env != "development",
    }
    
    return hodor.NewContract("minio", fmt.Sprintf("%s-storage", env), config)
}
```

## 🔧 Integration Examples

### **With Docker Compose**
```yaml
version: '3.8'
services:
  app:
    build: .
    environment:
      - ZBZ_MINIO_ENDPOINT=minio:9000
      - ZBZ_MINIO_ACCESS_KEY=minioadmin
      - ZBZ_MINIO_SECRET_KEY=minioadmin
      - ZBZ_MINIO_BUCKET=app-storage
      - ZBZ_MINIO_SSL=false
    depends_on:
      - minio
      
  minio:
    image: minio/minio
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      - MINIO_ROOT_USER=minioadmin
      - MINIO_ROOT_PASSWORD=minioadmin
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data

volumes:
  minio_data:
```

### **Kubernetes Deployment**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    spec:
      containers:
      - name: app
        image: myapp:latest
        env:
        - name: ZBZ_MINIO_ENDPOINT
          value: "s3.amazonaws.com"
        - name: ZBZ_MINIO_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: access-key
        - name: ZBZ_MINIO_SECRET_KEY  
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: secret-key
        - name: ZBZ_MINIO_BUCKET
          value: "myapp-k8s-storage"
        - name: ZBZ_MINIO_SSL
          value: "true"
```

### **Backup and Disaster Recovery**
```go
func setupBackupStrategy() {
    // Primary storage
    primaryConfig := map[string]any{
        "endpoint":   "s3.us-west-2.amazonaws.com",
        "access_key": os.Getenv("AWS_ACCESS_KEY_ID"),
        "secret_key": os.Getenv("AWS_SECRET_ACCESS_KEY"), 
        "bucket":     "myapp-primary",
        "ssl":        true,
    }
    
    primary := hodor.NewContract("minio", "primary", primaryConfig)
    
    // Backup storage (different region)
    backupConfig := map[string]any{
        "endpoint":   "s3.eu-west-1.amazonaws.com",
        "access_key": os.Getenv("AWS_ACCESS_KEY_ID"),
        "secret_key": os.Getenv("AWS_SECRET_ACCESS_KEY"),
        "bucket":     "myapp-backup",
        "ssl":        true,
    }
    
    backup := hodor.NewContract("minio", "backup", backupConfig)
    
    // Implement cross-region replication
    go replicateToBackup(primary, backup)
}
```

## 📊 Performance Considerations

### **Connection Pooling**
```go
// MinIO client automatically handles connection pooling
// No additional configuration needed for basic use cases

// For high-throughput applications, consider:
// - Multiple contract instances
// - Regional distribution
// - CDN integration for frequently accessed data
```

### **Batch Operations**
```go
func batchUpload(contract hodor.HodorContract, items map[string][]byte) error {
    // Process uploads concurrently for better performance
    var wg sync.WaitGroup
    errors := make(chan error, len(items))
    
    for key, data := range items {
        wg.Add(1)
        go func(k string, d []byte) {
            defer wg.Done()
            if err := contract.Set(k, d, 0); err != nil {
                errors <- fmt.Errorf("failed to upload %s: %w", k, err)
            }
        }(key, data)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for any errors
    for err := range errors {
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

## ⚠️ Security Considerations

### **Credential Management**
```go
// ✅ Good: Use environment variables or secrets management
os.Setenv("ZBZ_MINIO_ACCESS_KEY", getFromSecrets("minio.access_key"))
os.Setenv("ZBZ_MINIO_SECRET_KEY", getFromSecrets("minio.secret_key"))

// ❌ Bad: Hardcoded credentials in source code
config := map[string]any{
    "access_key": "AKIAIOSFODNN7EXAMPLE", // Don't do this
    "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", // Don't do this
}
```

### **SSL/TLS Configuration**
```go
// Always use SSL for production
config := map[string]any{
    "endpoint": "s3.amazonaws.com",
    "ssl":      true, // Required for production
}

// Only disable SSL for local development
if os.Getenv("ENV") == "development" {
    config["ssl"] = false
}
```

### **IAM Policies (AWS S3)**
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:PutObject", 
                "s3:DeleteObject",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::myapp-storage",
                "arn:aws:s3:::myapp-storage/*"
            ]
        }
    ]
}
```

## 🔮 Roadmap

- [ ] **Multipart Upload**: Support for large file uploads
- [ ] **Versioning**: Object versioning and rollback capabilities  
- [ ] **Lifecycle Policies**: Automatic TTL enforcement via S3 lifecycle rules
- [ ] **Encryption**: Client-side and server-side encryption support
- [ ] **Metrics**: Detailed operation metrics and monitoring
- [ ] **Caching**: Local caching layer for frequently accessed objects

---

The MinIO provider enables scalable, cloud-native storage with S3 compatibility, supporting everything from local development to multi-region production deployments.