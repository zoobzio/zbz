package catalog

import (
	"fmt"
	"strings"
	"testing"
)

func TestCerealIntegration(t *testing.T) {
	clearCache()
	
	// Test with limited permissions (only profile access)
	userPerms := []string{"profile"}
	redactionPlan := GetRedactionPlan[User](userPerms)
	
	// Should redact SSN and Salary (admin/hr scopes), but not Email/Phone (profile scope)
	expectedRedactions := map[string]string{
		"SSN":    "XXX-XX-XXXX",
		"Salary": "[REDACTED]",
	}
	
	for field, expectedValue := range expectedRedactions {
		if redactionPlan[field] != expectedValue {
			t.Errorf("Expected %s redaction '%s', got '%s'", field, expectedValue, redactionPlan[field])
		}
	}
	
	// Email and Phone should not be redacted (user has profile scope)
	if _, exists := redactionPlan["Email"]; exists {
		t.Error("Email should not be redacted for profile scope")
	}
	if _, exists := redactionPlan["Phone"]; exists {
		t.Error("Phone should not be redacted for profile scope")
	}
}

func TestValidationIntegration(t *testing.T) {
	clearCache()
	
	rules := GetValidationRules[User]()
	
	// Check that Name has required rule
	nameRules := rules["Name"]
	if len(nameRules) == 0 || nameRules[0] != "required" {
		t.Errorf("Expected Name to have 'required' rule, got %v", nameRules)
	}
	
	// Check that Email has required + email rules
	emailRules := rules["Email"]
	hasRequired := false
	hasEmail := false
	for _, rule := range emailRules {
		if rule == "required" {
			hasRequired = true
		}
		if rule == "email" {
			hasEmail = true
		}
	}
	if !hasRequired || !hasEmail {
		t.Errorf("Expected Email to have 'required' and 'email' rules, got %v", emailRules)
	}
	
	// Check that Salary has constraint rule
	salaryRules := rules["Salary"]
	hasConstraint := false
	for _, rule := range salaryRules {
		if rule == "gte=0" {
			hasConstraint = true
			break
		}
	}
	if !hasConstraint {
		t.Errorf("Expected Salary to have 'gte=0' constraint, got %v", salaryRules)
	}
}

func TestHTTPIntegration(t *testing.T) {
	clearCache()
	
	schema := GenerateOpenAPISchema[User]()
	
	// Check basic schema structure
	if schema["type"] != "object" {
		t.Error("Expected schema type to be 'object'")
	}
	
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}
	
	// Check that Name field has correct structure
	nameProperty, exists := properties["name"]
	if !exists {
		t.Fatal("Expected 'name' property to exist")
	}
	
	nameMap := nameProperty.(map[string]interface{})
	if nameMap["type"] != "string" {
		t.Error("Expected name type to be 'string'")
	}
	
	if nameMap["description"] != "User's full name" {
		t.Error("Expected name description to match tag")
	}
	
	if nameMap["example"] != "John Doe" {
		t.Error("Expected name example to match tag")
	}
	
	// Check required fields
	required, exists := schema["required"]
	if !exists {
		t.Fatal("Expected required fields list")
	}
	
	requiredList := required.([]string)
	if len(requiredList) == 0 {
		t.Error("Expected at least one required field")
	}
	
	// Check that security annotations are present
	ssnProperty := properties["ssn"].(map[string]interface{})
	scopes, exists := ssnProperty["x-required-scopes"]
	if !exists {
		t.Error("Expected SSN to have security scope annotation")
	}
	
	scopeList := scopes.([]string)
	if len(scopeList) == 0 || scopeList[0] != "admin" {
		t.Errorf("Expected SSN to require 'admin' scope, got %v", scopeList)
	}
	
	encryption, exists := ssnProperty["x-encryption"]
	if !exists || encryption != "pii" {
		t.Errorf("Expected SSN to have 'pii' encryption annotation, got %v", encryption)
	}
}

func TestDatabaseIntegration(t *testing.T) {
	clearCache()
	
	schema := GenerateTableSchema[User]()
	
	// Check that schema contains expected elements
	if !strings.Contains(schema, "CREATE TABLE user") {
		t.Error("Expected schema to contain CREATE TABLE statement")
	}
	
	// Check for standard container fields
	requiredFields := []string{"id VARCHAR(255) PRIMARY KEY", "created_at TIMESTAMP", "version INTEGER"}
	for _, field := range requiredFields {
		if !strings.Contains(schema, field) {
			t.Errorf("Expected schema to contain '%s'", field)
		}
	}
	
	// Check for user fields
	if !strings.Contains(schema, "name VARCHAR(255)") {
		t.Error("Expected schema to contain name field")
	}
	
	// Check for NOT NULL constraints on required fields
	if !strings.Contains(schema, "name VARCHAR(255) NOT NULL") {
		t.Error("Expected name field to be NOT NULL")
	}
	
	// Check for encryption annotations
	if !strings.Contains(schema, "-- ENCRYPT:pii") {
		t.Error("Expected schema to contain encryption annotations")
	}
	
	if !strings.Contains(schema, "-- ENCRYPT:financial") {
		t.Error("Expected schema to contain financial encryption annotation")
	}
}

func TestCompleteIntegrationWorkflow(t *testing.T) {
	clearCache()
	
	// Simulate complete workflow across all integration points
	fmt.Println("\n=== Complete Integration Test ===")
	
	// 1. User defines model (already done with User struct)
	
	// 2. Cereal determines what to redact
	limitedPerms := []string{"profile"}
	redactionPlan := GetRedactionPlan[User](limitedPerms)
	fmt.Printf("1. Cereal redaction plan: %v\n", redactionPlan)
	
	// 3. Validation gets rules  
	validationRules := GetValidationRules[User]()
	fmt.Printf("2. Validation rules count: %d\n", len(validationRules))
	
	// 4. HTTP generates OpenAPI
	schema := GenerateOpenAPISchema[User]()
	properties := schema["properties"].(map[string]interface{})
	fmt.Printf("3. OpenAPI properties count: %d\n", len(properties))
	
	// 5. Database generates schema
	tableSchema := GenerateTableSchema[User]()
	lineCount := len(strings.Split(tableSchema, "\n"))
	fmt.Printf("4. Database schema lines: %d\n", lineCount)
	
	// All systems working together - metadata extracted once, used everywhere
	fmt.Printf("5. Type metadata cache hit: %s\n", GetTypeName[User]())
}

func ExampleCompleteWorkflow() {
	// This example shows how catalog enables the entire zbz framework
	// User writes one struct with tags, gets enterprise features automatically
	
	type Employee struct {
		ID       string `json:"id" db:"employee_id" desc:"Employee identifier"`
		Name     string `json:"name" validate:"required" desc:"Full name" example:"Jane Smith"`
		Email    string `json:"email" validate:"required,email" scope:"hr" desc:"Work email"`
		Salary   int    `json:"salary" scope:"admin" encrypt:"financial" redact:"[CONFIDENTIAL]"`
		SSN      string `json:"ssn" scope:"admin" encrypt:"pii" redact:"XXX-XX-XXXX" validate:"ssn"`
	}
	
	// Implement ScopeProvider convention
	// func (e Employee) GetRequiredScopes() []string {
	//     return []string{"employee_data"}
	// }
	
	// Now every service gets rich metadata automatically:
	
	// 1. Cereal: Field-level redaction based on user permissions
	scopes := GetScopes[Employee]()
	fmt.Printf("Cereal found scopes: %v\n", scopes)
	
	// 2. Validation: Comprehensive validation rules
	validatedFields := GetValidationFields[Employee]()
	fmt.Printf("Validation found %d fields with rules\n", len(validatedFields))
	
	// 3. HTTP: Auto-generated OpenAPI with security annotations
	fields := GetFields[Employee]()
	fmt.Printf("HTTP found %d documented fields\n", len(fields))
	
	// 4. Database: Schema with encryption requirements
	encryptedFields := GetEncryptionFields[Employee]()
	fmt.Printf("Database found %d fields requiring encryption\n", len(encryptedFields))
	
	// 5. Security: Field-level access control
	redactionRules := GetRedactionRules[Employee]()
	fmt.Printf("Security found %d redaction rules\n", len(redactionRules))
	
	// Output:
	// Cereal found scopes: [hr admin]
	// Validation found 3 fields with rules
	// HTTP found 5 documented fields  
	// Database found 2 fields requiring encryption
	// Security found 2 redaction rules
}