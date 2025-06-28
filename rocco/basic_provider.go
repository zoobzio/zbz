package rocco

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// basicProvider implements internal basic username/password authentication
type basicProvider struct {
	config ProviderConfig
	users  map[string]*basicUser
	realm  string
}

// basicUser represents a user in the basic provider
type basicUser struct {
	ID          string            `json:"id"`
	Username    string            `json:"username"`
	Email       string            `json:"email"`
	DisplayName string            `json:"display_name"`
	PasswordHash string           `json:"password_hash"`
	Roles       []string          `json:"roles"`
	Permissions []string          `json:"permissions"`
	Attributes  map[string]any    `json:"attributes"`
	Active      bool              `json:"active"`
	CreatedAt   time.Time         `json:"created_at"`
}

// newBasicProvider creates the internal basic provider
func newBasicProvider() *basicProvider {
	p := &basicProvider{
		users: make(map[string]*basicUser),
		realm: "Rocco Application",
	}
	
	// Create default admin user for zero-config setup
	p.createDefaultAdmin()
	
	return p
}

// createDefaultAdmin creates a default admin user
func (p *basicProvider) createDefaultAdmin() {
	// Only create if no users exist
	if len(p.users) > 0 {
		return
	}
	
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	
	admin := &basicUser{
		ID:           "admin",
		Username:     "admin",
		Email:        "admin@localhost",
		DisplayName:  "Default Administrator",
		PasswordHash: string(hashedPassword),
		Roles:        []string{"admin", "user"},
		Permissions:  []string{"*"}, // Wildcard permission
		Attributes:   make(map[string]any),
		Active:       true,
		CreatedAt:    time.Now(),
	}
	
	p.users["admin"] = admin
}

// Provider interface implementation

func (p *basicProvider) Name() string {
	return "basic"
}

func (p *basicProvider) Type() ProviderType {
	return ProviderTypeBasic
}

func (p *basicProvider) Configure(config ProviderConfig) error {
	p.config = config
	
	if realm, ok := config.Settings["realm"].(string); ok {
		p.realm = realm
	}
	
	return nil
}

func (p *basicProvider) Validate() error {
	return nil
}

func (p *basicProvider) Authenticate(ctx context.Context, credentials Credentials) (*Identity, error) {
	if credentials.Type != "password" {
		return nil, ErrInvalidCredentials
	}
	
	user, exists := p.users[credentials.Username]
	if !exists || !user.Active {
		return nil, ErrInvalidCredentials
	}
	
	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(credentials.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	
	// Build identity
	identity := &Identity{
		ID:          user.ID,
		Provider:    p.Name(),
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Roles:       user.Roles,
		Permissions: user.Permissions,
		Attributes:  user.Attributes,
		IssuedAt:    time.Now(),
		LastActive:  time.Now(),
	}
	
	return identity, nil
}

func (p *basicProvider) Refresh(ctx context.Context, refreshToken string) (*Identity, error) {
	return nil, fmt.Errorf("basic provider does not support token refresh")
}

func (p *basicProvider) Revoke(ctx context.Context, token string) error {
	return nil // No-op for basic provider
}

func (p *basicProvider) CreateUser(ctx context.Context, user UserInfo) error {
	// Check if user already exists
	for _, existingUser := range p.users {
		if existingUser.Username == user.Username || existingUser.Email == user.Email {
			return fmt.Errorf("user already exists")
		}
	}
	
	// Create new user
	password, ok := user.Attributes["password"].(string)
	if !ok || password == "" {
		return fmt.Errorf("password required")
	}
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	
	// Generate ID if not provided
	userID := user.ID
	if userID == "" {
		userID = generateUserID()
	}
	
	basicUser := &basicUser{
		ID:           userID,
		Username:     user.Username,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		PasswordHash: string(hashedPassword),
		Attributes:   user.Attributes,
		Active:       true,
		CreatedAt:    time.Now(),
	}
	
	// Set roles and permissions from attributes
	if roles, ok := user.Attributes["roles"].([]string); ok {
		basicUser.Roles = roles
	}
	if permissions, ok := user.Attributes["permissions"].([]string); ok {
		basicUser.Permissions = permissions
	}
	
	p.users[user.Username] = basicUser
	return nil
}

func (p *basicProvider) UpdateUser(ctx context.Context, userID string, user UserInfo) error {
	// Find user by ID
	var existingUser *basicUser
	for _, basicUser := range p.users {
		if basicUser.ID == userID {
			existingUser = basicUser
			break
		}
	}
	
	if existingUser == nil {
		return fmt.Errorf("user not found")
	}
	
	// Update fields
	if user.Email != "" {
		existingUser.Email = user.Email
	}
	if user.DisplayName != "" {
		existingUser.DisplayName = user.DisplayName
	}
	
	// Update password if provided
	if password, ok := user.Attributes["password"].(string); ok && password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		existingUser.PasswordHash = string(hashedPassword)
	}
	
	// Update roles and permissions
	if roles, ok := user.Attributes["roles"].([]string); ok {
		existingUser.Roles = roles
	}
	if permissions, ok := user.Attributes["permissions"].([]string); ok {
		existingUser.Permissions = permissions
	}
	
	// Update attributes
	for k, v := range user.Attributes {
		if k != "password" {
			existingUser.Attributes[k] = v
		}
	}
	
	return nil
}

func (p *basicProvider) DeleteUser(ctx context.Context, userID string) error {
	for username, user := range p.users {
		if user.ID == userID {
			delete(p.users, username)
			return nil
		}
	}
	
	return fmt.Errorf("user not found")
}

func (p *basicProvider) GetUser(ctx context.Context, userID string) (*UserInfo, error) {
	for _, user := range p.users {
		if user.ID == userID {
			return &UserInfo{
				ID:          user.ID,
				Username:    user.Username,
				Email:       user.Email,
				DisplayName: user.DisplayName,
				Attributes:  user.Attributes,
			}, nil
		}
	}
	
	return nil, fmt.Errorf("user not found")
}

// Internal helper methods for the basic provider

// GetBasicProvider returns the internal basic provider from an Auth instance
func GetBasicProvider(auth Auth) (*basicProvider, error) {
	provider, err := auth.GetProvider("basic")
	if err != nil {
		return nil, err
	}
	
	if basicProv, ok := provider.(*basicProvider); ok {
		return basicProv, nil
	}
	
	return nil, fmt.Errorf("basic provider not found")
}

// Helper functions
func generateUserID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("user_%x", bytes)
}