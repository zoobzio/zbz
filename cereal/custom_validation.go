package cereal

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// CustomValidationFunc is the signature for custom validation functions
type CustomValidationFunc func(ctx context.Context, field reflect.Value, param string) error

// CerealEventHandler is a simple callback for cereal events - adapters can bridge to other systems
type CerealEventHandler func(action string, model interface{}, data map[string]interface{})

// ValidatorRegistration contains a validator function and its redaction value
type ValidatorRegistration struct {
	ValidateFunc   CustomValidationFunc
	RedactionValue string
}

// CustomValidationRegistry manages custom validation functions with redaction values
type CustomValidationRegistry struct {
	validators   map[string]ValidatorRegistration
	eventHandler CerealEventHandler
}

// Global custom validation registry
var customValidationRegistry = &CustomValidationRegistry{
	validators: make(map[string]ValidatorRegistration),
}

var builtinValidatorsRegistered = false

// ensureBuiltinValidators registers built-in validators if not already done
func ensureBuiltinValidators() {
	if !builtinValidatorsRegistered {
		RegisterBuiltinValidators()
		builtinValidatorsRegistered = true
	}
}

// RegisterValidator registers a custom validation function with its redaction value (like zlog fields)
func RegisterValidator(tag string, fn CustomValidationFunc, redactionValue string) error {
	if tag == "" {
		return fmt.Errorf("validation tag cannot be empty")
	}
	if fn == nil {
		return fmt.Errorf("validation function cannot be nil")
	}
	
	customValidationRegistry.validators[tag] = ValidatorRegistration{
		ValidateFunc:   fn,
		RedactionValue: redactionValue,
	}
	
	// Register with go-playground/validator
	if dv, ok := globalValidator.(*DefaultValidator); ok {
		return dv.validate.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
			ctx := context.Background()
			err := fn(ctx, fl.Field(), fl.Param())
			
			// Simple event emission
			if customValidationRegistry.eventHandler != nil {
				data := map[string]interface{}{
					"field_name":     fl.FieldName(),
					"field_value":    fl.Field().Interface(),
					"validation_tag": tag,
					"result":         "success",
				}
				
				if err != nil {
					data["result"] = "failure"
					data["error"] = err.Error()
				}
				
				customValidationRegistry.eventHandler("validation", fl.Top().Interface(), data)
			}
			
			return err == nil
		})
	}
	
	return nil
}

// OnEvent sets a simple callback for cereal events - adapters can bridge to other systems
func OnEvent(handler CerealEventHandler) {
	customValidationRegistry.eventHandler = handler
}

// GetRegisteredValidators returns all registered custom validator tags
func GetRegisteredValidators() []string {
	var tags []string
	for tag := range customValidationRegistry.validators {
		tags = append(tags, tag)
	}
	return tags
}

// getRegisteredRedactionValue returns the redaction value for a custom validator
func getRegisteredRedactionValue(tag string) (string, bool) {
	if registration, exists := customValidationRegistry.validators[tag]; exists {
		return registration.RedactionValue, true
	}
	return "", false
}

// Built-in custom validators

// SSN validator
func validateSSN(ctx context.Context, field reflect.Value, param string) error {
	if field.Kind() != reflect.String {
		return fmt.Errorf("ssn validation only applies to strings")
	}
	
	ssn := field.String()
	
	// Accept redacted SSN pattern as valid
	if ssn == "XXX-XX-XXXX" {
		return nil
	}
	
	// Basic SSN validation: XXX-XX-XXXX format
	if len(ssn) != 11 {
		return fmt.Errorf("ssn must be 11 characters long")
	}
	
	if ssn[3] != '-' || ssn[6] != '-' {
		return fmt.Errorf("ssn must follow XXX-XX-XXXX format")
	}
	
	// Check that non-dash characters are digits
	for i, r := range ssn {
		if i == 3 || i == 6 {
			continue // Skip dashes
		}
		if r < '0' || r > '9' {
			return fmt.Errorf("ssn must contain only digits and dashes")
		}
	}
	
	return nil
}

// Phone validator
func validatePhone(ctx context.Context, field reflect.Value, param string) error {
	if field.Kind() != reflect.String {
		return fmt.Errorf("phone validation only applies to strings")
	}
	
	phone := field.String()
	
	// Accept redacted phone pattern as valid
	if phone == "XXX-XXX-XXXX" {
		return nil
	}
	
	// Basic phone validation: XXX-XXX-XXXX format
	if len(phone) != 12 {
		return fmt.Errorf("phone must be 12 characters long")
	}
	
	if phone[3] != '-' || phone[7] != '-' {
		return fmt.Errorf("phone must follow XXX-XXX-XXXX format")
	}
	
	// Check that non-dash characters are digits
	for i, r := range phone {
		if i == 3 || i == 7 {
			continue // Skip dashes
		}
		if r < '0' || r > '9' {
			return fmt.Errorf("phone must contain only digits and dashes")
		}
	}
	
	return nil
}

// CreditCard validator (basic Luhn algorithm)
func validateCreditCard(ctx context.Context, field reflect.Value, param string) error {
	if field.Kind() != reflect.String {
		return fmt.Errorf("creditcard validation only applies to strings")
	}
	
	card := field.String()
	
	// Accept redacted credit card pattern as valid
	if card == "0000-0000-0000-0000" {
		return nil
	}
	
	// Remove spaces and dashes
	var digits []int
	for _, r := range card {
		if r >= '0' && r <= '9' {
			digits = append(digits, int(r-'0'))
		} else if r != ' ' && r != '-' {
			return fmt.Errorf("creditcard must contain only digits, spaces, and dashes")
		}
	}
	
	// Must be between 13 and 19 digits
	if len(digits) < 13 || len(digits) > 19 {
		return fmt.Errorf("creditcard must be between 13 and 19 digits")
	}
	
	// Luhn algorithm check
	sum := 0
	isEven := false
	
	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]
		
		if isEven {
			digit *= 2
			if digit > 9 {
				digit = digit/10 + digit%10
			}
		}
		
		sum += digit
		isEven = !isEven
	}
	
	if sum%10 != 0 {
		return fmt.Errorf("creditcard fails Luhn checksum validation")
	}
	
	return nil
}

// BusinessIdentifier validator (flexible business ID format)
func validateBusinessID(ctx context.Context, field reflect.Value, param string) error {
	if field.Kind() != reflect.String {
		return fmt.Errorf("businessid validation only applies to strings")
	}
	
	id := field.String()
	
	// Accept redacted business ID pattern as valid
	if id == "XXXXXXXX" {
		return nil
	}
	
	// Basic validation - alphanumeric with dashes, 6-20 characters
	if len(id) < 6 || len(id) > 20 {
		return fmt.Errorf("businessid must be between 6 and 20 characters")
	}
	
	for _, r := range id {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '-') {
			return fmt.Errorf("businessid must contain only alphanumeric characters and dashes")
		}
	}
	
	return nil
}

// RegisterBuiltinValidators registers all built-in custom validators with their redaction values
func RegisterBuiltinValidators() error {
	validators := map[string]struct {
		fn        CustomValidationFunc
		redaction string
	}{
		"ssn":        {validateSSN, "XXX-XX-XXXX"},
		"phone":      {validatePhone, "XXX-XXX-XXXX"},
		"creditcard": {validateCreditCard, "0000-0000-0000-0000"},
		"businessid": {validateBusinessID, "XXXXXXXX"},
	}
	
	for tag, validator := range validators {
		if err := RegisterValidator(tag, validator.fn, validator.redaction); err != nil {
			return fmt.Errorf("failed to register validator %s: %w", tag, err)
		}
	}
	
	return nil
}

// getCustomRedactedValue returns redaction value for registered custom validators
// Updated to work with new field processor system
func getCustomRedactedValue(validateTag string, fieldType string) (any, bool) {
	if redactionValue, exists := getRegisteredRedactionValue(validateTag); exists {
		return redactionValue, true
	}
	return nil, false
}