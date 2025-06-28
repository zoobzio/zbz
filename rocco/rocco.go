package rocco

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Global singleton instance for zero-config usage
var (
	defaultInstance Auth
	defaultOnce     sync.Once
)

// Default returns the default Rocco instance, creating it if necessary
func Default() Auth {
	defaultOnce.Do(func() {
		defaultInstance = NewDefault()
	})
	return defaultInstance
}

// NewDefault creates a new Rocco instance with sensible defaults
func NewDefault() Auth {
	auth := NewAuth(
		WithJWTSecret([]byte("rocco-default-secret-change-in-production")),
		WithTokenExpiry(15*time.Minute),
	)
	
	// Register the internal basic provider as default
	basicProvider := newBasicProvider()
	auth.RegisterProvider("basic", basicProvider)
	auth.SetDefaultProvider("basic")
	
	return auth
}

// Convenience functions that use the default instance

// Authenticate authenticates using the default instance
func Authenticate(credentials Credentials) (*Identity, error) {
	return Default().Authenticate(context.Background(), credentials)
}

// Middleware returns middleware using the default instance
func Middleware() func(http.Handler) http.Handler {
	return Default().Middleware()
}

// BouncerMiddleware returns bouncer middleware using the default instance
func BouncerMiddleware(rules ...BouncerRule) func(http.Handler) http.Handler {
	return Default().BouncerMiddleware(rules...)
}

// RegisterProvider registers a provider on the default instance
func RegisterProvider(name string, provider Provider) error {
	return Default().RegisterProvider(name, provider)
}

// SetDefaultProvider sets the default provider on the default instance
func SetDefaultProvider(name string) error {
	return Default().SetDefaultProvider(name)
}

// CreateUser creates a user using the default basic provider
func CreateUser(username, password, email string, roles []string) error {
	basicProvider, err := GetBasicProvider(Default())
	if err != nil {
		return err
	}
	
	userInfo := UserInfo{
		Username:    username,
		Email:       email,
		DisplayName: username,
		Attributes: map[string]any{
			"password":    password,
			"roles":       roles,
		},
	}
	
	return basicProvider.CreateUser(context.Background(), userInfo)
}

// Zero-config bootstrap helpers

// Bootstrap sets up Rocco with common configurations
func Bootstrap(options ...BootstrapOption) error {
	config := &BootstrapConfig{
		CreateDefaultUsers: true,
		EnableBasicAuth:    true,
		JWTSecret:         "rocco-bootstrap-secret",
	}
	
	// Apply options
	for _, opt := range options {
		opt(config)
	}
	
	// Get the default instance
	auth := Default()
	
	// Apply JWT secret if provided
	if config.JWTSecret != "" {
		if zAuth, ok := auth.(*zAuth); ok {
			zAuth.jwtSecret = []byte(config.JWTSecret)
		}
	}
	
	// Create additional default users if requested
	if config.CreateDefaultUsers {
		basicProvider, err := GetBasicProvider(auth)
		if err == nil {
			// Create a regular user
			userInfo := UserInfo{
				Username:    "user",
				Email:       "user@localhost",
				DisplayName: "Default User",
				Attributes: map[string]any{
					"password": "password",
					"roles":    []string{"user"},
				},
			}
			basicProvider.CreateUser(context.Background(), userInfo)
		}
	}
	
	return nil
}

// BootstrapConfig configures the bootstrap process
type BootstrapConfig struct {
	CreateDefaultUsers bool
	EnableBasicAuth    bool
	JWTSecret         string
	AdminUsername     string
	AdminPassword     string
}

// BootstrapOption configures bootstrap
type BootstrapOption func(*BootstrapConfig)

// WithJWTSecretBootstrap sets the JWT secret during bootstrap
func WithJWTSecretBootstrap(secret string) BootstrapOption {
	return func(c *BootstrapConfig) {
		c.JWTSecret = secret
	}
}

// WithDefaultUsers enables/disables default user creation
func WithDefaultUsers(create bool) BootstrapOption {
	return func(c *BootstrapConfig) {
		c.CreateDefaultUsers = create
	}
}

// WithCustomAdmin sets custom admin credentials
func WithCustomAdmin(username, password string) BootstrapOption {
	return func(c *BootstrapConfig) {
		c.AdminUsername = username
		c.AdminPassword = password
	}
}

// Quick setup for common scenarios

// SimpleAuth returns a basic auth setup for development
func SimpleAuth() Auth {
	auth := NewDefault()
	Bootstrap(
		WithJWTSecretBootstrap("simple-auth-secret"),
		WithDefaultUsers(true),
	)
	return auth
}

// ProductionAuth returns a production-ready auth setup (requires configuration)
func ProductionAuth(jwtSecret string) Auth {
	if jwtSecret == "" {
		panic("JWT secret required for production auth")
	}
	
	auth := NewAuth(
		WithJWTSecret([]byte(jwtSecret)),
		WithTokenExpiry(15*time.Minute),
	)
	
	// Still register basic provider but don't create default users
	basicProvider := newBasicProvider()
	auth.RegisterProvider("basic", basicProvider)
	auth.SetDefaultProvider("basic")
	
	return auth
}

// DevAuth returns a development auth setup with relaxed security
func DevAuth() Auth {
	auth := NewDefault()
	Bootstrap(
		WithJWTSecretBootstrap("dev-secret-not-secure"),
		WithDefaultUsers(true),
		WithCustomAdmin("dev", "dev"),
	)
	return auth
}