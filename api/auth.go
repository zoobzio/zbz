package zbz

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"zbz/api/auth"
	"zbz/api/http"
	"zbz/zlog"
)

// Type aliases from auth submodule for consistent API
type AuthDriver = auth.AuthDriver
type AuthToken = auth.AuthToken
type AuthUser = auth.AuthUser

// Auth is the ZBZ service that handles business logic using a driver
type Auth interface {
	// Framework-agnostic middleware functions
	Middleware() func(http.RequestContext, func())
	ScopeMiddleware(scope string) func(http.RequestContext, func())
	EnsureAuthMiddleware() func(http.RequestContext, func())
	
	// Direct handler functions for silent registration
	LoginHandler(ctx RequestContext)
	CallbackHandler(ctx RequestContext) 
	LogoutHandler(ctx RequestContext)
	
	// Handler contracts for engine registration (deprecated - use direct handlers)
	LoginContract() *HandlerContract
	CallbackContract() *HandlerContract
	LogoutContract() *HandlerContract
	
	// Token validation
	ValidateToken(tokenString string) (*AuthToken, error)
	
	// Service metadata
	ContractName() string
	ContractDescription() string
}

// zAuth implements Auth service using an AuthDriver
type zAuth struct {
	driver              AuthDriver
	contractName        string
	contractDescription string
}

// NewAuth creates an Auth service with the provided driver
func NewAuth(driver AuthDriver, name, description string) Auth {
	zlog.Zlog.Info("Creating auth service", 
		zlog.String("name", name),
		zlog.String("driver", driver.DriverName()))
	
	return &zAuth{
		driver:              driver,
		contractName:        name,
		contractDescription: description,
	}
}

// Middleware returns a framework-agnostic middleware function
func (a *zAuth) Middleware() func(http.RequestContext, func()) {
	return func(ctx http.RequestContext, next func()) {
		// Extract token from cookie
		tokenString, err := ctx.Cookie("auth_token")
		if err != nil {
			zlog.Zlog.Debug("No auth token cookie found")
			ctx.Set("error_message", "Authentication required")
			ctx.Status(401)
			return
		}

		// Validate token using driver
		token, err := a.driver.ValidateToken(tokenString)
		if err != nil {
			zlog.Zlog.Debug("Token validation failed", zlog.Err(err))
			ctx.Set("error_message", "Invalid authentication token")
			ctx.Status(401)
			return
		}

		// Try to get user data from cache via driver
		userData, err := a.driver.GetUserFromCache(token.Sub)
		if err != nil {
			// Cache miss - create userData from token
			userData = &AuthUser{
				Sub:         token.Sub,
				Email:       token.Email,
				Name:        token.Name,
				Permissions: token.Permissions,
			}

			// Cache for future requests via driver
			if cacheErr := a.driver.CacheUserData(token.Sub, userData, 24*60*60); cacheErr != nil {
				zlog.Zlog.Warn("Failed to cache user data", zlog.Err(cacheErr))
			}
		}

		// Attach user and permissions to context
		ctx.Set("user", userData)
		ctx.Set("permissions", userData.Permissions)
		ctx.Set("auth_token", token)

		next()
	}
}

// ScopeMiddleware creates middleware that checks for specific scope permissions
func (a *zAuth) ScopeMiddleware(scope string) func(http.RequestContext, func()) {
	return func(ctx http.RequestContext, next func()) {
		permissions, exists := ctx.Get("permissions")
		if !exists {
			zlog.Zlog.Debug("No permissions found in context")
			ctx.Set("error_message", "Authentication required")
			ctx.Status(401)
			return
		}

		perms, ok := permissions.([]string)
		if !ok {
			zlog.Zlog.Error("Invalid permissions type in context")
			ctx.Set("error_message", "Invalid permissions data")
			ctx.Status(500)
			return
		}

		// Check if user has required scope
		for _, perm := range perms {
			if perm == scope {
				next()
				return
			}
		}

		// Log access denial for debugging
		if user, exists := ctx.Get("user"); exists {
			if u, ok := user.(*AuthUser); ok {
				zlog.Zlog.Info("Access denied - insufficient permissions",
					zlog.String("user_id", u.Sub),
					zlog.String("required_scope", scope),
					zlog.String("user_permissions", fmt.Sprintf("%v", perms)))
			}
		}

		ctx.Set("error_message", "Insufficient permissions")
		ctx.Status(403)
	}
}

// EnsureAuthMiddleware redirects unauthenticated users to login with return URL
func (a *zAuth) EnsureAuthMiddleware() func(http.RequestContext, func()) {
	return func(ctx http.RequestContext, next func()) {
		// Extract token from cookie
		tokenString, err := ctx.Cookie("auth_token")
		if err != nil {
			// No token - redirect to login with return URL
			returnURL := ctx.Path()
			// Note: We don't have a method to get all query params, so just use the path for now
			loginURL := fmt.Sprintf("/auth/login?return_url=%s", returnURL)
			zlog.Zlog.Debug("No auth token - redirecting to login", 
				zlog.String("return_url", returnURL),
				zlog.String("login_url", loginURL))
			ctx.Redirect(302, loginURL)
			return
		}

		// Validate token using driver
		token, err := a.driver.ValidateToken(tokenString)
		if err != nil {
			// Invalid token - redirect to login with return URL
			returnURL := ctx.Path()
			// Note: We don't have a method to get all query params, so just use the path for now
			loginURL := fmt.Sprintf("/auth/login?return_url=%s", returnURL)
			zlog.Zlog.Debug("Invalid token - redirecting to login", 
				zlog.String("return_url", returnURL),
				zlog.String("login_url", loginURL),
				zlog.Err(err))
			ctx.Redirect(302, loginURL)
			return
		}

		// Try to get user data from cache via driver
		userData, err := a.driver.GetUserFromCache(token.Sub)
		if err != nil {
			// Cache miss - create userData from token
			userData = &AuthUser{
				Sub:         token.Sub,
				Email:       token.Email,
				Name:        token.Name,
				Permissions: token.Permissions,
			}

			// Cache for future requests via driver
			if cacheErr := a.driver.CacheUserData(token.Sub, userData, 24*60*60); cacheErr != nil {
				zlog.Zlog.Warn("Failed to cache user data", zlog.Err(cacheErr))
			}
		}

		// Attach user and permissions to context
		ctx.Set("user", userData)
		ctx.Set("permissions", userData.Permissions)
		ctx.Set("auth_token", token)

		next()
	}
}

// LoginContract returns a handler contract for login
func (a *zAuth) LoginContract() *HandlerContract {
	return &HandlerContract{
		Name:        "Login",
		Description: "Initiate authentication",
		Method:      "GET",
		Path:        "/auth/login",
		Handler:     a.loginHandler,
		Auth:        false,
	}
}

// loginHandler handles the login HTTP request
func (a *zAuth) loginHandler(ctx http.RequestContext) {
	// Check for return_url parameter
	returnURL := ctx.QueryParam("return_url")
	if returnURL == "" {
		returnURL = "/" // Default redirect after login
	}
	
	// Generate state for CSRF protection
	state, err := a.generateState()
	if err != nil {
		zlog.Zlog.Error("Failed to generate auth state", zlog.Err(err))
		ctx.Status(500)
		return
	}
	
	// Get login URL from driver
	loginURL := a.driver.GetLoginURL(state)
	zlog.Zlog.Info("Generated login URL", zlog.String("url", loginURL), zlog.String("state", state))
	
	// Store state and return URL in cookies for validation/redirect
	ctx.SetCookie("auth_state", state, 300, "/", "", false, true) // 5 min expiry
	ctx.SetCookie("return_url", returnURL, 300, "/", "", false, true) // 5 min expiry
	zlog.Zlog.Debug("Set auth cookies", zlog.String("state", state), zlog.String("return_url", returnURL))
	
	// Redirect to auth provider
	zlog.Zlog.Info("Redirecting to auth provider", zlog.String("url", loginURL))
	ctx.Redirect(302, loginURL)
}

// CallbackContract returns a handler contract for auth callback
func (a *zAuth) CallbackContract() *HandlerContract {
	return &HandlerContract{
		Name:        "Auth Callback",
		Description: "Handle authentication callback",
		Method:      "GET",
		Path:        "/auth/callback",
		Handler:     a.callbackHandler,
		Auth:        false,
	}
}

// callbackHandler handles the auth callback HTTP request
func (a *zAuth) callbackHandler(ctx http.RequestContext) {
	// Get code and state from query params
	code := ctx.QueryParam("code")
	state := ctx.QueryParam("state")
	
	if code == "" {
		zlog.Zlog.Error("No authorization code in callback")
		ctx.Status(400)
		return
	}
	
	// Validate state (CSRF protection)
	storedState, err := ctx.Cookie("auth_state")
	if err != nil || storedState != state {
		zlog.Zlog.Error("Invalid auth state", zlog.Err(err))
		ctx.Status(400)
		return
	}
	
	// Exchange code for token using driver
	token, err := a.driver.ExchangeCodeForToken(code, state)
	if err != nil {
		zlog.Zlog.Error("Failed to exchange code for token", zlog.Err(err))
		ctx.Status(400)
		return
	}
	
	// Set auth token cookie
	ctx.SetCookie("auth_token", token.Value, 86400, "/", "", false, true) // 24 hour expiry
	
	// Get return URL from cookie or default to "/"
	returnURL, err := ctx.Cookie("return_url")
	if err != nil || returnURL == "" {
		returnURL = "/"
	}
	
	// Clear state and return_url cookies
	ctx.SetCookie("auth_state", "", -1, "/", "", false, true)
	ctx.SetCookie("return_url", "", -1, "/", "", false, true)
	
	// Redirect to original destination or app root
	zlog.Zlog.Info("Authentication successful, redirecting", zlog.String("return_url", returnURL))
	ctx.Redirect(302, returnURL)
}

// LogoutContract returns a handler contract for logout
func (a *zAuth) LogoutContract() *HandlerContract {
	return &HandlerContract{
		Name:        "Logout",
		Description: "End authentication session",
		Method:      "GET",
		Path:        "/auth/logout",
		Handler:     a.logoutHandler,
		Auth:        false,
	}
}

// logoutHandler handles the logout HTTP request
func (a *zAuth) logoutHandler(ctx http.RequestContext) {
	// Clear auth token cookie
	ctx.SetCookie("auth_token", "", -1, "/", "", false, true)
	
	// Redirect to home or login page
	ctx.Redirect(302, "/")
}

// generateState creates a random state string for CSRF protection
func (a *zAuth) generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ValidateToken validates a token using the driver
func (a *zAuth) ValidateToken(tokenString string) (*AuthToken, error) {
	return a.driver.ValidateToken(tokenString)
}

// LoginHandler handles login requests (public wrapper for silent registration)
func (a *zAuth) LoginHandler(ctx RequestContext) {
	a.loginHandler(ctx)
}

// CallbackHandler handles auth callback requests (public wrapper for silent registration)  
func (a *zAuth) CallbackHandler(ctx RequestContext) {
	a.callbackHandler(ctx)
}

// LogoutHandler handles logout requests (public wrapper for silent registration)
func (a *zAuth) LogoutHandler(ctx RequestContext) {
	a.logoutHandler(ctx)
}

// ContractName returns the service name
func (a *zAuth) ContractName() string {
	return a.contractName
}

// ContractDescription returns the service description
func (a *zAuth) ContractDescription() string {
	return a.contractDescription
}

