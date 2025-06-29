package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	
	"zbz/rocco"
	"zbz/capitan"
)

// Example data models with cereal scoping
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email" scope:"admin,owner"`
	SSN      string `json:"ssn" scope:"admin" validate:"ssn"`
	Salary   int    `json:"salary" scope:"admin,hr"`
	Profile  string `json:"profile"`
}

type Document struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content" scope:"owner,admin"`
	Secret   string `json:"secret" scope:"admin"`
	AuthorID string `json:"author_id"`
}

func main() {
	fmt.Println("🚀 Rocco + Cereal Integration Demo")
	fmt.Println("===================================")
	
	// Set up event listeners to see the magic
	setupEventListeners()
	
	// Demo 1: User data with automatic field filtering
	fmt.Println("\n🔐 Demo 1: Automatic field-level security")
	demoFieldLevelSecurity()
	
	// Demo 2: Content-aware authorization with data scoping
	fmt.Println("\n📄 Demo 2: Content-aware authorization")
	demoContentAwareAuth()
	
	// Demo 3: Input validation preventing privilege escalation
	fmt.Println("\n⚡ Demo 3: Privilege escalation prevention")
	demoPrivilegeEscalation()
}

func setupEventListeners() {
	// Register simple event handlers to show the events firing
	capitan.RegisterByteHandler("auth.success", func(data []byte) error {
		var event map[string]any
		json.Unmarshal(data, &event)
		fmt.Printf("  🎉 AUTH SUCCESS: %s logged in via %s\n", 
			event["username"], event["provider"])
		return nil
	})
	
	capitan.RegisterByteHandler("auth.denied", func(data []byte) error {
		var event map[string]any
		json.Unmarshal(data, &event)
		fmt.Printf("  🚫 AUTH DENIED: %s (rule: %s)\n", 
			event["error"], event["rule_name"])
		return nil
	})
	
	capitan.RegisterByteHandler("security.data_filtered", func(data []byte) error {
		var event map[string]any
		json.Unmarshal(data, &event)
		fmt.Printf("  🔒 DATA FILTERED: %s for user %s\n", 
			event["data_type"], event["username"])
		return nil
	})
	
	capitan.RegisterByteHandler("security.privilege_escalation_attempt", func(data []byte) error {
		var event map[string]any
		json.Unmarshal(data, &event)
		fmt.Printf("  🚨 SECURITY ALERT: Privilege escalation attempt by %s\n", 
			event["username"])
		return nil
	})
}

func demoFieldLevelSecurity() {
	// Create test users with different permissions
	adminUser := &rocco.Identity{
		ID:          "admin-123",
		Username:    "admin",
		Permissions: []string{"admin"},
		Roles:       []string{"admin"},
	}
	
	regularUser := &rocco.Identity{
		ID:          "user-456", 
		Username:    "john",
		Permissions: []string{"user"},
		Roles:       []string{"user"},
	}
	
	// Sample user data
	userData := User{
		ID:       "user-789",
		Username: "jane_doe",
		Email:    "jane@company.com",
		SSN:      "123-45-6789",
		Salary:   75000,
		Profile:  "Software Engineer",
	}
	
	fmt.Printf("  Original data: %+v\n", userData)
	
	// Admin sees everything
	adminCtx := rocco.NewSecurityContext(adminUser)
	adminFiltered, _ := adminCtx.FilterData(userData)
	fmt.Printf("  Admin view: %+v\n", adminFiltered)
	
	// Regular user sees redacted sensitive fields
	userCtx := rocco.NewSecurityContext(regularUser)
	userFiltered, _ := userCtx.FilterData(userData)
	fmt.Printf("  User view: %+v\n", userFiltered)
}

func demoContentAwareAuth() {
	// Set up rocco auth
	auth := rocco.Default()
	
	// Create a document access handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get security context
		secCtx, ok := rocco.GetSecurityContext(r.Context())
		if !ok {
			http.Error(w, "No security context", http.StatusUnauthorized)
			return
		}
		
		// Sample document (normally from database)
		doc := Document{
			ID:       "doc-123",
			Title:    "Project Roadmap",
			Content:  "Confidential business strategy...",
			Secret:   "TOP SECRET: Acquisition plans",
			AuthorID: "admin-123",
		}
		
		// Apply security filtering automatically
		filteredDoc, err := secCtx.FilterData(doc)
		if err != nil {
			http.Error(w, "Security filtering failed", http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filteredDoc)
	})
	
	// Apply security middleware
	secureHandler := rocco.SecureDataRule("/api/docs/.*", 
		func(ctx context.Context) (any, error) {
			return Document{}, nil // Placeholder data extractor
		})
	
	protected := rocco.BouncerMiddleware(secureHandler)(rocco.Middleware()(handler))
	
	// Test with admin user
	adminCreds := rocco.Credentials{
		Type:     "password",
		Username: "admin",
		Password: "admin",
	}
	
	adminIdentity, _ := auth.Authenticate(context.Background(), adminCreds)
	
	req := httptest.NewRequest("GET", "/api/docs/123", nil)
	req.Header.Set("Authorization", "Bearer "+adminIdentity.AccessToken)
	w := httptest.NewRecorder()
	
	protected.ServeHTTP(w, req)
	
	fmt.Printf("  Admin document access: %s\n", w.Body.String())
}

func demoPrivilegeEscalation() {
	// Simulate a user trying to set admin-only fields
	regularUser := &rocco.Identity{
		ID:          "user-456",
		Username:    "attacker",
		Permissions: []string{"user"},
		Roles:       []string{"user"},
	}
	
	// Malicious input trying to set admin fields
	maliciousInput := User{
		ID:       "user-456",
		Username: "attacker",
		Email:    "attacker@evil.com", // Regular user can't set email
		SSN:      "000-00-0000",       // Regular user can't set SSN
		Salary:   1000000,             // Regular user can't set salary
		Profile:  "Updated profile",   // This should be allowed
	}
	
	secCtx := rocco.NewSecurityContext(regularUser)
	
	fmt.Printf("  Malicious input: %+v\n", maliciousInput)
	
	// This will trigger security validation and emit alerts
	err := secCtx.ValidateInput(&maliciousInput)
	if err != nil {
		fmt.Printf("  ✅ Attack prevented: %v\n", err)
	} else {
		fmt.Printf("  ❌ Attack succeeded - this shouldn't happen!\n")
	}
}