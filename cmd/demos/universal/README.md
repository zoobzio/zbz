# Universal Data Access Demos

These demos showcase the power of ZBZ's universal data access pattern - one API for all data operations across any provider.

## Available Demos

### 1. Basic Universal Interface Demo
Shows how the same `universal.DataAccess[T]` interface works with different providers:
- **Database**: PostgreSQL tables  
- **Cache**: Redis namespaces
- **Storage**: S3 buckets
- **Search**: Elasticsearch indices

### 2. Provider Orchestration Demo  
Demonstrates how providers orchestrate both universal operations and provider-specific features:
- Universal operations with automatic hooks
- Native provider access with instrumentation
- Provider-specific adapters and plugins

### 3. Cross-Provider Operations Demo
Shows data flowing seamlessly between different providers:
- Database → Cache (read-through caching)
- Storage → Search (content indexing) 
- Cache → Metrics (analytics pipeline)

### 4. Real-Time Sync Demo
Demonstrates flux-powered real-time synchronization:
- Universal subscriptions across providers
- Automatic data propagation
- Event-driven architecture

### 5. Hook Ecosystem Demo
Shows the zero-config observability via capitan hooks:
- Automatic telemetry collection
- Provider-agnostic monitoring
- Event-driven integrations

## Running Demos

```bash
# Run all universal demos
zbz demo universal all

# Run specific demos
zbz demo universal basic
zbz demo universal orchestration  
zbz demo universal cross-provider
zbz demo universal real-time
zbz demo universal hooks
```

## Demo Architecture

All demos use mock providers to avoid external dependencies, but demonstrate real-world patterns that work with production providers.

**Mock Providers Used:**
- `mock-postgres` - Simulates PostgreSQL operations
- `mock-redis` - Simulates Redis operations  
- `mock-s3` - Simulates S3 operations
- `mock-elasticsearch` - Simulates Elasticsearch operations

**Real-World Equivalents:**
```go
// Demo uses mocks
universal.Register("db", mockPostgres, config)

// Production uses real providers  
universal.Register("db", postgres.NewProvider, config)
```

The universal interface remains identical across mock and real providers!