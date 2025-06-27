package main

import (
	"fmt"
	"log"
	"time"
)

// Mock cache interfaces to demonstrate integration pattern
type CacheProvider interface {
	Set(key string, value []byte, ttl time.Duration) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

// Mock cereal interfaces (would import from zbz/cereal)
type CerealProvider interface {
	Marshal(data any) ([]byte, error)
	Unmarshal(data []byte, target any) error
	MarshalScoped(data any, userPermissions []string) ([]byte, error)
	UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error
	ContentType() string
	Format() string
	SupportsBinaryData() bool
	SupportsStreaming() bool
	Close() error
}

// CerealCache demonstrates how cache service integrates with cereal
type CerealCache struct {
	cacheProvider CacheProvider
	serializer    CerealProvider
}

// NewCerealCache creates a cache that uses cereal for serialization
func NewCerealCache(cacheProvider CacheProvider, serializer CerealProvider) *CerealCache {
	return &CerealCache{
		cacheProvider: cacheProvider,
		serializer:    serializer,
	}
}

// Set stores data using cereal serialization
func (cc *CerealCache) Set(key string, data any, ttl time.Duration) error {
	// Serialize using cereal
	serialized, err := cc.serializer.Marshal(data)
	if err != nil {
		return fmt.Errorf("cereal serialization failed: %w", err)
	}
	
	// Store in cache
	return cc.cacheProvider.Set(key, serialized, ttl)
}

// Get retrieves and deserializes data using cereal
func (cc *CerealCache) Get(key string, target any) error {
	// Get from cache
	data, err := cc.cacheProvider.Get(key)
	if err != nil {
		return err
	}
	
	// Deserialize using cereal
	return cc.serializer.Unmarshal(data, target)
}

// SetScoped stores data with field-level scoping
func (cc *CerealCache) SetScoped(key string, data any, userPermissions []string, ttl time.Duration) error {
	// Serialize with scoping using cereal
	serialized, err := cc.serializer.MarshalScoped(data, userPermissions)
	if err != nil {
		return fmt.Errorf("cereal scoped serialization failed: %w", err)
	}
	
	// Store in cache
	return cc.cacheProvider.Set(key, serialized, ttl)
}

// GetScoped retrieves data with scope validation
func (cc *CerealCache) GetScoped(key string, target any, userPermissions []string, operation string) error {
	// Get from cache
	data, err := cc.cacheProvider.Get(key)
	if err != nil {
		return err
	}
	
	// Deserialize with scope validation using cereal
	return cc.serializer.UnmarshalScoped(data, target, userPermissions, operation)
}

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

// Mock implementations for demonstration
type MemoryCacheProvider struct {
	data map[string][]byte
}

func NewMemoryCacheProvider() *MemoryCacheProvider {
	return &MemoryCacheProvider{
		data: make(map[string][]byte),
	}
}

func (m *MemoryCacheProvider) Set(key string, value []byte, ttl time.Duration) error {
	// Make a copy to avoid mutations
	data := make([]byte, len(value))
	copy(data, value)
	m.data[key] = data
	return nil
}

func (m *MemoryCacheProvider) Get(key string) ([]byte, error) {
	data, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}
	// Return a copy
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

func (m *MemoryCacheProvider) Delete(key string) error {
	delete(m.data, key)
	return nil
}

// Mock JSON cereal provider
type MockJSONCereal struct{}

func (m *MockJSONCereal) Marshal(data any) ([]byte, error) {
	// This would delegate to the actual cereal JSON provider
	return []byte(fmt.Sprintf(`{"mock": "serialized %T"}`, data)), nil
}

func (m *MockJSONCereal) Unmarshal(data []byte, target any) error {
	// This would delegate to the actual cereal JSON provider
	fmt.Printf("Mock deserializing %s into %T\n", string(data), target)
	return nil
}

func (m *MockJSONCereal) MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	// This would delegate to the actual cereal JSON provider with scoping
	return []byte(fmt.Sprintf(`{"mock": "scoped serialized %T", "permissions": %v}`, data, userPermissions)), nil
}

func (m *MockJSONCereal) UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	// This would delegate to the actual cereal JSON provider with scope validation
	fmt.Printf("Mock scoped deserializing %s into %T (perms: %v, op: %s)\n", string(data), target, userPermissions, operation)
	return nil
}

func (m *MockJSONCereal) ContentType() string { return "application/json" }
func (m *MockJSONCereal) Format() string { return "json" }
func (m *MockJSONCereal) SupportsBinaryData() bool { return false }
func (m *MockJSONCereal) SupportsStreaming() bool { return false }
func (m *MockJSONCereal) Close() error { return nil }

func main() {
	fmt.Println("🥣 Cereal-Cache Integration Demo")
	fmt.Println("================================")
	
	// Set up cache with cereal integration
	cacheProvider := NewMemoryCacheProvider()
	serializer := &MockJSONCereal{}
	cache := NewCerealCache(cacheProvider, serializer)
	
	// Test basic operations
	fmt.Println("\n📦 Basic Cache Operations:")
	
	user := User{
		ID:       1,
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "secret123",
		Internal: "internal-data",
	}
	
	// Store user
	err := cache.Set("user:1", user, time.Hour)
	if err != nil {
		log.Printf("Failed to set user: %v", err)
	} else {
		fmt.Println("✅ User stored successfully")
	}
	
	// Retrieve user
	var retrievedUser User
	err = cache.Get("user:1", &retrievedUser)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
	} else {
		fmt.Println("✅ User retrieved successfully")
	}
	
	// Test scoped operations
	fmt.Println("\n🔒 Scoped Cache Operations:")
	
	// Store with user permissions (filters sensitive fields)
	userPermissions := []string{"user"}
	err = cache.SetScoped("user:1:filtered", user, userPermissions, time.Hour)
	if err != nil {
		log.Printf("Failed to set scoped user: %v", err)
	} else {
		fmt.Println("✅ Scoped user stored successfully")
	}
	
	// Try to retrieve with admin permissions
	adminPermissions := []string{"admin"}
	err = cache.GetScoped("user:1:filtered", &retrievedUser, adminPermissions, "read")
	if err != nil {
		log.Printf("Failed to get scoped user: %v", err)
	} else {
		fmt.Println("✅ Scoped user retrieved with admin permissions")
	}
	
	// Test with product (no scoping)
	fmt.Println("\n📱 Product Operations (No Scoping):")
	
	product := Product{
		ID:    100,
		Name:  "Laptop",
		Price: 999.99,
	}
	
	err = cache.Set("product:100", product, time.Hour)
	if err != nil {
		log.Printf("Failed to set product: %v", err)
	} else {
		fmt.Println("✅ Product stored successfully")
	}
	
	var retrievedProduct Product
	err = cache.Get("product:100", &retrievedProduct)
	if err != nil {
		log.Printf("Failed to get product: %v", err)
	} else {
		fmt.Println("✅ Product retrieved successfully")
	}
	
	fmt.Println("\n🎉 Integration Demo Complete!")
	fmt.Println("=============================")
	fmt.Println("Key Benefits Demonstrated:")
	fmt.Println("✅ Unified serialization interface")
	fmt.Println("✅ Optional field-level scoping")
	fmt.Println("✅ Type-safe operations")
	fmt.Println("✅ Backward compatibility")
	fmt.Println("✅ Performance optimization potential")
}