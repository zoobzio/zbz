package core

import (
	"encoding/json"
	"testing"
)

// Test types for demonstration
type User struct {
	ID    int    `json:"id" zbz:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Product struct {
	SKU   string  `json:"sku" zbz:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func TestZbzModelBasics(t *testing.T) {
	user := User{
		ID:    123,
		Name:  "John Doe",
		Email: "john@example.com",
	}
	
	// Wrap user in ZbzModel
	model := Wrap(user)
	
	// Test basic accessors
	if model.ID() != "123" {
		t.Errorf("Expected ID '123', got '%s'", model.ID())
	}
	
	if model.Data().Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", model.Data().Name)
	}
	
	if model.Version() != 1 {
		t.Errorf("Expected version 1, got %d", model.Version())
	}
	
	if !model.IsNew() {
		t.Error("Expected model to be new (no created timestamp yet)")
	}
	
	if model.IsDeleted() {
		t.Error("Expected model to not be deleted")
	}
}

func TestZbzModelLifecycle(t *testing.T) {
	user := User{ID: 456, Name: "Jane Doe", Email: "jane@example.com"}
	model := Wrap(user)
	
	// Test soft delete
	if model.IsDeleted() {
		t.Error("Model should not be deleted initially")
	}
	
	model.SoftDelete()
	if !model.IsDeleted() {
		t.Error("Model should be deleted after SoftDelete()")
	}
	
	if model.DeletedAt() == nil {
		t.Error("DeletedAt should be set after soft delete")
	}
	
	// Test restore
	model.Restore()
	if model.IsDeleted() {
		t.Error("Model should not be deleted after Restore()")
	}
	
	if model.DeletedAt() != nil {
		t.Error("DeletedAt should be nil after restore")
	}
}

func TestZbzModelMetadata(t *testing.T) {
	user := User{ID: 789, Name: "Bob", Email: "bob@example.com"}
	model := Wrap(user)
	
	// Test metadata
	model.SetMetadata("source", "api")
	model.SetMetadata("tenant", "acme-corp")
	
	metadata := model.Metadata()
	if metadata["source"] != "api" {
		t.Errorf("Expected metadata source 'api', got '%v'", metadata["source"])
	}
	
	if metadata["tenant"] != "acme-corp" {
		t.Errorf("Expected metadata tenant 'acme-corp', got '%v'", metadata["tenant"])
	}
}

func TestZbzModelJSON(t *testing.T) {
	user := User{ID: 321, Name: "Alice", Email: "alice@example.com"}
	model := Wrap(user)
	model.SetMetadata("test", "value")
	
	// Test JSON marshaling
	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("Failed to marshal model: %v", err)
	}
	
	// Check that the JSON contains expected fields
	var jsonData map[string]any
	if err := json.Unmarshal(data, &jsonData); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	
	if jsonData["id"] != "321" {
		t.Errorf("Expected JSON id '321', got '%v'", jsonData["id"])
	}
	
	if jsonData["version"] != float64(1) {
		t.Errorf("Expected JSON version 1, got %v", jsonData["version"])
	}
	
	userData, ok := jsonData["data"].(map[string]any)
	if !ok {
		t.Fatal("Expected data field to be an object")
	}
	
	if userData["name"] != "Alice" {
		t.Errorf("Expected data.name 'Alice', got '%v'", userData["name"])
	}
	
	// Test JSON unmarshaling
	var restored ZbzModel[User]
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal model: %v", err)
	}
	
	if restored.ID() != "321" {
		t.Errorf("Expected restored ID '321', got '%s'", restored.ID())
	}
	
	if restored.Data().Name != "Alice" {
		t.Errorf("Expected restored name 'Alice', got '%s'", restored.Data().Name)
	}
}

func TestCoreBasics(t *testing.T) {
	// Create a core for User type
	userCore := NewCore[User]()
	
	if userCore.TypeName() != "core.User" {
		t.Errorf("Expected type name 'core.User', got '%s'", userCore.TypeName())
	}
	
	if userCore.Type().Name() != "User" {
		t.Errorf("Expected type name 'User', got '%s'", userCore.Type().Name())
	}
}

func TestCoreHooks(t *testing.T) {
	userCore := NewCore[User]()
	
	// Test hook registration
	hookID := userCore.OnAfterCreate(func(user ZbzModel[User]) error {
		return nil
	})
	
	if hookID == "" {
		t.Error("Hook ID should not be empty")
	}
	
	// We can't easily test the actual hook execution without setting up providers
	// but we can test hook registration and removal
	
	if err := userCore.RemoveHook(hookID); err != nil {
		t.Errorf("Failed to remove hook: %v", err)
	}
	
	// Try to remove the same hook again (should fail)
	if err := userCore.RemoveHook(hookID); err == nil {
		t.Error("Expected error when removing non-existent hook")
	}
}

func TestResourceChain(t *testing.T) {
	userCore := NewCore[User]()
	
	// Create a resource chain
	chain := ResourceChain{
		Name:    "user-profile",
		Primary: NewResourceURI("db://users/{id}"),
		Fallbacks: []ResourceURI{
			NewResourceURI("cache://users/{id}"),
		},
		Strategy: ReadThroughCacheFirst,
		TTL:      "15m",
	}
	
	// Register the chain
	if err := userCore.RegisterChain(chain); err != nil {
		t.Fatalf("Failed to register chain: %v", err)
	}
	
	// Retrieve the chain
	retrieved, err := userCore.GetRegisteredChain("user-profile")
	if err != nil {
		t.Fatalf("Failed to retrieve chain: %v", err)
	}
	
	if retrieved.Name != "user-profile" {
		t.Errorf("Expected chain name 'user-profile', got '%s'", retrieved.Name)
	}
	
	if retrieved.Strategy != ReadThroughCacheFirst {
		t.Errorf("Expected strategy ReadThroughCacheFirst, got %v", retrieved.Strategy)
	}
}

func TestChainValidation(t *testing.T) {
	userCore := NewCore[User]()
	
	// Test invalid chain (empty name)
	invalidChain := ResourceChain{
		Name:    "",
		Primary: NewResourceURI("db://users/{id}"),
	}
	
	if err := userCore.RegisterChain(invalidChain); err == nil {
		t.Error("Expected error for chain with empty name")
	}
	
	// Test invalid chain (empty primary)
	invalidChain2 := ResourceChain{
		Name:    "test",
		Primary: ResourceURI{},
	}
	
	if err := userCore.RegisterChain(invalidChain2); err == nil {
		t.Error("Expected error for chain with empty primary URI")
	}
}

func TestPackageLevelAPI(t *testing.T) {
	// Test core registration and retrieval
	userCore := NewCore[User]()
	if err := RegisterCore(userCore); err != nil {
		t.Fatalf("Failed to register core: %v", err)
	}
	
	retrieved, err := GetCore[User]()
	if err != nil {
		t.Fatalf("Failed to get core: %v", err)
	}
	
	if retrieved.TypeName() != userCore.TypeName() {
		t.Errorf("Retrieved core type name doesn't match registered core")
	}
	
	// Test auto-creation of core
	productCore, err := GetCore[Product]()
	if err != nil {
		t.Fatalf("Failed to get/create product core: %v", err)
	}
	
	if productCore.TypeName() != "core.Product" {
		t.Errorf("Expected product core type name 'core.Product', got '%s'", productCore.TypeName())
	}
}

func TestGlobalChainRegistry(t *testing.T) {
	// Test global chain registration
	chain := ResourceChain{
		Name:    "global-user-chain",
		Primary: NewResourceURI("db://users/{id}"),
		Fallbacks: []ResourceURI{
			NewResourceURI("cache://users/{id}"),
		},
		Strategy: ReadThroughCacheFirst,
		TTL:      "10m",
	}
	
	if err := RegisterChain(chain); err != nil {
		t.Fatalf("Failed to register global chain: %v", err)
	}
	
	retrieved, err := GetChainDefinition("global-user-chain")
	if err != nil {
		t.Fatalf("Failed to retrieve global chain: %v", err)
	}
	
	if retrieved.Name != "global-user-chain" {
		t.Errorf("Expected chain name 'global-user-chain', got '%s'", retrieved.Name)
	}
	
	// Test applying global chain to core
	if err := ApplyChainToCore[User]("global-user-chain"); err != nil {
		t.Fatalf("Failed to apply chain to core: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	// Ensure we have some cores and chains registered
	GetCore[User]()  // Auto-creates if not exists
	GetCore[Product]() // Auto-creates if not exists
	
	RegisterChain(ResourceChain{
		Name:    "health-test-chain",
		Primary: NewResourceURI("db://test/{id}"),
		Strategy: ReadThroughCacheFirst,
	})
	
	health := HealthCheck()
	
	registeredCores, ok := health["registered_cores"].(int)
	if !ok || registeredCores < 2 {
		t.Errorf("Expected at least 2 registered cores, got %v", health["registered_cores"])
	}
	
	registeredChains, ok := health["registered_chains"].(int)
	if !ok || registeredChains < 1 {
		t.Errorf("Expected at least 1 registered chain, got %v", health["registered_chains"])
	}
	
	coreTypes, ok := health["core_types"].([]string)
	if !ok || len(coreTypes) < 2 {
		t.Errorf("Expected core types list, got %v", health["core_types"])
	}
}

func TestHookChains(t *testing.T) {
	// Test hook chains for complex hook scenarios
	chain := NewHookChain[User]()
	
	executed := make([]string, 0)
	
	chain.Add(func(user ZbzModel[User]) error {
		executed = append(executed, "first")
		return nil
	}).Add(func(user ZbzModel[User]) error {
		executed = append(executed, "second")
		return nil
	})
	
	user := Wrap(User{ID: 999, Name: "Test", Email: "test@example.com"})
	
	if err := chain.Execute(user); err != nil {
		t.Fatalf("Hook chain execution failed: %v", err)
	}
	
	if len(executed) != 2 {
		t.Errorf("Expected 2 hooks to execute, got %d", len(executed))
	}
	
	if executed[0] != "first" || executed[1] != "second" {
		t.Errorf("Hooks executed in wrong order: %v", executed)
	}
}

// Benchmark tests

func BenchmarkZbzModelCreation(b *testing.B) {
	user := User{ID: 123, Name: "John", Email: "john@example.com"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Wrap(user)
	}
}

func BenchmarkZbzModelJSONMarshal(b *testing.B) {
	user := User{ID: 123, Name: "John", Email: "john@example.com"}
	model := Wrap(user)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(model)
	}
}

func BenchmarkCoreCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewCore[User]()
	}
}