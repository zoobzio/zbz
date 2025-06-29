package rocco

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"zbz/zlog"
	"zbz/capitan"
)

// zAuth is the default implementation of the Auth interface
type zAuth struct {
	providers       map[string]Provider
	defaultProvider string
	jwtSecret       []byte
	sessionStore    SessionStore
	
	// Configuration
	tokenExpiry     time.Duration
	refreshExpiry   time.Duration
	cookieName      string
	cookieDomain    string
	cookieSecure    bool
	
	mu sync.RWMutex
}

// NewAuth creates a new Auth instance with default configuration
func NewAuth(opts ...AuthOption) Auth {
	auth := &zAuth{
		providers:     make(map[string]Provider),
		tokenExpiry:   15 * time.Minute,
		refreshExpiry: 7 * 24 * time.Hour,
		cookieName:    "zbz_auth",
		cookieSecure:  true,
		sessionStore:  NewMemorySessionStore(), // Default in-memory store
	}
	
	// Apply options
	for _, opt := range opts {
		opt(auth)
	}
	
	// Generate default JWT secret if not provided
	if len(auth.jwtSecret) == 0 {
		auth.jwtSecret = []byte("zbz-default-secret-change-me-in-production")
		zlog.Warn("Using default JWT secret - change in production",
			zlog.String("security_risk", "default_secret"),
		)
	}
	
	zlog.Info("Authentication service ready",
		zlog.Duration("token_expiry", auth.tokenExpiry),
		zlog.Duration("refresh_expiry", auth.refreshExpiry),
		zlog.Int("providers_available", len(auth.providers)),
	)
	
	return auth
}

// AuthOption configures the Auth instance
type AuthOption func(*zAuth)

// WithJWTSecret sets the JWT signing secret
func WithJWTSecret(secret []byte) AuthOption {
	return func(a *zAuth) {
		a.jwtSecret = secret
	}
}

// WithTokenExpiry sets the access token expiry duration
func WithTokenExpiry(d time.Duration) AuthOption {
	return func(a *zAuth) {
		a.tokenExpiry = d
	}
}

// WithSessionStore sets a custom session store
func WithSessionStore(store SessionStore) AuthOption {
	return func(a *zAuth) {
		a.sessionStore = store
	}
}

// RegisterProvider registers an authentication provider
func (a *zAuth) RegisterProvider(name string, provider Provider) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if _, exists := a.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}
	
	// Validate provider
	if err := provider.Validate(); err != nil {
		zlog.Error("Authentication provider validation failed",
			zlog.String("provider", name),
			zlog.String("provider_type", string(provider.Type())),
			zlog.String("error", err.Error()),
		)
		return fmt.Errorf("provider validation failed: %w", err)
	}
	
	a.providers[name] = provider
	
	// Set as default if it's the first provider
	if len(a.providers) == 1 {
		a.defaultProvider = name
		zlog.Info("Default authentication provider configured",
			zlog.String("provider", name),
			zlog.String("provider_type", string(provider.Type())),
		)
	} else {
		zlog.Info("Authentication provider registered",
			zlog.String("provider", name),
			zlog.String("provider_type", string(provider.Type())),
			zlog.Int("total_providers", len(a.providers)),
		)
	}
	
	return nil
}

// GetProvider retrieves a provider by name
func (a *zAuth) GetProvider(name string) (Provider, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	provider, exists := a.providers[name]
	if !exists {
		return nil, ErrProviderNotFound
	}
	
	return provider, nil
}

// SetDefaultProvider sets the default provider
func (a *zAuth) SetDefaultProvider(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if _, exists := a.providers[name]; !exists {
		return ErrProviderNotFound
	}
	
	a.defaultProvider = name
	return nil
}

// Authenticate authenticates a user with the specified or default provider
func (a *zAuth) Authenticate(ctx context.Context, credentials Credentials) (*Identity, error) {
	start := time.Now()
	
	// Determine provider
	providerName := credentials.Extra["provider"]
	if providerName == "" {
		a.mu.RLock()
		providerName = a.defaultProvider
		a.mu.RUnlock()
	}
	
	zlog.Debug("Authentication attempt started",
		zlog.String("provider", providerName),
		zlog.String("username", credentials.Username),
		zlog.String("credential_type", credentials.Type),
	)
	
	provider, err := a.GetProvider(providerName)
	if err != nil {
		zlog.Error("Authentication provider not found",
			zlog.String("provider", providerName),
			zlog.String("username", credentials.Username),
		)
		return nil, err
	}
	
	// Authenticate with provider
	identity, err := provider.Authenticate(ctx, credentials)
	if err != nil {
		zlog.Warn("Authentication failed",
			zlog.String("provider", providerName),
			zlog.String("username", credentials.Username),
			zlog.String("error", err.Error()),
			zlog.Duration("duration", time.Since(start)),
		)
		// Emit failed auth event
		capitan.EmitEvent("auth.failed", map[string]any{
			"provider":  providerName,
			"username":  credentials.Username,
			"error":     err.Error(),
			"timestamp": time.Now(),
			"duration":  time.Since(start),
		})
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	
	// Generate JWT tokens
	accessToken, err := a.generateAccessToken(identity)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	
	refreshToken, err := a.generateRefreshToken(identity)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	
	// Update identity with tokens
	identity.AccessToken = accessToken
	identity.RefreshToken = refreshToken
	identity.TokenType = "Bearer"
	identity.ExpiresAt = time.Now().Add(a.tokenExpiry)
	identity.IssuedAt = time.Now()
	identity.LastActive = time.Now()
	
	zlog.Info("User authenticated successfully",
		zlog.String("user_id", identity.ID),
		zlog.String("username", identity.Username),
		zlog.String("provider", providerName),
		zlog.String("email", identity.Email),
		zlog.Strings("roles", identity.Roles),
		zlog.Duration("duration", time.Since(start)),
	)
	
	// Emit successful auth event
	capitan.EmitEvent("auth.success", map[string]any{
		"user_id":   identity.ID,
		"username":  identity.Username,
		"provider":  providerName,
		"email":     identity.Email,
		"roles":     identity.Roles,
		"timestamp": time.Now(),
		"duration":  time.Since(start),
	})
	
	// Create session
	session := &Session{
		ID:         generateSessionID(),
		IdentityID: identity.ID,
		Provider:   providerName,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		ExpiresAt:  time.Now().Add(a.refreshExpiry),
	}
	
	// Extract request metadata if available
	if req, ok := ctx.Value("http.request").(*http.Request); ok {
		session.IPAddress = getClientIP(req)
		session.UserAgent = req.Header.Get("User-Agent")
	}
	
	// Store session
	if err := a.sessionStore.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	
	identity.SessionID = session.ID
	
	return identity, nil
}

// Refresh refreshes an identity using a refresh token
func (a *zAuth) Refresh(ctx context.Context, refreshToken string) (*Identity, error) {
	// Parse refresh token
	claims, err := a.parseRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	
	// Get session
	sessionID := claims["session_id"].(string)
	session, err := a.sessionStore.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	
	// Check if session is still valid
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	
	// Get provider
	provider, err := a.GetProvider(session.Provider)
	if err != nil {
		return nil, err
	}
	
	// Refresh with provider
	identity, err := provider.Refresh(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("provider refresh failed: %w", err)
	}
	
	// Generate new tokens
	accessToken, err := a.generateAccessToken(identity)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	
	// Update identity
	identity.AccessToken = accessToken
	identity.RefreshToken = refreshToken // Keep same refresh token
	identity.TokenType = "Bearer"
	identity.ExpiresAt = time.Now().Add(a.tokenExpiry)
	identity.LastActive = time.Now()
	identity.SessionID = sessionID
	
	// Update session
	session.LastActive = time.Now()
	if err := a.sessionStore.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}
	
	return identity, nil
}

// Revoke revokes a token and its associated session
func (a *zAuth) Revoke(ctx context.Context, token string) error {
	// Try to parse as access token first
	claims, err := a.parseAccessToken(token)
	if err != nil {
		// Try as refresh token
		claims, err = a.parseRefreshToken(token)
		if err != nil {
			return fmt.Errorf("invalid token: %w", err)
		}
	}
	
	// Get session ID
	sessionID, ok := claims["session_id"].(string)
	if !ok {
		return fmt.Errorf("token missing session ID")
	}
	
	// Delete session
	if err := a.sessionStore.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	// Notify provider if it supports revocation
	if providerName, ok := claims["provider"].(string); ok {
		if provider, err := a.GetProvider(providerName); err == nil {
			provider.Revoke(ctx, token) // Ignore error, best effort
		}
	}
	
	return nil
}

// Authorize checks if an identity is authorized to access a resource
func (a *zAuth) Authorize(ctx context.Context, identity *Identity, resource Resource) error {
	if identity == nil {
		return ErrUnauthorized
	}
	
	// Check if token is still valid
	if time.Now().After(identity.ExpiresAt) {
		return ErrTokenExpired
	}
	
	// Basic resource type authorization
	switch resource.Type {
	case "api":
		// Check API permissions
		requiredPerm := fmt.Sprintf("api:%s:%s", resource.ID, resource.Action)
		if !hasPermission(identity.Permissions, requiredPerm) {
			// Check wildcard permissions
			wildcardPerm := fmt.Sprintf("api:*:%s", resource.Action)
			if !hasPermission(identity.Permissions, wildcardPerm) {
				return ErrForbidden
			}
		}
		
	case "admin":
		// Require admin role
		if !hasRole(identity.Roles, "admin") {
			return ErrForbidden
		}
		
	default:
		// Custom resource types can be handled by bouncer rules
	}
	
	return nil
}

// JWT token generation

func (a *zAuth) generateAccessToken(identity *Identity) (string, error) {
	claims := jwt.MapClaims{
		"sub":        identity.ID,
		"username":   identity.Username,
		"email":      identity.Email,
		"provider":   identity.Provider,
		"session_id": identity.SessionID,
		"exp":        time.Now().Add(a.tokenExpiry).Unix(),
		"iat":        time.Now().Unix(),
		"type":       "access",
		
		// Include authorization data
		"roles":       identity.Roles,
		"permissions": identity.Permissions,
		"scopes":      identity.Scopes,
	}
	
	// Add custom claims
	for k, v := range identity.Claims {
		if _, exists := claims[k]; !exists { // Don't override standard claims
			claims[k] = v
		}
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

func (a *zAuth) generateRefreshToken(identity *Identity) (string, error) {
	claims := jwt.MapClaims{
		"sub":        identity.ID,
		"provider":   identity.Provider,
		"session_id": identity.SessionID,
		"exp":        time.Now().Add(a.refreshExpiry).Unix(),
		"iat":        time.Now().Unix(),
		"type":       "refresh",
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

func (a *zAuth) parseAccessToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	
	// Verify it's an access token
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "access" {
		return nil, fmt.Errorf("not an access token")
	}
	
	return claims, nil
}

func (a *zAuth) parseRefreshToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	
	// Verify it's a refresh token
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
		return nil, fmt.Errorf("not a refresh token")
	}
	
	return claims, nil
}

// Helper functions

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func hasPermission(permissions []string, permission string) bool {
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func generateSessionID() string {
	// Simple implementation - in production use crypto/rand
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}

func getClientIP(r *http.Request) string {
	// Check common headers
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// Take first IP if multiple
		return strings.Split(ip, ",")[0]
	}
	
	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}