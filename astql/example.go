package astql

import (
	"context"
	"time"
)

// Example demonstrating ASTQL usage

// ExampleUser shows how struct tags affect query generation
type ExampleUser struct {
	ID        string `json:"id" db:"user_id" astql:"primary"`
	Email     string `json:"email" astql:"index:unique"`
	Name      string `json:"name" astql:"index:btree"`
	TenantID  string `json:"tenant_id" astql:"security:tenant"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
	DeletedAt *string `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ExampleUsage demonstrates how to use the ASTQL builder
func ExampleUsage() {
	// 1. Build a query using the fluent API
	query := Select("users").
		Fields("id", "email", "name").
		Where("tenant_id", EQ, "tenant123").
		Where("deleted_at", IS_NULL, nil).
		OrderByDesc("created_at").
		Limit(10)

	ast, err := query.Build()
	if err != nil {
		panic(err)
	}

	// 2. The AST can now be rendered to any query language by providers
	// SQL: SELECT id, email, name FROM users WHERE tenant_id = :tenant_id AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 10
	// MongoDB: { tenant_id: "tenant123", deleted_at: null }, { sort: { created_at: -1 }, limit: 10 }
	// ElasticSearch: { "query": { "bool": { "must": [...] } }, "sort": [...], "size": 10 }
	
	_ = ast // Use the AST
}

// ExampleEventDriven shows how ASTQL works in an event-driven way
func ExampleEventDriven() {
	// This would be called when a core registers a type
	// GenerateFromType[ExampleUser]()
	
	// The above call would emit events that listeners can react to:
	// 1. UniversalASTGenerated - for each CRUD operation
	// 2. Listeners write queries to files, cache them, etc.
	// 3. Flux can watch the query files for live reloading
	// 4. All without ASTQL knowing about file I/O, caching, or flux!
}

// ExampleZBZPatterns demonstrates ZBZ architectural patterns in ASTQL
func ExampleZBZPatterns() {
	// 1. UNIVERSAL PROVIDER CONFIG - Same struct for ALL providers!
	sqlConfig := DefaultProviderConfig()
	sqlConfig.ProviderType = "sql"
	sqlConfig.Host = "localhost"
	sqlConfig.Database = "myapp"
	sqlConfig.Settings = map[string]any{
		"sslmode": "require",
		"pool_size": 20,
	}
	
	mongoConfig := DefaultProviderConfig()
	mongoConfig.ProviderType = "mongo"
	mongoConfig.Host = "localhost" 
	mongoConfig.Database = "myapp"
	mongoConfig.Settings = map[string]any{
		"replica_set": "rs0",
		"read_preference": "secondary",
	}
	
	// 2. SINGLETON SERVICE PATTERN - Package-level functions
	RegisterProvider("sql", sqlConfig)
	RegisterProvider("mongo", mongoConfig)
	
	// 3. TYPE-SAFE GENERIC EXECUTION
	ctx := context.Background()
	user, err := Execute[ExampleUser](ctx, OperationURI{
		Scheme:    "sql",
		Resource:  "users", 
		Operation: "get",
	}, map[string]any{"id": "123"})
	
	_ = user
	_ = err
	
	// 4. EVENT-DRIVEN ARCHITECTURE - Everything emits events
	// RegisterProvider emitted: ProviderRegistered 
	// Execute will emit: QueryExecuting, QueryExecuted
	// GenerateFromType emits: UniversalASTGenerated
	
	// 5. CATALOG INTEGRATION - Auto-generation from types
	GenerateFromType[ExampleUser]() // Uses catalog.Select[T]()
}

// ExampleUniversalConfig shows the power of universal provider configuration
func ExampleUniversalConfig() {
	// THE SAME CONFIG STRUCT WORKS FOR ANY PROVIDER!
	
	// SQL (PostgreSQL)
	postgresConfig := ProviderConfig{
		ProviderType: "sql",
		Host: "postgres.example.com",
		Port: 5432,
		Database: "production",
		Username: "app_user",
		MaxConnections: 50,
		Timeout: 30 * time.Second,
		Settings: map[string]any{
			"sslmode": "require",
			"application_name": "myapp",
		},
	}
	
	// MongoDB
	mongoConfig := ProviderConfig{
		ProviderType: "mongo", 
		Host: "mongo.example.com",
		Port: 27017,
		Database: "production",
		Username: "app_user",
		MaxConnections: 20,
		Timeout: 15 * time.Second,
		Settings: map[string]any{
			"authSource": "admin",
			"replica_set": "rs0",
		},
	}
	
	// ElasticSearch
	elasticConfig := ProviderConfig{
		ProviderType: "elastic",
		Host: "elastic.example.com", 
		Port: 9200,
		Database: "production-index", // index name
		MaxConnections: 10,
		Settings: map[string]any{
			"sniff": true,
			"compression": true,
		},
	}
	
	// Redis
	redisConfig := ProviderConfig{
		ProviderType: "redis",
		Host: "redis.example.com",
		Port: 6379,
		Database: "0", // redis db number
		Settings: map[string]any{
			"pool_size": 20,
			"max_retries": 3,
		},
	}
	
	// Register them all with the SAME API!
	RegisterProvider("sql", postgresConfig)
	RegisterProvider("mongo", mongoConfig) 
	RegisterProvider("elastic", elasticConfig)
	RegisterProvider("redis", redisConfig)
	
	// Now you can execute queries against ANY provider with the same API!
}