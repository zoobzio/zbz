# ZBZ Providers

**Bring-Your-Own-Everything (BYOE) Service Orchestration for Go**

ZBZ provides a unified contract-based architecture where you can swap any external dependency through providers. Database, logging, auth, storage, documentation - all services use contracts with pluggable providers for maximum flexibility and vendor independence.

## 🌟 The ZBZ Pattern

ZBZ follows a consistent **Contract → Service → Provider** pattern across all integrations:

```go
// 1. Contract: YAML-based configuration defining what you want
contract := ServiceContract{
    Provider: "your-choice",     // Which adapter to use
    Config:   {...},             // Adapter-specific configuration
}

// 2. Service: Preprocessing layer with unified interface
service := contract.Service()    // Returns common interface

// 3. Provider: Pluggable implementation for external libraries
// Providers in providers/ directory adapt external libs to ZBZ interfaces
```

### Universal Benefits

- **🔄 Easy Swapping**: Change `provider: "postgres"` to `provider: "mysql"` - same code, different database
- **🛡️ Advanced Preprocessing**: Service layer adds features across ALL adapters (secrets, validation, metrics)
- **📋 Consistent Interface**: One API pattern for databases, logging, auth, storage, documentation
- **🎯 Purpose-Built**: Each provider optimized for its underlying technology's strengths
- **⚡ Performance First**: Service layer preprocessing keeps providers lean and fast

## 🏗️ Available Service Categories

### 🗂️ **Logging (zlog)** - Comprehensive BYOL System

```
providers/
├── zlog-zap/         # Ultra-high performance (50M+ logs/sec)
├── zlog-zerolog/     # Zero allocation logging
├── zlog-logrus/      # Hooks & extensibility
├── zlog-slog/        # Go standard library
├── zlog-simple/      # Zero dependencies, human-readable
└── zlog-apex/        # Beautiful developer experience
```

**Example Usage:**

```yaml
# Switch between any logger with one line change
provider: "zap" # → Ultra performance
# provider: "simple"   # → Human readable
# provider: "zerolog"  # → Zero allocations
```

```go
import _ "zbz/providers/zlog-zap"  // Just import the provider
zlog.Info("Server starting", zlog.String("host", "localhost"))
```

### 🗄️ **Storage (hodor)** - Pluggable Storage Backends

```
providers/
├── hodor-minio/      # S3-compatible object storage
└── hodor-memory/     # In-memory storage for testing
```

**Planned Additions:**

```
providers/
├── hodor-s3/         # Native AWS S3
├── hodor-gcs/        # Google Cloud Storage
├── hodor-azure/      # Azure Blob Storage
└── hodor-local/      # Local filesystem
```

### 🔐 **Authentication** - Unified Auth Providers _(Planned)_

```
providers/
├── auth-auth0/       # Auth0 OIDC integration
├── auth-firebase/    # Firebase Authentication
├── auth-cognito/     # AWS Cognito
├── auth-okta/        # Okta integration
└── auth-local/       # Local user database
```

### 💾 **Database** - Multi-Database Support _(Planned)_

```
providers/
├── database-postgresql/  # PostgreSQL with advanced features
├── database-mysql/       # MySQL/MariaDB support
├── database-sqlite/      # SQLite for embedded apps
├── database-mongodb/     # MongoDB document store
└── database-redis/       # Redis for caching/sessions
```

### 📚 **Documentation (docula)** - API Doc Generation _(Planned)_

```
providers/
├── docs-openapi/     # OpenAPI/Swagger generation
├── docs-postman/     # Postman collection export
├── docs-insomnia/    # Insomnia workspace export
└── docs-graphql/     # GraphQL schema documentation
```

## 🚀 Quick Architecture Overview

### Contract-First Design

```yaml
# services.yaml - Single file configures everything
logging:
  provider: "zap"
  config:
    level: "info"
    outputs: [...]

database:
  provider: "postgresql"
  config:
    dsn: "postgres://..."

auth:
  provider: "auth0"
  config:
    domain: "myapp.auth0.com"

storage:
  provider: "minio"
  config:
    endpoint: "localhost:9000"
```

### Unified Service Interfaces

```go
// Every service follows the same pattern
type Logger interface {
    Info(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    // ... standard methods
}

type Database interface {
    Query(sql string, args ...any) Rows
    Exec(sql string, args ...any) Result
    // ... standard methods
}

type Auth interface {
    Authenticate(token string) (User, error)
    Authorize(user User, resource string) bool
    // ... standard methods
}
```

### Service Layer Benefits

Every service layer provides:

- **🔐 Secret Management**: Automatic redaction of sensitive data
- **🏷️ PII Protection**: Hash personally identifiable information
- **🔗 Correlation**: Distributed tracing integration
- **📊 Metrics**: Automatic performance monitoring
- **✅ Validation**: Input validation and sanitization
- **🛡️ Security**: Rate limiting, audit logging

## 🎯 Real-World Example

### Development Environment

```yaml
# config/development.yaml
logging:
  provider: "simple" # Human-readable console logs

database:
  provider: "sqlite" # Local file database

auth:
  provider: "local" # Simple username/password

storage:
  provider: "memory" # In-memory for testing
```

### Production Environment

```yaml
# config/production.yaml
logging:
  provider: "zerolog" # Zero allocation, high performance
  config:
    level: "warn"
    format: "json"

database:
  provider: "postgresql" # Robust production database
  config:
    dsn: "postgres://prod-db"
    pool_size: 25

auth:
  provider: "auth0" # Enterprise authentication
  config:
    domain: "myapp.auth0.com"

storage:
  provider: "s3" # Scalable cloud storage
  config:
    bucket: "myapp-prod"
```

**Same application code works in both environments!**

## 🔄 Migration Strategy

### From Direct Dependencies

```go
// Before: Direct dependencies lock you in
import (
    "github.com/lib/pq"         // PostgreSQL only
    "go.uber.org/zap"           // Zap only
    "github.com/auth0/go-jwt"   // Auth0 only
)

// After: ZBZ providers give you flexibility
import (
    "zbz/api"
    _ "zbz/providers/database-postgresql"  // Easy to swap
    _ "zbz/providers/zlog-zap"             // Easy to swap
    _ "zbz/providers/auth-auth0"           // Easy to swap
)
```

### Gradual Adoption

```go
// Migrate service by service
engine := zbz.NewEngine()

// Migrate logging first
engine.SetLogger(contract.Zlog())

// Keep existing database for now
engine.SetDatabase(legacyDB)

// Migrate auth next sprint
engine.SetAuth(contract.Auth())
```

## 🛠️ Adding New Providers

Want to add support for your technology? Follow the pattern:

### 1. Create Provider Directory

```bash
mkdir providers/service-provider
cd providers/service-provider
```

### 2. Implement Service Interface

```go
// provider.go
type providerImpl struct {
    client   ProviderClient
    contract ServiceContract
}

func NewProviderImpl(contract ServiceContract) ServiceInterface {
    return &providerImpl{
        client:   setupProvider(contract.Config),
        contract: contract,
    }
}

func (p *providerImpl) Method(args) (result, error) {
    // Adapt external library to ZBZ interface
    return p.client.ExternalMethod(convertArgs(args))
}

func init() {
    service.RegisterProvider("provider", NewProviderImpl)
}
```

### 3. Add Configuration Support

```go
// Support your provider's specific config options
type ProviderConfig struct {
    Endpoint string `yaml:"endpoint"`
    APIKey   string `yaml:"api_key"`
    Timeout  int    `yaml:"timeout"`
    // ... provider-specific settings
}
```

### 4. Create Documentation

````markdown
# ZBZ Service - Provider Implementation

Brief description of the provider and its use cases.

## Features

- List key features
- Performance characteristics
- Unique capabilities

## Configuration

```yaml
provider: "provider"
config:
  endpoint: "https://api.provider.com"
  api_key: "your-key"
```
````

## When to Choose This Provider

✅ Perfect for: [use cases]
❓ Consider alternatives for: [other use cases]

```

## 🌟 Architecture Principles

### Service Independence
- Each service operates independently
- No dependencies between services
- Services communicate through well-defined interfaces
- Easy to replace, test, or mock individual services

### Contract-Driven Configuration
- YAML contracts define service requirements
- Single source of truth for service configuration
- Environment-specific overrides supported
- Validation ensures contracts are complete and correct

### Provider Simplicity
- Providers focus solely on translating between ZBZ interfaces and external libraries
- No business logic in providers
- Service layer handles all preprocessing, validation, security
- Providers stay under 500 lines when possible

### Performance First
- Service layer preprocessing is optimized for speed
- Providers avoid unnecessary allocations
- Lazy initialization where possible
- Benchmarks ensure no performance regression

## 🎉 The ZBZ Advantage

**Traditional Microservice Approach:**
- Different libraries across services
- Inconsistent APIs and patterns
- Vendor lock-in for each service
- Manual security and monitoring setup
- Complex service-to-service configuration

**ZBZ Orchestrated Approach:**
- ✅ Consistent patterns across all services
- ✅ Swap any dependency with configuration change
- ✅ Automatic security, monitoring, correlation
- ✅ Single configuration file for all services
- ✅ Service-level preprocessing benefits
- ✅ Battle-tested providers for proven libraries

## 🚀 The Result

**ZBZ provides the most comprehensive BYOE architecture for Go:**

- 🎯 **Contract-First Design** - Services defined by interfaces, not implementations
- 🔄 **Effortless Swapping** - Change providers without code changes
- 🛡️ **Universal Preprocessing** - Security and monitoring across all services
- ⚡ **Performance Optimized** - Service layer keeps providers lean and fast
- 🌍 **Vendor Independence** - Never get locked into a single technology stack

Build applications that can evolve with your needs, scale with your growth, and adapt to changing requirements.

**That's the power of ZBZ architecture.** 🚀
```
