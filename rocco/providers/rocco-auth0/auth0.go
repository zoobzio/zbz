package auth0

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"zbz/rocco"
)

// Provider implements Auth0 authentication
type Provider struct {
	config       rocco.ProviderConfig
	domain       string
	clientID     string
	clientSecret string
	audience     string
	scopes       []string
	httpClient   *http.Client
}

// NewProvider creates a new Auth0 provider
func NewProvider() *Provider {
	return &Provider{
		scopes:     []string{"openid", "profile", "email"},
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "rocco-auth0"
}

// Type returns the provider type
func (p *Provider) Type() rocco.ProviderType {
	return rocco.ProviderTypeOIDC
}

// Configure configures the Auth0 provider
func (p *Provider) Configure(config rocco.ProviderConfig) error {
	p.config = config
	
	settings := config.Settings
	
	domain, ok := settings["domain"].(string)
	if !ok {
		return fmt.Errorf("Auth0 domain is required")
	}
	p.domain = domain
	
	clientID, ok := settings["client_id"].(string)
	if !ok {
		return fmt.Errorf("Auth0 client ID is required")
	}
	p.clientID = clientID
	
	clientSecret, ok := settings["client_secret"].(string)
	if !ok {
		return fmt.Errorf("Auth0 client secret is required")
	}
	p.clientSecret = clientSecret
	
	if audience, ok := settings["audience"].(string); ok {
		p.audience = audience
	}
	
	if scopes, ok := settings["scopes"].([]string); ok {
		p.scopes = scopes
	}
	
	return nil
}

// Validate validates the provider configuration
func (p *Provider) Validate() error {
	if p.domain == "" {
		return fmt.Errorf("Auth0 domain is required")
	}
	if p.clientID == "" {
		return fmt.Errorf("Auth0 client ID is required")
	}
	if p.clientSecret == "" {
		return fmt.Errorf("Auth0 client secret is required")
	}
	return nil
}

// Authenticate authenticates using Auth0 OAuth2 flow
func (p *Provider) Authenticate(ctx context.Context, credentials rocco.Credentials) (*rocco.Identity, error) {
	switch credentials.Type {
	case "authorization_code":
		return p.exchangeCodeForToken(ctx, credentials)
	case "client_credentials":
		return p.clientCredentialsFlow(ctx, credentials)
	default:
		return nil, fmt.Errorf("unsupported credential type: %s", credentials.Type)
	}
}

// exchangeCodeForToken exchanges authorization code for tokens
func (p *Provider) exchangeCodeForToken(ctx context.Context, credentials rocco.Credentials) (*rocco.Identity, error) {
	// Prepare token request
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("code", credentials.Code)
	data.Set("redirect_uri", credentials.Extra["redirect_uri"])
	
	// Make token request
	tokenURL := fmt.Sprintf("https://%s/oauth/token", p.domain)
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse token response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	
	// Get user info
	userInfo, err := p.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	
	// Build identity
	identity := &rocco.Identity{
		ID:           userInfo["sub"].(string),
		Provider:     p.Name(),
		Username:     getStringFromMap(userInfo, "preferred_username", userInfo["sub"].(string)),
		Email:        getStringFromMap(userInfo, "email", ""),
		DisplayName:  getStringFromMap(userInfo, "name", ""),
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Scopes:       strings.Fields(tokenResp.Scope),
		Claims:       userInfo,
		IssuedAt:     time.Now(),
		LastActive:   time.Now(),
	}
	
	// Extract roles from user metadata if available
	if userMetadata, ok := userInfo["user_metadata"].(map[string]interface{}); ok {
		if roles, ok := userMetadata["roles"].([]interface{}); ok {
			identity.Roles = interfaceSliceToStringSlice(roles)
		}
	}
	
	// Extract app metadata
	if appMetadata, ok := userInfo["app_metadata"].(map[string]interface{}); ok {
		if permissions, ok := appMetadata["permissions"].([]interface{}); ok {
			identity.Permissions = interfaceSliceToStringSlice(permissions)
		}
		if roles, ok := appMetadata["roles"].([]interface{}); ok {
			identity.Roles = append(identity.Roles, interfaceSliceToStringSlice(roles)...)
		}
	}
	
	return identity, nil
}

// clientCredentialsFlow implements client credentials flow for machine-to-machine auth
func (p *Provider) clientCredentialsFlow(ctx context.Context, credentials rocco.Credentials) (*rocco.Identity, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	if p.audience != "" {
		data.Set("audience", p.audience)
	}
	
	tokenURL := fmt.Sprintf("https://%s/oauth/token", p.domain)
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
		Scope       string `json:"scope"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	
	// For client credentials, create a service identity
	identity := &rocco.Identity{
		ID:          p.clientID,
		Provider:    p.Name(),
		Username:    p.clientID,
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Scopes:      strings.Fields(tokenResp.Scope),
		IssuedAt:    time.Now(),
		LastActive:  time.Now(),
	}
	
	return identity, nil
}

// getUserInfo retrieves user information from Auth0
func (p *Provider) getUserInfo(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	userInfoURL := fmt.Sprintf("https://%s/userinfo", p.domain)
	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}
	
	return userInfo, nil
}

// Refresh refreshes an access token using the refresh token
func (p *Provider) Refresh(ctx context.Context, refreshToken string) (*rocco.Identity, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("refresh_token", refreshToken)
	
	tokenURL := fmt.Sprintf("https://%s/oauth/token", p.domain)
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}
	
	// Get updated user info
	userInfo, err := p.getUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	
	// Build refreshed identity
	identity := &rocco.Identity{
		ID:           userInfo["sub"].(string),
		Provider:     p.Name(),
		Username:     getStringFromMap(userInfo, "preferred_username", userInfo["sub"].(string)),
		Email:        getStringFromMap(userInfo, "email", ""),
		DisplayName:  getStringFromMap(userInfo, "name", ""),
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Scopes:       strings.Fields(tokenResp.Scope),
		Claims:       userInfo,
		IssuedAt:     time.Now(),
		LastActive:   time.Now(),
	}
	
	return identity, nil
}

// Revoke revokes tokens (Auth0 doesn't provide a standard revocation endpoint)
func (p *Provider) Revoke(ctx context.Context, token string) error {
	// Auth0 doesn't provide a standard token revocation endpoint
	// Tokens will expire naturally
	return nil
}

// User management operations (Auth0 Management API)

func (p *Provider) CreateUser(ctx context.Context, user rocco.UserInfo) error {
	return fmt.Errorf("user creation requires Auth0 Management API integration")
}

func (p *Provider) UpdateUser(ctx context.Context, userID string, user rocco.UserInfo) error {
	return fmt.Errorf("user updates require Auth0 Management API integration")
}

func (p *Provider) DeleteUser(ctx context.Context, userID string) error {
	return fmt.Errorf("user deletion requires Auth0 Management API integration")
}

func (p *Provider) GetUser(ctx context.Context, userID string) (*rocco.UserInfo, error) {
	return nil, fmt.Errorf("user retrieval requires Auth0 Management API integration")
}

// Helper functions

func getStringFromMap(m map[string]interface{}, key, defaultValue string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return defaultValue
}

func interfaceSliceToStringSlice(slice []interface{}) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// Configuration helper

// Auth0Config creates a provider configuration for Auth0
func Auth0Config(domain, clientID, clientSecret string, options ...Auth0Option) rocco.ProviderConfig {
	config := rocco.ProviderConfig{
		Enabled:     true,
		DisplayName: "Auth0",
		Settings: map[string]any{
			"domain":        domain,
			"client_id":     clientID,
			"client_secret": clientSecret,
			"scopes":        []string{"openid", "profile", "email"},
		},
	}
	
	// Apply options
	for _, opt := range options {
		opt(&config)
	}
	
	return config
}

// Auth0Option configures Auth0 provider
type Auth0Option func(*rocco.ProviderConfig)

// WithAudience sets the Auth0 audience
func WithAudience(audience string) Auth0Option {
	return func(config *rocco.ProviderConfig) {
		config.Settings["audience"] = audience
	}
}

// WithScopes sets the Auth0 scopes
func WithScopes(scopes []string) Auth0Option {
	return func(config *rocco.ProviderConfig) {
		config.Settings["scopes"] = scopes
	}
}