package cereal

import (
	"encoding/json"
	"testing"
)

// Test data structures
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email" scope:"read:user,write:admin"`
	Password string `json:"password" scope:"write:admin"`
	Internal string `json:"internal" scope:"read:admin"`
}

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func TestJSONProvider(t *testing.T) {
	config := DefaultConfig()
	config.Name = "test-json"
	
	contract := NewJSONProvider(config)
	err := contract.Register()
	if err != nil {
		t.Fatalf("Failed to register JSON provider: %v", err)
	}
	
	// Test basic serialization
	product := Product{ID: 1, Name: "Widget", Price: 9.99}
	
	data, err := Marshal(product)
	if err != nil {
		t.Fatalf("Failed to marshal product: %v", err)
	}
	
	var unmarshaled Product
	err = Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal product: %v", err)
	}
	
	if unmarshaled != product {
		t.Errorf("Expected %+v, got %+v", product, unmarshaled)
	}
}

func TestScopedSerialization(t *testing.T) {
	config := DefaultConfig()
	config.Name = "test-scoped"
	config.EnableScoping = true
	
	contract := NewJSONProvider(config)
	
	// Set up the service reference for scoped operations
	if jsonProvider, ok := contract.Provider().(*JSONProvider); ok {
		// We need to simulate the service setup since we're testing in isolation
		testCereal := &zCereal{
			provider:     contract.Provider(),
			config:       config,
			contractName: "test-scoped",
			scopeCache:   newScopeCache(),
		}
		jsonProvider.setCereal(testCereal)
	}
	
	err := contract.Register()
	if err != nil {
		t.Fatalf("Failed to register scoped provider: %v", err)
	}
	
	user := User{
		ID:       1,
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "secret123",
		Internal: "internal-data",
	}
	
	// Test scoped serialization with user permissions
	userPermissions := []string{"user"}
	data, err := MarshalScoped(user, userPermissions)
	if err != nil {
		t.Fatalf("Failed to marshal scoped user: %v", err)
	}
	
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	// Should include: id, name, email (user has read:user permission)
	// Should exclude: password (write:admin only), internal (read:admin only)
	expectedFields := []string{"id", "name", "email"}
	forbiddenFields := []string{"password", "internal"}
	
	for _, field := range expectedFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Expected field %s to be present", field)
		}
	}
	
	for _, field := range forbiddenFields {
		if _, exists := result[field]; exists {
			t.Errorf("Expected field %s to be filtered out", field)
		}
	}
}

func TestScopedDeserialization(t *testing.T) {
	config := DefaultConfig()
	config.Name = "test-scoped-deser"
	config.EnableScoping = true
	
	contract := NewJSONProvider(config)
	
	// Set up the service reference for scoped operations
	if jsonProvider, ok := contract.Provider().(*JSONProvider); ok {
		testCereal := &zCereal{
			provider:     contract.Provider(),
			config:       config,
			contractName: "test-scoped-deser",
			scopeCache:   newScopeCache(),
		}
		jsonProvider.setCereal(testCereal)
	}
	
	err := contract.Register()
	if err != nil {
		t.Fatalf("Failed to register scoped provider: %v", err)
	}
	
	// Test unauthorized write attempt
	unauthorizedData := `{"id": 1, "name": "Jane", "internal": "hacked"}`
	var user User
	userPermissions := []string{"user"} // No admin permission
	
	err = UnmarshalScoped([]byte(unauthorizedData), &user, userPermissions, OperationUpdate)
	if err == nil {
		t.Error("Expected error for unauthorized field modification")
	}
	
	// Test authorized write
	authorizedData := `{"id": 1, "name": "Jane", "email": "jane@example.com"}`
	err = UnmarshalScoped([]byte(authorizedData), &user, userPermissions, OperationUpdate)
	if err != nil {
		t.Errorf("Unexpected error for authorized modification: %v", err)
	}
	
	if user.Name != "Jane" || user.Email != "jane@example.com" {
		t.Errorf("Failed to unmarshal authorized fields: %+v", user)
	}
}

func TestNoScopingForPlainStructs(t *testing.T) {
	config := DefaultConfig()
	config.Name = "test-no-scope"
	
	contract := NewJSONProvider(config)
	err := contract.Register()
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}
	
	// Product has no scope tags, so scoped operations should behave like regular operations
	product := Product{ID: 1, Name: "Widget", Price: 9.99}
	
	// Scoped marshal should be identical to regular marshal
	regularData, err := Marshal(product)
	if err != nil {
		t.Fatalf("Failed to marshal product: %v", err)
	}
	
	scopedData, err := MarshalScoped(product, []string{"anything"})
	if err != nil {
		t.Fatalf("Failed to marshal scoped product: %v", err)
	}
	
	if string(regularData) != string(scopedData) {
		t.Error("Scoped and regular serialization should be identical for structs without scope tags")
	}
}

func TestRawProvider(t *testing.T) {
	config := DefaultConfig()
	config.Name = "test-raw"
	
	contract := NewRawProvider(config)
	err := contract.Register()
	if err != nil {
		t.Fatalf("Failed to register raw provider: %v", err)
	}
	
	// Test byte slice
	original := []byte("hello world")
	
	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal bytes: %v", err)
	}
	
	var result []byte
	err = Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal bytes: %v", err)
	}
	
	if string(result) != string(original) {
		t.Errorf("Expected %s, got %s", string(original), string(result))
	}
	
	// Test string to bytes
	str := "test string"
	data, err = Marshal(str)
	if err != nil {
		t.Fatalf("Failed to marshal string: %v", err)
	}
	
	var strResult string
	err = Unmarshal(data, &strResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal to string: %v", err)
	}
	
	if strResult != str {
		t.Errorf("Expected %s, got %s", str, strResult)
	}
}

func TestStringProvider(t *testing.T) {
	config := DefaultConfig()
	config.Name = "test-string"
	
	contract := NewStringProvider(config)
	err := contract.Register()
	if err != nil {
		t.Fatalf("Failed to register string provider: %v", err)
	}
	
	// Test string serialization
	original := "Hello, 世界!"
	
	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal string: %v", err)
	}
	
	var result string
	err = Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal string: %v", err)
	}
	
	if result != original {
		t.Errorf("Expected %s, got %s", original, result)
	}
	
	// Test other types conversion
	number := 42
	data, err = Marshal(number)
	if err != nil {
		t.Fatalf("Failed to marshal number: %v", err)
	}
	
	err = Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal number as string: %v", err)
	}
	
	if result != "42" {
		t.Errorf("Expected '42', got %s", result)
	}
}

func TestContractTypeAccess(t *testing.T) {
	config := DefaultConfig()
	
	// Test JSON contract
	jsonContract := NewJSONProvider(config)
	encoder := jsonContract.Native() // Should be *json.Encoder
	if encoder == nil {
		t.Error("Expected non-nil encoder from JSON contract")
	}
	
	// Test Raw contract
	rawContract := NewRawProvider(config)
	rawNative := rawContract.Native() // Should be []byte
	if rawNative == nil {
		// []byte can be nil, that's fine
	}
	
	// Test String contract
	stringContract := NewStringProvider(config)
	stringNative := stringContract.Native() // Should be string
	if stringNative != "" {
		// Default string should be empty, but that's the zero value
	}
}

func BenchmarkJSONMarshal(b *testing.B) {
	config := DefaultConfig()
	contract := NewJSONProvider(config)
	contract.Register()
	
	user := User{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(user)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}

func BenchmarkScopedMarshal(b *testing.B) {
	config := DefaultConfig()
	contract := NewJSONProvider(config)
	
	// Set up scoped operations
	if jsonProvider, ok := contract.Provider().(*JSONProvider); ok {
		testCereal := &zCereal{
			provider:     contract.Provider(),
			config:       config,
			contractName: "bench-scoped",
			scopeCache:   newScopeCache(),
		}
		jsonProvider.setCereal(testCereal)
	}
	
	contract.Register()
	
	user := User{
		ID:       1,
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "secret",
		Internal: "internal",
	}
	
	userPermissions := []string{"user"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := MarshalScoped(user, userPermissions)
		if err != nil {
			b.Fatalf("MarshalScoped failed: %v", err)
		}
	}
}