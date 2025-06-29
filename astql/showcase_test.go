package astql

import (
	"encoding/json"
	"testing"
)

// ShowcaseTest demonstrates the full power of ASTQL
func TestASTQLShowcase(t *testing.T) {
	// Create providers
	sqlProvider := &BasicSQLProvider{}
	mongoProvider := &BasicMongoProvider{}

	t.Run("🚀 Universal Query Generation Showcase", func(t *testing.T) {
		// Same AST generates queries for different databases!
		userQuery := Select("users").
			Fields("id", "email", "name", "created_at").
			Where("tenant_id", EQ, "tenant123").
			Where("age", GE, 18).
			Where("status", IN, []string{"active", "premium"}).
			Where("deleted_at", IS_NULL, nil).
			OrderByDesc("created_at").
			Limit(20).
			Offset(40).
			MustBuild()

		// Generate SQL
		sqlQuery, sqlParams, err := sqlProvider.RenderSQL(userQuery)
		if err != nil {
			t.Fatalf("SQL generation failed: %v", err)
		}

		// Generate MongoDB
		mongoQuery, err := mongoProvider.RenderMongo(userQuery)
		if err != nil {
			t.Fatalf("MongoDB generation failed: %v", err)
		}

		t.Logf("🔥 SAME AST → DIFFERENT QUERY LANGUAGES!")
		t.Logf("")
		t.Logf("📊 SQL Query:")
		t.Logf("   %s", sqlQuery)
		t.Logf("   Params: %v", sqlParams)
		t.Logf("")
		t.Logf("🍃 MongoDB Query:")
		t.Logf("   %s", mongoQuery)
		t.Logf("")

		// Verify both queries are valid
		if len(sqlParams) != 3 {
			t.Errorf("Expected 3 SQL parameters, got %d", len(sqlParams))
		}

		var mongoDoc map[string]any
		if err := json.Unmarshal([]byte(mongoQuery), &mongoDoc); err != nil {
			t.Errorf("Invalid MongoDB JSON: %v", err)
		}

		if mongoDoc["operation"] != "find" {
			t.Errorf("Expected MongoDB find operation")
		}
	})

	t.Run("🎯 CRUD Operations Suite", func(t *testing.T) {
		// CREATE
		createAST := Insert("users").
			Values(map[string]any{
				"id":        "user789",
				"email":     "showcase@example.com", 
				"name":      "Showcase User",
				"tenant_id": "tenant123",
				"age":       25,
				"status":    "active",
			}).
			Returning("id", "created_at").
			MustBuild()

		sqlCreate, _, _ := sqlProvider.RenderSQL(createAST)
		mongoCreate, _ := mongoProvider.RenderMongo(createAST)

		// READ
		readAST := Select("users").
			Where("id", EQ, "user789").
			Where("tenant_id", EQ, "tenant123").
			MustBuild()

		sqlRead, _, _ := sqlProvider.RenderSQL(readAST)
		mongoRead, _ := mongoProvider.RenderMongo(readAST)

		// UPDATE
		updateAST := Update("users").
			Set("name", "Updated Showcase User").
			Set("last_login", "now()").
			Where("id", EQ, "user789").
			Where("tenant_id", EQ, "tenant123").
			Returning("*").
			MustBuild()

		sqlUpdate, _, _ := sqlProvider.RenderSQL(updateAST)
		mongoUpdate, _ := mongoProvider.RenderMongo(updateAST)

		// DELETE
		deleteAST := Delete("users").
			Where("id", EQ, "user789").
			Where("tenant_id", EQ, "tenant123").
			MustBuild()

		sqlDelete, _, _ := sqlProvider.RenderSQL(deleteAST)
		mongoDelete, _ := mongoProvider.RenderMongo(deleteAST)

		t.Logf("🔧 COMPLETE CRUD OPERATIONS")
		t.Logf("")
		t.Logf("📝 CREATE:")
		t.Logf("   SQL: %s", sqlCreate)
		t.Logf("   MongoDB: %s", formatJSON(mongoCreate))
		t.Logf("")
		t.Logf("📖 READ:")
		t.Logf("   SQL: %s", sqlRead)
		t.Logf("   MongoDB: %s", formatJSON(mongoRead))
		t.Logf("")
		t.Logf("✏️  UPDATE:")
		t.Logf("   SQL: %s", sqlUpdate)
		t.Logf("   MongoDB: %s", formatJSON(mongoUpdate))
		t.Logf("")
		t.Logf("🗑️  DELETE:")
		t.Logf("   SQL: %s", sqlDelete)
		t.Logf("   MongoDB: %s", formatJSON(mongoDelete))
	})

	t.Run("⚡ Advanced Query Features", func(t *testing.T) {
		// Complex query with multiple features
		complexAST := Select("orders").
			Fields("o.id", "o.total", "u.name AS customer_name", "o.created_at").
			InnerJoin("users u", "o.user_id = u.id").
			Where("o.tenant_id", EQ, "tenant123").
			Where("o.total", GE, 100.0).
			Where("o.status", IN, []string{"pending", "processing", "shipped"}).
			Where("o.created_at", GE, "2024-01-01").
			Where("u.deleted_at", IS_NULL, nil).
			OrderByDesc("o.created_at").
			OrderByAsc("o.total").
			Limit(50).
			MustBuild()

		sqlComplex, params, _ := sqlProvider.RenderSQL(complexAST)

		// Aggregation query
		countAST := Count("orders").
			Where("tenant_id", EQ, "tenant123").
			Where("status", EQ, "completed").
			Where("created_at", GE, "2024-01-01").
			MustBuild()

		sqlCount, countParams, _ := sqlProvider.RenderSQL(countAST)
		mongoCount, _ := mongoProvider.RenderMongo(countAST)

		t.Logf("🔍 ADVANCED FEATURES")
		t.Logf("")
		t.Logf("🔗 Complex JOIN Query:")
		t.Logf("   %s", sqlComplex)
		t.Logf("   Parameters: %d", len(params))
		t.Logf("")
		t.Logf("📊 Aggregation (COUNT):")
		t.Logf("   SQL: %s", sqlCount)
		t.Logf("   Params: %v", countParams)
		t.Logf("   MongoDB: %s", formatJSON(mongoCount))
	})

	t.Run("🏗️ Builder API Fluency", func(t *testing.T) {
		// Demonstrate the fluent builder API
		query := Select("products").
			Fields("id", "name", "price", "category").
			Where("category", EQ, "electronics").
			Where("price", GE, 50.0).
			Where("price", LE, 500.0).
			Where("in_stock", EQ, true).
			OrWhere("featured", EQ, true).
			OrderByAsc("price").
			OrderByDesc("rating").
			Paginate(3, 12) // Page 3, 12 items per page

		ast, err := query.Build()
		if err != nil {
			t.Fatalf("Builder failed: %v", err)
		}

		sqlQuery, params, _ := sqlProvider.RenderSQL(ast)

		t.Logf("🎨 FLUENT BUILDER API")
		t.Logf("")
		t.Logf("   Generated SQL: %s", sqlQuery)
		t.Logf("   Parameters: %v", params)
		t.Logf("   Fields: %d", len(ast.Fields))
		t.Logf("   Conditions: %d", len(ast.Conditions))
		t.Logf("   Ordering: %d", len(ast.Ordering))
		t.Logf("   Limit: %d", *ast.Limit)
		t.Logf("   Offset: %d", *ast.Offset)
	})

	t.Run("🔐 Security Features", func(t *testing.T) {
		// All queries use named parameters - injection safe!
		maliciousInput := "'; DROP TABLE users; --"
		
		secureQuery := Select("users").
			Where("email", EQ, maliciousInput).
			Where("tenant_id", EQ, "tenant123").
			MustBuild()

		sqlSecure, params, _ := sqlProvider.RenderSQL(secureQuery)

		t.Logf("🛡️  INJECTION-SAFE QUERIES")
		t.Logf("")
		t.Logf("   Malicious Input: %s", maliciousInput)
		t.Logf("   Safe SQL: %s", sqlSecure)
		t.Logf("   Parameterized: %v", params)
		t.Logf("")
		t.Logf("   ✅ No SQL injection possible!")
		t.Logf("   ✅ All values properly parameterized!")

		// Verify the malicious input is safely parameterized
		if emailParam, exists := params["email"]; exists {
			if emailParam.(string) != maliciousInput {
				t.Errorf("Parameter not properly set")
			}
		} else {
			t.Errorf("Email parameter not found")
		}

		// Verify query doesn't contain raw malicious input
		if containsUnsafeSQL(sqlSecure) {
			t.Errorf("Query contains unsafe SQL!")
		}
	})

	t.Run("📈 Performance & Optimization", func(t *testing.T) {
		// Query with hints for optimization
		optimizedAST := &QueryAST{
			Operation: OpSelect,
			Target:    "users",
			Fields: []Field{
				{Name: "id"},
				{Name: "email"},
				{Name: "name"},
			},
			Conditions: []Condition{
				{Field: "tenant_id", Operator: EQ, Value: "tenant123", ParamName: "tenant_id"},
				{Field: "status", Operator: EQ, Value: "active", ParamName: "status"},
			},
			Ordering: []Order{
				{Field: "created_at", Direction: DESC},
			},
			Hints: []Hint{
				{Provider: "sql", Type: "index", Value: "idx_tenant_status"},
				{Provider: "sql", Type: "parallel", Value: "4"},
			},
			Limit: intPtr(100),
		}

		sqlOpt, params, _ := sqlProvider.RenderSQL(optimizedAST)

		t.Logf("⚡ PERFORMANCE OPTIMIZATIONS")
		t.Logf("")
		t.Logf("   Query: %s", sqlOpt)
		t.Logf("   Parameters: %v", params)
		t.Logf("   Hints: %d optimization hints", len(optimizedAST.Hints))
		t.Logf("   - Index hint: %s", optimizedAST.Hints[0].Value)
		t.Logf("   - Parallel hint: %s", optimizedAST.Hints[1].Value)
	})
}

// Helper functions

func formatJSON(jsonStr string) string {
	var obj map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return jsonStr
	}
	
	bytes, _ := json.MarshalIndent(obj, "   ", "  ")
	return string(bytes)
}

func containsUnsafeSQL(sql string) bool {
	unsafePatterns := []string{
		"DROP TABLE",
		"DELETE FROM",
		"'; ",
		"--",
		"/*",
		"*/",
	}
	
	for _, pattern := range unsafePatterns {
		if contains(sql, pattern) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    s[:len(substr)] == substr || 
		    s[len(s)-len(substr):] == substr || 
		    containsInside(s, substr))
}

func containsInside(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func intPtr(i int) *int {
	return &i
}