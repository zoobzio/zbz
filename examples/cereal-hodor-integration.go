package main

import (
	"fmt"
	"time"
)

// This example demonstrates how depot integrates with cereal for object metadata serialization

// Mock interfaces for demonstration
type DepotProvider interface {
	Set(key string, data []byte, ttl interface{}) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	Exists(key string) (bool, error)
	List(prefix string) ([]string, error)
	GetProvider() string
}

type CerealProvider interface {
	Marshal(data any) ([]byte, error)
	Unmarshal(data []byte, target any) error
	MarshalScoped(data any, userPermissions []string) ([]byte, error)
	UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error
	ContentType() string
	Format() string
}

// CerealDepot demonstrates depot integration with cereal for metadata serialization
type CerealDepot struct {
	depotProvider  DepotProvider
	cerealProvider CerealProvider
}

// NewCerealDepot creates a new depot service with cereal integration
func NewCerealDepot(depot DepotProvider, cereal CerealProvider) *CerealDepot {
	return &CerealDepot{
		depotProvider:  depot,
		cerealProvider: cereal,
	}
}

// SetWithMetadata stores data with structured metadata using cereal serialization
func (ch *CerealDepot) SetWithMetadata(key string, data []byte, metadata interface{}) error {
	// Serialize metadata using cereal
	metadataBytes, err := ch.cerealProvider.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}
	
	// Create combined payload with data and metadata
	payload := CombinedPayload{
		Data:     data,
		Metadata: metadataBytes,
	}
	
	// Serialize the combined payload
	payloadBytes, err := ch.cerealProvider.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload: %w", err)
	}
	
	// Store in depot
	return ch.depotProvider.Set(key, payloadBytes, 0)
}

// GetWithMetadata retrieves data and deserializes metadata using cereal
func (ch *CerealDepot) GetWithMetadata(key string, metadata interface{}) ([]byte, error) {
	// Get from depot
	payloadBytes, err := ch.depotProvider.Get(key)
	if err != nil {
		return nil, err
	}
	
	// Deserialize combined payload
	var payload CombinedPayload
	err = ch.cerealProvider.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize payload: %w", err)
	}
	
	// Deserialize metadata if provided
	if metadata != nil {
		err = ch.cerealProvider.Unmarshal(payload.Metadata, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
		}
	}
	
	return payload.Data, nil
}

// SetWithScopedMetadata stores data with scoped metadata (filters sensitive fields)
func (ch *CerealDepot) SetWithScopedMetadata(key string, data []byte, metadata interface{}, userPermissions []string) error {
	// Serialize metadata with scoping using cereal
	metadataBytes, err := ch.cerealProvider.MarshalScoped(metadata, userPermissions)
	if err != nil {
		return fmt.Errorf("failed to serialize scoped metadata: %w", err)
	}
	
	// Create combined payload
	payload := CombinedPayload{
		Data:     data,
		Metadata: metadataBytes,
	}
	
	// Serialize the combined payload
	payloadBytes, err := ch.cerealProvider.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload: %w", err)
	}
	
	// Store in depot
	return ch.depotProvider.Set(key, payloadBytes, 0)
}

// GetWithScopedMetadata retrieves data with scoped metadata deserialization
func (ch *CerealDepot) GetWithScopedMetadata(key string, metadata interface{}, userPermissions []string) ([]byte, error) {
	// Get from depot
	payloadBytes, err := ch.depotProvider.Get(key)
	if err != nil {
		return nil, err
	}
	
	// Deserialize combined payload
	var payload CombinedPayload
	err = ch.cerealProvider.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize payload: %w", err)
	}
	
	// Deserialize metadata with scoping if provided
	if metadata != nil {
		err = ch.cerealProvider.UnmarshalScoped(payload.Metadata, metadata, userPermissions, "read")
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize scoped metadata: %w", err)
		}
	}
	
	return payload.Data, nil
}

// SetJSON stores a JSON object using cereal serialization
func (ch *CerealDepot) SetJSON(key string, object interface{}) error {
	// Serialize object using cereal
	data, err := ch.cerealProvider.Marshal(object)
	if err != nil {
		return fmt.Errorf("failed to serialize object: %w", err)
	}
	
	// Store in depot
	return ch.depotProvider.Set(key, data, 0)
}

// GetJSON retrieves and deserializes a JSON object using cereal
func (ch *CerealDepot) GetJSON(key string, target interface{}) error {
	// Get from depot
	data, err := ch.depotProvider.Get(key)
	if err != nil {
		return err
	}
	
	// Deserialize using cereal
	return ch.cerealProvider.Unmarshal(data, target)
}

// SetJSONScoped stores a JSON object with field-level scoping
func (ch *CerealDepot) SetJSONScoped(key string, object interface{}, userPermissions []string) error {
	// Serialize object with scoping using cereal
	data, err := ch.cerealProvider.MarshalScoped(object, userPermissions)
	if err != nil {
		return fmt.Errorf("failed to serialize scoped object: %w", err)
	}
	
	// Store in depot
	return ch.depotProvider.Set(key, data, 0)
}

// GetJSONScoped retrieves and deserializes a JSON object with scoping
func (ch *CerealDepot) GetJSONScoped(key string, target interface{}, userPermissions []string) error {
	// Get from depot
	data, err := ch.depotProvider.Get(key)
	if err != nil {
		return err
	}
	
	// Deserialize with scoping using cereal
	return ch.cerealProvider.UnmarshalScoped(data, target, userPermissions, "read")
}

// CombinedPayload wraps data with metadata
type CombinedPayload struct {
	Data     []byte `json:"data"`
	Metadata []byte `json:"metadata"`
}

// Test data structures
type FileMetadata struct {
	ContentType   string            `json:"content_type"`
	Size          int64             `json:"size"`
	UploadedBy    string            `json:"uploaded_by"`
	UploadedAt    string            `json:"uploaded_at"`
	Tags          []string          `json:"tags"`
	Permissions   map[string]string `json:"permissions" scope:"read:admin"`
	EncryptionKey string            `json:"encryption_key" scope:"read:admin,write:admin"`
	Checksum      string            `json:"checksum"`
}

type DocumentMetadata struct {
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Version       int      `json:"version"`
	Classification string  `json:"classification" scope:"read:classified"`
	AccessLog     []string `json:"access_log" scope:"read:audit"`
	InternalNotes string   `json:"internal_notes" scope:"read:internal"`
}

type UserProfile struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email" scope:"read:user,write:admin"`
	FullName     string `json:"full_name"`
	Role         string `json:"role" scope:"read:admin"`
	LastLogin    string `json:"last_login" scope:"read:admin"`
	SessionToken string `json:"session_token" scope:"read:admin,write:admin"`
}

// Mock implementations
type MockDepot struct {
	data map[string][]byte
}

func NewMockDepot() *MockDepot {
	return &MockDepot{
		data: make(map[string][]byte),
	}
}

func (m *MockDepot) Set(key string, data []byte, ttl interface{}) error {
	m.data[key] = data
	fmt.Printf("Depot: Stored %d bytes at key '%s'\n", len(data), key)
	return nil
}

func (m *MockDepot) Get(key string) ([]byte, error) {
	data, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	fmt.Printf("Depot: Retrieved %d bytes from key '%s'\n", len(data), key)
	return data, nil
}

func (m *MockDepot) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockDepot) Exists(key string) (bool, error) {
	_, exists := m.data[key]
	return exists, nil
}

func (m *MockDepot) List(prefix string) ([]string, error) {
	var keys []string
	for key := range m.data {
		if len(prefix) == 0 || key[:len(prefix)] == prefix {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (m *MockDepot) GetProvider() string {
	return "mock"
}

type MockCereal struct{}

func (m *MockCereal) Marshal(data any) ([]byte, error) {
	return []byte(fmt.Sprintf(`{"mock_serialized": "%T"}`)), nil
}

func (m *MockCereal) Unmarshal(data []byte, target any) error {
	fmt.Printf("Cereal: Mock deserializing %s into %T\n", string(data), target)
	return nil
}

func (m *MockCereal) MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	// Simulate scoped serialization
	hasAdmin := contains(userPermissions, "admin")
	hasInternal := contains(userPermissions, "internal")
	hasClassified := contains(userPermissions, "classified")
	
	result := `{"mock_scoped": true, "permissions": [`
	for i, perm := range userPermissions {
		if i > 0 {
			result += ", "
		}
		result += `"` + perm + `"`
	}
	result += `], "filtered_fields": [`
	
	if !hasAdmin {
		result += `"admin_only"`
	}
	if !hasInternal {
		if !hasAdmin {
			result += `, `
		}
		result += `"internal"`
	}
	if !hasClassified {
		if !hasAdmin || !hasInternal {
			result += `, `
		}
		result += `"classified"`
	}
	
	result += `]}`
	
	return []byte(result), nil
}

func (m *MockCereal) UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	fmt.Printf("Cereal: Mock scoped deserializing %s into %T (perms: %v, op: %s)\n", string(data), target, userPermissions, operation)
	return nil
}

func (m *MockCereal) ContentType() string { return "application/json" }
func (m *MockCereal) Format() string { return "json" }

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func main() {
	fmt.Println("🗄️  Cereal-Depot Integration Demo")
	fmt.Println("=================================")
	
	// Set up services
	depot := NewMockDepot()
	cereal := &MockCereal{}
	storage := NewCerealDepot(depot, cereal)
	
	fmt.Println("\n📄 File Storage with Metadata:")
	
	// Store file with metadata
	fileData := []byte("This is the content of my important document...")
	fileMetadata := FileMetadata{
		ContentType:   "text/plain",
		Size:          int64(len(fileData)),
		UploadedBy:    "john.doe",
		UploadedAt:    time.Now().Format(time.RFC3339),
		Tags:          []string{"important", "document", "draft"},
		Permissions:   map[string]string{"read": "admin", "write": "owner"},
		EncryptionKey: "aes256-key-xyz789",
		Checksum:      "sha256:abc123def456",
	}
	
	err := storage.SetWithMetadata("documents/important.txt", fileData, fileMetadata)
	if err != nil {
		fmt.Printf("❌ Failed to store file: %v\n", err)
	} else {
		fmt.Println("✅ File stored with metadata successfully")
	}
	
	// Retrieve file with metadata
	var retrievedMetadata FileMetadata
	retrievedData, err := storage.GetWithMetadata("documents/important.txt", &retrievedMetadata)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve file: %v\n", err)
	} else {
		fmt.Printf("✅ File retrieved: %d bytes with metadata\n", len(retrievedData))
	}
	
	fmt.Println("\n📋 JSON Object Storage:")
	
	// Store user profile as JSON
	userProfile := UserProfile{
		UserID:       123,
		Username:     "johndoe",
		Email:        "john.doe@company.com",
		FullName:     "John Doe",
		Role:         "developer",
		LastLogin:    time.Now().Format(time.RFC3339),
		SessionToken: "jwt_token_abc123xyz789",
	}
	
	err = storage.SetJSON("users/123", userProfile)
	if err != nil {
		fmt.Printf("❌ Failed to store user profile: %v\n", err)
	} else {
		fmt.Println("✅ User profile stored successfully")
	}
	
	// Retrieve user profile
	var retrievedProfile UserProfile
	err = storage.GetJSON("users/123", &retrievedProfile)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve user profile: %v\n", err)
	} else {
		fmt.Println("✅ User profile retrieved successfully")
	}
	
	fmt.Println("\n🔒 Scoped File Storage (User Permissions):")
	
	// Store document with user-level permissions (sensitive fields filtered)
	documentMetadata := DocumentMetadata{
		Title:          "Quarterly Report 2024",
		Author:         "Jane Smith",
		Version:        3,
		Classification: "CONFIDENTIAL",
		AccessLog:      []string{"jane.smith", "john.doe", "admin"},
		InternalNotes:  "Need to review section 4 before final approval",
	}
	
	documentData := []byte("Q4 2024 Financial Report: Revenue increased by 15%...")
	userPermissions := []string{"user"}
	
	err = storage.SetWithScopedMetadata("documents/q4-report.pdf", documentData, documentMetadata, userPermissions)
	if err != nil {
		fmt.Printf("❌ Failed to store scoped document: %v\n", err)
	} else {
		fmt.Println("✅ Document stored with user-scoped metadata")
	}
	
	fmt.Println("\n🔒 Scoped JSON Storage (Admin Permissions):")
	
	// Store user profile with admin permissions (more fields visible)
	adminPermissions := []string{"admin"}
	err = storage.SetJSONScoped("users/123/admin-view", userProfile, adminPermissions)
	if err != nil {
		fmt.Printf("❌ Failed to store scoped user profile: %v\n", err)
	} else {
		fmt.Println("✅ User profile stored with admin-scoped serialization")
	}
	
	// Retrieve with different permission levels
	fmt.Println("\n👤 User Permission Level Retrieval:")
	var userViewProfile UserProfile
	err = storage.GetJSONScoped("users/123/admin-view", &userViewProfile, []string{"user"})
	if err != nil {
		fmt.Printf("❌ Failed to retrieve user-scoped profile: %v\n", err)
	} else {
		fmt.Println("✅ User profile retrieved with user permissions (filtered)")
	}
	
	fmt.Println("\n👑 Admin Permission Level Retrieval:")
	var adminViewProfile UserProfile
	err = storage.GetJSONScoped("users/123/admin-view", &adminViewProfile, adminPermissions)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve admin-scoped profile: %v\n", err)
	} else {
		fmt.Println("✅ User profile retrieved with admin permissions (full access)")
	}
	
	fmt.Println("\n🔍 Document Access with Classification:")
	
	// Try to retrieve classified document with different permissions
	classifiedPermissions := []string{"classified", "audit"}
	var classifiedMetadata DocumentMetadata
	_, err = storage.GetWithScopedMetadata("documents/q4-report.pdf", &classifiedMetadata, classifiedPermissions)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve classified document: %v\n", err)
	} else {
		fmt.Println("✅ Classified document retrieved with appropriate permissions")
	}
	
	fmt.Println("\n🎉 Cereal-Depot Integration Demo Complete!")
	fmt.Println("==========================================")
	fmt.Println("Key Benefits Demonstrated:")
	fmt.Println("✅ Unified serialization for object metadata")
	fmt.Println("✅ Field-level scoping for sensitive data")
	fmt.Println("✅ JSON object storage with automatic serialization")
	fmt.Println("✅ Permission-based data access control")
	fmt.Println("✅ Structured metadata management")
}