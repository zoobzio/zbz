package rocco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestZeroConfigSetup(t *testing.T) {
	// Zero-config usage - just works out of the box
	auth := Default()
	
	// Should have basic provider registered
	provider, err := auth.GetProvider("basic")
	if err != nil {
		t.Fatalf("Expected basic provider to be registered: %v", err)
	}
	
	if provider.Name() != "basic" {
		t.Errorf("Expected basic provider, got %s", provider.Name())
	}
}

func TestBasicAuthentication(t *testing.T) {
	// Use default instance with admin user
	auth := Default()
	
	// Authenticate with default admin user
	credentials := Credentials{
		Type:     "password",
		Username: "admin",
		Password: "admin",
	}
	
	identity, err := auth.Authenticate(context.Background(), credentials)
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}
	
	if identity.Username != "admin" {
		t.Errorf("Expected username 'admin', got %s", identity.Username)
	}
	
	if !contains(identity.Roles, "admin") {
		t.Errorf("Expected admin role, got %v", identity.Roles)
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	// Create a test handler that requires auth
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := GetIdentity(r.Context())
		if !ok {
			http.Error(w, "No identity", http.StatusUnauthorized)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"username": identity.Username,
			"provider": identity.Provider,
		})
	})
	
	// Wrap with Rocco middleware
	protected := Middleware()(handler)
	
	// Test without token
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 without token, got %d", w.Code)
	}
	
	// Test with valid token
	auth := Default()
	credentials := Credentials{
		Type:     "password",
		Username: "admin",
		Password: "admin",
	}
	
	identity, err := auth.Authenticate(context.Background(), credentials)
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}
	
	req = httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+identity.AccessToken)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 with valid token, got %d", w.Code)
	}
	
	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)
	
	if response["username"] != "admin" {
		t.Errorf("Expected username 'admin', got %s", response["username"])
	}
}

func TestBouncerMiddleware(t *testing.T) {
	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	
	// Create bouncer rules
	rules := []BouncerRule{
		RequireAuth("/admin/.*"),
		RequireRole("/admin/users/.*", "admin"),
		{
			Name:        "api_access",
			PathPattern: "/api/.*",
			Methods:     []string{"POST", "PUT", "DELETE"},
			RequireAuth: true,
		},
	}
	
	// Wrap with bouncer middleware
	protected := BouncerMiddleware(rules...)(Middleware()(handler))
	
	// Test public endpoint (should work)
	req := httptest.NewRequest("GET", "/public", nil)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for public endpoint, got %d", w.Code)
	}
	
	// Test admin endpoint without auth (should fail)
	req = httptest.NewRequest("GET", "/admin/dashboard", nil)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for admin endpoint without auth, got %d", w.Code)
	}
	
	// Test admin endpoint with auth (should work)
	auth := Default()
	credentials := Credentials{
		Type:     "password",
		Username: "admin",
		Password: "admin",
	}
	
	identity, _ := auth.Authenticate(nil, credentials)
	
	req = httptest.NewRequest("GET", "/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+identity.AccessToken)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for admin endpoint with auth, got %d", w.Code)
	}
}

func TestContentAwareBouncer(t *testing.T) {
	// Create user data access rule
	userDataRule := UserDataRule("/api/users/.*")
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user data"))
	})
	
	protected := BouncerMiddleware(userDataRule)(Middleware()(handler))
	
	// Create a regular user
	auth := Default()
	basicProvider, _ := GetBasicProvider(auth)
	
	userInfo := UserInfo{
		Username: "testuser",
		Email:    "test@example.com",
		Attributes: map[string]any{
			"password": "password",
			"roles":    []string{"user"},
		},
	}
	
	basicProvider.CreateUser(context.Background(), userInfo)
	
	// Authenticate as the user
	credentials := Credentials{
		Type:     "password",
		Username: "testuser",
		Password: "password",
	}
	
	identity, _ := auth.Authenticate(nil, credentials)
	
	// Test access to own data (should work)
	req := httptest.NewRequest("GET", "/api/users/"+identity.ID, nil)
	req.Header.Set("Authorization", "Bearer "+identity.AccessToken)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for own user data, got %d", w.Code)
	}
	
	// Test access to other user's data (should fail)
	req = httptest.NewRequest("GET", "/api/users/other-user-id", nil)
	req.Header.Set("Authorization", "Bearer "+identity.AccessToken)
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 for other user data, got %d", w.Code)
	}
}

func TestProviderHotSwap(t *testing.T) {
	// Start with basic provider
	auth := NewDefault()
	
	// Authenticate with basic
	credentials := Credentials{
		Type:     "password",
		Username: "admin",
		Password: "admin",
	}
	
	identity, err := auth.Authenticate(context.Background(), credentials)
	if err != nil {
		t.Fatalf("Basic auth failed: %v", err)
	}
	
	if identity.Provider != "basic" {
		t.Errorf("Expected basic provider, got %s", identity.Provider)
	}
	
	// Hot-swap to a mock OIDC provider (would be external package)
	mockProvider := &mockOIDCProvider{}
	auth.RegisterProvider("mock-oidc", mockProvider)
	auth.SetDefaultProvider("mock-oidc")
	
	// Now authentication uses the new provider
	oidcCredentials := Credentials{
		Type: "token",
		Token: "mock-oidc-token",
	}
	
	identity, err = auth.Authenticate(nil, oidcCredentials)
	if err != nil {
		t.Fatalf("OIDC auth failed: %v", err)
	}
	
	if identity.Provider != "mock-oidc" {
		t.Errorf("Expected mock-oidc provider, got %s", identity.Provider)
	}
}

func TestConvenienceFunctions(t *testing.T) {
	// Test package-level convenience functions
	err := CreateUser("newuser", "password", "new@example.com", []string{"user"})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	
	// Test authentication
	credentials := Credentials{
		Type:     "password",
		Username: "newuser",
		Password: "password",
	}
	
	identity, err := Default().Authenticate(context.Background(), credentials)
	
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}
	
	if identity.Username != "newuser" {
		t.Errorf("Expected username 'newuser', got %s", identity.Username)
	}
}

// Mock OIDC provider for testing
type mockOIDCProvider struct{}

func (m *mockOIDCProvider) Name() string { return "mock-oidc" }
func (m *mockOIDCProvider) Type() ProviderType { return ProviderTypeOIDC }
func (m *mockOIDCProvider) Configure(config ProviderConfig) error { return nil }
func (m *mockOIDCProvider) Validate() error { return nil }

func (m *mockOIDCProvider) Authenticate(ctx context.Context, credentials Credentials) (*Identity, error) {
	if credentials.Type == "token" && credentials.Token == "mock-oidc-token" {
		return &Identity{
			ID:       "oidc-user-123",
			Provider: "mock-oidc",
			Username: "oidcuser",
			Email:    "oidc@example.com",
			Roles:    []string{"user"},
		}, nil
	}
	return nil, ErrInvalidCredentials
}

func (m *mockOIDCProvider) Refresh(ctx context.Context, refreshToken string) (*Identity, error) {
	return nil, nil
}

func (m *mockOIDCProvider) Revoke(ctx context.Context, token string) error {
	return nil
}

func (m *mockOIDCProvider) CreateUser(ctx context.Context, user UserInfo) error {
	return fmt.Errorf("not supported")
}

func (m *mockOIDCProvider) UpdateUser(ctx context.Context, userID string, user UserInfo) error {
	return fmt.Errorf("not supported")
}

func (m *mockOIDCProvider) DeleteUser(ctx context.Context, userID string) error {
	return fmt.Errorf("not supported")
}

func (m *mockOIDCProvider) GetUser(ctx context.Context, userID string) (*UserInfo, error) {
	return nil, fmt.Errorf("not supported")
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}