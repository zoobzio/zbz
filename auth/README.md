# Auth Module

The auth module provides authentication drivers for ZBZ. It follows the driver pattern where you can plug in different authentication providers (Auth0, Firebase, AWS Cognito, etc.) while maintaining a consistent interface.

## Architecture

```
lib/auth.go (ZBZ Auth Service) → AuthDriver Interface → Concrete Implementations
                                      ↓
                        ┌─────────────────────────────────┐
                        │  auth0.go (Auth0 Driver)        │
                        │  firebase.go (Firebase Driver)  │
                        │  custom.go (Custom Driver)      │
                        └─────────────────────────────────┘
```

## Driver Interface

All auth drivers must implement the `AuthDriver` interface defined in `driver.go`:

```go
type AuthDriver interface {
    // Token operations
    ValidateToken(tokenString string) (*AuthToken, error)
    GetLoginURL(state string) string
    ExchangeCodeForToken(code, state string) (*AuthToken, error)
    
    // User operations (cache-based, no database)
    GetUserFromCache(userID string) (*AuthUser, error)
    CacheUserData(userID string, userData *AuthUser, ttl int) error
    
    // Driver metadata
    DriverName() string
    DriverVersion() string
}
```

## Creating a Custom Auth Driver

### 1. Implement the AuthDriver Interface

```go
package myauth

import (
    "context"
    "fmt"
    "zbz/lib/auth"
    "zbz/lib/cache"
)

type MyAuthDriver struct {
    clientID     string
    clientSecret string
    cache        cache.Cache
}

func NewMyAuthDriver(clientID, clientSecret string, cache cache.Cache) auth.AuthDriver {
    return &MyAuthDriver{
        clientID:     clientID,
        clientSecret: clientSecret,
        cache:        cache,
    }
}

func (m *MyAuthDriver) ValidateToken(tokenString string) (*auth.AuthToken, error) {
    // Implement token validation for your provider
    // Example: JWT validation, API call to provider, etc.
    
    // Parse/validate token
    claims, err := m.parseToken(tokenString)
    if err != nil {
        return nil, err
    }
    
    return &auth.AuthToken{
        Sub:         claims.Subject,
        Email:       claims.Email,
        Name:        claims.Name,
        Permissions: claims.Scopes,
        ExpiresAt:   claims.ExpiresAt,
    }, nil
}

func (m *MyAuthDriver) GetLoginURL(state string) string {
    // Build authorization URL for your provider
    return fmt.Sprintf("https://myprovider.com/oauth/authorize?client_id=%s&state=%s&redirect_uri=%s",
        m.clientID, state, "http://localhost:8080/auth/callback")
}

func (m *MyAuthDriver) ExchangeCodeForToken(code, state string) (*auth.AuthToken, error) {
    // Exchange authorization code for access token
    // Make HTTP request to your provider's token endpoint
    
    resp, err := m.callTokenEndpoint(code)
    if err != nil {
        return nil, err
    }
    
    return m.ValidateToken(resp.AccessToken)
}

func (m *MyAuthDriver) GetUserFromCache(userID string) (*auth.AuthUser, error) {
    ctx := context.Background()
    data, err := m.cache.Get(ctx, "user:"+userID)
    if err != nil {
        return nil, err
    }
    
    // Deserialize cached user data
    user := &auth.AuthUser{}
    err = json.Unmarshal(data, user)
    return user, err
}

func (m *MyAuthDriver) CacheUserData(userID string, userData *auth.AuthUser, ttl int) error {
    ctx := context.Background()
    data, err := json.Marshal(userData)
    if err != nil {
        return err
    }
    
    return m.cache.Set(ctx, "user:"+userID, data, ttl)
}

func (m *MyAuthDriver) DriverName() string {
    return "myauth"
}

func (m *MyAuthDriver) DriverVersion() string {
    return "1.0.0"
}
```

### 2. Integration with ZBZ

```go
package main

import (
    "zbz/lib"
    "zbz/lib/cache"
    "myauth"
)

func main() {
    engine := zbz.NewEngine()
    
    // Set up cache for user data
    cacheContract := zbz.CacheContract{
        BaseContract: zbz.BaseContract{
            Name:        "auth-cache",
            Description: "Cache for authentication data",
        },
        Service: "redis",
        URL:     "redis://localhost:6379",
    }
    
    cache := cacheContract.Cache()
    
    // Create auth driver
    authDriver := myauth.NewMyAuthDriver("client-id", "client-secret", cache)
    
    // Create auth contract
    authContract := zbz.AuthContract{
        BaseContract: zbz.BaseContract{
            Name:        "primary-auth",
            Description: "Primary authentication service",
        },
        Driver: authDriver,
    }
    
    // Set auth provider
    engine.SetAuth(&authContract)
    
    engine.Start(":8080")
}
```

## Built-in Auth0 Driver

The module includes an Auth0 driver (`auth0.go`) that supports:

- **OIDC/OAuth2** authentication flow
- **JWT token validation** with Auth0 public keys
- **User caching** via Redis/Memory cache
- **Scope-based permissions**

### Auth0 Configuration

```go
authDriver := auth.NewAuth0Auth(cache, &auth.AuthConfig{
    Domain:       "your-domain.auth0.com",
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    CallbackURL:  "http://localhost:8080/auth/callback",
})
```

## User Management

**Important**: Users are **cache-only** in ZBZ. No user database tables are created. Authentication data is stored in the cache service (Redis/Memory) for fast access.

### User Data Structure

```go
type AuthUser struct {
    Sub         string   `json:"sub"`         // Subject identifier (unique user ID)
    Email       string   `json:"email"`       // User email
    Name        string   `json:"name"`        // Display name
    Permissions []string `json:"permissions"` // User permissions/scopes
}
```

### Accessing User Data

```go
// In your handlers
func MyHandler(ctx zbz.RequestContext) {
    user, exists := ctx.Get("user")
    if !exists {
        ctx.Status(401)
        return
    }
    
    authUser := user.(*zbz.AuthUser)
    logger.Log.Info("User accessed resource", 
        logger.String("user_id", authUser.Sub),
        logger.String("email", authUser.Email))
}
```

## Security Best Practices

1. **Token Storage**: Store tokens securely (HTTP-only cookies, secure headers)
2. **State Parameter**: Always use state parameter to prevent CSRF
3. **Token Validation**: Validate tokens on every request
4. **Cache TTL**: Set appropriate TTL for cached user data
5. **HTTPS Only**: Always use HTTPS in production
6. **Scope Validation**: Check user permissions for protected resources

## Middleware Integration

The auth service provides framework-agnostic middleware that integrates with ZBZ's HTTP service:

```go
// Authentication middleware (checks if user is logged in)
authService.Middleware()

// Authorization middleware (checks specific permissions)
authService.ScopeMiddleware("read:users")
```

These are automatically integrated when you register handler contracts with `Auth: true` or `Scope: "permission"`.

## Testing

Test your auth driver with:

```go
func TestMyAuthDriver(t *testing.T) {
    cache := cache.NewMemoryCache()
    driver := myauth.NewMyAuthDriver("test-id", "test-secret", cache)
    
    // Test token validation
    token, err := driver.ValidateToken("valid-token")
    assert.NoError(t, err)
    assert.Equal(t, "user@example.com", token.Email)
    
    // Test user caching
    user := &auth.AuthUser{
        Sub:   "123",
        Email: "test@example.com",
        Name:  "Test User",
    }
    
    err = driver.CacheUserData("123", user, 3600)
    assert.NoError(t, err)
    
    cached, err := driver.GetUserFromCache("123")
    assert.NoError(t, err)
    assert.Equal(t, user.Email, cached.Email)
}
```