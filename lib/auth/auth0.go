package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"golang.org/x/oauth2"
	"zbz/lib/cache"
	"zbz/shared/logger"
)

// auth0Auth implements AuthDriver interface for Auth0
type auth0Auth struct {
	oauth           oauth2.Config
	jwksCache       jwk.Set
	jwksMutex       sync.RWMutex
	jwksURL         string
	lastJWKSRefresh time.Time
	config          *AuthConfig
	cache           cache.Cache
}

// NewAuth0Auth creates a new Auth0 authentication driver
func NewAuth0Auth(cache cache.Cache, config *AuthConfig) AuthDriver {
	jwksURL := fmt.Sprintf("https://%s/.well-known/jwks.json", config.Domain)

	oauth := oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("https://%s/authorize", config.Domain),
			TokenURL: fmt.Sprintf("https://%s/oauth/token", config.Domain),
		},
		Scopes: config.Scopes,
	}

	auth := &auth0Auth{
		oauth:   oauth,
		jwksURL: jwksURL,
		config:  config,
		cache:   cache,
	}

	// Load JWKS on startup
	if err := auth.refreshJWKS(); err != nil {
		logger.Fatal("Failed to load JWKS", logger.Err(err))
	}

	return auth
}

// DriverName returns the driver name
func (a *auth0Auth) DriverName() string {
	return "auth0"
}

// DriverVersion returns the driver version
func (a *auth0Auth) DriverVersion() string {
	return "1.0.0"
}

// GetUserFromCache retrieves user data from cache
func (a *auth0Auth) GetUserFromCache(userID string) (*AuthUser, error) {
	return a.getUserFromCache(userID)
}

// CacheUserData stores user data in cache
func (a *auth0Auth) CacheUserData(userID string, userData *AuthUser, ttl int) error {
	return a.cacheUserData(userID, userData, time.Duration(ttl)*time.Second)
}

// ValidateToken validates a JWT token and returns claims
func (a *auth0Auth) ValidateToken(tokenString string) (*AuthToken, error) {
	// Parse token without verification first to get key ID
	token, err := jwt.ParseWithClaims(tokenString, &Auth0Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("no key ID in token header")
		}

		// Get JWKS
		jwks, err := a.getJWKS()
		if err != nil {
			return nil, fmt.Errorf("failed to get JWKS: %w", err)
		}

		// Find the key
		key, found := jwks.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("key not found: %s", kid)
		}

		// Convert to raw key for verification
		var rawKey interface{}
		if err := key.Raw(&rawKey); err != nil {
			return nil, fmt.Errorf("failed to get raw key: %w", err)
		}

		return rawKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token is not valid")
	}

	claims, ok := token.Claims.(*Auth0Claims)
	if !ok {
		return nil, errors.New("invalid claims type")
	}

	return &AuthToken{
		Value:       tokenString,
		Sub:         claims.Sub,
		Email:       claims.Email,
		Name:        claims.Name,
		Permissions: claims.Permissions,
		ExpiresAt:   claims.ExpiresAt.Time.Format(time.RFC3339),
	}, nil
}

// GetLoginURL returns the OAuth login URL with state
func (a *auth0Auth) GetLoginURL(state string) string {
	return a.oauth.AuthCodeURL(state)
}

// ExchangeCodeForToken exchanges an authorization code for a token
func (a *auth0Auth) ExchangeCodeForToken(code, state string) (*AuthToken, error) {
	// Exchange code for token
	token, err := a.oauth.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Extract access token
	accessToken := token.AccessToken
	if accessToken == "" {
		return nil, errors.New("no access token received")
	}

	// Validate the token to get claims
	return a.ValidateToken(accessToken)
}

// Cache operations
func (a *auth0Auth) getUserFromCache(userID string) (*AuthUser, error) {
	ctx := context.Background()
	key := fmt.Sprintf("user:%s", userID)

	data, err := a.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var userData AuthUser
	if err := json.Unmarshal(data, &userData); err != nil {
		return nil, err
	}

	return &userData, nil
}

func (a *auth0Auth) cacheUserData(userID string, userData *AuthUser, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("user:%s", userID)

	data, err := json.Marshal(userData)
	if err != nil {
		return err
	}

	return a.cache.Set(ctx, key, data, ttl)
}

// Auth0Claims represents the claims in an Auth0 JWT token
type Auth0Claims struct {
	Sub         string   `json:"sub"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// refreshJWKS fetches and caches JWKS from Auth0
func (a *auth0Auth) refreshJWKS() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	set, err := jwk.Fetch(ctx, a.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	a.jwksMutex.Lock()
	a.jwksCache = set
	a.lastJWKSRefresh = time.Now()
	a.jwksMutex.Unlock()

	logger.Debug("JWKS refreshed successfully", logger.String("url", a.jwksURL))
	return nil
}

// getJWKS returns cached JWKS, refreshing if needed
func (a *auth0Auth) getJWKS() (jwk.Set, error) {
	a.jwksMutex.RLock()
	if time.Since(a.lastJWKSRefresh) < 24*time.Hour && a.jwksCache != nil {
		defer a.jwksMutex.RUnlock()
		return a.jwksCache, nil
	}
	a.jwksMutex.RUnlock()

	// Refresh needed
	if err := a.refreshJWKS(); err != nil {
		return nil, err
	}

	a.jwksMutex.RLock()
	defer a.jwksMutex.RUnlock()
	return a.jwksCache, nil
}
