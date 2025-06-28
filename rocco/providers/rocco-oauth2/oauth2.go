package oauth2

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

// Provider implements generic OAuth2 authentication
type Provider struct {
	config       rocco.ProviderConfig
	authURL      string
	tokenURL     string
	userInfoURL  string
	clientID     string
	clientSecret string
	scopes       []string
	httpClient   *http.Client
}

// NewProvider creates a new OAuth2 provider
func NewProvider() *Provider {
	return &Provider{
		scopes:     []string{"openid", "profile", "email"},
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "rocco-oauth2"
}

// Type returns the provider type
func (p *Provider) Type() rocco.ProviderType {
	return rocco.ProviderTypeOAuth2
}

// Configure configures the OAuth2 provider
func (p *Provider) Configure(config rocco.ProviderConfig) error {
	p.config = config
	
	settings := config.Settings
	
	authURL, ok := settings["auth_url"].(string)
	if !ok {
		return fmt.Errorf("OAuth2 authorization URL is required")
	}
	p.authURL = authURL
	
	tokenURL, ok := settings["token_url"].(string)
	if !ok {
		return fmt.Errorf("OAuth2 token URL is required")
	}
	p.tokenURL = tokenURL
	
	clientID, ok := settings["client_id"].(string)
	if !ok {
		return fmt.Errorf("OAuth2 client ID is required")
	}
	p.clientID = clientID
	
	clientSecret, ok := settings["client_secret"].(string)
	if !ok {
		return fmt.Errorf("OAuth2 client secret is required")
	}
	p.clientSecret = clientSecret
	
	if userInfoURL, ok := settings["userinfo_url"].(string); ok {
		p.userInfoURL = userInfoURL
	}
	
	if scopes, ok := settings["scopes"].([]string); ok {
		p.scopes = scopes
	}
	
	return nil
}

// Validate validates the provider configuration
func (p *Provider) Validate() error {
	if p.authURL == "" {
		return fmt.Errorf("OAuth2 authorization URL is required")
	}
	if p.tokenURL == "" {
		return fmt.Errorf("OAuth2 token URL is required")
	}
	if p.clientID == "" {
		return fmt.Errorf("OAuth2 client ID is required")
	}
	if p.clientSecret == "" {
		return fmt.Errorf("OAuth2 client secret is required")
	}
	return nil
}

// Authenticate authenticates using OAuth2 authorization code flow
func (p *Provider) Authenticate(ctx context.Context, credentials rocco.Credentials) (*rocco.Identity, error) {
	if credentials.Type != "authorization_code" {
		return nil, fmt.Errorf("OAuth2 provider only supports authorization_code flow")
	}
	
	return p.exchangeCodeForToken(ctx, credentials)
}

// exchangeCodeForToken exchanges authorization code for access token
func (p *Provider) exchangeCodeForToken(ctx context.Context, credentials rocco.Credentials) (*rocco.Identity, error) {
	// Prepare token request
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("code", credentials.Code)
	
	if redirectURI, exists := credentials.Extra["redirect_uri"]; exists {
		data.Set("redirect_uri", redirectURI)
	}
	
	// Make token request
	req, err := http.NewRequestWithContext(ctx, "POST", p.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	
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
	
	// Get user info if userinfo endpoint is configured
	var userInfo map[string]interface{}
	if p.userInfoURL != "" {
		userInfo, err = p.getUserInfo(ctx, tokenResp.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to get user info: %w", err)
		}
	}
	
	// Build identity
	identity := &rocco.Identity{
		Provider:     p.Name(),
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Scopes:       strings.Fields(tokenResp.Scope),
		Claims:       userInfo,
		IssuedAt:     time.Now(),
		LastActive:   time.Now(),
	}
	
	// Extract identity fields from user info
	if userInfo != nil {
		if sub, ok := userInfo["sub"].(string); ok {
			identity.ID = sub
		} else if id, ok := userInfo["id"].(string); ok {
			identity.ID = id
		} else {
			identity.ID = p.clientID // Fallback to client ID
		}
		
		if username, ok := userInfo["preferred_username"].(string); ok {
			identity.Username = username
		} else if username, ok := userInfo["username"].(string); ok {
			identity.Username = username
		} else if email, ok := userInfo["email"].(string); ok {
			identity.Username = email
		}
		
		if email, ok := userInfo["email"].(string); ok {
			identity.Email = email
		}
		
		if name, ok := userInfo["name"].(string); ok {
			identity.DisplayName = name
		}
		
		// Extract roles if present
		if roles, ok := userInfo["roles"].([]interface{}); ok {
			identity.Roles = interfaceSliceToStringSlice(roles)
		}
		
		// Extract permissions if present
		if permissions, ok := userInfo["permissions"].([]interface{}); ok {
			identity.Permissions = interfaceSliceToStringSlice(permissions)
		}
	}
	
	return identity, nil
}

// getUserInfo retrieves user information from userinfo endpoint
func (p *Provider) getUserInfo(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	
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
	
	req, err := http.NewRequestWithContext(ctx, "POST", p.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	
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
	var userInfo map[string]interface{}
	if p.userInfoURL != "" {
		userInfo, err = p.getUserInfo(ctx, tokenResp.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to get user info: %w", err)
		}
	}
	
	// Build refreshed identity
	identity := &rocco.Identity{
		Provider:     p.Name(),
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		Scopes:       strings.Fields(tokenResp.Scope),
		Claims:       userInfo,
		IssuedAt:     time.Now(),
		LastActive:   time.Now(),
	}
	
	// Update identity fields from user info
	if userInfo != nil {
		if sub, ok := userInfo["sub"].(string); ok {
			identity.ID = sub
		}
		if username, ok := userInfo["preferred_username"].(string); ok {
			identity.Username = username
		}
		if email, ok := userInfo["email"].(string); ok {
			identity.Email = email
		}
		if name, ok := userInfo["name"].(string); ok {
			identity.DisplayName = name
		}
	}
	
	return identity, nil
}

// Revoke revokes a token (if the provider supports it)
func (p *Provider) Revoke(ctx context.Context, token string) error {
	// Check if revocation endpoint is configured
	if revokeURL, ok := p.config.Settings["revoke_url"].(string); ok {
		data := url.Values{}
		data.Set("token", token)
		data.Set("client_id", p.clientID)
		data.Set("client_secret", p.clientSecret)
		
		req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, strings.NewReader(data.Encode()))
		if err != nil {
			return fmt.Errorf("failed to create revoke request: %w", err)
		}
		
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		
		resp, err := p.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("revoke request failed: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("revoke request failed with status %d: %s", resp.StatusCode, string(body))
		}
	}
	
	return nil
}

// User management operations (not supported by generic OAuth2)

func (p *Provider) CreateUser(ctx context.Context, user rocco.UserInfo) error {
	return fmt.Errorf("user creation not supported by OAuth2 provider")
}

func (p *Provider) UpdateUser(ctx context.Context, userID string, user rocco.UserInfo) error {
	return fmt.Errorf("user updates not supported by OAuth2 provider")
}

func (p *Provider) DeleteUser(ctx context.Context, userID string) error {
	return fmt.Errorf("user deletion not supported by OAuth2 provider")
}

func (p *Provider) GetUser(ctx context.Context, userID string) (*rocco.UserInfo, error) {
	return nil, fmt.Errorf("user retrieval not supported by OAuth2 provider")
}

// GetAuthorizationURL generates the authorization URL for OAuth2 flow
func (p *Provider) GetAuthorizationURL(redirectURI, state string, extraParams map[string]string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", strings.Join(p.scopes, " "))
	params.Set("state", state)
	
	// Add extra parameters
	for key, value := range extraParams {
		params.Set(key, value)
	}
	
	return p.authURL + "?" + params.Encode()
}

// Helper functions

func interfaceSliceToStringSlice(slice []interface{}) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// Configuration helpers

// GitHubConfig creates a provider configuration for GitHub OAuth2
func GitHubConfig(clientID, clientSecret string) rocco.ProviderConfig {
	return rocco.ProviderConfig{
		Enabled:     true,
		DisplayName: "GitHub",
		Settings: map[string]any{
			"auth_url":     "https://github.com/login/oauth/authorize",
			"token_url":    "https://github.com/login/oauth/access_token",
			"userinfo_url": "https://api.github.com/user",
			"client_id":    clientID,
			"client_secret": clientSecret,
			"scopes":       []string{"user:email"},
		},
	}
}

// GoogleConfig creates a provider configuration for Google OAuth2
func GoogleConfig(clientID, clientSecret string) rocco.ProviderConfig {
	return rocco.ProviderConfig{
		Enabled:     true,
		DisplayName: "Google",
		Settings: map[string]any{
			"auth_url":     "https://accounts.google.com/o/oauth2/v2/auth",
			"token_url":    "https://oauth2.googleapis.com/token",
			"userinfo_url": "https://openidconnect.googleapis.com/v1/userinfo",
			"revoke_url":   "https://oauth2.googleapis.com/revoke",
			"client_id":    clientID,
			"client_secret": clientSecret,
			"scopes":       []string{"openid", "profile", "email"},
		},
	}
}

// DiscordConfig creates a provider configuration for Discord OAuth2
func DiscordConfig(clientID, clientSecret string) rocco.ProviderConfig {
	return rocco.ProviderConfig{
		Enabled:     true,
		DisplayName: "Discord",
		Settings: map[string]any{
			"auth_url":     "https://discord.com/api/oauth2/authorize",
			"token_url":    "https://discord.com/api/oauth2/token",
			"userinfo_url": "https://discord.com/api/users/@me",
			"revoke_url":   "https://discord.com/api/oauth2/token/revoke",
			"client_id":    clientID,
			"client_secret": clientSecret,
			"scopes":       []string{"identify", "email"},
		},
	}
}