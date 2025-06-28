package jwt

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"zbz/rocco"
)

// Provider validates external JWT tokens
type Provider struct {
	config    rocco.ProviderConfig
	secret    []byte
	publicKey interface{} // For RSA/ECDSA verification
	issuer    string
	audience  string
	algorithm string
}

// NewProvider creates a new JWT provider
func NewProvider() *Provider {
	return &Provider{
		algorithm: "HS256",
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "rocco-jwt"
}

// Type returns the provider type
func (p *Provider) Type() rocco.ProviderType {
	return rocco.ProviderTypeJWT
}

// Configure configures the JWT provider
func (p *Provider) Configure(config rocco.ProviderConfig) error {
	p.config = config
	
	settings := config.Settings
	
	// Secret for HMAC algorithms
	if secret, ok := settings["secret"].(string); ok {
		p.secret = []byte(secret)
	}
	
	// Public key for RSA/ECDSA algorithms (would need proper key parsing)
	if publicKey, ok := settings["public_key"].(string); ok {
		// In production, would parse PEM format
		_ = publicKey
	}
	
	if issuer, ok := settings["issuer"].(string); ok {
		p.issuer = issuer
	}
	
	if audience, ok := settings["audience"].(string); ok {
		p.audience = audience
	}
	
	if algorithm, ok := settings["algorithm"].(string); ok {
		p.algorithm = algorithm
	}
	
	return nil
}

// Validate validates the provider configuration
func (p *Provider) Validate() error {
	switch p.algorithm {
	case "HS256", "HS384", "HS512":
		if len(p.secret) == 0 {
			return fmt.Errorf("JWT secret required for HMAC algorithms")
		}
	case "RS256", "RS384", "RS512", "ES256", "ES384", "ES512":
		if p.publicKey == nil {
			return fmt.Errorf("JWT public key required for RSA/ECDSA algorithms")
		}
	default:
		return fmt.Errorf("unsupported JWT algorithm: %s", p.algorithm)
	}
	
	return nil
}

// Authenticate validates a JWT token and extracts identity
func (p *Provider) Authenticate(ctx context.Context, credentials rocco.Credentials) (*rocco.Identity, error) {
	if credentials.Type != "token" {
		return nil, fmt.Errorf("JWT provider only supports token credentials")
	}
	
	// Parse and validate token
	token, err := jwt.Parse(credentials.Token, func(token *jwt.Token) (interface{}, error) {
		// Verify algorithm
		if token.Method.Alg() != p.algorithm {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		
		// Return appropriate key based on algorithm
		switch p.algorithm {
		case "HS256", "HS384", "HS512":
			return p.secret, nil
		case "RS256", "RS384", "RS512", "ES256", "ES384", "ES512":
			return p.publicKey, nil
		default:
			return nil, fmt.Errorf("unsupported algorithm: %s", p.algorithm)
		}
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT: %w", err)
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid JWT claims")
	}
	
	// Validate standard claims
	if err := p.validateClaims(claims); err != nil {
		return nil, fmt.Errorf("JWT validation failed: %w", err)
	}
	
	// Build identity from claims
	identity := &rocco.Identity{
		Provider:   p.Name(),
		Claims:     claims,
		IssuedAt:   time.Now(),
		LastActive: time.Now(),
	}
	
	// Extract standard fields
	if sub, ok := claims["sub"].(string); ok {
		identity.ID = sub
	}
	
	if username, ok := claims["preferred_username"].(string); ok {
		identity.Username = username
	} else if username, ok := claims["username"].(string); ok {
		identity.Username = username
	} else if email, ok := claims["email"].(string); ok {
		identity.Username = email
	}
	
	if email, ok := claims["email"].(string); ok {
		identity.Email = email
	}
	
	if name, ok := claims["name"].(string); ok {
		identity.DisplayName = name
	}
	
	// Extract expiry
	if exp, ok := claims["exp"].(float64); ok {
		identity.ExpiresAt = time.Unix(int64(exp), 0)
	}
	
	// Extract roles from various claim formats
	identity.Roles = p.extractStringSlice(claims, "roles", "realm_access.roles", "resource_access.roles")
	
	// Extract permissions/scopes
	identity.Permissions = p.extractStringSlice(claims, "permissions", "perms")
	identity.Scopes = p.extractStringSlice(claims, "scope", "scopes")
	
	// Convert space-separated scope string to slice
	if scopeStr, ok := claims["scope"].(string); ok && len(identity.Scopes) == 0 {
		identity.Scopes = strings.Fields(scopeStr)
	}
	
	// Extract groups
	identity.Groups = p.extractStringSlice(claims, "groups", "group")
	
	return identity, nil
}

// validateClaims validates standard JWT claims
func (p *Provider) validateClaims(claims jwt.MapClaims) error {
	now := time.Now()
	
	// Validate expiration
	if exp, ok := claims["exp"].(float64); ok {
		if now.Unix() > int64(exp) {
			return fmt.Errorf("token has expired")
		}
	}
	
	// Validate not before
	if nbf, ok := claims["nbf"].(float64); ok {
		if now.Unix() < int64(nbf) {
			return fmt.Errorf("token not yet valid")
		}
	}
	
	// Validate issuer
	if p.issuer != "" {
		if iss, ok := claims["iss"].(string); ok {
			if iss != p.issuer {
				return fmt.Errorf("invalid issuer: expected %s, got %s", p.issuer, iss)
			}
		} else {
			return fmt.Errorf("missing issuer claim")
		}
	}
	
	// Validate audience
	if p.audience != "" {
		if aud, ok := claims["aud"].(string); ok {
			if aud != p.audience {
				return fmt.Errorf("invalid audience: expected %s, got %s", p.audience, aud)
			}
		} else if audSlice, ok := claims["aud"].([]interface{}); ok {
			found := false
			for _, a := range audSlice {
				if audStr, ok := a.(string); ok && audStr == p.audience {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("audience %s not found in token", p.audience)
			}
		} else {
			return fmt.Errorf("missing audience claim")
		}
	}
	
	return nil
}

// extractStringSlice extracts string slices from various claim paths
func (p *Provider) extractStringSlice(claims jwt.MapClaims, paths ...string) []string {
	for _, path := range paths {
		if slice := p.getStringSliceFromPath(claims, path); len(slice) > 0 {
			return slice
		}
	}
	return nil
}

// getStringSliceFromPath extracts string slice from nested claim path
func (p *Provider) getStringSliceFromPath(claims jwt.MapClaims, path string) []string {
	parts := strings.Split(path, ".")
	current := claims
	
	// Navigate to nested claim
	for i, part := range parts[:len(parts)-1] {
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	
	// Extract final value
	finalKey := parts[len(parts)-1]
	if value, ok := current[finalKey]; ok {
		return interfaceToStringSlice(value)
	}
	
	return nil
}

// interfaceToStringSlice converts various interface types to string slice
func interfaceToStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	case string:
		return []string{v}
	default:
		return nil
	}
}

// Refresh is not supported for JWT validation (tokens are stateless)
func (p *Provider) Refresh(ctx context.Context, refreshToken string) (*rocco.Identity, error) {
	return nil, fmt.Errorf("JWT provider does not support token refresh")
}

// Revoke is not applicable for JWT validation (tokens are stateless)
func (p *Provider) Revoke(ctx context.Context, token string) error {
	return nil // No-op for stateless tokens
}

// User management operations (not supported)

func (p *Provider) CreateUser(ctx context.Context, user rocco.UserInfo) error {
	return fmt.Errorf("JWT provider does not support user creation")
}

func (p *Provider) UpdateUser(ctx context.Context, userID string, user rocco.UserInfo) error {
	return fmt.Errorf("JWT provider does not support user updates")
}

func (p *Provider) DeleteUser(ctx context.Context, userID string) error {
	return fmt.Errorf("JWT provider does not support user deletion")
}

func (p *Provider) GetUser(ctx context.Context, userID string) (*rocco.UserInfo, error) {
	return nil, fmt.Errorf("JWT provider does not support user retrieval")
}

// Configuration helpers

// HMACConfig creates a JWT provider configuration for HMAC signing
func HMACConfig(secret, issuer, audience string) rocco.ProviderConfig {
	return rocco.ProviderConfig{
		Enabled:     true,
		DisplayName: "JWT (HMAC)",
		Settings: map[string]any{
			"secret":    secret,
			"issuer":    issuer,
			"audience":  audience,
			"algorithm": "HS256",
		},
	}
}

// RSAConfig creates a JWT provider configuration for RSA signing
func RSAConfig(publicKey, issuer, audience string) rocco.ProviderConfig {
	return rocco.ProviderConfig{
		Enabled:     true,
		DisplayName: "JWT (RSA)",
		Settings: map[string]any{
			"public_key": publicKey,
			"issuer":     issuer,
			"audience":   audience,
			"algorithm":  "RS256",
		},
	}
}

// KeycloakConfig creates a JWT provider configuration for Keycloak
func KeycloakConfig(realm, audience, publicKey string) rocco.ProviderConfig {
	issuer := fmt.Sprintf("https://keycloak.example.com/realms/%s", realm)
	
	return rocco.ProviderConfig{
		Enabled:     true,
		DisplayName: "Keycloak JWT",
		Settings: map[string]any{
			"public_key": publicKey,
			"issuer":     issuer,
			"audience":   audience,
			"algorithm":  "RS256",
		},
	}
}

// Auth0Config creates a JWT provider configuration for Auth0
func Auth0Config(domain, audience, secret string) rocco.ProviderConfig {
	issuer := fmt.Sprintf("https://%s/", domain)
	
	return rocco.ProviderConfig{
		Enabled:     true,
		DisplayName: "Auth0 JWT",
		Settings: map[string]any{
			"secret":    secret,
			"issuer":    issuer,
			"audience":  audience,
			"algorithm": "HS256",
		},
	}
}