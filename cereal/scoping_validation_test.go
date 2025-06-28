package cereal

import (
	"encoding/json"
	"strings"
	"testing"
)

// Test struct with both validation and scoping
type SecureUser struct {
	Name     string `json:"name" validate:"required,min=2"`
	Email    string `json:"email" validate:"required,email"`
	SSN      string `json:"ssn" scope:"admin" validate:"required,len=11"`
	Salary   int    `json:"salary" scope:"hr" validate:"min=0"`
	IsActive bool   `json:"is_active"`
}

func TestScopingWithValidation_AdminAccess(t *testing.T) {
	user := SecureUser{
		Name:     "John Doe",
		Email:    "john@example.com",
		SSN:      "123-45-6789",
		Salary:   50000,
		IsActive: true,
	}

	// Admin should see all fields
	data, err := JSON.Marshal(user, "admin", "hr")
	if err != nil {
		t.Fatalf("Expected no error for admin, got: %v", err)
	}

	// Verify all fields are present and correct
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result["name"] != "John Doe" {
		t.Errorf("Expected name %s, got %v", "John Doe", result["name"])
	}
	if result["ssn"] != "123-45-6789" {
		t.Errorf("Expected SSN %s, got %v", "123-45-6789", result["ssn"])
	}
	if result["salary"] != float64(50000) { // JSON numbers are float64
		t.Errorf("Expected salary %d, got %v", 50000, result["salary"])
	}
}

func TestScopingWithValidation_LimitedAccess(t *testing.T) {
	user := SecureUser{
		Name:     "John Doe",
		Email:    "john@example.com",
		SSN:      "123-45-6789", // User shouldn't see this
		Salary:   50000,         // User shouldn't see this
		IsActive: true,
	}

	// Regular user should see redacted values but validation should still pass
	data, err := JSON.Marshal(user, "user") // No admin or hr permissions
	if err != nil {
		t.Fatalf("Expected no error for limited user, got: %v", err)
	}

	// Verify redacted fields are present but redacted
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Public fields should be unchanged
	if result["name"] != "John Doe" {
		t.Errorf("Expected name %s, got %v", "John Doe", result["name"])
	}
	if result["email"] != "john@example.com" {
		t.Errorf("Expected email %s, got %v", "john@example.com", result["email"])
	}
	if result["is_active"] != true {
		t.Errorf("Expected is_active %v, got %v", true, result["is_active"])
	}

	// Restricted fields should be redacted but satisfy validation
	if result["ssn"] != "XXX-XX-XXXX" { // len=11 constraint met with SSN format
		t.Errorf("Expected SSN to be redacted with proper format, got %v", result["ssn"])
	}
	if result["salary"] != float64(0) { // min=0 constraint met
		t.Errorf("Expected salary to be redacted to 0, got %v", result["salary"])
	}
}

func TestScopingWithValidation_ValidationStillWorks(t *testing.T) {
	// Test that validation still catches invalid data even with scoping
	invalidUser := SecureUser{
		Name:     "", // Invalid: required field empty
		Email:    "not-an-email", // Invalid: bad email format
		SSN:      "123-45-6789",
		Salary:   50000,
		IsActive: true,
	}

	// Should fail validation even with full admin access
	_, err := JSON.Marshal(invalidUser, "admin", "hr")
	if err == nil {
		t.Error("Expected validation error for invalid data, got nil")
	}

	// Check that it's a validation error
	if !strings.Contains(err.Error(), "validation failed") && !strings.Contains(err.Error(), "required") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

func TestScopingWithValidation_PartialAccess(t *testing.T) {
	user := SecureUser{
		Name:     "Jane Smith",
		Email:    "jane@example.com",
		SSN:      "987-65-4321",
		Salary:   75000,
		IsActive: false, // Original value - this field has no scope restriction
	}

	// User with only HR access should see salary but not SSN
	data, err := JSON.Marshal(user, "hr") // Has hr but not admin
	if err != nil {
		t.Fatalf("Expected no error for HR user, got: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Should see salary (has hr permission)
	if result["salary"] != float64(75000) {
		t.Errorf("Expected salary %d, got %v", 75000, result["salary"])
	}

	// Should not see SSN (needs admin permission)
	if result["ssn"] != "XXX-XX-XXXX" { // len=11 constraint satisfied with SSN format
		t.Errorf("Expected SSN to be redacted, got %v", result["ssn"])
	}
}

func TestRedactionValues_TypeSpecific(t *testing.T) {
	// Test different redaction values for different types
	type TestStruct struct {
		SecretString string   `json:"secret_string" scope:"admin" validate:"required"`
		SecretInt    int      `json:"secret_int" scope:"admin" validate:"min=0"`
		SecretFloat  float64  `json:"secret_float" scope:"admin" validate:"min=0"`
		SecretBool   bool     `json:"secret_bool" scope:"admin"`
		SecretSlice  []string `json:"secret_slice" scope:"admin" validate:"required"`
	}

	data := TestStruct{
		SecretString: "hidden",
		SecretInt:    42,
		SecretFloat:  3.14,
		SecretBool:   true,
		SecretSlice:  []string{"a", "b"},
	}

	// Marshal without admin permissions
	jsonData, err := JSON.Marshal(data, "user")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Check redaction values
	if result["secret_string"] != "[REDACTED]" {
		t.Errorf("Expected string redaction, got %v", result["secret_string"])
	}
	if result["secret_int"] != float64(0) { // Changed to 0 for min=0 compatibility
		t.Errorf("Expected int redaction, got %v", result["secret_int"])
	}
	if result["secret_float"] != float64(0) { // Changed to 0.0 for min=0 compatibility
		t.Errorf("Expected float redaction, got %v", result["secret_float"])
	}
	if result["secret_bool"] != true { // Boolean redacts to true
		t.Errorf("Expected bool redaction, got %v", result["secret_bool"])
	}
	// Empty slice should be present (not nil)
	if slice, ok := result["secret_slice"].([]interface{}); !ok || len(slice) != 0 {
		t.Errorf("Expected empty slice redaction, got %v", result["secret_slice"])
	}
}

func TestScopingWithValidation_RequiredFieldRedacted(t *testing.T) {
	// This is the critical test - ensuring required fields that are redacted don't break validation
	type CriticalTest struct {
		PublicField   string `json:"public" validate:"required"`
		RequiredAdmin string `json:"required_admin" scope:"admin" validate:"required,min=5"`
	}

	data := CriticalTest{
		PublicField:   "visible",
		RequiredAdmin: "secret123", // This meets validation requirements
	}

	// User without admin permission should still be able to marshal
	// The required_admin field gets redacted to "[REDACTED]" which satisfies required validation
	jsonData, err := JSON.Marshal(data, "user")
	if err != nil {
		t.Fatalf("Expected no error when redacting required field, got: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Public field should be normal
	if result["public"] != "visible" {
		t.Errorf("Expected public field to be visible, got %v", result["public"])
	}

	// Required admin field should be redacted but present (min=5 satisfied with 10-char string)
	if result["required_admin"] != "[REDACTED]" {
		t.Errorf("Expected required_admin to be redacted, got %v", result["required_admin"])
	}
}

func TestFormatSpecificRedaction(t *testing.T) {
	// Test that redacted values match expected format validators
	type FormatTest struct {
		Email     string `json:"email" scope:"admin" validate:"required,email"`
		URL       string `json:"url" scope:"admin" validate:"required,url"`
		UUID      string `json:"uuid" scope:"admin" validate:"required,uuid"`
		SSN       string `json:"ssn" scope:"admin" validate:"required,len=11"`
		Phone     string `json:"phone" scope:"admin" validate:"required,len=12"`
		CreditCard string `json:"credit_card" scope:"admin" validate:"required,len=16"`
		JSON      string `json:"json_data" scope:"admin" validate:"required,json"`
		Alpha     string `json:"alpha" scope:"admin" validate:"required,alpha,len=8"`
		Alphanum  string `json:"alphanum" scope:"admin" validate:"required,alphanum,len=10"`
		Numeric   string `json:"numeric" scope:"admin" validate:"required,numeric,len=6"`
	}

	data := FormatTest{
		Email:      "user@company.com",
		URL:        "https://api.company.com",
		UUID:       "550e8400-e29b-41d4-a716-446655440000",
		SSN:        "123-45-6789",
		Phone:      "555-123-4567",
		CreditCard: "1234567890123456",
		JSON:       `{"key":"value"}`,
		Alpha:      "ABCDEFGH",
		Alphanum:   "ABC123XYZ0",
		Numeric:    "123456",
	}

	// Marshal without admin permissions
	jsonData, err := JSON.Marshal(data, "user")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Check format-specific redacted values
	if result["email"] != "redacted@example.com" {
		t.Errorf("Expected email redaction, got %v", result["email"])
	}
	if result["url"] != "https://redacted.example.com" {
		t.Errorf("Expected URL redaction, got %v", result["url"])
	}
	if result["uuid"] != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("Expected UUID redaction, got %v", result["uuid"])
	}
	if result["ssn"] != "XXX-XX-XXXX" {
		t.Errorf("Expected SSN redaction, got %v", result["ssn"])
	}
	if result["phone"] != "XXX-XXX-XXXX" {
		t.Errorf("Expected phone redaction, got %v", result["phone"])
	}
	if result["credit_card"] != "0000000000000000" {
		t.Errorf("Expected credit card redaction, got %v", result["credit_card"])
	}
	if result["json_data"] != "{\"redacted\":true}" {
		t.Errorf("Expected JSON redaction, got %v", result["json_data"])
	}
	if result["alpha"] != "XXXXXXXX" {
		t.Errorf("Expected alpha redaction, got %v", result["alpha"])
	}
	if result["alphanum"] != "RXXXXXXXXX" {
		t.Errorf("Expected alphanum redaction, got %v", result["alphanum"])
	}
	if result["numeric"] != "000000" {
		t.Errorf("Expected numeric redaction, got %v", result["numeric"])
	}
}