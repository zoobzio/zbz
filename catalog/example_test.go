package catalog

import (
	"encoding/json"
	"fmt"
	"testing"
)

// Example user model with comprehensive metadata tags
type User struct {
	Name     string `json:"name" db:"name" desc:"User's full name" example:"John Doe" validate:"required"`
	Email    string `json:"email" db:"email" desc:"User's email address" example:"john@example.com" validate:"required,email" scope:"profile"`
	SSN      string `json:"ssn" db:"ssn" desc:"Social Security Number" example:"123-45-6789" validate:"ssn" scope:"admin" encrypt:"pii" redact:"XXX-XX-XXXX"`
	Salary   int    `json:"salary" db:"salary" desc:"Annual salary" example:"75000" validate:"gte=0" scope:"hr" encrypt:"financial" redact:"[REDACTED]"`
	Phone    string `json:"phone" db:"phone" desc:"Phone number" example:"555-123-4567" validate:"phone" scope:"profile" encrypt:"pii" redact:"XXX-XXX-XXXX"`
	IsActive bool   `json:"is_active" db:"is_active" desc:"Account status" example:"true"`
}

// Implement ScopeProvider convention to test function detection
func (u User) GetRequiredScopes() []string {
	return []string{"user_data"}
}

// Example financial model with different encryption requirements
type Transaction struct {
	ID          string  `json:"id" db:"id" desc:"Transaction identifier" example:"txn_123"`
	Amount      float64 `json:"amount" db:"amount" desc:"Transaction amount" validate:"required,gt=0" encrypt:"homomorphic" scope:"finance"`
	AccountID   string  `json:"account_id" db:"account_id" desc:"Account identifier" validate:"required" scope:"finance" encrypt:"financial"`
	Description string  `json:"description" db:"description" desc:"Transaction description" example:"Coffee purchase"`
	Timestamp   string  `json:"timestamp" db:"timestamp" desc:"When transaction occurred" example:"2024-01-01T12:00:00Z"`
}

func (t Transaction) GetRequiredScopes() []string {
	return []string{"financial_data"}
}

func TestLazyMetadataExtraction(t *testing.T) {
	// Clear any existing cache
	clearCache()
	
	// Access User metadata - should trigger automatic extraction
	userMeta := Select[User]()
	if userMeta.TypeName != "User" {
		t.Errorf("Expected TypeName 'User', got '%s'", userMeta.TypeName)
	}
	
	// Access Transaction metadata - should trigger automatic extraction  
	transactionMeta := Select[Transaction]()
	if transactionMeta.TypeName != "Transaction" {
		t.Errorf("Expected TypeName 'Transaction', got '%s'", transactionMeta.TypeName)
	}
	
	// Verify both are now cached
	types := listRegisteredTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 cached types, got %d", len(types))
	}
}

func TestUserMetadataExtraction(t *testing.T) {
	clearCache()
	
	// Automatic extraction on first access
	metadata := Select[User]()
	
	// Verify basic metadata
	if metadata.TypeName != "User" {
		t.Errorf("Expected TypeName 'User', got '%s'", metadata.TypeName)
	}
	
	if len(metadata.Fields) != 6 {
		t.Errorf("Expected 6 fields, got %d", len(metadata.Fields))
	}
	
	// Test specific field metadata
	var ssnField *FieldMetadata
	for _, field := range metadata.Fields {
		if field.Name == "SSN" {
			ssnField = &field
			break
		}
	}
	
	if ssnField == nil {
		t.Fatal("SSN field not found")
	}
	
	// Verify SSN field has comprehensive metadata
	if ssnField.JSONName != "ssn" {
		t.Errorf("Expected JSON name 'ssn', got '%s'", ssnField.JSONName)
	}
	
	if ssnField.Description != "Social Security Number" {
		t.Errorf("Expected description 'Social Security Number', got '%s'", ssnField.Description)
	}
	
	if len(ssnField.Scopes) != 1 || ssnField.Scopes[0] != "admin" {
		t.Errorf("Expected scope 'admin', got %v", ssnField.Scopes)
	}
	
	if ssnField.Encryption.Type != "pii" {
		t.Errorf("Expected encryption type 'pii', got '%s'", ssnField.Encryption.Type)
	}
	
	if ssnField.Redaction.Value != "XXX-XX-XXXX" {
		t.Errorf("Expected redaction value 'XXX-XX-XXXX', got '%s'", ssnField.Redaction.Value)
	}
	
	if len(ssnField.Validation.CustomRules) == 0 || ssnField.Validation.CustomRules[0] != "ssn" {
		t.Errorf("Expected custom validation rule 'ssn', got %v", ssnField.Validation.CustomRules)
	}
}

func TestTransactionMetadataExtraction(t *testing.T) {
	clearCache()
	
	// Automatic extraction on first access
	metadata := Select[Transaction]()
	
	// Find amount field to test homomorphic encryption
	var amountField *FieldMetadata
	for _, field := range metadata.Fields {
		if field.Name == "Amount" {
			amountField = &field
			break
		}
	}
	
	if amountField == nil {
		t.Fatal("Amount field not found")
	}
	
	if amountField.Encryption.Type != "homomorphic" {
		t.Errorf("Expected homomorphic encryption, got '%s'", amountField.Encryption.Type)
	}
	
	if len(amountField.Scopes) != 1 || amountField.Scopes[0] != "finance" {
		t.Errorf("Expected scope 'finance', got %v", amountField.Scopes)
	}
}

func TestFunctionDetection(t *testing.T) {
	clearCache()
	
	// Automatic extraction on first access
	metadata := Select[User]()
	
	// Should detect ScopeProvider implementation
	if len(metadata.Functions) != 1 {
		t.Errorf("Expected 1 function, got %d", len(metadata.Functions))
	}
	
	function := metadata.Functions[0]
	if function.Name != "GetRequiredScopes" {
		t.Errorf("Expected function 'GetRequiredScopes', got '%s'", function.Name)
	}
	
	if function.Convention != "ScopeProvider" {
		t.Errorf("Expected convention 'ScopeProvider', got '%s'", function.Convention)
	}
}

func TestMetadataJSON(t *testing.T) {
	clearCache()
	
	// Automatic extraction on first access
	metadata := Select[User]()
	
	// Test that metadata can be serialized to JSON (useful for API endpoints)
	jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal metadata to JSON: %v", err)
	}
	
	// Print for manual inspection
	fmt.Printf("User Model Metadata JSON:\n%s\n", string(jsonBytes))
	
	// Verify it can be unmarshaled
	var unmarshaled ModelMetadata
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal metadata from JSON: %v", err)
	}
	
	if unmarshaled.TypeName != "User" {
		t.Errorf("Unmarshaled metadata has wrong type name: %s", unmarshaled.TypeName)
	}
}

func TestContainerCreation(t *testing.T) {
	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
	}
	
	// Use the new Wrap function which triggers metadata extraction
	container := Wrap(user)
	
	if container.ID == "" {
		t.Error("Container ID should not be empty")
	}
	
	if container.Version != 1 {
		t.Errorf("Expected version 1, got %d", container.Version)
	}
	
	retrievedUser := container.GetData()
	if retrievedUser.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", retrievedUser.Name)
	}
	
	// Test data update
	updatedUser := User{Name: "Jane Doe", Email: "jane@example.com"}
	originalVersion := container.Version
	container.UpdateData(updatedUser)
	
	if container.Version != originalVersion+1 {
		t.Errorf("Expected version to increment to %d, got %d", originalVersion+1, container.Version)
	}
	
	if container.GetData().Name != "Jane Doe" {
		t.Errorf("Expected updated name 'Jane Doe', got '%s'", container.GetData().Name)
	}
}

func TestGenericAPI(t *testing.T) {
	clearCache()
	
	// Test specialized accessor functions
	fields := GetFields[User]()
	if len(fields) != 6 {
		t.Errorf("Expected 6 fields, got %d", len(fields))
	}
	
	scopes := GetScopes[User]()
	expectedScopes := []string{"profile", "admin", "hr"}
	if len(scopes) != len(expectedScopes) {
		t.Errorf("Expected %d unique scopes, got %d: %v", len(expectedScopes), len(scopes), scopes)
	}
	
	encryptedFields := GetEncryptionFields[User]()
	if len(encryptedFields) != 3 { // SSN, Salary, Phone
		t.Errorf("Expected 3 encrypted fields, got %d", len(encryptedFields))
	}
	
	validatedFields := GetValidationFields[User]()
	if len(validatedFields) != 5 { // Name, Email, SSN, Salary, Phone
		t.Errorf("Expected 5 validated fields, got %d", len(validatedFields))
	}
	
	redactionRules := GetRedactionRules[User]()
	if len(redactionRules) != 3 { // SSN, Salary, Phone
		t.Errorf("Expected 3 redaction rules, got %d", len(redactionRules))
	}
	
	if redactionRules["SSN"] != "XXX-XX-XXXX" {
		t.Errorf("Expected SSN redaction 'XXX-XX-XXXX', got '%s'", redactionRules["SSN"])
	}
	
	// Test convention detection
	if !HasConvention[User]("ScopeProvider") {
		t.Error("User should implement ScopeProvider convention")
	}
	
	if HasConvention[User]("NonExistentConvention") {
		t.Error("User should not implement NonExistentConvention")
	}
	
	// Test type name extraction
	typeName := GetTypeName[User]()
	if typeName != "User" {
		t.Errorf("Expected type name 'User', got '%s'", typeName)
	}
}

// Example showing how other packages would consume this metadata
func ExampleMetadataConsumption() {
	clearCache()
	
	// Simulating cereal package using clean generic API
	scopes := GetScopes[User]()
	fmt.Printf("Cereal found %d unique scopes\n", len(scopes))
	
	// Simulating validation package using metadata for rules
	validatedFields := GetValidationFields[User]()
	fmt.Printf("Validation found %d fields with rules\n", len(validatedFields))
	
	// Simulating docs package using metadata for OpenAPI
	fields := GetFields[User]()
	exampleCount := 0
	for _, field := range fields {
		if field.Example != nil {
			exampleCount++
		}
	}
	fmt.Printf("Docs found %d fields with examples\n", exampleCount)
	
	// Simulating encryption package using metadata for field-level control
	encryptedFields := GetEncryptionFields[User]()
	fmt.Printf("Encryption found %d fields requiring encryption\n", len(encryptedFields))
}

