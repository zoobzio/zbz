package rocco

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Auth is the main authentication interface following ZBZ principles
type Auth interface {
	// Provider management
	RegisterProvider(name string, provider Provider) error
	GetProvider(name string) (Provider, error)
	SetDefaultProvider(name string) error
	
	// Authentication flow
	Authenticate(ctx context.Context, credentials Credentials) (*Identity, error)
	Refresh(ctx context.Context, refreshToken string) (*Identity, error)
	Revoke(ctx context.Context, token string) error
	
	// Authorization
	Authorize(ctx context.Context, identity *Identity, resource Resource) error
	
	// Middleware
	Middleware() func(http.Handler) http.Handler
	BouncerMiddleware(rules ...BouncerRule) func(http.Handler) http.Handler
}

// Provider interface for pluggable auth backends
type Provider interface {
	// Provider metadata
	Name() string
	Type() ProviderType
	
	// Authentication
	Authenticate(ctx context.Context, credentials Credentials) (*Identity, error)
	Refresh(ctx context.Context, refreshToken string) (*Identity, error)
	Revoke(ctx context.Context, token string) error
	
	// User management (optional)
	CreateUser(ctx context.Context, user UserInfo) error
	UpdateUser(ctx context.Context, userID string, user UserInfo) error
	DeleteUser(ctx context.Context, userID string) error
	GetUser(ctx context.Context, userID string) (*UserInfo, error)
	
	// Provider-specific configuration
	Configure(config ProviderConfig) error
	Validate() error
}

// ProviderType defines the type of authentication provider
type ProviderType string

const (
	ProviderTypeOIDC      ProviderType = "oidc"
	ProviderTypeOAuth2    ProviderType = "oauth2"
	ProviderTypeSAML      ProviderType = "saml"
	ProviderTypeBasic     ProviderType = "basic"
	ProviderTypeAPIKey    ProviderType = "api_key"
	ProviderTypeJWT       ProviderType = "jwt"
	ProviderTypeCustom    ProviderType = "custom"
)

// Credentials represents authentication credentials
type Credentials struct {
	Type     string            // "password", "token", "oauth_code", etc.
	Username string            
	Password string            
	Token    string            
	Code     string            // OAuth authorization code
	State    string            // OAuth state parameter
	Extra    map[string]string // Provider-specific fields
}

// Identity represents an authenticated user/service
type Identity struct {
	// Core identity
	ID          string            `json:"id"`
	Provider    string            `json:"provider"`
	Username    string            `json:"username"`
	Email       string            `json:"email"`
	DisplayName string            `json:"display_name"`
	
	// Authentication details
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	TokenType    string            `json:"token_type"`
	ExpiresAt    time.Time         `json:"expires_at"`
	
	// Authorization data
	Roles       []string          `json:"roles"`
	Permissions []string          `json:"permissions"`
	Scopes      []string          `json:"scopes"`
	Groups      []string          `json:"groups"`
	
	// Metadata
	Attributes  map[string]any    `json:"attributes,omitempty"`
	Claims      map[string]any    `json:"claims,omitempty"`
	
	// Session tracking
	SessionID   string            `json:"session_id,omitempty"`
	IssuedAt    time.Time         `json:"issued_at"`
	LastActive  time.Time         `json:"last_active"`
}

// Resource represents a protected resource for authorization
type Resource struct {
	Type       string            // "api", "page", "data", etc.
	ID         string            // Resource identifier
	Action     string            // "read", "write", "delete", etc.
	Attributes map[string]any    // Resource-specific attributes
}

// BouncerRule defines content-aware authorization rules
type BouncerRule struct {
	Name        string
	Description string
	
	// Matchers
	PathPattern string            // URL path pattern (supports wildcards)
	Methods     []string          // HTTP methods
	Headers     map[string]string // Required headers
	
	// Requirements
	RequireAuth      bool     // Requires authentication
	RequiredRoles    []string // Required roles (OR)
	RequiredScopes   []string // Required scopes (OR)
	RequiredClaims   map[string]any // Required claims
	
	// Content-aware rules
	ContentInspector func(r *http.Request) (Resource, error) // Extract resource from request
	Authorizer       func(identity *Identity, resource Resource) error // Custom authorization logic
	
	// Actions
	OnSuccess func(w http.ResponseWriter, r *http.Request, identity *Identity)
	OnFailure func(w http.ResponseWriter, r *http.Request, err error)
}

// UserInfo represents user profile information
type UserInfo struct {
	ID          string            `json:"id"`
	Username    string            `json:"username"`
	Email       string            `json:"email"`
	DisplayName string            `json:"display_name"`
	FirstName   string            `json:"first_name"`
	LastName    string            `json:"last_name"`
	Picture     string            `json:"picture"`
	Locale      string            `json:"locale"`
	Attributes  map[string]any    `json:"attributes"`
}

// ProviderConfig represents provider-specific configuration
type ProviderConfig struct {
	// Common fields
	Enabled      bool              `json:"enabled"`
	DisplayName  string            `json:"display_name"`
	Description  string            `json:"description"`
	Icon         string            `json:"icon"`
	
	// Provider-specific settings stored as nested config
	Settings     map[string]any    `json:"settings"`
}

// Session represents an active user session
type Session struct {
	ID           string            `json:"id"`
	IdentityID   string            `json:"identity_id"`
	Provider     string            `json:"provider"`
	
	CreatedAt    time.Time         `json:"created_at"`
	LastActive   time.Time         `json:"last_active"`
	ExpiresAt    time.Time         `json:"expires_at"`
	
	IPAddress    string            `json:"ip_address"`
	UserAgent    string            `json:"user_agent"`
	
	Data         map[string]any    `json:"data"`
}

// Errors
var (
	ErrProviderNotFound      = fmt.Errorf("authentication provider not found")
	ErrInvalidCredentials    = fmt.Errorf("invalid credentials")
	ErrTokenExpired          = fmt.Errorf("token expired")
	ErrUnauthorized          = fmt.Errorf("unauthorized")
	ErrForbidden             = fmt.Errorf("forbidden")
	ErrProviderMisconfigured = fmt.Errorf("provider misconfigured")
)

// Context keys
type contextKey string

const (
	ContextKeyIdentity contextKey = "auth:identity"
	ContextKeySession  contextKey = "auth:session"
	ContextKeyProvider contextKey = "auth:provider"
)

// GetIdentity retrieves identity from context
func GetIdentity(ctx context.Context) (*Identity, bool) {
	identity, ok := ctx.Value(ContextKeyIdentity).(*Identity)
	return identity, ok
}

// WithIdentity adds identity to context
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, ContextKeyIdentity, identity)
}