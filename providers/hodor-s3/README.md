# Hodor S3 Provider

> **Amazon S3 storage provider for production cloud storage**

The S3 provider implements Hodor's storage interface using Amazon S3, providing managed object storage with global availability, durability, and scalability. This provider is ideal for production workloads requiring reliable cloud storage.

## 🚀 Quick Start

```go
import _ "zbz/providers/hodor-s3" // Auto-registers the driver

// Environment-based configuration
os.Setenv("ZBZ_S3_REGION", "us-west-2")
os.Setenv("ZBZ_S3_BUCKET", "my-production-bucket")
os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")

// Create S3 storage contract
contract := hodor.NewContract("s3", "production-storage", nil)

// Use with any Hodor-compatible service
err := contract.Set("config.json", []byte(`{"env": "production"}`), 0)
data, err := contract.Get("config.json")
```

## ✨ Features

### **AWS Native Integration**
- Native AWS SDK integration for optimal performance
- Automatic credential discovery (IAM roles, profiles, environment)
- Multi-region support for global deployments
- Versioning and lifecycle policy support

### **TTL Support via Metadata**
```go
// Set data with expiration using S3 metadata
contract.Set("session:user123", sessionData, 30*time.Minute)

// Automatic TTL checking during operations
exists, _ := contract.Exists("session:user123") // false after 30 minutes
```

### **Automatic Bucket Management**
```go
// Buckets are created automatically if they don't exist
contract := hodor.NewContract("s3", "auto-bucket", map[string]any{
    "region": "us-east-1",
    "bucket": "my-new-bucket", // Created automatically
})
```

### **IAM Integration**
```go
// Works with IAM roles, profiles, and environment credentials
// No hardcoded credentials needed in production

// EC2 instance with IAM role
contract := hodor.NewContract("s3", "ec2-storage", map[string]any{
    "region": "us-west-2",
    "bucket": "my-ec2-bucket",
    // Credentials automatically from instance metadata
})
```

## 🔧 Configuration

### **Environment Variables**
| Variable | Default | Description |
|----------|---------|-------------|
| `ZBZ_S3_REGION` | `us-west-2` | AWS region |
| `ZBZ_S3_BUCKET` | `zbz-storage` | S3 bucket name |
| `AWS_ACCESS_KEY_ID` | - | AWS access key (optional with IAM) |
| `AWS_SECRET_ACCESS_KEY` | - | AWS secret key (optional with IAM) |
| `AWS_PROFILE` | - | AWS profile name |

### **Programmatic Configuration**
```go
config := map[string]any{
    "region":     "us-west-2",
    "bucket":     "my-production-bucket",
    "access_key": "AKIAIOSFODNN7EXAMPLE", 
    "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
}

contract := hodor.NewContract("s3", "production", config)
```

### **AWS Credential Methods**
```go
// 1. Environment variables
os.Setenv("AWS_ACCESS_KEY_ID", "your-access-key")
os.Setenv("AWS_SECRET_ACCESS_KEY", "your-secret-key")

// 2. AWS Profile
os.Setenv("AWS_PROFILE", "production")

// 3. IAM Role (EC2, ECS, Lambda)
// No configuration needed - automatic

// 4. Programmatic
config := map[string]any{
    "access_key": "your-access-key",
    "secret_key": "your-secret-key",
}
```

## ☁️ AWS Deployment Patterns

### **Multi-Region Setup**
```go
func setupMultiRegionStorage() {
    regions := []string{"us-west-2", "eu-west-1", "ap-southeast-1"}
    
    for _, region := range regions {
        config := map[string]any{
            "region": region,
            "bucket": fmt.Sprintf("myapp-storage-%s", region),
        }
        
        contract := hodor.NewContract("s3", fmt.Sprintf("storage-%s", region), config)
        contract.Register(fmt.Sprintf("storage-%s", region))
    }
}
```

### **Environment-Based Buckets**
```go
func setupEnvironmentStorage(env string) hodor.HodorContract {
    bucketName := fmt.Sprintf("myapp-%s", env)
    
    config := map[string]any{
        "region": "us-west-2",
        "bucket": bucketName,
    }
    
    return hodor.NewContract("s3", fmt.Sprintf("%s-storage", env), config)
}

// Usage
prodStorage := setupEnvironmentStorage("production")
stagingStorage := setupEnvironmentStorage("staging")
devStorage := setupEnvironmentStorage("development")
```

### **IAM Role-Based Access**
```go
// EC2 Instance with IAM role
func setupEC2Storage() hodor.HodorContract {
    // No credentials needed - uses instance metadata
    config := map[string]any{
        "region": "us-west-2",
        "bucket": "myapp-ec2-storage",
    }
    
    return hodor.NewContract("s3", "ec2-storage", config)
}

// ECS Task with IAM role
func setupECSStorage() hodor.HodorContract {
    // No credentials needed - uses task metadata
    config := map[string]any{
        "region": os.Getenv("AWS_REGION"),
        "bucket": "myapp-ecs-storage",
    }
    
    return hodor.NewContract("s3", "ecs-storage", config)
}
```

## 🎯 Production Use Cases

### **Configuration Management**
```go
func setupProductionConfig() error {
    config := map[string]any{
        "region": "us-west-2",
        "bucket": "myapp-configs-prod",
    }
    
    storage := hodor.NewContract("s3", "config-storage", config)
    storage.Register("configs")
    
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
func setupS3Logging() {
    config := map[string]any{
        "region": "us-west-2",
        "bucket": "myapp-logs-prod",
    }
    
    storage := hodor.NewContract("s3", "log-storage", config)
    
    zapConfig := zap.Config{
        Name:   "production",
        Level:  "info",
        Format: "json",
    }
    
    logger := zap.NewWithHodor(zapConfig, &storage)
    zlog.Configure(logger.Zlog())
    
    zlog.Info("S3 logging initialized")
}
```

### **Backup Strategy**
```go
func setupBackupStrategy() {
    // Primary storage
    primary := hodor.NewContract("s3", "primary", map[string]any{
        "region": "us-west-2",
        "bucket": "myapp-primary",
    })
    
    // Cross-region backup
    backup := hodor.NewContract("s3", "backup", map[string]any{
        "region": "eu-west-1",
        "bucket": "myapp-backup",
    })
    
    // Implement replication
    primary.Subscribe("*", func(event hodor.ChangeEvent) {
        switch event.Type {
        case hodor.EventCreate, hodor.EventUpdate:
            data, err := primary.Get(event.Key)
            if err == nil {
                backup.Set(event.Key, data, 0)
                log.Printf("Replicated %s to backup region", event.Key)
            }
        case hodor.EventDelete:
            backup.Delete(event.Key)
            log.Printf("Deleted %s from backup region", event.Key)
        }
    })
}
```

## 🔧 Integration Examples

### **Docker Deployment**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o myapp .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/myapp .

# S3 configuration via environment
ENV ZBZ_S3_REGION=us-west-2
ENV ZBZ_S3_BUCKET=myapp-docker-storage

CMD ["./myapp"]
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
      serviceAccountName: myapp-sa
      containers:
      - name: app
        image: myapp:latest
        env:
        - name: ZBZ_S3_REGION
          value: "us-west-2"
        - name: ZBZ_S3_BUCKET
          value: "myapp-k8s-storage"
        # AWS credentials via IRSA (IAM Roles for Service Accounts)
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-sa
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/MyAppS3Role
```

### **Terraform Infrastructure**
```hcl
# S3 bucket for application storage
resource "aws_s3_bucket" "app_storage" {
  bucket = "myapp-${var.environment}-storage"
}

resource "aws_s3_bucket_versioning" "app_storage" {
  bucket = aws_s3_bucket.app_storage.id
  versioning_configuration {
    status = "Enabled"
  }
}

# IAM role for EC2 instances
resource "aws_iam_role" "app_s3_role" {
  name = "myapp-s3-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "app_s3_policy" {
  name = "myapp-s3-policy"
  role = aws_iam_role.app_s3_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.app_storage.arn,
          "${aws_s3_bucket.app_storage.arn}/*"
        ]
      }
    ]
  })
}
```

## 📊 Performance Optimization

### **Connection Pooling**
```go
// AWS SDK automatically handles connection pooling
// Configure via session options for high-throughput apps

func setupHighThroughputS3() hodor.HodorContract {
    // Custom session with optimized settings
    sess := session.Must(session.NewSession(&aws.Config{
        Region:     aws.String("us-west-2"),
        MaxRetries: aws.Int(3),
        HTTPClientOptions: session.HTTPClientOptions{
            Timeout: 30 * time.Second,
        },
    }))
    
    // Use custom session in provider
    config := map[string]any{
        "bucket":  "high-throughput-bucket",
        "session": sess, // Custom session
    }
    
    return hodor.NewContract("s3", "high-throughput", config)
}
```

### **Concurrent Operations**
```go
func concurrentUploads(contract hodor.HodorContract, items map[string][]byte) error {
    semaphore := make(chan struct{}, 20) // Limit to 20 concurrent uploads
    var wg sync.WaitGroup
    errors := make(chan error, len(items))
    
    for key, data := range items {
        wg.Add(1)
        go func(k string, d []byte) {
            defer wg.Done()
            semaphore <- struct{}{}        // Acquire
            defer func() { <-semaphore }() // Release
            
            if err := contract.Set(k, d, 0); err != nil {
                errors <- fmt.Errorf("failed to upload %s: %w", k, err)
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

## ⚠️ Security Best Practices

### **IAM Policies**
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:PutObject",
                "s3:DeleteObject"
            ],
            "Resource": "arn:aws:s3:::myapp-storage/*"
        },
        {
            "Effect": "Allow",
            "Action": "s3:ListBucket",
            "Resource": "arn:aws:s3:::myapp-storage",
            "Condition": {
                "StringLike": {
                    "s3:prefix": ["configs/*", "logs/*"]
                }
            }
        }
    ]
}
```

### **Encryption**
```go
// Server-side encryption configuration
config := map[string]any{
    "region": "us-west-2",
    "bucket": "myapp-encrypted-storage",
    "encryption": map[string]any{
        "type": "AES256", // or "aws:kms"
        "kms_key_id": "arn:aws:kms:us-west-2:123456789012:key/12345678-1234-1234-1234-123456789012",
    },
}
```

### **Access Logging**
```go
func setupAccessLogging() {
    // Enable S3 access logging for security auditing
    config := map[string]any{
        "region": "us-west-2",
        "bucket": "myapp-storage",
        "access_logging": map[string]any{
            "enabled": true,
            "target_bucket": "myapp-access-logs",
            "target_prefix": "access-logs/",
        },
    }
    
    contract := hodor.NewContract("s3", "audited-storage", config)
}
```

## 🔮 Roadmap

- [ ] **S3 Transfer Acceleration**: Support for accelerated uploads
- [ ] **Multipart Upload**: Support for large file uploads
- [ ] **Server-Side Encryption**: Built-in encryption support
- [ ] **Lifecycle Policies**: Automatic object lifecycle management
- [ ] **CloudWatch Metrics**: Integration with AWS CloudWatch
- [ ] **VPC Endpoints**: Support for VPC endpoint configuration

---

The S3 provider enables scalable, managed cloud storage with AWS native integration, supporting everything from development to enterprise production deployments.