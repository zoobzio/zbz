package universal

import (
	"context"
	"fmt"
	"log"
	"time"

	"zbz/universal"
)

// ComplianceTest validates that all providers correctly implement universal data access
func ComplianceTest() {
	fmt.Println("🧪 Universal Interface Compliance Test")
	fmt.Println("======================================")
	fmt.Println()
	
	ctx := context.Background()
	
	// Setup test providers
	setupTestProviders()
	
	// Test data types
	type TestUser struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	
	type TestSession struct {
		ID      string    `json:"id"`
		UserID  int       `json:"user_id"`
		Created time.Time `json:"created"`
	}
	
	// Test cases - each provider should behave identically
	providers := []struct {
		name   string
		scheme string
		uriPrefix string
	}{
		{
			name:   "Database",
			scheme: "db",
			uriPrefix: "db://users/",
		},
		{
			name:   "Cache", 
			scheme: "cache",
			uriPrefix: "cache://users/",
		},
		{
			name:   "Storage",
			scheme: "storage",
			uriPrefix: "storage://users/",
		},
		{
			name:   "Search",
			scheme: "search",
			uriPrefix: "search://users/",
		},
	}
	
	// Run compliance tests for each provider
	for _, provider := range providers {
		fmt.Printf("Testing %s provider (scheme: %s)...\n", provider.name, provider.scheme)
		
		// Test basic CRUD operations using URIs
		testBasicCRUD[TestUser](ctx, provider.uriPrefix)
		
		// Test pattern matching
		testPatternMatching[TestUser](ctx, provider.uriPrefix)
		
		// Test error handling
		testErrorHandling[TestUser](ctx, provider.uriPrefix)
		
		fmt.Printf("✅ %s provider passed all compliance tests\n\n", provider.name)
	}
	
	fmt.Println("✅ All providers passed compliance tests!")
}

// testBasicCRUD tests basic CRUD operations
func testBasicCRUD[T any](ctx context.Context, uriPrefix string) {
	type TestUser struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	
	user := TestUser{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}
	
	// Create URI for user
	userURI, err := universal.ParseResourceURI(uriPrefix + "1")
	if err != nil {
		log.Fatalf("Failed to parse URI: %v", err)
	}
	
	// Test Set
	if err := universal.Set(ctx, userURI, user); err != nil {
		log.Fatalf("Set failed: %v", err)
	}
	
	// Test Get
	retrieved, err := universal.Get[TestUser](ctx, userURI)
	if err != nil {
		log.Fatalf("Get failed: %v", err)
	}
	
	// Verify data
	if retrieved.ID != user.ID || retrieved.Name != user.Name {
		log.Fatalf("Retrieved data doesn't match: got %+v, want %+v", retrieved, user)
	}
	
	// Test Exists
	exists, err := universal.Exists[TestUser](ctx, userURI)
	if err != nil {
		log.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		log.Fatal("Resource should exist after Set")
	}
	
	// Test Delete
	if err := universal.Delete[TestUser](ctx, userURI); err != nil {
		log.Fatalf("Delete failed: %v", err)
	}
	
	// Verify deletion
	exists, err = universal.Exists[TestUser](ctx, userURI)
	if err != nil {
		log.Fatalf("Exists after delete failed: %v", err)
	}
	if exists {
		log.Fatal("Resource should not exist after Delete")
	}
}

// testPatternMatching tests pattern-based operations
func testPatternMatching[T any](ctx context.Context, uriPrefix string) {
	type TestUser struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	// Create multiple items
	for i := 1; i <= 5; i++ {
		user := TestUser{
			ID:    i,
			Name:  fmt.Sprintf("User %d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
		}
		
		uri, _ := universal.ParseResourceURI(fmt.Sprintf("%s%d", uriPrefix, i))
		if err := universal.Set(ctx, uri, user); err != nil {
			log.Fatalf("Failed to create test user %d: %v", i, err)
		}
	}
	
	// Test List with pattern
	pattern, _ := universal.ParseResourceURI(uriPrefix + "*")
	items, err := universal.List[TestUser](ctx, pattern)
	if err != nil {
		log.Fatalf("List failed: %v", err)
	}
	
	if len(items) != 5 {
		log.Fatalf("Expected 5 items, got %d", len(items))
	}
	
	// Test Count
	count, err := universal.Count[TestUser](ctx, pattern)
	if err != nil {
		log.Fatalf("Count failed: %v", err)
	}
	
	if count != 5 {
		log.Fatalf("Expected count 5, got %d", count)
	}
	
	// Cleanup
	for i := 1; i <= 5; i++ {
		uri, _ := universal.ParseResourceURI(fmt.Sprintf("%s%d", uriPrefix, i))
		_ = universal.Delete[TestUser](ctx, uri)
	}
}

// testErrorHandling tests error cases
func testErrorHandling[T any](ctx context.Context, uriPrefix string) {
	type TestUser struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	// Test Get non-existent
	uri, _ := universal.ParseResourceURI(uriPrefix + "nonexistent")
	_, err := universal.Get[TestUser](ctx, uri)
	if err == nil {
		log.Fatal("Expected error for non-existent resource")
	}
	
	// Test Delete non-existent (should not error in most implementations)
	err = universal.Delete[TestUser](ctx, uri)
	// Most providers don't error on delete non-existent
	
	// Test Exists non-existent
	exists, err := universal.Exists[TestUser](ctx, uri)
	if err != nil {
		log.Fatalf("Exists should not error: %v", err)
	}
	if exists {
		log.Fatal("Non-existent resource should return false")
	}
}

// setupTestProviders initializes test providers
func setupTestProviders() {
	// In a real test, you would register actual providers here
	// For now, we'll use mock providers that satisfy the interface
	
	// Example:
	// universal.SetupDatabase(mockdb.NewProvider, universal.DefaultDataProviderConfig())
	// universal.SetupCache(mockcache.NewProvider, universal.DefaultDataProviderConfig())
	// etc.
	
	fmt.Println("⚠️  Note: This test requires actual provider implementations")
	fmt.Println("   In production, register real providers before running tests")
	fmt.Println()
}


