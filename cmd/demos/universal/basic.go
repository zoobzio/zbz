package universal

import (
	"context"
	"fmt"
	"log"
	"time"

	"zbz/universal"
)

// BasicDemo demonstrates the core universal data access pattern
func BasicDemo() {
	fmt.Println("🚀 Universal Data Access - Basic Demo")
	fmt.Println("=====================================")
	fmt.Println()
	
	ctx := context.Background()
	
	// Setup mock providers to avoid external dependencies
	setupMockProviders()
	
	// Demo data types
	type User struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	
	type Session struct {
		ID     string    `json:"id"`
		UserID int       `json:"user_id"`
		Expires time.Time `json:"expires"`
	}
	
	type Document struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
		Tags    []string `json:"tags"`
	}
	
	fmt.Println("📋 Demo: Same Interface, Different Providers")
	fmt.Println("============================================")
	
	// Using URI-based routing - the URI scheme determines the provider!
	
	// 1. Database operations
	fmt.Println("\n🗄️  Database Provider (scheme: db://)")
	user := User{ID: 1, Name: "Alice Smith", Email: "alice@example.com"}
	userURI := universal.NewResourceURI("db://users/1")
	
	// Set user
	if err := universal.Set(ctx, userURI, user); err != nil {
		log.Fatalf("Failed to set user: %v", err)
	}
	fmt.Printf("   ✅ Created user: %+v\n", user)
	
	// Get user
	retrieved, err := universal.Get[User](ctx, userURI)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("   ✅ Retrieved user: %+v\n", retrieved)
	
	// 2. Cache operations
	fmt.Println("\n💾 Cache Provider (scheme: cache://)")
	session := Session{
		ID:     "sess_123",
		UserID: 1,
		Expires: time.Now().Add(24 * time.Hour),
	}
	sessionURI := universal.NewResourceURI("cache://sessions/sess_123")
	
	// Set session
	if err := universal.Set(ctx, sessionURI, session); err != nil {
		log.Fatalf("Failed to set session: %v", err)
	}
	fmt.Printf("   ✅ Cached session: %+v\n", session)
	
	// Check if exists
	exists, err := universal.Exists[Session](ctx, sessionURI)
	if err != nil {
		log.Fatalf("Failed to check session existence: %v", err)
	}
	fmt.Printf("   ✅ Session exists: %v\n", exists)
	
	// 3. Storage operations
	fmt.Println("\n📁 Storage Provider (scheme: storage://)")
	doc := Document{
		ID:      "doc_456",
		Title:   "Universal Data Access Guide",
		Content: "A comprehensive guide to the universal data pattern...",
		Tags:    []string{"guide", "universal", "zbz"},
	}
	docURI := universal.NewResourceURI("storage://documents/doc_456")
	
	// Set document
	if err := universal.Set(ctx, docURI, doc); err != nil {
		log.Fatalf("Failed to store document: %v", err)
	}
	fmt.Printf("   ✅ Stored document: %s\n", doc.Title)
	
	// List documents with pattern
	pattern := universal.NewResourceURI("storage://documents/*")
	docs, err := universal.List[Document](ctx, pattern)
	if err != nil {
		log.Fatalf("Failed to list documents: %v", err)
	}
	fmt.Printf("   ✅ Found %d documents\n", len(docs))
	
	// 4. Search operations
	fmt.Println("\n🔍 Search Provider (scheme: search://)")
	searchURI := universal.NewResourceURI("search://products/widget-123")
	product := map[string]any{
		"id":    "widget-123",
		"name":  "Super Widget",
		"price": 29.99,
		"tags":  []string{"widget", "super", "new"},
	}
	
	// Index product
	if err := universal.Set(ctx, searchURI, product); err != nil {
		log.Fatalf("Failed to index product: %v", err)
	}
	fmt.Printf("   ✅ Indexed product: %v\n", product["name"])
	
	// Count indexed items
	searchPattern := universal.NewResourceURI("search://products/*")
	count, err := universal.Count[map[string]any](ctx, searchPattern)
	if err != nil {
		log.Fatalf("Failed to count products: %v", err)
	}
	fmt.Printf("   ✅ Total products indexed: %d\n", count)
	
	fmt.Println("\n🎯 Key Insights:")
	fmt.Println("   • URI scheme (db://, cache://, etc.) routes to the right provider")
	fmt.Println("   • Same functions (Get, Set, Delete, etc.) work across all providers")
	fmt.Println("   • Type safety maintained with generics")
	fmt.Println("   • No provider-specific code in your application!")
	
	// Cleanup
	fmt.Println("\n🧹 Cleaning up...")
	_ = universal.Delete[User](ctx, userURI)
	_ = universal.Delete[Session](ctx, sessionURI)
	_ = universal.Delete[Document](ctx, docURI)
	_ = universal.Delete[map[string]any](ctx, searchURI)
	
	fmt.Println("✅ Demo completed successfully!")
}

// setupMockProviders sets up mock providers for the demo
func setupMockProviders() {
	// In a real application, you would register actual providers:
	// universal.SetupDatabase(postgres.NewProvider, config)
	// universal.SetupCache(redis.NewProvider, config)
	// universal.SetupStorage(s3.NewProvider, config)
	// universal.SetupSearch(elasticsearch.NewProvider, config)
	
	fmt.Println("⚠️  Note: This demo requires provider implementations")
	fmt.Println("   In production, register real providers like:")
	fmt.Println("   - PostgreSQL for db://")
	fmt.Println("   - Redis for cache://")
	fmt.Println("   - S3 for storage://")
	fmt.Println("   - Elasticsearch for search://")
	fmt.Println()
}

// mustCreate panics if error (for demo simplicity)
func mustCreate[T any](val T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return val
}