# Hodor GCS Provider

> **Google Cloud Storage provider for scalable cloud object storage**

The GCS provider implements Hodor's storage interface using Google Cloud Storage, providing managed object storage with global availability, strong consistency, and integration with Google Cloud ecosystem. This provider is ideal for applications deployed on Google Cloud Platform or requiring GCS-specific features.

## 🚀 Quick Start

```go
import _ "zbz/providers/hodor-gcs" // Auto-registers the driver

// Environment-based configuration
os.Setenv("ZBZ_GCS_PROJECT_ID", "my-gcp-project")
os.Setenv("ZBZ_GCS_BUCKET", "my-production-bucket")
os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/path/to/service-account.json")

// Create GCS storage contract
contract := hodor.NewContract("gcs", "production-storage", nil)

// Use with any Hodor-compatible service
err := contract.Set("config.json", []byte(`{"env": "production"}`), 0)
data, err := contract.Get("config.json")
```

## ✨ Features

### **Google Cloud Native Integration**
- Native Google Cloud SDK integration for optimal performance
- Automatic credential discovery (ADC, service accounts, metadata server)
- Multi-region support with global consistency
- Integration with Google Cloud IAM and logging

### **TTL Support via Metadata**
```go
// Set data with expiration using GCS metadata
contract.Set("session:user123", sessionData, 30*time.Minute)

// Automatic TTL checking during operations
exists, _ := contract.Exists("session:user123") // false after 30 minutes
```

### **Automatic Bucket Management**
```go
// Buckets are created automatically if they don't exist
contract := hodor.NewContract("gcs", "auto-bucket", map[string]any{
    "project_id": "my-gcp-project",
    "bucket":     "my-new-bucket", // Created automatically
})
```

### **Application Default Credentials (ADC)**
```go
// Works with ADC, service accounts, and metadata server
// No hardcoded credentials needed in production

// GCE instance with service account
contract := hodor.NewContract("gcs", "gce-storage", map[string]any{
    "project_id": "my-project",
    "bucket":     "my-gce-bucket",
    // Credentials automatically from metadata server
})
```

## 🔧 Configuration

### **Environment Variables**
| Variable | Default | Description |
|----------|---------|-------------|
| `ZBZ_GCS_PROJECT_ID` | - | Google Cloud project ID |
| `ZBZ_GCS_BUCKET` | `zbz-storage` | GCS bucket name |
| `GOOGLE_CLOUD_PROJECT` | - | Alternative project ID variable |
| `GOOGLE_APPLICATION_CREDENTIALS` | - | Path to service account JSON |

### **Programmatic Configuration**
```go
config := map[string]any{
    "project_id":  "my-gcp-project",
    "bucket":      "my-production-bucket",
    "credentials": "/path/to/service-account.json",
}

contract := hodor.NewContract("gcs", "production", config)
```

### **Google Cloud Credential Methods**
```go
// 1. Service Account JSON file
os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/path/to/service-account.json")

// 2. Application Default Credentials (ADC)
// No configuration needed - automatic discovery

// 3. GCE/GKE metadata server
// No configuration needed - automatic

// 4. Programmatic
config := map[string]any{
    "credentials": "/path/to/service-account.json",
}
```

## ☁️ Google Cloud Deployment Patterns

### **Multi-Region Setup**
```go
func setupMultiRegionStorage() {
    regions := []string{"us-central1", "europe-west1", "asia-east1"}
    
    for _, region := range regions {
        config := map[string]any{
            "project_id": "my-gcp-project",
            "bucket":     fmt.Sprintf("myapp-storage-%s", region),
            "location":   region,
        }
        
        contract := hodor.NewContract("gcs", fmt.Sprintf("storage-%s", region), config)
        contract.Register(fmt.Sprintf("storage-%s", region))
    }
}
```

### **Environment-Based Buckets**
```go
func setupEnvironmentStorage(env string) hodor.HodorContract {
    projectID := fmt.Sprintf("myproject-%s", env)
    bucketName := fmt.Sprintf("myapp-%s-storage", env)
    
    config := map[string]any{
        "project_id": projectID,
        "bucket":     bucketName,
    }
    
    return hodor.NewContract("gcs", fmt.Sprintf("%s-storage", env), config)
}

// Usage
prodStorage := setupEnvironmentStorage("prod")
stagingStorage := setupEnvironmentStorage("staging")
devStorage := setupEnvironmentStorage("dev")
```

### **Service Account Integration**
```go
// GCE Instance with service account
func setupGCEStorage() hodor.HodorContract {
    // No credentials needed - uses metadata server
    config := map[string]any{
        "project_id": "my-project",
        "bucket":     "myapp-gce-storage",
    }
    
    return hodor.NewContract("gcs", "gce-storage", config)
}

// GKE Pod with Workload Identity
func setupGKEStorage() hodor.HodorContract {
    config := map[string]any{
        "project_id": os.Getenv("GOOGLE_CLOUD_PROJECT"),
        "bucket":     "myapp-gke-storage",
    }
    
    return hodor.NewContract("gcs", "gke-storage", config)
}
```

## 🎯 Production Use Cases

### **Configuration Management**
```go
func setupProductionConfig() error {
    config := map[string]any{
        "project_id": "my-production-project",
        "bucket":     "myapp-configs-prod",
    }
    
    storage := hodor.NewContract("gcs", "config-storage", config)
    storage.Register("configs")
    
    // Reactive configuration with GCS backend
    _, err := flux.Sync[AppConfig](storage, "app.json", func(old, new AppConfig) {
        log.Printf("Configuration updated from GCS")
        applyConfiguration(new)
    })
    
    return err
}
```

### **Distributed Logging**
```go
func setupGCSLogging() {
    config := map[string]any{
        "project_id": "my-production-project",
        "bucket":     "myapp-logs-prod",
    }
    
    storage := hodor.NewContract("gcs", "log-storage", config)
    
    zapConfig := zap.Config{
        Name:   "production",
        Level:  "info",
        Format: "json",
    }
    
    logger := zap.NewWithHodor(zapConfig, &storage)
    zlog.Configure(logger.Zlog())
    
    zlog.Info("GCS logging initialized")
}
```

### **Data Pipeline Storage**
```go
func setupDataPipeline() {
    // Raw data ingestion
    rawStorage := hodor.NewContract("gcs", "raw-data", map[string]any{
        "project_id": "my-data-project",
        "bucket":     "myapp-raw-data",
    })
    
    // Processed data storage
    processedStorage := hodor.NewContract("gcs", "processed-data", map[string]any{
        "project_id": "my-data-project", 
        "bucket":     "myapp-processed-data",
    })
    
    // ML model storage
    modelStorage := hodor.NewContract("gcs", "ml-models", map[string]any{
        "project_id": "my-ml-project",
        "bucket":     "myapp-ml-models",
    })
    
    // Register all storage contracts
    rawStorage.Register("raw-data")
    processedStorage.Register("processed-data")
    modelStorage.Register("ml-models")
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

# Copy service account credentials
COPY service-account.json /etc/gcp/service-account.json
COPY --from=builder /app/myapp .

# GCS configuration via environment
ENV ZBZ_GCS_PROJECT_ID=my-gcp-project
ENV ZBZ_GCS_BUCKET=myapp-docker-storage
ENV GOOGLE_APPLICATION_CREDENTIALS=/etc/gcp/service-account.json

CMD ["./myapp"]
```

### **Kubernetes Deployment with Workload Identity**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    spec:
      serviceAccountName: myapp-ksa
      containers:
      - name: app
        image: myapp:latest
        env:
        - name: ZBZ_GCS_PROJECT_ID
          value: "my-gcp-project"
        - name: ZBZ_GCS_BUCKET
          value: "myapp-gke-storage"
        # No GOOGLE_APPLICATION_CREDENTIALS needed with Workload Identity
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp-ksa
  annotations:
    iam.gke.io/gcp-service-account: myapp-gsa@my-gcp-project.iam.gserviceaccount.com
```

### **Cloud Run Deployment**
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: myapp
  annotations:
    run.googleapis.com/ingress: all
spec:
  template:
    metadata:
      annotations:
        run.googleapis.com/service-account: myapp-gsa@my-gcp-project.iam.gserviceaccount.com
    spec:
      containers:
      - image: gcr.io/my-gcp-project/myapp:latest
        env:
        - name: ZBZ_GCS_PROJECT_ID
          value: "my-gcp-project"
        - name: ZBZ_GCS_BUCKET
          value: "myapp-cloudrun-storage"
```

### **Terraform Infrastructure**
```hcl
# GCS bucket for application storage
resource "google_storage_bucket" "app_storage" {
  name     = "myapp-${var.environment}-storage"
  location = "US"
  
  versioning {
    enabled = true
  }
  
  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type = "Delete"
    }
  }
}

# Service account for application
resource "google_service_account" "app_sa" {
  account_id   = "myapp-storage"
  display_name = "MyApp Storage Service Account"
}

# IAM binding for bucket access
resource "google_storage_bucket_iam_member" "app_storage_admin" {
  bucket = google_storage_bucket.app_storage.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.app_sa.email}"
}

# Workload Identity binding for GKE
resource "google_service_account_iam_member" "workload_identity" {
  service_account_id = google_service_account.app_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${var.namespace}/${var.k8s_sa_name}]"
}
```

## 📊 Performance Optimization

### **Connection Management**
```go
// GCS client automatically handles connection pooling
// Configure retry and timeout settings for better performance

func setupOptimizedGCS() hodor.HodorContract {
    config := map[string]any{
        "project_id": "my-project",
        "bucket":     "optimized-bucket",
        "retry_config": map[string]any{
            "max_retries": 3,
            "timeout":     "60s",
        },
    }
    
    return hodor.NewContract("gcs", "optimized", config)
}
```

### **Parallel Operations**
```go
func parallelUploads(contract hodor.HodorContract, items map[string][]byte) error {
    const maxConcurrency = 10
    semaphore := make(chan struct{}, maxConcurrency)
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

### **IAM Roles**
```bash
# Minimal IAM roles for application
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:myapp-gsa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.objectAdmin"

# For specific bucket access only
gsutil iam ch serviceAccount:myapp-gsa@PROJECT_ID.iam.gserviceaccount.com:objectAdmin gs://myapp-storage
```

### **Bucket Security**
```go
// Configure bucket with security best practices
config := map[string]any{
    "project_id": "my-secure-project",
    "bucket":     "myapp-secure-storage",
    "security": map[string]any{
        "uniform_bucket_level_access": true,
        "public_access_prevention":    "enforced",
        "encryption": map[string]any{
            "default_kms_key": "projects/my-project/locations/us/keyRings/my-ring/cryptoKeys/my-key",
        },
    },
}
```

### **VPC Security**
```go
func setupVPCSecureStorage() hodor.HodorContract {
    // Configure VPC-native access
    config := map[string]any{
        "project_id": "my-secure-project",
        "bucket":     "myapp-vpc-storage",
        "vpc_config": map[string]any{
            "private_google_access": true,
            "authorized_networks":   []string{"10.0.0.0/8"},
        },
    }
    
    return hodor.NewContract("gcs", "vpc-secure", config)
}
```

## 🔮 Roadmap

- [ ] **Customer-Managed Encryption**: CMEK support for enhanced security
- [ ] **Lifecycle Management**: Integration with GCS lifecycle policies
- [ ] **Transfer Service**: Integration with Storage Transfer Service
- [ ] **BigQuery Integration**: Direct integration with BigQuery for analytics
- [ ] **Cloud Monitoring**: Integration with Google Cloud Monitoring
- [ ] **VPC Service Controls**: Support for VPC Service Controls

---

The GCS provider enables scalable, managed cloud storage with Google Cloud native integration, supporting everything from simple applications to complex data pipelines and ML workloads.