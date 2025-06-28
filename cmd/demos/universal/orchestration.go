package universal

import (
	"context"
	"fmt"
	"log"

	"zbz/universal"
)

// OrchestrationDemo demonstrates how providers orchestrate universal and provider-specific operations
func OrchestrationDemo() {
	fmt.Println("🎭 Universal Data Access - Provider Orchestration Demo")
	fmt.Println("=====================================================")
	fmt.Println()
	
	ctx := context.Background()
	
	// Setup enhanced mock providers with provider-specific features
	setupEnhancedMockProviders()
	
	type User struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	
	fmt.Println("🎯 Demo: Universal API + Provider-Specific Features")
	fmt.Println("===================================================")
	
	// Demo 1: Universal operations work identically
	fmt.Println("\n📋 Demo: Universal Operations (Same Interface)")
	fmt.Println("==============================================")
	
	user := User{ID: 1, Name: "Alice Smith", Email: "alice@example.com"}
	dbURI := universal.NewResourceURI("db://users/1")
	cacheURI := universal.NewResourceURI("cache://users/1")
	
	// Store in database
	fmt.Println("📤 Storing user in database...")
	if err := universal.Set(ctx, dbURI, user); err != nil {
		log.Fatalf("Failed to store in database: %v", err)
	}
	fmt.Printf("   ✅ Stored in database: %+v\n", user)
	
	// Store in cache
	fmt.Println("📤 Storing user in cache...")
	if err := universal.Set(ctx, cacheURI, user); err != nil {
		log.Fatalf("Failed to store in cache: %v", err)
	}
	fmt.Printf("   ✅ Stored in cache: %+v\n", user)
	
	// Demo 2: Complex operations via Operation URIs
	fmt.Println("\n🔧 Demo: Complex Operations (via Operation URIs)")
	fmt.Println("================================================")
	
	// Database-specific query
	dbQueryURI := universal.NewOperationURI("db://queries/find-by-email")
	queryParams := map[string]any{"email": "alice@example.com"}
	
	fmt.Println("🔍 Executing database query...")
	result, err := universal.Execute[User](ctx, dbQueryURI, queryParams)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("   ✅ Query result: %+v\n", result)
	
	// Cache-specific operation  
	cacheOpURI := universal.NewOperationURI("cache://operations/ttl-update")
	ttlParams := map[string]any{"key": "users/1", "ttl": 3600}
	
	fmt.Println("⏱️  Updating cache TTL...")
	_, err = universal.Execute[User](ctx, cacheOpURI, ttlParams)
	if err != nil {
		log.Fatalf("TTL update failed: %v", err)
	}
	fmt.Println("   ✅ TTL updated to 1 hour")
	
	// Demo 3: Cross-provider data flow
	fmt.Println("\n🔄 Demo: Cross-Provider Data Flow")
	fmt.Println("==================================")
	
	// Read from database
	fmt.Println("📥 Reading from database...")
	dbUser, err := universal.Get[User](ctx, dbURI)
	if err != nil {
		log.Fatalf("Failed to read from database: %v", err)
	}
	
	// Write to cache (read-through cache pattern)
	fmt.Println("💾 Caching database result...")
	if err := universal.Set(ctx, cacheURI, dbUser); err != nil {
		log.Fatalf("Failed to cache: %v", err)
	}
	fmt.Println("   ✅ Cached database result")
	
	// Write to search index
	searchURI := universal.NewResourceURI("search://users/1")
	fmt.Println("🔍 Indexing for search...")
	if err := universal.Set(ctx, searchURI, dbUser); err != nil {
		log.Fatalf("Failed to index: %v", err)
	}
	fmt.Println("   ✅ Indexed in search provider")
	
	// Demo 4: Batch operations across providers
	fmt.Println("\n📦 Demo: Batch Operations")
	fmt.Println("=========================")
	
	// Create multiple users
	users := []User{
		{ID: 2, Name: "Bob Jones", Email: "bob@example.com"},
		{ID: 3, Name: "Carol White", Email: "carol@example.com"},
		{ID: 4, Name: "David Brown", Email: "david@example.com"},
	}
	
	fmt.Println("📤 Batch storing users...")
	for _, u := range users {
		dbURI := universal.NewResourceURI(fmt.Sprintf("db://users/%d", u.ID))
		cacheURI := universal.NewResourceURI(fmt.Sprintf("cache://users/%d", u.ID))
		
		// Store in both database and cache
		if err := universal.Set(ctx, dbURI, u); err != nil {
			log.Printf("Failed to store user %d in database: %v", u.ID, err)
			continue
		}
		if err := universal.Set(ctx, cacheURI, u); err != nil {
			log.Printf("Failed to cache user %d: %v", u.ID, err)
		}
	}
	fmt.Printf("   ✅ Stored %d users across providers\n", len(users))
	
	// Count in each provider
	dbPattern := universal.NewResourceURI("db://users/*")
	dbCount, _ := universal.Count[User](ctx, dbPattern)
	
	cachePattern := universal.NewResourceURI("cache://users/*")
	cacheCount, _ := universal.Count[User](ctx, cachePattern)
	
	fmt.Printf("   📊 Database count: %d\n", dbCount)
	fmt.Printf("   📊 Cache count: %d\n", cacheCount)
	
	fmt.Println("\n🎯 Key Insights:")
	fmt.Println("   • Providers handle both universal and specific operations")
	fmt.Println("   • Operation URIs enable provider-specific features")
	fmt.Println("   • Data flows seamlessly between providers")
	fmt.Println("   • Each provider optimizes for its use case")
	
	// Cleanup
	fmt.Println("\n🧹 Cleaning up...")
	for i := 1; i <= 4; i++ {
		dbURI := universal.NewResourceURI(fmt.Sprintf("db://users/%d", i))
		cacheURI := universal.NewResourceURI(fmt.Sprintf("cache://users/%d", i))
		searchURI := universal.NewResourceURI(fmt.Sprintf("search://users/%d", i))
		
		_ = universal.Delete[User](ctx, dbURI)
		_ = universal.Delete[User](ctx, cacheURI)
		_ = universal.Delete[User](ctx, searchURI)
	}
	
	fmt.Println("✅ Orchestration demo completed!")
}

// setupEnhancedMockProviders sets up providers with enhanced features
func setupEnhancedMockProviders() {
	fmt.Println("⚠️  Note: This demo requires provider implementations")
	fmt.Println("   Providers should support:")
	fmt.Println("   - Universal operations (Get, Set, Delete, etc.)")
	fmt.Println("   - Provider-specific operations via Execute()")
	fmt.Println("   - Hook emission for observability")
	fmt.Println()
}

// mustParseURI panics if URI parsing fails (for demo simplicity)
func mustParseURI(uri string) universal.ResourceURI {
	parsed, err := universal.ParseResourceURI(uri)
	if err != nil {
		log.Fatalf("Failed to parse URI %s: %v", uri, err)
	}
	return parsed
}