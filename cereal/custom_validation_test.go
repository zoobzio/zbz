package cereal

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestCustomValidators_BuiltIn(t *testing.T) {
	// Test struct using custom validators instead of generic constraints
	type SecureData struct {
		Name       string `json:"name" validate:"required"`
		SSN        string `json:"ssn" scope:"admin" validate:"required,ssn"`
		Phone      string `json:"phone" scope:"hr" validate:"required,phone"`
		CreditCard string `json:"credit_card" scope:"finance" validate:"required,creditcard"`
		BusinessID string `json:"business_id" scope:"admin" validate:"required,businessid"`
	}

	data := SecureData{
		Name:       "John Doe",
		SSN:        "123-45-6789",
		Phone:      "555-123-4567",
		CreditCard: "4111111111111111", // Valid test credit card
		BusinessID: "BUS-12345",
	}

	// Test full access (admin + hr + finance)
	fullData, err := JSON.Marshal(data, "admin", "hr", "finance")
	if err != nil {
		t.Fatalf("Expected no error with full access, got: %v", err)
	}

	var fullResult map[string]interface{}
	err = json.Unmarshal(fullData, &fullResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal full result: %v", err)
	}

	// Verify all original values are present
	if fullResult["ssn"] != "123-45-6789" {
		t.Errorf("Expected original SSN, got %v", fullResult["ssn"])
	}

	// Test limited access (user only)
	limitedData, err := JSON.Marshal(data, "user")
	if err != nil {
		t.Fatalf("Expected no error with limited access, got: %v", err)
	}

	var limitedResult map[string]interface{}
	err = json.Unmarshal(limitedData, &limitedResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal limited result: %v", err)
	}

	// Verify custom redacted values
	if limitedResult["ssn"] != "XXX-XX-XXXX" {
		t.Errorf("Expected SSN redaction, got %v", limitedResult["ssn"])
	}
	if limitedResult["phone"] != "XXX-XXX-XXXX" {
		t.Errorf("Expected phone redaction, got %v", limitedResult["phone"])
	}
	if limitedResult["credit_card"] != "0000-0000-0000-0000" {
		t.Errorf("Expected credit card redaction, got %v", limitedResult["credit_card"])
	}
	if limitedResult["business_id"] != "XXXXXXXX" {
		t.Errorf("Expected business ID redaction, got %v", limitedResult["business_id"])
	}
}

func TestCustomValidators_ValidationLogic(t *testing.T) {
	// Test that custom validators actually validate
	type TestData struct {
		ValidSSN   string `json:"valid_ssn" validate:"ssn"`
		InvalidSSN string `json:"invalid_ssn" validate:"ssn"`
		ValidPhone string `json:"valid_phone" validate:"phone"`
	}

	validData := TestData{
		ValidSSN:   "123-45-6789",
		InvalidSSN: "123456789", // Missing dashes
		ValidPhone: "555-123-4567",
	}

	// This should fail validation due to invalid SSN
	_, err := JSON.Marshal(validData)
	if err == nil {
		t.Error("Expected validation error for invalid SSN format")
	}

	// Test with all valid data
	validData.InvalidSSN = "987-65-4321"
	_, err = JSON.Marshal(validData)
	if err != nil {
		t.Errorf("Expected no error with valid data, got: %v", err)
	}
}

func TestCustomValidators_EventHandling(t *testing.T) {
	// Test validation event handling
	var events []map[string]interface{}
	
	OnEvent(func(action string, model interface{}, data map[string]interface{}) {
		events = append(events, data)
	})
	
	// Reset event handler after test
	defer OnEvent(nil)

	type EventTest struct {
		SSN string `json:"ssn" validate:"ssn"`
	}

	// Valid SSN should generate success event
	validData := EventTest{SSN: "123-45-6789"}
	_, err := JSON.Marshal(validData)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have at least one event
	if len(events) == 0 {
		t.Error("Expected validation events to be captured")
	}

	// Check that event has correct data
	found := false
	for _, event := range events {
		if validationTag, ok := event["validation_tag"].(string); ok && validationTag == "ssn" {
			if result, ok := event["result"].(string); ok && result == "success" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Expected to find successful SSN validation event")
	}
}

func TestCustomValidators_Registration(t *testing.T) {
	// Test registering a custom validator
	err := RegisterValidator("custom_test", func(ctx context.Context, field reflect.Value, param string) error {
		if field.String() != "expected" {
			return fmt.Errorf("value must be 'expected'")
		}
		return nil
	}, "[CUSTOM_REDACTED]")
	if err != nil {
		t.Fatalf("Failed to register custom validator: %v", err)
	}

	// Check that it's registered
	validators := GetRegisteredValidators()
	found := false
	for _, tag := range validators {
		if tag == "custom_test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected custom_test to be in registered validators")
	}

	// Test using the custom validator
	type CustomTest struct {
		Field string `json:"field" validate:"custom_test"`
	}

	// Should fail validation
	invalidData := CustomTest{Field: "wrong"}
	_, err = JSON.Marshal(invalidData)
	if err == nil {
		t.Error("Expected validation error for custom validator")
	}

	// Should pass validation
	validData := CustomTest{Field: "expected"}
	_, err = JSON.Marshal(validData)
	if err != nil {
		t.Errorf("Expected no error with valid custom data, got: %v", err)
	}
}

func TestCustomValidators_CreditCardLuhn(t *testing.T) {
	// Test credit card Luhn algorithm validation
	type CreditCardTest struct {
		ValidCard   string `json:"valid_card" validate:"creditcard"`
		InvalidCard string `json:"invalid_card" validate:"creditcard"`
	}

	testData := CreditCardTest{
		ValidCard:   "4111111111111111", // Valid test card (Luhn passes)
		InvalidCard: "4111111111111112", // Invalid (Luhn fails)
	}

	// Should fail due to invalid card
	_, err := JSON.Marshal(testData)
	if err == nil {
		t.Error("Expected validation error for invalid credit card")
	}

	// Test with valid card only
	validOnlyData := CreditCardTest{
		ValidCard:   "4111111111111111",
		InvalidCard: "5555555555554444", // Another valid test card
	}

	_, err = JSON.Marshal(validOnlyData)
	if err != nil {
		t.Errorf("Expected no error with valid credit cards, got: %v", err)
	}
}

func TestCustomValidators_ScopingIntegration(t *testing.T) {
	// Test that custom validators work correctly with scoping
	type ScopedCustom struct {
		Public     string `json:"public" validate:"required"`
		AdminSSN   string `json:"admin_ssn" scope:"admin" validate:"required,ssn"`
		AdminPhone string `json:"admin_phone" scope:"admin" validate:"required,phone"`
	}

	data := ScopedCustom{
		Public:     "visible",
		AdminSSN:   "123-45-6789",
		AdminPhone: "555-123-4567",
	}

	// User without admin permission should get redacted values that still validate
	userData, err := JSON.Marshal(data, "user")
	if err != nil {
		t.Fatalf("Expected no error for user access, got: %v", err)
	}

	var userResult map[string]interface{}
	err = json.Unmarshal(userData, &userResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal user result: %v", err)
	}

	// Verify redacted values
	if userResult["admin_ssn"] != "XXX-XX-XXXX" {
		t.Errorf("Expected redacted SSN, got %v", userResult["admin_ssn"])
	}
	if userResult["admin_phone"] != "XXX-XXX-XXXX" {
		t.Errorf("Expected redacted phone, got %v", userResult["admin_phone"])
	}

	// Admin should see original values
	adminData, err := JSON.Marshal(data, "admin")
	if err != nil {
		t.Fatalf("Expected no error for admin access, got: %v", err)
	}

	var adminResult map[string]interface{}
	err = json.Unmarshal(adminData, &adminResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal admin result: %v", err)
	}

	if adminResult["admin_ssn"] != "123-45-6789" {
		t.Errorf("Expected original SSN for admin, got %v", adminResult["admin_ssn"])
	}
}