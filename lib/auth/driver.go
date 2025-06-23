package auth

// AuthDriver defines the interface that auth adapters must implement
// This is what user-initialized drivers (Auth0, Cognito, Firebase, etc.) implement
// Drivers are pure external service adapters with no HTTP knowledge
type AuthDriver interface {
	// Token operations
	ValidateToken(tokenString string) (*AuthToken, error)

	// OAuth flow operations
	GetLoginURL(state string) string
	ExchangeCodeForToken(code, state string) (*AuthToken, error)

	// User data management
	GetUserFromCache(userID string) (*AuthUser, error)
	CacheUserData(userID string, userData *AuthUser, ttl int) error

	// Driver metadata
	DriverName() string
	DriverVersion() string
}

// AuthToken represents an authentication token with claims
type AuthToken struct {
	Value       string   `json:"value"`       // Token string
	Sub         string   `json:"sub"`         // Subject (user ID)
	Email       string   `json:"email"`       // User email
	Name        string   `json:"name"`        // User name
	Permissions []string `json:"permissions"` // User permissions
	ExpiresAt   string   `json:"expires_at"`  // Token expiration
}

// AuthUser represents user data in the authentication system
type AuthUser struct {
	Sub         string   `json:"sub"`         // Subject identifier (unique user ID)
	Email       string   `json:"email"`       // User email
	Name        string   `json:"name"`        // User display name
	Permissions []string `json:"permissions"` // User permissions/scopes
	UpdatedAt   string   `json:"updated_at"`  // Last updated timestamp
}

// AuthConfig defines configuration for auth services
type AuthConfig struct {
	// OAuth2 configuration
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Domain       string
	Scopes       []string

	// Service-specific config
	Config map[string]any
}

// AuthLifecycleHooks defines hooks for auth events
type AuthLifecycleHooks struct {
	// Called after successful login
	OnLogin func(user *AuthUser) error

	// Called after successful logout
	OnLogout func(user *AuthUser) error

	// Called when user is first authenticated
	OnFirstLogin func(user *AuthUser) error

	// Called on token refresh
	OnTokenRefresh func(user *AuthUser) error
}
