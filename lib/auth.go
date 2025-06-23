package zbz

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// Auth is an interface that defines methods for user authentication
type Auth interface {
	// Authentication handlers
	LoginHandler(ctx *gin.Context)
	CallbackHandler(ctx *gin.Context)
	LogoutHandler(ctx *gin.Context)

	// Middleware
	TokenMiddleware() gin.HandlerFunc
	ScopeMiddleware(scope string) gin.HandlerFunc

	// Token validation
	ValidateToken(tokenString string) (*AuthClaims, error)

	// Engine integration
	SetUserCoreGetter(getter func() Core)
}

// AuthClaims represents the claims in a JWT token
type AuthClaims struct {
	Sub         string   `json:"sub"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// Auth0UserData represents user data from Auth0 Management API
type Auth0UserData struct {
	Sub         string   `json:"sub"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	UpdatedAt   string   `json:"updated_at"`
}

// zAuth implements the Auth interface with Auth0 and JWKS integration
type zAuth struct {
	oauth           oauth2.Config
	jwksCache       jwk.Set
	jwksMutex       sync.RWMutex
	jwksURL         string
	redisClient     *redis.Client
	lastJWKSRefresh time.Time
	getUserCore     func() Core // Function to get user core from engine
}

// NewAuth initializes Auth0 authentication with JWKS and Redis
func NewAuth(redisClient *redis.Client) Auth {
	domain := config.AuthDomain()
	jwksURL := fmt.Sprintf("https://%s/.well-known/jwks.json", domain)

	oauth := oauth2.Config{
		ClientID:     config.AuthClientID(),
		ClientSecret: config.AuthClientSecret(),
		RedirectURL:  config.AuthCallback(),
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("https://%s/authorize", domain),
			TokenURL: fmt.Sprintf("https://%s/oauth/token", domain),
		},
		Scopes: []string{"openid", "profile", "email", "read:current_user"},
	}

	auth := &zAuth{
		oauth:       oauth,
		jwksURL:     jwksURL,
		redisClient: redisClient,
	}

	// Load JWKS on startup
	if err := auth.refreshJWKS(); err != nil {
		Log.Fatal("Failed to load JWKS", zap.Error(err))
	}

	return auth
}

// SetUserCoreGetter sets the function to get the user core from the engine
func (a *zAuth) SetUserCoreGetter(getter func() Core) {
	a.getUserCore = getter
}

// refreshJWKS fetches and caches JWKS from Auth0
func (a *zAuth) refreshJWKS() error {
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

	Log.Debug("JWKS refreshed successfully", zap.String("url", a.jwksURL))
	return nil
}

// getJWKS returns cached JWKS, refreshing if needed
func (a *zAuth) getJWKS() (jwk.Set, error) {
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

// ValidateToken validates a JWT token and returns claims
func (a *zAuth) ValidateToken(tokenString string) (*AuthClaims, error) {
	// Parse token without verification first to get key ID
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("token missing kid header")
		}

		// Get JWKS
		jwksSet, err := a.getJWKS()
		if err != nil {
			return nil, fmt.Errorf("failed to get JWKS: %w", err)
		}

		// Find key by ID
		key, found := jwksSet.LookupKeyID(kid)
		if !found {
			// Try refreshing JWKS in case of key rotation
			if refreshErr := a.refreshJWKS(); refreshErr == nil {
				if jwksSet, err = a.getJWKS(); err == nil {
					key, found = jwksSet.LookupKeyID(kid)
				}
			}
			if !found {
				return nil, fmt.Errorf("key %s not found in JWKS", kid)
			}
		}

		// Convert JWK to crypto key
		var rawKey interface{}
		if err := key.Raw(&rawKey); err != nil {
			return nil, fmt.Errorf("failed to get raw key: %w", err)
		}

		return rawKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// getUserFromCache retrieves user data from Redis cache
func (a *zAuth) getUserFromCache(auth0ID string) (*Auth0UserData, error) {
	ctx := context.Background()
	key := fmt.Sprintf("user:%s", auth0ID)

	data, err := a.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var userData Auth0UserData
	if err := json.Unmarshal([]byte(data), &userData); err != nil {
		return nil, err
	}

	return &userData, nil
}

// cacheUserData stores user data in Redis with token TTL
func (a *zAuth) cacheUserData(auth0ID string, userData *Auth0UserData, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("user:%s", auth0ID)

	data, err := json.Marshal(userData)
	if err != nil {
		return err
	}

	return a.redisClient.Set(ctx, key, data, ttl).Err()
}

// createOrUpdateUser creates or updates a user using the Core system
func (a *zAuth) createOrUpdateUser(userData *Auth0UserData) (*User, error) {
	// For now, just create a user object with the auth data
	// TODO: Implement proper user lookup and creation using Core operations
	user := &User{
		Model: Model{
			ID:        uuid.NewString(), // Use standard UUID generator
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:    userData.Name,
		Email:   userData.Email,
		Auth0ID: userData.Sub,
	}

	return user, nil
}

// TokenMiddleware validates JWT tokens and attaches user to context
func (a *zAuth) TokenMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Extract token from cookie
		cookie, err := ctx.Request.Cookie("auth_token")
		if err != nil {
			Log.Debug("No auth token cookie found")
			ctx.Set("error_message", "Authentication required")
			ctx.Status(http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := a.ValidateToken(cookie.Value)
		if err != nil {
			Log.Debug("Token validation failed", zap.Error(err))
			ctx.Set("error_message", "Invalid authentication token")
			ctx.Status(http.StatusUnauthorized)
			return
		}

		// Try to get user data from cache
		userData, err := a.getUserFromCache(claims.Sub)
		if err != nil {
			// Cache miss - create userData from token claims
			userData = &Auth0UserData{
				Sub:         claims.Sub,
				Email:       claims.Email,
				Name:        claims.Name,
				Permissions: claims.Permissions,
			}

			// Cache for future requests (use token TTL)
			ttl := time.Until(claims.ExpiresAt.Time)
			if ttl > 0 {
				if cacheErr := a.cacheUserData(claims.Sub, userData, ttl); cacheErr != nil {
					Log.Warn("Failed to cache user data", zap.Error(cacheErr))
				}
			}
		}

		// Get database connection
		_, exists := ctx.Get("db")
		if !exists {
			Log.Error("Database connection not available in context")
			ctx.Set("error_message", "Database connection unavailable")
			ctx.Status(http.StatusInternalServerError)
			return
		}

		// Create or update user in database
		user, err := a.createOrUpdateUser(userData)
		if err != nil {
			Log.Error("Failed to create/update user", zap.Error(err))
			ctx.Set("error_message", "Failed to process user data")
			ctx.Status(http.StatusInternalServerError)
			return
		}

		// Attach user and permissions to context
		ctx.Set("user", user)
		ctx.Set("permissions", userData.Permissions)
		ctx.Set("auth_claims", claims)

		ctx.Next()
	}
}

// ScopeMiddleware creates middleware that checks for specific scope permissions
func (a *zAuth) ScopeMiddleware(scope string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		permissions, exists := ctx.Get("permissions")
		if !exists {
			Log.Debug("No permissions found in context")
			ctx.Set("error_message", "Authentication required")
			ctx.Status(http.StatusUnauthorized)
			return
		}

		perms, ok := permissions.([]string)
		if !ok {
			Log.Error("Invalid permissions type in context")
			ctx.Set("error_message", "Invalid permissions data")
			ctx.Status(http.StatusInternalServerError)
			return
		}

		// Check if user has required scope
		for _, perm := range perms {
			if perm == scope {
				ctx.Next()
				return
			}
		}

		user, _ := ctx.Get("user")
		if u, ok := user.(*User); ok {
			Log.Info("Access denied - insufficient permissions",
				zap.String("user_id", u.ID),
				zap.String("required_scope", scope),
				zap.Strings("user_permissions", perms))
		}

		ctx.Set("error_message", "Insufficient permissions")
		ctx.Status(http.StatusForbidden)
	}
}

// LoginHandler initiates OAuth2 flow with Auth0
func (a *zAuth) LoginHandler(ctx *gin.Context) {
	// Generate state parameter
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		Log.Error("Failed to generate state", zap.Error(err))
		ctx.Set("error_message", "Failed to initiate login")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	state := base64.StdEncoding.EncodeToString(b)

	// Get redirect URL from query parameter
	redirectURL := ctx.Query("redirect")
	if redirectURL == "" {
		redirectURL = "/"
	}

	// Encode redirect URL in state parameter
	stateData := map[string]string{
		"csrf":     state,
		"redirect": redirectURL,
	}
	stateJSON, _ := json.Marshal(stateData)
	encodedState := base64.StdEncoding.EncodeToString(stateJSON)

	// Store CSRF token in secure cookie
	ctx.SetSameSite(http.SameSiteStrictMode)
	ctx.SetCookie("auth_state", state, 600, "/", "", true, true) // 10 min expiry

	// Redirect to Auth0
	authURL := a.oauth.AuthCodeURL(encodedState)
	ctx.Redirect(http.StatusTemporaryRedirect, authURL)
}

// CallbackHandler handles OAuth2 callback from Auth0
func (a *zAuth) CallbackHandler(ctx *gin.Context) {
	// Verify state parameter
	encodedState := ctx.Query("state")
	if encodedState == "" {
		ctx.Set("error_message", "Missing state parameter")
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Decode state
	stateJSON, err := base64.StdEncoding.DecodeString(encodedState)
	if err != nil {
		ctx.Set("error_message", "Invalid state parameter")
		ctx.Status(http.StatusBadRequest)
		return
	}

	var stateData map[string]string
	if err := json.Unmarshal(stateJSON, &stateData); err != nil {
		ctx.Set("error_message", "Invalid state parameter")
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Verify CSRF token
	csrfCookie, err := ctx.Request.Cookie("auth_state")
	if err != nil || csrfCookie.Value != stateData["csrf"] {
		ctx.Set("error_message", "Invalid state parameter")
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Clear CSRF cookie
	ctx.SetCookie("auth_state", "", -1, "/", "", true, true)

	// Exchange code for token
	token, err := a.oauth.Exchange(ctx.Request.Context(), ctx.Query("code"))
	if err != nil {
		Log.Error("Failed to exchange code for token", zap.Error(err))
		ctx.Set("error_message", "Failed to complete authentication")
		ctx.Status(http.StatusUnauthorized)
		return
	}

	// Get ID token
	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		ctx.Set("error_message", "No ID token received")
		ctx.Status(http.StatusUnauthorized)
		return
	}

	// Validate the ID token
	claims, err := a.ValidateToken(idToken)
	if err != nil {
		Log.Error("Failed to validate ID token", zap.Error(err))
		ctx.Set("error_message", "Invalid ID token")
		ctx.Status(http.StatusUnauthorized)
		return
	}

	// Store token in secure cookie
	ctx.SetSameSite(http.SameSiteStrictMode)
	maxAge := int(time.Until(claims.ExpiresAt.Time).Seconds())
	ctx.SetCookie("auth_token", idToken, maxAge, "/", "", true, true)

	// Cache user data in Redis
	userData := &Auth0UserData{
		Sub:         claims.Sub,
		Email:       claims.Email,
		Name:        claims.Name,
		Permissions: claims.Permissions,
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if err := a.cacheUserData(claims.Sub, userData, ttl); err != nil {
		Log.Warn("Failed to cache user data", zap.Error(err))
	}

	// Redirect to intended destination
	redirectURL := stateData["redirect"]
	if redirectURL == "" {
		redirectURL = "/"
	}

	ctx.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// LogoutHandler handles user logout
func (a *zAuth) LogoutHandler(ctx *gin.Context) {
	// Clear auth cookie
	ctx.SetCookie("auth_token", "", -1, "/", "", true, true)

	// Build Auth0 logout URL
	logoutURL, err := url.Parse(fmt.Sprintf("https://%s/v2/logout", config.AuthDomain()))
	if err != nil {
		Log.Error("Failed to parse logout URL", zap.Error(err))
		ctx.Set("error_message", "Failed to logout")
		ctx.Status(http.StatusInternalServerError)
		return
	}

	// Determine return URL
	scheme := "http"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}
	returnTo := fmt.Sprintf("%s://%s", scheme, ctx.Request.Host)

	// Add query parameters
	params := url.Values{}
	params.Add("returnTo", returnTo)
	params.Add("client_id", a.oauth.ClientID)
	logoutURL.RawQuery = params.Encode()

	ctx.Redirect(http.StatusTemporaryRedirect, logoutURL.String())
}
