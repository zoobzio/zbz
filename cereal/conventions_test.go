package cereal

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Test models that implement conventions

// BasicUser - no conventions implemented
type BasicUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ScopedUser - implements ScopeProvider convention (simplified)
type ScopedUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Salary   int    `json:"salary"`
	SocialID string `json:"social_id"`
}

// Implement ScopeProvider convention (simplified - only required scopes)
func (u ScopedUser) GetRequiredScopes() []string {
	if u.Role == "admin" {
		return []string{"admin_data"}
	}
	if u.Role == "hr" {
		return []string{"hr_data"}
	}
	return []string{"user_data"}
}

// ValidatedUser - example user with custom validator (validation handled by other services, not cereal)
type ValidatedUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age" validate:"gte=18"`
}

// SimpleUser - only implements ScopeProvider (cereal's only concern)
type SimpleUser struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	Department string `json:"department"`
}

// Implement ScopeProvider (cereal's only convention)
func (u SimpleUser) GetRequiredScopes() []string {
	return []string{"user_management"}
}

// Tests

func TestScopeProvider_BasicDetection(t *testing.T) {
	// Test basic user (no conventions)
	basicUser := BasicUser{Name: "John", Email: "john@example.com"}
	
	// Should not implement ScopeProvider
	if _, ok := interface{}(basicUser).(ScopeProvider); ok {
		t.Error("BasicUser should not implement ScopeProvider")
	}
}

func TestScopeProvider_Implementation(t *testing.T) {
	// Test scoped user
	scopedUser := ScopedUser{Name: "Jane", Email: "jane@example.com", Role: "admin"}
	
	// Should implement ScopeProvider
	scopeProvider, ok := interface{}(scopedUser).(ScopeProvider)
	if !ok {
		t.Fatal("ScopedUser should implement ScopeProvider")
	}
	
	requiredScopes := scopeProvider.GetRequiredScopes()
	expectedScopes := []string{"admin_data"}
	if len(requiredScopes) != 1 || requiredScopes[0] != expectedScopes[0] {
		t.Errorf("Expected scopes %v, got %v", expectedScopes, requiredScopes)
	}
}

func TestCustomValidator_Registration(t *testing.T) {
	// Test the new registration pattern (like zlog fields)
	testValidator := func(ctx context.Context, field reflect.Value, param string) error {
		if field.String() == "invalid" {
			return fmt.Errorf("test validation failed")
		}
		return nil
	}
	
	err := RegisterValidator("test_validator", testValidator, "[REDACTED]")
	if err != nil {
		t.Fatalf("Failed to register custom validator: %v", err)
	}
	
	// Check it was registered
	validators := GetRegisteredValidators()
	found := false
	for _, tag := range validators {
		if tag == "test_validator" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Custom validator was not registered")
	}
}

func TestScopeEnforcement_Success(t *testing.T) {
	user := ScopedUser{Name: "Bob", Email: "bob@example.com", Role: "hr"}
	
	// User with correct permissions should succeed
	data, err := JSON.Marshal(user, "hr_data")
	if err != nil {
		t.Fatalf("Expected no error with correct permissions, got: %v", err)
	}
	
	// Verify data was serialized
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if result["name"] != "Bob" {
		t.Errorf("Expected name 'Bob', got %v", result["name"])
	}
}

func TestScopeEnforcement_Failure(t *testing.T) {
	user := ScopedUser{Name: "Charlie", Email: "charlie@example.com", Role: "admin"}
	
	// User without correct permissions should fail
	_, err := JSON.Marshal(user, "user_data") // admin requires "admin_data"
	if err == nil {
		t.Error("Expected error for insufficient permissions")
	}
	
	if !strings.Contains(err.Error(), "scope check failed") {
		t.Errorf("Expected scope check error, got: %v", err)
	}
}

func TestScopeEnforcement_NoConventions(t *testing.T) {
	user := BasicUser{Name: "David", Email: "david@example.com"}
	
	// Basic user with no conventions should always succeed
	data, err := JSON.Marshal(user, "any_permission")
	if err != nil {
		t.Fatalf("Expected no error for user without conventions, got: %v", err)
	}
	
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	if result["name"] != "David" {
		t.Errorf("Expected name 'David', got %v", result["name"])
	}
}

func TestValidation_StandardValidators(t *testing.T) {
	user := ValidatedUser{Name: "Eve", Email: "eve@example.com", Age: 25}
	
	// Valid user should serialize successfully
	_, err := JSON.Marshal(user)
	if err != nil {
		t.Fatalf("Expected no error for valid user, got: %v", err)
	}
}

func TestValidation_StandardValidators_Failure(t *testing.T) {
	user := ValidatedUser{Name: "Frank", Email: "frank@example.com", Age: 16} // Under 18
	
	// Invalid user should fail validation (using standard gte validator)
	_, err := JSON.Marshal(user)
	if err == nil {
		t.Error("Expected validation error for underage user")
	}
	
	if !strings.Contains(err.Error(), "gte") {
		t.Errorf("Expected gte validation error, got: %v", err)
	}
}

func TestSimpleWorkflow(t *testing.T) {
	user := SimpleUser{
		Name:       "Grace",
		Email:      "grace@example.com",
		Role:       "manager",
		Department: "Sales",
	}
	
	// Test simple scope workflow
	data, err := JSON.Marshal(user, "user_management")
	if err != nil {
		t.Fatalf("Expected no error for simple workflow, got: %v", err)
	}
	
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	// Verify data integrity
	if result["name"] != "Grace" {
		t.Errorf("Expected name 'Grace', got %v", result["name"])
	}
	if result["role"] != "manager" {
		t.Errorf("Expected role 'manager', got %v", result["role"])
	}
}

func TestEventCallback(t *testing.T) {
	// Test simple event callback system
	var capturedEvents []map[string]interface{}
	
	OnEvent(func(action string, model interface{}, data map[string]interface{}) {
		eventData := map[string]interface{}{
			"action": action,
			"model":  model,
			"data":   data,
		}
		capturedEvents = append(capturedEvents, eventData)
	})
	
	// Register a test validator that will trigger an event
	testValidator := func(ctx context.Context, field reflect.Value, param string) error {
		return nil // Always pass
	}
	
	err := RegisterValidator("test_event", testValidator, "[TEST]")
	if err != nil {
		t.Fatalf("Failed to register test validator: %v", err)
	}
	
	// Create a struct with the test validator and marshal it
	type TestStruct struct {
		TestField string `validate:"test_event"`
	}
	
	testData := TestStruct{TestField: "test"}
	_, err = JSON.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}
	
	// Should have captured validation events
	if len(capturedEvents) == 0 {
		t.Error("Expected to capture events, but none were captured")
	}
}