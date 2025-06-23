# HTTP Module

The HTTP module provides HTTP server drivers for ZBZ. It follows the driver pattern where you can plug in different HTTP frameworks (Gin, Echo, FastHTTP, etc.) while maintaining a consistent interface for request handling and middleware.

## Architecture

```
lib/http.go (ZBZ HTTP Service) → HTTPDriver Interface → Concrete Implementations
                                      ↓
                        ┌─────────────────────────────────┐
                        │  gin.go (Gin Driver)            │
                        │  echo.go (Echo Driver)          │
                        │  fiber.go (Fiber Driver)        │
                        └─────────────────────────────────┘
```

## HTTP Driver Interface

All HTTP drivers must implement the `HTTPDriver` interface defined in `driver.go`:

```go
type HTTPDriver interface {
    // Route registration
    AddRoute(method, path string, handler func(RequestContext)) error
    
    // Middleware management
    AddMiddleware(middleware func(RequestContext, func())) error
    
    // Server lifecycle
    Start(address string) error
    Stop() error
    
    // Driver metadata
    DriverName() string
    DriverVersion() string
}
```

## Request Context Interface

ZBZ provides a framework-agnostic `RequestContext` interface that abstracts HTTP request/response operations:

```go
type RequestContext interface {
    // Request information
    Method() string
    Path() string
    PathParam(name string) string
    QueryParam(name string) string
    Header(name string) string
    BodyBytes() ([]byte, error)
    Cookie(name string) (string, error)
    
    // Response operations
    Status(code int)
    SetHeader(name, value string)
    SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)
    JSON(data any) error
    Data(contentType string, data []byte) error
    HTML(name string, data any) error
    Redirect(code int, url string)
    
    // Context storage
    Set(key string, value any)
    Get(key string) (any, bool)
    
    // Framework integration
    Context() context.Context
    Unwrap() any  // Returns underlying framework context
}
```

## Built-in Gin Driver

The module includes a Gin driver (`gin.go`) that provides:

- **OpenTelemetry Integration**: Automatic trace creation with span context
- **Structured Logging**: Request/response logging with trace IDs
- **Performance Optimization**: Release mode by default (no debug spam)
- **Panic Recovery**: Built-in panic recovery middleware
- **ZBZ Integration**: Seamless RequestContext abstraction

### Gin Driver Features

```go
// Create Gin driver
driver := http.NewGinDriver(false) // false = production mode

// Built-in middleware includes:
// - OpenTelemetry tracing with proper span creation
// - Request/response logging with trace IDs
// - Panic recovery
// - Header propagation for distributed tracing
```

## Creating a Custom HTTP Driver

### 1. Implement the HTTPDriver Interface

```go
package myhttp

import (
    "context"
    "zbz/lib/http"
)

type MyHTTPDriver struct {
    server   MyHTTPServer
    routes   []Route
    address  string
}

func NewMyHTTPDriver() http.HTTPDriver {
    return &MyHTTPDriver{
        server: initializeMyServer(),
        routes: make([]Route, 0),
    }
}

func (m *MyHTTPDriver) AddRoute(method, path string, handler func(http.RequestContext)) error {
    // Convert ZBZ handler to your framework's handler format
    frameworkHandler := func(frameworkCtx YourFrameworkContext) {
        // Create RequestContext wrapper
        ctx := NewMyRequestContext(frameworkCtx)
        handler(ctx)
    }
    
    // Register with your framework
    return m.server.AddRoute(method, path, frameworkHandler)
}

func (m *MyHTTPDriver) AddMiddleware(middleware func(http.RequestContext, func())) error {
    // Convert ZBZ middleware to your framework's middleware format
    frameworkMiddleware := func(frameworkCtx YourFrameworkContext, next func()) {
        ctx := NewMyRequestContext(frameworkCtx)
        middleware(ctx, next)
    }
    
    m.server.Use(frameworkMiddleware)
    return nil
}

func (m *MyHTTPDriver) Start(address string) error {
    m.address = address
    return m.server.Listen(address)
}

func (m *MyHTTPDriver) Stop() error {
    return m.server.Shutdown()
}

func (m *MyHTTPDriver) DriverName() string {
    return "myhttp"
}

func (m *MyHTTPDriver) DriverVersion() string {
    return "1.0.0"
}
```

### 2. Implement RequestContext Wrapper

```go
type MyRequestContext struct {
    ctx YourFrameworkContext
}

func NewMyRequestContext(ctx YourFrameworkContext) http.RequestContext {
    return &MyRequestContext{ctx: ctx}
}

func (m *MyRequestContext) Method() string {
    return m.ctx.Request.Method
}

func (m *MyRequestContext) Path() string {
    return m.ctx.Request.URL.Path
}

func (m *MyRequestContext) PathParam(name string) string {
    return m.ctx.Param(name)
}

func (m *MyRequestContext) QueryParam(name string) string {
    return m.ctx.Query(name)
}

func (m *MyRequestContext) Header(name string) string {
    return m.ctx.Request.Header.Get(name)
}

func (m *MyRequestContext) BodyBytes() ([]byte, error) {
    return m.ctx.ReadBody()
}

func (m *MyRequestContext) Cookie(name string) (string, error) {
    cookie, err := m.ctx.Request.Cookie(name)
    if err != nil {
        return "", err
    }
    return cookie.Value, nil
}

func (m *MyRequestContext) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
    cookie := &http.Cookie{
        Name:     name,
        Value:    value,
        MaxAge:   maxAge,
        Path:     path,
        Domain:   domain,
        Secure:   secure,
        HttpOnly: httpOnly,
    }
    m.ctx.Response.Header.Set("Set-Cookie", cookie.String())
}

func (m *MyRequestContext) Status(code int) {
    m.ctx.Response.Status(code)
}

func (m *MyRequestContext) SetHeader(name, value string) {
    m.ctx.Response.Header.Set(name, value)
}

func (m *MyRequestContext) JSON(data any) error {
    return m.ctx.JSON(data)
}

func (m *MyRequestContext) Data(contentType string, data []byte) error {
    m.ctx.Response.Header.Set("Content-Type", contentType)
    _, err := m.ctx.Response.Write(data)
    return err
}

func (m *MyRequestContext) HTML(name string, data any) error {
    return m.ctx.RenderHTML(name, data)
}

func (m *MyRequestContext) Redirect(code int, url string) {
    m.ctx.Redirect(code, url)
}

func (m *MyRequestContext) Set(key string, value any) {
    m.ctx.Set(key, value)
}

func (m *MyRequestContext) Get(key string) (any, bool) {
    return m.ctx.Get(key)
}

func (m *MyRequestContext) Context() context.Context {
    return m.ctx.Request.Context()
}

func (m *MyRequestContext) Unwrap() any {
    return m.ctx
}
```

### 3. Integration with ZBZ

```go
package main

import (
    "zbz/lib"
    "zbz/lib/http"
    myhttp "mypackage/http"
)

func main() {
    engine := zbz.NewEngine()
    
    // Create custom HTTP driver
    httpDriver := myhttp.NewMyHTTPDriver()
    
    // Create HTTP contract
    httpContract := zbz.HTTPContract{
        BaseContract: zbz.BaseContract{
            Name:        "Custom HTTP Server",
            Description: "My custom HTTP implementation",
        },
        Driver: httpDriver,
    }
    
    // Set HTTP provider
    engine.SetHTTP(&httpContract)
    
    // Register handlers
    engine.RegisterHandler(&zbz.HandlerContract{
        Name:        "Health Check",
        Description: "Application health status",
        Method:      "GET",
        Path:        "/health",
        Handler:     healthHandler,
        Auth:        false,
    })
    
    engine.Start(":8080")
}

func healthHandler(ctx zbz.RequestContext) {
    ctx.JSON(map[string]string{"status": "healthy"})
}
```

## Handler Registration

ZBZ handlers are registered through `HandlerContract` which provides metadata and behavior configuration:

```go
type HandlerContract struct {
    Name        string                             // Handler identifier
    Description string                             // Handler purpose
    Method      string                             // HTTP method (GET, POST, etc.)
    Path        string                             // URL path with parameters
    Handler     func(RequestContext)               // Handler function
    Auth        bool                               // Require authentication
    Scope       string                             // Required permission scope
}
```

### Handler Examples

```go
// Simple JSON API endpoint
engine.RegisterHandler(&zbz.HandlerContract{
    Name:        "Get User",
    Description: "Retrieve user by ID",
    Method:      "GET",
    Path:        "/users/:id",
    Handler:     getUserHandler,
    Auth:        true,
    Scope:       "read:users",
})

func getUserHandler(ctx zbz.RequestContext) {
    userID := ctx.PathParam("id")
    
    // Get authenticated user from context
    user, exists := ctx.Get("user")
    if !exists {
        ctx.Status(401)
        return
    }
    
    authUser := user.(*zbz.AuthUser)
    logger.Log.Info("User accessed user data",
        logger.String("requesting_user", authUser.Sub),
        logger.String("target_user", userID))
    
    // Fetch user data...
    userData := map[string]string{
        "id":    userID,
        "name":  "John Doe",
        "email": "john@example.com",
    }
    
    ctx.JSON(userData)
}

// File upload endpoint
engine.RegisterHandler(&zbz.HandlerContract{
    Name:        "Upload File",
    Description: "Upload user avatar",
    Method:      "POST",
    Path:        "/users/:id/avatar",
    Handler:     uploadAvatarHandler,
    Auth:        true,
})

func uploadAvatarHandler(ctx zbz.RequestContext) {
    userID := ctx.PathParam("id")
    
    // Read uploaded file
    fileData, err := ctx.BodyBytes()
    if err != nil {
        ctx.Set("error_message", "Failed to read uploaded file")
        ctx.Status(400)
        return
    }
    
    // Process file upload...
    logger.Log.Info("File uploaded",
        logger.String("user_id", userID),
        logger.Int("file_size", len(fileData)))
    
    ctx.JSON(map[string]string{"message": "Avatar uploaded successfully"})
}
```

## Middleware System

ZBZ provides a layered middleware system with automatic integration:

### Built-in Middleware

1. **Auth Middleware**: Automatic authentication check when `Auth: true`
2. **Scope Middleware**: Permission validation when `Scope: "permission"`
3. **Error Middleware**: Automatic error response formatting
4. **Logging Middleware**: Request/response logging with trace IDs
5. **Tracing Middleware**: OpenTelemetry span creation and propagation

### Custom Middleware

```go
// Global middleware (applied to all routes)
engine.RegisterMiddleware(func(ctx zbz.RequestContext, next func()) {
    // CORS headers
    ctx.SetHeader("Access-Control-Allow-Origin", "*")
    ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
    
    if ctx.Method() == "OPTIONS" {
        ctx.Status(200)
        return
    }
    
    next()
})

// Rate limiting middleware
func rateLimitMiddleware(ctx zbz.RequestContext, next func()) {
    clientIP := ctx.Header("X-Forwarded-For")
    if clientIP == "" {
        clientIP = ctx.Header("X-Real-IP")
    }
    
    // Check rate limit for client IP
    if isRateLimited(clientIP) {
        ctx.Set("error_message", "Rate limit exceeded")
        ctx.Status(429)
        return
    }
    
    next()
}
```

## OpenTelemetry Integration

The HTTP module provides automatic OpenTelemetry integration:

### Automatic Tracing

- **Span Creation**: Every HTTP request creates a new span
- **Trace Propagation**: Extracts trace context from headers
- **Span Attributes**: HTTP method, URL, status code, response size
- **Error Recording**: Automatic error recording for 4xx/5xx responses

### Trace Context Access

```go
func myHandler(ctx zbz.RequestContext) {
    // Get OpenTelemetry context
    otelCtx := ctx.Context()
    
    // Create child span for business logic
    tracer := otel.Tracer("my-service")
    _, span := tracer.Start(otelCtx, "business-operation")
    defer span.End()
    
    // Business logic...
    span.SetAttributes(attribute.String("user_id", "123"))
}
```

## Error Handling

ZBZ provides automatic error response formatting:

```go
func errorProneHandler(ctx zbz.RequestContext) {
    // Simple error responses
    ctx.Set("error_message", "Custom error message")
    ctx.Status(400)
    return // ZBZ automatically formats error response
    
    // Or use default error messages
    ctx.Status(404) // Uses default "Not Found" message
}

// Error responses are automatically formatted as:
// {
//   "error": "Custom error message",
//   "status": 400,
//   "timestamp": "2024-01-15T10:30:00Z"
// }
```

## Testing HTTP Drivers

Test your HTTP driver implementation:

```go
func TestMyHTTPDriver(t *testing.T) {
    driver := myhttp.NewMyHTTPDriver()
    
    // Test route registration
    handler := func(ctx http.RequestContext) {
        ctx.JSON(map[string]string{"message": "test"})
    }
    
    err := driver.AddRoute("GET", "/test", handler)
    assert.NoError(t, err)
    
    // Test middleware registration
    middleware := func(ctx http.RequestContext, next func()) {
        ctx.SetHeader("X-Test", "middleware")
        next()
    }
    
    err = driver.AddMiddleware(middleware)
    assert.NoError(t, err)
    
    // Test server lifecycle
    go func() {
        err := driver.Start(":0") // Random port
        assert.NoError(t, err)
    }()
    
    time.Sleep(100 * time.Millisecond)
    
    err = driver.Stop()
    assert.NoError(t, err)
}

func TestRequestContext(t *testing.T) {
    // Test with actual HTTP request
    req := httptest.NewRequest("GET", "/test?param=value", nil)
    req.Header.Set("Authorization", "Bearer token")
    
    frameworkCtx := createTestContext(req)
    ctx := NewMyRequestContext(frameworkCtx)
    
    // Test request methods
    assert.Equal(t, "GET", ctx.Method())
    assert.Equal(t, "/test", ctx.Path())
    assert.Equal(t, "value", ctx.QueryParam("param"))
    assert.Equal(t, "Bearer token", ctx.Header("Authorization"))
    
    // Test response methods
    ctx.Status(200)
    ctx.SetHeader("Content-Type", "application/json")
    err := ctx.JSON(map[string]string{"test": "data"})
    assert.NoError(t, err)
}
```

## Performance Considerations

### Driver-Specific Optimizations

- **Connection Pooling**: Configure appropriate connection pools
- **Keep-Alive**: Enable HTTP keep-alive for persistent connections
- **Compression**: Enable gzip/deflate compression for responses
- **Static Files**: Use framework-specific static file serving

### Gin Driver Optimizations

```go
// Production configuration
driver := http.NewGinDriver(false) // Release mode

// The Gin driver automatically:
// - Disables debug logging
// - Uses efficient JSON serialization
// - Includes recovery middleware
// - Optimizes route matching
```

### Middleware Ordering

Middleware executes in registration order. Optimize by placing:
1. **CORS/Security headers** - First
2. **Rate limiting** - Early
3. **Authentication** - Before business logic
4. **Logging/Tracing** - Throughout
5. **Error handling** - Last

## Framework-Specific Notes

### Gin Integration
- Automatically handles JSON marshaling
- Built-in template rendering support
- Efficient route parameter extraction
- Middleware composition

### Echo Integration (Future)
- Context-based request handling
- Built-in validation support
- WebSocket support
- HTTP/2 support

### Fiber Integration (Future)
- FastHTTP-based for performance
- Express.js-like API
- Built-in rate limiting
- WebSocket support

## Security Considerations

1. **HTTPS Only**: Always use HTTPS in production
2. **Header Security**: Set appropriate security headers (CORS, CSP, etc.)
3. **Input Validation**: Validate all request inputs
4. **Rate Limiting**: Implement rate limiting for public endpoints
5. **Authentication**: Use ZBZ auth middleware for protected routes
6. **Error Messages**: Don't expose sensitive information in error responses