# Universal Data Access Ecosystem Tests

These tests validate the universal data access pattern at the ecosystem level, ensuring the entire system works together seamlessly.

## Test Philosophy

**Ecosystem-Level Testing**: Instead of testing individual packages in isolation, these tests validate:
- **Cross-package integration** - Multiple services working together
- **Universal interface compliance** - All providers implement the same interface correctly  
- **Hook emission consistency** - All operations emit the expected capitan hooks
- **Provider orchestration** - Universal and native operations work together
- **Type safety guarantees** - Compile-time safety across all operations

## Test Categories

### 1. Universal Interface Compliance Tests
Validates that all providers correctly implement `universal.DataAccess[T]`:
- ✅ All CRUD operations work identically
- ✅ URI parsing and routing is consistent
- ✅ Type conversion is handled properly
- ✅ Error handling follows universal patterns

### 2. Provider Orchestration Tests
Validates that providers correctly orchestrate universal and provider-specific operations:
- ✅ Universal operations delegate to provider correctly
- ✅ Native operations are instrumented with hooks
- ✅ GetNative() returns properly instrumented clients
- ✅ Provider-specific features work alongside universal features

### 3. Cross-Provider Integration Tests
Validates that data can flow seamlessly between different providers:
- ✅ Database → Cache synchronization
- ✅ Storage → Search indexing
- ✅ Real-time change propagation
- ✅ Type-safe operations across provider boundaries

### 4. Hook Ecosystem Tests
Validates the zero-config observability system:
- ✅ All operations emit expected capitan hooks
- ✅ Hook data contains correct metadata
- ✅ Native operations are properly instrumented
- ✅ Hook listeners receive events correctly

### 5. Performance and Load Tests
Validates system performance under load:
- ✅ Universal interface overhead is minimal
- ✅ Provider orchestration is efficient
- ✅ Memory usage is reasonable
- ✅ Concurrent operations are thread-safe

## Running Tests

```bash
# Run all ecosystem tests
zbz test universal all

# Run specific test categories
zbz test universal compliance
zbz test universal orchestration
zbz test universal integration
zbz test universal hooks
zbz test universal performance

# Run tests for specific providers
zbz test universal --provider=postgres
zbz test universal --provider=redis,s3

# Run tests with different verbosity
zbz test universal --verbose
zbz test universal --quiet
```

## Test Data and Providers

Tests use a combination of:
- **Mock providers** for isolated testing
- **Real providers** for integration testing (when available)
- **Generated test data** for comprehensive coverage
- **User-provided data** for custom validation

## Test Configuration

Tests can be configured via YAML:

```yaml
# test-config.yaml
universal_tests:
  providers:
    - name: postgres
      type: database
      config:
        host: localhost
        port: 5432
        database: test_db
    - name: redis  
      type: cache
      config:
        host: localhost
        port: 6379
  
  test_data:
    user_count: 1000
    session_count: 5000
    document_count: 100
    
  performance:
    max_latency: 100ms
    min_throughput: 1000ops/sec
```

This ensures tests validate real-world usage patterns while remaining reliable and fast.