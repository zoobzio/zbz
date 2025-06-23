package zbz

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"zbz/lib/auth"
	"zbz/lib/http"
	"zbz/shared/logger"
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
	
	// Handler contracts for engine registration
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
	logger.Info("Creating auth service", 
		logger.String("name", name),
		logger.String("driver", driver.DriverName()))
	
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
			logger.Debug("No auth token cookie found")
			ctx.Set("error_message", "Authentication required")
			ctx.Status(401)
			return
		}

		// Validate token using driver
		token, err := a.driver.ValidateToken(tokenString)
		if err != nil {
			logger.Debug("Token validation failed", logger.Err(err))
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
				logger.Warn("Failed to cache user data", logger.Err(cacheErr))
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
			logger.Debug("No permissions found in context")
			ctx.Set("error_message", "Authentication required")
			ctx.Status(401)
			return
		}

		perms, ok := permissions.([]string)
		if !ok {
			logger.Error("Invalid permissions type in context")
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
				logger.Info("Access denied - insufficient permissions",
					logger.String("user_id", u.Sub),
					logger.String("required_scope", scope),
					logger.String("user_permissions", fmt.Sprintf("%v", perms)))
			}
		}

		ctx.Set("error_message", "Insufficient permissions")
		ctx.Status(403)
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
	// Generate state for CSRF protection
	state, err := a.generateState()
	if err != nil {
		logger.Error("Failed to generate auth state", logger.Err(err))
		ctx.Status(500)
		return
	}
	
	// Get login URL from driver
	loginURL := a.driver.GetLoginURL(state)
	
	// Store state in session/cookie for validation
	ctx.SetCookie("auth_state", state, 300, "/", "", false, true) // 5 min expiry
	
	// Redirect to auth provider
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
		logger.Error("No authorization code in callback")
		ctx.Status(400)
		return
	}
	
	// Validate state (CSRF protection)
	storedState, err := ctx.Cookie("auth_state")
	if err != nil || storedState != state {
		logger.Error("Invalid auth state", logger.Err(err))
		ctx.Status(400)
		return
	}
	
	// Exchange code for token using driver
	token, err := a.driver.ExchangeCodeForToken(code, state)
	if err != nil {
		logger.Error("Failed to exchange code for token", logger.Err(err))
		ctx.Status(400)
		return
	}
	
	// Set auth token cookie
	ctx.SetCookie("auth_token", token.Value, 86400, "/", "", false, true) // 24 hour expiry
	
	// Clear state cookie
	ctx.SetCookie("auth_state", "", -1, "/", "", false, true)
	
	// Redirect to app
	ctx.Redirect(302, "/")
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

// ContractName returns the service name
func (a *zAuth) ContractName() string {
	return a.contractName
}

// ContractDescription returns the service description
func (a *zAuth) ContractDescription() string {
	return a.contractDescription
}

