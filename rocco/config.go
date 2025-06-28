package rocco

import (
	"fmt"
	"time"
)

// Config represents the universal auth configuration
type Config struct {
	// Global auth settings
	Enabled         bool          `json:"enabled" yaml:"enabled" toml:"enabled"`
	DefaultProvider string        `json:"default_provider" yaml:"default_provider" toml:"default_provider"`
	
	// JWT settings
	JWT JWTConfig `json:"jwt" yaml:"jwt" toml:"jwt"`
	
	// Session settings
	Session SessionConfig `json:"session" yaml:"session" toml:"session"`
	
	// Cookie settings  
	Cookie CookieConfig `json:"cookie" yaml:"cookie" toml:"cookie"`
	
	// Providers configuration
	Providers map[string]ProviderConfig `json:"providers" yaml:"providers" toml:"providers"`
	
	// Security settings
	Security SecurityConfig `json:"security" yaml:"security" toml:"security"`
	
	// Bouncer rules
	Rules []RuleConfig `json:"rules" yaml:"rules" toml:"rules"`
}

// JWTConfig configures JWT token handling
type JWTConfig struct {
	Secret         string        `json:"secret" yaml:"secret" toml:"secret"`
	AccessExpiry   time.Duration `json:"access_expiry" yaml:"access_expiry" toml:"access_expiry"`
	RefreshExpiry  time.Duration `json:"refresh_expiry" yaml:"refresh_expiry" toml:"refresh_expiry"`
	Issuer         string        `json:"issuer" yaml:"issuer" toml:"issuer"`
	Audience       string        `json:"audience" yaml:"audience" toml:"audience"`
	Algorithm      string        `json:"algorithm" yaml:"algorithm" toml:"algorithm"`
}

// SessionConfig configures session management
type SessionConfig struct {
	Store     string        `json:"store" yaml:"store" toml:"store"` // "memory", "redis", "database"
	Expiry    time.Duration `json:"expiry" yaml:"expiry" toml:"expiry"`
	Cleanup   time.Duration `json:"cleanup" yaml:"cleanup" toml:"cleanup"`
	
	// Store-specific configuration
	Redis    RedisConfig    `json:"redis" yaml:"redis" toml:"redis"`
	Database DatabaseConfig `json:"database" yaml:"database" toml:"database"`
}

// CookieConfig configures authentication cookies
type CookieConfig struct {
	Name     string        `json:"name" yaml:"name" toml:"name"`
	Domain   string        `json:"domain" yaml:"domain" toml:"domain"`
	Path     string        `json:"path" yaml:"path" toml:"path"`
	Secure   bool          `json:"secure" yaml:"secure" toml:"secure"`
	HttpOnly bool          `json:"http_only" yaml:"http_only" toml:"http_only"`
	SameSite string        `json:"same_site" yaml:"same_site" toml:"same_site"`
	MaxAge   time.Duration `json:"max_age" yaml:"max_age" toml:"max_age"`
}

// SecurityConfig configures security features
type SecurityConfig struct {
	RateLimiting    RateLimitConfig    `json:"rate_limiting" yaml:"rate_limiting" toml:"rate_limiting"`
	CORS           CORSConfig         `json:"cors" yaml:"cors" toml:"cors"`
	CSRF           CSRFConfig         `json:"csrf" yaml:"csrf" toml:"csrf"`
	ContentSecurity ContentSecurityConfig `json:"content_security" yaml:"content_security" toml:"content_security"`
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	Enabled     bool          `json:"enabled" yaml:"enabled" toml:"enabled"`
	Requests    int           `json:"requests" yaml:"requests" toml:"requests"`
	Window      time.Duration `json:"window" yaml:"window" toml:"window"`
	BanDuration time.Duration `json:"ban_duration" yaml:"ban_duration" toml:"ban_duration"`
}

// CORSConfig configures CORS settings
type CORSConfig struct {
	Enabled          bool     `json:"enabled" yaml:"enabled" toml:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins" yaml:"allowed_origins" toml:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods" yaml:"allowed_methods" toml:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers" yaml:"allowed_headers" toml:"allowed_headers"`
	AllowCredentials bool     `json:"allow_credentials" yaml:"allow_credentials" toml:"allow_credentials"`
}

// CSRFConfig configures CSRF protection
type CSRFConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled" toml:"enabled"`
	TokenName string `json:"token_name" yaml:"token_name" toml:"token_name"`
	Secret    string `json:"secret" yaml:"secret" toml:"secret"`
}

// ContentSecurityConfig configures content security features
type ContentSecurityConfig struct {
	EnableXSSProtection      bool `json:"enable_xss_protection" yaml:"enable_xss_protection" toml:"enable_xss_protection"`
	EnableContentTypeNoSniff bool `json:"enable_content_type_no_sniff" yaml:"enable_content_type_no_sniff" toml:"enable_content_type_no_sniff"`
	EnableFrameOptions       bool `json:"enable_frame_options" yaml:"enable_frame_options" toml:"enable_frame_options"`
}

// RuleConfig configures bouncer rules
type RuleConfig struct {
	Name           string            `json:"name" yaml:"name" toml:"name"`
	Description    string            `json:"description" yaml:"description" toml:"description"`
	Enabled        bool              `json:"enabled" yaml:"enabled" toml:"enabled"`
	PathPattern    string            `json:"path_pattern" yaml:"path_pattern" toml:"path_pattern"`
	Methods        []string          `json:"methods" yaml:"methods" toml:"methods"`
	RequireAuth    bool              `json:"require_auth" yaml:"require_auth" toml:"require_auth"`
	RequiredRoles  []string          `json:"required_roles" yaml:"required_roles" toml:"required_roles"`
	RequiredScopes []string          `json:"required_scopes" yaml:"required_scopes" toml:"required_scopes"`
	Headers        map[string]string `json:"headers" yaml:"headers" toml:"headers"`
}

// RedisConfig configures Redis session store
type RedisConfig struct {
	Address  string `json:"address" yaml:"address" toml:"address"`
	Password string `json:"password" yaml:"password" toml:"password"`
	Database int    `json:"database" yaml:"database" toml:"database"`
	PoolSize int    `json:"pool_size" yaml:"pool_size" toml:"pool_size"`
}

// DatabaseConfig configures database session store
type DatabaseConfig struct {
	Driver     string `json:"driver" yaml:"driver" toml:"driver"`
	DSN        string `json:"dsn" yaml:"dsn" toml:"dsn"`
	TableName  string `json:"table_name" yaml:"table_name" toml:"table_name"`
	MaxIdle    int    `json:"max_idle" yaml:"max_idle" toml:"max_idle"`
	MaxOpen    int    `json:"max_open" yaml:"max_open" toml:"max_open"`
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:         true,
		DefaultProvider: "default",
		
		JWT: JWTConfig{
			Secret:        "change-me-in-production",
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
			Algorithm:     "HS256",
		},
		
		Session: SessionConfig{
			Store:   "memory",
			Expiry:  24 * time.Hour,
			Cleanup: 5 * time.Minute,
		},
		
		Cookie: CookieConfig{
			Name:     "zbz_auth",
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			SameSite: "lax",
			MaxAge:   24 * time.Hour,
		},
		
		Providers: make(map[string]ProviderConfig),
		
		Security: SecurityConfig{
			RateLimiting: RateLimitConfig{
				Enabled:     true,
				Requests:    100,
				Window:      time.Minute,
				BanDuration: 5 * time.Minute,
			},
			CORS: CORSConfig{
				Enabled:          true,
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Requested-With"},
				AllowCredentials: true,
			},
			CSRF: CSRFConfig{
				Enabled:   true,
				TokenName: "X-CSRF-Token",
			},
			ContentSecurity: ContentSecurityConfig{
				EnableXSSProtection:      true,
				EnableContentTypeNoSniff: true,
				EnableFrameOptions:       true,
			},
		},
		
		Rules: []RuleConfig{
			{
				Name:        "admin_area",
				Description: "Protect admin area",
				Enabled:     true,
				PathPattern: "/admin/.*",
				RequireAuth: true,
				RequiredRoles: []string{"admin"},
			},
			{
				Name:        "api_access",
				Description: "Protect API endpoints",
				Enabled:     true,
				PathPattern: "/api/.*",
				Methods:     []string{"POST", "PUT", "DELETE"},
				RequireAuth: true,
			},
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil // Skip validation if auth is disabled
	}
	
	// Validate JWT config
	if c.JWT.Secret == "" || c.JWT.Secret == "change-me-in-production" {
		return fmt.Errorf("JWT secret must be set and not use default value")
	}
	
	if c.JWT.AccessExpiry <= 0 {
		return fmt.Errorf("JWT access expiry must be positive")
	}
	
	if c.JWT.RefreshExpiry <= 0 {
		return fmt.Errorf("JWT refresh expiry must be positive")
	}
	
	// Validate session config
	if c.Session.Store == "" {
		return fmt.Errorf("session store type must be specified")
	}
	
	if c.Session.Expiry <= 0 {
		return fmt.Errorf("session expiry must be positive")
	}
	
	// Validate provider config
	if c.DefaultProvider != "" {
		if _, exists := c.Providers[c.DefaultProvider]; !exists {
			return fmt.Errorf("default provider %s not found in providers config", c.DefaultProvider)
		}
	}
	
	// Validate each provider
	for name, provider := range c.Providers {
		if !provider.Enabled {
			continue
		}
		
		if provider.DisplayName == "" {
			return fmt.Errorf("provider %s missing display name", name)
		}
	}
	
	return nil
}

// ApplyToAuth applies the configuration to an Auth instance
func (c *Config) ApplyToAuth(auth *zAuth) error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	// Apply JWT settings
	if c.JWT.Secret != "" {
		auth.jwtSecret = []byte(c.JWT.Secret)
	}
	auth.tokenExpiry = c.JWT.AccessExpiry
	auth.refreshExpiry = c.JWT.RefreshExpiry
	
	// Apply cookie settings
	auth.cookieName = c.Cookie.Name
	auth.cookieDomain = c.Cookie.Domain
	auth.cookieSecure = c.Cookie.Secure
	
	// Apply session store
	switch c.Session.Store {
	case "memory":
		auth.sessionStore = NewMemorySessionStore()
	case "redis":
		// Would create Redis store with c.Session.Redis config
		return fmt.Errorf("Redis session store not implemented")
	case "database":
		// Would create database store with c.Session.Database config
		return fmt.Errorf("Database session store not implemented")
	default:
		return fmt.Errorf("unknown session store type: %s", c.Session.Store)
	}
	
	return nil
}

// LoadConfig loads configuration from various sources (placeholder)
func LoadConfig(sources ...string) (*Config, error) {
	// This would load from files, environment variables, etc.
	// For now, return default config
	return DefaultConfig(), nil
}

// Provider-specific configuration helpers

// OIDCProviderConfig creates OIDC provider configuration
func OIDCProviderConfig(issuer, clientID, clientSecret string) ProviderConfig {
	return ProviderConfig{
		Enabled:     true,
		DisplayName: "OpenID Connect",
		Settings: map[string]any{
			"issuer":        issuer,
			"client_id":     clientID,
			"client_secret": clientSecret,
			"scopes":        []string{"openid", "profile", "email"},
		},
	}
}

// OAuth2ProviderConfig creates OAuth2 provider configuration
func OAuth2ProviderConfig(authURL, tokenURL, clientID, clientSecret string) ProviderConfig {
	return ProviderConfig{
		Enabled:     true,
		DisplayName: "OAuth2",
		Settings: map[string]any{
			"auth_url":      authURL,
			"token_url":     tokenURL,
			"client_id":     clientID,
			"client_secret": clientSecret,
		},
	}
}

// BasicProviderConfig creates basic auth provider configuration
func BasicProviderConfig() ProviderConfig {
	return ProviderConfig{
		Enabled:     true,
		DisplayName: "Basic Authentication",
		Settings: map[string]any{
			"realm": "ZBZ Application",
		},
	}
}